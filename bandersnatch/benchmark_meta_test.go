package bandersnatch

import (
	"fmt"
	"math/rand"
	"reflect"
	"testing"

	"github.com/GottfriedHerold/Bandersnatch/internal/callcounters"
	"github.com/GottfriedHerold/Bandersnatch/internal/testutils"
)

const dumpSizeBench_curve = 1 << 8

// benchmark functions write to DumpXXX variables.
// These are "exported" (well, sort of: in non-test builds this file is ignored) to disabuse the compiler from potential optimizations
// based on the fact that they are never read from within the bandersnatch module

var DumpBools_curve [dumpSizeBench_curve]bool
var DumpXTW_full [dumpSizeBench_curve]Point_xtw_full
var DumpXTW_subgroup [dumpSizeBench_curve]Point_xtw_subgroup
var DumpAXTW_full [dumpSizeBench_curve]Point_axtw_full
var DumpAXTW_subgroup [dumpSizeBench_curve]Point_axtw_subgroup
var DumpEFGH_full [dumpSizeBench_curve]Point_efgh_full
var DumpEFGH_subgroup [dumpSizeBench_curve]Point_efgh_subgroup
var DumpCurvePoint [dumpSizeBench_curve]CurvePointPtrInterfaceTestSample

// currently unused

// makeZeroInitializedPointsFromPointType creates a slice of the given size of pointers to zero-initialized curve Points from a PointType.
func makeZeroInitializedPointsFromPointType(amount int, whichType PointType) (ret []CurvePointPtrInterfaceTestSample) {
	_, ok := makeCurvePointPtrInterface(whichType).(CurvePointPtrInterfaceTestSample)
	if !ok {
		panic("makePointsFromPointType only works with types T where pointers-to-T satisfy the CurvePointPtrInterfaceTestSample interface")
	}
	ret = make([]CurvePointPtrInterfaceTestSample, amount)
	for i := 0; i < amount; i++ {
		ret[i] = makeCurvePointPtrInterface(whichType).(CurvePointPtrInterfaceTestSample)
	}
	return
}

// Creating pseudo-reandom curve points (such as for the sake of benchmarking) is extremely slow due to the need to compute a square root per point.
// For that reason, we cache the points such that when running the full benchmark suite, at least we only have to do that once.
// The cache is per point-type and per seed value.

type (
	// pseudoRandomCurvePointCache holds a cache of CurvePoints (via pointers stored in CurvePointPtrInterfaceTestSample's) that are computed only once.
	pseudoRandomCurvePointCache struct {
		rng       *rand.Rand                         // rng state. This is used to grow the cache on demand
		elements  []CurvePointPtrInterfaceTestSample // the actual elements stored in the cache
		pointType PointType                          // the PointType of the elements. We store it once here for simplicity. For len(elemenets) > 0, this can be recovered from any elements[i]
	}

	// pseudoRandomCurvePointCacheKey is the struct used as map key for our caches (we maintain an independent cache per seed and pointType)
	pseudoRandomCurvePointCacheKey struct {
		pointType PointType
		seed      int64
	}
)

// TODO: Protect with mutex

// cachedPseudoRandomCurvePoints holds a cache of Curve Points per pointType and seed.
// Note that the map is to pointers of struct. (This extra indirection actually is probably unneeded, because the elements of the pseudoRandomCurvePointCache struct that actually change have shallow copy behaviour)
// Note that no map entry is nil.
var cachedPseudoRandomCurvePoints map[pseudoRandomCurvePointCacheKey]*pseudoRandomCurvePointCache = make(map[pseudoRandomCurvePointCacheKey]*pseudoRandomCurvePointCache)

// getElements retrieves amount many actual curve points from the cache *as a copy*, growing the cache as needed.
//
// When calling this with values amount1 < amount2 (in any order), the returned slice for amount1 will be a (copy of) an initial segment of the one for amount2.
func (pc *pseudoRandomCurvePointCache) getElements(amount int) (ret []CurvePointPtrInterfaceTestSample) {
	// prpare memory to hold the result
	ret = make([]CurvePointPtrInterfaceTestSample, amount)
	if amount == 0 {
		return
	}
	// We assume the pseudoRandomCurvePointCache is already initialized. We only put initialized ones into the global map.
	testutils.Assert(pc.rng != nil)
	testutils.Assert(pc.elements != nil)
	currentLen := len(pc.elements)
	// grow the cache as needed:
	if amount > currentLen {
		for i := 0; i < amount-currentLen; i++ {
			// Type assertion is guaranteed to be ok. This is checked when creating pc with newPrecomputedCurvePointSlice.
			temp := makeCurvePointPtrInterface(pc.pointType).(CurvePointPtrInterfaceTestSample)
			temp.sampleRandomUnsafe(pc.rng)
			pc.elements = append(pc.elements, temp)
		}
	}
	testutils.Assert(len(pc.elements) >= amount)
	// Fill ret with clones of the stored elements.
	for i := 0; i < amount; i++ {
		// Again, type assertion is guaranteed to be ok. This is checked when creating pc with newPrecomputedCurvePointSlice.
		ret[i] = pc.elements[i].Clone().(CurvePointPtrInterfaceTestSample)
	}
	return
}

// newPrecomputedCurvePointSlice creates a new (pointer-to-) pseudoRandomCurvePointCache.
//
// This function is used to initialize our caches for precomputed benchmark samples.
func newPrecomputedCurvePointSlice(seed int64, pointType PointType) (pc *pseudoRandomCurvePointCache) {
	// We asserts that pointType satisfied CurvePointPtrInterfaceTestSample. We panic on error.
	_, ok := makeCurvePointPtrInterface(pointType).(CurvePointPtrInterfaceTestSample)
	if !ok {
		panic("bandersnatch / benchmarking framework: Trying to construct precomputed sample cache for PointType that satisfies the CurvePointPtrInterface interface, but not CurvePointPtrInterfaceTestSample.")
	}
	pc = new(pseudoRandomCurvePointCache)
	pc.rng = rand.New(rand.NewSource(seed))
	pc.elements = make([]CurvePointPtrInterfaceTestSample, 0, 256)
	pc.pointType = pointType
	return
}

// getPrecomputedCurvePointSlice samples amount many curve points of the given pointType with given seed.
//
// We use caching to speed up multiple calls to this function with the same (seed, pointType) pairs.
// Note that while the returned elements are slices of pointers, multiple calls return pointers to independent copies.
func getPrecomputedCurvePointSlice(seed int64, pointType PointType, amount int) []CurvePointPtrInterfaceTestSample {
	mapKey := pseudoRandomCurvePointCacheKey{seed: seed, pointType: pointType}
	pc := cachedPseudoRandomCurvePoints[mapKey]
	// pc == nil means that mapKey was not present in the map yet. In that case, we add it.
	// Note that pc is a pointer, so pc.getElements(amount) actually grows the cache as desired (and not just in a copy). This would also work if not a pointer, actually.
	if pc == nil {
		pc = newPrecomputedCurvePointSlice(seed, pointType)
		cachedPseudoRandomCurvePoints[mapKey] = pc
	}
	return pc.getElements(amount)
}

// currently unused:

// initFromPrecomputedCurvePointSlice is equivalent to calling getPrecomputedCurvePointSlice and storing the result in writeTo.
// writeTo must be a slice of a concrete point type implementing CurvePointPtrInterfaceTestSample.
// The pointType and amount args to getPrecomputedCurvePointSlice are derived from writeTo.
func initFromPrecomputedCurvePointSlice(writeTo interface{}, seed int64) {
	writeToReflected := reflect.ValueOf(writeTo)
	if writeToReflected.Kind() != reflect.Slice {
		panic("initFromPrecomputedCurvePointSlice must be called with slice argument")
	}
	inputLength := writeToReflected.Len()
	pointType := writeToReflected.Type().Elem()
	// This is mostly to abort on slice of interface instead of slice of concrete type.
	if pointType.Kind() != reflect.Struct {
		panic("initFromPrecomputeCurvePointSlice must be called with a slice of a concrete struct")
	}
	pointPtrType := reflect.PtrTo(pointType)
	// targetInterface is just a reflect.Type for CurvePointPtrInterfaceTestSample.
	// The roundabout way is needed because a direct callt to reflect.TypeOf would try to reach into and get the concrete type contained inside a variable of interface type.
	// TODO: Should we make this a global constant?
	var targetInterface reflect.Type = reflect.TypeOf((*CurvePointPtrInterfaceTestSample)(nil)).Elem()
	if !pointPtrType.Implements(targetInterface) {
		panic("initFromPrecomputedCurvePointSlice must be called with a slice of a concrete type that implements CurvePointPtrInterfaceTestSample")
	}
	// We could this more efficiently with less copying, but we don't care.
	outputSlice := getPrecomputedCurvePointSlice(seed, pointPtrType, inputLength)
	for i := 0; i < inputLength; i++ {
		curvePointPtr := outputSlice[i]
		newPointReflected := reflect.ValueOf(curvePointPtr)
		writeTarget := writeToReflected.Index(i)
		testutils.Assert(writeTarget.CanSet())
		writeTarget.Set(newPointReflected.Elem())
	}
}

// currently unused:

// initSliceWithTestPointType initializes a slice of CurvePointPtrInterfaceTestSample with (pointers to) fresh zero-initialzed actual points of the given type.
func initSliceWithTestPointType(writeTo []CurvePointPtrInterfaceTestSample, pointType PointType) {
	for i := 0; i < len(writeTo); i++ {
		writeTo[i] = makeCurvePointPtrInterface(pointType).(CurvePointPtrInterfaceTestSample)
	}
}

// Note: We have versions of prepareBenchmark, postProcessBenchmark and resetBenchmark for field elements and CurvePoints.
// At the moment, these are identical.
// The reason is that we might eventually move everything field element-related to a sub-module

// prepareBenchmarkCurvePoints runs some setup code and should be called in every (sub-)test before the actual code that is to be benchmarked.
// Note that it resets all counters.
func prepareBenchmarkCurvePoints(b *testing.B) {
	b.Cleanup(func() { postProcessBenchmarkCurvePoints(b) })
	resetBenchmarkCurvePoints(b)
}

// postProcessBenchmarkCurvePoints should be called at the end of each sub-test (preferably using b.Cleanup(...) )
// Currently, the only thing it does is make sure call counters are included in the benchmark if the current build includes them
// Calling prepareBenchmarkCurvePoints at the beginning of the benchmark takes care of it.
func postProcessBenchmarkCurvePoints(b *testing.B) {
	BenchmarkWithCallCounters(b)
}

// resetBenchmarkCurvePoints resets the benchmark counters; this should be called after any expensive setup that we do not want to include in the benchmark.
func resetBenchmarkCurvePoints(b *testing.B) {
	callcounters.ResetAllCounters()
	b.ResetTimer()
}

// currently unused in favor of benchmarkForPointTypes

// callWithAllOptions(f, [condition, ] arg1, arg2, ...) takes a function f, an optional condition and an arbitrary number of arg_i's.
// Each arg_i needs to be an array or slice. It then calls f(x_1, x2,...) for each tuple (x1, x2, ...) where x_i is from arg_i, provided
// condtion(x_1, x_2, ...) returns true.
func callWithAllOptions(fun interface{}, args ...interface{}) {
	funReflected := reflect.ValueOf(fun)
	testutils.Assert(funReflected.Kind() == reflect.Func, "callWithAllOptions's first argument must be a function")
	// funType := funReflected.Type()
	var haveCond bool = false
	var condVariadic bool = false
	var condFun reflect.Value
	startIndex := 0
	argLen := len(args)
	if (len(args) >= 1) && reflect.ValueOf(args[0]).Kind() == reflect.Func {
		haveCond = true
		startIndex = 1
		argLen--
		condFun = reflect.ValueOf(args[0])
		condVariadic = condFun.Type().IsVariadic()
		if !condVariadic {
			testutils.Assert(condFun.Type().NumIn() == argLen, "condition function has mismatch in number of arguments")
		}
	}
	var numCalls uint64 = 1
	for i := startIndex; i < len(args); i++ {
		argReflected := reflect.ValueOf(args[i])
		testutils.Assert(argReflected.Kind() == reflect.Slice || argReflected.Kind() == reflect.Array, "non-condition arguments to callWithAllOptions must be slices or arrays")
		numCalls *= uint64(argReflected.Len())
	}
	for callId := uint64(0); callId < numCalls; callId++ {
		var inputArgs []reflect.Value = make([]reflect.Value, argLen)
		r := callId
		for i := 0; i < argLen; i++ {
			currentSlice := reflect.ValueOf(args[i+startIndex])
			L := currentSlice.Len()
			currentIndex := r % uint64(L)
			inputArgs[i] = currentSlice.Index(int(currentIndex))
			r -= currentIndex
			r /= uint64(L)
		}
		if haveCond {
			var ok bool
			ok = condFun.Call(inputArgs)[0].Interface().(bool)
			if !ok {
				continue
			}
		}
		funReflected.Call(inputArgs)
	}
}

/*
func TestCallWithAllOptions(t *testing.T) {
	f1 := func(a int, b float64) {
		fmt.Println(a, b)
	}
	ints := []int{2, 3, 4}
	floats := []float64{1.0, 2.5}
	callWithAllOptions(f1, ints, floats)
	allTrue := func(...interface{}) bool {
		return true
	}
	allFalse := func(...interface{}) bool {
		return false
	}
	cmp := func(a int, b float64) bool {
		return float64(a) > b
	}
	callWithAllOptions(f1, allTrue, ints, floats)
	callWithAllOptions(f1, allFalse, ints, floats)
	callWithAllOptions(f1, cmp, ints, floats)
}
*/

// benchmarkForPointTypes(bOuter, samples, fun, [condition, ], [formatString,], pointTypeArrayOrSlices...) runs benchmark for the given function fun.
//
// bOuter ist the *testing.B benchmark environment from Go's testing framework
// samples is the size of the precomputed sample table
// fun(bInner *testing.B, slice1,slice2,...,sliceL []CurvePointPtrInterfaceTestSample) is a benchmark
// condition is an optional function condition(pointType1, pointType2, ..., pointTypeL) bool
// formatString is an optional format string [Note: the order of condition and formatString can be swapped, but they must come before the pointTypeArrayOrSlices]
// pointTypeArrayOrSlices... are any number L of (optional: pointers-to) slices or arrays of PointTypes
//
// We then consider all L-tuples (pointType_1, ... pointType_L) with pointType_i from pointTypeArrayOrSlice_i.
// If condition is present, we restict to those L-tuples for which condition(pointType_1, ..., pointType_L) returns true.
//
// We then construct slices slice1, ..., sliceL of length samples of types pointType1,...,pointTypeL and run fun(bInner, slice1,...) as a sub-benchmark (using bOuter.Run)
// formatString is used to construct the subbenchmark's tag "example tag %[1]v %[2]v..." using Sprintf, where %[i]v is the tag of pointType_i.
//
// Note that Go's benchmarking/testing framework allows filtering for tags and only selectively run benchmarks for tags matching a regexp!
func benchmarkForPointTypes(bOuter *testing.B, samples int, fun interface{}, args ...interface{}) {
	// parse fun:
	funReflected := reflect.ValueOf(fun)
	if funReflected.Kind() != reflect.Func {
		panic("second argument to benchmark for PointTypes must be function")
	}
	funType := funReflected.Type()
	funVariadic := funType.IsVariadic() // if fun is itself variadic, counting the expected number of arguments does not work.

	// condFun is the optional condition function
	var haveCond bool = false
	var condFun reflect.Value

	// formatString is the optional format string argument
	var formatString string
	var haveFmt bool = false

	// process variadic args to benchmarkForPointType in a loop. This is used to detect whether a condition function / format string is present.
	// We need to keep the index where we first encounter a slice of PointTypes, as further processing only accepts slice of PointTypes
	argsIndex := 0
argParse:
	for ; argsIndex < len(args); argsIndex++ {
		argReflected := reflect.ValueOf(args[argsIndex])
		// Dereference arguments if they are pointers
		if argReflected.Kind() == reflect.Ptr {
			argReflected = argReflected.Elem()
		}
		switch argReflected.Kind() {
		case reflect.Func: // condition argument
			if haveCond {
				panic("two conditions provided to benchmarkForPointTypes")
			}
			haveCond = true
			condFun = argReflected
			testutils.Assert(condFun.Type().NumOut() == 1, "condition function must have exactly one return value")
			testutils.Assert(condFun.Type().Out(0).Kind() == reflect.Bool, "condition function must return bool")
		case reflect.String: // formatString argument
			if haveFmt {
				panic("two format tags given")
			}
			haveFmt = true
			formatString = argReflected.Interface().(string) // reflect.Value.String() is a weird special case we want to avoid
		case reflect.Slice, reflect.Array: // start of pointTypeArrayOrSlices. We break detecting poential condition / format string args
			break argParse
		default:
			panic("Invalid argument to benchmarkForPointTypes")
		}
	}
	// any remaining args must be pointTypeArrayOrSlices. We num parse those
	sliceArgs := len(args) - argsIndex
	if !funVariadic {
		testutils.Assert(sliceArgs == funType.NumIn()-1, "The function provided to benchmarkForPointTypes must take as many non-testing.B arguments as there are PointType slice arguments")
	}
	// process remaining arguments (must be of type slice/array of PointTypes).
	// We derive the product of their lengths; this is then used to create all tuples with a single for loop with that length (we do not want to nest for loops with recursion).
	numCalls := 1
	for i := argsIndex; i < len(args); i++ {
		argReflected := reflect.ValueOf(args[i])
		// dereferency args[i] as needed
		if argReflected.Kind() == reflect.Ptr {
			argReflected = argReflected.Elem()
		}
		testutils.Assert(argReflected.Kind() == reflect.Slice || argReflected.Kind() == reflect.Array, "trailing arguments to benchmarkForPointTypes must be (pointers to / or plain values of) slices or arrays")
		numCalls *= argReflected.Len()
	}
	// We now construct numCalls many sliceArgs-tuples.
	for callIndex := 0; callIndex < numCalls; callIndex++ {
		// translate callIndex into a sliceArgs-tuple of indices and obtain the actual pointTypes
		var actualTypes []PointType = make([]PointType, sliceArgs)
		var tags []string = make([]string, sliceArgs)
		r := callIndex
		for i := 0; i < sliceArgs; i++ {
			argReflected := reflect.ValueOf(args[i+argsIndex])
			if argReflected.Kind() == reflect.Ptr {
				argReflected = argReflected.Elem()
			}
			sliceLen := argReflected.Len()
			sliceIndex := r % sliceLen
			selectedPointType, ok := argReflected.Index(sliceIndex).Interface().(PointType)
			if !ok {
				panic("variadic slice/array arguments to benchmarkForPointTypes must contain values of type PointType")
			}
			actualTypes[i] = selectedPointType
			tags[i] = pointTypeToTag(selectedPointType)
			r -= sliceIndex
			r /= sliceLen
		}
		// if we have a condition function, use it to filter
		if haveCond {
			var condInputArgs []reflect.Value = make([]reflect.Value, sliceArgs)
			for i := 0; i < sliceArgs; i++ {
				condInputArgs[i] = reflect.ValueOf(actualTypes[i])
			}
			condOutput := condFun.Call(condInputArgs) // Note: condFun.Call returns a slice, since functions can in theory have multiple return values. We ensured it only has 1 during parsing.
			ok := condOutput[0].Bool()                // condOutput[0] is the only return value of condFun
			if !ok {
				continue
			}
		}
		// create tag
		var tag string
		if haveFmt {
			var tagsAsinterface []interface{} = make([]interface{}, len(tags))
			for i := 0; i < len(tags); i++ {
				tagsAsinterface[i] = tags[i]
			}
			tag = fmt.Sprintf(formatString, tagsAsinterface...)
		} else {
			for i := 0; i < len(tags); i++ {
				tag = tag + tags[i]
				if i+1 < len(tags) {
					tag = tag + ","
				}
			}
		}
		// create input args
		var inputArgs []reflect.Value = make([]reflect.Value, sliceArgs+1) // +1 for testing.B
		for i := 0; i < sliceArgs; i++ {
			arg := getPrecomputedCurvePointSlice(int64(i), actualTypes[i], samples)
			inputArgs[i+1] = reflect.ValueOf(arg)
		}
		bOuter.Run(tag, func(bSubtest *testing.B) {
			inputArgs[0] = reflect.ValueOf(bSubtest)
			prepareBenchmarkCurvePoints(bSubtest)
			funReflected.Call(inputArgs)
		})
	}
}

// OLD CODE

/*
// Calling DumpCPI_<foo>[i].Add(...) will only work if
// DumpCPI_<foo>[i] is non-nil (i.e. has a concrete type to select the actual CurvePoint type)
// AND the contained pointer value DumpCPI_<foo>[i].(*foo) of type *foo is non-nil.
// These are initialized in init().
// plain DumpCPI is (intentionally) uninitialized and needs to be set to a concrete type *after* prepare_<foo>
var DumpCPI_XTW, DumpCPI_AXTW, DumpCPI_EFGH, DumpCPI [benchS]CurvePointPtrInterfaceWrite_FullCurve

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
*/
