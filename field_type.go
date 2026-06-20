package gexcels

import (
	"fmt"
	"strings"
)

// FieldType 字段类型
type FieldType int8

// 字段类型
const (
	FTUnknown = FieldType(iota)
	FTInt32
	FTInt64
	FTFloat32
	FTFloat64
	FTBool
	FTString
	FTStruct
	FTArray
)

// primitiveFieldTypeStrings primitive FieldType 到其字符形式的映射
var primitiveFieldTypeStrings = map[FieldType]string{
	FTInt32:   "int32",
	FTInt64:   "int64",
	FTFloat32: "float32",
	FTFloat64: "float64",
	FTBool:    "bool",
	FTString:  "string",
}

// Primitive 返回是否primitive类型
func (ft FieldType) Primitive() bool {
	return primitiveFieldTypeStrings[ft] != ""
}

// PrimitiveString 返回primitive字符串形式
func (ft FieldType) PrimitiveString() string {
	return primitiveFieldTypeStrings[ft]
}

// primitiveFieldStringTypes primitive FieldType 字符串到值映射
var primitiveFieldStringTypes = map[string]FieldType{
	"int32":   FTInt32,
	"int64":   FTInt64,
	"float32": FTFloat32,
	"float64": FTFloat64,
	"bool":    FTBool,
	"string":  FTString,
}

// ParsePrimitiveFieldType 解析 primitive FieldType 的字符形式, 返回类型 FieldType
func ParsePrimitiveFieldType(s string) FieldType {
	return primitiveFieldStringTypes[s]
}

// 字段类型参数索引.
const (
	_              = iota
	ftpName        // 类型名称. for FTStruct
	ftpElementType // 元素类型. for FTArray
)

// FieldTypeInfo 字段类型信息
type FieldTypeInfo struct {
	Type   FieldType   // 类型
	params map[int]any // 参数
}

// setName 设置类型名称
func (i *FieldTypeInfo) setName(name string) {
	i.params[ftpName] = name
}

// GetName 获取类型名称
func (i *FieldTypeInfo) GetName() string {
	return i.params[ftpName].(string)
}

// setElementType 设置元素类型
func (i *FieldTypeInfo) setElementType(et *FieldTypeInfo) {
	i.params[ftpElementType] = et
}

// GetElementType 获取元素类型
func (i *FieldTypeInfo) GetElementType() *FieldTypeInfo {
	return i.params[ftpElementType].(*FieldTypeInfo)
}

// String 将 FieldTypeInfo 转换为字符串形式
func (i *FieldTypeInfo) String() string {
	switch i.Type {
	case FTStruct:
		return i.GetName()
	case FTArray:
		return "[]" + i.GetElementType().String()
	default:
		return primitiveFieldTypeStrings[i.Type]
	}
}

func newFieldTypeInfo(ft FieldType) *FieldTypeInfo {
	return &FieldTypeInfo{Type: ft, params: make(map[int]any)}
}

// NewPrimitiveFieldTypeInfo 创建primitive字段类型信息
func NewPrimitiveFieldTypeInfo(ft FieldType) *FieldTypeInfo {
	if !ft.Primitive() {
		panic(fmt.Sprintf("gexcels: NewPrimitiveFieldTypeInfo: field type %d must be primitive", ft))
	}
	return &FieldTypeInfo{Type: ft}
}

// NewPrimitiveArrayFieldTypeInfo 创建primitive数组字段类型
func NewPrimitiveArrayFieldTypeInfo(et FieldType) *FieldTypeInfo {
	if !et.Primitive() {
		panic("gexcels.NewPrimitiveArrayFieldType: element type must be primitive")
	}

	info := newFieldTypeInfo(FTArray)
	info.setElementType(NewPrimitiveFieldTypeInfo(et))
	return info
}

// NewStructFieldTypeInfo 创建结构体字段类型信息
func NewStructFieldTypeInfo(structName string) *FieldTypeInfo {
	if !MatchName(structName) {
		panic("gexcels: NewStructFieldTypeInfo: struct name " + structName + " invalid")
	}
	info := newFieldTypeInfo(FTStruct)
	info.setName(structName)
	return info
}

// NewStructArrayFieldTypeInfo 创建结构体数组字段类型
func NewStructArrayFieldTypeInfo(structName string) *FieldTypeInfo {
	if !MatchName(structName) {
		panic("gexcels: NewStructFieldTypeInfo: struct name " + structName + " invalid")
	}
	info := newFieldTypeInfo(FTArray)
	info.setElementType(NewStructFieldTypeInfo(structName))
	return info
}

// ParseFieldTypeInfo 解析字段类型信息
func ParseFieldTypeInfo(s string) (*FieldTypeInfo, error) {
	var (
		ft FieldType
		et FieldType
	)

	if strings.HasPrefix(s, "[]") {
		ft = FTArray
		s = s[2:]
	}

	if ft == FTArray {
		et = ParsePrimitiveFieldType(s)
		if et == FTUnknown {
			if !MatchName(s) {
				return nil, fmt.Errorf("gexcels: ParseFieldTypeInfo: array element struct name %s invalid", s)
			}
			return NewStructArrayFieldTypeInfo(s), nil
		}
		return NewPrimitiveArrayFieldTypeInfo(et), nil
	} else if ft = ParsePrimitiveFieldType(s); ft == FTUnknown {
		if !MatchName(s) {
			return nil, fmt.Errorf("gexcels: ParseFieldTypeInfo: array element struct name %s invalid", s)
		}
		return NewStructFieldTypeInfo(s), nil
	} else {
		return NewPrimitiveFieldTypeInfo(ft), nil
	}
}
