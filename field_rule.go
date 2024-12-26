package gexcels

import (
	"fmt"
	"github.com/pkg/errors"
	"regexp"
	"strconv"
	"strings"
)

// FRNameValueSep 字段规则名称和值分隔符
const FRNameValueSep = "="

// FRValueSep 字段规则值分隔符
const FRValueSep = ","

// 字段规则名称
const (
	FRNUnique       = "UNIQUE" // 唯一键
	FRNLink         = "LINK"   // 外链，链接其它表的字段，该字段必须是ID，或者Unique
	FRNCompositeKey = "CKEY"   // 组合键
	FRNGroup        = "GROUP"  // 分组 将具备相同属性的数据聚合在一起
)

// FieldRule 规则接口
type FieldRule interface {
	// FRName 规则名称
	FRName() string

	// FRKey 规则的唯一键
	FRKey() string

	// ParseValue 解析规则的值
	ParseValue(value string) error

	// String 输出字符串形式
	String() string
}

// frCreators 字段规则构造器
var frCreators = map[string]func() FieldRule{
	FRNUnique:       func() FieldRule { return &FRUnique{} },
	FRNLink:         func() FieldRule { return &FRLink{} },
	FRNCompositeKey: func() FieldRule { return &FRCompositeKey{} },
	FRNGroup:        func() FieldRule { return &FRGroup{} },
}

// FRRegexp 字段规则正则表达式
var FRRegexp = regexp.MustCompile(`^(` + NamePattern + `)(?:` + FRNameValueSep + `([\w,.]*))?$`)

// ParseFieldRule 解析字段规则
func ParseFieldRule(s string) (FieldRule, error) {
	matches := FRRegexp.FindStringSubmatch(s)
	if len(matches) != 3 {
		return nil, fmt.Errorf("gexcels: field-rule %s invalid", s)
	}

	frName := strings.ToUpper(matches[1])
	frValue := matches[2]

	creator := frCreators[frName]
	if creator == nil {
		return nil, fmt.Errorf("gexcels: field-rule %s not exist", frName)
	}

	fr := creator()
	if err := fr.ParseValue(frValue); err != nil {
		return nil, err
	}

	return fr, nil
}

// errFRValueInvalid 生成表示字段规则值无效的错误
func errFRValueInvalid(ruleName string, value string, eg string) error {
	return fmt.Errorf("gexcels: field-rule %s value %s invalid, .e.g %s", ruleName, value, eg)
}

// FRUnique 唯一规则，表示该字段的值在配置表返回内具有唯一性
// .e.g: unique
type FRUnique struct{}

// NewFRUnique 创建唯一规则
func NewFRUnique() *FRUnique {
	return &FRUnique{}
}

func (r *FRUnique) FRName() string { return FRNUnique }

func (r *FRUnique) FRKey() string { return FRNUnique }

// ErrFRUniqueMustNoValue 指定 FRUnique 规则不能具备value的错误
var ErrFRUniqueMustNoValue = errors.New("gexcels: field-rule unique must no value")

func (r *FRUnique) ParseValue(value string) error {
	if value != "" {
		return ErrFRUniqueMustNoValue
	}
	return nil
}

func (r *FRUnique) String() string { return FRNUnique }

// FRLink 链接规则，指向其它配置表的主键或者unique字段
// .e.g: link=tableName.fieldName
type FRLink struct {
	TableName string // 目标配置表
	FieldName string // 目标字段，ID或者unique
}

// frLinkValueRegexp 字段链接规则值正则表达式
var frLinkValueRegexp = regexp.MustCompile(`^(` + NamePattern + `)\.(` + NamePattern + `)$`)

// NewFRLink 创建链接规则
func NewFRLink(tableName, fieldName string) *FRLink {
	if !MatchName(tableName) {
		panic("gexcels: NewFRLink: tableName " + tableName + " invalid")
	}
	if !MatchName(fieldName) {
		panic("gexcels: NewFRLink: fieldName " + fieldName + " invalid")
	}
	return &FRLink{
		TableName: tableName,
		FieldName: fieldName,
	}
}

func (r *FRLink) FRName() string { return FRNLink }

func (r *FRLink) FRKey() string {
	return FRNLink
}

func (r *FRLink) ParseValue(value string) error {
	matches := frLinkValueRegexp.FindStringSubmatch(value)
	if len(matches) != 3 {
		return errFRValueInvalid(FRNLink, value, "tableName.fieldName")
	}

	r.TableName = matches[1]
	r.FieldName = matches[2]
	return nil
}

func (r *FRLink) String() string {
	return FRNLink + FRNameValueSep + r.TableName + "." + r.FieldName
}

// FRCompositeKey 组合键，表示该字段属于某个组合键的一部分
// .e.g: composite-key=keyName,index
type FRCompositeKey struct {
	KeyName string // 组合键名称
	Index   int    // 索引，字段在组合键中的顺位
}

// frCompositeKeyValueRegexp 字段组合键规则值正则表达式
var frCompositeKeyValueRegexp = regexp.MustCompile(`^(` + NamePattern + `),(\d)$`)

// NewFRCompositeKey 创建组合键规则
func NewFRCompositeKey(keyName string, index int) *FRCompositeKey {
	if !MatchName(keyName) {
		panic("gexcels: NewFRCompositeKey: keyName " + keyName + " invalid")
	}
	if index < 0 || index > 9 {
		panic("gexcels: NewFRCompositeKey: index must be in range [0,9]")
	}
	return &FRCompositeKey{
		KeyName: keyName,
		Index:   index,
	}
}

func (r *FRCompositeKey) FRName() string { return FRNCompositeKey }

func (r *FRCompositeKey) FRKey() string {
	return FRNCompositeKey + ":" + r.KeyName
}

func (r *FRCompositeKey) ParseValue(value string) error {
	matches := frCompositeKeyValueRegexp.FindStringSubmatch(value)
	if len(matches) != 3 {
		return errFRValueInvalid(FRNCompositeKey, value, "keyName,index")
	}

	r.KeyName = matches[1]
	r.Index, _ = strconv.Atoi(matches[2])
	return nil
}

func (r *FRCompositeKey) String() string {
	return FRNCompositeKey + FRNameValueSep + r.KeyName + FRValueSep + strconv.Itoa(r.Index)
}

// FRGroup 分组,表示字段为某个分组的一部分
// .e.g: group=groupName,index
type FRGroup struct {
	GroupName string // 分组名称
	Index     int    // 索引，在分组中的顺位
}

// frGroupValueRegexp 字段组合键规则值正则表达式
var frGroupValueRegexp = regexp.MustCompile(`^(` + NamePattern + `),(\d)$`)

// NewFRGroup 创建分组规则
func NewFRGroup(groupName string, index int) *FRGroup {
	if !MatchName(groupName) {
		panic("gexcels: NewFRGroup: groupName " + groupName + " invalid")
	}

	if index < 0 || index > 9 {
		panic("gexcels: NewFRGroup: index must be in range [0,9]")
	}

	return &FRGroup{
		GroupName: groupName,
		Index:     index,
	}
}

func (r *FRGroup) FRName() string { return FRNGroup }

func (r *FRGroup) FRKey() string {
	return FRNGroup + ":" + r.GroupName
}

func (r *FRGroup) ParseValue(value string) error {
	matches := frGroupValueRegexp.FindStringSubmatch(value)
	if len(matches) != 3 {
		return errFRValueInvalid(FRNGroup, value, "groupName,index")
	}

	r.GroupName = matches[1]
	r.Index, _ = strconv.Atoi(matches[2])
	return nil
}

func (r *FRGroup) String() string {
	return FRNGroup + FRNameValueSep + r.GroupName + FRValueSep + strconv.Itoa(r.Index)
}
