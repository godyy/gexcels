package code

import (
	"errors"
	"fmt"
	"github.com/godyy/gexcels/export"
	"github.com/godyy/gexcels/internal/utils"
	"github.com/godyy/gexcels/parse"
	pkg_errors "github.com/pkg/errors"
	"os"
)

// ErrNoParserSpecified 未指定解析器
var ErrNoParserSpecified = errors.New("export code: no parser specified")

// ErrNoPathSpecified 未指定导出路径
var ErrNoPathSpecified = errors.New("export code: no path specified")

// ErrNoOptionsSpecified 未指定选项
var ErrNoOptionsSpecified = errors.New("export code: no options specified")

// ErrNoKindOptionsSpecified 未指定代码分类选项
var ErrNoKindOptionsSpecified = errors.New("export code: no kind options specified")

const (
	defaultTableManagerName = "Manager" // 默认的管理器名称
)

// Options 导出代码选项
type Options struct {
	DataKind         export.DataKind // 数据分类
	TableManagerName string          // 代码中的表管理器名称 default by defaultTableManagerName
}

func (opt *Options) init() {
	if opt.TableManagerName == "" {
		opt.TableManagerName = defaultTableManagerName
	} else {
		opt.TableManagerName = utils.CamelCase(opt.TableManagerName, true)
	}
}

// kindOptions 代码分类选项
type kindOptions interface {
	kind() export.CodeKind // 代码分类
}

// exporter 定义一个代码导出器需要具备的功能
type exporter interface {
	kind() export.CodeKind
	export() error
}

// exporterBase 导出器基础数据
type exporterBase struct {
	parser  *parse.Parser // 解析出的数据
	path    string        // 导出路径
	options *Options      // 选项
}

func newExporterBase(parser *parse.Parser, path string, options *Options) exporterBase {
	return exporterBase{
		parser:  parser,
		path:    path,
		options: options,
	}
}

// creator 导出器构造函数
type creator func(parser *parse.Parser, path string, options *Options, kindOptions kindOptions) (exporter, error)

// creators 导出器构造函数映射
var creators = map[export.CodeKind]creator{
	export.CodeGo: createGoExporter,
}

// doExport 将解析出的配置表导出为代码
func doExport(p *parse.Parser, path string, options *Options, kindOptions kindOptions) error {
	if p == nil {
		return ErrNoParserSpecified
	}

	if path == "" {
		return ErrNoPathSpecified
	}

	if options == nil {
		return ErrNoOptionsSpecified
	}

	if !options.DataKind.Valid() {
		return fmt.Errorf("export code: code-kind %d invalid", options.DataKind)
	}
	options.init()

	if kindOptions == nil {
		return ErrNoKindOptionsSpecified
	}

	if !kindOptions.kind().Valid() {
		return fmt.Errorf("export code: code-kind %d invalid", kindOptions.kind())
	}

	creator := creators[kindOptions.kind()]
	if creator == nil {
		return fmt.Errorf("export code: %s: exporter not found", kindOptions.kind())
	}

	expo, err := creator(p, path, options, kindOptions)
	if err != nil {
		return pkg_errors.WithMessagef(err, "export code: %s: create exporter failed", kindOptions.kind())
	}

	if err := os.MkdirAll(path, os.ModePerm); err != nil {
		return err
	}

	return expo.export()
}
