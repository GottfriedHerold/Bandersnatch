package bandersnatch

import (
	"math/big"
)

// Data used to speed up exponentiation with the endomorphism:
// Consider the lattice L consisting of vectors (u,v), s.t. u*P + v*psi(P) = neutral element for any elliptic curve point P in the subgroup and phi the endomorphism.
// Because psi acts by multiplication by EndoEigenvalue==sqrt(-2) on the p253-subgroup, this is equivalent to u + v* EndoEigenvalue = 0 mod p253.
// Clearly, a basis for L is given by (p253,0) and (EndoEigenvalue, -1)
// We use psi to speed up arbitrary exponentiations by exponent t by noting that for P in the subgroup, t*P = a*P + b*psi(P), where (a,b) - (t,0) is in L.
// To find good, i.e. short (a,b), we need to solve a close(st) vector problem for the lattice L.
// Ideally, closest is for the infinity norm, but 2-norm would be good as well; we do not care about optimality too much anyway;
// While we actually solve it optimally, this is mostly because a) we easily can do so in dimension 2 and b) it makes testing a bit easier.

// LLL-reduced basis for lattice L (computed with SAGE) used in GLV reduction. The basis consists of the two vectors (lBasis_11, lBasis_12) and (lBasis_21, lBasis_22).

// The Voronoi cell wrt infinity-norm looks like in voronoi.svg. The 6 Voronoi-relevant vectors (colored lattice points in the figure) are given by +/- lBasis_1, +/- lBasis 2 and +/-(lBasis_1 + lBasis_2).
const (
	lBasis_11 = 113482231691339203864511368254957623327
	lBasis_12 = 10741319382058138887739339959866629956
	lBasis_21 = -21482638764116277775478679919733259912
	lBasis_22 = 113482231691339203864511368254957623327

	// Note: lBasis_11 == lBasis_22 and lBasis_21 = -2*lBasis_12. This special structure is due to EndoEigenvalue^2 == -2 mod p253:
	// For any (u,v) is in L, we have (-2v, u) in L, which is short (and a candidate for a vector of a reduced basis) if (u,v) is short.
	// Proof: Since u + v * \sqrt(2) = 0 mod p253, multiplying by \sqrt(2) gives \sqrt(2) * u - 2 v = 0 bmod p253, i.e .(-2v, u) is in L.

	lBasis_11_string = "113482231691339203864511368254957623327"
	lBasis_12_string = "10741319382058138887739339959866629956"
	lBasis_21_string = "-21482638764116277775478679919733259912"
	lBasis_22_string = "113482231691339203864511368254957623327"
)

var (
	lBasis_11_Int = initIntFromString(lBasis_11_string)
	lBasis_12_Int = initIntFromString(lBasis_12_string)
	lBasis_21_Int = initIntFromString(lBasis_21_string)
	lBasis_22_Int = initIntFromString(lBasis_22_string)
)

// (p253-1)/2. We can represent Z/p253 by numbers from -halfGroupOrder, ... , + halfGroupOrder.
const (
	halfGroupOrder        = (GroupOrder - 1) / 2
	halfGroupOrder_string = "6554484396890773809930967563523245729654577946720285125893201653364843836400"
)

var halfGroupOrder_Int = initIntFromString(halfGroupOrder_string)

// infty_norm computes the max of the absolute values of x and y.
func infty_norm(x, y *big.Int) (result *big.Int) {
	result = big.NewInt(0)
	if x.CmpAbs(y) > 0 { // |x| > |y|
		result.Abs(x)
	} else {
		result.Abs(y)
	}
	return
}

// TODO: Usage of big.Int may not be the best here.

// GLV_representation(t) outputs a pair u,v of big.Ints such that t*P = u*P + v*\Psi(P) for the endomorphism Psi for any P in the subgroup.
// We choose the pair u,v such that max{|u|, |v|} is minimized.
func GLV_representation(t *big.Int) (u_final *big.Int, v_final *big.Int) {
	// By the remark above, we essentially need to solve a closest vector problem here with target (t,0).
	// For this, we write (t,0) as alpha*lBasis_1 + beta*lBasis_2 with real-valued alpha, beta.
	// A close lattice point to (t,0) is then given by round(alpha)*lBasis_1 + round(beta)*lBasis_2 where round(.) rounds to the nearest integer.
	// The (preliminary, only near-optimal) result is then (t,0) - round(alpha)*lBasis_1 - round(beta)*lBasis_2
	// The latter is equal to (alpha-round(alpha)) * lBasis_1 + (beta-round(beta)) * lBasis_2

	// Now, note that (alpha, beta) = 1/det(B) * tilde(B) * (t,0) by definition, where the cofactor matrix tilde(B) = det(B)*B^{-1} is actually an integral matrix and B is the Basis matrix for lBasis_1,lBasis_2
	// By multipying everything with det(B) == p253, we can replace rounding floats to the nearest integer and taking the difference by rounding an integer to the next multiple of p253, i.e. working modulo p253.

	var delta_alpha *big.Int = big.NewInt(0) // p253 * (alpha - round(alpha))
	var delta_beta *big.Int = big.NewInt(0)  // p253 * (alpha - round())

	var u *big.Int = big.NewInt(0)
	var v *big.Int = big.NewInt(0)
	u_final = big.NewInt(0)
	v_final = big.NewInt(0)

	delta_alpha.Mul(t, lBasis_22_Int)                // First component of (t,0) * tilde(B)
	delta_alpha.Add(delta_alpha, halfGroupOrder_Int) // temporarily add (p253-1)/2. This is to transform rounding to truncating towards -infty (which is what big.Int's mod does).

	delta_beta.Mul(t, lBasis_12_Int)               // Second component of (t,0) * tilde(B) correct up to sign
	delta_beta.Sub(halfGroupOrder_Int, delta_beta) // temporarily add (p253-1)/2 and fix sign

	// take mod p253. The mod operation of big.Int results in numbers from 0 to p253-1 (even if some input is negative)
	delta_alpha.Mod(delta_alpha, GroupOrder_Int)
	delta_beta.Mod(delta_beta, GroupOrder_Int)

	// subtract (p253-1)/2 to undo the addition above. delta_alpha and delta_beta are now in the range -halfGroupOrder .. +halfGroupOrder
	delta_alpha.Sub(delta_alpha, halfGroupOrder_Int)
	delta_beta.Sub(delta_beta, halfGroupOrder_Int)

	// Multiply by 1/det B * B:
	var temp *big.Int = big.NewInt(0)
	u.Mul(lBasis_11_Int, delta_alpha)
	temp.Mul(lBasis_21_Int, delta_beta)
	u.Add(u, temp)

	v.Mul(lBasis_12_Int, delta_alpha)
	temp.Mul(lBasis_22_Int, delta_beta)
	v.Add(v, temp)

	u.Div(u, GroupOrder_Int) // Note: Division is exact.
	v.Div(v, GroupOrder_Int) // Note: Division is exact.

	// (u,v) already is a good solution. We can try to make (u,v) smaller by trying to add/subtract one of lBasis_1 or lBasis_2.
	// Due to the fact that the elementary cell associated to the basis B is contained in the union of the Voronoi cells around 0 and +/- lBasis_1 and +/- lBasis_2, this actually gives the global optimum.
	// Looking a voronoi.svg, we can actually use some sign information to limit the options we need to consider further.
	// NOTE: We constructed (u,v) using a naive Babai rounding rather than with Babai's nearest plane algorithm. The latter would have given a better (u,v) on average, but required more cases in post-processing to find the true
	// global optimum.

	// Note we look at (u,v) +/- lBasis_1 and (u,v) +/- lBasis_2. If we find a smaller vector, we do NOT greedily replace (u,v) and then try to improve further; this might acutally lead to a non-optimal solutions.
	// We know a priori that one of the 5 options (including (u,v) itself) starting from (u,v) is actually the global optimum.
	// NOTE: We do not really need to find the global optimum, but since we know the Voronoi relevant vectors, we can easily test for optimality. This is what we do in our tests, as it gives a clear and testable criterion.

	u_final.Set(u)
	v_final.Set(v)
	norm := infty_norm(u, v)
	var norm2 *big.Int
	if u.Sign() > 0 {
		delta_alpha.Sub(u, lBasis_11_Int)
		delta_beta.Sub(v, lBasis_12_Int)
		norm2 = infty_norm(delta_alpha, delta_beta)
		if norm2.CmpAbs(norm) < 0 {
			u_final.Set(delta_alpha)
			v_final.Set(delta_beta)
			norm.Set(norm2)
		}
	} else {
		delta_alpha.Add(u, lBasis_11_Int)
		delta_beta.Add(v, lBasis_12_Int)
		norm2 = infty_norm(delta_alpha, delta_beta)
		if norm2.CmpAbs(norm) < 0 {
			u_final.Set(delta_alpha)
			v_final.Set(delta_beta)
			norm.Set(norm2)
		}
	}

	if v.Sign() > 0 {
		delta_alpha.Sub(u, lBasis_21_Int)
		delta_beta.Sub(v, lBasis_22_Int)
		norm2 = infty_norm(delta_alpha, delta_beta)
		if norm2.CmpAbs(norm) < 0 {
			u_final.Set(delta_alpha)
			v_final.Set(delta_beta)
			norm.Set(norm2)
		}
	} else {
		delta_alpha.Add(u, lBasis_21_Int)
		delta_beta.Add(v, lBasis_22_Int)
		norm2 = infty_norm(delta_alpha, delta_beta)
		if norm2.CmpAbs(norm) < 0 {
			u_final.Set(delta_alpha)
			v_final.Set(delta_beta)
			norm.Set(norm2)
		}
	}
	return
}

type decompositionCoefficient struct {
	position uint
	coeff    uint
	sign     int
}

// decomposeUnalignedSignedAdic_Int(input, maxbits) outputs a list of exponents e_i and coeffs c_i of the same length s.t.
// a) input = \sum_i c_i * 2^{e_i} for the original value of input
// b) the e_i are ascending (this might change)
// c) All c_i are odd with |c_i| having at most maxbits bits. Note that both input and the c_i carry signs.
// The function is allowed to write to input. If the caller needs to re-use input, make a copy first.
func decomposeUnalignedSignedAdic_Int(input *big.Int, maxbits int) (decomposition []decompositionCoefficient) {
	var globalSign int = input.Sign() // big.Int internally stores sign bit + Abs(input). We only read the latter, so we need to correct the sign. globalSign is in {-1,0,+1}
	inputBitLen := input.BitLen()     // bitlength of Abs(input)
	// 1 + inputBitLen / maxbits is a reasonable estimate for the capacity (it is in fact a upper bound, but just need an estimate)
	decomposition = make([]decompositionCoefficient, 0, 1+inputBitLen/maxbits)
	// exponents = make([]uint, 0, 1+inputBitLen/maxbits)
	// coeffs = make([]int, 0, 1+inputBitLen/maxbits)
	var carry uint // bool? uint?
	// Scan input bits from lsb to msb
	var i int
	for i = 0; i < inputBitLen; { // increment of i done inside loop, as the stride is variable
		if input.Bit(i) == carry {
			i++
			continue
		}
		v := getBitRange(input, i, i+maxbits)
		v += carry
		if v%2 == 0 {
			panic("Cannot happen")
		}
		carry = input.Bit(i + maxbits)
		if carry == 1 {
			// change v to v - (2 << maxbits).
			decomposition = append(decomposition, decompositionCoefficient{position: uint(i), coeff: (1 << maxbits) - v, sign: -globalSign})
		} else {
			decomposition = append(decomposition, decompositionCoefficient{position: uint(i), coeff: v, sign: globalSign})
		}
		i += maxbits + 1 // Note: The +1 comes from the sign ambiguity
	}
	if carry == 1 {
		decomposition = append(decomposition, decompositionCoefficient{position: uint(i), coeff: 1, sign: globalSign})
	}
	return
}

// getBitRange(x, low, high) interprets Abs(x) as a slice of bits in low-endian order and retuns the value of x[low:high], interpreted as a (usual) int.
// We only require this to be correct if low <= high and high - low <= 8, say (not sure what bound we need)
func getBitRange(input *big.Int, lowend int, highend int) uint {
	// naive implementation:
	var result uint = 0
	for shift := 0; shift < highend-lowend; shift++ {
		result += input.Bit(shift+lowend) << shift
	}
	return result
}
