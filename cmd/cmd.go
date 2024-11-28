package main

import (
	"flag"
	"github.com/godyy/gexcels"
	"log"
	"strings"
)

var (
	excelDir   = flag.String("excel-dir", "", "excel directory")
	tag        = flag.String("tag", "", "specify tags for filtering fields, e.g. \"c/s\"")
	structFile = flag.String("struct-file", "", "struct file relative to path setting by \"-excel-dir\"")
	sCodeKind  = flag.String("code-kind", "go", "for exporting code, only has \"go\" now")
	sDataKind  = flag.String("data-kind", "json", "for exporting data, has [\"json\", \"bytes\"]")
	codeDir    = flag.String("code-dir", "", "output code directory")
	dataDir    = flag.String("data-dir", "", "output data directory")
	goPackage  = flag.String("go-package", "", "go package name for exporting go code")
)

func main() {
	var (
		codeKind     gexcels.CodeKind
		dataKind     gexcels.DataKind
		parseOptions gexcels.ParseOptions
		codeOptions  gexcels.ExportCodeOptions
	)

	flag.Parse()

	if *excelDir == "" {
		log.Fatalf("excel directory empty")
	}

	codeKind = gexcels.CodeKindFromString(*sCodeKind)
	if codeKind.Valid() {
		log.Fatalf("code kind \"%s\" invalid", *sCodeKind)
	}

	dataKind = gexcels.DataKindFromString(*sDataKind)
	if dataKind.Valid() {
		log.Fatalf("data kind \"%s\" invalid", *sDataKind)
	}

	switch codeKind {
	case gexcels.CodeGo:
		if *goPackage == "" {
			log.Fatalf("go package name empty")
		}
		codeOptions.KindOptions = &gexcels.CodeGoOptions{
			PkgName: *goPackage,
		}
	}

	if *codeDir == "" {
		log.Fatalf("output code directory empty")
	}

	if *dataDir == "" {
		log.Fatalf("output data directory empty")
	}

	// 解析
	parseOptions.StructFilePath = *structFile
	if *tag != "" {
		parseOptions.Tag = strings.Split(*tag, "/")
	}
	parser, err := gexcels.Parse(*excelDir, &parseOptions)
	if err != nil {
		log.Fatalf("parse failed: %v", err)
	}

	// 导出代码
	if err := gexcels.ExportCode(parser, codeKind, *codeDir, &codeOptions); err != nil {
		log.Fatalf("export code failed: %v", err)
	}

	// 导出数据
	if err := gexcels.ExportData(parser, dataKind, *dataDir); err != nil {
		log.Fatalf("export data failed: %v", err)
	}

	log.Println("export completed.")
}
