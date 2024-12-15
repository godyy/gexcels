package parse

import (
	"errors"
	"fmt"
	"github.com/godyy/gexcels/internal/define"
	"strings"
)

// Exported Errors
var (
	// ErrNoPathSpecified 未指定路径
	ErrNoPathSpecified = errors.New("parse: no path specified")

	// ErrLinkErrorsFound 发现link错误
	ErrLinkErrorsFound = errors.New("parse: link errors found")
)

var (
	// errSheetRowsOrColsNotMatch 表格行或列数量不匹配
	errSheetRowsOrColsNotMatch = errors.New("sheet rows or cols not match")

	// errSheetNameInvalid 表格名无效
	errSheetNameInvalid = errors.New("sheet-name invalid")

	// errFirstFieldMustID 第一个字段必须是ID
	errFirstFieldMustID = fmt.Errorf("first field name must %s", define.TableFieldIDName)

	// errIDNonPrimitive ID字段非primitive类型
	errIDNonPrimitive = fmt.Errorf("field %s type must be primitive", define.TableFieldIDName)

	// errStructFieldDefineInvalid 结构体字段定义无效
	errStructFieldDefineInvalid = errors.New("struct field definition invalid")

	// errStringLengthExceedLimit 字符串长度超过限制
	errStringLengthExceedLimit = errors.New("string length exceed limit")

	// errFieldNumberExceedLimit 字段编号超过限制
	errFieldNumberExceedLimit = errors.New("field number exceed limit")

	// errArrayLengthExceedLimit 数组长度超过限制
	errArrayLengthExceedLimit = errors.New("array length exceed limit")
)

// errFieldTypeInvalid 字段类型无效
func errFieldTypeInvalid(ft interface{}) error {
	return fmt.Errorf("field type %v invalid", ft)
}

// errFieldNameInvalid 字段名无效
func errFieldNameInvalid(name string) error {
	return fmt.Errorf("field name %s invalid", name)
}

// errFieldNameDuplicate 字段名重复
func errFieldNameDuplicate(name string) error {
	return fmt.Errorf("field name %s duplicate", name)
}

// errStructNotDefine 未定义的结构体
func errStructNotDefine(structName string) error {
	return fmt.Errorf("struct %s not define", structName)
}

// errArrayElementInvalid 数组元素类型无效
func errArrayElementInvalid(elementType FieldType) error {
	return fmt.Errorf("array element type cant be %v", elementType)
}

// errFieldRuleDefineInvalid 字段规则定义无效
func errFieldRuleDefineInvalid(rule string) error {
	return fmt.Errorf("field rule definition {%s} invalid", rule)
}

// errFieldRuleTypeInvalid 字段规则类型无效
func errFieldRuleTypeInvalid(ruleType fieldRuleType) error {
	return fmt.Errorf("field rule type {%s} invalid", ruleType)
}

// errFieldRuleMultiple 字段规则重复
func errFieldRuleMultiple(ruleType fieldRuleType) error {
	return fmt.Errorf("field rule %s multiple", ruleType)
}

// errFieldRuleOnNonPrimitiveField 字段规则应用在非primitive字段上
func errFieldRuleOnNonPrimitiveField(ruleType fieldRuleType) error {
	return fmt.Errorf("field rule %s on non-primitive field", ruleType)
}

// errFieldRuleOnGlobalTable 字段规则应用在全局配置表上
func errFieldRuleOnGlobalTable(ruleType fieldRuleType) error {
	return fmt.Errorf("field rule %s on global table", ruleType)
}

// tableLinkError 封装TableLink失败错误
type tableLinkError struct {
	srcTable string
	srcField []string
	dstTable string
	dstField string
	msg      string
}

func (err *tableLinkError) Error() string {
	return fmt.Sprintf(" link [%s.%s -> %s.%s] %s", err.srcTable, strings.Join(err.srcField, "."), err.dstTable, err.dstField, err.msg)
}

// errTableLink 生成配置表链接错误
func errTableLink(srcTable string, srcField []string, dstTable string, dstField string, f string, args ...interface{}) *tableLinkError {
	return &tableLinkError{
		srcTable: srcTable,
		srcField: srcField,
		dstTable: dstTable,
		dstField: dstField,
		msg:      fmt.Sprintf(f, args...),
	}
}

// errTableLinkByTableLink 通过配置表link规则生成配置表链接错误
func errTableLinkByTableLink(srcTable string, link *TableLink, f string, args ...interface{}) *tableLinkError {
	return errTableLink(srcTable, link.srcField, link.dstTable, link.dstField, f, args...)
}
