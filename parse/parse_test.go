package parse

import (
	"testing"
)

func TestParse(t *testing.T) {
	options := &Options{}
	_, err := Parse("../internal/test/excels", options)
	if err != nil {
		t.Fatal(err)
	}

	_, err = Parse("../internal/test/excels", options)
	if err != nil {
		t.Fatal(err)
	}
}

func TestParseOnlyFields(t *testing.T) {
	options := &Options{OnlyFields: true}
	_, err := Parse("../internal/test/excels", options)
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
