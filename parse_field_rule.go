package gexcels

const (
	fieldRuleUnique       = "unique"  // 唯一键
	fieldRuleLink         = "link"    // 外链，链接其它表的字段，该字段必须是ID，或者Unique
	fieldRuleCompositeKey = "compkey" // 组合键

	ruleSep = "|"
)

// rule 规则接口
type rule interface {
	// ruleType 规则类型
	ruleType() string
}

// 类型规则创建器
var ruleTypeCreators = map[string]func() rule{
	fieldRuleUnique:       func() rule { return &RuleUnique{} },
	fieldRuleLink:         func() rule { return &RuleLink{} },
	fieldRuleCompositeKey: func() rule { return &RuleCompositeKey{} },
}

// 根据规则类型创建规则
func createRuleByType(ruleType string) rule {
	return ruleTypeCreators[ruleType]()
}

// RuleUnique 唯一规则，表示该字段的值在配置表返回内具有唯一性
type RuleUnique struct {
	ExportMethod bool
}

func (r *RuleUnique) ruleType() string { return fieldRuleUnique }

// RuleLink 链接规则，指向其它配置表的主键或者unique字段
type RuleLink struct {
	DstTable string // 目标配置表
	DstField string // 目标字段，ID或者unique
}

func (r *RuleLink) ruleType() string { return fieldRuleLink }

// RuleCompositeKey 组合键
type RuleCompositeKey struct {
	KeyIndex map[string]int // [keyName]index
}

func (r *RuleCompositeKey) ruleType() string { return fieldRuleCompositeKey }

func (r *RuleCompositeKey) addKeyIndex(key string, index int) bool {
	if r.KeyIndex == nil {
		r.KeyIndex = make(map[string]int)
		r.KeyIndex[key] = index
		return true
	} else if _, ok := r.KeyIndex[key]; !ok {
		r.KeyIndex[key] = index
		return true
	} else {
		return false
	}
}

func (f *Field) addRule(r rule) bool {
	if f.rules == nil {
		f.rules = make(map[string]rule)
	}

	if _, ok := f.rules[r.ruleType()]; ok {
		return false
	} else {
		f.rules[r.ruleType()] = r
		return true
	}
}

func (f *Field) getRule(ruleType string) rule {
	if len(f.rules) <= 0 {
		return nil
	}

	return f.rules[ruleType]
}

func (f *Field) Unique() bool {
	return convertRule[*RuleUnique](f.getRule(fieldRuleUnique)) != nil
}

func (f *Field) ExportUniqueMethod() bool {
	ruleUnique := convertRule[*RuleUnique](f.getRule(fieldRuleUnique))
	return ruleUnique != nil && ruleUnique.ExportMethod
}

// checkFieldLinkType 检查字段的类型是否一致
func checkFieldLinkType(srcField, dstField *Field) bool {
	srcFieldType := srcField.Type
	if srcFieldType == FTArray {
		srcFieldType = srcField.ElementType
	}
	return srcFieldType == dstField.Type
}

// convertRule 转换规则到指定类型
func convertRule[Rule rule](r rule) (rr Rule) {
	if r == nil {
		return
	}
	if v, ok := r.(Rule); ok {
		rr = v
	}
	return
}

// getCreateFieldRule 获取或创建字段规则
func getCreateFieldRule(fd *Field, ruleType string) rule {
	r := fd.getRule(ruleType)
	if r != nil {
		return r
	}
	r = createRuleByType(ruleType)
	fd.addRule(r)
	return r
}
