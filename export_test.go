package gexcels

import (
	"testing"
)

func TestExportGoAndJson(t *testing.T) {
	excelsPath := "./internal/test/excels"
	exportGoPath := "./internal/test/export/go_json"
	exportJsonPath := "./internal/test/export/json"

	p, err := Parse(excelsPath, &ParseOptions{
		Tag:            []string{"s"},
		StructFilePath: "struct_define.xlsx",
	})
	if err != nil {
		t.Fatalf("parse %s, %v", excelsPath, err)
	}

	if err := ExportCode(p, CodeGo, exportGoPath, &ExportCodeOptions{
		StructFileName:   "structs",
		TableMgrFileName: "table_manager",
		DataKind:         DataJson,
		KindOptions:      &CodeGoOptions{PkgName: "test"},
	}); err != nil {
		t.Fatalf("export go to %s, %v", exportGoPath, err)
	}

	if err := ExportData(p, DataJson, exportJsonPath); err != nil {
		t.Fatalf("export json to %s, %v", exportJsonPath, err)
	}
}

func TestExportGoAndBytes(t *testing.T) {
	excelsPath := "./internal/test/excels"
	exportGoPath := "./internal/test/export/go_bytes"
	exportBytesPath := "./internal/test/export/bytes"

	p, err := Parse(excelsPath, &ParseOptions{
		Tag:            []string{"s"},
		StructFilePath: "struct_define.xlsx",
	})
	if err != nil {
		t.Fatalf("parse %s, %v", excelsPath, err)
	}

	if err := ExportCode(p, CodeGo, exportGoPath, &ExportCodeOptions{
		StructFileName:   "structs",
		TableMgrFileName: "table_manager",
		DataKind:         DataBytes,
		KindOptions:      &CodeGoOptions{PkgName: "test"},
	}); err != nil {
		t.Fatalf("export go to %s, %v", exportGoPath, err)
	}

	if err := ExportData(p, DataBytes, exportBytesPath); err != nil {
		t.Fatalf("export bytes to %s, %v", exportBytesPath, err)
	}
}
