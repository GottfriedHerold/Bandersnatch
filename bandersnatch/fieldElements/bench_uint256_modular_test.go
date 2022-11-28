package fieldElements

import "testing"

// This file is part of the fieldElements package and contains the benchmarks for the functions defined in uint256_modular.go
// This means we benchmark functions defined on uint256 that perfom arithmetic operations that work modulo the BaseFieldSize
// Note that this is not the same as benchmarking field elements themselves

// Benchmarks all follow the same pattern in order to make the overhead comparable.
// We benchmark functions for (fixed, pseudo-)random inputs that satisfy the preconditions of the functions.
// NOTE: We try to keep the order the same as the defintions in uint256_modular.go

func Benchmark_uint256_Modular(b *testing.B) {
	b.Run("Add_b (conditional subtraction)", benchmarkUint256m_AddAndReduce_b)
	b.Run("Add_c (conditional subtraction)", benchmarkUint256m_AddAndReduce_c)
	b.Run("AddEq_a (Luan's reduce and check)", benchmarkUint256m_AddEqAndReduce_a)
	b.Run("Sub_c (conditional subtraction)", benchmarkUint256m_SubAndReduce_c)
	b.Run("Sub_b (pre-conditional subtraction)", benchmarkUint256m_SubAndReduce_b)
	b.Run("SubEq_a (Luan's reduce and check)", benchmarkUint256m_SubEqAndReduce_a)
	b.Run("Invert_a (HAC version with standard improvement)", benchmarkUint256m_ModularInverse_a)
	b.Run("Reduce_ca (conditional subtraction)", benchmarkUint256m_CopyAndReduce_ca)
	b.Run("Reduce_fb (conditional subtraction)", benchmarkUint256m_CopyAndReduce_fb)
	b.Run("IsFullyReduced_a (if-chain)", benchmarkUint256m_IsFullyReduced_a)
	b.Run("Barret512->256_a", benchmarkUint256m_Reduction512To256_a)
	b.Run("BarretReduce_fa", benchmarkUint256m_CopyAndReduceBarret_fa)
	b.Run("ComputeNeg_a (Reduce and check)", benchmark_ComputeModularNegative_f)
	b.Run("DoubleEq_a (Reduce and check)", benchmarkUint256m_CopyAndDoubleEqAndReduce_a)
	b.Run("MulEq_a (Barret)", benchmarkUint256m_MulEqBarret_a)
	b.Run("Mul_a (Barret)", benchmarkUint256m_MulBarret_a)
	b.Run("SquareEq_a (Barret)", benchmarkUint256m_CopyAndSquareEqBarret_a)
	b.Run("Square_a (Barret)", benchmarkUint256m_SquareBarret_a)
	b.Run("Jacobi symbol (simple binary-gcd-like)", benchmarkUint256m_JacobiV1_a)
	b.Run("Exponentiation", benchmarkUint256m_Exponentiation)
}

// For Copy-And-Pasting
/*
	var bench_x []uint256 = CachedUint256.GetElements(pc_uint256_, benchS)
	var bench_y []uint256 = CachedUint256.GetElements(pc_uint256_, benchS)
	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		DumpUint256[n%benchS]._(&bench_x[n%benchS], &bench_y[n%benchS])
	}
*/

/*
	var bench_x []uint256 = CachedUint256.GetElements(pc_uint256_, benchS)
	var bench_y []uint256 = CachedUint256.GetElements(pc_uint256_, benchS)
	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		bench_x[n%benchS]._(&bench_y[n%benchS])
	}
	b.StopTimer()
	for n := 0; n < b.N; n++ {
		DumpUint256[n%benchS] = bench_x[n%benchS]
	}
*/

func benchmarkUint256m_AddAndReduce_b(b *testing.B) {
	var bench_x []Uint256 = CachedUint256.GetElements(pc_uint256_b, benchS)
	var bench_y []Uint256 = CachedUint256.GetElements(pc_uint256_b, benchS)
	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		DumpUint256[n%benchS].addAndReduce_b_c(&bench_x[n%benchS], &bench_y[n%benchS])
	}
}

func benchmarkUint256m_AddAndReduce_c(b *testing.B) {
	var bench_x []Uint256 = CachedUint256.GetElements(pc_uint256_c, benchS)
	var bench_y []Uint256 = CachedUint256.GetElements(pc_uint256_c, benchS)
	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		DumpUint256[n%benchS].addAndReduce_b_c(&bench_x[n%benchS], &bench_y[n%benchS])
	}
}

func benchmarkUint256m_AddEqAndReduce_a(b *testing.B) {
	var bench_x []Uint256 = CachedUint256.GetElements(pc_uint256_a, benchS)
	var bench_y []Uint256 = CachedUint256.GetElements(pc_uint256_a, benchS)
	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		bench_x[n%benchS].AddEqAndReduce_a(&bench_y[n%benchS])
	}
	b.StopTimer()
	for n := 0; n < b.N; n++ {
		DumpUint256[n%benchS] = bench_x[n%benchS]
	}
}

func benchmarkUint256m_SubAndReduce_c(b *testing.B) {
	var bench_x []Uint256 = CachedUint256.GetElements(pc_uint256_c, benchS)
	var bench_y []Uint256 = CachedUint256.GetElements(pc_uint256_c, benchS)
	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		DumpUint256[n%benchS].SubAndReduce_c(&bench_x[n%benchS], &bench_y[n%benchS])
	}
}

func benchmarkUint256m_SubAndReduce_b(b *testing.B) {
	var bench_x []Uint256 = CachedUint256.GetElements(pc_uint256_b, benchS)
	var bench_y []Uint256 = CachedUint256.GetElements(pc_uint256_b, benchS)
	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		DumpUint256[n%benchS].SubAndReduce_b(&bench_x[n%benchS], &bench_y[n%benchS])
	}
}

func benchmarkUint256m_SubEqAndReduce_a(b *testing.B) {
	var bench_x []Uint256 = CachedUint256.GetElements(pc_uint256_a, benchS)
	var bench_y []Uint256 = CachedUint256.GetElements(pc_uint256_a, benchS)
	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		bench_x[n%benchS].SubEqAndReduce_a(&bench_y[n%benchS])
	}
	b.StopTimer()
	for n := 0; n < b.N; n++ {
		DumpUint256[n%benchS] = bench_x[n%benchS]
	}
}

func benchmarkUint256m_ModularInverse_a(b *testing.B) {
	var bench_x []Uint256 = CachedUint256.GetElements(pc_uint256_a, benchS)
	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		DumpUint256[n%benchS].ModularInverse_a_NAIVEHAC(&bench_x[n%benchS])
	}
}

func benchmarkUint256m_CopyAndReduce_ca(b *testing.B) {
	var bench_x []Uint256 = CachedUint256.GetElements(pc_uint256_a, benchS)
	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		DumpUint256[n%benchS] = bench_x[n%benchS]
		DumpUint256[n%benchS].Reduce_ca()
	}
}

func benchmarkUint256m_CopyAndReduce_fb(b *testing.B) {
	var bench_x []Uint256 = CachedUint256.GetElements(pc_uint256_b, benchS)
	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		DumpUint256[n%benchS] = bench_x[n%benchS]
		DumpUint256[n%benchS].Reduce_fb()
	}
}

func benchmarkUint256m_IsFullyReduced_a(b *testing.B) {
	var bench_x []Uint256 = CachedUint256.GetElements(pc_uint256_a, benchS)
	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		DumpBools_fe[n%benchS] = bench_x[n%benchS].is_fully_reduced()
	}
}

func benchmarkUint256m_CopyAndReduceBarret_fa(b *testing.B) {
	var bench_x []Uint256 = CachedUint256.GetElements(pc_uint256_a, benchS)
	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		DumpUint256[n%benchS] = bench_x[n%benchS]
		DumpUint256[n%benchS].reduceBarret_fa()
	}
}

// benchmark Barret reduction from [0,2**512) to [0..2**256) range
func benchmarkUint256m_Reduction512To256_a(b *testing.B) {
	var bench_x []Uint512 = CachedUint512.GetElements(10, benchS)
	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		DumpUint256[n%benchS].ReduceUint512ToUint256_a(bench_x[n%benchS]) // pass-by-value. This is dubious, but that's what Luans code does and we want to benchmark as-is first.
	}
}

// deprecated function:
func benchmark_ComputeModularNegative_f(b *testing.B) {
	var bench_x []Uint256 = CachedUint256.GetElements(pc_uint256_a, benchS)
	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		DumpUint256[n%benchS] = bench_x[n%benchS].ComputeModularNegative_Weak_a()
	}
}

func benchmarkUint256m_CopyAndDoubleEqAndReduce_a(b *testing.B) {
	var bench_x []Uint256 = CachedUint256.GetElements(pc_uint256_a, benchS)
	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		DumpUint256[n%benchS] = bench_x[n%benchS]
		DumpUint256[n%benchS].DoubleEqAndReduce_a()
	}
}

func benchmarkUint256m_MulEqBarret_a(b *testing.B) {
	var bench_x []Uint256 = CachedUint256.GetElements(pc_uint256_a, benchS)
	var bench_y []Uint256 = CachedUint256.GetElements(pc_uint256_a, benchS)
	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		DumpUint256[n%benchS] = bench_x[n%benchS]
		DumpUint256[n%benchS].MulEqAndReduce_a(&bench_y[n%benchS])
	}
}

func benchmarkUint256m_CopyAndSquareEqBarret_a(b *testing.B) {
	var bench_x []Uint256 = CachedUint256.GetElements(pc_uint256_a, benchS)
	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		DumpUint256[n%benchS] = bench_x[n%benchS]
		DumpUint256[n%benchS].SquareEqAndReduce_a()
	}
}

func benchmarkUint256m_MulBarret_a(b *testing.B) {
	var bench_x []Uint256 = CachedUint256.GetElements(pc_uint256_a, benchS)
	var bench_y []Uint256 = CachedUint256.GetElements(pc_uint256_a, benchS)
	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		DumpUint256[n%benchS].MulAndReduce_a(&bench_x[n%benchS], &bench_y[n%benchS])
	}
}

func benchmarkUint256m_SquareBarret_a(b *testing.B) {
	var bench_x []Uint256 = CachedUint256.GetElements(pc_uint256_a, benchS)
	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		DumpUint256[n%benchS].SquareAndReduce_a(&bench_x[n%benchS])
	}
}

func benchmarkUint256m_JacobiV1_a(b *testing.B) {
	var bench_x []Uint256 = CachedUint256.GetElements(pc_uint256_a, benchS)
	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		DumpUint64[n%benchS] = uint64(bench_x[n%benchS].jacobiV1_a())
	}
}

func benchmarkUint256m_Exponentiation(b *testing.B) {
	var bench_basis []Uint256 = CachedUint256.GetElements(pc_uint256_a, benchS)
	var bench_exponents []Uint256 = CachedUint256.GetElements(pc_uint256_f, benchS) // NOTE: Using fully reduced exponents here. This is more meaningful
	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		DumpUint256[n%benchS].ModularExponentiation_fa(&bench_basis[n%benchS], &bench_exponents[n%benchS])
	}
}
