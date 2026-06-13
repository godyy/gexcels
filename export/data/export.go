package data

import (
	"errors"

	internal_define "github.com/godyy/gexcels/export"
)

// ErrNoParserSpecified 未指定解析器
var ErrNoParserSpecified = errors.New("export data: no parser specified")

// ErrNoPathSpecified 未指定导出路径
var ErrNoPathSpecified = errors.New("export data: no path specified")

// exporter 定义数据导出器需具备的功能
type exporter interface {
	kind() internal_define.DataKind
	export() error
}

// doExport 执行导出逻辑.
func doExport(expo exporter) error {
	if err := expo.export(); err != nil {
		return err
	}
	return nil
}

// genTableFileName 生成表文件名
func genTableFileName(tableName string, ext string) string {
	fileName := tableName
	fileName += ext
	return fileName
}
