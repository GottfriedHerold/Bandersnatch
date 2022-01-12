//go:build ignore

package bandersnatch

import (
	"math/rand"
	"reflect"
	"testing"

	"github.com/GottfriedHerold/Bandersnatch/internal/callcounters"
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

// NOTE: The various prepareBench... functions
// contain common setup functionality for benchmarking. They are all supposed to be run before any actual benchmarking starts.
// (In fact, they automatically reset all benchmark timers)
// They set up the global variables that benchmarks consume and reset the benchmarking timers and call counters.
// If using sub-benchmarks with b.Run(...), the prepareBench... functions can generally be run in the outer benchmark.
// However, the inner sub-benchmarks might still need to reset call counters; also rerunning prepareBench... is
// generally safer as it ensures that global variables are restored. (Note that functions might inadvertedly change those)

// For the teardown code (which atm only makes sure that call counters are added to the benchmarking output if those are active in the current build),
// this needs to be called *inside the sub-test*.

// For benchmarking curve point functionalities,
// it is suggested to use setupBenchmarkCurvePoints(b) INSIDE each innermost sub-test.
// Be aware that go's testing/benchmarking framework runs the benchmark several times with
// varying values of b.N until the statistics stabilize -- In particular, call counters need to reset upon retrying with a different b.N.
// setupBenchmarkCurvePoints(b) takes care of that

// prepareBenchTests_8 runs common setup code for benchmarks for the naive 8-bit field element operations
func prepareBenchTests_8(b *testing.B) {
	var drng *rand.Rand = rand.New(rand.NewSource(666))
	for i := 0; i < benchS; i++ {
		bench_x_8[i].setRandomUnsafe(drng)
		bench_y_8[i].setRandomUnsafe(drng)
		bench_z_8[i].setRandomUnsafe(drng)
	}
	callcounters.ResetAllCounters()
	b.ResetTimer()
}

// prepareBenchTests_64 runs common setup code for benchmarks of the field element implementation.
func prepareBenchTests_64(b *testing.B) {
	var drng *rand.Rand = rand.New(rand.NewSource(666)) // same number as in prepareBenchTests_8 by design
	for i := 0; i < benchS; i++ {
		bench_x_64[i].setRandomUnsafe(drng)
		bench_y_64[i].setRandomUnsafe(drng)
		bench_z_64[i].setRandomUnsafe(drng)
	}
	callcounters.ResetAllCounters()
	b.ResetTimer()
}

// sampling (a large number of) random curve points for benchmarking is
// quite slow. So we create those once and for all. Then our
// actual setup code just copies those to a place where they can be used.
// (Note: we run a prepareBench... function in each benchmark that does that copying.
// The reason we don't just run a global setup inside some init() is that
// we want to keep benchmarks independent: Some benchmarks might actually silently change those benchmark samples --
// either due to bugs, which would be hard to find or because some "read-only" functions actually
// change the internal representation of the objects -- both field elements and curve points in projective coordinate
// do not have a unique internal represenation, after all )

// these variable is used to retain a copy of the random points created for benchmarking.
// This allows to restore them to a consistent value quickly.
// 667 resp. 668 is the (arbitrary, but fixed for reproducibility) seed used in the sampling.

// Number of indenpendent sample families we keep around
const consistentRandom667samples = 4

var (
	consistentRandom667xtw             [consistentRandom667samples][benchS]Point_xtw
	consistentRandom667xtw_initialized bool = false
)

var consistentRandomPoints668 [benchS]struct {
	xtw1, xtw2   Point_xtw
	axtw1, axtw2 Point_axtw
	efgh1, efgh2 Point_efgh
}
var consistentRandomPoints668initialized bool = false

// makeConsistentRandom667xtwAvailalbe initializes consistentRandom667xtw
// We do not do this as part of some init(), because some benchmarks do not need it.
// (in particular, if one only wants to benchmarks the field element arithmetic)
func makeConsistentRandom667xtwAvailiable() {
	if !consistentRandom667xtw_initialized {
		var rng *rand.Rand = rand.New(rand.NewSource(667))
		for s := 0; s < consistentRandom667samples; s++ {
			for i := 0; i < benchS; i++ {
				consistentRandom667xtw[s][i] = makeRandomPointInSubgroup_t(rng)
			}
		}
		consistentRandom667xtw_initialized = true
	}

}

// makeConsistentRandom668Available initializes consistentRandomPoints668, similar to makeConsistentRandom667xtwAvailiable
func makeConsistentRandom668Available() {
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
}

// prepareBenchInterfaces fills target (supposed to be one of the bench_CPI<i> with consistent random values of type pointType)
// family denotes an integer between 0 and consistentRandom667 samples that is used to differentiate different sets of random points.
// (calling the function with the same value for familiy will give identical values, which might not be wanted in some circumstances)
// This function is supposed to be called at the beginning of a benchmark before any actual benchmarking starts.
func prepareBenchInterfaces(b *testing.B, target *[benchS]CurvePointPtrInterface_FullCurve, pointType PointType, family int) {
	if family >= consistentRandom667samples {
		panic("suffix number too large")
	}
	makeConsistentRandom667xtwAvailiable()

	// values is now a reflect.Value encapsulating a pointer to a [benchS]Point_<foo>
	values := reflect.New(reflect.ArrayOf(benchS, pointType.Elem()))

	for i := 0; i < benchS; i++ {
		(*target)[i] = values.Elem().Index(i).Addr().Interface().(CurvePointPtrInterface_FullCurve)
		(*target)[i].SetFrom(consistentRandom667xtw[family][i].Clone())
	}
	callcounters.ResetAllCounters()
	b.ResetTimer()
}

// prepareDumpCPI fills DumpCPI (which acts as receiver in benchmarking, where the computed values are dumped to)) to be filled
// with values of concrete type pointType.
// This function is supposed to be called at the beginning of a benchmark before any actual benchmarking starts.
func prepareDumpCPI(b *testing.B, pointType PointType) {
	values := reflect.New(reflect.ArrayOf(benchS, pointType.Elem()))

	for i := 0; i < benchS; i++ {
		DumpCPI[i] = values.Elem().Index(i).Addr().Interface().(CurvePointPtrInterfaceWrite_FullCurve)
	}
	callcounters.ResetAllCounters()
	b.ResetTimer()
}

// prepareBenchTest_Curve runs setup code used for benchmarking curve points.
// This code should be run before *any* curve point benchmark.
// Note that you might need to run additional setup routines depending on the actual benchmark.
func prepareBenchTest_Curve(b *testing.B) {
	// make a bunch of reproducible random points.
	makeConsistentRandom668Available()
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
	callcounters.ResetAllCounters()
	b.ResetTimer()
}

// postProcessBenchmarkCurvePoints should be called at the end of each sub-test (preferably using b.Cleanup(...) )
// Currently, the only thing it does is make sure call counters are included in the benchmark if the current build includes them
func postProcessBenchmarkCurvePoints(b *testing.B) {
	BenchmarkWithCallCounters(b)
}

// setupBenchmarkCurvePoints resets all timers and call counters and makes sure call counters are included in the benchmark if available.
// This function only makes sense if called in inner sub-tests.
func setupBenchmarkCurvePoints(b *testing.B) {
	callcounters.ResetAllCounters()
	b.Cleanup(func() { postProcessBenchmarkCurvePoints(b) })
	b.ResetTimer()
}

// benchmarkForAllPointTypesNoneary runs a given "noneary" benchmark function for multiple point types.
// noneary here means that the actual function is run multiple times with Dump_CPI set multiple point types.
// (i.e. the function to be benchmarked has the form Dump_CPI[i].some_fun() where some_fun has 0 non-receiver arguments.
func benchmarkForAllPointTypesNoneary(b *testing.B, receiverTypes []PointType, fun func(*testing.B)) {
	f := func(b *testing.B) {
		setupBenchmarkCurvePoints(b)
		fun(b)
	}
	prepareBenchTest_Curve(b)
	for _, receiverType := range receiverTypes {
		prepareDumpCPI(b, receiverType)
		var tag string = PointTypeToTag(receiverType)
		b.Run(tag, f)
	}
}

// benchmarkForAllPointTypesUnary runs a given benchmark for function for multiple point types.
// Unary here means that the function will be called with the global variables Dump_CPI and bench_CPI1 set to various point Types.
// (i.e the function to be benchmarked probably has the form Dump_CPI[i].some_fun(arg1) with 1 non-receiver argument)
func benchmarkForAllPointTypesUnary(b *testing.B, receiverTypes []PointType, arg1Types []PointType, fun func(*testing.B)) {
	f := func(b *testing.B) {
		setupBenchmarkCurvePoints(b)
		fun(b)
	}
	prepareBenchTest_Curve(b)
	for _, receiverType := range receiverTypes {
		prepareDumpCPI(b, receiverType)
		for _, arg1Type := range arg1Types {
			prepareBenchInterfaces(b, &bench_CPI1, arg1Type, 1)
			var tag string = PointTypeToTag(arg1Type) + "->" + PointTypeToTag(receiverType)
			b.Run(tag, f)
		}
	}
}

// benchmarkForAllPointTypesBinary runs a given benchmark for function for multiple point types.
// Binary here means that the function will be called with the global variables Dump_CPI and bench_CPI1, bench_CPI2 set to various point Types.
// (i.e the function to be benchmarked probably has the form Dump_CPI[i].some_fun(arg1, arg2) with 2 non-receiver argument)
func benchmarkForAllPointTypesBinary(b *testing.B, receiverTypes []PointType, arg1Types []PointType, arg2Types []PointType, fun func(*testing.B)) {
	f := func(b *testing.B) {
		setupBenchmarkCurvePoints(b)
		fun(b)
	}
	prepareBenchTest_Curve(b)
	for _, receiverType := range receiverTypes {
		prepareDumpCPI(b, receiverType)
		for _, arg1Type := range arg1Types {
			prepareBenchInterfaces(b, &bench_CPI1, arg1Type, 2)
			for _, arg2Type := range arg2Types {
				prepareBenchInterfaces(b, &bench_CPI2, arg2Type, 3)
				var tag string = PointTypeToTag(arg1Type) + PointTypeToTag(arg2Type) + "->" + PointTypeToTag(receiverType)
				b.Run(tag, f)
			}
		}
	}
}

// benchmarkForAllPointTypesBinaryCommutative runs a given benchmark for function for multiple point types.
// Binary here means that the function will be called with the global variables Dump_CPI and bench_CPI1, bench_CPI2 set to various point Types.
// (i.e the function to be benchmarked probably has the form Dump_CPI[i].some_fun(arg1, arg2) with 2 non-receiver argument)
// Being commutative means that it will not run *both* bench_CPI1 = a, bench_CPI2 = b and bench_CPI1=b, bench_CPI = a for a != b.
// (this is to not clog up the output for functions where the dispatch takes care of this anyway.)
func benchmarkForAllPointTypesBinaryCommutative(b *testing.B, receiverTypes []PointType, argTypes []PointType, fun func(*testing.B)) {
	f := func(b *testing.B) {
		setupBenchmarkCurvePoints(b)
		fun(b)
	}
	prepareBenchTest_Curve(b)
	for _, receiverType := range receiverTypes {
		prepareDumpCPI(b, receiverType)
		for i, arg1Type := range argTypes {
			prepareBenchInterfaces(b, &bench_CPI1, arg1Type, 0)
			for j, arg2Type := range argTypes {
				if i > j {
					continue
				}
				prepareBenchInterfaces(b, &bench_CPI2, arg2Type, 2)
				var tag string = PointTypeToTag(arg1Type) + PointTypeToTag(arg2Type) + "->" + PointTypeToTag(receiverType)
				b.Run(tag, f)
			}
		}
	}
}
