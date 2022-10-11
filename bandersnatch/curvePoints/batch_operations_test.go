package curvePoints

import (
	"math/rand"
	"testing"

	"github.com/GottfriedHerold/Bandersnatch/internal/testutils"
	"github.com/GottfriedHerold/Bandersnatch/internal/utils"
)

var _ BatchNormalizerForZ = CurvePointSlice_xtw_full{}
var _ BatchNormalizerForZ = CurvePointSlice_xtw_subgroup{}

var _ BatchNormalizerForY = CurvePointSlice_xtw_full{}
var _ BatchNormalizerForY = CurvePointSlice_xtw_subgroup{}

// Test BatchNormalizeForZ.
// The tests work by calling testBatchNormalizerForZ(t, vec), which calls
// vec.BatchNormalizeForZ() and tests its properties.
//
// We do this for multiple versions to generate vec; generating vec in various ways is delegated to verious generic functions.

func TestBatchNormalizerForZ_xtw_full(t *testing.T) {
	testBatchNormalizeZZeroed[Point_xtw_full, *Point_xtw_full, CurvePointSlice_xtw_full](t)
	testBatchNormalizeZRandom[Point_xtw_full, *Point_xtw_full, CurvePointSlice_xtw_full](t)
	testBatchNormalizeZWithInfinity[Point_xtw_full, *Point_xtw_full, CurvePointSlice_xtw_full](t)
}

func TestBatchNormalizerForZ_xtw_subgroup(t *testing.T) {
	testBatchNormalizeZZeroed[Point_xtw_subgroup, *Point_xtw_subgroup, CurvePointSlice_xtw_subgroup](t)
	testBatchNormalizeZRandom[Point_xtw_subgroup, *Point_xtw_subgroup, CurvePointSlice_xtw_subgroup](t)
}

func TestBatchNormalizerForY_xtw_full(t *testing.T) {
	testBatchNormalizeYZeroed[Point_xtw_full, *Point_xtw_full, CurvePointSlice_xtw_full](t)
	testBatchNormalizeYRandom[Point_xtw_full, *Point_xtw_full, CurvePointSlice_xtw_full](t)
	testBatchNormalizeYWithInfinity[Point_xtw_full, *Point_xtw_full, CurvePointSlice_xtw_full](t)
}

func TestBatchNormalizerForY_xtw_subgroup(t *testing.T) {
	testBatchNormalizeYZeroed[Point_xtw_subgroup, *Point_xtw_subgroup, CurvePointSlice_xtw_subgroup](t)
	testBatchNormalizeYRandom[Point_xtw_subgroup, *Point_xtw_subgroup, CurvePointSlice_xtw_subgroup](t)
}

func TestBatchNormalizeForZ_axtw_full(t *testing.T) {
	testBatchNormalizeZZeroed[Point_axtw_full, *Point_axtw_full, CurvePointSlice_axtw_full](t)
	testBatchNormalizeZRandom[Point_axtw_full, *Point_axtw_full, CurvePointSlice_axtw_full](t)
}

func TestBatchNormalizeForZ_axtw_subgroup(t *testing.T) {
	testBatchNormalizeZZeroed[Point_axtw_subgroup, *Point_axtw_subgroup, CurvePointSlice_axtw_subgroup](t)
	testBatchNormalizeZRandom[Point_axtw_subgroup, *Point_axtw_subgroup, CurvePointSlice_axtw_subgroup](t)
}

// testBatchNormalizerForZ is a testing sub-routine; it ensures that BatchNormalizeForZ works correctly when called on vec.
func testBatchNormalizerForZ(t *testing.T, vec BatchNormalizerForZ) {
	L := vec.Len()

	// Make a copy of vec before we do anything.
	var vecCopy []CurvePointPtrInterface = make([]CurvePointPtrInterface, L)
	for i := 0; i < L; i++ {
		vecCopy[i] = vec.GetByIndex(i).Clone()
	}

	// call BatchNormalizeForZ and check whether the NaP-handler was called.
	// The call is wrapped in wasInvalidPointEncountered to detect whether the NaP-handler was called
	var failingIndices []int
	do_batch_invert := func() {
		failingIndices = vec.BatchNormalizeForZ()
	}
	detectedNaP := wasInvalidPointEncountered(do_batch_invert)

	// failingIndices may is nil rather than an empty slice if no error occured.
	failuresEncountered := (failingIndices != nil)
	if failuresEncountered {
		testutils.FatalUnless(t, len(failingIndices) > 0, "failingIndices was non-nil slice of lenght 0")
	}

	// ensure failingIndices is nil or a sortedlist of indices in range.
	if failuresEncountered && len(failingIndices) > 1 {
		for i := 0; i < len(failingIndices)-1; i++ {
			testutils.FatalUnless(t, failingIndices[i] < failingIndices[i+1], "failingIndices returned by BatchNormalizeForZ not sorted")
		}
	}

	for _, i := range failingIndices {
		testutils.FatalUnless(t, i >= 0, "failingIndices returned by BatchNormalizerForZ not in range")
		testutils.FatalUnless(t, i < L, "failingIndices returned by BatchNormalizerForZ not in range")
	}

	// Check that the operation on each point was correct:
	for i := 0; i < L; i++ {
		original := vecCopy[i]
		modified := vec.GetByIndex(i)
		// NaP-Case.
		if original.IsNaP() {
			testutils.FatalUnless(t, modified.IsNaP(), "BatchNormalizeZ turned NaP into non-NaP")
			// We do not require detecting NaPs, so just keeping it a NaP is OK.
			if failuresEncountered && utils.ElementInList(i, failingIndices) {
				testutils.FatalUnless(t, detectedNaP, "failingIndices failed to call NaP handler")
			}
			continue // skip other properties
		}
		// If non-NaP, we should not change the point semantically. Note the .Clone() is to prevent the equality-test from potentially modifying the representation.
		testutils.FatalUnless(t, original.IsEqual(modified.Clone()), "BatchNormalizeZ changed point")
		if original.IsAtInfinity() {
			testutils.FatalUnless(t, failuresEncountered && utils.ElementInList(i, failingIndices), "Point at infinity did not register as failure in BatchNormalizeForZ")
		} else {
			testutils.FatalUnless(t, !utils.ElementInList(i, failingIndices), "Good point was registered as failing in BatchNormalizeForZ")
			Z_decaf := modified.Z_decaf_projective()
			testutils.FatalUnless(t, Z_decaf.IsOne(), "BatchNormalizeForZ did not set Z to 1")
		}
	}
}

// testBatchNormalizerForZ is a testing sub-routine; it ensures that BatchNormalizeForZ works correctly when called on vec.
func testBatchNormalizerForY(t *testing.T, vec BatchNormalizerForY) {
	L := vec.Len()

	// Make a copy of vec before we do anything.
	var vecCopy []CurvePointPtrInterface = make([]CurvePointPtrInterface, L)
	for i := 0; i < L; i++ {
		vecCopy[i] = vec.GetByIndex(i).Clone()
	}

	// call BatchNormalizeForY and check whether the NaP-handler was called.
	// The call is wrapped in wasInvalidPointEncountered to detect whether the NaP-handler was called
	var failingIndices []int
	do_batch_invert := func() {
		failingIndices = vec.BatchNormalizeForY()
	}
	detectedNaP := wasInvalidPointEncountered(do_batch_invert)

	// failingIndices may is nil rather than an empty slice if no error occured.
	failuresEncountered := (failingIndices != nil)
	if failuresEncountered {
		testutils.FatalUnless(t, len(failingIndices) > 0, "failingIndices was non-nil slice of lenght 0")
	}

	// ensure failingIndices is nil or a sortedlist of indices in range.
	if failuresEncountered && len(failingIndices) > 1 {
		for i := 0; i < len(failingIndices)-1; i++ {
			testutils.FatalUnless(t, failingIndices[i] < failingIndices[i+1], "failingIndices returned by BatchNormalizeForY not sorted")
		}
	}

	for _, i := range failingIndices {
		testutils.FatalUnless(t, i >= 0, "failingIndices returned by BatchNormalizerForY not in range")
		testutils.FatalUnless(t, i < L, "failingIndices returned by BatchNormalizerForY not in range")
	}

	// Check that the operation on each point was correct:
	for i := 0; i < L; i++ {
		original := vecCopy[i]
		modified := vec.GetByIndex(i)
		// NaP-Case.
		if original.IsNaP() {
			testutils.FatalUnless(t, modified.IsNaP(), "BatchNormalizeY turned NaP into non-NaP")
			// We do not require detecting NaPs, so just keeping it a NaP is OK.
			if failuresEncountered && utils.ElementInList(i, failingIndices) {
				testutils.FatalUnless(t, detectedNaP, "failingIndices failed to call NaP handler")
			}
			continue // skip other properties
		}
		// If non-NaP, we should not change the point semantically. Note the .Clone() is to prevent the equality-test from potentially modifying the representation.
		testutils.FatalUnless(t, original.IsEqual(modified.Clone()), "BatchNormalizeY changed point")
		if original.IsAtInfinity() {
			testutils.FatalUnless(t, failuresEncountered && utils.ElementInList(i, failingIndices), "Point at infinity did not register as failure in BatchNormalizeForY")
		} else {
			testutils.FatalUnless(t, !utils.ElementInList(i, failingIndices), "Good point was registered as failing in BatchNormalizeForY")
			Y_decaf := modified.Y_decaf_projective()
			testutils.FatalUnless(t, Y_decaf.IsOne(), "BatchNormalizeForY did not set Z to 1")
		}
	}
}

const sizeForBatchNormalizerTests = 300

// individual functions to create vec and call testBatchNormalizerForZ follow.
// Note that we call the test twice to check whether it still works on the updated points.

func testBatchNormalizeZZeroed[PointType any, PointPtr interface {
	*PointType
	CurvePointPtrInterface
}, SliceType BatchNormalizerForZ](t *testing.T) {
	// drng := rand.New(rand.NewSource(1))
	var points [sizeForBatchNormalizerTests]PointType
	pointSlice := AsCurvePointSlice[PointType, PointPtr](points[:]).(SliceType) // this is a view into points and tracks changes.
	testBatchNormalizerForZ(t, pointSlice)                                      // all-NaP
	testBatchNormalizerForZ(t, pointSlice)                                      // all-NaP
	var points2 [0]PointType
	pointSlice2 := AsCurvePointSlice[PointType, PointPtr](points2[:]).(SliceType) // this is a view into points and tracks changes.
	testBatchNormalizerForZ(t, pointSlice2)                                       // length-0 slice
}

func testBatchNormalizeZRandom[PointType any, PointPtr interface {
	*PointType
	CurvePointPtrInterface
	CurvePointPtrInterfaceTestSample
}, SliceType BatchNormalizerForZ](t *testing.T) {
	drng := rand.New(rand.NewSource(1))

	var points [sizeForBatchNormalizerTests]PointType
	pointSlice := AsCurvePointSlice[PointType, PointPtr](points[:]).(SliceType) // this is a view into points and tracks changes.
	for i := 0; i < len(points); i++ {
		PointPtr(&points[i]).sampleRandomUnsafe(drng)
	}
	testBatchNormalizerForZ(t, pointSlice) // all-NaP
	testBatchNormalizerForZ(t, pointSlice) // all-NaP
	for i := 0; i < len(points); i++ {
		PointPtr(&points[i]).sampleRandomUnsafe(drng)
	}
	testBatchNormalizerForZ(t, pointSlice) // all-NaP
	testBatchNormalizerForZ(t, pointSlice) // all-NaP
	for i := 0; i < len(points); i++ {
		point := PointPtr(&points[i])
		switch i % 2 {
		case 0:
			point.sampleRandomUnsafe(drng)
		case 1:
			point.sampleNaP(drng, i)
		}
	}
	testBatchNormalizerForZ(t, pointSlice) // all-NaP
	testBatchNormalizerForZ(t, pointSlice) // all-NaP
}

func testBatchNormalizeZWithInfinity[PointType any, PointPtr interface {
	*PointType
	CurvePointPtrInterface
	CurvePointPtrInterfaceTestSample
	curvePointPtrInterfaceTestSampleE
	curvePointPtrInterfaceTestSampleA
}, SliceType BatchNormalizerForZ](t *testing.T) {
	drng := rand.New(rand.NewSource(1))

	var points [sizeForBatchNormalizerTests]PointType
	pointSlice := AsCurvePointSlice[PointType, PointPtr](points[:]).(SliceType) // this is a view into points and tracks changes.
	for i := 0; i < len(points); i++ {
		point := PointPtr(&points[i])
		switch i % 6 {
		case 0:
			point.SetE1()
		case 1:
			point.SetE2()
		case 2:
			point.SetAffineTwoTorsion()
		case 3:
			point.sampleNaP(drng, i)
		default:
			point.sampleRandomUnsafe(drng)
		}
	}
	testBatchNormalizerForZ(t, pointSlice) // all-NaP
	testBatchNormalizerForZ(t, pointSlice) // all-NaP
}

func testBatchNormalizeYZeroed[PointType any, PointPtr interface {
	*PointType
	CurvePointPtrInterface
}, SliceType BatchNormalizerForY](t *testing.T) {
	// drng := rand.New(rand.NewSource(1))
	var points [sizeForBatchNormalizerTests]PointType
	pointSlice := AsCurvePointSlice[PointType, PointPtr](points[:]).(SliceType) // this is a view into points and tracks changes.
	testBatchNormalizerForY(t, pointSlice)                                      // all-NaP
	testBatchNormalizerForY(t, pointSlice)                                      // all-NaP
	var points2 [0]PointType
	pointSlice2 := AsCurvePointSlice[PointType, PointPtr](points2[:]).(SliceType) // this is a view into points and tracks changes.
	testBatchNormalizerForY(t, pointSlice2)                                       // length-0 slice
}

func testBatchNormalizeYRandom[PointType any, PointPtr interface {
	*PointType
	CurvePointPtrInterface
	CurvePointPtrInterfaceTestSample
}, SliceType BatchNormalizerForY](t *testing.T) {
	drng := rand.New(rand.NewSource(1))

	var points [sizeForBatchNormalizerTests]PointType
	pointSlice := AsCurvePointSlice[PointType, PointPtr](points[:]).(SliceType) // this is a view into points and tracks changes.
	for i := 0; i < len(points); i++ {
		PointPtr(&points[i]).sampleRandomUnsafe(drng)
	}
	testBatchNormalizerForY(t, pointSlice) // all-NaP
	testBatchNormalizerForY(t, pointSlice) // all-NaP
	for i := 0; i < len(points); i++ {
		PointPtr(&points[i]).sampleRandomUnsafe(drng)
	}
	testBatchNormalizerForY(t, pointSlice) // all-NaP
	testBatchNormalizerForY(t, pointSlice) // all-NaP
	for i := 0; i < len(points); i++ {
		point := PointPtr(&points[i])
		switch i % 2 {
		case 0:
			point.sampleRandomUnsafe(drng)
		case 1:
			point.sampleNaP(drng, i)
		}
	}
	testBatchNormalizerForY(t, pointSlice) // all-NaP
	testBatchNormalizerForY(t, pointSlice) // all-NaP
}

func testBatchNormalizeYWithInfinity[PointType any, PointPtr interface {
	*PointType
	CurvePointPtrInterface
	CurvePointPtrInterfaceTestSample
	curvePointPtrInterfaceTestSampleE
	curvePointPtrInterfaceTestSampleA
}, SliceType BatchNormalizerForY](t *testing.T) {
	drng := rand.New(rand.NewSource(1))

	var points [sizeForBatchNormalizerTests]PointType
	pointSlice := AsCurvePointSlice[PointType, PointPtr](points[:]).(SliceType) // this is a view into points and tracks changes.
	for i := 0; i < len(points); i++ {
		point := PointPtr(&points[i])
		switch i % 6 {
		case 0:
			point.SetE1()
		case 1:
			point.SetE2()
		case 2:
			point.SetAffineTwoTorsion()
		case 3:
			point.sampleNaP(drng, i)
		default:
			point.sampleRandomUnsafe(drng)
		}
	}
	testBatchNormalizerForY(t, pointSlice) // all-NaP
	testBatchNormalizerForY(t, pointSlice) // all-NaP
}
