package data

import (
	"context"
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

	p, err := parse.Parse(excelsPath, &parse.Options{Tags: []gexcels.Tag{"s"}})
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

	p, err := parse.Parse(excelsPath, &parse.Options{Tags: []gexcels.Tag{"s"}})
	if err != nil {
		t.Fatalf("parse %s, %v", excelsPath, err)
	}

	if err := ExportBytes(p, exportBytesPath); err != nil {
		t.Fatalf("export bytes to %s, %v", exportBytesPath, err)
	}
}

func TestExportBson(t *testing.T) {
	excelsPath := "../../internal/test/excels"
	mongoURI := "mongodb://localhost:27017"
	dbName := "gexcels_test"

	p, err := parse.Parse(excelsPath, &parse.Options{Tags: []gexcels.Tag{"s"}})
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

	if err := client.Database(dbName).Drop(context.Background()); err != nil {
		t.Fatalf("drop database %s, %v", dbName, err)
	}

	if err := ExportBson(p, client.Database(dbName)); err != nil {
		t.Fatalf("export bson to %s, %v", dbName, err)
	}
}
