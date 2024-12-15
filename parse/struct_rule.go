package parse

import (
	"fmt"
	"strings"
)

// parseStructRule 解析结构体规则
func (p *Parser) parseStructRule(sd *Struct, ruleStr string) error {
	ruleStr = strings.TrimSpace(ruleStr)
	if ruleStr == "" {
		return nil
	}

	rules := strings.Split(ruleStr, fieldRuleSep)
	for _, rule := range rules {
		rule = strings.TrimSpace(rule)
		matches := ruleRegexp.FindStringSubmatch(rule)
		if len(matches) != 3 {
			return errFieldRuleDefineInvalid(rule)
		}

		ruleType := fieldRuleType(strings.ToLower(matches[1]))
		switch ruleType {
		case fieldRuleLink:
			values := structRuleLinkRegexp.FindStringSubmatch(matches[2])
			if len(values) != 4 {
				return fmt.Errorf("fieldRule link value (%s) invalid", matches[2])
			}
			localFieldName := values[1]
			localFd := sd.getFieldByName(localFieldName)
			if localFd == nil {
				return fmt.Errorf("fieldRule link local field[%s] not found", localFieldName)
			}
			if !localFd.Type.Primitive() && !(localFd.Type == FTArray && localFd.ElementType.Primitive()) {
				return fmt.Errorf("fieldRule link local field[%s] must be primitive", localFieldName)
			}
			rule := &fieldLink{
				dstTable: values[2],
				dstField: values[3],
			}
			if !localFd.addRule(rule) {
				return fmt.Errorf("multiple fieldRule link on field[%s]", localFieldName)
			}
		default:
			return errFieldRuleTypeInvalid(fieldRuleType(matches[1]))
		}
	}

	return nil
}

// parseStructRuleLink 解析结构体链接规则
func (p *Parser) parseStructRuleLink(sd *Struct, value string) error {
	values := structRuleLinkRegexp.FindStringSubmatch(value)
	if len(values) != 4 {
		return fmt.Errorf("fieldRule link value (%s) invalid", value)
	}
	localFieldName := values[1]
	localFd := sd.getFieldByName(localFieldName)
	if localFd == nil {
		return fmt.Errorf("fieldRule link local field[%s] not found", localFieldName)
	}
	if !localFd.Type.Primitive() && !(localFd.Type == FTArray && localFd.ElementType.Primitive()) {
		return fmt.Errorf("fieldRule link local field[%s] must be primitive", localFieldName)
	}
	rule := &fieldLink{
		dstTable: values[2],
		dstField: values[3],
	}
	if !localFd.addRule(rule) {
		return fmt.Errorf("multiple fieldRule link on field[%s]", localFieldName)
	}
	return nil
}
