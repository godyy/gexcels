package data

import (
	"testing"

	"github.com/godyy/gexcels"
	"github.com/godyy/gexcels/parse"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/writeconcern"
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

	p, err = parse.Parse(excelsPath, &parse.Options{FileTags: []gexcels.Tag{"test"}})
	if err != nil {
		t.Fatalf("parse %s with FileTags[test], %v", excelsPath, err)
	}

	if err := ExportJson(p, exportJsonPath); err != nil {
		t.Fatalf("export json to %s with FileTags[test], %v", exportJsonPath, err)
	}
}

func TestExportBytes(t *testing.T) {
	excelsPath := "../../internal/test/excels"
	exportBytesPath := "../../internal/test/export/data"

	p, err := parse.Parse(excelsPath, &parse.Options{})
	if err != nil {
		t.Fatalf("parse %s, %v", excelsPath, err)
	}

	if err := ExportBytes(p, exportBytesPath); err != nil {
		t.Fatalf("export bytes to %s, %v", exportBytesPath, err)
	}

	p, err = parse.Parse(excelsPath, &parse.Options{FileTags: []gexcels.Tag{"test"}})
	if err != nil {
		t.Fatalf("parse %s with FileTags[test], %v", excelsPath, err)
	}

	if err := ExportBytes(p, exportBytesPath); err != nil {
		t.Fatalf("export bytes to %s with FileTags[test], %v", exportBytesPath, err)
	}
}

func TestExportBson(t *testing.T) {
	excelsPath := "../../internal/test/excels"
	mongoURI := "mongodb://localhost:27017"
	dbName := "gexcels_test"

	p, err := parse.Parse(excelsPath, &parse.Options{})
	if err != nil {
		t.Fatalf("parse %s, %v", excelsPath, err)
	}

	opts := options.Client()
	opts.ApplyURI(mongoURI)
	opts.SetWriteConcern(writeconcern.Majority())
	client, err := mongo.Connect(opts)
	if err != nil {
		t.Fatalf("connect to %s, %v", mongoURI, err)
	}

	if err := ExportBson(p, client.Database(dbName)); err != nil {
		t.Fatalf("export bson to %s, %v", dbName, err)
	}
}
