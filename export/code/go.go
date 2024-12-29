package code

import (
	"errors"
	"fmt"
	"github.com/godyy/gexcels"
	"github.com/godyy/gexcels/export"
	"github.com/godyy/gexcels/internal/log"
	"github.com/godyy/gexcels/internal/utils"
	"github.com/godyy/gexcels/parse"
	pkg_errors "github.com/pkg/errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ErrGoPkgEmpty 空go包名错误
var ErrGoPkgEmpty = errors.New("export code: go: package empty")

// GoOptions go代码导出选项
type GoOptions struct {
	PkgName string // 代码所在package的名称
}

func (gp *GoOptions) kind() export.CodeKind {
	return export.CodeGo
}

// ExportGo 将解析出的配置表导出为go代码
func ExportGo(p *parse.Parser, path string, options *Options, goOptions *GoOptions) error {
	return doExport(p, path, options, goOptions)
}

// goExporter go代码导出器
type goExporter struct {
	exporterBase                       // base
	kindOptions      *GoOptions        // 分类选项
	fieldNames       map[string]string // 导出的字段名
	fieldTypes       map[string]string // 导出的字段类型
	structNames      map[string]string // 导出的结构体名
	tableNames       map[string]string // 导出的表名
	tableStructNames map[string]string // 导出的表结构体名
	entryStructNames map[string]string // 导出的条目结构体名
}

func createGoExporter(p *parse.Parser, path string, options *Options, kindOptions kindOptions) (exporter, error) {
	goOptions := kindOptions.(*GoOptions)
	if goOptions.PkgName == "" {
		return nil, ErrGoPkgEmpty
	}
	return &goExporter{
		exporterBase:     newExporterBase(p, path, options),
		kindOptions:      goOptions,
		fieldNames:       make(map[string]string),
		fieldTypes:       make(map[string]string),
		structNames:      make(map[string]string),
		tableNames:       make(map[string]string),
		tableStructNames: make(map[string]string),
		entryStructNames: make(map[string]string),
	}, nil
}

func (e *goExporter) kind() export.CodeKind { return export.CodeGo }

func (e *goExporter) export() error {
	log.Printf("export code go to [%s]", e.path)

	if err := e.exportStructsFile(); err != nil {
		return err
	}

	if err := e.exportTableFiles(); err != nil {
		return err
	}

	if err := e.exportTableMgrFile(); err != nil {
		return err
	}

	if err := e.exportLoadHelperFile(); err != nil {
		return err
	}

	_ = exec.Command("go", "fmt", e.path).Run()

	return nil
}

// exportStructsFile 将结构体定义导出为go代码文件
func (e *goExporter) exportStructsFile() error {
	structs := e.parser.Structs
	if len(structs) == 0 {
		return nil
	}

	content := e.GenStructsFile()
	filePath := filepath.Join(e.path, e.kindOptions.PkgName+"_structs_.go")
	if err := os.WriteFile(filePath, ([]byte)(content), os.ModePerm); err != nil {
		return pkg_errors.WithMessagef(err, "export code: go: structs to [%s]", filePath)
	}

	log.PrintfGreen("export code: go: structs to [%s]", filePath)
	return nil
}

// exportTableFiles 将所有配置表定义导出为go代码文件
func (e *goExporter) exportTableFiles() error {
	for _, table := range e.parser.Tables {
		if err := e.exportTableFile(table); err != nil {
			return err
		}
		e.resetAfterExportTable()
	}
	return nil
}

// resetAfterExportTable 导出配置表后重置数据
func (e *goExporter) resetAfterExportTable() {
	e.fieldNames = make(map[string]string)
	e.fieldTypes = make(map[string]string)
}

// 将配置表定义导出为go代码文件
func (e *goExporter) exportTableFile(td *parse.Table) error {
	tableName := e.GetTableName(td)
	content := ""
	if td.IsGlobal {
		content = e.GenGlobalTableFile(td)
	} else {
		content = e.GenNormalTableFile(td)
	}
	fileName := tableName + ".go"
	filePath := filepath.Join(e.path, fileName)
	if err := os.WriteFile(filePath, ([]byte)(content), os.ModePerm); err != nil {
		return pkg_errors.WithMessagef(err, "export code: go: table[%s] to [%s]", td.Name, filePath)
	}
	log.PrintfGreen("export code: go: table[%s] to [%s]", td.Name, filePath)
	return nil
}

// 将配置表管理器导出为go代码
func (e *goExporter) exportTableMgrFile() error {
	content := e.GenTableMgrFile()
	filePath := filepath.Join(e.path, e.kindOptions.PkgName+"_manager_.go")
	if err := os.WriteFile(filePath, ([]byte)(content), os.ModePerm); err != nil {
		return pkg_errors.WithMessagef(err, "export code: go: table_manager to [%s]", filePath)
	}
	log.PrintfGreen("export code: go: table_manager to [%s]", filePath)
	return nil
}

// exportLoadHelperFile 导出加载帮助文件
func (e *goExporter) exportLoadHelperFile() error {
	switch e.options.DataKind {
	case export.DataJson:
		if err := e.exportJsonLoadHelperFile(); err != nil {
			return err
		}
	case export.DataBytes:
		if err := e.exportBytesLoadHelperFile(); err != nil {
			return err
		}
	default:
		panic(fmt.Sprintf("export code: go: load helper file: data-kind %d invalid", e.options.DataKind))
	}
	return nil
}

// exportJsonLoadHelperFile 导出json加载帮助文件
func (e *goExporter) exportJsonLoadHelperFile() error {
	content := e.GenJsonLoadHelperFile()
	filePath := filepath.Join(e.path, e.kindOptions.PkgName+"_load_.go")
	if err := os.WriteFile(filePath, ([]byte)(content), os.ModePerm); err != nil {
		return pkg_errors.WithMessagef(err, "export code: go: json_load_helper to [%s]", filePath)
	}
	log.PrintfGreen("export code: go: json_load_helper to [%s]", filePath)
	return nil
}

// exportBytesLoadHelperFile 导出bytes加载帮助文件
func (e *goExporter) exportBytesLoadHelperFile() error {
	content := e.GenBytesLoadHelperFile()
	filePath := filepath.Join(e.path, e.kindOptions.PkgName+"_load_.go")
	if err := os.WriteFile(filePath, ([]byte)(content), os.ModePerm); err != nil {
		return pkg_errors.WithMessagef(err, "export code: go: bytes_load_helper to [%s]", filePath)
	}
	log.PrintfGreen("export code: go: bytes_load_helper to [%s]", filePath)
	return nil
}

// GetFieldName 获取字段名称
func (e *goExporter) GetFieldName(fd *gexcels.Field) string {
	if s, ok := e.fieldNames[fd.Name]; ok {
		return s
	} else {
		s = e.GenFieldName(fd)
		e.fieldNames[fd.Name] = s
		return s
	}
}

// GetFieldType 获取字段类型
func (e *goExporter) GetFieldType(fd *gexcels.Field) string {
	if s, ok := e.fieldTypes[fd.Name+":"+fd.FieldTypeInfo.String()]; ok {
		return s
	} else {
		s = e.GenFieldType(fd)
		e.fieldTypes[fd.Name+":"+fd.FieldTypeInfo.String()] = s
		return s
	}
}

// GetStructName 获取结构体名称
func (e *goExporter) GetStructName(sd *parse.Struct) string {
	if s, ok := e.structNames[sd.Name]; ok {
		return s
	} else {
		s = e.GenStructName(sd)
		e.structNames[sd.Name] = s
		return s
	}
}

// GetTableName 获取表名称
func (e *goExporter) GetTableName(td *parse.Table) string {
	if s, ok := e.tableNames[td.Name]; ok {
		return s
	} else {
		s = e.GenTableName(td)
		e.tableNames[td.Name] = s
		return s
	}
}

// GetTableStructName 获取表结构体名称
func (e *goExporter) GetTableStructName(td *parse.Table) string {
	if s, ok := e.tableStructNames[td.Name]; ok {
		return s
	} else {
		s = e.GenTableStructName(td)
		e.tableStructNames[td.Name] = s
		return s
	}
}

// GetEntryFieldName 获取条目字段名称
func (e *goExporter) GetEntryFieldName(fd *gexcels.TableField) string {
	key := fd.Name
	if fd.Col == gexcels.TableColFieldID {
		key = "table:" + key
	}
	if s, ok := e.fieldNames[key]; ok {
		return s
	} else {
		s = e.GenEntryFieldName(fd)
		e.fieldNames[key] = s
		return s
	}
}

// GetEntryFieldType 获取条目字段类型
func (e *goExporter) GetEntryFieldType(fd *gexcels.TableField) string {
	return e.GetFieldType(fd.Field)
}

// GetEntryStructName 获取条目结构体名称
func (e *goExporter) GetEntryStructName(td *parse.Table) string {
	if s, ok := e.entryStructNames[td.Name]; ok {
		return s
	} else {
		s = e.GenEntryStructName(td)
		e.entryStructNames[td.Name] = s
		return s
	}
}

// GetTableUniqueKeyFieldName 获取表唯一键字段名称
func (e *goExporter) GetTableUniqueKeyFieldName(fd *gexcels.TableField) string {
	key := gexcels.FRNUnique + ":" + fd.Name
	if s, ok := e.fieldNames[key]; ok {
		return s
	} else {
		s = e.GenTableUniqueKeyFieldName(fd)
		e.fieldNames[key] = s
		return s
	}
}

// GetTableCompositeKeyFieldName 获取表组合键字段名称
func (e *goExporter) GetTableCompositeKeyFieldName(ck *parse.TableCompositeKey) string {
	key := gexcels.FRNCompositeKey + ":" + ck.Name
	if s, ok := e.fieldNames[key]; ok {
		return s
	} else {
		s = e.GenTableCompositeKeyFieldName(ck)
		e.fieldNames[key] = s
		return s
	}
}

// GetTableCompositeKeyFieldType 获取表组合键字段类型
func (e *goExporter) GetTableCompositeKeyFieldType(td *parse.Table, ck *parse.TableCompositeKey) string {
	key := gexcels.FRNCompositeKey + ":" + ck.Name
	if s, ok := e.fieldTypes[key]; ok {
		return s
	} else {
		s = e.GenTableCompositeKeyFieldType(td, ck.FieldNames())
		e.fieldTypes[key] = s
		return s
	}
}

// GetTableGroupFieldName 获取表分组字段名称
func (e *goExporter) GetTableGroupFieldName(group *parse.TableGroup) string {
	key := gexcels.FRNGroup + ":" + group.Name
	if s, ok := e.fieldNames[key]; ok {
		return s
	} else {
		s = e.GenTableGroupFieldName(group)
		e.fieldNames[key] = s
		return s
	}
}

// GetTableGroupFieldType 获取表分组字段类型
func (e *goExporter) GetTableGroupFieldType(td *parse.Table, group *parse.TableGroup) string {
	key := gexcels.FRNGroup + ":" + group.Name
	if s, ok := e.fieldTypes[key]; ok {
		return s
	} else {
		s = e.GenTableGroupFieldType(td, group.FieldNames())
		e.fieldTypes[key] = s
		return s
	}
}

// GenFieldName 生成字段名称
func (e *goExporter) GenFieldName(fd *gexcels.Field) string {
	return utils.CamelCase(fd.Name, true)
}

// GenStructName 生成结构体名称
func (e *goExporter) GenStructName(sd *parse.Struct) string {
	return utils.CamelCase(sd.Name, true)
}

// GenTableName 生成表名称
func (e *goExporter) GenTableName(td *parse.Table) string {
	return td.Name
}

// GenTableStructName 生成表结构体名称
func (e *goExporter) GenTableStructName(td *parse.Table) string {
	if td.IsGlobal {
		return utils.CamelCase(td.Name, true)
	} else {
		return "Table" + utils.CamelCase(td.Name, true)
	}
}

// entryFieldIDExportedName 条目ID字段导出名
const entryFieldIDExportedName = "ID"

// GenEntryFieldName 生成条目字段名称
func (e *goExporter) GenEntryFieldName(fd *gexcels.TableField) string {
	if fd.Col == gexcels.TableColFieldID {
		return entryFieldIDExportedName
	}
	return e.GenFieldName(fd.Field)
}

// GenEntryStructName 生成条目结构体名称
func (e *goExporter) GenEntryStructName(td *parse.Table) string {
	return utils.CamelCase(td.Name, true)
}

// GenPrimitiveFieldType 生成primitive字段类型
func (e *goExporter) GenPrimitiveFieldType(t gexcels.FieldType) string {
	switch t {
	case gexcels.FTInt32:
		return "int32"
	case gexcels.FTInt64:
		return "int64"
	case gexcels.FTFloat32:
		return "float32"
	case gexcels.FTFloat64:
		return "float64"
	case gexcels.FTBool:
		return "bool"
	case gexcels.FTString:
		return "string"
	default:
		return ""
	}
}

// GenFieldType 生成字段类型
func (e *goExporter) GenFieldType(fd *gexcels.Field) string {
	if fd.Type.Primitive() {
		return e.GenPrimitiveFieldType(fd.Type)
	} else if fd.Type == gexcels.FTStruct {
		sd := e.parser.GetStructByName(fd.StructName)
		return "*" + e.GetStructName(sd)
	} else if fd.Type == gexcels.FTArray {
		if fd.ElementType.Primitive() {
			return "[]" + e.GenPrimitiveFieldType(fd.ElementType)
		} else if fd.ElementType == gexcels.FTStruct {
			sd := e.parser.GetStructByName(fd.StructName)
			return "[]*" + e.GetStructName(sd)
		} else {
			panic(fmt.Sprintf("export code: go: GenFieldType: array element type %d invalid", fd.ElementType))
		}
	} else {
		panic(fmt.Sprintf("export code: go: GenFieldType: field type %d invalid", fd.Type))
	}
}

// GenStructFieldTag 生成结构体字段标签
func (e *goExporter) GenStructFieldTag(fd *gexcels.Field) string {
	switch e.options.DataKind {
	case export.DataJson:
		return "`json:\"" + fd.Name + ",omitempty\"`"
	default:
		return ""
	}
}

// GenEntryFieldTag 生成条目字段标签
func (e *goExporter) GenEntryFieldTag(fd *gexcels.TableField) string {
	switch e.options.DataKind {
	case export.DataJson:
		name := fd.Name
		if fd.Col == gexcels.TableColFieldID {
			name = gexcels.TableFieldIDJsonTagName
		}
		return "`json:\"" + name + ",omitempty\"`"
	default:
		return ""
	}
}

// GenStructField 生成结构体字段
func (e *goExporter) GenStructField(fd *gexcels.Field) string {
	var sb strings.Builder
	sb.WriteString(e.GetFieldName(fd))
	sb.WriteString(" ")
	sb.WriteString(e.GetFieldType(fd))
	sb.WriteString(" ")
	if fieldTag := e.GenStructFieldTag(fd); fieldTag != "" {
		sb.WriteString(fieldTag)
	}
	sb.WriteString(" // " + fd.Desc)
	return sb.String()
}

// GenTableStructField 生成表结构体字段
func (e *goExporter) GenTableStructField(fd *gexcels.TableField) string {
	return e.GenStructField(fd.Field)
}

// GenTableUniqueKeyFieldName 生成表唯一键字段名称
func (e *goExporter) GenTableUniqueKeyFieldName(fd *gexcels.TableField) string {
	return "by" + e.GetEntryFieldName(fd)
}

// GenTableUniqueKeyMethodName 生成表唯一键方法名称
func (e *goExporter) GenTableUniqueKeyMethodName(fd *gexcels.TableField) string {
	return "By" + e.GenEntryFieldName(fd)
}

// GenTableCompositeKeyFieldName 生成表组合键字段名称
func (e *goExporter) GenTableCompositeKeyFieldName(ck *parse.TableCompositeKey) string {
	var sb strings.Builder
	sb.WriteString("by" + gexcels.FRNCompositeKey)
	sb.WriteString(utils.CamelCase(ck.Name, true))
	return sb.String()
}

// GenTableCompositeKeyFieldType 生成表组合键字段类型
func (e *goExporter) GenTableCompositeKeyFieldType(td *parse.Table, fields []string) string {
	var sb strings.Builder
	for _, fieldName := range fields {
		field := td.GetFieldByName(fieldName)
		sb.WriteString(fmt.Sprintf("map[%s]", e.GetFieldType(field.Field)))
	}
	sb.WriteString("*" + e.GetEntryStructName(td))
	return sb.String()
}

// GenTableCompositeKeyMethodName 生成表组合键方法名称
func (e *goExporter) GenTableCompositeKeyMethodName(ck *parse.TableCompositeKey) string {
	return utils.CamelCase(e.GetTableCompositeKeyFieldName(ck), true)
}

// GenTableCompositeKeyInitMethodName 生成表组合键初始化方法名称
func (e *goExporter) GenTableCompositeKeyInitMethodName(ck *parse.TableCompositeKey) string {
	return "init" + utils.CamelCase(e.GetTableCompositeKeyFieldName(ck), true)
}

// GenTableGroupFieldName 生成表分组字段名称
func (e *goExporter) GenTableGroupFieldName(group *parse.TableGroup) string {
	var sb strings.Builder
	sb.WriteString("byGroup")
	sb.WriteString(utils.CamelCase(group.Name, true))
	return sb.String()
}

// GenTableGroupFieldType 生成表分组字段类型
func (e *goExporter) GenTableGroupFieldType(td *parse.Table, fields []string) string {
	var sb strings.Builder
	for _, fieldName := range fields {
		field := td.GetFieldByName(fieldName)
		sb.WriteString(fmt.Sprintf("map[%s]", e.GetFieldType(field.Field)))
	}
	sb.WriteString("[]*" + e.GetEntryStructName(td))
	return sb.String()
}

// GenTableGroupMethodName 生成表分组方法名称
func (e *goExporter) GenTableGroupMethodName(group *parse.TableGroup) string {
	return utils.CamelCase(e.GetTableGroupFieldName(group), true)
}

// GenTableGroupInitMethodName 生成表分组初始化方法名称
func (e *goExporter) GenTableGroupInitMethodName(group *parse.TableGroup) string {
	return "init" + utils.CamelCase(e.GetTableGroupFieldName(group), true)
}

// GenTableLoadDataMethod 生成表加载数据方法
func (e *goExporter) GenTableLoadDataMethod(td *parse.Table) string {
	switch e.options.DataKind {
	case export.DataJson:
		if td.IsGlobal {
			return e.GenGlobalTableLoadJson(td)
		} else {
			return e.GenNormalTableLoadJson(td)
		}
	case export.DataBytes:
		if td.IsGlobal {
			return e.GenGlobalTableLoadBytes(td)
		} else {
			return e.GenNormalTableLoadBytes(td)
		}
	default:
		panic(fmt.Sprintf("export code: go: generate table load data method: data-kind %d invalid", e.options.DataKind))
	}
}

// GenMgrTableFieldName 生成管理器表字段名称
func (e *goExporter) GenMgrTableFieldName(td *parse.Table) string {
	return utils.CamelCase(td.Name)
}

// GenMgrTableMethodName 生成管理器表方法名称
func (e *goExporter) GenMgrTableMethodName(td *parse.Table) string {
	return utils.CamelCase(td.Name, true)
}
