package parse

import (
	"regexp"
)

// namePattern 名称匹配模式
const namePattern = `[A-Za-z]+\w*`

var (
	// nameRegexp 名称匹配正则表达式
	nameRegexp = regexp.MustCompile(namePattern)

	// fieldNameRegexp 字段名匹配正则表达式
	fieldNameRegexp = nameRegexp

	// fieldTypeRegexp 字段类型匹配正则表达式
	fieldTypeRegexp = regexp.MustCompile(`(?:\[\])?` + namePattern)

	// structNameRegexp 结构体名匹配正则表达式
	structNameRegexp = nameRegexp

	// structFieldsCheckRegexp 结构体字段检查正则表达式
	structFieldsCheckRegexp = regexp.MustCompile(`^` + namePattern + `:(?:\[\])?` + namePattern + `(?::"[^"]*")?(?:\s*,\s*(` + namePattern + `:(?:\[\])?` + namePattern + `(?::"[^"]*")?))*$`)

	// structFieldRegexp 结构体字段匹配正则表达式
	structFieldRegexp = regexp.MustCompile(`(` + namePattern + `):((?:\[\])?` + namePattern + `)(?::"([^"]*)")?`)

	// tableSheetNameRegexp 配置表sheet名匹配正则表达式
	tableSheetNameRegexp = regexp.MustCompile(`^(.*)\|(` + namePattern + `)$`)

	// globalTableNameRegexp 全局配置表名匹配正则表达式
	globalTableNameRegexp = regexp.MustCompile(`Global\w+`)

	// tableTagRegexp 配置表内标签匹配正则表达式
	tableTagRegexp = regexp.MustCompile(tagPattern + `(?:` + tagSep + tagPattern + `)*`)

	// ruleRegexp 规则匹配正则表达式
	ruleRegexp = regexp.MustCompile(`^(\w+)\s*(?:=\s*([\w,.]+))?$`)

	// structRuleLinkRegexp 结构体字段链接规则匹配正则表达式
	structRuleLinkRegexp = regexp.MustCompile(`^(` + namePattern + `),(` + namePattern + `).(` + namePattern + `)$`)

	// tableFieldRuleLinkRegexp 配置表字段链接规则匹配正则表达式
	tableFieldRuleLinkRegexp = regexp.MustCompile(`^(` + namePattern + `).(` + namePattern + `)$`)

	// tableFieldRuleCompositeKeyRegexp 配置表字段组合键规则匹配正则表达式
	tableFieldRuleCompositeKeyRegexp = regexp.MustCompile(`^(\w+),([0-9])$`)

	// tableFieldRuleGroupRegexp 配置表字段分组规则匹配正则表达式
	tableFieldRuleGroupRegexp = regexp.MustCompile(`^(\w+),([0-9]+)$`)

	// commentRegexp 注释匹配正则表达式
	commentRegexp = regexp.MustCompile(`^#[\s\S]*$`)

	// tableFileNameRegexp 配置表文件名匹配正则表达式
	tableFileNameRegexp = regexp.MustCompile(`^[^.]+(?:\.(` + tagPattern + `))?$`)

	// structFileNameRegexp 结构体文件名匹配正则表达式
	structFileNameRegexp = regexp.MustCompile(`^(?:[^.]*\.)?struct(?:\.([0-9]*))?(?:\.(` + tagPattern + `))?$`)

	// structSheetNameRegexp 结构体sheet名匹配正则表达式
	structSheetNameRegexp = regexp.MustCompile(`^(.*)\|(Struct\w*)(?:\.([0-9]*))?`)
)
