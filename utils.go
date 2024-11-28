package gexcels

import (
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
