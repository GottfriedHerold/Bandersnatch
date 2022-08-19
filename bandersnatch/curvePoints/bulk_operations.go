package curvePoints

import (
	"errors"

	"github.com/GottfriedHerold/Bandersnatch/bandersnatch/fieldElements"
)

// normalizeAffineZ normalizes the internal representation of a slice of Point_xtw_full with an equivalent one with Z==1.
//
// This is MUCH more efficient than calling normalizeAffineZ individually on each point.
func (points CurvePointSlice_xtw_full) normalizeAffineZ() {
	L := len(points)
	// We bulk-invert all the points[i].z's.
	// For efficiency, we skip all the ones.

	var Ones []bool = make([]bool, L) // boolean array with One[i] being true if points[i] already has z==1
	var numOnes int = 0               // number of elements encountered that already had z==1
	var Inversions []*FieldElement = make([]*FieldElement, L)
	for i := 0; i < L; i++ {
		if points[i].z.IsOne() {
			Ones[i] = true
			numOnes++
			continue
		}
		Inversions[i-numOnes] = &points[i].z
	}

	Inversions = Inversions[0 : L-numOnes]
	var err error = fieldElements.MultiInvertEq(Inversions...) // decay returned error type to plain error
	if err != nil {
		if !errors.Is(err, fieldElements.ErrDivisionByZero) {
			panic(ErrorPrefix + "fieldElements.MultiInvertEq returned error of unexpected type")
		}
		// Reset Inversions without the Ones skipped. This is needed to get a correct error message (the error refers to indices, which would be wrong if we skip the ones)
		Inversions = Inversions[0:L]
		for i := 0; i < L; i++ {
			Inversions[i] = &points[i].z
		}
		err = fieldElements.GenerateMultiDivisionByZeroError(Inversions, ErrorPrefix+"bulk normalization of Z-coordinate to 1 failed, because some Z-coordinates were 0.")
		panic(err)
	}
	for i := 0; i < L; i++ {
		if Ones[i] {
			continue
		}
		point := &points[i]
		point.x.MulEq(&point.z)
		point.y.MulEq(&point.z)
		point.t.MulEq(&point.z)
		point.z.SetOne()
	}
}

// normalizeAffineZ normalizes the internal representation of a slice of Point_xtw_full with an equivalent one with Z==1.
//
// This is MUCH more efficient than calling normalizeAffineZ individually on each point.
func (points CurvePointSlice_xtw_subgroup) normalizeAffineZ() {
	L := len(points)
	// We bulk-invert all the points[i].z's.
	// For efficiency, we skip all the ones.
	var Ones []bool = make([]bool, L) // boolean array with One[i] being true if points[i] already has z==1
	var numOnes int = 0               // number of elements encountered that already had z==1
	var Inversions []*FieldElement = make([]*FieldElement, L)
	for i := 0; i < L; i++ {
		if points[i].z.IsOne() {
			Ones[i] = true
			numOnes++
			continue
		}
		Inversions[i-numOnes] = &points[i].z
	}

	Inversions = Inversions[0 : L-numOnes]
	var err error = fieldElements.MultiInvertEq(Inversions...) // decay returned error type to plain error
	if err != nil {
		if !errors.Is(err, fieldElements.ErrDivisionByZero) {
			panic(ErrorPrefix + "fieldElements.MultiInvertEq returned error of unexpected type")
		}
		// Reset Inversions without the Ones skipped. This is needed to get a correct error message (the error refers to indices, which would be wrong if we skip the ones)
		Inversions = Inversions[0:L]
		for i := 0; i < L; i++ {
			Inversions[i] = &points[i].z
		}
		err = fieldElements.GenerateMultiDivisionByZeroError(Inversions, ErrorPrefix+"bulk normalization of Z-coordinate to 1 failed, because some Z-coordinates were 0.")
		panic(err)
	}

	for i := 0; i < L; i++ {
		if Ones[i] {
			continue
		}
		point := &points[i]
		point.x.MulEq(&point.z)
		point.y.MulEq(&point.z)
		point.t.MulEq(&point.z)
		point.z.SetOne()
	}
}
