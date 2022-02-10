package bandersnatch

import (
	"errors"
	"fmt"
	"io"
)

type CurvePointDeserializer interface {
	Deserialize(outputPoint CurvePointPtrInterfaceWrite, inputStream io.Reader, trustLevel IsPointTrusted) (bytesRead int, err error)
}

type CurvePointSerializer interface {
	CurvePointDeserializer
	Serialize(inputPoint CurvePointPtrInterfaceRead, outputStream io.Writer) (bytesWritten int, err error)
	// SerializeSlice(inputPoints []CurvePointPtrInterfaceRead)
}

var (
	ErrCannotSerializePointAtInfinity = errors.New("serialization: cannot serialize point at infinity")
	ErrCannotSerializeNaP             = errors.New("serialization: cannot serialize NaP")
	ErrCannotDeserializeXYAllZero     = errors.New("deserialization: trying to deserialize a point with coordinates x==y==0")
)

// Note: If X/Z is not on the curve, we might get either a "not on curve" or "not in subgroup" error. Should we clarify the wording to reflect that?
var (
	ErrXNotInSubgroup     = errors.New("deserialization: received affine X coordinate does not correspond to any point in the p253 subgroup of the Bandersnatch curve")
	ErrXNotOnCurve        = errors.New("deserialization: received affine X coordinate does not correspond to any (finite, rational) point of the Bandersnatch curve")
	ErrYNotOnCurve        = errors.New("deserialization: encountered affine Y coordinate that does not correspond to any (finite, rational) point of the Bandersnatch curve")
	ErrNotInSubgroup      = errors.New("deserialization: received affine X and Y coordinates do not correspond to a point in the p253 subgroup of the Bandersnatch curve")
	ErrNotOnCurve         = errors.New("deserialization: received affine X and Y coordinates do not correspond to a point on the Bandersnatch curve")
	ErrWrongSignY         = errors.New("deserialization: encountered affine Y coordinate with unexpected Sign bit")
	ErrUnrecognizedFormat = errors.New("deserialization: could not automatically detect serialization format")
)

// Bandersnatch-specific MapToFieldElement. We prefer this over more common choices such as X/Z or Y/Z, because
// a) it maps the neutral element to 0.
// b) preimages are a coset wrt a subgroup of the curve; in particular, it is injective on the prime-order subgroup.
// c) it can be computed working modulo A.

// MapToFieldElement maps a CurvePoint to a FieldElement as X/Y. Note that for the prime-order subgroup, Y is never 0 and this function is actually injective.
// In general, preimages of MapToFieldElement have the form {P, P+A} with A the affine two-torsion point.
func MapToFieldElement(input CurvePointPtrInterfaceRead) (ret FieldElement) {
	if input.IsAtInfinity() {
		panic("Called MapToFieldElement on point at infinity")
	}
	// Note: IsAtInfinity should already have detected NaPs.
	// Still, if the nap-handler ignores it, we prefer to panic right now with a more meaningul error message rather than divide by zero.
	if input.IsNaP() {
		panic("Called MapToFieldElement on a NaP")
	}
	ret = input.Y_decaf_projective()
	ret.InvEq()
	temp := input.X_decaf_projective()
	ret.MulEq(&temp)
	return
}

// Note: We do not guarantee consistent return values because the modular square root algorithms might be randomized.
// (An optimized implementation for hardwired field size probably is not, but a generic one for field size mod 8 = 1 is reasonably likely randomized)

// recoverYFromXAffine computes y from x such that (x,y) is on the curve. Note that the result only depends on x up to sign.
// For valid input x, for which some y exists in the first place, there are always exactly two possible y which differ by sign. (Note y!=0 for affine points)
// recoverYFromXAffine makes no guarantees about the choice of y. It need not even be consistent for multiple calls with the same x.
// If legendreCheckX is set to true, we also check that the resulting (+/-x,+/-y) is on the subgroup for some choice of signs.
// (This is equivalent to running legendreCheckA_affineX, but we reuse some computation)
// Returns err==nil if no error occurred (meaning that some y existed and the subgroup check, if requested, did not fail).
//
// Possible errors are ErrXNotOnCurve and ErrXNotInSubgroup
func recoverYFromXAffine(x *FieldElement, legendreCheckX bool) (y FieldElement, err error) {

	// We have y^2 = (1-ax^2) / (1-dx^2)
	// So, we first compute (1-ax^2) / 1-dx^2
	var num, denom FieldElement

	num.Square(x)                        // x^2, only compute this once
	denom.Mul(&num, &CurveParameterD_fe) // dx^2
	num.multiply_by_five()               // 5x^2 = -ax^2
	num.AddEq(&FieldElementOne)          // 1 - ax^2
	denom.Sub(&FieldElementOne, &denom)  // 1 - dx^2
	// Note that x is in the correct subgroup iff *both* num and denom are squares
	if legendreCheckX {
		if num.Jacobi() < 0 {
			// This is only done for better error messages.
			// While computationally expensive, we do not expect this branch to be taken often.
			if denom.Jacobi() > 0 {
				err = ErrXNotOnCurve
			} else {
				err = ErrXNotInSubgroup
			}
			return
		}
	}
	num.DivideEq(&denom) // (1-ax^2)/(1-dx^2). Note that 1-dx^2 cannot be 0, as d is a non-square.
	if !y.SquareRoot(&num) {
		err = ErrXNotOnCurve
		return
	}
	err = nil // err is nil at this point anyway, but we prefer to be explicit.
	return
}

// recoverXFromYAffine obtains an x coordinate from an y coordinate, s.t. (x,y) are a valid affine rational point.
// If no y exists, returns ErrYNotOnCurve. Note that we have two choices for x, since (-x,y) is also on the curve if (x,y) is.
// We make no guarantees about which x we return; it need not even be consistent for multiple calls with the same y.
func recoverXFromYAffine(y *FieldElement) (x FieldElement, err error) {
	var num, denom FieldElement
	num.Square(y)                        // y^2, only compute once
	denom.Mul(&num, &CurveParameterD_fe) // dy^2
	num.SubEq(&FieldElementOne)          // y^2 - 1
	denom.SubEq(&CurveParameterA_fe)     // dy^2 - a
	if denom.IsZero() {
		err = ErrYNotOnCurve // Note: This case really corresponds to the points at infinity. We might want a more specific error.
		x.SetZero()
		return
	}
	num.DivideEq(&denom) // (y^2 - 1) / (dy^2 - a)
	ok := x.SquareRoot(&num)
	if !ok {
		x.SetZero()
		err = ErrYNotOnCurve
	}
	return
}

// These are "Deserialization"-helper routines that do not take an io.Reader as input, but rather Field Elements.
// We also export these to the user

// TODO / QUESTION: Concrete Point type as return type or Interface?

// NOTE: We return a NaP when we detect an error (in addition to returning an actually informative error)
// This is done as footgun-protection: If the user makes the mistake of not checking the return value,
// we at least have the chance that our NaP guards will catch it when the point is used.
// We do not announce this as part of the interface, because
// a) We do not want to guantee this behaviour
// b) We do not want to encourage users to not check the err value
// c) This is not reliable: Not everything inside the library checks for NaPs (in particular, coordinate functions don't)

// CurvePointFromXYAffine_full constructs a curve point with the given affine x and y coordinates. trustLevel should be one of
// TrustedInput or UntrustedInput.
//
// It returns an error, if the provided x and y coordinates are invalid. In this case, the returned point must not be used.
// If trustLevel is TrustedInput, you *MUST* call this only with valid x and y coordinates; we are free to skip some tests.
// The library makes no guarantees whatsoever about what happens if you violate this.
//
// Note that it is impossible to construct a point at infinity with this function.
// In the (likely!) case that you want to ensure that the constructed point is on the prime-order subgroup, use CurvePointFromXYAffine_subgroup instead.
//
// Possible error values are ErrNotOnCurve and ErrCannotDeserializeXYAllZero
func CurvePointFromXYAffine_full(x *FieldElement, y *FieldElement, trustLevel IsPointTrusted) (point Point_axtw_full, err error) {
	point.x = *x
	point.y = *y
	point.t.Mul(x, y)
	if !trustLevel.Bool() {
		// We do not use point.Validate, because this would not distinguish error reasons.
		// Explicitly check this special case (X==Y==0) to give a more informative error message.
		// Note that isPointOnCurve would also catch it.
		if point.IsNaP() {
			point = Point_axtw_full{} // standard NaP
			err = ErrCannotDeserializeXYAllZero
			return
		}
		if !point.isPointOnCurve() {
			err = ErrNotOnCurve
			// some extra footgun-protection for users who don't check the error return value (which is a mistake).
			point = Point_axtw_full{}
			return
		}
	}
	return
}

// CurvePointFromXYAffine_subgroup constructs a rational point on the prime-order subgroup of the Bandersnatch curve with the given affine x and y coordinates.
// trustLevel should be one of TrustedInput or UntrustedInput.
//
// It returns an error if the provided x and y coordinates are invalid. In this case, the returned point must not be used.
// If trustLevel is TrustedInput, you *MUST* call this only with valid x and y coordinates that are on the subgroup; we are free to skip some tests.
// The library makes no guarantees whatsoever about what happens if you violate this.
func CurvePointFromXYAffine_subgroup(x *FieldElement, y *FieldElement, trustLevel IsPointTrusted) (point Point_axtw_subgroup, err error) {
	point_full, err := CurvePointFromXYAffine_full(x, y, trustLevel)
	if err != nil {
		return
	}
	if !point.SetFromSubgroupPoint(&point_full, trustLevel) {
		err = ErrNotInSubgroup
	}
	return
}

// NOTE: For the current implementation of FullCurvePointFromXAndSigny, trustLevel actually does not influence whether we perform checks.
// We always check if the x coordinate corresponds to a curve point.
// However, for trustedInput, we panic on failure.

// TODO: Document possible errors?

// CurvePointFromXAndSignY_full constructs an elliptic curve point from the given (affine) x coordinate and the sign (+1 or -1) of the y coordinate.
// trustLevel should be one of TrustedInput or UntrustedInput.
//
// It returns an error if the provided x coordinate is invalid. In this case, the returned point must not be used.
// If trustLevel is TrustedInput, you *MUST* call this only with valid x coordinate; we are free to skip some tests.
// The library makes no guarantees whatsoever about what happens if you violate this.
//
// Note that it is impossible to construct a point at infinity with this function.
// In the (likely!) case that you want to ensure that the constructed point is on the prime-order subgroup, use CurvePointFromXAndSignY_subgroup instead.
func CurvePointFromXAndSignY_full(x *FieldElement, signY int, trustLevel IsPointTrusted) (point Point_axtw_full, err error) {
	signValid := (signY == 1 || signY == -1)
	if !signValid {
		// Unsure if we shouldn't outright panic. This is as likely to be a bug in the calling code as it is malicious input.
		// TODO: write warning to stderr?
		// Q: Consider treating signY == 0 specially (after all, sign(0)==0, so this is reasonably an ErrNotOnCurve error)
		err = fmt.Errorf("bandersnatch: FullCurvePointFromXAndSignY expects the sign argument to be either +1 or -1")
		if trustLevel.Bool() {
			panic(err)
		}
		return
	}
	point.x = *x
	point.y, err = recoverYFromXAffine(x, false)
	if err != nil {
		if trustLevel.Bool() {
			err = fmt.Errorf("bandersnatch: FullCurvePointFromXAndSignY encountered error on trusted input. Error was %w", err)
			panic(err)
		}
		point = Point_axtw_full{}
		return
	}
	// Note: recoverYFromXAffine failed if x did not correspond to a point on the curve.
	// This is done irrespectively of trustLevel.
	if point.y.Sign() != signY {
		point.y.NegEq()
	}
	point.t.Mul(&point.x, &point.y)
	return
}

// FullCurvePointFromXAndSignY constructs an elliptic curve point from the given (affine) x coordinate and the sign (+1 or -1) of the y coordinate.
// trustLevel should be one of TrustedInput or UntrustedInput.
//
// It returns an error if the provided x coordinate is invalid. In this case, the returned point must not be used.
// If trustLevel is TrustedInput, you *MUST* call this only with valid x coordinate; we are free to skip some tests.
// The library makes no guarantees whatsoever about what happens if you violate this.
func SubgroupCurvePointFromXAndSignY(x *FieldElement, signY int, trustLevel IsPointTrusted) (point Point_axtw_subgroup, err error) {
	signValid := (signY == 1 || signY == -1)
	if !signValid {
		// Unsure if we shouldn't outright panic. This is as likely to be a bug in the calling code as it is malicious input.
		// TODO: write warning to stderr?
		// Q: Consider treating signY == 0 specially (after all, sign(0)==0, so this is reasonably an ErrNotOnCurve error)
		err = fmt.Errorf("bandersnatch: SubgroupCurvePointFromXAndSignY expects the sign argument to be either +1 or -1")
		if trustLevel.Bool() {
			panic(err)
		}
		return
	}
	if trustLevel.Bool() {
		// trusted input case:
		var point_full Point_axtw_full
		point_full, err = CurvePointFromXAndSignY_full(x, signY, trustLevel)
		// err==nil, because FullCurvePointFromXAndSignY panics on error for trusted input.
		assert(err == nil, "bandersnatch: error encountered upon trusted construction of curve point with SubgroupCurvePointFromXAndSignY")
		ok := point.SetFromSubgroupPoint(&point_full, trustLevel)
		// It should not be possible to trigger this, even with crafted input, because
		// SetFromSubgroupPoint does not perform checks for trusted input apart from Non-NaP-ness.
		// If the input is not in the subgroup, we actually DO output garbage.
		assert(ok, "bandersnatch: unexpected error during trusted construction of curve point with SubgroupCurvePointFromXAndSignY")
		return
	} else {
		// untrusted input case:
		point.x = *x
		point.y, err = recoverYFromXAffine(x, true)
		if err != nil {
			point = Point_axtw_subgroup{}
			return
		}
		if point.y.Sign() != signY {
			point.y.NegEq()
		}
		if !legendreCheckE1_affineY(point.y) {
			err = ErrNotInSubgroup
			point = Point_axtw_subgroup{}
			return
		}
		point.t.Mul(&point.x, &point.y)
		return
	}
}

var ErrInvalidZeroSignX = errors.New("bandersnatch: When constructing curve point with FullCurvePointFromYAndSignX, signX==0, but curve point did not have x==0")

func FullCurvePointFromYAndSignX(y *FieldElement, signX int, trustLevel IsPointTrusted) (point Point_axtw_full, err error) {
	if signX == 0 {
		if ok, sign := y.CmpAbs(&FieldElementOne); ok {
			if sign {
				point = NeutralElement_axtw_full
				return
			} else {
				point = AffineOrderTwoPoint_axtw
				return
			}
		} else {
			point = Point_axtw_full{}
			err = ErrInvalidZeroSignX
			return
		}
	}
	if !(signX == +1 || signX == -1) {
		panic("bandersnatch: FullCurvePointFromYAndSignX and SubgroupCurvePointFromYSignX expect signX from {-1,0,+1}")
	}
	point.y = *y
	point.x, err = recoverXFromYAffine(y)
	if err != nil {
		if trustLevel.Bool() {
			err = fmt.Errorf("bandersnatch: FullCurvePointFromYAndSignX encountered error on trusted input. Error was %w", err)
			panic(err)
		}
		point = Point_axtw_full{}
		return
	}
	// if recoverXFromYAffine returns err==nil, we are guaranteed that we are on the curve.
	if point.x.Sign() != signX {
		point.x.NegEq()
	}
	point.t.Mul(&point.x, &point.y)
	return
}

func SubgroupCurvePointFromYAndSignX(y *FieldElement, signX int, trustLevel IsPointTrusted) (point Point_axtw_subgroup, err error) {
	point_full, err := FullCurvePointFromYAndSignX(y, signX, trustLevel)
	if err != nil {
		return
	}
	ok := point.SetFromSubgroupPoint(&point_full, trustLevel)
	if !ok {
		err = ErrNotInSubgroup
		point = Point_axtw_subgroup{}
	}
	return
}

// This does not have a Full variant: The fact that the curve point is on the subgroup is actually required to reconstruct the point!
//
// Note that this function only requires 1 Legendre symbol computation for untrusted input rather than 2.

func SubgroupCurvePointFromXTimesSignY(xSignY *FieldElement, trustLevel IsPointTrusted) (point Point_axtw_subgroup, err error) {
	point.x = *xSignY // this is only correct up to sign, but point.x is only defined up to sign anyway.

	// Note that recoverYFromXAffine only depends on the square of x, so the sign of xSignY does not matter.
	point.y, err = recoverYFromXAffine(xSignY, !trustLevel.Bool())
	if err != nil {
		point = Point_axtw_subgroup{}
		return
	}

	// point.x, point.y are now guaranteed to satisfy the curve equation (if we set t := x * y, which we will do later).
	// The +/- ambiguity of both p.x and p.y corresponds to the set of 4 points of the form {P, -P, P+A, -P+A} for the affine 2-torsion point A.
	// Due to working mod A, we just need to fix the sign:
	// if the Sign of y is +1, we are good (having set point.x = *xSignY), otherwise we need to fix it.
	if point.y.Sign() < 0 {
		point.y.NegEq() // point.x.NegEq() would work just as well, giving a point that differs by +A
	}

	// Set t coordinate correctly:
	point.t.Mul(&point.x, &point.y)
	return
}

func SubgroupCurvePointFromXYTimesSignY(xSignY *FieldElement, ySignY *FieldElement, trustlevel IsPointTrusted) (point Point_axtw_subgroup, err error) {
	// If Sign(Y) == 1, then this is obviously correct (provided the input is good);
	// However, if Sign(Y) == -1, this just differs by + A, which we do not care about.
	point.x = *xSignY
	point.y = *ySignY
	point.t.Mul(xSignY, ySignY)
	if !trustlevel.Bool() {
		// y * Sign(Y) must have Sign > 0. This also check that y!=0
		if ySignY.Sign() <= 0 {
			err = ErrWrongSignY
			return
		}

		// We compute 1-ax^2 - y^2 + dt^2, which is 0 iff the point is on the curve (and finite).
		// Observe that the subexpression 1-ax^2 is also used in the subgroup check, so we do that along the way.

		var accumulator, temp FieldElement

		accumulator.Square(xSignY) // x^2

		accumulator.multiply_by_five()      // 5x^2 == -ax^2
		accumulator.AddEq(&FieldElementOne) // 1+5x^2 == 1-ax^2

		if accumulator.Jacobi() < 0 {
			err = ErrNotInSubgroup
			// no return. This way, if we have both "not on curve" and "not in subgroup", we get "not on curve", which is more informative.
			// We also do not set point to a NaP, because we use point.t
		}

		temp.Square(&point.y)           // y^2
		accumulator.SubEq(&temp)        // 1-ax^2 - y^2
		temp.Square(&point.t)           // t^2 == x^2y^2
		temp.MulEq(&CurveParameterD_fe) // dt^2
		accumulator.AddEq(&temp)        // 1 - ax^2 - y^2 + dt^2
		if !accumulator.IsZero() {
			err = ErrNotOnCurve
		}
		if err != nil {
			point = Point_axtw_subgroup{}
		}
	}
	return
}
