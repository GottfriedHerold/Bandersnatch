package bandersnatch

// this file contains internal function that add torsion points to given points.
// This is mostly used in testing.

func (p *point_xtw_base) torsionAddA() {
	p.x.NegEq()
	p.y.NegEq()
}

func (p *point_axtw_base) torsionAddA() {
	p.x.NegEq()
	p.y.NegEq()
}

func (p *point_efgh_base) torsionAddA() {
	p.e.NegEq()
	p.h.NegEq()
}

func (p *point_xtw_base) torsionAddE1() {
	// Not the most efficient way
	p.x, p.y = p.y, p.x
	p.t, p.z = p.z, p.t
	p.x.MulEq(&squareRootDbyA_fe)
	p.y.MulEq(&squareRootDbyA_fe)
	p.y.MulEq(&CurveParameterA_fe)
	p.z.MulEq(&CurveParameterD_fe)
}

func (p *point_axtw_base) torsionAddE1() {
	if p.IsNaP() {
		*p = point_axtw_base{}
		return
	}
	if p.x.IsZero() {
		panic("Adding infinity to finite two-torsion for axtw is not possible")
	}
	// Not the most efficient way
	p.t.MulEq(&CurveParameterD_fe)
	p.t.InvEq()
	p.x, p.y = p.y, p.x
	p.x.MulEq(&squareRootDbyA_fe)
	p.x.MulEq(&p.t)
	p.y.MulEq(&squareRootDbyA_fe)
	p.y.MulEq(&CurveParameterA_fe)
	p.y.MulEq(&p.t)
}

func (p *point_efgh_base) torsionAddE1() {
	if p.IsNaP() {
		*p = point_efgh_base{}
		return
	}
	// New e;f;g;h = G;\sqrt{d/a}H; a\sqrt{d/a}E;F
	p.e, p.f, p.g, p.h = p.g, p.h, p.e, p.f
	p.f.MulEq(&squareRootDbyA_fe)
	p.g.MulEq(&squareRootDbyA_fe)
	p.g.MulEq(&CurveParameterA_fe)
}

func (p *point_xtw_base) torsionAddE2() {
	p.torsionAddE1()
	p.torsionAddA()
}

func (p *point_axtw_base) torsionAddE2() {
	p.torsionAddE1()
	p.torsionAddA()
}

func (p *point_efgh_base) torsionAddE2() {
	p.torsionAddE1()
	p.torsionAddA()
}
