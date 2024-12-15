package code

import (
	"github.com/godyy/gexcels/export"
	"github.com/godyy/gexcels/parse"
	"testing"
)

func TestExportGo(t *testing.T) {
	excelsPath := "../../internal/test/excels"
	exportGoPath := "../../internal/test/export/go_json"

	p, err := parse.Parse(excelsPath, &parse.Options{})
	if err != nil {
		t.Fatalf("parse %s, %v", excelsPath, err)
	}

	if err := ExportGo(p, exportGoPath, &Options{
		DataKind: export.DataJson,
	}, &GoOptions{PkgName: "test"}); err != nil {
		t.Fatalf("export go to %s, %v", exportGoPath, err)
	}
}
