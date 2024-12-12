package gexcels

import (
	"errors"
	"fmt"
	"strings"
)

var (
	// 未指定路径
	errNoPathSpecified = errors.New("no path specified")

	// 表格行或列数量不匹配
	errSheetRowsOrColsNotMatch = errors.New("sheet rows or cols not match")

	// 表格名无效
	errSheetNameInvalid = errors.New("sheet-name invalid")

	// 第一个字段必须是ID
	errFirstFieldMustID = fmt.Errorf("first field name must %s", idExportName)

	// ID字段非primitive类型
	errIDNonPrimitive = fmt.Errorf("field %s type must be primitive", idExportName)

	// 结构体字段定义无效
	errStructFieldDefineInvalid = errors.New("struct field definition invalid")

	// unique规则应用在非primitive字段上
	errRuleUniqueOnNonPrimitiveField = errors.New("rule unique on non-primitive field")

	// link规则应用在非primitive字段上
	errRuleLinkOnNonPrimitiveField = errors.New("rule link on non-primitive field")

	// 发现link错误
	errLinkErrorsFound = errors.New("link errors found")

	// 组合键规则应用在非primitive字段上
	errRuleCompositeKeyOnNonPrimitiveField = errors.New("rule composite key on non-primitive field")

	// 组合键应用在全局配置表上
	errRuleCompositeKeyOnGlobalTable = errors.New("rule composite key on global table")

	// 字符串长度超过限制
	errStringLengthExceedLimit = errors.New("string length exceed limit")

	// 字段编号超过限制
	errFieldNumberExceedLimit = errors.New("field number exceed limit")

	// 数组长度超过限制
	errArrayLengthExceedLimit = errors.New("array length exceed limit")
)

func errFieldTypeInvalid(ft interface{}) error {
	return fmt.Errorf("field type %v invalid", ft)
}

func errFieldNameInvalid(name string) error {
	return fmt.Errorf("field name %s invalid", name)
}

func errFieldNameDuplicate(name string) error {
	return fmt.Errorf("field name %s duplicate", name)
}

func errStructNotDefine(structName string) error {
	return fmt.Errorf("struct %s not define", structName)
}

func errArrayElementInvalid(elementType FieldType) error {
	return fmt.Errorf("array element type cant be %v", elementType)
}

func errRuleDefineInvalid(rule string) error {
	return fmt.Errorf("rule definition {%s} invalid", rule)
}

func errRuleTypeInvalid(ruleType string) error {
	return fmt.Errorf("rule type {%s} invalid", ruleType)
}

func errRuleMultiple(ruleType string) error {
	return fmt.Errorf("rule %s multiple", ruleType)
}

// errTableLink 封装TableLink失败错误
type errTableLink struct {
	srcTable string
	srcField []string
	dstTable string
	dstField string
	msg      string
}

func (err *errTableLink) Error() string {
	return fmt.Sprintf("link [%s.%s -> %s.%s] %s", err.srcTable, strings.Join(err.srcField, "."), err.dstTable, err.dstField, err.msg)
}

func makeErrTableLink(srcTable string, srcField []string, dstTable string, dstField string, f string, args ...interface{}) *errTableLink {
	return &errTableLink{
		srcTable: srcTable,
		srcField: srcField,
		dstTable: dstTable,
		dstField: dstField,
		msg:      fmt.Sprintf(f, args...),
	}
}

func makeErrTableLinkByTableLink(srcTable string, link *TableLink, f string, args ...interface{}) *errTableLink {
	return makeErrTableLink(srcTable, link.SrcField, link.DstTable, link.DstField, f, args...)
}
