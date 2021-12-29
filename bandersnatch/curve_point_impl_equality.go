package bandersnatch

/*
	Note: Suffixes like _ttt or _tta refer to the type of input point (with order output, input1 [,input2] )
	t denote extended projective,
	a denotes extended affine (i.e. Z==1)
	s denotes double-projective
*/

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

// is_equal_tt checks whether two points in the subgroup are equal. On the p523+A subgroup, it checks for equality modulo the affine order-2 point.
func (p1 *Point_xtw) is_equal_tt(p2 *Point_xtw) bool {
	// We check whether x1/y1 == x2/y2. Note that the map Curve -> Field given by x/y is 2:1 with preimages of the form {P, P+A} for the affine 2 torsion point A.
	result, error_code := check_equality_of_quotients(&p1.x, &p1.y, &p2.x, &p2.y)
	if error_code != 0 {
		switch error_code {
		case 1:
			return napEncountered("When comparing two xtw points, the first one was invalid", true, p1, p2)
		case 2:
			return napEncountered("When comparing two xtw points, the second one was invalid", true, p1, p2)
		case 3:
			return napEncountered("When comparing two xtw points, both were invalid", true, p1, p2)
		}
	}
	return result
}

func (p1 *Point_xtw) is_equal_ta(p2 *Point_axtw) bool {
	// We check whether x1/y1 == x2/y2. Note that the map Curve -> Field given by x/y is 2:1 with preimages of the form {P, P+A} for the affine 2 torsion point A.
	result, error_code := check_equality_of_quotients(&p1.x, &p1.y, &p2.x, &p2.y)
	if error_code != 0 {
		switch error_code {
		case 1:
			return napEncountered("When comparing an axtw and xtw point, the axtw one was invalid", true, p1, p2)
		case 2:
			return napEncountered("When comparing an axtw and xtw point, the xtw one was invalid", true, p1, p2)
		case 3:
			return napEncountered("When comparing an axtw and xtw point, both were invalid", true, p1, p2)
		}
	}
	return result
}

func (p1 *Point_axtw) is_equal_at(p2 *Point_xtw) bool {
	return p2.is_equal_ta(p1)
}

func (p1 *Point_axtw) is_equal_aa(p2 *Point_axtw) bool {
	// We check whether x1/y1 == x2/y2. Note that the map Curve -> Field given by x/y is 2:1 with preimages of the form {P, P+A} for the affine 2 torsion point A.

	// Note: Due to z1==z2 == 1, we actually have (x1,y1) == +/- (x2,y2) on equality. We could probably use this to speed this up.
	result, error_code := check_equality_of_quotients(&p1.x, &p1.y, &p2.x, &p2.y)
	if error_code != 0 {
		switch error_code {
		case 1:
			return napEncountered("When comparing two axtw points, the first one was invalid", true, p1, p2)
		case 2:
			return napEncountered("When comparing two axtw points, the second one was invalid", true, p1, p2)
		case 3:
			return napEncountered("When comparing two axtw points, both were invalid", true, p1, p2)
		}
	}
	return result
}

// is_equal_exact_tt checks whether p1 == p2. This works for all rational points (including points at infinity), not only those in the subgroup. It does *not* identify P with P+A
// We assume both points not to be singular.
func (p1 *Point_xtw) is_equal_exact_tt(p2 *Point_xtw) bool {
	if p1.IsNaP() {
		return napEncountered("When comparing two xtw points exactly, the first one was invalid", true, p1, p2)
	}
	if p2.IsNaP() {
		return napEncountered("When comparing two xtw points exactly, the second one was invalid", true, p1, p2)
	}
	var temp1, temp2 FieldElement
	if p1.z.IsZero() {
		if !p2.z.IsZero() {
			return false
		}
		// z == 0 implies y == 0 and t,x non-zero, so p1.y == p2.y == p1.z == p2.z == 0
		temp1.Mul(&p1.x, &p2.t)
		temp2.Mul(&p1.t, &p2.x)
		return temp1.IsEqual(&temp2)
	}
	if p2.z.IsZero() {
		return false // p1.z != 0, because these cases were already done above.
	}
	// p1, p2 both have z!=0. We need to check both x1/z1 == x2/z2 and y1/z1 == y2/z2
	temp1.Mul(&p1.x, &p2.z)
	temp2.Mul(&p1.z, &p2.x)
	if !temp1.IsEqual(&temp2) {
		return false
	}
	// Note that we actually know that y1/z1 == +/- y2/z2, as the curve equations only has 2 solutions for given y.
	temp1.Mul(&p1.y, &p2.z)
	temp2.Mul(&p1.z, &p2.y)
	return temp1.IsEqual(&temp2)
}

func (p1 *Point_xtw) is_equal_exact_ta(p2 *Point_axtw) bool {
	if p1.IsNaP() {
		return napEncountered("When comparing an axtw and xtw point exactly, the xtw one was invalid", true, p1, p2)
	}
	if p2.IsNaP() {
		return napEncountered("When comparing and axtw and xtw point exactly, the axtw one was invalid", true, p1, p2)
	}
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

func (p1 *Point_axtw) is_equal_exact_at(p2 *Point_xtw) bool {
	return p2.is_equal_exact_ta(p1)
}

func (p1 *Point_axtw) is_equal_exact_aa(p2 *Point_axtw) bool {
	if p1.IsNaP() {
		return napEncountered("When comparing two axtw points, the first one was invalid", true, p1, p2)
	}
	if p2.IsNaP() {
		return napEncountered("When comparing two axtw points, the second one was invalid", true, p1, p2)
	}
	return p1.x.IsEqual(&p2.x) && p1.y.IsEqual(&p2.y)
}
