package parse

import (
	"github.com/godyy/gexcels"
	"testing"
)

func TestParse(t *testing.T) {
	options := &Options{}
	_, err := Parse("../internal/test/excels", options)
	if err != nil {
		t.Fatal(err)
	}

	options.FileTags = []gexcels.Tag{"test"}
	_, err = Parse("../internal/test/excels", options)
	if err != nil {
		t.Fatal(err)
	}
}

func TestRegexp(t *testing.T) {
	t.Log(tableFileNameRegexp.FindStringSubmatch("test"))
	t.Log(tableFileNameRegexp.FindStringSubmatch("test.test"))
	t.Log(tableFileNameRegexp.FindStringSubmatch("test."))

	t.Log(structFileNameRegexp.FindStringSubmatch("common.struct"))
	t.Log(structFileNameRegexp.FindStringSubmatch("common.struct.test"))
	t.Log(structFileNameRegexp.FindStringSubmatch("common.struct."))
	t.Log(structFileNameRegexp.FindStringSubmatch("common.struct.0"))
	t.Log(structFileNameRegexp.FindStringSubmatch("common.struct.0.test"))
	t.Log(structFileNameRegexp.FindStringSubmatch("common.struct.0"))

	t.Log(structSheetNameRegexp.FindStringSubmatch("|Struct"))
	t.Log(structSheetNameRegexp.FindStringSubmatch("|Struct.0"))
	t.Log(structSheetNameRegexp.FindStringSubmatch("|StructCommon"))
	t.Log(structSheetNameRegexp.FindStringSubmatch("|StructCommon.0"))
}
