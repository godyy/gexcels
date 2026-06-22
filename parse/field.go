package parse

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/godyy/gexcels"
	"github.com/godyy/gutils/buffer/bytes"
	pkg_errors "github.com/pkg/errors"
)

const (
	maxStringLength = bytes.MaxStringLength // 支持的string最大长度
	maxArrayLength  = 32767                 // 最大数组长度
)

// getPrimitiveReflectType 后去primitive类型的反射类型
func getPrimitiveReflectType(ft gexcels.FieldType) reflect.Type {
	switch ft {
	case gexcels.FTInt32:
		return reflect.TypeOf(int32(0))
	case gexcels.FTInt64:
		return reflect.TypeOf(int64(0))
	case gexcels.FTFloat32:
		return reflect.TypeOf(float32(0))
	case gexcels.FTFloat64:
		return reflect.TypeOf(float64(0))
	case gexcels.FTBool:
		return reflect.TypeOf(false)
	case gexcels.FTString:
		return reflect.TypeOf("")
	default:
		panic("parse: getPrimitiveReflectType: field type " + strconv.Itoa(int(ft)) + " invalid")
	}
}

// parsePrimitiveValue 解析primitive字段值
func parsePrimitiveValue(ft gexcels.FieldType, s string) (any, error) {
	switch ft {
	case gexcels.FTInt32:
		s = strings.TrimSpace(s)
		if s == "" {
			return int32(0), nil
		}
		i, err := strconv.ParseInt(s, 10, 32)
		if err != nil {
			return nil, err
		}
		return int32(i), nil

	case gexcels.FTInt64:
		s = strings.TrimSpace(s)
		if s == "" {
			return int64(0), nil
		}
		i, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return nil, err
		}
		return int64(i), nil

	case gexcels.FTFloat32:
		s = strings.TrimSpace(s)
		if s == "" {
			return float32(0), nil
		}
		f, err := strconv.ParseFloat(s, 32)
		if err != nil {
			return nil, err
		}
		return float32(f), nil

	case gexcels.FTFloat64:
		if s == "" {
			return float64(0), nil
		}
		f, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return nil, err
		}
		return float64(f), nil

	case gexcels.FTBool:
		if s == "" {
			return false, nil
		}
		b, err := strconv.ParseBool(s)
		if err != nil {
			return nil, err
		}
		return b, nil

	case gexcels.FTString:
		if len(s) > maxStringLength {
			return nil, errStringLengthExceedLimit
		}
		return s, nil
	default:
		return nil, errFieldTypeInvalid(ft)
	}
}

// makePrimitiveArray 构造元素为primitive的数组
func makePrimitiveArray(ft gexcels.FieldType) reflect.Value {
	switch ft {
	case gexcels.FTInt32:
		return reflect.New(reflect.SliceOf(reflect.TypeFor[int32]()))
	case gexcels.FTInt64:
		return reflect.New(reflect.SliceOf(reflect.TypeFor[int64]()))
	case gexcels.FTFloat32:
		return reflect.New(reflect.SliceOf(reflect.TypeFor[float32]()))
	case gexcels.FTFloat64:
		return reflect.New(reflect.SliceOf(reflect.TypeFor[float64]()))
	case gexcels.FTBool:
		return reflect.New(reflect.SliceOf(reflect.TypeFor[bool]()))
	case gexcels.FTString:
		return reflect.New(reflect.SliceOf(reflect.TypeFor[string]()))
	default:
		panic(errArrayElementInvalid(ft))
	}
}

// addCustomFieldType 添加自定义字段类型
func (p *Parser) addCustomFieldType(ft *gexcels.FieldTypeInfo) error {
	if ft, ok := p.customFieldTypes[ft.GetName()]; ok {
		return fmt.Errorf("custom field type [%s,%s] already define", ft.Type, ft.GetName())
	}
	p.customFieldTypes[ft.GetName()] = ft
	return nil
}

// getCustomFieldType 获取自定义字段类型
func (p *Parser) getCustomFieldType(name string) *gexcels.FieldTypeInfo {
	return p.customFieldTypes[name]
}

// hasCustomFieldType 是否有自定义字段类型
func (p *Parser) hasCustomFieldType(name string) bool {
	_, ok := p.customFieldTypes[name]
	return ok
}

// parseFieldTypeInfo 解析字段类型信息
func (p *Parser) parseFieldTypeInfo(typeStr string) (*gexcels.FieldTypeInfo, error) {
	var (
		ft gexcels.FieldType
		et gexcels.FieldType
	)

	if strings.HasPrefix(typeStr, "[]") {
		ft = gexcels.FTArray
		typeStr = typeStr[2:]
	}

	if ft == gexcels.FTArray {
		et = gexcels.ParsePrimitiveFieldType(typeStr)
		if et != gexcels.FTUnknown {
			return gexcels.NewPrimitiveArrayFieldTypeInfo(et), nil
		} else {
			if !gexcels.MatchName(typeStr) {
				return nil, fmt.Errorf("parseFieldTypeInfo: array element name %s invalid", typeStr)
			}
			customFieldTypeInfo := p.getCustomFieldType(typeStr)
			if customFieldTypeInfo == nil {
				return nil, fmt.Errorf("parseFieldTypeInfo: array element custom field type %s not found", typeStr)
			}
			return gexcels.NewArrayFieldTypeInfo(customFieldTypeInfo), nil
		}
	} else if ft = gexcels.ParsePrimitiveFieldType(typeStr); ft == gexcels.FTUnknown {
		if !gexcels.MatchName(typeStr) {
			return nil, fmt.Errorf("parseFieldTypeInfo: custom field type name %s invalid", typeStr)
		}
		customFieldTypeInfo := p.getCustomFieldType(typeStr)
		if customFieldTypeInfo == nil {
			return nil, fmt.Errorf("parseFieldTypeInfo: custom field type %s not found", typeStr)
		}
		return customFieldTypeInfo, nil
	} else {
		return gexcels.NewPrimitiveFieldTypeInfo(ft), nil
	}
}

// parseFieldValue 解析字段值
func (p *Parser) parseFieldValue(fd *gexcels.Field, s string) (any, error) {
	s = strings.TrimSpace(s)

	if fd.Type == gexcels.FTEnum {
		return p.parseEnumFieldValue(fd, s)
	} else if fd.Type == gexcels.FTArray {
		return p.parseArrayFieldValue(fd, s)
	} else if fd.Type == gexcels.FTStruct {
		return p.parseStructFieldValue(fd, s)
	} else {
		return parsePrimitiveValue(fd.Type, s)
	}
}

// parseEnumFieldValue 解析枚举字段值
func (p *Parser) parseEnumFieldValue(fd *gexcels.Field, s string) (any, error) {
	if s == "" {
		return nil, errors.New("empty enum item name")
	}

	enumName := fd.FieldTypeInfo.GetName()
	enum := p.GetEnum(enumName)
	if enum == nil {
		return nil, errEnumNotDefine(enumName)
	}

	item := enum.GetItemByName(s)
	if item == nil {
		return nil, errEnumItemNotDefine(enumName, s)
	}

	return enum.GetItemValue(item.Index), nil
}

// parseArrayFieldValue 解析数组字段值
func (p *Parser) parseArrayFieldValue(fd *gexcels.Field, s string) (any, error) {
	if s == "" {
		return nil, nil
	}

	var (
		arrayPtr reflect.Value
	)

	elementType := fd.GetElementType()
	if elementType.Type.Primitive() {
		arrayPtr = makePrimitiveArray(elementType.Type)
	} else if elementType.Type == gexcels.FTEnum {
		arrayPtr = reflect.New(reflect.SliceOf(reflect.TypeOf((*any)(nil)).Elem()))
	} else if elementType.Type == gexcels.FTStruct {
		sd := p.GetStructByName(elementType.GetName())
		if sd == nil {
			return nil, errStructNotDefine(elementType.GetName())
		}
		arrayPtr = reflect.New(reflect.SliceOf(reflect.PointerTo(sd.ReflectType)))
	} else {
		return nil, errArrayElementInvalid(elementType.Type)
	}

	if err := json.Unmarshal([]byte(s), arrayPtr.Interface()); err != nil {
		return nil, err
	}
	if arrayPtr.Elem().Len() > maxArrayLength {
		return nil, errArrayLengthExceedLimit
	}

	if err := p.adjustArrayFieldValue(fd, arrayPtr.Elem()); err != nil {
		return nil, pkg_errors.WithMessage(err, "adjust")
	}

	return arrayPtr.Elem().Interface(), nil
}

// adjustArrayFieldValue 调整数组字段值
func (p *Parser) adjustArrayFieldValue(fd *gexcels.Field, rv reflect.Value) error {
	elementType := fd.GetElementType()
	if elementType.Type == gexcels.FTEnum {
		enum := p.GetEnum(elementType.GetName())
		for i := 0; i < rv.Len(); i++ {
			e := rv.Index(i)
			itemName, ok := e.Interface().(string)
			if !ok {
				return fmt.Errorf("enum array element %d must be string", i)
			}
			itemValue, ok := enum.GetItemValueByName(itemName)
			if !ok {
				return pkg_errors.WithMessagef(errEnumItemNotDefine(enum.Name, itemName), "enum array element %d", i)
			}
			e.Set(reflect.ValueOf(itemValue))
		}
	} else if elementType.Type == gexcels.FTStruct {
		sd := p.GetStructByName(elementType.GetName())
		for i := 0; i < rv.Len(); i++ {
			e := rv.Index(i)
			if err := p.adjustStructValue(sd, e.Elem()); err != nil {
				return pkg_errors.WithMessagef(err, "struct array element %d", i)
			}
		}
	}
	return nil
}

// getNestedField 获取嵌套的字段
func (p *Parser) getNestedField(fd *gexcels.Field, fieldPath []string, depth int) *gexcels.Field {
	if depth >= len(fieldPath) {
		return fd
	}

	if fd.Type == gexcels.FTStruct || (fd.Type == gexcels.FTArray && fd.GetElementType().Type == gexcels.FTStruct) {
		structName := ""
		if fd.Type == gexcels.FTStruct {
			structName = fd.GetName()
		} else {
			structName = fd.GetElementType().GetName()
		}
		sd := p.GetStructByName(structName)
		nestedFd := sd.GetFieldByName(fieldPath[depth])
		if nestedFd == nil {
			return nil
		}
		return p.getNestedField(nestedFd, fieldPath, depth+1)
	}

	return nil
}

// getFieldReflectType 获取字段的反射类型
func (p *Parser) getFieldReflectType(fd *gexcels.Field) (reflect.Type, error) {
	if fd.Type.Primitive() {
		return getPrimitiveReflectType(fd.Type), nil
	} else if fd.Type == gexcels.FTEnum {
		return reflect.TypeOf((*any)(nil)).Elem(), nil
	} else if fd.Type == gexcels.FTStruct {
		sd := p.GetStructByName(fd.GetName())
		if sd == nil {
			return nil, errStructNotDefine(fd.GetName())
		}
		return sd.ReflectType, nil
	} else if fd.Type == gexcels.FTArray {
		var elementReflectType reflect.Type
		elementType := fd.GetElementType()
		if elementType.Type.Primitive() {
			elementReflectType = getPrimitiveReflectType(elementType.Type)
		} else if elementType.Type == gexcels.FTEnum {
			elementReflectType = reflect.TypeOf((*any)(nil)).Elem()
		} else if elementType.Type == gexcels.FTStruct {
			sd := p.GetStructByName(elementType.GetName())
			if sd == nil {
				return nil, fmt.Errorf("array element struct %s not define", elementType.GetName())
			}
			elementReflectType = reflect.PointerTo(sd.ReflectType)
		} else {
			panic(errArrayElementInvalid(elementType.Type))
		}
		return reflect.SliceOf(elementReflectType), nil
	} else {
		panic(errFieldTypeInvalid(fd.Type))
	}
}
