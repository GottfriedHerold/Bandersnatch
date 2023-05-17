package errorsWithData

import (
	"testing"

	"github.com/GottfriedHerold/Bandersnatch/internal/testutils"
	"github.com/GottfriedHerold/Bandersnatch/internal/utils"
)

func TestMergeMaps(t *testing.T) {
	var m1 ParamMap = ParamMap{}
	var m2 ParamMap = ParamMap{"Foo": 5}

	mergeMaps(&m2, ParamMap{}, PreferPreviousData)
	testutils.FatalUnless(t, utils.CompareParamMaps(m2, ParamMap{"Foo": 5}), "")
	mergeMaps(&m2, ParamMap{}, ReplacePreviousData)
	testutils.FatalUnless(t, utils.CompareParamMaps(m2, ParamMap{"Foo": 5}), "")
	mergeMaps(&m2, ParamMap{}, AssertDataIsNotReplaced)
	testutils.FatalUnless(t, utils.CompareParamMaps(m2, ParamMap{"Foo": 5}), "")
	mergeMaps(&m1, ParamMap{"Bar": 5}, AssertDataIsNotReplaced)
	mergeMaps(&m1, ParamMap{"Bar": 6}, ReplacePreviousData)
	mergeMaps(&m1, ParamMap{"Bar": uint(7)}, PreferPreviousData)
	testutils.FatalUnless(t, utils.CompareParamMaps(m1, ParamMap{"Bar": 6}), "")

	testutils.FatalUnless(t, testutils.CheckPanic(mergeMaps, &m1, ParamMap{"Bar": nil}, AssertDataIsNotReplaced), "")
}

func TestFillMapFromStruct2(t *testing.T) {
	var m1 ParamMap = ParamMap{}
	var m2 ParamMap = ParamMap{"Foo": 5}
	type T1 struct{ Foo int }
	type T2 struct{ Bar uint }
	fillMapFromStruct(&T1{Foo: 4}, &m2, PreferPreviousData)
	testutils.FatalUnless(t, utils.CompareParamMaps(m2, ParamMap{"Foo": 5}), "")
	delete(m2, "Foo")
	fillMapFromStruct(&T1{Foo: 4}, &m2, PreferPreviousData)
	testutils.FatalUnless(t, utils.CompareParamMaps(m2, ParamMap{"Foo": 4}), "")
	fillMapFromStruct(&T1{Foo: 6}, &m2, ReplacePreviousData)
	testutils.FatalUnless(t, utils.CompareParamMaps(m2, ParamMap{"Foo": 6}), "")
	fillMapFromStruct(&T1{Foo: 6}, &m2, AssertDataIsNotReplaced)
	testutils.FatalUnless(t, utils.CompareParamMaps(m2, ParamMap{"Foo": 6}), "")

	fillMapFromStruct(&T2{Bar: 5}, &m1, AssertDataIsNotReplaced)
	fillMapFromStruct(&T2{Bar: 6}, &m1, ReplacePreviousData)
	fillMapFromStruct(&T2{Bar: uint(7)}, &m1, PreferPreviousData)
	testutils.FatalUnless(t, utils.CompareParamMaps(m1, ParamMap{"Bar": uint(6)}), "%v", m1)

	testutils.FatalUnless(t, testutils.CheckPanic(fillMapFromStruct[T2], &T2{Bar: 8}, &m1, AssertDataIsNotReplaced), "")
}

/*
func TestFillMapFromStruct(t *testing.T) {
	var m map[string]any
	var empty struct{}
	fillMapFromStruct(&empty, &m, AssertDataIsNotReplaced) // note: implied type parameter is unnamed
	if m == nil {
		t.Fatalf("E1")
	}
	if len(m) != 0 {
		t.Fatalf("E2")
	}
	m["x"] = 1
	type T1 struct {
		Name1 int
		Name2 string
		Name3 error // NOTE: interface type
	}
	type NestedT1 struct {
		T1
		Name1 uint // shadows T2.name1
		Name4 byte
	}
	var t1 T1 = T1{Name1: 1, Name2: "foo", Name3: io.EOF}
	fillMapFromStruct(&t1, &m, ReplacePreviousData)
	if m["x"] != 1 || m["Name1"] != int(1) || m["Name2"] != "foo" || m["Name3"] != io.EOF {
		t.Fatalf("E3")
	}
	t1copy, err := makeStructFromMap[T1](m)
	if err != nil {
		t.Fatalf("E4")
	}
	if t1copy != t1 {
		t.Fatalf("E5")
	}
	var m2 map[string]any
	t1other := T1{Name1: 2, Name2: "bar", Name3: nil}
	tEmbed := NestedT1{T1: t1other, Name1: 3, Name4: 4}
	fillMapFromStruct(&tEmbed, &m2, ReplacePreviousData)
	if m2["Name3"] != nil {
		t.Fatalf("E6")
	}
	if m2["Name1"] != uint(3) {
		t.Fatalf("E7")
	}
	_, ok := m2["T1"]
	if ok {
		t.Fatalf("E8")
	}
	t1EmbedRetrieved, _ := makeStructFromMap[NestedT1](m2)
	// Roundtrip will not work, because shadowed fields differ
	if t1EmbedRetrieved == tEmbed {
		t.Fatalf("E9")
	}
	// After zeroing shadowed field, it should behave like roundtrip
	tEmbed.T1.Name1 = 0
	if t1EmbedRetrieved != tEmbed {
		t.Fatalf("E10")
	}
}
*/

func TestPrintPreviousDataTreatment(t *testing.T) {
	s1 := AssertDataIsNotReplaced.String()
	s2 := PreferPreviousData.String()
	s3 := ReplacePreviousData.String()
	testutils.FatalUnless(t, s1 != "", "")
	testutils.FatalUnless(t, s2 != "", "")
	testutils.FatalUnless(t, s3 != "", "")

	testutils.FatalUnless(t, s1 != s2, "")
	testutils.FatalUnless(t, s1 != s3, "")
	testutils.FatalUnless(t, s2 != s3, "")
}
