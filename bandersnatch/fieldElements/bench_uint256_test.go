package fieldElements

import "testing"

func benchmark_LongMul(b *testing.B) {
	var bench_x []Uint256 = CachedUint256.GetElements(pc_uint256_a, benchS)
	var bench_y []Uint256 = CachedUint256.GetElements(pc_uint256_a, benchS)
	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		DumpUint512[n%benchS].LongMul(&bench_x[n%benchS], &bench_y[n%benchS])
	}
}

func benchmark_LongSquare(b *testing.B) {
	var bench_x []Uint256 = CachedUint256.GetElements(pc_uint256_a, benchS)
	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		DumpUint512[n%benchS].LongSquare(&bench_x[n%benchS])
	}
}

func benchmark_Add256(b *testing.B) {
	var bench_x []Uint256 = CachedUint256.GetElements(pc_uint256_a, benchS)
	var bench_y []Uint256 = CachedUint256.GetElements(pc_uint256_a, benchS)
	for n := 0; n < b.N; n++ {
		DumpUint256[n%benchS].Add(&bench_x[n%benchS], &bench_y[n%benchS])
	}
}

func benchmark_Add256GetCarry(b *testing.B) {
	var bench_x []Uint256 = CachedUint256.GetElements(pc_uint256_a, benchS)
	var bench_y []Uint256 = CachedUint256.GetElements(pc_uint256_a, benchS)
	for n := 0; n < b.N; n++ {
		_ = DumpUint256[n%benchS].AddAndReturnCarry(&bench_x[n%benchS], &bench_y[n%benchS])
	}
}

func benchmark_Sub256(b *testing.B) {
	var bench_x []Uint256 = CachedUint256.GetElements(pc_uint256_a, benchS)
	var bench_y []Uint256 = CachedUint256.GetElements(pc_uint256_a, benchS)
	for n := 0; n < b.N; n++ {
		DumpUint256[n%benchS].Sub(&bench_x[n%benchS], &bench_y[n%benchS])
	}
}

func benchmark_Sub256GetBorrow(b *testing.B) {
	var bench_x []Uint256 = CachedUint256.GetElements(pc_uint256_a, benchS)
	var bench_y []Uint256 = CachedUint256.GetElements(pc_uint256_a, benchS)
	for n := 0; n < b.N; n++ {
		_ = DumpUint256[n%benchS].SubAndReturnBorrow(&bench_x[n%benchS], &bench_y[n%benchS])
	}
}

func benchmark_IsZeroUint256(b *testing.B) {
	var bench_x []Uint256 = CachedUint256.GetElements(pc_uint256_a, benchS)
	for n := 0; n < b.N; n++ {
		DumpBools_fe[n%benchS] = bench_x[n%benchS].IsZero()
	}
}

// Benchmark for uint256 methods that do not involve modular reduction wrt. BaseFieldSize

func Benchmark_uint256(b *testing.B) {
	b.Run("LongMul256->512", benchmark_LongMul)
	b.Run("LongSquare256->512", benchmark_LongSquare)
	b.Run("Add256 (no modular reduction)", benchmark_Add256)
	b.Run("Add256C (no modular reduction, retain carry", benchmark_Add256GetCarry)
	b.Run("Sub256 (no modular reduction)", benchmark_Sub256)
	b.Run("Sub256B (no modular reduction, retain borrow", benchmark_Sub256GetBorrow)
	b.Run("IsZero (test for exactly 0, no reduction)", benchmark_IsZeroUint256)
}
