package bandersnatch

import (
	"math/big"
)

// Point_xtw describes points on the p253-subgroup of the Bandersnatch curve in extended twisted Edwards coordinates.
// Extended means that we additionally store T with T = X*Y/Z. Note that Z is never 0 for points in the subgroup, but see code comment about desingularisation.)
// cf. https://iacr.org/archive/asiacrypt2008/53500329/53500329.pdf
type Point_xtw struct {
	thisCurvePointCanOnlyRepresentSubgroup
	thisCurvePointCanRepresentInfinity
	x FieldElement
	y FieldElement
	z FieldElement
	t FieldElement
}

/*
	NOTE: Points described by Point_xtw should really been seen as solutions to the set of homogeneous equations

	ax^2 + y^2 = z^2 + dt^2
	x*y = z*t

	with addition law for P3 = P1 + P2 given by:
	X3 = (X1Y2 + Y1X2)(Z1Z2 - dT1T2)
	Y3 = (Y1Y2 - aX1X2)(Z1Z2 + dT1T2)
	T3 = (X1Y2 + Y1X2)(Y1Y2-aX1X2)
	Z3 = (Z1Z2 - dT1T2)(Z1Z2 + dT1T2)

	which we call the extended twisted Edwards model. We treat this as a curve model (like Weierstrass, Montgomery, (twisted) Edwards) rather than a redundant coordinate representaion.

	Clearly, the set of affine solutions corresponds exactly to the set of affine solutions of the usual twisted Edwards equation ax^2 + y^2 = 1+dx^2y^2
	(with z==1, t==x*y), but there are differences in the behaviour at infinity:
	Notably, the twisted Edwards curve has 2+2 points at infinity and the curve is actually singular there:
	Those are double (in the sense that a desingularization results in two points) points at (1:0:0), (0:1:0) each.
	By contrast, the extended twisted Edwards model has no singularities for a != +/-d (over the algebraic closure, to be clear).
	In fact, the additional t coordinate both improves efficiency and is a very convenient desingularization, where things become more clear.
	The (not neccessarily rational) points at infinity (z==0) of this model are in (x:y:t:z) coordinates:
	(0:sqrtz(d):1:0), (0:-sqrt(d):1:0), (sqrt(d/a):0:1:0), (-sqrt(d/a):0:1:0)
	The first two point have order 4 (doubling them gives (0:-1:0:1)), the latter two points have order 2.
	Now, in the case usually considered in the literature, d is a non-square and a is a square.
	Then all these points at infinity are actually not rational and we even get a unified point addition law that works for all rational points.

	In the bandersnatch case, both a and d are non-squares. This means we get two bona-fide rational(!) points at infinity of order 2.
	The addition law above no longer works in all cases. A lengthy analysis (TODO: Make a pdf and write up the proof or find one in literature) shows that the following holds

	Theorem:
	for P1,P2 rational, the extended Edwards addition law given above for P1 + P2 does not work if and only if P1 - P2 is a (rational, order-2) point at infinity.

	Consequences:
	The addition law works for all points in the subgroup of size 2*p253, generated by the large-prime p253 subgroup and the affine point of order 2.
	If P1,P2 are both contained in a cyclic subgroup generated by Q, then the addition law can only fail in the following cases:
		One of P1,P2 is the neutral element, the other one is equal to Q and is a point at infinity.
		Q has order 2*p253, P1 = alpha * Q, P2 = beta * Q with alpha-beta == p253 mod 2*p253. We can actually ensure that never happens in our exponentiation algorithms.
*/

// example point on the subgroup specified in the bandersnatch paper
var example_generator_x *big.Int = initIntFromString("0x29c132cc2c0b34c5743711777bbe42f32b79c022ad998465e1e71866a252ae18")
var example_generator_y *big.Int = initIntFromString("0x2a6c669eda123e0f157d8b50badcd586358cad81eee464605e3167b6cc974166")
var example_generator_t *big.Int = new(big.Int).Mul(example_generator_x, example_generator_y)
var example_generator_xtw Point_xtw = func() (ret Point_xtw) {
	ret.x.SetInt(example_generator_x)
	ret.y.SetInt(example_generator_y)
	ret.t.SetInt(example_generator_t)
	ret.z.SetOne()
	return
}()

/*
	Basic functions for Point_xtw
*/

// NeutralElement_<foo> denotes the Neutral Element of the Bandersnatch curve in <foo> coordinates.
var (
	NeutralElement_xtw Point_xtw = Point_xtw{x: FieldElementZero, y: FieldElementOne, t: FieldElementZero, z: FieldElementOne}
)

// These are the three points of order 2 that we can represent with extended twisted coordinates. None of these is in the p253-subgroup, of course.
// Although we do not need or use this, note that SqrtDDivA_fe := sqrt(d/a) == sqrt(2) - 1 due to the way the bandersnatch curve was constructed.
var (
	orderTwoPoint_xtw      Point_xtw = Point_xtw{x: FieldElementZero, y: FieldElementMinusOne, t: FieldElementZero, z: FieldElementOne}
	exceptionalPoint_1_xtw Point_xtw = Point_xtw{x: squareRootDbyA_fe, y: FieldElementZero, t: FieldElementOne, z: FieldElementZero}
	exceptionalPoint_2_xtw Point_xtw = Point_xtw{x: squareRootDbyA_fe, y: FieldElementZero, t: FieldElementMinusOne, z: FieldElementZero}
)

// normalizeAffineZ replaces the internal representation with an equivalent one with Z==1, unless the point is at infinity (in which case we panic).
// This is used to convert to or output affine coordinates.
func (p *Point_xtw) normalizeAffineZ() {
	if p.IsNaP() {
		napEncountered("Try to converting invalid point xtw to coos with z==1", false, p)
		// If the above did not panic, we replace the NaP p by an default NaP with x==y==t==z==0.
		*p = Point_xtw{z: FieldElementOne} // invalid point
		return
	}

	// We reasonably likely call normalizeAffineZ several times in a row on the same point. If Z==1 to start with, do nothing.
	if p.z.IsOne() {
		return
	}

	var temp FieldElement
	if p.z.IsZero() {
		if p.IsNaP() {
			napEncountered("Try to converting invalid point xtw to coos with z==1", false, p)
			// If the above did not panic, we replace the NaP p by an default NaP with x==y==t==z==0.
			*p = Point_xtw{z: FieldElementOne} // invalid point
			return
		}
		panic("Trying to make point at infinity affine")
	}
	temp.Inv(&p.z)
	p.x.MulEq(&temp)
	p.y.MulEq(&temp)
	p.t.MulEq(&temp)
	p.z.SetOne()
}

func (p *Point_xtw) normalizeSubgroup() {
	panic(0)
	// TODO !
}

// X_affine returns the X coordinate of the given point in affine twisted Edwards coordinates.
func (p *Point_xtw) X_affine() FieldElement {
	panic(0)
	p.normalizeAffineZ()
	return p.x
}

// Y_affine returns the Y coordinate of the given point in affine twisted Edwards coordinates.
func (p *Point_xtw) Y_affine() FieldElement {
	panic(0)
	p.normalizeAffineZ()
	return p.y
}

// T_affine returns the T coordinate (i.e. T=XY) of the given point in affine twisted Edwards coordinates.
func (p *Point_xtw) T_affine() FieldElement {
	panic(0)
	p.normalizeAffineZ()
	return p.t
}

// X_projective returns the X coordinate of the given point P in projective twisted Edwards coordinates.
// Note that calling functions on P other than X_projective(), Y_projective(), Z() might change the representations of P at will,
// so callers must not interleave calling other functions.
func (p *Point_xtw) X_projective() FieldElement {
	panic(0)
	return p.x
}

// Y_projective returns the Y coordinate of the given point P in projective twisted Edwards coordinates.
// Note that calling functions on P other than X_projective(), Y_projective(), Z() might change the representations of P at will,
// so callers must not interleave calling other functions.
func (p *Point_xtw) Y_projective() FieldElement {
	panic(0)
	return p.y
}

// Z_projective returns the Z coordinate of the given point P in projective twisted Edwards coordinates.
// Note that calling functions on P other than X_projective(), Y_projective(), Z() might change the representations of P at will,
// so callers must not interleave calling other functions.
func (p *Point_xtw) Z_projective() FieldElement {
	panic(0)
	return p.z
}

// T_projective returns the T coordinate of the given point P in projective twisted Edwards coordinates (i.e. T = XY/Z).
// Note that calling functions on P other than X_projective(), Y_projective(), Z() might change the representations of P at will,
// so callers must not interleave calling other functions.
func (p *Point_xtw) T_projective() FieldElement {
	panic(0)
	return p.t
}

// TODO
/*
// SerializeLong serialize the given point in long serialization format. err==nil iff everything worked OK.
func (p *Point_xtw) SerializeLong(output io.Writer) (bytes_written int, err error) {
	return default_SerializeLong(p, output)
}

// SerializeShort serialize the given point in short serialization format. err==nil iff everything worked OK.
func (p *Point_xtw) SerializeShort(output io.Writer) (bytes_written int, err error) {
	return default_SerializeShort(p, output)
}
*/

// TODO !

/*
// DeserializeShort deserialize from the given input byte stream (expecting it to start with a point in short serialization format) and store the result in the receiver.
// err==nil iff no error occured. trusted should be one of the constants TrustedInput or UntrustedInput.
// For UntrustedInput, we perform a specially-tailored efficient curve and subgroup membership tests.
// Note that long format is considerably more efficient to deserialize.
func (p *Point_xtw) DeserializeShort(input io.Reader, trusted IsPointTrusted) (bytes_read int, err error) {
	return default_DeserializeShort(p, input, trusted)
}

// DeserializeLong deserialize from the given input byte stream (expecting it to start with a point in long serialization format) and store the result in the receiver.
// err==nil iff no error occured. trusted should be one of the constants TrustedInput or UntrustedInput.
// For UntrustedInput, we perform a specially-tailored efficient curve and subgroup membership tests.
// Note that long format is considerably more efficient to deserialize.
func (p *Point_xtw) DeserializeLong(input io.Reader, trusted IsPointTrusted) (bytes_read int, err error) {
	return default_DeserializeLong(p, input, trusted)
}

// DeserializeAuto deserialize from the given input byte stream (expecting it to start with a point in either short or long serialization format -- it autodetects that) and store the result in the receiver.
// err==nil iff no error occured. trusted should be one of the constants TrustedInput or UntrustedInput.
// For UntrustedInput, we perform a specially-tailored efficient curve and subgroup membership tests.
// Note that long format is considerably more efficient to deserialize.
func (p *Point_xtw) DeserializeAuto(input io.Reader, trusted IsPointTrusted) (bytes_read int, err error) {
	return default_DeserializeAuto(p, input, trusted)
}

*/

// String prints the point in X:Y:T:Z - format
func (p *Point_xtw) String() string {
	// Not the most efficient way to concatenate strings, but good enough.
	// TODO: Normalize?
	return p.x.String() + ":" + p.y.String() + ":" + p.t.String() + ":" + p.z.String() + " modulo A"
}

// AffineExtended returns a copy of the point in affine extended coordinates.
func (p *Point_xtw) AffineExtended() Point_axtw {
	p.normalizeAffineZ()
	return Point_axtw{x: p.x, y: p.y, t: p.t}
}

func (p *Point_xtw) ToDecaf_xtw() Point_xtw {
	return *p
}

func (p *Point_xtw) ToDecaf_axtw() Point_axtw {
	p.normalizeAffineZ()
	return Point_axtw{x: p.x, y: p.y, t: p.t}
}

// TODO !

// ExtendedTwistedEdwards() returns a copy of the given point in extended twited Edwards coordinates.
func (p *Point_xtw) ExtendedTwistedEdwards() Point_xtw {
	return *p // Note that Go forces the caller to make a copy.
}

// IsNeutralElement checks if the point P is the neutral element of the curve (modulo the identification of P with P+A).
// Use IsNeutralElement_FullCurve if you do not want this identification.
func (P *Point_xtw) IsNeutralElement() bool {

	// NOTE: This asserts that P is in the correct subgroup or that we work modulo the affine order-2 point A (x=0, y=-c, t=0, z=c).
	if P.x.IsZero() {
		if P.y.IsZero() {
			// Handle error: Singular point
			return napEncountered("compared invalid xtw point to zero", true, P)
		}
		return true
	}
	return false
}

// Clone creates a copy of the point of the same type and returns it (Note that the returned value has interface type and containing a value of type *Point_xtw)
func (p *Point_xtw) Clone() interface{} {
	p_copy := *p
	return &p_copy
}

/*
// IsNeutralElement_FullCurve tests for zero-ness. It does *NOT* identify P with P+A and works for points outside the subgroup.
// We only assume that x,y,t,z satisfy the curve equations.
func (p *Point_xtw) IsNeutralElement_FullCurve() bool {
	if !p.x.IsZero() {
		return false
	}
	if p.IsNaP() {
		return napEncountered("compared invalid xtw point to zero exactly", true, p)
	}
	if !p.t.IsZero() {
		panic("Non-NaP xtw point with x==0, but t!=0 encountered.")
	}
	// we know x==0, y!=0 (because otherwise, we have a NaP), t==0.
	// This implies z == +/- y
	return p.y.IsEqual(&p.z)
}
*/

// SetNeutral sets the Point P to the neutral element of the curve.
func (p *Point_xtw) SetNeutral() {
	*p = NeutralElement_xtw
}

// IsSingular checks whether the point is singular (x==y==0, indeed most likely x==y==t==z==0). Singular points must never appear if the library is used correctly. They can appear by
// a) performing operations on points that are not in the correct subgroup
// b) zero-initialized points are singular (Go lacks constructors to fix that).
// The reason why we check x==y==0 and do not check t,z is due to what happens if we perform mixed additions.
func (p *Point_xtw) IsNaP() bool {
	return p.x.IsZero() && p.y.IsZero()
}

// z.Add(x,y) computes z = x+y according to the elliptic curve group law.
func (p *Point_xtw) Add(x CurvePointPtrInterfaceRead, y CurvePointPtrInterfaceRead) {
	switch x := x.(type) {
	case *Point_xtw:
		switch y := y.(type) {
		case *Point_xtw:
			p.add_ttt(x, y)
		case *Point_axtw:
			p.add_tta(x, y)
		default:
			var y_converted Point_xtw = convertToPoint_xtw(y)
			p.add_ttt(x, &y_converted)
		}
	case *Point_axtw:
		switch y := y.(type) {
		case *Point_xtw:
			p.add_tta(y, x)
		case *Point_axtw:
			p.add_taa(x, y)
		default:
			var y_converted Point_xtw = convertToPoint_xtw(y)
			p.add_tta(&y_converted, x)

		}
	default: // for x
		var x_converted Point_xtw = convertToPoint_xtw(x)

		switch y := y.(type) {
		case *Point_xtw:
			p.add_ttt(&x_converted, y)
		case *Point_axtw:
			p.add_tta(&x_converted, y)
		default:
			var y_converted Point_xtw = convertToPoint_xtw(y)
			p.add_ttt(&x_converted, &y_converted)
		}
	}
}

// z.Sub(x,y) computes z = x-y according to the elliptic curve group law.
func (p *Point_xtw) Sub(x CurvePointPtrInterfaceRead, y CurvePointPtrInterfaceRead) {
	switch x := x.(type) {
	case *Point_xtw: // for x
		switch y := y.(type) {
		case *Point_xtw:
			p.sub_ttt(x, y)
		case *Point_axtw:
			p.sub_tta(x, y)
		default:
			var y_converted Point_xtw = convertToPoint_xtw(y)
			p.sub_ttt(x, &y_converted)
		}
	case *Point_axtw: // for x
		switch y := y.(type) {
		case *Point_xtw:
			p.sub_tat(x, y)
		case *Point_axtw:
			p.sub_taa(x, y)
		default:
			var y_converted Point_xtw = convertToPoint_xtw(y)
			p.sub_tat(x, &y_converted)
		}
	default: // for x
		var x_converted Point_xtw = convertToPoint_xtw(x)
		switch y := y.(type) {
		case *Point_xtw:
			p.sub_ttt(&x_converted, y)
		case *Point_axtw:
			p.sub_tta(&x_converted, y)
		default:
			var y_converted Point_xtw = convertToPoint_xtw(y)
			p.sub_ttt(&x_converted, &y_converted)
		}
	}
}

// z.Double(x) computes z = x+x according to the elliptic curve group law.
func (p *Point_xtw) Double(input CurvePointPtrInterfaceRead) {
	switch input := input.(type) {
	case *Point_xtw:
		p.double_tt(input)
	case *Point_axtw:
		p.double_ta(input)
	default:
		// TODO !
		default_Double(p, input)
	}
}

// z.Sub(x,y) computes z = x-y according to the elliptic curve group law.
func (p *Point_xtw) Neg(input CurvePointPtrInterfaceRead) {
	switch input := input.(type) {
	case *Point_xtw:
		p.neg_tt(input)
	case *Point_axtw:
		p.neg_ta(input)
	default:
		*p = convertToPoint_xtw(input)
		p.NegEq()
	}
}

// z.Endo(x) compute z = \Psi(x) where \Psi is the non-trivial degree-2 endomorphism described in the bandersnatch paper.
func (p *Point_xtw) Endo(input CurvePointPtrInterfaceRead) {
	switch input := input.(type) {
	case *Point_xtw:
		p.computeEndomorphism_tt(input)
	case *Point_axtw:
		p.computeEndomorphism_ta(input)
	case *Point_efgh:
		p.computeEndomorphism_ts(input)
	default:
		p_converted := convertToPoint_xtw(p)
		p.computeEndomorphism_tt(&p_converted)
	}
}

/*
// Endo_FullCurve computes the efficient order-2 endomorphism on the given input point (of any coordinate format).
// This function works even if the input may be a point at infinity; note that the output is never at infinity anyway.
// Be aware that the statement that the endomorpism acts by multiplication by the constant sqrt(2) mod p253 is only meaningful/true on the p253 subgroup.
func (output *Point_xtw) Endo_FullCurve(input CurvePointPtrInterfaceRead_FullCurve) {
	if input.IsNaP() {
		_ = napEncountered("Computing endomorphism on invalid point", false, input)
		*output = Point_xtw{} // NaN-like behaviour.
	} else if input.IsAtInfinity() {
		*output = orderTwoPoint_xtw
	} else {
		output.Endo(input)
	}
}
*/

func (p *Point_xtw) IsAtInfinity() bool {
	if p.IsNaP() {
		return napEncountered("checking whether NaP point is at infinity", false, p)
	}
	return false

	// TODO !
	panic(0)
	if p.IsNaP() {
		return napEncountered("checking whether NaP point is at infinity", false, p)
	}
	if p.z.IsZero() {
		// The only valid points (albeit not in subgroup) with z == 0 are the two exceptional points with z==y==0
		// We catch x==y==0 above (which already means the user of the library screwed up).
		// Anything else means we screwed up even worse.
		if p.t.IsZero() {
			panic("xtw point with z==t==0 encountered, but (x,y) != (0,0), so this was not NaP. This must never happen.")
		}
		if p.x.IsZero() {
			panic("xtw point with z==0 and x==0 encountered, but y!=0, so this was not NaP. This must never happen.")
		}
		return true
	}
	return false
}

// IsEqual compares two curve points for equality, working modulo the P = P + A identification. The two points do not have the be in the same coordinate format.
func (p *Point_xtw) IsEqual(other CurvePointPtrInterfaceRead) bool {
	switch other := other.(type) {
	case *Point_xtw:
		return p.is_equal_tt(other)
	case *Point_axtw:
		return p.is_equal_ta(other)
	case *Point_efgh:
		other_converted := convertToPoint_xtw(other)
		return p.IsEqual(&other_converted)
	default:
		// TODO !
		panic(0)
		/*
			if p.IsNaP() || other.IsNaP() {
				return napEncountered("point was invalid when comparing points for equality", true, p, other)
			}
			var temp1, temp2 FieldElement
			var temp_fe FieldElement = other_real.Y_projective()
			temp1.Mul(&p.x, &temp_fe)
			temp_fe = other_real.X_projective()
			temp2.Mul(&p.y, &temp_fe)
			return temp1.IsEqual(&temp2)
		*/
	}
}

/*
// IsEqual_FullCurve compares two curve points for equality WITHOUT working modulo the P = P+A identification. The two points do not have to be in the same coordinate format.
// This also works for points outside the subgroup or even at infinity.
func (p *Point_xtw) IsEqual_FullCurve(other CurvePointPtrInterfaceRead_FullCurve) bool {
	if p.IsNaP() || other.IsNaP() {
		return napEncountered("point was invalid when comparing points for equality", true, p, other)
	}
	switch other_real := other.(type) {
	case *Point_xtw:
		return p.is_equal_exact_tt(other_real)
	case *Point_axtw:
		return p.is_equal_exact_ta(other_real)
	default:
		other_temp := other.ExtendedTwistedEdwards()
		return p.is_equal_exact_tt(&other_temp)
	}
}
*/

// EndoEq applies the endomorphism on the given point. p.EndoEq() is shorthand for p.Endo(&p).
func (p *Point_xtw) EndoEq() {
	p.computeEndomorphism_tt(p)
}

// AddEq adds (via the elliptic curve group addition law) the given curve point x (in any coordinate format) to the received p, overwriting p.
func (p *Point_xtw) AddEq(x CurvePointPtrInterfaceRead) {
	p.Add(p, x)
}

// SubEq subtracts (via the elliptic curve group addition law) the given curve point x (in any coordinate format) from the received p, overwriting p.
func (p *Point_xtw) SubEq(x CurvePointPtrInterfaceRead) {
	p.Sub(p, x)
}

// DoubleEq doubles the received point p, overwriting p.
func (p *Point_xtw) DoubleEq() {
	p.double_tt(p)
}

// NeqEq replaces the given point by its negative (wrt the elliptic curve group addition law)
func (p *Point_xtw) NegEq() {
	p.x.NegEq()
	p.t.NegEq()
}

// TODO !

// SetFrom initializes the point from the given input point (which may have a different coordinate format)
func (p *Point_xtw) SetFrom(input CurvePointPtrInterfaceRead) {
	switch input := input.(type) {
	case *Point_xtw:
		*p = *input
	case *Point_axtw:
		p.x = input.x
		p.y = input.y
		p.t = input.t
		p.z.SetOne()
	case *Point_efgh:
		*p = input.ToDecaf_xtw()
	default:
		// TODO !
		*p = convertToPoint_xtw(input)
		return
		panic(0)
		p.x = input.X_projective()
		p.y = input.Y_projective()
		p.z = input.Z_projective()
		p.t.Mul(&p.x, &p.y)
		p.x.MulEq(&p.z)
		p.y.MulEq(&p.z)
		p.z.SquareEq()
	}
}
