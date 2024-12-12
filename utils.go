package gexcels

import (
	"reflect"
	"strconv"
	"strings"
	"unicode"
)

func toCamelCase(s string, title ...bool) string {
	var result strings.Builder
	upperNext := false // 标记下一个字符是否应该大写

	for i, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			if i == 0 {
				// 首字符小写
				if len(title) > 0 && title[0] {
					result.WriteRune(unicode.ToUpper(r))
				} else {
					result.WriteRune(unicode.ToLower(r))
				}
			} else if upperNext {
				// 每个单词的首字母大写
				result.WriteRune(unicode.ToUpper(r))
				upperNext = false
			} else {
				result.WriteRune(r)
			}
		} else {
			// 遇到分隔符，将 upperNext 标记为 true
			upperNext = true
		}
	}

	return result.String()
}

func toLowerSnake(s string) string {
	var result strings.Builder
	snake := false
	for i, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			if snake || (unicode.IsUpper(r) && i > 0) {
				result.WriteRune('_')
				snake = false
			}

			result.WriteRune(unicode.ToLower(r))
		} else {
			snake = true
		}
	}
	return result.String()
}

func titleName(s string) string {
	return strings.Title(s)
}

func convertUniqueValue2String(v interface{}) string {
	switch o := v.(type) {
	case string:
		return o
	//case int:
	//	return strconv.FormatInt(int64(o), 10)
	//case int8:
	//	return strconv.FormatInt(int64(o), 10)
	//case int16:
	//	return strconv.FormatInt(int64(o), 10)
	case int32:
		return strconv.FormatInt(int64(o), 10)
	case int64:
		return strconv.FormatInt(o, 10)
	//case uint:
	//	return strconv.FormatUint(uint64(o), 10)
	//case uint8:
	//	return strconv.FormatUint(uint64(o), 10)
	//case uint16:
	//	return strconv.FormatUint(uint64(o), 10)
	//case uint32:
	//	return strconv.FormatUint(uint64(o), 10)
	//case uint64:
	//	return strconv.FormatUint(o, 10)
	case float32:
		return strconv.FormatFloat(float64(o), 'f', -1, 32)
	case float64:
		return strconv.FormatFloat(o, 'f', -1, 32)
	case bool:
		return strconv.FormatBool(o)
	default:
		panic("invalid unique value type " + reflect.TypeOf(v).String())
	}
}
