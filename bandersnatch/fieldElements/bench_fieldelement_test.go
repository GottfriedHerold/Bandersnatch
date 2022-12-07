package fieldElements

import (
	"math/big"
	"testing"

	"github.com/GottfriedHerold/Bandersnatch/internal/testutils"
)

// This file is part of the fieldElements package. See the documentation of field_element.go for general remarks.

// This file contains a generic benchmark suite for the generic [FieldElementInterface], so we can use this
// to benchmark an arbitrary implementation of that interface.

// Note that there is an overhead that comes from the use of generics.
// We also have an (non-generic) old and less complete benchmark suite for bsFieldElement_MontgomeryNonUnique.
// The latter is kept as a comparison benchmark to measure the overhead from using generics.

// Run benchmark suite for all field element types that we defined
func BenchmarkAllFieldElementTypes(b *testing.B) {
	b.Log("NOTE: Benchmarking all field element implementations via generic benchmark. Being generic means some overhead. Take note if timings from non-generic benchmarks deviate.")
	b.Run("MontgomeryNonUnique", benchmarkFE_all[bsFieldElement_MontgomeryNonUnique])
	b.Run("big.Int Wrapper", benchmarkFE_all[bsFieldElement_BigInt])
}

// benchmarkFE_all is a generic benchmark function that runs all benchmark functions defined below for the given type parameter.
func benchmarkFE_all[FE any, FEPtr interface {
	*FE
	FieldElementInterface[FEPtr]
}](b *testing.B) {
	b.Run("Add", benchmarkFE_Add[FE, FEPtr])
	b.Run("AddEq", benchmarkFE_AddEq[FE, FEPtr])
	b.Run("Double", benchmarkFE_Double[FE, FEPtr])
	b.Run("DoubleEq", benchmarkFE_DoubleEq[FE, FEPtr])
	b.Run("Sub", benchmarkFE_Sub[FE, FEPtr])
	b.Run("SubEq", benchmarkFE_SubEq[FE, FEPtr])
	b.Run("Neg", benchmarkFE_Neg[FE, FEPtr])
	b.Run("NegEq", benchmarkFE_NegEq[FE, FEPtr])

	b.Run("Mul", benchmarkFE_Mul[FE, FEPtr])
	b.Run("MulEq", benchmarkFE_MulEq[FE, FEPtr])
	b.Run("Square", benchmarkFE_Square[FE, FEPtr])
	b.Run("SquareEq", benchmarkFE_SquareEq[FE, FEPtr])
	b.Run("MulFive", benchmarkFE_MulFive[FE, FEPtr])
	b.Run("MulFiveEq", benchmarkFE_MulFiveEq[FE, FEPtr])

	b.Run("Inv", benchmarkFE_Inv[FE, FEPtr])
	b.Run("InvEq", benchmarkFE_InvEq[FE, FEPtr])
	b.Run("Divide", benchmarkFE_Divide[FE, FEPtr])
	b.Run("DivideEq", benchmarkFE_DivideEq[FE, FEPtr])

	b.Run("IsEqual", benchmarkFE_IsEqual[FE, FEPtr])
	b.Run("CmpAbs", benchmarkFE_CmpAbs[FE, FEPtr])
	b.Run("Sign", benchmarkFE_Sign[FE, FEPtr])

	b.Run("Jacobi", benchmarkFE_Jacobi[FE, FEPtr])
	// We have two separate benchmarks for the SquareRoot method, called only on squares or only on non-squares.
	b.Run("SquareRoot (Squares)", benchmarkFE_SquareRootOnSquares[FE, FEPtr])
	b.Run("SquareRoot (NonSquares)", benchmarkFE_SquareRootOnNonSquares[FE, FEPtr])

	b.Run("SetUint256", benchmarkFE_SetUint256[FE, FEPtr])
	b.Run("ToUint256", benchmarkFE_ToUint256[FE, FEPtr])
	b.Run("SetBigInt", benchmarkFE_SetBigInt[FE, FEPtr])
	b.Run("ToBigInt", benchmarkFE_ToBigInt[FE, FEPtr])

	b.Run("AddInt64", benchmarkFE_AddInt64[FE, FEPtr])
	b.Run("AddUint64", benchmarkFE_AddUint64[FE, FEPtr])
	b.Run("SubInt64", benchmarkFE_SubInt64[FE, FEPtr])
	b.Run("SubUint64", benchmarkFE_SubUint64[FE, FEPtr])
	b.Run("MulInt64", benchmarkFE_MulInt64[FE, FEPtr])
	b.Run("MulUint64", benchmarkFE_MulUint64[FE, FEPtr])
	b.Run("DivideInt64", benchmarkFE_DivideInt64[FE, FEPtr])
	b.Run("DivideUint64", benchmarkFE_DivideUint64[FE, FEPtr])
}

// actual benchmark suite. We benchmark (almost) all methods of the interface
// function names as benchmarkFE_FOO for the FOO method

func benchmarkFE_Add[FE any, FEPtr interface {
	*FE
	FieldElementInterface[FEPtr]
}](b *testing.B) {
	var bench_x []FE = GetPrecomputedFieldElements[FE, FEPtr](10001, benchS)
	var bench_y []FE = GetPrecomputedFieldElements[FE, FEPtr](10002, benchS)
	var res [benchS]FE
	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		FEPtr(&res[n%benchS]).Add(&bench_x[n%benchS], &bench_y[n%benchS])
	}
}

func benchmarkFE_AddEq[FE any, FEPtr interface {
	*FE
	FieldElementInterface[FEPtr]
}](b *testing.B) {
	var bench_x []FE = GetPrecomputedFieldElements[FE, FEPtr](10001, benchS)
	var res [benchS]FE
	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		FEPtr(&res[n%benchS]).AddEq(&bench_x[n%benchS])
	}
}

func benchmarkFE_Double[FE any, FEPtr interface {
	*FE
	FieldElementInterface[FEPtr]
}](b *testing.B) {
	var bench_x []FE = GetPrecomputedFieldElements[FE, FEPtr](10001, benchS)
	var res [benchS]FE
	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		FEPtr(&res[n%benchS]).Double(&bench_x[n%benchS])
	}
}

func benchmarkFE_DoubleEq[FE any, FEPtr interface {
	*FE
	FieldElementInterface[FEPtr]
}](b *testing.B) {
	var bench_x []FE = GetPrecomputedFieldElements[FE, FEPtr](10001, benchS)
	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		FEPtr(&bench_x[n%benchS]).DoubleEq()
	}
}

func benchmarkFE_Sub[FE any, FEPtr interface {
	*FE
	FieldElementInterface[FEPtr]
}](b *testing.B) {
	var bench_x []FE = GetPrecomputedFieldElements[FE, FEPtr](10001, benchS)
	var bench_y []FE = GetPrecomputedFieldElements[FE, FEPtr](10002, benchS)
	var res [benchS]FE
	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		FEPtr(&res[n%benchS]).Sub(&bench_x[n%benchS], &bench_y[n%benchS])
	}
}

func benchmarkFE_SubEq[FE any, FEPtr interface {
	*FE
	FieldElementInterface[FEPtr]
}](b *testing.B) {
	var bench_x []FE = GetPrecomputedFieldElements[FE, FEPtr](10001, benchS)
	var res [benchS]FE
	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		FEPtr(&res[n%benchS]).SubEq(&bench_x[n%benchS])
	}
}

func benchmarkFE_Neg[FE any, FEPtr interface {
	*FE
	FieldElementInterface[FEPtr]
}](b *testing.B) {
	var bench_x []FE = GetPrecomputedFieldElements[FE, FEPtr](10001, benchS)
	var res [benchS]FE
	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		FEPtr(&res[n%benchS]).Neg(&bench_x[n%benchS])
	}
}

func benchmarkFE_NegEq[FE any, FEPtr interface {
	*FE
	FieldElementInterface[FEPtr]
}](b *testing.B) {
	var bench_x []FE = GetPrecomputedFieldElements[FE, FEPtr](10001, benchS)
	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		FEPtr(&bench_x[n%benchS]).NegEq()
	}
}

func benchmarkFE_Mul[FE any, FEPtr interface {
	*FE
	FieldElementInterface[FEPtr]
}](b *testing.B) {
	var bench_x []FE = GetPrecomputedFieldElements[FE, FEPtr](10001, benchS)
	var bench_y []FE = GetPrecomputedFieldElements[FE, FEPtr](10002, benchS)
	var res [benchS]FE
	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		FEPtr(&res[n%benchS]).Mul(&bench_x[n%benchS], &bench_y[n%benchS])
	}
}

func benchmarkFE_MulEq[FE any, FEPtr interface {
	*FE
	FieldElementInterface[FEPtr]
}](b *testing.B) {
	var bench_x []FE = GetPrecomputedFieldElements[FE, FEPtr](10001, benchS)
	var res [benchS]FE
	for i := 0; i < benchS; i++ {
		FEPtr(&res[i]).SetOne()
	}
	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		FEPtr(&res[n%benchS]).MulEq(&bench_x[n%benchS])
	}
}

func benchmarkFE_Square[FE any, FEPtr interface {
	*FE
	FieldElementInterface[FEPtr]
}](b *testing.B) {
	var bench_x []FE = GetPrecomputedFieldElements[FE, FEPtr](10001, benchS)
	var res [benchS]FE
	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		FEPtr(&res[n%benchS]).Square(&bench_x[n%benchS])
	}
}

func benchmarkFE_SquareEq[FE any, FEPtr interface {
	*FE
	FieldElementInterface[FEPtr]
}](b *testing.B) {
	var bench_x []FE = GetPrecomputedFieldElements[FE, FEPtr](10001, benchS)
	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		FEPtr(&bench_x[n%benchS]).SquareEq()
	}
}

func benchmarkFE_MulFive[FE any, FEPtr interface {
	*FE
	FieldElementInterface[FEPtr]
}](b *testing.B) {
	var bench_x []FE = GetPrecomputedFieldElements[FE, FEPtr](10001, benchS)
	var res [benchS]FE
	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		FEPtr(&res[n%benchS]).MulFive(&bench_x[n%benchS])
	}
}

func benchmarkFE_MulFiveEq[FE any, FEPtr interface {
	*FE
	FieldElementInterface[FEPtr]
}](b *testing.B) {
	var bench_x []FE = GetPrecomputedFieldElements[FE, FEPtr](10001, benchS)
	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		FEPtr(&bench_x[n%benchS]).MulEqFive()
	}
}

func benchmarkFE_Inv[FE any, FEPtr interface {
	*FE
	FieldElementInterface[FEPtr]
}](b *testing.B) {
	var bench_x []FE = GetPrecomputedFieldElementsNonZero[FE, FEPtr](10001, benchS)
	var res [benchS]FE
	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		FEPtr(&res[n%benchS]).Inv(&bench_x[n%benchS])
	}
}

func benchmarkFE_InvEq[FE any, FEPtr interface {
	*FE
	FieldElementInterface[FEPtr]
}](b *testing.B) {
	var bench_x []FE = GetPrecomputedFieldElementsNonZero[FE, FEPtr](10001, benchS)
	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		FEPtr(&bench_x[n%benchS]).InvEq()
	}
}

func benchmarkFE_Divide[FE any, FEPtr interface {
	*FE
	FieldElementInterface[FEPtr]
}](b *testing.B) {
	var bench_x []FE = GetPrecomputedFieldElements[FE, FEPtr](10001, benchS)
	var bench_y []FE = GetPrecomputedFieldElementsNonZero[FE, FEPtr](10002, benchS)
	var res [benchS]FE
	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		FEPtr(&res[n%benchS]).Divide(&bench_x[n%benchS], &bench_y[n%benchS])
	}
}

func benchmarkFE_DivideEq[FE any, FEPtr interface {
	*FE
	FieldElementInterface[FEPtr]
}](b *testing.B) {
	var bench_x []FE = GetPrecomputedFieldElements[FE, FEPtr](10001, benchS)
	var bench_y []FE = GetPrecomputedFieldElementsNonZero[FE, FEPtr](10002, benchS)
	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		FEPtr(&bench_x[n%benchS]).DivideEq(&bench_y[n%benchS])
	}
}

func benchmarkFE_IsEqual[FE any, FEPtr interface {
	*FE
	FieldElementInterface[FEPtr]
}](b *testing.B) {
	var bench_x []FE = GetPrecomputedFieldElements[FE, FEPtr](10001, benchS)
	var bench_y []FE = GetPrecomputedFieldElements[FE, FEPtr](10001, benchS) // same randomness
	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		DumpBools_fe[n%benchS] = FEPtr(&bench_x[n%benchS]).IsEqual(&bench_y[n%benchS])
	}
}

func benchmarkFE_CmpAbs[FE any, FEPtr interface {
	*FE
	FieldElementInterface[FEPtr]
}](b *testing.B) {
	var bench_x []FE = GetPrecomputedFieldElements[FE, FEPtr](10001, benchS)
	var bench_y []FE = GetPrecomputedFieldElements[FE, FEPtr](10001, benchS) // same randomness
	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		DumpBools_fe[n%benchS], DumpBools_fe[(n+1)%benchS] = FEPtr(&bench_x[n%benchS]).CmpAbs(&bench_y[n%benchS])
	}
}

func benchmarkFE_Sign[FE any, FEPtr interface {
	*FE
	FieldElementInterface[FEPtr]
}](b *testing.B) {
	var bench_x []FE = GetPrecomputedFieldElements[FE, FEPtr](10001, benchS)
	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		DumpInt[n%benchS] = FEPtr(&bench_x[n%benchS]).Sign()
	}
}

func benchmarkFE_Jacobi[FE any, FEPtr interface {
	*FE
	FieldElementInterface[FEPtr]
}](b *testing.B) {
	var bench_x []FE = GetPrecomputedFieldElements[FE, FEPtr](10001, benchS)
	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		DumpInt[n%benchS] = FEPtr(&bench_x[n%benchS]).Jacobi()
	}
}

func benchmarkFE_SquareRootOnSquares[FE any, FEPtr interface {
	*FE
	FieldElementInterface[FEPtr]
}](b *testing.B) {
	var bench_x []FE = GetPrecomputedFieldElements[FE, FEPtr](10001, benchS)
	var res [benchS]FE
	for i := 0; i < benchS; i++ {
		FEPtr(&bench_x[i]).SquareEq()
	}
	// testutils.MakeVariableEscape(b, &res)
	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		DumpBools_fe[n%benchS] = FEPtr(&res[n%benchS]).SquareRoot(&bench_x[n%benchS])
	}
}

func benchmarkFE_SquareRootOnNonSquares[FE any, FEPtr interface {
	*FE
	FieldElementInterface[FEPtr]
}](b *testing.B) {
	var bench_x []FE = GetPrecomputedFieldElements[FE, FEPtr](10001, benchS)
	var res [benchS]FE
	for i := 0; i < benchS; i++ {
		if FEPtr(&bench_x[i]).IsZero() {
			continue
		}
		FEPtr(&bench_x[i]).SquareEq()
		FEPtr(&bench_x[i]).MulEqFive() // 5 being a nonsquare
		testutils.Assert(FEPtr(&bench_x[i]).Jacobi() == -1)
	}
	// testutils.MakeVariableEscape(b, &res)
	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		DumpBools_fe[n%benchS] = FEPtr(&res[n%benchS]).SquareRoot(&bench_x[n%benchS])
	}
}

func benchmarkFE_ToUint256[FE any, FEPtr interface {
	*FE
	FieldElementInterface[FEPtr]
}](b *testing.B) {
	var bench_x []FE = GetPrecomputedFieldElements[FE, FEPtr](10001, benchS)
	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		FEPtr(&bench_x[n%benchS]).ToUint256(&DumpUint256[n%benchS])
	}
}

func benchmarkFE_SetUint256[FE any, FEPtr interface {
	*FE
	FieldElementInterface[FEPtr]
}](b *testing.B) {
	var res [benchS]FE
	var xVals []Uint256 = CachedUint256.GetElements(SeedAndRange{seed: 10001, allowedRange: twoTo256_Int}, benchS)
	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		FEPtr(&res[n%benchS]).SetUint256(&xVals[n%benchS])
	}
}

func benchmarkFE_ToBigInt[FE any, FEPtr interface {
	*FE
	FieldElementInterface[FEPtr]
}](b *testing.B) {
	var bench_x []FE = GetPrecomputedFieldElements[FE, FEPtr](10001, benchS)
	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		DumpBigInt[n%benchS] = FEPtr(&bench_x[n%benchS]).ToBigInt()
	}
}

func benchmarkFE_SetBigInt[FE any, FEPtr interface {
	*FE
	FieldElementInterface[FEPtr]
}](b *testing.B) {
	var res [benchS]FE
	var xVals []*big.Int = CachedBigInt.GetElements(SeedAndRange{seed: 10001, allowedRange: twoTo256_Int}, benchS)
	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		FEPtr(&res[n%benchS]).SetBigInt(xVals[n%benchS])
	}
}

func benchmarkFE_AddInt64[FE any, FEPtr interface {
	*FE
	FieldElementInterface[FEPtr]
}](b *testing.B) {
	var bench_x []FE = GetPrecomputedFieldElements[FE, FEPtr](10001, benchS)
	var bench_y []int64 = CachedInt64.GetElements(10002, benchS)
	var res [benchS]FE
	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		FEPtr(&res[n%benchS]).AddInt64(&bench_x[n%benchS], bench_y[n%benchS])
	}
}

func benchmarkFE_AddUint64[FE any, FEPtr interface {
	*FE
	FieldElementInterface[FEPtr]
}](b *testing.B) {
	var bench_x []FE = GetPrecomputedFieldElements[FE, FEPtr](10001, benchS)
	var bench_y []uint64 = CachedUint64.GetElements(10002, benchS)
	var res [benchS]FE
	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		FEPtr(&res[n%benchS]).AddUint64(&bench_x[n%benchS], bench_y[n%benchS])
	}
}

func benchmarkFE_SubInt64[FE any, FEPtr interface {
	*FE
	FieldElementInterface[FEPtr]
}](b *testing.B) {
	var bench_x []FE = GetPrecomputedFieldElements[FE, FEPtr](10001, benchS)
	var bench_y []int64 = CachedInt64.GetElements(10002, benchS)
	var res [benchS]FE
	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		FEPtr(&res[n%benchS]).SubInt64(&bench_x[n%benchS], bench_y[n%benchS])
	}
}

func benchmarkFE_SubUint64[FE any, FEPtr interface {
	*FE
	FieldElementInterface[FEPtr]
}](b *testing.B) {
	var bench_x []FE = GetPrecomputedFieldElements[FE, FEPtr](10001, benchS)
	var bench_y []uint64 = CachedUint64.GetElements(10002, benchS)
	var res [benchS]FE
	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		FEPtr(&res[n%benchS]).SubUint64(&bench_x[n%benchS], bench_y[n%benchS])
	}
}

func benchmarkFE_MulInt64[FE any, FEPtr interface {
	*FE
	FieldElementInterface[FEPtr]
}](b *testing.B) {
	var bench_x []FE = GetPrecomputedFieldElements[FE, FEPtr](10001, benchS)
	var bench_y []int64 = CachedInt64.GetElements(10002, benchS)
	var res [benchS]FE
	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		FEPtr(&res[n%benchS]).MulInt64(&bench_x[n%benchS], bench_y[n%benchS])
	}
}

func benchmarkFE_MulUint64[FE any, FEPtr interface {
	*FE
	FieldElementInterface[FEPtr]
}](b *testing.B) {
	var bench_x []FE = GetPrecomputedFieldElements[FE, FEPtr](10001, benchS)
	var bench_y []uint64 = CachedUint64.GetElements(10002, benchS)
	var res [benchS]FE
	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		FEPtr(&res[n%benchS]).MulUint64(&bench_x[n%benchS], bench_y[n%benchS])
	}
}

func benchmarkFE_DivideInt64[FE any, FEPtr interface {
	*FE
	FieldElementInterface[FEPtr]
}](b *testing.B) {
	var bench_x []FE = GetPrecomputedFieldElements[FE, FEPtr](10001, benchS)
	var bench_y []int64 = CachedInt64.GetElements(10002, benchS)
	for i := 0; i < benchS; i++ {
		if bench_y[i] == 0 {
			bench_y[i] = 1
		}
	}
	var res [benchS]FE
	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		FEPtr(&res[n%benchS]).DivideInt64(&bench_x[n%benchS], bench_y[n%benchS])
	}
}

func benchmarkFE_DivideUint64[FE any, FEPtr interface {
	*FE
	FieldElementInterface[FEPtr]
}](b *testing.B) {
	var bench_x []FE = GetPrecomputedFieldElements[FE, FEPtr](10001, benchS)
	var bench_y []uint64 = CachedUint64.GetElements(10002, benchS)
	for i := 0; i < benchS; i++ {
		if bench_y[i] == 0 {
			bench_y[i] = 1
		}
	}

	var res [benchS]FE
	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		FEPtr(&res[n%benchS]).DivideUint64(&bench_x[n%benchS], bench_y[n%benchS])
	}
}

// For copy&pasting

/*
func benchmarkFE_[FE any, FEPtr interface {
	*FE
	FieldElementInterface[FEPtr]
}](b *testing.B) {
	var bench_x []FE = GetPrecomputedFieldElements[FE, FEPtr](10001, benchS)
	var bench_y []FE = GetPrecomputedFieldElements[FE, FEPtr](10002, benchS)
	var res [benchS]FE
	prepareBenchmarkFieldElements(b)
	for n:=0; n < b.N; n++{

	}
}
*/
