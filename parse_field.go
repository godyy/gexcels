package gexcels

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/godyy/gutils/buffer/bytes"
)

const (
	maxStringLength = bytes.MaxStringLength // 支持的string最大长度
	maxArrayLength  = 32767                 // 最大数组长度
)

// Field 定义字段
type Field struct {
	Name        string    // 名称
	Desc        string    // 描述
	Type        FieldType // 类型
	ElementType FieldType // 元素类型
	StructName  string    // 结构体名称
	Struct      *Struct   // 结构体定义
	ExportName  string    // 导出名

	rules map[string]rule // 规则, 各类型至多一个
}

func newField(name, desc string) *Field {
	return &Field{
		Name:       name,
		Desc:       desc,
		ExportName: exportFieldName(name),
	}
}

func (p *Parser) getFieldReflectType(fd *Field) (reflect.Type, error) {
	if fd.Type.Primitive() {
		return ConvertPrimitiveType2ReflectType(fd.Type), nil
	} else if fd.Type == FTStruct {
		sd := p.getStruct(fd.StructName)
		if sd == nil {
			return nil, errStructNotDefine(fd.StructName)
		}
		return sd.ReflectType, nil
	} else if fd.Type == FTArray {
		var elementType reflect.Type
		if fd.ElementType.Primitive() {
			elementType = ConvertPrimitiveType2ReflectType(fd.Type)
		} else if fd.ElementType == FTStruct {
			sd := p.getStruct(fd.StructName)
			if sd == nil {
				return nil, fmt.Errorf("array element struct %s not define", fd.StructName)
			}
			elementType = reflect.PointerTo(sd.ReflectType)
		} else {
			panic(errArrayElementInvalid(fd.Type))
		}
		return reflect.SliceOf(elementType), nil
	} else {
		panic(errFieldTypeInvalid(fd.Type))
	}
}

func (p *Parser) parseFieldType(fd *Field, typeStr string) error {
	typeStr = strings.TrimSpace(typeStr)

	if typeStr == "" {
		return errFieldTypeEmpty
	}

	isArray := strings.HasPrefix(typeStr, "[]")
	if isArray {
		typeStr = typeStr[2:]
	}

	if ft := ConvertPrimitiveFieldString2Type(typeStr); ft != FTUnknown {
		if isArray {
			fd.Type = FTArray
			fd.ElementType = ft
		} else {
			fd.Type = ft
		}
	} else {
		sd := p.getStruct(typeStr)
		if sd == nil {
			return errStructNotDefine(typeStr)
		}
		if isArray {
			fd.Type = FTArray
			fd.ElementType = FTStruct
		} else {
			fd.Type = FTStruct
		}
		fd.StructName = typeStr
		fd.Struct = sd
	}

	return nil
}

func (p *Parser) parseFieldValue(fd *Field, s string) (interface{}, error) {
	s = strings.TrimSpace(s)

	if s == "" {
		return nil, nil
	}

	if fd.Type == FTArray {
		return p.parseArrayFieldValue(fd, s)
	} else if fd.Type == FTStruct {
		return p.parseStructFieldValue(fd, s)
	} else {
		return parsePrimitiveValue(fd.Type, s)
	}
}

func (p *Parser) parseArrayFieldValue(fd *Field, s string) (interface{}, error) {
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
	} else if fd.ElementType == FTStruct {
		sd := p.getStruct(fd.StructName)
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

func (p *Parser) getNestedField(fd *Field, fieldPath []string, depth int) *Field {
	if depth >= len(fieldPath) {
		return fd
	}

	if fd.Type == FTStruct || (fd.Type == FTArray && fd.ElementType == FTStruct) {
		sd := p.getStruct(fd.StructName)
		nestedFd := sd.getFieldByName(fieldPath[depth])
		if nestedFd == nil {
			return nil
		}
		return p.getNestedField(nestedFd, fieldPath, depth+1)
	}

	return nil
}
