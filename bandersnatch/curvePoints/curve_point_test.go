package curvePoints

import (
	"testing"
)

/*
	This file contains tests to ensure that intended struct (pointers) satisfy the CurvePointPtrInterface interface.
	The tests that the actual implementations satisfies the required properties (such as commutativity of addition etc.)
	are organized in the curve_point_test_*_test.go files
*/

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

// These variables are used by various generic tests, which are then run against all of these concrete types.
// We assume that allTestPointTypes contains the union of all others here.
var allTestPointTypes = []PointType{pointTypeXTWFull, pointTypeXTWSubgroup, pointTypeAXTWFull, pointTypeAXTWSubgroup, pointTypeEFGHFull, pointTypeEFGHSubgroup}
var allXTWTestPointTypes = []PointType{pointTypeXTWFull, pointTypeXTWSubgroup}
var allAXTWTestPointTypes = []PointType{pointTypeAXTWFull, pointTypeAXTWSubgroup}
var allEFGHTestPointTypes = []PointType{pointTypeEFGHFull, pointTypeEFGHSubgroup}
var allFullCurveTestPointTypes = []PointType{pointTypeXTWFull, pointTypeAXTWFull, pointTypeEFGHFull}
var allSubgroupCurveTestPointTypes = []PointType{pointTypeXTWSubgroup, pointTypeAXTWSubgroup, pointTypeEFGHSubgroup}

// We might remove this
var allBasePointTypes = []PointType{pointTypeXTWBase, pointTypeAXTWBase, pointTypeEFGHBase}

// TestAllTestPointTypesSatisfyInterface ensures that all elements from allTestPointTypes, allFullCurveTestPointTypes and allSubgroupCurveTestPointTypes satisfy
// some required properties:
// instances satisfy curvePointPtrInterfaceTestSample
// depending on the values of CanRepresentInfinity / CanOnlyRepresentSubgroup, additional requirements:
// - CurvePointPtrInterfaceDistinguishInfinity
func TestAllTestPointTypesSatisfyInterface(t *testing.T) {
	for _, pointType := range allBasePointTypes {
		// This will panic on failure
		_ = makeCurvePointPtrInterfaceBase(pointType)
	}

	for _, pointType := range allTestPointTypes {
		pointInstance, ok := makeCurvePointPtrInterface(pointType).(CurvePointPtrInterfaceTestSample)
		pointString := pointTypeToString(pointType)
		if !ok {
			t.Fatalf("Point type %v not compatible with curvePointPtrInterfaceTestSample", pointString)
		}
		// Note that pointInstance is a nil pointer (of the appropriate type).
		// So this also tests that certain functions can be called with nil receivers.

		// TODO: This might go away together with CanRepresentInfinity.
		if pointInstance.CanRepresentInfinity() {
			_, ok = pointInstance.(CurvePointPtrInterfaceDistinguishInfinity)
			if !ok {
				t.Fatalf("Curve point type %v can represent infinity, but does not provide interface to distinguish", pointString)
			}
			_, ok = pointInstance.(curvePointPtrInterfaceTestSampleE)
			if !ok {
				t.Fatalf("Curve point type %v can represent infinity, but cannot set to points at infinity", pointString)
			}
		}
		if !pointInstance.CanOnlyRepresentSubgroup() {
			_, ok = pointInstance.(curvePointPtrInterfaceTestSampleA)
			if !ok {
				t.Fatalf("Curve point type %v can represent points outside prime-order subgroup, but cannot set to affine order-2 point.", pointString)
			}

			// The property checked here is not really needed, but our testing framework assumes it for simplicity.
			_, ok = pointInstance.(torsionAdder)
			if !ok {
				t.Fatalf("Curve point type %v can represent points outside prime-order subgroup, but does not satisfy torsionAdder interface", pointString)
			}
		}
	}

	for _, pointType := range allFullCurveTestPointTypes {
		pointString := pointTypeToString(pointType)
		pointInstance, ok := makeCurvePointPtrInterface(pointType).(CurvePointPtrInterfaceTestSample)
		if !ok {
			t.Fatalf("Point type %v is not compatible with curvePointPtrInterfaceTestSampe", pointString)
		}
		if pointInstance.CanOnlyRepresentSubgroup() {
			t.Fatalf("Curve point type %v in allFullCurveTestPointTypes, but reports to only support subgroup elements.", pointString)
		}
	}

	for _, pointType := range allSubgroupCurveTestPointTypes {
		pointString := pointTypeToString(pointType)
		pointInstance, ok := makeCurvePointPtrInterface(pointType).(CurvePointPtrInterfaceTestSample)
		if !ok {
			t.Fatalf("Point type %v is not compatible with curvePointPtrInterfaceTestSampe", pointString)
		}
		if !pointInstance.CanOnlyRepresentSubgroup() {
			t.Fatalf("Curve point type %v in allSubgroupCurveTestPointTypes, but reports to NOT only support subgroup elements.", pointString)
		}
	}

}

type bulkNormalizerAffineZ interface {
	normalizeAffineZ()
	CurvePointSlice
}

type normalizerAffineZ interface {
	normalizeAffineZ()
}

func testMultiAffineZWorks(t *testing.T, vec bulkNormalizerAffineZ) {
	L := vec.Len()
	var vecCopy []CurvePointPtrInterface = make([]CurvePointPtrInterface, L)
	for i := 0; i < L; i++ {
		vecCopy[i] = vec.GetByIndex(i).Clone()
	}

	vec.normalizeAffineZ()

	// not merged with loop above because we want to give preference to panics in the vec-version.
	for i := 0; i < L; i++ {
		vecCopy[i].(normalizerAffineZ).normalizeAffineZ()
	}
	for i := 0; i < L; i++ {
		if !vecCopy[i].IsEqual(vec.GetByIndex(i)) {
			t.Fatal("Bulk-NormalizeAffineZ differs from NormalizeAffineZ")
		}
	}
}
