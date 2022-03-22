package bandersnatch

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

func BenchmarkCurvePointSliceAccess(bOuter *testing.B) {
	const size = 256
	prepareBenchmarkCurvePoints(bOuter)
	A1_slice := getPrecomputedCurvePointSlice(2, pointTypeAXTWSubgroup, size)
	A2 := getPrecomputedCurvePointSlice(3, pointTypeAXTWSubgroup, size)
	_ = A2
	var A1 [size]Point_axtw_subgroup
	for i := 0; i < size; i++ {
		A1[i].SetFrom(A1_slice[i])
	}
	resetBenchmarkCurvePoints(bOuter)
	fun_direct := func(bInner *testing.B) {
		prepareBenchmarkCurvePoints(bInner)
		for n := 0; n < bInner.N; n++ {
			DumpAXTW_subgroup[n%dumpSizeBench_curve] = A1[n%size]
		}
	}
	fun_SetFrom := func(bInner *testing.B) {
		prepareBenchmarkCurvePoints(bInner)
		for n := 0; n < bInner.N; n++ {
			DumpAXTW_subgroup[n%dumpSizeBench_curve].SetFrom(&A1[n%size])
		}
	}
	fun_genericArray := func(bInner *testing.B) {
		prepareBenchmarkCurvePoints(bInner)
		for n := 0; n < bInner.N; n++ {
			DumpAXTW_subgroup[n%dumpSizeBench_curve].SetFrom(getElementFromCurvePointSlice(&A1, n%size))
		}
	}
	fun_genericSlice := func(bInner *testing.B) {
		prepareBenchmarkCurvePoints(bInner)
		for n := 0; n < bInner.N; n++ {
			DumpAXTW_subgroup[n%dumpSizeBench_curve].SetFrom(getElementFromCurvePointSlice(A1[:], n%size))
		}
	}
	fun_pointSliceReader := func(bInner *testing.B) {
		prepareBenchmarkCurvePoints(bInner)
		for n := 0; n < bInner.N; n++ {
			DumpAXTW_subgroup[n%dumpSizeBench_curve].SetFrom(CurvePointSlice_axtw_subgroup(A1[:]).GetByIndex(n % size))
		}
	}
	bOuter.Run("direct access", fun_direct)
	bOuter.Run("SetFrom", fun_SetFrom)
	bOuter.Run("SetFrom getElementFromCurvePointSlice(&arr)", fun_genericArray)
	bOuter.Run("SetFrom getElementFromCurvePointSlice(arr[:])", fun_genericSlice)
	bOuter.Run("SetFrom via PointSliceReader interface", fun_pointSliceReader)
}

func TestCurvePointSlices(t *testing.T) {
	var A1 [2]Point_axtw_subgroup
	var A2 []Point_axtw_subgroup = A1[:]
	var A3 [2]*Point_axtw_subgroup
	for i := 0; i < len(A3); i++ {
		A3[i] = new(Point_axtw_subgroup)
	}
	var A4 []*Point_axtw_subgroup = A3[:]

	var A5 [2]CurvePointPtrInterface
	for i := 0; i < len(A5); i++ {
		A5[i] = makeCurvePointPtrInterface(pointTypeAXTWSubgroup)
	}
	var A6 []CurvePointPtrInterface = A5[:]

	P := getElementFromCurvePointSlice(A2, 1)
	if !(P == &A2[1]) {
		t.Fatal("getElementFromCurvePointSlice does not work for []concrete type")
	}

	P = getElementFromCurvePointSlice(&A2, 1)
	if !(P == &A2[1]) {
		t.Fatal("getElementFromCurvePointSlice does not work for *[]concrete type")
	}

	P = getElementFromCurvePointSlice(A4, 1)
	if !(P == A4[1]) {
		t.Fatal("getElementFromCurvePointSlice does not work for []*concrete type")
	}

	P = getElementFromCurvePointSlice(&A4, 1)
	if !(P == A4[1]) {
		t.Fatal("getElementFromCurvePointSlice does not work for *[]*concrete type")
	}

	P = getElementFromCurvePointSlice(A3, 1)
	if !(P == A3[1]) {
		t.Fatal("getElementFromCurvePointSlice does not work for [2]*concrete type")
	}

	P = getElementFromCurvePointSlice(&A3, 1)
	if !(P == A3[1]) {
		t.Fatal("getElementFromCurvePointSlice does not work for *[2]*concrete type")
	}

	P = getElementFromCurvePointSlice(&A1, 1)
	if !(P == &A1[1]) {
		t.Fatal("getElementFromCurvePointSlice does not work for *[2]concrete type")
	}

	P = getElementFromCurvePointSlice(A5, 1)
	if !(P == A5[1]) {
		t.Fatal("getElementFromCurvePointSlice does not work for [2]interface type")
	}

	P = getElementFromCurvePointSlice(&A5, 1)
	if !(P == A5[1]) {
		t.Fatal("getElementFromCurvePointSlice does not work for *[2]interface type")
	}

	P = getElementFromCurvePointSlice(A6, 1)
	if !(P == A6[1]) {
		t.Fatal("getElementFromCurvePointSlice does not work for []interface type")
	}

	P = getElementFromCurvePointSlice(&A6, 1)
	if !(P == A6[1]) {
		t.Fatal("getElementFromCurvePointSlice does not work for *[]interface type")
	}

}

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

type BulkNormalizerAffineZ interface {
	normalizeAffineZ()
	CurvePointSlice
}

type NormalizerAffineZ interface {
	normalizeAffineZ()
}

func testMultiAffineZWorks(t *testing.T, vec BulkNormalizerAffineZ) {
	L := vec.Len()
	var vecCopy []CurvePointPtrInterface = make([]CurvePointPtrInterface, L)
	for i := 0; i < L; i++ {
		vecCopy[i] = vec.GetByIndex(i).Clone()
	}

	vec.normalizeAffineZ()

	// not merged with loop above because we want to give preference to panics in the vec-version.
	for i := 0; i < L; i++ {
		vecCopy[i].(NormalizerAffineZ).normalizeAffineZ()
	}
	for i := 0; i < L; i++ {
		if !vecCopy[i].IsEqual(vec.GetByIndex(i)) {
			t.Fatal("Bulk-NormalizeAffineZ differs from NormalizeAffineZ")
		}
	}
}
