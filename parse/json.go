package parse

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/godyy/gexcels"
	"github.com/ohler55/ojg/sen"
	pkg_errors "github.com/pkg/errors"
)

var json = &jsonHelper{}

type jsonHelper struct{}

func (j *jsonHelper) Unmarshal(b []byte, v any) error {
	return sen.Unmarshal(b, v)
}

// convertFieldJSONValue 将json raw对象转换为字段类型值.
func (p *Parser) convertFieldJSONValue(ti *gexcels.FieldTypeInfo, raw any, path string) (any, error) {
	if ti == nil {
		return nil, fmt.Errorf("%s type is nil", path)
	}

	switch ti.Type {
	case gexcels.FTInt32, gexcels.FTInt64, gexcels.FTFloat32, gexcels.FTFloat64, gexcels.FTBool, gexcels.FTString:
		return convertFieldJSONPrimitive(ti.Type, raw, path)
	case gexcels.FTEnum:
		return p.convertJSONEnumValue(ti.GetName(), raw, path)
	case gexcels.FTStruct:
		return p.convertJSONStructValue(ti.GetName(), raw, path)
	case gexcels.FTArray:
		arr, ok := raw.([]any)
		if !ok {
			return nil, fmt.Errorf("%s must be json array", path)
		}
		if len(arr) > maxArrayLength {
			return nil, errArrayLengthExceedLimit
		}
		return p.convertJSONArrayValue(ti.GetElementType(), arr, path)
	default:
		return nil, errFieldTypeInvalid(ti.Type)
	}
}

// convertFieldJSONPrimitive 将json raw对象转换为原始类型值
func convertFieldJSONPrimitive(ft gexcels.FieldType, raw any, path string) (any, error) {
	if raw == nil {
		switch ft {
		case gexcels.FTInt32:
			return int32(0), nil
		case gexcels.FTInt64:
			return int64(0), nil
		case gexcels.FTFloat32:
			return float32(0), nil
		case gexcels.FTFloat64:
			return float64(0), nil
		case gexcels.FTBool:
			return false, nil
		case gexcels.FTString:
			return "", nil
		default:
			return nil, errFieldTypeInvalid(ft)
		}
	}

	switch ft {
	case gexcels.FTInt32:
		i64, err := convertJSONInt64(raw, path)
		if err != nil {
			return nil, err
		}
		if i64 < math.MinInt32 || i64 > math.MaxInt32 {
			return nil, fmt.Errorf("%s out of int32 range", path)
		}
		return int32(i64), nil
	case gexcels.FTInt64:
		return convertJSONInt64(raw, path)
	case gexcels.FTFloat32:
		f64, err := convertJSONFloat64(raw, path)
		if err != nil {
			return nil, err
		}
		if f64 > math.MaxFloat32 || f64 < -math.MaxFloat32 {
			return nil, fmt.Errorf("%s out of float32 range", path)
		}
		return float32(f64), nil
	case gexcels.FTFloat64:
		return convertJSONFloat64(raw, path)
	case gexcels.FTBool:
		switch v := raw.(type) {
		case bool:
			return v, nil
		case string:
			b, err := strconv.ParseBool(strings.TrimSpace(v))
			if err != nil {
				return nil, fmt.Errorf("%s invalid bool", path)
			}
			return b, nil
		default:
			return nil, fmt.Errorf("%s must be bool", path)
		}
	case gexcels.FTString:
		s, ok := raw.(string)
		if !ok {
			return nil, fmt.Errorf("%s must be string", path)
		}
		if len(s) > maxStringLength {
			return nil, pkg_errors.WithMessagef(errStringLengthExceedLimit, "field %s", path)
		}
		return s, nil
	default:
		return nil, errFieldTypeInvalid(ft)
	}
}

// convertJSONInt64 将json raw对象转换为int64类型值
func convertJSONInt64(raw any, path string) (int64, error) {
	switch v := raw.(type) {
	case int64:
		return v, nil
	case int32:
		return int64(v), nil
	case int:
		return int64(v), nil
	case float64:
		if v != math.Trunc(v) {
			return 0, fmt.Errorf("%s must be integer", path)
		}
		if v < math.MinInt64 || v > math.MaxInt64 {
			return 0, fmt.Errorf("%s out of int64 range", path)
		}
		return int64(v), nil
	case float32:
		f := float64(v)
		if f != math.Trunc(f) {
			return 0, fmt.Errorf("%s must be integer", path)
		}
		return int64(f), nil
	case string:
		s := strings.TrimSpace(v)
		if s == "" {
			return 0, nil
		}
		i, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("%s invalid int", path)
		}
		return i, nil
	default:
		return 0, fmt.Errorf("%s must be number", path)
	}
}

// convertJSONFloat64 将json raw对象转换为float64类型值
func convertJSONFloat64(raw any, path string) (float64, error) {
	switch v := raw.(type) {
	case float64:
		return v, nil
	case float32:
		return float64(v), nil
	case int64:
		return float64(v), nil
	case int32:
		return float64(v), nil
	case int:
		return float64(v), nil
	case string:
		s := strings.TrimSpace(v)
		if s == "" {
			return 0, nil
		}
		f, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return 0, fmt.Errorf("%s invalid float", path)
		}
		return f, nil
	default:
		return 0, fmt.Errorf("%s must be number", path)
	}
}

// convertJSONEnumValue 将json raw对象转换为枚举类型值
func (p *Parser) convertJSONEnumValue(enumName string, raw any, path string) (any, error) {
	enum := p.GetEnum(enumName)
	if enum == nil {
		return nil, errEnumNotDefine(enumName)
	}

	s, ok := raw.(string)
	if !ok {
		return nil, fmt.Errorf("%s must be string", path)
	}
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, fmt.Errorf("%s is empty", path)
	}

	val, ok := enum.GetItemValueByName(s)
	if !ok {
		return nil, errEnumItemNotDefine(enumName, s)
	}
	return val, nil
}

// convertJSONStructValue 将json raw对象转换为结构体类型值, 结构体类型用 map[string]any 替代.
func (p *Parser) convertJSONStructValue(structName string, raw any, path string) (any, error) {
	sd := p.GetStructByName(structName)
	if sd == nil {
		return nil, errStructNotDefine(structName)
	}

	m, ok := raw.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("%s must be json object", path)
	}

	out := make(map[string]any)
	for _, fd := range sd.Fields {
		fieldPath := path + "." + fd.Name
		v, ok := m[fd.Name]
		if !ok || v == nil {
			if fd.Type == gexcels.FTEnum {
				return nil, fmt.Errorf("%s must be string", fieldPath)
			}
			continue
		}

		c, err := p.convertFieldJSONValue(fd.FieldTypeInfo, v, fieldPath)
		if err != nil {
			return nil, err
		}
		if c != nil {
			out[fd.Name] = c
		}
	}

	return out, nil
}

// convertJSONArrayValue 将json raw对象转换为数组类型值.
func (p *Parser) convertJSONArrayValue(elem *gexcels.FieldTypeInfo, raw any, path string) (any, error) {
	if elem == nil {
		return nil, fmt.Errorf("%s element type is nil", path)
	}

	arr, ok := raw.([]any)
	if !ok {
		return nil, fmt.Errorf("%s must be json array", path)
	}
	if len(arr) > maxArrayLength {
		return nil, errArrayLengthExceedLimit
	}

	switch elem.Type {
	case gexcels.FTInt32:
		out := make([]int32, len(arr))
		for i, v := range arr {
			c, err := convertFieldJSONPrimitive(gexcels.FTInt32, v, fmt.Sprintf("%s[%d]", path, i))
			if err != nil {
				return nil, err
			}
			out[i] = c.(int32)
		}
		return out, nil
	case gexcels.FTInt64:
		out := make([]int64, len(arr))
		for i, v := range arr {
			c, err := convertFieldJSONPrimitive(gexcels.FTInt64, v, fmt.Sprintf("%s[%d]", path, i))
			if err != nil {
				return nil, err
			}
			out[i] = c.(int64)
		}
		return out, nil
	case gexcels.FTFloat32:
		out := make([]float32, len(arr))
		for i, v := range arr {
			c, err := convertFieldJSONPrimitive(gexcels.FTFloat32, v, fmt.Sprintf("%s[%d]", path, i))
			if err != nil {
				return nil, err
			}
			out[i] = c.(float32)
		}
		return out, nil
	case gexcels.FTFloat64:
		out := make([]float64, len(arr))
		for i, v := range arr {
			c, err := convertFieldJSONPrimitive(gexcels.FTFloat64, v, fmt.Sprintf("%s[%d]", path, i))
			if err != nil {
				return nil, err
			}
			out[i] = c.(float64)
		}
		return out, nil
	case gexcels.FTBool:
		out := make([]bool, len(arr))
		for i, v := range arr {
			c, err := convertFieldJSONPrimitive(gexcels.FTBool, v, fmt.Sprintf("%s[%d]", path, i))
			if err != nil {
				return nil, err
			}
			out[i] = c.(bool)
		}
		return out, nil
	case gexcels.FTString:
		out := make([]string, len(arr))
		for i, v := range arr {
			c, err := convertFieldJSONPrimitive(gexcels.FTString, v, fmt.Sprintf("%s[%d]", path, i))
			if err != nil {
				return nil, err
			}
			out[i] = c.(string)
		}
		return out, nil
	case gexcels.FTEnum:
		out := make([]any, len(arr))
		for i, v := range arr {
			c, err := p.convertJSONEnumValue(elem.GetName(), v, fmt.Sprintf("%s[%d]", path, i))
			if err != nil {
				return nil, err
			}
			out[i] = c
		}
		return out, nil
	case gexcels.FTStruct:
		out := make([]any, len(arr))
		for i, v := range arr {
			c, err := p.convertJSONStructValue(elem.GetName(), v, fmt.Sprintf("%s[%d]", path, i))
			if err != nil {
				return nil, err
			}
			out[i] = c
		}
		return out, nil
	default:
		return nil, errArrayElementInvalid(elem.Type)
	}
}
