package bandersnatch

import (
	"math/rand"
	"testing"

	"github.com/GottfriedHerold/Bandersnatch/internal/callcounters"
)

// This file contains code that is shared by a lot of benchmarking code
// such as setup and/or teardown code as well as integrating call counters into go's
// default benchmarking framework.

// This file contains the code specific for benchmarking field elements;
// (code for field elements is separate to facilitate moving all field-element related code into a subpackage)

// size of precomputed slices used in benchmarks
const benchSizeFe = 2 << 8

// benchmark functions write to DumpXXX variables.
// These are "exported"[*]  package-level variables to prevent the compiler from optimizations
// based on the fact that they are never read from within the bandersnatch module
//
// [*] in non-test builds this file is ignored anyway
var DumpBools_Fe [benchSizeFe]bool
var DumpFe_64 [benchSizeFe]bsFieldElement_64
var DumpFe_8 [benchSizeFe]bsFieldElement_8

type precomputedFieldElementSlice_64 struct {
	rng      *rand.Rand
	elements []bsFieldElement_64
}

type precomputedFieldElementSlice_8 struct {
	rng      *rand.Rand
	elements []bsFieldElement_8
}

var (
	precomputedFieldElementSlices_64 map[int64]*precomputedFieldElementSlice_64 = make(map[int64]*precomputedFieldElementSlice_64)
	precomputedFieldElementSlices_8  map[int64]*precomputedFieldElementSlice_8  = make(map[int64]*precomputedFieldElementSlice_8)
)

func (pc *precomputedFieldElementSlice_64) getElements(number int) (ret []bsFieldElement_64) {
	assert(pc.rng != nil)
	assert(pc.elements != nil)
	currentLen := len(pc.elements)
	if number > currentLen {
		var temp bsFieldElement_64
		for i := 0; i < number-currentLen; i++ {
			temp.setRandomUnsafe(pc.rng)
			pc.elements = append(pc.elements, temp)
		}
	}
	assert(len(pc.elements) <= number)

	ret = make([]bsFieldElement_64, number)
	copied := copy(ret, pc.elements)
	assert(copied == number)
	return
}

func (pc *precomputedFieldElementSlice_8) getElements(number int) (ret []bsFieldElement_8) {
	assert(pc.rng != nil)
	assert(pc.elements != nil)
	currentLen := len(pc.elements)
	if number > currentLen {
		var temp bsFieldElement_8
		for i := 0; i < number-currentLen; i++ {
			temp.setRandomUnsafe(pc.rng)
			pc.elements = append(pc.elements, temp)
		}
	}
	assert(len(pc.elements) <= number)

	ret = make([]bsFieldElement_8, number)
	copied := copy(ret, pc.elements)
	assert(copied == number)
	return
}

func newPrecomputedFieldElementSlice_64(seed int64) (pc *precomputedFieldElementSlice_64) {
	pc = new(precomputedFieldElementSlice_64)
	pc.rng = rand.New(rand.NewSource(seed))
	pc.elements = make([]bsFieldElement_64, 0)
	return
}

func newPrecomputedFieldElementSlice_8(seed int64) (pc *precomputedFieldElementSlice_8) {
	pc = new(precomputedFieldElementSlice_8)
	pc.rng = rand.New(rand.NewSource(seed))
	pc.elements = make([]bsFieldElement_8, 0)
	return
}

func getPrecomputedFieldElementSlice_64(seed int64, number int) []bsFieldElement_64 {
	pc := precomputedFieldElementSlices_64[seed]
	if pc == nil {
		pc = newPrecomputedFieldElementSlice_64(seed)
		precomputedFieldElementSlices_64[seed] = pc
	}
	return pc.getElements(number)
}

func getPrecomputedFieldElementSlice_8(seed int64, number int) []bsFieldElement_8 {
	pc := precomputedFieldElementSlices_8[seed]
	if pc == nil {
		pc = newPrecomputedFieldElementSlice_8(seed)
		precomputedFieldElementSlices_8[seed] = pc
	}
	return pc.getElements(number)
}

func prepareBenchmarkFieldElements(b *testing.B) {
	b.Cleanup(func() { postProcessBenchmarkFieldElements(b) })
	callcounters.ResetAllCounters()
	b.ResetTimer()
}

// postProcessBenchmarkFieldElements should be called at the end of each sub-test (preferably using b.Cleanup(...) )
// Currently, the only thing it does is make sure call counters are included in the benchmark if the current build includes them
func postProcessBenchmarkFieldElements(b *testing.B) {
	BenchmarkWithCallCounters(b)
}

// OLD CODE BELOW

/*

// NOTE: The various prepareBench... functions
// contain common setup functionality for benchmarking. They are all supposed to be run before any actual benchmarking starts.
// (In fact, they automatically reset all benchmark timers)
// They set up the global variables that benchmarks consume and reset the benchmarking timers and call counters.
// If using sub-benchmarks with b.Run(...), the prepareBench... functions can generally be run in the outer benchmark.
// However, the inner sub-benchmarks might still need to reset call counters; also rerunning prepareBench... is
// generally safer as it ensures that certain global variables are restored. (Note that functions might inadvertedly change those)

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
	for i := 0; i < benchSizeFe; i++ {
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
	for i := 0; i < benchSizeFe; i++ {
		bench_x_64[i].setRandomUnsafe(drng)
		bench_y_64[i].setRandomUnsafe(drng)
		bench_z_64[i].setRandomUnsafe(drng)
	}
	callcounters.ResetAllCounters()
	b.ResetTimer()
}

*/
