package gexcels

import (
	"fmt"
	"os"

	pkg_errors "github.com/pkg/errors"
)

// CodeKind 导出代码类别
type CodeKind int8

const (
	_      = CodeKind(iota)
	CodeGo // golang
	codeKindMax
)

var codeKindStrings = [...]string{
	CodeGo: "go",
}

func (ck CodeKind) Valid() bool {
	return ck > 0 && ck < codeKindMax
}

func (ck CodeKind) String() string {
	return codeKindStrings[ck]
}

var codeStringKinds = map[string]CodeKind{
	CodeGo.String(): CodeGo,
}

func CodeKindFromString(s string) CodeKind {
	return codeStringKinds[s]
}

const (
	defaultStructFileName   = "structs"
	defaultTableMgrFileName = "table_manager"
)

// ExportCodeOptions 导出代码选项
type ExportCodeOptions struct {
	StructFileName   string          // 结构体文件名
	TableMgrFileName string          // 配置表管理器文件名
	DataKind         DataKind        // 数据分类
	KindOptions      CodeKindOptions // 分类选项
}

func (opt *ExportCodeOptions) isTableNameOccupied(tableName string) bool {
	return tableName != opt.StructFileName && tableName != opt.TableMgrFileName
}

// CodeKindOptions 代码分类选项
type CodeKindOptions interface {
	codeKind() CodeKind // 代码分类
}

func convertCodeKindOptions[CodeOptions CodeKindOptions](options CodeKindOptions) CodeOptions {
	codeOptions, _ := options.(CodeOptions)
	return codeOptions
}

// codeExporter 定义一个代码导出器需要具备的功能
type codeExporter interface {
	codeKind() CodeKind
	checkOptions(options CodeKindOptions) error
	export(p *Parser, path string, options *ExportCodeOptions) error
}

var codeExporters = [...]codeExporter{
	CodeGo: &goExporter{},
}

// ExportCode 将解析出的配置表导出为代码
func ExportCode(p *Parser, kind CodeKind, path string, options *ExportCodeOptions) error {
	if p == nil {
		panic("no parser specified")
	}

	if path == "" {
		return errExportPathEmpty
	}

	if !kind.Valid() {
		return errCodeKindInvalid(kind)
	}

	expo := codeExporters[kind]
	if expo == nil {
		return errExporterNotFound
	}

	if options.StructFileName == "" {
		options.StructFileName = defaultStructFileName
	}

	if options.TableMgrFileName == "" {
		options.TableMgrFileName = defaultTableMgrFileName
	}

	if !options.DataKind.Valid() {
		return errDataKindInvalid(options.DataKind)
	}

	if err := expo.checkOptions(options.KindOptions); err != nil {
		return pkg_errors.WithMessagef(err, "check %s options", kind)
	}

	if err := os.MkdirAll(path, os.ModePerm); err != nil {
		return err
	}

	return expo.export(p, path, options)
}

func exportStructName(structName string) string {
	return toCamelCase(structName, true)
}

func exportFieldName(fieldName string) string {
	return toCamelCase(fieldName, true)
}

func exportNormalTableStructName(tableName string) string {
	return fmt.Sprintf("Table%s", toCamelCase(tableName, true))
}

func exportGlobalTableStructName(tableName string) string {
	return toCamelCase(tableName, true)
}
