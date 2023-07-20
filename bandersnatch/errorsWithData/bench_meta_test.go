package errorsWithData

import "testing"

const benchS = 256

var Dump_ParamMaps [benchS]ParamMap // exported to avoid compiler optimizations
var Dump_TokenList [benchS]tokenList

func prepareBenchmarkErrorsWithData(b *testing.B) {
	for i := 0; i < benchS; i++ {
		Dump_ParamMaps[i] = make(ParamMap)
		Dump_TokenList[i] = nil
	}
	b.Cleanup(func() {
		for i := 0; i < benchS; i++ {
			Dump_ParamMaps[i] = make(ParamMap)
			Dump_TokenList[i] = nil
		}
	})
	b.ResetTimer()
}
