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
	FTEnum
	FTStruct
	FTArray
	FTMap
)

// fieldTypeStrings 字段类型字符串映射
var fieldTypeStrings = [...]string{
	FTUnknown: "unknown",
	FTInt32:   "int32",
	FTInt64:   "int64",
	FTFloat32: "float32",
	FTFloat64: "float64",
	FTBool:    "bool",
	FTString:  "string",
	FTEnum:    "enum",
	FTStruct:  "struct",
	FTArray:   "array",
	FTMap:     "map",
}

func (ft FieldType) String() string {
	return fieldTypeStrings[ft]
}

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

// mapKeyFieldTypes map key 类型映射
// 仅支持 int32, int64, string, enum 类型
var mapKeyFieldTypes = map[FieldType]bool{
	FTInt32:  true,
	FTInt64:  true,
	FTString: true,
	FTEnum:   true,
}

// CanMapKey 返回是否可以作为 map key 类型
func (ft FieldType) CanMapKey() bool {
	return mapKeyFieldTypes[ft]
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

// MaxStringLen 最大字符串长度 FTString.
const MaxStringLen = 32767

// MaxElementLen 最大元素数量, 用于限制 FTArray 和 FTMap 类型字段的长度.
const MaxElementLen = 32767

// 字段类型参数索引.
const (
	_              = iota
	ftpName        // 类型名称. for FTStruct, FTEnum
	ftpElementType // 元素类型. for FTArray
	ftpMapKeyType  // map key type. for FTMap
	ftpMapValType  // map value type. for FTMap
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

// setMapKeyType 设置 map 的 key 类型
func (i *FieldTypeInfo) setMapKeyType(kt *FieldTypeInfo) {
	i.params[ftpMapKeyType] = kt
}

// GetMapKeyType 获取 map 的 key 类型
func (i *FieldTypeInfo) GetMapKeyType() *FieldTypeInfo {
	return i.params[ftpMapKeyType].(*FieldTypeInfo)
}

// setMapValueType 设置 map 的 value 类型
func (i *FieldTypeInfo) setMapValueType(vt *FieldTypeInfo) {
	i.params[ftpMapValType] = vt
}

// GetMapValueType 获取 map 的 value 类型
func (i *FieldTypeInfo) GetMapValueType() *FieldTypeInfo {
	return i.params[ftpMapValType].(*FieldTypeInfo)
}

// String 将 FieldTypeInfo 转换为字符串形式
func (i *FieldTypeInfo) String() string {
	switch i.Type {
	case FTEnum:
		return i.GetName()
	case FTStruct:
		return i.GetName()
	case FTArray:
		return "[]" + i.GetElementType().String()
	case FTMap:
		return "map[" + i.GetMapKeyType().String() + "]" + i.GetMapValueType().String()
	default:
		return i.Type.String()
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

// NewEnumFieldTypeInfo 创建枚举字段类型信息
func NewEnumFieldTypeInfo(enumName string) *FieldTypeInfo {
	if !MatchName(enumName) {
		panic("gexcels: NewEnumFieldTypeInfo: enum name " + enumName + " invalid")
	}
	info := newFieldTypeInfo(FTEnum)
	info.setName(enumName)
	return info
}

// NewArrayFieldTypeInfo 创建数组字段类型信息
func NewArrayFieldTypeInfo(et *FieldTypeInfo) *FieldTypeInfo {
	info := newFieldTypeInfo(FTArray)
	info.setElementType(et)
	return info
}

// NewMapFieldTypeInfo 创建 map 字段类型信息
func NewMapFieldTypeInfo(kt *FieldTypeInfo, vt *FieldTypeInfo) *FieldTypeInfo {
	if kt == nil {
		panic("gexcels: NewMapFieldTypeInfo: key type is nil")
	}
	if vt == nil {
		panic("gexcels: NewMapFieldTypeInfo: value type is nil")
	}
	info := newFieldTypeInfo(FTMap)
	info.setMapKeyType(kt)
	info.setMapValueType(vt)
	return info
}

// ParseFieldTypeInfo 解析字段类型信息
func ParseFieldTypeInfo(s string) (*FieldTypeInfo, error) {
	var parse func(s string) (*FieldTypeInfo, error)
	parse = func(s string) (*FieldTypeInfo, error) {
		s = strings.TrimSpace(s)
		if s == "" {
			return nil, fmt.Errorf("gexcels: ParseFieldTypeInfo: empty type")
		}

		if strings.HasPrefix(s, "[]") {
			elem, err := parse(s[2:])
			if err != nil {
				return nil, err
			}
			return NewArrayFieldTypeInfo(elem), nil
		}

		if strings.HasPrefix(s, "map[") {
			// map 类型表达式：map[kt]vt
			// 注意：这里不依赖正则，目的是支持 vt 继续递归（如 map[string][]Foo）
			end := strings.IndexByte(s, ']')
			if end <= len("map[") {
				return nil, fmt.Errorf("gexcels: ParseFieldTypeInfo: map key type invalid: %s", s)
			}
			keyStr := strings.TrimSpace(s[len("map["):end])
			valStr := strings.TrimSpace(s[end+1:])
			if valStr == "" {
				return nil, fmt.Errorf("gexcels: ParseFieldTypeInfo: map value type empty: %s", s)
			}

			keyType := ParsePrimitiveFieldType(keyStr)
			if keyType == FTUnknown {
				return nil, fmt.Errorf("gexcels: ParseFieldTypeInfo: map key type %s invalid", keyStr)
			}
			kt := NewPrimitiveFieldTypeInfo(keyType)
			vt, err := parse(valStr)
			if err != nil {
				return nil, err
			}
			return NewMapFieldTypeInfo(kt, vt), nil
		}

		if ft := ParsePrimitiveFieldType(s); ft != FTUnknown {
			return NewPrimitiveFieldTypeInfo(ft), nil
		}

		if !MatchName(s) {
			return nil, fmt.Errorf("gexcels: ParseFieldTypeInfo: struct name %s invalid", s)
		}
		return NewStructFieldTypeInfo(s), nil
	}

	return parse(s)
}
