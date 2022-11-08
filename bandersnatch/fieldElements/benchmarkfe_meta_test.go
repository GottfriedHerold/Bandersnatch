package fieldElements

import (
	"math/rand"
	"testing"

	"github.com/GottfriedHerold/Bandersnatch/internal/callcounters"
	"github.com/GottfriedHerold/Bandersnatch/internal/testutils"
)

// This file contains code that is shared by a lot of benchmarking code
// such as setup and/or teardown code as well as integrating call counters into go's
// default benchmarking framework.

// This file contains the code specific for benchmarking field elements;
// (There is similar code for curve operations)

// We have benchmarking code for the actual FieldElement = bsFieldElement_64 implementation
// and also (nearly identical) benchmarking code for the reference bsFieldElement_8 implementation. The latter is just for comparison.

// The concrete functionality provided is this:
//
// We provide global variables that the benchmarked functions write their result to.
// (This is to avoid the compiler from outsmarting us -- writing to a global constant forces the compiler to actually do the computation)
// We provide a facility to sample slices of random-looking field elements.
// We provide a setup function that ensures call counters are handled correctly.

// size of Dump slices used in benchmarks. The _fe is from when we had no separate packages for field elements and curve points.
const dumpSizeBench_fe = 1 << 8

// benchmark functions write to DumpXXX variables.
// These are "exported"[*]  package-level variables to prevent the compiler from optimizations
// based on the fact that they are never read from within the module (I doubt the compiler would do this, but anyway...)
//
// [*] in non-test builds this file is ignored anyway
var DumpBools_fe [dumpSizeBench_fe]bool // the _fe is because we use a separate global variable for benchmarking curve operations.

var DumpFe_64 [dumpSizeBench_fe]bsFieldElement_64
var DumpFe_8 [dumpSizeBench_fe]bsFieldElement_8
var DumpUint256 [dumpSizeBench_fe]uint256
var DumpUint512 [dumpSizeBench_fe]uint512

// prepareBenchmarkFieldElements runs some setup code and should be called in every (sub-)test before the actual code that is to be benchmarked.
// Note that it resets all counters.
func prepareBenchmarkFieldElements(b *testing.B) {
	b.Cleanup(func() { postProcessBenchmarkFieldElements(b) })
	resetBenchmarkFieldElements(b)
}

// postProcessBenchmarkFieldElements should be called at the end of each sub-test (preferably using b.Cleanup(...) )
// Currently, the only thing it does is make sure call counters are included in the benchmark if the current build includes them
func postProcessBenchmarkFieldElements(b *testing.B) {
	BenchmarkWithCallCounters(b)
}

// resetBenchmarkFieldElements resets the benchmark counters; this should be called after any expensive setup that we do not want to include in the benchmark.
func resetBenchmarkFieldElements(b *testing.B) {
	callcounters.ResetAllCounters()
	b.ResetTimer()
}

// Benchmarks operate on random-looking field elements.
// We generate these before the actual benchmark timer starts (resp. we reset the timer after creating them)
// In order to not have to generate them freshly for every single benchmark, we keep caches of precomputed pseudorandom field elements around.
// (NOTE: The interface always provides a fresh *copy* of what is cached, so modifications are safe -- Note that even seemingly read-only functions might change
// the internal represensation to an equivalent one)

// pseudoRandomFieldElementCache_64 resp. pseudoRandomFieldElementCache_8 is a cache for the outpt of a slice of field elements for a given random seed.
type (
	pseudoRandomFieldElementCache_64 struct {
		rng      *rand.Rand
		elements []bsFieldElement_64
	}

	pseudoRandomFieldElementCache_8 struct {
		rng      *rand.Rand
		elements []bsFieldElement_8
	}
)

// TODO: Add mutexes
// Question: Unify this using generics (we use this pattern multiple times)?

// cachedPseudoRandomFieldElements_64 resp. _8 hold per-seed caches (the map key) of pseudo-random field elements.
var (
	cachedPseudoRandomFieldElements_64 map[int64]*pseudoRandomFieldElementCache_64 = make(map[int64]*pseudoRandomFieldElementCache_64)
	cachedPseudoRandomFieldElements_8  map[int64]*pseudoRandomFieldElementCache_8  = make(map[int64]*pseudoRandomFieldElementCache_8)
)

// getElements retrieves the first amount many elements from the given cache, expanding the cache as needed.
func (pc *pseudoRandomFieldElementCache_64) getElements(amount int) (ret []bsFieldElement_64) {
	testutils.Assert(pc.rng != nil)
	testutils.Assert(pc.elements != nil)
	currentLen := len(pc.elements)
	if amount > currentLen {
		var temp bsFieldElement_64
		for i := 0; i < amount-currentLen; i++ {
			temp.SetRandomUnsafe(pc.rng)
			pc.elements = append(pc.elements, temp)
		}
	}
	testutils.Assert(len(pc.elements) >= amount)

	ret = make([]bsFieldElement_64, amount)
	copied := copy(ret, pc.elements)
	testutils.Assert(copied == amount)
	return
}

// getElements retrieves the first amount many elements from the given cache, expanding the cache as needed.
func (pc *pseudoRandomFieldElementCache_8) getElements(number int) (ret []bsFieldElement_8) {
	testutils.Assert(pc.rng != nil)
	testutils.Assert(pc.elements != nil)
	currentLen := len(pc.elements)
	if number > currentLen {
		var temp bsFieldElement_8
		for i := 0; i < number-currentLen; i++ {
			temp.setRandomUnsafe(pc.rng)
			pc.elements = append(pc.elements, temp)
		}
	}
	testutils.Assert(len(pc.elements) >= number)

	ret = make([]bsFieldElement_8, number)
	copied := copy(ret, pc.elements)
	testutils.Assert(copied == number)
	return
}

// newPrecomputedFieldElementSlice_64 generates a new pseduoRandomFieldElementCache_64 and returns a pointer to it.
func newPrecomputedFieldElementSlice_64(seed int64) (pc *pseudoRandomFieldElementCache_64) {
	pc = new(pseudoRandomFieldElementCache_64)
	pc.rng = rand.New(rand.NewSource(seed))
	pc.elements = make([]bsFieldElement_64, 0)
	return
}

// newPrecomputedFieldElementSlice_8 generates a new pseduoRandomFieldElementCache_8 and returns a pointer to it.
func newPrecomputedFieldElementSlice_8(seed int64) (pc *pseudoRandomFieldElementCache_8) {
	pc = new(pseudoRandomFieldElementCache_8)
	pc.rng = rand.New(rand.NewSource(seed))
	pc.elements = make([]bsFieldElement_8, 0)
	return
}

// getPrecomptedFieldElementSlice_64 returns a slice of amount many pseudo-random field elements generated using the given seed value.
func getPrecomputedFieldElementSlice_64(seed int64, amount int) []bsFieldElement_64 {
	pc := cachedPseudoRandomFieldElements_64[seed]
	if pc == nil {
		pc = newPrecomputedFieldElementSlice_64(seed)
		cachedPseudoRandomFieldElements_64[seed] = pc
	}
	return pc.getElements(amount)
}

// getPrecomptedFieldElementSlice_8 returns a slice of amount many pseudo-random field elements generated using the given seed value.
func getPrecomputedFieldElementSlice_8(seed int64, amount int) []bsFieldElement_8 {
	pc := cachedPseudoRandomFieldElements_8[seed]
	if pc == nil {
		pc = newPrecomputedFieldElementSlice_8(seed)
		cachedPseudoRandomFieldElements_8[seed] = pc
	}
	return pc.getElements(amount)
}
