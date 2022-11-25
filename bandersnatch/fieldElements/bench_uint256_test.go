package fieldElements

import "testing"

// This file is part of the fieldElements package and contains the benchmarks for the functions defined in uint256.go
// This means we benchmark functions defined on uint256 that perfom arithmetic operations that treat uint256 as integers (possibly modulo 2^256) and do NOT work modulo BaseFieldSize.

// Benchmarks all follow the same pattern in order to make the overhead comparable.
// We benchmark functions for (fixed, pseudo-)random inputs that satisfy the preconditions of the functions.
// NOTE: We try to keep the order the same as the defintions in uint256_modular.go

// Benchmark_uint256 runs benchmarks for uint256 methods that do not involve modular reduction wrt. BaseFieldSize
func Benchmark_uint256(b *testing.B) {
	b.Run("trivial copying", benchmarkUint256_Copy)
	b.Run("LongMul256->512", benchmarkUint256_LongMul)
	b.Run("LongSquare256->512", benchmarkUint256_LongSquare)
	b.Run("Mul256 (multiplication modulo 2^256)", benchmarkUint256_Mul)
	b.Run("Add256 (no modular reduction)", benchmarkUint256_Add)
	b.Run("Add256C (no modular reduction, retain carry", benchmarkUint256_AddAndReturnCarry)
	b.Run("Sub256 (no modular reduction)", benchmarkUint256_Sub)
	b.Run("Sub256B (no modular reduction, retain borrow", benchmarkUint256_SubAndGetBorrow)
	b.Run("IsZero (test for exactly 0, no reduction)", benchmarkUint256_IsZero)
	b.Run("Increment", benchmarkUint256_Increment)
	b.Run("Decrement", benchmarkUint256_Decrement)
	b.Run("IncrementEq (with Copy)", benchmarkUint256_CopyAndIncEq)
	b.Run("DecrementEq (with Copy)", benchmarkUint256_CopyAndDecEq)
}

func benchmarkUint256_Copy(b *testing.B) {
	var bench_x []Uint256 = CachedUint256.GetElements(pc_uint256_a, benchS)
	prepareBenchmarkFieldElements(b)
	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		DumpUint256[n%benchS] = bench_x[n%benchS]
	}
}

func benchmarkUint256_LongMul(b *testing.B) {
	var bench_x []Uint256 = CachedUint256.GetElements(pc_uint256_a, benchS)
	var bench_y []Uint256 = CachedUint256.GetElements(pc_uint256_a, benchS)
	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		DumpUint512[n%benchS].LongMul(&bench_x[n%benchS], &bench_y[n%benchS])
	}
}

func benchmarkUint256_LongSquare(b *testing.B) {
	var bench_x []Uint256 = CachedUint256.GetElements(pc_uint256_a, benchS)
	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		DumpUint512[n%benchS].LongSquare(&bench_x[n%benchS])
	}
}

func benchmarkUint256_Add(b *testing.B) {
	var bench_x []Uint256 = CachedUint256.GetElements(pc_uint256_a, benchS)
	var bench_y []Uint256 = CachedUint256.GetElements(pc_uint256_a, benchS)
	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		DumpUint256[n%benchS].Add(&bench_x[n%benchS], &bench_y[n%benchS])
	}
}

func benchmarkUint256_AddAndReturnCarry(b *testing.B) {
	var bench_x []Uint256 = CachedUint256.GetElements(pc_uint256_a, benchS)
	var bench_y []Uint256 = CachedUint256.GetElements(pc_uint256_a, benchS)
	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		_ = DumpUint256[n%benchS].AddAndReturnCarry(&bench_x[n%benchS], &bench_y[n%benchS])
	}
}

func benchmarkUint256_Sub(b *testing.B) {
	var bench_x []Uint256 = CachedUint256.GetElements(pc_uint256_a, benchS)
	var bench_y []Uint256 = CachedUint256.GetElements(pc_uint256_a, benchS)
	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		DumpUint256[n%benchS].Sub(&bench_x[n%benchS], &bench_y[n%benchS])
	}
}

func benchmarkUint256_SubAndGetBorrow(b *testing.B) {
	var bench_x []Uint256 = CachedUint256.GetElements(pc_uint256_a, benchS)
	var bench_y []Uint256 = CachedUint256.GetElements(pc_uint256_a, benchS)
	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		_ = DumpUint256[n%benchS].SubAndReturnBorrow(&bench_x[n%benchS], &bench_y[n%benchS])
	}
}

func benchmarkUint256_Mul(b *testing.B) {
	var bench_x []Uint256 = CachedUint256.GetElements(pc_uint256_a, benchS)
	var bench_y []Uint256 = CachedUint256.GetElements(pc_uint256_a, benchS)
	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		DumpUint256[n%benchS].Mul(&bench_x[n%benchS], &bench_y[n%benchS])
	}
}

func benchmarkUint256_IsZero(b *testing.B) {
	var bench_x []Uint256 = CachedUint256.GetElements(pc_uint256_a, benchS)
	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		DumpBools_fe[n%benchS] = bench_x[n%benchS].IsZero()
	}
}

func benchmarkUint256_Increment(b *testing.B) {
	var bench_x []Uint256 = CachedUint256.GetElements(pc_uint256_a, benchS)
	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		DumpUint256[n%benchS].Increment(&bench_x[n%benchS])
	}
}

func benchmarkUint256_Decrement(b *testing.B) {
	var bench_x []Uint256 = CachedUint256.GetElements(pc_uint256_a, benchS)
	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		DumpUint256[n%benchS].Decrement(&bench_x[n%benchS])
	}
}

func benchmarkUint256_CopyAndIncEq(b *testing.B) {
	var bench_x []Uint256 = CachedUint256.GetElements(pc_uint256_a, benchS)
	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		DumpUint256[n%benchS] = bench_x[n%benchS]
		DumpUint256[n%benchS].IncrementEq()
	}
}

func benchmarkUint256_CopyAndDecEq(b *testing.B) {
	var bench_x []Uint256 = CachedUint256.GetElements(pc_uint256_a, benchS)
	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		DumpUint256[n%benchS] = bench_x[n%benchS]
		DumpUint256[n%benchS].DecrementEq()
	}
}
