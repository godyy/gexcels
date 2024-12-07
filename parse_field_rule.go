package gexcels

const (
	fieldRuleUnique = "unique" // 唯一键
	fieldRuleLink   = "link"   // 外链，链接其它表的字段，该字段必须是ID，或者Unique

	ruleSep = "|"
)

// rule 规则接口
type rule interface {
	// ruleType 规则类型
	ruleType() string
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

// checkFieldLinkType 检查字段的类型是否一致
func checkFieldLinkType(srcField, dstField *Field) bool {
	srcFieldType := srcField.Type
	if srcFieldType == FTArray {
		srcFieldType = srcField.ElementType
	}
	return srcFieldType == dstField.Type
}

func convertRule[Rule rule](r rule) (rr Rule) {
	if r == nil {
		return
	}
	if v, ok := r.(Rule); ok {
		rr = v
	}
	return
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
