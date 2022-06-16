package bandersnatch

// These functions perform subgroup checks.
// They are defined not on curvepoints, but rather on coordinates, because
// they are involved in actually constructing curve points
// (The alternative is initializing a curve point with these coordinates and running a function on those, but this at least temporarily breaks invariants and we wish to avoid that.)
//
// We have variants optimized for the coordinate system used

// legendreCheckA_affineX returns true for points of the form P or P+A and false for P+E1 or P+E2 where P is in the prime-order subgroup.
// The input is the affine x coordinate. Only x^2 matters, so calling with -x instead of x is fine.
func legendreCheckA_affineX(x FieldElement) bool {
	// x is passed by value. We use it as a temporary.
	x.SquareEq()
	x.multiply_by_five()
	x.AddEq(&FieldElementOne) // 1 + 5x^2 = 1-ax^2
	return x.Jacobi() >= 0    // cannot be ==0, since a is a non-square
}

// legendreCheckA_projectiveXZ returns true for points of the form P or P+A and false for P+E1 or P+E2 where P is in the prime-order subgroup.
// The inputs are projective x and z coordinates. Note that the result only depends on x, z through x^2 and z^2.
func legendreCheckA_projectiveXZ(x FieldElement, z FieldElement) bool {
	// x, z are passed by value. We use them as temporaries.
	x.SquareEq()
	x.multiply_by_five()
	z.SquareEq()
	x.AddEq(&z)
	return x.Jacobi() >= 0
}

// legendreCheckA_EG returns true for points of the form P or P+A and false for P+E1 or P+E2 where P is in the prime-order subgroup.
// The inputs are E and G coordinates. The result only depends on E^2 and G^2, so signs of the inputs do not matter.
func legendreCheckA_EG(e FieldElement, g FieldElement) bool {
	e.SquareEq()
	g.SquareEq()
	e.multiply_by_five()
	g.AddEq(&e)
	return g.Jacobi() >= 0
}

// legendreCheckE1 returns true for N, E1, E2 and points of the form P, P+E1 with P!=N in the prime-order subgroup.
// It returns false for A or points of the form P+A or P+E2 with P!=N.
// (i.e. it returns true for the p253+{N,E1} subgroup with an exception at E2 )
//
// legendreCheckE1_affineY takes the affine y coordinate.
// Since affine coordinates rule out E1, E2, the exception at E2 cannot occur
func legendreCheckE1_affineY(y FieldElement) bool {
	// TODO: formula not optimized

	var acc FieldElement
	// r := \sqrt(d/a), for a fixed particular choice of square root
	acc.Square(&y)                // y^2
	acc.MulEq(&squareRootDbyA_fe) // r * y^2
	var rAndOne FieldElement
	rAndOne.Add(&FieldElementOne, &squareRootDbyA_fe)
	// y passed by value
	y.MulEq(&rAndOne)
	acc.SubEq(&y)               // -(r+1)y + r*y^2
	acc.AddEq(&FieldElementOne) // ry^2 -(r+1)y + 1
	// Note: This is zero for the neutral element
	return acc.Jacobi() <= 0
}

// legendreCheckE1 returns true for N, E1, E2 and points of the form P, P+E1 with P!=N in the prime-order subgroup.
// It returns false for A or points of the form P+A or P+E2 with P!=N.
// (i.e. it returns true for the p253+{N,E1} subgroup with an exception at E2 )
//
// legendreCheckE1_projectiveYZ takes projective Y and Z coordinates.
func legendreCheckE1_projectiveYZ(y FieldElement, z FieldElement) bool {
	var acc, temp, rAndOne FieldElement
	rAndOne.Add(&FieldElementOne, &squareRootDbyA_fe)
	acc.Square(&z)
	temp.Mul(&y, &z)
	temp.MulEq(&rAndOne)
	acc.SubEq(&temp)
	temp.Square(&y)
	temp.MulEq(&squareRootDbyA_fe)
	acc.AddEq(&temp)
	// Note: acc is zero for neutral element and *both* points at infinity.
	return acc.Jacobi() <= 0
}

// legendreCheckE1_FH returns true for points from p253+{N,E1} and false for points from p253+{A,E2}
//
// Note that the _FH variant gives the "correct" result for E2 (as opposed to the others)
func legendreCheckE1_FH(f FieldElement, h FieldElement) bool {
	// identical to YZ, actually. But special cases resulting in 0 differ.
	var acc, temp, rAndOne FieldElement
	rAndOne.Add(&FieldElementOne, &squareRootDbyA_fe)
	acc.Square(&f)
	temp.Mul(&h, &f)
	temp.MulEq(&rAndOne)
	acc.SubEq(&temp)
	temp.Square(&h)
	temp.MulEq(&squareRootDbyA_fe)
	acc.AddEq(&temp)
	// Note: acc is zero for neutral element and E1.
	return acc.Jacobi() <= 0
}

// isPointOnCurve checks whether the given point is actually on the curve.
// Note: This does NOT verify that the point is in the correct subgroup.
// Note2: On encountering singular values (0:0:0:0), we just return false *without* calling any error handler.
func (p *point_xtw_base) isPointOnCurve() bool {

	// Singular points are not on the curve
	if p.IsNaP() {
		return false
	}

	// check whether x*y == t*z
	var u, v FieldElement
	u.Mul(&p.x, &p.y)
	v.Mul(&p.t, &p.z)
	if !u.IsEqual(&v) {
		return false
	}

	// We now check the main curve equation, i.e. whether ax^2 + y^2 == z^2 + dt^2
	u.Square(&p.t)
	u.MulEq(&CurveParameterD_fe) // u = d*t^2
	v.Square(&p.z)
	u.AddEq(&v) // u= dt^2 + z^2
	v.Square(&p.y)
	u.SubEq(&v) // u = z^2 + dt^2 - y^2
	v.Square(&p.x)
	v.multiply_by_five()
	u.AddEq(&v) // u = z^2 + dt^2 - y^2 + 5x^2 ==  z^2 + dt^2 - y^2 - ax^2
	return u.IsZero()
}

func (p *point_axtw_base) isPointOnCurve() bool {
	// Singular points are not on the curve
	if p.IsNaP() {
		return false
	}

	// check whether x*y == t
	var u FieldElement
	u.Mul(&p.x, &p.y)
	if !u.IsEqual(&p.t) {
		return false
	}

	var v FieldElement
	// We now check the main curve equation, i.e. whether ax^2 + y^2 == 1 + dt^2
	u.Square(&p.t)
	u.MulEq(&CurveParameterD_fe) // u = d*t^2
	u.AddEq(&FieldElementOne)    // u = dt^2 + 1 = dt^2 + z^2
	v.Square(&p.y)
	u.SubEq(&v) // u = z^2 + dt^2 - y^2
	v.Square(&p.x)
	v.multiply_by_five()
	u.AddEq(&v) // u = z^2 + dt^2 - y^2 + 5x^2 ==  z^2 + dt^2 - y^2 - ax^2
	return u.IsZero()
}

func (p *point_efgh_base) isPointOnCurve() bool {
	if p.IsNaP() {
		return false
	}
	// direct check might only be slightly faster.
	p_xtw := p.toDecaf_xtw()
	return p_xtw.isPointOnCurve()
}
