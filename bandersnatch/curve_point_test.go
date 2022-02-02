package bandersnatch

import (
	"math/big"
	"testing"
)

var _ CurvePointPtrInterfaceBaseRead = &point_efgh_base{}
var _ CurvePointPtrInterfaceBaseRead = &point_xtw_base{}
var _ CurvePointPtrInterfaceBaseRead = &point_axtw_base{}

var _ CurvePointPtrInterfaceRead = &Point_efgh_subgroup{}
var _ CurvePointPtrInterfaceRead = &Point_efgh_full{}
var _ CurvePointPtrInterfaceWrite = &Point_efgh_subgroup{}
var _ CurvePointPtrInterfaceWrite = &Point_efgh_full{}

var _ CurvePointPtrInterfaceRead = &Point_axtw_subgroup{}
var _ CurvePointPtrInterfaceRead = &Point_axtw_full{}
var _ CurvePointPtrInterfaceWrite = &Point_axtw_subgroup{}
var _ CurvePointPtrInterfaceWrite = &Point_axtw_full{}

var _ CurvePointPtrInterfaceRead = &Point_xtw_subgroup{}
var _ CurvePointPtrInterfaceRead = &Point_xtw_full{}
var _ CurvePointPtrInterfaceWrite = &Point_xtw_subgroup{}
var _ CurvePointPtrInterfaceWrite = &Point_xtw_full{}

var _ CurvePointPtrInterfaceCooReadExtended = &Point_xtw_full{}
var _ CurvePointPtrInterfaceCooReadExtended = &Point_xtw_subgroup{}
var _ CurvePointPtrInterfaceCooReadExtended = &Point_efgh_full{}
var _ CurvePointPtrInterfaceCooReadExtended = &Point_efgh_subgroup{}
var _ CurvePointPtrInterfaceCooReadExtended = &Point_axtw_full{}
var _ CurvePointPtrInterfaceCooReadExtended = &Point_axtw_subgroup{}

var _ CurvePointPtrInterfaceDistinguishInfinity = &Point_efgh_full{}
var _ CurvePointPtrInterfaceDistinguishInfinity = &Point_xtw_full{}

var _ torsionAdder = &point_xtw_base{}
var _ torsionAdder = &point_axtw_base{}
var _ torsionAdder = &point_efgh_base{}

/*
	This file contains tests on curve points that can be expressed as properties on the exported interface of CurvePointPtrInterface.
	Using our testing framework and a little bit of reflection (hidden in helper functions) and interfaces, these tests are then run on all concrete curve point types.
*/

// Tests properties of some global parameters
func TestGlobalParameter(t *testing.T) {
	if big.Jacobi(big.NewInt(CurveParameterA), BaseFieldSize) == 1 {
		t.Fatal("Parameter a of curve is a square")
	}
	if big.Jacobi(CurveParameterD_Int, BaseFieldSize) == 1 {
		t.Fatal("Parameter d of curve is a square")
	}
	var temp FieldElement
	temp.Square(&squareRootDbyA_fe)
	temp.multiply_by_five()
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
