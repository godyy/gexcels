package data

import (
	"encoding/base64"
	"fmt"
	internal_define "github.com/godyy/gexcels/export/internal/define"
	"github.com/godyy/gexcels/internal/log"
	"github.com/godyy/gexcels/parse"
	"github.com/godyy/gutils/buffer/bytes"
	pkg_errors "github.com/pkg/errors"
	"os"
	"path/filepath"
	"reflect"
)

// bytesOption bytes导出选项
type bytesOption struct {
}

func (opts *bytesOption) kind() internal_define.DataKind {
	return internal_define.DataBytes
}

// ExportBytes 导出bytes格式数据
func ExportBytes(p *parse.Parser, path string) error {
	return doExport(p, path, &bytesOption{})
}

// bytesExporter bytes导出器
type bytesExporter struct {
	exporterBase
	buf *bytes.Buffer
}

func createBytesExporter(p *parse.Parser, path string, options kindOptions) (exporter, error) {
	return &bytesExporter{
		exporterBase: newExporterBase(p, path),
		buf:          bytes.NewBuffer(make([]byte, 0, 128)),
	}, nil
}

func (e *bytesExporter) kind() internal_define.DataKind {
	return internal_define.DataBytes
}

func (e *bytesExporter) export() error {
	log.Printf("export data bytes to [%s]", e.path)

	for _, table := range e.parser.Tables {
		if err := e.exportTableFile(table); err != nil {
			return pkg_errors.WithMessage(err, "export data: bytes")
		}
	}
	return nil
}

// exportTableFile 导出配置表文件
func (e *bytesExporter) exportTableFile(td *parse.Table) error {
	if td.IsGlobal {
		return e.exportGlobalTableFile(td)
	} else {
		return e.exportNormalTableFile(td)
	}
}

// exportNormalTableFile 导出常规配置表文件
func (e *bytesExporter) exportNormalTableFile(td *parse.Table) error {
	fileName := genTableFileName(td.Name, td.Tag, ".bytes")
	filePath := filepath.Join(e.path, fileName)
	file, err := os.OpenFile(filePath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, os.ModePerm)
	if err != nil {
		return pkg_errors.WithMessagef(err, "table[%s] to %s", td.Name, filePath)
	}
	defer file.Close()

	if _, err = e.buf.WriteVarint32(int32(len(td.Entries))); err == nil {
		str := base64.StdEncoding.EncodeToString(e.buf.Data())
		_, err = file.WriteString(str + "\n")
	}
	if err != nil {
		return pkg_errors.WithMessagef(err, "table[%s] to %s, write entry count", td.Name, filePath)
	}
	e.buf.Reset()

	fieldID := td.GetFieldID()
	for _, entry := range td.Entries {
		if err := e.encodeTableEntry(td, entry); err != nil {
			return pkg_errors.WithMessagef(err, "table[%s] to %s", td.Name, filePath)
		}
		entryStr := base64.StdEncoding.EncodeToString(e.buf.Data())
		if _, err := file.WriteString(entryStr + "\n"); err != nil {
			return pkg_errors.WithMessagef(err, "table[%s] to %s, write entry[%v]", td.Name, filePath, entry[fieldID.Name])
		}
		e.buf.Reset()
	}
	log.PrintfGreen("export data: bytes: table[%s] to [%s]", td.Name, filePath)
	return nil
}

// exportGlobalTableFile 导出全局配置表文件
func (e *bytesExporter) exportGlobalTableFile(td *parse.Table) error {
	fileName := genTableFileName(td.Name, td.Tag, ".bytes")
	filePath := filepath.Join(e.path, fileName)
	file, err := os.OpenFile(filePath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, os.ModePerm)
	if err != nil {
		return pkg_errors.WithMessagef(err, "table[%s] to %s", td.Name, filePath)
	}
	defer file.Close()
	for i, fd := range td.Fields {
		value := td.GetEntryByName(fd.Name)
		if value == nil {
			continue
		}
		if err := e.encodeFieldValue(fd.Field, i, value); err != nil {
			return pkg_errors.WithMessagef(err, "table[%s] to %s", fd.Name, filePath)
		}
		valueStr := base64.StdEncoding.EncodeToString(e.buf.Data())
		if _, err := file.WriteString(valueStr + "\n"); err != nil {
			return pkg_errors.WithMessagef(err, "table[%s] to %s, write field[%v]", td.Name, filePath, fd.Name)
		}
		e.buf.Reset()
	}
	log.PrintfGreen("export data: bytes: table[%s] to [%s]", td.Name, filePath)
	return nil
}

// encodeTableEntry 编码配置表条目
func (e *bytesExporter) encodeTableEntry(td *parse.Table, entry parse.TableEntry) error {
	fieldID := td.GetFieldID()
	for i, fd := range td.Fields {
		fieldValue := entry[fd.Name]
		if fieldValue == nil {
			continue
		}
		if err := e.encodeFieldValue(fd.Field, i, fieldValue); err != nil {
			return pkg_errors.WithMessagef(err, "entry[%v]", entry[fieldID.Name])
		}
	}
	return nil
}

// encodeFieldValue 编码字段值
func (e *bytesExporter) encodeFieldValue(fd *parse.Field, index int, value interface{}) error {
	// field index
	if _, err := e.buf.WriteVarint16(int16(index + 1)); err != nil {
		return pkg_errors.WithMessagef(err, "field[%s] index", fd.Name)
	}

	// field value
	if fd.Type.Primitive() {
		return e.encodePrimitiveValue(fd.Type, value)
	} else if fd.Type == parse.FTStruct {
		return e.encodeStructField(e.parser.GetStructByName(fd.StructName), value)
	} else if fd.Type == parse.FTArray {
		return e.encodeArrayValue(fd, value)
	} else {
		panic(fmt.Sprintf("export data: bytes: encodeFieldValue: field type %d invalid", fd.Type))
	}
}

// encodePrimitiveValue 编码primitive值
func (e *bytesExporter) encodePrimitiveValue(ft parse.FieldType, value interface{}) error {
	switch ft {
	case parse.FTInt32:
		_, err := e.buf.WriteVarint32(value.(int32))
		return err
	case parse.FTInt64:
		_, err := e.buf.WriteVarint64(value.(int64))
		return err
	case parse.FTFloat32:
		return e.buf.WriteFloat32(value.(float32))
	case parse.FTFloat64:
		return e.buf.WriteFloat64(value.(float64))
	case parse.FTBool:
		return e.buf.WriteBool(value.(bool))
	case parse.FTString:
		return e.buf.WriteString(value.(string))
	default:
		panic(fmt.Sprintf("export data: bytes: encodePrimitiveValue: field type %d invalid", ft))
	}
}

// encodeStructField 编码struct字段
func (e *bytesExporter) encodeStructField(sd *parse.Struct, value interface{}) error {
	var (
		v         = reflect.ValueOf(value).Elem()
		lastIndex int
	)
	for i, fd := range sd.Fields {
		fv := v.FieldByName(parse.GenStructFieldName(fd.Name))
		if !fv.IsValid() {
			continue
		}
		if err := e.encodeFieldValue(fd, i, fv.Interface()); err != nil {
			return pkg_errors.WithMessagef(err, "field[%s]", fd.Name)
		}
		lastIndex = i
	}
	if lastIndex < len(sd.Fields)-1 {
		if _, err := e.buf.WriteVarint16(0); err != nil {
			return pkg_errors.WithMessage(err, "struct end marker")
		}
	}
	return nil
}

// encodeArrayValue 编码数组值
func (e *bytesExporter) encodeArrayValue(fd *parse.Field, value interface{}) error {
	v := reflect.ValueOf(value)
	if _, err := e.buf.WriteVarint16(int16(v.Len())); err != nil {
		return pkg_errors.WithMessagef(err, "field[%s] length", fd.Name)
	}
	if fd.ElementType.Primitive() {
		for i := 0; i < v.Len(); i++ {
			vv := v.Index(i)
			if err := e.encodePrimitiveValue(fd.ElementType, vv.Interface()); err != nil {
				return pkg_errors.WithMessagef(err, "field[%s][%d]", fd.Name, i)
			}
		}
	} else if fd.ElementType == parse.FTStruct {
		sd := e.parser.GetStructByName(fd.StructName)
		for i := 0; i < v.Len(); i++ {
			vv := v.Index(i)
			if err := e.encodeStructField(sd, vv.Interface()); err != nil {
				return pkg_errors.WithMessagef(err, "field[%s][%d]", fd.Name, i)
			}
		}
	} else {
		panic(fmt.Sprintf("export data: bytes: encodeArrayValue: element type %d invalid", fd.Type))
	}
	return nil
}
