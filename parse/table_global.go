package parse

import (
	"fmt"
	pkg_errors "github.com/pkg/errors"
	"github.com/tealeg/xlsx/v3"
	"strings"
)

const (
	globalSkipRows  = 1 // 解析global配置表字段时需要跳过的行数
	globalTableCols = 5 // global配置表列数

	// global 配置表列号
	globalColFieldTag   = 0 // Tag
	globalColFieldName  = 1 // 字段名
	globalColFieldType  = 2 // 字段类型
	globalColFieldValue = 3 // 字段值
	globalColFieldRule  = 4 // 字段规则
	globalColFieldDesc  = 5 // 字段描述
)

// parseGlobalTable 解析sheet中的global配置表到td
func (p *Parser) parseGlobalTable(sheet *xlsx.Sheet, tag Tag, name, desc string) (*Table, error) {
	if sheet.MaxRow < globalSkipRows || sheet.MaxCol < globalTableCols {
		return nil, errSheetRowsOrColsNotMatch
	}

	td := newTable(tag, name, desc, true)
	fieldCount := sheet.MaxRow - globalSkipRows
	td.Fields = make([]*TableField, 0, fieldCount)
	td.fieldByName = make(map[string]int, fieldCount)
	td.entryByName = make(map[string]interface{}, fieldCount)

	for i := globalSkipRows; i < sheet.MaxRow; i++ {
		row, err := sheet.Row(i)
		if err != nil {
			return nil, pkg_errors.WithMessagef(err, "get row[%d]", i+1)
		}

		if isRowComment(row) {
			continue
		}

		rowTag := strings.TrimSpace(row.GetCell(globalColFieldTag).Value)
		if ok, valid := p.checkTag(rowTag); !valid {
			return nil, fmt.Errorf("row[%d] tag(%s) invalid", i+1, rowTag)
		} else if !ok {
			continue
		}

		if err := p.parseGlobalTableField(td, row); err != nil {
			return nil, pkg_errors.WithMessagef(err, "field {%s}", row.GetCell(globalColFieldName).Value)
		}
	}
	return td, nil
}

// parseGlobalTableField 解析row定义的字段到global配置表td
func (p *Parser) parseGlobalTableField(td *Table, row *xlsx.Row) error {
	fieldName := strings.TrimSpace(row.GetCell(globalColFieldName).Value)
	if fieldName == "" {
		return nil
	}

	if !fieldNameRegexp.MatchString(fieldName) {
		return errFieldNameInvalid(fieldName)
	}

	if td.hasField(fieldName) {
		return errFieldNameDuplicate(fieldName)
	}

	fd := &TableField{
		Field: newField(fieldName, row.GetCell(globalColFieldDesc).Value),
	}

	if err := p.parseFieldType(fd.Field, row.GetCell(globalColFieldType).Value); err != nil {
		return err
	}

	val, err := p.parseFieldValue(fd.Field, row.GetCell(globalColFieldValue).Value)
	if err != nil {
		return err
	}

	if err := p.parseTableFieldRules(td, fd, row.GetCell(globalColFieldRule).Value); err != nil {
		return err
	}

	td.addField(fd)
	td.addEntryByName(fd.Name, val)

	return nil
}