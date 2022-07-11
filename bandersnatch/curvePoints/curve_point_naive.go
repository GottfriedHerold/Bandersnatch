package curvePoints

import "math/big"

/*
	This file contains naive implementations of various elliptic curve operations that are not actually used in production, but only serve
	to compare the actual implementation against for the purpose of testing correctness and debugging.
*/

// naive implementation using the affine definition. This is just used to test the other formulas against.
func (out *point_xtw_base) addNaive_ttt(input1, input2 *point_xtw_base) {
	var x1, y1, z1inv, x2, y2, z2inv FieldElement

	z1inv.Inv(&input1.z)
	z2inv.Inv(&input2.z)
	x1.Mul(&input1.x, &z1inv)
	y1.Mul(&input1.y, &z1inv)
	x2.Mul(&input2.x, &z2inv)
	y2.Mul(&input2.y, &z2inv)

	var denom_common FieldElement
	denom_common.Mul(&x1, &x2)
	denom_common.MulEq(&y1)
	denom_common.MulEq(&y2)
	denom_common.MulEq(&CurveParameterD_fe) // denom_common == dx1x2y1y2

	var denom_x, denom_y FieldElement
	denom_x.Add(&fieldElementOne, &denom_common) // denom_x = 1+dx1x2y1y2
	denom_y.Sub(&fieldElementOne, &denom_common) // denom_y = 1-dx1x2y1y2

	var numerator_x, numerator_y, temp FieldElement
	numerator_x.Mul(&x1, &y2)
	temp.Mul(&y1, &x2)
	numerator_x.AddEq(&temp) // x1y2+y1x2

	numerator_y.Mul(&x1, &x2)
	numerator_y.Multiply_by_five()
	temp.Mul(&y1, &y2)
	numerator_y.AddEq(&temp) // x1x2 + 5y1y2 = x1x2 - ax1x2

	out.t.Mul(&numerator_x, &numerator_y)
	out.z.Mul(&denom_x, &denom_y)
	out.x.Mul(&numerator_x, &denom_y)
	out.y.Mul(&numerator_y, &denom_x)
}

// exp_naive_xx(p, exponent) computes the exponentiation p^exponent (multiplicative notation) resp. p*exponent (additive notation) and stores the result in the receiver. exponent has type math.big.
// This functon uses a very simple squre-and-multiply method and is only used for either debugging or subgroup membership tests (because our faster methods implicitly assume the point to be in the subgroup for correctness)
// If using this function for inputs that might not be on the subgroup, be wary of the fact that the group law is not complete in that case.
func (out *point_xtw_base) exp_naive_xx(p *point_xtw_base, exponent *big.Int) {
	// simple square-and-multiply

	// We need to deal with the fact that exponent may be negative.
	var absexponent *big.Int       // to be set to abs(exponent)
	var to_add point_xtw_base = *p // to be set to +/- p
	if exponent.Sign() < 0 {
		absexponent = new(big.Int).Abs(exponent)
		to_add.neg_tt(p)
	} else {
		absexponent = new(big.Int).Set(exponent)
	}

	// actual square-and-multiply algorithm. We process bits from msb to lsb.
	bitlen := absexponent.BitLen()
	var accumulator point_xtw_base = neutralElement_xtwbase
	for i := bitlen - 1; i >= 0; i-- {
		accumulator.DoubleEq()
		if absexponent.Bit(i) == 1 {
			accumulator.add_ttt(&accumulator, &to_add)
		}
	}
	*out = accumulator
}
