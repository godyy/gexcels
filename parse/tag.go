package parse

import (
	"regexp"
)

// tagPattern 标签模式
const tagPattern = `\w*`

// tagSep 标签分隔符
const tagSep = "/"

// tagRegexp 标签格式正则表达式
var tagRegexp = regexp.MustCompile(tagPattern)

// Tag 标签
// 用于匹配需要导出的配置表字段或过滤表文件名
type Tag string

// TagEmpty 空标签
const TagEmpty = Tag("")

// TagAny 任意标签
const TagAny = Tag("*")

// Valid 返回标签格式是否有效
func (t Tag) Valid() bool {
	return tagRegexp.MatchString(string(t))
}
