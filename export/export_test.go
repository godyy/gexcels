package export

import (
	"testing"

	"github.com/godyy/gexcels/export/code"
	"github.com/godyy/gexcels/export/data"
	"github.com/godyy/gexcels/export/internal/define"
	"github.com/godyy/gexcels/parse"
)

func TestExportGoAndJson(t *testing.T) {
	excelsPath := "../internal/test/excels"
	exportGoPath := "../internal/test/export/go_json"
	exportJsonPath := "../internal/test/export/data"

	p, err := parse.Parse(excelsPath, &parse.Options{})
	if err != nil {
		t.Fatalf("parse %s, %v", excelsPath, err)
	}

	if err := code.ExportGo(p, exportGoPath, &code.Options{
		DataKind: define.DataJson,
	}, &code.GoOptions{PkgName: "test"}); err != nil {
		t.Fatalf("export go to %s, %v", exportGoPath, err)
	}

	if err := data.ExportJson(p, exportJsonPath); err != nil {
		t.Fatalf("export json to %s, %v", exportJsonPath, err)
	}
}

func TestExportGoAndBytes(t *testing.T) {
	excelsPath := "../internal/test/excels"
	exportGoPath := "../internal/test/export/go_bytes"
	exportBytesPath := "../internal/test/export/data"

	p, err := parse.Parse(excelsPath, &parse.Options{FileTags: []parse.Tag{"test"}})
	if err != nil {
		t.Fatalf("parse %s, %v", excelsPath, err)
	}

	if err := code.ExportGo(p, exportGoPath, &code.Options{
		DataKind: define.DataBytes,
	}, &code.GoOptions{PkgName: "test"}); err != nil {
		t.Fatalf("export go to %s, %v", exportGoPath, err)
	}

	if err := data.ExportBytes(p, exportBytesPath); err != nil {
		t.Fatalf("export bytes to %s, %v", exportBytesPath, err)
	}
}
