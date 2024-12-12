package gexcels

import (
	"fmt"
	"io/fs"
	"math"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"

	pkg_errors "github.com/pkg/errors"
	"github.com/tealeg/xlsx/v3"
)

// ParseOptions 解析选项
type ParseOptions struct {
	Tag            []string // 字段标签
	StructFilePath string   // 结构体定义文件路径
}

func defaultParseOptions() *ParseOptions {
	return &ParseOptions{
		Tag:            nil,
		StructFilePath: "",
	}
}

// Parse excel配置表解析
// path 指定excel文件目录。
// tag 指定字段标签，用于筛选需要导出的字段。
func Parse(path string, options ...*ParseOptions) (*Parser, error) {
	if path == "" {
		return nil, errNoPathSpecified
	}

	var opts *ParseOptions
	if len(options) > 0 {
		opts = options[0]
	}
	if opts == nil {
		opts = defaultParseOptions()
	}

	printf("parse file inside [%s] with tag(%v)", path, opts.Tag)

	p := newParser(path, opts)

	if err := p.parseStructs(); err != nil {
		return nil, err
	}

	if err := filepath.Walk(path, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && filepath.Ext(info.Name()) == ".xlsx" && (p.structFilePath == "" || path != p.structFilePath) {
			file, err := xlsx.OpenFile(path)
			if err != nil {
				return err
			}
			for _, sheet := range file.Sheets {
				if !tableSheetNameRegexp.MatchString(sheet.Name) {
					continue
				}
				td, err := p.parseTableOfSheet(sheet)
				if err != nil {
					return pkg_errors.WithMessagef(err, "[%s][%s]", path, sheet.Name)
				}
				if p.hasTable(td.Name) {
					return fmt.Errorf("[%s][%s]: table name duplicate", path, sheet.Name)
				}
				p.addTable(td)
				printfGreen("[%s][%s] parsed", path, sheet.Name)
			}
		}
		return nil
	}); err != nil {
		return nil, err
	}

	if errs := p.checkLinksBetweenTable(); errs != nil {
		for _, err := range errs {
			errorln(err)
		}
		return nil, errLinkErrorsFound
	}

	return p, nil
}

// Parser excel配置表解析器
type Parser struct {
	path           string         // 配置路径
	options        *ParseOptions  // 选项
	structFilePath string         // 结构体定义文件路径
	structs        []*Struct      // 解析出的结构体
	structByName   map[string]int // 结构体名称映射
	tables         []*Table       // 解析出的配置表
	tableByName    map[string]int // 配置表名称映射
}

func newParser(path string, options *ParseOptions) *Parser {
	p := &Parser{
		path:         path,
		options:      options,
		structs:      make([]*Struct, 0),
		structByName: make(map[string]int),
		tables:       make([]*Table, 0),
		tableByName:  make(map[string]int),
	}

	if options.StructFilePath != "" {
		p.structFilePath = filepath.Join(path, options.StructFilePath)
	}

	return p
}

// 检查Tag是否满足need
func (p *Parser) checkTagNeed(tag string) bool {
	if len(p.options.Tag) <= 0 {
		return true
	}
	for _, v := range p.options.Tag {
		if strings.Contains(tag, v) {
			return true
		}
	}
	return false
}

// parsePrimitiveValue 解析primitive字段值
func parsePrimitiveValue(ft FieldType, s string) (interface{}, error) {
	switch ft {
	case FTInt32:
		s = strings.TrimSpace(s)
		if s == "" {
			return int32(0), nil
		}
		i, err := strconv.ParseInt(s, 10, 32)
		if err != nil {
			return nil, err
		}
		return int32(i), nil

	case FTInt64:
		s = strings.TrimSpace(s)
		if s == "" {
			return int64(0), nil
		}
		i, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return nil, err
		}
		return int64(i), nil

	case FTFloat32:
		s = strings.TrimSpace(s)
		if s == "" {
			return float32(0), nil
		}
		f, err := strconv.ParseFloat(s, 32)
		if err != nil {
			return nil, err
		}
		return float32(f), nil

	case FTFloat64:
		if s == "" {
			return float64(0), nil
		}
		f, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return nil, err
		}
		return float64(f), nil

	case FTBool:
		if s == "" {
			return false, nil
		}
		b, err := strconv.ParseBool(s)
		if err != nil {
			return nil, err
		}
		return b, nil

	case FTString:
		if len(s) > maxStringLength {
			return nil, errStringLengthExceedLimit
		}
		return s, nil
	default:
		return nil, errFieldTypeInvalid(ft)
	}
}

// 构造元素为primitive的数组
func makePrimitiveArray(ft FieldType) interface{} {
	switch ft {
	case FTInt32:
		return make([]int32, 0)
	case FTInt64:
		return make([]int64, 0)
	case FTFloat32:
		return make([]float32, 0)
	case FTFloat64:
		return make([]float64, 0)
	case FTBool:
		return make([]bool, 0)
	case FTString:
		return make([]string, 0)
	default:
		panic(errArrayElementInvalid(ft))
	}
}

// 检查primitive字段值
func checkPrimitiveValue(ft FieldType, val interface{}) error {
	switch ft {
	case FTInt32, FTInt64:
		switch val.(type) {
		case int8, int16, int32, int64, uint8, uint16, uint32, uint64:
			return nil
		case float32:
			f := float64(val.(float32))
			if math.Trunc(f) == f {
				return nil
			}
		case float64:
			f := val.(float64)
			if math.Trunc(f) == f {
				return nil
			}
		}
	case FTFloat32, FTFloat64:
		switch val.(type) {
		case int8, int16, int32, int64, uint8, uint16, uint32, uint64, float32, float64:
			return nil
		}
	case FTBool:
		switch val.(type) {
		case bool:
			return nil
		}
	case FTString:
		switch val.(type) {
		case string:
			return nil
		}
	default:
		return errFieldTypeInvalid(ft)
	}
	return fmt.Errorf("value {%v} not match type %s", val, ConvertPrimitiveFieldType2String(ft))
}

// 尝试将val转换为ft指定的类型
func convertValue2PrimitiveFieldType(val interface{}, ft FieldType) (interface{}, error) {
	switch ft {
	case FTInt32:
		switch val.(type) {
		case int8, int16, int32, int64, uint8, uint16, uint32, uint64:
			return reflect.ValueOf(val).Convert(reflect.TypeOf(int32(0))).Interface(), nil
		case float32:
			f := float64(val.(float32))
			if math.Trunc(f) == f {
				return int32(f), nil
			}
		case float64:
			f := val.(float64)
			if math.Trunc(f) == f {
				return int32(f), nil
			}
		}

	case FTInt64:
		switch val.(type) {
		case int8, int16, int32, int64, uint8, uint16, uint32, uint64:
			return reflect.ValueOf(val).Convert(reflect.TypeOf(int64(0))).Interface(), nil
		case float32:
			f := float64(val.(float32))
			if math.Trunc(f) == f {
				return int64(f), nil
			}
		case float64:
			f := val.(float64)
			if math.Trunc(f) == f {
				return int64(f), nil
			}
		}

	case FTFloat32:
		v := reflect.ValueOf(val)
		t := reflect.TypeOf(float32(0))
		if v.CanConvert(t) {
			return v.Convert(t).Interface(), nil
		}
	case FTFloat64:
		v := reflect.ValueOf(val)
		t := reflect.TypeOf(float64(0))
		if v.CanConvert(t) {
			return v.Convert(t).Interface(), nil
		}

	case FTBool:
		if v, ok := val.(bool); ok {
			return v, nil
		}

	case FTString:
		if v, ok := val.(string); ok {
			return v, nil
		}

	default:
		return nil, errFieldTypeInvalid(ft)
	}

	return nil, fmt.Errorf("value (%v) cant convert to type %s", val, ConvertPrimitiveFieldType2String(ft))
}

// 检查tag是否合法
func checkTagValid(tag string) bool {
	return tag == "" || tagRegexp.MatchString(tag)
}

// 获取sheet中的value
func getSheetValue(sheet *xlsx.Sheet, row, col int, trim ...bool) (string, error) {
	cell, err := sheet.Cell(row, col)
	if err != nil {
		return "", err
	}

	if len(trim) > 0 && trim[0] {
		return strings.TrimSpace(cell.Value), nil
	} else {
		return cell.Value, nil
	}
}

// 是否满足注释格式
func isComment(s string) bool {
	return commentRegexp.MatchString(s)
}
