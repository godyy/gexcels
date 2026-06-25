package gexcels

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/pkg/errors"
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
var FRRegexp = regexp.MustCompile(`^(` + NamePattern + `)(?:` + FRNameValueSep + `([^\s]+))?$`)

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
func errFRValueInvalid(ruleName string, value string, f string, args ...any) error {
	return fmt.Errorf("gexcels: field-rule %s value %s invalid, %s", ruleName, value, fmt.Sprintf(f, args...))
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
// 格式: LINK=[Table.Field][,k1:Table.Field][,k2:Table.Field]...
//
//	Table.Field 表示字段的叶子值需要链接的表名和字段名.
//	kx:Table.Field 表示字段中的第x层级的map键需要映射到的表名和字段名，其中x>=1.
type FRLink struct {
	Value *FRLinkTarget         // 叶子值链接目标
	Keys  map[int]*FRLinkTarget // 各层级map键链接目标，key为map层级
}

// FRLinkTarget 链接规则目标
type FRLinkTarget struct {
	TableName string // 目标配置表名
	FieldName string // 目标字段名
}

// frLinkPairRegexp 字段链接规则值分段正则表达式
var frLinkPairRegexp = regexp.MustCompile(`^(?:(` + NamePattern + `):)?(` + NamePattern + `)\.(` + NamePattern + `)$`)

// NewFRLink 创建链接规则
func NewFRLink(tableName, fieldName string) *FRLink {
	if !MatchName(tableName) {
		panic("gexcels: NewFRLink: tableName " + tableName + " invalid")
	}
	if !MatchName(fieldName) {
		panic("gexcels: NewFRLink: fieldName " + fieldName + " invalid")
	}
	return &FRLink{
		Value: &FRLinkTarget{
			TableName: tableName,
			FieldName: fieldName,
		},
	}
}

func (r *FRLink) FRName() string { return FRNLink }

func (r *FRLink) FRKey() string {
	return FRNLink
}

// addKeyTarget 添加map键链接目标
func (r *FRLink) addKeyTarget(level int, target *FRLinkTarget) bool {
	if r.Keys == nil {
		r.Keys = make(map[int]*FRLinkTarget)
	}
	if _, ok := r.Keys[level]; ok {
		return false
	}
	r.Keys[level] = target
	return true
}

// parseLinkValue 解析链接规则值
func parseLinkValue(s string) (string, *FRLinkTarget, bool) {
	s = strings.TrimSpace(s)
	matches := frLinkPairRegexp.FindStringSubmatch(s)
	if len(matches) < 3 {
		return "", nil, false
	}
	if len(matches) > 3 {
		return strings.ToLower(matches[1]), &FRLinkTarget{TableName: matches[2], FieldName: matches[3]}, true
	} else {
		return "", &FRLinkTarget{TableName: matches[1], FieldName: matches[2]}, true
	}
}

func (r *FRLink) ParseValue(value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return errFRValueInvalid(FRNLink, value, "e.g. [Table.Field][,k1:Table.Field][,k2:Table.Field]...")
	}

	r.Value = nil
	r.Keys = nil
	parts := strings.Split(value, FRValueSep)
	for _, part := range parts {
		kindPrefix, target, ok := parseLinkValue(part)
		if !ok {
			return errFRValueInvalid(FRNLink, value, "invalid link pair %s", part)
		}

		switch {
		case kindPrefix == "":
			if r.Value != nil {
				return errFRValueInvalid(FRNLink, value, "e.g. [Table.Field][,k1:Table.Field][,k2:Table.Field]...")
			}
			r.Value = target
			continue
		case strings.HasPrefix(kindPrefix, "k"):
			level, err := strconv.Atoi(kindPrefix[1:])
			if err != nil {
				return errFRValueInvalid(FRNLink, value, "kx:Table.Field")
			}
			if !r.addKeyTarget(level, target) {
				return errFRValueInvalid(FRNLink, value, "duplicate k%d:Table.Field", level)
			}
		default:
			return errFRValueInvalid(FRNLink, value, "invalid kind prefix %s", kindPrefix)
		}
	}

	return nil
}

func (r *FRLink) String() string {
	var sb strings.Builder
	sb.WriteString(FRNLink)
	sb.WriteString(FRNameValueSep)

	first := true
	if r.Value != nil {
		sb.WriteString(r.Value.TableName)
		sb.WriteString(".")
		sb.WriteString(r.Value.FieldName)
		first = false
	}

	if len(r.Keys) > 0 {
		levels := make([]int, 0, len(r.Keys))
		for level := range r.Keys {
			levels = append(levels, level)
		}
		sort.Ints(levels)
		for _, level := range levels {
			if !first {
				sb.WriteString(FRValueSep)
			}
			sb.WriteString("k")
			sb.WriteString(strconv.Itoa(level))
			sb.WriteString("=")
			t := r.Keys[level]
			sb.WriteString(t.TableName)
			sb.WriteString(".")
			sb.WriteString(t.FieldName)
			first = false
		}
	}

	return sb.String()
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
		return errFRValueInvalid(FRNCompositeKey, value, "e.g. keyName,index")
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
		return errFRValueInvalid(FRNGroup, value, "e.g. groupName,index")
	}

	r.GroupName = matches[1]
	r.Index, _ = strconv.Atoi(matches[2])
	return nil
}

func (r *FRGroup) String() string {
	return FRNGroup + FRNameValueSep + r.GroupName + FRValueSep + strconv.Itoa(r.Index)
}
