package curvePoints

import (
	"fmt"

	"github.com/GottfriedHerold/Bandersnatch/bandersnatch/fieldElements"
)

// BatchNormalizeForZ is (almost) equivalent to calling NormalizeForZ on each entry, but much more efficient.
//
// If non-nil, the returned slice is sorted and indicites the (0-based) indices where BatchNormalizeForZ failed.
// NOTE: The only observable difference to calling NormalizeForZ individually is for NaPs.
// This method may fail to recognize NaPs and change their internal representation to one with Z_decaf_projective() returning 1 without indicating it in failingIndices.
func (points CurvePointSlice_xtw_full) BatchNormalizeForZ() (failingIndices []int) {
	L := len(points)

	// We bulk-invert all the points[i].z's.
	// For efficiency, we use the SkipZeros version (because that skip all the ones as well.)
	var Inversions []*FieldElement = make([]*FieldElement, L)
	for i := 0; i < L; i++ {
		Inversions[i] = &points[i].z
	}
	failingIndices = fieldElements.MultiInvertEqSkipZeros(Inversions...)
	// Note that we just replaced every z by z^-1 in every point. Consequently, our points are in an invalid state for now.

	// We now go over all points to adjust the actual x,y,t coordinates and set z to 1.
	// However, we need to treat the points were z==0 was encountered specially.

	// A naive check whether a given index is in failingIndices would give quadratic running time just for the checks, so we rather use the fact that
	// failingIndices is sorted and make a single pass.

	// nextFailingIndex is the first/next index where we need to treat the point specially.
	nextFailingIndex := L
	failingIndicesProcessed := 0

	if failingIndices != nil {
		nextFailingIndex = failingIndices[0]
	}

	for i := 0; i < L; i++ {
		point := &points[i]
		if i != nextFailingIndex {
			// point.z holds the inverse to the "real z", so adjust accordingly.
			point.x.MulEq(&point.z)
			point.y.MulEq(&point.z)
			point.t.MulEq(&point.z)
			point.z.SetOne()
		} else {
			// i is in failingIndices, so z was unchanged at 0. We need to treat these cases specially:

			// First ensure the next entry in failingIndices will we handled properly:
			failingIndicesProcessed++
			if failingIndicesProcessed == len(failingIndices) {
				nextFailingIndex = L
			} else {
				nextFailingIndex = failingIndices[failingIndicesProcessed]
			}

			// We normalize points with Z==0:
			// NaPs become standard (0:0:0:0)-NaPs
			if point.IsNaP() {
				napEncountered("NaP encountered in BatchNormalizeForZ for a slice of Point_xtw_full points", false, point)
				*point = Point_xtw_full{} // standard-NAP
				continue
			}
			// Points at infinity get replaced by default representations of the appropriate point at infinity:
			if point.IsE1() {
				point.SetE1()
				continue
			}
			if point.IsE2() {
				point.SetE2()
				continue
			}
			panic(fmt.Errorf(ErrorPrefix+"Point with Z==0 encountered that was neither a NaP nor a point at inifity. This is not supposed to be possible. The Point in question was %v", *point))
		}
	}
	return
}

// BatchNormalizeForZ is (almost) equivalent to calling NormalizeForZ on each entry, but much more efficient.
//
// If non-nil, the returned slice is sorted and indicites the (0-based) indices where BatchNormalizeForZ failed.
// NOTE: The only observable difference to calling NormalizeForZ individually is for NaPs.
// This method may fail to recognize NaPs and change their internal representation to one with Z_decaf_projective() returning 1 without indicating it in failingIndices.
func (points CurvePointSlice_xtw_subgroup) BatchNormalizeForZ() (failingIndices []int) {
	L := len(points)

	// We bulk-invert all the points[i].z's.
	// For efficiency, we use the SkipZeros version (because that skip all the ones as well.)
	var Inversions []*FieldElement = make([]*FieldElement, L)
	for i := 0; i < L; i++ {
		Inversions[i] = &points[i].z
	}
	failingIndices = fieldElements.MultiInvertEqSkipZeros(Inversions...)
	// Note that we just replaced every z by z^-1 in every point. Consequently, our points are in an invalid state for now.

	// We now go over all points to adjust the actual x,y,t coordinates and set z to 1.
	// However, we need to treat the points were z==0 was encountered specially.

	// A naive check whether a given index is in failingIndices would give quadratic running time just for the checks, so we rather use the fact that
	// failingIndices is sorted and make a single pass.

	// nextFailingIndex is the first/next index where we need to treat the point specially.
	nextFailingIndex := L
	failingIndicesProcessed := 0

	if failingIndices != nil {
		nextFailingIndex = failingIndices[0]
	}

	for i := 0; i < L; i++ {
		point := &points[i]
		if i != nextFailingIndex {
			// point.z holds the inverse to the "real z", so adjust accordingly.
			point.x.MulEq(&point.z)
			point.y.MulEq(&point.z)
			point.t.MulEq(&point.z)
			point.z.SetOne()
		} else {
			// i is in failingIndices, so z was unchanged at 0. We need to treat these cases specially:

			// First ensure the next entry in failingIndices will we handled properly:
			failingIndicesProcessed++
			if failingIndicesProcessed == len(failingIndices) {
				nextFailingIndex = L
			} else {
				nextFailingIndex = failingIndices[failingIndicesProcessed]
			}

			// We normalize points with Z==0:
			// NaPs become standard (0:0:0:0)-NaPs
			if point.IsNaP() {
				napEncountered("NaP encountered in BatchNormalizeForZ for slice of Point_xtw_subgroup points", false, point)
				*point = Point_xtw_subgroup{} // standard-NAP
				continue
			}
			panic(fmt.Errorf(ErrorPrefix+"Point with Z==0 encountered that was neither a NaP nor a point at inifity. This is not supposed to be possible. The Point in question was %v", *point))
		}
	}
	return
}

// BatchNormalizeForY is (almost) equivalent to calling NormalizeForY on each entry, but much more efficient.
//
// If non-nil, the returned slice is sorted and indicites the (0-based) indices where BatchNormalizeForZ failed.
// NOTE: The only observable difference to calling NormalizeForY individually is for NaPs.
// This method may fail to recognize NaPs and change their internal representation to one with Z_decaf_projective() returning 1 without indicating it in failingIndices.
func (points CurvePointSlice_xtw_full) BatchNormalizeForY() (failingIndices []int) {
	L := len(points)

	// We bulk-invert all the points[i].y's.
	// For efficiency, we use the SkipZeros version (because that skip all the ones as well.)
	var Inversions []*FieldElement = make([]*FieldElement, L)
	for i := 0; i < L; i++ {
		Inversions[i] = &points[i].y
	}
	failingIndices = fieldElements.MultiInvertEqSkipZeros(Inversions...)
	// Note that we just replaced every y by y^-1 in every point. Consequently, our points are in an invalid state for now.

	// We now go over all points to adjust the actual x,t,z coordinates and set y to 1.
	// However, we need to treat the points were y==0 was encountered specially.

	// A naive check whether a given index is in failingIndices would give quadratic running time just for the checks, so we rather use the fact that
	// failingIndices is sorted and make a single pass.

	// nextFailingIndex is the first/next index where we need to treat the point specially.
	nextFailingIndex := L
	failingIndicesProcessed := 0

	if failingIndices != nil {
		nextFailingIndex = failingIndices[0]
	}

	for i := 0; i < L; i++ {
		point := &points[i]
		if i != nextFailingIndex {
			// point.y holds the inverse to the "real y", so adjust accordingly.
			point.x.MulEq(&point.y)
			point.t.MulEq(&point.y)
			point.z.MulEq(&point.y)
			point.y.SetOne()
		} else {
			// i is in failingIndices, so y was unchanged at 0. We need to treat these cases specially:

			// First ensure the next entry in failingIndices will we handled properly:
			failingIndicesProcessed++
			if failingIndicesProcessed == len(failingIndices) {
				nextFailingIndex = L
			} else {
				nextFailingIndex = failingIndices[failingIndicesProcessed]
			}

			// We normalize points with Y==0:
			// NaPs become standard (0:0:0:0)-NaPs
			if point.IsNaP() {
				napEncountered("NaP encountered in BatchNormalizeForY for a slice of Point_xtw_full points", false, point)
				*point = Point_xtw_full{} // standard-NAP
				continue
			}
			// Points at infinity get replaced by default representations of the appropriate point at infinity:
			if point.IsE1() {
				point.SetE1()
				continue
			}
			if point.IsE2() {
				point.SetE2()
				continue
			}
			panic(fmt.Errorf(ErrorPrefix+"Point with Y==0 encountered that was neither a NaP nor a point at inifity. This is not supposed to be possible. The Point in question was %v", *point))
		}
	}
	return
}

// BatchNormalizeForY is (almost) equivalent to calling NormalizeForY on each entry, but much more efficient.
//
// If non-nil, the returned slice is sorted and indicites the (0-based) indices where BatchNormalizeForY failed.
// NOTE: The only observable difference to calling NormalizeForY individually is for NaPs.
// This method may fail to recognize NaPs and change their internal representation to one with Y_decaf_projective() returning 1 without indicating it in failingIndices.
func (points CurvePointSlice_xtw_subgroup) BatchNormalizeForY() (failingIndices []int) {
	L := len(points)

	// We bulk-invert all the points[i].y's.
	// For efficiency, we use the SkipZeros version (because that skip all the ones as well.)
	var Inversions []*FieldElement = make([]*FieldElement, L)
	for i := 0; i < L; i++ {
		Inversions[i] = &points[i].y
	}
	failingIndices = fieldElements.MultiInvertEqSkipZeros(Inversions...)
	// Note that we just replaced every y by y^-1 in every point. Consequently, our points are in an invalid state for now.

	// We now go over all points to adjust the actual x,y,t,z coordinates.
	// However, we need to treat the points were y==0 was encountered specially.

	// A naive check whether a given index is in failingIndices would give quadratic running time just for the checks, so we rather use the fact that
	// failingIndices is sorted and make a single pass.

	// nextFailingIndex is the first/next index where we need to treat the point specially.
	nextFailingIndex := L
	failingIndicesProcessed := 0

	if failingIndices != nil {
		nextFailingIndex = failingIndices[0]
	}

	for i := 0; i < L; i++ {
		point := &points[i]
		if i != nextFailingIndex {
			// point.z holds the inverse to the "real z", so adjust accordingly.
			point.x.MulEq(&point.y)
			point.t.MulEq(&point.y)
			point.z.MulEq(&point.y)

			point.y.SetOne()
		} else {
			// i is in failingIndices, so y was unchanged at 0. We need to treat these cases specially:

			// First ensure the next entry in failingIndices will we handled properly:
			failingIndicesProcessed++
			if failingIndicesProcessed == len(failingIndices) {
				nextFailingIndex = L
			} else {
				nextFailingIndex = failingIndices[failingIndicesProcessed]
			}

			if !point.IsNaP() {
				panic(fmt.Errorf(ErrorPrefix+"Point with Y==0 encountered that was neither a NaP nor a point at inifity. This is not supposed to be possible. The Point in question was %v", *point))
			}

			// We normalize points with Y==0:
			// NaPs become standard (0:0:0:0)-NaPs
			napEncountered("NaP encountered in BatchNormalizeForY for slice of Point_xtw_subgroup points", false, point)
			*point = Point_xtw_subgroup{} // standard-NAP
		}
	}
	return
}

// BatchNormalizeForZ is (almost) equivalent to calling NormalizeForZ on each entry, but much more efficient.
//
// If non-nil, the returned slice is sorted and indicites the (0-based) indices where BatchNormalizeForZ failed.
// NOTE: The only observable difference to calling NormalizeForZ individually is for NaPs.
// This method may fail to recognize NaPs and change their internal representation to one with Z_decaf_projective() returning 1 without indicating it in failingIndices.
func (points CurvePointSlice_axtw_full) BatchNormalizeForZ() (failingIndices []int) {
	return nil
}

// BatchNormalizeForZ is (almost) equivalent to calling NormalizeForZ on each entry, but much more efficient.
//
// If non-nil, the returned slice is sorted and indicites the (0-based) indices where BatchNormalizeForZ failed.
// NOTE: The only observable difference to calling NormalizeForZ individually is for NaPs.
// This method may fail to recognize NaPs and change their internal representation to one with Z_decaf_projective() returning 1 without indicating it in failingIndices.
func (points CurvePointSlice_axtw_subgroup) BatchNormalizeForZ() (failingIndices []int) {
	return nil
}
