package parse

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/godyy/gexcels"
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
		if len(s) > gexcels.MaxStringLen {
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

// parseFieldTypeInfo 解析字段类型信息
func (p *Parser) parseFieldTypeInfo(typeStr string) (*gexcels.FieldTypeInfo, error) {
	var parse func(s string) (*gexcels.FieldTypeInfo, error)
	parse = func(s string) (*gexcels.FieldTypeInfo, error) {
		s = strings.TrimSpace(s)
		if s == "" {
			return nil, fmt.Errorf("parseFieldTypeInfo: empty type")
		}

		if strings.HasPrefix(s, "[]") {
			// 数组类型：元素类型继续递归解析
			elem, err := parse(s[2:])
			if err != nil {
				return nil, err
			}
			return gexcels.NewArrayFieldTypeInfo(elem), nil
		}

		if strings.HasPrefix(s, "map[") {
			// map 类型：map[kt]vt
			// 约束：Excel 单元格用 JSON object 表达，key 天然是 string。
			// 这里支持将该 string key 转换为 primitive/enum 作为真实 key 类型。
			end := strings.IndexByte(s, ']')
			if end <= len("map[") {
				return nil, fmt.Errorf("parseFieldTypeInfo: map key type invalid: %s", s)
			}
			keyStr := strings.TrimSpace(s[len("map["):end])
			valStr := strings.TrimSpace(s[end+1:])
			if valStr == "" {
				return nil, fmt.Errorf("parseFieldTypeInfo: map value type empty: %s", s)
			}

			var kt *gexcels.FieldTypeInfo
			if keyType := gexcels.ParsePrimitiveFieldType(keyStr); keyType != gexcels.FTUnknown {
				kt = gexcels.NewPrimitiveFieldTypeInfo(keyType)
			} else {
				if !gexcels.MatchName(keyStr) {
					return nil, fmt.Errorf("parseFieldTypeInfo: map key type %s invalid", keyStr)
				}
				customFieldTypeInfo := p.getCustomFieldType(keyStr)
				if customFieldTypeInfo == nil {
					return nil, fmt.Errorf("parseFieldTypeInfo: map key custom field type %s not found", keyStr)
				}
				kt = customFieldTypeInfo
			}
			if !kt.Type.CanMapKey() {
				return nil, fmt.Errorf("parseFieldTypeInfo: map key type %s invalid", keyStr)
			}

			vt, err := parse(valStr)
			if err != nil {
				return nil, err
			}
			return gexcels.NewMapFieldTypeInfo(kt, vt), nil
		}

		if ft := gexcels.ParsePrimitiveFieldType(s); ft != gexcels.FTUnknown {
			return gexcels.NewPrimitiveFieldTypeInfo(ft), nil
		}

		if !gexcels.MatchName(s) {
			return nil, fmt.Errorf("parseFieldTypeInfo: custom field type name %s invalid", s)
		}
		customFieldTypeInfo := p.getCustomFieldType(s)
		if customFieldTypeInfo == nil {
			return nil, fmt.Errorf("parseFieldTypeInfo: custom field type %s not found", s)
		}
		return customFieldTypeInfo, nil
	}

	return parse(typeStr)
}

// parseFieldValue 解析字段值
func (p *Parser) parseFieldValue(fd *gexcels.Field, s string) (any, error) {
	s = strings.TrimSpace(s)

	if fd.Type == gexcels.FTEnum {
		return p.parseEnumFieldValue(fd, s)
	} else if fd.Type == gexcels.FTArray {
		return p.parseArrayFieldValue(fd, s)
	} else if fd.Type == gexcels.FTMap {
		return p.parseMapFieldValue(fd, s)
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

	return p.convertJSONArrayValue(fd.FieldTypeInfo, raw, fd.Name)
}

// parseMapFieldValue 解析 map 字段值
func (p *Parser) parseMapFieldValue(fd *gexcels.Field, s string) (any, error) {
	if s == "" {
		return nil, nil
	}

	// map 的值采用 JSON object 表达，并复用 json 转换链路做递归类型转换
	var raw any
	if err := json.Unmarshal(([]byte)(s), &raw); err != nil {
		return nil, err
	}
	return p.convertFieldJSONValue(fd.FieldTypeInfo, raw, fd.Name)
}
