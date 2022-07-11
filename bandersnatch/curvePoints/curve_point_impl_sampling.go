package curvePoints

import (
	"math/rand"
)

// Creates a random point on the curve, which does not neccessarily need to be in the correct subgroup.
func makeRandomPointOnCurve_a(rnd *rand.Rand) point_axtw_base {
	var x, y, t FieldElement

	// Set x randomly, compute y from x
	for {
		x.SetRandomUnsafe(rnd)
		// x.SetUInt64(1)
		var err error
		y, err = recoverYFromXAffine(&x, false)
		if err == nil {
			break
		}
	}

	// We don't really know the distribution of the sign of y
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
	z.SetRandomUnsafeNonZero(rnd)
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
		p.z.SetRandomUnsafe(rnd)
	}
	switch index % 4 {
	case 0:
		p.t.SetZero()
	case 1:
		p.t.SetOne()
	case 2:
		p.t.SetRandomUnsafe(rnd)
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
		p.t.SetRandomUnsafe(rnd)
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
		other1.SetRandomUnsafe(rnd)
		other2.SetRandomUnsafe(rnd)
	case 3:
		other1.SetRandomUnsafe(rnd)
		other2.SetZero()
	case 4:
		other1.SetZero()
		other2.SetRandomUnsafe(rnd)
	}
}
