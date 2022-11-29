package fieldElements

// This file contains efficient square root algorithms, which involves a considerable amount of pre-computed constants.

// type to hold all the precomputed data. We reserve the option to change this type (squaring might be better with a different type), hence the alias.
type _FESquareRoot = bsFieldElement_MontgomeryNonUnique

// getDyadicPower returns i, s.t. z is a primitive 2^i'th root of unity. It panics on 0; if z!=0 is not a primitive 2^n'th root of unity for any n, returns -1.
// This method is only used for testing.
func getDyadicPower[FE any, FEPtr interface {
	*FE
	FieldElementInterface[*FE]
}](z FEPtr) int {
	if z.IsZero() {
		panic(ErrorPrefix + "getDyadicPower called on 0")
	}
	x := *z
	for i := 0; i < BaseField2Adicity; i++ {
		// x == z^(2^i)
		if FEPtr(&x).IsOne() {
			return i
		}
		FEPtr(&x).SquareEq()
		// x == z^(2^(i+1))
	}
	if FEPtr(&x).IsOne() {
		return BaseField2Adicity
	}
	return -1
}

// These two functions compute powers of a given number. We probably want to use optimized "addition" chains (should be called multiplication chains in this context)
// for those. Using a single function for makePowersForSquareRoot is done such that the computations may share intermediate results.
// We use a dummy implementation for now.

// makePowersForSquareRoot computes certain powers of z.
//
// z.makePowersForSquareRoot(&x,&y)
// sets x := z^tonelliShanksExponent, y := z^BaseFieldMultiplicativeOddOrder. z itself is unchanged.
//
// We assume that x,y,z do not alias.
func (z *_FESquareRoot) makePowersForSquareRoot(squareRootCandidate *_FESquareRoot, rootOfUnity *_FESquareRoot) {
	// TODO: Optimize this!
	// var inv _FESquareRoot
	// inv.Inv(z)
	squareRootCandidate.Exp(z, &tonelliShanksExponent_uint256)
	rootOfUnity.Exp(z, &BaseFieldMultiplicateOddOrder_uint256)
}

// exponentiateToDyadicRootOfUnity sets z := x^BaseFieldMultiplicativeOddOrder. If x is non-zero, the resulting z will be a (primitive iff x is a non-square) 2^32th root of unity.
func (z *_FESquareRoot) exponentiateToDyadicRootOfUnity(x *_FESquareRoot) {
	z.Exp(x, &BaseFieldMultiplicateOddOrder_uint256)
}

// invSqrtEqDyadic asserts that z is a 2^32 root of unity and tries to set z := 1/sqrt(z).
//
// If z is actually a 2^32th *primitive* root of unity, the square root does not exist and we return false. (What we do with z is unspecified)
// Otherwise, z is changed to 1/sqrt(z) and we return true
func (z *_FESquareRoot) invSqrtEqDyadic() (ok bool) {
	panic(0)
}
