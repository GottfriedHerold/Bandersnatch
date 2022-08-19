package curvePoints

import "github.com/GottfriedHerold/Bandersnatch/bandersnatch/fieldElements"

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

	// TODO: Redo!!!

	/*

		// We catch panics due to division by zero here. The reason is that that the error message we would get by default is misleading:
		// Due to skipping the ones, the indices are wrong. Further, we want to distinguish NaPs from infinite points.
		defer func() {
			err := recover()
			if err != nil {
				switch err := err.(type) {
				case *errMultiInversionEncounteredZero:
					err.s += "\nNOTE: For this call site, indices in MultiInvertEq skip curve points with z==1"
					for i := 0; i < L; i++ {
						if points[i].IsNaP() {
							panic(fmt.Errorf("bandersnatch / curve Points: trying to bulk-convert multiple points to affine coordinates, out of which some were invalid: The first error was at index %v, which was a NAP", i))
						}
						if points[i].IsAtInfinity() {
							panic(fmt.Errorf("bandersnatch / curve Points: trying to bulk-convert multiple points to affine coordinates, out of which some were invalid: The first error was at index %v, which was a point at infinity", i))
						}
					}
				default:
					// This should not happen.
					panic(err)
				}

			}
		}()

	*/

	Inversions = Inversions[0 : L-numOnes]
	fieldElements.MultiInvertEq(Inversions...)
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

	// TODO!

	/*

		// We catch panics due to division by zero here. The reason is that that the error message we would get by default is misleading:
		// Due to skipping the ones, the indices are wrong. Further, we want to distinguish NaPs from infinite points.
		defer func() {
			err := recover()
			if err != nil {
				switch err := err.(type) {
				case *errMultiInversionEncounteredZero:
					err.s += "\nNOTE: For this call site, indices in MultiInvertEq skip curve points with z==1"
					for i := 0; i < L; i++ {
						if points[i].IsNaP() {
							panic(fmt.Errorf("bandersnatch / curve Points: trying to bulk-convert multiple points to affine coordinates, out of which some were invalid: The first error was at index %v, which was a NAP", i))
						}
						if points[i].IsAtInfinity() {
							panic(fmt.Errorf("bandersnatch / curve Points: trying to bulk-convert multiple points to affine coordinates, out of which some were invalid: The first error was at index %v, which was a point at infinity", i))
						}
					}
				default:
					// This should not happen.
					panic(err)
				}

			}
		}()

	*/
	Inversions = Inversions[0 : L-numOnes]
	fieldElements.MultiInvertEq(Inversions...)
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
