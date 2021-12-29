package bandersnatch

import "math/big"

// exp_naive_xx(p, exponent) computes the exponentiation p^exponent (multiplicative notation) resp. p*exponent (additive notation) and stores the result in the receiver. exponent has type math.big.
// This functon uses a very simple squre-and-multiply method and is only used for either debugging or subgroup membership tests (because our faster methods implicitly assume the point to be in the subgroup for correctness)
// If using this function for inputs that might not be on the subgroup, be wary of the fact that the group law is not complete in that case.
func (out *Point_xtw) exp_naive_xx(p *Point_xtw, exponent *big.Int) {
	// simple square-and-multiply

	// We need to deal with the fact that exponent may be negative.
	var absexponent *big.Int  // to be set to abs(exponent)
	var to_add Point_xtw = *p // to be set to +/- p
	if exponent.Sign() < 0 {
		absexponent = new(big.Int).Abs(exponent)
		to_add.neg_tt(p)
	} else {
		absexponent = new(big.Int).Set(exponent)
	}

	// actual square-and-multiply algorithm. We process bits from msb to lsb.
	bitlen := absexponent.BitLen()
	var accumulator Point_xtw = NeutralElement_xtw
	for i := bitlen - 1; i >= 0; i-- {
		accumulator.double_tt(&accumulator)
		if absexponent.Bit(i) == 1 {
			accumulator.add_ttt(&accumulator, &to_add)
		}
	}
	*out = accumulator
}
