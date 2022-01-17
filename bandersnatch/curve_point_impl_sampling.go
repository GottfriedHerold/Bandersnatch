package bandersnatch

import (
	"errors"
	"math/rand"
)

// TODO: MOVE

// Note: If X/Z is not on the curve, we might get either a "not on curve" or "not in subgroup" error. Should we clarify the wording to reflect that?
var (
	ErrXNotInSubgroup     = errors.New("deserialization: received affine X coordinate does not correspond to any point in the p253 subgroup of the Bandersnatch curve")
	ErrXNotOnCurve        = errors.New("deserialization: received affine X coordinate does not correspond to any (finite, rational) point of the Bandersnatch curve")
	ErrNotInSubgroup      = errors.New("deserialization: received affine X and Y coordinates do not correspond to a point in the p253 subgroup of the Bandersnatch curve")
	ErrNotOnCurve         = errors.New("deserialization: received affine X and Y corrdinates do not correspond to a point on the Bandersnatch curve")
	ErrWrongSignY         = errors.New("deserialization: encountered affine Y coordinate with unexpected Sign bit")
	ErrUnrecognizedFormat = errors.New("deserialization: could not automatically detect serialization format")
)

// recoverYFromXAffine computes y from x such that (x,y) is on the curve. Note that the result only depends on x up to sign.
// For valid input x, for which some y exists in the first place, there are always exatly two possible y which differ by sign.
// recoverYFromXAffine makes no guarantees about the choice of y. It need not even be consistent for multiple calls with the same x.
// If checkSubgroup is set to true, we also check that the resulting (+/-x,+/-y) is on the subgroup for some choice of signs.
// Returns err==nil if no error occurred (meaning that some y existed and the subgroup check, if requrested, did not fail).
func recoverYFromXAffine(x *FieldElement, checkSubgroup bool) (y FieldElement, err error) {

	// We have y^2 = (1-ax^2) / (1-dx^2)
	// So, we first compute (1-ax^2) / 1-dx^2
	var num, denom FieldElement

	num.Square(x)                        // x^2, only compute this once
	denom.Mul(&num, &CurveParameterD_fe) // dx^2
	num.multiply_by_five()               // 5x^2 = -ax^2
	num.AddEq(&FieldElementOne)          // 1 - ax^2
	denom.Sub(&FieldElementOne, &denom)  // 1 - dx^2
	// Note that x is in the correct subgroup iff *both* num and denom are squares
	if checkSubgroup {
		if num.Jacobi() < 0 {
			err = ErrXNotInSubgroup
			return
		}
	}
	num.DivideEq(&denom) // (1-ax^2)/(1-dx^2). Note that 1-dx^2 cannot be 0, as d is a non-square.
	if !y.SquareRoot(&num) {
		err = ErrXNotOnCurve
		return
	}
	err = nil // err is nil at this point anyway, but we prefer to be explicit.
	return
}

// Creates a random point on the curve, which does not neccessarily need to be in the correct subgroup.
func makeRandomPointOnCurve_a(rnd *rand.Rand) point_axtw_base {
	var x, y, t FieldElement

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

	// We don't really know the distribution of y
	if rnd.Intn(2) == 0 {
		y.NegEq()
	}

	// Set t = x*y. If we would set z to 1, this is now a correct point.
	t.Mul(&x, &y)

	return point_axtw_base{x: x, y: y, t: t}
}

func makeRandomPointOnCurve_t(rnd *rand.Rand) (ret point_xtw_base) {
	rnd_axtw := makeRandomPointOnCurve_a(rnd)
	var z FieldElement
	z.setRandomUnsafeNonZero(rnd)
	ret.x.Mul(&z, &rnd_axtw.x)
	ret.y.Mul(&z, &rnd_axtw.y)
	ret.t.Mul(&z, &rnd_axtw.t)
	ret.z = z
	return
}
