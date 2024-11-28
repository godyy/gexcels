package gexcels

import (
	"os"
)

// DataKind 导出数据类别
type DataKind int8

const (
	_         = DataKind(iota)
	DataJson  // json
	DataBytes // bytes
	dataKindMax
)

var dataKindStrings = [...]string{
	DataJson:  "json",
	DataBytes: "bytes",
}

func (dk DataKind) Valid() bool {
	return dk > 0 && dk < dataKindMax
}

func (dk DataKind) String() string {
	return dataKindStrings[dk]
}

var dataStringKinds = map[string]DataKind{
	DataJson.String():  DataJson,
	DataBytes.String(): DataBytes,
}

func DataKindFromString(s string) DataKind {
	return dataStringKinds[s]
}

// dataExporter 定义数据导出器需具备的功能
type dataExporter interface {
	export(p *Parser, path string) error
}

var dataExporters = [...]dataExporter{
	DataJson:  &jsonExporter{},
	DataBytes: &bytesExporter{},
}

// ExportData 导出数据到指定路径
func ExportData(p *Parser, kind DataKind, path string) error {
	if p == nil {
		panic("no parser specified")
	}

	if path == "" {
		return errExportPathEmpty
	}

	if !kind.Valid() {
		return errDataKindInvalid(kind)
	}

	expo := dataExporters[kind]
	if expo == nil {
		return errExporterNotFound
	}

	if err := os.MkdirAll(path, os.ModePerm); err != nil {
		return err
	}

	return expo.export(p, path)
}
