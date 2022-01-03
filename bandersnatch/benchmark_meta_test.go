package bandersnatch

import (
	"math/rand"
	"reflect"
	"testing"
)

// This file contains code that is shared by a lot of benchmarking code
// such as setup and/or teardown code as well as integrating call counters into go's
// default benchmarking framework.

// benchmarks read from and write to these global variable in order to
// a) allow easy initialization outside of the measured function
// b) avoid the compiler optimizating away the writes.
const benchS = 2 << 8

// benchmark functions write to DumpXXX variables.
// These are "exported" (in non-test builds this file is ignored) to prevent the compiler from optimizations
// based on the fact that they are never read from within the bandersnatch module
var DumpBools [benchS]bool
var DumpFe_64 [benchS]bsFieldElement_64
var DumpFe_8 [benchS]bsFieldElement_8
var DumpXTW [benchS]Point_xtw
var DumpAXTW [benchS]Point_axtw
var DumpEFGH [benchS]Point_efgh

// Calling DumpCPI_<foo>[i].Add(...) will only work if
// DumpCPI_<foo>[i] is non-nil (i.e. has a concrete type to select the actual CurvePoint type)
// AND the contained pointer value DumpCPI_<foo>[i].(*foo) of type *foo is non-nil.
// These are initialized in init().
// plain DumpCPI is (intentionally) uninitialized and needs to be set to a concrete type *after* prepare_<foo>
var DumpCPI_XTW, DumpCPI_AXTW, DumpCPI_EFGH, DumpCPI [benchS]CurvePointPtrInterfaceWrite_FullCurve

var bench_x_64 [benchS]bsFieldElement_64
var bench_y_64 [benchS]bsFieldElement_64
var bench_z_64 [benchS]bsFieldElement_64
var bench_x_8 [benchS]bsFieldElement_8
var bench_y_8 [benchS]bsFieldElement_8
var bench_z_8 [benchS]bsFieldElement_8

// Need to call PrepareBenchInterfaces *after* calling prepareBenchTests_Curve
var bench_CPI1 [benchS]CurvePointPtrInterface_FullCurve
var bench_CPI2 [benchS]CurvePointPtrInterface_FullCurve
var bench_CPI3 [benchS]CurvePointPtrInterface_FullCurve

var bench_xtw1, bench_xtw2 [benchS]Point_xtw
var bench_axtw1, bench_axtw2 [benchS]Point_axtw
var bench_efgh1, bench_efgh2 [benchS]Point_efgh

func init() {
	var points_xtw [benchS]Point_xtw
	var points_axtw [benchS]Point_axtw
	var points_efgh [benchS]Point_efgh

	for i := 0; i < benchS; i++ {
		DumpCPI_XTW[i] = &points_xtw[i]
		DumpCPI_AXTW[i] = &points_axtw[i]
		DumpCPI_EFGH[i] = &points_efgh[i]
	}
}

func prepareBenchTests_8(b *testing.B) {
	var drng *rand.Rand = rand.New(rand.NewSource(666))
	for i := 0; i < benchS; i++ {
		bench_x_8[i].setRandomUnsafe(drng)
		bench_y_8[i].setRandomUnsafe(drng)
		bench_z_8[i].setRandomUnsafe(drng)
	}
	ResetCallCounters()
	b.ResetTimer()
}

func prepareBenchTests_64(b *testing.B) {
	var drng *rand.Rand = rand.New(rand.NewSource(666)) // same number as in prepareBenchTests_8 by design
	for i := 0; i < benchS; i++ {
		bench_x_64[i].setRandomUnsafe(drng)
		bench_y_64[i].setRandomUnsafe(drng)
		bench_z_64[i].setRandomUnsafe(drng)
	}
	ResetCallCounters()
	b.ResetTimer()
}

var consistentRandom667xtw [benchS]Point_xtw
var consistentRandom667xtw_initialized bool = false

func prepareBenchInterfaces(b *testing.B, target *[benchS]CurvePointPtrInterface_FullCurve, pointType PointType) {
	// only do this once, and only if needed. It's slow.
	if !consistentRandom667xtw_initialized {
		var rng *rand.Rand = rand.New(rand.NewSource(667))
		for i := 0; i < benchS; i++ {
			consistentRandom667xtw[i] = makeRandomPointInSubgroup_t(rng)
		}
		consistentRandom667xtw_initialized = true
	}

	// values is now a reflect.Value encapsulating a pointer to a [benchS]Point_<foo>
	values := reflect.New(reflect.ArrayOf(benchS, pointType.Elem()))

	for i := 0; i < benchS; i++ {
		(*target)[i] = values.Elem().Index(i).Addr().Interface().(CurvePointPtrInterface_FullCurve)
		(*target)[i].SetFrom(&consistentRandom667xtw[i])
	}
	ResetCallCounters()
	b.ResetTimer()
}

func prepareDumpCPI(b *testing.B, pointType PointType) {
	values := reflect.New(reflect.ArrayOf(benchS, pointType.Elem()))

	for i := 0; i < benchS; i++ {
		DumpCPI[i] = values.Elem().Index(i).Addr().Interface().(CurvePointPtrInterfaceWrite_FullCurve)
	}
	ResetCallCounters()
	b.ResetTimer()
}

var consistentRandomPoints668 [benchS]struct {
	xtw1, xtw2   Point_xtw
	axtw1, axtw2 Point_axtw
	efgh1, efgh2 Point_efgh
}
var consistentRandomPoints668initialized bool = false

func prepareBenchTest_Curve(b *testing.B) {
	// make a bunch of random points. We store them so we do not have to recompute them
	// if this function called again. makeRandomPointInSubgroup is *SLOW*
	if !consistentRandomPoints668initialized {
		var rng *rand.Rand = rand.New(rand.NewSource(668))
		for i := 0; i < benchS; i++ {
			consistentRandomPoints668[i].xtw1 = makeRandomPointInSubgroup_t(rng)
			consistentRandomPoints668[i].xtw2 = makeRandomPointInSubgroup_t(rng)
			consistentRandomPoints668[i].axtw1 = makeRandomPointInSubgroup_a(rng)
			consistentRandomPoints668[i].axtw2 = makeRandomPointInSubgroup_a(rng)
			consistentRandomPoints668[i].efgh1 = makeRandomPointInSubgroup_s(rng)
			consistentRandomPoints668[i].efgh2 = makeRandomPointInSubgroup_s(rng)
		}
		consistentRandomPoints668initialized = true
	}
	for i := 0; i < benchS; i++ {
		bench_CPI1[i] = nil
		bench_CPI2[i] = nil
		bench_CPI3[i] = nil
		bench_xtw1[i] = consistentRandomPoints668[i].xtw1
		bench_xtw2[i] = consistentRandomPoints668[i].xtw2
		bench_axtw1[i] = consistentRandomPoints668[i].axtw1
		bench_axtw2[i] = consistentRandomPoints668[i].axtw2
		bench_efgh1[i] = consistentRandomPoints668[i].efgh1
		bench_efgh2[i] = consistentRandomPoints668[i].efgh2
	}
	ResetCallCounters()
	b.ResetTimer()
}

func benchmarkForAllPointTypesNoneary(b *testing.B, receiverTypes []PointType, fun func(*testing.B)) {
	prepareBenchTest_Curve(b)
	for _, receiverType := range receiverTypes {
		prepareDumpCPI(b, receiverType)
		var tag string = PointTypeToTag(receiverType)
		b.Run(tag, fun)
	}
}

func benchmarkForAllPointTypesUnary(b *testing.B, receiverTypes []PointType, arg1Types []PointType, fun func(*testing.B)) {
	prepareBenchTest_Curve(b)
	for _, receiverType := range receiverTypes {
		prepareDumpCPI(b, receiverType)
		for _, arg1Type := range arg1Types {
			prepareBenchInterfaces(b, &bench_CPI1, arg1Type)
			var tag string = PointTypeToTag(arg1Type) + "->" + PointTypeToTag(receiverType)
			b.Run(tag, fun)
		}
	}
}

func benchmarkForAllPointTypesBinary(b *testing.B, receiverTypes []PointType, arg1Types []PointType, arg2Types []PointType, fun func(*testing.B)) {
	prepareBenchTest_Curve(b)
	for _, receiverType := range receiverTypes {
		prepareDumpCPI(b, receiverType)
		for _, arg1Type := range arg1Types {
			prepareBenchInterfaces(b, &bench_CPI1, arg1Type)
			for _, arg2Type := range arg2Types {
				prepareBenchInterfaces(b, &bench_CPI2, arg2Type)
				var tag string = PointTypeToTag(arg1Type) + PointTypeToTag(arg2Type) + "->" + PointTypeToTag(receiverType)
				b.Run(tag, fun)
			}

		}
	}
}

func benchmarkForAllPointTypesBinaryCommutative(b *testing.B, receiverTypes []PointType, argTypes []PointType, fun func(*testing.B)) {
	prepareBenchTest_Curve(b)
	for _, receiverType := range receiverTypes {
		prepareDumpCPI(b, receiverType)
		for i, arg1Type := range argTypes {
			prepareBenchInterfaces(b, &bench_CPI1, arg1Type)
			for j, arg2Type := range argTypes {
				if i > j {
					continue
				}
				prepareBenchInterfaces(b, &bench_CPI2, arg2Type)
				var tag string = PointTypeToTag(arg1Type) + PointTypeToTag(arg2Type) + "->" + PointTypeToTag(receiverType)
				b.Run(tag, fun)
			}

		}
	}
}
