package bandersnatch

import "math/rand"

// Point_efgh describes points (usually on the p253-subgroup of) the bandersnatch curve in E:G, H:F - coordinates (called double-projective or "efgh"-coos), i.e.
// we represent X/Z as E/G and Y/Z as H/F. From a computational view, this effectively means that we use a separate denominator for X and Y (instead of a joint one Z).
// We can recover X:Y:Z coordinates by computing Z = F*G, X = E*F, Y = G*H. Then T = E*H. This is meaningful even if one of E,G is zero. There are no rational points with F=0 or H=0.
// Observe that in fact all default formulae in extended twisted edwards coordinates *produce* points in such efgh coordinates and then transform them into the desired form
// Using double-projective coordinates can be used to make this explicit and can save computation if a coordinate is unused:
// The doubling formula and the endomorphism can be expressed in double-projective coordinates by first converting to extended twisted edwards and then computing the double/endo(rather than the other way round).
// Since these formulae do not use the input's t coordinate, this saves a multiplication.
// (In fact, for the endomorphism, some further optimisation is possible if the input is in efgh-coordinates)
// On the p253-subgroup, the only coordinate that may be zero is actually e.

// Note: Conversion from X:Y:T:Z to EFGH is available as e.g.
// E:=X, F:=X, G:=Z, H:=T or
// E:=T, F:=X, G:=Y, H:=T or
// E:=X, F:=Z, G:=Z, H:=Y or
// (The first two options have singularities at neutral and affine-order-2, the third option at the points at infinity)

type point_efgh_base struct {
	thisCurvePointCanRepresentFullCurve
	thisCurvePointCanRepresentInfinity
	e FieldElement
	f FieldElement
	g FieldElement
	h FieldElement
}

type Point_efgh_full struct {
	point_efgh_base
}

type Point_efgh_subgroup struct {
	thisCurvePointCanOnlyRepresentSubgroup
	thisCurvePointCannotRepresentInfinity
	point_efgh_base
}

var (
	NeutralElement_efgh     = point_efgh_base{e: FieldElementZero, f: FieldElementOne, g: FieldElementOne, h: FieldElementOne}        // Note: g!=0 is actually arbitrary.
	orderTwoPoint_efgh      = point_efgh_base{e: FieldElementZero, f: FieldElementOne, g: FieldElementOne, h: FieldElementMinusOne}   // Note: g!=0 is actually arbitrary.
	exceptionalPoint_1_efgh = point_efgh_base{e: FieldElementOne, f: squareRootDbyA_fe, g: FieldElementZero, h: FieldElementOne}      // Note: e!=0 is actually arbitrary.
	exceptionalPoint_2_efgh = point_efgh_base{e: FieldElementOne, f: squareRootDbyA_fe, g: FieldElementZero, h: FieldElementMinusOne} // Note: e!=0 is actually arbitrary.
)

// normalize_affine puts the point in an equivalent "normalized" state with f==g==1.
// NaPs will be put into the uninitialized, default e==f==g==h==0 NaP state. Points at infinity panic.
func (p *point_efgh_base) normalize_affine() {
	if p.is_normalized() {
		return
	}
	var temp FieldElement
	temp.Mul(&p.f, &p.g)
	if temp.IsZero() {
		if p.IsNaP() {
			napEncountered("Trying to normalize singular point", false, p)
			// If the error handler did not panic, we intentionally set the NaP p to a "full" NaP with all coos 0 (rather than at least 2).
			// This has the effect that all conversion routines that start by calling normalize_affine will only need to worry about NaPs with e==f==g==h==0
			*p = point_efgh_base{}
			return
		}
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

func (p *point_efgh_base) rerandomizeRepresentation(rnd *rand.Rand) {
	var m FieldElement
	m.setRandomUnsafeNonZero(rnd)
	p.e.MulEq(&m)
	p.f.MulEq(&m)
	m.setRandomUnsafeNonZero(rnd)
	p.g.MulEq(&m)
	p.h.MulEq(&m)
}

func (p *Point_efgh_subgroup) rerandomizeRepresentation(rnd *rand.Rand) {
	p.point_efgh_base.rerandomizeRepresentation(rnd)
	if rnd.Intn(2) == 0 {
		p.flipDecaf()
	}
}

// is_normalized checks whether p is in that form.
func (p *point_efgh_base) is_normalized() bool {
	return p.f.IsOne() && p.g.IsOne()
}

func (p *Point_efgh_subgroup) normalizeSubgroup() {
	if !legendreCheckE1_FH(p.e, p.g) {
		p.flipDecaf()
	}
}

// IsAtInfinity tests whether the point is an infinite (neccessarily order-2) point.
// Note that these points are NOT in the p253-subgroup, so these are not supposed to appear under normal operation.
func (p *point_efgh_base) IsAtInfinity() bool {
	if p.IsNaP() {
		return napEncountered("NaP encountered when asking where efgh-point is at infinity", true, p)
	}
	// The only valid (non-NaP) points with g==0 are are those at infinity
	return p.g.IsZero()
}

func (p *Point_efgh_subgroup) IsAtInfinity() bool {
	if p.IsNaP() {
		return napEncountered("NaP encountered when asking where efgh-point is at infinity", true, p)
	}
	return false
}

func (p *point_efgh_base) flipDecaf() {
	// this preserves is_normalized
	p.e.NegEq()
	p.h.NegEq()
}

func (p *Point_efgh_subgroup) HasDecaf() bool {
	return true
}

// X_projective returns the X coordinate of the given point p in projective twisted Edwards coordinates.
// Note that calling functions on P other than X_projective(), Y_projective(), T_projective(), Z_projective() might change the representations of P at will,
// so callers must not interleave calling other functions.
func (p *Point_efgh_subgroup) X_projective() (X FieldElement) {
	p.normalizeSubgroup()
	X.Mul(&p.e, &p.f)
	return
}

func (p *Point_efgh_full) X_projective() (X FieldElement) {
	X.Mul(&p.e, &p.f)
	return
}

func (p *point_efgh_base) X_decaf_projective() (X FieldElement) {
	X.Mul(&p.e, &p.f)
	return
}

// Y_projective returns the Y coordinate of the given point p in projective twisted Edwards coordinates.
// Note that calling functions on p other than X_projective(), Y_projective(), T_projective(), Z_projective() might change the representations of p at will,
// so callers must not interleave calling other functions.
func (p *Point_efgh_subgroup) Y_projective() (Y FieldElement) {
	p.normalizeSubgroup()
	Y.Mul(&p.g, &p.h)
	return
}

// Y_projective returns the Y coordinate of the given point p in projective twisted Edwards coordinates.
// Note that calling functions on p other than X_projective(), Y_projective(), T_projective(), Z_projective() might change the representations of p at will,
// so callers must not interleave calling other functions.
func (p *Point_efgh_full) Y_projective() (Y FieldElement) {
	Y.Mul(&p.g, &p.h)
	return
}

func (p *point_efgh_base) Y_decaf_projective() (Y FieldElement) {
	Y.Mul(&p.g, &p.h)
	return
}

// T_projective returns the T coordinate of the given point p in projective twisted Edwards coordinates.
// Note that calling functions on p other than X_projective(), Y_projective(), T_projective(), Z_projective() might change the representations of p at will,
// so callers must not interleave calling other functions.
func (p *Point_efgh_subgroup) T_projective() (T FieldElement) {
	p.normalizeSubgroup()
	T.Mul(&p.e, &p.h)
	return
}

// T_projective returns the T coordinate of the given point p in projective twisted Edwards coordinates.
// Note that calling functions on p other than X_projective(), Y_projective(), T_projective(), Z_projective() might change the representations of p at will,
// so callers must not interleave calling other functions.
func (p *Point_efgh_full) T_projective() (T FieldElement) {
	T.Mul(&p.e, &p.h)
	return
}

func (p *point_efgh_base) T_decaf_projective() (T FieldElement) {
	T.Mul(&p.e, &p.h)
	return
}

// Z_projective returns the Z coordinate of the given point p in projective twisted Edwards coordinates.
// Note that calling functions on p other than X_projective(), Y_projective(), T_projective(), Z_projective() might change the representations of p at will,
// so callers must not interleave calling other functions.
func (p *Point_efgh_subgroup) Z_projective() (Z FieldElement) {
	p.normalizeSubgroup()
	Z.Mul(&p.f, &p.g)
	return
}

// Z_projective returns the Z coordinate of the given point p in projective twisted Edwards coordinates.
// Note that calling functions on p other than X_projective(), Y_projective(), T_projective(), Z_projective() might change the representations of p at will,
// so callers must not interleave calling other functions.
func (p *Point_efgh_full) Z_projective() (Z FieldElement) {
	Z.Mul(&p.f, &p.g)
	return
}

func (p *point_efgh_base) Z_decaf_projective() (Z FieldElement) {
	Z.Mul(&p.f, &p.g)
	return
}

func (p *Point_efgh_subgroup) XYZ_projective() (X FieldElement, Y FieldElement, Z FieldElement) {
	p.normalizeSubgroup()
	X.Mul(&p.e, &p.f)
	Y.Mul(&p.g, &p.h)
	Z.Mul(&p.f, &p.g)
	return
}

func (p *Point_efgh_full) XYZ_projective() (X FieldElement, Y FieldElement, Z FieldElement) {
	X.Mul(&p.e, &p.f)
	Y.Mul(&p.g, &p.h)
	Z.Mul(&p.f, &p.g)
	return
}

func (p *Point_efgh_subgroup) XYTZ_projective() (X FieldElement, Y FieldElement, T FieldElement, Z FieldElement) {
	p.normalizeSubgroup()
	X.Mul(&p.e, &p.f)
	Y.Mul(&p.g, &p.h)
	T.Mul(&p.e, &p.h)
	Z.Mul(&p.f, &p.g)
	return
}

func (p *Point_efgh_full) XYTZ_projective() (X FieldElement, Y FieldElement, T FieldElement, Z FieldElement) {
	X.Mul(&p.e, &p.f)
	Y.Mul(&p.g, &p.h)
	T.Mul(&p.e, &p.h)
	Z.Mul(&p.f, &p.g)
	return
}

// X_affine returns the X coordinate of the given point in affine twisted Edwards coordinates, (i.e. X/Z in projective coos)
func (p *Point_efgh_subgroup) X_affine() FieldElement {
	p.normalize_affine()
	p.normalizeSubgroup()
	return p.e
}

// X_affine returns the X coordinate of the given point in affine twisted Edwards coordinates, (i.e. X/Z in projective coos)
func (p *Point_efgh_full) X_affine() FieldElement {
	p.normalize_affine()
	return p.e
}

func (p *point_efgh_base) X_decaf_affine() FieldElement {
	p.normalize_affine()
	return p.e
}

// Y_affine returns the Y coordinate of the given point in affine twisted Edwards coordinates, (i.e. Y/Z in projective coos)
func (p *Point_efgh_subgroup) Y_affine() FieldElement {
	p.normalize_affine()
	p.normalizeSubgroup()
	return p.h
}

// Y_affine returns the Y coordinate of the given point in affine twisted Edwards coordinates, (i.e. Y/Z in projective coos)
func (p *Point_efgh_full) Y_affine() FieldElement {
	p.normalize_affine()
	return p.h
}

func (p *point_efgh_base) Y_decaf_affine() FieldElement {
	p.normalize_affine()
	return p.h
}

// T_affine returns the T coordinate of the given point in affine twisted Edwards coordinates, (i.e. T/Z == X*Y/Z^2 in projective coos)
func (p *Point_efgh_subgroup) T_affine() (T FieldElement) {
	p.normalize_affine()
	p.normalizeSubgroup()
	T.Mul(&p.e, &p.h)
	return
}

// T_affine returns the T coordinate of the given point in affine twisted Edwards coordinates, (i.e. T/Z == X*Y/Z^2 in projective coos)
func (p *Point_efgh_full) T_affine() (T FieldElement) {
	p.normalize_affine()
	T.Mul(&p.e, &p.h)
	return
}

func (p *point_efgh_base) T_decaf_affine() (T FieldElement) {
	p.normalize_affine()
	T.Mul(&p.e, &p.h)
	return
}

func (p *Point_efgh_subgroup) XY_affine() (X FieldElement, Y FieldElement) {
	p.normalizeSubgroup()
	p.normalize_affine()
	return p.e, p.h
}

func (p *Point_efgh_full) XY_affine() (X FieldElement, Y FieldElement) {
	p.normalize_affine()
	return p.e, p.h
}

func (p *Point_efgh_subgroup) XYT_affine() (X FieldElement, Y FieldElement, T FieldElement) {
	p.normalizeSubgroup()
	p.normalize_affine()
	X = p.e
	Y = p.f
	T.Mul(&X, &Y)
	return
}

func (p *Point_efgh_full) XYT_affine() (X FieldElement, Y FieldElement, T FieldElement) {
	p.normalize_affine()
	X = p.e
	Y = p.f
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

// IsNeutralElement checks if the point P is the neutral element of the curve.
// Use IsNeutralElement_FullCurve if you do not want this identification.
func (p *Point_efgh_full) IsNeutralElement() bool {
	// The only valid points with e==0 are the neutral element and the affine order-2 point
	if p.IsNaP() {
		return napEncountered("Comparing NaP with neutral element for efgh_full", true, p)
	}
	return p.e.IsZero() && p.f.IsEqual(&p.h)
}

func (p *point_efgh_base) IsE1() bool {
	var tmp FieldElement
	tmp.Mul(&p.h, &squareRootDbyA_fe)
	return tmp.IsEqual(&p.f)
}

func (p *point_efgh_base) Clone() interface{} {
	var copy point_efgh_base = *p
	return &copy
}

func (p *Point_efgh_full) Clone() interface{} {
	var copy Point_efgh_full = *p
	return &copy
}

func (p *Point_efgh_subgroup) Clone() interface{} {
	var copy Point_efgh_subgroup = *p
	return &copy
}

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

func (p *Point_efgh_full) IsAtInfinity() bool {
	if p.IsNaP() {
		return napEncountered("NaP encountered when asking where efgh-point is at infinity", true, p)
	}
	// The only valid (non-NaP) points with g==0 are are those at infinity
	return p.g.IsZero()
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

func (P *point_efgh_base) ToDecaf_xtw() (ret point_xtw_base) {
	ret.x.Mul(&P.e, &P.f)
	ret.y.Mul(&P.g, &P.h)
	ret.t.Mul(&P.e, &P.h)
	ret.z.Mul(&P.f, &P.g)
	return
}

func (P *point_efgh_base) ToDecaf_axtw() (ret point_axtw_base) {
	// TODO ! Review
	// Note: Going eghj -> axtw directly is cheaper by 1 multiplication compared to going via xtw.
	// The reason is that we normalize first and then compute the t coordinate. This effectively saves comptuing t *= z^-1.
	P.normalize_affine()
	ret.x = P.e
	ret.y = P.h
	ret.t.Mul(&P.e, &P.h)
	return
}

// String() returns a (somewhat) human-readable string describing the point. Useful for debugging.
func (p *point_efgh_base) String() (ret string) {
	ret = "E=" + p.e.String() + " F=" + p.f.String() + " G=" + p.g.String() + " H=" + p.h.String()
	return
}

func (p *Point_efgh_subgroup) String() (ret string) {
	ret = p.point_efgh_base.String()
	if !legendreCheckE1_FH(p.f, p.h) {
		ret += " [modified by +A]"
	}
	return
}

// z.Add(x,y) computes z = x+y according to the elliptic curve group law.
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

func (p *Point_efgh_full) Add(x, y CurvePointPtrInterfaceRead) {
	panic(0)
}

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

func (p *Point_efgh_full) Sub(x, y CurvePointPtrInterfaceRead) {
	panic(0)
}

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

// z.Neg(x) computes z = -x according to the elliptic curve group law.
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

func (p *Point_efgh_full) Neg(input CurvePointPtrInterfaceRead) {
	p.SetFrom(input)
	p.NegEq()
}

// z.Endo(x) compute z = \Psi(x) where \Psi is the non-trivial degree-2 endomorphism described in the bandersnatch paper.
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
				p.point_efgh_base = NeutralElement_efgh
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
			p.point_efgh_base = NeutralElement_efgh
		}
	case *Point_efgh_subgroup:
		p.computeEndomorphism_ss(&input.point_efgh_base)
	default:
		ensureSubgroupOnly(input)
		if input.IsNaP() {
			napEncountered("Computing endomorphism of NaP", false, input)
		}
		if input.IsNeutralElement() {
			p.point_efgh_base = NeutralElement_efgh
		} else {
			var inputConverted point_xtw_base
			inputConverted.x = input.X_decaf_projective()
			inputConverted.y = input.Y_decaf_projective()
			inputConverted.z = input.Z_decaf_projective()
			// computeEndomorphism_st promises not to use t
			p.computeEndomorphism_st(&inputConverted)
		}
	}
}

func (p *Point_efgh_full) Endo(input CurvePointPtrInterfaceRead) {
	// handle exceptions right away. This could be done more efficiently,
	// because not all exceptions can appear in all cases, but we keep things simple.
	if input.IsNaP() {
		napEncountered("Computing endomorphism of NaP", false, input)
		p.point_efgh_base = point_efgh_base{}
		return
	}
	if input.IsAtInfinity() {
		p.point_efgh_base = orderTwoPoint_efgh
		return
	}
	inputX := input.X_decaf_projective()
	if inputX.IsZero() {
		p.point_efgh_base = NeutralElement_efgh
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
		var inputConverted point_xtw_base
		inputConverted.x = inputX
		inputConverted.y = input.Y_decaf_projective()
		inputConverted.z = input.Z_decaf_projective()
		// computeEndomorphism_st promises not to use t
		p.computeEndomorphism_st(&inputConverted)
	}
}

func (p *point_efgh_base) SetNeutral() {
	*p = NeutralElement_efgh
}

// AddEq adds (via the elliptic curve group addition law) the given curve point x (in any coordinate format) to the received p, overwriting p.
func (p *Point_efgh_full) AddEq(input CurvePointPtrInterfaceRead) {
	p.Add(p, input)
}

// AddEq adds (via the elliptic curve group addition law) the given curve point x (in any coordinate format) to the received p, overwriting p.
func (p *Point_efgh_subgroup) AddEq(input CurvePointPtrInterfaceRead) {
	p.Add(p, input)
}

// SubEq subtracts (via the elliptic curve group addition law) the given curve point x (in any coordinate format) from the received p, overwriting p.
func (p *Point_efgh_full) SubEq(input CurvePointPtrInterfaceRead) {
	p.Sub(p, input)
}

// SubEq subtracts (via the elliptic curve group addition law) the given curve point x (in any coordinate format) from the received p, overwriting p.
func (p *Point_efgh_subgroup) SubEq(input CurvePointPtrInterfaceRead) {
	p.Sub(p, input)
}

// DoubleEq doubles the received point p, overwriting p.
func (p *point_efgh_base) DoubleEq() {
	p.double_ss(p)
}

// NeqEq replaces the given point by its negative (wrt the elliptic curve group addition law)
func (p *point_efgh_base) NegEq() {
	p.e.NegEq()
}

// EndoEq applies the endomorphism on the given point. p.EndoEq() is shorthand for p.Endo(&p).
func (p *Point_efgh_subgroup) EndoEq() {
	p.computeEndomorphism_ss(&p.point_efgh_base)
}

// EndoEq applies the endomorphism on the given point. p.EndoEq() is shorthand for p.Endo(&p).
func (p *Point_efgh_full) EndoEq() {
	if p.IsAtInfinity() {
		p.point_efgh_base = orderTwoPoint_efgh
	} else {
		p.computeEndomorphism_ss(&p.point_efgh_base)
	}
}

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
			if input.(CurvePointPtrInterfaceReadCanDistinguishInfinity).IsE1() {
				p.point_efgh_base = exceptionalPoint_1_efgh
			} else {
				p.point_efgh_base = exceptionalPoint_2_efgh
			}
		} else {
			p.e, p.h, p.f = input.XYZ_projective()
			p.g = p.f
		}
	}
}

func (p *Point_efgh_full) IsInSubgroup() bool {
	return legendreCheckA_EG(p.e, p.g) && legendreCheckE1_FH(p.f, p.h)
}

func (p *point_efgh_base) Validate() bool {
	return p.isPointOnCurve()
}

func (p *Point_efgh_subgroup) Validate() bool {
	return p.point_efgh_base.isPointOnCurve() && legendreCheckA_EG(p.e, p.g)
}

func (p *Point_efgh_full) sampleRandomUnsafe(rnd *rand.Rand) {
	var p_axtw Point_axtw_full
	p_axtw.sampleRandomUnsafe(rnd)
	p.SetFrom(&p_axtw)
	p.rerandomizeRepresentation(rnd)
}

func (p *Point_efgh_subgroup) sampleRandomUnsafe(rnd *rand.Rand) {
	var p_axtw Point_axtw_subgroup
	p_axtw.sampleRandomUnsafe(rnd)
	p.SetFrom(&p_axtw)
	p.rerandomizeRepresentation(rnd)
}

func (p *Point_efgh_full) SetAffineTwoTorsion() {
	p.point_efgh_base = orderTwoPoint_efgh
}

func (p *Point_efgh_full) SetE1() {
	p.point_efgh_base = exceptionalPoint_1_efgh
}

func (p *Point_efgh_full) SetE2() {
	p.point_efgh_base = exceptionalPoint_2_efgh
}
