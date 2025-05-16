package parse

import (
	"testing"

	"github.com/godyy/gexcels"
)

func TestParse(t *testing.T) {
	options := &Options{}
	_, err := Parse("../internal/test/excels", options)
	if err != nil {
		t.Fatal(err)
	}

	options.FileTags = []gexcels.Tag{gexcels.TagEmpty, "test"}
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
	t.Log(tableFileNameRegexp.FindStringSubmatch("test"))
	t.Log(tableFileNameRegexp.FindStringSubmatch("test.test"))
	t.Log(tableFileNameRegexp.FindStringSubmatch("test."))

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
