package curvePoints

import (
	"math/rand"
	"testing"
)

var _ bulkNormalizerAffineZ = CurvePointSlice_xtw_full{}
var _ bulkNormalizerAffineZ = CurvePointSlice_xtw_subgroup{}

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

func TestBulkNormalizeAffineZ_full(t *testing.T) {
	drng := rand.New(rand.NewSource(1))
	const size = 300
	var points [size]Point_xtw_full
	for i := 0; i < len(points); i++ {
		points[i].sampleRandomUnsafe(drng)
	}
	testMultiAffineZWorks(t, CurvePointSlice_xtw_full(points[:]))
	for i := 0; i < len(points); i++ {
		points[i].sampleRandomUnsafe(drng)
	}
	testMultiAffineZWorks(t, CurvePointSlice_xtw_full(points[0:1]))
	for i := 0; i < len(points); i++ {
		points[i].sampleRandomUnsafe(drng)
		if i%2 == 0 {
			points[i].normalizeAffineZ()
		}
	}
	// points[200].SetE1()
	testMultiAffineZWorks(t, CurvePointSlice_xtw_full(points[:]))
}

func TestBulkNormalizeAffineZ_subgroup(t *testing.T) {
	drng := rand.New(rand.NewSource(1))
	const size = 300
	var points [size]Point_xtw_subgroup
	for i := 0; i < len(points); i++ {
		points[i].sampleRandomUnsafe(drng)
	}
	testMultiAffineZWorks(t, CurvePointSlice_xtw_subgroup(points[:]))
	for i := 0; i < len(points); i++ {
		points[i].sampleRandomUnsafe(drng)
	}
	testMultiAffineZWorks(t, CurvePointSlice_xtw_subgroup(points[0:1]))
	for i := 0; i < len(points); i++ {
		points[i].sampleRandomUnsafe(drng)
		if i%2 == 0 {
			points[i].normalizeAffineZ()
		}
	}
	testMultiAffineZWorks(t, CurvePointSlice_xtw_subgroup(points[:]))
}
