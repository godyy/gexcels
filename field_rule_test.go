package gexcels

import "testing"

func TestFieldRule(t *testing.T) {
	frUnique := NewFRUnique()
	frUniqueString := frUnique.String()
	t.Log(frUnique.FRName() + ":" + frUniqueString)

	frLink := NewFRLink("tableName", "fieldName")
	frLinkString := frLink.String()
	t.Log(frUnique.FRName() + ":" + frLinkString)

	frCompositeKey := NewFRCompositeKey("ck", 1)
	frCompositeKeyString := frCompositeKey.String()
	t.Log(frCompositeKey.FRName() + ":" + frCompositeKeyString)

	frGroup := NewFRGroup("group", 2)
	frGroupString := frGroup.String()
	t.Log(frGroup.FRName() + ":" + frGroupString)

	fr, err := ParseFieldRule(frUniqueString)
	if err != nil {
		t.Fatal("parse", frUniqueString, err)
	}
	t.Log(fr)

	fr, err = ParseFieldRule(frLinkString)
	if err != nil {
		t.Fatal("parse", frLinkString, err)
	}
	t.Log(fr)

	fr, err = ParseFieldRule(frCompositeKeyString)
	if err != nil {
		t.Fatal("parse", frCompositeKeyString, err)
	}
	t.Log(fr)

	fr, err = ParseFieldRule(frGroupString)
	if err != nil {
		t.Fatal("parse", frGroupString, err)
	}
	t.Log(fr)
}
