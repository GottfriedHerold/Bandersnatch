package bandersnatch

import (
	"math/rand"
)

type point_axtw_base struct {
	thisCurvePointCannotRepresentInfinity
	thisCurvePointCanRepresentFullCurve
	x FieldElement
	y FieldElement
	t FieldElement
}

// Point_axtw describes points on the p253-subgroup of the Bandersnatch curve in affine extended twisted Edwards coordinates.
// Extended means that we additionally store T with T = X*Y.
// a Point_axtw with coos x:y:t corresponds to a Point_xtw with coos x:y:t:1 (i.e. with z==1). Note that on the p253 subgroup, all points have z!=0 (and also y!=0).
type Point_axtw_subgroup struct {
	thisCurvePointCanOnlyRepresentSubgroup
	point_axtw_base
}

type Point_axtw_full struct {
	point_axtw_base
}

// NeutralElement_axtw denotes the Neutral Element of the Bandersnatch curve in affine extended twisted Edwards coordinates.
var NeutralElement_axtw point_axtw_base = point_axtw_base{x: FieldElementZero, y: FieldElementOne, t: FieldElementZero}
var OrderTwoPoint_axtw point_axtw_base = point_axtw_base{x: FieldElementZero, y: FieldElementMinusOne, t: FieldElementZero}

func (p *point_axtw_base) normalizeSubgroup() {
	if !legendreCheckE1_affineY(p.y) {
		p.flipDecaf()
	}
}

func (p *Point_axtw_full) normalizeSubgroup() {
	panic("Calling normalize subgroup on Point_axtw_full (included via struct embedding). This is probably a mistake.")
}

func (p *point_axtw_base) flipDecaf() {
	p.x.NegEq()
	p.y.NegEq()
}

func (p *Point_axtw_subgroup) HasDecaf() bool {
	return true
}

func (p *point_axtw_base) rerandomizeRepresentation(rnd *rand.Rand) {
	// do nothing
}

func (p *Point_axtw_subgroup) rerandomizeRepresentation(rnd *rand.Rand) {
	if rnd.Intn(2) == 0 {
		p.flipDecaf()
	}
}

// Note: The general CurvePointPtrInterface ask that calls to <foo>_projective and <foo>_affine must
// note be interleaved with other calls. This warning is omitted here, as it actually does not apply to Point_axtw.

// X_projective returns the X coordinate of the given point P in projective twisted Edwards coordinates.
// Since Point_axtw stores affine coordinates, this is the same as X_affine()
func (p *Point_axtw_subgroup) X_projective() FieldElement {
	p.normalizeSubgroup()
	return p.x
}

func (p *Point_axtw_full) X_projective() FieldElement {
	return p.x
}

func (p *point_axtw_base) X_decaf_projective() FieldElement {
	return p.x
}

// Y_projective returns the Y coordinate of the given point P in projective twisted Edwards coordinates.
// Since Point_axtw stores affine coordinates, this is the same as Y_affine()
func (p *Point_axtw_subgroup) Y_projective() FieldElement {
	p.normalizeSubgroup()
	return p.y
}

func (p *Point_axtw_full) Y_projective() FieldElement {
	return p.y
}

func (p *point_axtw_base) Y_decaf_projective() FieldElement {
	return p.y
}

// T_projective returns the T=X*Y coordinate of the given point P in projective twisted Edwards coordinates.
func (p *Point_axtw_subgroup) T_projective() FieldElement {
	p.normalizeSubgroup()
	return p.t
}

func (p *Point_axtw_full) T_projective() FieldElement {
	return p.t
}

func (p *point_axtw_base) T_decaf_projective() FieldElement {
	return p.t
}

// Z_projective returns the Z coordinate of the given point P in projective twisted Edwards coordinates.
// Since Point_axtw stores affine coordinates, this always returns 1.
func (p *point_axtw_base) Z_projective() FieldElement {
	return FieldElementOne
}

func (p *point_axtw_base) Z_decaf_projective() FieldElement {
	return FieldElementOne
}

func (p *Point_axtw_subgroup) XYZ_projective() (FieldElement, FieldElement, FieldElement) {
	p.normalizeSubgroup()
	return p.x, p.y, FieldElementOne
}

func (p *Point_axtw_full) XYZ_projective() (FieldElement, FieldElement, FieldElement) {
	return p.x, p.y, FieldElementOne
}

func (p *Point_axtw_subgroup) XYTZ_projective() (FieldElement, FieldElement, FieldElement, FieldElement) {
	p.normalizeSubgroup()
	return p.x, p.y, p.t, FieldElementOne
}

func (p *Point_axtw_full) XYTZ_projective() (FieldElement, FieldElement, FieldElement, FieldElement) {
	return p.x, p.y, p.t, FieldElementOne
}

// X_affine returns the X coordinate of the given point in affine twisted Edwards coordinates, i.e. X/Z
func (p *Point_axtw_subgroup) X_affine() FieldElement {
	p.normalizeSubgroup()
	return p.x
}

// X_affine returns the X coordinate of the given point in affine twisted Edwards coordinates, i.e. X/Z
func (p *Point_axtw_full) X_affine() FieldElement {
	return p.x
}

// Y_affine returns the Y coordinate of the given point in affine twisted Edwards coordinates, i.e. Y/Z
func (p *Point_axtw_subgroup) Y_affine() FieldElement {
	p.normalizeSubgroup()
	return p.y
}

// Y_affine returns the Y coordinate of the given point in affine twisted Edwards coordinates, i.e. Y/Z
func (p *Point_axtw_full) Y_affine() FieldElement {
	return p.y
}

func (p *Point_axtw_subgroup) XY_affine() (FieldElement, FieldElement) {
	p.normalizeSubgroup()
	return p.x, p.y
}

func (p *Point_axtw_full) XY_affine() (FieldElement, FieldElement) {
	return p.x, p.y
}

func (p *Point_axtw_subgroup) XYT_affine() (FieldElement, FieldElement, FieldElement) {
	p.normalizeSubgroup()
	return p.x, p.y, p.t
}

func (p *Point_axtw_full) XYT_affine() (FieldElement, FieldElement, FieldElement) {
	return p.x, p.y, p.t
}

func (p *point_axtw_base) X_decaf_affine() FieldElement {
	return p.x
}

func (p *point_axtw_base) Y_decaf_affine() FieldElement {
	return p.y
}

func (p *point_axtw_base) T_decaf_affine() FieldElement {
	return p.t
}

// IsNeutralElement checks if the point P is the neutral element of the curve.
func (p *Point_axtw_subgroup) IsNeutralElement() bool {
	// NOTE: This is only correct since we work modulo the affine order-2 point (x=0, y=-c, t=0, z=c).
	if p.x.IsZero() {
		if p.y.IsZero() {
			return napEncountered("When checking whether an axtw point is the neutral element, an NaP was encountered", true, p)
		}
		return true
	}
	return false
}

// IsNeutralElement tests if the point is the neutral element.
func (p *Point_axtw_full) IsNeutralElement() bool {
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

// IsEqual compares two curve points for equality.
// The two points do not have the be in the same coordinate format.
func (p *Point_axtw_subgroup) IsEqual(other CurvePointPtrInterfaceRead) bool {
	if p.IsNaP() || other.IsNaP() {
		return napEncountered("When comparing an axtw point with another point, a NaP was encountered", true, p, other)
	}
	switch other := other.(type) {
	case *Point_xtw_subgroup:
		ret, _ := p.isEqual_moduloA_at(&other.point_xtw_base)
		return ret
	case *Point_xtw_full:
		p.normalizeSubgroup()
		return p.isEqual_exact_at(&other.point_xtw_base)
	case *Point_axtw_subgroup:
		return p.isEqual_moduloA_aa(&other.point_axtw_base)
	case *Point_axtw_full:
		p.normalizeSubgroup()
		return p.isEqual_exact_aa(&other.point_axtw_base)
	default:
		if other.CanOnlyRepresentSubgroup() {
			return p.isEqual_moduloA_aany(other)
		} else {
			p.normalizeSubgroup()
			return p.isEqual_exact_aany(other)
		}
	}
}

func (p *Point_axtw_full) IsEqual(other CurvePointPtrInterfaceRead) bool {
	if p.IsNaP() || other.IsNaP() {
		return napEncountered("When comparing an axtw point with another point, a NaP was encountered", true, p, other)
	}
	switch other := other.(type) {
	case *Point_xtw_subgroup:
		other.normalizeSubgroup()
		return p.isEqual_exact_at(&other.point_xtw_base)
	case *Point_xtw_full:
		return p.isEqual_exact_at(&other.point_xtw_base)
	case *Point_axtw_subgroup:
		other.normalizeSubgroup()
		return p.isEqual_exact_aa(&other.point_axtw_base)
	case *Point_axtw_full:
		return p.isEqual_exact_aa(&other.point_axtw_base)
	default:
		return p.isEqual_exact_aany(other)
	}
}

// IsAtInfinity tests whether the point is an infinite (neccessarily order-2) point. Since these points cannot be represented in affine coordinates in the first place, this always returns false.
func (p *point_axtw_base) IsAtInfinity() bool {
	if p.IsNaP() {
		napEncountered("When checking whether an axtw point is infinite, a NaP was encountered", false, p)
		// we also return false in this case (unless the error handler panics).
	}
	return false
}

// IsNaP checks whether the point is a NaP (Not-a-point). NaPs must never appear if the library is used correctly. They can appear by
// a) performing operations on points that are not in the correct subgroup or that are NaPs.
// b) zero-initialized points are NaPs (Go lacks constructors to fix that).
// For Point_axtw, NaPs have x==y==0. (Actually, we expect only x==y==t==0 to happen).
func (p *point_axtw_base) IsNaP() bool {
	return p.x.IsZero() && p.y.IsZero()
}

func (p *point_axtw_base) ToDecaf_xtw() point_xtw_base {
	return point_xtw_base{x: p.x, y: p.y, t: p.t, z: FieldElementOne}
}

func (p *point_axtw_base) ToDecaf_axtw() (ret point_axtw_base) {
	ret = *p
	return
}

func (p *Point_axtw_full) IsInSubgroup() bool {
	return legendreCheckA_affineX(p.x) && legendreCheckE1_affineY(p.y)
}

// Clone creates a copy of the given point as an interface. (Be aware that the returned interface value stores a pointer)
func (p *point_axtw_base) Clone() interface{} {
	p_copy := *p
	return &p_copy
}

// Clone creates a copy of the given point as an interface. (Be aware that the returned interface value stores a pointer)
func (p *Point_axtw_subgroup) Clone() interface{} {
	p_copy := *p
	return &p_copy
}

// Clone creates a copy of the given point as an interface. (Be aware that the returned interface value stores a pointer)
func (p *Point_axtw_full) Clone() interface{} {
	p_copy := *p
	return &p_copy
}

// Point_axtw::SerializeShort, Point_axtw::SerializeLong and Point_axtw::SerializeAuto are defined directly in curve_point_impl_serialize.go

// String prints the point in X:Y:T - format
func (p *point_axtw_base) String() string {
	// Not the most efficient way, but good enough.
	return p.x.String() + ":" + p.y.String() + ":" + p.t.String()
}
func (p *Point_axtw_subgroup) String() (ret string) {
	ret = p.point_axtw_base.String()
	if !legendreCheckE1_affineY(p.y) {
		ret += " [+A]"
	}
	return
}

// SetFrom initializes the point from the given input point (which may have a different coordinate format)
func (p *Point_axtw_subgroup) SetFrom(input CurvePointPtrInterfaceRead) {
	switch input := input.(type) {
	case *Point_efgh_subgroup:
		p.point_axtw_base = input.ToDecaf_axtw()
	case *Point_axtw_subgroup:
		*p = *input
	case *Point_xtw_subgroup:
		// This check is mostly to avoid potential division by zero.
		if input.IsNaP() {
			napEncountered("Converting NaP from xtw to axtw", false, input)
			*p = Point_axtw_subgroup{}
			return
		}
		var zInv FieldElement // cannot be zero unless input is NaP, which was handled above
		zInv.Inv(&input.z)
		p.x.Mul(&zInv, &input.x)
		p.y.Mul(&zInv, &input.y)
		p.t.Mul(&zInv, &input.t)
	default:
		ensureSubgroupOnly(input)
		if input.IsNaP() {
			napEncountered("Converting NaP to axtw", false, input)
			*p = Point_axtw_subgroup{}
			return
		}
		p.x = input.X_decaf_affine()
		p.y = input.Y_decaf_affine()
		p.t = input.T_decaf_affine()
	}
}

// SetFrom initializes the point from the given input point (which may have a different coordinate format)
func (p *Point_axtw_full) SetFrom(input CurvePointPtrInterfaceRead) {
	switch input := input.(type) {
	case *Point_efgh_full:
		p.point_axtw_base = input.ToDecaf_axtw()
	case *Point_efgh_subgroup:
		input.normalizeSubgroup()
		p.point_axtw_base = input.ToDecaf_axtw()
	case CurvePointPtrInterfaceCooReadAffineT:
		if input.(CurvePointPtrInterfaceRead).IsNaP() {
			napEncountered("Converting NaP to axtw_full", false, input.(CurvePointPtrInterfaceRead))
			*p = Point_axtw_full{}
			return
		}
		p.x, p.y, p.t = input.XYT_affine()
	default:
		if input.(CurvePointPtrInterfaceRead).IsNaP() {
			napEncountered("Converting NaP to axtw_full", false, input.(CurvePointPtrInterfaceRead))
			*p = Point_axtw_full{}
			return
		}
		p.x, p.y = input.XY_affine()
		p.t.Mul(&p.x, &p.y)
	}
}

// Add performs curve point addition according to the group law.
// Use p.Add(&x, &y) for p := x + y.
// TODO: Export variants for specific types
func (p *Point_axtw_subgroup) Add(x, y CurvePointPtrInterfaceRead) {
	var temp Point_efgh_subgroup
	temp.Add(x, y)
	p.point_axtw_base = temp.ToDecaf_axtw()
}

func (p *Point_axtw_full) Add(x, y CurvePointPtrInterfaceRead) {
	var temp Point_efgh_full
	temp.Add(x, y)
	p.x, p.y, p.t = temp.XYT_affine()
}

// Sub performs curve point addition according to the group law.
// Use p.Sub(&x, &y) for p := x - y.
func (p *Point_axtw_subgroup) Sub(x, y CurvePointPtrInterfaceRead) {
	var temp Point_efgh_subgroup
	temp.Sub(x, y)
	p.point_axtw_base = temp.ToDecaf_axtw()
}

func (p *Point_axtw_full) Sub(x, y CurvePointPtrInterfaceRead) {
	var temp Point_efgh_full
	temp.Sub(x, y)
	p.x, p.y, p.t = temp.XYT_affine()
}

// Double computes the sum of a point with itself. p.double(x) means p := x + x
func (p *point_axtw_base) Double(in CurvePointPtrInterfaceRead) {
	var temp point_efgh_base
	temp.Double(in)
	*p = temp.ToDecaf_axtw()
}

// Neg computes the negative of the point wrt the elliptic curve group law.
// Use p.Neg(&input) for p := -input.
func (p *Point_axtw_subgroup) Neg(input CurvePointPtrInterfaceRead) {
	p.SetFrom(input)
	p.NegEq()
}

func (p *Point_axtw_full) Neg(input CurvePointPtrInterfaceRead) {
	p.SetFrom(input)
	p.NegEq()
}

// Endo computes the efficient order-2 endomorphism on the given point.
func (p *Point_axtw_subgroup) Endo(input CurvePointPtrInterfaceRead) {
	var temp Point_efgh_subgroup
	temp.Endo(input)
	p.point_axtw_base = temp.ToDecaf_axtw()
}

func (p *Point_axtw_full) Endo(input CurvePointPtrInterfaceRead) {
	var temp Point_efgh_full
	temp.Endo(input)
	p.x, p.y, p.t = temp.XYT_affine()
}

// SetNeutral sets the Point p to the neutral element of the curve.
func (p *point_axtw_base) SetNeutral() {
	*p = NeutralElement_axtw
}

// AddEq adds (via the elliptic curve group addition law) the given curve point x (in any coordinate format) to the received p, overwriting p.
func (p *Point_axtw_full) AddEq(x CurvePointPtrInterfaceRead) {
	p.Add(p, x)
}

// AddEq adds (via the elliptic curve group addition law) the given curve point x (in any coordinate format) to the received p, overwriting p.
func (p *Point_axtw_subgroup) AddEq(x CurvePointPtrInterfaceRead) {
	p.Add(p, x)
}

// SubEq subtracts (via the elliptic curve group addition law) the given curve point x (in any coordinate format) from the received p, overwriting p.
func (p *Point_axtw_subgroup) SubEq(x CurvePointPtrInterfaceRead) {
	p.Sub(p, x)
}

// SubEq subtracts (via the elliptic curve group addition law) the given curve point x (in any coordinate format) from the received p, overwriting p.
func (p *Point_axtw_full) SubEq(x CurvePointPtrInterfaceRead) {
	p.Sub(p, x)
}

// DoubleEq doubles the received point p, overwriting p.
func (p *point_axtw_base) DoubleEq() {
	var temp point_efgh_base
	temp.add_saa(p, p)
	*p = temp.ToDecaf_axtw()
}

// NeqEq replaces the given point by its negative (wrt the elliptic curve group addition law)
func (p *point_axtw_base) NegEq() {
	p.x.NegEq()
	p.t.NegEq()
}

// EndoEq applies the endomorphism on the given point. p.EndoEq() is shorthand for p.Endo(&p).
func (p *Point_axtw_subgroup) EndoEq() {
	p.Endo(p)
}

func (p *Point_axtw_full) EndoEq() {
	p.Endo(p)
}

func (p *point_axtw_base) Validate() bool {
	return p.isPointOnCurve()
}

func (p *Point_axtw_subgroup) Validate() bool {
	return p.point_axtw_base.isPointOnCurve() && legendreCheckA_affineX(p.x)
}

func (p *Point_axtw_full) sampleRandomUnsafe(rnd *rand.Rand) {
	p.point_axtw_base = makeRandomPointOnCurve_a(rnd)
}

func (p *Point_axtw_subgroup) sampleRandomUnsafe(rnd *rand.Rand) {
	p.point_axtw_base = makeRandomPointOnCurve_a(rnd)
	p.point_axtw_base.DoubleEq()
}

func (p *Point_axtw_full) SetAffineTwoTorsion() {
	p.point_axtw_base = OrderTwoPoint_axtw
}
