package export

// CodeKind 导出代码类别
type CodeKind int8

const (
	_      = CodeKind(iota)
	CodeGo // golang
	codeKindMax
)

// Valid 是否有效
func (ck CodeKind) Valid() bool {
	return ck > 0 && ck < codeKindMax
}

var codeKindStrings = [...]string{
	CodeGo: "go",
}

// String 转换为字符串
func (ck CodeKind) String() string {
	return codeKindStrings[ck]
}

var codeStringKinds = map[string]CodeKind{
	CodeGo.String(): CodeGo,
}

// FromString 从字符串转换，返回是否成功
func (ck *CodeKind) FromString(s string) bool {
	if v, ok := codeStringKinds[s]; ok {
		*ck = v
		return true
	}
	return false
}
