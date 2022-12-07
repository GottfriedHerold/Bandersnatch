package fieldElements

import "testing"

// This file is part of the fieldElements package. See the documentation of field_element.go for general remarks.

// This file contains the benchmarks for the functions defined in uint256_montgomery.go
// This means we benchmark functions defined on uint256 that perfom arithmetic operations that perform Montgomery multiplications (and helper functions for that)
// Note that this is not the same as benchmarking operations on field elements themselves.
// Instead, this serves to benchmark sub-routines and try out different internal approaches.

// Benchmarks all follow the same pattern in order to make the overhead comparable.
// We benchmark functions for (fixed, pseudo-)random inputs that satisfy the preconditions of the functions.

// NOTE: We try to keep the order the same as the defintions in uint256_montgomery.go
// NOTE2: We might remove tests here as the function become deprecated; these are a bit of a testbed.

func Benchmark_uint256_MontgomeryFuns(b *testing.B) {
	b.Run("ToNonMontgomery", benchmarkUint256Mont_ToNonMontgomery)
	b.Run("FromMontgomery", benchmarkUint256Mont_FromMontgomery)
	b.Run("MulFourOne", benchmarkUint256Mont_mul_four_one_64)
	b.Run("AddMulShift", benchmarkUint256Mont_add_mul_shift_64)
	b.Run("Mul uint256 x uint64, target", benchmarkUint256Mont_mul256by64)
	b.Run("Montgomery step (est.)", benchmarkUint256Mont_montgomery_step64)
	b.Run("shiftOnce (inaccurate)", benchmarkUint256Mont_shift_once)
	b.Run("Montgomery Mul V1", benchmarkUint256Mont_MulMontgomery)
	b.Run("Montgomery Mul V2", benchmark_Uint256Mont_MulMontgomeryV2)
	b.Log("INFO: Exponentiation algorithm used by default is " + uint256MontgomeryExponentiationAlgUsed)
	b.Run("Exponentiation (Montgomery, sliding window)", benchmarkUint256Mont_ExponentiationMontgomerySlW)
	b.Run("Exponentiation (Montgomery, square-and-multiply)", benchmarkUint256Mont_ExponentiationMontgomerySqM)

}

func benchmarkUint256Mont_ToNonMontgomery(b *testing.B) {
	var bench_x []Uint256 = CachedUint256.GetElements(pc_uint256_c, benchS)
	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		DumpUint256[n%benchS] = bench_x[n%benchS].ToNonMontgomery_fc()
	}
}

func benchmarkUint256Mont_FromMontgomery(b *testing.B) {
	var bench_x []Uint256 = CachedUint256.GetElements(pc_uint256_c, benchS)
	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		DumpUint256[n%benchS].FromMontgomeryRepresentation_fc(&bench_x[n%benchS])
	}
}

func benchmarkUint256Mont_mul_four_one_64(b *testing.B) {
	var bench_x []Uint256 = CachedUint256.GetElements(pc_uint256_a, benchS)
	var bench_y []uint64 = CachedUint64.GetElements(1, benchS)
	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		DumpUint64[n%benchS], DumpUint256[n%benchS] = mul_four_one_64(&bench_x[n%benchS], bench_y[n%benchS])
	}
}

func benchmarkUint256Mont_add_mul_shift_64(b *testing.B) {
	var bench_x []Uint256 = CachedUint256.GetElements(pc_uint256_a, benchS)
	var bench_y []uint64 = CachedUint64.GetElements(1, benchS)
	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		DumpUint64[n%benchS] = add_mul_shift_64(&DumpUint256[n%benchS], &bench_x[n%benchS], bench_y[n%benchS])
	}
}

func benchmarkUint256Mont_shift_once(b *testing.B) {
	var bench_x []Uint256 = CachedUint256.GetElements(pc_uint256_a, benchS)
	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		DumpUint64[n%benchS] = bench_x[n%benchS].ShiftRightEq_64() // modifies bench_x
	}
}

func benchmarkUint256Mont_mul256by64(b *testing.B) {
	var bench_x []Uint256 = CachedUint256.GetElements(pc_uint256_a, benchS)
	var bench_y []uint64 = CachedUint64.GetElements(1, benchS)
	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		LongMulUint64(&DumpUint320[n%benchS], &bench_x[n%benchS], bench_y[n%benchS])
	}
}

func benchmarkUint256Mont_montgomery_step64(b *testing.B) {
	var bench_x []Uint256 = CachedUint256.GetElements(pc_uint256_a, benchS)
	var bench_y []uint64 = CachedUint64.GetElements(1, benchS)
	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		montgomery_step_64(&bench_x[n%benchS], bench_y[n%benchS])
	}
}

func benchmarkUint256Mont_MulMontgomery(b *testing.B) {
	var bench_x []Uint256 = CachedUint256.GetElements(pc_uint256_c, benchS)
	var bench_y []Uint256 = CachedUint256.GetElements(pc_uint256_c, benchS)
	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		DumpUint256[n%benchS].mulMontgomerySlow_c(&bench_x[n%benchS], &bench_y[n%benchS])
	}
}

func benchmark_Uint256Mont_MulMontgomeryV2(b *testing.B) {
	var bench_x []Uint256 = CachedUint256.GetElements(pc_uint256_c, benchS)
	var bench_y []Uint256 = CachedUint256.GetElements(pc_uint256_c, benchS)
	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		DumpUint256[n%benchS].mulMontgomery_Unrolled_c(&bench_x[n%benchS], &bench_y[n%benchS])
	}
}

func benchmarkUint256Mont_ExponentiationMontgomerySlW(b *testing.B) {
	var bench_basis []Uint256 = CachedUint256.GetElements(pc_uint256_a, benchS)
	// NOTE: Using fully reduced exponents here. This is more meaningful (reduction for exponents would be modulo BaseFieldSize-1 is base != 0, so "reduced is a misnomer" -- this is just about the size of exponents)
	var bench_exponents []Uint256 = CachedUint256.GetElements(pc_uint256_f, benchS)
	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		DumpUint256[n%benchS].modularExponentiationSlidingWindowMontgomery_fa(&bench_basis[n%benchS], &bench_exponents[n%benchS])
	}
}

func benchmarkUint256Mont_ExponentiationMontgomerySqM(b *testing.B) {
	var bench_basis []Uint256 = CachedUint256.GetElements(pc_uint256_a, benchS)
	// NOTE: Using fully reduced exponents here. This is more meaningful (reduction for exponents would be modulo BaseFieldSize-1 is base != 0, so "reduced is a misnomer" -- this is just about the size of exponents)
	var bench_exponents []Uint256 = CachedUint256.GetElements(pc_uint256_f, benchS)
	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		DumpUint256[n%benchS].modularExponentiationSquareAndMultiplyMontgomery_fa(&bench_basis[n%benchS], &bench_exponents[n%benchS])
	}
}
