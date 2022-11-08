package testutils

import (
	"math/rand"
	"testing"

	"github.com/GottfriedHerold/Bandersnatch/internal/utils"
)

// TODO: Test thread-safety?

var testPrecomputedCache = MakePrecomputedCache[int64, uint64](
	func(key int64) *rand.Rand {
		return rand.New(rand.NewSource(key))
	},
	func(rng *rand.Rand, key int64) uint64 {
		return rng.Uint64()
	},
	func(in uint64) uint64 {
		return in
	},
)

func TestRetrievePrecomputedData(t *testing.T) {
	const key1 = 10
	const key2 = 11
	data1 := testPrecomputedCache.GetElements(key1, 0)
	FatalUnless(t, data1 != nil, "nil returned")
	FatalUnless(t, len(data1) == 0, "invalid length")
	data21 := testPrecomputedCache.GetElements(key2, 100)
	data22 := testPrecomputedCache.GetElements(key2, 100)
	FatalUnless(t, data21 != nil, "nil returned")
	FatalUnless(t, data22 != nil, "nil returned")
	FatalUnless(t, len(data21) == 100, "invalid length")
	FatalUnless(t, len(data22) == 100, "invalid length")
	FatalUnless(t, &data21[0] != &data22[0], "aliasing")
	FatalUnless(t, utils.CompareSlices(data21, data22), "SlicesUnequal")
	data23 := testPrecomputedCache.GetElements(key2, 50)
	data24 := testPrecomputedCache.GetElements(key2, 200)
	FatalUnless(t, data23 != nil, "nil returned")
	FatalUnless(t, data24 != nil, "nil returned")
	FatalUnless(t, len(data23) == 50, "invalid length")
	FatalUnless(t, len(data24) == 200, "invalid length")
	FatalUnless(t, utils.CompareSlices(data23, data24[0:50]), "No prefix")
	// fmt.Println(data24)  -- manually insepct that it "looks random enough"

	FatalUnless(t, CheckPanic(func() { testPrecomputedCache.PrepopulateCache(key2, []uint64{}) }), "did not panic")
	testPrecomputedCache.PrepopulateCache(key1, []uint64{1, 2, 3})
	data3 := testPrecomputedCache.GetElements(key1, 4)
	FatalUnless(t, utils.CompareSlices(data3[0:3], []uint64{1, 2, 3}), "Did not get back prepopulated data")

}
