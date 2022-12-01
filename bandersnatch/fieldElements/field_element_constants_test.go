package fieldElements

import (
	"math/big"
	"testing"

	"github.com/GottfriedHerold/Bandersnatch/bandersnatch/common"
	"github.com/GottfriedHerold/Bandersnatch/internal/testutils"
	"github.com/GottfriedHerold/Bandersnatch/internal/utils"
)

// We make copies of all our (exported and internal) "constant" arrays/structs etc.
// Since Go lacks const-arrays or structs, these may theoretically be modified (for constants of pointer type such *big.Int, we need to check the pointed-to value)
//

var (
	BaseFieldSize_Int_COPY     = BaseFieldSize_Int
	BaseFieldSize_Int_DEEPCOPY = new(big.Int).Set(BaseFieldSize_Int)
)
var (
	baseFieldSize_Int_COPY     = baseFieldSize_Int
	baseFieldSize_Int_DEEPCOPY = new(big.Int).Set(baseFieldSize_Int)
)

var baseFieldSize_uint256_COPY = baseFieldSize_uint256

var (
	BaseFieldSize_64_COPY = BaseFieldSize_64
	BaseFieldSize_32_COPY = BaseFieldSize_32
	BaseFieldSize_16_COPY = BaseFieldSize_16
	BaseFieldSize_8_COPY  = BaseFieldSize_8
)

var (
	BaseFieldMultiplicateOddOrder_uint256_COPY = BaseFieldMultiplicateOddOrder_uint256
	tonelliShanksExponent_uint256_COPY         = tonelliShanksExponent_uint256
)

var (
	zero_uint256_COPY       = zero_uint256
	one_uint256_COPY        = one_uint256
	two_uint256_COPY        = two_uint256
	uint256Max_uint256_COPY = uint256Max_uint256
)

var (
	twoTo256_Int_COPY     = twoTo256_Int
	twoTo256_Int_DEEPCOPY = new(big.Int).Set(twoTo256_Int)
	twoTo512_Int_COPY     = twoTo512_Int
	twoTo512_Int_DEEPCOPY = new(big.Int).Set(twoTo512_Int)
)

var (
	twiceBaseFieldSize_Int_COPY     = twiceBaseFieldSize_Int
	twiceBaseFieldSize_Int_DEEPCOPY = new(big.Int).Set(twiceBaseFieldSize_Int)
	twiceBaseFieldSize_uint256_COPY = twiceBaseFieldSize_uint256
	twiceBaseFieldSize_64_COPY      = twiceBaseFieldSize_64
)

var (
	thriceBaseFieldSize_Int_COPY              = thriceBaseFieldSize_Int
	thriceBaseFieldSize_Int_DEEPCOPY          = new(big.Int).Set(thriceBaseFieldSize_Int)
	thriceBaseFieldSizeMod2To256_Int_COPY     = thriceBaseFieldSizeMod2To256_Int
	thriceBaseFieldSizeMod2To256_Int_DEEPCOPY = new(big.Int).Set(thriceBaseFieldSizeMod2To256_Int)
	thriceBaseFieldSizeMod2To256_uint256_COPY = thriceBaseFieldSizeMod2To256_uint256
	thriceBaseFieldSize_64_COPY               = thriceBaseFieldSize_64
)

var (
	twoTo256ModBaseField_Int_COPY     = twoTo256ModBaseField_Int
	twoTo256ModBaseField_Int_DEEPCOPY = new(big.Int).Set(twoTo256ModBaseField_Int)
	twoTo256ModBaseField_uint256_COPY = twoTo256ModBaseField_uint256
)

var (
	twoTo512ModBaseField_Int_COPY     = twoTo512ModBaseField_Int
	twoTo512ModBaseField_Int_DEEPCOPY = new(big.Int).Set(twoTo512ModBaseField_Int)
	twoTo512ModBaseField_uint256_COPY = twoTo512ModBaseField_uint256
)

var (
	minusOneHalfModBaseField_Int_COPY     = minusOneHalfModBaseField_Int
	minusOneHalfModBaseField_Int_DEEPCOPY = new(big.Int).Set(minusOneHalfModBaseField_Int)
	minusOneHalfModBaseField_uint256_COPY = minusOneHalfModBaseField_uint256
)

var (
	oneHalfModBaseField_Int_COPY     = oneHalfModBaseField_Int
	oneHalfModBaseField_Int_DEEPCOPY = new(big.Int).Set(oneHalfModBaseField_Int)
	oneHalfModBaseField_uint256_COPY = oneHalfModBaseField_uint256
)

var (
	montgomeryBound_Int_COPY     = montgomeryBound_Int
	montgomeryBound_Int_DEEPCOPY = new(big.Int).Set(montgomeryBound_Int)
	montgomeryBound_uint256_COPY = montgomeryBound_uint256
)

var minus2To256ModBaseField_uint256_COPY = minus2To256ModBaseField_uint256

var (
	FieldElementOne_COPY      = FieldElementOne
	FieldElementZero_COPY     = FieldElementZero
	FieldElementMinusOne_COPY = FieldElementMinusOne
	FieldElementTwo_COPY      = FieldElementTwo
	DyadicRootOfUnity_fe_COPY = DyadicRootOfUnity_fe
	dyadicRootOfUnity_fe_COPY = dyadicRootOfUnity_fe
)

func TestEnsureFieldElementConstantsWereNotChanged(t *testing.T) {
	ensureFieldElementConstantsWereNotChanged()
}

func TestValidityOfConstants(t *testing.T) {
	prepareTestFieldElements(t)
	var temp_fe FieldElement
	var temp_uint256 Uint256
	var temp_Int *big.Int = big.NewInt(0)

	testutils.Assert(BaseFieldBitLength == BaseFieldSize_Int.BitLen())
	testutils.Assert(BaseFieldByteLength == len(BaseFieldSize_Int.Bytes()))
	testutils.Assert(BaseFieldSize_Int.ProbablyPrime(20))
	testutils.Assert(BaseFieldSize_64 == baseFieldSize_uint256)
	temp_uint256.SetBigInt(baseFieldSize_Int)
	testutils.Assert(temp_uint256 == baseFieldSize_uint256)

	testutils.Assert(1<<BaseField2Adicity*BaseFieldMultiplicativeOddOrder == BaseFieldSize_untyped-1)

	temp_uint256 = Uint256{}
	testutils.Assert(temp_uint256 == zero_uint256)
	temp_uint256.SetBigInt(common.One_Int)
	testutils.Assert(temp_uint256 == one_uint256)
	temp_uint256.SetBigInt(common.Two_Int)
	testutils.Assert(temp_uint256 == two_uint256)
	temp_Int.Set(twoTo256_Int)
	temp_Int.Sub(temp_Int, common.One_Int)
	temp_uint256.SetBigInt(temp_Int)
	testutils.Assert(temp_uint256 == uint256Max_uint256)
	temp_uint256.Add(&one_uint256, &uint256Max_uint256)
	testutils.Assert(temp_uint256.IsZero())

	temp_Int.Lsh(common.One_Int, 256)
	testutils.Assert(twoTo256_Int.Cmp(temp_Int) == 0)
	temp_Int.Lsh(common.One_Int, 512)
	testutils.Assert(twoTo512_Int.Cmp(temp_Int) == 0)

	temp_Int.Add(baseFieldSize_Int, baseFieldSize_Int)
	testutils.Assert(twiceBaseFieldSize_Int.Cmp(temp_Int) == 0)
	testutils.Assert(twiceBaseFieldSize_64 == twiceBaseFieldSize_uint256)

	testutils.Assert(thriceBaseFieldSizeMod2To256_untyped-twiceBaseFieldSize_untyped == BaseFieldSize_untyped-uint256Max_untyped-1)
	temp_Int.SetUint64(3)
	temp_Int.Mul(baseFieldSize_Int, temp_Int)
	testutils.Assert(thriceBaseFieldSize_Int.Cmp(temp_Int) == 0)
	temp_Int.Mod(temp_Int, twoTo256_Int)
	testutils.Assert(temp_Int.Cmp(thriceBaseFieldSizeMod2To256_Int) == 0)
	testutils.Assert(utils.CompareSlices(thriceBaseFieldSize_64[0:4], thriceBaseFieldSizeMod2To256_uint256[0:4]))
	testutils.Assert(thriceBaseFieldSize_64[4] == 1)

	testutils.Assert(twoTo256ModBaseField_untyped < BaseFieldSize_untyped)
	temp_Int.Mod(twoTo256_Int, baseFieldSize_Int)
	testutils.Assert(temp_Int.Cmp(twoTo256ModBaseField_Int) == 0)
	temp_uint256.SetBigInt(twoTo256ModBaseField_Int)
	testutils.Assert(temp_uint256 == twoTo256ModBaseField_uint256)

	testutils.Assert(twoTo512ModBaseField_untyped < BaseFieldSize_untyped)
	temp_Int.Mod(twoTo512_Int, baseFieldSize_Int)
	testutils.Assert(temp_Int.Cmp(twoTo512ModBaseField_Int) == 0)
	temp_uint256.SetBigInt(twoTo512ModBaseField_Int)
	testutils.Assert(temp_uint256 == twoTo512ModBaseField_uint256)

	temp_Int.Add(montgomeryBound_Int, baseFieldSize_Int)
	testutils.Assert(temp_Int.Cmp(twoTo256_Int) == 0)

	testutils.Assert(minusOneHalfModBaseField_Int.Sign() > 0)
	testutils.Assert(minusOneHalfModBaseField_Int.Cmp(baseFieldSize_Int) < 0)
	temp_uint256.SetBigInt(minusOneHalfModBaseField_Int)
	testutils.Assert(temp_uint256 == minusOneHalfModBaseField_uint256)
	temp_Int.Add(minusOneHalfModBaseField_Int, minusOneHalfModBaseField_Int)
	temp_Int.Add(temp_Int, common.One_Int)
	temp_Int.Mod(temp_Int, baseFieldSize_Int)
	testutils.Assert(temp_Int.Sign() == 0)

	testutils.Assert(oneHalfModBaseField_Int.Sign() > 0)
	testutils.Assert(oneHalfModBaseField_Int.Cmp(baseFieldSize_Int) < 0)
	temp_uint256.SetBigInt(oneHalfModBaseField_Int)
	testutils.Assert(temp_uint256 == oneHalfModBaseField_uint256)
	temp_Int.Add(oneHalfModBaseField_Int, oneHalfModBaseField_Int)
	temp_Int.Sub(temp_Int, common.One_Int)
	temp_Int.Mod(temp_Int, baseFieldSize_Int)
	testutils.Assert(temp_Int.Sign() == 0)

	testutils.Assert(minus2To256ModBaseField_untyped < BaseFieldSize_untyped)
	testutils.Assert(minus2To256ModBaseField_untyped > 0)

	testutils.FatalUnless(t, minus2To256ModBaseField_uint256.is_fully_reduced(), "-(2**256) not fully reduced")
	temp_uint256.Add(&minus2To256ModBaseField_uint256, &twoTo256ModBaseField_uint256)
	temp_uint256.reduceBarret_fa()
	testutils.FatalUnless(t, temp_uint256.IsZero(), "-(2**256) invalid")

	testutils.FatalUnless(t, FieldElementOne.IsOne(), "Exported FieldElementOne is not 1")
	testutils.FatalUnless(t, FieldElementZero.IsZero(), "Exported FieldElementZero is not 0")

	testutils.Assert((negativeInverseModulus_uint64*baseFieldSize_0+1)%(1<<64) == 0)

	testutils.FatalUnless(t, FieldElementZero.IsZero(), "0 is not zero")
	testutils.FatalUnless(t, FieldElementOne.IsOne(), "1 is not one")
	temp_fe.Add(&FieldElementOne, &FieldElementOne)
	testutils.FatalUnless(t, FieldElementTwo.IsEqual(&temp_fe), "Exported FieldElementTwo is not 1+1")
	temp_fe.Add(&FieldElementOne, &FieldElementMinusOne)
	testutils.FatalUnless(t, temp_fe.IsZero(), "Exported FieldElementMinusOne is not -1")

	testutils.FatalUnless(t, sqrtAlg_OrderAsDyadicRootOfUnity(&dyadicRootOfUnity_fe) == BaseField2Adicity, "dyadicRootOfUnity_fe is not a primitive root of unity of the expected order")
	testutils.FatalUnless(t, sqrtAlg_OrderAsDyadicRootOfUnity(&DyadicRootOfUnity_fe) == BaseField2Adicity, "DyadicRootOfUnity_fe is not a primitive root of unity of the expected order")
	testutils.FatalUnless(t, utils.IsEqualAsBigInt(&DyadicRootOfUnity_fe, &dyadicRootOfUnity_fe), "DyadicRootOfUnity_fe and dyadicRootOfUnity_fe differ")

}

func ensureFieldElementConstantsWereNotChanged() {
	testutils.Assert(BaseFieldSize_Int_COPY == BaseFieldSize_Int)
	testutils.Assert(BaseFieldSize_Int_DEEPCOPY.Cmp(BaseFieldSize_Int) == 0)

	testutils.Assert(baseFieldSize_Int_COPY == baseFieldSize_Int)
	testutils.Assert(baseFieldSize_Int_DEEPCOPY.Cmp(baseFieldSize_Int) == 0)

	testutils.Assert(baseFieldSize_Int_DEEPCOPY.Cmp(BaseFieldSize_Int_DEEPCOPY) == 0)

	testutils.Assert(baseFieldSize_uint256 == baseFieldSize_uint256_COPY)
	testutils.Assert(BaseFieldSize_64 == BaseFieldSize_64_COPY)
	testutils.Assert(BaseFieldSize_32 == BaseFieldSize_32_COPY)
	testutils.Assert(BaseFieldSize_16 == BaseFieldSize_16_COPY)
	testutils.Assert(BaseFieldSize_8 == BaseFieldSize_8_COPY)

	testutils.Assert(&baseFieldSize_uint256[0] != &baseFieldSize_uint256_COPY[0])
	testutils.Assert(&BaseFieldSize_64[0] != &BaseFieldSize_64_COPY[0])
	testutils.Assert(&BaseFieldSize_32[0] != &BaseFieldSize_32_COPY[0])
	testutils.Assert(&BaseFieldSize_16[0] != &BaseFieldSize_16_COPY[0])
	testutils.Assert(&BaseFieldSize_8[0] != &BaseFieldSize_8_COPY[0])

	testutils.Assert(BaseFieldMultiplicateOddOrder_uint256_COPY == BaseFieldMultiplicateOddOrder_uint256)
	testutils.Assert(tonelliShanksExponent_uint256_COPY == tonelliShanksExponent_uint256)

	testutils.Assert(zero_uint256_COPY == zero_uint256)
	testutils.Assert(one_uint256_COPY == one_uint256)
	testutils.Assert(two_uint256_COPY == two_uint256)
	testutils.Assert(uint256Max_uint256_COPY == uint256Max_uint256)

	testutils.Assert(twoTo256_Int_COPY == twoTo256_Int)
	testutils.Assert(twoTo256_Int_DEEPCOPY.Cmp(twoTo256_Int) == 0)
	testutils.Assert(twoTo512_Int_COPY == twoTo512_Int)
	testutils.Assert(twoTo512_Int_DEEPCOPY.Cmp(twoTo512_Int) == 0)

	testutils.Assert(twiceBaseFieldSize_Int_COPY == twiceBaseFieldSize_Int)
	testutils.Assert(twiceBaseFieldSize_Int_DEEPCOPY.Cmp(twiceBaseFieldSize_Int) == 0)
	testutils.Assert(twiceBaseFieldSize_uint256_COPY == twiceBaseFieldSize_uint256)
	testutils.Assert(twiceBaseFieldSize_64_COPY == twiceBaseFieldSize_64)

	testutils.Assert(thriceBaseFieldSize_Int_COPY == thriceBaseFieldSize_Int)
	testutils.Assert(thriceBaseFieldSize_Int_DEEPCOPY.Cmp(thriceBaseFieldSize_Int) == 0)
	testutils.Assert(thriceBaseFieldSizeMod2To256_Int_COPY == thriceBaseFieldSizeMod2To256_Int)
	testutils.Assert(thriceBaseFieldSizeMod2To256_Int_DEEPCOPY.Cmp(thriceBaseFieldSizeMod2To256_Int) == 0)
	testutils.Assert(thriceBaseFieldSizeMod2To256_uint256_COPY == thriceBaseFieldSizeMod2To256_uint256)
	testutils.Assert(thriceBaseFieldSize_64_COPY == thriceBaseFieldSize_64)

	testutils.Assert(twoTo256ModBaseField_Int_COPY == twoTo256ModBaseField_Int)
	testutils.Assert(twoTo256ModBaseField_Int_DEEPCOPY.Cmp(twoTo256ModBaseField_Int) == 0)
	testutils.Assert(twoTo256ModBaseField_uint256_COPY == twoTo256ModBaseField_uint256)

	testutils.Assert(twoTo512ModBaseField_Int_COPY == twoTo512ModBaseField_Int)
	testutils.Assert(twoTo512ModBaseField_Int_DEEPCOPY.Cmp(twoTo512ModBaseField_Int) == 0)
	testutils.Assert(twoTo512ModBaseField_uint256_COPY == twoTo512ModBaseField_uint256)

	testutils.Assert(montgomeryBound_Int_COPY == montgomeryBound_Int)
	testutils.Assert(montgomeryBound_Int_DEEPCOPY.Cmp(montgomeryBound_Int) == 0)
	testutils.Assert(montgomeryBound_uint256_COPY == montgomeryBound_uint256)

	testutils.Assert(minusOneHalfModBaseField_Int_COPY == minusOneHalfModBaseField_Int)
	testutils.Assert(minusOneHalfModBaseField_Int_DEEPCOPY.Cmp(minusOneHalfModBaseField_Int) == 0)
	testutils.Assert(minusOneHalfModBaseField_uint256_COPY == minusOneHalfModBaseField_uint256)

	testutils.Assert(oneHalfModBaseField_Int_COPY == oneHalfModBaseField_Int)
	testutils.Assert(oneHalfModBaseField_Int_DEEPCOPY.Cmp(oneHalfModBaseField_Int) == 0)
	testutils.Assert(oneHalfModBaseField_uint256_COPY == oneHalfModBaseField_uint256)

	testutils.Assert(minus2To256ModBaseField_uint256_COPY == minus2To256ModBaseField_uint256)

	testutils.Assert(FieldElementOne_COPY == FieldElementOne)
	testutils.Assert(FieldElementZero_COPY == FieldElementZero)
	testutils.Assert(FieldElementMinusOne_COPY == FieldElementMinusOne)
	testutils.Assert(FieldElementTwo_COPY == FieldElementTwo)

	testutils.Assert(DyadicRootOfUnity_fe_COPY == DyadicRootOfUnity_fe)
	testutils.Assert(dyadicRootOfUnity_fe_COPY == dyadicRootOfUnity_fe)
}
