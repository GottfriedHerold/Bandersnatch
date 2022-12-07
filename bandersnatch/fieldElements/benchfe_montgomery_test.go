package fieldElements

import (
	"fmt"
	"testing"

	"github.com/GottfriedHerold/Bandersnatch/internal/testutils"
)

// This file is part of the fieldElements package. See the documentation of field_element.go for general remarks.

// This file is an old version of the benchmarking suite back from when Go did not even have generics.
// It only benchmarks the bsFieldElement_MontgomeryNonUnique implementation of the [FieldElementInterface] interface
// and does not cover the full interface. We keep it around to compare it against the full generic benchmarking suite, defined in bench_fieldelement_test.go
// This way we can estimate the overhead the use of generics in the full suite.

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
	var bench_x_64 []bsFieldElement_MontgomeryNonUnique = GetPrecomputedFieldElements[bsFieldElement_MontgomeryNonUnique](1, benchS)
	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		_ = bench_x_64[n%benchS]
	}
}

func BenchmarkDummyReadStore_64(b *testing.B) {
	var bench_x_64 []bsFieldElement_MontgomeryNonUnique = GetPrecomputedFieldElements[bsFieldElement_MontgomeryNonUnique](1, benchS)
	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		DumpFe_64[n%benchS] = bench_x_64[n%benchS]
	}
}

func BenchmarkAdd_64(b *testing.B) {
	var bench_x_64 []bsFieldElement_MontgomeryNonUnique = GetPrecomputedFieldElements[bsFieldElement_MontgomeryNonUnique](1, benchS)
	var bench_y_64 []bsFieldElement_MontgomeryNonUnique = GetPrecomputedFieldElements[bsFieldElement_MontgomeryNonUnique](2, benchS)
	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		DumpFe_64[n%benchS].Add(&bench_x_64[n%benchS], &bench_y_64[n%benchS])
	}
}

func BenchmarkAddEq_64(b *testing.B) {
	var bench_x_64 []bsFieldElement_MontgomeryNonUnique = GetPrecomputedFieldElements[bsFieldElement_MontgomeryNonUnique](1, benchS)
	var bench_y_64 []bsFieldElement_MontgomeryNonUnique = GetPrecomputedFieldElements[bsFieldElement_MontgomeryNonUnique](2, benchS)
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
	var bench_x_64 []bsFieldElement_MontgomeryNonUnique = GetPrecomputedFieldElements[bsFieldElement_MontgomeryNonUnique](1, benchS)
	var bench_y_64 []bsFieldElement_MontgomeryNonUnique = GetPrecomputedFieldElements[bsFieldElement_MontgomeryNonUnique](2, benchS)
	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		DumpFe_64[n%benchS].Sub(&bench_x_64[n%benchS], &bench_y_64[n%benchS])
	}
}

func BenchmarkSubEq_64(b *testing.B) {
	var bench_x_64 []bsFieldElement_MontgomeryNonUnique = GetPrecomputedFieldElements[bsFieldElement_MontgomeryNonUnique](1, benchS)
	var bench_y_64 []bsFieldElement_MontgomeryNonUnique = GetPrecomputedFieldElements[bsFieldElement_MontgomeryNonUnique](2, benchS)
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
	var bench_x_64 []bsFieldElement_MontgomeryNonUnique = GetPrecomputedFieldElements[bsFieldElement_MontgomeryNonUnique](1, benchS)
	var bench_y_64 []bsFieldElement_MontgomeryNonUnique = GetPrecomputedFieldElements[bsFieldElement_MontgomeryNonUnique](2, benchS)
	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		DumpFe_64[n%benchS].Mul(&bench_x_64[n%benchS], &bench_y_64[n%benchS])
	}
}

func BenchmarkMulEq_64(b *testing.B) {
	var bench_x_64 []bsFieldElement_MontgomeryNonUnique = GetPrecomputedFieldElements[bsFieldElement_MontgomeryNonUnique](1, benchS)
	var bench_y_64 []bsFieldElement_MontgomeryNonUnique = GetPrecomputedFieldElements[bsFieldElement_MontgomeryNonUnique](2, benchS)
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
	var bench_x_64 []bsFieldElement_MontgomeryNonUnique = GetPrecomputedFieldElements[bsFieldElement_MontgomeryNonUnique](1, benchS)
	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		bench_x_64[n%benchS].MulEqFive()
	}
	b.StopTimer()
	// This is just to really ensure the compiler does not optimize things away.
	for n := 0; n < b.N; n++ {
		DumpFe_64[n%benchS] = bench_x_64[n%benchS]
	}
}

func BenchmarkMultiplyByFiveNaive_64(b *testing.B) {
	var bench_x_64 []bsFieldElement_MontgomeryNonUnique = GetPrecomputedFieldElements[bsFieldElement_MontgomeryNonUnique](1, benchS)
	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		var temp bsFieldElement_MontgomeryNonUnique
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
	var bench_x_64 []bsFieldElement_MontgomeryNonUnique = GetPrecomputedFieldElements[bsFieldElement_MontgomeryNonUnique](1, benchS)
	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		DumpFe_64[n%benchS].Square(&bench_x_64[n%benchS])
	}
}

func BenchmarkSquareEq_64(b *testing.B) {
	var bench_x_64 []bsFieldElement_MontgomeryNonUnique = GetPrecomputedFieldElements[bsFieldElement_MontgomeryNonUnique](1, benchS)
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
	var bench_x_64 []bsFieldElement_MontgomeryNonUnique = GetPrecomputedFieldElements[bsFieldElement_MontgomeryNonUnique](1, benchS)
	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		DumpFe_64[n%benchS].Inv(&bench_x_64[n%benchS])
	}
}

func BenchmarkInvEq_64(b *testing.B) {
	var bench_x_64 []bsFieldElement_MontgomeryNonUnique = GetPrecomputedFieldElements[bsFieldElement_MontgomeryNonUnique](1, benchS)
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
	var bench_x_64 []bsFieldElement_MontgomeryNonUnique = GetPrecomputedFieldElements[bsFieldElement_MontgomeryNonUnique](1, benchS)
	var bench_y_64 []bsFieldElement_MontgomeryNonUnique = GetPrecomputedFieldElements[bsFieldElement_MontgomeryNonUnique](2, benchS)
	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		DumpFe_64[n%benchS].Divide(&bench_x_64[n%benchS], &bench_y_64[n%benchS])
	}
}

func BenchmarkDivideEq_64(b *testing.B) {
	var bench_x_64 []bsFieldElement_MontgomeryNonUnique = GetPrecomputedFieldElements[bsFieldElement_MontgomeryNonUnique](1, benchS)
	var bench_y_64 []bsFieldElement_MontgomeryNonUnique = GetPrecomputedFieldElements[bsFieldElement_MontgomeryNonUnique](2, benchS)
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
	var bench_x_64 []bsFieldElement_MontgomeryNonUnique = GetPrecomputedFieldElements[bsFieldElement_MontgomeryNonUnique](1, benchS)
	var bench_y_64 []bsFieldElement_MontgomeryNonUnique = GetPrecomputedFieldElements[bsFieldElement_MontgomeryNonUnique](2, benchS)
	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		DumpBools_fe[n%benchS] = bench_x_64[n%benchS].IsEqual(&bench_y_64[n%benchS])
	}
}

func BenchmarkNeg_64(b *testing.B) {
	var bench_x_64 []bsFieldElement_MontgomeryNonUnique = GetPrecomputedFieldElements[bsFieldElement_MontgomeryNonUnique](1, benchS)
	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		DumpFe_64[n%benchS].Neg(&bench_x_64[n%benchS])
	}
}

func BenchmarkNegEq_64(b *testing.B) {
	var bench_x_64 []bsFieldElement_MontgomeryNonUnique = GetPrecomputedFieldElements[bsFieldElement_MontgomeryNonUnique](1, benchS)
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
	var dumpInt [dumpSizeBench_fe]int
	var bench_x_64 []bsFieldElement_MontgomeryNonUnique = GetPrecomputedFieldElements[bsFieldElement_MontgomeryNonUnique](1, benchS)
	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		dumpInt[n%benchS] = bench_x_64[n%benchS].Sign()
	}
	b.StopTimer()
	// This is just to really ensure the compiler does not optimize things away.
	for n := 0; n < b.N; n++ {
		DumpBools_fe[n%benchS] = (dumpInt[n%benchS] == 1)
	}
}

func BenchmarkJacobi_64(b *testing.B) {
	var bench_x_64 []bsFieldElement_MontgomeryNonUnique = GetPrecomputedFieldElements[bsFieldElement_MontgomeryNonUnique](1, benchS)
	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		DumpInt[n%benchS] = bench_x_64[n%benchS].Jacobi()
	}
}

func BenchmarkSquareRoot_64(b *testing.B) {
	var bench_x_64 []bsFieldElement_MontgomeryNonUnique = GetPrecomputedFieldElements[bsFieldElement_MontgomeryNonUnique](1, benchS)
	for i := 0; i < len(bench_x_64); i++ {
		bench_x_64[i].SquareEq()
	}
	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		DumpFe_64[n%benchS].SquareRoot(&bench_x_64[n%benchS])
	}
}

func BenchmarkMultiInversion(bOuter *testing.B) {
	var bench_x_64 []bsFieldElement_MontgomeryNonUnique = GetPrecomputedFieldElements[bsFieldElement_MontgomeryNonUnique](1, benchS+256)
	testutils.Assert(benchS >= 256)
	batchSizes := []int{1, 2, 4, 16, 64, 256}
	makeBenchmarkFunctionMultiInvertEqSlice := func(batchSize int) func(*testing.B) {
		return func(bInner *testing.B) {
			prepareBenchmarkFieldElements(bInner)
			for n := 0; n < bInner.N; n++ {
				MultiInvertEqSlice(bench_x_64[n%benchS : n%benchS+batchSize])
			}
		}
	}
	makeBenchmarkFunctionMultiInvertEq := func(batchSize int) func(*testing.B) {
		return func(bInner *testing.B) {
			var Ptrs [benchS + 256]*bsFieldElement_MontgomeryNonUnique
			for i := 0; i < len(Ptrs); i++ {
				Ptrs[i] = &bench_x_64[i]
			}
			prepareBenchmarkFieldElements(bInner)
			for n := 0; n < bInner.N; n++ {
				MultiInvertEq(Ptrs[n%benchS : n%benchS+batchSize]...)
			}
		}
	}
	prepareBenchmarkFieldElements(bOuter)
	for _, batchSize := range batchSizes {
		fun := makeBenchmarkFunctionMultiInvertEqSlice(batchSize)
		tag := fmt.Sprintf("MultiInvertEqSlice of size %v", batchSize)
		bOuter.Run(tag, fun)
	}
	for _, batchSize := range batchSizes {
		fun := makeBenchmarkFunctionMultiInvertEq(batchSize)
		tag := fmt.Sprintf("MultiInvertEq of size %v", batchSize)
		bOuter.Run(tag, fun)
	}

}
