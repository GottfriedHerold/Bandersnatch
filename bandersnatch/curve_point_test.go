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

var _ Rerandomizeable = &point_efgh_base{}
var _ Rerandomizeable = &Point_efgh_subgroup{}
var _ Rerandomizeable = &Point_efgh_full{}
var _ Rerandomizeable = &point_xtw_base{}
var _ Rerandomizeable = &Point_xtw_full{}
var _ Rerandomizeable = &Point_xtw_subgroup{}
var _ Rerandomizeable = &point_axtw_base{}
var _ Rerandomizeable = &Point_axtw_subgroup{}
var _ Rerandomizeable = &Point_axtw_full{}

var _ CurvePointPtrInterfaceCooReadExtended = &Point_xtw_full{}
var _ CurvePointPtrInterfaceCooReadExtended = &Point_xtw_subgroup{}
var _ CurvePointPtrInterfaceCooReadExtended = &Point_efgh_full{}
var _ CurvePointPtrInterfaceCooReadExtended = &Point_efgh_subgroup{}

var _ CurvePointPtrInterfaceReadCanDistinguishInfinity = &point_efgh_base{}
var _ CurvePointPtrInterfaceReadCanDistinguishInfinity = &point_xtw_base{}

var _ Validateable = &point_xtw_base{}
var _ Validateable = &Point_xtw_full{}
var _ Validateable = &Point_xtw_subgroup{}
var _ Validateable = &point_axtw_base{}
var _ Validateable = &Point_axtw_full{}
var _ Validateable = &Point_axtw_subgroup{}
var _ Validateable = &point_efgh_base{}
var _ Validateable = &Point_efgh_full{}
var _ Validateable = &Point_efgh_subgroup{}

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

/*
	checkfun_<foo> are functions of type checkfun (i.e. func(TestSample)(bool, string))
	They are to be run on TestSamples containing a Tuple of CurvePointPtrInterfaces and Flags and return true, <ignored> on success
	and false, optional_error_reason on failure.

	Be aware that our checkfunction also verify the intended behaviour at NaP's (even though we might not guarantee it)

	In some cases, the checkfunction needs an extra argument.
	E.g. when testing addition z.Add(x,y), the arguments x,y are given by the TestSample, but we need to specify the type of the receiver z intended to store the argument
	(this is important, as it selects the actuall method used), so we need an extra argument of type PointType (which is based on reflect.Type).
	In order to do that, we define functions with names
	make_checkfun_<foo>(extra arguments) that return checkfunctions with the extra arguments bound.
*/
