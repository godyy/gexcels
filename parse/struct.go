package parse

import (
	"encoding/json"
	"fmt"
	pkg_errors "github.com/pkg/errors"
	"github.com/tealeg/xlsx/v3"
	"reflect"
	"sort"
	"strconv"
	"strings"
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

func newStruct(name, desc string) *Struct {
	return &Struct{
		Name: name,
		Desc: desc,
	}
}

// getFieldByName 根据名称获取字段
func (sd *Struct) getFieldByName(name string) *Field {
	if i, ok := sd.FieldByName[name]; ok {
		return sd.Fields[i]
	} else {
		return nil
	}
}

// 结构体表各列定义
const (
	structColTag    = 0                 // tag列
	structColName   = 1                 // 名称列
	structColFields = 2                 // 字段列
	structColRule   = 3                 // 规则列
	structColDesc   = 4                 // 描述
	structTableCols = structColDesc + 1 // 列数量

)

// 读取结构体定义时的跳过行数
const structSkipRows = 1

// addStruct 添加结构体
func (p *Parser) addStruct(sd *Struct) {
	p.Structs = append(p.Structs, sd)
	p.structByName[sd.Name] = sd
}

// hasStruct 是否存在结构体
func (p *Parser) hasStruct(name string) bool {
	_, ok := p.structByName[name]
	return ok
}

// GetStructByName 根据名称获取结构体
func (p *Parser) GetStructByName(name string) *Struct {
	return p.structByName[name]
}

// parseStructs 解析结构体定义
func (p *Parser) parseStructs(files []*structFileInfo) error {
	for _, file := range files {
		if err := p.parseStructFile(file); err != nil {
			return err
		}
	}
	return nil
}

// structSheetInfo 结构体sheet信息
type structSheetInfo struct {
	*xlsx.Sheet     // sheet
	priority    int // 优先级
}

// parseStructFile 解析结构体定义文件
func (p *Parser) parseStructFile(info *structFileInfo) error {
	file, err := xlsx.OpenFile(info.path)
	if err != nil {
		return err
	}

	sheets := make([]*structSheetInfo, 0, len(file.Sheets))
	for _, sheet := range file.Sheets {
		matches := structSheetNameRegexp.FindStringSubmatch(sheet.Name)
		if len(matches) != 4 {
			continue
		}

		priority, _ := strconv.Atoi(matches[3])
		sheets = append(sheets, &structSheetInfo{
			Sheet:    sheet,
			priority: priority,
		})
	}

	sort.SliceStable(sheets, func(i, j int) bool {
		return sheets[i].priority < sheets[j].priority
	})

	for _, sheet := range sheets {
		if err := p.parseStructOfSheet(sheet.Sheet); err != nil {
			return pkg_errors.WithMessagef(err, "struct file(%s).sheet(%s)", info.path, sheet.Name)
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

		if isRowComment(row) {
			continue
		}

		rowTag := strings.TrimSpace(row.GetCell(structColTag).Value)
		if ok, valid := p.checkTag(rowTag); !valid {
			return fmt.Errorf("row[%d] tag(%s) invalid", i, rowTag)
		} else if !ok {
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
	if sdName == "" {
		return nil, nil
	}
	if !structNameRegexp.MatchString(sdName) {
		return nil, fmt.Errorf("struct name %s invalid", sdName)
	}

	sdDesc := row.GetCell(structColDesc).Value
	sd := newStruct(sdName, sdDesc)

	if err := p.parseStructFields(sd, strings.TrimSpace(row.GetCell(structColFields).Value)); err != nil {
		return nil, pkg_errors.WithMessagef(err, " struct %s fields", sd.Name)
	}

	if err := p.parseStructRule(sd, row.GetCell(structColRule).Value); err != nil {
		return nil, err
	}

	reflectStructFields := make([]reflect.StructField, len(sd.Fields))
	for i, fd := range sd.Fields {
		fdReflectType, err := p.getFieldReflectType(fd)
		if err != nil {
			return nil, nil
		}
		reflectStructFields[i] = reflect.StructField{
			Name: GenStructFieldName(fd.Name),
			Type: fdReflectType,
			Tag:  reflect.StructTag(genStructFieldJsonTag(fd)),
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
			return pkg_errors.WithMessagef(err, " field[%s]", field[0])
		}
		sd.Fields = append(sd.Fields, fieldDef)
		sd.FieldByName[fieldDef.Name] = i
	}

	return nil
}

// parseStructField 解析结构体字段
func (p *Parser) parseStructField(def []string) (*Field, error) {
	if len(def) != 3 {
		return nil, errStructFieldDefineInvalid
	}

	fd := newField(def[0], def[2])
	if err := p.parseFieldType(fd, def[1]); err != nil {
		return nil, pkg_errors.WithMessage(err, "field type")
	}

	return fd, nil
}

// parseStructFieldValue 解析结构体字段值
func (p *Parser) parseStructFieldValue(fd *Field, s string) (interface{}, error) {
	if s == "" {
		return nil, nil
	}

	sd := p.GetStructByName(fd.StructName)
	if sd == nil {
		return nil, errStructNotDefine(fd.StructName)
	}

	val := reflect.New(sd.ReflectType).Interface()
	if err := json.Unmarshal(([]byte)(s), val); err != nil {
		return nil, err
	}

	return val, nil
}

// genStructFieldJsonTag 生成结构体字段的json tag
func genStructFieldJsonTag(fd *Field) string {
	return "`json:\"" + fd.Name + ",omitempty\"`"
}
