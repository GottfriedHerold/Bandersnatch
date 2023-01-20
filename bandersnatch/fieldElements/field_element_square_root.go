package fieldElements

// This file is part of the fieldElements package. See the documentation of field_element.go for general remarks.

// This file contains an efficient square root algorithms, which involves a considerable amount of pre-computed constants.

// The algorithm used is essentiall Tonelli-Shanks.
// Note that BaseFieldSize mod 4 == 1, with a large power of 2, namely 2^32 dividing BaseFieldSize-1 (BaseField2Adicity==32).
// This means that plain Tonelli-Shanks would first power the argument to the odd part of p+1 and then essentially solve a dlog problem
// bit-by-bit in a group of size 2^32. The latter would take about 32^2/2 multiplications and would dominate the running time.
// To improve that, we precompute a lookup table of dlogs of a subgroup of order 2^sqrtParam_Blocksize == 256. Then
// we can retrieve sqrtParam_BlockSize == 8 bits at a time for the dlog problem, reducing the complexity by a lot.

// feType_SquareRoot is the field element type used internally during square root computations.
// We reserve the option to change this type (We use more squarings than general multiplications and that might be better with a different type), hence the alias.
// The actual square root algorithm used for the exported field element types is supposed to convert to feType_SquareRoot, use the algorithms here and convert back.
// The conversion is much faster than the actual algorithm anyway.
// Note that the algorithm kind-of depends on that type, since we use a lookup-table feType_SquareRoot -> exponent to compute (small) dlogs of roots of unity.
// This lookup works by directly looking at (a hash of) the internal representation; no need for Montgomery conversions and such.
type feType_SquareRoot = bsFieldElement_MontgomeryNonUnique

// The implementation of invSqrtEqDyadic relies on a large set of precomputed values

// These parameters guide efficiency / precomputation - tradeoff. We make them const to avoid allocations with make(...) outside of global initialization.
const (
	// Note: The BlockSize parameter constrols the tradeoff.
	// Adjusting this parameter may require to adjust sqrtAlg_NegDlogInSmallDyadicSubgroup, as we need a collision-free (non-cryptographic) hash function for the
	// appropriate roots of unity. This is checked unconditionally on startup and we panic during initialization of global variables on failure (i.e. if there is a collision).
	// Note that the functions in this file have been successfully tested for different values of sqrtParam_BlockSize.
	sqrtParam_BlockSize            = 8                                                                     // SquareRoot computation involves a dlog computation for 2^32th roots of unity. We retrieve this in blocks with this many bits via a lookup-table.
	sqrtParam_TotalBits            = BaseField2Adicity                                                     // == 32, total number of bits for the dyadic part of the base field computation. The field has 2^32th roots of unity.
	sqrtParam_Blocks               = (sqrtParam_TotalBits + sqrtParam_BlockSize - 1) / sqrtParam_BlockSize // Number of blocks needed
	sqrtParam_HaveNonFullBlock     = (sqrtParam_TotalBits%sqrtParam_BlockSize != 0)                        // Set to true if we have block with fewer bits
	sqrtParam_FirstBlockUnusedBits = sqrtParam_Blocks*sqrtParam_BlockSize - sqrtParam_TotalBits            // number of unused bits in the first reconstructed block.
	sqrtParam_BitMask              = (1 << sqrtParam_BlockSize) - 1                                        // bitmask to pick up the last sqrtParam_BlockSize bits.
)

// NOTE: These "variables" are actually pre-computed constants that must not change.

// sqrtPrecomp_PrimitiveDyadicRoots[i] equals DyadicRootOfUnity^(2^i) for 0 <= i <= 32
//
// This means that it is a 32-i'th primitive root of unitity, obtained by repeatedly squaring a 2^32th primitive root of unity [DyadicRootOfUnity_fe].
var sqrtPrecomp_PrimitiveDyadicRoots [BaseField2Adicity + 1]feType_SquareRoot = func() (ret [BaseField2Adicity + 1]feType_SquareRoot) {
	ret[0] = dyadicRootOfUnity_fe
	ret[0].Normalize()
	for i := 1; i <= BaseField2Adicity; i++ { // Note <= here
		ret[i].Square(&ret[i-1])
		ret[i].Normalize()
	}
	// 31th one must be -1. We check that here.
	x, ok := ret[BaseField2Adicity-1].ToInt64()
	if ok != nil || x != -1 {
		panic(ErrorPrefix + "something is wrong with the dyadic roots of unity")
	}
	return
}() // immediately invoked lambda

// sqrtPrecomp_PrecomputedBlocks[i][j] == g^(j << (i* BlockSize)), where g is the fixed primitive 2^32th root of unity.
// This means that the exponent is equal to 0x00000...0000jjjjjj0000....0000, where only the i'th least significant block of size BlockSize is set
// and that value is j.
//
// Note: accessed through sqrtAlg_getPrecomputedRootOfUnity
var sqrtPrecomp_PrecomputedBlocks [sqrtParam_Blocks][1 << sqrtParam_BlockSize]feType_SquareRoot = func() (blocks [sqrtParam_Blocks][1 << sqrtParam_BlockSize]feType_SquareRoot) {
	for i := 0; i < sqrtParam_Blocks; i++ {
		blocks[i][0].SetOne()
		for j := 1; j < (1 << sqrtParam_BlockSize); j++ {
			blocks[i][j].Mul(&blocks[i][j-1], &sqrtPrecomp_PrimitiveDyadicRoots[i*sqrtParam_BlockSize])
			blocks[i][j].Normalize()
		}
	}
	return
}() // immediately invoked lambda

// sqrtPrecomp_ReconstructionDyadicRoot is a primitive 2^BlockSize'th root of unity.
//
// We allow computing dlog wrt. this base (with 2^sqrtParam_BlockSize many possible values) via a look-up-table.
var sqrtPrecomp_ReconstructionDyadicRoot = sqrtPrecomp_PrimitiveDyadicRoots[BaseField2Adicity-sqrtParam_BlockSize] // primitive root of unity of order 2^sqrtParam_BlockSize

// sqrtAlg_OrderAsDyadicRootOfUnity returns i, s.t. z is a primitive 2^i'th root of unity.
//
// It panics on 0; if z!=0 is not a primitive 2^n'th root of unity for any n, the answer is unspecified.
//
// This method is only used for testing.
func sqrtAlg_OrderAsDyadicRootOfUnity[FE any, FEPtr interface {
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
	return -1 // not a root of unity at all
}

// These two functions compute certain powers of a given number. We use hand-optimized "addition chains" (should be called multiplication chains in this context)
// for those.
// Using a single function for makePowersForSquareRoot is done such that the computations may share intermediate results.

// *** NOTE: ExpOddOrder modifies the receiver and takes the input as argument. computeRelevantPowers takes the input as receiver and modifies the arguments. ***
// This is a bit weird, but it's an internal function anyway.

// sqrtAlg_ComputeRelevantPowers computes certain powers of z.
//
// z.sqrtAlg_ComputeRelevantPowers(&x,&y)
// sets x := z^tonelliShanksExponent, y := z^BaseFieldMultiplicativeOddOrder. z itself is unchanged.
//
// We assume that x,y,z do not alias.
func (z *feType_SquareRoot) sqrtAlg_ComputeRelevantPowers(squareRootCandidate *feType_SquareRoot, rootOfUnity *feType_SquareRoot) {
	SquareEqNTimes := func(z *feType_SquareRoot, n int) {
		for i := 0; i < n; i++ {
			z.SquareEq()
		}
	}

	// hand-crafted sliding window-type algorithm with window-size 5
	// Note that we precompute and use z^255 multiple times (even though it's not size 5)
	// and some windows actually overlap(!)

	var z2, z3, z7, z6, z9, z11, z13, z19, z21, z25, z27, z29, z31, z255 feType_SquareRoot
	var acc feType_SquareRoot
	z2.Square(z)             // 0b10
	z3.Mul(z, &z2)           // 0b11
	z6.Square(&z3)           // 0b110
	z7.Mul(z, &z6)           // 0b111
	z9.Mul(&z7, &z2)         // 0b1001
	z11.Mul(&z9, &z2)        // 0b1011
	z13.Mul(&z11, &z2)       // 0b1101
	z19.Mul(&z13, &z6)       // 0b10011
	z21.Mul(&z2, &z19)       // 0b10101
	z25.Mul(&z19, &z6)       // 0b11001
	z27.Mul(&z25, &z2)       // 0b11011
	z29.Mul(&z27, &z2)       // 0b11101
	z31.Mul(&z29, &z2)       // 0b11111
	acc.Mul(&z27, &z29)      // 56
	acc.SquareEq()           // 112
	acc.SquareEq()           // 224
	z255.Mul(&acc, &z31)     // 0b11111111 = 255
	acc.SquareEq()           // 448
	acc.SquareEq()           // 896
	acc.MulEq(&z31)          // 0b1110011111 = 927
	SquareEqNTimes(&acc, 6)  // 0b1110011111000000
	acc.MulEq(&z27)          // 0b1110011111011011
	SquareEqNTimes(&acc, 6)  // 0b1110011111011011000000
	acc.MulEq(&z19)          // 0b1110011111011011010011
	SquareEqNTimes(&acc, 5)  // 0b111001111101101101001100000
	acc.MulEq(&z21)          // 0b111001111101101101001110101
	SquareEqNTimes(&acc, 7)  // 0b1110011111011011010011101010000000
	acc.MulEq(&z25)          // 0b1110011111011011010011101010011001
	SquareEqNTimes(&acc, 6)  // 0b1110011111011011010011101010011001000000
	acc.MulEq(&z19)          // 0b1110011111011011010011101010011001010011
	SquareEqNTimes(&acc, 5)  // 0b111001111101101101001110101001100101001100000
	acc.MulEq(&z7)           // 0b111001111101101101001110101001100101001100111
	SquareEqNTimes(&acc, 5)  // 0b11100111110110110100111010100110010100110011100000
	acc.MulEq(&z11)          // 0b11100111110110110100111010100110010100110011101011
	SquareEqNTimes(&acc, 5)  // 0b1110011111011011010011101010011001010011001110101100000
	acc.MulEq(&z29)          // 0b1110011111011011010011101010011001010011001110101111101
	SquareEqNTimes(&acc, 5)  // 0b111001111101101101001110101001100101001100111010111110100000
	acc.MulEq(&z9)           // 0b111001111101101101001110101001100101001100111010111110101001
	SquareEqNTimes(&acc, 7)  // 0b1110011111011011010011101010011001010011001110101111101010010000000
	acc.MulEq(&z3)           // 0b1110011111011011010011101010011001010011001110101111101010010000011
	SquareEqNTimes(&acc, 7)  // 0b11100111110110110100111010100110010100110011101011111010100100000110000000
	acc.MulEq(&z25)          // 0b11100111110110110100111010100110010100110011101011111010100100000110011001
	SquareEqNTimes(&acc, 5)  // 0b1110011111011011010011101010011001010011001110101111101010010000011001100100000
	acc.MulEq(&z25)          // 0b1110011111011011010011101010011001010011001110101111101010010000011001100111001
	SquareEqNTimes(&acc, 5)  // 0b111001111101101101001110101001100101001100111010111110101001000001100110011100100000
	acc.MulEq(&z27)          // 0b111001111101101101001110101001100101001100111010111110101001000001100110011100111011
	SquareEqNTimes(&acc, 8)  // 0b11100111110110110100111010100110010100110011101011111010100100000110011001110011101100000000
	acc.MulEq(z)             // 0b11100111110110110100111010100110010100110011101011111010100100000110011001110011101100000001
	SquareEqNTimes(&acc, 8)  // 0b1110011111011011010011101010011001010011001110101111101010010000011001100111001110110000000100000000
	acc.MulEq(z)             // 0b1110011111011011010011101010011001010011001110101111101010010000011001100111001110110000000100000001
	SquareEqNTimes(&acc, 6)  // 0b1110011111011011010011101010011001010011001110101111101010010000011001100111001110110000000100000001000000
	acc.MulEq(&z13)          // 0b1110011111011011010011101010011001010011001110101111101010010000011001100111001110110000000100000001001101
	SquareEqNTimes(&acc, 7)  // 0b11100111110110110100111010100110010100110011101011111010100100000110011001110011101100000001000000010011010000000
	acc.MulEq(&z7)           // 0b11100111110110110100111010100110010100110011101011111010100100000110011001110011101100000001000000010011010000111
	SquareEqNTimes(&acc, 3)  // 0b11100111110110110100111010100110010100110011101011111010100100000110011001110011101100000001000000010011010000111000
	acc.MulEq(&z3)           // 0b11100111110110110100111010100110010100110011101011111010100100000110011001110011101100000001000000010011010000111011
	SquareEqNTimes(&acc, 13) // 0b111001111101101101001110101001100101001100111010111110101001000001100110011100111011000000010000000100110100001110110000000000000
	acc.MulEq(&z21)          // 0b111001111101101101001110101001100101001100111010111110101001000001100110011100111011000000010000000100110100001110110000000010101
	SquareEqNTimes(&acc, 5)  // 0b11100111110110110100111010100110010100110011101011111010100100000110011001110011101100000001000000010011010000111011000000001010100000
	acc.MulEq(&z9)           // 0b11100111110110110100111010100110010100110011101011111010100100000110011001110011101100000001000000010011010000111011000000001010101001
	SquareEqNTimes(&acc, 5)  // 0b1110011111011011010011101010011001010011001110101111101010010000011001100111001110110000000100000001001101000011101100000000101010100100000
	acc.MulEq(&z27)          // 0b1110011111011011010011101010011001010011001110101111101010010000011001100111001110110000000100000001001101000011101100000000101010100111011
	SquareEqNTimes(&acc, 5)  // 0b111001111101101101001110101001100101001100111010111110101001000001100110011100111011000000010000000100110100001110110000000010101010011101100000
	acc.MulEq(&z27)          // 0b111001111101101101001110101001100101001100111010111110101001000001100110011100111011000000010000000100110100001110110000000010101010011101111011
	SquareEqNTimes(&acc, 5)  // 0b11100111110110110100111010100110010100110011101011111010100100000110011001110011101100000001000000010011010000111011000000001010101001110111101100000
	acc.MulEq(&z9)           // 0b11100111110110110100111010100110010100110011101011111010100100000110011001110011101100000001000000010011010000111011000000001010101001110111101101001
	SquareEqNTimes(&acc, 10) // 0b111001111101101101001110101001100101001100111010111110101001000001100110011100111011000000010000000100110100001110110000000010101010011101111011010010000000000
	acc.MulEq(z)             // 0b111001111101101101001110101001100101001100111010111110101001000001100110011100111011000000010000000100110100001110110000000010101010011101111011010010000000001
	SquareEqNTimes(&acc, 7)  // 0b1110011111011011010011101010011001010011001110101111101010010000011001100111001110110000000100000001001101000011101100000000101010100111011110110100100000000010000000
	acc.MulEq(&z255)         // 0b1110011111011011010011101010011001010011001110101111101010010000011001100111001110110000000100000001001101000011101100000000101010100111011110110100100000000101111111
	SquareEqNTimes(&acc, 8)  // 0b111001111101101101001110101001100101001100111010111110101001000001100110011100111011000000010000000100110100001110110000000010101010011101111011010010000000010111111100000000
	acc.MulEq(&z255)         // 0b111001111101101101001110101001100101001100111010111110101001000001100110011100111011000000010000000100110100001110110000000010101010011101111011010010000000010111111111111111
	SquareEqNTimes(&acc, 6)  // 0b111001111101101101001110101001100101001100111010111110101001000001100110011100111011000000010000000100110100001110110000000010101010011101111011010010000000010111111111111111000000
	acc.MulEq(&z11)          // 0b111001111101101101001110101001100101001100111010111110101001000001100110011100111011000000010000000100110100001110110000000010101010011101111011010010000000010111111111111111001011
	SquareEqNTimes(&acc, 9)  // 0b111001111101101101001110101001100101001100111010111110101001000001100110011100111011000000010000000100110100001110110000000010101010011101111011010010000000010111111111111111001011000000000
	acc.MulEq(&z255)         // 0b111001111101101101001110101001100101001100111010111110101001000001100110011100111011000000010000000100110100001110110000000010101010011101111011010010000000010111111111111111001011011111111
	SquareEqNTimes(&acc, 2)  // 0b11100111110110110100111010100110010100110011101011111010100100000110011001110011101100000001000000010011010000111011000000001010101001110111101101001000000001011111111111111100101101111111100
	acc.MulEq(z)             // 0b11100111110110110100111010100110010100110011101011111010100100000110011001110011101100000001000000010011010000111011000000001010101001110111101101001000000001011111111111111100101101111111101
	SquareEqNTimes(&acc, 7)  // 0b111001111101101101001110101001100101001100111010111110101001000001100110011100111011000000010000000100110100001110110000000010101010011101111011010010000000010111111111111111001011011111111010000000
	acc.MulEq(&z255)         // 0b111001111101101101001110101001100101001100111010111110101001000001100110011100111011000000010000000100110100001110110000000010101010011101111011010010000000010111111111111111001011011111111101111111
	SquareEqNTimes(&acc, 8)  // 0b11100111110110110100111010100110010100110011101011111010100100000110011001110011101100000001000000010011010000111011000000001010101001110111101101001000000001011111111111111100101101111111110111111100000000
	acc.MulEq(&z255)         // 0b11100111110110110100111010100110010100110011101011111010100100000110011001110011101100000001000000010011010000111011000000001010101001110111101101001000000001011111111111111100101101111111110111111111111111
	SquareEqNTimes(&acc, 8)  // 0b1110011111011011010011101010011001010011001110101111101010010000011001100111001110110000000100000001001101000011101100000000101010100111011110110100100000000101111111111111110010110111111111011111111111111100000000
	acc.MulEq(&z255)         // 0b1110011111011011010011101010011001010011001110101111101010010000011001100111001110110000000100000001001101000011101100000000101010100111011110110100100000000101111111111111110010110111111111011111111111111111111111
	SquareEqNTimes(&acc, 8)  // 0b111001111101101101001110101001100101001100111010111110101001000001100110011100111011000000010000000100110100001110110000000010101010011101111011010010000000010111111111111111001011011111111101111111111111111111111100000000
	acc.MulEq(&z255)         // 0b111001111101101101001110101001100101001100111010111110101001000001100110011100111011000000010000000100110100001110110000000010101010011101111011010010000000010111111111111111001011011111111101111111111111111111111111111111
	// acc is now z^((BaseFieldMultiplicativeOddOrder - 1)/2)
	rootOfUnity.Square(&acc)         // BaseFieldMultiplicativeOddOrder - 1
	rootOfUnity.MulEq(z)             // BaseFieldMultiplicativeOddOrder
	squareRootCandidate.Mul(&acc, z) // (BaseFieldMultiplicativeOddOrder + 1)/2
}

// sqrtAlg_ExpOddOrder sets z := input^BaseFieldMultiplicativeOddOrder.
//
// If input is non-zero, the resulting z will be a (primitive iff input is a non-square) 2^32th root of unity.
//
// This function might be removed
func (z *feType_SquareRoot) sqrtAlg_ExpOddOrder(input *feType_SquareRoot) {
	z.Exp(input, &BaseFieldMultiplicateOddOrder_uint256)
}

// sqrtAlg_GetPrecomputedRootOfUnity sets target to g^(multiplier << (order * sqrtParam_BlockSize)), where g is the fixed primitive 2^32th root of unity.
//
// We assume that order 0 <= order*sqrtParam_BlockSize <= 32 and that multiplier is in [0, 1 <<sqrtParam_BlockSize)
func sqrtAlg_GetPrecomputedRootOfUnity(target *feType_SquareRoot, multiplier int, order uint) {
	*target = sqrtPrecomp_PrecomputedBlocks[order][multiplier]

}

// sqrtAlg_NegDlogInSmallDyadicSubgroup takes a (not neccessarily primitive) root of unity x of order 2^sqrtParam_BlockSize.
// x has the form sqrtPrecomp_ReconstructionDyadicRoot^a and returns its negative dlog -a.
//
// The returned value is only meaningful modulo 1<<sqrtParam_BlockSize and is fully reduced, i.e. in [0, 1<<sqrtParam_BlockSize )
//
// NOTE: If x is not a root of unity as asserted, the behaviour is undefined.
func sqrtAlg_NegDlogInSmallDyadicSubgroup(x *feType_SquareRoot) uint {
	x.Normalize()
	return sqrtPrecomp_dlogLUT[uint16(x.words[0]&0xFFFF)]
}

// sqrtPrecomp_dlogLUT is a lookup table used to implement the map sqrtPrecompt_reconstructionDyadicRoot^a -> -a
var sqrtPrecomp_dlogLUT map[uint16]uint = func() (ret map[uint16]uint) {
	const LUTSize = 1 << sqrtParam_BlockSize // 256
	ret = make(map[uint16]uint, LUTSize)

	var rootOfUnity feType_SquareRoot
	rootOfUnity.SetOne()
	for i := 0; i < LUTSize; i++ {
		const mask = LUTSize - 1
		// the LUTSize many roots of unity all (by chance) have distinct values for .words[0]&0xFFFF. Note that this uses the Montgomery representation.
		ret[uint16(rootOfUnity.words[0]&0xFFFF)] = uint((-i) & mask)
		rootOfUnity.MulEq(&sqrtPrecomp_ReconstructionDyadicRoot)
	}
	// This effectively checks the above claim (that .words[0]&0xFFFF is distinct).
	// Note that this might fail if we adjust the sqrtParam_BlockSize parameter and this check will alert us.
	if len(ret) != LUTSize {
		panic(ErrorPrefix + "failed to store all appropriate roots of unity in a map")
	}
	return
}() // immediately invoked lambda

// invSqrtEqDyadic asserts that z is a 2^32 root of unity and tries to set z := 1/sqrt(z).
//
// If z is actually a 2^32th *primitive* root of unity, the square root does not exist and we return false without modifying z.
// Otherwise, z is changed to 1/sqrt(z) and we return true
func (z *feType_SquareRoot) invSqrtEqDyadic() (ok bool) {

	// The algorithm works by essentially computing the dlog of z and then halving it.

	// negExponent is intended to hold the negative of the dlog of z.
	// We determine this 32-bit value (usually) _sqrtBlockSize many bits at a time, starting with the least-significant bits.
	//
	// If _sqrtBlockSize does not divide 32, the *first* iteration will determine fewer bits.
	var negExponent uint

	var temp, temp2 feType_SquareRoot

	// set powers[i] to z^(1<< (i*blocksize))
	var powers [sqrtParam_Blocks]feType_SquareRoot
	powers[0] = *z
	for i := 1; i < sqrtParam_Blocks; i++ {
		powers[i] = powers[i-1]
		for j := 0; j < sqrtParam_BlockSize; j++ {
			powers[i].SquareEq()
		}
	}

	// looking at the dlogs, powers[i] is essentially the wanted exponent, left-shifted by i*_sqrtBlockSize and taken mod 1<<32
	// dlogHighDyadicRootNeg essentially (up to sign) reads off the _sqrtBlockSize many most significant bits. (returned as low-order bits)

	// first iteration may be slightly special if BlockSize does not divide 32
	negExponent = sqrtAlg_NegDlogInSmallDyadicSubgroup(&powers[sqrtParam_Blocks-1])
	negExponent >>= sqrtParam_FirstBlockUnusedBits

	// if the exponent we just got is odd, there is no square root, no point in determining the other bits.
	if negExponent&1 == 1 {
		return false
	}

	// Get remaining bits
	for i := 1; i < sqrtParam_Blocks; i++ {
		temp2 = powers[sqrtParam_Blocks-1-i]
		// We essentially un-set the bits we already know from powers[_sqrtNumBlocks-1-i]
		for j := 0; j < i; j++ {
			sqrtAlg_GetPrecomputedRootOfUnity(&temp, int((negExponent>>(j*sqrtParam_BlockSize))&sqrtParam_BitMask), uint(j+sqrtParam_Blocks-1-i))
			temp2.MulEq(&temp)
		}
		newBits := sqrtAlg_NegDlogInSmallDyadicSubgroup(&temp2)
		negExponent |= newBits << (sqrtParam_BlockSize*i - sqrtParam_FirstBlockUnusedBits)
	}

	// var tmp _FESquareRoot

	// negExponent is now the negative dlog of z.

	// Take the square root
	negExponent >>= 1
	// Write to z:
	z.SetOne()
	for i := 0; i < sqrtParam_Blocks; i++ {
		sqrtAlg_GetPrecomputedRootOfUnity(&temp, int((negExponent>>(i*sqrtParam_BlockSize))&sqrtParam_BitMask), uint(i))
		z.MulEq(&temp)
	}
	return true
}
