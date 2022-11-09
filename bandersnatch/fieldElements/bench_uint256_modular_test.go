package fieldElements

import "testing"

func Benchmark_uint256_Modular(b *testing.B) {
	b.Run("Add_b (conditional subtraction)", benchmarkAddAndReduce_b)
	b.Run("Add_c (conditional subtraction)", benchmarkAddAndReduce_c)
	b.Run("AddEq_a (Luan's reduce and check)", benchmarkAddEqAndReduce_a)
	b.Run("Sub_c (conditional subtraction)", benchmarkSubAndReduce_c)
	b.Run("Sub_b (pre-conditional subtraction)", benchmarkSubAndReduce_b)
	b.Run("SubEq_a (Luan's reduce and check)", benchmarkSubEqAndReduce_a)
	b.Run("Invert_a (Luan from HAC)", benchmarkModularInverse_a)
	b.Run("Reduce_ca (conditional subtraction)", benchmark_Copy_Reduce_ca)
	b.Run("Reduce_fb (conditional subtraction)", benchmark_Copy_Reduce_fb)
	b.Run("IsFullyReduced_a (if-chain)", benchmark_IsFullyReduced_a)
	b.Run("Barret512->256_a", benchmark_BarretReduction512_a)
	b.Run("BarretReduce_fa", benchmark_Copy_Reduce_barret_fa)
	b.Run("ComputeNeg_f (Reduce and check)", benchmark_ComputeModularNegative_f)
	b.Run("DoubleEq_a (Reduce and check)", benchmark_Copy_DoubleEqAndReduce_a)
	b.Run("MulEq_a (Barret)", benchmark_MulEqBarret_a)
	b.Run("SquareEq_a (Barret)", benchmark_Copy_SquareEqBarret_a)
	
}

// TODO: Move to different file:

var (
	pc_uint256_a CachedPRGUint256Key = CachedPRGUint256Key{
		seed:         1,
		allowedRange: twoTo256_Int,
	}
	pc_uint256_b CachedPRGUint256Key = CachedPRGUint256Key{
		seed:         1,
		allowedRange: doubleBaseFieldSize_Int,
	}
	pc_uint256_c CachedPRGUint256Key = CachedPRGUint256Key{
		seed:         1,
		allowedRange: montgomeryRepBound_Int,
	}
	pc_uint256_f CachedPRGUint256Key = CachedPRGUint256Key{
		seed:         1,
		allowedRange: BaseFieldSize_Int,
	}
)

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

func benchmarkAddAndReduce_b(b *testing.B) {
	var bench_x []uint256 = CachedUint256.GetElements(pc_uint256_b, benchS)
	var bench_y []uint256 = CachedUint256.GetElements(pc_uint256_b, benchS)
	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		DumpUint256[n%benchS].AddAndReduce_b_c(&bench_x[n%benchS], &bench_y[n%benchS])
	}
}

func benchmarkAddAndReduce_c(b *testing.B) {
	var bench_x []uint256 = CachedUint256.GetElements(pc_uint256_c, benchS)
	var bench_y []uint256 = CachedUint256.GetElements(pc_uint256_c, benchS)
	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		DumpUint256[n%benchS].AddAndReduce_b_c(&bench_x[n%benchS], &bench_y[n%benchS])
	}
}

func benchmarkAddEqAndReduce_a(b *testing.B) {
	var bench_x []uint256 = CachedUint256.GetElements(pc_uint256_a, benchS)
	var bench_y []uint256 = CachedUint256.GetElements(pc_uint256_a, benchS)
	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		bench_x[n%benchS].AddEqAndReduce_a(&bench_y[n%benchS])
	}
	b.StopTimer()
	for n := 0; n < b.N; n++ {
		DumpUint256[n%benchS] = bench_x[n%benchS]
	}
}

func benchmarkSubAndReduce_c(b *testing.B) {
	var bench_x []uint256 = CachedUint256.GetElements(pc_uint256_c, benchS)
	var bench_y []uint256 = CachedUint256.GetElements(pc_uint256_c, benchS)
	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		DumpUint256[n%benchS].SubAndReduce_c(&bench_x[n%benchS], &bench_y[n%benchS])
	}
}

func benchmarkSubAndReduce_b(b *testing.B) {
	var bench_x []uint256 = CachedUint256.GetElements(pc_uint256_b, benchS)
	var bench_y []uint256 = CachedUint256.GetElements(pc_uint256_b, benchS)
	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		DumpUint256[n%benchS].SubAndReduce_b(&bench_x[n%benchS], &bench_y[n%benchS])
	}
}

func benchmarkSubEqAndReduce_a(b *testing.B) {
	var bench_x []uint256 = CachedUint256.GetElements(pc_uint256_a, benchS)
	var bench_y []uint256 = CachedUint256.GetElements(pc_uint256_a, benchS)
	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		bench_x[n%benchS].SubEqAndReduce_a(&bench_y[n%benchS])
	}
	b.StopTimer()
	for n := 0; n < b.N; n++ {
		DumpUint256[n%benchS] = bench_x[n%benchS]
	}
}

func benchmarkModularInverse_a(b *testing.B) {
	var bench_x []uint256 = CachedUint256.GetElements(pc_uint256_a, benchS)
	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		DumpUint256[n%benchS].ModularInverse_a(&bench_x[n%benchS])
	}
}

func benchmark_Copy_Reduce_ca(b *testing.B) {
	var bench_x []uint256 = CachedUint256.GetElements(pc_uint256_a, benchS)
	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		DumpUint256[n%benchS] = bench_x[n%benchS]
		DumpUint256[n%benchS].reduce_ca()
	}
}

func benchmark_Copy_Reduce_fb(b *testing.B) {
	var bench_x []uint256 = CachedUint256.GetElements(pc_uint256_b, benchS)
	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		DumpUint256[n%benchS] = bench_x[n%benchS]
		DumpUint256[n%benchS].reduce_fb()
	}
}

func benchmark_IsFullyReduced_a(b *testing.B) {
	var bench_x []uint256 = CachedUint256.GetElements(pc_uint256_a, benchS)
	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		DumpBools_fe[n%benchS] = bench_x[n%benchS].is_fully_reduced()
	}
}

func benchmark_Copy_Reduce_barret_fa(b *testing.B) {
	var bench_x []uint256 = CachedUint256.GetElements(pc_uint256_a, benchS)
	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		DumpUint256[n%benchS] = bench_x[n%benchS]
		DumpUint256[n%benchS].reduceBarret_fa()
	}
}

// benchmark Barret reduction from [0,2**512) to [0..2**256) range
func benchmark_BarretReduction512_a(b *testing.B) {
	var bench_x []uint512 = CachedUint512.GetElements(10, benchS)
	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		DumpUint256[n%benchS].ReduceUint512ToUint256_a(bench_x[n%benchS]) // pass-by-value. This is dubious, but that's what Luans code does and we want to benchmark as-is first.
	}
}

// deprecated function:
func benchmark_ComputeModularNegative_f(b *testing.B) {
	var bench_x []uint256 = CachedUint256.GetElements(pc_uint256_f, benchS)
	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		DumpUint256[n%benchS] = bench_x[n%benchS].ComputeModularNegative_Weak_f()
	}
}

func benchmark_Copy_DoubleEqAndReduce_a(b *testing.B) {
	var bench_x []uint256 = CachedUint256.GetElements(pc_uint256_a, benchS)
	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		DumpUint256[n%benchS] = bench_x[n%benchS]
		DumpUint256[n%benchS].DoubleEqAndReduce_a()
	}
}

func benchmark_MulEqBarret_a(b *testing.B) {
	var bench_x []uint256 = CachedUint256.GetElements(pc_uint256_a, benchS)
	var bench_y []uint256 = CachedUint256.GetElements(pc_uint256_a, benchS)
	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		DumpUint256[n%benchS] = bench_x[n%benchS]
		DumpUint256[n%benchS].MulEqAndReduce_a(&bench_y[n%benchS])
	}
}

func benchmark_Copy_SquareEqBarret_a(b *testing.B) {
	var bench_x []uint256 = CachedUint256.GetElements(pc_uint256_a, benchS)
	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		DumpUint256[n%benchS] = bench_x[n%benchS]
		DumpUint256[n%benchS].SquareEqAndReduce_a()
	}
}

