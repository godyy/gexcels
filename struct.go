package gexcels

import "strings"

// 结构体表各列定义
const (
	TableStructColTag    = 0 // tag列
	TableStructColName   = 1 // 名称列
	TableStructColFields = 2 // 字段列
	TableStructColRule   = 3 // 规则列
	TableStructColDesc   = 4 // 描述
)

// TableStructCols 结构体配置表列数量
const TableStructCols = TableStructColDesc + 1

// TableStructFirstRow 结构体配置表首行
const TableStructFirstRow = 1

// StructFieldSep 结构体字段分隔符
const StructFieldSep = ","

// Struct 结构体
type Struct struct {
	Name        string            // 名称
	Desc        string            // 描述
	Fields      []*Field          // 字段
	FieldByName map[string]*Field // 字段名称到字段映射
}

// NewStruct 创建结构体
func NewStruct(name, desc string) *Struct {
	if !MatchName(name) {
		panic("gexcels: NewStruct: name " + name + " invalid")
	}
	return &Struct{
		Name: name,
		Desc: desc,
	}
}

func (sd *Struct) AddField(field *Field) bool {
	if field == nil {
		panic("gexcels: Struct.AddField: field is nil")
	}
	if _, ok := sd.FieldByName[field.Name]; ok {
		return false
	}
	sd.Fields = append(sd.Fields, field)
	sd.FieldByName[field.Name] = field
	return true
}

// GetFieldByName 根据名称获取字段
func (sd *Struct) GetFieldByName(name string) *Field {
	return sd.FieldByName[name]
}

// FieldsString 将结构体字段转换为字符串形式
func (sd *Struct) FieldsString() string {
	var sb strings.Builder
	for i, fd := range sd.Fields {
		if i > 0 {
			sb.WriteString(StructFieldSep)
		}
		sb.WriteString(fd.Name + ":" + fd.FieldTypeInfo.String())
		if fd.Desc != "" {
			sb.WriteString(":\"" + fd.Desc + "\"")
		}
	}
	return sb.String()
}
