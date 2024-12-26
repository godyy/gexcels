package parse

import (
	"encoding/json"
	"fmt"
	"github.com/godyy/gexcels"
	"github.com/godyy/gexcels/internal/utils"
	pkg_errors "github.com/pkg/errors"
	"github.com/tealeg/xlsx/v3"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// structFieldsRegexp 结构体字段定义正则表达式
var structFieldsRegexp = regexp.MustCompile(`^` + gexcels.NamePattern + `:(?:\[\])?` + gexcels.NamePattern + `(?::"[^"]*")?(?:\s*,\s*(` + gexcels.NamePattern + `:(?:\[\])?` + gexcels.NamePattern + `(?::"[^"]*")?))*$`)

// structFieldRegexp 结构体单字段正则表达式
var structFieldRegexp = regexp.MustCompile(`(` + gexcels.NamePattern + `):((?:\[\])?` + gexcels.NamePattern + `)(?::"([^"]*)")?`)

// Struct 结构体
type Struct struct {
	*gexcels.Struct              // 基础信息
	ReflectType     reflect.Type // 反射类型
}

// newStruct 创建结构体
func newStruct(name, desc string) *Struct {
	return &Struct{
		Struct: gexcels.NewStruct(name, desc),
	}
}

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

// structSheetNameRegexp 结构体sheet名匹配正则表达式
var structSheetNameRegexp = regexp.MustCompile(`^(.*)\|(Struct\w*)(?:\.([0-9]*))?`)

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
		if err := p.parseStructSheet(sheet.Sheet); err != nil {
			return pkg_errors.WithMessagef(err, "struct file(%s).sheet(%s)", info.path, sheet.Name)
		}
	}
	return nil
}

// structSheetInfo 结构体sheet信息
type structSheetInfo struct {
	*xlsx.Sheet     // sheet
	priority    int // 优先级
}

// parseStructSheet 解析sheet中定义的结构体
func (p *Parser) parseStructSheet(sheet *xlsx.Sheet) error {
	if sheet.MaxRow < gexcels.TableStructFirstRow || sheet.MaxCol < gexcels.TableStructCols {
		return errSheetRowsOrColsNotMatch
	}
	for i := gexcels.TableStructFirstRow; i < sheet.MaxRow; i++ {
		row, err := sheet.Row(i)
		if err != nil {
			return pkg_errors.WithMessagef(err, "row[%d]", i)
		}

		if isRowComment(row) {
			continue
		}

		rowTag := strings.TrimSpace(row.GetCell(gexcels.TableStructColTag).Value)
		if ok, valid := p.checkTag(rowTag); !valid {
			return fmt.Errorf("row[%d] tag(%s) invalid", i, rowTag)
		} else if !ok {
			continue
		}

		sd, err := p.parseStructRow(row)
		if err != nil {
			return pkg_errors.WithMessagef(err, "row[%d] %s", i, row.GetCell(gexcels.TableStructColName).Value)
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
	sdName := strings.TrimSpace(row.GetCell(gexcels.TableStructColName).Value)
	if sdName == "" {
		return nil, nil
	}
	if !gexcels.MatchName(sdName) {
		return nil, fmt.Errorf("struct name %s invalid", sdName)
	}

	sdDesc := row.GetCell(gexcels.TableStructColDesc).Value
	sd := newStruct(sdName, sdDesc)

	if err := p.parseStructFields(sd, strings.TrimSpace(row.GetCell(gexcels.TableStructColFields).Value)); err != nil {
		return nil, pkg_errors.WithMessagef(err, " struct %s fields", sd.Name)
	}

	if err := p.parseStructRule(sd, row.GetCell(gexcels.TableStructColRule).Value); err != nil {
		return nil, err
	}

	reflectStructFields := make([]reflect.StructField, len(sd.Fields))
	for i, fd := range sd.Fields {
		fdReflectType, err := p.getFieldReflectType(fd)
		if err != nil {
			return nil, nil
		}
		reflectStructFields[i] = reflect.StructField{
			Name: utils.CamelCase(fd.Name, true),
			Type: fdReflectType,
			Tag:  reflect.StructTag(genStructFieldJsonTag(fd)),
		}
	}
	sd.ReflectType = reflect.StructOf(reflectStructFields)

	return sd, nil
}

// parseStructFields 解析所有结构体字段
func (p *Parser) parseStructFields(sd *Struct, fields string) error {
	if !structFieldsRegexp.MatchString(fields) {
		return fmt.Errorf("fields (%s) invalid", fields)
	}

	matchFields := structFieldRegexp.FindAllStringSubmatch(fields, -1)
	if len(matchFields) <= 0 {
		return fmt.Errorf("fields (%s) invalid", fields)
	}

	sd.Fields = make([]*gexcels.Field, 0, len(matchFields))
	sd.FieldByName = make(map[string]*gexcels.Field, len(matchFields))
	for _, field := range matchFields {
		fieldDef, err := p.parseStructField(field[1:])
		if err != nil {
			return pkg_errors.WithMessagef(err, "field[%s]", field[0])
		}
		if sd.FieldByName[fieldDef.Name] != nil {
			return errFieldNameDuplicate(fieldDef.Name)
		}
		sd.Fields = append(sd.Fields, fieldDef)
		sd.FieldByName[fieldDef.Name] = fieldDef
	}

	return nil
}

// parseStructField 解析结构体字段
func (p *Parser) parseStructField(def []string) (*gexcels.Field, error) {
	if len(def) != 3 {
		return nil, errStructFieldDefineInvalid
	}

	fdTypeInfo, err := p.parseFieldTypeInfo(def[1])
	if err != nil {
		return nil, err
	}

	fd := gexcels.NewField(def[0], def[2], fdTypeInfo)
	return fd, nil
}

// parseStructFieldValue 解析结构体字段值
func (p *Parser) parseStructFieldValue(fd *gexcels.Field, s string) (interface{}, error) {
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

// structRuleLinkRegexp 结构体字段链接规则匹配正则表达式
var structRuleLinkRegexp = regexp.MustCompile(`^(` + gexcels.NamePattern + `),(` + gexcels.NamePattern + `).(` + gexcels.NamePattern + `)$`)

// parseStructRule 解析结构体规则
func (p *Parser) parseStructRule(sd *Struct, ruleStr string) error {
	ruleStr = strings.TrimSpace(ruleStr)
	if ruleStr == "" {
		return nil
	}

	rules := strings.Split(ruleStr, p.options.FieldRuleSep)
	for _, rule := range rules {
		rule = strings.TrimSpace(rule)

		matches := gexcels.FRRegexp.FindStringSubmatch(rule)
		if len(matches) != 3 {
			return errFieldRuleDefineInvalid(rule)
		}

		switch strings.ToUpper(matches[1]) {
		case gexcels.FRNLink:
			return p.parseStructRuleLink(sd, matches[2])
		default:
			return errFieldRuleInvalid(matches[1])
		}
	}

	return nil
}

// parseStructRuleLink 解析结构体链接规则
func (p *Parser) parseStructRuleLink(sd *Struct, value string) error {
	values := structRuleLinkRegexp.FindStringSubmatch(value)
	if len(values) != 4 {
		return fmt.Errorf("FRLink value (%s) invalid", value)
	}
	localFieldName := values[1]
	localFd := sd.GetFieldByName(localFieldName)
	if localFd == nil {
		return fmt.Errorf("FRLink local field[%s] not found", localFieldName)
	}
	if !localFd.Type.Primitive() && !(localFd.Type == gexcels.FTArray && localFd.ElementType.Primitive()) {
		return fmt.Errorf("FRLink local field[%s] must be primitive", localFieldName)
	}
	rule := gexcels.NewFRLink(values[2], values[3])
	if !localFd.AddRule(rule) {
		return fmt.Errorf("multiple FRLink on field[%s]", localFieldName)
	}
	return nil
}

// genStructFieldJsonTag 生成结构体字段的json tag
func genStructFieldJsonTag(fd *gexcels.Field) string {
	return "`json:\"" + fd.Name + ",omitempty\"`"
}
