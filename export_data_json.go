package gexcels

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"

	pkg_errors "github.com/pkg/errors"
)

// jsonExporter json数据导出器
type jsonExporter struct{}

func (je *jsonExporter) export(p *Parser, path string) error {
	printf("export data json to [%s]", path)

	for _, table := range p.tables {
		if err := je.exportTableFile(table, path); err != nil {
			return err
		}
	}

	return nil
}

// 导出配置表json文件
func (je *jsonExporter) exportTableFile(td *Table, path string) error {
	var (
		jsonMarshaler = tableJsonMarshaler{table: td}
		bytes         []byte
		err           error
	)

	bytes, err = json.MarshalIndent(&jsonMarshaler, "", "\t")
	if err != nil {
		return pkg_errors.WithMessagef(err, "export json: marshal table[%s]", td.Name)
	}

	fileName := td.ExportTableName + ".json"
	filePath := filepath.Join(path, fileName)
	if err := os.WriteFile(filePath, bytes, os.ModePerm); err != nil {
		return pkg_errors.WithMessagef(err, "export json: table[%s] to [%s]", td.Name, filePath)
	}
	printfGreen("export json: table[%s] to [%s]", td.Name, filePath)
	return nil
}

type tableJsonMarshaler struct {
	table *Table
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

func (e *tableJsonMarshaler) marshalNormal() ([]byte, error) {
	buf := bytes.NewBuffer(nil)
	buf.WriteString("[")
	for i, entry := range e.table.Entries {
		if i > 0 {
			buf.WriteString(",")
		}
		entryJson, err := marshalJsonTableEntry(e.table, entry)
		if err != nil {
			return nil, pkg_errors.WithMessagef(err, "entry[%v]", entry[idExportName])
		}
		if _, err := buf.Write(entryJson); err != nil {
			return nil, pkg_errors.WithMessagef(err, "entry[%v]", entry[idExportName])
		}
	}
	buf.WriteString("]")
	return buf.Bytes(), nil
}

func (e *tableJsonMarshaler) marshalGlobal() ([]byte, error) {
	buf := bytes.NewBuffer(nil)
	buf.WriteString("{")
	for i, fd := range e.table.Fields {
		fieldVal := e.table.getEntryByName(fd.Name)
		if fieldVal == nil {
			continue
		}
		if i > 0 {
			buf.WriteString(",")
		}
		buf.WriteString(jsonFieldNamePrefix(fd.Name))
		entryJson, err := marshalJsonFieldValue(fd.Field, fieldVal)
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

func marshalJsonTableEntry(td *Table, entry TableEntry) ([]byte, error) {
	n := 0
	buf := bytes.NewBuffer(nil)
	buf.WriteString("{")
	for i, fd := range td.Fields {
		fieldVal := entry[fd.Name]
		if fieldVal == nil {
			continue
		}
		if n > 0 {
			buf.WriteString(",")
		}
		if i == 0 {
			buf.WriteString(jsonFieldNamePrefix(idJsonName))
		} else {
			buf.WriteString(jsonFieldNamePrefix(fd.Name))
		}
		fieldJson, err := marshalJsonFieldValue(fd.Field, fieldVal)
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

func marshalJsonFieldValue(fd *Field, val interface{}) ([]byte, error) {
	if fd.Type.Primitive() {
		return json.Marshal(val)
	} else if fd.Type == FTStruct {
		return marshalJsonStructValue(fd.Struct, val)
	} else if fd.Type == FTArray {
		return marshalJsonArrayValue(fd, val)
	} else {
		panic(errFieldTypeInvalid(fd.Type))
	}
}

func marshalJsonStructValue(sd *Struct, val interface{}) ([]byte, error) {
	refVal := reflect.ValueOf(val)
	if refVal.Kind() == reflect.Pointer {
		refVal = refVal.Elem()
	}
	n := 0
	buf := bytes.NewBuffer(nil)
	buf.WriteString("{")
	for _, fd := range sd.Fields {
		refFieldVal := refVal.FieldByName(fd.ExportName)
		if !refFieldVal.IsValid() {
			continue
		}
		if n > 0 {
			buf.WriteString(",")
		}
		buf.WriteString(jsonFieldNamePrefix(fd.Name))
		fieldValJson, err := marshalJsonFieldValue(fd, refFieldVal.Interface())
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

func marshalJsonArrayValue(fd *Field, val interface{}) ([]byte, error) {
	if fd.ElementType.Primitive() {
		return json.Marshal(val)
	} else if fd.ElementType == FTStruct {
		refVal := reflect.ValueOf(val)
		buf := bytes.NewBuffer(nil)
		buf.WriteString("[")
		for i := 0; i < refVal.Len(); i++ {
			if i > 0 {
				buf.WriteString(",")
			}
			elemValJson, err := marshalJsonStructValue(fd.Struct, refVal.Index(i).Interface())
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
		panic(errArrayElementInvalid(fd.ElementType))
	}
}

func jsonFieldNamePrefix(name string) string {
	return fmt.Sprintf(`"%s": `, name)
}
