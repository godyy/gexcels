package parse

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/godyy/gexcels"
	pkg_errors "github.com/pkg/errors"
	"github.com/tealeg/xlsx/v3"
)

// Struct 结构体
type Struct struct {
	*gexcels.Struct // 基础信息
}

// newStruct 创建结构体
func newStruct(name, desc string) *Struct {
	return &Struct{
		Struct: gexcels.NewStruct(name, desc),
	}
}

// addStruct 添加结构体
func (p *Parser) addStruct(sd *Struct) error {
	if err := p.addCustomFieldType(gexcels.NewStructFieldTypeInfo(sd.Name)); err != nil {
		return err
	}
	p.Structs = append(p.Structs, sd)
	p.structByName[sd.Name] = sd
	return nil
}

// GetStructByName 根据名称获取结构体
func (p *Parser) GetStructByName(name string) *Struct {
	return p.structByName[name]
}

// parseStructs 解析结构体定义
func (p *Parser) parseStructs(files []string) error {
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
func (p *Parser) parseStructFile(path string) error {
	file, err := xlsx.OpenFile(path)
	if err != nil {
		return err
	}

	sheets := make([]*prioritySheetInfo, 0, len(file.Sheets))
	for _, sheet := range file.Sheets {
		matches := structSheetNameRegexp.FindStringSubmatch(sheet.Name)
		if len(matches) != 4 {
			continue
		}

		priority, _ := strconv.Atoi(matches[3])
		sheets = append(sheets, &prioritySheetInfo{
			Sheet:    sheet,
			priority: priority,
		})
	}

	sort.SliceStable(sheets, func(i, j int) bool {
		return sheets[i].priority < sheets[j].priority
	})

	for _, sheet := range sheets {
		if err := p.parseStructSheet(sheet.Sheet); err != nil {
			return pkg_errors.WithMessagef(err, "struct file(%s).sheet(%s)", path, sheet.Name)
		}
	}
	return nil
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
		if err := p.addStruct(sd); err != nil {
			return pkg_errors.WithMessagef(err, "add struct %s", sd.Name)
		}
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

	return sd, nil
}

// parseStructFields 解析所有结构体字段
func (p *Parser) parseStructFields(sd *Struct, fields string) error {
	fields = strings.TrimSpace(fields)
	if fields == "" {
		return fmt.Errorf("fields empty")
	}

	// 字段定义字符串以前通过正则限制类型表达式形态，但会阻碍 map 等复杂类型。
	// 这里改为分割 + 逐字段解析，保证类型表达式可扩展（数组/map/自定义类型等）。
	fieldTokens, err := splitStructFieldTokens(fields)
	if err != nil {
		return pkg_errors.WithMessagef(err, "fields (%s) invalid", fields)
	}
	if len(fieldTokens) > gexcels.StructMaxField {
		return fmt.Errorf("field number %d exceed limit, max %d", len(fieldTokens), gexcels.StructMaxField)
	}

	sd.Fields = make([]*gexcels.Field, 0, len(fieldTokens))
	sd.FieldByName = make(map[string]*gexcels.Field, len(fieldTokens))
	for _, token := range fieldTokens {
		fieldDef, err := p.parseStructFieldToken(token)
		if err != nil {
			return pkg_errors.WithMessagef(err, "field[%s]", token)
		}
		if !sd.AddField(fieldDef) {
			return errFieldNameDuplicate(fieldDef.Name)
		}
	}

	return nil
}

// splitStructFieldTokens 按逗号分割字段定义，但需要跳过 desc 的引号内容。
// e.g: A:int32:"a,b",MM:map[string]int32:"m"
func splitStructFieldTokens(s string) ([]string, error) {
	var (
		out     []string
		inQuote bool
		start   int
	)
	for i := 0; i < len(s); i++ {
		if s[i] == '"' {
			inQuote = !inQuote
			continue
		}
		if s[i] == ',' && !inQuote {
			token := strings.TrimSpace(s[start:i])
			if token == "" {
				return nil, errStructFieldDefineInvalid
			}
			out = append(out, token)
			start = i + 1
		}
	}
	last := strings.TrimSpace(s[start:])
	if last == "" {
		return nil, errStructFieldDefineInvalid
	}
	out = append(out, last)
	return out, nil
}

// parseStructFieldToken 解析结构体字段定义
func (p *Parser) parseStructFieldToken(token string) (*gexcels.Field, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil, errStructFieldDefineInvalid
	}

	// token 形如：Name:Type 或 Name:Type:"Desc"
	sep := strings.IndexByte(token, ':')
	if sep <= 0 {
		return nil, errStructFieldDefineInvalid
	}
	name := strings.TrimSpace(token[:sep])
	if !gexcels.MatchName(name) {
		return nil, errStructFieldDefineInvalid
	}

	rest := strings.TrimSpace(token[sep+1:])
	if rest == "" {
		return nil, errStructFieldDefineInvalid
	}

	typeStr := rest
	desc := ""
	if i := strings.Index(rest, `:"`); i >= 0 {
		typeStr = strings.TrimSpace(rest[:i])
		descPart := rest[i+2:]
		if !strings.HasSuffix(descPart, `"`) {
			return nil, errStructFieldDefineInvalid
		}
		desc = descPart[:len(descPart)-1]
	}
	if typeStr == "" {
		return nil, errStructFieldDefineInvalid
	}

	fdTypeInfo, err := p.parseFieldTypeInfo(typeStr)
	if err != nil {
		return nil, err
	}
	return gexcels.NewField(name, desc, fdTypeInfo), nil
}

// parseStructFieldValue 解析结构体字段值
func (p *Parser) parseStructFieldValue(fd *gexcels.Field, s string) (any, error) {
	if s == "" {
		return nil, nil
	}

	sd := p.GetStructByName(fd.GetName())
	if sd == nil {
		return nil, errStructNotDefine(fd.GetName())
	}

	var raw any
	if err := json.Unmarshal(([]byte)(s), &raw); err != nil {
		return nil, err
	}

	return p.convertJSONStructValue(fd.FieldTypeInfo, raw, fd.Name)
}

// structRuleLinkRegexp 结构体字段链接规则匹配正则表达式
// 格式：LocalField,<LinkValue>
// 其中 <LinkValue> 与表字段规则 LINK 的 value 部分一致（见 gexcels.FRLink.ParseValue）。
var structRuleLinkRegexp = regexp.MustCompile(`^(` + gexcels.NamePattern + `),(.+)$`)

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
	if len(values) != 3 {
		return fmt.Errorf("FRLink value (%s) invalid", value)
	}
	localFieldName := values[1]
	localFd := sd.GetFieldByName(localFieldName)
	if localFd == nil {
		return fmt.Errorf("FRLink local field[%s] not found", localFieldName)
	}

	rule := &gexcels.FRLink{}
	if err := rule.ParseValue(values[2]); err != nil {
		return err
	}
	if !localFd.AddRule(rule) {
		return fmt.Errorf("multiple FRLink on field[%s]", localFieldName)
	}
	return nil
}
