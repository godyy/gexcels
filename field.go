package gexcels

// Field 封装字段信息
type Field struct {
	Name           string // 名称
	Desc           string // 描述
	*FieldTypeInfo        // 类型信息

	rules   []FieldRule          // 规则
	ruleMap map[string]FieldRule // 规则映射
}

// NewField 创建字段
func NewField(name, desc string, typeInfo *FieldTypeInfo) *Field {
	if !MatchName(name) {
		panic("gexcels: NewField: name " + name + " invalid")
	}
	if typeInfo == nil {
		panic("gexcels: NewField: typeInfo is nil")
	}
	return &Field{
		Name:          name,
		Desc:          desc,
		FieldTypeInfo: typeInfo,
	}
}

// AddRule 添加规则
func (f *Field) AddRule(rule FieldRule) bool {
	key := rule.FRKey()
	if len(f.ruleMap) > 0 {
		if _, ok := f.ruleMap[key]; ok {
			return false
		}
	} else {
		f.ruleMap = make(map[string]FieldRule)
	}
	f.rules = append(f.rules, rule)
	f.ruleMap[key] = rule
	return true
}

// GetRule 获取规则
func (f *Field) GetRule(key string) FieldRule {
	if len(f.ruleMap) == 0 {
		return nil
	}
	return f.ruleMap[key]
}

// Rules 获取规则数据
func (f *Field) Rules() []FieldRule {
	return f.rules
}

// GetFRLink 获得 FRLink
func (f *Field) GetFRLink() *FRLink {
	fr := f.GetRule(FRNLink)
	if fr == nil {
		return nil
	}
	return fr.(*FRLink)
}

// HasFRUnique 是否具备 FRUnique
func (f *Field) HasFRUnique() bool {
	return f.GetRule(FRNUnique) != nil
}
