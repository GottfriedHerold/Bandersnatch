package fieldElements

import (
	"math/big"
	"testing"

	"github.com/GottfriedHerold/Bandersnatch/bandersnatch/common"
	"github.com/GottfriedHerold/Bandersnatch/internal/testutils"
)

// This file contains tests for uint256_modular.go
//
// We test that
//  - the function defined there preserve the reducedness as specified
//  - the function matches the behaviour of big.Int, which we assume to be correct.

func TestUint256AddAndReduce_b(t *testing.T) {
	prepareTestFieldElements(t)

	const num = 256

	xs := CachedUint256.GetElements(pc_uint256_b, num)
	ys := CachedUint256.GetElements(pc_uint256_b, num)
	var z Uint256
	var zInt *big.Int = new(big.Int)
	var resInt *big.Int = new(big.Int)
	var xInt *big.Int = new(big.Int)
	var yInt *big.Int = new(big.Int)
	for _, x := range xs {
		for _, y := range ys {
			z.addAndReduce_b_c(&x, &y)
			testutils.FatalUnless(t, z.IsReduced_b(), "AddAndReduce_b_c does not reduce for b")

			xInt = x.ToBigInt()
			yInt = y.ToBigInt()
			zInt.Add(xInt, yInt)
			zInt.Mod(zInt, baseFieldSize_Int)

			resInt = z.ToBigInt()
			resInt.Mod(resInt, baseFieldSize_Int)

			testutils.FatalUnless(t, resInt.Cmp(zInt) == 0, "AddAndReduce does not macht big.Int's Add")
		}
	}
}

func TestUint256AddAndReduce_c(t *testing.T) {
	prepareTestFieldElements(t)

	const num = 256

	xs := CachedUint256.GetElements(pc_uint256_c, num)
	ys := CachedUint256.GetElements(pc_uint256_c, num)
	var z Uint256
	var zInt *big.Int = new(big.Int)
	var resInt *big.Int = new(big.Int)
	var xInt *big.Int = new(big.Int)
	var yInt *big.Int = new(big.Int)
	for _, x := range xs {
		for _, y := range ys {
			z.addAndReduce_b_c(&x, &y)
			testutils.FatalUnless(t, z.IsReduced_c(), "AddAndReduce_b_c does not reduce for c")

			xInt = x.ToBigInt()
			yInt = y.ToBigInt()
			zInt.Add(xInt, yInt)
			zInt.Mod(zInt, baseFieldSize_Int)

			resInt = z.ToBigInt()
			resInt.Mod(resInt, baseFieldSize_Int)

			testutils.FatalUnless(t, resInt.Cmp(zInt) == 0, "AddAndReduce does not macht big.Int's Add")
		}
	}
}

func TestUint256AddEqAndReduce_a(t *testing.T) {
	prepareTestFieldElements(t)

	const num = 256

	xs := CachedUint256.GetElements(pc_uint256_a, num)
	ys := CachedUint256.GetElements(pc_uint256_a, num)
	var z Uint256
	var zInt *big.Int = new(big.Int)
	var resInt *big.Int = new(big.Int)
	var xInt *big.Int = new(big.Int)
	var yInt *big.Int = new(big.Int)
	for _, x := range xs {
		for _, y := range ys {
			z = x
			z.AddEqAndReduce_a(&y)
			testutils.FatalUnless(t, z.IsReduced_a(), "AddEqAndReduce_a does not reduce properly")

			xInt = x.ToBigInt()
			yInt = y.ToBigInt()
			zInt.Add(xInt, yInt)
			zInt.Mod(zInt, baseFieldSize_Int)

			resInt = z.ToBigInt()
			resInt.Mod(resInt, baseFieldSize_Int)

			testutils.FatalUnless(t, resInt.Cmp(zInt) == 0, "AddEqAndReduce_a does not macht big.Int's Add")
		}
	}
}

func TestUint256SubAndReduce_c(t *testing.T) {
	prepareTestFieldElements(t)

	const num = 256

	xs := CachedUint256.GetElements(pc_uint256_c, num)
	ys := CachedUint256.GetElements(pc_uint256_c, num)
	var z Uint256
	var zInt *big.Int = new(big.Int)
	var resInt *big.Int = new(big.Int)
	var xInt *big.Int = new(big.Int)
	var yInt *big.Int = new(big.Int)
	for _, x := range xs {
		for _, y := range ys {
			z.SubAndReduce_c(&x, &y)
			testutils.FatalUnless(t, z.IsReduced_c(), "SubAndReduce_c does not properly reduce")

			xInt = x.ToBigInt()
			yInt = y.ToBigInt()
			zInt.Sub(xInt, yInt)
			zInt.Mod(zInt, baseFieldSize_Int)

			resInt = z.ToBigInt()
			resInt.Mod(resInt, baseFieldSize_Int)

			testutils.FatalUnless(t, resInt.Cmp(zInt) == 0, "SubAndReduce_c does not macht big.Int's Sub")
		}
	}
}

func TestUint256SubAndReduce_b(t *testing.T) {
	prepareTestFieldElements(t)

	const num = 256

	xs := CachedUint256.GetElements(pc_uint256_b, num)
	ys := CachedUint256.GetElements(pc_uint256_b, num)
	var z Uint256
	var zInt *big.Int = new(big.Int)
	var resInt *big.Int = new(big.Int)
	var xInt *big.Int = new(big.Int)
	var yInt *big.Int = new(big.Int)
	for _, x := range xs {
		for _, y := range ys {
			z.SubAndReduce_b(&x, &y)
			testutils.FatalUnless(t, z.IsReduced_b(), "SubAndReduce_b does not properly reduce")

			xInt = x.ToBigInt()
			yInt = y.ToBigInt()
			zInt.Sub(xInt, yInt)
			zInt.Mod(zInt, baseFieldSize_Int)

			resInt = z.ToBigInt()
			resInt.Mod(resInt, baseFieldSize_Int)

			testutils.FatalUnless(t, resInt.Cmp(zInt) == 0, "SubAndReduce_b does not macht big.Int's Sub")
		}
	}
}

func TestUint256SubEqAndReduce_a(t *testing.T) {
	prepareTestFieldElements(t)

	const num = 256

	xs := CachedUint256.GetElements(pc_uint256_a, num)
	ys := CachedUint256.GetElements(pc_uint256_a, num)
	var z Uint256
	var zInt *big.Int = new(big.Int)
	var resInt *big.Int = new(big.Int)
	var xInt *big.Int = new(big.Int)
	var yInt *big.Int = new(big.Int)
	for _, x := range xs {
		for _, y := range ys {
			z = x
			z.SubEqAndReduce_a(&y)
			testutils.FatalUnless(t, z.IsReduced_a(), "SubEqAndReduce_a does not reduce properly")

			xInt = x.ToBigInt()
			yInt = y.ToBigInt()
			zInt.Sub(xInt, yInt)
			zInt.Mod(zInt, baseFieldSize_Int)

			resInt = z.ToBigInt()
			resInt.Mod(resInt, baseFieldSize_Int)

			testutils.FatalUnless(t, resInt.Cmp(zInt) == 0, "SubEqAndReduce_a does not macht big.Int's Sub")
		}
	}
}

func TestUint256_ModularInverse_a_NAIVEHAC(t *testing.T) {
	prepareTestFieldElements(t)

	const num = 1000

	xs := CachedUint256.GetElements(pc_uint256_a, num)

	var z Uint256
	var zInt *big.Int = new(big.Int)
	var resInt *big.Int = new(big.Int)
	var xInt *big.Int = new(big.Int)

	for i, x := range xs {
		if i == 0 {
			x = Uint256{}
		}
		if i == 1 {
			x = baseFieldSize_uint256
		}
		if i == 2 {
			x = twiceBaseFieldSize_uint256
		}
		xCopy := x
		xCopy.Reduce()
		if xCopy.IsZero() {
			xInt = x.ToBigInt()
			testutils.FatalUnless(t, zInt.ModInverse(xInt, baseFieldSize_Int) == nil, "Cannot happen")
			z = Uint256{10, 20, 30, 50}
			ok := z.ModularInverse_a_NAIVEHAC(&x)
			testutils.FatalUnless(t, !ok, "ModularInverse_a_NAIVEHAC did not recognize non-invertible elements")
			testutils.FatalUnless(t, z == Uint256{10, 20, 30, 50}, "ModularInverse_a_NAIVEHAC changed receiver upon getting zero")

		} else {

			ok := z.ModularInverse_a_NAIVEHAC(&x)
			testutils.FatalUnless(t, z.IsReduced_a(), "ModularInverse_a does not reduce properly") // no-op, really
			testutils.FatalUnless(t, ok, "ModularInverse_a_NAIVEHAC did not recognize invertible element")

			xInt = x.ToBigInt()
			okBigInt := zInt.ModInverse(xInt, baseFieldSize_Int)
			testutils.FatalUnless(t, okBigInt != nil, "Cannot happen")

			resInt = z.ToBigInt()
			resInt.Mod(resInt, baseFieldSize_Int)

			testutils.FatalUnless(t, resInt.Cmp(zInt) == 0, "ModuceInverse_a_NAIVEHAC does not macht big.Int's ModInverse")
		}

	}
}

func testReductionFunction(t *testing.T, reductionFunction func(*Uint256), reducedInputSeed SeedAndRange, outputReducednessCheck func(*Uint256) bool, funName string) {
	prepareTestFieldElements(t)

	const num = 1000

	xs := CachedUint256.GetElements(reducedInputSeed, num)

	for _, x := range xs {
		xInt := x.ToBigInt()
		testutils.FatalUnless(t, xInt.Cmp(reducedInputSeed.allowedRange) < 0, "Generation of input samples was wrong")
	}

	xs = append(xs, zero_uint256)
	xs = append(xs, baseFieldSize_uint256)
	xs = append(xs, twiceBaseFieldSize_uint256)
	xs = append(xs, twoTo256ModBaseField_uint256)

	for i := 0; i < 4; i++ {
		for _, y := range []Uint256{baseFieldSize_uint256, twiceBaseFieldSize_uint256, zero_uint256, twoTo256ModBaseField_uint256} {
			temp := y
			temp[i] += 1
			xs = append(xs, temp)
			temp = y
			temp[i] -= 1
			xs = append(xs, temp)
		}
	}

	xs = append(xs, uint256Max_uint256)

	var z Uint256
	var zInt *big.Int = new(big.Int)
	var xInt *big.Int = new(big.Int)

	for _, x := range xs {
		z = x

		xInt = x.ToBigInt()
		if xInt.Cmp(reducedInputSeed.allowedRange) >= 0 {
			continue
		}

		reductionFunction(&z)

		testutils.FatalUnless(t, outputReducednessCheck(&z), funName+" does not properly reduce")

		xInt.Mod(xInt, baseFieldSize_Int)

		zInt = z.ToBigInt()
		zInt.Mod(zInt, baseFieldSize_Int)

		testutils.FatalUnless(t, xInt.Cmp(zInt) == 0, funName+" does not preserve element modulo BaseFieldSize")
	}

}

func TestUint256_Reduce(t *testing.T) {
	testReductionFunction(t, (*Uint256).Reduce_ca, pc_uint256_a, (*Uint256).IsReduced_c, "reduce_ca")
	testReductionFunction(t, (*Uint256).Reduce_fb, pc_uint256_b, (*Uint256).IsReduced_f, "reduce_fb")
	testReductionFunction(t, (*Uint256).Reduce, pc_uint256_a, (*Uint256).IsReduced_f, "Reduce")
	testReductionFunction(t, (*Uint256).Reduce_fa, pc_uint256_a, (*Uint256).IsReduced_f, "Reduce_fa")

	testReductionFunction(t, (*Uint256).reduce_fa_barret, pc_uint256_a, (*Uint256).IsReduced_f, "reduce_fa_barret")
	testReductionFunction(t, (*Uint256).reduce_fa_optimistic, pc_uint256_a, (*Uint256).IsReduced_f, "reduce_fa_optimistic")
	testReductionFunction(t, (*Uint256).reduce_fa_loop, pc_uint256_a, (*Uint256).IsReduced_f, "reduce_fa_loop")
	testReductionFunction(t, (*Uint256).reduce_fb_exact, pc_uint256_b, (*Uint256).IsReduced_f, "reduce_fb_exact")
	testReductionFunction(t, (*Uint256).reduce_fb_optimistic, pc_uint256_b, (*Uint256).IsReduced_f, "reduce_fb_optimistic")
}

func TestUint256_IsFullyReduced(t *testing.T) {
	prepareTestFieldElements(t)

	const num = 1000

	xs := CachedUint256.GetElements(pc_uint256_a, num) // TODO: Use a different seed here with special prepopulated values
	for i, x := range xs {
		if i == 0 {
			x = Uint256{}
		}
		if i == 1 {
			x = one_uint256
		}
		if i == 2 {
			x = baseFieldSize_uint256
		}
		IsFullyReduced := x.is_fully_reduced()
		isFullyReduced2 := x.IsReduced_f()
		xInt := x.ToBigInt()
		IsFullyReduced_Int := xInt.Cmp(baseFieldSize_Int) < 0
		testutils.FatalUnless(t, IsFullyReduced == IsFullyReduced_Int, "is_fully_reduced does not match big.Int")
		testutils.FatalUnless(t, IsFullyReduced == isFullyReduced2, "is_fully_reduced does not IsReduced_f")
	}
}

func TestUint256_ReduceUint512ToUint256(t *testing.T) {
	prepareTestFieldElements(t)

	const num = 1000

	xs := CachedUint512.GetElements(1, num) // TODO: Use a different seed here with special prepopulated values
	var z Uint256
	for _, x := range xs {
		xInt := x.ToBigInt()
		xInt.Mod(xInt, baseFieldSize_Int)

		z.ReduceUint512ToUint256_a(x) // NOTE: Non-Pointer argument
		testutils.FatalUnless(t, z.IsReduced_a(), "cannot happen")

		zInt := z.ToBigInt()
		zInt.Mod(zInt, baseFieldSize_Int)

		testutils.FatalUnless(t, zInt.Cmp(xInt) == 0, "ReduceUint512ToUint256 does not preverse x mod BaseFieldSize")
	}
}

func TestUint256_DoubleEqAndReduce_a(t *testing.T) {
	prepareTestFieldElements(t)
	const num = 1000

	xs := CachedUint256.GetElements(pc_uint256_a, num)
	var z Uint256
	for _, x := range xs {

		z = x
		z.DoubleEqAndReduce_a()
		testutils.FatalUnless(t, z.IsReduced_a(), "cannot happen")
		zInt := z.ToBigInt()
		zInt.Mod(zInt, baseFieldSize_Int)

		xInt := x.ToBigInt()
		xInt.Add(xInt, xInt)
		xInt.Mod(xInt, baseFieldSize_Int)

		testutils.FatalUnless(t, xInt.Cmp(zInt) == 0, "DoubleEqAndReduce_a does not match big.Int addition")
	}
}

func TestUint256MulEqAndReduce_a(t *testing.T) {
	prepareTestFieldElements(t)

	const num = 256

	xs := CachedUint256.GetElements(pc_uint256_a, num)
	ys := CachedUint256.GetElements(pc_uint256_a, num)
	var z Uint256
	var zInt *big.Int = new(big.Int)
	var resInt *big.Int = new(big.Int)
	var xInt *big.Int = new(big.Int)
	var yInt *big.Int = new(big.Int)
	for _, x := range xs {
		for _, y := range ys {
			z = x
			z.MulEqAndReduce_a(&y)
			testutils.FatalUnless(t, z.IsReduced_a(), "MulEqAndReduce_a does not reduce properly")

			xInt = x.ToBigInt()
			yInt = y.ToBigInt()
			zInt.Mul(xInt, yInt)
			zInt.Mod(zInt, baseFieldSize_Int)

			resInt = z.ToBigInt()
			resInt.Mod(resInt, baseFieldSize_Int)

			testutils.FatalUnless(t, resInt.Cmp(zInt) == 0, "MulEqAndReduce_a does not macht big.Int's Mul")
		}
	}
}

func TestUint256MulAndReduce_a(t *testing.T) {
	prepareTestFieldElements(t)

	const num = 256

	xs := CachedUint256.GetElements(pc_uint256_a, num)
	ys := CachedUint256.GetElements(pc_uint256_a, num)
	var z Uint256
	var zInt *big.Int = new(big.Int)
	var resInt *big.Int = new(big.Int)
	var xInt *big.Int = new(big.Int)
	var yInt *big.Int = new(big.Int)
	for _, x := range xs {
		for _, y := range ys {
			z.MulAndReduce_a(&x, &y)
			testutils.FatalUnless(t, z.IsReduced_a(), "MulAndReduce_a does not reduce properly")

			xInt = x.ToBigInt()
			yInt = y.ToBigInt()
			zInt.Mul(xInt, yInt)
			zInt.Mod(zInt, baseFieldSize_Int)

			resInt = z.ToBigInt()
			resInt.Mod(resInt, baseFieldSize_Int)

			testutils.FatalUnless(t, resInt.Cmp(zInt) == 0, "MulAndReduce_a does not macht big.Int's Mul")
		}
	}
}

func TestUint256_SquareEqAndReduce_a(t *testing.T) {
	prepareTestFieldElements(t)
	const num = 1000

	xs := CachedUint256.GetElements(pc_uint256_a, num)
	var z Uint256
	for _, x := range xs {

		z = x
		z.SquareEqAndReduce_a()
		testutils.FatalUnless(t, z.IsReduced_a(), "cannot happen")
		zInt := z.ToBigInt()
		zInt.Mod(zInt, baseFieldSize_Int)

		xInt := x.ToBigInt()
		xInt.Mul(xInt, xInt)
		xInt.Mod(xInt, baseFieldSize_Int)

		testutils.FatalUnless(t, xInt.Cmp(zInt) == 0, "SquareEqAndReduce_a does not match big.Int squaring")
	}
}

func TestUint256_SquareAndReduce_a(t *testing.T) {
	prepareTestFieldElements(t)
	const num = 1000

	xs := CachedUint256.GetElements(pc_uint256_a, num)
	var z Uint256
	for _, x := range xs {

		z.SquareAndReduce_a(&x)
		testutils.FatalUnless(t, z.IsReduced_a(), "cannot happen")
		zInt := z.ToBigInt()
		zInt.Mod(zInt, baseFieldSize_Int)

		xInt := x.ToBigInt()
		xInt.Mul(xInt, xInt)
		xInt.Mod(xInt, baseFieldSize_Int)

		testutils.FatalUnless(t, xInt.Cmp(zInt) == 0, "SquareAndReduce_a does not match big.Int squaring")
	}
}

func testUint256_Jacobi(t *testing.T, jacobifun func(*Uint256) int) {
	prepareTestFieldElements(t)
	const num = 1000

	var zeroUint Uint256
	testutils.FatalUnless(t, jacobifun(&zeroUint) == 0, "Jacobi symbol of 0 is not 0")
	testutils.FatalUnless(t, jacobifun(&baseFieldSize_uint256) == 0, "Jacobi of BaseFieldSize is not 0")
	testutils.FatalUnless(t, jacobifun(&twiceBaseFieldSize_uint256) == 0, "Jacobi of 2*BaseFieldSize is not 0")

	xs := CachedUint256.GetElements(pc_uint256_a, num)
	for _, x := range xs {
		xCopy := x
		xInt := x.ToBigInt()
		bigIntAns := big.Jacobi(xInt, baseFieldSize_Int)
		funAns := jacobifun(&xCopy)
		testutils.FatalUnless(t, xCopy == x, "Jacobi function modified argument")
		testutils.FatalUnless(t, funAns == bigIntAns, "Jacobi function does not agree with big.Int Jacobi function")
	}

	for i := uint(0); i < num; i++ {
		xInt := new(big.Int).Lsh(common.One_Int, i)
		xInt.Mod(xInt, baseFieldSize_Int)
		yInt := new(big.Int).Add(xInt, common.One_Int)
		yInt.Mod(yInt, baseFieldSize_Int)
		zInt := new(big.Int).Sub(xInt, common.One_Int)
		zInt.Mod(zInt, baseFieldSize_Int)
		var x, y, z Uint256
		x.SetBigInt(xInt)
		y.SetBigInt(yInt)
		z.SetBigInt(zInt)
		xJac := jacobifun(&x)
		yJac := jacobifun(&y)
		zJac := jacobifun(&z)
		xIntJac := big.Jacobi(xInt, baseFieldSize_Int)
		yIntJac := big.Jacobi(yInt, baseFieldSize_Int)
		zIntJac := big.Jacobi(zInt, baseFieldSize_Int)
		testutils.Assert(xIntJac == 1) // Jacobi symbol of 2 is +1, because BaseFieldSize % 8 == 1. Powering to i does not change this.
		testutils.FatalUnless(t, xJac == xIntJac, "Jacobi symbol wrong for power of 2")
		testutils.FatalUnless(t, yJac == yIntJac, "Jacobi symbol wrong for 1+power of 2")
		testutils.FatalUnless(t, zJac == zIntJac, "Jacobi symbol wrong for power of 2 -1")
	}

}

func TestUint256_Jacobi(t *testing.T) {
	testUint256_Jacobi(t, (*Uint256).jacobiV1_a)
}

// Test non-Montgomery variant of Exponentiation

func TestUint256_ModularExponentiation(t *testing.T) {
	prepareTestFieldElements(t)
	const num = 1000

	bases := CachedUint256.GetElements(SeedAndRange{seed: 10003, allowedRange: twoTo256_Int}, num)
	exponents := CachedUint256.GetElements(SeedAndRange{seed: 10004, allowedRange: twoTo256_Int}, num)

	var target Uint256
	target.ModularExponentiation_fa(&zero_uint256, &zero_uint256)
	testutils.FatalUnless(t, target.IsOne(), "0^0 != 1")

	for i, basis := range bases {
		exponent := exponents[i]
		exponentCopy := exponent
		basisCopy := basis
		basisInt := basis.ToBigInt()
		exponentInt := exponent.ToBigInt()
		target.ModularExponentiation_fa(&basis, &exponent)
		testutils.FatalUnless(t, target.IsReduced_f(), "ModularExponentiation_fa does not fully reduce")
		targetInt := new(big.Int).Exp(basisInt, exponentInt, baseFieldSize_Int)
		testutils.FatalUnless(t, target.ToBigInt().Cmp(targetInt) == 0, "ModularExponentiaion_fa does not match big.Int")

		target = basis
		target.ModularExponentiation_fa(&target, &exponent)
		testutils.FatalUnless(t, target.ToBigInt().Cmp(targetInt) == 0, "ModularExponentiaion_fa does not work for aliasing args")

		dummy1 := basisCopy
		dummy2 := basisCopy
		target.ModularExponentiation_fa(&dummy1, &dummy2)
		dummy1.ModularExponentiation_fa(&dummy1, &dummy1)
		testutils.FatalUnless(t, dummy1 == target, "ModularExponentiation_fa does not work for target, basis, exponent all aliasing")

		target.ModularExponentiation_fa(&basis, &one_uint256)
		dummy1 = basis
		dummy1.Reduce()
		testutils.FatalUnless(t, target == dummy1, "x^1 != x modulo BaseFieldSize")
		target.ModularExponentiation_fa(&basis, &zero_uint256)
		testutils.FatalUnless(t, target.IsOne(), "x^0 != 1")

		testutils.FatalUnless(t, exponent == exponentCopy, "Argument was modified during ModularExponentiation")
		testutils.FatalUnless(t, basis == basisCopy, "Argument was modified during ModularExponentiation")

	}

}
