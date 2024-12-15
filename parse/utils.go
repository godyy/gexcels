package parse

import "github.com/godyy/gexcels/internal/utils"

// GenStructFieldName 生成结构体的字段名
func GenStructFieldName(fieldName string) string {
	return utils.CamelCase(fieldName, true)
}
