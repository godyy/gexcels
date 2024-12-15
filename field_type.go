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

// FieldTypeInfo 字段类型信息
type FieldTypeInfo struct {
	Type        FieldType // 类型
	ElementType FieldType // 元素类型
	StructName  string    // 结构体名称
}

// String 将 FieldTypeInfo 转换为字符串形式
func (i *FieldTypeInfo) String() string {
	s := primitiveFieldTypeStrings[i.Type]
	if s != "" {
		return s
	}

	if i.Type == FTStruct {
		return i.StructName
	} else if i.Type == FTArray {
		s = primitiveFieldTypeStrings[i.ElementType]
		if s != "" {
			return "[]" + s
		}
		if i.ElementType == FTStruct {
			return "[]" + i.StructName
		} else {
			panic("gexcels: FieldType.String: element type invalid")
		}
	} else {
		panic("gexcels: FieldType.String: field type invalid")
	}
}

// NewPrimitiveFieldTypeInfo 创建primitive字段类型信息
func NewPrimitiveFieldTypeInfo(ft FieldType) *FieldTypeInfo {
	if ft.Primitive() {
		panic("gexcels: NewPrimitiveFieldTypeInfo: field type must be primitive")
	}
	return &FieldTypeInfo{Type: ft}
}

// NewPrimitiveArrayFieldTypeInfo 创建primitive数组字段类型
func NewPrimitiveArrayFieldTypeInfo(et FieldType) *FieldTypeInfo {
	if et.Primitive() {
		panic("gexcels.NewPrimitiveArrayFieldType: element type must be primitive")
	}
	return &FieldTypeInfo{
		Type:        FTArray,
		ElementType: et,
	}
}

// NewStructFieldTypeInfo 创建结构体字段类型信息
func NewStructFieldTypeInfo(structName string) *FieldTypeInfo {
	if !MatchName(structName) {
		panic("gexcels: NewStructFieldTypeInfo: struct name " + structName + " invalid")
	}
	return &FieldTypeInfo{
		Type:       FTStruct,
		StructName: structName,
	}
}

// NewStructArrayFieldTypeInfo 创建结构体数组字段类型
func NewStructArrayFieldTypeInfo(structName string) *FieldTypeInfo {
	if !MatchName(structName) {
		panic("gexcels: NewStructFieldTypeInfo: struct name " + structName + " invalid")
	}
	return &FieldTypeInfo{
		Type:        FTArray,
		ElementType: FTStruct,
		StructName:  structName,
	}
}

// ParseFieldTypeInfo 解析字段类型信息
func ParseFieldTypeInfo(s string) (*FieldTypeInfo, error) {
	var (
		ft         FieldType
		et         FieldType
		structName string
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
			et = FTStruct
			structName = s
		}
	} else if ft = ParsePrimitiveFieldType(s); ft == FTUnknown {
		if !MatchName(s) {
			return nil, fmt.Errorf("gexcels: ParseFieldTypeInfo: array element struct name %s invalid", s)
		}
		ft = FTStruct
		structName = s
	}

	return &FieldTypeInfo{
		Type:        ft,
		ElementType: et,
		StructName:  structName,
	}, nil
}
