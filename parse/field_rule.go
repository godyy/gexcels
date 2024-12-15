package parse

// addRule 添加规则
// 每个类型的规则只能存在一个
func (f *Field) addRule(r fieldRule) bool {
	if f.rules == nil {
		f.rules = make(map[fieldRuleType]fieldRule)
	}

	if _, ok := f.rules[r.ruleType()]; ok {
		return false
	} else {
		f.rules[r.ruleType()] = r
		return true
	}
}

// getRule 按照规则类型获取规则
func (f *Field) getRule(ruleType fieldRuleType) fieldRule {
	if len(f.rules) <= 0 {
		return nil
	}

	return f.rules[ruleType]
}

// getCreateRule 获取或创建规则
func (f *Field) getCreateRule(ruleType fieldRuleType) fieldRule {
	r := f.getRule(ruleType)
	if r != nil {
		return r
	}
	r = createFieldRuleByType(ruleType)
	f.addRule(r)
	return r
}

// Unique 返回字段是否具备Unique规则
func (f *Field) Unique() bool {
	return convertFieldRule[*fieldUnique](f.getRule(fieldRuleUnique)) != nil
}

// ExportUniqueMethod 返回字段是否具备Unique规则且需要导出方法
func (f *Field) ExportUniqueMethod() bool {
	ruleUnique := convertFieldRule[*fieldUnique](f.getRule(fieldRuleUnique))
	return ruleUnique != nil && ruleUnique.exportMethod
}

// fieldRuleType 字段规则类型
type fieldRuleType string

// 字段规则类型值
const (
	fieldRuleUnique       = fieldRuleType("unique") // 唯一键
	fieldRuleLink         = fieldRuleType("link")   // 外链，链接其它表的字段，该字段必须是ID，或者Unique
	fieldRuleCompositeKey = fieldRuleType("cmpkey") // 组合键
	fieldRuleGroup        = fieldRuleType("group")  // 分组 将具备相同属性的数据聚合在一起
)

func (t fieldRuleType) String() string { return string(t) }

// fieldRuleSep 字段规则分隔符
const fieldRuleSep = "|"

// fieldRule 规则接口
type fieldRule interface {
	// ruleType 规则类型
	ruleType() fieldRuleType
}

// 字段类型规则创建器
var ruleTypeCreators = map[fieldRuleType]func() fieldRule{
	fieldRuleUnique:       func() fieldRule { return &fieldUnique{} },
	fieldRuleLink:         func() fieldRule { return &fieldLink{} },
	fieldRuleCompositeKey: func() fieldRule { return &fieldCompositeKey{} },
	fieldRuleGroup:        func() fieldRule { return &fieldGroup{} },
}

// 根据规则类型创建字段规则
func createFieldRuleByType(ruleType fieldRuleType) fieldRule {
	return ruleTypeCreators[ruleType]()
}

// fieldUnique 唯一规则，表示该字段的值在配置表返回内具有唯一性
type fieldUnique struct {
	exportMethod bool // 是否导出方法
}

func (r *fieldUnique) ruleType() fieldRuleType { return fieldRuleUnique }

// fieldLink 链接规则，指向其它配置表的主键或者unique字段
type fieldLink struct {
	dstTable string // 目标配置表
	dstField string // 目标字段，ID或者unique
}

func (r *fieldLink) ruleType() fieldRuleType { return fieldRuleLink }

// fieldCompositeKey 组合键
type fieldCompositeKey struct {
	keyIndex map[string]int // [keyName]index
}

func (r *fieldCompositeKey) ruleType() fieldRuleType { return fieldRuleCompositeKey }

// addKeyIndex 添加组合键及在键内的索引
func (r *fieldCompositeKey) addKeyIndex(key string, index int) bool {
	if r.keyIndex == nil {
		r.keyIndex = make(map[string]int)
		r.keyIndex[key] = index
		return true
	} else if _, ok := r.keyIndex[key]; !ok {
		r.keyIndex[key] = index
		return true
	} else {
		return false
	}
}

// fieldGroup 分组
type fieldGroup struct {
	groupIndex map[string]int // [groupName]int
}

func (r *fieldGroup) ruleType() fieldRuleType { return fieldRuleGroup }

// addGroupIndex 添加分组及在分组内的索引
func (r *fieldGroup) addGroupIndex(group string, index int) bool {
	if r.groupIndex == nil {
		r.groupIndex = make(map[string]int)
		r.groupIndex[group] = index
		return true
	} else if _, ok := r.groupIndex[group]; !ok {
		r.groupIndex[group] = index
		return true
	} else {
		return false
	}
}

// checkFieldTypeOnLink 检查两个链接的字段类型是否一致
func checkFieldTypeOnLink(srcField, dstField *Field) bool {
	srcFieldType := srcField.Type
	if srcFieldType == FTArray {
		srcFieldType = srcField.ElementType
	}
	return srcFieldType == dstField.Type
}

// convertFieldRule 转换规则到指定类型
func convertFieldRule[Rule fieldRule](r fieldRule) (rr Rule) {
	if r == nil {
		return
	}
	if v, ok := r.(Rule); ok {
		rr = v
	}
	return
}
