package bandersnatch

import (
	"testing"
)

const benchS = 1 << 8

func init() {
	assert(benchS <= benchSizeFe)
}

// BenchmarkEnsureBuildFlags is not really a benchmark.
// its only purpose is to cause Go's default benchmark runner to emit a diagnostic message if call counters are active.
func BenchmarkEnsureBuildFlags(b *testing.B) {
	if CallCountersActive {
		b.Skipf("Warning: call Counters are active in this build. These dominate the running times of fast operations such as field element additions.")
	} else {
		b.SkipNow()
	}
}

func BenchmarkDummyRead_64(b *testing.B) {
	var bench_x_64 []bsFieldElement_64 = getPrecomputedFieldElementSlice_64(1, benchS)
	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		_ = bench_x_64[n%benchS]
	}
}

func BenchmarkDummyReadStore_64(b *testing.B) {
	var bench_x_64 []bsFieldElement_64 = getPrecomputedFieldElementSlice_64(1, benchS)
	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		DumpFe_64[n&benchS] = bench_x_64[n%benchS]
	}
}

func BenchmarkAdd_64(b *testing.B) {
	var bench_x_64 []bsFieldElement_64 = getPrecomputedFieldElementSlice_64(1, benchS)
	var bench_y_64 []bsFieldElement_64 = getPrecomputedFieldElementSlice_64(2, benchS)
	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		DumpFe_64[n%benchS].Add(&bench_x_64[n%benchS], &bench_y_64[n%benchS])
	}
}

func BenchmarkAddEq_64(b *testing.B) {
	var bench_x_64 []bsFieldElement_64 = getPrecomputedFieldElementSlice_64(1, benchS)
	var bench_y_64 []bsFieldElement_64 = getPrecomputedFieldElementSlice_64(2, benchS)
	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		bench_x_64[n%benchS].AddEq(&bench_y_64[n%benchS])
	}
	b.StopTimer()
	// This is just to really ensure the compiler does not optimize things away.
	for n := 0; n < b.N; n++ {
		DumpFe_64[n%benchS] = bench_x_64[n%benchS]
	}
}

func BenchmarkSub_64(b *testing.B) {
	var bench_x_64 []bsFieldElement_64 = getPrecomputedFieldElementSlice_64(1, benchS)
	var bench_y_64 []bsFieldElement_64 = getPrecomputedFieldElementSlice_64(2, benchS)
	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		DumpFe_64[n%benchS].Sub(&bench_x_64[n%benchS], &bench_y_64[n%benchS])
	}
}

func BenchmarkSubEq_64(b *testing.B) {
	var bench_x_64 []bsFieldElement_64 = getPrecomputedFieldElementSlice_64(1, benchS)
	var bench_y_64 []bsFieldElement_64 = getPrecomputedFieldElementSlice_64(2, benchS)
	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		bench_x_64[n%benchS].SubEq(&bench_y_64[n%benchS])
	}
	b.StopTimer()
	// This is just to really ensure the compiler does not optimize things away.
	for n := 0; n < b.N; n++ {
		DumpFe_64[n%benchS] = bench_x_64[n%benchS]
	}
}

func BenchmarkMul_64(b *testing.B) {
	var bench_x_64 []bsFieldElement_64 = getPrecomputedFieldElementSlice_64(1, benchS)
	var bench_y_64 []bsFieldElement_64 = getPrecomputedFieldElementSlice_64(2, benchS)
	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		DumpFe_64[n%benchS].Mul(&bench_x_64[n%benchS], &bench_y_64[n%benchS])
	}
}

func BenchmarkMulEq_64(b *testing.B) {
	var bench_x_64 []bsFieldElement_64 = getPrecomputedFieldElementSlice_64(1, benchS)
	var bench_y_64 []bsFieldElement_64 = getPrecomputedFieldElementSlice_64(2, benchS)
	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		bench_x_64[n%benchS].MulEq(&bench_y_64[n%benchS])
	}
	b.StopTimer()
	// This is just to really ensure the compiler does not optimize things away.
	for n := 0; n < b.N; n++ {
		DumpFe_64[n%benchS] = bench_x_64[n%benchS]
	}
}

func BenchmarkMultiplyByFive_64(b *testing.B) {
	var bench_x_64 []bsFieldElement_64 = getPrecomputedFieldElementSlice_64(1, benchS)
	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		bench_x_64[n%benchS].multiply_by_five()
	}
	b.StopTimer()
	// This is just to really ensure the compiler does not optimize things away.
	for n := 0; n < b.N; n++ {
		DumpFe_64[n%benchS] = bench_x_64[n%benchS]
	}
}

func BenchmarkMultiplyByFiveNaive_64(b *testing.B) {
	var bench_x_64 []bsFieldElement_64 = getPrecomputedFieldElementSlice_64(1, benchS)
	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		var temp bsFieldElement_64
		temp.Add(&bench_x_64[n%benchS], &bench_x_64[n%benchS])
		temp.Add(&temp, &temp)
		bench_x_64[n%benchS].Add(&temp, &bench_x_64[n%benchS])
	}
	b.StopTimer()
	// This is just to really ensure the compiler does not optimize things away.
	for n := 0; n < b.N; n++ {
		DumpFe_64[n%benchS] = bench_x_64[n%benchS]
	}
}

func BenchmarkSquare_64(b *testing.B) {
	var bench_x_64 []bsFieldElement_64 = getPrecomputedFieldElementSlice_64(1, benchS)
	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		DumpFe_64[n%benchS].Square(&bench_x_64[n%benchS])
	}
}

func BenchmarkSquareEq_64(b *testing.B) {
	var bench_x_64 []bsFieldElement_64 = getPrecomputedFieldElementSlice_64(1, benchS)
	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		bench_x_64[n%benchS].SquareEq()
	}
	b.StopTimer()
	// This is just to really ensure the compiler does not optimize things away.
	for n := 0; n < b.N; n++ {
		DumpFe_64[n%benchS] = bench_x_64[n%benchS]
	}
}

func BenchmarkInv_64(b *testing.B) {
	var bench_x_64 []bsFieldElement_64 = getPrecomputedFieldElementSlice_64(1, benchS)
	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		DumpFe_64[n%benchS].Inv(&bench_x_64[n%benchS])
	}
}

func BenchmarkInvEq_64(b *testing.B) {
	var bench_x_64 []bsFieldElement_64 = getPrecomputedFieldElementSlice_64(1, benchS)
	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		bench_x_64[n%benchS].InvEq()
	}
	b.StopTimer()
	// This is just to really ensure the compiler does not optimize things away.
	for n := 0; n < b.N; n++ {
		DumpFe_64[n%benchS] = bench_x_64[n%benchS]
	}
}

func BenchmarkDivide_64(b *testing.B) {
	var bench_x_64 []bsFieldElement_64 = getPrecomputedFieldElementSlice_64(1, benchS)
	var bench_y_64 []bsFieldElement_64 = getPrecomputedFieldElementSlice_64(2, benchS)
	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		DumpFe_64[n%benchS].Divide(&bench_x_64[n%benchS], &bench_y_64[n%benchS])
	}
}

func BenchmarkDivideEq_64(b *testing.B) {
	var bench_x_64 []bsFieldElement_64 = getPrecomputedFieldElementSlice_64(1, benchS)
	var bench_y_64 []bsFieldElement_64 = getPrecomputedFieldElementSlice_64(2, benchS)
	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		bench_x_64[n%benchS].DivideEq(&bench_y_64[n%benchS])
	}
	b.StopTimer()
	// This is just to really ensure the compiler does not optimize things away.
	for n := 0; n < b.N; n++ {
		DumpFe_64[n%benchS] = bench_x_64[n%benchS]
	}
}

func BenchmarkIsEqual_64(b *testing.B) {
	var bench_x_64 []bsFieldElement_64 = getPrecomputedFieldElementSlice_64(1, benchS)
	var bench_y_64 []bsFieldElement_64 = getPrecomputedFieldElementSlice_64(2, benchS)
	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		DumpBools_Fe[n%benchS] = bench_x_64[n%benchS].IsEqual(&bench_y_64[n%benchS])
	}
}

func BenchmarkNeg_64(b *testing.B) {
	var bench_x_64 []bsFieldElement_64 = getPrecomputedFieldElementSlice_64(1, benchS)
	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		DumpFe_64[n%benchS].Neg(&bench_x_64[n%benchS])
	}
}

func BenchmarkNegEq_64(b *testing.B) {
	var bench_x_64 []bsFieldElement_64 = getPrecomputedFieldElementSlice_64(1, benchS)
	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		bench_x_64[n%benchS].NegEq()
	}
	b.StopTimer()
	// This is just to really ensure the compiler does not optimize things away.
	for n := 0; n < b.N; n++ {
		DumpFe_64[n%benchS] = bench_x_64[n%benchS]
	}
}

func BenchmarkSign_64(b *testing.B) {
	var dumpInt [benchSizeFe]int
	var bench_x_64 []bsFieldElement_64 = getPrecomputedFieldElementSlice_64(1, benchS)
	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		dumpInt[n%benchS] = bench_x_64[n%benchS].Sign()
	}
	b.StopTimer()
	// This is just to really ensure the compiler does not optimize things away.
	for n := 0; n < b.N; n++ {
		DumpBools_Fe[n%benchS] = (dumpInt[n%benchS] == 1)
	}
}

func BenchmarkJacobi_64(b *testing.B) {
	var dumpInt [benchSizeFe]int
	var bench_x_64 []bsFieldElement_64 = getPrecomputedFieldElementSlice_64(1, benchS)
	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		dumpInt[n%benchS] = bench_x_64[n%benchS].Jacobi()
	}
	b.StopTimer()
	// This is just to really ensure the compiler does not optimize things away.
	for n := 0; n < b.N; n++ {
		DumpBools_Fe[n%benchS] = (dumpInt[n%benchS] == 1)
	}
}

func BenchmarkSquareRoot_64(b *testing.B) {
	var bench_x_64 []bsFieldElement_64 = getPrecomputedFieldElementSlice_64(1, benchS)
	for i := 0; i < len(bench_x_64); i++ {
		bench_x_64[i].SquareEq()
	}
	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		DumpFe_64[n%benchS].SquareRoot(&bench_x_64[n%benchS])
	}
}
