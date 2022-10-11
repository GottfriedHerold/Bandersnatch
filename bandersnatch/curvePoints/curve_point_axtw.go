package curvePoints

import (
	"math/rand"
)

// This file contains the implementation of the CurvePointPtrInterface by types
// Point_axtw_subgroup and Point_axtw_full.
// These represent points on the bandersnatch curve in *affine* (extended) coordinates.
// The _subgroup resp. _full versions differ in whether they only store elements on the prime-order subgroup;
//
// NOTE: As of now, points at infinity cannot be stored by this (which is an issue only for the _full version, of course)
// NOTE2: As of now, representations are non-unique

// point_axtw_base is a struct holding x,y,t values that can be used to represent an elliptic curve point on the Bandersnatch curve.
// Note that this is just a container for coordinates. It (or pointers to it) does not satisfy the CurvePointPtrInterface.
// Indeed, there is the question how to interpret x,y,t coordinates as coos of a point and depending on context, we
// either work modulo A or not.
// We use struct embedding to create point_axtw_subgroup and point_axtw_full from it.
type point_axtw_base struct {
	thisCurvePointCannotRepresentInfinity
	thisCurvePointCanRepresentFullCurve
	x FieldElement
	y FieldElement
	t FieldElement
}

// Point_axtw_subgroup describes points on the p253-subgroup of the Bandersnatch curve in affine extended twisted Edwards coordinates.
// Extended means that we additionally store T with T = X*Y.
// This type can only hold elements from the p253 subgroup.
type Point_axtw_subgroup struct {
	thisCurvePointCanOnlyRepresentSubgroup
	// This stores x,y,t coordinate.
	point_axtw_base
}

// Point_axtw_full describes a non-infinite rational point of the Bandersnatch elliptic curve in affine extended twisted Edwards coordinates.
// Extended means that we additionally store T with T = X*Y.
// Note that, being a twisted Edwards curve, the neutral element is NOT at infinity.
// The two rational points that cannot be represented are 2-torsion points outside the prime-order subgroup.
// Performing operations that would store a point at infinity of this type result in a panic.
type Point_axtw_full struct {
	point_axtw_base
}

// neutralElement_axtwbase are coordinates of the neutral element in axtw form.
var neutralElement_axtwbase point_axtw_base = point_axtw_base{x: fieldElementZero, y: fieldElementOne, t: fieldElementZero}

// affineOrderTwoPoint_axtwbase is the affine point of order two
var affineOrderTwoPoint_axtwbase point_axtw_base = point_axtw_base{x: fieldElementZero, y: fieldElementMinusOne, t: fieldElementZero}

// NeutralElement_axtw_full denotes the neutral element of the Bandersnatch curve (with type Point_axtw_full).
var NeutralElement_axtw_full Point_axtw_full = Point_axtw_full{point_axtw_base: neutralElement_axtwbase}

// NeutralElement_axtw_subgroup denotes the neutral element of the Bandersnatch curve (with type Point_axtw_subgroup).
var NeutralElement_axtw_subgroup Point_axtw_subgroup = Point_axtw_subgroup{point_axtw_base: neutralElement_axtwbase}

// AffineOrderTwoPoint_axtw denotes the affine point of order two of the Bandersnatch curve (with type Point_axtw_full). This point is not on the p253 prime order subgroup.
var AffineOrderTwoPoint_axtw Point_axtw_full = Point_axtw_full{point_axtw_base: affineOrderTwoPoint_axtwbase}

// normalizeSubgroup changes the internal representation of p, such that the coordinates of p
// corresponds exactly to p (without working modulo the affine two-torsion point).
func (p *Point_axtw_subgroup) normalizeSubgroup() {
	if !legendreCheckE1_affineY(p.y) {
		p.flipDecaf()
	}
}

// might define on point_axtw_base

// flipDecaf changes the internal representation of P from P to P+A or vice versa (note that we work modulo A).
// flipDecaf is needed to satisfy the (optional) curvePointPtrInterfaceDecaf interface that is recognized by the testing framework.
func (p *Point_axtw_subgroup) flipDecaf() {
	p.x.NegEq()
	p.y.NegEq()
}

// TODO: Might go away from requirements completely.

// NOTE: point_axtw_full and point_axtw_base get HasDecaf from struct-embedded thisCurvePointCanRepresentFullCurve

// HasDecaf needs to return true for flipDecaf to be recognized by the testing framework.
func (p *Point_axtw_subgroup) HasDecaf() bool {
	return true
}

// rerandomizeRepresentation is needed to satisfy the CurvePointPtrInterfaceTestSample interface for testing. It changes the internal representation to an equivalent one.
func (p *point_axtw_base) rerandomizeRepresentation(rnd *rand.Rand) {
	// do nothing: The representation is actually unique (up to internal representation of field elements)
	// TODO: Rerandomize field elements?
}

// rerandomizeRepresentation is needed to satisfy the CurvePointPtrInterfaceTestSample interface for testing. It changes the internal representation to an equivalent one.
func (p *Point_axtw_subgroup) rerandomizeRepresentation(rnd *rand.Rand) {
	if rnd.Intn(2) == 0 {
		p.flipDecaf()
	}
}

// X_projective returns the X coordinate of the given point in projective twisted Edwards coordinates.
//
// CAVEAT: Subsequent calls to any <foo>_projective methods on the same point are only guaranteed to be consistent if nothing else is done with the point between the calls.
// This includes that you may not be able to use the point as argument to even seemingly read-only methods, as these might touch the redundant internal representation.
func (p *Point_axtw_subgroup) X_projective() FieldElement {
	// Since Point_axtw stores affine coordinates, this is the same as X_affine()
	p.normalizeSubgroup()
	return p.x
}

// X_projective returns the X coordinate of the given point in projective twisted Edwards coordinates.
//
// CAVEAT: Subsequent calls to any <foo>_projective methods on the same point are only guaranteed to be consistent if nothing else is done with the point between the calls.
// This includes that you may not be able to use the point as argument to even seemingly read-only methods, as these might touch the redundant internal representation.
func (p *Point_axtw_full) X_projective() FieldElement {
	return p.x
}

// X_decaf_projective returns the X coordinate of either P or P+A in projective twisted Edwards coordinates, where A is the affine point of order two.
//
// CAVEAT: Subsequent calls to any <foo>_decaf_projective methods are only guaranteed to be consistent if nothing else is done with the point between those calls.
// The consistency guarantee includes that different <foo>_decaf_projective methods make the same P vs. P+A choice.
// The requirements include not using the point as (pointer) argument to seemingly read-only methods (as these might change the internal representation) and not using <foo>_decaf_affine methods.
//
// Note: If P has extended projective Edwards coordinates X:Y:T:Z, then P+A has coordinates -X:-Y:T:Z == X:Y:-T:-Z
func (p *point_axtw_base) X_decaf_projective() FieldElement {
	return p.x
}

// Y_projective returns the Y coordinate of the given point in projective twisted Edwards coordinates.
//
// CAVEAT: Subsequent calls to any <foo>_projective methods on the same point are only guaranteed to be consistent if nothing else is done with the point between the calls.
// This includes that you may not be able to use the point as argument to even seemingly read-only methods, as these might touch the redundant internal representation.
func (p *Point_axtw_subgroup) Y_projective() FieldElement {
	p.normalizeSubgroup()
	return p.y
}

// Y_projective returns the Y coordinate of the given point in projective twisted Edwards coordinates.
//
// CAVEAT: Subsequent calls to any <foo>_projective methods on the same point are only guaranteed to be consistent if nothing else is done with the point between the calls.
// This includes that you may not be able to use the point as argument to even seemingly read-only methods, as these might touch the redundant internal representation.
func (p *Point_axtw_full) Y_projective() FieldElement {
	return p.y
}

// Y_decaf_projective returns the Y coordinate of either P or P+A in projective twisted Edwards coordinates, where A is the affine point of order two.
//
// CAVEAT: Subsequent calls to any <foo>_decaf_projective methods are only guaranteed to be consistent if nothing else is done with the point between those calls.
// The consistency guarantee includes that different <foo>_decaf_projective methods make the same P vs. P+A choice.
// The requirements include not using the point as (pointer) argument to seemingly read-only methods (as these might change the internal representation) and not using <foo>_decaf_affine methods.
//
// Note: If P has extended projective Edwards coordinates X:Y:T:Z, then P+A has coordinates -X:-Y:T:Z == X:Y:-T:-Z
func (p *point_axtw_base) Y_decaf_projective() FieldElement {
	return p.y
}

// T_projective returns the T coordinate of the given point in projective twisted Edwards coordinates. This coordinate satisfies X*Y = T*Z.
//
// CAVEAT: Subsequent calls to any <foo>_projective methods on the same point are only guaranteed to be consistent if nothing else is done with the point between the calls.
// This includes that you may not be able to use the point as argument to even seemingly read-only methods, as these might touch the redundant internal representation.
func (p *Point_axtw_subgroup) T_projective() FieldElement {
	p.normalizeSubgroup()
	return p.t
}

// T_projective returns the T coordinate of the given point in projective twisted Edwards coordinates. This coordinate satisfies X*Y = T*Z.
//
// CAVEAT: Subsequent calls to any <foo>_projective methods on the same point are only guaranteed to be consistent if nothing else is done with the point between the calls.
// This includes that you may not be able to use the point as argument to even seemingly read-only methods, as these might touch the redundant internal representation.
func (p *Point_axtw_full) T_projective() FieldElement {
	return p.t
}

// T_decaf_projective returns the T coordinate of either P or P+A in projective twisted Edwards coordinates, where A is the affine point of order two.
//
// CAVEAT: Subsequent calls to any <foo>_decaf_projective methods are only guaranteed to be consistent if nothing else is done with the point between those calls.
// The consistency guarantee includes that different <foo>_decaf_projective methods make the same P vs. P+A choice.
// The requirements include not using the point as (pointer) argument to seemingly read-only methods (as these might change the internal representation) and not using <foo>_decaf_affine methods.
//
// Note: If P has extended projective Edwards coordinates X:Y:T:Z, then P+A has coordinates -X:-Y:T:Z == X:Y:-T:-Z
func (p *point_axtw_base) T_decaf_projective() FieldElement {
	return p.t
}

// Z_projective returns the Z coordinate of the given point in projective twisted Edwards coordinates.
//
// CAVEAT: Subsequent calls to any <foo>_projective methods on the same point are only guaranteed to be consistent if nothing else is done with the point between the calls.
// This includes that you may not be able to use the point as argument to even seemingly read-only methods, as these might touch the redundant internal representation.
// For Point_axtw_full and Point_axtw_subgroup, this always returns 1.
func (p *point_axtw_base) Z_projective() FieldElement {
	return fieldElementOne
}

// Z_decaf_projective returns the Z coordinate of either P or P+A in projective twisted Edwards coordinates, where A is the affine point of order two.
//
// CAVEAT: Subsequent calls to any <foo>_decaf_projective methods are only guaranteed to be consistent if nothing else is done with the point between those calls.
// The consistency guarantee includes that different <foo>_decaf_projective methods make the same P vs. P+A choice.
// The requirements include not using the point as (pointer) argument to seemingly read-only methods (as these might change the internal representation) and not using <foo>_decaf_affine methods.
//
// Note: If P has extended projective Edwards coordinates X:Y:T:Z, then P+A has coordinates -X:-Y:T:Z == X:Y:-T:-Z
// For Point_axt_subgroup and Point_axtw_full, Z_decaf_projective always returns 1.
func (p *point_axtw_base) Z_decaf_projective() FieldElement {
	return fieldElementOne
}

// XYZ_projective returns X,Y and Z coordinates in projective twisted Edwards coordinates in a single call.
// It is equivalent to calling X_projective(), Y_projective(), Z_projective(), but more efficient.
func (p *Point_axtw_subgroup) XYZ_projective() (FieldElement, FieldElement, FieldElement) {
	p.normalizeSubgroup()
	return p.x, p.y, fieldElementOne
}

// XYZ_projective returns X,Y and Z coordinates in projective twisted Edwards coordinates in a single call.
// It is equivalent to calling X_projective(), Y_projective(), Z_projective(), but may be more efficient.
func (p *Point_axtw_full) XYZ_projective() (FieldElement, FieldElement, FieldElement) {
	return p.x, p.y, fieldElementOne
}

// XYTZ_projective returns X,Y,T and Z coordinates in projective twisted Edwards coordinates in a single call.
// It is equivalent to calling X_projective(), Y_projective(), T_projective(), Z_projective(), but considerably more efficient.
func (p *Point_axtw_subgroup) XYTZ_projective() (FieldElement, FieldElement, FieldElement, FieldElement) {
	p.normalizeSubgroup()
	return p.x, p.y, p.t, fieldElementOne
}

// XYTZ_projective returns X,Y,T and Z coordinates in projective twisted Edwards coordinates in a single call.
// It is equivalent to calling X_projective(), Y_projective(), T_projective(), Z_projective(), but may be more efficient.
func (p *Point_axtw_full) XYTZ_projective() (FieldElement, FieldElement, FieldElement, FieldElement) {
	return p.x, p.y, p.t, fieldElementOne
}

// X_affine returns the X coordinate of the given point in affine twisted Edwards coordinates, i.e. X/Z.
func (p *Point_axtw_subgroup) X_affine() FieldElement {
	p.normalizeSubgroup()
	return p.x
}

// X_affine returns the X coordinate of the given point in affine twisted Edwards coordinates, i.e. X/Z.
func (p *Point_axtw_full) X_affine() FieldElement {
	return p.x
}

// Y_affine returns the Y coordinate of the given point in affine twisted Edwards coordinates, i.e. Y/Z.
func (p *Point_axtw_subgroup) Y_affine() FieldElement {
	p.normalizeSubgroup()
	return p.y
}

// Y_affine returns the Y coordinate of the given point in affine twisted Edwards coordinates, i.e. Y/Z.
func (p *Point_axtw_full) Y_affine() FieldElement {
	return p.y
}

// XY_affine returns the X and Y coordinate of the given point in affine twisted Edwards coordinates. It is equivalent to calling X_affine and Y_affine, but may be more efficient.
func (p *Point_axtw_subgroup) XY_affine() (FieldElement, FieldElement) {
	p.normalizeSubgroup()
	return p.x, p.y
}

// T_affine returns the T coordinate of the given point in affine twisted Edwards coordinates, i.e. X/Z * Y/Z.
func (p *Point_axtw_full) T_affine() FieldElement {
	return p.t
}

// T_affine returns the T coordinate of the given point in affine twisted Edwards coordinates, i.e. X/Z * Y/Z.
func (p *Point_axtw_subgroup) T_affine() FieldElement {
	p.normalizeSubgroup()
	return p.t
}

// XY_affine returns the X and Y coordinate of the given point in affine twisted Edwards coordinates. It is equivalent to calling X_affine and Y_affine, but may be more efficient.
func (p *Point_axtw_full) XY_affine() (FieldElement, FieldElement) {
	return p.x, p.y
}

// XYT_affine returns the X, Y and T=X*Y coordinate of the given point in affine twisted Edwards coordinates. It is equivalent to calling X_affine, Y_affine and T_affine, but may be more efficient.
func (p *Point_axtw_subgroup) XYT_affine() (FieldElement, FieldElement, FieldElement) {
	p.normalizeSubgroup()
	return p.x, p.y, p.t
}

// XYT_affine returns the X, Y and T=X*Y coordinate of the given point in affine twisted Edwards coordinates. It is equivalent to calling X_affine, Y_affine and T_affine, but may be more efficient.
func (p *Point_axtw_full) XYT_affine() (FieldElement, FieldElement, FieldElement) {
	return p.x, p.y, p.t
}

// X_decaf_affine returns the X coordinate of either P or P+A in affine twisted Edwards coordinates, where A is the affine point of order two.
//
// CAVEAT: Subsequent calls to any <foo>_decaf_affine methods are only guaranteed to be consistent if nothing else is done with the point between those calls.
// The consistency guarantee includes that different <foo>_decaf_affine methods make the same P vs. P+A choice.
// The requirements include not using the point as (pointer) argument to seemingly read-only methods (as these might change the internal representation) and not using <foo>_decaf_projective methods.
//
// Note: If P has extended projective Edwards coordinates (with Z==1) X:Y:T:1, then P+A has coordinates -X:-Y:T:1.
func (p *point_axtw_base) X_decaf_affine() FieldElement {
	return p.x
}

// Y_decaf_affine returns the Y coordinate of either P or P+A in affine twisted Edwards coordinates, where A is the affine point of order two.
//
// CAVEAT: Subsequent calls to any <foo>_decaf_affine methods are only guaranteed to be consistent if nothing else is done with the point between those calls.
// The consistency guarantee includes that different <foo>_decaf_affine methods make the same P vs. P+A choice.
// The requirements include not using the point as (pointer) argument to seemingly read-only methods (as these might change the internal representation) and not using <foo>_decaf_projective methods.
//
// Note: If P has extended projective Edwards coordinates (with Z==1) X:Y:T:1, then P+A has coordinates -X:-Y:T:1.
func (p *point_axtw_base) Y_decaf_affine() FieldElement {
	return p.y
}

// T_decaf_affine returns the T coordinate of either P or P+A in affine twisted Edwards coordinates, where A is the affine point of order two.
//
// CAVEAT: Subsequent calls to any <foo>_decaf_affine methods are only guaranteed to be consistent if nothing else is done with the point between those calls.
// The consistency guarantee includes that different <foo>_decaf_affine methods make the same P vs. P+A choice.
// The requirements include not using the point as (pointer) argument to seemingly read-only methods (as these might change the internal representation) and not using <foo>_decaf_projective methods.
//
// Note: If P has extended projective Edwards coordinates (with Z==1) X:Y:T:1, then P+A has coordinates -X:-Y:T:1.
// In particular, T_decaf_affine and T_affine match (except for the requirements on not interleaving method calls).
func (p *point_axtw_base) T_decaf_affine() FieldElement {
	return p.t
}

// IsNeutralElement checks if the given point p is the neutral element of the curve.
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

// IsNeutralElement checks if the given point p is the neutral element of the curve.
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
// The two points do not have to be in the same coordinate format.
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

// IsEqual compares two curve points for equality.
// The two points do not have to be in the same coordinate format.
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

// IsAtInfinity tests whether the point is an infinite (neccessarily order-2) point.
// Since these points cannot be represented in affine coordinates and we panic if we would assign them, this never returns true for Point_axtw_full and Point_axtw_subgroup.
//
// Note that for the Bandersnatch curve in twisted Edwards coordinates, there are two rational points at infinity; these points are not in the p253-subgroup and differ from the neutral element.
func (p *point_axtw_base) IsAtInfinity() bool {
	if p.IsNaP() {
		napEncountered("When checking whether an axtw point is infinite, a NaP was encountered", false, p)
		// we also return false in this case (unless the error handler panics).
	}
	return false
}

// IsNaP checks whether the point is a NaP (Not-a-point). NaPs must never appear if the library is used correctly. They can appear by
// a) zero-initialized points are NaPs (Go lacks constructors to fix that).
// b) performing operations on NaPs.
// c) bugs (either in the library or as a corner case due to wrong usage of doing untrusted conversion to subgroup of points outside the subgroup)
// For Point_axtw_full and Point_axtw_subgroup, NaPs have x==y==0. (Actually, we expect only x==y==t==0 to happen).
func (p *point_axtw_base) IsNaP() bool {
	return p.x.IsZero() && p.y.IsZero()
}

/*
func (p *point_axtw_base) toDecaf_xtw() point_xtw_base {
	return point_xtw_base{x: p.x, y: p.y, t: p.t, z: fieldElementOne}
}

func (p *point_axtw_base) toDecaf_axtw() (ret point_axtw_base) {
	ret = *p
	return
}
*/

// IsInSubgroup checks whether the given curve point is in the p253 prime-order subgroup.
func (p *Point_axtw_full) IsInSubgroup() bool {
	if p.IsNaP() {
		return napEncountered("Checking whether NaP is in subgroup", false, p)
	}
	return legendreCheckA_affineX(p.x) && legendreCheckE1_affineY(p.y)
}

// Clone returns a pointer to an independent copy of the given base point struct.
// The returned pointer is returned in a CurvePointPtrInterfaceBaseRead interface, but the actual value is guaranteed to have the same type as the receiver.
func (p *point_axtw_base) Clone() CurvePointPtrInterfaceBaseRead {
	p_copy := *p
	return &p_copy
}

// Clone returns a pointer to an independent copy of the given point.
// The returned pointer is returned in a CurvePointPtrInterface interface, but the actual value is guaranteed to have the same type as the receiver.
//
// Note: Point_axtw_subgroup internally stores no pointers.
// var copy Point_axtw_subgroup = original (non-pointer) just works.
func (p *Point_axtw_subgroup) Clone() CurvePointPtrInterface {
	p_copy := *p
	return &p_copy
}

// Clone returns a pointer to an independent copy of the given point.
// The returned pointer is returned in a CurvePointPtrInterface interface, but the actual value is guaranteed to have the same type as the receiver.
//
// Note: Point_axtw_full internally stores no pointers.
// var copy Point_axtw_full = original (non-pointer) just works.
func (p *Point_axtw_full) Clone() CurvePointPtrInterface {
	p_copy := *p
	return &p_copy
}

// TODO: Internal debug format vs. unified printing.

// Point_axtw::SerializeShort, Point_axtw::SerializeLong and Point_axtw::SerializeAuto are defined directly in curve_point_impl_serialize.go

// String is defined to satisfy the fmt.Stringer interface and allows points to be used in most fmt routines.
// Note that String() is defined on value receivers (as opposed to everything else) for an easier interface when using fmt routines.
// For Point_axtw_full, it prints the point in (affine) X:Y:T - format
func (p point_axtw_base) String() string {
	// Not the most efficient way, but good enough.
	return p.x.String() + ":" + p.y.String() + ":" + p.t.String()
}

// String is defined to satisfy the fmt.Stringer interface and allows points to be used in most fmt routines.
// Note that String() is defined on value receivers (as opposed to everything else) for an easier interface when using fmt routines.
//
// NOTE: The output format is not stable yet.
func (p Point_axtw_subgroup) String() (ret string) {
	ret = p.point_axtw_base.String()
	if !legendreCheckE1_affineY(p.y) {
		ret += " [+A]"
	}
	return
}

// SetFromSubgroupPoint sets the receiver to a copy of the input, which needs to be in the prime-order subgroup.
// This method can be used to convert from point types capable of holding points not in the prime-order subgroup to point types that do not.
// The second argument needs to be either TrustedInput or UntrustedInput.
// For UntrustedInput, we actually check whether the input is in the subgroup; For TrustedInput, we assume it to be the case.
// The return value indicates success. On failure, the receiver is unchanged.
//
// NOTE: Calling this checks for NaPs even for TrustedInput. Other than that, we make no guarantees whatsoever when calling it on points outside the subgroup with TrustedInput.
func (p *Point_axtw_subgroup) SetFromSubgroupPoint(input CurvePointPtrInterfaceRead, trusted IsInputTrusted) (ok bool) {
	if input.IsNaP() {
		napEncountered("Converting NaP point to affine subgroup point", false, input)
		// *p = Point_axtw_subgroup{}
		return false
	}
	if input.CanOnlyRepresentSubgroup() {
		p.SetFrom(input)
		return true
	}

	if !trusted.Bool() {
		if !input.IsInSubgroup() {
			return false
		}
	}
	switch input := input.(type) {
	case *Point_efgh_full:
		p.point_axtw_base = input.toDecaf_axtw()
	case *Point_xtw_full:
		input.normalizeAffineZ()
		p.x = input.x
		p.y = input.y
		p.t = input.t
	case *Point_axtw_full:
		p.point_axtw_base = input.point_axtw_base
	case CurvePointPtrInterfaceCooReadExtended:
		p.x, p.y, p.t = input.XYT_affine()
	default:
		p.x, p.y = input.XY_affine()
		p.t.Mul(&p.x, &p.y)
	}
	return true
}

// SetFromSubgroupPoint sets the receiver to a copy of the input, which needs to be in the prime-order subgroup.
// This method can be used to convert from point types capable of holding points not in the prime-order subgroup to point types that do not.
// The second argument needs to be either TrustedInput or UntrustedInput.
// For UntrustedInput, we actually check whether the input is in the subgroup; For TrustedInput, we assume it to be the case.
// The return value indicates success. On failure, the receiver is unchanged.
//
// NOTE: Calling this checks for NaPs even for TrustedInput. Other than that, we make no guarantees whatsoever when calling it on points outside the subgroup with TrustedInput.
func (p *Point_axtw_full) SetFromSubgroupPoint(input CurvePointPtrInterfaceRead, trusted IsInputTrusted) (ok bool) {
	if input.IsNaP() {
		napEncountered("Converting NaP point to affine subgroup point", false, input)
		// *p = Point_axtw_full{}
		return false
	}
	if !trusted.Bool() {
		if !input.IsInSubgroup() {
			return false
		}
	}
	p.SetFrom(input)
	return true
}

// SetFrom initializes the point from the given input point (which may have a different coordinate format).
//
// NOTE: To initialize a Point of type Point_axtw_subgroup with an input of a type that can hold points outside the subgroup, you need to use SetFromSubgroupPoint instead.
func (p *Point_axtw_subgroup) SetFrom(input CurvePointPtrInterfaceRead) {
	switch input := input.(type) {
	case *Point_efgh_subgroup:
		p.point_axtw_base = input.toDecaf_axtw()
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
		p.point_axtw_base = input.toDecaf_axtw()
	case *Point_efgh_subgroup:
		input.normalizeSubgroup()
		p.point_axtw_base = input.toDecaf_axtw()
	case CurvePointPtrInterfaceCooReadExtended:
		if input.(CurvePointPtrInterfaceRead).IsNaP() {
			napEncountered("Converting NaP to axtw_full", false, input.(CurvePointPtrInterfaceRead))
			*p = Point_axtw_full{}
			return
		}
		p.x, p.y, p.t = input.XYT_affine()
	default:
		if input.IsNaP() {
			napEncountered("Converting NaP to axtw_full", false, input)
			*p = Point_axtw_full{}
			return
		}
		p.x, p.y = input.XY_affine()
		p.t.Mul(&p.x, &p.y)
	}
}

// Add performs curve point addition according to the elliptic curve group law.
// Use p.Add(&x, &y) for p := x + y.
func (p *Point_axtw_subgroup) Add(x, y CurvePointPtrInterfaceRead) {
	var temp Point_efgh_subgroup
	temp.Add(x, y)
	p.point_axtw_base = temp.toDecaf_axtw()
}

// Add performs curve point addition according to the elliptic curve group law.
// Use p.Add(&x, &y) for p = x + y.
func (p *Point_axtw_full) Add(x, y CurvePointPtrInterfaceRead) {
	var temp Point_efgh_full
	temp.Add(x, y)
	p.x, p.y, p.t = temp.XYT_affine()
}

// Sub performs curve point subtraction according to the elliptic curve group law.
// Use p.Sub(&x, &y) for p = x - y.
func (p *Point_axtw_subgroup) Sub(x, y CurvePointPtrInterfaceRead) {
	var temp Point_efgh_subgroup
	temp.Sub(x, y)
	p.point_axtw_base = temp.toDecaf_axtw()
}

// Sub performs curve point subtraction according to the elliptic curve group law.
// Use p.Sub(&x, &y) for p = x - y.
func (p *Point_axtw_full) Sub(x, y CurvePointPtrInterfaceRead) {
	var temp Point_efgh_full
	temp.Sub(x, y)
	p.x, p.y, p.t = temp.XYT_affine()
}

// Double computes the sum of a point with itself.
// p.Double(&x) means p = x + x.
//
// Note that x + x is always in the prime-order subgroup.
// As opposed to p.Add(&x, &x), p.Double(&x) works even if the type of p can only hold subgroup curve points and the type of x can hold general points.
func (p *point_axtw_base) Double(in CurvePointPtrInterfaceRead) {
	var temp point_efgh_base
	temp.Double(in)
	*p = temp.toDecaf_axtw()
}

// Neg computes the negative of the point wrt the elliptic curve group law.
// Use p.Neg(&input) for p = -input.
func (p *Point_axtw_subgroup) Neg(input CurvePointPtrInterfaceRead) {
	p.SetFrom(input)
	p.NegEq()
}

// Neg computes the negative of the point wrt the elliptic curve group law.
// Use p.Neg(&input) for p = -input.
func (p *Point_axtw_full) Neg(input CurvePointPtrInterfaceRead) {
	p.SetFrom(input)
	p.NegEq()
}

// Endo computes the efficient order-2 endomorphism on the given point described in the Bandersnatch paper.
//
// On the prime-order subgroup, this endomorphism acts as multiplication by the constant given as EndomorphismEivenvalue, which is
// a square root of -2.
func (p *Point_axtw_subgroup) Endo(input CurvePointPtrInterfaceRead) {
	var temp Point_efgh_subgroup
	temp.Endo(input)
	p.point_axtw_base = temp.toDecaf_axtw()
}

// Endo computes the efficient order-2 endomorphism on the given point described in the Bandersnatch paper.
//
// On the prime-order subgroup, this endomorphism acts as multiplication by the constant given as EndomorphismEivenvalue, which is
// a square root of -2.
func (p *Point_axtw_full) Endo(input CurvePointPtrInterfaceRead) {
	var temp Point_efgh_full
	temp.Endo(input)
	p.x, p.y, p.t = temp.XYT_affine()
}

// SetNeutral sets the given point to the neutral element of the curve.
func (p *point_axtw_base) SetNeutral() {
	*p = neutralElement_axtwbase
}

// AddEq adds (via the elliptic curve group addition law) the given curve point x to the received p, overwriting p.
//
// p.AddEq(&x) is equivalent to p.AddEq(&p, &x)
func (p *Point_axtw_full) AddEq(x CurvePointPtrInterfaceRead) {
	p.Add(p, x)
}

// AddEq adds (via the elliptic curve group addition law) the given curve point x to the received p, overwriting p.
//
// p.AddEq(&x) is equivalent to p.AddEq(&p, &x)
func (p *Point_axtw_subgroup) AddEq(x CurvePointPtrInterfaceRead) {
	p.Add(p, x)
}

// SubEq subtracts (via the elliptic curve group addition law) the curve point x from the received p, overwriting p.
//
// p.SubEq(&x) is equivalent to p.SubEq(&p, &x)
func (p *Point_axtw_subgroup) SubEq(x CurvePointPtrInterfaceRead) {
	p.Sub(p, x)
}

// SubEq subtracts (via the elliptic curve group addition law) the curve point x from the received p, overwriting p.
//
// p.SubEq(&x) is equivalent to p.SubEq(&p, &x)
func (p *Point_axtw_full) SubEq(x CurvePointPtrInterfaceRead) {
	p.Sub(p, x)
}

// DoubleEq doubles the received point p, overwriting p.
//
// p.DoubleEq() is equivalent to p.Double(&p)
func (p *point_axtw_base) DoubleEq() {
	var temp point_efgh_base
	temp.add_saa(p, p)
	*p = temp.toDecaf_axtw()
}

// NegEq replaces the given point by its negative (wrt the elliptic curve group addition law).
//
// p.NegEq() is equivalent to p.NegEq(&p)
func (p *point_axtw_base) NegEq() {
	p.x.NegEq()
	p.t.NegEq()
}

// EndoEq applies the endomorphism on the given point p, overwriting it.
//
// p.EndoEq() is equivalent to p.Endo(&p).
func (p *Point_axtw_subgroup) EndoEq() {
	p.Endo(p)
}

// EndoEq applies the endomorphism on the given point p, overwriting it.
//
// p.EndoEq() is equivalent to p.Endo(&p).
func (p *Point_axtw_full) EndoEq() {
	p.Endo(p)
}

// Validate checks whether the point is a valid curve point.
//
// NOTE: Outside of NaPs, it should not be possible to create points that fail Validate when using the interface correctly.
// Validate is used only in testing and is required by the CurvePointPtrInterfaceTestSample interface.
func (p *point_axtw_base) Validate() bool {
	return p.isPointOnCurve()
}

// Validate checks whether the point is a valid curve point.
// For Point_axtw_subgroup, this includes verifying membership in the prime-order subgroup.
//
// NOTE: Outside of NaPs, it should not be possible to create points that fail Validate when using the interface correctly.
// Validate is used only in testing and is required by the CurvePointPtrInterfaceTestSample interface.
func (p *Point_axtw_subgroup) Validate() bool {
	return p.point_axtw_base.isPointOnCurve() && legendreCheckA_affineX(p.x)
}

// sampleRandomUnsafe samples a (pseudo-)random curvepoint.
// It is used in testing only and required by the CurvePointPtrInterfaceTestValue interface.
//
// NOTE: While good enough for testing, the randomness quality is insufficient for cryptographic purposes.
// This is why we do not export this.
func (p *Point_axtw_full) sampleRandomUnsafe(rnd *rand.Rand) {
	p.point_axtw_base = makeRandomPointOnCurve_a(rnd)
}

// sampleRandomUnsafe samples a (pseudo-)random curvepoint.
// It is used in testing only and required by the CurvePointPtrInterfaceTestValue interface.
//
// NOTE: While good enough for testing, the randomness quality is insufficient for cryptographic purposes.
// This is why we do not export this.
func (p *Point_axtw_subgroup) sampleRandomUnsafe(rnd *rand.Rand) {
	p.point_axtw_base = makeRandomPointOnCurve_a(rnd)
	p.point_axtw_base.DoubleEq()
}

// SetAffineTwoTorsion sets the point to the affine-order two point.
// This function is required to satisfy the curvePointPtrInterfaceTestSampleA interface, which
// our testing framework mandates that Point_axtw_full must satisfy.
func (p *Point_axtw_full) SetAffineTwoTorsion() {
	p.point_axtw_base = affineOrderTwoPoint_axtwbase
}

// NormalizeForZ tries to change the internal representation of the point to one where Z_decaf_projective() outputs 1.
//
// The return value indicates success. On success, a subsequent Z_decaf_projective() without intervening calls is guaranteed to return 1.
// The function will fail and return false for NaPs and points at infinity.
// On such failure, the representation may still change (to a different NaP or to a different representation of the point at infinity)
func (p *point_axtw_base) NormalizeForZ() (ok bool) {
	if p.IsNaP() {
		napEncountered("called NormalizerForZ on NaP of type axtw", false)
		return false
	}
	return true
}
