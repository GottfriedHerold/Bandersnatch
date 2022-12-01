package fieldElements

import (
	"testing"
)

func BenchmarkSqrtHelperFunctions(b *testing.B) {
	b.Run("Dlog in small-order subgroup of roots of unity", benchmarkSqrt_NegDlogInSmallSubgroup)
	b.Run("special-purpose exponentiations", benchmarkSqrt_Exponentiations)
	b.Run("inverse square root of 2^32th roots of unity", benchmarkSqrt_InvSqrtEq)
}

func benchmarkSqrt_NegDlogInSmallSubgroup(b *testing.B) {
	var args []uint64 = CachedUint64.GetElements(1, benchS)
	var bench_fe [benchS]feType_SquareRoot
	for i := 0; i < len(bench_fe); i++ {
		var e Uint256
		e.SetUint64(args[i] % (1 << sqrtParam_BlockSize))
		bench_fe[i].Exp(&sqrtPrecomp_ReconstructionDyadicRoot, &e)
	}
	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		DumpUint64[n%benchS] = uint64(sqrtAlg_NegDlogInSmallDyadicSubgroup(&bench_fe[n%benchS]))
	}
}

func benchmarkSqrt_Exponentiations(b *testing.B) {
	var xs []feType_SquareRoot = GetPrecomputedFieldElements[feType_SquareRoot](10001, benchS)
	var D1, D2 [benchS]feType_SquareRoot
	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		xs[n%benchS].sqrtAlg_ComputeRelevantPowers(&D1[n%benchS], &D2[n%benchS])
	}
}

func benchmarkSqrt_InvSqrtEq(b *testing.B) {
	var xs []feType_SquareRoot = CachedRootsOfUnity.GetElements(10001, benchS)
	var Dump_FE [benchS]feType_SquareRoot
	// We only test the algorithm against numbers that are actually squares.
	// For non-squares, the algorithm is actually (a bit) faster, as this is detected early.
	for i := 0; i < len(xs); i++ {
		xs[i].SquareEq()
	}
	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		Dump_FE[n%benchS] = xs[n%benchS]
		_ = Dump_FE[n%benchS].invSqrtEqDyadic()
	}
}
