package parse

import (
	"fmt"
	pkg_errors "github.com/pkg/errors"
	"strconv"
	"strings"
)

// parseTableFieldRules 解析配置表字段规则
func (p *Parser) parseTableFieldRules(td *Table, fd *TableField, s string) error {
	if s == "" {
		return nil
	}

	rules := strings.Split(s, fieldRuleSep)
	for _, rule := range rules {
		matches := ruleRegexp.FindStringSubmatch(rule)
		if len(matches) != 3 {
			return errFieldRuleDefineInvalid(rule)
		}

		ruleType := fieldRuleType(strings.ToLower(matches[1]))
		switch ruleType {
		case fieldRuleUnique:
			if td.IsGlobal {
				return errFieldRuleOnGlobalTable(ruleType)
			}
			if err := p.parseTableFieldRuleUnique(td, fd, matches[2]); err != nil {
				return err
			}
		case fieldRuleLink:
			if err := p.parseTableFieldRuleLink(td, fd, matches[2]); err != nil {
				return err
			}
		case fieldRuleCompositeKey:
			if td.IsGlobal {
				return errFieldRuleOnGlobalTable(ruleType)
			}
			if err := p.parseTableFieldRuleCompositeKey(td, fd, matches[2]); err != nil {
				return err
			}
		case fieldRuleGroup:
			if td.IsGlobal {
				return errFieldRuleOnGlobalTable(ruleType)
			}
			if err := p.parseTableFieldRuleGroup(td, fd, matches[2]); err != nil {
				return err
			}
		default:
			return errFieldRuleTypeInvalid(ruleType)
		}
	}

	return nil
}

// parseTableFieldRuleUnique 解析配置表字段Unique规则
func (p *Parser) parseTableFieldRuleUnique(td *Table, fd *TableField, value string) error {
	if !fd.Type.Primitive() {
		return errFieldRuleOnNonPrimitiveField(fieldRuleUnique)
	}
	exportMethod := false
	if value != "" {
		b, err := strconv.ParseBool(value)
		if err != nil {
			return pkg_errors.WithMessagef(err, "unique value \"%s\" invalid", value)
		}
		exportMethod = b
	}
	if !fd.addRule(&fieldUnique{exportMethod: exportMethod}) {
		return errFieldRuleMultiple(fieldRuleUnique)
	}
	return nil
}

// parseTableFieldRuleLink 解析配置表字段链接规则
func (p *Parser) parseTableFieldRuleLink(td *Table, fd *TableField, value string) error {
	if !fd.Type.Primitive() && !(fd.Type == FTArray && fd.ElementType.Primitive()) {
		return errFieldRuleOnNonPrimitiveField(fieldRuleLink)
	}

	ss := tableFieldRuleLinkRegexp.FindStringSubmatch(value)
	if len(ss) != 3 {
		return fmt.Errorf("link value \"%s\" invalid", value)
	}

	if !fd.addRule(&fieldLink{
		dstTable: ss[1],
		dstField: ss[2],
	}) {
		return errFieldRuleMultiple(fieldRuleLink)
	}

	td.addLink(newTableLink([]string{fd.Name}, ss[1], ss[2]))

	return nil
}

// parseTableFieldRuleCompositeKey 解析配置表字段组合键规则
func (p *Parser) parseTableFieldRuleCompositeKey(td *Table, fd *TableField, value string) error {
	if !fd.Type.Primitive() {
		return errFieldRuleOnNonPrimitiveField(fieldRuleCompositeKey)
	}

	ss := tableFieldRuleCompositeKeyRegexp.FindStringSubmatch(value)
	if len(ss) != 3 {
		return fmt.Errorf("composite-key value \"%s\" invalid", value)
	}

	name := ss[1]
	index, _ := strconv.Atoi(ss[2])

	rule := convertFieldRule[*fieldCompositeKey](fd.Field.getCreateRule(fieldRuleCompositeKey))
	if !rule.addKeyIndex(name, index) {
		return fmt.Errorf("composite-key %s keyIndex %d duplicate on field", name, index)
	}

	if !td.addCompositeKey(name, index, fd.Name) {
		return fmt.Errorf("composite-key %s keyIndex %d duplicate", name, index)
	}

	return nil
}

// parseTableFieldRuleGroup 解析配置表字段分组规则
func (p *Parser) parseTableFieldRuleGroup(td *Table, fd *TableField, value string) error {
	if !fd.Type.Primitive() {
		return errFieldRuleOnNonPrimitiveField(fieldRuleGroup)
	}

	ss := tableFieldRuleGroupRegexp.FindStringSubmatch(value)
	if len(ss) != 3 {
		return fmt.Errorf("group value \"%s\" invalid", value)
	}

	name := ss[1]
	index, _ := strconv.Atoi(ss[2])

	rule := convertFieldRule[*fieldGroup](fd.Field.getCreateRule(fieldRuleGroup))
	if !rule.addGroupIndex(name, index) {
		return fmt.Errorf("group %s index %d duplicate on field", name, index)
	}

	if !td.addGroup(name, index, fd.Name) {
		return fmt.Errorf("group %s index %d duplicate", name, index)
	}

	return nil

}
