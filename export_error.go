package gexcels

import (
	"errors"
	"fmt"
)

var errExportPathEmpty = errors.New("export path is empty")
var errExporterNotFound = errors.New("exporter not found")
var errNonCodeKindOptions = errors.New("code kind options not specified")
var errGoPkgEmpty = errors.New("go package is empty")

func errCodeKindInvalid(codeKind CodeKind) error {
	return fmt.Errorf("code-kind %d invalid", codeKind)
}

func errDataKindInvalid(dataKind DataKind) error {
	return fmt.Errorf("data-kind %d invalid", dataKind)
}
