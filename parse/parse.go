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
	// Tags 用于匹配字段或结构体定义的标签.
	// 默认为空，匹配空标签.
	// 如果包含 gexcels.TagAny, 可匹配任意标签，包括空标签.
	// 如果想在匹配其它标签时匹配空标签，需要在 Tags 中包含 gexcels.TagEmpty.
	Tags []gexcels.Tag

	// FieldRuleSep 字段规则分隔符，默认为'|'
	FieldRuleSep string

	// OnlyFields 是否仅解析字段定义, 默认为 false
	OnlyFields bool

	tagMap map[gexcels.Tag]bool
}

func (opt *Options) init() error {
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

	if err := p.parseEnums(result.enumFiles); err != nil {
		return pkg_errors.WithMessage(err, "parse")
	}

	if err := p.parseStructs(result.structFiles); err != nil {
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

// 配置表文件信息
type tableFileInfo struct {
	path string // 路径
}

// structFileNameRegexp 结构体文件名匹配正则表达式
var structFileNameRegexp = regexp.MustCompile(`^(?:[^.]*\.)?struct(?:\.([0-9]*))?$`)

// enumFileNameRegexp 枚举文件名匹配正则表达式
var enumFileNameRegexp = regexp.MustCompile(`^(?:[^.]*\.)?enum(?:\.([0-9]*))?$`)

// searchFiles 搜索文件
func (p *Parser) searchFiles() (result struct {
	enumFiles   []*priorityFileInfo
	structFiles []*priorityFileInfo
	tableFiles  []*tableFileInfo
	err         error
}) {
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

		fileName = strings.TrimSuffix(fileName, ext)

		// 匹配枚举文件
		if matches := enumFileNameRegexp.FindStringSubmatch(fileName); len(matches) > 0 {
			priority, _ := strconv.Atoi(matches[1])
			result.enumFiles = append(result.enumFiles, &priorityFileInfo{
				path:     path,
				priority: priority,
			})
			return nil
		}

		// 匹配结构体文件
		if matches := structFileNameRegexp.FindStringSubmatch(fileName); len(matches) > 0 {
			priority, _ := strconv.Atoi(matches[1])
			result.structFiles = append(result.structFiles, &priorityFileInfo{
				path:     path,
				priority: priority,
			})
			return nil
		}

		// 匹配配置表文件
		result.tableFiles = append(result.tableFiles, &tableFileInfo{path: path})

		return nil
	}); err != nil {
		result.err = pkg_errors.WithMessage(err, "search files")
		return
	}

	sort.SliceStable(result.enumFiles, func(i, j int) bool {
		return result.enumFiles[i].priority < result.enumFiles[j].priority
	})
	sort.SliceStable(result.structFiles, func(i, j int) bool {
		return result.structFiles[i].priority < result.structFiles[j].priority
	})

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
	return commentRegexp.MatchString(value)
}
