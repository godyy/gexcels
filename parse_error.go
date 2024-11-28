package gexcels

import (
	"errors"
	"fmt"
	"strings"
)

var errNoPathSpecified = errors.New("no path specified")
var errFirstFieldMustId = fmt.Errorf("first field name must %s", idExportName)
var errIDNonPrimitive = fmt.Errorf("field %s type must be primitive", idExportName)
var errFieldTypeEmpty = errors.New("field type empty")
var errSheetRowsOrColsNotMatch = errors.New("sheet rows or cols not match")
var errSheetNameInvalid = errors.New("sheet-name invalid")
var errFieldDefineInvalid = errors.New("field definition invalid")
var errRuleUniqueOnNonPrimitiveField = errors.New("rule Unique on non-primitive field")
var errRuleUniqueHasValue = errors.New("Unique has value")
var errRuleLinkOnNonPrimitiveField = errors.New("rule link on non-primitive field")
var errLinkErrorFound = errors.New("link errors found")
var errStringLengthExceedLimit = errors.New("string length exceed limit")
var errFieldNumberExceedLimit = errors.New("field number exceed limit")
var errArrayLengthExceedLimit = errors.New("array length exceed limit")

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
