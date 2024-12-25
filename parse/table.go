package parse

import (
	"errors"
	"fmt"
	"github.com/godyy/gexcels"
	"github.com/godyy/gexcels/internal/log"
	pkg_errors "github.com/pkg/errors"
	"github.com/tealeg/xlsx/v3"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

// tableTagRegexp 配置表内标签匹配正则表达式
var tableTagRegexp = regexp.MustCompile(gexcels.TagPattern + `(?:` + gexcels.TagSep + gexcels.TagPattern + `)*`)

// Table 配置表定义
type Table struct {
	*gexcels.Table                                    // 基础数据
	Entries        []gexcels.TableEntry               // for normal
	entryByID      map[interface{}]gexcels.TableEntry // for normal
	entryByName    map[string]interface{}             // for global
	uniqueValues   map[string]bool                    // 唯一键值存在映射 [fieldName+fieldValue]

	links         []*TableLink         // 外链规则
	CompositeKeys []*TableCompositeKey // 组合键
	Groups        []*TableGroup        // 分组
}

func newTable(tag gexcels.Tag, name, desc string, isGlobal bool) *Table {
	return &Table{
		Table: gexcels.NewTable(tag, name, desc, isGlobal),
	}
}

// addEntry 添加条目
func (td *Table) addEntry(id interface{}, entry gexcels.TableEntry) {
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
		td, err := p.parseTableOfSheet(sheet, info.tag)
		if err != nil {
			if errors.Is(err, errSheetNameInvalid) {
				continue
			}
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

// tableSheetNameRegexp 配置表sheet名匹配正则表达式
var tableSheetNameRegexp = regexp.MustCompile(`^(.*)\|(` + gexcels.NamePattern + `)$`)

// parseTableOfSheet 解析sheet中定义的配置表
func (p *Parser) parseTableOfSheet(sheet *xlsx.Sheet, tag gexcels.Tag) (*Table, error) {
	nameMatches := tableSheetNameRegexp.FindStringSubmatch(strings.TrimSpace(sheet.Name))
	if len(nameMatches) <= 0 {
		return nil, errSheetNameInvalid
	}

	if strings.HasPrefix(nameMatches[2], gexcels.GlobalTableNamePrefix) {
		return p.parseGlobalTable(sheet, tag, nameMatches[2], nameMatches[1])
	} else {
		return p.parseTable(sheet, tag, nameMatches[2], nameMatches[1])
	}
}

// parseTable 解析sheet中的配置表内容到td指定的配置表中
func (p *Parser) parseTable(sheet *xlsx.Sheet, tag gexcels.Tag, name, desc string) (*Table, error) {
	if sheet.MaxRow < gexcels.TableRowFirstEntry || sheet.MaxCol < 1 {
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
	td.Fields = make([]*gexcels.TableField, 0, sheet.MaxCol)
	td.FieldByName = make(map[string]*gexcels.TableField, sheet.MaxCol)

	for i := 0; i < sheet.MaxCol; i++ {
		if i > 0 {
			tag, err := getSheetValue(sheet, gexcels.TableRowFieldTag, i, true)
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
func (p *Parser) parseTableField(td *Table, sheet *xlsx.Sheet, col int) (*gexcels.TableField, error) {
	fieldName, err := getSheetValue(sheet, gexcels.TableRowFieldName, col, true)
	if err != nil {
		return nil, pkg_errors.WithMessage(err, "get name cell")
	}
	if fieldName == "" {
		return nil, nil
	}
	if !gexcels.MatchName(fieldName) {
		return nil, errFieldNameInvalid(fieldName)
	}
	if td.HasField(fieldName) {
		return nil, errFieldNameDuplicate(fieldName)
	}

	fieldDesc, err := getSheetValue(sheet, gexcels.TableRowFieldDesc, col)
	if err != nil {
		return nil, pkg_errors.WithMessage(err, "get desc cell")
	}

	fieldType, err := getSheetValue(sheet, gexcels.TableRowFieldType, col, true)
	if err != nil {
		return nil, pkg_errors.WithMessage(err, "get type cell")
	}
	if fieldType == "" {
		return nil, nil
	}

	fieldTypeInfo, err := p.parseFieldTypeInfo(fieldType)
	if err != nil {
		return nil, pkg_errors.WithMessagef(err, "field type (%s)", fieldType)
	}

	fd := gexcels.NewTableField(
		gexcels.NewField(fieldName, fieldDesc, fieldTypeInfo),
		col,
	)

	if col == gexcels.TableColFieldID {
		if fd.Name != gexcels.TableFieldIDName {
			return nil, errFirstFieldMustID
		}
		if !fd.Type.Primitive() {
			return nil, errIDNonPrimitive
		}
		fd.AddRule(gexcels.NewFRUnique())
	}

	td.AddField(fd)

	if col != 0 {
		fieldRule, err := getSheetValue(sheet, gexcels.TableRowFieldRule, col, true)
		if err != nil {
			return nil, pkg_errors.WithMessage(err, "get fieldRule cell")
		}
		if err = p.parseTableFieldRules(td, fd, fieldRule); err != nil {
			return nil, err
		}
		if fd.Type == gexcels.FTStruct || (fd.Type == gexcels.FTArray && fd.ElementType == gexcels.FTStruct) {
			var fieldPath []string
			var links []*TableLink
			p.getFieldTableLinks(fd.Field, &fieldPath, &links)
			td.addLink(links...)
		}
	}

	return fd, nil
}

// parseTableFieldRules 解析配置表字段规则
func (p *Parser) parseTableFieldRules(td *Table, fd *gexcels.TableField, s string) error {
	if s == "" {
		return nil
	}

	rules := strings.Split(s, gexcels.FRSep)
	for _, rule := range rules {
		rule = strings.TrimSpace(rule)
		fr, err := gexcels.ParseFieldRule(rule)
		if err != nil {
			return err
		}

		if !fd.AddRule(fr) {
			return errFieldRuleMultiple(fr.FRName())
		}

		switch r := fr.(type) {
		case *gexcels.FRUnique:
			if td.IsGlobal {
				return errFieldRuleOnGlobalTable(fr.FRName())
			}
			if !fd.Type.Primitive() {
				return errFieldRuleOnNonPrimitiveField(fr.FRName())
			}
		case *gexcels.FRLink:
			if !fd.Type.Primitive() && !(fd.Type == gexcels.FTArray && fd.ElementType.Primitive()) {
				return errFieldRuleOnNonPrimitiveField(fr.FRName())
			}
			td.addLink(newTableLink([]string{fd.Name}, r.TableName, r.FieldName))
		case *gexcels.FRCompositeKey:
			if td.IsGlobal {
				return errFieldRuleOnGlobalTable(fr.FRName())
			}
			if !fd.Type.Primitive() {
				return errFieldRuleOnNonPrimitiveField(fr.FRName())
			}
			if !td.addCompositeKey(r.KeyName, r.Index, fd.Name) {
				return fmt.Errorf("composite-key %s keyIndex %d duplicate", r.KeyName, r.Index)
			}
		case *gexcels.FRGroup:
			if td.IsGlobal {
				return errFieldRuleOnGlobalTable(fr.FRName())
			}
			if !fd.Type.Primitive() {
				return errFieldRuleOnNonPrimitiveField(fr.FRName())
			}
			if !td.addGroup(r.GroupName, r.Index, fd.Name) {
				return fmt.Errorf("group %s index %d duplicate", r.GroupName, r.Index)
			}
		}
	}

	return nil
}

// parseTableEntries 解析sheet中定义的条目数据到td指定的配置表
func (p *Parser) parseTableEntries(td *Table, sheet *xlsx.Sheet) error {
	entryCount := sheet.MaxRow - gexcels.TableRowFirstEntry
	if entryCount <= 0 {
		return nil
	}

	td.Entries = make([]gexcels.TableEntry, 0, entryCount)
	td.entryByID = make(map[interface{}]gexcels.TableEntry, entryCount)

	var (
		row *xlsx.Row
		fd  *gexcels.TableField
		id  interface{}
		val interface{}
		err error
	)

	for i := gexcels.TableRowFirstEntry; i < sheet.MaxRow; i++ {
		row, err = sheet.Row(i)
		if err != nil {
			return pkg_errors.WithMessagef(err, "get row[%d]", i+1)
		}

		if isRowComment(row) {
			continue
		}

		entry := make(gexcels.TableEntry, len(td.Fields))
		skip := false
		for k := 0; k < len(td.Fields); k++ {
			fd = td.Fields[k]
			value := row.GetCell(fd.Col).Value

			if fd.Col == gexcels.TableColFieldID {
				if value == "" {
					skip = true
					break
				}
			}

			val, err = p.parseFieldValue(fd.Field, value)
			if err != nil {
				return pkg_errors.WithMessagef(err, "row[%d] %s={%s}", i+1, fd.Name, value)
			}

			if fd.Col == gexcels.TableColFieldID {
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

			if fd.Col != gexcels.TableColFieldID && fd.Unique() {
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

// parseGlobalTable 解析sheet中的global配置表到td
func (p *Parser) parseGlobalTable(sheet *xlsx.Sheet, tag gexcels.Tag, name, desc string) (*Table, error) {
	if sheet.MaxRow < gexcels.GlobalTableSkipRows || sheet.MaxCol < gexcels.GlobalTableCols {
		return nil, errSheetRowsOrColsNotMatch
	}

	td := newTable(tag, name, desc, true)
	fieldCount := sheet.MaxRow - gexcels.GlobalTableSkipRows
	td.Fields = make([]*gexcels.TableField, 0, fieldCount)
	td.FieldByName = make(map[string]*gexcels.TableField, fieldCount)
	td.entryByName = make(map[string]interface{}, fieldCount)

	for i := gexcels.GlobalTableSkipRows; i < sheet.MaxRow; i++ {
		row, err := sheet.Row(i)
		if err != nil {
			return nil, pkg_errors.WithMessagef(err, "get row[%d]", i+1)
		}

		if isRowComment(row) {
			continue
		}

		rowTag := strings.TrimSpace(row.GetCell(gexcels.GlobalTableColFieldTag).Value)
		if ok, valid := p.checkTag(rowTag); !valid {
			return nil, fmt.Errorf("row[%d] tag(%s) invalid", i+1, rowTag)
		} else if !ok {
			continue
		}

		if err := p.parseGlobalTableField(td, row); err != nil {
			return nil, pkg_errors.WithMessagef(err, "field {%s}", row.GetCell(gexcels.GlobalTableColFieldName).Value)
		}
	}
	return td, nil
}

// parseGlobalTableField 解析row定义的字段到global配置表td
func (p *Parser) parseGlobalTableField(td *Table, row *xlsx.Row) error {
	fieldName := strings.TrimSpace(row.GetCell(gexcels.GlobalTableColFieldName).Value)
	if fieldName == "" {
		return nil
	}

	if !gexcels.MatchName(fieldName) {
		return errFieldNameInvalid(fieldName)
	}

	if td.HasField(fieldName) {
		return errFieldNameDuplicate(fieldName)
	}

	fieldTypeInfo, err := p.parseFieldTypeInfo(row.GetCell(gexcels.GlobalTableColFieldType).Value)
	if err != nil {
		return err
	}

	fd := gexcels.NewTableField(
		gexcels.NewField(fieldName, row.GetCell(gexcels.GlobalTableColFieldDesc).Value, fieldTypeInfo),
		0,
	)

	val, err := p.parseFieldValue(fd.Field, row.GetCell(gexcels.GlobalTableColFieldValue).Value)
	if err != nil {
		return err
	}

	if err := p.parseTableFieldRules(td, fd, row.GetCell(gexcels.GlobalTableColFieldRule).Value); err != nil {
		return err
	}

	td.AddField(fd)
	td.addEntryByName(fd.Name, val)

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
