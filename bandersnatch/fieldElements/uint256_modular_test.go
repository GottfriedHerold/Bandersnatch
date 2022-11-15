package fieldElements

import (
	"math/big"
	"testing"

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
	var z uint256
	var zInt *big.Int = new(big.Int)
	var resInt *big.Int = new(big.Int)
	var xInt *big.Int = new(big.Int)
	var yInt *big.Int = new(big.Int)
	for _, x := range xs {
		for _, y := range ys {
			z.AddAndReduce_b_c(&x, &y)
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
	var z uint256
	var zInt *big.Int = new(big.Int)
	var resInt *big.Int = new(big.Int)
	var xInt *big.Int = new(big.Int)
	var yInt *big.Int = new(big.Int)
	for _, x := range xs {
		for _, y := range ys {
			z.AddAndReduce_b_c(&x, &y)
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
	var z uint256
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
	var z uint256
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
	var z uint256
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
	var z uint256
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

	var z uint256
	var zInt *big.Int = new(big.Int)
	var resInt *big.Int = new(big.Int)
	var xInt *big.Int = new(big.Int)

	for i, x := range xs {
		if i == 0 {
			x = uint256{}
		}
		if i == 1 {
			x = baseFieldSize_uint256
		}
		if i == 2 {
			x = twiceBaseFieldSize_uint256
		}
		xCopy := x
		xCopy.reduceBarret_fa()
		if xCopy.IsZero() {
			xInt = x.ToBigInt()
			testutils.FatalUnless(t, zInt.ModInverse(xInt, baseFieldSize_Int) == nil, "Cannot happen")
			z = uint256{10, 20, 30, 50}
			ok := z.ModularInverse_a_NAIVEHAC(&x)
			testutils.FatalUnless(t, !ok, "ModularInverse_a_NAIVEHAC did not recognize non-invertible elements")
			testutils.FatalUnless(t, z == uint256{10, 20, 30, 50}, "ModularInverse_a_NAIVEHAC changed receiver upon getting zero")

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

func testReductionFunction(t *testing.T, reductionFunction func(*uint256), reducedInputSeed SeedAndRange, outputReducednessCheck func(*uint256) bool, funName string) {
	prepareTestFieldElements(t)

	const num = 1000

	xs := CachedUint256.GetElements(reducedInputSeed, num)

	var z uint256
	var zInt *big.Int = new(big.Int)
	var xInt *big.Int = new(big.Int)

	for _, x := range xs {
		z = x
		reductionFunction(&z)

		testutils.FatalUnless(t, outputReducednessCheck(&z), funName+" does not properly reduce")

		xInt = x.ToBigInt()
		xInt.Mod(xInt, baseFieldSize_Int)

		zInt = z.ToBigInt()
		zInt.Mod(zInt, baseFieldSize_Int)

		testutils.FatalUnless(t, xInt.Cmp(zInt) == 0, funName+" does not preserve element modulo BaseFieldSize")
	}
}

func TestUint256_Reduce(t *testing.T) {
	testReductionFunction(t, (*uint256).reduce_ca, pc_uint256_a, (*uint256).IsReduced_c, "reduce_ca")
	testReductionFunction(t, (*uint256).reduce_fb, pc_uint256_b, (*uint256).IsReduced_f, "reduce_fb")
	testReductionFunction(t, (*uint256).reduceBarret_fa, pc_uint256_a, (*uint256).IsReduced_f, "reduceBarret_fa")
}

func TestUint256_IsFullyReduced(t *testing.T) {
	prepareTestFieldElements(t)

	const num = 1000

	xs := CachedUint256.GetElements(pc_uint256_a, num) // TODO: Use a different seed here with special prepopulated values
	for i, x := range xs {
		if i == 0 {
			x = uint256{}
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
	var z uint256
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
	var z uint256
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
	var z uint256
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
	var z uint256
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
	var z uint256
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
	var z uint256
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
