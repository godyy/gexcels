package gexcels

import (
	"fmt"
	"regexp"
)

const namePattern = `[A-Za-z]+\w*`

var (
	nameRegexp = regexp.MustCompile(namePattern)

	fieldNameRegexp = nameRegexp

	fieldTypeRegexp = regexp.MustCompile(fmt.Sprintf(`(?:\[\])?%s`, namePattern))

	structNameRegexp = nameRegexp

	structFieldsCheckRegexp = regexp.MustCompile(fmt.Sprintf(`^%s:(?:\[\])?%s(?::"[^"]*")?(?:\s*,\s*(%s:(?:\[\])?%s(?::"[^"]*")?))*$`,
		namePattern, namePattern, namePattern, namePattern))

	structFieldRegexp = regexp.MustCompile(fmt.Sprintf(`(%s):((?:\[\])?%s)(?::"([^"]*)")?`, namePattern, namePattern))

	tableSheetNameRegexp = regexp.MustCompile(fmt.Sprintf(`^(.*)\|(%s)$`, namePattern))

	globalTableNameRegexp = regexp.MustCompile(`Global\w+`)

	tagRegexp = regexp.MustCompile(fmt.Sprintf(`%s(?:/%s)*`, namePattern, namePattern))

	ruleRegexp = regexp.MustCompile(`^(\w+)\s*(?:=\s*([\w,\.]+))?$`)

	structRuleLinkRegexp = regexp.MustCompile(fmt.Sprintf(`^(%s),(%s).(%s)$`, namePattern, namePattern, namePattern))

	tableFieldRuleLinkRegexp = regexp.MustCompile(fmt.Sprintf(`^(%s).(%s)$`, namePattern, namePattern))

	tableFieldRuleCompositeKey = regexp.MustCompile(fmt.Sprintf(`^(\w+),([0-9])$`))

	commentRegexp = regexp.MustCompile(`^#[\s\S]*$`)
)
