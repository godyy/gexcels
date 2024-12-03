package gexcels

import (
	"strings"
	"text/template"
)

// templateGoStruct go结构体模版
var templateGoStruct = template.Must(template.New("go_struct").
	Funcs(template.FuncMap{
		"genFieldType": genGoStructFieldType,
		"genFieldTag":  genGoStructFieldTag,
	}).
	Parse(`type {{.Struct.ExportName}} struct {
	{{range $index,$item := .Struct.Fields}}{{if $index}}{{"\n"}}{{end}}{{$item.ExportName}} {{genFieldType $item}} {{genFieldTag $item.Name $.DataKind}} // {{$item.Desc}}{{end}}
}`))

// genGoStruct 生成go结构体文本
func genGoStruct(sd *Struct, dataKind DataKind) string {
	var sb strings.Builder
	if err := templateGoStruct.Execute(&sb, map[string]interface{}{
		"Struct":   sd,
		"DataKind": dataKind,
	}); err != nil {
		panic(err)
	}
	return sb.String()
}

// templateGoTableStruct go table结构体模版
var templateGoTableStruct = template.Must(template.New("go_struct").
	Funcs(template.FuncMap{
		"genFieldType": genGoStructFieldType,
		"genFieldTag":  genGoStructFieldTag,
	}).
	Parse(`type {{.Table.ExportEntryStructName}} struct {
	{{.IDFieldName}} {{.IDFieldType}} {{.IDFieldTag}} // {{.IDFieldDesc}}
	{{range $index,$item := .Table.Fields}}{{if eq $index 0}}{{continue}}{{else if gt $index 1}}{{"\n"}}{{end}}{{$item.ExportName}} {{genFieldType $item.Field}} {{genFieldTag $item.Name $.DataKind}} // {{$item.Desc}}{{end}}
}`))

// genGoTableStruct 生成go配置表结构体文本
func genGoTableStruct(td *Table, dataKind DataKind) string {
	var sb strings.Builder
	if err := templateGoTableStruct.Execute(&sb, map[string]interface{}{
		"Table":       td,
		"DataKind":    dataKind,
		"IDFieldName": idExportName,
		"IDFieldType": genGoStructFieldType(td.Fields[0].Field),
		"IDFieldTag":  genGoStructFieldTag(idJsonName, dataKind),
		"IDFieldDesc": td.Fields[0].Desc,
	}); err != nil {
		panic(err)
	}
	return sb.String()
}

// templateGoNormalTableFile go常规配置表文件模版
var templateGoNormalTableFile = template.Must(template.New("go_table_file").
	Funcs(template.FuncMap{
		"genEntryStruct":        genGoTableStruct,
		"genPrimitiveFieldType": genGoPrimitiveType,
	}).
	Parse(`package {{.PkgName}}

{{genEntryStruct .Table .DataKind}}

// {{.Table.ExportTableStructName}} {{.Table.Desc}}
type {{.Table.ExportTableStructName}} struct {
	entries []*{{.Table.ExportEntryStructName}}
	{{$hasUnique := 0}}{{range $index, $item := .Table.Fields}}{{if $item.Unique}}{{if $hasUnique}}{{"\n"}}{{end}}{{$hasUnique = true}}by{{$item.ExportName}} map[{{genPrimitiveFieldType $item.Type}}]*{{$.Table.ExportEntryStructName}}{{end}}{{end}}
}

func (t *{{.Table.ExportTableStructName}}) name() string {
	return "{{.Table.ExportTableName}}"
}

func (t *{{.Table.ExportTableStructName}}) List() []*{{.Table.ExportEntryStructName}} { return t.entries }

{{$hasUnique := 0}}{{range $index, $item := .Table.Fields}}{{if $item.Unique}}{{if $hasUnique}}{{"\n\n"}}{{end}}{{$hasUnique = true}}func (t *{{$.Table.ExportTableStructName}}) GetBy{{$item.ExportName}}({{if eq $index 0}}id{{else}}v{{end}} {{genPrimitiveFieldType $item.Type}}) *{{$.Table.ExportEntryStructName}} {
	return t.by{{$item.ExportName}}[{{if eq $index 0}}id{{else}}v{{end}}]
}{{end}}{{end}}
`))

// genGoNormalTableFile 生成go常规配置表代码文本
func genGoNormalTableFile(td *Table, pkgName string, dataKind DataKind) string {
	var sb strings.Builder
	if err := templateGoNormalTableFile.Execute(&sb, map[string]interface{}{
		"Table":    td,
		"PkgName":  pkgName,
		"DataKind": dataKind,
	}); err != nil {
		panic(err)
	}
	return sb.String()
}

// templateGoGlobalTableFile go全局配置表文件模版
var templateGoGlobalTableFile = template.Must(template.New("go_global_table_file").
	Funcs(template.FuncMap{
		"genStructField": genGoStructField,
	}).
	Parse(`package {{.PkgName}}

// {{.Table.ExportTableStructName}} {{.Table.Desc}}
type {{.Table.ExportTableStructName}} struct {
	{{range $index, $item := .Table.Fields}}{{if $index}}{{"\n"}}{{end}}{{genStructField $item.Field $.DataKind}}{{end}}
}

func (t *{{.Table.ExportTableStructName}}) name() string {
	return "{{.Table.ExportTableName}}"
}
`))

// genGoGlobalTableFile 生成go全局配置表代码文本
func genGoGlobalTableFile(td *Table, pkgName string, dataKind DataKind) string {
	fields := make([]string, len(td.Fields))
	for i, fd := range td.Fields {
		fields[i] = genGoStructField(fd.Field, dataKind)
	}
	var sb strings.Builder
	if err := templateGoGlobalTableFile.Execute(&sb, map[string]interface{}{
		"PkgName":  pkgName,
		"Table":    td,
		"DataKind": dataKind,
	}); err != nil {
		panic(err)
	}
	return sb.String()
}

// templateGoTableMgrFile go配置表管理器文件模版
var templateGoTableMgrFile = template.Must(template.New("go_table_mgr_file").
	Funcs(template.FuncMap{
		"toFieldName": toCamelCase,
	}).
	Parse(`package {{.PkgName}}

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	pkg_errors "github.com/pkg/errors"
)

// ErrTableNotExist 配置表不存在
var ErrTableNotExist = errors.New("table not exist")

// table 配置表接口抽象
type table interface{
	name() string 				// 配置表名
	loadData(data []byte) error // 数据加载
}

// TableManager 配置表管理器
type TableManager struct {
	tables map[string]table
	{{range $index, $item := .Tables}}{{if $index}}{{"\n"}}{{end}}{{toFieldName $item.Name}} *{{$item.ExportTableStructName}}{{end}}
}

func NewTableManager() *TableManager {
	tm := &TableManager{
		tables: make(map[string]table),
	}
	tm.initTables()
	return tm
}

func (tm *TableManager) initTables() {
	{{range $index, $item := .Tables}}{{if $index}}{{"\n"}}{{end}}tm.{{$fieldName := toFieldName $item.Name}}{{$fieldName}} = new({{$item.ExportTableStructName}})
	tm.addTable(tm.{{$fieldName}}){{end}}
}

func (tm *TableManager) addTable(table table) {
	if _, ok := tm.tables[table.name()]; ok {
		panic("table " + table.name() + " already exists")
	}
	tm.tables[table.name()] = table
}

func (tm *TableManager) getTable(name string) table {
	return tm.tables[name]
}

func (tm *TableManager) loadTableData(tableName string, data []byte) error {
	table := tm.getTable(tableName)
	if table == nil {
		return ErrTableNotExist
	}
	return table.loadData(data)
}

func (tm *TableManager) LoadAllTableData(data map[string][]byte) error {
	if len(data) == 0 {
		return nil
	}
	for tableName, data := range data {
		if err := tm.loadTableData(tableName, data); err != nil {
			return pkg_errors.WithMessagef(err, "load table[%s] data", tableName)
		}
	}
	return nil
}

func (tm *TableManager) loadTableFromFile(file string) error {
	tableName := strings.TrimSuffix(filepath.Base(file), filepath.Ext(file))
	table := tm.getTable(tableName)
	if table == nil {
		return ErrTableNotExist
	}
	data, err := os.ReadFile(file)
	if err != nil {
		return err
	}
	return table.loadData(data)
}

func (tm *TableManager) loadFromDir(dir string, ext string) error {
	return filepath.Walk(dir, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || filepath.Ext(info.Name()) != ext {
			return nil
		}
		if err := tm.loadTableFromFile(path); err != nil {
			return pkg_errors.WithMessagef(err, "load table in %s", path)
		}
		return nil
	})
}

{{range $index, $item := .Tables}}{{if $index}}{{"\n\n"}}{{end}}{{$methodName := toFieldName $item.Name true}}// {{$methodName}} {{$item.Desc}}{{"\n"}}func (tm *TableManager) {{$methodName}}() *{{$item.ExportTableStructName}} { return tm.{{toFieldName $item.Name}} }{{end}}
`))

// genGoTableMgrFile 生成go配置表管理器代码文本
func genGoTableMgrFile(tables []*Table, pkgName string, dataKind DataKind) string {
	var sb strings.Builder
	if err := templateGoTableMgrFile.Execute(&sb, map[string]interface{}{
		"PkgName":  pkgName,
		"Tables":   tables,
		"DataKind": dataKind,
	}); err != nil {
		panic(err)
	}
	return sb.String()
}

// templateGoNormalTableLoadJson go常规配置表json加载模版
var templateGoNormalTableLoadJson = template.Must(template.New("go_normal_table_load_json").
	Funcs(template.FuncMap{
		"genPrimitiveFieldType": genGoPrimitiveType,
	}).
	Parse(`func (t *{{.Table.ExportTableStructName}}) loadData(data []byte) error {
	if err := loadHelper.unmarshal(data, &t.entries); err != nil {
		return err
	}
	{{$hasUnique := false}}
	{{range $index,$item := .Table.Fields}}{{if $item.Unique}}{{if $hasUnique}}{{"\n"}}{{end}}{{$hasUnique = true}}t.by{{$item.ExportName}} = make(map[{{genPrimitiveFieldType $item.Type}}]*{{$.Table.ExportEntryStructName}}, len(t.entries)){{end}}{{end}}
	for _, e := range t.entries {
		{{$hasUnique := false}}
		{{range $index, $item := .Table.Fields}}{{if $item.Unique}}{{if $hasUnique}}{{"\n"}}{{end}}{{$hasUnique = true}}t.by{{$item.ExportName}}[e.{{$item.ExportName}}] = e{{end}}{{end}}
	}
	return nil
}`))

// genGoNormalTableLoadJson 生成go常规配置表json加载代码
func genGoNormalTableLoadJson(td *Table) string {
	uniqueFields := make([]struct {
		FieldName string
		FieldType string
	}, 0)
	for i := 1; i < len(td.Fields); i++ {
		fd := td.Fields[i]
		if fd.Unique() {
			uniqueFields = append(uniqueFields, struct {
				FieldName string
				FieldType string
			}{FieldName: fd.ExportName, FieldType: genGoPrimitiveType(fd.Type)})
		}
	}
	var sb strings.Builder
	if err := templateGoNormalTableLoadJson.Execute(&sb, map[string]interface{}{
		"Table": td,
	}); err != nil {
		panic(err)
	}
	return sb.String()
}

// templateGoGlobalTableLoadJson go全局配置表json加载模版
var templateGoGlobalTableLoadJson = template.Must(template.New("go_global_table_load_json").Parse(`func (t *{{.TableName}}) loadData(data []byte) error {
	return loadHelper.unmarshal(data, t)
}`))

// genGoGlobalTableLoadJson 生成go全局配置表json加载代码
func genGoGlobalTableLoadJson(td *Table) string {
	var sb strings.Builder
	if err := templateGoGlobalTableLoadJson.Execute(&sb, map[string]interface{}{
		"TableName": td.ExportTableStructName,
	}); err != nil {
		panic(err)
	}
	return sb.String()
}

// templateGoLoadJsonFile go配置表json加载代码文件模版
var templateGoLoadJsonFile = template.Must(template.New("go_load_json_file").
	Funcs(template.FuncMap{
		"genNormalLoadJson": genGoNormalTableLoadJson,
		"genGlobalLoadJson": genGoGlobalTableLoadJson,
	}).
	Parse(`package {{.PkgName}}

import (
	"encoding/json"
)

func (tm *TableManager) LoadFromDir(dir string) error {
	return tm.loadFromDir(dir, ".json")
}

{{range $index,$item := .Tables}}{{if $index}}{{"\n\n"}}{{end}}{{if $item.IsGlobal}}{{genGlobalLoadJson $item}}{{else}}{{genNormalLoadJson $item}}{{end}}{{end}}

type jsonHelper struct{}

var loadHelper = &jsonHelper{}

func (jh *jsonHelper) unmarshal(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}
`))

// genGoLoadJsonFile 生成go配置表json加载代码
func genGoLoadJsonFile(tds []*Table, pkgName string) string {
	var sb strings.Builder
	if err := templateGoLoadJsonFile.Execute(&sb, map[string]interface{}{
		"Tables":  tds,
		"PkgName": pkgName,
	}); err != nil {
		panic(err)
	}
	return sb.String()
}

// templateGoNormalLoadBytes go常规配置表bytes加载模版
var templateGoNormalLoadBytes = template.Must(template.New("go_normal_table_load_bytes").
	Funcs(template.FuncMap{
		"genPrimitiveFieldType": genGoPrimitiveType,
	}).
	Parse(`func (t *{{.Table.ExportTableStructName}}) loadData(data []byte) error {
	if err := loadHelper.loadEntries(data, reflect.ValueOf(&t.entries).Elem()); err != nil {
		return err
	}
	{{$hasUnique := false}}
	{{range $index,$item := .Table.Fields}}{{if $item.Unique}}{{if $hasUnique}}{{"\n"}}{{end}}{{$hasUnique = true}}t.by{{$item.ExportName}} = make(map[{{genPrimitiveFieldType $item.Type}}]*{{$.Table.ExportEntryStructName}}, len(t.entries)){{end}}{{end}}
	for _, e := range t.entries {
		{{$hasUnique := false}}
		{{range $index, $item := .Table.Fields}}{{if $item.Unique}}{{if $hasUnique}}{{"\n"}}{{end}}{{$hasUnique = true}}t.by{{$item.ExportName}}[e.{{$item.ExportName}}] = e{{end}}{{end}}
	}
	return nil
}`))

// genGoNormalTableLoadBytes 生成go常规配置表bytes加载代码
func genGoNormalTableLoadBytes(td *Table) string {
	var sb strings.Builder
	if err := templateGoNormalLoadBytes.Execute(&sb, map[string]interface{}{
		"Table": td,
	}); err != nil {
		panic(err)
	}
	return sb.String()
}

// templateGoGlobalLoadBytes go全局配置表bytes加载模版
var templateGoGlobalLoadBytes = template.Must(template.New("go_global_table_load_bytes").Parse(`func (t *{{.TableName}}) loadData(data []byte) error {
	return loadHelper.loadGlobal(data, reflect.ValueOf(t))
}`))

// genGoGlobalTableLoadBytes 生成go全局配置表bytes加载代码
func genGoGlobalTableLoadBytes(td *Table) string {
	var sb strings.Builder
	if err := templateGoGlobalLoadBytes.Execute(&sb, map[string]interface{}{
		"TableName": td.ExportTableStructName,
	}); err != nil {
		panic(err)
	}
	return sb.String()
}

// templateGoLoadBytesFile go配置表bytes加载代码文件模版
var templateGoLoadBytesFile = template.Must(template.New("go_load_bytes_file").
	Funcs(template.FuncMap{
		"genNormalLoadBytes": genGoNormalTableLoadBytes,
		"genGlobalLoadBytes": genGoGlobalTableLoadBytes,
	}).
	Parse(`package {{.PkgName}}

import (
	"bufio"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"reflect"

	"github.com/godyy/gutils/buffer/bytes"
	pkg_errors "github.com/pkg/errors"
)

func (tm *TableManager) LoadFromDir(dir string) error {
	return tm.loadFromDir(dir, ".bytes")
}

{{range $index,$item := .Tables}}{{if $index}}{{"\n"}}{{end}}{{if $item.IsGlobal}}{{genGlobalLoadBytes $item}}{{else}}{{genNormalLoadBytes $item}}{{end}}{{end}}

var loadHelper = &bytesHelper{}

type bytesHelper struct{}

func (bh *bytesHelper) readFieldIndex(buf *bytes.Buffer) (int16, error) {
	return buf.ReadVarint16()
}

func (bh *bytesHelper) loadLine(dataBuf *bufio.Reader, lineBuf *bytes.Buffer) error {
	var (
		line, l  []byte
		isPrefix bool
		err      error
	)
	for {
		l, isPrefix, err = dataBuf.ReadLine()
		if err != nil {
			return err
		}
		if isPrefix {
			line = append(line, l...)
		} else {
			if line == nil {
				line = l
			} else {
				line = append(line, l...)
			}
			break
		}
	}
	line, err = base64.StdEncoding.DecodeString(string(line))
	if err != nil {
		return err
	}
	lineBuf.SetBuf(line)
	return nil
}

func (bh *bytesHelper) loadEntries(data []byte, entries reflect.Value) error {
	var (
		dataBuf = bufio.NewReader(bytes.NewBuffer(data))
		lineBuf bytes.Buffer
	)

	if err := bh.loadLine(dataBuf, &lineBuf); err != nil {
		if errors.Is(err, io.EOF) {
			return nil
		}
		return pkg_errors.WithMessage(err, "load entry count line")
	}

	n, err := lineBuf.ReadVarint32()
	if err != nil {
		return pkg_errors.WithMessage(err, "load entry count")
	}

	if n == 0 {
		return nil
	}

	entryType := entries.Type().Elem()
	entryArray := reflect.MakeSlice(entries.Type(), 0, int(n))
	for i := int32(0); i < n; i++ {
		if err := bh.loadLine(dataBuf, &lineBuf); err != nil {
			return pkg_errors.WithMessagef(err, "load entry[%d] line", i)
		}
		entry := reflect.New(entryType.Elem())
		if err := bh.loadValue(&lineBuf, entry.Elem()); err != nil {
			return pkg_errors.WithMessagef(err, "load entry[%d]", i)
		}
		entryArray = reflect.Append(entryArray, entry)
	}
	entries.Set(entryArray)

	return nil
}

func (bh *bytesHelper) loadGlobal(data []byte, s reflect.Value) error {
	var (
		dataBuf = bufio.NewReader(bytes.NewBuffer(data))
		lineBuf bytes.Buffer
	)

	v := s.Elem()
	n := v.NumField()
	i := 0
	for {
		if err := bh.loadLine(dataBuf, &lineBuf); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return pkg_errors.WithMessagef(err, "load line[%d]", i)
		}
		index, err := bh.readFieldIndex(&lineBuf)
		if err != nil {
			return pkg_errors.WithMessage(err, "load field index")
		}
		field := v.Field(int(index - 1))
		if err := bh.loadValue(&lineBuf, field); err != nil {
			return pkg_errors.WithMessagef(err, "load field[%d]", index-1)
		}
		i++
		if int(index) >= n {
			break
		}
	}

	return nil
}

func (bh *bytesHelper) loadValue(buf *bytes.Buffer, v reflect.Value) (err error) {
	switch v.Kind() {
	case reflect.Int32: // FTInt32
		var i32 int32
		i32, err = buf.ReadVarint32()
		if err == nil {
			v.SetInt(int64(i32))
		}
	case reflect.Int64: // FTInt64
		var i64 int64
		i64, err = buf.ReadVarint64()
		if err == nil {
			v.SetInt(i64)
		}
	case reflect.Float32: // FTFloat32
		var f32 float32
		f32, err = buf.ReadFloat32()
		if err == nil {
			v.SetFloat(float64(f32))
		}
	case reflect.Float64: // FTFloat64
		var f64 float64
		f64, err = buf.ReadFloat64()
		if err == nil {
			v.SetFloat(f64)
		}
	case reflect.Bool: // FTBool
		var b bool
		b, err = buf.ReadBool()
		if err == nil {
			v.SetBool(b)
		}
	case reflect.String: // FTString
		var s string
		s, err = buf.ReadString()
		if err == nil {
			v.SetString(s)
		}
	case reflect.Ptr, reflect.Struct: // FTStruct
		err = bh.loadStruct(buf, v)
	case reflect.Slice: // FTArray
		err = bh.loadArray(buf, v)
	default:
		panic(fmt.Errorf("%s invalid", v.String()))
	}
	return err
}

func (bh *bytesHelper) loadStruct(buf *bytes.Buffer, s reflect.Value) error {
	var (
		v     reflect.Value
		index int16
		err   error
	)

	if s.Kind() == reflect.Ptr {
		ptr := reflect.New(s.Type().Elem())
		v = ptr.Elem()
		s.Set(ptr)
	} else {
		v = s
	}

	n := v.NumField()
	for {
		index, err = bh.readFieldIndex(buf)
		if err != nil {
			if err == io.EOF {
				break
			}
			return pkg_errors.WithMessage(err, "load field index")
		}
		if index == 0 {
			break
		}
		field := v.Field(int(index - 1))
		if err = bh.loadValue(buf, field); err != nil {
			return pkg_errors.WithMessagef(err, "load field[%d]", index)
		}
		if int(index) >= n {
			break
		}
	}

	return nil
}

func (bh *bytesHelper) loadArrayLength(buf *bytes.Buffer) (int, error) {
	arrayLen, err := buf.ReadVarint16()
	if err != nil {
		return 0, pkg_errors.WithMessage(err, "load array length")
	}
	return int(arrayLen), nil
}

func (bh *bytesHelper) loadArray(buf *bytes.Buffer, array reflect.Value) error {
	elementType := array.Type().Elem()
	arrayLen, err := bh.loadArrayLength(buf)
	if err != nil {
		return err
	}
	if arrayLen == 0 {
		return nil
	}
	arrayValue := reflect.MakeSlice(array.Type(), 0, arrayLen)
	for i := 0; i < arrayLen; i++ {
		elementPtr := reflect.New(elementType)
		element := elementPtr.Elem()
		if err := bh.loadValue(buf, element); err != nil {
			return pkg_errors.WithMessagef(err, "load array[%d]", i)
		}
		arrayValue = reflect.Append(arrayValue, element)
	}
	array.Set(arrayValue)
	return nil
}
`))

// genGoLoadBytesFile 生成go配置表bytes加载代码
func genGoLoadBytesFile(tds []*Table, pkgName string) string {
	var sb strings.Builder
	if err := templateGoLoadBytesFile.Execute(&sb, map[string]interface{}{
		"Tables":  tds,
		"PkgName": pkgName,
	}); err != nil {
		panic(err)
	}
	return sb.String()
}
