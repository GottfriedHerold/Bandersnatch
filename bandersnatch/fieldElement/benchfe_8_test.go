package fieldElement

import (
	"testing"
)

// This file runs essentially the same benchmarks as benchfe_64_test.go does

/*
	var bench_x_8 []bsFieldElement_8 = getPrecomputedFieldElementSlice_8(1, benchS)
	var bench_y_8 []bsFieldElement_8 = getPrecomputedFieldElementSlice_8(2, benchS)
	prepareBenchmarkFieldElements(b)
*/

func BenchmarkDummyRead_8(b *testing.B) {
	var bench_x_8 []bsFieldElement_8 = getPrecomputedFieldElementSlice_8(1, benchS)
	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		_ = bench_x_8[n%benchS]
	}
}

func BenchmarkDummyReadStore_8(b *testing.B) {
	var bench_x_8 []bsFieldElement_8 = getPrecomputedFieldElementSlice_8(1, benchS)
	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		DumpFe_8[n%benchS] = bench_x_8[n%benchS]
	}
}

func BenchmarkAdd_8(b *testing.B) {
	var bench_x_8 []bsFieldElement_8 = getPrecomputedFieldElementSlice_8(1, benchS)
	var bench_y_8 []bsFieldElement_8 = getPrecomputedFieldElementSlice_8(2, benchS)
	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		DumpFe_8[n%benchS].Add(&bench_x_8[n%benchS], &bench_y_8[n%benchS])
	}
}

func BenchmarkAddEq_8(b *testing.B) {
	var bench_x_8 []bsFieldElement_8 = getPrecomputedFieldElementSlice_8(1, benchS)
	var bench_y_8 []bsFieldElement_8 = getPrecomputedFieldElementSlice_8(2, benchS)
	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		bench_x_8[n%benchS].AddEq(&bench_y_8[n%benchS])
	}
	b.StopTimer()
	// This is just to really ensure the compiler does not optimize things away.
	for n := 0; n < b.N; n++ {
		DumpFe_8[n%benchS] = bench_x_8[n%benchS]
	}
}

func BenchmarkSub_8(b *testing.B) {
	var bench_x_8 []bsFieldElement_8 = getPrecomputedFieldElementSlice_8(1, benchS)
	var bench_y_8 []bsFieldElement_8 = getPrecomputedFieldElementSlice_8(2, benchS)
	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		DumpFe_8[n%benchS].Sub(&bench_x_8[n%benchS], &bench_y_8[n%benchS])
	}
}

func BenchmarkSubEq_8(b *testing.B) {
	var bench_x_8 []bsFieldElement_8 = getPrecomputedFieldElementSlice_8(1, benchS)
	var bench_y_8 []bsFieldElement_8 = getPrecomputedFieldElementSlice_8(2, benchS)
	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		bench_x_8[n%benchS].SubEq(&bench_y_8[n%benchS])
	}
	b.StopTimer()
	// This is just to really ensure the compiler does not optimize things away.
	for n := 0; n < b.N; n++ {
		DumpFe_8[n%benchS] = bench_x_8[n%benchS]
	}
}

func BenchmarkMul_8(b *testing.B) {
	var bench_x_8 []bsFieldElement_8 = getPrecomputedFieldElementSlice_8(1, benchS)
	var bench_y_8 []bsFieldElement_8 = getPrecomputedFieldElementSlice_8(2, benchS)
	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		DumpFe_8[n%benchS].Mul(&bench_x_8[n%benchS], &bench_y_8[n%benchS])
	}
}

func BenchmarkMulEq_8(b *testing.B) {
	var bench_x_8 []bsFieldElement_8 = getPrecomputedFieldElementSlice_8(1, benchS)
	var bench_y_8 []bsFieldElement_8 = getPrecomputedFieldElementSlice_8(2, benchS)
	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		bench_x_8[n%benchS].MulEq(&bench_y_8[n%benchS])
	}
	b.StopTimer()
	// This is just to really ensure the compiler does not optimize things away.
	for n := 0; n < b.N; n++ {
		DumpFe_8[n%benchS] = bench_x_8[n%benchS]
	}
}

func BenchmarkSquare_8(b *testing.B) {
	var bench_x_8 []bsFieldElement_8 = getPrecomputedFieldElementSlice_8(1, benchS)
	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		DumpFe_8[n%benchS].Square(&bench_x_8[n%benchS])
	}
}

func BenchmarkSquareEq_8(b *testing.B) {
	var bench_x_8 []bsFieldElement_8 = getPrecomputedFieldElementSlice_8(1, benchS)
	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		bench_x_8[n%benchS].SquareEq()
	}
	b.StopTimer()
	// This is just to really ensure the compiler does not optimize things away.
	for n := 0; n < b.N; n++ {
		DumpFe_8[n%benchS] = bench_x_8[n%benchS]
	}
}

func BenchmarkInv_8(b *testing.B) {
	var bench_x_8 []bsFieldElement_8 = getPrecomputedFieldElementSlice_8(1, benchS)
	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		DumpFe_8[n%benchS].Inv(&bench_x_8[n%benchS])
	}
}

func BenchmarkInvEq_8(b *testing.B) {
	var bench_x_8 []bsFieldElement_8 = getPrecomputedFieldElementSlice_8(1, benchS)
	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		bench_x_8[n%benchS].InvEq()
	}
	b.StopTimer()
	// This is just to really ensure the compiler does not optimize things away.
	for n := 0; n < b.N; n++ {
		DumpFe_8[n%benchS] = bench_x_8[n%benchS]
	}
}

func BenchmarkDivide_8(b *testing.B) {
	var bench_x_8 []bsFieldElement_8 = getPrecomputedFieldElementSlice_8(1, benchS)
	var bench_y_8 []bsFieldElement_8 = getPrecomputedFieldElementSlice_8(2, benchS)
	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		DumpFe_8[n%benchS].Divide(&bench_x_8[n%benchS], &bench_y_8[n%benchS])
	}
}

func BenchmarkDivideEq_8(b *testing.B) {
	var bench_x_8 []bsFieldElement_8 = getPrecomputedFieldElementSlice_8(1, benchS)
	var bench_y_8 []bsFieldElement_8 = getPrecomputedFieldElementSlice_8(2, benchS)
	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		bench_x_8[n%benchS].DivideEq(&bench_y_8[n%benchS])
	}
	b.StopTimer()
	// This is just to really ensure the compiler does not optimize things away.
	for n := 0; n < b.N; n++ {
		DumpFe_8[n%benchS] = bench_x_8[n%benchS]
	}
}

func BenchmarkIsEqual_8(b *testing.B) {
	var bench_x_8 []bsFieldElement_8 = getPrecomputedFieldElementSlice_8(1, benchS)
	var bench_y_8 []bsFieldElement_8 = getPrecomputedFieldElementSlice_8(2, benchS)
	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		DumpBools_fe[n%benchS] = bench_x_8[n%benchS].IsEqual(&bench_y_8[n%benchS])
	}
}

func BenchmarkNeg_8(b *testing.B) {
	var bench_x_8 []bsFieldElement_8 = getPrecomputedFieldElementSlice_8(1, benchS)
	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		DumpFe_8[n%benchS].Neg(&bench_x_8[n%benchS])
	}
}

func BenchmarkNegEq_8(b *testing.B) {
	var bench_x_8 []bsFieldElement_8 = getPrecomputedFieldElementSlice_8(1, benchS)
	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		bench_x_8[n%benchS].NegEq()
	}
	b.StopTimer()
	// This is just to really ensure the compiler does not optimize things away.
	for n := 0; n < b.N; n++ {
		DumpFe_8[n%benchS] = bench_x_8[n%benchS]
	}
}

func BenchmarkSign_8(b *testing.B) {
	var dumpInt [dumpSizeBench_fe]int
	var bench_x_8 []bsFieldElement_8 = getPrecomputedFieldElementSlice_8(1, benchS)
	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		dumpInt[n%benchS] = bench_x_8[n%benchS].Sign()
	}
	b.StopTimer()
	// This is just to really ensure the compiler does not optimize things away.
	for n := 0; n < b.N; n++ {
		DumpBools_fe[n%benchS] = (dumpInt[n%benchS] == 1)
	}
}

func BenchmarkJacobi_8(b *testing.B) {
	var dumpInt [dumpSizeBench_fe]int
	var bench_x_8 []bsFieldElement_8 = getPrecomputedFieldElementSlice_8(1, benchS)
	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		dumpInt[n%benchS] = bench_x_8[n%benchS].Jacobi()
	}
	b.StopTimer()
	// This is just to really ensure the compiler does not optimize things away.
	for n := 0; n < b.N; n++ {
		DumpBools_fe[n%benchS] = (dumpInt[n%benchS] == 1)
	}
}

func BenchmarkSquareRoot_8(b *testing.B) {
	var bench_x_8 []bsFieldElement_8 = getPrecomputedFieldElementSlice_8(1, benchS)
	prepareBenchmarkFieldElements(b)
	for i := 0; i < benchS; i++ {
		bench_x_8[i].SquareEq()
	}
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		DumpFe_8[n%benchS].SquareRoot(&bench_x_8[n%benchS])
	}
}
