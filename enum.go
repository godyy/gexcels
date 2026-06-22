package gexcels

import (
	"fmt"
)

const (
	EnumColBegin    = 0 // 开始定义枚举的列索引
	EnumColItemName = 0 // 枚举项名称列索引
	EnumColType     = 1 // 枚举项类型列索引
	EnumColValue    = 1 // 枚举项值列索引
	EnumColDesc     = 2 // 枚举描述列索引

	EnumCols = 3 // 枚举列数
)

// Enum 枚举.
type Enum struct {
	Name        string               // 枚举名称
	Type        FieldType            // 值类型
	Desc        string               // 描述
	Items       []*EnumItem          // 枚举项列表
	ItemByName  map[string]*EnumItem // 枚举项名称映射
	ItemByValue map[string]*EnumItem // 枚举项值映射
}

// NewEnum 创建枚举.
func NewEnum(name string, typ FieldType, desc string) *Enum {
	if !CheckEnumType(typ) {
		panic(fmt.Errorf("invalid enum type %d", typ))
	}
	return &Enum{
		Name:        name,
		Type:        typ,
		Desc:        desc,
		ItemByName:  make(map[string]*EnumItem),
		ItemByValue: make(map[string]*EnumItem),
	}
}

// EnumItem 枚举项.
type EnumItem struct {
	Name  string // 枚举项名称
	Desc  string // 枚举项描述
	Value string // 枚举项值
	Index int    // 枚举项索引
}

// NewEnumItem 创建枚举项.
func NewEnumItem(name, desc, value string) *EnumItem {
	return &EnumItem{
		Name:  name,
		Desc:  desc,
		Value: value,
	}
}

// ItemAmount 获取枚举项数量.
func (e *Enum) ItemAmount() int {
	return len(e.Items)
}

// AddItem 添加枚举项.
func (e *Enum) AddItem(item *EnumItem) bool {
	if _, ok := e.ItemByName[item.Name]; ok {
		return false
	} else if _, ok := e.ItemByValue[item.Value]; ok {
		return false
	}
	item.Index = len(e.Items)
	e.ItemByName[item.Name] = item
	e.ItemByValue[item.Value] = item
	e.Items = append(e.Items, item)
	return true
}

// GetItem 获取枚举项.
func (e *Enum) GetItem(index int) *EnumItem {
	return e.Items[index]
}

// GetItemByName 根据名称获取枚举项.
func (e *Enum) GetItemByName(name string) *EnumItem {
	return e.ItemByName[name]
}

// GetItemByValue 根据值获取枚举项.
func (e *Enum) GetItemByValue(value string) *EnumItem {
	return e.ItemByValue[value]
}

// CheckEnumType 检查枚举类型是否有效.
func CheckEnumType(typ FieldType) bool {
	return typ == FTInt32 || typ == FTString
}
