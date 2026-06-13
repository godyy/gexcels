package main

import (
	"flag"
	"log"
	"strings"

	"github.com/godyy/gexcels"
	"github.com/godyy/gexcels/export"
	"github.com/godyy/gexcels/export/code"
	"github.com/godyy/gexcels/export/data"
	"github.com/godyy/gexcels/parse"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/writeconcern"
)

var (
	excelDir  = flag.String("excel-dir", "", "excel directory")
	tag       = flag.String("tag", "", "specify tags for filtering fields, e.g. \"c/s\"")
	sCodeKind = flag.String("code-kind", "go", "for exporting code, only has \"go\" now")
	sDataKind = flag.String("data-kind", "json", "for exporting data, has [\"json\", \"bytes\", \"bson\"]")
	codeDir   = flag.String("code-dir", "", "output code directory")
	dataDir   = flag.String("data-dir", "", "output data directory")
	goPackage = flag.String("go-package", "", "go package name for exporting go code")
	mongoURI  = flag.String("mongo-uri", "", "mongo uri for exporting bson data, must specified when data kind is \"bson\"")
	mongoDB   = flag.String("mongo-db", "", "mongo db name for exporting bson data, must specified when data kind is \"bson\"")
)

func main() {
	var (
		codeKind     export.CodeKind
		dataKind     export.DataKind
		parseOptions parse.Options
		codeOptions  code.Options
	)

	flag.Parse()

	if *excelDir == "" {
		log.Fatalf("excel directory empty")
	}

	codeKind.FromString(*sCodeKind)
	if !codeKind.Valid() {
		log.Fatalf("code kind \"%s\" invalid", *sCodeKind)
	}

	dataKind.FromString(*sDataKind)
	if !dataKind.Valid() {
		log.Fatalf("data kind \"%s\" invalid", *sDataKind)
	}
	codeOptions.DataKind = dataKind

	switch codeKind {
	case export.CodeGo:
		if *goPackage == "" {
			log.Fatalf("go package name empty")
		}
	}

	if *codeDir == "" {
		log.Fatalf("output code directory empty")
	}

	if dataKind != export.DataBson && *dataDir == "" {
		log.Fatalf("output data directory empty")
	}

	// 解析
	if *tag != "" {
		s := strings.Split(*tag, "/")
		parseOptions.Tags = make([]gexcels.Tag, len(s))
		for i, v := range s {
			tag := gexcels.Tag(v)
			if !tag.Valid() {
				log.Fatalf("tag \"%s\" invalid", v)
			}
			parseOptions.Tags[i] = gexcels.Tag(v)
		}
	}
	parser, err := parse.Parse(*excelDir, &parseOptions)
	if err != nil {
		log.Fatalf("parse failed: %v", err)
	}

	// 导出代码
	switch codeKind {
	case export.CodeGo:
		if err := code.ExportGo(parser, *codeDir, &codeOptions, &code.GoOptions{PkgName: *goPackage}); err != nil {
			log.Fatalf("export code failed: %v", err)
		}
	}

	// 导出数据
	switch dataKind {
	case export.DataJson:
		if err := data.ExportJson(parser, *dataDir); err != nil {
			log.Fatalf("export json data failed: %v", err)
		}
	case export.DataBytes:
		if err := data.ExportBytes(parser, *dataDir); err != nil {
			log.Fatalf("export bytes data failed: %v", err)
		}
	case export.DataBson:
		if *mongoURI == "" {
			log.Fatal("export bson data, but mongo-uri not specified")
		}
		if *mongoDB == "" {
			log.Fatal("export bson data, but mongo-db not specified")
		}
		opts := options.Client()
		opts.ApplyURI(*mongoURI).SetWriteConcern(writeconcern.Majority())
		mongoCli, err := mongo.Connect(opts)
		if err != nil {
			log.Fatalf("export bson data, context to mongo[%s] failed: %v", *mongoURI, err)
		}
		mongoDB := mongoCli.Database(*mongoDB)
		if err := data.ExportBson(parser, mongoDB); err != nil {
			log.Fatalf("export bson data failed: %v", err)
		}
	}

	log.Println("export completed.")
}
