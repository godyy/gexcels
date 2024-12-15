package parse

import (
	"fmt"
	"github.com/godyy/gexcels/internal/define"
	"github.com/godyy/gexcels/internal/log"
	pkg_errors "github.com/pkg/errors"
	"github.com/tealeg/xlsx/v3"
	"reflect"
	"strconv"
	"strings"
)

// TableField 配置表字段
type TableField struct {
	*Field     // base
	Col    int // 非Global配置时有效
}

// TableEntry 配置条目数据
type TableEntry map[string]interface{}

// Table 配置表定义
type Table struct {
	Tag          Tag                        // 标签
	Name         string                     // 表名
	Desc         string                     // 描述
	IsGlobal     bool                       // 是否全局配置表
	Fields       []*TableField              // 字段
	fieldByName  map[string]int             // 字段名映射
	Entries      []TableEntry               // for normal
	entryByID    map[interface{}]TableEntry // for normal
	entryByName  map[string]interface{}     // for global
	uniqueValues map[string]bool            // 唯一键值存在映射 [fieldName+fieldValue]

	links         []*TableLink         // 外链规则
	CompositeKeys []*TableCompositeKey // 组合键
	Groups        []*TableGroup        // 分组
}

func newTable(tag Tag, name, desc string, isGlobal bool) *Table {
	table := &Table{
		Tag:      tag,
		Name:     name,
		Desc:     desc,
		IsGlobal: isGlobal,
	}
	return table
}

// addField 添加字段
func (td *Table) addField(fd *TableField) {
	if len(td.Fields) > tableMaxField {
		panic(errFieldNumberExceedLimit)
	}
	td.Fields = append(td.Fields, fd)
	td.fieldByName[fd.Name] = len(td.Fields) - 1
}

// GetFieldByName 获取字段
func (td *Table) GetFieldByName(name string) *TableField {
	if i, ok := td.fieldByName[name]; ok {
		return td.Fields[i]
	} else {
		return nil
	}
}

// hasField 是否存在字段
func (td *Table) hasField(name string) bool {
	_, ok := td.fieldByName[name]
	return ok
}

// GetFieldID 获取ID字段
func (td *Table) GetFieldID() *TableField {
	return td.Fields[define.TableColFieldID]
}

// addEntry 添加条目
func (td *Table) addEntry(id interface{}, entry TableEntry) {
	td.Entries = append(td.Entries, entry)
	td.entryByID[id] = entry
}

// hasEntry 是否存在条目
func (td *Table) hasEntry(id interface{}) bool {
	_, ok := td.entryByID[id]
	return ok
}

// addEntryByName 通过名称添加条目
func (td *Table) addEntryByName(name string, entry interface{}) {
	td.entryByName[name] = entry
}

// GetEntryByName 通过名称获取条目
func (td *Table) GetEntryByName(name string) interface{} {
	if len(td.entryByName) <= 0 {
		return nil
	}
	return td.entryByName[name]
}

// addUniqueValue 添加唯一值
func (td *Table) addUniqueValue(fieldName string, value interface{}) bool {
	s := convertUniqueValue2String(value)
	key := fieldName + ":" + s
	if td.uniqueValues == nil {
		td.uniqueValues = make(map[string]bool)
		td.uniqueValues[key] = true
		return true
	} else if !td.uniqueValues[key] {
		td.uniqueValues[key] = true
		return true
	} else {
		return false
	}
}

// hasUniqueValue 是否存在唯一值
func (td *Table) hasUniqueValue(fieldName string, value interface{}) bool {
	if td.uniqueValues == nil {
		return false
	}
	s := convertUniqueValue2String(value)
	key := fieldName + ":" + s
	return td.uniqueValues[key]
}

// addTable 添加表
func (p *Parser) addTable(td *Table) {
	p.Tables = append(p.Tables, td)
	p.tableByName[td.Name] = td
}

// getTableByName 根据名称获取表
func (p *Parser) getTableByName(name string) *Table {
	return p.tableByName[name]
}

// hasTable 是否存在表
func (p *Parser) hasTable(name string) bool {
	_, ok := p.tableByName[name]
	return ok
}

const (
	// 配置表表头行定义
	tableRowFieldName = 0 // 字段名称行号
	tableRowFieldDesc = 1 // 字段描述行号
	tableRowFieldType = 2 // 字段类型行号
	tableRowFieldRule = 3 // 字段规则行号
	tableRowFieldTag  = 4 // 字段tag行号

	// 第一条目行号
	tableRowFirstEntry = tableRowFieldTag + 1

	tableMaxField = 32767 // 配置表字段上限
)

// parseTables 解析配置表
func (p *Parser) parseTables(files []*tableFileInfo) error {
	for _, file := range files {
		if err := p.parseTableFile(file); err != nil {
			return err
		}
	}
	return nil
}

// parseTableFile 解析配置表文件
func (p *Parser) parseTableFile(info *tableFileInfo) error {
	file, err := xlsx.OpenFile(info.path)
	if err != nil {
		return err
	}
	for _, sheet := range file.Sheets {
		if !tableSheetNameRegexp.MatchString(sheet.Name) {
			continue
		}
		td, err := p.parseTableOfSheet(sheet, info.tag)
		if err != nil {
			return pkg_errors.WithMessagef(err, "[%s][%s]", info.path, sheet.Name)
		}
		if p.hasTable(td.Name) {
			return fmt.Errorf("[%s][%s]: table name duplicate", info.path, sheet.Name)
		}
		p.addTable(td)
		log.PrintfGreen("[%s][%s] parsed", info.path, sheet.Name)
	}
	return nil
}

// parseTableOfSheet 解析sheet中定义的配置表
func (p *Parser) parseTableOfSheet(sheet *xlsx.Sheet, tag Tag) (*Table, error) {
	nameMatches := tableSheetNameRegexp.FindStringSubmatch(strings.TrimSpace(sheet.Name))
	if len(nameMatches) <= 0 {
		return nil, errSheetNameInvalid
	}

	if globalTableNameRegexp.MatchString(nameMatches[2]) {
		return p.parseGlobalTable(sheet, tag, nameMatches[2], nameMatches[1])
	} else {
		return p.parseTable(sheet, tag, nameMatches[2], nameMatches[1])
	}
}

// parseTable 解析sheet中的配置表内容到td指定的配置表中
func (p *Parser) parseTable(sheet *xlsx.Sheet, tag Tag, name, desc string) (*Table, error) {
	if sheet.MaxRow < tableRowFirstEntry || sheet.MaxCol < 1 {
		return nil, errSheetRowsOrColsNotMatch
	}

	td := newTable(tag, name, desc, false)

	// 解析字段
	if err := p.parseTableFields(td, sheet); err != nil {
		return nil, pkg_errors.WithMessage(err, " fields")
	}

	// 解析条目
	if err := p.parseTableEntries(td, sheet); err != nil {
		return nil, pkg_errors.WithMessage(err, " entries")
	}

	return td, nil
}

// parseTableFields 解析sheet中的字段定义到td指定的配置表中
func (p *Parser) parseTableFields(td *Table, sheet *xlsx.Sheet) error {
	td.Fields = make([]*TableField, 0, sheet.MaxCol)
	td.fieldByName = make(map[string]int, sheet.MaxCol)

	for i := 0; i < sheet.MaxCol; i++ {
		if i > 0 {
			tag, err := getSheetValue(sheet, tableRowFieldTag, i, true)
			if err != nil {
				return pkg_errors.WithMessagef(err, " get coll[%d] tag cell", i)
			}
			if ok, valid := p.checkTag(tag); !valid {
				return fmt.Errorf("col[%d] tag(%s) invalid", i, tag)
			} else if !ok {
				continue
			}
		}

		_, err := p.parseTableField(td, sheet, i)
		if err != nil {
			return pkg_errors.WithMessagef(err, "coll[%d]", i)
		}
	}

	return nil
}

// parseTableField 解析sheet中col列定义的字段到td指定的配置表中
func (p *Parser) parseTableField(td *Table, sheet *xlsx.Sheet, col int) (*TableField, error) {
	fieldName, err := getSheetValue(sheet, tableRowFieldName, col, true)
	if err != nil {
		return nil, pkg_errors.WithMessage(err, "get name cell")
	}
	if fieldName == "" {
		return nil, nil
	}
	if !fieldNameRegexp.MatchString(fieldName) {
		return nil, errFieldNameInvalid(fieldName)
	}
	if td.hasField(fieldName) {
		return nil, errFieldNameDuplicate(fieldName)
	}

	fieldDesc, err := getSheetValue(sheet, tableRowFieldDesc, col)
	if err != nil {
		return nil, pkg_errors.WithMessage(err, "get desc cell")
	}

	fieldType, err := getSheetValue(sheet, tableRowFieldType, col, true)
	if err != nil {
		return nil, pkg_errors.WithMessage(err, "get type cell")
	}
	if fieldType == "" {
		return nil, nil
	}
	if !fieldTypeRegexp.MatchString(fieldType) {
		return nil, errFieldTypeInvalid(fieldType)
	}

	fd := &TableField{
		Field: newField(fieldName, fieldDesc),
		Col:   col,
	}

	if err := p.parseFieldType(fd.Field, fieldType); err != nil {
		return nil, pkg_errors.WithMessagef(err, "field type (%s)", fieldType)
	}

	if col == define.TableColFieldID {
		if fd.Name != define.TableFieldIDName {
			return nil, errFirstFieldMustID
		}
		if !fd.Type.Primitive() {
			return nil, errIDNonPrimitive
		}
		fd.addRule(&fieldUnique{exportMethod: true})
	}

	td.addField(fd)

	if col != 0 {
		fieldRule, err := getSheetValue(sheet, tableRowFieldRule, col, true)
		if err != nil {
			return nil, pkg_errors.WithMessage(err, "get fieldRule cell")
		}
		if err = p.parseTableFieldRules(td, fd, fieldRule); err != nil {
			return nil, err
		}
		if fd.Type == FTStruct || (fd.Type == FTArray && fd.ElementType == FTStruct) {
			var fieldPath []string
			var links []*TableLink
			p.getFieldTableLinks(fd.Field, &fieldPath, &links)
			td.addLink(links...)
		}
	}

	return fd, nil
}

// parseTableEntries 解析sheet中定义的条目数据到td指定的配置表
func (p *Parser) parseTableEntries(td *Table, sheet *xlsx.Sheet) error {
	entryCount := sheet.MaxRow - tableRowFirstEntry
	if entryCount <= 0 {
		return nil
	}

	td.Entries = make([]TableEntry, 0, entryCount)
	td.entryByID = make(map[interface{}]TableEntry, entryCount)

	var (
		row *xlsx.Row
		fd  *TableField
		id  interface{}
		val interface{}
		err error
	)

	for i := tableRowFirstEntry; i < sheet.MaxRow; i++ {
		row, err = sheet.Row(i)
		if err != nil {
			return pkg_errors.WithMessagef(err, "get row[%d]", i+1)
		}

		if isRowComment(row) {
			continue
		}

		entry := make(TableEntry, len(td.Fields))
		skip := false
		for k := 0; k < len(td.Fields); k++ {
			fd = td.Fields[k]
			value := row.GetCell(fd.Col).Value

			if fd.Col == define.TableColFieldID {
				if value == "" {
					skip = true
					break
				}
			}

			val, err = p.parseFieldValue(fd.Field, value)
			if err != nil {
				return pkg_errors.WithMessagef(err, "row[%d] %s={%s}", i+1, fd.Name, value)
			}

			if fd.Col == define.TableColFieldID {
				if val == nil {
					return fmt.Errorf("row[%d] %s empty", i+1, fd.Name)
				}
				if td.hasEntry(val) {
					return fmt.Errorf("row[%d] %s=%s duplicate", i+1, fd.Name, value)
				}
				id = val
			}

			if val != nil {
				entry[fd.Name] = val
			}

			if fd.Col != define.TableColFieldID && fd.Unique() {
				if !td.addUniqueValue(fd.Name, val) {
					return fmt.Errorf("row[%d] %s=%s duplicate", i+1, fd.Name, value)
				}
			}
		}

		if skip {
			continue
		}

		if err := td.addCompositeKeyValue(entry); err != nil {
			return err
		}

		td.addEntry(id, entry)
	}

	return nil
}

// convertUniqueValue2String 将唯一值转换为字符串
func convertUniqueValue2String(v interface{}) string {
	switch o := v.(type) {
	case string:
		return o
	case int32:
		return strconv.FormatInt(int64(o), 10)
	case int64:
		return strconv.FormatInt(o, 10)
	case float32:
		return strconv.FormatFloat(float64(o), 'f', -1, 32)
	case float64:
		return strconv.FormatFloat(o, 'f', -1, 32)
	case bool:
		return strconv.FormatBool(o)
	default:
		panic("invalid unique value type " + reflect.TypeOf(v).String())
	}
}
