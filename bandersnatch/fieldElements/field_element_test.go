package fieldElements

import (
	"math"
	"testing"

	"github.com/GottfriedHerold/Bandersnatch/internal/testutils"
	"github.com/GottfriedHerold/Bandersnatch/internal/utils"
)

var _ FieldElementInterface_common = &bsFieldElement_MontgomeryNonUnique{}
var _ FieldElementInterface[*bsFieldElement_MontgomeryNonUnique] = &bsFieldElement_MontgomeryNonUnique{}

// var fatalUnless := testutils.fatalUnless

func TestFieldElementProperties(t *testing.T) {
	t.Run("Montgomery implementation", testProperties[bsFieldElement_MontgomeryNonUnique])
}

func testProperties[FE any, FEPtr interface {
	*FE
	FieldElementInterface[FEPtr]
}](t *testing.T) {
	t.Run("Constants", testFEProperty_Constants[FE, FEPtr])
	t.Run("Commutativity and inversion", testFEProperty_CommutativiteAndInverses[FE, FEPtr])
	t.Run("Aliasing and Eq", testFEProperty_Aliasing[FE, FEPtr])
	t.Run("Associativity", testFEProperty_Associativity[FE, FEPtr])
	t.Run("Distributivity", testFEProperty_Distributivity[FE, FEPtr])
	t.Run("MulByFive", testFEProperty_MulFive[FE, FEPtr])
	t.Run("raw serialization", testFEProperty_BytesRoundtrip[FE, FEPtr])
	t.Run("internal representation", testFEProperty_InternalRep[FE, FEPtr])
	t.Run("Small-Arg Operations", testFEProperty_SmallOps[FE, FEPtr])
}

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

// This tests that Add, Sub, Mul, Divide work properly if receiver and/or some arguments alias (all combination)
// Also tests this for AddEq, SubEq, MulEq, DivideEq and compares against Add,Sub,Mul, Divide
// Checks results against Double, Zero, Square, 1 and DoubleEq,SquareEq

func testFEProperty_Aliasing[FE any, FEPtr interface {
	*FE
	FieldElementInterface[FEPtr]
}](t *testing.T) {
	prepareTestFieldElements(t)

	safeTarget := FEPtr(new(FE))

	var target1Val, target2Val, target3Val, target4Val FE
	target1 := FEPtr(&target1Val)
	target2 := FEPtr(&target2Val)
	target3 := FEPtr(&target3Val)
	target4 := FEPtr(&target4Val)

	// Check that x.Fun(x) and t.Fun(x) and x.FunEq() match for Fun that doesn't read from the receiver.
	const num = 100

	var xs []FE = GetPrecomputedFieldElements[FE, FEPtr](100, num)
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
		safeTarget.MulFive(xCopy1)
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
	xs = GetPrecomputedFieldElements[FE, FEPtr](100, num)
	ys := GetPrecomputedFieldElements[FE, FEPtr](100, num)
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

func testFEProperty_CommutativiteAndInverses[FE any, FEPtr interface {
	*FE
	FieldElementInterface[FEPtr]
}](t *testing.T) {
	prepareTestFieldElements(t)
	const num = 1000
	var xs []FE = GetPrecomputedFieldElements[FE, FEPtr](100, num)
	var ys []FE = GetPrecomputedFieldElements[FE, FEPtr](100, num)

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

func testFEProperty_Associativity[FE any, FEPtr interface {
	*FE
	FieldElementInterface[FEPtr]
}](t *testing.T) {
	prepareTestFieldElements(t)
	const num = 100
	var xs []FE = GetPrecomputedFieldElements[FE, FEPtr](100, num)
	var ys []FE = GetPrecomputedFieldElements[FE, FEPtr](100, num)
	var zs []FE = GetPrecomputedFieldElements[FE, FEPtr](100, num)

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

func testFEProperty_Distributivity[FE any, FEPtr interface {
	*FE
	FieldElementInterface[FEPtr]
}](t *testing.T) {
	prepareTestFieldElements(t)
	const num = 100
	var xs []FE = GetPrecomputedFieldElements[FE, FEPtr](100, num)
	var ys []FE = GetPrecomputedFieldElements[FE, FEPtr](100, num)
	var zs []FE = GetPrecomputedFieldElements[FE, FEPtr](100, num)

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

func testFEProperty_MulFive[FE any, FEPtr interface {
	*FE
	FieldElementInterface[FEPtr]
}](t *testing.T) {
	prepareTestFieldElements(t)
	var target1Val, target2Val, target3Val, target4Val FE
	target1 := FEPtr(&target1Val)
	target2 := FEPtr(&target2Val)
	target3 := FEPtr(&target3Val)
	target4 := FEPtr(&target4Val)

	const num = 1000
	var xs []FE = GetPrecomputedFieldElements[FE, FEPtr](100, num)

	for _, xVal := range xs {
		x := FEPtr(&xVal)

		target1Val = xVal
		target1.MulEqFive()
		target2.SetUint64(5)
		target2.MulEq(x)
		target3.MulInt64(x, 5)
		target4.MulUint64(x, 5)
		testutils.FatalUnless(t, target1.IsEqual(target2), "MulByfive does not match multiplication by 5")
		testutils.FatalUnless(t, target1.IsEqual(target3), "MulByfive does not match multiplication by 5")
		testutils.FatalUnless(t, target1.IsEqual(target4), "MulByfive does not match multiplication by 5")
	}
}

func testFEProperty_BytesRoundtrip[FE any, FEPtr interface {
	*FE
	FieldElementInterface[FEPtr]
}](t *testing.T) {
	prepareTestFieldElements(t)

	LEN := FEPtr(nil).BytesLength()
	var buf []byte = make([]byte, LEN+1)
	var buf2 []byte = make([]byte, LEN+1)

	var xCopyVal FE
	xCopy := FEPtr(&xCopyVal)
	var target1Val, target2Val FE
	target1 := FEPtr(&target1Val)
	target2 := FEPtr(&target2Val)

	const num = 1000
	var xs []FE = GetPrecomputedFieldElements[FE, FEPtr](100, num)

	for _, xVal := range xs {
		x := FEPtr(&xVal)
		xCopyVal = xVal

		xCopy.ToBytes(buf)
		target1.SetBytes(buf)
		// bufs are 1 longer than needed. We check that this is actually ignored
		irrelevant := buf[LEN]
		buf[LEN] += 1
		target2.SetBytes(buf)
		testutils.Assert(copy(buf2[0:LEN], buf[0:LEN]) == 32)
		xCopy.ToBytes(buf)
		testutils.FatalUnless(t, target1.IsEqual(x), "Roundtrip failure for raw serialization")
		testutils.FatalUnless(t, target2.IsEqual(x), "Roundtrip failure for raw serialization")
		testutils.FatalUnless(t, buf[LEN] == irrelevant+1, "ToBytes writes more bytes")
		target1.ToBytes(buf2)
		testutils.FatalUnless(t, utils.CompareSlices(buf[0:32], buf2[0:32]), "Rountrip failure for raw serialization")
	}
}

// Checks that the internal representation works as expected:
func testFEProperty_InternalRep[FE any, FEPtr interface {
	*FE
	FieldElementInterface[FEPtr]
}](t *testing.T) {
	prepareTestFieldElements(t)

	LEN := FEPtr(nil).BytesLength()
	var buf []byte = make([]byte, LEN)
	var buf2 []byte = make([]byte, LEN)
	var buf3 []byte = make([]byte, LEN)

	var xCopyVal1, xCopyVal2 FE
	xCopy1 := FEPtr(&xCopyVal1)
	xCopy2 := FEPtr(&xCopyVal2)

	const num = 10000
	var xs []FE = GetPrecomputedFieldElements[FE, FEPtr](100, num)

	for _, xVal := range xs {
		xCopyVal1 = xVal
		xCopy1.Normalize()
		xCopy1.ToBytes(buf)

		for i := uint64(0); i < 20; i++ {
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

// Checks that the internal representation works as expected:
func testFEProperty_SmallOps[FE any, FEPtr interface {
	*FE
	FieldElementInterface[FEPtr]
}](t *testing.T) {
	prepareTestFieldElements(t)

	var xCopyVal FE
	var target1Val, target2Val FE
	var yCopyVal FE
	xCopy := FEPtr(&xCopyVal)

	yFE := FEPtr(&yCopyVal)
	target1 := FEPtr(&target1Val)
	target2 := FEPtr(&target2Val)

	const num = 1000
	var xs []FE = GetPrecomputedFieldElements[FE, FEPtr](100, num)

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
