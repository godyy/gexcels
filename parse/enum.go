package parse

import (
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strconv"

	"github.com/godyy/gexcels"
	pkg_errors "github.com/pkg/errors"
	"github.com/tealeg/xlsx/v3"
)

// Enum 枚举.
type Enum struct {
	*gexcels.Enum       // 基础信息
	ItemValues    []any // 枚举项值列表
}

// newEnum 创建枚举.
func newEnum(name string, typ gexcels.FieldType, desc string) *Enum {
	return &Enum{
		Enum: gexcels.NewEnum(name, typ, desc),
	}
}

// addItem 添加枚举项
func (e *Enum) addItem(item *gexcels.EnumItem, value any) bool {
	if !e.Enum.AddItem(item) {
		return false
	}
	e.ItemValues = append(e.ItemValues, value)
	return true
}

// GetItemValue 获取枚举项值
func (e *Enum) GetItemValue(index int) any {
	return e.ItemValues[index]
}

// GetItemValueByName 获取枚举项值
func (e *Enum) GetItemValueByName(itemName string) (any, bool) {
	if item := e.GetItemByName(itemName); item != nil {
		return e.GetItemValue(item.Index), true
	} else {
		return nil, false
	}
}

// enumBeginRegexp 匹配开始定义枚举的正则表达式.
var enumBeginRegexp = regexp.MustCompile(`^([a-zA-Z][a-zA-Z0-9_]*)_BEGIN$`)

// addEnum 添加枚举
func (p *Parser) addEnum(enum *Enum) error {
	if err := p.addCustomFieldType(gexcels.NewEnumFieldTypeInfo(enum.Name)); err != nil {
		return err
	}

	p.Enums = append(p.Enums, enum)
	p.enumByName[enum.Name] = enum
	return nil
}

// GetEnum 获取枚举
func (p *Parser) GetEnum(name string) *Enum {
	return p.enumByName[name]
}

// parseEnums 解析枚举.
func (p *Parser) parseEnums(enumFiles []*priorityFileInfo) error {
	for _, file := range enumFiles {
		if err := p.parseEnumFile(file); err != nil {
			return err
		}
	}
	return nil
}

// enumSheetNameRegexp 枚举sheet名匹配正则表达式
var enumSheetNameRegexp = regexp.MustCompile(`^(.*)\|(Enum\w*)(?:\.([0-9]*))?`)

// parseEnumFile 解析枚举文件.
func (p *Parser) parseEnumFile(info *priorityFileInfo) error {
	file, err := xlsx.OpenFile(info.path)
	if err != nil {
		return err
	}

	sheets := make([]*prioritySheetInfo, 0, len(file.Sheets))
	for _, sheet := range file.Sheets {
		matches := enumSheetNameRegexp.FindStringSubmatch(sheet.Name)
		if len(matches) != 4 {
			continue
		}

		priority, _ := strconv.Atoi(matches[3])
		sheets = append(sheets, &prioritySheetInfo{
			Sheet:    sheet,
			priority: priority,
		})
	}

	sort.SliceStable(sheets, func(i, j int) bool {
		return sheets[i].priority < sheets[j].priority
	})

	for _, sheet := range sheets {
		if err := p.parseEnumSheet(sheet.Sheet); err != nil {
			return pkg_errors.WithMessagef(err, "enum file(%s).sheet(%s)", info.path, sheet.Name)
		}
	}
	return nil
}

// parseEnumOfSheet 解析枚举表
func (p *Parser) parseEnumSheet(sheet *xlsx.Sheet) error {
	if sheet.MaxCol < gexcels.EnumCols {
		return errors.New("enum sheet has no enough columns")
	}

	var (
		enum *Enum
		row  int
		err  error
	)

	for row < sheet.MaxRow {
		var newRow int
		enum, newRow, err = p.parseEnum(sheet, row)
		if err != nil {
			return pkg_errors.WithMessagef(err, "parse enum start at %d", row)
		}
		if enum != nil {
			if err := p.addEnum(enum); err != nil {
				return pkg_errors.WithMessagef(err, "add enum %s", enum.Name)
			}
		}
		row = newRow
	}
	return nil
}

// parseEnum 解析枚举.
func (p *Parser) parseEnum(sheet *xlsx.Sheet, row int) (*Enum, int, error) {
	enumBegin, err := getSheetValue(sheet, row, gexcels.EnumColBegin, true)
	if err != nil {
		return nil, 0, pkg_errors.WithMessagef(err, "get begin cell")
	}
	if enumBegin == "" {
		return nil, row + 1, nil
	}
	matches := enumBeginRegexp.FindStringSubmatch(enumBegin)
	if len(matches) != 2 {
		return nil, 0, fmt.Errorf("invalid begin \"%s\"", enumBegin)
	}
	enumName := matches[1]

	enumType, err := getSheetValue(sheet, row, gexcels.EnumColType, true)
	if err != nil {
		return nil, 0, pkg_errors.WithMessage(err, "get type cell")
	}
	enumTypeInfo, err := p.parseFieldTypeInfo(enumType)
	if err != nil {
		return nil, 0, pkg_errors.WithMessage(err, "parse type")
	}
	if !gexcels.CheckEnumType(enumTypeInfo.Type) {
		return nil, row, fmt.Errorf("invalid enum type %s", enumType)
	}

	enumDesc, err := getSheetValue(sheet, row, gexcels.EnumColDesc, true)
	if err != nil {
		return nil, 0, pkg_errors.WithMessage(err, "get desc cell")
	}

	enum := newEnum(enumName, enumTypeInfo.Type, enumDesc)
	row += 1
	for row < sheet.MaxRow {
		itemName, err := getSheetValue(sheet, row, gexcels.EnumColItemName, true)
		if err != nil {
			return nil, 0, pkg_errors.WithMessagef(err, "get [%d] item name cell", enum.ItemAmount())
		}
		if itemName == "" {
			row++
			break
		}
		if !gexcels.MatchName(itemName) {
			return nil, 0, fmt.Errorf("invalid item name %s", itemName)
		}
		itemDesc, err := getSheetValue(sheet, row, gexcels.EnumColDesc, true)
		if err != nil {
			return nil, 0, pkg_errors.WithMessagef(err, "get [%d] item desc cell", enum.ItemAmount())
		}
		itemValue, err := getSheetValue(sheet, row, gexcels.EnumColValue)
		if err != nil {
			return nil, 0, pkg_errors.WithMessagef(err, "get [%d] item value cell", enum.ItemAmount())
		}
		value, err := parsePrimitiveValue(enumTypeInfo.Type, itemValue)
		if err != nil {
			return nil, 0, pkg_errors.WithMessage(err, "parse [%d ]item value")
		}
		enum.addItem(gexcels.NewEnumItem(itemName, itemDesc, itemValue), value)
		row++
	}

	return enum, row, nil
}
