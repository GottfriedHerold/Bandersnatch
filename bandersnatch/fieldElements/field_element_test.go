package fieldElements

import (
	"errors"
	"fmt"
	"math"
	"math/big"
	"testing"

	"github.com/GottfriedHerold/Bandersnatch/bandersnatch/common"
	"github.com/GottfriedHerold/Bandersnatch/bandersnatch/errorsWithData"
	"github.com/GottfriedHerold/Bandersnatch/internal/testutils"
	"github.com/GottfriedHerold/Bandersnatch/internal/utils"
)

var _ FieldElementInterface_common = &bsFieldElement_MontgomeryNonUnique{}
var _ FieldElementInterface[*bsFieldElement_MontgomeryNonUnique] = &bsFieldElement_MontgomeryNonUnique{}

var _ FieldElementInterface_common = &bsFieldElement_BigInt{}
var _ FieldElementInterface[*bsFieldElement_BigInt] = &bsFieldElement_BigInt{}

// var fatalUnless = testutils.FatalUnless

func TestFieldElementProperties(t *testing.T) {
	t.Run("Montgomery implementation", testAllFieldElementProperties[bsFieldElement_MontgomeryNonUnique])
	t.Run("trivial big.Int implementation", testAllFieldElementProperties[bsFieldElement_BigInt])
}

func testAllFieldElementProperties[FE any, FEPtr interface {
	*FE
	FieldElementInterface[FEPtr]
}](t *testing.T) {
	t.Run("Constants", testFEProperty_Constants[FE, FEPtr])
	t.Run("BigInt roundtrip", testFEProperty_BigIntRoundtrip[FE, FEPtr](10001, 10002, 1000))
	t.Run("Uint256 roundtrip", testFEProperty_Uint256Roundtrip[FE, FEPtr](10001, 10002, 1000))
	t.Run("Uint64 and Int64 roundtrip", testFEProperty_SmallIntConversion[FE, FEPtr](10001, 10002, 1000))
	t.Run("Commutativity and inversion", testFEProperty_CommutativiteAndInverses[FE, FEPtr](10001, 10001, 100, 50))
	t.Run("Aliasing and Eq", testFEProperty_Aliasing[FE, FEPtr](10001, 10001, 1000, 100, 100))
	t.Run("Associativity", testFEProperty_Associativity[FE, FEPtr](10001, 10001, 10001, 100, 50, 30))
	t.Run("Distributivity", testFEProperty_Distributivity[FE, FEPtr](10001, 10001, 10001, 100, 30, 30))
	t.Run("MulByFive", testFEProperty_MulFive[FE, FEPtr](10001, 1000))
	t.Run("raw serialization", testFEProperty_BytesRoundtrip[FE, FEPtr](10001, 1000))
	t.Run("internal representation", testFEProperty_InternalRep[FE, FEPtr](10001, 1000, 100))
	t.Run("Small-Arg Operations", testFEProperty_SmallOps[FE, FEPtr](10001, 1000))
	t.Run("Sign", testFEProperty_Sign[FE, FEPtr](10001, 100, 100))
	t.Run("CmpAbs", testFEProperty_CmpAbs[FE, FEPtr](10001, 1000))
	t.Run("Formatted output", testFEProperty_FormattedOutput[FE, FEPtr](10001, 1000))
}

// For copy&pasting:
/*
func testFEProperty__[FE any, FEPtr interface {
	*FE
	FieldElementInterface[FEPtr]
}]() func(t *testing.T) {
	return func(t *testing.T){
	prepareTestFieldElements(t)
	}
}
*/

// test that SetOne, SetZero behave as expected wrt. IsZero, IsOne

func testFEProperty_Constants[FE any, FEPtr interface {
	*FE
	FieldElementInterface[FEPtr]
}](t *testing.T) {
	prepareTestFieldElements(t)
	var feVal, feVal2 FE
	var fe FEPtr = FEPtr(&feVal)
	var fe2 = FEPtr(&feVal2)
	fe.SetOne()
	fe2.SetOne()

	// Check 1 and 0

	testutils.FatalUnless(t, fe.IsOne(), "1 != 1")
	testutils.FatalUnless(t, fe.IsEqual(fe2), "1 != 1")
	testutils.FatalUnless(t, !fe.IsZero(), "one is zero")
	testutils.FatalUnless(t, !fe2.IsZero(), "one is zero")

	fe.SetZero()
	fe2.SetZero()

	testutils.FatalUnless(t, fe.IsZero(), "0 != 0")
	testutils.FatalUnless(t, fe2.IsZero(), "0 != 0")
	testutils.FatalUnless(t, fe.IsEqual(fe2), "zeros differ")
	testutils.FatalUnless(t, !fe.IsOne(), "zero is one")
}

// Very extensive test:
// This tests that Add, Sub, Mul, Divide work properly if receiver and/or some arguments alias (all combination)
// Also tests this for AddEq, SubEq, MulEq, DivideEq and compares against Add,Sub,Mul, Divide
// Checks results against Double, Zero, Square, 1 and DoubleEq,SquareEq
// Checks MulFive vs MulFiveEq vs Mul(5)
// Checks the int64 and uint64 variants OpUint64 and OpInt64 for Op in {Add, Sub, Mul, Divide} against aliased and convert+Op.
//
// Note that other tests essentially rely.

func testFEProperty_Aliasing[FE any, FEPtr interface {
	*FE
	FieldElementInterface[FEPtr]
}](seedx int64, seedy int64, numUnary int, numX int, numY int) func(t *testing.T) {
	return func(t *testing.T) {
		prepareTestFieldElements(t)

		safeTarget := FEPtr(new(FE))

		var target1Val, target2Val, target3Val, target4Val FE
		target1 := FEPtr(&target1Val)
		target2 := FEPtr(&target2Val)
		target3 := FEPtr(&target3Val)
		target4 := FEPtr(&target4Val)

		// Check that x.Fun(x) and t.Fun(x) and x.FunEq() match for Fun that doesn't read from the receiver.
		// const num = 100

		var xs []FE = GetPrecomputedFieldElements[FE, FEPtr](seedx, numUnary)
		for i, xVal := range xs {
			x := FEPtr(&xVal)
			switch i {
			case 0:
				x.SetZero()
			case 1:
				x.SetOne()
			}
			var xCopy1Val, xCopy2Val FE
			xCopy1 := FEPtr(&xCopy1Val)
			xCopy2 := FEPtr(&xCopy2Val)

			// Neg:
			xCopy1Val = xVal
			xCopy2Val = xVal
			safeTarget.Neg(xCopy1)
			xCopy2.Neg(xCopy2)
			xCopy1.NegEq()
			testutils.FatalUnless(t, safeTarget.IsEqual(xCopy2), "Aliasing fails for Neg")
			testutils.FatalUnless(t, safeTarget.IsEqual(xCopy1), "Aliasing Eq fails for Neg")

			// Square:
			xCopy1Val = xVal
			xCopy2Val = xVal
			safeTarget.Square(xCopy1)
			xCopy2.Square(xCopy2)
			xCopy1.SquareEq()
			testutils.FatalUnless(t, safeTarget.IsEqual(xCopy2), "Aliasing fails for Square")
			testutils.FatalUnless(t, safeTarget.IsEqual(xCopy1), "Aliasing Eq fails for Square")

			// Double:
			xCopy1Val = xVal
			xCopy2Val = xVal
			safeTarget.Double(xCopy1)
			xCopy2.Double(xCopy2)
			xCopy1.DoubleEq()
			testutils.FatalUnless(t, safeTarget.IsEqual(xCopy2), "Aliasing fails for Double")
			testutils.FatalUnless(t, safeTarget.IsEqual(xCopy1), "Aliasing Eq fails for Double")

			// MulFive
			xCopy1Val = xVal
			xCopy2Val = xVal
			safeTarget.MulInt64(xCopy1, 5)
			xCopy2.MulFive(xCopy2)
			xCopy1.MulEqFive()
			testutils.FatalUnless(t, safeTarget.IsEqual(xCopy2), "Aliasing fails for MulFive")
			testutils.FatalUnless(t, safeTarget.IsEqual(xCopy1), "Aliasing Eq fails for MulEqFive")

			//Inv:
			xCopy1Val = xVal
			xCopy2Val = xVal
			safeTarget.SetUint64(2)
			panic1 := testutils.CheckPanic(func() { safeTarget.Inv(xCopy1) })
			panic2 := testutils.CheckPanic(func() { xCopy2.Inv(xCopy2) })
			panic3 := testutils.CheckPanic(func() { xCopy1.InvEq() })
			if x.IsZero() {
				testutils.FatalUnless(t, panic1, "Inv did not panic")
				testutils.FatalUnless(t, panic2, "Inv did not panic")
				testutils.FatalUnless(t, panic3, "Inv did not panic")
				testutils.FatalUnless(t, xCopy2.IsZero(), "Inv on 0 modifies argument")
				testutils.FatalUnless(t, xCopy1.IsZero(), "Inv on 0 modifies argument")
				v, err := safeTarget.ToUint64()
				testutils.FatalUnless(t, err == nil, "Inv by 0 changed receiver")
				testutils.FatalUnless(t, v == 2, "Inv by 0 changed receiver")
			} else {
				testutils.FatalUnless(t, !panic1, "Inv did panic on non-zero argument")
				testutils.FatalUnless(t, !panic2, "Inv did panic on non-zero argument")
				testutils.FatalUnless(t, !panic3, "Inv did panic on non-zero argument")
				testutils.FatalUnless(t, safeTarget.IsEqual(xCopy2), "Aliasing fails for Inv")
				testutils.FatalUnless(t, safeTarget.IsEqual(xCopy1), "Aliasing Eq fails for Inv")
			}

			// Small-Op by constant
			xCopy1Val = xVal
			xCopy2Val = xVal
			safeTarget.AddUint64(xCopy1, 5)
			xCopy2.AddUint64(xCopy2, 5)
			testutils.FatalUnless(t, safeTarget.IsEqual(xCopy2), "Aliasing failure for AddUint64")

			xCopy1Val = xVal
			xCopy2Val = xVal
			safeTarget.SubUint64(xCopy1, 5)
			xCopy2.SubUint64(xCopy2, 5)
			testutils.FatalUnless(t, safeTarget.IsEqual(xCopy2), "Aliasing failure for SubUint64")

			xCopy1Val = xVal
			xCopy2Val = xVal
			safeTarget.MulUint64(xCopy1, 5)
			xCopy2.MulUint64(xCopy2, 5)
			testutils.FatalUnless(t, safeTarget.IsEqual(xCopy2), "Aliasing failure for MulUint64")

			xCopy1Val = xVal
			xCopy2Val = xVal
			safeTarget.DivideUint64(xCopy1, 5)
			xCopy2.DivideUint64(xCopy2, 5)
			testutils.FatalUnless(t, safeTarget.IsEqual(xCopy2), "Aliasing failure for DivideUint64")

			xCopy1Val = xVal
			xCopy2Val = xVal
			safeTarget.AddInt64(xCopy1, 5)
			xCopy2.AddInt64(xCopy2, 5)
			testutils.FatalUnless(t, safeTarget.IsEqual(xCopy2), "Aliasing failure for AddInt64")

			xCopy1Val = xVal
			xCopy2Val = xVal
			safeTarget.SubInt64(xCopy1, 5)
			xCopy2.SubInt64(xCopy2, 5)
			testutils.FatalUnless(t, safeTarget.IsEqual(xCopy2), "Aliasing failure for SubInt64")

			xCopy1Val = xVal
			xCopy2Val = xVal
			safeTarget.MulInt64(xCopy1, 5)
			xCopy2.MulInt64(xCopy2, 5)
			testutils.FatalUnless(t, safeTarget.IsEqual(xCopy2), "Aliasing failure for MulInt64")

			xCopy1Val = xVal
			xCopy2Val = xVal
			safeTarget.DivideInt64(xCopy1, 5)
			xCopy2.DivideInt64(xCopy2, 5)
			testutils.FatalUnless(t, safeTarget.IsEqual(xCopy2), "Aliasing failure for DivideInt64")

			// Binary functions, both arguments alias
			// Add
			xCopy1Val = xVal
			xCopy2Val = xVal
			target1Val = xVal
			target4Val = xVal
			safeTarget.Add(xCopy1, xCopy2)
			target1.Add(target1, target1)
			target2.Add(xCopy1, xCopy1)
			target3.Double(xCopy2)
			target4.AddEq(target4)
			testutils.FatalUnless(t, safeTarget.IsEqual(target1), "Aliasing failure for Add")
			testutils.FatalUnless(t, safeTarget.IsEqual(target2), "Aliasing failure for Add")
			testutils.FatalUnless(t, safeTarget.IsEqual(target3), "Add(x,x) differs from Double")
			testutils.FatalUnless(t, safeTarget.IsEqual(target4), "Aliasing failure for AddEq")

			testutils.FatalUnless(t, xCopy1.IsEqual(x), "unexpected modification")
			testutils.FatalUnless(t, xCopy2.IsEqual(x), "unexpected modification")

			// Sub
			xCopy1Val = xVal
			xCopy2Val = xVal
			target1Val = xVal
			target4Val = xVal
			safeTarget.Sub(xCopy1, xCopy2)
			target1.Sub(target1, target1)
			target2.Sub(xCopy1, xCopy1)
			target3.SetZero()
			target4.SubEq(target4)
			testutils.FatalUnless(t, safeTarget.IsEqual(target1), "Aliasing failure for Sub")
			testutils.FatalUnless(t, safeTarget.IsEqual(target2), "Aliasing failure for Sub")
			testutils.FatalUnless(t, safeTarget.IsEqual(target3), "Sub(x,x) differs from 0")
			testutils.FatalUnless(t, safeTarget.IsEqual(target4), "Aliasing failure for SubEq")
			testutils.FatalUnless(t, xCopy1.IsEqual(x), "unexpected modification")
			testutils.FatalUnless(t, xCopy2.IsEqual(x), "unexpected modification")

			// Mul
			xCopy1Val = xVal
			xCopy2Val = xVal
			target1Val = xVal
			target4Val = xVal

			safeTarget.Mul(xCopy1, xCopy2)
			target1.Mul(target1, target1)
			target2.Mul(xCopy1, xCopy1)
			target3.Square(xCopy2)
			target4.MulEq(target4)
			testutils.FatalUnless(t, safeTarget.IsEqual(target1), "Aliasing failure for Mul")
			testutils.FatalUnless(t, safeTarget.IsEqual(target2), "Aliasing failure for Mul")
			testutils.FatalUnless(t, safeTarget.IsEqual(target3), "Mul(x,x) differs from Square")
			testutils.FatalUnless(t, safeTarget.IsEqual(target4), "Aliasing failure for MulEq")

			testutils.FatalUnless(t, xCopy1.IsEqual(x), "unexpected modification")
			testutils.FatalUnless(t, xCopy2.IsEqual(x), "unexpected modification")

			// Divide
			xCopy1Val = xVal
			xCopy2Val = xVal
			target1Val = xVal
			target4Val = xVal
			safeTarget.SetUint64(2)
			target2.SetUint64(2)
			panic1 = testutils.CheckPanic(func() { safeTarget.Divide(xCopy1, xCopy2) })
			panic2 = testutils.CheckPanic(func() { target1.Divide(target1, target1) })
			panic3 = testutils.CheckPanic(func() { target2.Divide(xCopy1, xCopy1) })
			panic4 := testutils.CheckPanic(func() { target4.DivideEq(target4) })

			if x.IsZero() {
				testutils.FatalUnless(t, panic1, "0/0 did not panic")
				testutils.FatalUnless(t, panic2, "0/0 did not panic")
				testutils.FatalUnless(t, panic3, "0/0 did not panic")
				testutils.FatalUnless(t, panic4, "0/0 did not panic")

				testutils.FatalUnless(t, safeTarget.IsEqual(target2), "0/0 Division modified receiver")
				v, err := safeTarget.ToUint64()
				testutils.FatalUnless(t, err == nil, "0/0 changed receiver")
				testutils.FatalUnless(t, v == 2, "0/0 changed receiver")
				testutils.FatalUnless(t, target1.IsZero(), "0/0 changed receiver")
				testutils.FatalUnless(t, target4.IsZero(), "0/0 changed receiver")
			} else {
				testutils.FatalUnless(t, !panic1, "Divide by non-zero did panic")
				testutils.FatalUnless(t, !panic2, "Divide by non-zero did panic")
				testutils.FatalUnless(t, !panic3, "Divide by non-zero did panic")
				testutils.FatalUnless(t, !panic4, "Divide by non-zero did panic")
				testutils.FatalUnless(t, safeTarget.IsEqual(target1), "Aliasing failure for Divide")
				testutils.FatalUnless(t, safeTarget.IsEqual(target2), "Aliasing failure for Divide")
				testutils.FatalUnless(t, safeTarget.IsOne(), "Divide(x,x) not 1")
				testutils.FatalUnless(t, safeTarget.IsEqual(target4), "Aliasing failure for DivideEq")
				testutils.FatalUnless(t, xCopy1.IsEqual(x), "unexpected modification")
				testutils.FatalUnless(t, xCopy2.IsEqual(x), "unexpected modification")
			}
		}

		// binary functions:
		xs = GetPrecomputedFieldElements[FE, FEPtr](seedx, numX)
		ys := GetPrecomputedFieldElements[FE, FEPtr](seedy, numY)
		for i, xVal := range xs {
			x := FEPtr(&xVal)
			switch i {
			case 0:
				x.SetZero()
			case 1:
				x.SetOne()
			}
			for j, yVal := range ys {
				y := FEPtr(&yVal)
				switch j {
				case 0:
					y.SetZero()
				case 1:
					y.SetOne()
				case 2:
					y.Neg(x)
				}
				// x, y are pointers to xVal, yVal

				var xCopyVal FE
				var yCopyVal FE
				xCopy := FEPtr(&xCopyVal)
				yCopy := FEPtr(&yCopyVal)

				// Check x.Op(x,y) and y.Op(x,y) and x.OpEq(y)
				// Add
				xCopyVal = xVal
				yCopyVal = yVal
				target1Val = xVal
				target2Val = yVal
				target3Val = xVal
				safeTarget.Add(xCopy, yCopy)
				target1.Add(target1, yCopy)
				target2.Add(xCopy, target2)
				target3.AddEq(yCopy)
				testutils.FatalUnless(t, safeTarget.IsEqual(target1), "Aliasing failure for Add")
				testutils.FatalUnless(t, safeTarget.IsEqual(target2), "Aliasing failure for Add")
				testutils.FatalUnless(t, safeTarget.IsEqual(target3), "Aliasing failure for Add vs Eq")

				// Sub
				xCopyVal = xVal
				yCopyVal = yVal
				target1Val = xVal
				target2Val = yVal
				target3Val = xVal
				safeTarget.Sub(xCopy, yCopy)
				target1.Sub(target1, yCopy)
				target2.Sub(xCopy, target2)
				target3.SubEq(yCopy)
				testutils.FatalUnless(t, safeTarget.IsEqual(target1), "Aliasing failure for Sub")
				testutils.FatalUnless(t, safeTarget.IsEqual(target2), "Aliasing failure for Sub")
				testutils.FatalUnless(t, safeTarget.IsEqual(target3), "Aliasing failure for Sub vs Eq")

				// Mul
				xCopyVal = xVal
				yCopyVal = yVal
				target1Val = xVal
				target2Val = yVal
				target3Val = xVal
				safeTarget.Mul(xCopy, yCopy)
				target1.Mul(target1, yCopy)
				target2.Mul(xCopy, target2)
				target3.MulEq(yCopy)
				testutils.FatalUnless(t, safeTarget.IsEqual(target1), "Aliasing failure for Mul")
				testutils.FatalUnless(t, safeTarget.IsEqual(target2), "Aliasing failure for Mul")
				testutils.FatalUnless(t, safeTarget.IsEqual(target3), "Aliasing failure for Mul vs Eq")

				// Divide
				xCopyVal = xVal
				yCopyVal = yVal
				target1Val = xVal
				target2Val = yVal
				target3Val = xVal
				panic0 := testutils.CheckPanic(func() { safeTarget.Divide(xCopy, yCopy) })
				panic1 := testutils.CheckPanic(func() { target1.Divide(target1, yCopy) })
				panic2 := testutils.CheckPanic(func() { target2.Divide(xCopy, target2) })
				panic3 := testutils.CheckPanic(func() { target3.DivideEq(yCopy) })
				if y.IsZero() {
					testutils.FatalUnless(t, panic0, "Divide by 0 did not panic")
					testutils.FatalUnless(t, panic1, "Divide by 0 did not panic")
					testutils.FatalUnless(t, panic2, "Divide by 0 did not panic")
					testutils.FatalUnless(t, panic3, "DivideEq by 0 did not panic")
					testutils.FatalUnless(t, target1.IsEqual(x), "Divide by 0 changed argument")
					testutils.FatalUnless(t, target2.IsZero(), "Divide by 0 changed argument")
					testutils.FatalUnless(t, target3.IsEqual(x), "DivideEq by 0 changed argument")
				} else {
					testutils.FatalUnless(t, !panic0, "Divide by non-zero did panic")
					testutils.FatalUnless(t, !panic1, "Divide by non-zero did panic")
					testutils.FatalUnless(t, !panic2, "Divide by non-zero did panic")
					testutils.FatalUnless(t, !panic3, "DivideEq by non-zero did panic")
					testutils.FatalUnless(t, safeTarget.IsEqual(target1), "Aliasing failure for Divide")
					testutils.FatalUnless(t, safeTarget.IsEqual(target2), "Aliasing failure for Divide")
					testutils.FatalUnless(t, safeTarget.IsEqual(target3), "Aliasing failure for Divide vs Eq")
				}
			}
		}
	}
}

// Checks that
//
//	x + 0 == 0 + x == x, x + y == y + x.
//	x * 1 == 1 * x == x, x * y == y * x
//	0 - x == -x  (Sub(0, x) and Neg(x) agree)
//	Divide(1,x) and Inv(x) agree
//	1/x * x == 1
//	x + (-x) == 0
//	(x / y) * y == x
//	(x - y) + y == x
func testFEProperty_CommutativiteAndInverses[FE any, FEPtr interface {
	*FE
	FieldElementInterface[FEPtr]
}](seedX int64, seedY int64, numX int, numY int) func(t *testing.T) {
	return func(t *testing.T) {
		prepareTestFieldElements(t)

		var xs []FE = GetPrecomputedFieldElements[FE, FEPtr](seedX, numX)
		var ys []FE = GetPrecomputedFieldElements[FE, FEPtr](seedY, numY)

		z1 := FEPtr(new(FE))
		z2 := FEPtr(new(FE))

		for _, xVal := range xs {
			xValCopy := xVal
			x := FEPtr(&xVal)

			// x + 0 == x == 0 + x
			z2.SetZero()
			z1.Add(z2, x)
			testutils.FatalUnless(t, z1.IsEqual(x), "x+0 != x")
			z1.Add(x, z2)
			testutils.FatalUnless(t, z1.IsEqual(x), "x+0 != x")
			testutils.FatalUnless(t, x.IsEqual(&xValCopy), "")

			// x * 1 == x == 1 * x
			z2.SetOne()
			z1.Mul(z2, x)
			testutils.FatalUnless(t, z1.IsEqual(x), "x * 1 != x")
			z1.Mul(x, z2)
			testutils.FatalUnless(t, z1.IsEqual(x), "x * 1 != x")
			testutils.FatalUnless(t, x.IsEqual(&xValCopy), "")

			// x - x == 0 checked in Aliasing tests
			// x / x == 1 checked in Aliasing tests

			// 0 - x is the same as Neg and -x + x == 0
			z1.Neg(x)
			z2.SetZero()
			z2.Sub(z2, x)
			testutils.FatalUnless(t, z1.IsEqual(z2), "-x != 0-x")
			testutils.FatalUnless(t, x.IsEqual(&xValCopy), "")
			z1.AddEq(x)
			testutils.FatalUnless(t, z1.IsZero(), "-x + x != 0")
			testutils.FatalUnless(t, x.IsEqual(&xValCopy), "")

			// x / 1 == 1
			z2.SetOne()
			z1.Divide(x, z2)
			testutils.FatalUnless(t, z1.IsEqual(x), "x / 1 != x")
			testutils.FatalUnless(t, x.IsEqual(&xValCopy), "")
			testutils.FatalUnless(t, z2.IsOne(), "")

			// 1/(1/x) == x
			if !x.IsZero() {
				z2.SetOne()
				z1.Divide(z2, x)
				z2.Inv(x)
				testutils.FatalUnless(t, z1.IsEqual(z2), "1/x does not match Inv")
				z1.MulEq(x)
				z2.InvEq()
				testutils.FatalUnless(t, z1.IsOne(), "x * 1/x != 1")
				testutils.FatalUnless(t, z2.IsEqual(x), "1/(1/x) != x")
			}

			for _, yVal := range ys {

				yValCopy := yVal
				y := FEPtr(&yVal)

				// addition commutes
				z1.Add(x, y)
				z2.Add(y, x)

				testutils.FatalUnless(t, z1.IsEqual(z2), "x + y != y + X")
				testutils.FatalUnless(t, x.IsEqual(&xValCopy), "")
				testutils.FatalUnless(t, y.IsEqual(&yValCopy), "")

				// multiplication commutes
				z1.Mul(x, y)
				z2.Mul(y, x)
				testutils.FatalUnless(t, z1.IsEqual(z2), "x * y != y * x")
				testutils.FatalUnless(t, x.IsEqual(&xValCopy), "")
				testutils.FatalUnless(t, y.IsEqual(&yValCopy), "")

				// Check def. of subtraction
				z1.Sub(x, y)
				z2.Add(z1, y)
				testutils.FatalUnless(t, z2.IsEqual(&xVal), "x-y+y != x")
				testutils.FatalUnless(t, x.IsEqual(&xValCopy), "")
				testutils.FatalUnless(t, y.IsEqual(&yValCopy), "")

				// Check def. of division
				if !y.IsZero() {
					z1.Divide(x, y)
					z2.Mul(z1, y)
					testutils.FatalUnless(t, z2.IsEqual(&xVal), "x/y * y != x")
					testutils.FatalUnless(t, x.IsEqual(&xValCopy), "")
					testutils.FatalUnless(t, y.IsEqual(&yValCopy), "")
				}
			}
		}
	}
}

// check that (assuming commutativity)
//
//	(x + y) + z == x + (y + z)
//	(x * y) * z == x * (y * z)
func testFEProperty_Associativity[FE any, FEPtr interface {
	*FE
	FieldElementInterface[FEPtr]
}](seedX int64, seedY int64, seedZ int64, numX int, numY int, numZ int) func(t *testing.T) {
	return func(t *testing.T) {
		prepareTestFieldElements(t)

		var xs []FE = GetPrecomputedFieldElements[FE, FEPtr](seedX, numX)
		var ys []FE = GetPrecomputedFieldElements[FE, FEPtr](seedY, numY)
		var zs []FE = GetPrecomputedFieldElements[FE, FEPtr](seedZ, numZ)

		var target1Val, target2Val FE
		target1 := FEPtr(&target1Val)
		target2 := FEPtr(&target2Val)

		for _, xVal := range xs {

			x := FEPtr(&xVal)
			for _, yVal := range ys {

				y := FEPtr(&yVal)
				for _, zVal := range zs {

					z := FEPtr(&zVal)

					target1.Add(x, y)
					target1.Add(target1, z)
					target2.Add(y, z)
					target2.Add(x, target2)

					testutils.FatalUnless(t, target1.IsEqual(target2), "(x+y)+z != x+(y+z)")

					target1.Mul(x, y)
					target1.Mul(target1, z)
					target2.Mul(y, z)
					target2.Mul(x, target2)

					testutils.FatalUnless(t, target1.IsEqual(target2), "(x*y)*z != x*(y*z)")

				}
			}
		}
	}
}

// Check (assuming commutativity) that x * z + y * z == (x + y) * z
func testFEProperty_Distributivity[FE any, FEPtr interface {
	*FE
	FieldElementInterface[FEPtr]
}](seedX int64, seedY int64, seedZ int64, numX int, numY int, numZ int) func(t *testing.T) {
	return func(t *testing.T) {
		prepareTestFieldElements(t)

		var xs []FE = GetPrecomputedFieldElements[FE, FEPtr](seedX, numX)
		var ys []FE = GetPrecomputedFieldElements[FE, FEPtr](seedY, numY)
		var zs []FE = GetPrecomputedFieldElements[FE, FEPtr](seedZ, numZ)

		var target1Val, target2Val FE
		target1 := FEPtr(&target1Val)
		target2 := FEPtr(&target2Val)

		for _, xVal := range xs {

			x := FEPtr(&xVal)
			for _, yVal := range ys {

				y := FEPtr(&yVal)
				for _, zVal := range zs {

					z := FEPtr(&zVal)

					target1.Mul(x, z)
					target2.Mul(y, z)
					target1.AddEq(target2) // x*z + y*z

					target2.Add(x, y)
					target2.MulEq(z)

					testutils.FatalUnless(t, target1.IsEqual(target2), "x*z + y*z != (x+y) * z)")
				}
			}
		}
	}
}

// Test that MulFive actually multiplies by five. This is kind-of redundant with the aliasing tests, but kept for clarity.

func testFEProperty_MulFive[FE any, FEPtr interface {
	*FE
	FieldElementInterface[FEPtr]
}](seedX int64, numX int) func(t *testing.T) {
	return func(t *testing.T) {
		prepareTestFieldElements(t)
		var target1Val, target2Val, target3Val, target4Val, target5Val, target6Val FE
		target1 := FEPtr(&target1Val)
		target2 := FEPtr(&target2Val)
		target3 := FEPtr(&target3Val)
		target4 := FEPtr(&target4Val)
		target5 := FEPtr(&target5Val)
		target6 := FEPtr(&target6Val)

		var xs []FE = GetPrecomputedFieldElements[FE, FEPtr](seedX, numX)

		for _, xVal := range xs {
			x := FEPtr(&xVal)

			// multiply by 5 in six different ways:
			target1Val = xVal
			target1.MulEqFive()  // 1: MulEqFive
			target2.SetUint64(5) // 2: convert uint64(5) to fieldElement and MulEq
			target2.MulEq(x)
			target3.MulInt64(x, 5)  // 3: MulInt64(., 5)
			target4.MulUint64(x, 5) // 4: MulUint64(., 5)
			target5.Double(x)       // 5: Double twice, then add
			target5.DoubleEq()
			target5.AddEq(x)
			target6.Add(x, x) // 6: x + x + x + x + x
			target6.Add(target6, x)
			target6.Add(target6, x)
			target6.Add(target6, x)

			testutils.FatalUnless(t, target1.IsEqual(target2), "MulByfive does not match multiplication by 5")
			testutils.FatalUnless(t, target1.IsEqual(target3), "MulByfive does not match multiplication by 5")
			testutils.FatalUnless(t, target1.IsEqual(target4), "MulByfive does not match multiplication by 5")
			testutils.FatalUnless(t, target1.IsEqual(target5), "MulByfive does not match multiplication by 5")
			testutils.FatalUnless(t, target1.IsEqual(target6), "MulByfive does not match multiplication by 5")

		}
	}
}

// Checks that SetBigInt/ToBigInt roundtrip

func testFEProperty_BigIntRoundtrip[FE any, FEPtr interface {
	*FE
	FieldElementInterface[FEPtr]
}](seedFE int64, seedBigInt int64, num int) func(t *testing.T) {
	return func(t *testing.T) {
		prepareTestFieldElements(t)

		// test FE -> BigInt -> FE roundtrip
		var x2Val FE
		x2 := FEPtr(&x2Val)
		var xs []FE = GetPrecomputedFieldElements[FE, FEPtr](seedFE, num)
		for _, xVal := range xs {
			xCopy1 := xVal

			x1 := FEPtr(&xCopy1)
			xInt := x1.ToBigInt()
			x2.SetBigInt(xInt)
			testutils.FatalUnless(t, x1.IsEqual(x2), "FieldElement -> BigInt -> FieldElement does not roundtrip")
		}

		// test BigInt -> FE -> BigInt roundtrip
		rangeInt := new(big.Int).Add(twoTo256_Int, twoTo256_Int)                                                       // 2*2^512
		var bigInts []*big.Int = CachedBigInt.GetElements(SeedAndRange{seed: seedBigInt, allowedRange: rangeInt}, num) // "real" range of bigInt is [-2^512, 2^512), we subtract to center around 0 later.

		// temporary values used in the actual test
		var feVal FE
		fe := FEPtr(&feVal)
		big1 := new(big.Int)
		big2 := new(big.Int)

		for _, testBigInt := range bigInts {
			big1.Sub(testBigInt, twoTo512_Int) // we want our random big.Int's to cover negative numbers as well.
			fe.SetBigInt(big1)
			big2.Mod(big1, baseFieldSize_Int) // expected value after roundtrip.
			afterRoundtrip := fe.ToBigInt()

			testutils.FatalUnless(t, big2.Cmp(afterRoundtrip) == 0, "BigInt -> Field Element -> BigInt does not roundtrip modulo BaseFieldSize")
		}
	}
}

// Checks that SetUint256 and ToUint256 roundtrip as appropriate.

func testFEProperty_Uint256Roundtrip[FE any, FEPtr interface {
	*FE
	FieldElementInterface[FEPtr]
}](seedFE int64, seedUint256 int64, num int) func(t *testing.T) {
	return func(t *testing.T) {
		prepareTestFieldElements(t)

		// test FE -> Uint256 -> FE roundtrip
		var xUint256 Uint256
		var x2Val FE
		x2 := FEPtr(&x2Val)
		var xs []FE = GetPrecomputedFieldElements[FE, FEPtr](seedFE, num)
		for _, xVal := range xs {
			xCopy1 := xVal

			x1 := FEPtr(&xCopy1)
			x1.ToUint256(&xUint256)
			x2.SetUint256(&xUint256)
			testutils.FatalUnless(t, x1.IsEqual(x2), "FieldElement -> Uint256 -> FieldElements roundtrip failure")
		}

		// test uint256 -> FE -> uint256 roundtrip modulo reduction
		var uint256s []Uint256 = CachedUint256.GetElements(SeedAndRange{seed: seedUint256, allowedRange: twoTo256_Int}, num)

		// temporary values
		var feVal FE
		fe := FEPtr(&feVal)
		var uint256Reduced Uint256
		var uint256_1 Uint256
		var uint256_2 Uint256
		for _, testedUint256 := range uint256s {
			uint256_1 = testedUint256
			fe.SetUint256(&uint256_1)
			fe.ToUint256(&uint256_2)
			uint256Reduced = testedUint256
			uint256Reduced.reduceBarret_fa()

			testutils.FatalUnless(t, uint256Reduced == uint256_2, "Uint256 -> Field Element -> Uint256 roundtrip (modulo BaseField) failure")
			testutils.FatalUnless(t, testedUint256 == uint256_1, "")
		}
	}
}

// Checks that SetUint64, SetInt64, ToUint64, ToInt64 work correctly

func testFEProperty_SmallIntConversion[FE any, FEPtr interface {
	*FE
	FieldElementInterface[FEPtr]
}](seedUint64 int64, seedInt64 int64, num int) func(t *testing.T) {
	return func(t *testing.T) {
		prepareTestFieldElements(t)
		// const num = 1000
		var uint64s []uint64 = CachedUint64.GetElements(seedUint64, num)
		var int64s []int64 = CachedInt64.GetElements(seedInt64, num)
		var feVal FE
		fe := FEPtr(&feVal)

		// roundtrip uint64 -> FE -> uint64
		for _, u64 := range uint64s {
			fe.SetUint64(u64)
			retrieve, err := fe.ToUint64()
			testutils.FatalUnless(t, err == nil, "Uint64 -> FE -> Uint64 caused error")
			testutils.FatalUnless(t, retrieve == u64, "Uint64 -> FE -> Uint64 did not roundtrip")

			asInt64, err := fe.ToInt64()
			if u64 <= math.MaxInt64 {
				testutils.FatalUnless(t, err == nil, "ToInt64 caused error on small element")
				testutils.FatalUnless(t, uint64(asInt64) == u64, "ToInt64 did not return expected value")
			} else {
				testutils.FatalUnless(t, err != nil, "ToInt64 caused no error, even though we expected it")
				testutils.FatalUnless(t, errors.Is(err, ErrCannotRepresentFieldElement), "ToInt64 did not return expected error")
				feAny, ok := errorsWithData.GetParameterFromError(err, "FieldElement")
				testutils.FatalUnless(t, ok, "")
				feFe := feAny.(FE)
				testutils.FatalUnless(t, fe.IsEqual(&feFe), "error did not contain erroneous field element")
			}

		}

		// roundtrip int64 -> FE -> int64
		for _, i64 := range int64s {
			fe.SetInt64(i64)
			retrieve, err := fe.ToInt64()
			testutils.FatalUnless(t, err == nil, "Int64 -> FE -> Int64 caused error")
			testutils.FatalUnless(t, retrieve == i64, "Int64 -> FE -> Int64 did not roundtrip")

			asUint64, err := fe.ToUint64()
			if i64 >= 0 {
				testutils.FatalUnless(t, err == nil, "ToUint64 caused error on small element")
				testutils.FatalUnless(t, uint64(i64) == asUint64, "ToUint64 did not return expected value")
			} else {
				testutils.FatalUnless(t, err != nil, "ToUint64 caused no error, even though we expected it")
				testutils.FatalUnless(t, errors.Is(err, ErrCannotRepresentFieldElement), "ToUint64 did not return expected error")
				feAny, ok := errorsWithData.GetParameterFromError(err, "FieldElement")
				testutils.FatalUnless(t, ok, "")
				feFe := feAny.(FE)
				testutils.FatalUnless(t, fe.IsEqual(&feFe), "error did not contain erroneous field element")
			}
		}

		// test for special values of xInt
		twoTo63 := new(big.Int).Lsh(common.One_Int, 63)
		minusTwoTo63 := new(big.Int).Neg(twoTo63)
		for i := uint(0); i < BaseFieldBitLength-1; i++ {
			twoToi := new(big.Int).Lsh(common.One_Int, i)
			minusToi := new(big.Int).Neg(twoToi)
			FFi := new(big.Int).Sub(twoToi, common.One_Int)
			minusFFi := new(big.Int).Neg(FFi)

			for _, xInt := range []*big.Int{twoToi, minusToi, FFi, minusFFi} {

				fe.SetBigInt(xInt)
				var uint64good bool = xInt.Sign() >= 0 && xInt.Cmp(common.TwoTo64_Int) < 0
				var int64good bool = xInt.Cmp(minusTwoTo63) >= 0 && xInt.Cmp(twoTo63) < 0

				asUint64, err := fe.ToUint64()
				testutils.FatalUnless(t, (err == nil) == uint64good, "ToUint64 does not have expected error behaviour")
				if err == nil {
					testutils.FatalUnless(t, asUint64 == xInt.Uint64(), "")
				} else {
					feAny, ok := errorsWithData.GetParameterFromError(err, "FieldElement")
					testutils.FatalUnless(t, ok, "")
					feFe := feAny.(FE)
					feBig := FEPtr(&feFe).ToBigInt()
					xModBaseField := new(big.Int).Mod(xInt, baseFieldSize_Int)
					testutils.FatalUnless(t, feBig.Cmp(xModBaseField) == 0, "error did not contain erroneous field element")
				}

				asInt64, err := fe.ToInt64()
				testutils.FatalUnless(t, (err == nil) == int64good, "ToInt64 does not have expected error behaviour")
				if err == nil {
					testutils.FatalUnless(t, asInt64 == xInt.Int64(), "")
				} else {
					feAny, ok := errorsWithData.GetParameterFromError(err, "FieldElement")
					testutils.FatalUnless(t, ok, "")
					feFe := feAny.(FE)
					feBig := FEPtr(&feFe).ToBigInt()
					xModBaseField := new(big.Int).Mod(xInt, baseFieldSize_Int)
					testutils.FatalUnless(t, feBig.Cmp(xModBaseField) == 0, "error did not contain erroneous field element")
				}

			}
		}
	}
}

// Checks that the raw SetBytes/ToBytes interface roundtrips as expected

func testFEProperty_BytesRoundtrip[FE any, FEPtr interface {
	*FE
	FieldElementInterface[FEPtr]
}](seed int64, num int) func(t *testing.T) {
	return func(t *testing.T) {
		prepareTestFieldElements(t)

		LEN := FEPtr(nil).BytesLength()
		var buf []byte = make([]byte, LEN+1)
		var buf2 []byte = make([]byte, LEN+1)

		var xCopyVal FE
		xCopy := FEPtr(&xCopyVal)
		var target1Val, target2Val FE
		target1 := FEPtr(&target1Val)
		target2 := FEPtr(&target2Val)

		var xs []FE = GetPrecomputedFieldElements[FE, FEPtr](seed, num)

		for _, xVal := range xs {
			x := FEPtr(&xVal)
			xCopyVal = xVal

			xCopy.ToBytes(buf)
			target1.SetBytes(buf)
			// bufs are 1 longer than needed. We check that this is actually ignored
			irrelevant := buf[LEN]
			buf[LEN] += 1
			target2.SetBytes(buf)
			testutils.Assert(copy(buf2[0:LEN], buf[0:LEN]) == LEN)
			xCopy.ToBytes(buf)
			testutils.FatalUnless(t, target1.IsEqual(x), "Roundtrip failure for raw serialization")
			testutils.FatalUnless(t, target2.IsEqual(x), "Roundtrip failure for raw serialization")
			testutils.FatalUnless(t, buf[LEN] == irrelevant+1, "ToBytes writes more bytes")
			target1.ToBytes(buf2)
			testutils.FatalUnless(t, utils.CompareSlices(buf[0:LEN], buf2[0:LEN]), "Rountrip failure for raw serialization")
		}
	}
}

// Checks that the internal representation works as expected wrt. Normalize and RerandomizeRepresentation:
func testFEProperty_InternalRep[FE any, FEPtr interface {
	*FE
	FieldElementInterface[FEPtr]
}](seed int64, num int, numReps uint64) func(t *testing.T) {
	return func(t *testing.T) {
		prepareTestFieldElements(t)

		LEN := FEPtr(nil).BytesLength()
		var buf []byte = make([]byte, LEN)
		var buf2 []byte = make([]byte, LEN)
		var buf3 []byte = make([]byte, LEN)

		var xCopyVal1, xCopyVal2 FE
		xCopy1 := FEPtr(&xCopyVal1)
		xCopy2 := FEPtr(&xCopyVal2)

		var xs []FE = GetPrecomputedFieldElements[FE, FEPtr](seed, num)

		for _, xVal := range xs {
			xCopyVal1 = xVal
			xCopy1.Normalize()
			xCopy1.ToBytes(buf)

			for i := uint64(0); i < numReps; i++ {
				// Be wary that IsEqual may change the internal representation
				xCopyVal1 = xVal
				xCopyVal2 = xVal
				// Check that the result of RerandomizeRepresentation does not depend on the particular representation we started with
				xCopy1.RerandomizeRepresentation(i)
				xCopy2.RerandomizeRepresentation(i + 1)
				xCopy2.RerandomizeRepresentation(i)
				xCopy1.ToBytes(buf2)
				xCopy2.ToBytes(buf3)
				testutils.FatalUnless(t, utils.CompareSlices(buf2, buf3), "RerandomizeRepresentation does not factor through starting representation")
				xCopyVal2 = xVal
				// Check that RerandomizeRepresentation does not change the value
				testutils.FatalUnless(t, xCopy1.IsEqual(xCopy2), "RerandomizeRepresentation changes value")
				// Check that after, Normaliz, the representation is fixed.
				xCopy1.RerandomizeRepresentation(i)
				xCopy1.Normalize()
				xCopy1.ToBytes(buf2)
				testutils.FatalUnless(t, utils.CompareSlices(buf, buf2), "Normalize does not give unique representation")
			}
		}
	}
}

// Checks that AddInt64, AddUint64, SubInt64, SubUint64, MulUint64, MulInt64, DivideInt64, DivideUint64 work as expected:
func testFEProperty_SmallOps[FE any, FEPtr interface {
	*FE
	FieldElementInterface[FEPtr]
}](seedX int64, numX int) func(t *testing.T) {
	return func(t *testing.T) {
		prepareTestFieldElements(t)

		var xCopyVal FE
		var target1Val, target2Val FE
		var yCopyVal FE
		xCopy := FEPtr(&xCopyVal)

		yFE := FEPtr(&yCopyVal)
		target1 := FEPtr(&target1Val)
		target2 := FEPtr(&target2Val)

		// const num = 1000
		var xs []FE = GetPrecomputedFieldElements[FE, FEPtr](seedX, numX)

		var ysUint64 = []uint64{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 15, 16, 17, 255, 256, 257, 12345, 1<<16 - 1, 1 << 16, 1<<16 + 1, 1 << 31, 1<<63 - 1, 1 << 63, 1<<63 + 1, math.MaxUint64}
		var ysInt64 = []int64{0, 1, -1, 2, -2, 3, -3, 4, -4, 5, -5, 6, -6, 7, -7, 8, -8, 9, -9, 10, -10,
			15, -15, 16, -16, 17, -17, 255, 256, 257, -255, -256, -257,
			12345, -12345,
			1<<16 - 1, 1 << 16, 1<<16 + 1, -(1<<16 - 1), -(1 << 16), -(1 << 16) + 1,
			1<<15 - 1, 1 << 15, 1<<15 + 1, -(1<<15 - 1), -(1 << 15), -(1 << 15) + 1,
			1<<31 - 1, 1 << 31, 1<<31 + 1, -(1<<31 - 1), -(1 << 31), -(1 << 31) + 1,
			1<<32 - 1, 1 << 32, 1<<32 + 1, -(1<<32 - 1), -(1 << 32), -(1 << 32) + 1,
			math.MaxInt64, -(1 << 63), -(1 << 63) + 1}

		for _, xVal := range xs {

			// Uint64 versions:
			for _, y := range ysUint64 {
				yFE.SetUint64(y)

				xCopyVal = xVal
				target1.AddUint64(xCopy, y)
				target2.Add(xCopy, yFE)
				testutils.FatalUnless(t, target1.IsEqual(target2), "AddUint64 does not work as expected")

				xCopyVal = xVal
				target1.SubUint64(xCopy, y)
				target2.Sub(xCopy, yFE)
				testutils.FatalUnless(t, target1.IsEqual(target2), "SubUint64 does not work as expected")

				xCopyVal = xVal
				target1.MulUint64(xCopy, y)
				target2.Mul(xCopy, yFE)
				testutils.FatalUnless(t, target1.IsEqual(target2), "MulUint64 does not work as expected")

				xCopyVal = xVal
				target1.SetUint64(101)
				target2.SetUint64(101)
				panic1 := testutils.CheckPanic(func() { target1.DivideUint64(xCopy, y) })
				panic2 := testutils.CheckPanic(func() { target2.Divide(xCopy, yFE) })
				if y != 0 {
					testutils.FatalUnless(t, (!panic1) && (!panic2), "Unexpected Panic for DivideUint64")
					testutils.FatalUnless(t, target1.IsEqual(target2), "DivideUint64 does not work as expected")
				} else {
					testutils.FatalUnless(t, panic1 && panic2, "DivideUint64 by 0 did not panic")
					testutils.FatalUnless(t, target1.IsEqual(target2), "DivideUint64 by 0 changed argument")
					val, err := target1.ToUint64()
					testutils.FatalUnless(t, (val == 101) && (err == nil), "DivideUint64 by 0 changed argument")
				}

			}
			// Int64 versions:
			for _, y := range ysInt64 {
				yFE.SetInt64(y)

				xCopyVal = xVal
				target1.AddInt64(xCopy, y)
				target2.Add(xCopy, yFE)
				testutils.FatalUnless(t, target1.IsEqual(target2), "AddInt64 does not work as expected")

				xCopyVal = xVal
				target1.SubInt64(xCopy, y)
				target2.Sub(xCopy, yFE)
				testutils.FatalUnless(t, target1.IsEqual(target2), "SubInt64 does not work as expected")

				xCopyVal = xVal
				target1.MulInt64(xCopy, y)
				target2.Mul(xCopy, yFE)
				testutils.FatalUnless(t, target1.IsEqual(target2), "MulInt64 does not work as expected")

				xCopyVal = xVal
				target1.SetUint64(101)
				target2.SetUint64(101)
				panic1 := testutils.CheckPanic(func() { target1.DivideInt64(xCopy, y) })
				panic2 := testutils.CheckPanic(func() { target2.Divide(xCopy, yFE) })
				if y != 0 {
					testutils.FatalUnless(t, (!panic1) && (!panic2), "Unexpected Panic for DivideInt64")
					testutils.FatalUnless(t, target1.IsEqual(target2), "DivideInt64 does not work as expected")
				} else {
					testutils.FatalUnless(t, panic1 && panic2, "DivideInt64 by 0 did not panic")
					testutils.FatalUnless(t, target1.IsEqual(target2), "DivideInt64 by 0 changed argument")
					val, err := target1.ToUint64()
					testutils.FatalUnless(t, (val == 101) && (err == nil), "DivideInt64 by 0 changed argument")
				}

			}
		}
	}
}

// Checking properties of Sign. Notably, these are
//
// x.Sign() is -1,+1,0 - valued
// x.Sign() == 0 iff x == 0
// if x == -y, x.Sign() == -y.Sign()
//
// Additionally, we ask that x.Sign() matches the sign of the smallest-absolute-value representation (canonical sign).
// Note that this latter property is kind-of-optional in that we don't really need that property.
// In fact, returning the sign of the montgomery representation would be more efficient if we use montgomery representation; however, this gives weird semantics to Sign.

func testFEProperty_Sign[FE any, FEPtr interface {
	*FE
	FieldElementInterface[FEPtr]
}](seedX int64, numX int, numReps uint64) func(t *testing.T) {
	return func(t *testing.T) {
		prepareTestFieldElements(t)

		var zVal FE
		z := FEPtr(&zVal)

		// Check Signs for 0, 1/2, -1/2. These are special values for Sign (1/2 = 1 + (-1/2) and this is where Sign flips )
		// Notably we expect Sign(0) == 0, Sign(1/2) == -1, Sign(-1/2) == +1

		// (The latter two COULD differ if we relaxed the condition on Sign)

		z.SetZero()
		for i := uint64(0); i < numReps; i++ {
			z.RerandomizeRepresentation(i)
			testutils.FatalUnless(t, z.Sign() == 0, "Sign(0) != 0")
		}

		z.SetUint256(&oneHalfModBaseField_uint256)
		for i := uint64(0); i < numReps; i++ {
			z.RerandomizeRepresentation(i)
			testutils.FatalUnless(t, z.Sign() == -1, "Sign(1/2) != -1")
		}

		z.SetUint256(&minusOneHalfModBaseField_uint256)
		for i := uint64(0); i < numReps; i++ {
			z.RerandomizeRepresentation(i)
			testutils.FatalUnless(t, z.Sign() == +1, "Sign(-11/2) != 1")
		}

		var x2Val FE
		x2 := FEPtr(&x2Val)

		var xs []FE = GetPrecomputedFieldElements[FE, FEPtr](seedX, numX)
		for _, xVal := range xs {
			x := FEPtr(&xVal)
			var xUint256 Uint256
			x.ToUint256(&xUint256)
			s1 := x.Sign()
			x2.Neg(x)
			s2 := x2.Sign()

			testutils.FatalUnless(t, s1 != s2, "Sign(-x) != -Sign(x)")
			testutils.FatalUnless(t, s1 == 1 || s1 == 0 || s1 == -1, "Sign is not in -1,0,+1")
			testutils.FatalUnless(t, (s1 == 0) == x.IsZero(), "Sign == 0 differs from IsZero")

			// Check that Sign matches the sign of the representation in [-BaseFieldSize/2, +BaseFieldSize/2] (where BaseFieldSize/2 is half-integer, hence bound is implicitly excluded)
			// Note that we could remove this check if we relaxed the condition on Sign.

			var s3 int
			if xUint256.IsZero() {
				s3 = 0
			} else if xUint256.Cmp(&oneHalfModBaseField_uint256) < 0 {
				s3 = +1
			} else {
				s3 = -1
			}

			if s3 != s1 {
				t.Errorf("Sign of field element does not match sign of (representation of smallest absolute value)")
			}
		}
	}
}

// test CmpAbs

func testFEProperty_CmpAbs[FE any, FEPtr interface {
	*FE
	FieldElementInterface[FEPtr]
}](seedX int64, numX int) func(t *testing.T) {
	return func(t *testing.T) {
		prepareTestFieldElements(t)

		var fes []FE = GetPrecomputedFieldElements[FE, FEPtr](seedX, numX)
		for i, xVal := range fes {
			x := FEPtr(&xVal)
			for j, yVal := range fes {
				y := FEPtr(&yVal)
				equal := x.IsEqual(y)

				if i == j {
					testutils.FatalUnless(t, equal, "IsEqual does not behave as expected")
				} else if equal {
					t.Log(t, "Random field elements were equal. This is neglibly unlike to happend by accident. Unless you have been tampering with random element generation, this is almost surely a bug.")
				}

				var yNegVal FE
				yNeg := FEPtr(&yNegVal)
				yNeg.Neg(y)

				antiEqual := x.IsEqual(yNeg)

				res1, res2 := x.CmpAbs(y)
				testutils.FatalUnless(t, res1 == equal || antiEqual, "First return value of CmpAbs wrong")
				testutils.FatalUnless(t, res2 == equal, "Second return value of CmpAbs wrong")

				res1, res2 = x.CmpAbs(yNeg)

				testutils.FatalUnless(t, res1 == equal || antiEqual, "First return value of CmpAbs wrong")
				testutils.FatalUnless(t, res2 == antiEqual, "Second return value of CmpAbs wrong")
			}
		}
	}
}

func testFEProperty_FormattedOutput[FE any, FEPtr interface {
	*FE
	FieldElementInterface[FEPtr]
}](seed int64, num int) func(t *testing.T) {
	return func(t *testing.T) {
		prepareTestFieldElements(t)
		var xs []FE = GetPrecomputedFieldElements[FE, FEPtr](seed, num)

		for _, xVal := range xs {
			x := FEPtr(&xVal)
			xInt := x.ToBigInt()

			for _, formatString := range []string{"%x", "%X", "%b", "%o", "%O", "%d"} {
				bigIntString := fmt.Sprintf(formatString, xInt)
				xString1 := fmt.Sprintf(formatString, xVal)
				xString2 := fmt.Sprintf(formatString, x)
				testutils.FatalUnless(t, xString1 == xString2, "Format should be defined on value receiver")
				testutils.FatalUnless(t, xString1 == bigIntString, "Formatted output differs between field element and big.Int for format %v", formatString)
			}

		}

	}
}
