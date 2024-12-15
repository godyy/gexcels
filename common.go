package gexcels

import "regexp"

// NamePattern 名称匹配模式
const NamePattern = `[A-Za-z]\w*[A-Za-z0-9]`

// nameRegexp 名称匹配正则表达式
var nameRegexp = regexp.MustCompile(NamePattern)

// MatchName 检查名称是否匹配
func MatchName(name string) bool {
	return nameRegexp.MatchString(name)
}
