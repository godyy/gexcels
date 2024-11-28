package gexcels

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	pkg_errors "github.com/pkg/errors"
)

// CodeGoOptions go代码导出选项
type CodeGoOptions struct {
	PkgName string // 代码所在package的名称
}

func (gp *CodeGoOptions) codeKind() CodeKind {
	return CodeGo
}

// goExporter go代码导出器
type goExporter struct{}

func (ge *goExporter) codeKind() CodeKind {
	return CodeGo
}

func (ge *goExporter) checkOptions(options CodeKindOptions) error {
	goOptions := convertCodeKindOptions[*CodeGoOptions](options)
	if goOptions == nil {
		return errNonCodeKindOptions
	}
	if goOptions.PkgName == "" {
		return errGoPkgEmpty
	}
	return nil
}

func (ge *goExporter) export(p *Parser, path string, options *ExportCodeOptions) error {
	printf("export code go to [%s]", path)

	kindOptions := convertCodeKindOptions[*CodeGoOptions](options.KindOptions)

	if err := ge.exportStructsFile(p.structs, path, options, kindOptions); err != nil {
		return err
	}

	if err := ge.exportTableFiles(p.tables, path, options, kindOptions); err != nil {
		return err
	}

	if err := ge.exportTableMgrFile(p.tables, path, options, kindOptions); err != nil {
		return err
	}

	if err := ge.exportLoadFile(p, path, options, kindOptions); err != nil {
		return err
	}

	_ = exec.Command("go", "fmt", path).Run()

	return nil
}

// 将结构体定义导出为go代码文件
func (ge *goExporter) exportStructsFile(structs []*Struct, path string, options *ExportCodeOptions, kindOptions *CodeGoOptions) error {
	if len(structs) == 0 {
		return nil
	}

	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("package %s\n", kindOptions.PkgName))

	for _, sd := range structs {
		sb.WriteString("\n")
		sb.WriteString(fmt.Sprintf("// %s %s\n", sd.ExportName, sd.Desc))
		sb.WriteString(genGoStruct(sd, options.DataKind))
		sb.WriteString("\n")
	}

	filePath := filepath.Join(path, options.StructFileName+".go")
	if err := os.WriteFile(filePath, ([]byte)(sb.String()), os.ModePerm); err != nil {
		return pkg_errors.WithMessagef(err, "export go: structs to [%s]", filePath)
	}

	printfGreen("export go: structs to [%s]", filePath)
	return nil
}

// 将所有配置表定义导出为go代码文件
func (ge *goExporter) exportTableFiles(tds []*Table, path string, options *ExportCodeOptions, kindOptions *CodeGoOptions) error {
	for _, table := range tds {
		if err := ge.exportTableFile(table, path, options, kindOptions); err != nil {
			return err
		}
	}
	return nil
}

// 将配置表定义导出为go代码文件
func (ge *goExporter) exportTableFile(td *Table, path string, options *ExportCodeOptions, kindOptions *CodeGoOptions) error {
	if !options.isTableNameOccupied(td.ExportTableName) {
		return fmt.Errorf("export go: table[%s], table name [%s] occupied", td.Name, td.ExportTableName)
	}
	content := ""
	if td.IsGlobal {
		content = genGoGlobalTableFile(td, kindOptions.PkgName, options.DataKind)
	} else {
		content = genGoNormalTableFile(td, kindOptions.PkgName, options.DataKind)
	}
	fileName := td.ExportTableName + ".go"
	filePath := filepath.Join(path, fileName)
	if err := os.WriteFile(filePath, ([]byte)(content), os.ModePerm); err != nil {
		return pkg_errors.WithMessagef(err, "export go: table[%s] to [%s]", td.Name, filePath)
	}
	printfGreen("export go: table[%s] to [%s]", td.Name, filePath)
	return nil
}

// 将配置表管理器导出为go代码
func (ge *goExporter) exportTableMgrFile(tds []*Table, path string, options *ExportCodeOptions, kindOptions *CodeGoOptions) error {
	content := genGoTableMgrFile(tds, kindOptions.PkgName, options.DataKind)
	filePath := filepath.Join(path, options.TableMgrFileName+".go")
	if err := os.WriteFile(filePath, ([]byte)(content), os.ModePerm); err != nil {
		return pkg_errors.WithMessagef(err, "export go: table_manager to [%s]", filePath)
	}
	printfGreen("export go: table_manager to [%s]", filePath)
	return nil
}

func (ge *goExporter) exportLoadFile(p *Parser, path string, options *ExportCodeOptions, kindOptions *CodeGoOptions) error {
	switch options.DataKind {
	case DataJson:
		if err := ge.exportLoadJsonFile(p.tables, path, kindOptions.PkgName); err != nil {
			return err
		}
	case DataBytes:
		if err := ge.exportLoadBytesFile(p.tables, path, kindOptions.PkgName); err != nil {
			return err
		}
	default:
		return errDataKindInvalid(options.DataKind)
	}
	return nil
}

func (ge *goExporter) exportLoadJsonFile(tds []*Table, path, pkgName string) error {
	content := genGoLoadJsonFile(tds, pkgName)
	filePath := filepath.Join(path, "load_json.go")
	if err := os.WriteFile(filePath, ([]byte)(content), os.ModePerm); err != nil {
		return pkg_errors.WithMessagef(err, "export go: load_json to [%s]", filePath)
	}
	printfGreen("export go: load_json to [%s]", filePath)
	return nil
}

func (ge *goExporter) exportLoadBytesFile(tds []*Table, path, pkgName string) error {
	content := genGoLoadBytesFile(tds, pkgName)
	filePath := filepath.Join(path, "load_bytes.go")
	if err := os.WriteFile(filePath, ([]byte)(content), os.ModePerm); err != nil {
		return pkg_errors.WithMessagef(err, "export go: load_bytes to [%s]", filePath)
	}
	printfGreen("export go: load_bytes to [%s]", filePath)
	return nil
}

// 生成go primitive类型
func genGoPrimitiveType(t FieldType) string {
	switch t {
	case FTInt32:
		return "int32"
	case FTInt64:
		return "int64"
	case FTFloat32:
		return "float32"
	case FTFloat64:
		return "float64"
	case FTBool:
		return "bool"
	case FTString:
		return "string"
	default:
		return ""
	}
}

func genGoStructFieldType(fd *Field) string {
	if fd.Type.Primitive() {
		return genGoPrimitiveType(fd.Type)
	} else if fd.Type == FTStruct {
		return "*" + exportStructName(fd.StructName)
	} else if fd.Type == FTArray {
		if fd.ElementType.Primitive() {
			return "[]" + genGoPrimitiveType(fd.ElementType)
		} else if fd.ElementType == FTStruct {
			return "[]*" + exportStructName(fd.StructName)
		} else {
			panic(errArrayElementInvalid(fd.ElementType))
		}
	} else {
		panic(errFieldTypeInvalid(fd.Type))
	}
}

func genGoStructFieldTag(fieldName string, dataKind DataKind) string {
	switch dataKind {
	case DataJson:
		return "`json:\"" + fieldName + ",omitempty\"`"
	default:
		return ""
	}
}

func genGoStructFieldJsonTag(fieldName string) string {
	return "`json:\"" + fieldName + ",omitempty\"`"
}

func genGoStructField(fd *Field, dataKind DataKind) string {
	var sb strings.Builder
	sb.WriteString(fd.ExportName)
	sb.WriteString(" ")
	sb.WriteString(genGoStructFieldType(fd))
	sb.WriteString(" ")
	if fieldTag := genGoStructFieldTag(fd.Name, dataKind); fieldTag != "" {
		sb.WriteString(fieldTag)
	}
	sb.WriteString(" // " + fd.Desc)
	return sb.String()
}
