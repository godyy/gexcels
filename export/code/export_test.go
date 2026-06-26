package code

import (
	"os"
	"strings"
	"testing"

	"github.com/godyy/gexcels"
	"github.com/godyy/gexcels/export"
	"github.com/godyy/gexcels/parse"
)

func parseTestParser(t *testing.T) *parse.Parser {
	t.Helper()

	excelsPath := "../../internal/test/excels"
	p, err := parse.Parse(excelsPath, &parse.Options{
		OnlyFields: false,
		Tags:       []gexcels.Tag{"s"},
	})
	if err != nil {
		t.Fatalf("parse %s, %v", excelsPath, err)
	}
	return p
}

func TestExportGoJson(t *testing.T) {
	exportGoPath := "../../internal/test/export/go_json"
	p := parseTestParser(t)

	if err := ExportGo(p, exportGoPath, &Options{
		DataKind: export.DataJson,
	}, &GoOptions{PkgName: "test"}); err != nil {
		t.Fatalf("export go to %s, %v", exportGoPath, err)
	}
}

func TestExportGoBytes(t *testing.T) {
	exportGoPath := "../../internal/test/export/go_bytes"
	p := parseTestParser(t)

	if err := ExportGo(p, exportGoPath, &Options{
		DataKind: export.DataBytes,
	}, &GoOptions{PkgName: "test"}); err != nil {
		t.Fatalf("export go to %s, %v", exportGoPath, err)
	}
}

func TestExportGoBson(t *testing.T) {
	exportGoPath := "../../internal/test/export/go_bson"
	p := parseTestParser(t)

	if err := ExportGo(p, exportGoPath, &Options{
		DataKind: export.DataBson,
	}, &GoOptions{PkgName: "test"}); err != nil {
		t.Fatalf("export go to %s, %v", exportGoPath, err)
	}
}

func TestExportCSharpJson(t *testing.T) {
	exportPath := "../../internal/test/export/csharp_json"
	p := parseTestParser(t)

	if err := ExportCSharp(p, exportPath, &Options{
		DataKind: export.DataJson,
	}, &CSharpOptions{
		Namespace:       "Test.Config",
		TablesClassName: "ConfigTables",
	}); err != nil {
		t.Fatalf("export csharp to %s, %v", exportPath, err)
	}

	tablesBytes, err := os.ReadFile(exportPath + "/test_config_tables.cs")
	if err != nil {
		t.Fatalf("read generated csharp tables file, %v", err)
	}
	tablesCode := string(tablesBytes)
	if !strings.Contains(tablesCode, "public static class ConfigTables") {
		t.Fatalf("generated csharp tables file missing custom tables class name")
	}
	if !strings.Contains(tablesCode, "/// ConfigTables provides centralized access to all exported tables.") {
		t.Fatalf("generated csharp tables file missing tables class comment")
	}
	if !strings.Contains(tablesCode, "static ConfigTables()") {
		t.Fatalf("generated csharp tables file missing custom static constructor name")
	}
	if strings.Contains(tablesCode, "static Tables()") {
		t.Fatalf("generated csharp tables file should not keep default static constructor name")
	}
	if !strings.Contains(tablesCode, "global::System.Threading.Volatile.Read(ref sItem)") {
		t.Fatalf("generated csharp tables file missing volatile read")
	}
	if !strings.Contains(tablesCode, "global::System.Threading.Interlocked.Exchange(ref sItem, table);") {
		t.Fatalf("generated csharp tables file missing interlocked exchange")
	}
	if !strings.Contains(tablesCode, "RegisterLoadFunc(ItemTable.TableName") {
		t.Fatalf("generated csharp tables file missing load function registration")
	}
	if !strings.Contains(tablesCode, "public static void RegisterAfterLoadFunc(string tableName, global::System.Func<global::System.Threading.Tasks.Task> func, int priority)") {
		t.Fatalf("generated csharp tables file missing public after load registration api")
	}
	if strings.Contains(tablesCode, "RegisterAfterLoadFunc(ItemTable.TableName") {
		t.Fatalf("generated csharp tables file should not auto register after load hooks")
	}
	if strings.Contains(tablesCode, "switch (tableName)") {
		t.Fatalf("generated csharp tables file should not use switch dispatch")
	}

	itemBytes, err := os.ReadFile(exportPath + "/item.cs")
	if err != nil {
		t.Fatalf("read generated csharp item table file, %v", err)
	}
	if strings.Contains(string(itemBytes), "AfterLoadAsync") {
		t.Fatalf("generated csharp item table file should not auto generate after load hook")
	}
	if !strings.Contains(string(itemBytes), "/// 道具") {
		t.Fatalf("generated csharp item table file missing table class comment")
	}
	itemCode := string(itemBytes)
	if !strings.Contains(itemCode, "entries = await LoadHelper.LoadTableAsync<global::System.Collections.Generic.List<Item>>(basePath, TableName);") {
		t.Fatalf("generated csharp item table file should assign loaded entries directly")
	}
	if strings.Contains(itemCode, "entries.Clear();") || strings.Contains(itemCode, "entries.AddRange(") {
		t.Fatalf("generated csharp item table file should not rebuild entries via clear/addrange")
	}

	loadHelperBytes, err := os.ReadFile(exportPath + "/test_config_load_helper.cs")
	if err != nil {
		t.Fatalf("read generated csharp load helper file, %v", err)
	}
	loadHelperCode := string(loadHelperBytes)
	if !strings.Contains(loadHelperCode, "/// LoadHelper centralizes json-based table loading.") {
		t.Fatalf("generated csharp load helper file missing class comment")
	}
	if !strings.Contains(loadHelperCode, "/// LoadTableAsync loads one table object from its json file.") {
		t.Fatalf("generated csharp load helper file missing method comment")
	}
}

func TestExportCSharpBson(t *testing.T) {
	exportPath := "../../internal/test/export/csharp_bson"
	p := parseTestParser(t)

	if err := ExportCSharp(p, exportPath, &Options{
		DataKind: export.DataBson,
	}, &CSharpOptions{
		Namespace:       "Test.Config",
		TablesClassName: "ConfigTables",
	}); err != nil {
		t.Fatalf("export csharp to %s, %v", exportPath, err)
	}

	globalBytes, err := os.ReadFile(exportPath + "/global_test.cs")
	if err != nil {
		t.Fatalf("read generated csharp global table file, %v", err)
	}
	globalCode := string(globalBytes)
	if !strings.Contains(globalCode, `[global::MongoDB.Bson.Serialization.Attributes.BsonElementAttribute("maxInt32")]`) {
		t.Fatalf("generated csharp global table file missing bson element for MaxInt32")
	}
	if !strings.Contains(globalCode, `[global::MongoDB.Bson.Serialization.Attributes.BsonElementAttribute("systemName")]`) {
		t.Fatalf("generated csharp global table file missing bson element for SystemName")
	}
	if strings.Contains(globalCode, "BsonIdAttribute") {
		t.Fatalf("generated csharp global table file should not use BsonId on global fields")
	}
	if !strings.Contains(globalCode, "internal static async global::System.Threading.Tasks.Task<GlobalTestTable> LoadAsync(global::MongoDB.Driver.IMongoDatabase db)") {
		t.Fatalf("generated csharp global table file should load via static factory-style method")
	}
	if strings.Contains(globalCode, "CopyFrom(") {
		t.Fatalf("generated csharp global table file should not keep copy helper")
	}

	tablesBytes, err := os.ReadFile(exportPath + "/test_config_tables.cs")
	if err != nil {
		t.Fatalf("read generated csharp tables file, %v", err)
	}
	tablesCode := string(tablesBytes)
	if !strings.Contains(tablesCode, "var table = await GlobalTestTable.LoadAsync(db);") {
		t.Fatalf("generated csharp tables file should publish global tables from loaded instance directly")
	}
}

func TestExportCSharpBytes(t *testing.T) {
	exportPath := "../../internal/test/export/csharp_bytes"
	p := parseTestParser(t)

	if err := ExportCSharp(p, exportPath, &Options{
		DataKind: export.DataBytes,
	}, &CSharpOptions{
		Namespace:       "Test.Config",
		TablesClassName: "ConfigTables",
	}); err != nil {
		t.Fatalf("export csharp to %s, %v", exportPath, err)
	}

	itemBytes, err := os.ReadFile(exportPath + "/item.cs")
	if err != nil {
		t.Fatalf("read generated csharp bytes item table file, %v", err)
	}
	itemCode := string(itemBytes)
	if !strings.Contains(itemCode, "internal async global::System.Threading.Tasks.Task LoadAsync(string basePath)") {
		t.Fatalf("generated csharp bytes item table file missing bytes load method")
	}
	if !strings.Contains(itemCode, "entries = await LoadHelper.LoadTableAsync<Item>(basePath, TableName);") {
		t.Fatalf("generated csharp bytes item table file should use bytes load helper")
	}

	globalBytes, err := os.ReadFile(exportPath + "/global_test.cs")
	if err != nil {
		t.Fatalf("read generated csharp bytes global table file, %v", err)
	}
	globalCode := string(globalBytes)
	if !strings.Contains(globalCode, "internal static async global::System.Threading.Tasks.Task<GlobalTestTable> LoadAsync(string basePath)") {
		t.Fatalf("generated csharp bytes global table file missing bytes global load method")
	}
	if !strings.Contains(globalCode, "return await LoadHelper.LoadGlobalAsync<GlobalTestTable>(basePath, TableName);") {
		t.Fatalf("generated csharp bytes global table file should use bytes global load helper")
	}

	loadHelperBytes, err := os.ReadFile(exportPath + "/test_config_load_helper.cs")
	if err != nil {
		t.Fatalf("read generated csharp bytes load helper file, %v", err)
	}
	loadHelperCode := string(loadHelperBytes)
	if !strings.Contains(loadHelperCode, "internal static class LoadHelper") {
		t.Fatalf("generated csharp bytes load helper file missing load helper class")
	}
	if !strings.Contains(loadHelperCode, "public static async global::System.Threading.Tasks.Task<global::System.Collections.Generic.List<T>> LoadTableAsync<T>(string basePath, string tableName)") {
		t.Fatalf("generated csharp bytes load helper file missing bytes table load helper")
	}
	if !strings.Contains(loadHelperCode, "private sealed class BytesReader") {
		t.Fatalf("generated csharp bytes load helper file missing bytes reader")
	}

	tablesBytes, err := os.ReadFile(exportPath + "/test_config_tables.cs")
	if err != nil {
		t.Fatalf("read generated csharp bytes tables file, %v", err)
	}
	tablesCode := string(tablesBytes)
	if !strings.Contains(tablesCode, "private delegate global::System.Threading.Tasks.Task LoadFunc(string basePath);") {
		t.Fatalf("generated csharp bytes tables file should use file-based load delegate")
	}
}
