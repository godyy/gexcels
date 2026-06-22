package parse

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/godyy/gexcels"
	"github.com/godyy/gutils/buffer/bytes"
)

const (
	maxStringLength = bytes.MaxStringLength // 支持的string最大长度
	maxArrayLength  = 32767                 // 最大数组长度
)

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

	var raw any
	if err := json.Unmarshal(([]byte)(s), &raw); err != nil {
		return nil, err
	}

	return p.convertJSONArrayValue(fd.GetElementType(), raw, fd.Name)
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
