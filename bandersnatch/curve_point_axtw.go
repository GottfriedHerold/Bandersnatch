package bandersnatch

// Point_axtw describes points on the p253-subgroup of the Bandersnatch curve in affine extended twisted Edwards coordinates.
// Extended means that we additionally store T with T = X*Y.
// a Point_axtw with coos x:y:t corresponds to a Point_xtw with coos x:y:t:1 (i.e. with z==1). Note that on the p253 subgroup, all points have z!=0 (and also y!=0).
type Point_axtw struct {
	x FieldElement
	y FieldElement
	t FieldElement
}

// NeutralElement_axtw denotes the Neutral Element of the Bandersnatch curve in affine extended twisted Edwards coordinates.
var NeutralElement_axtw Point_axtw = Point_axtw{x: FieldElementZero, y: FieldElementOne, t: FieldElementZero}

// var orderTwoPoint_axtw Point_axtw = Point_axtw{x: FieldElementZero, y: FieldElementMinusOne, t: FieldElementMinusOne} -- UNUSED

// Note: The general CurvePointPtrInterface ask that calls to <foo>_projective and <foo>_affine must
// note be inteleaved with other calls. This warning is omitted here, as it actually does not apply to Point_axtw.

// X_projective returns the X coordinate of the given point P in projective twisted Edwards coordinates.
// Since Point_axtw stores affine coordinates, this is the same as X_affine()
func (p *Point_axtw) X_projective() FieldElement {
	return p.x
}

// Y_projective returns the Y coordinate of the given point P in projective twisted Edwards coordinates.
// Since Point_axtw stores affine coordinates, this is the same as Y_affine()
func (p *Point_axtw) Y_projective() FieldElement {
	return p.y
}

// T_projective returns the T=X*Y coordinate of the given point P in projective twisted Edwards coordinates.
func (p *Point_axtw) T_projective() FieldElement {
	return p.t
}

// Z_projective returns the Z coordinate of the given point P in projective twisted Edwards coordinates.
// Since Point_axtw stores affine coordinates, this always returns 1.
func (p *Point_axtw) Z_projective() FieldElement {
	return FieldElementOne
}

// X_affine returns the X coordinate of the given point in affine twisted Edwards coordinates, i.e. X/Z
func (p *Point_axtw) X_affine() FieldElement {
	return p.x
}

// Y_affine returns the Y coordinate of the given point in affine twisted Edwards coordinates, i.e. Y/Z
func (p *Point_axtw) Y_affine() FieldElement {
	return p.y
}

// IsNeutralElement checks if the point P is the neutral element of the curve (modulo the identification of P with P+A).
// Use IsNeutralElement_FullCurve if you do not want this identification.
func (p *Point_axtw) IsNeutralElement() bool {

	// NOTE: This is only correct since we work modulo the affine order-2 point (x=0, y=-c, t=0, z=c).
	if p.x.IsZero() {
		if p.y.IsZero() {
			return napEncountered("When checking whether an axtw point is the neutral element, an NaP was encountered", true, p)
		}
		return true
	}
	return false
}

// IsNeutralElement_FullCurve tests for zero-ness like IsNeutralElement. The difference is that it does *NOT* identify P with P+A. We only assume that x,y,t,z satisfy the curve equations.
func (p *Point_axtw) IsNeutralElement_FullCurve() bool {
	if !p.x.IsZero() {
		return false
	}
	if p.y.IsZero() {
		return napEncountered("When checking whether an axtw point is exactly the neutral element, a NaP was encountered", true, p)
	}
	if !p.t.IsZero() {
		panic("axtw Point with x==0, y!=0, t!=0 encountered. This must never happen")
	}
	return p.y.IsOne() // p.y must be either 1 or -1 at this point.
}

// IsEqual compares two curve points for equality, working modulo the P = P + A identification. The two points do not have the be in the same coordinate format.
// TODO/Question: Should we export the variants for specific non-interface types to get more type safety?
func (p *Point_axtw) IsEqual(other CurvePointPtrInterfaceRead) bool {
	switch other := other.(type) {
	case *Point_xtw:
		return p.is_equal_at(other)
	case *Point_axtw:
		return p.is_equal_aa(other)
	default:
		if p.IsNaP() || other.IsNaP() {
			return napEncountered("When comparing an axtw point with another point, a NaP was encountered", true, p, other)
		}
		// We check whether x1/y1 == x2/y2

		var temp1, temp2 FieldElement
		var temp_fe FieldElement = other.Y_projective()
		// Note: p and other cannot alias due to type, so using p is safe between calls to Y_projective and X_projective
		temp1.Mul(&p.x, &temp_fe)
		temp_fe = other.X_projective()
		temp2.Mul(&p.y, &temp_fe)
		return temp1.IsEqual(&temp2)
	}
}

// IsEqual_FullCurve compares two curve points for equality WITHOUT working modulo the P = P+A identification. The two points do not have to be in the same coordinate format.
// This also works if the other point is at infinity (which can only happen if the types differ, of course)
func (p *Point_axtw) IsEqual_FullCurve(other CurvePointPtrInterfaceRead_FullCurve) bool {
	if p.IsNaP() || other.IsNaP() {
		return napEncountered("When comparing an axtw point exactly with another point, a NaP was encountered", true, p, other)
	}
	switch other := other.(type) {
	case *Point_xtw:
		return p.is_equal_exact_at(other)
	case *Point_axtw:
		return p.is_equal_exact_aa(other)
	default:
		other_copy := other.ExtendedTwistedEdwards()
		return p.is_equal_exact_at(&other_copy)
	}
}

// IsAtInfinity tests whether the point is an infinite (neccessarily order-2) point. Since these points cannot be represented in affine coordinates in the first place, this always returns false.
func (p *Point_axtw) IsAtInfinity() bool {
	if p.IsNaP() {
		napEncountered("When chekcking whether an axtw point is infinite, a NaP was encountered", false, p)
		// we also return false in this case (unless the error handler panics).
	}
	return false
}

func (p *Point_axtw) CanRepresentInfinity() bool {
	return false
}

// IsNaP checks whether the point is a NaP (Not-a-point). NaPs must never appear if the library is used correctly. They can appear by
// a) performing operations on points that are not in the correct subgroup or that are NaPs.
// b) zero-initialized points are NaPs (Go lacks constructors to fix that).
// For Point_axtw, NaPs have x==y==0. (Actually, we expect only x==y==t==0 to happen).
func (p *Point_axtw) IsNaP() bool {
	return p.x.IsZero() && p.y.IsZero()
}

// AffineExtended returns a copy of the point in affine extended coordinates (i.e. a copy)
func (p *Point_axtw) AffineExtended() Point_axtw {
	// technically, we could return *p. There is no way for the caller to modify it without copying it on the caller side.
	return Point_axtw{x: p.x, y: p.y, t: p.t}
}

// ExtendedTwistedEdwards returns a copy of the point in extended twisted Edwards coordinates.
func (p *Point_axtw) ExtendedTwistedEdwards() Point_xtw {
	return Point_xtw{x: p.x, y: p.y, t: p.t, z: FieldElementOne}
}

// Clone creates a copy of the given point as a CurvePointPtrInterfaceRead. (Be aware that the returned interface value stores a pointer)
func (p *Point_axtw) Clone() CurvePointPtrInterfaceRead {
	p_copy := *p
	return &p_copy
}

// Point_axtw::SerializeShort, Point_axtw::SerializeLong and Point_axtw::SerializeAuto are defined directly in curve_point_impl_serialize.go

// String prints the point in X:Y:T - format
func (p *Point_axtw) String() string {
	// Not the most efficient way, but good enough.
	return p.x.String() + ":" + p.y.String() + ":" + p.t.String()
}

// SetFrom initializes the point from the given input point (which may have a different coordinate format)
func (p *Point_axtw) SetFrom(input CurvePointPtrInterfaceRead) {
	*p = input.AffineExtended()
}

// Add performs curve point addition according to the group law.
// Use p.Add(&x, &y) for p := x + y.
// TODO: Export variants for specific types
func (p *Point_axtw) Add(x, y CurvePointPtrInterfaceRead) {
	var temp Point_efgh
	temp.Add(x, y)
	*p = temp.AffineExtended()
}

// Sub performs curve point addition according to the group law.
// Use p.Sub(&x, &y) for p := x - y.
func (p *Point_axtw) Sub(x, y CurvePointPtrInterfaceRead) {
	var temp Point_efgh
	temp.Sub(x, y)
	*p = temp.AffineExtended()
}

func (p *Point_axtw) Double(in CurvePointPtrInterfaceRead) {
	// TODO: Use specialized formulas.
	p.Add(in, in)
}

// Neg computes the negative of the point wrt the elliptic curve group law.
// Use p.Neg(&input) for p := -input.
func (p *Point_axtw) Neg(input CurvePointPtrInterfaceRead) {
	switch input := input.(type) {
	case *Point_axtw:
		p.x.Neg(&input.x)
		p.y = input.y
		p.t.Neg(&input.t)
	case *Point_xtw:
		*p = input.AffineExtended()
		p.NegEq()
	case *Point_efgh:
		*p = input.AffineExtended()
		p.NegEq()
	default:
		*p = input.AffineExtended()
		p.NegEq()
	}
}

// Endo computes the efficient order-2 endomorphism on the given point.
func (p *Point_axtw) Endo(input CurvePointPtrInterfaceRead) {
	var temp Point_efgh
	temp.Endo(input)
	*p = temp.AffineExtended()
}

// Endo_FullCurve computes the efficient order-2 endomorphism on the given input point (of any coordinate format).
// This function works even if the input may be a point at infinity; note that the output is never at infinity anyway.
// Be aware that the statement that the endomorpism acts by multiplication by the constant sqrt(2) mod p253 is only meaningful/true on the p253 subgroup.
func (p *Point_axtw) Endo_FullCurve(input CurvePointPtrInterfaceRead_FullCurve) {
	var temp Point_efgh
	temp.Endo_FullCurve(input)
	*p = temp.AffineExtended()
}

// SetNeutral sets the Point p to the neutral element of the curve.
func (p *Point_axtw) SetNeutral() {
	*p = NeutralElement_axtw
}

// AddEq adds (via the elliptic curve group addition law) the given curve point x (in any coordinate format) to the received p, overwriting p.
func (p *Point_axtw) AddEq(x CurvePointPtrInterfaceRead) {
	p.Add(p, x)
}

// SubEq subtracts (via the elliptic curve group addition law) the given curve point x (in any coordinate format) from the received p, overwriting p.
func (p *Point_axtw) SubEq(x CurvePointPtrInterfaceRead) {
	p.Sub(p, x)
}

// DoubleEq doubles the received point p, overwriting p.
func (p *Point_axtw) DoubleEq() {
	var temp Point_efgh
	temp.add_saa(p, p)
	*p = temp.AffineExtended()
}

// NeqEq replaces the given point by its negative (wrt the elliptic curve group addition law)
func (p *Point_axtw) NegEq() {
	p.x.NegEq()
	p.t.NegEq()
}

// EndoEq applies the endomorphism on the given point. p.EndoEq() is shorthand for p.Endo(&p).
func (p *Point_axtw) EndoEq() {
	var temp Point_efgh
	temp.computeEndomorphism_sa(p)
	*p = temp.AffineExtended()
}
