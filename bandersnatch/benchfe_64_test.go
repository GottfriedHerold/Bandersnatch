package bandersnatch

import (
	"testing"
)

func BenchmarkEnsureBuildFlags(b *testing.B) {
	if CallCountersActive {
		b.Skipf("Warning: call Counters are active in this build. These dominate the running times of fast operations such as field element additions.")
	} else {
		b.SkipNow()
	}
}

func BenchmarkDummyRead_64(b *testing.B) {
	prepareBenchTests_64(b)
	for n := 0; n < b.N; n++ {
		_ = bench_x_64[n%benchS]
	}
}

func BenchmarkAdd_64(b *testing.B) {
	prepareBenchTests_64(b)
	for n := 0; n < b.N; n++ {
		DumpFe_64[n%benchS].Add(&bench_x_64[n%benchS], &bench_y_64[n%benchS])
	}
}

func BenchmarkAddEq_64(b *testing.B) {
	prepareBenchTests_64(b)
	for n := 0; n < b.N; n++ {
		bench_x_64[n%benchS].AddEq(&bench_y_64[n%benchS])
	}
}

func BenchmarkSub_64(b *testing.B) {
	prepareBenchTests_64(b)
	for n := 0; n < b.N; n++ {
		DumpFe_64[n%benchS].Sub(&bench_x_64[n%benchS], &bench_y_64[n%benchS])
	}
}

func BenchmarkSubEq_64(b *testing.B) {
	prepareBenchTests_64(b)
	for n := 0; n < b.N; n++ {
		bench_x_64[n%benchS].SubEq(&bench_y_64[n%benchS])
	}
}

func BenchmarkMul_64(b *testing.B) {
	prepareBenchTests_64(b)
	for n := 0; n < b.N; n++ {
		DumpFe_64[n%benchS].Mul(&bench_x_64[n%benchS], &bench_y_64[n%benchS])
	}
}

func BenchmarkMulEq_64(b *testing.B) {
	prepareBenchTests_64(b)
	for n := 0; n < b.N; n++ {
		bench_x_64[n%benchS].MulEq(&bench_y_64[n%benchS])
	}
}

func BenchmarkMultiplyByFive_64(b *testing.B) {
	prepareBenchTests_64(b)
	for n := 0; n < b.N; n++ {
		bench_x_64[n%benchS].multiply_by_five()
	}
}

func BenchmarkMultiplyByFiveNaive_64(b *testing.B) {
	prepareBenchTests_64(b)
	for n := 0; n < b.N; n++ {
		var temp bsFieldElement_64
		temp.Add(&bench_x_64[n%benchS], &bench_x_64[n%benchS])
		temp.Add(&temp, &temp)
		bench_x_64[n%benchS].Add(&temp, &bench_x_64[n%benchS])
	}
}

func BenchmarkSquare_64(b *testing.B) {
	prepareBenchTests_64(b)
	for n := 0; n < b.N; n++ {
		DumpFe_64[n%benchS].Square(&bench_x_64[n%benchS])
	}
}

func BenchmarkSquareEq_64(b *testing.B) {
	prepareBenchTests_64(b)
	for n := 0; n < b.N; n++ {
		bench_x_64[n%benchS].SquareEq()
	}
}

func BenchmarkInv_64(b *testing.B) {
	prepareBenchTests_64(b)
	for n := 0; n < b.N; n++ {
		DumpFe_64[n%benchS].Inv(&bench_x_64[n%benchS])
	}
}

func BenchmarkInvEq_64(b *testing.B) {
	prepareBenchTests_64(b)
	for n := 0; n < b.N; n++ {
		bench_x_64[n%benchS].InvEq()
	}
}

func BenchmarkDivide_64(b *testing.B) {
	prepareBenchTests_64(b)
	for n := 0; n < b.N; n++ {
		DumpFe_64[n%benchS].Divide(&bench_x_64[n%benchS], &bench_y_64[n%benchS])
	}
}

func BenchmarkDivideEq_64(b *testing.B) {
	prepareBenchTests_64(b)
	for n := 0; n < b.N; n++ {
		bench_x_64[n%benchS].DivideEq(&bench_y_64[n%benchS])
	}
}

func BenchmarkIsEqual_64(b *testing.B) {
	prepareBenchTests_64(b)
	for n := 0; n < b.N; n++ {
		DumpBools[n%benchS] = bench_x_64[n%benchS].IsEqual(&bench_y_64[n%benchS])
	}
}

func BenchmarkNeg_64(b *testing.B) {
	prepareBenchTests_64(b)
	for n := 0; n < b.N; n++ {
		DumpFe_64[n%benchS].Neg(&bench_x_64[n%benchS])
	}
}

func BenchmarkNegEq_64(b *testing.B) {
	prepareBenchTests_64(b)
	for n := 0; n < b.N; n++ {
		bench_x_64[n%benchS].NegEq()
	}
}

func BenchmarkSign_64(b *testing.B) {
	prepareBenchTests_64(b)
	for n := 0; n < b.N; n++ {
		_ = bench_x_64[n%benchS].Sign()
	}
}

func BenchmarkJacobi_64(b *testing.B) {
	prepareBenchTests_64(b)
	for n := 0; n < b.N; n++ {
		_ = bench_x_64[n%benchS].Jacobi()
	}
}

func BenchmarkSquareRoot_64(b *testing.B) {
	prepareBenchTests_64(b)
	for i := 0; i < benchS; i++ {
		bench_x_64[i].SquareEq()
	}
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		DumpFe_64[n%benchS].SquareRoot(&bench_x_64[n%benchS])
	}
}
