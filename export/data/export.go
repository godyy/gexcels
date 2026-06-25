package data

import (
	"errors"
	"fmt"
	"reflect"
	"slices"
	"sort"

	"github.com/godyy/gexcels"
	internal_define "github.com/godyy/gexcels/export"
	"github.com/godyy/gexcels/parse"
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

// baseExporter 基础导出器结构.
type baseExporter struct {
	parser *parse.Parser
}

func newBaseExporter(parser *parse.Parser) baseExporter {
	return baseExporter{parser: parser}
}

// sortMapKeys 对 map 键进行排序.
func (e *baseExporter) sortMapKeys(kt *gexcels.FieldTypeInfo, keys []reflect.Value) []reflect.Value {
	if !kt.Type.CanMapKey() {
		panic(fmt.Sprintf("export data: sortMapKeys: field type %s invalid", kt.Type.String()))
	}

	ft := kt.Type
	if ft == gexcels.FTEnum {
		ft = e.parser.GetEnum(kt.GetName()).Type
	}
	keys = slices.Clone(keys)

	sort.SliceStable(keys, func(i, j int) bool {
		switch ft {
		case gexcels.FTInt32:
			return keys[i].Int() < keys[j].Int()
		case gexcels.FTInt64:
			return keys[i].Int() < keys[j].Int()
		case gexcels.FTString:
			return keys[i].String() < keys[j].String()
		default:
			return false
		}
	})
	return keys
}
