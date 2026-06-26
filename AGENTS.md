# gexcels 开发/协作指南（给后续智能体/开发者）

本仓库是一个 Go 编写的 Excel 配置表解析与代码/数据导出工具：
- 解析：读取 `.xlsx`，抽取枚举、结构体与普通/全局配置表，做类型/规则校验，并在表间做 LINK 关联检查。
- 导出：基于解析结果导出 Go/C# 代码，以及 JSON/Bytes/BSON 数据。

## 目录与入口

- `cmd/`
  - `cmd/cmd.go`：CLI 入口（参数解析、调用 parse + export）。
  - `cmd/Makefile`：交叉编译示例。
- `parse/`：解析核心逻辑（强依赖解析顺序：Enum -> Struct -> Table）。
- `export/`
  - `export/code/`：导出代码（Go / C#）。
  - `export/data/`：导出数据（json/bytes/bson）。
- 根目录（`package gexcels`）：领域模型与常量（字段类型、规则、表头布局等）。
- `internal/test/excels/`：测试用样例 xlsx。

Go 版本与依赖见 `go.mod`（Go `1.24`；xlsx 解析使用 `github.com/tealeg/xlsx/v3`；错误包装使用 `github.com/pkg/errors`）。

## 快速命令

- 运行测试：
  - `go test ./...`
- 构建 CLI（仓库根目录执行）：
  - `go build ./cmd`
- 交叉编译（在 `cmd/` 目录执行）：
  - `make build` 或 `make all`

## 代码导出约定

- `export.CodeKind` 现支持 `go`、`csharp`，并兼容 `c#`、`cs` 两个别名。
- CLI 入口在 `cmd/cmd.go`：
  - Go 导出：`-code-kind go -go-package <pkg>`
  - C# 导出：`-code-kind csharp -csharp-namespace <namespace> [-csharp-tables-class Tables]`
- `-csharp-namespace` 是必填项；`-csharp-tables-class` 默认为 `Tables`，并且类名必须满足 `gexcels.MatchName`。
- 新增代码导出类型时，至少需要同步修改：
  - `export/code_kind.go`：注册 kind 与字符串映射
  - `export/code/export.go`：注册导出器构造函数
  - `cmd/cmd.go`：补充 CLI 参数校验与分发
  - `export/code/export_test.go`：补充对应导出回归测试

## Excel 约定（非常关键）

### 文件名分类（默认扫描）

`parse.Parse(excelDir, ...)` 会递归扫描目录内的 `.xlsx`：
- 枚举文件：文件名（不含扩展名）匹配 `^(?:[^.]*\.)?enum$`
  - 例如：`enum.xlsx`、`core.enum.xlsx`
- 结构体文件：文件名（不含扩展名）匹配 `^(?:[^.]*\.)?struct(?:\.([0-9]*))?$`
  - 例如：`struct.xlsx`、`core.struct.0.xlsx`（数字用于优先级排序）
- 其它 `.xlsx`：当作“普通配置表文件”

如果通过 `parse.Options` 显式指定 `EnumFiles/StructFiles`，则会禁用默认扫描逻辑并按给定顺序解析。

### Sheet 命名分类

Sheet 名通过正则分类（关键点是以 `|` 分隔“描述”和“类型/表名”）：
- 枚举 Sheet：`^(.*)\|(Enum\w*)`
- 结构体 Sheet：`^(.*)\|(Struct\w*)(?:\.([0-9]*))?`
  - `.数字` 是结构体 Sheet 级别的优先级（越小越先解析）
- 配置表 Sheet：`^(.*)\|([A-Za-z]\w*[A-Za-z0-9])$`
  - 若表名以 `Global` 开头，则按“全局表”解析，否则按“普通表”解析

### 注释行

若行第 0 列以 `#` 开头，该行会被跳过（适用于结构体表、普通表、全局表的数据区域）。

### Tag 过滤

解析器支持 tag 过滤（用于枚举/字段/结构体定义行过滤）：
- CLI：`-tag "c/s"`，以 `/` 分隔多个 tag。
- `parse.Options.Tags` 为空时默认只匹配空 tag。
- 若包含 `*`（`gexcels.TagAny`），则匹配任意 tag（包括空）。
- 表内 tag 格式：`\w*(?:/\w*)*`（空串也合法）。

### 枚举表布局

枚举解析位于 `parse/enum.go`，基础列定义在 `gexcels/enum.go`：
- 从第 1 行开始逐行解析（第 0 行通常是表头；见 `gexcels.EnumRowFirstEntry`）。
- 起始行：当第 1 列出现 `Name_BEGIN`（`([a-zA-Z][a-zA-Z0-9_]*)_BEGIN`）时开始一个枚举定义。
- 列：
  - 0：Tag（枚举定义的 tag；不匹配则跳过该枚举整段定义）
  - 1：BEGIN / ItemName
  - 2：Type / ItemValue
  - 3：Desc
- 枚举类型：只允许 `int32` 或 `string`（见 `gexcels.CheckEnumType`）。

### 结构体表布局

结构体解析位于 `parse/struct.go`，列定义在 `gexcels/struct.go`：
- 从第 1 行开始逐行解析（第 0 行通常是表头）。
- 列：
  - 0：Tag
  - 1：Name
  - 2：Fields（形如 `a:int32:"desc",b:[]Foo`）
  - 3：Rule（目前仅支持 `LINK=...`；分隔符见 `parse.Options.FieldRuleSep`，默认 `|`）
  - 4：Desc

### 普通配置表布局

普通表解析位于 `parse/table.go`，表头定义在 `gexcels/table.go`：
- 行：
  - 0：字段名
  - 1：字段描述
  - 2：字段类型
  - 3：字段规则
  - 4：字段 tag
  - 5+：数据行
- 强约束：
  - 第 0 列必须为字段名 `ID`（`gexcels.TableFieldIDName`）
  - `ID` 必须是 primitive 类型（int32/int64/float32/float64/bool/string）
  - `ID` 自动附加 `UNIQUE` 规则

### 全局配置表布局

全局表的表名以 `Global` 开头（见 `gexcels.GlobalTableNamePrefix`），解析位于 `parse/table.go`：
- 从第 1 行开始逐行解析（第 0 行通常是表头）。
- 列定义（见 `gexcels/table.go`）：
  - 0：Tag
  - 1：Name
  - 2：Type
  - 3：Value
  - 4：Rule
  - 5：Desc（可选，但建议提供）

## 解析流水线（不要破坏顺序）

核心流程在 `parse/parse.go`：
1. `searchFiles()`：按文件名规则搜集 enum/struct/table 文件。
2. `parseEnums()`：先解析枚举，并注册为自定义字段类型（供后续字段类型解析使用）。
3. `parseStructs()`：再解析结构体，并注册为自定义字段类型。
4. `parseTables()`：最后解析普通/全局表（字段类型会引用 enum/struct）。
5. `checkLinksBetweenTable()`：对 `LINK`（包括结构体嵌套字段上的 LINK）做跨表值校验。

注意：字段类型解析在 `parse/field.go` 的 `parseFieldTypeInfo()`，遇到非 primitive 类型会去查已注册的“自定义字段类型”（枚举/结构体）。因此顺序错误会导致 “custom field type not found”。

## 字段规则（Rule）语法

规则解析定义在 `gexcels/field_rule.go`，主要规则：
- `UNIQUE`
- `LINK=...`
- `CKEY=KeyName,Index`（组合键）
- `GROUP=GroupName,Index`（分组）

配置表中同一字段可通过 `|`（可配置）写多个规则；结构体规则目前仅支持 `LINK`（格式见下文；实现见 `parse/struct.go`）。

### LINK 规则（支持嵌套/多维数组/map key 分层）

LINK 用于做跨表值校验：源字段（或其嵌套容器中的叶子 primitive/enum）必须能在目标表目标字段中找到匹配值。

**表字段 LINK（普通表/全局表字段规则中使用）**
- 格式：`LINK=[DstTable.DstField][,k1:KeyTable1.KeyField][,k2:KeyTable2.KeyField]...`
- 示例（仅 value link）：`LINK=Item.ID`
- 示例（仅 key link）：`LINK=k1:Key1.ID,k2:Key2.ID`
- 示例（混合）：`LINK=Item.ID,k1:Key1.ID,k2:Key2.ID`

说明：
- value link 会沿类型递归取“值叶子”：`array -> elem`、`map -> value`、`enum -> underlying primitive`、`struct -> 必须在结构体字段上定义 LINK`。
- key link 用 `kN:<Table.Field>` 指定要校验第 N 层 map 的 key（N 从 1 开始）。
- map 层级编号只与 map 出现顺序有关：数组嵌套不计数。例：`[]map[int32][]map[int32]int32` 里只有两层 map，因此可用 `k1`、`k2`。
- map key 允许的类型：`int32/int64/string/enum`（enum 会按其底层 primitive 参与校验）。

**结构体字段 LINK（在结构体 Rule 列中使用）**
- 基本格式：`LINK=<LocalField>,<LinkValue>`
- 其中 `<LinkValue>` 与表字段 LINK 的 value 完全一致（同一套语法），例如：
  - `LINK=FooID,Item.ID`
  - `LINK=MM,k1:Key1.ID,k2:Key2.ID`

重要约束：
- 结构体类型的 LINK 必须在结构体字段自身声明；不支持在上层表字段通过“路径”指定结构体内部字段的 LINK。

## 导出行为概览

CLI 主流程（`cmd/cmd.go`）：
- `parse.Parse(excelDir, &parse.Options{...})`
- 导出代码：`export/code.ExportGo(...)` 或 `export/code.ExportCSharp(...)`
- 导出数据：`export/data.ExportJson|ExportBytes|ExportBson(...)`

### C# 代码导出

C# 导出核心位于 `export/code/csharp.go` 与 `export/code/csharp_template.go`：
- 支持三种数据格式：`json`、`bytes`、`bson`。
- 导出顺序与 Go 版一致：枚举 -> 结构体 -> 各配置表 -> `Tables` 静态管理类 -> `LoadHelper`。
- 生成文件组织：
  - `<namespace>_enums.cs`
  - `<namespace>_structs.cs`
  - 每张表一个 `<table>.cs`
  - `<namespace>_tables.cs`
  - `<namespace>_load_helper.cs`
- 命名规则：
  - 普通表会生成 “entry 类 + table 类”，table 类名为 `<TableName>Table`
  - `Tables` 静态管理类名来自 `CSharpOptions.TablesClassName`
  - 普通字段/属性名转为 CamelCase，`ID` 特殊映射为 `Id`
- 类型映射关键点：
  - `int32` 枚举导出为原生 C# `enum`
  - `string` 枚举退化为 `static class + const string`
  - 数组导出为 `List<T>`，map 导出为 `Dictionary<TKey, TValue>`
- 运行时加载关键点：
  - 普通表会构建 `UNIQUE` / `CKEY` / `GROUP` 对应的查询索引
  - `Tables` 通过 `Volatile.Read` / `Interlocked.Exchange` 发布最新表实例
  - 对外暴露全量重载 `LoadAsync(...)` 和按调用顺序局部重载 `LoadTableAsync(...)`
  - 支持外部注册 `RegisterAfterLoadFunc(tableName, func, priority)`，不会为单表自动生成 after-load hook

### Go 代码导出

Go 导出模板位于 `export/code/go_template.go`：
- 生成的加载代码现在会导出公共类型 `AfterLoadFunc` 和公共函数 `RegisterAfterLoadFunc(tableName, f, priority)`。
- 外部调用方可以在加载前注册表级“加载后处理函数”；同一张表只允许注册一次，重复注册会 `panic`。
- after-load 回调按 `priority` 升序执行；执行失败时会通过 `pkg/errors.WithMessagef` 补充 `table[...] after load` 上下文。

JSON 导出要点（`export/data/json.go`）：
- 普通表导出为 JSON 数组；每条 entry 的 `ID` 字段输出为 `id`（见 `export.TableFieldIDJsonName`）。
- 全局表导出为 JSON 对象（key 为字段名）。

BSON 导出需要 MongoDB 连接信息（`-mongo-uri`、`-mongo-db`）。

## 代码约定（修改时遵循）

- 错误处理：
  - 需要叠加上下文时优先使用 `github.com/pkg/errors` 的 `WithMessage/WithMessagef`。
  - 纯透传且上下文已足够时直接 `return err`，避免层层包装导致信息噪音。
- 命名与正则：
  - 名称合法性通常由 `gexcels.MatchName`（`gexcels.NamePattern`）约束。
- 修改解析行为时优先补充/更新测试：
  - `parse/parse_test.go` 使用 `internal/test/excels/` 内样例做回归。
