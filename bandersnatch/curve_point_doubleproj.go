//go:build ignore

package bandersnatch

/*
	Without the P=P+A identification, all finite rational points have a unique "normalized" representative with p==f==1.
	When converting to affine coordinates (or reading out any affine coordinate), we might as well (additionally) change the internal representation
	to the normalized one at essentially the same cost.
	Since multiple such coordinate reads are likely to come in a row, we check whether we already are in that form to save some work.
*/

func (p *Point_efgh) XYZ_projective() (X FieldElement, Y FieldElement, Z FieldElement) {
	p.normalizeSubgroup()
	X.Mul(&p.e, &p.f)
	Y.Mul(&p.g, &p.h)
	Z.Mul(&p.f, &p.g)
	return
}

func (p *Point_efgh) XYTZ_projective() (X FieldElement, Y FieldElement, T FieldElement, Z FieldElement) {
	p.normalizeSubgroup()
	X.Mul(&p.e, &p.f)
	Y.Mul(&p.g, &p.h)
	T.Mul(&p.e, &p.h)
	Z.Mul(&p.f, &p.g)
	return
}

// X_affine returns the X coordinate of the given point in affine twisted Edwards coordinates, (i.e. X/Z in projective coos)
func (p *Point_efgh) X_affine() FieldElement {
	p.normalize_affine()
	p.normalizeSubgroup()
	return p.e
}

// Y_affine returns the Y coordinate of the given point in affine twisted Edwards coordinates, (i.e. Y/Z in projective coos)
func (p *Point_efgh) Y_affine() FieldElement {
	p.normalize_affine()
	p.normalizeSubgroup()
	return p.h
}

// T_affine returns the T coordinate of the given point in affine twisted Edwards coordinates, (i.e. T/Z == X*Y/Z^2 in projective coos)
func (p *Point_efgh) T_affine() (T FieldElement) {
	p.normalize_affine()
	p.normalizeSubgroup()
	T.Mul(&p.e, &p.h)
	return
}

func (p *Point_efgh) XY_affine() (X FieldElement, Y FieldElement) {
	p.normalizeSubgroup()
	p.normalize_affine()
	return p.e, p.h
}

func (p *Point_efgh) XYT_affine() (X FieldElement, Y FieldElement, T FieldElement) {
	p.normalizeSubgroup()
	p.normalize_affine()
	X = p.e
	Y = p.f
	T.Mul(&X, &Y)
	return
}

// IsNeutralElement checks if the point P is the neutral element of the curve (modulo the identification of P with P+A).
// Use IsNeutralElement_FullCurve if you do not want this identification.
func (p *Point_efgh) IsNeutralElement() bool {
	// The only valid points with e==0 are the neutral element and the affine order-2 point
	if p.IsNaP() {
		return napEncountered("Comparing NaP with neutral element for efgh", true, p)
	}
	return p.e.IsZero()
}

/*
// IsNeutralElement_FullCurve tests for zero-ness like IsNeutralElement. The difference is that it does *NOT* identify P with P+A. We only assume that the point satisfies the curve equations.
func (p *Point_efgh) IsNeutralElement_FullCurve() bool {
	return p.IsNeutralElement() && p.f.IsEqual(&p.h)
}
*/

// IsEqual compares two curve points for equality, working modulo the P = P + A identification. The two points do not have the be in the same coordinate format.
func (p *Point_efgh) IsEqual(other CurvePointPtrInterfaceRead) bool {
	if p.IsNaP() || other.IsNaP() {
		return napEncountered("NaP encountered when comparing efgh-point with other point", true, p, other)
	}
	switch other := other.(type) {
	// Not sure if specific implementation can be faster.
	default:
		// TODO !
		panic("Comparison not implemented yet")

		// This basically checks x1/y1 == x2/y2.
		var other_x = other.X_projective()
		var other_y = other.Y_projective()
		// other.x * p.y == other.y * p.x
		// Note that this works out correctly even if something is 0 here:
		// p.f,p.h are never 0. At most one one of other_x and other_y is 0 and at most one of p.e and p.g is 0.
		// We get true only for other_x==p.e==0 and other_y==p.g==0, which is indeed the correct behaviour.
		other_x.MulEq(&p.g)
		other_x.MulEq(&p.h)
		other_y.MulEq(&p.e)
		other_y.MulEq(&p.f)
		return other_x.IsEqual(&other_y)
	}
}

/*

// IsEqual_FullCurve compares two curve points for equality WITHOUT working modulo the P = P+A identification. The two points do not have to be in the same coordinate format.
// This also works for points outside the subgroup or even at infinity.
func (p *Point_efgh) IsEqual_FullCurve(other CurvePointPtrInterfaceRead_FullCurve) bool {
	temp := p.ExtendedTwistedEdwards()
	return temp.IsEqual_FullCurve(other)
}
*/

// IsAtInfinity tests whether the point is an infinite (neccessarily order-2) point.
// Note that these points are NOT in the p253-subgroup, so these are not supposed to appear under normal operation.
func (p *Point_efgh) IsAtInfinity() bool {
	if p.IsNaP() {
		return napEncountered("NaP encountered when asking where efgh-point is at infinity", true, p)
	}
	// The only valid (non-NaP) points with g==0 are are those at infinity
	return p.g.IsZero()
}

// CanRepresentInfinity reports whether the underlying point type can represent points at infinity, i.e. returns true for Point_efgh
func (p *Point_efgh) CanRepresentInfinity() bool {
	return true
}

// IsNaP checks whether the point is a NaP (Not-a-point). NaPs must never appear if the library is used correctly. They can appear by
// a) performing operations on points that are not in the correct subgroup or that are NaPs.
// b) zero-initialized points are NaPs (Go lacks constructors to fix that).
// For Point_efgh, NaP points have either f==h==0 ("true" NaP-type1) or e==g==0 ("true" NaP-type2) or e==h==0 (result of working on affine NaP).
// Note that no valid points ever have h==0 or f==0.
func (p *Point_efgh) IsNaP() bool {
	// Note: The panicking cases are not supposed to be possible to arise from working within the provided interface, even if you start with uninitialzed points.
	if p.h.IsZero() {
		if !(p.f.IsZero() || p.e.IsZero()) {
			panic("efgh-Point is NaP with h==0, but ef != 0")
		}
		return true
	}

	if p.g.IsZero() && p.e.IsZero() {
		return true
		// This is for testing only. -- remove / reconsider later; maybe we can avoid NaP-type2.
		// panic("Type-2 efgh NaP encountered")
	}

	if p.f.IsZero() {
		panic("efgh-Point with f==0 and h!=0 encountered")
	}

	return false
}

// AffineExtended returns a copy of the point in affine extended coordinates
func (P *Point_efgh) AffineExtended() (ret Point_axtw) {
	// Note: Going eghj -> axtw directly is cheaper by 1 multiplication compared to going via xtw.
	// The reason is that we normalize first and then compute the t coordinate. This effectively saves comptuing t *= z^-1.
	P.normalize_affine()
	ret.x = P.e
	ret.y = P.h
	ret.t.Mul(&P.e, &P.h)
	return
}

// ExtendedTwistedEdwards returns a copy of the point in extended twisted Edwards coordinates.
func (P *Point_efgh) ExtendedTwistedEdwards() (ret Point_xtw) {
	ret.x.Mul(&P.e, &P.f)
	ret.y.Mul(&P.g, &P.h)
	ret.t.Mul(&P.e, &P.h)
	ret.z.Mul(&P.f, &P.g)
	return
}

func (P *Point_efgh) ToDecaf_xtw() (ret Point_xtw) {
	ret.x.Mul(&P.e, &P.f)
	ret.y.Mul(&P.g, &P.h)
	ret.t.Mul(&P.e, &P.h)
	ret.z.Mul(&P.f, &P.g)
	return
}

func (P *Point_efgh) ToDecaf_axtw() (ret Point_axtw) {
	// TODO ! Review
	// Note: Going eghj -> axtw directly is cheaper by 1 multiplication compared to going via xtw.
	// The reason is that we normalize first and then compute the t coordinate. This effectively saves comptuing t *= z^-1.

	P.normalize_affine()
	ret.x = P.e
	ret.y = P.h
	ret.t.Mul(&P.e, &P.h)
	return
}

// Clone creates a copy of the given point as a CurvePointPtrInterfaceRead. (Be aware that the returned interface value stores a *pointer* of type *Point_efgh)
func (P *Point_efgh) Clone() interface{} {
	p_copy := *P
	return &p_copy
}

/*
// SerializeShort serializes the given point in short serialization format by writing to output. err==nil iff no error occurred.
func (p *Point_efgh) SerializeShort(output io.Writer) (bytes_written int, err error) {
	return default_SerializeShort(p, output)
}

// SerializeLong serializes the given point in long serialization format by writing to output. err==nil iff no error occurred.
func (p *Point_efgh) SerializeLong(output io.Writer) (bytes_written int, err error) {
	return default_SerializeLong(p, output)
}
*/

// String() returns a (somewhat) human-readable string describing the point. Useful for debugging.
func (p *Point_efgh) String() (ret string) {
	ret = "E=" + p.e.String() + " F=" + p.f.String() + " G=" + p.g.String() + " H=" + p.h.String()
	if !legendreCheckE1_FH(p.f, p.h) {
		ret += " [modified by +A]"
	}
	return
}

// z.Add(x,y) computes z = x+y according to the elliptic curve group law.
func (p *Point_efgh) Add(x, y CurvePointPtrInterfaceRead) {
	switch x := x.(type) {
	case *Point_xtw:
		switch y := y.(type) {
		case *Point_xtw:
			p.add_stt(x, y)
		case *Point_axtw:
			p.add_sta(x, y)
		default: // including *Point_efgh
			// TODO !
			panic(0)
			/*
				var y_conv Point_xtw = y.ExtendedTwistedEdwards()
				p.add_stt(x, &y_conv)
			*/
		}
	case *Point_axtw:
		switch y := y.(type) {
		case *Point_xtw:
			p.add_sta(y, x)
		case *Point_axtw:
			p.add_saa(x, y)
		default: // including *Point_efgh
			// TODO !
			panic(0)
			/*
				var y_conv Point_xtw = y.ExtendedTwistedEdwards()
					p.add_sta(&y_conv, x)
			*/
		}
	default:
		// TODO !
		panic(0)
		/*
			var x_conv Point_xtw = x.ExtendedTwistedEdwards()
			switch y := y.(type) {
			case *Point_xtw:
				p.add_stt(&x_conv, y)
			case *Point_axtw:
				p.add_sta(&x_conv, y)
			default: // including *Point_efgh
				var y_conv Point_xtw = y.ExtendedTwistedEdwards()
				p.add_stt(&x_conv, &y_conv)
			}
		*/
	}
}

// z.Sub(x,y) computes z = x-y according to the elliptic curve group law.
func (p *Point_efgh) Sub(x, y CurvePointPtrInterfaceRead) {
	switch x := x.(type) {
	case *Point_xtw:
		switch y := y.(type) {
		case *Point_xtw:
			p.sub_stt(x, y)
		case *Point_axtw:
			p.sub_sta(x, y)
		default:
			// TODO !
			/*
				var y_conv Point_xtw = y.ExtendedTwistedEdwards()
				p.sub_stt(x, &y_conv)
			*/
		}
	case *Point_axtw:
		switch y := y.(type) {
		case *Point_xtw:
			p.sub_sat(x, y)
		case *Point_axtw:
			p.sub_saa(x, y)
		default:
			// TODO !
			/*
				var y_conv Point_xtw = y.ExtendedTwistedEdwards()
				p.sub_sat(x, &y_conv)
			*/
		}
	default:
		// TODO !
		/*
			var x_conv Point_xtw = x.ExtendedTwistedEdwards()
			p.Sub(&x_conv, y)
		*/
	}
}

// z.Double(x) computes z = x+x according to the elliptic curve group law.
func (p *Point_efgh) Double(x CurvePointPtrInterfaceRead) {
	// TODO: improve!
	default_Double(p, x)
}

// z.Neg(x) computes z = -x according to the elliptic curve group law.
func (p *Point_efgh) Neg(input CurvePointPtrInterfaceRead) {
	switch input := input.(type) {
	case *Point_efgh:
		p.neg_ss(input)
	default:
		p.SetFrom(input)
		p.NegEq()
	}
}

// z.Endo(x) compute z = \Psi(x) where \Psi is the non-trivial degree-2 endomorphism described in the bandersnatch paper.
func (p *Point_efgh) Endo(input CurvePointPtrInterfaceRead) {
	switch input := input.(type) {
	case *Point_efgh:
		p.computeEndomorphism_ss(input)
	case *Point_xtw:
		p.computeEndomorphism_st(input)
	case *Point_axtw:
		p.computeEndomorphism_sa(input)
	default:
		// TODO !
		/*
			var input_conv = input.ExtendedTwistedEdwards()
			p.computeEndomorphism_st(&input_conv)
		*/
	}
}

/*
// Endo_FullCurve computes the efficient order-2 endomorphism on the given input point (of any coordinate format).
// This function works even if the input may be a point at infinity; note that the output is never at infinity anyway.
// Be aware that the statement that the endomorpism acts by multiplication by the constant sqrt(2) mod p253 is only meaningful/true on the p253 subgroup.
func (p *Point_efgh) Endo_FullCurve(input CurvePointPtrInterfaceRead_FullCurve) {
	switch input := input.(type) {
	case *Point_efgh:
		if input.IsAtInfinity() {
			*p = orderTwoPoint_efgh
		} else {
			p.computeEndomorphism_ss(input)
		}
	case *Point_axtw:
		p.computeEndomorphism_sa(input)
	case *Point_xtw:
		if input.IsAtInfinity() {
			*p = orderTwoPoint_efgh
		} else {
			p.computeEndomorphism_st(input)
		}
	default:
		if input.IsAtInfinity() {
			*p = orderTwoPoint_efgh
		} else {
			var input_conv = input.ExtendedTwistedEdwards()
			p.computeEndomorphism_st(&input_conv)
		}
	}
}
*/

// SetNeutral sets the Point p to the neutral element of the curve.
func (p *Point_efgh) SetNeutral() {
	*p = NeutralElement_efgh
}

// AddEq adds (via the elliptic curve group addition law) the given curve point x (in any coordinate format) to the received p, overwriting p.
func (p *Point_efgh) AddEq(input CurvePointPtrInterfaceRead) {
	p.Add(p, input)
}

// SubEq subtracts (via the elliptic curve group addition law) the given curve point x (in any coordinate format) from the received p, overwriting p.
func (p *Point_efgh) SubEq(input CurvePointPtrInterfaceRead) {
	p.Sub(p, input)
}

// DoubleEq doubles the received point p, overwriting p.
func (p *Point_efgh) DoubleEq() {
	p.Double(p)
}

// NeqEq replaces the given point by its negative (wrt the elliptic curve group addition law)
func (p *Point_efgh) NegEq() {
	p.e.NegEq()
}

// Note: EndoEq uses Endo, not Endo_FullCurve

// EndoEq applies the endomorphism on the given point. p.EndoEq() is shorthand for p.Endo(&p).
func (p *Point_efgh) EndoEq() {
	p.computeEndomorphism_ss(p)
}

// Note: We usually want to convert FROM efgh to other types, not TO efgh. So this function is rarely used.

// SetFrom initializes the point from the given input point (which may have a different coordinate format)
func (p *Point_efgh) SetFrom(input CurvePointPtrInterfaceRead) {
	switch input := input.(type) {
	case *Point_efgh:
		*p = *input
	case *Point_xtw:
		if !input.z.IsZero() {
			// usual case: This is singular iff input is at infinity (which means y==z==0)
			p.e = input.x
			p.f = input.z
			p.g = input.z
			p.h = input.y
		} else { // Point at infinity or NaP
			// usually equivalent to the above, but singular iff input has x==t==0
			p.e = input.x
			p.f = input.x
			p.g.SetZero() // = input.z
			p.h = input.t
		}
	case *Point_axtw:
		p.e = input.x
		p.f.SetOne()
		p.g.SetOne()
		p.h = input.y
	default:
		// TODO !
		panic(0)
		/*
			if input.IsNaP() {
				napEncountered("Trying to convert NaP of unknown type to efgh", false, input)
				*p = Point_efgh{}
			} else if !input.CanRepresentInfinity() {
				p.e = input.X_projective()
				p.f = input.Z_projective()
				p.g = p.f
				p.h = input.Y_projective()
			} else if !input.(CurvePointPtrInterfaceRead_FullCurve).IsAtInfinity() {
				p.e = input.X_projective()
				p.f = input.Z_projective()
				p.g = p.f
				p.h = input.Y_projective()
			} else {
				// The general interface does not allow to distinguish the two points at infinity.
				// We could fix that, but it seems hardly worth it.
				panic("Trying to convert point at infinity of unknown type into efgh format")
			}
		*/
	}
}

/*
// DeserialzeShort deserialize from the given input byte stream (expecting it to start with a point in short serialization format) and store the result in the receiver.
// err==nil iff no error occured. trusted should be one of the constants TrustedInput or UntrustedInput.
// For UntrustedInput, we perform a specially-tailored efficient curve and subgroup membership tests.
// Note that long format is considerably more efficient to deserialize.
func (p *Point_efgh) DeserializeShort(input io.Reader, trusted IsPointTrusted) (bytes_read int, err error) {
	return default_DeserializeShort(p, input, trusted)
}

// DeserialzeLong deserialize from the given input byte stream (expecting it to start with a point in long serialization format) and store the result in the receiver.
// err==nil iff no error occured. trusted should be one of the constants TrustedInput or UntrustedInput.
// For UntrustedInput, we perform a specially-tailored efficient curve and subgroup membership tests.
// Note that long format is considerably more efficient to deserialize.
func (p *Point_efgh) DeserializeLong(input io.Reader, trusted IsPointTrusted) (bytes_read int, err error) {
	return default_DeserializeLong(p, input, trusted)
}

// DeserialzeAuto deserialize from the given input byte stream (expecting it to start with a point in either short or long serialization format -- it autodetects that) and store the result in the receiver.
// err==nil iff no error occured. trusted should be one of the constants TrustedInput or UntrustedInput.
// For UntrustedInput, we perform a specially-tailored efficient curve and subgroup membership tests.
// Note that long format is considerably more efficient to deserialize.
func (p *Point_efgh) DeserializeAuto(input io.Reader, trusted IsPointTrusted) (bytes_read int, err error) {
	return default_DeserializeAuto(p, input, trusted)
}
*/
