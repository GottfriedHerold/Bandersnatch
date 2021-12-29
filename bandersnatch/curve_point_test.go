package bandersnatch

import (
	"math/big"
	"testing"
)

/*
	This file contains tests on curve points that can be expressed properties on the exported interface of CurvePointPtrInterface.
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
}

// Ensures that types satisfy the intended interfaces.
// Note that the package will not compile anyway if these are not satisfied.
func TestInterfaces(t *testing.T) {
	var _ CurvePointPtrInterfaceRead = &Point_xtw{}
	var _ CurvePointPtrInterfaceWrite = &Point_xtw{}

	var _ CurvePointPtrInterfaceRead = &Point_axtw{}
	var _ CurvePointPtrInterfaceWrite = &Point_axtw{}

	var _ CurvePointPtrInterfaceRead = &Point_efgh{}
	var _ CurvePointPtrInterfaceWrite = &Point_efgh{}

	var _ CurvePointPtrInterface_FullCurve = &Point_xtw{}
	var _ CurvePointPtrInterface_FullCurve = &Point_axtw{}
	var _ CurvePointPtrInterface_FullCurve = &Point_efgh{}
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

var allTestPointTypes = []PointType{pointTypeXTW, pointTypeAXTW, pointTypeEFGH}
