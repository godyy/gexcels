package gexcels

import (
	"testing"
)

func TestParse(t *testing.T) {
	options := &ParseOptions{
		Tag:            []string{"s"},
		StructFilePath: "struct_define.xlsx",
	}
	p, err := Parse("./internal/test/excels", options)
	if err != nil {
		t.Fatal(err)
	}

	t.Log(p.checkLinksBetweenTable())
}
