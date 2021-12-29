package bandersnatch

import (
	"math/rand"
)

/*
	This file contains naive implementations of various elliptic curve operations that are not actually used in production, but only serve
	to compare the actual implementation against for the purpose of testing correctness and debugging.
*/

// naive implementation using the affine definition. This is just used to test the other formulas against.
func (out *Point_xtw) addNaive_ttt(input1, input2 *Point_xtw) {
	var x1, y1, z1inv, x2, y2, z2inv bsFieldElement_64

	z1inv.Inv(&input1.z)
	z2inv.Inv(&input2.z)
	x1.Mul(&input1.x, &z1inv)
	y1.Mul(&input1.y, &z1inv)
	x2.Mul(&input2.x, &z2inv)
	y2.Mul(&input2.y, &z2inv)

	var denom_common bsFieldElement_64
	denom_common.Mul(&x1, &x2)
	denom_common.MulEq(&y1)
	denom_common.MulEq(&y2)
	denom_common.MulEq(&CurveParameterD_fe) // denom_common == dx1x2y1y2

	var denom_x, denom_y bsFieldElement_64
	denom_x.Add(&bsFieldElement_64_one, &denom_common) // denom_x = 1+dx1x2y1y2
	denom_y.Sub(&bsFieldElement_64_one, &denom_common) // denom_y = 1-dx1x2y1y2

	var numerator_x, numerator_y, temp bsFieldElement_64
	numerator_x.Mul(&x1, &y2)
	temp.Mul(&y1, &x2)
	numerator_x.AddEq(&temp) // x1y2+y1x2

	numerator_y.Mul(&x1, &x2)
	numerator_y.multiply_by_five()
	temp.Mul(&y1, &y2)
	numerator_y.AddEq(&temp) // x1x2 + 5y1y2 = x1x2 - ax1x2

	out.t.Mul(&numerator_x, &numerator_y)
	out.z.Mul(&denom_x, &denom_y)
	out.x.Mul(&numerator_x, &denom_y)
	out.y.Mul(&numerator_y, &denom_x)
}

// Creates a random point on the curve, which does not neccessarily need to be in the correct subgroup.
func makeRandomPointOnCurve_t(rnd *rand.Rand) Point_xtw {

	var x, y, t, z bsFieldElement_64

	// Set x randomly, compute y from x
	for {
		x.setRandomUnsafe(rnd)
		// x.SetUInt64(1)
		var err error
		y, err = recoverYFromXAffine(&x, false)
		if err == nil {
			break
		}
	}

	// Set t = x*y. If we would set z to 1, this is now a correct point.
	t.Mul(&x, &y)

	// As this is only used for debugging, we set z randomly and scale the point.
	z.setRandomUnsafe(rnd)
	if z.IsZero() { // This should only happens with negligle proability anyway.
		z.SetOne()
	}
	x.MulEq(&z)
	y.MulEq(&z)
	t.MulEq(&z)

	return Point_xtw{x: x, y: y, z: z, t: t}
}

// Creates a random point on the correct subgroup
func makeRandomPointInSubgroup_t(rnd *rand.Rand) Point_xtw {
	r := makeRandomPointOnCurve_t(rnd)
	// r.clearCofactor2()
	r.DoubleEq()
	return r
}
