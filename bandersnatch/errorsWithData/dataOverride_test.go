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

	testutils.FatalUnless(t, testutils.CheckPanic(mergeMaps, &m1, ParamMap{"Bar":nil}, AssertDataIsNotReplaced), "")
}
