package gexcels

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	pkg_errors "github.com/pkg/errors"
	"github.com/tealeg/xlsx/v3"
)

const (
	structColTag    = 0                 // tag列
	structColName   = 1                 // 名称列
	structColFields = 2                 // 字段列
	structColRule   = 3                 // 规则列
	structColDesc   = 4                 // 描述
	structTableCols = structColDesc + 1 // 列数量

	structSkipRows = 1
)

// Struct 结构体
type Struct struct {
	Name        string         // 名称
	Desc        string         // 描述
	Fields      []*Field       // 字段
	FieldByName map[string]int // 字段名称到字段映射
	ReflectType reflect.Type   // 反射类型
	ExportName  string         // 导出名
}

func (sd *Struct) getFieldByName(name string) *Field {
	if i, ok := sd.FieldByName[name]; ok {
		return sd.Fields[i]
	} else {
		return nil
	}
}

func (p *Parser) addStruct(sd *Struct) {
	p.structs = append(p.structs, sd)
	p.structByName[sd.Name] = len(p.structs) - 1
}

func (p *Parser) getStruct(name string) *Struct {
	if i, ok := p.structByName[name]; ok {
		return p.structs[i]
	} else {
		return nil
	}
}

func (p *Parser) hasStruct(name string) bool {
	_, ok := p.structByName[name]
	return ok
}

// parseStructs 解析结构体定义
func (p *Parser) parseStructs() error {
	if p.structFilePath == "" {
		return nil
	}

	file, err := xlsx.OpenFile(p.structFilePath)
	if err != nil {
		return err
	}

	for _, sheet := range file.Sheets {
		if err := p.parseStructOfSheet(sheet); err != nil {
			return pkg_errors.WithMessagef(err, "struct file(%s).sheet(%s)", p.structFilePath, sheet.Name)
		}
	}
	return nil
}

// parseStructOfSheet 解析sheet中定义的结构体
func (p *Parser) parseStructOfSheet(sheet *xlsx.Sheet) error {
	if sheet.MaxRow < structSkipRows || sheet.MaxCol < structTableCols {
		return errSheetRowsOrColsNotMatch
	}
	for i := structSkipRows; i < sheet.MaxRow; i++ {
		row, err := sheet.Row(i)
		if err != nil {
			return pkg_errors.WithMessagef(err, "row[%d]", i)
		}
		rowTag := strings.TrimSpace(row.GetCell(structColTag).Value)
		if !checkTagValid(rowTag) {
			return fmt.Errorf("row[%d] tag(%s) invalid", i, rowTag)
		}
		if !p.checkTagNeed(rowTag) {
			continue
		}
		sd, err := p.parseStructRow(row)
		if err != nil {
			return pkg_errors.WithMessagef(err, "row[%d] %s", i, row.GetCell(structColName).Value)
		}
		if p.hasStruct(sd.Name) {
			return fmt.Errorf("struct %s already exist", sd.Name)
		}
		p.addStruct(sd)
	}

	return nil
}

// parseStructRow 解析row中定义的结构体
func (p *Parser) parseStructRow(row *xlsx.Row) (*Struct, error) {
	sdName := strings.TrimSpace(row.GetCell(structColName).Value)
	if sdName == "" || isComment(sdName) {
		return nil, nil
	}
	if !structNameRegexp.MatchString(sdName) {
		return nil, fmt.Errorf("struct name %s invalid", sdName)
	}

	sd := &Struct{
		Name:       sdName,
		ExportName: exportStructName(sdName),
	}

	if err := p.parseStructFields(sd, strings.TrimSpace(row.GetCell(structColFields).Value)); err != nil {
		return nil, pkg_errors.WithMessagef(err, "struct %s fields", sd.Name)
	}

	if err := p.parseStructRule(sd, row.GetCell(structColRule).Value); err != nil {
		return nil, err
	}

	sd.Desc = row.GetCell(structColDesc).Value

	reflectStructFields := make([]reflect.StructField, len(sd.Fields))
	for i, fd := range sd.Fields {
		fdReflectType, err := p.getFieldReflectType(fd)
		if err != nil {
			return nil, nil
		}
		reflectStructFields[i] = reflect.StructField{
			Name: fd.ExportName,
			Type: fdReflectType,
			Tag:  reflect.StructTag(genGoStructFieldJsonTag(fd.Name)),
		}
	}
	sd.ReflectType = reflect.StructOf(reflectStructFields)

	return sd, nil
}

// parseStructFields 解析所有结构体字段
func (p *Parser) parseStructFields(sd *Struct, fields string) error {
	if !structFieldsCheckRegexp.MatchString(fields) {
		return fmt.Errorf("fields (%s) invalid", fields)
	}

	matchFields := structFieldRegexp.FindAllStringSubmatch(fields, -1)
	if len(matchFields) <= 0 {
		return fmt.Errorf("fields (%s) invalid", fields)
	}

	sd.Fields = make([]*Field, 0, len(matchFields))
	sd.FieldByName = make(map[string]int, len(matchFields))
	for i, field := range matchFields {
		fieldDef, err := p.parseStructField(field[1:])
		if err != nil {
			return pkg_errors.WithMessagef(err, "field[%s]", field[0])
		}
		sd.Fields = append(sd.Fields, fieldDef)
		sd.FieldByName[fieldDef.Name] = i
	}

	return nil
}

// parseStructField 解析结构体字段
func (p *Parser) parseStructField(def []string) (*Field, error) {
	if len(def) != 3 {
		return nil, errFieldDefineInvalid
	}

	fd := newField(def[0], def[2])
	if err := p.parseFieldType(fd, def[1]); err != nil {
		return nil, pkg_errors.WithMessage(err, "field type")
	}

	return fd, nil
}

// parseStructRule 解析结构体规则
func (p *Parser) parseStructRule(sd *Struct, ruleStr string) error {
	ruleStr = strings.TrimSpace(ruleStr)
	if ruleStr == "" {
		return nil
	}

	rules := strings.Split(ruleStr, ruleSep)
	for _, rule := range rules {
		rule = strings.TrimSpace(rule)
		matches := ruleRegexp.FindStringSubmatch(rule)
		if len(matches) != 3 {
			return errRuleDefineInvalid(rule)
		}

		ruleType := strings.ToLower(matches[1])
		switch ruleType {
		case fieldRuleLink:
			values := structRuleLinkRegexp.FindStringSubmatch(matches[2])
			if len(values) != 4 {
				return fmt.Errorf("rule link value (%s) invalid", matches[2])
			}
			localFieldName := values[1]
			localFd := sd.getFieldByName(localFieldName)
			if localFd == nil {
				return fmt.Errorf("rule link local field[%s] not found", localFieldName)
			}
			if !localFd.Type.Primitive() && !(localFd.Type == FTArray && localFd.ElementType.Primitive()) {
				return fmt.Errorf("rule link local field[%s] must be primitive", localFieldName)
			}
			rule := &RuleLink{
				DstTable: values[2],
				DstField: values[3],
			}
			if !localFd.addRule(rule) {
				return fmt.Errorf("multiple rule link on field[%s]", localFieldName)
			}
		default:
			return errRuleTypeInvalid(matches[1])
		}
	}

	return nil
}

// parseStructFieldValue 解析结构体字段值
func (p *Parser) parseStructFieldValue(fd *Field, s string) (interface{}, error) {
	if s == "" {
		return nil, nil
	}

	sd := p.getStruct(fd.StructName)
	if sd == nil {
		return nil, errStructNotDefine(fd.StructName)
	}

	val := reflect.New(sd.ReflectType).Interface()
	if err := json.Unmarshal(([]byte)(s), val); err != nil {
		return nil, err
	}

	return val, nil
}
