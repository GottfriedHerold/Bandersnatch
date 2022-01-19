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

func (p *point_xtw_base) sampleNaP(rnd *rand.Rand, index int) {
	p.x.SetZero()
	p.y.SetZero()
	switch index % 3 {
	case 0:
		p.z.SetZero()
	case 1:
		p.z.SetOne()
	case 2:
		p.z.setRandomUnsafe(rnd)
	}
	switch index % 4 {
	case 0:
		p.t.SetZero()
	case 1:
		p.t.SetOne()
	case 2:
		p.t.setRandomUnsafe(rnd)
	case 3:
		p.t.Mul(&p.x, &p.y)
		p.x.MulEq(&p.z)
		p.y.MulEq(&p.z)
		p.z.SquareEq()
	}
}

func (p *point_axtw_base) sampleNaP(rnd *rand.Rand, index int) {
	p.x.SetZero()
	p.y.SetZero()
	switch index % 3 {
	case 0:
		p.t.SetZero()
	case 1:
		p.t.SetOne()
	case 2:
		p.t.setRandomUnsafe(rnd)
	}
}

func (p *point_efgh_base) sampleNaP(rnd *rand.Rand, index int) {
	var other1, other2 *FieldElement
	switch index % 3 {
	case 0:
		p.f.SetZero()
		p.h.SetZero()
		other1 = &p.e
		other2 = &p.g
	case 1:
		p.e.SetZero()
		p.g.SetZero()
		other1 = &p.f
		other2 = &p.h
	case 2:
		p.e.SetZero()
		p.h.SetZero()
		other1 = &p.f
		other2 = &p.g
	}
	switch index % 5 {
	case 0:
		other1.SetZero()
		other2.SetZero()
	case 1:
		other1.SetOne()
		other2.SetOne()
	case 2:
		other1.setRandomUnsafe(rnd)
		other2.setRandomUnsafe(rnd)
	case 3:
		other1.setRandomUnsafe(rnd)
		other2.SetZero()
	case 4:
		other1.SetZero()
		other2.setRandomUnsafe(rnd)
	}
}
