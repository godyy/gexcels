package gexcels

import (
	"regexp"
)

// Tag 标签
// 用于匹配需要导出的配置表字段或过滤表文件名
type Tag string

const (
	// TagEmpty 空标签
	TagEmpty = Tag("")

	// TagAny 任意标签
	TagAny = Tag("*")
)

// TagPattern 标签模式
const TagPattern = `\w*`

// TagSep 标签分隔符
const TagSep = "/"

// tagRegexp 标签格式正则表达式
var tagRegexp = regexp.MustCompile(TagPattern)

// Valid 返回标签格式是否有效
func (t Tag) Valid() bool {
	return tagRegexp.MatchString(string(t))
}
