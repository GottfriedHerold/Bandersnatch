//go:build ignore

package bandersnatch

import (
	"testing"
)

func BenchmarkCurveNegUntyped(b *testing.B) {
	benchmarkForAllPointTypesUnary(b, []PointType{pointTypeXTW, pointTypeEFGH}, allTestPointTypes, func(b *testing.B) {
		for n := 0; n < b.N; n++ {
			DumpCPI[n%benchS].Neg(bench_CPI1[n%benchS])
		}
	})
}

func BenchmarkCurveClone(b *testing.B) {
	prepareBenchTest_Curve(b)
	for _, receiverType := range allTestPointTypes {
		prepareBenchInterfaces(b, &bench_CPI1, receiverType, 1)
		var tag string = PointTypeToTag(receiverType)
		b.Run(tag, func(b *testing.B) {
			setupBenchmarkCurvePoints(b)
			for n := 0; n < b.N; n++ {
				_ = bench_CPI1[n%benchS].Clone()
			}
		})
	}
}

func BenchmarkCurveEndoUntyped(b *testing.B) {
	benchmarkForAllPointTypesUnary(b, allTestPointTypes, allTestPointTypes, func(b *testing.B) {
		for n := 0; n < b.N; n++ {
			DumpCPI[n%benchS].Endo(bench_CPI1[n%benchS])
		}
	})
}

func BenchmarkCurveConversionUntyped(b *testing.B) {
	benchmarkForAllPointTypesUnary(b, []PointType{pointTypeXTW, pointTypeEFGH}, allTestPointTypes, func(b *testing.B) {
		for n := 0; n < b.N; n++ {
			DumpCPI[n%benchS].SetFrom(bench_CPI1[n%benchS])
		}
	})
	prepareBenchTest_Curve(b)
	for _, argType := range allTestPointTypes {
		prepareBenchInterfaces(b, &bench_CPI1, argType, 1)
		b.Run(PointTypeToTag(argType)+"(cloned)->a", func(b *testing.B) {
			setupBenchmarkCurvePoints(b)
			for n := 0; n < b.N; n++ {
				DumpAXTW[n%benchS].SetFrom(bench_CPI1[n%benchS].Clone())
			}
		})
	}
}
