package data

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"reflect"

	"github.com/godyy/gexcels"
	internal_define "github.com/godyy/gexcels/export"
	"github.com/godyy/gexcels/internal/log"
	"github.com/godyy/gexcels/parse"
	"github.com/godyy/gutils/buffer/bytes"
	pkg_errors "github.com/pkg/errors"
)

// ExportBytes 导出bytes格式数据
func ExportBytes(p *parse.Parser, path string) error {
	if p == nil {
		return ErrNoParserSpecified
	}

	if path == "" {
		return ErrNoPathSpecified
	}

	return doExport(&bytesExporter{baseExporter: newBaseExporter(p), path: path, buf: bytes.NewBuffer(make([]byte, 0, 128))})
}

// bytesExporter bytes导出器
type bytesExporter struct {
	baseExporter
	path string
	buf  *bytes.Buffer
}

func (e *bytesExporter) kind() internal_define.DataKind {
	return internal_define.DataBytes
}

func (e *bytesExporter) export() error {
	log.Printf("export data bytes to [%s]", e.path)

	if err := os.MkdirAll(e.path, os.ModePerm); err != nil {
		return pkg_errors.WithMessagef(err, "mkdir")
	}

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
	fileName := genTableFileName(td.Name, ".bytes")
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
	fileName := genTableFileName(td.Name, ".bytes")
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
func (e *bytesExporter) encodeTableEntry(td *parse.Table, entry gexcels.TableEntry) error {
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
func (e *bytesExporter) encodeFieldValue(fd *gexcels.Field, index int, value any) error {
	// field index
	if _, err := e.buf.WriteVarint16(int16(index + 1)); err != nil {
		return pkg_errors.WithMessagef(err, "field[%s] index", fd.Name)
	}

	return e.encodeValue(fd.FieldTypeInfo, value, "field["+fd.Name+"]")
}

// encodePrimitiveValue 编码primitive值
func (e *bytesExporter) encodePrimitiveValue(ft gexcels.FieldType, value any) error {
	switch ft {
	case gexcels.FTInt32:
		_, err := e.buf.WriteVarint32(value.(int32))
		return err
	case gexcels.FTInt64:
		_, err := e.buf.WriteVarint64(value.(int64))
		return err
	case gexcels.FTFloat32:
		return e.buf.WriteFloat32(value.(float32))
	case gexcels.FTFloat64:
		return e.buf.WriteFloat64(value.(float64))
	case gexcels.FTBool:
		return e.buf.WriteBool(value.(bool))
	case gexcels.FTString:
		return e.buf.WriteString(value.(string))
	default:
		panic(fmt.Sprintf("export data: bytes: encodePrimitiveValue: field type %d invalid", ft))
	}
}

// encodeEnumField 编码枚举字段
func (e *bytesExporter) encodeEnumField(ft *gexcels.FieldTypeInfo, value any) error {
	enum := e.parser.GetEnum(ft.GetName())
	return e.encodePrimitiveValue(enum.Type, value)
}

// encodeStructField 编码struct字段
func (e *bytesExporter) encodeStructField(ft *gexcels.FieldTypeInfo, value any) error {
	var (
		sd        = e.parser.GetStructByName(ft.GetName())
		v         = reflect.ValueOf(value)
		lastIndex int
	)
	for i, fd := range sd.Fields {
		fv := v.MapIndex(reflect.ValueOf(fd.Name))
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

// encodeValue 按字段类型递归编码值。
// 复用 FieldTypeInfo，保证 array/map 等复杂类型无需在多处重复分支逻辑。
func (e *bytesExporter) encodeValue(ti *gexcels.FieldTypeInfo, value any, path string) error {
	if ti == nil {
		return fmt.Errorf("%s type is nil", path)
	}

	switch ti.Type {
	case gexcels.FTInt32, gexcels.FTInt64, gexcels.FTFloat32, gexcels.FTFloat64, gexcels.FTBool, gexcels.FTString:
		return e.encodePrimitiveValue(ti.Type, value)
	case gexcels.FTEnum:
		return e.encodeEnumField(ti, value)
	case gexcels.FTStruct:
		return e.encodeStructField(ti, value)
	case gexcels.FTArray:
		return e.encodeArrayValue(ti, value, path)
	case gexcels.FTMap:
		return e.encodeMapValue(ti, value, path)
	default:
		return fmt.Errorf("%s type %d invalid", path, ti.Type)
	}
}

// encodeArrayValue 编码数组：长度 + N 个元素（元素类型递归编码）。
func (e *bytesExporter) encodeArrayValue(ft *gexcels.FieldTypeInfo, value any, path string) error {
	elemType := ft.GetElementType()
	v := reflect.ValueOf(value)
	if v.Kind() != reflect.Slice {
		return fmt.Errorf("%s must be slice", path)
	}
	if _, err := e.buf.WriteVarint16(int16(v.Len())); err != nil {
		return pkg_errors.WithMessagef(err, "%s length", path)
	}
	for i := 0; i < v.Len(); i++ {
		vv := v.Index(i)
		if err := e.encodeValue(elemType, vv.Interface(), fmt.Sprintf("%s[%d]", path, i)); err != nil {
			return err
		}
	}
	return nil
}

// encodeMapValue 编码 map：长度 + (key,value)*N。
func (e *bytesExporter) encodeMapValue(ft *gexcels.FieldTypeInfo, value any, path string) error {
	keyType := ft.GetMapKeyType()
	valueType := ft.GetMapValueType()
	if keyType == nil || valueType == nil {
		return fmt.Errorf("%s map key/value type nil", path)
	}

	v := reflect.ValueOf(value)
	if v.Kind() != reflect.Map {
		return fmt.Errorf("%s must be map", path)
	}

	if _, err := e.buf.WriteVarint16(int16(v.Len())); err != nil {
		return pkg_errors.WithMessagef(err, "%s length", path)
	}

	keys := e.sortMapKeys(keyType, v.MapKeys())
	for _, key := range keys {
		if err := e.encodeValue(keyType, key.Interface(), path+" key"); err != nil {
			return err
		}

		value := v.MapIndex(key)
		if err := e.encodeValue(valueType, value.Interface(), fmt.Sprintf("%s[%v]", path, key.Interface())); err != nil {
			return err
		}
	}

	return nil
}
