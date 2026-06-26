package code

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/godyy/gexcels"
	"github.com/godyy/gexcels/export"
	"github.com/godyy/gexcels/internal/log"
	"github.com/godyy/gexcels/internal/utils"
	"github.com/godyy/gexcels/parse"
	pkg_errors "github.com/pkg/errors"
)

// ErrCSharpNamespaceEmpty C# 命名空间为空
var ErrCSharpNamespaceEmpty = errors.New("export code: csharp: namespace empty")

// ErrCSharpDataKindUnsupported C# 导出暂不支持的数据类型
var ErrCSharpDataKindUnsupported = errors.New("export code: csharp: data kind unsupported")

// CSharpOptions C# 代码导出选项
type CSharpOptions struct {
	Namespace       string // 代码命名空间
	TablesClassName string // 静态管理类名称
}

// kind 返回代码导出类型。
func (cp *CSharpOptions) kind() export.CodeKind {
	return export.CodeCSharp
}

// ExportCSharp 将解析出的配置表导出为 C# 代码
func ExportCSharp(p *parse.Parser, path string, options *Options, csharpOptions *CSharpOptions) error {
	return doExport(p, path, options, csharpOptions)
}

// csharpExporter C# 代码导出器。
// 它主要负责两类事情：
// 1. 维护各种导出名称缓存，保证同一解析对象生成稳定名称；
// 2. 给 csharp_template.go 中的模版提供命名、类型和特性等辅助方法。
type csharpExporter struct {
	exporterBase
	kindOptions       *CSharpOptions
	fieldNames        map[string]string
	enumNames         map[string]string
	structNames       map[string]string
	entryClassNames   map[string]string
	tableClassNames   map[string]string
	tablePropertyName map[string]string
}

// createCSharpExporter 创建并校验 C# 导出器。
// 这里会补齐默认选项，并在真正导出前提前发现非法命名。
func createCSharpExporter(p *parse.Parser, path string, options *Options, kindOptions kindOptions) (exporter, error) {
	csharpOptions := kindOptions.(*CSharpOptions)
	if csharpOptions.Namespace == "" {
		return nil, ErrCSharpNamespaceEmpty
	}
	if csharpOptions.TablesClassName == "" {
		csharpOptions.TablesClassName = "Tables"
	}
	if !gexcels.MatchName(csharpOptions.TablesClassName) {
		return nil, fmt.Errorf("export code: csharp: tables class name %s invalid", csharpOptions.TablesClassName)
	}

	return &csharpExporter{
		exporterBase:      newExporterBase(p, path, options),
		kindOptions:       csharpOptions,
		fieldNames:        make(map[string]string),
		enumNames:         make(map[string]string),
		structNames:       make(map[string]string),
		entryClassNames:   make(map[string]string),
		tableClassNames:   make(map[string]string),
		tablePropertyName: make(map[string]string),
	}, nil
}

// kind 返回当前导出器处理的代码类型。
func (e *csharpExporter) kind() export.CodeKind { return export.CodeCSharp }

// export 执行 C# 导出主流程。
// 生成顺序与 Go 版保持一致：枚举、结构体、各配置表、Tables、LoadHelper。
func (e *csharpExporter) export() error {
	if e.options.DataKind != export.DataJson && e.options.DataKind != export.DataBson && e.options.DataKind != export.DataBytes {
		return pkg_errors.WithMessagef(ErrCSharpDataKindUnsupported, "export code: csharp: data kind %s", e.options.DataKind)
	}

	log.Printf("export code csharp to [%s]", e.path)

	if err := e.exportEnumsFile(); err != nil {
		return err
	}
	if err := e.exportStructsFile(); err != nil {
		return err
	}
	if err := e.exportTableFiles(); err != nil {
		return err
	}
	if err := e.exportTablesFile(); err != nil {
		return err
	}
	if err := e.exportLoadHelperFile(); err != nil {
		return err
	}

	return nil
}

// exportEnumsFile 导出枚举定义文件。
func (e *csharpExporter) exportEnumsFile() error {
	if len(e.parser.Enums) == 0 {
		return nil
	}

	filePath := filepath.Join(e.path, e.namespaceFilePrefix()+"_enums.cs")
	if err := os.WriteFile(filePath, []byte(e.GenEnumsFile()), os.ModePerm); err != nil {
		return pkg_errors.WithMessagef(err, "export code: csharp: enums to [%s]", filePath)
	}
	log.PrintfGreen("export code: csharp: enums to [%s]", filePath)
	return nil
}

// exportStructsFile 导出结构体定义文件。
func (e *csharpExporter) exportStructsFile() error {
	if len(e.parser.Structs) == 0 {
		return nil
	}

	filePath := filepath.Join(e.path, e.namespaceFilePrefix()+"_structs.cs")
	if err := os.WriteFile(filePath, []byte(e.GenStructsFile()), os.ModePerm); err != nil {
		return pkg_errors.WithMessagef(err, "export code: csharp: structs to [%s]", filePath)
	}
	log.PrintfGreen("export code: csharp: structs to [%s]", filePath)
	return nil
}

// exportTableFiles 逐表导出配置表文件。
func (e *csharpExporter) exportTableFiles() error {
	for _, td := range e.parser.Tables {
		filePath := filepath.Join(e.path, utils.LowerSnake(td.Name)+".cs")
		content := e.GenTableFile(td)
		if err := os.WriteFile(filePath, []byte(content), os.ModePerm); err != nil {
			return pkg_errors.WithMessagef(err, "export code: csharp: table[%s] to [%s]", td.Name, filePath)
		}
		log.PrintfGreen("export code: csharp: table[%s] to [%s]", td.Name, filePath)
	}
	return nil
}

// exportTablesFile 导出静态表管理类文件。
func (e *csharpExporter) exportTablesFile() error {
	filePath := filepath.Join(e.path, e.namespaceFilePrefix()+"_tables.cs")
	if err := os.WriteFile(filePath, []byte(e.GenTablesFile()), os.ModePerm); err != nil {
		return pkg_errors.WithMessagef(err, "export code: csharp: tables to [%s]", filePath)
	}
	log.PrintfGreen("export code: csharp: tables to [%s]", filePath)
	return nil
}

// exportLoadHelperFile 导出底层加载辅助类文件。
func (e *csharpExporter) exportLoadHelperFile() error {
	filePath := filepath.Join(e.path, e.namespaceFilePrefix()+"_load_helper.cs")
	if err := os.WriteFile(filePath, []byte(e.GenLoadHelperFile()), os.ModePerm); err != nil {
		return pkg_errors.WithMessagef(err, "export code: csharp: load helper to [%s]", filePath)
	}
	log.PrintfGreen("export code: csharp: load helper to [%s]", filePath)
	return nil
}

// namespaceFilePrefix 生成当前命名空间对应的文件名前缀。
func (e *csharpExporter) namespaceFilePrefix() string {
	return utils.LowerSnake(e.kindOptions.Namespace)
}

// GetTablesClassName 返回静态表管理类名。
func (e *csharpExporter) GetTablesClassName() string {
	return e.kindOptions.TablesClassName
}

// needsBsonAnnotations 判断当前导出是否需要生成 BSON 特性。
func (e *csharpExporter) needsBsonAnnotations() bool {
	return e.options.DataKind == export.DataBson
}

// GetFieldName 返回结构体字段的导出属性名，并做结果缓存。
func (e *csharpExporter) GetFieldName(fd *gexcels.Field) string {
	if name, ok := e.fieldNames["field:"+fd.Name]; ok {
		return name
	}
	name := e.GenFieldName(fd.Name)
	e.fieldNames["field:"+fd.Name] = name
	return name
}

// GetEntryFieldName 返回配置表条目字段的导出属性名，并做结果缓存。
func (e *csharpExporter) GetEntryFieldName(fd *gexcels.TableField) string {
	key := "entry:" + fd.Name
	if name, ok := e.fieldNames[key]; ok {
		return name
	}
	name := e.GenFieldName(fd.Name)
	e.fieldNames[key] = name
	return name
}

// GenFieldName 将表字段/结构体字段名转换为 C# 导出属性名。
// 这里对 ID 做特殊处理，保持生成结果符合 C# 常见命名习惯。
func (e *csharpExporter) GenFieldName(name string) string {
	if name == gexcels.TableFieldIDName {
		return "Id"
	}
	return utils.CamelCase(name, true)
}

// GenParamName 将字段名转换为方法参数名。
// 与属性名不同，参数统一使用小驼峰形式。
func (e *csharpExporter) GenParamName(name string) string {
	if name == gexcels.TableFieldIDName {
		return "id"
	}
	return utils.CamelCase(name, false)
}

// GetEnumName 返回枚举的导出类型名，并做结果缓存。
func (e *csharpExporter) GetEnumName(enum *parse.Enum) string {
	if name, ok := e.enumNames[enum.Name]; ok {
		return name
	}
	name := utils.CamelCase(enum.Name, true)
	e.enumNames[enum.Name] = name
	return name
}

// GenEnumItemName 返回枚举项的导出名称。
func (e *csharpExporter) GenEnumItemName(name string) string {
	return utils.CamelCase(name, true)
}

// GetStructName 返回结构体的导出类型名，并做结果缓存。
func (e *csharpExporter) GetStructName(sd *parse.Struct) string {
	if name, ok := e.structNames[sd.Name]; ok {
		return name
	}
	name := utils.CamelCase(sd.Name, true)
	e.structNames[sd.Name] = name
	return name
}

// GetEntryClassName 返回普通表条目类名，并做结果缓存。
func (e *csharpExporter) GetEntryClassName(td *parse.Table) string {
	if name, ok := e.entryClassNames[td.Name]; ok {
		return name
	}
	name := utils.CamelCase(td.Name, true)
	e.entryClassNames[td.Name] = name
	return name
}

// GetTableClassName 返回表管理类名，并追加 Table 后缀以区别条目类型。
func (e *csharpExporter) GetTableClassName(td *parse.Table) string {
	if name, ok := e.tableClassNames[td.Name]; ok {
		return name
	}
	name := utils.CamelCase(td.Name, true) + "Table"
	e.tableClassNames[td.Name] = name
	return name
}

// GetTablePropertyName 返回 Tables 静态管理类中暴露给外部访问的属性名。
func (e *csharpExporter) GetTablePropertyName(td *parse.Table) string {
	if name, ok := e.tablePropertyName[td.Name]; ok {
		return name
	}
	name := utils.CamelCase(td.Name, true)
	e.tablePropertyName[td.Name] = name
	return name
}

// GenTableStorageFieldName 返回 Tables 内部私有静态存储字段名。
// 该字段配合 Volatile.Read/Interlocked.Exchange 完成并发安全发布。
func (e *csharpExporter) GenTableStorageFieldName(td *parse.Table) string {
	return "s" + e.GetTablePropertyName(td)
}

// GenGlobalTypeName 返回带 global:: 前缀的外部类型全名。
func (e *csharpExporter) GenGlobalTypeName(typeName string) string {
	return "global::" + typeName
}

// GenListType 返回使用全限定名称的 List 泛型类型。
func (e *csharpExporter) GenListType(elemType string) string {
	return fmt.Sprintf("%s<%s>", e.GenGlobalTypeName("System.Collections.Generic.List"), elemType)
}

// GenDictionaryType 返回使用全限定名称的 Dictionary 泛型类型。
func (e *csharpExporter) GenDictionaryType(keyType, valueType string) string {
	return fmt.Sprintf("%s<%s, %s>", e.GenGlobalTypeName("System.Collections.Generic.Dictionary"), keyType, valueType)
}

// GenIReadOnlyListType 返回使用全限定名称的 IReadOnlyList 泛型类型。
func (e *csharpExporter) GenIReadOnlyListType(elemType string) string {
	return fmt.Sprintf("%s<%s>", e.GenGlobalTypeName("System.Collections.Generic.IReadOnlyList"), elemType)
}

// GenMongoDatabaseType 返回 mongodb 数据库接口的全限定类型名。
func (e *csharpExporter) GenMongoDatabaseType() string {
	return e.GenGlobalTypeName("MongoDB.Driver.IMongoDatabase")
}

// GenPrimitiveType 将 gexcels 基础字段类型映射为 C# 基础类型名。
func (e *csharpExporter) GenPrimitiveType(ft gexcels.FieldType) string {
	switch ft {
	case gexcels.FTInt32:
		return "int"
	case gexcels.FTInt64:
		return "long"
	case gexcels.FTFloat32:
		return "float"
	case gexcels.FTFloat64:
		return "double"
	case gexcels.FTBool:
		return "bool"
	case gexcels.FTString:
		return "string"
	default:
		return ""
	}
}

// canUseNativeEnum 判断 C# 是否可以直接使用 enum 类型。
// C# 只支持整数底层类型，因此 string 枚举退化为 string 常量集合。
func (e *csharpExporter) canUseNativeEnum(enum *parse.Enum) bool {
	return enum.Type == gexcels.FTInt32
}

// GenTypeInfo 将字段类型信息递归转换为 C# 类型声明。
// 这里会处理 primitive、enum、struct、array、map 等所有导出场景。
func (e *csharpExporter) GenTypeInfo(ti *gexcels.FieldTypeInfo) string {
	switch ti.Type {
	case gexcels.FTInt32, gexcels.FTInt64, gexcels.FTFloat32, gexcels.FTFloat64, gexcels.FTBool, gexcels.FTString:
		return e.GenPrimitiveType(ti.Type)
	case gexcels.FTEnum:
		enum := e.parser.GetEnum(ti.GetName())
		if enum == nil {
			panic(fmt.Sprintf("export code: csharp: enum %s not found", ti.GetName()))
		}
		if e.canUseNativeEnum(enum) {
			return e.GetEnumName(enum)
		}
		return "string"
	case gexcels.FTStruct:
		sd := e.parser.GetStructByName(ti.GetName())
		if sd == nil {
			panic(fmt.Sprintf("export code: csharp: struct %s not found", ti.GetName()))
		}
		return e.GetStructName(sd)
	case gexcels.FTArray:
		return e.GenListType(e.GenTypeInfo(ti.GetElementType()))
	case gexcels.FTMap:
		return e.GenDictionaryType(e.GenMapKeyType(ti.GetMapKeyType()), e.GenTypeInfo(ti.GetMapValueType()))
	default:
		panic(fmt.Sprintf("export code: csharp: field type %d invalid", ti.Type))
	}
}

// GenMapKeyType 返回 map key 的 C# 类型。
// 若 key 是枚举，则退化到底层 primitive，避免把 string 常量枚举用作字典键类型。
func (e *csharpExporter) GenMapKeyType(ti *gexcels.FieldTypeInfo) string {
	if ti.Type == gexcels.FTEnum {
		enum := e.parser.GetEnum(ti.GetName())
		if enum == nil {
			panic(fmt.Sprintf("export code: csharp: enum %s not found", ti.GetName()))
		}
		return e.GenPrimitiveType(enum.Type)
	}
	return e.GenTypeInfo(ti)
}

// GenIndexKeyType 返回索引字段使用的键类型。
// 当前索引只针对 primitive 字段和其底层可比较类型。
func (e *csharpExporter) GenIndexKeyType(fd *gexcels.TableField) string {
	return e.GenPrimitiveType(fd.Type)
}

// GenUniqueIndexFieldName 返回唯一索引字段名。
func (e *csharpExporter) GenUniqueIndexFieldName(fd *gexcels.TableField) string {
	return "by" + e.GetEntryFieldName(fd)
}

// GenUniqueMethodName 返回唯一索引查询方法名。
func (e *csharpExporter) GenUniqueMethodName(fd *gexcels.TableField) string {
	return "By" + e.GetEntryFieldName(fd)
}

// GenCompositeIndexFieldName 返回组合键索引字段名。
func (e *csharpExporter) GenCompositeIndexFieldName(ck *parse.TableCompositeKey) string {
	return "byCompositeKey" + utils.CamelCase(ck.Name, true)
}

// GenCompositeMethodName 返回组合键查询方法名。
func (e *csharpExporter) GenCompositeMethodName(ck *parse.TableCompositeKey) string {
	return "ByCompositeKey" + utils.CamelCase(ck.Name, true)
}

// GenCompositeInitMethodName 返回组合键索引初始化方法名。
func (e *csharpExporter) GenCompositeInitMethodName(ck *parse.TableCompositeKey) string {
	return "InitCompositeKey" + utils.CamelCase(ck.Name, true)
}

// GenGroupIndexFieldName 返回分组索引字段名。
func (e *csharpExporter) GenGroupIndexFieldName(group *parse.TableGroup) string {
	return "byGroup" + utils.CamelCase(group.Name, true)
}

// GenGroupMethodName 返回分组查询方法名。
func (e *csharpExporter) GenGroupMethodName(group *parse.TableGroup) string {
	return "ByGroup" + utils.CamelCase(group.Name, true)
}

// GenGroupInitMethodName 返回分组索引初始化方法名。
func (e *csharpExporter) GenGroupInitMethodName(group *parse.TableGroup) string {
	return "InitGroup" + utils.CamelCase(group.Name, true)
}

// GenCompositeIndexType 递归构造组合键索引的嵌套 Dictionary 类型。
// 末级 value 为对应表的 entry 类型。
func (e *csharpExporter) GenCompositeIndexType(td *parse.Table, fields []string) string {
	valueType := e.GetEntryClassName(td)
	for i := len(fields) - 1; i >= 0; i-- {
		fd := td.GetFieldByName(fields[i])
		valueType = e.GenDictionaryType(e.GenIndexKeyType(fd), valueType)
	}
	return valueType
}

// GenGroupIndexType 递归构造分组索引的嵌套 Dictionary 类型。
// 末级 value 为 entry 列表。
func (e *csharpExporter) GenGroupIndexType(td *parse.Table, fields []string) string {
	valueType := e.GenListType(e.GetEntryClassName(td))
	for i := len(fields) - 1; i >= 0; i-- {
		fd := td.GetFieldByName(fields[i])
		valueType = e.GenDictionaryType(e.GenIndexKeyType(fd), valueType)
	}
	return valueType
}

// GenNormalTableDataFieldName 返回普通表字段实际序列化时使用的数据字段名。
// 只有普通表 ID 字段会在 json/bson 下映射到各自约定的导出字段名。
func (e *csharpExporter) GenNormalTableDataFieldName(fd *gexcels.TableField) string {
	if fd.Col == gexcels.TableColFieldID {
		switch e.options.DataKind {
		case export.DataJson:
			return export.TableFieldIDJsonName
		case export.DataBson:
			return export.TableFieldIDBsonName
		}
	}
	return fd.Name
}

// GenPropertyAttributes 根据导出格式生成属性特性。
// json 使用 JsonPropertyName，bson 使用 BsonId/BsonElement。
func (e *csharpExporter) GenPropertyAttributes(dataName string, isID bool) []string {
	switch e.options.DataKind {
	case export.DataJson:
		return []string{fmt.Sprintf("[%s(%s)]", e.GenGlobalTypeName("System.Text.Json.Serialization.JsonPropertyNameAttribute"), strconv.Quote(dataName))}
	case export.DataBson:
		if isID {
			return []string{fmt.Sprintf("[%s]", e.GenGlobalTypeName("MongoDB.Bson.Serialization.Attributes.BsonIdAttribute"))}
		}
		return []string{fmt.Sprintf("[%s(%s)]", e.GenGlobalTypeName("MongoDB.Bson.Serialization.Attributes.BsonElementAttribute"), strconv.Quote(dataName))}
	default:
		return nil
	}
}

// GenClassAttributes 返回当前导出场景需要附加到类型上的公共特性。
func (e *csharpExporter) GenClassAttributes() []string {
	if e.needsBsonAnnotations() {
		return []string{fmt.Sprintf("[%s]", e.GenGlobalTypeName("MongoDB.Bson.Serialization.Attributes.BsonIgnoreExtraElementsAttribute"))}
	}
	return nil
}

// sanitizeCSharpComment 清理注释文本中的换行，避免把多行描述直接写进单行注释里。
func sanitizeCSharpComment(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	replacer := strings.NewReplacer("\r", " ", "\n", " ")
	return replacer.Replace(s)
}
