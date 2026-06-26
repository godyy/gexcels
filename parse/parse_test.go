package parse

import (
	"path/filepath"
	"testing"

	"github.com/godyy/gexcels"
	"github.com/tealeg/xlsx/v3"
)

func TestParse(t *testing.T) {
	options := &Options{Tags: []gexcels.Tag{"s"}}
	_, err := Parse("../internal/test/excels", options)
	if err != nil {
		t.Fatal(err)
	}
}

func TestParseOnlyFields(t *testing.T) {
	options := &Options{OnlyFields: true, Tags: []gexcels.Tag{"s"}}
	_, err := Parse("../internal/test/excels", options)
	if err != nil {
		t.Fatal(err)
	}
}

func TestParseMapType(t *testing.T) {
	dir := t.TempDir()

	{
		f := xlsx.NewFile()
		sheet, err := f.AddSheet("desc|Struct")
		if err != nil {
			t.Fatal(err)
		}

		row0 := sheet.AddRow()
		for i := 0; i < gexcels.TableStructCols; i++ {
			row0.AddCell()
		}

		row1 := sheet.AddRow()
		row1.AddCell().SetString("")
		row1.AddCell().SetString("Foo")
		row1.AddCell().SetString(`AA:int32:"a",MM:map[string]int32:"m"`)
		row1.AddCell().SetString("")
		row1.AddCell().SetString("foo")

		if err := f.Save(filepath.Join(dir, "struct.xlsx")); err != nil {
			t.Fatal(err)
		}
	}

	{
		f := xlsx.NewFile()
		sheet, err := f.AddSheet("desc|Test")
		if err != nil {
			t.Fatal(err)
		}

		row0 := sheet.AddRow()
		row0.AddCell().SetString("ID")
		row0.AddCell().SetString("MM")
		row0.AddCell().SetString("Obj")

		row1 := sheet.AddRow()
		row1.AddCell().SetString("id")
		row1.AddCell().SetString("m")
		row1.AddCell().SetString("obj")

		row2 := sheet.AddRow()
		row2.AddCell().SetString("int32")
		row2.AddCell().SetString("map[string]int32")
		row2.AddCell().SetString("map[string]Foo")

		row3 := sheet.AddRow()
		row3.AddCell().SetString("")
		row3.AddCell().SetString("")
		row3.AddCell().SetString("")

		row4 := sheet.AddRow()
		row4.AddCell().SetString("")
		row4.AddCell().SetString("")
		row4.AddCell().SetString("")

		row5 := sheet.AddRow()
		row5.AddCell().SetString("1")
		row5.AddCell().SetString(`{"a":1,"b":2}`)
		row5.AddCell().SetString(`{"k1":{"AA":11,"MM":{"x":5}}}`)

		if err := f.Save(filepath.Join(dir, "test.xlsx")); err != nil {
			t.Fatal(err)
		}
	}

	p, err := Parse(dir, &Options{})
	if err != nil {
		t.Fatal(err)
	}

	sd := p.GetStructByName("Foo")
	if sd == nil {
		t.Fatalf("struct Foo not found")
	}
	fd := sd.GetFieldByName("MM")
	if fd == nil || fd.Type != gexcels.FTMap {
		t.Fatalf("struct Foo.MM not map")
	}
	if fd.GetMapKeyType().Type != gexcels.FTString || fd.GetMapValueType().Type != gexcels.FTInt32 {
		t.Fatalf("struct Foo.M type invalid: %s", fd.FieldTypeInfo.String())
	}

	var td *Table
	for _, t2 := range p.Tables {
		if t2.Name == "Test" {
			td = t2
			break
		}
	}
	if td == nil {
		t.Fatalf("table Test not found")
	}
	if len(td.Entries) != 1 {
		t.Fatalf("table Test entry amount invalid: %d", len(td.Entries))
	}

	entry := td.Entries[0]
	m, ok := entry["MM"].(map[string]any)
	if !ok {
		t.Fatalf("table Test.MM not map[string]any")
	}
	if m["a"] != int32(1) || m["b"] != int32(2) {
		t.Fatalf("table Test.MM values invalid: %+v", m)
	}

	obj, ok := entry["Obj"].(map[string]any)
	if !ok {
		t.Fatalf("table Test.Obj not map[string]any")
	}
	k1, ok := obj["k1"].(map[string]any)
	if !ok {
		t.Fatalf("table Test.Obj[k1] not object")
	}
	if k1["AA"] != int32(11) {
		t.Fatalf("table Test.Obj[k1].AA invalid: %v", k1["AA"])
	}
	m2, ok := k1["MM"].(map[string]any)
	if !ok {
		t.Fatalf("table Test.Obj[k1].MM not map")
	}
	if m2["x"] != int32(5) {
		t.Fatalf("table Test.Obj[k1].MM[x] invalid: %v", m2["x"])
	}
}

func TestParseMapKeyTypes(t *testing.T) {
	dir := t.TempDir()

	{
		f := xlsx.NewFile()
		sheet, err := f.AddSheet("desc|EnumKey")
		if err != nil {
			t.Fatal(err)
		}

		row0 := sheet.AddRow()
		row0.AddCell().SetString("EK_BEGIN")
		row0.AddCell().SetString("int32")
		row0.AddCell().SetString("enum key")

		row1 := sheet.AddRow()
		row1.AddCell().SetString("AA")
		row1.AddCell().SetString("1")
		row1.AddCell().SetString("a")

		row2 := sheet.AddRow()
		row2.AddCell().SetString("BB")
		row2.AddCell().SetString("2")
		row2.AddCell().SetString("b")

		row3 := sheet.AddRow()
		row3.AddCell().SetString("")

		if err := f.Save(filepath.Join(dir, "enum.xlsx")); err != nil {
			t.Fatal(err)
		}
	}

	{
		f := xlsx.NewFile()
		sheet, err := f.AddSheet("desc|KeyMap")
		if err != nil {
			t.Fatal(err)
		}

		row0 := sheet.AddRow()
		row0.AddCell().SetString("ID")
		row0.AddCell().SetString("MInt32")
		row0.AddCell().SetString("MEnum")

		row1 := sheet.AddRow()
		row1.AddCell().SetString("id")
		row1.AddCell().SetString("m int32")
		row1.AddCell().SetString("m enum")

		row2 := sheet.AddRow()
		row2.AddCell().SetString("int32")
		row2.AddCell().SetString("map[int32]int32")
		row2.AddCell().SetString("map[EK]int32")

		row3 := sheet.AddRow()
		row3.AddCell().SetString("")
		row3.AddCell().SetString("")
		row3.AddCell().SetString("")

		row4 := sheet.AddRow()
		row4.AddCell().SetString("")
		row4.AddCell().SetString("")
		row4.AddCell().SetString("")

		row5 := sheet.AddRow()
		row5.AddCell().SetString("1")
		row5.AddCell().SetString(`{"1":10,"2":20}`)
		row5.AddCell().SetString(`{"AA":100,"BB":200}`)

		if err := f.Save(filepath.Join(dir, "test.xlsx")); err != nil {
			t.Fatal(err)
		}
	}

	p, err := Parse(dir, &Options{})
	if err != nil {
		t.Fatal(err)
	}

	var td *Table
	for _, t2 := range p.Tables {
		if t2.Name == "KeyMap" {
			td = t2
			break
		}
	}
	if td == nil {
		t.Fatalf("table KeyMap not found")
	}
	if len(td.Entries) != 1 {
		t.Fatalf("table KeyMap entry amount invalid: %d", len(td.Entries))
	}

	entry := td.Entries[0]

	mInt32, ok := entry["MInt32"].(map[int32]any)
	if !ok {
		t.Fatalf("table KeyMap.MInt32 not map[int32]any")
	}
	if mInt32[int32(1)] != int32(10) || mInt32[int32(2)] != int32(20) {
		t.Fatalf("table KeyMap.MInt32 values invalid: %+v", mInt32)
	}

	mEnum, ok := entry["MEnum"].(map[int32]any)
	if !ok {
		t.Fatalf("table KeyMap.MEnum not map[int32]any")
	}
	if mEnum[int32(1)] != int32(100) || mEnum[int32(2)] != int32(200) {
		t.Fatalf("table KeyMap.MEnum values invalid: %+v", mEnum)
	}
}

func TestParseLinkNestedAndMapKeyValue(t *testing.T) {
	dir := t.TempDir()

	{
		f := xlsx.NewFile()
		sheet, err := f.AddSheet("desc|Struct")
		if err != nil {
			t.Fatal(err)
		}

		row0 := sheet.AddRow()
		for i := 0; i < gexcels.TableStructCols; i++ {
			row0.AddCell()
		}

		row1 := sheet.AddRow()
		row1.AddCell().SetString("")
		row1.AddCell().SetString("Foo")
		row1.AddCell().SetString(`AA:int32:"a",MM:[]map[int32][]map[int32]int32:"m"`)
		row1.AddCell().SetString("LINK=AA,Item.ID|LINK=MM,k1=Key1.ID,k2=Key2.ID")
		row1.AddCell().SetString("foo")

		if err := f.Save(filepath.Join(dir, "struct.xlsx")); err != nil {
			t.Fatal(err)
		}
	}

	{
		f := xlsx.NewFile()

		{
			sheet, err := f.AddSheet("desc|Item")
			if err != nil {
				t.Fatal(err)
			}

			row0 := sheet.AddRow()
			row0.AddCell().SetString("ID")

			row1 := sheet.AddRow()
			row1.AddCell().SetString("id")

			row2 := sheet.AddRow()
			row2.AddCell().SetString("int32")

			row3 := sheet.AddRow()
			row3.AddCell().SetString("")

			row4 := sheet.AddRow()
			row4.AddCell().SetString("")

			row5 := sheet.AddRow()
			row5.AddCell().SetString("1")

			row6 := sheet.AddRow()
			row6.AddCell().SetString("2")
		}

		{
			sheet, err := f.AddSheet("desc|Key1")
			if err != nil {
				t.Fatal(err)
			}

			row0 := sheet.AddRow()
			row0.AddCell().SetString("ID")

			row1 := sheet.AddRow()
			row1.AddCell().SetString("id")

			row2 := sheet.AddRow()
			row2.AddCell().SetString("int32")

			row3 := sheet.AddRow()
			row3.AddCell().SetString("")

			row4 := sheet.AddRow()
			row4.AddCell().SetString("")

			row5 := sheet.AddRow()
			row5.AddCell().SetString("1")

			row6 := sheet.AddRow()
			row6.AddCell().SetString("2")
		}

		{
			sheet, err := f.AddSheet("desc|Key2")
			if err != nil {
				t.Fatal(err)
			}

			row0 := sheet.AddRow()
			row0.AddCell().SetString("ID")

			row1 := sheet.AddRow()
			row1.AddCell().SetString("id")

			row2 := sheet.AddRow()
			row2.AddCell().SetString("int32")

			row3 := sheet.AddRow()
			row3.AddCell().SetString("")

			row4 := sheet.AddRow()
			row4.AddCell().SetString("")

			row5 := sheet.AddRow()
			row5.AddCell().SetString("1")

			row6 := sheet.AddRow()
			row6.AddCell().SetString("2")
		}

		{
			sheet, err := f.AddSheet("desc|Src")
			if err != nil {
				t.Fatal(err)
			}

			row0 := sheet.AddRow()
			row0.AddCell().SetString("ID")
			row0.AddCell().SetString("MM")
			row0.AddCell().SetString("MS")
			row0.AddCell().SetString("Arr")
			row0.AddCell().SetString("NM")

			row1 := sheet.AddRow()
			row1.AddCell().SetString("id")
			row1.AddCell().SetString("mm")
			row1.AddCell().SetString("ms")
			row1.AddCell().SetString("arr")
			row1.AddCell().SetString("nm")

			row2 := sheet.AddRow()
			row2.AddCell().SetString("int32")
			row2.AddCell().SetString("map[int32]int32")
			row2.AddCell().SetString("map[int32]Foo")
			row2.AddCell().SetString("[][]int32")
			row2.AddCell().SetString("[]map[int32][]map[int32]int32")

			row3 := sheet.AddRow()
			row3.AddCell().SetString("")
			row3.AddCell().SetString("LINK=Item.ID,k1=Key1.ID")
			row3.AddCell().SetString("LINK=k1=Key1.ID")
			row3.AddCell().SetString("LINK=Item.ID")
			row3.AddCell().SetString("LINK=Item.ID,k1=Key1.ID,k2=Key2.ID")

			row4 := sheet.AddRow()
			row4.AddCell().SetString("")
			row4.AddCell().SetString("")
			row4.AddCell().SetString("")
			row4.AddCell().SetString("")
			row4.AddCell().SetString("")

			row5 := sheet.AddRow()
			row5.AddCell().SetString("1")
			row5.AddCell().SetString(`{"1":2,"2":1}`)
			row5.AddCell().SetString(`{"1":{"AA":1},"2":{"AA":2}}`)
			row5.AddCell().SetString(`[[1,2],[2]]`)
			row5.AddCell().SetString(`[{"1":[{"1":1,"2":2}],"2":[{"2":1}]}]`)
		}

		if err := f.Save(filepath.Join(dir, "test.xlsx")); err != nil {
			t.Fatal(err)
		}
	}

	_, err := Parse(dir, &Options{})
	if err != nil {
		t.Fatal(err)
	}
}

func TestRegexp(t *testing.T) {
	t.Log(structFileNameRegexp.FindStringSubmatch("core.struct"))
	t.Log(structFileNameRegexp.FindStringSubmatch("core.struct.test"))
	t.Log(structFileNameRegexp.FindStringSubmatch("core.struct."))
	t.Log(structFileNameRegexp.FindStringSubmatch("core.struct.0"))
	t.Log(structFileNameRegexp.FindStringSubmatch("core.struct.0.test"))
	t.Log(structFileNameRegexp.FindStringSubmatch("core.struct.0"))

	t.Log(structSheetNameRegexp.FindStringSubmatch("|Struct"))
	t.Log(structSheetNameRegexp.FindStringSubmatch("|Struct.0"))
	t.Log(structSheetNameRegexp.FindStringSubmatch("|StructCommon"))
	t.Log(structSheetNameRegexp.FindStringSubmatch("|StructCommon.0"))
}
