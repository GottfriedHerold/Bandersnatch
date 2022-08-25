package curvePoints

import (
	"fmt"

	"github.com/GottfriedHerold/Bandersnatch/bandersnatch/fieldElements"
)

// NormalizeSlice normalizes the internal representation of a slice of Point_xtw_full with an equivalent one with Z==1.
//
// This is MUCH more efficient than calling normalizeAffineZ individually on each point.
//
// If none of the points have Z-coordinate 0, return nil. Otherwise, we return a sorted list of indices of those elements which were 0.
// Those points with Z-coordinate 0 are set to an appropriate standard representation.
func (points CurvePointSlice_xtw_full) NormalizeSlice() (zeroIndices []int) {
	L := len(points)

	// We bulk-invert all the points[i].z's.
	// For efficiency, we use the SkipZeros version (because that skip all the ones as well.)
	var Inversions []*FieldElement = make([]*FieldElement, L)
	for i := 0; i < L; i++ {
		Inversions[i] = &points[i].z
	}
	zeroIndices = fieldElements.MultiInvertEqSkipZeros(Inversions...)

	// nextZeroIndex is the first/next index where we need to treat the point specially.
	nextZeroIndex := L
	zeroIndicesProcessed := 0

	if zeroIndices != nil {
		nextZeroIndex = zeroIndices[0]
	}

	for i := 0; i < L; i++ {
		point := &points[i]
		if i != nextZeroIndex {
			point.x.MulEq(&point.z)
			point.y.MulEq(&point.z)
			point.t.MulEq(&point.z)
			point.z.SetOne()
		} else {
			// i is in zeroIndices. We need to treat these cases specially:

			// Ensure next zero index is found:
			zeroIndicesProcessed++
			if zeroIndicesProcessed == len(zeroIndices) {
				nextZeroIndex = L
			} else {
				nextZeroIndex = zeroIndices[zeroIndicesProcessed]
			}

			// We normalize points with Z==0:
			// NaPs become standard (0:0:0:0)-NaPs
			if point.IsNaP() {
				napEncountered("NaP encountered when bulk-normalizing a slice of Point_xtw_full points", false, point)
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

// NormalizeSlice normalizes the internal representation of a slice of Point_xtw_subgroup with an equivalent one with Z==1.
//
// This is MUCH more efficient than calling normalizeAffineZ individually on each point.
//
// If none of the points have Z-coordinate 0, return nil. Otherwise, we return a sorted list of indices of those elements which were 0.
// Those points with Z-coordinate 0 are set to an appropriate standard representation.
func (points CurvePointSlice_xtw_subgroup) NormalizeSlice() (zeroIndices []int) {
	L := len(points)

	// We bulk-invert all the points[i].z's.
	// For efficiency, we use the SkipZeros version (because that skip all the ones as well.)
	var Inversions []*FieldElement = make([]*FieldElement, L)
	for i := 0; i < L; i++ {
		Inversions[i] = &points[i].z
	}
	zeroIndices = fieldElements.MultiInvertEqSkipZeros(Inversions...)

	// nextZeroIndex is the first/next index where we need to treat the point specially.
	nextZeroIndex := L
	zeroIndicesProcessed := 0

	if zeroIndices != nil {
		nextZeroIndex = zeroIndices[0]
	}

	for i := 0; i < L; i++ {
		point := &points[i]
		if i != nextZeroIndex {
			point.x.MulEq(&point.z)
			point.y.MulEq(&point.z)
			point.t.MulEq(&point.z)
			point.z.SetOne()
		} else {
			// i is in zeroIndices. We need to treat these cases specially:

			// Ensure next zero index is found:
			zeroIndicesProcessed++
			if zeroIndicesProcessed == len(zeroIndices) {
				nextZeroIndex = L
			} else {
				nextZeroIndex = zeroIndices[zeroIndicesProcessed]
			}

			// We normalize points with Z==0:
			// NaPs become standard (0:0:0:0)-NaPs
			if point.IsNaP() {
				napEncountered("NaP encountered when bulk-normalizing a slice of Point_xtw_subgroup points", false, point)
				*point = Point_xtw_subgroup{} // standard-NAP
				continue
			}

			// Points at infinity are not possible for this point type!

			panic(fmt.Errorf(ErrorPrefix+"Point with Z==0 encountered that was neither a NaP nor a point at inifity. This is not supposed to be possible. The Point in question was %v", *point))
		}
	}
	return
}
