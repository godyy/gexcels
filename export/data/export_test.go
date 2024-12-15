package data

import (
	"github.com/godyy/gexcels"
	"github.com/godyy/gexcels/parse"
	"testing"
)

func TestExportJson(t *testing.T) {
	excelsPath := "../../internal/test/excels"
	exportJsonPath := "../../internal/test/export/data"

	p, err := parse.Parse(excelsPath, &parse.Options{})
	if err != nil {
		t.Fatalf("parse %s, %v", excelsPath, err)
	}

	if err := ExportJson(p, exportJsonPath); err != nil {
		t.Fatalf("export json to %s, %v", exportJsonPath, err)
	}
}

func TestExportBytes(t *testing.T) {
	excelsPath := "../../internal/test/excels"
	exportBytesPath := "../../internal/test/export/data"

	p, err := parse.Parse(excelsPath, &parse.Options{FileTags: []gexcels.Tag{"test"}})
	if err != nil {
		t.Fatalf("parse %s, %v", excelsPath, err)
	}

	if err := ExportBytes(p, exportBytesPath); err != nil {
		t.Fatalf("export bytes to %s, %v", exportBytesPath, err)
	}
}
