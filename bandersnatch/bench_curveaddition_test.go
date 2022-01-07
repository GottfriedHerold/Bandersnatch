package bandersnatch

import (
	"testing"
)

/*
	This file contains benchmarks for curve point additions and similar operations.

	The naming convention used here is that "Typed" benchmarks directly call the actual
	implementation of the specific operation for the given curve point type, whereas
	"Untyped" benchmarks the call to the interface (which just dispatches to the correct implementations)

	For Untyped, we have several helper functions that run iterate over curve point types and run sub-benchmarks on them.
	The naming convention for sub-benchmarks is arg1, arg2, ... -> receiver type

	To simplify benchmarking, we have some joint setup and "teardown" code.
	For joint setup, we need to call prepareBenchTest_Curve, which (re-)sets up global variables related
	to testing curve points:
	bench_xtw1, bench_xtw2, bench_axtw1, bench_axtw2, bench_efgh1, bench_efgh2 (for reading from)
	and
	DumpCPI_XTW, DumpCPI_AXTW, DumpCPI_EFGH (for writing to -- other variables named Dump<foo> that are not of interface type can be written to without setup)
	This setup adjusts global variables and can be called either outside subtests before b.Run(<subtest>) or immediately at the beginning inside a subtest.

	prepareBenchTest_Curve only has to be called once even for multiple sub-tests.
	(Of course, this assumes that "read-only" global variables are not written to -- but see general warning below)
	TODO: Change to always prepareBenchTest_Curve(b) / setupBenchmarkCurvePoints(b) inside subtest?

	For "teardown", postProcessBenchmarkCurvePoints(b) needs to be called

	To make sure that call counters work in builds that are enabled

	(the latter is not about releasing ressources, but about adding extra output to the benchmark and has to be called from inside the innermost)
*/

/*
	A general warning:
	Certain functions with arguments that should be read-only *DO* change the internal
	representation of their arguments to speed up further calls.
	This is most obvious for calls that require or convert to affine coordinates, where
	the read-only argument is actually changed 	and subsequent calls realize there is not much to do.
	This screws up benchmarking, of course and furthermore also potentially affects
	other (sub-)benchmarks if prepareBenchTest_Curve(b) is not called again.
	Less obvious is changing the internal representation of field elements. We largely ignore this issue.

*/

func BenchmarkCurveAddTyped(b *testing.B) {
	prepareBenchTest_Curve(b)
	b.Run("naive->t", func(b *testing.B) {
		setupBenchmarkCurvePoints(b)
		for n := 0; n < b.N; n++ {
			DumpXTW[n%benchS].addNaive_ttt(&bench_xtw1[n%benchS], &bench_xtw2[n%benchS])
		}
	})

	b.Run("tt->t", func(b *testing.B) {
		setupBenchmarkCurvePoints(b)
		for n := 0; n < b.N; n++ {
			DumpXTW[n%benchS].add_ttt(&bench_xtw1[n%benchS], &bench_xtw2[n%benchS])
		}
	})
	b.Run("ta->t", func(b *testing.B) {
		setupBenchmarkCurvePoints(b)
		for n := 0; n < b.N; n++ {
			DumpXTW[n%benchS].add_tta(&bench_xtw1[n%benchS], &bench_axtw2[n%benchS])
		}
	})
	b.Run("aa->t", func(b *testing.B) {
		setupBenchmarkCurvePoints(b)
		for n := 0; n < b.N; n++ {
			DumpXTW[n%benchS].add_taa(&bench_axtw1[n%benchS], &bench_axtw2[n%benchS])
		}
	})

	b.Run("tt->s", func(b *testing.B) {
		setupBenchmarkCurvePoints(b)
		for n := 0; n < b.N; n++ {
			DumpEFGH[n%benchS].add_stt(&bench_xtw1[n%benchS], &bench_xtw2[n%benchS])
		}
	})

	b.Run("ta->s", func(b *testing.B) {
		setupBenchmarkCurvePoints(b)
		for n := 0; n < b.N; n++ {
			DumpEFGH[n%benchS].add_sta(&bench_xtw1[n%benchS], &bench_axtw2[n%benchS])
		}
	})

	b.Run("aa->s", func(b *testing.B) {
		setupBenchmarkCurvePoints(b)
		for n := 0; n < b.N; n++ {
			DumpEFGH[n%benchS].add_saa(&bench_axtw1[n%benchS], &bench_axtw2[n%benchS])
		}
	})
}

func BenchmarkCurveAddEqTyped(b *testing.B) {
	prepareBenchTest_Curve(b)
	b.Run("tt->t", func(b *testing.B) {
		setupBenchmarkCurvePoints(b)
		for n := 0; n < b.N; n++ {
			bench_xtw1[n%benchS].add_ttt(&bench_xtw1[n%benchS], &bench_xtw2[n%benchS])
		}
	})
	b.Run("ta->t", func(b *testing.B) {
		setupBenchmarkCurvePoints(b)
		for n := 0; n < b.N; n++ {
			bench_xtw2[n%benchS].add_tta(&bench_xtw2[n%benchS], &bench_axtw2[n%benchS])
		}
	})
}

func BenchmarkCurveSubTyped(b *testing.B) {
	prepareBenchTest_Curve(b)
	b.Run("tt->t", func(b *testing.B) {
		setupBenchmarkCurvePoints(b)
		for n := 0; n < b.N; n++ {
			DumpXTW[n%benchS].sub_ttt(&bench_xtw1[n%benchS], &bench_xtw2[n%benchS])
		}
	})

	b.Run("ta->t", func(b *testing.B) {
		setupBenchmarkCurvePoints(b)
		for n := 0; n < b.N; n++ {
			DumpXTW[n%benchS].sub_tta(&bench_xtw1[n%benchS], &bench_axtw2[n%benchS])
		}
	})

	b.Run("at->t", func(b *testing.B) {
		setupBenchmarkCurvePoints(b)
		for n := 0; n < b.N; n++ {
			DumpXTW[n%benchS].sub_tat(&bench_axtw1[n%benchS], &bench_xtw2[n%benchS])
		}
	})

	b.Run("aa->t", func(b *testing.B) {
		setupBenchmarkCurvePoints(b)
		for n := 0; n < b.N; n++ {
			DumpXTW[n%benchS].sub_taa(&bench_axtw1[n%benchS], &bench_axtw2[n%benchS])
		}
	})

	b.Run("tt->s", func(b *testing.B) {
		setupBenchmarkCurvePoints(b)
		for n := 0; n < b.N; n++ {
			DumpEFGH[n%benchS].sub_stt(&bench_xtw1[n%benchS], &bench_xtw2[n%benchS])
		}
	})

	b.Run("ta->s", func(b *testing.B) {
		setupBenchmarkCurvePoints(b)
		for n := 0; n < b.N; n++ {
			DumpEFGH[n%benchS].sub_sta(&bench_xtw1[n%benchS], &bench_axtw2[n%benchS])
		}
	})

	b.Run("at->s", func(b *testing.B) {
		setupBenchmarkCurvePoints(b)
		for n := 0; n < b.N; n++ {
			DumpEFGH[n%benchS].sub_sat(&bench_axtw1[n%benchS], &bench_xtw2[n%benchS])
		}
	})

	b.Run("aa->s", func(b *testing.B) {
		setupBenchmarkCurvePoints(b)
		for n := 0; n < b.N; n++ {
			DumpEFGH[n%benchS].sub_saa(&bench_axtw1[n%benchS], &bench_axtw2[n%benchS])
		}
	})
}

func BenchmarkCurveSubEqTyped(b *testing.B) {
	prepareBenchTest_Curve(b)
	b.Run("tt->t", func(b *testing.B) {
		setupBenchmarkCurvePoints(b)
		for n := 0; n < b.N; n++ {
			bench_xtw1[n%benchS].sub_ttt(&bench_xtw1[n%benchS], &bench_xtw2[n%benchS])
		}
	})

	b.Run("ta->t", func(b *testing.B) {
		setupBenchmarkCurvePoints(b)
		for n := 0; n < b.N; n++ {
			bench_xtw2[n%benchS].sub_tta(&bench_xtw2[n%benchS], &bench_axtw2[n%benchS])
		}
	})
}

func BenchmarkCurveAddUntyped(b *testing.B) {
	benchmarkForAllPointTypesBinaryCommutative(b, allTestPointTypes, allTestPointTypes, func(b *testing.B) {
		for n := 0; n < b.N; n++ {
			DumpCPI[n%benchS].Add(bench_CPI1[n%benchS], bench_CPI2[n%benchS])
		}
	})
}

func BenchmarkCurveSubUntyped(b *testing.B) {
	benchmarkForAllPointTypesBinary(b, allTestPointTypes, allTestPointTypes, allTestPointTypes, func(b *testing.B) {
		for n := 0; n < b.N; n++ {
			DumpCPI[n%benchS].Sub(bench_CPI1[n%benchS], bench_CPI2[n%benchS])
		}
	})
}

func BenchmarkCurveDoubleUntyped(b *testing.B) {
	benchmarkForAllPointTypesUnary(b, allTestPointTypes, allTestPointTypes, func(b *testing.B) {
		for n := 0; n < b.N; n++ {
			DumpCPI[n%benchS].Double(bench_CPI1[n%benchS])
		}
	})
}
