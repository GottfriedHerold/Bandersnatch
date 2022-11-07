package fieldElements

import "testing"

func Benchmark_uint256(b *testing.B) {
	b.Run("Neg", benchmarkNegEq)
	b.Run("Double", benchmarkDoubleEq)
	b.Run("Sub", benchmarkSubEq_ReduceWeak)
	b.Run("Add", benchmarkAddEq_ReduceWeak)

	b.Run("Square", benchmarkSquareEq)
	b.Run("Multiply", benchmarkMulEq)
	// b.Run("Invert", benchmarkInv)

}

func benchmarkAddEq_ReduceWeak(b *testing.B) {
	x := uint256{257, 479, 487, 491}
	y := uint256{997, 499, 503, 509}

	for i := 0; i < b.N; i += 2 {
		x.AddEqAndReduce_a(&y)
		y.AddEqAndReduce_a(&x)
	}

}

func benchmarkSubEq_ReduceWeak(b *testing.B) {
	x := uint256{257, 479, 487, 491}
	y := uint256{997, 499, 503, 509}

	for i := 0; i < b.N; i += 2 {
		x.SubEqAndReduce_a(&y)
		y.SubEqAndReduce_a(&x)
	}
}

func benchmarkInv(b *testing.B) {
	var a uint256
	count := 0
	//runs over the test values
OL:
	for {
		for _, _a := range testval {
			a = _a
			a.ModularInverse_a(&a)
			a.ModularInverse_a(&a)

			count += 2

			if count >= b.N {
				break OL
			}
		}
	}
}

func benchmarkMulEq(b *testing.B) {
	x := uint256{257, 479, 487, 491}
	y := uint256{997, 499, 503, 509}

	for i := 0; i < b.N; i += 2 {
		x.MulEqAndReduce_a(&y)
		y.MulEqAndReduce_a(&x)
	}
}

func benchmarkSquareEq(b *testing.B) {
	x := uint256{257, 479, 487, 491}
	y := uint256{997, 499, 503, 509}

	for i := 0; i < b.N; i += 2 {
		x.SquareEqAndReduce_a()
		y.SquareEqAndReduce_a()
	}
}

func benchmarkNegEq(b *testing.B) {
	x := uint256{257, 479, 487, 491}
	y := uint256{997, 499, 503, 509}

	for i := 0; i < b.N; i += 2 {
		x.ComputeModularNegative_Weak_f()
		y.ComputeModularNegative_Weak_f()
	}
}

func benchmarkDoubleEq(b *testing.B) {
	x := uint256{257, 479, 487, 491}
	y := uint256{997, 499, 503, 509}

	for i := 0; i < b.N; i += 2 {
		x.DoubleEqAndReduce_a()
		y.DoubleEqAndReduce_a()
	}
}
