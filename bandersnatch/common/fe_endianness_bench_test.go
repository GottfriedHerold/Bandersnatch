package common

import (
	"math/rand"
	"testing"

	"github.com/GottfriedHerold/Bandersnatch/internal/testutils"
	"github.com/GottfriedHerold/Bandersnatch/internal/utils"
)

// Benchmark for FieldElementEndianness.
// We compare various ways of passing parameters. This kind of micro-optimization is kind-of relevant, unfortunately.

func BenchmarkFieldElementEndianness(b *testing.B) {
	b.Run("PutUint256LittleEndian", utils.Bind2(benchmarkFieldElementEndianness_writeToBuf, LittleEndian))
	b.Run("PutUint256BigEndian", utils.Bind2(benchmarkFieldElementEndianness_writeToBuf, BigEndian))

	b.Run("PutUint256LittleEndian_ptr", utils.Bind2(benchmarkFieldElementEndianness_writeToBuf_ptr, LittleEndian))
	b.Run("PutUint256BigEndian_ptr", utils.Bind2(benchmarkFieldElementEndianness_writeToBuf_ptr, BigEndian))

	b.Run("PutUint256_array_LittleEndian", utils.Bind2(benchmarkFieldElementEndianness_writeToBuf_array, LittleEndian))
	b.Run("PutUint256_array_BigEndian", utils.Bind2(benchmarkFieldElementEndianness_writeToBuf_array, BigEndian))

	b.Run("Uint256_LittleEndian", utils.Bind2(benchmarkFieldElementEndianness_ReadUint256, LittleEndian))
	b.Run("Uint256_BigEndian", utils.Bind2(benchmarkFieldElementEndianness_ReadUint256, BigEndian))
	b.Run("Uint256_indirect_LittleEndian", utils.Bind2(benchmarkFieldElementEndianness_ReadUint256_indirect, LittleEndian))
	b.Run("Uint256_indirect_BigEndian", utils.Bind2(benchmarkFieldElementEndianness_ReadUint256_indirect, BigEndian))
	b.Run("Uint256_array_LittleEndian", utils.Bind2(benchmarkFieldElementEndianness_ReadUint256_array, LittleEndian))
	b.Run("Uint256_array_BigEndian", utils.Bind2(benchmarkFieldElementEndianness_ReadUint256_array, BigEndian))

	b.Run("DirectCopy_LittleEndian", benchmarkCopyLittleEndian)

}

var precomputedUint64 = testutils.MakePrecomputedCache[int64, uint64](
	testutils.DefaultCreateRandFromSeed,
	func(rnd *rand.Rand, key int64) uint64 {
		return rnd.Uint64()
	},
	nil,
)

var precomputedbytes = testutils.MakePrecomputedCache[int64, byte](
	testutils.DefaultCreateRandFromSeed,
	func(rnd *rand.Rand, key int64) byte {
		return byte(rnd.Uint64() % 0xFF)
	},
	nil,
)

func benchmarkFieldElementEndianness_writeToBuf(b *testing.B, fe FieldElementEndianness) {

	const cycle = 256

	var testdata []uint64 = precomputedUint64.GetElements(10001, 4*cycle)

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		var buf3 [32]byte

		nn := n % 256

		fe.PutUint256(buf3[:], *(*[4]uint64)(testdata[nn*4 : (nn+1)*4]))

	}
}

func benchmarkFieldElementEndianness_writeToBuf_ptr(b *testing.B, fe FieldElementEndianness) {

	const cycle = 256

	var testdata []uint64 = precomputedUint64.GetElements(10001, 4*cycle)

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		var buf3 [32]byte

		nn := n % 256

		fe.PutUint256_ptr(buf3[:], (*[4]uint64)(testdata[nn*4:(nn+1)*4]))

	}
}

func benchmarkFieldElementEndianness_writeToBuf_array(b *testing.B, fe FieldElementEndianness) {

	const cycle = 256

	var testdata []uint64 = precomputedUint64.GetElements(10001, 4*cycle)

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		var buf3 [32]byte

		nn := n % 256

		fe.PutUint256_array(&buf3, (*[4]uint64)(testdata[nn*4:(nn+1)*4]))

	}
}

func benchmarkFieldElementEndianness_ReadUint256(b *testing.B, fe FieldElementEndianness) {
	const cycle = 256

	var testdata []byte = precomputedbytes.GetElements(10001, 32*cycle)

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		nn := n % 256
		var res [4]uint64
		res = fe.Uint256(testdata[nn*32 : (nn+1)*32])
		_ = res
	}

}

func benchmarkFieldElementEndianness_ReadUint256_indirect(b *testing.B, fe FieldElementEndianness) {
	const cycle = 256

	var testdata []byte = precomputedbytes.GetElements(10001, 32*cycle)

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		nn := n % 256
		var res [4]uint64
		fe.Uint256_indirect(testdata[nn*32:(nn+1)*32], &res)
		_ = res
	}

}

func benchmarkFieldElementEndianness_ReadUint256_array(b *testing.B, fe FieldElementEndianness) {
	const cycle = 256

	var testdata []byte = precomputedbytes.GetElements(10001, 32*cycle)

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		nn := n % 256
		var res [4]uint64
		fe.Uint256_array((*[32]byte)(testdata[nn*32:(nn+1)*32]), &res)
		_ = res
	}

}

func benchmarkCopyLittleEndian(b *testing.B) {
	const cycle = 256

	var testdata []byte = precomputedbytes.GetElements(10001, 32*cycle)

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		nn := n % 256
		var res [32]byte
		copy(res[:], testdata[32*nn:32*(nn+1)])
		_ = res
	}

}
