package fieldElements

import "testing"

func Benchmark_uint256_MontgomeryFuns(b *testing.B) {
	b.Run("ToNonMontgomery", benchmarkToNonMontgomery)
	b.Run("FromMontgomery", benchmarkFromMontgomery)
	b.Run("MulFourOne", benchmark_mul_four_one_64)
	b.Run("AddMulShift", benchmark_add_mul_shift_64)
	b.Run("Mul uint256 x uint64, target", benchmark_mul256by64)
	b.Run("Montgomery step (est.)", benchmark_montgomery_step64)
	b.Run("shiftOnce (inaccurate)", benchmark_shift_once)
	b.Run("Montgomery Mul V1", benchmark_MulMontgomery)
	b.Run("Montgomery Mul V2", benchmark_MulMontgomeryV2)
}

func benchmarkToNonMontgomery(b *testing.B) {
	var bench_x []Uint256 = CachedUint256.GetElements(pc_uint256_c, benchS)
	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		DumpUint256[n%benchS] = bench_x[n%benchS].ToNonMontgomery_fc()
	}
}

func benchmarkFromMontgomery(b *testing.B) {
	var bench_x []Uint256 = CachedUint256.GetElements(pc_uint256_c, benchS)
	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		DumpUint256[n%benchS].FromMontgomeryRepresentation_fc(&bench_x[n%benchS])
	}
}

func benchmark_mul_four_one_64(b *testing.B) {
	var bench_x []Uint256 = CachedUint256.GetElements(pc_uint256_a, benchS)
	var bench_y []uint64 = CachedUint64.GetElements(1, benchS)
	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		DumpUint64[n%benchS], DumpUint256[n%benchS] = mul_four_one_64(&bench_x[n%benchS], bench_y[n%benchS])
	}
}

func benchmark_add_mul_shift_64(b *testing.B) {
	var bench_x []Uint256 = CachedUint256.GetElements(pc_uint256_a, benchS)
	var bench_y []uint64 = CachedUint64.GetElements(1, benchS)
	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		DumpUint64[n%benchS] = add_mul_shift_64(&DumpUint256[n%benchS], &bench_x[n%benchS], bench_y[n%benchS])
	}
}

func benchmark_shift_once(b *testing.B) {
	var bench_x []Uint256 = CachedUint256.GetElements(pc_uint256_a, benchS)
	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		DumpUint64[n%benchS] = bench_x[n%benchS].ShiftRight_64() // modifies bench_x
	}
}

func benchmark_mul256by64(b *testing.B) {
	var bench_x []Uint256 = CachedUint256.GetElements(pc_uint256_a, benchS)
	var bench_y []uint64 = CachedUint64.GetElements(1, benchS)
	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		LongMul256By64(&DumpUint320[n%benchS], &bench_x[n%benchS], bench_y[n%benchS])
	}
}

func benchmark_montgomery_step64(b *testing.B) {
	var bench_x []Uint256 = CachedUint256.GetElements(pc_uint256_a, benchS)
	var bench_y []uint64 = CachedUint64.GetElements(1, benchS)
	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		montgomery_step_64(&bench_x[n%benchS], bench_y[n%benchS])
	}
}

func benchmark_MulMontgomery(b *testing.B) {
	var bench_x []Uint256 = CachedUint256.GetElements(pc_uint256_c, benchS)
	var bench_y []Uint256 = CachedUint256.GetElements(pc_uint256_c, benchS)
	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		DumpUint256[n%benchS].MulMontgomerySlow_c(&bench_x[n%benchS], &bench_y[n%benchS])
	}
}

func benchmark_MulMontgomeryV2(b *testing.B) {
	var bench_x []Uint256 = CachedUint256.GetElements(pc_uint256_c, benchS)
	var bench_y []Uint256 = CachedUint256.GetElements(pc_uint256_c, benchS)
	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		DumpUint256[n%benchS].MulMontgomery_c(&bench_x[n%benchS], &bench_y[n%benchS])
	}
}
