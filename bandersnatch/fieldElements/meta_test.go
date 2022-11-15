package fieldElements

import (
	"math/big"
	"math/rand"
	"reflect"
	"sync"
	"testing"

	"github.com/GottfriedHerold/Bandersnatch/bandersnatch/common"
	"github.com/GottfriedHerold/Bandersnatch/internal/callcounters"
	"github.com/GottfriedHerold/Bandersnatch/internal/testutils"
	"github.com/GottfriedHerold/Bandersnatch/internal/utils"
)

// This file contains code that is shared by a lot of benchmarking and testing code
// such as setup and/or teardown code as well as integrating call counters into go's
// default benchmarking framework.

// This file contains the code specific for benchmarking and testing field elements;
// (There is similar code for curve operations)

// We have benchmarking code for the actual FieldElement = bsFieldElement_64 implementation
// and also (nearly identical) benchmarking code for the reference bsFieldElement_8 implementation. The latter is just for comparison.

// The concrete functionality provided is this:
//
// We provide global variables that the benchmarked functions write their result to.
// (This is to avoid the compiler from outsmarting us -- writing to a global constant forces the compiler to actually do the computation)
// We provide a facility to sample slices of random-looking field elements.
// We provide a setup function that ensures call counters are handled correctly.
// It's also nice to have a common setup code entry point. we could extend this to ensure that certain tables are in the cache etc.

// As a leftover from when everything was one package, some names are tagged with _fe to distinguish between field element and curve point code/variables/constants/types.
// We don't bother to change that.

// size of Dump slices used in benchmarks. [Note that benchS is a separate variable for which we assert benchS <= dumpSizeBench_fe]
const dumpSizeBench_fe = 1 << 8

// benchmark functions write to DumpXXX variables.
// These are "exported"[*]  package-level variables to prevent the compiler from optimizations
// based on the fact that they are never read from within the module (I doubt the compiler would do this, but anyway...)
//
// [*] in non-test builds this file is ignored anyway
var DumpBools_fe [dumpSizeBench_fe]bool

var DumpFe_64 [dumpSizeBench_fe]bsFieldElement_64
var DumpFe_8 [dumpSizeBench_fe]bsFieldElement_8
var DumpUint256 [dumpSizeBench_fe]uint256
var DumpUint512 [dumpSizeBench_fe]uint512
var DumpBigInt [dumpSizeBench_fe]*big.Int = func() (_DumpBigInt [dumpSizeBench_fe]*big.Int) {
	for i := 0; i < dumpSizeBench_fe; i++ {
		_DumpBigInt[i] = big.NewInt(0)
		_DumpBigInt[i].Set(twoTo256_Int) // to reserve memory
	}
	return
}()

// prepareBenchmarkFieldElements runs some setup code and should be called in every (sub-)benchmark before the actual code that is to be benchmarked.
// Note that it resets all counters.
func prepareBenchmarkFieldElements(b *testing.B) {
	b.Cleanup(func() { postProcessBenchmarkFieldElements(b); ensureFieldElementConstantsWereNotChanged() })
	resetBenchmarkFieldElements(b)
}

// prepareTestFieldElements run common setup code and should be called in every (sub-)test for field elements.
func prepareTestFieldElements(t *testing.T) {
	t.Cleanup(ensureFieldElementConstantsWereNotChanged)
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

// SeedAndRange is a type used as key to certain functions to compute precomputed slices.
type SeedAndRange struct {
	seed         int64    // randomness seed
	allowedRange *big.Int // created elements are in [0, allowedRange)
}

var (
	pc_uint256_a SeedAndRange = SeedAndRange{
		seed:         1,
		allowedRange: twoTo256_Int,
	}
	pc_uint256_b SeedAndRange = SeedAndRange{
		seed:         1,
		allowedRange: twiceBaseFieldSize_Int,
	}
	pc_uint256_c SeedAndRange = SeedAndRange{
		seed:         1,
		allowedRange: montgomeryBound_Int,
	}
	pc_uint256_f SeedAndRange = SeedAndRange{
		seed:         1,
		allowedRange: baseFieldSize_Int,
	}
)

// CachedUint256 is used to retrieved precomputed slices of uint256's. The key of type SeedAndRange allows to select an rng seed an a range.
//
// Usage: CachedUint256.GetElements(key, amount)
var CachedUint256 = testutils.MakePrecomputedCache[SeedAndRange, uint256](
	// creating random seed:
	func(key SeedAndRange) *rand.Rand {
		testutils.Assert(key.allowedRange != nil)
		testutils.Assert(key.allowedRange.Sign() > 0)
		testutils.Assert(key.allowedRange.BitLen() <= 256 || key.allowedRange.Cmp(common.TwoTo256_Int) == 0)

		return rand.New(rand.NewSource(key.seed))
	},
	// sampling random uint256's
	func(rng *rand.Rand, key SeedAndRange) uint256 {
		var rnd_Int *big.Int = big.NewInt(0)
		rnd_Int.Rand(rng, key.allowedRange)
		return utils.BigIntToUIntArray(rnd_Int)
	},
	// copy function: nil is OK here; this selects a trivial function
	nil,
)

// CachedUint512 is used to retrieved precomputed slices of uint512's. The key is of type int64 and is used as an rng seed.
//
// NOTE: As opposed to CachedUint256, we don't need to select a range. We always use the full [0, 2**512) range.
//
// Usage: CachedUint512.GetElements(key, amount)
var CachedUint512 = testutils.MakePrecomputedCache[int64, uint512](
	// creating random seed:
	testutils.DefaultCreateRandFromSeed,
	// sampling random uint256's
	func(rng *rand.Rand, key int64) (ret uint512) {
		for i := 0; i < 8; i++ {
			ret[i] = rng.Uint64()
		}
		return
	},
	// copy function: nil is OK here; this selects a trivial function
	nil,
)

// CachedBigInt is used to retrieved precomputed slices of *big.Int's. The key of type SeedAndRange allows to select an rng seed an a range.
//
// Usage: CachedBigInt.GetElements(key, amount). The retrieved elements are always fresh (deep) copies.
var CachedBigInt = testutils.MakePrecomputedCache[SeedAndRange, *big.Int](
	// create rng
	func(key SeedAndRange) *rand.Rand {
		testutils.Assert(key.allowedRange != nil)
		testutils.Assert(key.allowedRange.Sign() > 0)
		return rand.New(rand.NewSource(key.seed))
	},
	// sample *big.Int's
	func(rng *rand.Rand, key SeedAndRange) *big.Int {
		return new(big.Int).Rand(rng, key.allowedRange)
	},
	// copying: We return a deep copy
	func(in *big.Int) (ret *big.Int) {
		return new(big.Int).Set(in)
	},
)

// _makePrecomputedCacheForFieldElements is an utility function for GetPrecomputedFieldElements.
// It is used to generially create a testutils.PrecomputedCache[int64, FieldElementType] for arbitrary FieldElementType
func _makePrecomputedCacheForFieldElements[FieldElementType any, FieldElementPtr interface {
	*FieldElementType
	SetRandomUnsafe(*rand.Rand)
}]() testutils.PrecomputedCache[int64, FieldElementType] {
	return testutils.MakePrecomputedCache[int64, FieldElementType](
		func(key int64) *rand.Rand {
			return rand.New(rand.NewSource(key))
		},
		func(rnd *rand.Rand, key int64) (ret FieldElementType) {
			retPtr := &ret
			FieldElementPtr(retPtr).SetRandomUnsafe(rnd)
			return
		},
		nil,
	)
}

// global mutex-protected map reflect.Type -> *testutils.PrecomputedCache[int64, type]. This is used to implement GetPrecomputedFieldElements[type, *type]
var (
	_cachedFieldElements      map[reflect.Type]any = make(map[reflect.Type]any) // _cachedFieldElements[feType] has dynamic type *testutils.PrecomputedCache[int64, feType]
	_cachedFieldElementsMutex sync.RWMutex
)

// GetPrecomputedFieldElements[FieldElementType](key, amount) provides the same functionality as
// testutils.PrecomputedCache.GetElements(), but for arbitrary [FieldElementType].
//
// NOTE: This is thread-safe!
func GetPrecomputedFieldElements[FieldElementType any, FieldElementPtr interface {
	*FieldElementType
	SetRandomUnsafe(*rand.Rand)
}](key int64, amount int) []FieldElementType {
	feType := utils.TypeOfType[FieldElementType]()
	_cachedFieldElementsMutex.RLock()
	var cache any
	cache, ok := _cachedFieldElements[feType] // retrieve cache (of dynamic type *testutils.PrecomputedCache[int64, FieldElementType] if non-nil)
	_cachedFieldElementsMutex.RUnlock()
	// If the cache did not exist yet, create it:
	if !ok {
		newTypedCache := _makePrecomputedCacheForFieldElements[FieldElementType, FieldElementPtr]()
		_cachedFieldElementsMutex.Lock()
		cache, ok = _cachedFieldElements[feType]
		if !ok {
			_cachedFieldElements[feType] = &newTypedCache // We put a pointer into the map. This simplifies reasoning, as the pointer never changes.
			cache = &newTypedCache
		}
		_cachedFieldElementsMutex.Unlock()
	}
	typedCache := cache.(*testutils.PrecomputedCache[int64, FieldElementType]) // restore type information
	return typedCache.GetElements(key, amount)
}

// These are deprecated in favor of the above. GetPrecomputedFieldElements[bsFieldElement_8] does a better job.

// Benchmarks operate on random-looking field elements.
// We generate these before the actual benchmark timer starts (resp. we reset the timer after creating them)
// In order to not have to generate them freshly for every single benchmark, we keep caches of precomputed pseudorandom field elements around.
// (NOTE: The interface always provides a fresh *copy* of what is cached, so modifications are safe -- Note that even seemingly read-only functions might change
// the internal represensation to an equivalent one)

// pseudoRandomFieldElementCache_64 resp. pseudoRandomFieldElementCache_8 is a cache for the outpt of a slice of field elements for a given random seed.
// DEPRECATED
type (
	pseudoRandomFieldElementCache_8 struct {
		rng      *rand.Rand
		elements []bsFieldElement_8
	}
)

// cachedPseudoRandomFieldElements_64 resp. _8 hold per-seed caches (the map key) of pseudo-random field elements.
// DEPRECATED
var (
	cachedPseudoRandomFieldElements_8 map[int64]*pseudoRandomFieldElementCache_8 = make(map[int64]*pseudoRandomFieldElementCache_8)
)

// getElements retrieves the first amount many elements from the given cache, expanding the cache as needed.
// DEPRECATED
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

// newPrecomputedFieldElementSlice_8 generates a new pseduoRandomFieldElementCache_8 and returns a pointer to it.
// DEPRECATED
func newPrecomputedFieldElementSlice_8(seed int64) (pc *pseudoRandomFieldElementCache_8) {
	pc = new(pseudoRandomFieldElementCache_8)
	pc.rng = rand.New(rand.NewSource(seed))
	pc.elements = make([]bsFieldElement_8, 0)
	return
}

// getPrecomptedFieldElementSlice_8 returns a slice of amount many pseudo-random field elements generated using the given seed value.
// DEPRECATED
func getPrecomputedFieldElementSlice_8(seed int64, amount int) []bsFieldElement_8 {
	pc := cachedPseudoRandomFieldElements_8[seed]
	if pc == nil {
		pc = newPrecomputedFieldElementSlice_8(seed)
		cachedPseudoRandomFieldElements_8[seed] = pc
	}
	return pc.getElements(amount)
}
