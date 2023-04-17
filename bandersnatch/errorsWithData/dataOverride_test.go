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
