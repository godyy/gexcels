package code

import (
	"testing"

	"github.com/godyy/gexcels"
	"github.com/godyy/gexcels/export"
	"github.com/godyy/gexcels/parse"
)

func TestExportGoJson(t *testing.T) {
	excelsPath := "../../internal/test/excels"
	exportGoPath := "../../internal/test/export/go_json"

	p, err := parse.Parse(excelsPath, &parse.Options{
		OnlyFields: false,
		Tags:       []gexcels.Tag{"s"},
	})
	if err != nil {
		t.Fatalf("parse %s, %v", excelsPath, err)
	}

	if err := ExportGo(p, exportGoPath, &Options{
		DataKind: export.DataJson,
	}, &GoOptions{PkgName: "test"}); err != nil {
		t.Fatalf("export go to %s, %v", exportGoPath, err)
	}
}

func TestExportGoBytes(t *testing.T) {
	excelsPath := "../../internal/test/excels"
	exportGoPath := "../../internal/test/export/go_bytes"

	p, err := parse.Parse(excelsPath, &parse.Options{
		OnlyFields: false,
		Tags:       []gexcels.Tag{"s"},
	})
	if err != nil {
		t.Fatalf("parse %s, %v", excelsPath, err)
	}

	if err := ExportGo(p, exportGoPath, &Options{
		DataKind: export.DataBytes,
	}, &GoOptions{PkgName: "test"}); err != nil {
		t.Fatalf("export go to %s, %v", exportGoPath, err)
	}
}

func TestExportGoBson(t *testing.T) {
	excelsPath := "../../internal/test/excels"
	exportGoPath := "../../internal/test/export/go_bson"

	p, err := parse.Parse(excelsPath, &parse.Options{
		OnlyFields: false,
		Tags:       []gexcels.Tag{"s"},
	})
	if err != nil {
		t.Fatalf("parse %s, %v", excelsPath, err)
	}

	if err := ExportGo(p, exportGoPath, &Options{
		DataKind: export.DataBson,
	}, &GoOptions{PkgName: "test"}); err != nil {
		t.Fatalf("export go to %s, %v", exportGoPath, err)
	}
}
