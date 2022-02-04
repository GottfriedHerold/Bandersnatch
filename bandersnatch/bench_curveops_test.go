package bandersnatch

import (
	"testing"
)

const benchSizeCurvePoint = 1 << 8

//

// filterTypes_CompatibilityCond checks returns true unless receiverType can only represent subgroup elements, but some pointType among others cannot.
// Used to filter pointType combinations.
func filterTypes_CompatibilityCond(receiverType PointType, others ...PointType) bool {
	if typeCanOnlyRepresentSubgroup(receiverType) {
		for _, other := range others {
			if !typeCanOnlyRepresentSubgroup(other) {
				return false
			}
		}
	}
	return true
}

// filterPointTypes_SameSubgroup returns true iff either all PointTypes can only represent subgroup elements or none can.
// Used to filter pointType combinations.
func filterPointTypes_SameSubgroup(types ...PointType) bool {
	if len(types) == 0 {
		return true
	}
	match := typeCanOnlyRepresentSubgroup(types[0])
	for _, otherType := range types[1:] {
		if match != typeCanOnlyRepresentSubgroup(otherType) {
			return false
		}
	}
	return true
}

// For Copy&Pasting

/*
func BenchmarkCurveTemplate(bOuter *testing.B) {
	benchmarkForPointTypes(bOuter, benchSizeCurvePoint, func(b *testing.B, rec, x1, x2 []CurvePointPtrInterfaceTestSample) {
		for n := 0; n < b.N; n++ {
			//
		}
	}, "%[2]v,%[3]v->%[1]v", filterTypes_CompatibilityCond, allTestPointTypes, allTestPointTypes, allTestPointTypes)
}
*/

func BenchmarkAllCurveTypes_Neg(bOuter *testing.B) {
	// We need to clone the argument (or do some more complicated stuff), because
	// receiver.Neg(arg) may actually change (e.g. normalize to affine) the argument.
	bOuter.Logf("INFO: Benchmarking Negation includes a call to Clone()")
	benchmarkForPointTypes(bOuter, benchSizeCurvePoint,
		func(b *testing.B, receivers []CurvePointPtrInterfaceTestSample, points []CurvePointPtrInterfaceTestSample) {
			for n := 0; n < b.N; n++ {
				receivers[n%benchSizeCurvePoint].Neg(points[n%benchSizeCurvePoint].Clone())
			}
		}, "Neg(%[2]v)->%[1]v", filterPointTypes_SameSubgroup, allTestPointTypes, allTestPointTypes)
}

/*
func Benchmark1(bouter *testing.B) {
	callWithAllOptions(func(receiverType PointType, argType PointType) {
		tags := pointTypeToTag(argType) + "->" + pointTypeToTag(receiverType)
		initSliceWithTestPointType(DumpCurvePoint[:], receiverType)
		points1 := getPrecomputedCurvePointSlice(1, argType, benchSizeCurvePoint)
		bouter.Run(tags, func(b *testing.B) {
			prepareBenchmarkCurvePoints(b)
			for n := 0; n < b.N; n++ {
				DumpCurvePoint[n%benchSizeCurvePoint].Neg(points1[n%benchSizeCurvePoint])
			}
		})
	}, subgroupCompatibilityCond, allTestPointTypes, allTestPointTypes)
}

func Benchmark2(bOuter *testing.B) {
	bOuter.Run("ts->ss", func(b *testing.B) {
		var points1 [benchSizeCurvePoint]Point_xtw_subgroup
		initFromPrecomputedCurvePointSlice(points1[:], 1)
		prepareBenchmarkCurvePoints(b)
		for n := 0; n < b.N; n++ {
			DumpEFGH_subgroup[n%benchSizeCurvePoint].Neg(&points1[n%benchSizeCurvePoint])
		}
	})
}
*/

func BenchmarkAllCurveTypes_Clone(bOuter *testing.B) {
	benchmarkForPointTypes(bOuter, benchSizeCurvePoint, func(b *testing.B, inputs []CurvePointPtrInterfaceTestSample) {
		for n := 0; n < b.N; n++ {
			DumpCurvePoint[n%benchSizeCurvePoint] = inputs[n%benchSizeCurvePoint].Clone().(CurvePointPtrInterfaceTestSample)
		}
	}, "Clone(%v)", allTestPointTypes)
}

func BenchmarkAllCurveTypes_Endo(bOuter *testing.B) {
	benchmarkForPointTypes(bOuter, benchSizeCurvePoint, func(b *testing.B, receivers []CurvePointPtrInterfaceTestSample, inputs []CurvePointPtrInterfaceTestSample) {
		for n := 0; n < b.N; n++ {
			receivers[n%benchSizeCurvePoint].Endo(inputs[n%benchSizeCurvePoint])
		}
	}, "Endo(%[2]v)->%[1]v", filterPointTypes_SameSubgroup, allTestPointTypes, allTestPointTypes)

}

func BenchmarkAllCurveTypes_EndoEq(bOuter *testing.B) {
	benchmarkForPointTypes(bOuter, benchSizeCurvePoint, func(b *testing.B, receivers []CurvePointPtrInterfaceTestSample) {
		for n := 0; n < b.N; n++ {
			receivers[n%benchSizeCurvePoint].EndoEq()
		}
	}, "EndoEq(%v)", allTestPointTypes)
}

func BenchmarkAllCurveTypes_SetFrom(bOuter *testing.B) {
	// We need to clone the argument (or do some more complicated stuff), because
	// receiver.SetFrom(input) may actually change (e.g. normalize to affine) the argument.
	// Results would be completely wrong for conversion to affine.
	bOuter.Logf("INFO: Benchmarking Conversion via SetFrom includes a call to Clone()")
	benchmarkForPointTypes(bOuter, benchSizeCurvePoint, func(b *testing.B, receivers, inputs []CurvePointPtrInterfaceTestSample) {
		for n := 0; n < b.N; n++ {
			receivers[n%benchSizeCurvePoint].SetFrom(inputs[n%benchSizeCurvePoint].Clone())
		}
	}, "SetFrom(%[2]v)->%[1]v", filterTypes_CompatibilityCond, allTestPointTypes, allTestPointTypes)
}

func BenchmarkAllCurveTypes_SetFromSubgroupUntrusted(bOuter *testing.B) {
	// We need to clone the argument (or do some more complicated stuff), because
	// receiver.SetFromSubgroup(input, trust) may actually change (e.g. normalize to affine) the argument.
	// Results would be completely wrong for conversion to affine.
	bOuter.Logf("INFO: Benchmarking Conversion via SetFromSubgroup includes a call to Clone()")
	benchmarkForPointTypes(bOuter, benchSizeCurvePoint, func(b *testing.B, receivers, inputs []CurvePointPtrInterfaceTestSample) {
		for n := 0; n < b.N; n++ {
			_ = receivers[n%benchSizeCurvePoint].SetFromSubgroupPoint(inputs[n%benchSizeCurvePoint].Clone(), UntrustedInput)
		}
	}, "SetFromSubgroup(%[2]v,Untrusted)->%[1]v", allTestPointTypes, allTestPointTypes)
}

func BenchmarkAllCurveTypes_SetFromSubgroupTrusted(bOuter *testing.B) {
	// We need to clone the argument (or do some more complicated stuff), because
	// receiver.SetFromSubgroup(input, trust) may actually change (e.g. normalize to affine) the argument.
	// Results would be completely wrong for conversion to affine.
	bOuter.Logf("INFO: Benchmarking Conversion via SetFromSubgroup includes a call to Clone()")
	benchmarkForPointTypes(bOuter, benchSizeCurvePoint, func(b *testing.B, receivers, inputs []CurvePointPtrInterfaceTestSample) {
		for i := 0; i < benchSizeCurvePoint; i++ {
			inputs[i].DoubleEq()
		}
		resetBenchmarkCurvePoints(b)
		for n := 0; n < b.N; n++ {
			_ = receivers[n%benchSizeCurvePoint].SetFromSubgroupPoint(inputs[n%benchSizeCurvePoint].Clone(), TrustedInput)
		}
	}, "SetFromSubgroup(%[2]v,Trusted)->%[1]v", allTestPointTypes, allTestPointTypes)
}

func BenchmarkAllCurveTypes_Double(bOuter *testing.B) {
	benchmarkForPointTypes(bOuter, benchSizeCurvePoint, func(b *testing.B, receivers, inputs []CurvePointPtrInterfaceTestSample) {
		for n := 0; n < b.N; n++ {
			receivers[n%benchSizeCurvePoint].Double(inputs[n%benchSizeCurvePoint])
		}
	}, "Double(%[2]v)->%[1]v", allTestPointTypes, allTestPointTypes)
}

func BenchmarkAllCurveTypes_DoubleEq(bOuter *testing.B) {
	benchmarkForPointTypes(bOuter, benchSizeCurvePoint, func(b *testing.B, receivers []CurvePointPtrInterfaceTestSample) {
		for n := 0; n < b.N; n++ {
			receivers[n%benchSizeCurvePoint].DoubleEq()
		}
	}, "DoubleEq(%v)", allTestPointTypes)
}

func BenchmarkAllCurveTypes_Add(bOuter *testing.B) {
	benchmarkForPointTypes(bOuter, benchSizeCurvePoint, func(b *testing.B, rec, x1, x2 []CurvePointPtrInterfaceTestSample) {
		for n := 0; n < b.N; n++ {
			rec[n%benchSizeCurvePoint].Add(x1[n%benchSizeCurvePoint], x2[n%benchSizeCurvePoint])
		}
	}, "Add(%[2]v,%[3]v)->%[1]v", filterPointTypes_SameSubgroup, allTestPointTypes, allTestPointTypes, allTestPointTypes)
}

func BenchmarkAllCurveTypes_AddEq(bOuter *testing.B) {
	benchmarkForPointTypes(bOuter, benchSizeCurvePoint, func(b *testing.B, rec, x1 []CurvePointPtrInterfaceTestSample) {
		for n := 0; n < b.N; n++ {
			rec[n%benchSizeCurvePoint].AddEq(x1[n%benchSizeCurvePoint])
		}
	}, "AddEq(%[1]v,%[2]v)->%[1]v", filterPointTypes_SameSubgroup, allTestPointTypes, allTestPointTypes)
}

func BenchmarkAllCurveTypes_Sub(bOuter *testing.B) {
	benchmarkForPointTypes(bOuter, benchSizeCurvePoint, func(b *testing.B, rec, x1, x2 []CurvePointPtrInterfaceTestSample) {
		for n := 0; n < b.N; n++ {
			rec[n%benchSizeCurvePoint].Sub(x1[n%benchSizeCurvePoint], x2[n%benchSizeCurvePoint])
		}
	}, "Sub(%[2]v,%[3]v)->%[1]v", filterPointTypes_SameSubgroup, allTestPointTypes, allTestPointTypes, allTestPointTypes)
}

func BenchmarkAllCurveTypes_SubEq(bOuter *testing.B) {
	benchmarkForPointTypes(bOuter, benchSizeCurvePoint, func(b *testing.B, rec, x1 []CurvePointPtrInterfaceTestSample) {
		for n := 0; n < b.N; n++ {
			rec[n%benchSizeCurvePoint].SubEq(x1[n%benchSizeCurvePoint])
		}
	}, "SubEq(%[1]v,%[2]v)->%[1]v", filterPointTypes_SameSubgroup, allTestPointTypes, allTestPointTypes)
}
