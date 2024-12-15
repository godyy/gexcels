package parse

import (
	"encoding/json"
	"fmt"
	"github.com/godyy/gexcels"
	"github.com/godyy/gutils/buffer/bytes"
	"reflect"
	"strconv"
	"strings"
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
func parsePrimitiveValue(ft gexcels.FieldType, s string) (interface{}, error) {
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
func makePrimitiveArray(ft gexcels.FieldType) interface{} {
	switch ft {
	case gexcels.FTInt32:
		return make([]int32, 0)
	case gexcels.FTInt64:
		return make([]int64, 0)
	case gexcels.FTFloat32:
		return make([]float32, 0)
	case gexcels.FTFloat64:
		return make([]float64, 0)
	case gexcels.FTBool:
		return make([]bool, 0)
	case gexcels.FTString:
		return make([]string, 0)
	default:
		panic(errArrayElementInvalid(ft))
	}
}

// parseFieldTypeInfo 解析字段类型信息
func (p *Parser) parseFieldTypeInfo(typeStr string) (*gexcels.FieldTypeInfo, error) {
	return gexcels.ParseFieldTypeInfo(strings.TrimSpace(typeStr))
}

// parseFieldValue 解析字段值
func (p *Parser) parseFieldValue(fd *gexcels.Field, s string) (interface{}, error) {
	s = strings.TrimSpace(s)

	if s == "" {
		return nil, nil
	}

	if fd.Type == gexcels.FTArray {
		return p.parseArrayFieldValue(fd, s)
	} else if fd.Type == gexcels.FTStruct {
		return p.parseStructFieldValue(fd, s)
	} else {
		return parsePrimitiveValue(fd.Type, s)
	}
}

// parseArrayFieldValue 解析数组字段值
func (p *Parser) parseArrayFieldValue(fd *gexcels.Field, s string) (interface{}, error) {
	if s == "" {
		return nil, nil
	}

	if fd.ElementType.Primitive() {
		array := makePrimitiveArray(fd.ElementType)
		if err := json.Unmarshal([]byte(s), &array); err != nil {
			return nil, err
		}
		if reflect.ValueOf(array).Len() > maxArrayLength {
			return nil, errArrayLengthExceedLimit
		}
		return array, nil
	} else if fd.ElementType == gexcels.FTStruct {
		sd := p.GetStructByName(fd.StructName)
		if sd == nil {
			return nil, errStructNotDefine(fd.StructName)
		}
		arrayPtr := reflect.New(reflect.SliceOf(reflect.PointerTo(sd.ReflectType)))
		if err := json.Unmarshal([]byte(s), arrayPtr.Interface()); err != nil {
			return nil, err
		}
		array := arrayPtr.Elem()
		if array.Len() > maxArrayLength {
			return nil, errArrayLengthExceedLimit
		}
		return array.Interface(), nil
	} else {
		return nil, errArrayElementInvalid(fd.ElementType)
	}
}

// getNestedField 获取嵌套的字段
func (p *Parser) getNestedField(fd *gexcels.Field, fieldPath []string, depth int) *gexcels.Field {
	if depth >= len(fieldPath) {
		return fd
	}

	if fd.Type == gexcels.FTStruct || (fd.Type == gexcels.FTArray && fd.ElementType == gexcels.FTStruct) {
		sd := p.GetStructByName(fd.StructName)
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
	} else if fd.Type == gexcels.FTStruct {
		sd := p.GetStructByName(fd.StructName)
		if sd == nil {
			return nil, errStructNotDefine(fd.StructName)
		}
		return sd.ReflectType, nil
	} else if fd.Type == gexcels.FTArray {
		var elementType reflect.Type
		if fd.ElementType.Primitive() {
			elementType = getPrimitiveReflectType(fd.ElementType)
		} else if fd.ElementType == gexcels.FTStruct {
			sd := p.GetStructByName(fd.StructName)
			if sd == nil {
				return nil, fmt.Errorf("array element struct %s not define", fd.StructName)
			}
			elementType = reflect.PointerTo(sd.ReflectType)
		} else {
			panic(errArrayElementInvalid(fd.ElementType))
		}
		return reflect.SliceOf(elementType), nil
	} else {
		panic(errFieldTypeInvalid(fd.Type))
	}
}
