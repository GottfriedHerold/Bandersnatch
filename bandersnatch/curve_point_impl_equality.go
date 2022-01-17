package bandersnatch

/*
	Note: Suffixes like _ttt or _tta refer to the type of input point (with order output, input1 [,input2] )
	t denote extended projective,
	a denotes extended affine (i.e. Z==1)
	s denotes double-projective
*/

// checkEqualityOfQuotients(x1, y1, x2, y2) checks whether x1*y2 == x2 * y1 (i.e. x1/y1 == x2/y2).
// The second return value is true iff x1*y2 == x2*y1 == 0.
func checkEqualityOfQuotients(x1, y1, x2, y2 *FieldElement) (result bool, zero bool) {
	var temp1, temp2 FieldElement
	temp1.Mul(x1, y2)
	temp2.Mul(y1, x2)
	result = temp1.IsEqual(&temp2)
	if result {
		zero = temp1.IsZero()
	}
	return
}

/*
// check_equality_of_quotients checks whether x1/y1 == x2/y2. The second err argument is 0 unless x1==y1==0 (err==1) or x2==y2==0 (err==2) or both (err==3). If err != 0, always returns false.
// In the special case where both x1!=0, x2!=0 (but may be different) and y1==y2==0, returns true and err = 0.
func check_equality_of_quotients(x1, y1, x2, y2 *FieldElement) (result bool, err int) {
	var temp1, temp2 FieldElement
	temp1.Mul(x1, y2)
	temp2.Mul(y1, x2)
	err = 0
	if temp1.IsEqual(&temp2) {
		result = true
		if temp1.IsZero() {
			if y1.IsZero() && x1.IsZero() {
				result = false
				err += 1
			}
			if y2.IsZero() && x2.IsZero() {
				result = false
				err += 2
			}
		}
	} else {
		result = false
	}
	return
}
*/

// exceptional cases: NaP [output true, true]
// equal 2-torsion outputs also output true, true
func (p1 *point_xtw_base) isEqual_moduloA_tt(p2 *point_xtw_base) (ret bool, zero bool) {
	// We check whether x1/y1 == x2/y2.
	// Note that the map Curve -> Field given by x/y is 2:1 with preimages of the form {P, P+A} for the affine 2 torsion point A.
	return checkEqualityOfQuotients(&p1.x, &p1.y, &p2.x, &p2.y)
}

// exceptional cases: NaP [p1 NaP gives true, true]
func (p1 *point_xtw_base) isEqual_moduloA_tany(p2 CurvePointPtrInterfaceBaseRead) (ret bool, zero bool) {
	p2_x := p2.X_decaf_projective()
	p2_y := p2.Y_decaf_projective()
	return checkEqualityOfQuotients(&p1.x, &p1.y, &p2_x, &p2_y)
}

// exceptional cases: NaP [output true, true]
// equal 2-torsion outputs also output true, true
func (p1 *point_xtw_base) isEqual_moduloA_ta(p2 *point_axtw_base) (ret bool, zero bool) {
	return p2.isEqual_moduloA_at(p1)
}

// exceptional cases: NaP [output true, true]
// equal 2-torsion outputs also output true, true
func (p1 *point_axtw_base) isEqual_moduloA_at(p2 *point_xtw_base) (ret bool, zero bool) {
	return checkEqualityOfQuotients(&p1.x, &p1.y, &p2.x, &p2.y)
}

// exceptional cases: NaPs compare equal only to NaP
func (p1 *point_axtw_base) isEqual_moduloA_aa(p2 *point_axtw_base) (ret bool) {
	// We check (x1,y1) = +/-(x2,y2)
	absEqual, exact := p1.x.CmpAbs(&p2.x)
	if !absEqual {
		return false
	}
	if exact {
		return p1.y.IsEqual(&p2.y)
	} else {
		var tmp FieldElement
		tmp.Neg(&p1.y)
		return tmp.IsEqual(&p2.y)
	}
}

// exceptional cases: NaP compares equal only to p2.X_decaf == p2.Y_decaf == 0
func (p1 *point_axtw_base) isEqual_moduloA_aany(p2 CurvePointPtrInterfaceBaseRead) (ret bool) {
	p2_x := p2.X_decaf_projective()
	absEqual, exact := p1.x.CmpAbs(&p2_x)
	if !absEqual {
		return false
	}
	p2_y := p2.Y_decaf_projective()
	if exact {
		return p1.y.IsEqual(&p2_y)
	} else {
		p2_y.NegEq()
		return p1.y.IsEqual(&p2_y)
	}
}

// exceptional cases: NaP [output is ?, true]
func (p1 *point_xtw_base) isEqual_exact_tt(p2 *point_xtw_base) (ret bool, potentialNaP bool) {
	// In the usual case, we check equality of x/y and y/z.
	// Checking x/y first is optimized for detecting NaPs.
	ret, potentialNaP = checkEqualityOfQuotients(&p1.x, &p1.y, &p2.x, &p2.y)
	if !ret {
		return // return false, false [zero == false is guaranteed]
	}
	if !potentialNaP {
		// easy case: x/y is equal for both points and neither is a NaP or 2-torsion.
		// Checking equality of y/z is enough.
		var temp1, temp2 FieldElement
		temp1.Mul(&p1.y, &p2.z)
		temp2.Mul(&p1.z, &p2.y)
		ret = temp1.IsEqual(&temp2)
		return
	}
	// This is the difficult case: We have x1 * y2 == x2 * y1 == 0
	// There are four cases here:
	// 1) p1 is NaP  -- Caller needs to deal with this
	// 2) p2 is NaP  -- Caller needs to deal with this
	// 3) p1, p2 \in {N,A}
	// 4) p1, p2 \in {E1, E2}
	if p1.x.IsZero() {
		// same as above, really
		var temp1, temp2 FieldElement
		temp1.Mul(&p1.y, &p2.z)
		temp2.Mul(&p1.z, &p2.y)
		ret = temp1.IsEqual(&temp2)
		return
	} else {
		// y1 == y2 == 0 (and also z1 == z2 == 0)
		var temp1, temp2 FieldElement
		temp1.Mul(&p1.x, &p2.t)
		temp2.Mul(&p1.t, &p2.x)
		ret = temp1.IsEqual(&temp2)
		return
	}
}

// exceptional cases: NaPs lead to arbitrary result
func (p1 *point_xtw_base) isEqual_exact_ta(p2 *point_axtw_base) bool {
	if p1.z.IsZero() {
		return false
	}
	// Check x1/z1 == x2/z2 (Note z2 ==1, so this means x1 == z1 * x2)
	var temp FieldElement
	temp.Mul(&p1.z, &p2.x)
	if !temp.IsEqual(&p1.x) {
		return false
	}
	// Check y1/z1 == y2/z2 (Note z2 ==1, so this means y1 == z1 * y2)
	temp.Mul(&p1.z, &p2.y)
	return temp.IsEqual(&p1.y)
}

// exceptional cases: NaPs lead to arbitrary results
func (p1 *point_axtw_base) isEqual_exact_at(p2 *point_xtw_base) bool {
	return p2.isEqual_exact_ta(p1)
}

// exceptional cases: NaPs only compare equal to NaPs
func (p1 *point_axtw_base) isEqual_exact_aa(p2 *point_axtw_base) bool {
	return p1.x.IsEqual(&p2.x) && p1.y.IsEqual(&p2.y)
}

// exceptional cases: NaPs [arbitrary behaviour]
func (p1 *point_xtw_base) isEqual_exact_tany(p2 CurvePointPtrInterfaceRead) bool {
	// check x/y first. We take decaf-coos first for this.
	p2_x := p2.X_decaf_projective()
	p2_y := p2.Y_decaf_projective()
	ret, zero := checkEqualityOfQuotients(&p1.x, &p1.y, &p2_x, &p2_y)
	if !ret {
		return false // return false, false [zero == false is guaranteed]
	}
	if !zero || p2_x.IsZero() {
		// need to check y/z
		p2_y = p2.Y_projective()
		p2_z := p2.Z_projective()
		p2_z.MulEq(&p1.y)
		p2_y.MulEq(&p1.z)
		return p2_y.IsEqual(&p2_z)
	}
	// If we get here, both points are at infinity
	if !p2.CanRepresentInfinity() {
		return false
	}
	return p1.IsE1() == p2.(CurvePointPtrInterfaceReadCanDistinguishInfinity).IsE1()
}

// exceptional cases: NaPs
func (p1 *point_axtw_base) isEqual_exact_aany(p2 CurvePointPtrInterfaceRead) bool {
	if p2.IsAtInfinity() {
		return false
	}
	p2_z := p2.Z_projective()
	p2_x := p2.X_projective()
	// Check x1/z1 == x2/z2 (Note z1==1, so this means x1 * z2 == x2)
	var temp FieldElement
	temp.Mul(&p1.x, &p2_z)
	if !temp.IsEqual(&p2_x) {
		return false
	}
	p2_y := p2.Y_projective()
	// Check y1/z1 == y2/z2 (Note z1==1, so this means y1 * z2 == y2)
	temp.Mul(&p1.y, &p2_z)
	return temp.IsEqual(&p2_y)
}

// exceptional cases: NaPs
func (p *point_efgh_base) isEqual_exact_ss(other *point_efgh_base) bool {
	var tmp1, tmp2 FieldElement
	// check whether h1/f1 == h2/f2. Note that none of these number can be zero.
	tmp1.Mul(&p.h, &other.f)
	tmp2.Mul(&p.f, &other.h)

	if !tmp1.IsEqual(&tmp2) {
		return false
	}
	// If we get here, we have h1/f1 == h2/f2. Since h/f = Y/Z, this means p = +/-other.
	tmp1.Mul(&p.e, &other.g)
	tmp2.Mul(&p.g, &other.e)
	// We know these are equal up to sign.
	// The case where tmp1 and/or tmp2 are zero is actually correct, as can be seen by going through the cases.
	// [In fact, if either tmp1==0 or tmp2==0, then we actually must return true, as h/f has a value that uniquely determines the point]
	return tmp1.IsEqual(&tmp2)
}

// exceptional cases: NaPs
func (p *point_efgh_base) isEqual_moduloA_ss(other *point_efgh_base) bool {
	var tmp1, tmp2 FieldElement
	var yExactlyEqual bool
	// check whether h1/f1 == +/- h2/f2. Note that nothing here is zero.
	tmp1.Mul(&p.h, &other.f)
	tmp2.Mul(&p.f, &other.h)
	if tmp1.IsEqual(&tmp2) {
		yExactlyEqual = true
	} else {
		tmp1.NegEq()
		if !tmp1.IsEqual(&tmp2) {
			return false
		}
		yExactlyEqual = false // automatic, but we are explicit
	}
	// if we get here, we have h1/f1 == +/-h2/f2 (with sign determined by yExactlyEqual).
	// This means that the y-coos of both points are the same up to sign.
	// We know that one of the following holds (exactly, without modulo A):
	// P == other [&& yExactlyEqual = true]
	// P == -other [&& yExactlyEqual = true]
	// P == other + A [&& yExactlyEqual = false]
	// P == -other + A [&& yExactlyEqual = false]

	// We now check whether x/z=e/g is equal up to sign. The sign must be the same as
	// for equality of h/f up to sign.
	tmp1.Mul(&p.e, &other.g)
	tmp2.Mul(&p.g, &other.e)
	// Note: If tmp1 or tmp2 is 0, then the other one must be as well
	// (because the h1/f1 == +/-h2/f2 check can only pass in that case if either both p and other are among N,A or
	// both are at infinity.)
	// In this case, we return true, which is correct.
	if !yExactlyEqual {
		tmp1.NegEq()
	}
	return tmp1.IsEqual(&tmp2)
}

// exceptional cases: NaPs [infinity handled inside function]
func (p *point_efgh_base) isEqual_exact_sany(other CurvePointPtrInterfaceRead) bool {
	other_x, other_y, other_z := other.XYZ_projective()
	// we check whether other.y/other.z == p.h/p.f
	var tmp FieldElement
	tmp.Mul(&other_z, &p.h)
	other_y.MulEq(&p.f) // may overwrite
	if !tmp.IsEqual(&other_y) {
		return false
	}
	if tmp.IsZero() {
		// We have other_y == other_z == 0, i.e. other is at infinity
		if p.g.IsZero() {
			// p is at infinity as well. Check whether they are the same point at infinity.
			return p.IsE1() == other.(CurvePointPtrInterfaceReadCanDistinguishInfinity).IsE1()
		} else {
			return false
		}
	}
	// check whether other.x/other.z == p.e/p.g
	// Note that neither other.z or other.g can be zero at this point:
	// other.z == 0 was caught already, p.g==0 implies p is at infinity and p.h/p.f
	// has a value (+/- sqrt(d/a)) that is impossible for y/z to have.
	other_x.MulEq(&p.g) // may overwrite
	tmp.Mul(&other_z, &p.e)
	return tmp.IsEqual(&other_x)
}

// exceptional cases: NaPs
func (p *point_efgh_base) isEqual_moduloA_sany(other CurvePointPtrInterfaceRead) bool {
	// We check whether x/y is equal for p and other
	other_x := other.X_decaf_projective()
	other_y := other.Y_decaf_projective()

	// If some expression is 0, the following is actually correct:
	// p.f,p.h != 0 always holds.
	// If other_x = 0, then other_y !=0, and equality holds iff p.e = 0, which is correct [both points neutral]
	// If other_y = 0, then other_x !=0, and equality holds iff p.g = 0 (which is correct, but proably won't happen as we are unlikely to call this outside of the subgroup)
	// Otherwise for other_x, other_y !=0, if either p.e or p.g are zero (they cannot be both zero) we return false
	other_x.MulEq(&p.g)
	other_x.MulEq(&p.h)
	other_y.MulEq(&p.e)
	other_y.MulEq(&p.f)
	return other_x.IsEqual(&other_y)
}
