package gexcels

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"reflect"

	"github.com/godyy/gutils/buffer/bytes"
	pkg_errors "github.com/pkg/errors"
)

// bytesExporter
type bytesExporter struct{}

func (be *bytesExporter) export(p *Parser, path string) error {
	printf("export data bytes to [%s]", path)

	buf := bytes.NewBuffer(make([]byte, 0, 128))
	for _, table := range p.tables {
		if err := be.exportTableFile(p, table, path, buf); err != nil {
			return err
		}
	}
	return nil
}

func (be *bytesExporter) exportTableFile(p *Parser, td *Table, path string, buf *bytes.Buffer) error {
	if td.IsGlobal {
		return be.exportGlobalTableFile(p, td, path, buf)
	} else {
		return be.exportNormalTableFile(p, td, path, buf)
	}
}

func (be *bytesExporter) exportNormalTableFile(p *Parser, td *Table, path string, buf *bytes.Buffer) error {
	fileName := td.ExportTableName + ".bytes"
	filePath := filepath.Join(path, fileName)
	file, err := os.OpenFile(filePath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, os.ModePerm)
	if err != nil {
		return pkg_errors.WithMessagef(err, "export bytes: table[%s] to %s", td.Name, filePath)
	}
	defer file.Close()

	if _, err = buf.WriteVarint32(int32(len(td.Entries))); err == nil {
		str := base64.StdEncoding.EncodeToString(buf.Data())
		_, err = file.WriteString(str + "\n")
	}
	if err != nil {
		return pkg_errors.WithMessagef(err, "export bytes: table[%s] to %s, write entry count", td.Name, filePath)
	}
	buf.Reset()

	for _, entry := range td.Entries {
		if err := be.encodeTableEntry(p, td, entry, buf); err != nil {
			return pkg_errors.WithMessagef(err, "export bytes: table[%s] to %s", td.Name, filePath)
		}
		entryStr := base64.StdEncoding.EncodeToString(buf.Data())
		if _, err := file.WriteString(entryStr + "\n"); err != nil {
			return pkg_errors.WithMessagef(err, "export bytes: table[%s] to %s, write entry[%v]", td.Name, filePath, entry[idExportName])
		}
		buf.Reset()
	}
	printfGreen("export bytes: table[%s] to [%s]", td.Name, filePath)
	return nil
}

func (be *bytesExporter) exportGlobalTableFile(p *Parser, td *Table, path string, buf *bytes.Buffer) error {
	fileName := td.ExportTableName + ".bytes"
	filePath := filepath.Join(path, fileName)
	file, err := os.OpenFile(filePath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, os.ModePerm)
	if err != nil {
		return pkg_errors.WithMessagef(err, "export bytes: table[%s] to %s", td.Name, filePath)
	}
	defer file.Close()
	for i, fd := range td.Fields {
		value := td.getEntryByName(fd.Name)
		if value == nil {
			continue
		}
		if err := be.encodeFieldValue(p, fd.Field, i, value, buf); err != nil {
			return pkg_errors.WithMessagef(err, "export bytes: table[%s] to %s", fd.Name, filePath)
		}
		valueStr := base64.StdEncoding.EncodeToString(buf.Data())
		if _, err := file.WriteString(valueStr + "\n"); err != nil {
			return pkg_errors.WithMessagef(err, "export bytes: table[%s] to %s, write field[%v]", td.Name, filePath, fd.Name)
		}
		buf.Reset()
	}
	printfGreen("export bytes: table[%s] to [%s]", td.Name, filePath)
	return nil
}

func (be *bytesExporter) encodeTableEntry(p *Parser, td *Table, entry TableEntry, buf *bytes.Buffer) error {
	for i, fd := range td.Fields {
		fieldValue := entry[fd.Name]
		if fieldValue == nil {
			continue
		}
		if err := be.encodeFieldValue(p, fd.Field, i, fieldValue, buf); err != nil {
			return pkg_errors.WithMessagef(err, "entry[%v]", entry[idExportName])
		}
	}
	return nil
}

func (be *bytesExporter) encodeFieldValue(p *Parser, fd *Field, index int, value interface{}, buf *bytes.Buffer) error {
	// field index
	if _, err := buf.WriteVarint16(int16(index + 1)); err != nil {
		return pkg_errors.WithMessagef(err, "field[%s] index", fd.Name)
	}

	// field value
	if fd.Type.Primitive() {
		return be.encodePrimitiveValue(fd.Type, value, buf)
	} else if fd.Type == FTStruct {
		return be.encodeStructField(p, p.getStruct(fd.StructName), value, buf)
	} else if fd.Type == FTArray {
		return be.encodeArrayValue(p, fd, value, buf)
	} else {
		panic(errFieldTypeInvalid(fd.Type))
	}
}

func (be *bytesExporter) encodePrimitiveValue(ft FieldType, value interface{}, buf *bytes.Buffer) error {
	switch ft {
	case FTInt32:
		_, err := buf.WriteVarint32(value.(int32))
		return err
	case FTInt64:
		_, err := buf.WriteVarint64(value.(int64))
		return err
	case FTFloat32:
		return buf.WriteFloat32(value.(float32))
	case FTFloat64:
		return buf.WriteFloat64(value.(float64))
	case FTBool:
		return buf.WriteBool(value.(bool))
	case FTString:
		return buf.WriteString(value.(string))
	default:
		return errFieldTypeInvalid(ft)
	}
}

func (be *bytesExporter) encodeStructField(p *Parser, sd *Struct, value interface{}, buf *bytes.Buffer) error {
	var (
		v         = reflect.ValueOf(value).Elem()
		lastIndex int
	)
	for i, fd := range sd.Fields {
		fv := v.FieldByName(fd.ExportName)
		if !fv.IsValid() {
			continue
		}
		if err := be.encodeFieldValue(p, fd, i, fv.Interface(), buf); err != nil {
			return pkg_errors.WithMessagef(err, "field[%s]", fd.Name)
		}
		lastIndex = i
	}
	if lastIndex < len(sd.Fields)-1 {
		if _, err := buf.WriteVarint16(0); err != nil {
			return pkg_errors.WithMessage(err, "struct end marker")
		}
	}
	return nil
}

func (be *bytesExporter) encodeArrayValue(p *Parser, fd *Field, value interface{}, buf *bytes.Buffer) error {
	v := reflect.ValueOf(value)
	if _, err := buf.WriteVarint16(int16(v.Len())); err != nil {
		return pkg_errors.WithMessagef(err, "field[%s] length", fd.Name)
	}
	if fd.ElementType.Primitive() {
		for i := 0; i < v.Len(); i++ {
			vv := v.Index(i)
			if err := be.encodePrimitiveValue(fd.ElementType, vv.Interface(), buf); err != nil {
				return pkg_errors.WithMessagef(err, "field[%s][%d]", fd.Name, i)
			}
		}
	} else if fd.ElementType == FTStruct {
		sd := p.getStruct(fd.StructName)
		for i := 0; i < v.Len(); i++ {
			vv := v.Index(i)
			if err := be.encodeStructField(p, sd, vv.Interface(), buf); err != nil {
				return pkg_errors.WithMessagef(err, "field[%s][%d]", fd.Name, i)
			}
		}
	} else {
		panic(errArrayElementInvalid(fd.ElementType))
	}
	return nil
}
