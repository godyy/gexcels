package data

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/godyy/gexcels"
	internal_define "github.com/godyy/gexcels/export"
	"github.com/godyy/gexcels/internal/log"
	"github.com/godyy/gexcels/internal/utils"
	"github.com/godyy/gexcels/parse"
	pkg_errors "github.com/pkg/errors"
	"os"
	"path/filepath"
	"reflect"
)

// jsonOptions json导出选项
type jsonOptions struct {
}

func (opts *jsonOptions) kind() internal_define.DataKind {
	return internal_define.DataJson
}

// ExportJson 导出json格式配置表数据
func ExportJson(p *parse.Parser, path string) error {
	return doExport(p, path, &jsonOptions{})
}

// jsonExporter json数据导出器
type jsonExporter struct {
	exporterBase
}

func createJsonExporter(p *parse.Parser, path string, options kindOptions) (exporter, error) {
	return &jsonExporter{
		exporterBase: newExporterBase(p, path),
	}, nil
}

func (e *jsonExporter) kind() internal_define.DataKind {
	return internal_define.DataJson
}

func (e *jsonExporter) export() error {
	log.Printf("export data json to [%s]", e.path)

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

	fileName := genTableFileName(td.Name, td.Tag, ".json")
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
			buf.WriteString(e.jsonFieldNamePrefix(gexcels.TableFieldIDJsonTagName))
		} else {
			buf.WriteString(e.jsonFieldNamePrefix(fd.Name))
		}
		fieldJson, err := e.marshalJsonFieldValue(fd.Field, fieldVal)
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
func (e *jsonExporter) marshalJsonFieldValue(fd *gexcels.Field, val interface{}) ([]byte, error) {
	if fd.Type.Primitive() {
		return json.Marshal(val)
	} else if fd.Type == gexcels.FTStruct {
		sd := e.parser.GetStructByName(fd.StructName)
		return e.marshalJsonStructValue(sd, val)
	} else if fd.Type == gexcels.FTArray {
		return e.marshalJsonArrayValue(fd, val)
	} else {
		panic(fmt.Sprintf("marshalFieldValue: field type %d invalid", fd.Type))
	}
}

// marshalJsonStructValue 编码结构体值
func (e *jsonExporter) marshalJsonStructValue(sd *parse.Struct, val interface{}) ([]byte, error) {
	refVal := reflect.ValueOf(val)
	if refVal.Kind() == reflect.Pointer {
		refVal = refVal.Elem()
	}
	n := 0
	buf := bytes.NewBuffer(nil)
	buf.WriteString("{")
	for _, fd := range sd.Fields {
		refFieldVal := refVal.FieldByName(utils.CamelCase(fd.Name, true))
		if !refFieldVal.IsValid() {
			continue
		}
		if n > 0 {
			buf.WriteString(",")
		}
		buf.WriteString(e.jsonFieldNamePrefix(fd.Name))
		fieldValJson, err := e.marshalJsonFieldValue(fd, refFieldVal.Interface())
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

// marshalJsonArrayValue 编码数组值
func (e *jsonExporter) marshalJsonArrayValue(fd *gexcels.Field, val interface{}) ([]byte, error) {
	if fd.ElementType.Primitive() {
		return json.Marshal(val)
	} else if fd.ElementType == gexcels.FTStruct {
		sd := e.parser.GetStructByName(fd.StructName)
		refVal := reflect.ValueOf(val)
		buf := bytes.NewBuffer(nil)
		buf.WriteString("[")
		for i := 0; i < refVal.Len(); i++ {
			if i > 0 {
				buf.WriteString(",")
			}
			elemValJson, err := e.marshalJsonStructValue(sd, refVal.Index(i).Interface())
			if err != nil {
				return nil, pkg_errors.WithMessagef(err, "[%d]%s", i, fd.StructName)
			}
			if _, err := buf.Write(elemValJson); err != nil {
				return nil, pkg_errors.WithMessagef(err, "[%d]%s", i, fd.StructName)
			}
		}
		buf.WriteString("]")
		return buf.Bytes(), nil
	} else {
		panic(fmt.Sprintf("export data: json: marshalArrayValue: elment type %d invalid", fd.ElementType))
	}
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
		entryJson, err := e.exporter.marshalJsonFieldValue(fd.Field, fieldVal)
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
