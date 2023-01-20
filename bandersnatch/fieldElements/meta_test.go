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

// This file is part of the fieldElements package. See the documentation of field_element.go for general remarks.

// This file contains code that is shared by a lot of benchmarking and testing code
// such as setup and/or teardown code as well as integrating call counters into go's
// default benchmarking framework.

// This file contains the code specific for benchmarking and testing field elements;
// (There is similar code for curve operations)

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

const benchS = 1 << 8

// benchmark functions write to DumpXXX variables.
// These are "exported"[*]  package-level variables to prevent the compiler from optimizations
// based on the fact that they are never read from within the module (I doubt the compiler would do this, but anyway...)
//
// [*] in non-test builds this file is ignored anyway
var (
	DumpBools_fe [dumpSizeBench_fe]bool
	DumpFe_64    [dumpSizeBench_fe]bsFieldElement_MontgomeryNonUnique
	DumpUint256  [dumpSizeBench_fe]Uint256
	DumpUint512  [dumpSizeBench_fe]Uint512
	DumpUint64   [dumpSizeBench_fe]uint64
	DumpUint320  [dumpSizeBench_fe][5]uint64
	DumpInt      [dumpSizeBench_fe]int
	DumpBigInt   [dumpSizeBench_fe]*big.Int = func() (_DumpBigInt [dumpSizeBench_fe]*big.Int) {
		for i := 0; i < dumpSizeBench_fe; i++ {
			_DumpBigInt[i] = big.NewInt(0)
			_DumpBigInt[i].Set(twoTo256_Int) // to reserve memory
		}
		return
	}()
)

// prepareBenchmarkFieldElements runs some setup code and should be called in every (sub-)benchmark before the actual code that is to be benchmarked.
// Note that it resets all counters.
func prepareBenchmarkFieldElements(b *testing.B) {
	b.Cleanup(func() { postProcessBenchmarkFieldElements(b); ensureFieldElementConstantsWereNotChanged() })
	resetBenchmarkFieldElements(b)
}

// prepareTestFieldElements run common setup code / registers teardown code, and should be called in every (sub-)test for field elements.
func prepareTestFieldElements(t *testing.T) {
	t.Cleanup(ensureFieldElementConstantsWereNotChanged) // registers a teardown-function that will detect any modification of (supposed) constants.
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

// Note: My Go linter complains about unneccessary type arguments (as those could be inferred); they are here on purpose, for the sake of readability.

// CachedUint256 is used to retrieve precomputed slices of uint256's. The key of type SeedAndRange allows to select an rng seed an a range.
//
// Usage: CachedUint256.GetElements(SeedAndRange{seed: rngseed, allowedRange: upperBound}, amount). Note that upperBound is strict, i.e. the outputs are in [0, upperBound)
var CachedUint256 = testutils.MakePrecomputedCache[SeedAndRange, Uint256](
	// creating random seed:
	func(key SeedAndRange) *rand.Rand {
		testutils.Assert(key.allowedRange != nil, "SeedAndRange argument to CachedUint256.GetElements lacks an allowedRange parameter")
		testutils.Assert(key.allowedRange.Sign() > 0, "CachedUint256.GetElements called with allowedRange <=0")
		testutils.Assert(key.allowedRange.BitLen() <= 256 || key.allowedRange.Cmp(common.TwoTo256_Int) == 0, "CachedUint256.GetElements called with too large allowedRange")

		return rand.New(rand.NewSource(key.seed))
	},
	// sampling random uint256's
	func(rng *rand.Rand, key SeedAndRange) Uint256 {
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
var CachedUint512 = testutils.MakePrecomputedCache[int64, Uint512](
	// creating random seed:
	testutils.DefaultCreateRandFromSeed,
	// sampling random uint256's
	func(rng *rand.Rand, key int64) (ret Uint512) {
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

// CachedUint64 is used to retrieve precomputed slices of random (seeded by int64 key) uint64's.
var CachedUint64 = testutils.MakePrecomputedCache[int64, uint64](
	testutils.DefaultCreateRandFromSeed,
	func(rng *rand.Rand, key int64) uint64 {
		return rng.Uint64()
	},
	nil,
)

// CachedUint64 is used to retrieve precomputed slices of random (seeded by int64 key) int64's. Note that these coincide with CachedUint64.
var CachedInt64 = testutils.MakePrecomputedCache[int64, int64](
	testutils.DefaultCreateRandFromSeed,
	func(rng *rand.Rand, key int64) int64 {
		return int64(rng.Uint64())
	},
	nil,
)

// _makePrecomputedCacheForFieldElements is an utility function for GetPrecomputedFieldElements.
// It is used to generially create a testutils.PrecomputedCache[int64, FieldElementType] for arbitrary FieldElementType
func _makePrecomputedCacheForFieldElements[FieldElementType any, FieldElementPtr interface {
	*FieldElementType
	FieldElementInterface[FieldElementPtr]
}]() testutils.PrecomputedCache[int64, FieldElementType] {
	return testutils.MakePrecomputedCache[int64, FieldElementType](
		func(key int64) *rand.Rand {
			return rand.New(rand.NewSource(key))
		},
		func(rnd *rand.Rand, key int64) (ret FieldElementType) {

			retInt := new(big.Int).Rand(rnd, baseFieldSize_Int)
			FieldElementPtr(&ret).SetBigInt(retInt)
			FieldElementPtr(&ret).RerandomizeRepresentation(rnd.Uint64())
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
	FieldElementInterface[FieldElementPtr]
}](key int64, amount int) []FieldElementType {
	var feType reflect.Type = utils.TypeOfType[FieldElementType]()
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
		} // else: throw away newTypedCache
		_cachedFieldElementsMutex.Unlock()
	}
	typedCache := cache.(*testutils.PrecomputedCache[int64, FieldElementType]) // restore type information
	return typedCache.GetElements(key, amount)
}

// GetPrecomputedFieldElementsNonZero[FieldElementType, FieldElementPtr](key, amount) provides the same functionality as
// GetPrecomputedFieldElement[FieldElementType, FieldElementPtr](key, amount), but it replaces all zeros by 1 afterwards.
// The randomness is the very same as in the variant without NonZero.
//
// NOTE: This is thread-safe!
func GetPrecomputedFieldElementsNonZero[FieldElementType any, FieldElementPtr interface {
	*FieldElementType
	FieldElementInterface[FieldElementPtr]
}](key int64, amount int) []FieldElementType {
	ret := GetPrecomputedFieldElements[FieldElementType, FieldElementPtr](key, amount)
	for i := 0; i < amount; i++ {
		if FieldElementPtr(&ret[i]).IsZero() {
			FieldElementPtr(&ret[i]).SetOne()
		}
	}
	return ret
}

// CachedRootsOfUnity stores (precomputed) values x^r for random r and x a 2^32th root of unity in the field.
var CachedRootsOfUnity = testutils.MakePrecomputedCache[int64, feType_SquareRoot](
	testutils.DefaultCreateRandFromSeed,
	func(rng *rand.Rand, key int64) (ret feType_SquareRoot) {
		r := rng.Uint32()
		ret.SetOne()
		for i := 0; i < 32; i++ {
			if r&(1<<i) != 0 {
				ret.MulEq(&sqrtPrecomp_PrimitiveDyadicRoots[i])
			}
		}
		return
	},
	nil,
)

// CachedRootsOfUnityWithExponent stores (precomputed) values x^r for random r and x the designated 2^32th root of unity in the field, together with the exponent r that was used.
var CachedRootsOfUnityWithExponent = testutils.MakePrecomputedCache[int64, struct {
	fe       feType_SquareRoot
	exponent uint32
}](
	testutils.DefaultCreateRandFromSeed,
	func(rng *rand.Rand, key int64) (ret struct {
		fe       feType_SquareRoot
		exponent uint32
	}) {
		ret.exponent = rng.Uint32()
		ret.fe.SetOne()
		for i := 0; i < 32; i++ {
			if ret.exponent&(1<<i) != 0 {
				ret.fe.MulEq(&sqrtPrecomp_PrimitiveDyadicRoots[i])
			}
		}
		return
	},
	nil,
)
