package gexcels

import (
	"fmt"
	"sort"
	"strings"

	pkg_errors "github.com/pkg/errors"
	"github.com/tealeg/xlsx/v3"
)

const (
	idExportName = "ID" // ID字段名
	idJsonName   = "id" // ID字段json名

	tableRowFieldName  = 0                    // 字段名称行号
	tableRowFieldDesc  = 1                    // 字段描述行号
	tableRowFieldType  = 2                    // 字段类型行号
	tableRowFieldRule  = 3                    // 字段规则行号
	tableRowFieldTag   = 4                    // 字段tag行号
	tableRowFirstEntry = tableRowFieldTag + 1 // 第一条目行号

	tableColID = 0 // ID字段列号

	tableMaxField = 32767 // 配置表字段上限
)

// TableField 配置表字段
type TableField struct {
	*Field     // base
	Col    int // 非Global配置时有效
}

// TableLink 配置表链接
type TableLink struct {
	SrcField []string // 源字段，可以指向结构体内部字段
	DstTable string   // 目标配置表
	DstField string   // 目标字段，ID或者unique
}

func newTableLink(srcField []string, dstTable, dstField string) *TableLink {
	return &TableLink{
		SrcField: srcField,
		DstTable: dstTable,
		DstField: dstField,
	}
}

// TableCompositeKey 配置表组合键
type TableCompositeKey struct {
	KeyName        string         // key名
	Key2FieldIndex map[int]string // index映射 [keyIndex]fieldName
	KeyIndexes     []int          // 键索引
}

func newTableCompositeKey(keyName string) *TableCompositeKey {
	return &TableCompositeKey{
		KeyName:        keyName,
		Key2FieldIndex: make(map[int]string),
	}
}

func (tck *TableCompositeKey) addIndex(key int, fieldName string) bool {
	if _, ok := tck.Key2FieldIndex[key]; ok {
		return false
	} else {
		tck.Key2FieldIndex[key] = fieldName
		return true
	}
}

func (tck *TableCompositeKey) SortedFields() []string {
	if len(tck.KeyIndexes) == 0 {
		tck.KeyIndexes = make([]int, 0, len(tck.Key2FieldIndex))
		for i := range tck.Key2FieldIndex {
			tck.KeyIndexes = append(tck.KeyIndexes, i)
		}
		sort.Ints(tck.KeyIndexes)
	}
	fields := make([]string, len(tck.KeyIndexes))
	for i, v := range tck.KeyIndexes {
		fields[i] = tck.Key2FieldIndex[v]
	}
	return fields
}

// TableEntry 配置条目数据
type TableEntry map[string]interface{}

// Table 配置表定义
type Table struct {
	Name                  string                     // 表名
	Desc                  string                     // 描述
	IsGlobal              bool                       // 是否全局配置表
	Fields                []*TableField              // 字段
	FieldByName           map[string]int             // 字段名映射
	Entries               []TableEntry               // for normal
	EntryByID             map[interface{}]TableEntry // for normal
	EntryByName           map[string]interface{}     // for global
	UniqueValues          map[string]map[string]bool // [fieldName][fieldValue]
	ExportEntryStructName string                     // 导出结构体名
	ExportTableStructName string                     // 导出表结构名
	ExportTableName       string                     // 导出的表名

	Links         []*TableLink         // 外链规则
	CompositeKeys []*TableCompositeKey // 组合键
}

func newTable(name, desc string, isGlobal bool) *Table {
	table := &Table{
		Name:            name,
		Desc:            desc,
		IsGlobal:        isGlobal,
		ExportTableName: toLowerSnake(name),
	}
	if isGlobal {
		table.ExportTableStructName = exportGlobalTableStructName(name)
	} else {
		table.ExportTableStructName = exportNormalTableStructName(name)
		table.ExportEntryStructName = exportStructName(name)
	}
	return table
}

func (td *Table) addField(fd *TableField) {
	if len(td.Fields) > tableMaxField {
		panic(errFieldNumberExceedLimit)
	}
	td.Fields = append(td.Fields, fd)
	td.FieldByName[fd.Name] = len(td.Fields) - 1
}

func (td *Table) GetFieldByName(name string) *TableField {
	if i, ok := td.FieldByName[name]; ok {
		return td.Fields[i]
	} else {
		return nil
	}
}

func (td *Table) hasField(name string) bool {
	_, ok := td.FieldByName[name]
	return ok
}

func (td *Table) addEntry(id interface{}, entry TableEntry) {
	td.Entries = append(td.Entries, entry)
	td.EntryByID[id] = entry
}

func (td *Table) hasEntry(id interface{}) bool {
	_, ok := td.EntryByID[id]
	return ok
}

func (td *Table) addEntryByName(name string, entry interface{}) {
	td.EntryByName[name] = entry
}

func (td *Table) getEntryByName(name string) interface{} {
	if len(td.EntryByName) <= 0 {
		return nil
	}
	return td.EntryByName[name]
}

func (td *Table) addUniqueValue(fieldName string, value interface{}) bool {
	s := convertUniqueValue2String(value)

	if td.UniqueValues == nil {
		td.UniqueValues = make(map[string]map[string]bool)
	}

	if values := td.UniqueValues[fieldName]; values == nil {
		values = make(map[string]bool)
		td.UniqueValues[fieldName] = values
		values[s] = true
		return true
	} else if ok := values[s]; ok {
		return false
	} else {
		values[s] = true
		return true
	}
}

func (td *Table) hasUniqueValue(fieldName string, value interface{}) bool {
	s := convertUniqueValue2String(value)

	if td.UniqueValues == nil {
		return false
	}

	values := td.UniqueValues[fieldName]
	if values == nil {
		return false
	}

	return values[s]
}

func (td *Table) addLink(rule ...*TableLink) {
	td.Links = append(td.Links, rule...)
}

func (td *Table) addCompositeKey(keyName string, keyIndex int, fieldName string) bool {
	var ck *TableCompositeKey
	if n := sort.Search(len(td.CompositeKeys), func(i int) bool {
		return keyName <= td.CompositeKeys[i].KeyName
	}); n < len(td.CompositeKeys) && keyName == td.CompositeKeys[n].KeyName {
		ck = td.CompositeKeys[n]
	} else {
		ck = newTableCompositeKey(keyName)
		td.CompositeKeys = append(td.CompositeKeys, nil)
		copy(td.CompositeKeys[n+1:], td.CompositeKeys[n:])
		td.CompositeKeys[n] = ck
	}

	return ck.addIndex(keyIndex, fieldName)
}

func (p *Parser) addTable(td *Table) {
	p.tables = append(p.tables, td)
	p.tableByName[td.Name] = len(p.tables) - 1
}

func (p *Parser) getTableByName(name string) *Table {
	if i, ok := p.tableByName[name]; ok {
		return p.tables[i]
	} else {
		return nil
	}
}

func (p *Parser) hasTable(name string) bool {
	_, ok := p.tableByName[name]
	return ok
}

// parseTableOfSheet 解析sheet中定义的配置表
func (p *Parser) parseTableOfSheet(sheet *xlsx.Sheet) (*Table, error) {
	nameMatches := tableSheetNameRegexp.FindStringSubmatch(strings.TrimSpace(sheet.Name))
	if len(nameMatches) <= 0 {
		return nil, errSheetNameInvalid
	}

	if globalTableNameRegexp.MatchString(nameMatches[2]) {
		return p.parseGlobalTable(sheet, nameMatches[2], nameMatches[1])
	} else {
		return p.parseTable(sheet, nameMatches[2], nameMatches[1])
	}
}

// parseTable 解析sheet中的配置表内容到td指定的配置表中
func (p *Parser) parseTable(sheet *xlsx.Sheet, name, desc string) (*Table, error) {
	if sheet.MaxRow < tableRowFirstEntry || sheet.MaxCol < 1 {
		return nil, errSheetRowsOrColsNotMatch
	}

	td := newTable(name, desc, false)

	// 解析字段
	if err := p.parseTableFields(td, sheet); err != nil {
		return nil, pkg_errors.WithMessage(err, "parse fields")
	}

	// 解析条目
	if err := p.parseTableEntries(td, sheet); err != nil {
		return nil, pkg_errors.WithMessage(err, "parse entries")
	}

	return td, nil
}

// parseTableFields 解析sheet中的字段定义到td指定的配置表中
func (p *Parser) parseTableFields(td *Table, sheet *xlsx.Sheet) error {
	td.Fields = make([]*TableField, 0, sheet.MaxCol)
	td.FieldByName = make(map[string]int, sheet.MaxCol)

	for i := 0; i < sheet.MaxCol; i++ {
		if i > 0 {
			tag, err := getSheetValue(sheet, tableRowFieldTag, i, true)
			if err != nil {
				return pkg_errors.WithMessagef(err, "get coll[%d] tag cell", i)
			}
			if !checkTagValid(tag) {
				return fmt.Errorf("col[%d] tag(%s) invalid", i, tag)
			}
			if !p.checkTagNeed(tag) {
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

	if col == 0 {
		if fd.Name != idExportName {
			return nil, errFirstFieldMustID
		}
		if !fd.Type.Primitive() {
			return nil, errIDNonPrimitive
		}
		fd.addRule(&RuleUnique{ExportMethod: true})
	}

	td.addField(fd)

	if col != 0 {
		fieldRule, err := getSheetValue(sheet, tableRowFieldRule, col, true)
		if err != nil {
			return nil, pkg_errors.WithMessage(err, "get rule cell")
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
	td.EntryByID = make(map[interface{}]TableEntry, entryCount)

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

		entry := make(TableEntry, len(td.Fields))

		// ID
		idValue := strings.TrimSpace(row.GetCell(tableColID).Value)
		if idValue == "" || isComment(idValue) {
			continue
		}
		fd = td.Fields[0]
		id, err = p.parseFieldValue(fd.Field, idValue)
		if err != nil {
			return pkg_errors.WithMessagef(err, "row[%d] parse ID=%s", i+1, idValue)
		}
		if id == nil {
			return fmt.Errorf("row[%d] ID empty", i+1)
		}
		if td.hasEntry(id) {
			return fmt.Errorf("row[%d] ID=%s duplicate", i+1, idValue)
		}
		entry[fd.Name] = id

		// others.
		for k := 1; k < len(td.Fields); k++ {
			fd = td.Fields[k]
			val, err = p.parseFieldValue(fd.Field, row.GetCell(fd.Col).Value)
			if err != nil {
				return pkg_errors.WithMessagef(err, "row[%d] {ID=%v}.%s={%s}", i+1, id, fd.Name, row.GetCell(fd.Col).Value)
			}
			if val != nil {
				entry[fd.Name] = val
			}
			if fd.Unique() {
				if !td.addUniqueValue(fd.Name, val) {
					return fmt.Errorf("row[%d] {ID=%v}.%s={%s} duplicate", i+1, id, fd.Name, row.GetCell(fd.Col).Value)
				}
			}
		}

		td.addEntry(id, entry)
	}

	return nil
}
