package data

import (
	"errors"
	"fmt"
	internal_define "github.com/godyy/gexcels/export/internal/define"
	"github.com/godyy/gexcels/parse"
	pkg_errors "github.com/pkg/errors"
	"os"
)

// ErrNoParserSpecified 未指定解析器
var ErrNoParserSpecified = errors.New("export data: no parser specified")

// ErrNoPathSpecified 未指定导出路径
var ErrNoPathSpecified = errors.New("export data: no path specified")

// kindOptions 分类选项
type kindOptions interface {
	kind() internal_define.DataKind
}

// exporter 定义数据导出器需具备的功能
type exporter interface {
	kind() internal_define.DataKind
	export() error
}

// exporterBase 定义导出器基础数据
type exporterBase struct {
	parser *parse.Parser // 解析出的数据
	path   string        // 导出路径
}

func newExporterBase(parser *parse.Parser, path string) exporterBase {
	return exporterBase{
		parser: parser,
		path:   path,
	}
}

// creator 导出器构造函数
type creator func(p *parse.Parser, path string, options kindOptions) (exporter, error)

// creators 导出器构造函数映射
var creators = [...]creator{
	internal_define.DataJson:  createJsonExporter,
	internal_define.DataBytes: createBytesExporter,
}

// doExport 导出数据到指定路径
func doExport(p *parse.Parser, path string, options kindOptions) error {
	if p == nil {
		return ErrNoParserSpecified
	}

	if path == "" {
		return ErrNoPathSpecified
	}

	if !options.kind().Valid() {
		return fmt.Errorf("export data: data-kind %d invalid", options.kind())
	}

	creator := creators[options.kind()]
	if creator == nil {
		return fmt.Errorf("export data: data %s exporter not found", options.kind())
	}

	expo, err := creator(p, path, options)
	if err != nil {
		return pkg_errors.WithMessagef(err, "export data: %s: create exporter faield", options.kind())
	}

	if err := os.MkdirAll(path, os.ModePerm); err != nil {
		return err
	}

	return expo.export()
}

// genTableFileName 生成表文件名
func genTableFileName(tableName string, tag parse.Tag, ext string) string {
	fileName := tableName
	if tag != "" {
		fileName += "." + string(tag)
	}
	fileName += ext
	return fileName
}
