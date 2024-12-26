package parse

import (
	"fmt"
	"github.com/godyy/gexcels"
	"github.com/godyy/gexcels/internal/log"
	pkg_errors "github.com/pkg/errors"
	"github.com/tealeg/xlsx/v3"
	"io/fs"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// Options 解析选项
type Options struct {
	// Tags 用于匹配字段或结构体定义的标签.
	// 默认为空，匹配空标签.
	// 如果包含 TagAny, 可匹配任意标签，包括空标签.
	// 如果想在匹配其它标签时匹配空标签，需要在 Tags 中包含 gexcels.TagEmpty.
	Tags []gexcels.Tag

	// FileTags  用于匹配文件名的标签
	// 默认为空，匹配空标签.
	// 如果包含 TagAny, 可匹配任意标签，包括空标签.
	// 如果想在匹配其它标签时匹配空标签，需要在 FileTags 中包含 gexcels.TagEmpty.
	FileTags []gexcels.Tag

	// FieldRuleSep 字段规则分隔符，默认为'|'
	FieldRuleSep string

	// OnlyFields 是否仅解析字段定义, 默认为 false
	OnlyFields bool

	tagMap     map[gexcels.Tag]bool
	fileTagMap map[gexcels.Tag]bool
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

	opt.fileTagMap = make(map[gexcels.Tag]bool, len(opt.FileTags))
	for _, tag := range opt.FileTags {
		if !tag.Valid() {
			return fmt.Errorf("parse: file tag %s invalid", tag)
		}
		opt.fileTagMap[tag] = true
	}
	if len(opt.fileTagMap) == 0 {
		opt.fileTagMap[gexcels.TagEmpty] = true
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

// checkTag 检查文件tag
func (opt *Options) checkFileTag(tag string) bool {
	if opt.fileTagMap[gexcels.TagAny] {
		return true
	}
	return opt.fileTagMap[gexcels.Tag(tag)]
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
	path         string             // 配置路径
	options      *Options           // 选项
	Structs      []*Struct          // 解析出的结构体
	structByName map[string]*Struct // 结构体名称映射
	Tables       []*Table           // 解析出的配置表
	tableByName  map[string]*Table  // 配置表名称映射
}

func newParser(path string, options *Options) *Parser {
	p := &Parser{
		path:         path,
		options:      options,
		Structs:      make([]*Struct, 0),
		structByName: make(map[string]*Struct),
		Tables:       make([]*Table, 0),
		tableByName:  make(map[string]*Table),
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

// checkFileTag 检查文件tag
func (p *Parser) checkFileTag(tag string) bool {
	return p.options.checkFileTag(tag)
}

func (p *Parser) parse() error {
	structs, tables, err := p.searchFiles()
	if err != nil {
		return pkg_errors.WithMessage(err, "parse")
	}

	if err := p.parseStructs(structs); err != nil {
		return pkg_errors.WithMessage(err, "parse")
	}

	if err := p.parseTables(tables); err != nil {
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

// structFileInfo 结构体文件信息
type structFileInfo struct {
	path     string // 路径
	priority int    // 优先级
}

// 配置表文件信息
type tableFileInfo struct {
	path string      // 路径
	tag  gexcels.Tag // 标签
}

// tableFileNameRegexp 配置表文件名匹配正则表达式
var tableFileNameRegexp = regexp.MustCompile(`^[^.]+(?:\.(` + gexcels.TagPattern + `))?$`)

// structFileNameRegexp 结构体文件名匹配正则表达式
var structFileNameRegexp = regexp.MustCompile(`^(?:[^.]*\.)?struct(?:\.([0-9]*))?(?:\.(` + gexcels.TagPattern + `))?$`)

// searchFiles 搜索文件
func (p *Parser) searchFiles() (structs []*structFileInfo, tables []*tableFileInfo, err error) {
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

		// 匹配结构体文件
		if matches := structFileNameRegexp.FindStringSubmatch(fileName); len(matches) == 3 {
			if p.checkFileTag(matches[2]) {
				priority, _ := strconv.Atoi(matches[1])
				structs = append(structs, &structFileInfo{
					path:     path,
					priority: priority,
				})
			}
			return nil
		}

		// 匹配配置表文件
		if matches := tableFileNameRegexp.FindStringSubmatch(fileName); len(matches) == 2 {
			if p.checkFileTag(matches[1]) {
				tables = append(tables, &tableFileInfo{
					path: path,
					tag:  gexcels.Tag(matches[1]),
				})
			}
			return nil
		}

		return nil
	}); err != nil {
		return nil, nil, pkg_errors.WithMessage(err, "search files")
	}

	sort.SliceStable(structs, func(i, j int) bool {
		return structs[i].priority < structs[j].priority
	})

	return
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
