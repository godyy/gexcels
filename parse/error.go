package parse

import (
	"errors"
	"fmt"
	"strings"

	"github.com/godyy/gexcels"
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
	errFirstFieldMustID = fmt.Errorf("first field name must %s", gexcels.TableFieldIDName)

	// errIDNonPrimitive ID字段非primitive类型
	errIDNonPrimitive = fmt.Errorf("field %s type must be primitive", gexcels.TableFieldIDName)

	// errStructFieldDefineInvalid 结构体字段定义无效
	errStructFieldDefineInvalid = errors.New("struct field definition invalid")

	// errStringLengthExceedLimit 字符串长度超过限制
	errStringLengthExceedLimit = errors.New("string length exceed limit")

	// errArrayLengthExceedLimit 数组长度超过限制
	errArrayLengthExceedLimit = errors.New("array length exceed limit")
)

// errFieldTypeInvalid 字段类型无效
func errFieldTypeInvalid(ft any) error {
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

// errEnumNotDefine 未定义的枚举
func errEnumNotDefine(enumName string) error {
	return fmt.Errorf("enum %s not define", enumName)
}

// errEnumItemNotDefine 未定义的枚举项
func errEnumItemNotDefine(enumName string, itemName string) error {
	return fmt.Errorf("enum %s item %s not define", enumName, itemName)
}

// errStructNotDefine 未定义的结构体
func errStructNotDefine(structName string) error {
	return fmt.Errorf("struct %s not define", structName)
}

// errArrayElementInvalid 数组元素类型无效
func errArrayElementInvalid(elementType gexcels.FieldType) error {
	return fmt.Errorf("array element type cant be %v", elementType)
}

// errFieldRuleDefineInvalid 字段规则定义无效
func errFieldRuleDefineInvalid(rule string) error {
	return fmt.Errorf("field rule definition {%s} invalid", rule)
}

// errFieldRuleInvalid 字段规则类型无效
func errFieldRuleInvalid(name string) error {
	return fmt.Errorf("field rule {%s} invalid", name)
}

// errFieldRuleMultiple 字段规则重复
func errFieldRuleMultiple(name string) error {
	return fmt.Errorf("field rule %s multiple", name)
}

// errFieldRuleOnNonPrimitiveField 字段规则应用在非primitive字段上
func errFieldRuleOnNonPrimitiveField(name string) error {
	return fmt.Errorf("field rule %s on non-primitive field", name)
}

// errFieldRuleOnGlobalTable 字段规则应用在全局配置表上
func errFieldRuleOnGlobalTable(name string) error {
	return fmt.Errorf("field rule %s on global table", name)
}

// tableLinkError 封装TableLink失败错误
type tableLinkError struct {
	srcTable string
	link     *TableLink
	msg      string
}

func (err *tableLinkError) Error() string {
	srcField := strings.Join(err.link.srcField, ".")
	if err.link.kind == tableLinkKindMapKey {
		srcField += fmt.Sprintf("[k%d]", err.link.mapLevel)
	}
	return fmt.Sprintf(" link [%s.%s -> %s.%s] %s", err.srcTable, srcField, err.link.dstTable, err.link.dstField, err.msg)
}

// errTableLink 通过配置表link规则生成配置表链接错误
func errTableLink(srcTable string, link *TableLink, f string, args ...any) *tableLinkError {
	err := &tableLinkError{
		srcTable: srcTable,
		link:     link,
		msg:      fmt.Sprintf(f, args...),
	}
	return err
}
