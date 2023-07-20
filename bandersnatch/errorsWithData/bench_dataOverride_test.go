package errorsWithData

import "testing"

func BenchmarkMergeMaps(b *testing.B) {
	var inputMap ParamMap = make(ParamMap)
	inputMap["Param1"] = int(5)
	inputMap["Param2"] = nil
	inputMap["Param3"] = "some string"

	prepareBenchmarkErrorsWithData(b)
	for n := 0; n < b.N; n++ {
		mergeMaps(&Dump_ParamMaps[n%benchS], inputMap, AssertDataIsNotReplaced)
	}
}

func BenchmarkFillMapFromStruct(bOuter *testing.B) {
	var outputMap ParamMap = make(ParamMap)
	type S1 struct {
		Param1 int
		Param2 *int
		Param3 string
	}
	var some_int int = 5
	var s1 S1 = S1{Param1: 5, Param2: &some_int, Param3: "some string"}
	bOuter.Run("PreferOld", func(bInner *testing.B) {
		prepareBenchmarkErrorsWithData(bInner)
		for n := 0; n < bInner.N; n++ {
			fillMapFromStruct(&s1, &outputMap, PreferPreviousData)
			// outputMap = nil
		}
	})
	bOuter.Run("PreferNew", func(bInner *testing.B) {
		prepareBenchmarkErrorsWithData(bInner)
		for n := 0; n < bInner.N; n++ {
			fillMapFromStruct(&s1, &outputMap, ReplacePreviousData)
			// outputMap = nil
		}
	})
	bOuter.Run("AssertEqual", func(bInner *testing.B) {
		prepareBenchmarkErrorsWithData(bInner)
		for n := 0; n < bInner.N; n++ {
			fillMapFromStruct(&s1, &outputMap, AssertDataIsNotReplaced)
			// outputMap = nil
		}
	})

}
