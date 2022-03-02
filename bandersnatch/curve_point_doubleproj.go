package bandersnatch

import "math/rand"

// Point_efgh describes points (usually on the p253-subgroup of) the bandersnatch curve in E:G, H:F - coordinates (called double-projective or "efgh"-coos), i.e.
// we represent X/Z as E/G and Y/Z as H/F. From a computational view, this effectively means that we use a separate denominator for X and Y (instead of a joint one Z).
// We can recover X:Y:Z coordinates by computing Z = F*G, X = E*F, Y = G*H. Then T = E*H. This is meaningful even if one of E,G is zero. There are no rational points with F=0 or H=0.
// Observe that in fact all default formulae in extended twisted edwards coordinates *produce* points in such efgh coordinates and then transform them into the desired form.
// Using double-projective coordinates can be used to make this explicit.

// Usually, we prefer inputs to be affine over projective over efgh. Output is always interally efgh first and then converted.
// By making this explicit, we can, in fact, save computation if a coordinate is unused:
// The doubling formula and the endomorphism can be computed directly from efgh-coordinates without going through X:Y:T:Z coordinates more efficiently that converting + conputing would cost.

// Generally, if an output of a curve operation is only used once, it should remain in efgh-coordinates.
// It is better to let the next operation on the point perform the conversion internally as part of the operation when and if it is needed.
// If an output is used more than once, it should be converted.

// On the p253-subgroup, the only coordinate that may be zero is actually e.

// Note: Conversion from X:Y:T:Z to EFGH is available as e.g.
// E:=X, F:=X, G:=Z, H:=T or
// E:=T, F:=X, G:=Y, H:=T or
// E:=X, F:=Z, G:=Z, H:=Y or
// (The first two options have singularities at neutral and affine-order-2, the third option at the points at infinity)

// point_efgh_base is a struct that holds e,f,g,h coordinates of a purported curve point.
// It is struct-embedded in Point_efgh_full and Point_efgh_subgroup.
// Note that it does *NOT* model a curve point; Point_efgh_full and Point_efgh_subgroup do. Notably,
// Point_efgh_full and Point_efgh_subgroup actually interpret the coordinates in point_efgh_base in a different way:
// Point_efgh_full does it in the canonical way, whereas Point_efgh_subgroup does it modulo A.
// (note that the latter is not really visible through the default interface)
type point_efgh_base struct {
	thisCurvePointCanRepresentFullCurve
	thisCurvePointCanRepresentInfinity
	e FieldElement
	f FieldElement
	g FieldElement
	h FieldElement
}

// Point_efgh_full describes rational points of the Bandersnatch curve. This type can holds all rational points, including points at infinity and points outside the prime-order subgroup.
//
// Note: Using this point type is the most efficient type supported for obtaining output, but not when used as input.
// Consequently, this type should be used when obtaining a curve point in the middle of a computation and using it exactly once.
type Point_efgh_full struct {
	point_efgh_base // e,f,g,h (struct embedding)
}

// Point_efgh_full describes points on the p253 prime-order subgroup of the Bandersnatch curve.
// This type can only hold points in that subgroup (and hence, cannot holds points at infinty).
//
// Note: On a twisted Edwards curve, the neutral element is NOT at infinity.
//
// Note: Using this point type is the most efficient type supported for obtaining output, but not when used as input.
// Consequently, this type should be used when obtaining a curve point in the middle of a computation and using it exactly once.
type Point_efgh_subgroup struct {
	thisCurvePointCanOnlyRepresentSubgroup
	thisCurvePointCannotRepresentInfinity
	point_efgh_base
}

// two-torsion elements as _efghbase
var (
	neutralElement_efghbase     = point_efgh_base{e: FieldElementZero, f: FieldElementOne, g: FieldElementOne, h: FieldElementOne}        // Note: g!=0 is actually arbitrary.
	orderTwoPoint_efghbase      = point_efgh_base{e: FieldElementZero, f: FieldElementOne, g: FieldElementOne, h: FieldElementMinusOne}   // Note: g!=0 is actually arbitrary.
	exceptionalPoint_1_efghbase = point_efgh_base{e: FieldElementOne, f: squareRootDbyA_fe, g: FieldElementZero, h: FieldElementOne}      // Note: e!=0 is actually arbitrary.
	exceptionalPoint_2_efghbase = point_efgh_base{e: FieldElementOne, f: squareRootDbyA_fe, g: FieldElementZero, h: FieldElementMinusOne} // Note: e!=0 is actually arbitrary.
)

// NeutralElement_efgh_full denote the neutral element of the Bandersnatch curve (with type Point_efgh_full).
var NeutralElement_efgh_full Point_efgh_full = Point_efgh_full{point_efgh_base: neutralElement_efghbase}

// NeutralElement_efgh_subgroup denote the neutral element of the Bandersnatch curve (with type Point_efgh_subgroup).
var NeutralElement_efgh_subgroup Point_efgh_subgroup = Point_efgh_subgroup{point_efgh_base: neutralElement_efghbase}

// AffineOrderTwoPoint_efgh denotes the denotes the affine point of order two of the Bandersnatch curve (with type Point_efgh_full). These points are not on the p253 prime order subgroup.
var AffineOrderTwoPoint_efgh Point_efgh_full = Point_efgh_full{point_efgh_base: orderTwoPoint_efghbase}

// InfinitePoint1_efgh and InfinitePoint2_efgh denote the the infinite points E1 and E2 of order two on the Bandersnatch curve (of type Point_efgh_full). This point is not on the p253 prime order subgroup.
// The distinction between these two points is essentially arbitrary, but done in a way consistent with IsE1() and IsE2() and conversion to other point types.
var (
	InfinitePoint1_efgh Point_efgh_full = Point_efgh_full{point_efgh_base: exceptionalPoint_1_efghbase}
	InfinitePoint2_efgh Point_efgh_full = Point_efgh_full{point_efgh_base: exceptionalPoint_2_efghbase}
)

// normalize_affine puts the point in an equivalent "normalized" state with f==g==1.
// NaPs will be put into the uninitialized, default e==f==g==h==0 NaP state.
// Points at infinity panic.
// This roughly corresponds to setting Z==1 for affine coordinates.
func (p *point_efgh_base) normalize_affine() {
	if p.is_normalized() {
		return
	}
	if p.IsNaP() {
		napEncountered("Trying to normalize singular point", false, p)
		// If the error handler did not panic, we intentionally set the NaP p to a "full" NaP with all coos 0 (rather than at least 2).
		// This has the effect that all conversion routines that start by calling normalize_affine will only need to worry about NaPs with e==f==g==h==0
		*p = point_efgh_base{}
		return
	}
	var temp FieldElement
	temp.Mul(&p.f, &p.g)
	if temp.IsZero() {
		panic("Trying to normalize point at infinity")
	}
	temp.Inv(&temp)
	p.e.MulEq(&p.f)
	p.h.MulEq(&p.g)
	p.e.MulEq(&temp)
	p.h.MulEq(&temp)
	p.f.SetOne()
	p.g.SetOne()
}

// is_normalized checks whether p is already in the form given by normalize_affine().
// There are usecases in which normalize_affine is called several times on the same point in a row.
// This function is then used to detect that no action is needed.
func (p *point_efgh_base) is_normalized() bool {
	// Note: We return false on NaPs. Either way is fine.
	return p.f.IsOne() && p.g.IsOne()
}

// normalizeSubgroup changes the internal representation of p, such that the coordinates of p
// corresponds exactly to p (without working modulo the affine two-torsion point).
func (p *Point_efgh_subgroup) normalizeSubgroup() {
	if !legendreCheckE1_FH(p.f, p.h) {
		p.flipDecaf()
	}
}

// rerandomizeRepresentation is needed to satisfy the CurvePointPtrInterfaceTestSample interface for testing.
// It changes the internal representation to an equivalent one.
func (p *point_efgh_base) rerandomizeRepresentation(rnd *rand.Rand) {
	var m FieldElement
	m.setRandomUnsafeNonZero(rnd)
	p.e.MulEq(&m)
	p.g.MulEq(&m)
	m.setRandomUnsafeNonZero(rnd)
	p.f.MulEq(&m)
	p.h.MulEq(&m)
}

// rerandomizeRepresentation is needed to satisfy the CurvePointPtrInterfaceTestSample interface for testing.
// It changes the internal representation to an equivalent one.
func (p *Point_efgh_subgroup) rerandomizeRepresentation(rnd *rand.Rand) {
	p.point_efgh_base.rerandomizeRepresentation(rnd)
	if rnd.Intn(2) == 0 {
		p.flipDecaf()
	}
}

// IsAtInfinity tests whether the point is an infinite (neccessarily order-2) point.
//
// Note that for the Bandersnatch curve in twisted Edwards coordinates, there are two rational points at infinity; these points are not in the p253-subgroup and differ from the neutral element.
func (p *point_efgh_base) IsAtInfinity() bool {
	if p.IsNaP() {
		return napEncountered("NaP encountered when asking where efgh-point is at infinity", true, p)
	}
	// The only valid (non-NaP) points with g==0 are are those at infinity
	return p.g.IsZero()
}

// IsAtInfinity tests whether the point is an infinite (neccessarily order-2) point.
//
// Note that for the Bandersnatch curve in twisted Edwards coordinates, there are two rational points at infinity; these points are not in the p253-subgroup and differ from the neutral element.
// Consequently, this cannot return true for Points of type Point_efgh_subgroup.
func (p *Point_efgh_subgroup) IsAtInfinity() bool {
	if p.IsNaP() {
		return napEncountered("NaP encountered when asking where efgh-point is at infinity", true, p)
	}
	return false
}

// IsAtInfinity tests whether the point is an infinite (neccessarily order-2) point.
//
// Note that for the Bandersnatch curve in twisted Edwards coordinates, there are two rational points at infinity; these points are not in the p253-subgroup and differ from the neutral element.
func (p *Point_efgh_full) IsAtInfinity() bool {
	if p.IsNaP() {
		return napEncountered("NaP encountered when asking where efgh-point is at infinity", true, p)
	}
	// The only valid (non-NaP) points with g==0 are are those at infinity
	return p.g.IsZero()
}

// TODO: Define on _base ?

// flipDecaf changes the internal representation of P from P to P+A or vice versa (note that we work modulo A).
// flipDecaf is needed to satisfy the (optional) curvePointPtrInterfaceDecaf interface that is recognized by the testing framework.
func (p *Point_efgh_subgroup) flipDecaf() {
	// this preserves is_normalized
	p.e.NegEq()
	p.h.NegEq()
}

// TODO: This might go away.

// HasDecaf needs to return true for flipDecaf to be recognized by the testing framework.
func (p *Point_efgh_subgroup) HasDecaf() bool {
	return true
}

// X_projective returns the X coordinate of the given point in projective twisted Edwards coordinates.
//
// CAVEAT: Subsequent calls to any <foo>_projective methods on the same point are only guaranteed to be consistent if nothing else is done with the point between the calls.
// This includes that you may not be able to use the point as argument to even seemingly read-only methods, as these might touch the redundant internal representation.
func (p *Point_efgh_subgroup) X_projective() (X FieldElement) {
	p.normalizeSubgroup()
	X.Mul(&p.e, &p.f)
	return
}

// X_projective returns the X coordinate of the given point in projective twisted Edwards coordinates.
//
// CAVEAT: Subsequent calls to any <foo>_projective methods on the same point are only guaranteed to be consistent if nothing else is done with the point between the calls.
// This includes that you may not be able to use the point as argument to even seemingly read-only methods, as these might touch the redundant internal representation.
func (p *Point_efgh_full) X_projective() (X FieldElement) {
	X.Mul(&p.e, &p.f)
	return
}

// X_decaf_projective returns the X coordinate of either P or P+A in projective twisted Edwards coordinates, where A is the affine point of order two.
//
// CAVEAT: Subsequent calls to any <foo>_decaf_projective methods are only guaranteed to be consistent if nothing else is done with the point between those calls.
// The consistency guarantee includes that different <foo>_decaf_projective methods make the same P vs. P+A choice.
// The requirements include not using the point as (pointer) argument to seemingly read-only methods (as these might change the internal representation) and not using <foo>_decaf_affine methods.
//
// Note: If P has extended projective Edwards coordinates X:Y:T:Z, then P+A has coordinates -X:-Y:T:Z == X:Y:-T:-Z
func (p *point_efgh_base) X_decaf_projective() (X FieldElement) {
	X.Mul(&p.e, &p.f)
	return
}

// Y_projective returns the Y coordinate of the given point in projective twisted Edwards coordinates.
//
// CAVEAT: Subsequent calls to any <foo>_projective methods on the same point are only guaranteed to be consistent if nothing else is done with the point between the calls.
// This includes that you may not be able to use the point as argument to even seemingly read-only methods, as these might touch the redundant internal representation.
func (p *Point_efgh_subgroup) Y_projective() (Y FieldElement) {
	p.normalizeSubgroup()
	Y.Mul(&p.g, &p.h)
	return
}

// Y_projective returns the Y coordinate of the given point in projective twisted Edwards coordinates.
//
// CAVEAT: Subsequent calls to any <foo>_projective methods on the same point are only guaranteed to be consistent if nothing else is done with the point between the calls.
// This includes that you may not be able to use the point as argument to even seemingly read-only methods, as these might touch the redundant internal representation.
func (p *Point_efgh_full) Y_projective() (Y FieldElement) {
	Y.Mul(&p.g, &p.h)
	return
}

// Y_decaf_projective returns the Y coordinate of either P or P+A in projective twisted Edwards coordinates, where A is the affine point of order two.
//
// CAVEAT: Subsequent calls to any <foo>_decaf_projective methods are only guaranteed to be consistent if nothing else is done with the point between those calls.
// The consistency guarantee includes that different <foo>_decaf_projective methods make the same P vs. P+A choice.
// The requirements include not using the point as (pointer) argument to seemingly read-only methods (as these might change the internal representation) and not using <foo>_decaf_affine methods.
//
// Note: If P has extended projective Edwards coordinates X:Y:T:Z, then P+A has coordinates -X:-Y:T:Z == X:Y:-T:-Z
func (p *point_efgh_base) Y_decaf_projective() (Y FieldElement) {
	Y.Mul(&p.g, &p.h)
	return
}

// T_projective returns the T coordinate of the given point in projective twisted Edwards coordinates. This coordinate satisfies X*Y = T*Z.
//
// CAVEAT: Subsequent calls to any <foo>_projective methods on the same point are only guaranteed to be consistent if nothing else is done with the point between the calls.
// This includes that you may not be able to use the point as argument to even seemingly read-only methods, as these might touch the redundant internal representation.
func (p *Point_efgh_subgroup) T_projective() (T FieldElement) {
	p.normalizeSubgroup()
	T.Mul(&p.e, &p.h)
	return
}

// T_projective returns the T coordinate of the given point in projective twisted Edwards coordinates. This coordinate satisfies X*Y = T*Z.
//
// CAVEAT: Subsequent calls to any <foo>_projective methods on the same point are only guaranteed to be consistent if nothing else is done with the point between the calls.
// This includes that you may not be able to use the point as argument to even seemingly read-only methods, as these might touch the redundant internal representation.
func (p *Point_efgh_full) T_projective() (T FieldElement) {
	T.Mul(&p.e, &p.h)
	return
}

// T_decaf_projective returns the T coordinate of either P or P+A in projective twisted Edwards coordinates, where A is the affine point of order two.
//
// CAVEAT: Subsequent calls to any <foo>_decaf_projective methods are only guaranteed to be consistent if nothing else is done with the point between those calls.
// The consistency guarantee includes that different <foo>_decaf_projective methods make the same P vs. P+A choice.
// The requirements include not using the point as (pointer) argument to seemingly read-only methods (as these might change the internal representation) and not using <foo>_decaf_affine methods.
//
// Note: If P has extended projective Edwards coordinates X:Y:T:Z, then P+A has coordinates -X:-Y:T:Z == X:Y:-T:-Z
func (p *point_efgh_base) T_decaf_projective() (T FieldElement) {
	T.Mul(&p.e, &p.h)
	return
}

// Z_projective returns the Z coordinate of the given point in projective twisted Edwards coordinates.
//
// CAVEAT: Subsequent calls to any <foo>_projective methods on the same point are only guaranteed to be consistent if nothing else is done with the point between the calls.
// This includes that you may not be able to use the point as argument to even seemingly read-only methods, as these might touch the redundant internal representation.
func (p *Point_efgh_subgroup) Z_projective() (Z FieldElement) {
	p.normalizeSubgroup()
	Z.Mul(&p.f, &p.g)
	return
}

// Z_projective returns the Z coordinate of the given point in projective twisted Edwards coordinates.
//
// CAVEAT: Subsequent calls to any <foo>_projective methods on the same point are only guaranteed to be consistent if nothing else is done with the point between the calls.
// This includes that you may not be able to use the point as argument to even seemingly read-only methods, as these might touch the redundant internal representation.
func (p *Point_efgh_full) Z_projective() (Z FieldElement) {
	Z.Mul(&p.f, &p.g)
	return
}

// Z_decaf_projective returns the Z coordinate of either P or P+A in projective twisted Edwards coordinates, where A is the affine point of order two.
//
// CAVEAT: Subsequent calls to any <foo>_decaf_projective methods are only guaranteed to be consistent if nothing else is done with the point between those calls.
// The consistency guarantee includes that different <foo>_decaf_projective methods make the same P vs. P+A choice.
// The requirements include not using the point as (pointer) argument to seemingly read-only methods (as these might change the internal representation) and not using <foo>_decaf_affine methods.
//
// Note: If P has extended projective Edwards coordinates X:Y:T:Z, then P+A has coordinates -X:-Y:T:Z == X:Y:-T:-Z
func (p *point_efgh_base) Z_decaf_projective() (Z FieldElement) {
	Z.Mul(&p.f, &p.g)
	return
}

// XYZ_projective returns X,Y and Z coordinates in projective twisted Edwards coordinates in a single call.
// It is equivalent to calling X_projective(), Y_projective(), Z_projective(), but may be more efficient.
func (p *Point_efgh_subgroup) XYZ_projective() (X FieldElement, Y FieldElement, Z FieldElement) {
	p.normalizeSubgroup()
	X.Mul(&p.e, &p.f)
	Y.Mul(&p.g, &p.h)
	Z.Mul(&p.f, &p.g)
	return
}

// XYZ_projective returns X,Y and Z coordinates in projective twisted Edwards coordinates in a single call.
// It is equivalent to calling X_projective(), Y_projective(), Z_projective(), but may be more efficient.
func (p *Point_efgh_full) XYZ_projective() (X FieldElement, Y FieldElement, Z FieldElement) {
	X.Mul(&p.e, &p.f)
	Y.Mul(&p.g, &p.h)
	Z.Mul(&p.f, &p.g)
	return
}

// XYTZ_projective returns X,Y,T and Z coordinates in projective twisted Edwards coordinates in a single call.
// It is equivalent to calling X_projective(), Y_projective(), T_projective(), Z_projective(), but may be more efficient.
func (p *Point_efgh_subgroup) XYTZ_projective() (X FieldElement, Y FieldElement, T FieldElement, Z FieldElement) {
	p.normalizeSubgroup()
	X.Mul(&p.e, &p.f)
	Y.Mul(&p.g, &p.h)
	T.Mul(&p.e, &p.h)
	Z.Mul(&p.f, &p.g)
	return
}

// XYTZ_projective returns X,Y,T and Z coordinates in projective twisted Edwards coordinates in a single call.
// It is equivalent to calling X_projective(), Y_projective(), T_projective(), Z_projective(), but may be more efficient.
func (p *Point_efgh_full) XYTZ_projective() (X FieldElement, Y FieldElement, T FieldElement, Z FieldElement) {
	X.Mul(&p.e, &p.f)
	Y.Mul(&p.g, &p.h)
	T.Mul(&p.e, &p.h)
	Z.Mul(&p.f, &p.g)
	return
}

// X_affine returns the X coordinate of the given point in affine twisted Edwards coordinates, i.e. X/Z.
func (p *Point_efgh_subgroup) X_affine() FieldElement {
	p.normalize_affine()
	p.normalizeSubgroup()
	return p.e
}

// X_affine returns the X coordinate of the given point in affine twisted Edwards coordinates, i.e. X/Z.
func (p *Point_efgh_full) X_affine() FieldElement {
	p.normalize_affine()
	return p.e
}

// X_decaf_affine returns the X coordinate of either P or P+A in affine twisted Edwards coordinates, where A is the affine point of order two.
//
// CAVEAT: Subsequent calls to any <foo>_decaf_affine methods are only guaranteed to be consistent if nothing else is done with the point between those calls.
// The consistency guarantee includes that different <foo>_decaf_affine methods make the same P vs. P+A choice.
// The requirements include not using the point as (pointer) argument to seemingly read-only methods (as these might change the internal representation) and not using <foo>_decaf_projective methods.
//
// Note: If P has extended projective Edwards coordinates (with Z==1) X:Y:T:1, then P+A has coordinates -X:-Y:T:1.
func (p *point_efgh_base) X_decaf_affine() FieldElement {
	p.normalize_affine()
	return p.e
}

// Y_affine returns the Y coordinate of the given point in affine twisted Edwards coordinates, i.e. Y/Z.
func (p *Point_efgh_subgroup) Y_affine() FieldElement {
	p.normalize_affine()
	p.normalizeSubgroup()
	return p.h
}

// Y_affine returns the Y coordinate of the given point in affine twisted Edwards coordinates, i.e. Y/Z.
func (p *Point_efgh_full) Y_affine() FieldElement {
	p.normalize_affine()
	return p.h
}

// Y_decaf_affine returns the Y coordinate of either P or P+A in affine twisted Edwards coordinates, where A is the affine point of order two.
//
// CAVEAT: Subsequent calls to any <foo>_decaf_affine methods are only guaranteed to be consistent if nothing else is done with the point between those calls.
// The consistency guarantee includes that different <foo>_decaf_affine methods make the same P vs. P+A choice.
// The requirements include not using the point as (pointer) argument to seemingly read-only methods (as these might change the internal representation) and not using <foo>_decaf_projective methods.
//
// Note: If P has extended projective Edwards coordinates (with Z==1) X:Y:T:1, then P+A has coordinates -X:-Y:T:1.
func (p *point_efgh_base) Y_decaf_affine() FieldElement {
	p.normalize_affine()
	return p.h
}

// T_affine returns the T coordinate of the given point in affine twisted Edwards coordinates, i.e. X/Z * Y/Z.
func (p *Point_efgh_subgroup) T_affine() (T FieldElement) {
	p.normalize_affine()
	p.normalizeSubgroup()
	T.Mul(&p.e, &p.h)
	return
}

// T_affine returns the T coordinate of the given point in affine twisted Edwards coordinates, i.e. X/Z * Y/Z.
func (p *Point_efgh_full) T_affine() (T FieldElement) {
	p.normalize_affine()
	T.Mul(&p.e, &p.h)
	return
}

// T_decaf_affine returns the T coordinate of either P or P+A in affine twisted Edwards coordinates, where A is the affine point of order two.
//
// CAVEAT: Subsequent calls to any <foo>_decaf_affine methods are only guaranteed to be consistent if nothing else is done with the point between those calls.
// The consistency guarantee includes that different <foo>_decaf_affine methods make the same P vs. P+A choice.
// The requirements include not using the point as (pointer) argument to seemingly read-only methods (as these might change the internal representation) and not using <foo>_decaf_projective methods.
//
// Note: If P has extended projective Edwards coordinates (with Z==1) X:Y:T:1, then P+A has coordinates -X:-Y:T:1.
// In particular, T_decaf_affine and T_affine match (except for the requirements on not interleaving method calls).
func (p *point_efgh_base) T_decaf_affine() (T FieldElement) {
	p.normalize_affine()
	T.Mul(&p.e, &p.h)
	return
}

// XY_affine returns the X and Y coordinate of the given point in affine twisted Edwards coordinates. It is equivalent to calling X_affine and Y_affine, but may be more efficient.
func (p *Point_efgh_subgroup) XY_affine() (X FieldElement, Y FieldElement) {
	p.normalizeSubgroup()
	p.normalize_affine()
	return p.e, p.h
}

// XY_affine returns the X and Y coordinate of the given point in affine twisted Edwards coordinates. It is equivalent to calling X_affine and Y_affine, but may be more efficient.
func (p *Point_efgh_full) XY_affine() (X FieldElement, Y FieldElement) {
	p.normalize_affine()
	return p.e, p.h
}

// XYT_affine returns the X, Y and T=X*Y coordinate of the given point in affine twisted Edwards coordinates. It is equivalent to calling X_affine, Y_affine and T_affine, but may be more efficient.
func (p *Point_efgh_subgroup) XYT_affine() (X FieldElement, Y FieldElement, T FieldElement) {
	p.normalizeSubgroup()
	p.normalize_affine()
	X = p.e
	Y = p.h
	T.Mul(&X, &Y)
	return
}

// XYT_affine returns the X, Y and T=X*Y coordinate of the given point in affine twisted Edwards coordinates. It is equivalent to calling X_affine, Y_affine and T_affine, but may be more efficient.
func (p *Point_efgh_full) XYT_affine() (X FieldElement, Y FieldElement, T FieldElement) {
	p.normalize_affine()
	X = p.e
	Y = p.h
	T.Mul(&X, &Y)
	return
}

// IsNeutralElement checks if the point P is the neutral element of the curve.
func (p *Point_efgh_subgroup) IsNeutralElement() bool {
	// The only valid points with e==0 are the neutral element and the affine order-2 point
	if p.IsNaP() {
		return napEncountered("Comparing NaP with neutral element for efgh_subgroup", true, p)
	}
	return p.e.IsZero()
}

// IsNeutralElement checks if the given point p is the neutral element of the curve.
func (p *Point_efgh_full) IsNeutralElement() bool {
	// The only valid points with e==0 are the neutral element and the affine order-2 point
	if p.IsNaP() {
		return napEncountered("Comparing NaP with neutral element for efgh_full", true, p)
	}
	return p.e.IsZero() && p.f.IsEqual(&p.h)
}

// TODO: Review whether we need to define this on _base

// IsE1 checks if the given point is the E1 point at infinity of the curve.
//
// Note that none of the points at infinity is in the p253 prime-order subgroup and so the method cannot return true for Point_efgh_subgroup.
// Also note that for twisted Edwards curves, the neutral element is NOT at infinity.
func (p *point_efgh_base) IsE1() bool {
	if !p.IsAtInfinity() {
		return false
	}
	var tmp FieldElement
	tmp.Mul(&p.h, &squareRootDbyA_fe)
	return tmp.IsEqual(&p.f)
}

// IsE2 checks if the given point is the E2 point at infinity of the curve.
//
// Note that none of the points at infinity is in the p253 prime-order subgroup and so the method cannot return true for Point_efgh_subgroup.
// Also note that for twisted Edwards curves, the neutral element is NOT at infinity.
func (p *point_efgh_base) IsE2() bool {
	if !p.IsAtInfinity() {
		return false
	}
	var tmp FieldElement
	tmp.Mul(&p.h, &squareRootDbyA_fe)
	tmp.NegEq()
	return tmp.IsEqual(&p.f)
}

// Clone returns a pointer to an independent copy of the given base point struct.
// The returned pointer is returned in a CurvePointPtrInterfaceBaseRead interface, but the actual value is guaranteed to have the same type as the receiver.
func (p *point_efgh_base) Clone() CurvePointPtrInterfaceBaseRead {
	var copy point_efgh_base = *p
	return &copy
}

// Clone returns a pointer to an independent copy of the given point.
// The returned pointer is returned in a CurvePointPtrInterface interface, but the actual value is guaranteed to have the same type as the receiver.
func (p *Point_efgh_full) Clone() CurvePointPtrInterface {
	var copy Point_efgh_full = *p
	return &copy
}

// Clone returns a pointer to an independent copy of the given point.
// The returned pointer is returned in a CurvePointPtrInterface interface, but the actual value is guaranteed to have the same type as the receiver.
func (p *Point_efgh_subgroup) Clone() CurvePointPtrInterface {
	var copy Point_efgh_subgroup = *p
	return &copy
}

// IsEqual compares two curve points for equality.
// The two points do not have to be in the same coordinate format.
func (p *Point_efgh_subgroup) IsEqual(other CurvePointPtrInterfaceRead) bool {
	if p.IsNaP() || other.IsNaP() {
		return napEncountered("NaP encountered when comparing efgh-point with other point", true, p, other)
	}
	switch other := other.(type) {
	case *Point_efgh_subgroup:
		return p.isEqual_moduloA_ss(&other.point_efgh_base)
	case *Point_efgh_full:
		p.normalizeSubgroup()
		return p.isEqual_exact_ss(&other.point_efgh_base)
	default:
		if other.CanOnlyRepresentSubgroup() {
			return p.point_efgh_base.isEqual_moduloA_sany(other)
		} else {
			p.normalizeSubgroup()
			return p.point_efgh_base.isEqual_exact_sany(other)
		}
	}
}

// IsEqual compares two curve points for equality.
// The two points do not have to be in the same coordinate format.
func (p *Point_efgh_full) IsEqual(other CurvePointPtrInterfaceRead) bool {
	if p.IsNaP() || other.IsNaP() {
		return napEncountered("NaP encountered when comparing efgh-point with other point", true, p, other)
	}
	switch other := other.(type) {
	case *Point_efgh_subgroup:
		other.normalizeSubgroup()
		return p.isEqual_exact_ss(&other.point_efgh_base)
	case *Point_efgh_full:
		return p.isEqual_exact_ss(&other.point_efgh_base)
	default:
		return p.point_efgh_base.isEqual_exact_sany(other)
	}
}

// IsNaP checks whether the point is a NaP (Not-a-point). NaPs must never appear if the library is used correctly. They can appear by
// a) performing operations on points that are not in the correct subgroup or that are NaPs.
// b) zero-initialized points are NaPs (Go lacks constructors to fix that).
// For Point_efgh, NaP points have either f==h==0 ("true" NaP-type1) or e==g==0 ("true" NaP-type2) or e==h==0 (result of working on affine NaP).
// Note that no valid points ever have h==0 or f==0.
func (p *point_efgh_base) IsNaP() bool {
	// Note: The panicking cases are not supposed to be possible to arise from working within the provided interface, even if you start with uninitialized points.
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

// TODO: This might go away.

// toDecaf_xtw converts the point to xtw coordinates.
// Note that this differs from the SetFrom functionality in that it operates on <foo>_base types.
// This is essentially equivalent to calling X_decaf_projective, Y_decaf_projective, T_decaf_projective, Z_decaf_projective.
// internal use only.
func (p *point_efgh_base) toDecaf_xtw() (ret point_xtw_base) {
	ret.x.Mul(&p.e, &p.f)
	ret.y.Mul(&p.g, &p.h)
	ret.t.Mul(&p.e, &p.h)
	ret.z.Mul(&p.f, &p.g)
	return
}

// TODO: This might go away.

// toDecaf_axtw converts the point to axtw coordinates.
// Note that this differs from the SetFrom functionality in that it operates on <foo>_base types.
// This is essentially equivalent to calling X_decaf_affine, Y_decaf_affine, Z_decaf_affine.
// internal use only.
func (p *point_efgh_base) toDecaf_axtw() (ret point_axtw_base) {
	// TODO ! Review
	// Note: Going eghj -> axtw directly is cheaper by 1 multiplication compared to going via xtw.
	// The reason is that we normalize first and then compute the t coordinate. This effectively saves comptuing t *= z^-1.
	p.normalize_affine()
	ret.x = p.e
	ret.y = p.h
	ret.t.Mul(&p.e, &p.h)
	return
}

// String is defined to satisfy the fmt.Stringer interface and allows points to be used in most fmt routines.
// Note that String() is defined on value receivers (as opposed to everything else) for an easier interface when using fmt routines.
//
// NOTE: Output format of String is not stable yet.
func (p point_efgh_base) String() (ret string) {
	ret = "E=" + p.e.String() + " F=" + p.f.String() + " G=" + p.g.String() + " H=" + p.h.String()
	return
}

// String is defined to satisfy the fmt.Stringer interface and allows points to be used in most fmt routines.
// Note that String() is defined on value receivers (as opposed to everything else) for an easier interface when using fmt routines.
//
// NOTE: Output format of String is not stable yet.
func (p Point_efgh_subgroup) String() (ret string) {
	ret = p.point_efgh_base.String()
	if !legendreCheckE1_FH(p.f, p.h) {
		ret += " [modified by +A]"
	}
	return
}

// Add performs curve point addition according to the elliptic curve group law.
// Use p.Add(&x, &y) for p = x + y.
func (p *Point_efgh_subgroup) Add(x, y CurvePointPtrInterfaceRead) {
	switch x := x.(type) {
	case *Point_xtw_subgroup:
		switch y := y.(type) {
		case *Point_xtw_subgroup:
			p.add_stt(&x.point_xtw_base, &y.point_xtw_base)
		case *Point_axtw_subgroup:
			p.add_sta(&x.point_xtw_base, &y.point_axtw_base)
		default: // including *Point_efgh_subgroup
			ensureSubgroupOnly(y)
			var y_conv Point_xtw_subgroup
			y_conv.SetFrom(y)
			p.add_stt(&x.point_xtw_base, &y_conv.point_xtw_base)
		}
	case *Point_axtw_subgroup:
		switch y := y.(type) {
		case *Point_xtw_subgroup:
			p.add_sta(&y.point_xtw_base, &x.point_axtw_base)
		case *Point_axtw_subgroup:
			p.add_saa(&x.point_axtw_base, &y.point_axtw_base)
		default: // including *Point_efgh_subgroup
			ensureSubgroupOnly(y)
			var y_conv Point_xtw_subgroup
			y_conv.SetFrom(y)
			p.add_sta(&y_conv.point_xtw_base, &x.point_axtw_base)
		}
	default: // including *Point_efgh_subgroup
		ensureSubgroupOnly(x)
		var x_conv Point_xtw_subgroup
		x_conv.SetFrom(x)
		switch y := y.(type) {
		case *Point_xtw_subgroup:
			p.add_stt(&x_conv.point_xtw_base, &y.point_xtw_base)
		case *Point_axtw_subgroup:
			p.add_sta(&x_conv.point_xtw_base, &y.point_axtw_base)
		default: // including *Point_efgh_subgroup
			var y_conv Point_xtw_subgroup
			y_conv.SetFrom(y)
			p.add_stt(&x_conv.point_xtw_base, &y_conv.point_xtw_base)
		}
	}
}

// Add performs curve point addition according to the elliptic curve group law.
// Use p.Add(&x, &y) for p = x + y.
func (p *Point_efgh_full) Add(x, y CurvePointPtrInterfaceRead) {
	var x_conv, y_conv Point_xtw_full
	x_conv.SetFrom(x)
	y_conv.SetFrom(y)
	p.add_safe_stt(&x_conv.point_xtw_base, &y_conv.point_xtw_base)
}

// Sub performs curve point subtraction according to the elliptic curve group law.
// Use p.Sub(&x, &y) for p = x - y.
func (p *Point_efgh_subgroup) Sub(x, y CurvePointPtrInterfaceRead) {
	switch x := x.(type) {
	case *Point_xtw_subgroup:
		switch y := y.(type) {
		case *Point_xtw_subgroup:
			p.sub_stt(&x.point_xtw_base, &y.point_xtw_base)
		case *Point_axtw_subgroup:
			p.sub_sta(&x.point_xtw_base, &y.point_axtw_base)
		default: // including *Point_efgh_subgroup
			ensureSubgroupOnly(y)
			var y_conv Point_xtw_subgroup
			y_conv.SetFrom(y)
			p.sub_stt(&x.point_xtw_base, &y_conv.point_xtw_base)
		}
	case *Point_axtw_subgroup:
		switch y := y.(type) {
		case *Point_xtw_subgroup:
			p.sub_sat(&x.point_axtw_base, &y.point_xtw_base)
		case *Point_axtw_subgroup:
			p.sub_saa(&x.point_axtw_base, &y.point_axtw_base)
		default: // including *Point_efgh_subgroup
			ensureSubgroupOnly(y)
			var y_conv Point_xtw_subgroup
			y_conv.SetFrom(y)
			p.sub_sat(&x.point_axtw_base, &y_conv.point_xtw_base)
		}
	default: // including *Point_efgh_subgroup
		ensureSubgroupOnly(x)
		var x_conv Point_xtw_subgroup
		x_conv.SetFrom(x)
		switch y := y.(type) {
		case *Point_xtw_subgroup:
			p.sub_stt(&x_conv.point_xtw_base, &y.point_xtw_base)
		case *Point_axtw_subgroup:
			p.sub_sta(&x_conv.point_xtw_base, &y.point_axtw_base)
		default: // including *Point_efgh_subgroup
			var y_conv Point_xtw_subgroup
			y_conv.SetFrom(y)
			p.sub_stt(&x_conv.point_xtw_base, &y_conv.point_xtw_base)
		}
	}
}

// Sub performs curve point subtraction according to the elliptic curve group law.
// Use p.Sub(&x, &y) for p = x - y.
func (p *Point_efgh_full) Sub(x, y CurvePointPtrInterfaceRead) {
	var x_conv, y_conv Point_xtw_full
	x_conv.SetFrom(x)
	y_conv.SetFrom(y)
	p.sub_safe_stt(&x_conv.point_xtw_base, &y_conv.point_xtw_base)
}

// Double computes the sum of a point with itself.
// p.Double(&x) means p = x + x.
//
// Note that x + x is always in the prime-order subgroup.
// As opposed to p.Add(&x, &x), p.Double(&x) works even if the type of p can only hold subgroup curve points and the type of x can hold general points.
func (p *point_efgh_base) Double(x CurvePointPtrInterfaceRead) {
	switch x := x.(type) {
	case *Point_xtw_full:
		p.double_st(&x.point_xtw_base)
	case *Point_xtw_subgroup:
		p.double_st(&x.point_xtw_base)
	case *Point_axtw_full:
		p.double_sa(&x.point_axtw_base)
		if x.IsNaP() {
			napEncountered("NaP encountered while Doubling axtw->efgh", false, x)
			*p = point_efgh_base{}
		}
	case *Point_axtw_subgroup:
		p.double_sa(&x.point_axtw_base)
		if x.IsNaP() {
			napEncountered("NaP encountered while Doubling axtw->efgh", false, x)
			*p = point_efgh_base{}
		}
	case *Point_efgh_full:
		p.double_ss(&x.point_efgh_base)
	case *Point_efgh_subgroup:
		p.double_ss(&x.point_efgh_base)
	}
}

// Neg computes the negative of the point wrt the elliptic curve group law.
// Use p.Neg(&input) for p = -input.
func (p *Point_efgh_subgroup) Neg(input CurvePointPtrInterfaceRead) {
	switch input := input.(type) {
	case *Point_efgh_subgroup:
		p.neg_ss(&input.point_efgh_base)
	default:
		ensureSubgroupOnly(input)
		p.SetFrom(input)
		p.NegEq()
	}
}

// Neg computes the negative of the point wrt the elliptic curve group law.
// Use p.Neg(&input) for p = -input.
func (p *Point_efgh_full) Neg(input CurvePointPtrInterfaceRead) {
	p.SetFrom(input)
	p.NegEq()
}

// Endo computes the efficient order-2 endomorphism on the given point described in the Bandersnatch paper.
//
// On the prime-order subgroup, this endomorphism acts as multiplication by the constant given as EndomorphismEivenvalue, which is
// a square root of -2.
func (p *Point_efgh_subgroup) Endo(input CurvePointPtrInterfaceRead) {
	switch input := input.(type) {
	case *Point_xtw_subgroup:
		p.computeEndomorphism_st(&input.point_xtw_base)
		// handle exceptional cases: They all give p.g==0.
		// Note that input and p cannot alias.
		if p.g.IsZero() {
			if input.IsNaP() {
				napEncountered("Computing endomorphism of NaP of type xtw_subgroup", false, input)
				p.point_efgh_base = point_efgh_base{}
			} else {
				if !input.IsNeutralElement() {
					panic("Internal error")
				}
				p.point_efgh_base = neutralElement_efghbase
			}
		}
	case *Point_axtw_subgroup:
		if input.IsNaP() {
			napEncountered("Computing endomorphism of NaP of type axtw_subgroup", false, input)
			p.point_efgh_base = point_efgh_base{}
			return
		}
		p.computeEndomorphism_sa(&input.point_axtw_base)
		// handle exceptional cases:
		if p.g.IsZero() {
			p.point_efgh_base = neutralElement_efghbase
		}
	case *Point_efgh_subgroup:
		p.computeEndomorphism_ss(&input.point_efgh_base)
	default:
		ensureSubgroupOnly(input)
		if input.IsNaP() {
			napEncountered("Computing endomorphism of NaP", false, input)
		}
		if input.IsNeutralElement() {
			p.point_efgh_base = neutralElement_efghbase
		} else {
			var inputConverted point_xtw_base
			inputConverted.x = input.X_decaf_projective()
			inputConverted.y = input.Y_decaf_projective()
			inputConverted.z = input.Z_decaf_projective()
			// computeEndomorphism_st promises not to use the input's t
			p.computeEndomorphism_st(&inputConverted)
		}
	}
}

// Endo computes the efficient order-2 endomorphism on the given point described in the Bandersnatch paper.
//
// On the prime-order subgroup, this endomorphism acts as multiplication by the constant given as EndomorphismEivenvalue, which is
// a square root of -2.
func (p *Point_efgh_full) Endo(input CurvePointPtrInterfaceRead) {
	// handle exceptions right away. This could be done more efficiently,
	// because not all exceptions can appear in all cases, but we keep things simple.
	if input.IsNaP() {
		napEncountered("Computing endomorphism of NaP", false, input)
		p.point_efgh_base = point_efgh_base{}
		return
	}
	if input.IsAtInfinity() {
		p.point_efgh_base = orderTwoPoint_efghbase
		return
	}
	// check whether input is neutral element of affine two-torsion.
	// This can be done by checking whether X is zero.
	inputX := input.X_decaf_projective()
	if inputX.IsZero() {
		p.point_efgh_base = neutralElement_efghbase
		return
	}
	switch input := input.(type) {
	case *Point_xtw_subgroup:
		p.computeEndomorphism_st(&input.point_xtw_base)
	case *Point_xtw_full:
		p.computeEndomorphism_st(&input.point_xtw_base)
	case *Point_axtw_subgroup:
		p.computeEndomorphism_sa(&input.point_axtw_base)
	case *Point_axtw_full:
		p.computeEndomorphism_sa(&input.point_axtw_base)
	case *Point_efgh_full:
		p.computeEndomorphism_ss(&input.point_efgh_base)
	case *Point_efgh_subgroup:
		p.computeEndomorphism_ss(&input.point_efgh_base)
	default:
		// make a copy of the point in point_xtw_base coordinates.
		// Note that we can work with a P vs. P+A ambiguity, because
		// Endo(P) == Endo(P+A) anyway.
		var inputConverted point_xtw_base
		inputConverted.x = inputX
		inputConverted.y = input.Y_decaf_projective()
		inputConverted.z = input.Z_decaf_projective()
		// computeEndomorphism_st promises not to use t
		p.computeEndomorphism_st(&inputConverted)
	}
}

// SetNeutral sets the given point to the neutral element of the curve.
func (p *point_efgh_base) SetNeutral() {
	*p = neutralElement_efghbase
}

// AddEq adds (via the elliptic curve group addition law) the given curve point x to the received p, overwriting p.
//
// p.AddEq(&x) is equivalent to p.AddEq(&p, &x)
func (p *Point_efgh_full) AddEq(input CurvePointPtrInterfaceRead) {
	p.Add(p, input)
}

// AddEq adds (via the elliptic curve group addition law) the given curve point x to the received p, overwriting p.
//
// p.AddEq(&x) is equivalent to p.AddEq(&p, &x)
func (p *Point_efgh_subgroup) AddEq(input CurvePointPtrInterfaceRead) {
	p.Add(p, input)
}

// SubEq subtracts (via the elliptic curve group addition law) the given curve point x (in any coordinate format) from the received p, overwriting p.
func (p *Point_efgh_full) SubEq(input CurvePointPtrInterfaceRead) {
	p.Sub(p, input)
}

// SubEq subtracts (via the elliptic curve group addition law) the curve point x from the received p, overwriting p.
//
// p.SubEq(&x) is equivalent to p.SubEq(&p, &x)
func (p *Point_efgh_subgroup) SubEq(input CurvePointPtrInterfaceRead) {
	p.Sub(p, input)
}

// DoubleEq doubles the received point p, overwriting p.
//
// p.DoubleEq() is equivalent to p.Double(&p)
func (p *point_efgh_base) DoubleEq() {
	p.double_ss(p)
}

// NegEq replaces the given point by its negative (wrt the elliptic curve group addition law).
//
// p.NegEq() is equivalent to p.NegEq(&p)
func (p *point_efgh_base) NegEq() {
	p.e.NegEq()
}

// EndoEq applies the endomorphism on the given point p, overwriting it.
//
// p.EndoEq() is equivalent to p.Endo(&p).
func (p *Point_efgh_subgroup) EndoEq() {
	p.computeEndomorphism_ss(&p.point_efgh_base)
}

// EndoEq applies the endomorphism on the given point p, overwriting it.
//
// p.EndoEq() is equivalent to p.Endo(&p).
func (p *Point_efgh_full) EndoEq() {
	if p.IsAtInfinity() {
		p.point_efgh_base = orderTwoPoint_efghbase
	} else {
		p.computeEndomorphism_ss(&p.point_efgh_base)
	}
}

// SetFromSubgroupPoint sets the receiver to a copy of the input, which needs to be in the prime-order subgroup.
// This method can be used to convert from point types capable of holding points not in the prime-order subgroup to point types that do not.
// The second argument needs to be either TrustedInput or UntrustedInput.
// For UntrustedInput, we actually check whether the input is in the subgroup; For TrustedInput, we assume it to be the case.
// The return value indicates success. On failure, the receiver is unchanged.
//
// NOTE: Calling this checks for NaPs even for TrustedInput.
// We make no guarantees whatsoever when calling it on points outside the subgroup with TrustedInput.
func (p *Point_efgh_subgroup) SetFromSubgroupPoint(input CurvePointPtrInterfaceRead, trusted IsPointTrusted) (ok bool) {
	if input.IsNaP() {
		napEncountered("Converting NaP point to efgh_subgroup", false, input)
		// *p = Point_efgh_subgroup{}
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
	// Note: p cannot alias input if we get here.
	switch input := input.(type) {
	case *Point_efgh_full:
		p.point_efgh_base = input.point_efgh_base
	case *Point_xtw_full:
		p.e = input.x
		p.f = input.z // Note: Cannot be 0 for points in subgroup
		p.g = input.z // Note: Cannot be 0 for points in subgroup
		p.h = input.y
	case *Point_axtw_full:
		p.e = input.x
		p.f.SetOne()
		p.g.SetOne()
		p.h = input.y
	default:

		p.e, p.h, p.f = input.XYZ_projective()
		p.g = p.f
	}
	return true
}

// SetFromSubgroupPoint sets the receiver to a copy of the input, which needs to be in the prime-order subgroup.
// This method can be used to convert from point types capable of holding points not in the prime-order subgroup to point types that do not.
// The second argument needs to be either TrustedInput or UntrustedInput.
// For UntrustedInput, we actually check whether the input is in the subgroup; For TrustedInput, we assume it to be the case.
// The return value indicates success. On failure, the receiver is unchanged.
//
// NOTE: Calling this checks for NaPs even for TrustedInput.
// We make no guarantees whatsoever when calling it on points outside the subgroup with TrustedInput.
func (p *Point_efgh_full) SetFromSubgroupPoint(input CurvePointPtrInterfaceRead, trusted IsPointTrusted) (ok bool) {
	if input.IsNaP() {
		napEncountered("Converting NaP to efgh", false, input)
		// *p = Point_efgh_full{}
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
// NOTE: To intialize a Point of type Point_efgh_subgroup with an input of a type that can hold points outside the subgroup, you need to use SetFromSubgroupPoint instead.
func (p *Point_efgh_subgroup) SetFrom(input CurvePointPtrInterfaceRead) {
	switch input := input.(type) {
	case *Point_efgh_subgroup:
		*p = *input
	case *Point_xtw_subgroup:
		p.e = input.x
		p.f = input.z
		p.g = input.z
		p.h = input.y
	case *Point_axtw_subgroup:
		p.e = input.x
		p.f.SetOne()
		p.g.SetOne()
		p.h = input.y
	default:
		// Note: p cannot alias subgroup
		if input.IsNaP() {
			napEncountered("Converting NaP of unknown type to efgh_subgroup", false, input)
			*p = Point_efgh_subgroup{}
			return
		}
		ensureSubgroupOnly(input)
		p.e = input.X_decaf_projective()
		p.f = input.Z_decaf_projective()
		p.g = p.f
		p.h = input.Y_decaf_projective()
	}
}

// SetFrom initializes the point from the given input point (which may have a different coordinate format).
//
// NOTE: To intialize a Point of type Point_efgh_subgroup with an input of a type that can hold points outside the subgroup, you need to use SetFromSubgroupPoint instead.
func (p *Point_efgh_full) SetFrom(input CurvePointPtrInterfaceRead) {
	switch input := input.(type) {
	case *Point_efgh_subgroup:
		input.normalizeSubgroup()
		p.point_efgh_base = input.point_efgh_base
	case *Point_efgh_full:
		*p = *input
	default:
		if input.IsNaP() {
			napEncountered("Converting NaP to efgh_full", false, input)
			*p = Point_efgh_full{}
		} else if input.IsAtInfinity() {
			if input.(CurvePointPtrInterfaceDistinguishInfinity).IsE1() {
				p.point_efgh_base = exceptionalPoint_1_efghbase
			} else {
				p.point_efgh_base = exceptionalPoint_2_efghbase
			}
		} else {
			p.e, p.h, p.f = input.XYZ_projective()
			p.g = p.f
		}
	}
}

// IsInSubgroup checks whether the given curve point is in the p253 prime-order subgroup.
func (p *Point_efgh_full) IsInSubgroup() bool {
	return legendreCheckA_EG(p.e, p.g) && legendreCheckE1_FH(p.f, p.h)
}

// Validate checks whether the point is a valid curve point.
//
// NOTE: Outside of NaPs, it should not be possible to create points that fail Validate when using the interface correctly.
// Validate is used only in testing and is required by the CurvePointPtrInterfaceTestSample interface.
func (p *point_efgh_base) Validate() bool {
	return p.isPointOnCurve()
}

// Validate checks whether the point is a valid curve point.
//
// NOTE: Outside of NaPs, it should not be possible to create points that fail Validate when using the interface correctly.
// Validate is used only in testing and is required by the CurvePointPtrInterfaceTestSample interface.
func (p *Point_efgh_subgroup) Validate() bool {
	return p.point_efgh_base.isPointOnCurve() && legendreCheckA_EG(p.e, p.g)
}

// sampleRandomUnsafe samples a (pseudo-)random curvepoint.
// It is used in testing only and required by the CurvePointPtrInterfaceTestValue interface.
//
// NOTE: While good enough for testing, the randomness quality is insufficient for cryptographic purposes.
// This is why we do not export this.
func (p *Point_efgh_full) sampleRandomUnsafe(rnd *rand.Rand) {
	var p_axtw Point_axtw_full
	p_axtw.sampleRandomUnsafe(rnd)
	p.SetFrom(&p_axtw)
	p.rerandomizeRepresentation(rnd)
}

// sampleRandomUnsafe samples a (pseudo-)random curvepoint.
// It is used in testing only and required by the CurvePointPtrInterfaceTestValue interface.
//
// NOTE: While good enough for testing, the randomness quality is insufficient for cryptographic purposes.
// This is why we do not export this.
func (p *Point_efgh_subgroup) sampleRandomUnsafe(rnd *rand.Rand) {
	var p_axtw Point_axtw_subgroup
	p_axtw.sampleRandomUnsafe(rnd)
	p.SetFrom(&p_axtw)
	p.rerandomizeRepresentation(rnd)
}

// SetAffineTwoTorsion sets the point to the affine-order two point.
// This function is required in order to satisfy the curvePointPtrInterfaceTestSampleA interface, which
// our testing framework mandates that Point_efgh_full must satisfy.
func (p *Point_efgh_full) SetAffineTwoTorsion() {
	p.point_efgh_base = orderTwoPoint_efghbase
}

// SetE1 sets the point to the E1 point at infinity.
//
// This function is required in order to satisfy the curvePointPtrInterfaceTestSampleE interface, which
// our testing framework mandates that Point_efgh_full must satisfy.
func (p *Point_efgh_full) SetE1() {
	p.point_efgh_base = exceptionalPoint_1_efghbase
}

// SetE1 sets the point to the E1 point at infinity.
//
// This function is required in order to satisfy the curvePointPtrInterfaceTestSampleE interface, which
// our testing framework mandates that Point_efgh_full must satisfy.
func (p *Point_efgh_full) SetE2() {
	p.point_efgh_base = exceptionalPoint_2_efghbase
}
