package curvePoints

import (
	"math/big"
	"testing"
)

// Tests properties of some global parameters of the bandersnatch curve.
func TestGlobalCurveParameter(t *testing.T) {
	if big.Jacobi(big.NewInt(CurveParameterA), BaseFieldSize_Int) == 1 {
		t.Fatal("Parameter a of curve is a square")
	}
	if big.Jacobi(CurveParameterD_Int, BaseFieldSize_Int) == 1 {
		t.Fatal("Parameter d of curve is a square")
	}
	var temp FieldElement
	temp.Square(&squareRootDbyA_fe)
	temp.Multiply_by_five()
	temp.Neg(&temp)
	if !temp.IsEqual(&CurveParameterD_fe) {
		t.Fatal("SqrtDDivA is not a square root of d/a")
	}
	if (Cofactor*GroupOrder - CurveOrder) != 0 {
		t.Fatal("Relationship between constants violated")
	}
	var tempInt *big.Int = big.NewInt(0)
	var twoInt *big.Int = big.NewInt(2)
	tempInt.Mul(EndomorphismEigenvalue_Int, EndomorphismEigenvalue_Int)
	tempInt.Add(tempInt, twoInt)
	tempInt.Mod(tempInt, GroupOrder_Int)
	if tempInt.Sign() != 0 {
		t.Fatal("EndomorphismEigentvalue_Int is not a square root of -2 modulo p253")
	}
}
