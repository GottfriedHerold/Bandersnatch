package common

/*
import (
	"encoding/binary"
	"math/rand"
	"testing"

	"github.com/GottfriedHerold/Bandersnatch/internal/testutils"
	"github.com/GottfriedHerold/Bandersnatch/internal/utils"
)

func BenchmarkFieldElementEndianness(b *testing.B) {
	b.Run("", utils.Bind2(benchmarkFieldElementEndianness_writeToBuf, LittleEndian))
	b.Run("", utils.Bind2(benchmarkFieldElementEndianness_writeToBuf, BigEndian))
}

var precomputedBytes = testutils.MakePrecomputedCache[int64, uint64](
	testutils.DefaultCreateRandFromSeed,
	func(rnd *rand.Rand, key int64) uint64 {
		return rnd.Uint64()
	},
	nil,
)

func benchmarkFieldElementEndianness_writeToBuf(b *testing.B, fe FieldElementEndianness) {

	const cycle = 256

	var testdata []uint64 = precomputedBytes.GetElements(10001, 4*cycle)
	// var buf [32 * cycle]byte
	var buf2 [4]uint64
	var buf3 [32]byte
	testutils.MakeVariableEscape(b, &buf3)

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		nn := n % 256
		buf2[0] = testdata[nn*4]
		buf2[1] = testdata[nn*4+1]
		buf2[2] = testdata[nn*4+2]
		buf2[3] = testdata[nn*4+3]

		binary.LittleEndian.PutUint64(buf3[0:8], buf2[0])
		binary.LittleEndian.PutUint64(buf3[8:16], buf2[1])
		binary.LittleEndian.PutUint64(buf3[16:24], buf2[2])
		binary.LittleEndian.PutUint64(buf3[24:32], buf2[3])
	}

}

*/
