package gexcels

import "reflect"

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

var fieldTypeStrings = [...]string{
	FTUnknown: "unknown",
	FTInt32:   "int32",
	FTInt64:   "int64",
	FTFloat32: "float32",
	FTFloat64: "float64",
	FTBool:    "bool",
	FTString:  "string",
	FTStruct:  "struct",
	FTArray:   "array",
}

func (ft FieldType) String() string {
	return fieldTypeStrings[ft]
}

// Primitive 返回是否primitive类型
func (ft FieldType) Primitive() bool {
	return ft >= FTInt32 && ft <= FTString
}

// primitive字段类型字符串到值映射
var primitiveFieldString2Types = map[string]FieldType{
	"int32":   FTInt32,
	"int64":   FTInt64,
	"float32": FTFloat32,
	"float64": FTFloat64,
	"bool":    FTBool,
	"string":  FTString,
}

// ConvertPrimitiveFieldString2Type
// 将primitive字段类型字符串转换为值，若无效返回FTUnknown
func ConvertPrimitiveFieldString2Type(s string) FieldType {
	if ft, ok := primitiveFieldString2Types[s]; ok {
		return ft
	} else {
		return FTUnknown
	}
}

// primitive字段类型值到字符串映射
var primitiveFieldType2Strings = [...]string{
	FTInt32:   "int32",
	FTInt64:   "int64",
	FTFloat32: "float32",
	FTFloat64: "float64",
	FTBool:    "bool",
	FTString:  "string",
}

// ConvertPrimitiveFieldType2String
// 将primitive字段类型值转换为字符串
func ConvertPrimitiveFieldType2String(ft FieldType) string {
	return primitiveFieldType2Strings[ft]
}

// primitive字段类型值到反射类型
var primitiveFieldType2ReflectType = [...]reflect.Type{
	FTInt32:   reflect.TypeOf(int32(0)),
	FTInt64:   reflect.TypeOf(int64(0)),
	FTFloat32: reflect.TypeOf(float32(0)),
	FTFloat64: reflect.TypeOf(float64(0)),
	FTBool:    reflect.TypeOf(false),
	FTString:  reflect.TypeOf(""),
}

// ConvertPrimitiveType2ReflectType
// 将primitive字段类型值转换为reflect.Type
func ConvertPrimitiveType2ReflectType(ft FieldType) reflect.Type {
	return primitiveFieldType2ReflectType[ft]
}
