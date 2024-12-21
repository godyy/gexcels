package gexcels

import (
	"errors"
	"reflect"
	"strconv"
	"strings"
)

// 配置表表头行定义
const (
	TableRowFieldName = 0 // 字段名称行号
	TableRowFieldDesc = 1 // 字段描述行号
	TableRowFieldType = 2 // 字段类型行号
	TableRowFieldRule = 3 // 字段规则行号
	TableRowFieldTag  = 4 // 字段tag行号
)

// TableRowFirstEntry 第一条目行号
const TableRowFirstEntry = TableRowFieldTag + 1

// TableColFieldID 配置表ID字段列号
const TableColFieldID = 0

// TableFieldIDName 配置表ID字段名
const TableFieldIDName = "ID"

// TableFieldIDJsonTagName 配置表ID字段json标签名
const TableFieldIDJsonTagName = "id"

// TableMaxField 配置表字段上限
const TableMaxField = 32767

// Global配置表各列定义
const (
	GlobalTableColFieldTag   = 0 // Tag
	GlobalTableColFieldName  = 1 // 字段名
	GlobalTableColFieldType  = 2 // 字段类型
	GlobalTableColFieldValue = 3 // 字段值
	GlobalTableColFieldRule  = 4 // 字段规则
	GlobalTableColFieldDesc  = 5 // 字段描述
)

// GlobalTableSkipRows Global配置表读去跳过行数
const GlobalTableSkipRows = 1

// GlobalTableCols Global配置表列数
const GlobalTableCols = 5

// GlobalTableNamePrefix 全局配置表名称前缀
const GlobalTableNamePrefix = "Global"

// TableField 配置表字段
type TableField struct {
	*Field     // base
	Col    int // 非Global配置时有效
}

// NewTableField 创建配置表字段
func NewTableField(field *Field, col int) *TableField {
	if field == nil {
		panic("gexcels: NewTableField: field is nil")
	}
	if col < 0 {
		panic("gexcels: NewTableField: col must be >= 0")
	}
	return &TableField{
		Field: field,
		Col:   col,
	}
}

func (tf *TableField) Unique() bool {
	return tf.HasFRUnique()
}

// TableEntry 配置条目数据
type TableEntry map[string]interface{}

// Table 配置表定义
type Table struct {
	Tag         Tag                    // 标签
	Name        string                 // 表名
	Desc        string                 // 描述
	IsGlobal    bool                   // 是否全局配置表
	Fields      []*TableField          // 字段
	FieldByName map[string]*TableField // 字段名映射
}

// NewTable 创建配置表
func NewTable(tag Tag, name, desc string, isGlobal bool) *Table {
	if !MatchName(name) {
		panic("gexcels: NewTable: table name " + name + " invalid")
	}
	table := &Table{
		Tag:      tag,
		Name:     name,
		Desc:     desc,
		IsGlobal: isGlobal,
	}
	if isGlobal && !strings.HasPrefix(name, GlobalTableNamePrefix) {
		table.Name = GlobalTableNamePrefix + name
	}
	return table
}

// ErrTableFieldNumberExceedLimit 配置表字段数量超过上限
var ErrTableFieldNumberExceedLimit = errors.New("gexcels: table field number exceed limit")

// AddField 添加字段
func (t *Table) AddField(fd *TableField) bool {
	if len(t.Fields) >= TableMaxField {
		panic(ErrTableFieldNumberExceedLimit)
	}

	if t.FieldByName == nil {
		t.FieldByName = make(map[string]*TableField)
	} else if _, ok := t.FieldByName[fd.Name]; ok {
		return false
	}

	t.Fields = append(t.Fields, fd)
	t.FieldByName[fd.Name] = fd
	return true
}

// GetFieldByName 获取字段
func (t *Table) GetFieldByName(name string) *TableField {
	return t.FieldByName[name]
}

// HasField 是否存在字段
func (t *Table) HasField(name string) bool {
	_, ok := t.FieldByName[name]
	return ok
}

// GetFieldID 获取ID字段
func (t *Table) GetFieldID() *TableField {
	return t.Fields[TableColFieldID]
}

// convertUniqueValue2String 将唯一值转换为字符串
func convertUniqueValue2String(v interface{}) string {
	switch o := v.(type) {
	case string:
		return o
	case int32:
		return strconv.FormatInt(int64(o), 10)
	case int64:
		return strconv.FormatInt(o, 10)
	case float32:
		return strconv.FormatFloat(float64(o), 'f', -1, 32)
	case float64:
		return strconv.FormatFloat(o, 'f', -1, 32)
	case bool:
		return strconv.FormatBool(o)
	default:
		panic("gexcels: invalid unique value type " + reflect.TypeOf(v).String())
	}
}
