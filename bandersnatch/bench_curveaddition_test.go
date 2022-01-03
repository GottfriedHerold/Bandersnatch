package bandersnatch

import "testing"

func BenchmarkCurveAddTyped(b *testing.B) {
	prepareBenchTest_Curve(b)
	b.Run("naive->t", func(b *testing.B) {
		for n := 0; n < b.N; n++ {
			DumpXTW[n%benchS].addNaive_ttt(&bench_xtw1[n%benchS], &bench_xtw2[n%benchS])
		}
	})

	b.Run("tt->t", func(b *testing.B) {
		for n := 0; n < b.N; n++ {
			DumpXTW[n%benchS].add_ttt(&bench_xtw1[n%benchS], &bench_xtw2[n%benchS])
		}
	})
	b.Run("ta->t", func(b *testing.B) {
		for n := 0; n < b.N; n++ {
			DumpXTW[n%benchS].add_tta(&bench_xtw1[n%benchS], &bench_axtw2[n%benchS])
		}
	})
	b.Run("aa->t", func(b *testing.B) {
		for n := 0; n < b.N; n++ {
			DumpXTW[n%benchS].add_taa(&bench_axtw1[n%benchS], &bench_axtw2[n%benchS])
		}
	})

	b.Run("tt->s", func(b *testing.B) {
		for n := 0; n < b.N; n++ {
			DumpEFGH[n%benchS].add_stt(&bench_xtw1[n%benchS], &bench_xtw2[n%benchS])
		}
	})

	b.Run("ta->s", func(b *testing.B) {
		for n := 0; n < b.N; n++ {
			DumpEFGH[n%benchS].add_sta(&bench_xtw1[n%benchS], &bench_axtw2[n%benchS])
		}
	})

	b.Run("aa->s", func(b *testing.B) {
		for n := 0; n < b.N; n++ {
			DumpEFGH[n%benchS].add_saa(&bench_axtw1[n%benchS], &bench_axtw2[n%benchS])
		}
	})
}

func BenchmarkCurveAddEqTyped(b *testing.B) {
	prepareBenchTest_Curve(b)
	b.Run("tt->t", func(b *testing.B) {
		for n := 0; n < b.N; n++ {
			bench_xtw1[n%benchS].add_ttt(&bench_xtw1[n%benchS], &bench_xtw2[n%benchS])
		}
	})
	b.Run("ta->t", func(b *testing.B) {
		for n := 0; n < b.N; n++ {
			bench_xtw2[n%benchS].add_tta(&bench_xtw2[n%benchS], &bench_axtw2[n%benchS])
		}
	})
}

func BenchmarkCurveSubTyped(b *testing.B) {
	prepareBenchTest_Curve(b)
	b.Run("tt->t", func(b *testing.B) {
		for n := 0; n < b.N; n++ {
			DumpXTW[n%benchS].sub_ttt(&bench_xtw1[n%benchS], &bench_xtw2[n%benchS])
		}
	})

	b.Run("ta->t", func(b *testing.B) {
		for n := 0; n < b.N; n++ {
			DumpXTW[n%benchS].sub_tta(&bench_xtw1[n%benchS], &bench_axtw2[n%benchS])
		}
	})

	b.Run("at->t", func(b *testing.B) {
		for n := 0; n < b.N; n++ {
			DumpXTW[n%benchS].sub_tat(&bench_axtw1[n%benchS], &bench_xtw2[n%benchS])
		}
	})

	b.Run("aa->t", func(b *testing.B) {
		for n := 0; n < b.N; n++ {
			DumpXTW[n%benchS].sub_taa(&bench_axtw1[n%benchS], &bench_axtw2[n%benchS])
		}
	})

	b.Run("tt->s", func(b *testing.B) {
		for n := 0; n < b.N; n++ {
			DumpEFGH[n%benchS].sub_stt(&bench_xtw1[n%benchS], &bench_xtw2[n%benchS])
		}
	})

	b.Run("ta->s", func(b *testing.B) {
		for n := 0; n < b.N; n++ {
			DumpEFGH[n%benchS].sub_sta(&bench_xtw1[n%benchS], &bench_axtw2[n%benchS])
		}
	})

	b.Run("at->s", func(b *testing.B) {
		for n := 0; n < b.N; n++ {
			DumpEFGH[n%benchS].sub_sat(&bench_axtw1[n%benchS], &bench_xtw2[n%benchS])
		}
	})

	b.Run("aa->s", func(b *testing.B) {
		for n := 0; n < b.N; n++ {
			DumpEFGH[n%benchS].sub_saa(&bench_axtw1[n%benchS], &bench_axtw2[n%benchS])
		}
	})
}

func BenchmarkCurveSubEqTyped(b *testing.B) {
	prepareBenchTest_Curve(b)
	b.Run("tt->t", func(b *testing.B) {
		for n := 0; n < b.N; n++ {
			bench_xtw1[n%benchS].sub_ttt(&bench_xtw1[n%benchS], &bench_xtw2[n%benchS])
		}
	})

	b.Run("ta->t", func(b *testing.B) {
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
