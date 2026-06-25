package data

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"

	"github.com/godyy/gexcels"
	"github.com/godyy/gexcels/export"
	internal_define "github.com/godyy/gexcels/export"
	"github.com/godyy/gexcels/internal/log"
	"github.com/godyy/gexcels/parse"
	pkg_errors "github.com/pkg/errors"
)

// ExportJson 导出json格式配置表数据
func ExportJson(p *parse.Parser, path string) error {
	if p == nil {
		return ErrNoParserSpecified
	}

	if path == "" {
		return ErrNoPathSpecified
	}

	return doExport(&jsonExporter{baseExporter: newBaseExporter(p), path: path})
}

// jsonExporter json数据导出器
type jsonExporter struct {
	baseExporter
	path string
}

func (e *jsonExporter) kind() internal_define.DataKind {
	return internal_define.DataJson
}

func (e *jsonExporter) export() error {
	log.Printf("export data json to [%s]", e.path)

	if err := os.MkdirAll(e.path, os.ModePerm); err != nil {
		return pkg_errors.WithMessagef(err, "mkdir")
	}

	for _, table := range e.parser.Tables {
		if err := e.exportTableFile(table); err != nil {
			return pkg_errors.WithMessage(err, "export data: json")
		}
	}

	return nil
}

// exportTableFile 导出配置表文件
func (e *jsonExporter) exportTableFile(td *parse.Table) error {
	var (
		jsonMarshaler = tableJsonMarshaler{exporter: e, table: td}
		bytes         []byte
		err           error
	)

	bytes, err = json.MarshalIndent(&jsonMarshaler, "", "\t")
	if err != nil {
		return pkg_errors.WithMessagef(err, "marshal table[%s]", td.Name)
	}

	fileName := genTableFileName(td.Name, ".json")
	filePath := filepath.Join(e.path, fileName)
	if err := os.WriteFile(filePath, bytes, os.ModePerm); err != nil {
		return pkg_errors.WithMessagef(err, "table[%s] to [%s]", td.Name, filePath)
	}
	log.PrintfGreen("export data: json: table[%s] to [%s]", td.Name, filePath)
	return nil
}

// marshalJsonTableEntry 编码表条目
func (e *jsonExporter) marshalJsonTableEntry(td *parse.Table, entry gexcels.TableEntry) ([]byte, error) {
	n := 0
	buf := bytes.NewBuffer(nil)
	buf.WriteString("{")
	for _, fd := range td.Fields {
		fieldVal := entry[fd.Name]
		if fieldVal == nil {
			continue
		}
		if n > 0 {
			buf.WriteString(",")
		}
		if fd.Col == gexcels.TableColFieldID {
			buf.WriteString(e.jsonFieldNamePrefix(export.TableFieldIDJsonName))
		} else {
			buf.WriteString(e.jsonFieldNamePrefix(fd.Name))
		}
		fieldJson, err := e.marshalJsonFieldValue(fd.Field.FieldTypeInfo, fieldVal)
		if err != nil {
			return nil, pkg_errors.WithMessagef(err, "field[%s]", fd.Name)
		}
		if _, err := buf.Write(fieldJson); err != nil {
			return nil, pkg_errors.WithMessagef(err, "field[%s]", fd.Name)
		}
		n++
	}
	buf.WriteString("}")
	return buf.Bytes(), nil
}

// marshalJsonFieldValue 编码字段值
func (e *jsonExporter) marshalJsonFieldValue(ft *gexcels.FieldTypeInfo, val any) ([]byte, error) {
	if ft.Type.Primitive() {
		return json.Marshal(val)
	} else if ft.Type == gexcels.FTEnum {
		return json.Marshal(val)
	} else if ft.Type == gexcels.FTStruct {
		return e.marshalJsonStructValue(ft, val)
	} else if ft.Type == gexcels.FTArray {
		return e.marshalJsonArrayValue(ft, val)
	} else if ft.Type == gexcels.FTMap {
		return e.marshalJsonMapValue(ft, val)
	} else {
		panic(fmt.Sprintf("marshalFieldValue: field type %d invalid", ft.Type))
	}
}

// marshalJsonStructValue 编码结构体值
func (e *jsonExporter) marshalJsonStructValue(ft *gexcels.FieldTypeInfo, val any) ([]byte, error) {
	if val == nil {
		return []byte("null"), nil
	}

	sd := e.parser.GetStructByName(ft.GetName())
	refVal := reflect.ValueOf(val)
	n := 0
	buf := bytes.NewBuffer(nil)
	buf.WriteString("{")
	for _, fd := range sd.Fields {
		refFieldVal := refVal.MapIndex(reflect.ValueOf(fd.Name))
		if !refFieldVal.IsValid() {
			continue
		}
		if n > 0 {
			buf.WriteString(",")
		}
		buf.WriteString(e.jsonFieldNamePrefix(fd.Name))
		fieldValJson, err := e.marshalJsonFieldValue(fd.FieldTypeInfo, refFieldVal.Interface())
		if err != nil {
			return nil, pkg_errors.WithMessagef(err, "field[%s]", fd.Name)
		}
		if _, err := buf.Write(fieldValJson); err != nil {
			return nil, pkg_errors.WithMessagef(err, "field[%s]", fd.Name)
		}
		n++
	}
	buf.WriteString("}")
	return buf.Bytes(), nil
}

// checkArrayCouldMarshal 检查数组类型是否直接编码
func (e *jsonExporter) checkArrayCouldMarshal(ft *gexcels.FieldTypeInfo) bool {
	if ft.Type != gexcels.FTArray {
		return false
	}
	elementType := ft.GetElementType()
	return elementType.Type.Primitive() || elementType.Type == gexcels.FTEnum ||
		(elementType.Type == gexcels.FTArray && e.checkArrayCouldMarshal(elementType))
}

// marshalJsonArrayValue 编码数组值
func (e *jsonExporter) marshalJsonArrayValue(ft *gexcels.FieldTypeInfo, val any) ([]byte, error) {
	if val == nil {
		return []byte("null"), nil
	}

	if e.checkArrayCouldMarshal(ft) {
		return json.Marshal(val)
	}

	elementType := ft.GetElementType()
	refVal := reflect.ValueOf(val)
	buf := bytes.NewBuffer(nil)
	buf.WriteString("[")
	for i := 0; i < refVal.Len(); i++ {
		if i > 0 {
			buf.WriteString(",")
		}
		elemValJson, err := e.marshalJsonFieldValue(elementType, refVal.Index(i).Interface())
		if err != nil {
			return nil, pkg_errors.WithMessagef(err, "[%d]", i)
		}
		if _, err := buf.Write(elemValJson); err != nil {
			return nil, pkg_errors.WithMessagef(err, "[%d]", i)
		}
	}
	buf.WriteString("]")
	return buf.Bytes(), nil
}

// marshalJsonMapValue 编码 map值
func (e *jsonExporter) marshalJsonMapValue(ft *gexcels.FieldTypeInfo, val any) ([]byte, error) {
	if val == nil {
		return []byte("null"), nil
	}

	keyType := ft.GetMapKeyType()
	valueType := ft.GetMapValueType()
	refVal := reflect.ValueOf(val)
	keys := e.sortMapKeys(keyType, refVal.MapKeys())
	buf := bytes.NewBuffer(nil)
	buf.WriteString("{")
	for i, key := range keys {
		if i > 0 {
			buf.WriteString(",")
		}

		keyJson, err := e.marshalJsonMapKey(keyType, key.Interface())
		if err != nil {
			return nil, pkg_errors.WithMessagef(err, "[%v]", key.Interface())
		}
		if _, err := buf.Write(keyJson); err != nil {
			return nil, pkg_errors.WithMessagef(err, "[%v]", key.Interface())
		}
		buf.WriteString(`: `)

		value := refVal.MapIndex(key)
		valueJson, err := e.marshalJsonFieldValue(valueType, value.Interface())
		if err != nil {
			return nil, pkg_errors.WithMessagef(err, "[%v]value", key.Interface())
		}
		if _, err := buf.Write(valueJson); err != nil {
			return nil, pkg_errors.WithMessagef(err, "[%v]value", key.Interface())
		}
	}
	buf.WriteString("}")
	return buf.Bytes(), nil
}

// marshalJsonMapKey 编码 map 键
func (e *jsonExporter) marshalJsonMapKey(ft *gexcels.FieldTypeInfo, val any) ([]byte, error) {
	keyJson, err := e.marshalJsonFieldValue(ft, val)
	if err != nil {
		return nil, err
	}
	if ft.Type == gexcels.FTString || (ft.Type == gexcels.FTEnum && e.parser.GetEnum(ft.GetName()).Type == gexcels.FTString) {
		return keyJson, nil
	}
	return []byte(`"` + string(keyJson) + `"`), nil
}

func (e *jsonExporter) jsonFieldNamePrefix(name string) string {
	return fmt.Sprintf(`"%s": `, name)
}

// tableJsonMarshaler 配置表json编码器
type tableJsonMarshaler struct {
	exporter *jsonExporter
	table    *parse.Table
}

func (e *tableJsonMarshaler) MarshalJSON() ([]byte, error) {
	if e.table == nil {
		return nil, nil
	}

	if e.table.IsGlobal {
		return e.marshalGlobal()
	} else {
		return e.marshalNormal()
	}
}

// marshalNormal 编码普通表
func (e *tableJsonMarshaler) marshalNormal() ([]byte, error) {
	buf := bytes.NewBuffer(nil)
	buf.WriteString("[")
	fieldID := e.table.GetFieldID()
	for i, entry := range e.table.Entries {
		if i > 0 {
			buf.WriteString(",")
		}
		entryJson, err := e.exporter.marshalJsonTableEntry(e.table, entry)
		if err != nil {
			return nil, pkg_errors.WithMessagef(err, "entry[%v]", entry[fieldID.Name])
		}
		if _, err := buf.Write(entryJson); err != nil {
			return nil, pkg_errors.WithMessagef(err, "entry[%v]", entry[fieldID.Name])
		}
	}
	buf.WriteString("]")
	return buf.Bytes(), nil
}

// marshalGlobal 编码全局表
func (e *tableJsonMarshaler) marshalGlobal() ([]byte, error) {
	buf := bytes.NewBuffer(nil)
	buf.WriteString("{")
	for i, fd := range e.table.Fields {
		fieldVal := e.table.GetEntryByName(fd.Name)
		if fieldVal == nil {
			continue
		}
		if i > 0 {
			buf.WriteString(",")
		}
		buf.WriteString(e.exporter.jsonFieldNamePrefix(fd.Name))
		entryJson, err := e.exporter.marshalJsonFieldValue(fd.Field.FieldTypeInfo, fieldVal)
		if err != nil {
			return nil, pkg_errors.WithMessagef(err, "field[%s]", fd.Name)
		}
		if _, err := buf.Write(entryJson); err != nil {
			return nil, pkg_errors.WithMessagef(err, "field[%s]", fd.Name)
		}
	}
	buf.WriteString("}")
	return buf.Bytes(), nil
}
