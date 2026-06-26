package parse

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/godyy/gexcels"
	"github.com/godyy/gexcels/internal/log"
	pkg_errors "github.com/pkg/errors"
	"github.com/tealeg/xlsx/v3"
)

// Options 解析选项
type Options struct {
	// EnumFiles 指定枚举文件.
	// 一旦指定了枚举文件, 系统将不再启动默认枚举文件解析功能.
	EnumFiles []string

	// StructFiles 指定结构体文件.
	// 文件按照给定的顺序进行解析.
	// 一旦指定了结构体文件, 系统将不再启动默认结构体文件解析功能.
	StructFiles []string

	// Tags 用于匹配字段或结构体定义的标签.
	// 默认为空，匹配空标签.
	// 如果包含 gexcels.TagAny, 可匹配任意标签，包括空标签.
	// 如果想在匹配其它标签时匹配空标签，需要在 Tags 中包含 gexcels.TagEmpty.
	Tags []gexcels.Tag

	// FieldRuleSep 字段规则分隔符，默认为'|'
	FieldRuleSep string

	// OnlyFields 是否仅解析字段定义, 默认为 false
	OnlyFields bool

	tagMap        map[gexcels.Tag]bool
	enumFileMap   map[string]bool
	structFileMap map[string]bool
}

func (opt *Options) init() error {
	if len(opt.EnumFiles) > 0 {
		opt.enumFileMap = make(map[string]bool, len(opt.EnumFiles))
		for _, file := range opt.EnumFiles {
			abs, _ := filepath.Abs(file)
			opt.enumFileMap[abs] = true
		}
	}
	if len(opt.StructFiles) > 0 {
		opt.structFileMap = make(map[string]bool, len(opt.StructFiles))
		for _, file := range opt.StructFiles {
			abs, _ := filepath.Abs(file)
			opt.structFileMap[abs] = true
		}
	}

	opt.tagMap = make(map[gexcels.Tag]bool, len(opt.Tags))
	for _, tag := range opt.Tags {
		if !tag.Valid() {
			return fmt.Errorf("parse: tag %s invalid", tag)
		}
		opt.tagMap[tag] = true
	}
	if len(opt.tagMap) == 0 {
		opt.tagMap[gexcels.TagEmpty] = true
	}

	if opt.FieldRuleSep == "" {
		opt.FieldRuleSep = "|"
	}

	return nil
}

// isEnumFile 是否为枚举文件
func (opt *Options) isEnumFile(path string) bool {
	if opt.enumFileMap == nil {
		return false
	}
	abs, _ := filepath.Abs(path)
	return opt.enumFileMap[abs]
}

// isStructFile 是否为结构体文件
func (opt *Options) isStructFile(path string) bool {
	if opt.structFileMap == nil {
		return false
	}
	abs, _ := filepath.Abs(path)
	return opt.structFileMap[abs]
}

// checkTag 检查tag
func (opt *Options) checkTag(tags []string) bool {
	if opt.tagMap[gexcels.TagAny] {
		return true
	}
	if len(tags) == 0 {
		return opt.tagMap[gexcels.TagEmpty]
	}
	for _, tag := range tags {
		if opt.tagMap[gexcels.Tag(tag)] {
			return true
		}
	}
	return false
}

func defaultOptions() *Options {
	return &Options{}
}

// Parse excel配置表解析
// path 指定excel文件目录。
// tag 指定字段标签，用于筛选需要导出的字段。
func Parse(path string, options ...*Options) (*Parser, error) {
	if path == "" {
		return nil, ErrNoPathSpecified
	}

	var opts *Options
	if len(options) > 0 {
		opts = options[0]
	}
	if opts == nil {
		opts = defaultOptions()
	}
	if err := opts.init(); err != nil {
		return nil, err
	}

	log.Printf("parse file inside [%s] with tag(%v)", path, opts.Tags)

	p := newParser(path, opts)
	if err := p.parse(); err != nil {
		return nil, err
	}

	return p, nil
}

// Parser excel配置表解析器
type Parser struct {
	path             string                            // 配置路径
	options          *Options                          // 选项
	Structs          []*Struct                         // 解析出的结构体
	structByName     map[string]*Struct                // 结构体名称映射
	Tables           []*Table                          // 解析出的配置表
	tableByName      map[string]*Table                 // 配置表名称映射
	Enums            []*Enum                           // 枚举列表
	enumByName       map[string]*Enum                  // 枚举名称映射
	customFieldTypes map[string]*gexcels.FieldTypeInfo // 自定义字段类型
}

func newParser(path string, options *Options) *Parser {
	p := &Parser{
		path:             path,
		options:          options,
		Structs:          make([]*Struct, 0),
		structByName:     make(map[string]*Struct),
		Tables:           make([]*Table, 0),
		tableByName:      make(map[string]*Table),
		Enums:            make([]*Enum, 0),
		enumByName:       make(map[string]*Enum),
		customFieldTypes: make(map[string]*gexcels.FieldTypeInfo),
	}
	return p
}

// checkTag 检查tag
func (p *Parser) checkTag(tag string) (bool, bool) {
	if !tableTagRegexp.MatchString(tag) {
		return false, false
	}
	tags := strings.Split(tag, gexcels.TagSep)
	return p.options.checkTag(tags), true
}

func (p *Parser) parse() error {
	result := p.searchFiles()
	if result.err != nil {
		return pkg_errors.WithMessage(result.err, "parse")
	}

	enumFiles := p.options.EnumFiles
	if len(enumFiles) == 0 {
		enumFiles = result.enumFiles
	}
	if err := p.parseEnums(enumFiles); err != nil {
		return pkg_errors.WithMessage(err, "parse")
	}

	structFiles := p.options.StructFiles
	if len(structFiles) == 0 {
		structFiles = result.structFiles
	}
	if err := p.parseStructs(structFiles); err != nil {
		return pkg_errors.WithMessage(err, "parse")
	}

	if err := p.parseTables(result.tableFiles); err != nil {
		return pkg_errors.WithMessage(err, "parse")
	}

	if errs := p.checkLinksBetweenTable(); errs != nil {
		for _, err := range errs {
			log.Errorln(err)
		}
		return ErrLinkErrorsFound
	}

	return nil
}

// priorityFileInfo 枚举文件信息
type priorityFileInfo struct {
	path     string // 路径
	priority int    // 优先级
}

// structFileNameRegexp 默认结构体文件名匹配正则表达式
var structFileNameRegexp = regexp.MustCompile(`^(?:[^.]*\.)?struct(?:\.([0-9]*))?$`)

// enumFileNameRegexp 默认枚举文件名匹配正则表达式
var enumFileNameRegexp = regexp.MustCompile(`^(?:[^.]*\.)?enum$`)

// searchFiles 搜索文件
func (p *Parser) searchFiles() (result struct {
	enumFiles   []string
	structFiles []string
	tableFiles  []string
	err         error
}) {
	var (
		structFiles  []priorityFileInfo
		searchEnum   = len(p.options.EnumFiles) == 0
		searchStruct = len(p.options.StructFiles) == 0
	)

	if err := filepath.Walk(p.path, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		fileName := filepath.Base(path)
		ext := filepath.Ext(fileName)

		if ext != ".xlsx" {
			return nil
		}

		if p.options.isEnumFile(path) || p.options.isStructFile(path) {
			return nil
		}

		fileName = strings.TrimSuffix(fileName, ext)

		// 匹配枚举文件
		if searchEnum {
			if matches := enumFileNameRegexp.FindStringSubmatch(fileName); len(matches) > 0 {
				result.enumFiles = append(result.enumFiles, path)
				return nil
			}
		}

		// 匹配结构体文件
		if searchStruct {
			if matches := structFileNameRegexp.FindStringSubmatch(fileName); len(matches) > 0 {
				priority, _ := strconv.Atoi(matches[1])
				structFiles = append(structFiles, priorityFileInfo{
					path:     path,
					priority: priority,
				})
				return nil
			}
		}

		// 匹配配置表文件
		result.tableFiles = append(result.tableFiles, path)

		return nil
	}); err != nil {
		result.err = pkg_errors.WithMessage(err, "search files")
		return
	}

	if len(structFiles) > 0 {
		sort.SliceStable(structFiles, func(i, j int) bool {
			return structFiles[i].priority < structFiles[j].priority
		})
		result.structFiles = make([]string, len(structFiles))
		for i, file := range structFiles {
			result.structFiles[i] = file.path
		}
	}

	return
}

// prioritySheetInfo 结构体sheet信息
type prioritySheetInfo struct {
	*xlsx.Sheet     // sheet
	priority    int // 优先级
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

// commentRegexp 注释匹配正则表达式
var commentRegexp = regexp.MustCompile(`^#[\s\S]*$`)

// 是否注释行
func isRowComment(row *xlsx.Row) bool {
	value := row.GetCell(0).Value
	return isValueComment(value)
}

// isSheetRowComment 是否是注释行
func isSheetRowComment(sheet *xlsx.Sheet, row int) bool {
	if row > sheet.MaxRow || sheet.MaxCol < 1 {
		return false
	}
	cell, _ := getSheetValue(sheet, row, 0, true)
	return isValueComment(cell)
}

// isValueComment 是否注释值
func isValueComment(value string) bool {
	return commentRegexp.MatchString(value)
}
