package parse

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
	FTInt32.String():   FTInt32,
	FTInt64.String():   FTInt64,
	FTFloat32.String(): FTFloat32,
	FTFloat64.String(): FTFloat64,
	FTBool.String():    FTBool,
	FTString.String():  FTString,
}

// FromPrimitiveString 从Primitive字段类型字符串转换为值，若无效返回false
func (ft *FieldType) FromPrimitiveString(s string) bool {
	t, ok := primitiveFieldString2Types[s]
	if ok {
		*ft = t
	}
	return ok
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

// ToReflectType 转换为reflect.Type，若无效返回nil
func (ft *FieldType) ToReflectType() reflect.Type {
	if ft.Primitive() {
		return primitiveFieldType2ReflectType[*ft]
	}
	return nil
}
