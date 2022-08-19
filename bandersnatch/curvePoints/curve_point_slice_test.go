package curvePoints

import (
	"testing"

	"github.com/GottfriedHerold/Bandersnatch/internal/testutils"
)

var _ CurvePointSlice = GenericPointSlice{}
var _ CurvePointSlice = curvePointSliceWrapper[Point_axtw_full, *Point_axtw_full]{}
var _ CurvePointSlice = curvePointPtrSliceWrapper[*Point_axtw_full]{}
var _ CurvePointSlice = curvePointPtrSliceWrapper[CurvePointPtrInterface]{}

func TestGenericPointSliceWrapping(t *testing.T) {
	const size = 256
	testutils.Assert(size >= 1)
	var A1_TestInterfaceSlice []CurvePointPtrInterfaceTestSample = getPrecomputedCurvePointSlice(2, pointTypeAXTWSubgroup, size)

	A2 := getPrecomputedCurvePointSlice(3, pointTypeAXTWSubgroup, size)
	var A2_Array [size]Point_axtw_subgroup
	for i := 0; i < size; i++ {
		A2_Array[i].SetFrom(A2[i])
	}

	type dummyAXTW struct {
		Point_axtw_subgroup
	}

	A3 := getPrecomputedCurvePointSlice(4, pointTypeAXTWSubgroup, size)
	var A3_Array [size]dummyAXTW
	for i := 0; i < size; i++ {
		A3_Array[i].SetFrom(A3[i])
	}

	// var temp1, temp2 Point_axtw_subgroup

	WrapA1 := AsCurvePointPtrSlice(A1_TestInterfaceSlice)
	WrapA2 := AsCurvePointSlice(A2_Array[:])
	WrapA3 := AsCurvePointSlice(A3_Array[:])

	if WrapA1.Len() != size {
		t.Fatalf("Wrapped A1 has invalid length")
	}

	if WrapA2.Len() != size {
		t.Fatalf("Wrapped A2 has invalid length")
	}

	if WrapA3.Len() != size {
		t.Fatalf("Wrapped A3 has invalid length")
	}

	// Note: These comparisons compare pointers
	if WrapA1.GetByIndex(1) != A1_TestInterfaceSlice[1] {
		t.Fatalf("Wrapped A1 does not retrieve element")
	}
	if WrapA2.GetByIndex(1) != &A2_Array[1] {
		t.Fatalf("Wrapped A2 does not retrieve element")
	}
	if WrapA3.GetByIndex(1) != &A3_Array[1] {
		t.Fatalf("Wrapped A3 does not retrieve element")
	}

}

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

	// REMOVED:

	// fun_genericArray := func(bInner *testing.B) {
	// 	prepareBenchmarkCurvePoints(bInner)
	//		for n := 0; n < bInner.N; n++ {
	//			DumpAXTW_subgroup[n%dumpSizeBench_curve].SetFrom(getElementFromCurvePointSlice(&A1, n%size))
	//		}
	//	}
	//	fun_genericSlice := func(bInner *testing.B) {
	//		prepareBenchmarkCurvePoints(bInner)
	//		for n := 0; n < bInner.N; n++ {
	//			DumpAXTW_subgroup[n%dumpSizeBench_curve].SetFrom(getElementFromCurvePointSlice(A1[:], n%size))
	//		}
	//	}
	fun_pointSliceReader := func(bInner *testing.B) {
		prepareBenchmarkCurvePoints(bInner)
		for n := 0; n < bInner.N; n++ {
			DumpAXTW_subgroup[n%dumpSizeBench_curve].SetFrom(CurvePointSlice_axtw_subgroup(A1[:]).GetByIndex(n % size))
		}
	}
	fun_GenericWrapper := func(bInner *testing.B) {
		generic := AsCurvePointSlice(A1[:])
		prepareBenchmarkCurvePoints(bInner)
		for n := 0; n < bInner.N; n++ {
			DumpAXTW_subgroup[n%dumpSizeBench_curve].SetFrom(generic.GetByIndex(n % size))
		}
	}
	fun_GenericWrapper2 := func(bInner *testing.B) {
		generic := AsCurvePointSlice(A1[:])
		prepareBenchmarkCurvePoints(bInner)
		for n := 0; n < bInner.N; n++ {
			DumpAXTW_subgroup[n%dumpSizeBench_curve].SetFrom(generic.GetByIndex(n % size).(*Point_axtw_subgroup))
		}
	}
	fun_GenericWrapperTyped := func(bInner *testing.B) {
		generic := AsCurvePointSlice(A1[:]).(CurvePointSlice_axtw_subgroup)
		prepareBenchmarkCurvePoints(bInner)
		for n := 0; n < bInner.N; n++ {
			DumpAXTW_subgroup[n%dumpSizeBench_curve].SetFrom(generic.GetByIndexTyped(n % size))
		}
	}
	fun_GenericWrapperTypeErased := func(bInner *testing.B) {
		var generic CurvePointSlice = AsCurvePointSlice(A1[:])
		prepareBenchmarkCurvePoints(bInner)
		for n := 0; n < bInner.N; n++ {
			DumpAXTW_subgroup[n%dumpSizeBench_curve].SetFrom(generic.GetByIndex(n % size))
		}
	}

	/*
		Generic := func(bInner *testing.B) {
			prepareBenchmarkCurvePoints(bInner)
			for n := 0; n < bInner.N; n++ {
				DumpAXTW_subgroup[n%dumpSizeBench_curve].SetFrom(TGetElementFrom(A1[:], n%size))
			}
		}
	*/

	bOuter.Run("direct access", fun_direct)
	bOuter.Run("SetFrom", fun_SetFrom)
	// bOuter.Run("SetFrom getElementFromCurvePointSlice(&arr)", fun_genericArray)
	// bOuter.Run("SetFrom getElementFromCurvePointSlice(arr[:])", fun_genericSlice)
	bOuter.Run("SetFrom via PointSliceReader interface", fun_pointSliceReader)
	bOuter.Run("SetFrom via AsCurvePointSlice", fun_GenericWrapper)
	bOuter.Run("SetFrom via AsCurvePointSlice with type assertion", fun_GenericWrapper2)
	bOuter.Run("SetFrom via AsCurvePointSlice with typed getter", fun_GenericWrapperTyped)
	bOuter.Run("SetFrom via AsCurvePointSlice (concrete type erased)", fun_GenericWrapperTypeErased)

	// bOuter.Run("SetFrom via Generic", Generic)
}

/// OLD CODE :

/*







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
*/
