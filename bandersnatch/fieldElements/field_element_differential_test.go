package fieldElements

import (
	"testing"

	"github.com/GottfriedHerold/Bandersnatch/internal/testutils"
)

// runs differential tests for field element implementations.
func TestDifferentialFieldElements(t *testing.T) {
	testFEDifferential[bsFieldElement_MontgomeryNonUnique, bsFieldElement_BigInt](10001, 10001, 10001, 1000, 100, 100)(t)
}

// utility stuff to iterate over methods (and their names) in a type-safe way (i.e. without using reflection).

type _functionAndName[funType any] struct {
	f    funType
	name string
}

func _getUnaryFuns[FE FieldElementInterface[FE]]() []_functionAndName[func(FE, FE)] {
	return []_functionAndName[func(FE, FE)]{{FE.Neg, "Neg"}, {FE.Double, "Double"}, {FE.MulFive, "MulFive"}, {FE.Square, "Square"}, {FE.Inv, "Inv"}}
}

func _getBinaryFuns[FE FieldElementInterface[FE]]() []_functionAndName[func(FE, FE, FE)] {
	return []_functionAndName[func(FE, FE, FE)]{{FE.Add, "Add"}, {FE.Sub, "Sub"}, {FE.Mul, "Mul"}, {FE.Divide, "Divide"}}
}

func _getBinaryInt64Funs[FE FieldElementInterface[FE]]() []_functionAndName[func(FE, FE, int64)] {
	return []_functionAndName[func(FE, FE, int64)]{{FE.AddInt64, "AddInt64"}, {FE.SubInt64, "SubInt64"}, {FE.MulInt64, "MulInt64"}, {FE.DivideInt64, "DivideInt64"}}
}

func _getBinaryUint64Funs[FE FieldElementInterface[FE]]() []_functionAndName[func(FE, FE, uint64)] {
	return []_functionAndName[func(FE, FE, uint64)]{{FE.AddUint64, "AddUint64"}, {FE.SubUint64, "SubUint64"}, {FE.MulUint64, "MulUint64"}, {FE.DivideUint64, "DividveUint64"}}
}

func _getNullaryEqFuns[FE FieldElementInterface[FE]]() []_functionAndName[func(FE)] {
	return []_functionAndName[func(FE)]{{FE.NegEq, "NegEq"}, {FE.DoubleEq, "DoubleEq"}, {FE.MulEqFive, "MulEqFive"}, {FE.SquareEq, "SquareEq"}, {FE.InvEq, "InvEq"}}
}

func _getUnaryEqFuns[FE FieldElementInterface[FE]]() []_functionAndName[func(FE, FE)] {
	return []_functionAndName[func(FE, FE)]{{FE.AddEq, "AddEq"}, {FE.SubEq, "SubEq"}, {FE.MulEq, "MulEq"}, {FE.DivideEq, "DivideEq"}}
}

func testFEDifferential[FE1 any, FE2 any, FEPtr1 interface {
	*FE1
	FieldElementInterface[FEPtr1]
}, FEPtr2 interface {
	*FE2
	FieldElementInterface[FEPtr2]
}](seedsUnary int64, seedsX int64, seedsY int64, numUnary int, numX int, numY int) func(t *testing.T) {
	return func(t *testing.T) {
		prepareTestFieldElements(t)

		// define variables used later
		var target1Val, fe1Val, yCopy1Val FE1
		var target2Val, fe2Val, yCopy2Val FE2
		var res1, res2 Uint256
		target1 := FEPtr1(&target1Val)
		fe1 := FEPtr1(&fe1Val)
		fe2 := FEPtr2(&fe2Val)
		yCopy1 := FEPtr1(&yCopy1Val)
		yCopy2 := FEPtr2(&yCopy2Val)
		target2 := FEPtr2(&target2Val)

		// get points -- NOTE: x1[i] and x2[i] represent the same points for the same seed (but with possibly different type) due to how GetPrecomputedFieldElements works.
		var x1s []FE1 = GetPrecomputedFieldElements[FE1, FEPtr1](seedsUnary, numUnary)
		var x2s []FE2 = GetPrecomputedFieldElements[FE2, FEPtr2](seedsUnary, numUnary)

		for i, x1Val := range x1s {
			x2Val := x2s[i]

			// operate on copies, because ops might change the point, thereby affecting later tests
			fe1Val = x1Val
			fe2Val = x2Val

			// Check that the input points are equal
			fe1.ToUint256(&res1)
			fe2.ToUint256(&res2)
			testutils.Assert(res1 == res2)

			// ToBigInt
			fe1Val = x1Val
			fe2Val = x2Val
			resInt1 := fe1.ToBigInt()
			resInt2 := fe2.ToBigInt()
			testutils.FatalUnless(t, resInt1.Cmp(resInt2) == 0, "Diffential test failed for ToBigInt")

			// Sign
			fe1Val = x1Val
			fe2Val = x2Val
			testutils.FatalUnless(t, fe1.Sign() == fe2.Sign(), "Differential test failed for Sign")

			// Jacobi
			fe1Val = x1Val
			fe2Val = x2Val
			testutils.FatalUnless(t, fe1.Jacobi() == fe2.Jacobi(), "Differential test failed for Jacobi")

			// Neg,Inv,Double,Square,MulFive
			unaryFuns1 := _getUnaryFuns[FEPtr1]()
			unaryFuns2 := _getUnaryFuns[FEPtr2]()

			for i, funAndName1 := range unaryFuns1 {
				fun1 := funAndName1.f
				fun2 := unaryFuns2[i].f
				fe1Val = x1Val
				fe2Val = x2Val
				didPanic1 := testutils.CheckPanic(fun1, target1, fe1)
				didPanic2 := testutils.CheckPanic(fun2, target2, fe2)
				testutils.FatalUnless(t, didPanic1 == didPanic2, "Differential Test had different panic behaviour for %v", funAndName1.name)
				testutils.FatalUnless(t, didPanic1 || IsEqualAsUint256(target1, target2), "Differential Test failed for %v", funAndName1.name)
			}

			// NegEq, InvEq, DoubleEq, MulEqFive, SquareEq
			nullaryEqFuns1 := _getNullaryEqFuns[FEPtr1]()
			nullaryEqFuns2 := _getNullaryEqFuns[FEPtr2]()

			for i, funAndName1 := range nullaryEqFuns1 {
				fun1 := funAndName1.f
				fun2 := nullaryEqFuns2[i].f
				target1Val = x1Val
				target2Val = x2Val
				didPanic1 := testutils.CheckPanic(fun1, target1)
				didPanic2 := testutils.CheckPanic(fun2, target2)
				testutils.FatalUnless(t, didPanic1 == didPanic2, "Differential Test had different panic behaviour for %v", funAndName1.name)
				testutils.FatalUnless(t, didPanic1 || IsEqualAsUint256(target1, target2), "Differential Test failed for %v", funAndName1.name)
			}

			// AddInt64, SubInt64, MulInt64, DivideInt64
			smallOpInt64Funs1 := _getBinaryInt64Funs[FEPtr1]()
			smallOpInt64Funs2 := _getBinaryInt64Funs[FEPtr2]()
			for i, funAndName1 := range smallOpInt64Funs1 {
				fun1 := funAndName1.f
				fun2 := smallOpInt64Funs2[i].f
				fe1Val = x1Val
				fe2Val = x2Val
				for _, smallarg := range []int64{0, 1, -1, -2, +2, -(1 << 63), (1 << 63) - 1, 255, 256, 257, -255, -256, -257, 1 << 16, -(1 << 16)} {
					didPanic1 := testutils.CheckPanic(fun1, target1, fe1, smallarg)
					didPanic2 := testutils.CheckPanic(fun2, target2, fe2, smallarg)
					testutils.FatalUnless(t, didPanic1 == didPanic2, "Differential Test had different panic behaviour for %v", funAndName1.name)
					testutils.FatalUnless(t, didPanic1 || IsEqualAsUint256(target1, target2), "Differential Test failed for %v", funAndName1.name)
				}
			}

			// AddUint64, SubUint64, MulUint64, DivideUint64
			smallOpUint64Funs1 := _getBinaryUint64Funs[FEPtr1]()
			smallOpUint64Funs2 := _getBinaryUint64Funs[FEPtr2]()
			for i, funAndName1 := range smallOpUint64Funs1 {
				fun1 := funAndName1.f
				fun2 := smallOpUint64Funs2[i].f
				fe1Val = x1Val
				fe2Val = x2Val
				for _, smallarg := range []uint64{0, 1, 2, 1 << 63, 1<<64 - 1, 255, 256, 257, 1 << 16, 1 << 32, 5, 7, 15} {
					didPanic1 := testutils.CheckPanic(fun1, target1, fe1, smallarg)
					didPanic2 := testutils.CheckPanic(fun2, target2, fe2, smallarg)
					testutils.FatalUnless(t, didPanic1 == didPanic2, "Differential Test had different panic behaviour for %v", funAndName1.name)
					testutils.FatalUnless(t, didPanic1 || IsEqualAsUint256(target1, target2), "Differential Test failed for %v", funAndName1.name)
				}
			}

		}

		x1s = GetPrecomputedFieldElements[FE1, FEPtr1](seedsX, numX)
		x2s = GetPrecomputedFieldElements[FE2, FEPtr2](seedsX, numX)
		y1s := GetPrecomputedFieldElements[FE1, FEPtr1](seedsY, numY)
		y2s := GetPrecomputedFieldElements[FE2, FEPtr2](seedsY, numY)

		for i, x1Val := range x1s {
			x2Val := x2s[i]
			for j, y1Val := range y1s {
				y2Val := y2s[j]

				// Add, Sub, Mul, Divide
				binaryFuns1 := _getBinaryFuns[FEPtr1]()
				binaryFuns2 := _getBinaryFuns[FEPtr2]()
				for i, funAndName1 := range binaryFuns1 {
					fun1 := funAndName1.f
					fun2 := binaryFuns2[i].f
					fe1Val = x1Val
					fe2Val = x2Val
					yCopy1Val = y1Val
					yCopy2Val = y2Val
					didPanic1 := testutils.CheckPanic(fun1, target1, fe1, yCopy1)
					didPanic2 := testutils.CheckPanic(fun2, target2, fe2, yCopy2)
					testutils.FatalUnless(t, didPanic1 == didPanic2, "Differential Test had different panic behaviour for %v", funAndName1.name)
					testutils.FatalUnless(t, didPanic1 || IsEqualAsUint256(target1, target2), "Differential Test failed for %v", funAndName1.name)
				}

				// AddEq, SubEq, MulEq, DivideEq
				binaryEqFuns1 := _getUnaryEqFuns[FEPtr1]()
				binaryEqFuns2 := _getUnaryEqFuns[FEPtr2]()
				for i, funAndName1 := range binaryEqFuns1 {
					fun1 := funAndName1.f
					fun2 := binaryEqFuns2[i].f
					target1Val = x1Val
					target2Val = x2Val
					yCopy1Val = y1Val
					yCopy2Val = y2Val
					didPanic1 := testutils.CheckPanic(fun1, target1, yCopy1)
					didPanic2 := testutils.CheckPanic(fun2, target2, yCopy2)
					testutils.FatalUnless(t, didPanic1 == didPanic2, "Differential Test had different panic behaviour for %v", funAndName1.name)
					testutils.FatalUnless(t, didPanic1 || IsEqualAsUint256(target1, target2), "Differential Test failed for %v", funAndName1.name)
				}

			}
		}
	}
}
