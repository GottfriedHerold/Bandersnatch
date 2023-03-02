package curvePoints

import (
	"fmt"

	"github.com/GottfriedHerold/Bandersnatch/bandersnatch/bandersnatchErrors"
	"github.com/GottfriedHerold/Bandersnatch/bandersnatch/errorsWithData"
)

// This file contains routines that map curve points to field elements and allow reconstructing them from field elements.
// This should be seen as part of serialization to byte streams: usually, we go curve points <-> field elements (+ some extra bits) <-> byte streams
//
// Note that the specific errors that functions in this file may output (wrapped) are not defined here, but
// in the bandersnatchErrors package. The reason is that some of these errors are shared between the curvePoints and the field element packages.
// panic error message are defined locally, however.

// Bandersnatch-specific MapToFieldElement. We prefer this over more common choices such as X/Z or Y/Z, because
// a) it maps the neutral element to 0.
// b) preimages are a coset wrt a subgroup of the curve; in particular, it is injective on the prime-order subgroup.
// c) it can be computed working modulo A.

// ErrorPrefix_CurveFieldElementSerializers is the prefix used by all
// panics or wrapping error messages orgininating from the curve point <-> field element transition components.
const ErrorPrefix_CurveFieldElementSerializers = "bandersnatch / curve point <-> field elements: "

// MapToFieldElement maps a CurvePoint to a FieldElement as X/Y.
// Note that for the prime-order subgroup, Y is never 0 and this function is actually injective.
// This function panics for points at infinity or NaPs.
//
// In general, preimages of MapToFieldElement have the form {P, P+A} with A the affine two-torsion point.
func MapToFieldElement(input CurvePointPtrInterfaceRead) (ret FieldElement) {
	if input.IsAtInfinity() {
		panic(ErrorPrefix_CurveFieldElementSerializers + "Called MapToFieldElement on point at infinity")
	}
	// Note: IsAtInfinity should already have detected NaPs.
	// Still, if the NaP-handler ignores it, we prefer to panic right now with a more meaningul error message rather than divide by zero later.
	if input.IsNaP() {
		panic(ErrorPrefix_CurveFieldElementSerializers + "Called MapToFieldElement on a NaP")
	}
	ret = input.Y_decaf_projective()
	ret.InvEq()
	temp := input.X_decaf_projective()
	ret.MulEq(&temp)
	return
}

// Note: We do not guarantee consistent return values because the modular square root algorithms might be randomized.
// An optimized implementation for hardwired field size probably is not, but a generic one for field size mod 8 = 1 is reasonably likely randomized.
// We do not wish to depend on particularities of the base field implementation.

// recoverYFromXAffine computes y from x such that (x,y) is a point on the Bandersnatch curve in affine twisted Edwards coordinates.
// Note that the result only depends on x up to sign.
// For valid input x, for which some y exists in the first place, there are always exactly two possible y which differ by sign. (Note y!=0 for affine points)
// recoverYFromXAffine makes no guarantees about the choice of y. It need not even be consistent for multiple calls with the same x.
// If legendreCheckX is set to true, we also check that the resulting (+/-x,+/-y) is on the subgroup for some of the four possible choices of signs.
// (This is equivalent to running legendreCheckA_affineX, but we reuse some computation)
// Returns err==nil if no error occurred (meaning that some y existed and the subgroup check, if requested, did not fail).
//
// Possible errors are (possibly errors wrapping) ErrXNotOnCurve and ErrXNotInSubgroup.
// Note that for ErrXNotInSubgroup, the returned y is still meaningful.
func recoverYFromXAffine(x *FieldElement, legendreCheckX bool) (y FieldElement, err errorsWithData.ErrorWithData[struct{ X FieldElement }]) {

	// We have y^2 = (1-ax^2) / (1-dx^2)
	// So, we first compute (1-ax^2) / 1-dx^2
	var num, denom FieldElement // will hold 1-ax^2 resp. 1-dx^2

	num.Square(x)                        // x^2, only compute this once
	denom.Mul(&num, &CurveParameterD_fe) // dx^2
	num.MulEqFive()                      // 5x^2 = -ax^2
	num.AddEq(&fieldElementOne)          // 1 - ax^2
	denom.Sub(&fieldElementOne, &denom)  // 1 - dx^2
	// Since both a and d are non-squares, we are guaranteed that both num and denom are non-zero.
	// This holds for any x, irrespective of whether x corresponds to a point on the curve.
	// Note that x corresponds to a point in the correct subgroup iff *both* num and denom are squares
	if legendreCheckX {
		if num.Jacobi() < 0 {
			// At this point, we already know that the given x does not correspond to any subgroup point.
			// In the interest of better error message, we check whether it actually correspond to any point on the curve.
			// While computationally expensive, we do not expect this branch to be taken often.

			// While we are at it, we might as well compute y. -- documentation of function changed to reflect that.

			/*  The following is correct, but does not compute y.


			if denom.Jacobi() > 0 {
				err = bandersnatchErrors.ErrXNotOnCurve
			} else {
				err = bandersnatchErrors.ErrXNotInSubgroup

			}
			return
			*/

			// The type of error depends on whether denom is a square or not.
			num.DivideEq(&denom)
			if y.SquareRoot(&num) {
				err = errorsWithData.NewErrorWithData_params[struct{ X FieldElement }](bandersnatchErrors.ErrXNotInSubgroup, "%w. The received X coordinates was %v{X}", "X", *x)

			} else {
				err = errorsWithData.NewErrorWithData_params[struct{ X FieldElement }](bandersnatchErrors.ErrXNotOnCurve, "%w. The received X coordinate was %v{X}", "X", *x)
				y.SetZero() // not needed, but we like being explicit.
			}
			return

		}
	}
	num.DivideEq(&denom) // (1-ax^2)/(1-dx^2). Note that 1-dx^2 cannot be 0, as d is a non-square.
	if !y.SquareRoot(&num) {
		err = errorsWithData.NewErrorWithData_params[struct{ X FieldElement }](bandersnatchErrors.ErrXNotOnCurve, "%w. The received X coordinate was %v{X}", "X", *x)
		return
	}
	err = nil // err is nil at this point anyway, but we prefer to be explicit.
	return
}

// recoverXFromYAffine obtains an x coordinate from an y coordinate, s.t. (x,y) are a valid affine rational point.
// If no y exists, returns (0, ErrYNotOnCurve). Note that we generally have two choices for x, since (-x,y) is also on the curve if (x,y) is.
// We make no guarantees about which x we return; it need not even be consistent for multiple calls with the same y.
//
// Possible errors: ErrYNotOnCurve
func recoverXFromYAffine(y *FieldElement) (x FieldElement, err errorsWithData.ErrorWithData[struct{ Y FieldElement }]) {
	// We solve ax^2 + y^2 = 1+dx^2y^2 for x,
	// which means x^2 = (y^2-1)/(dy^2 - a)
	var num, denom FieldElement
	num.Square(y)                        // y^2, only compute once
	denom.Mul(&num, &CurveParameterD_fe) // dy^2
	num.SubEq(&fieldElementOne)          // y^2 - 1
	denom.SubEq(&CurveParameterA_fe)     // dy^2 - a
	if denom.IsZero() {
		// Note: This case really corresponds to the points at infinity. We give a more specific error.
		err = errorsWithData.NewErrorWithData_struct(bandersnatchErrors.ErrYNotOnCurve, "%w. The affine Y/Z coordinate received was %v{Y}. Note that this particular value actually corresponds to a point at infinity, if one interprets Y/Z = 0/0 appropriately.", &struct{ Y FieldElement }{Y: *y})
		x.SetZero() // x = 0, just to be explicit. (it is zero-initialized to this value anyway)
		return
	}
	num.DivideEq(&denom) // (y^2 - 1) / (dy^2 - a)
	ok := x.SquareRoot(&num)
	if !ok {
		x.SetZero() // We prefer to have some consistent return value for x on error. It must not be used anyway.
		err = errorsWithData.NewErrorWithData_struct(bandersnatchErrors.ErrYNotOnCurve, "%w. The received affine Y/Z coordinate was %v{Y}", &struct{ Y FieldElement }{Y: *y})
	}
	return
}

// These are "Deserialization"-helper routines that do not take an io.Reader as input, but rather Field Elements.
// We also export these to the user

// TODO / QUESTION: Concrete Point type as return type or Interface?

// NOTE: We return a NaP when we detect an error (in addition to returning an actually informative error)
// This is done as footgun-protection: If the user makes the mistake of not checking the return value,
// we at least have the chance that our NaP guards will catch it when the point is used later.
// We do not announce this as part of the interface, because
// a) We do not want to guarantee this behaviour
// b) Not checking the err value is a serious bug on the user's side. We do not want to be even close to encouraging it.
// c) This is not reliable: Not everything inside the library checks for NaPs (in particular, coordinate functions don't)

// CurvePointFromXYAffine_full constructs a curve point with the given affine x and y coordinates. trustLevel should be one of
// TrustedInput or UntrustedInput.
//
// It returns an error if the provided x and y coordinates are invalid. In this case, the returned point must not be used.
// If trustLevel is TrustedInput, you *MUST* call this only with valid x and y coordinates; the library has the liberty to skip checks.
// The library makes no guarantees whatsoever about what happens if you violate this.
//
// Note that it is impossible to construct a point at infinity with this function.
// In the (likely!) case that you want to ensure that the constructed point is on the prime-order subgroup, use CurvePointFromXYAffine_subgroup instead.
//
// Possible error values are (possibly errors wrapping) ErrNotOnCurve and ErrCannotDeserializeXYAllZero, ErrCannotDeserializeNaP
// Note that ErrCannotDeserializeXYAllZero wraps ErrCannotDeserializeNaP.
func CurvePointFromXYAffine_full(x *FieldElement, y *FieldElement, trustLevel IsInputTrusted) (point Point_axtw_full, err errorsWithData.ErrorWithData[struct{ X, Y FieldElement }]) {
	point.x = *x
	point.y = *y
	point.t.Mul(x, y)
	if !trustLevel.Bool() {
		// We do not use point.Validate, because this would not distinguish error reasons.
		// We explicitly check the special case X==Y==0 to give a more informative error message.
		// Note that isPointOnCurve would also catch these cases.
		if point.IsNaP() {
			point = Point_axtw_full{} // standard NaP
			if x.IsZero() && y.IsZero() {
				err = errorsWithData.NewErrorWithData_struct(bandersnatchErrors.ErrCannotDeserializeXYAllZero, "%w", &struct{ X, Y FieldElement }{X: *x, Y: *y})
			} else {
				err = errorsWithData.NewErrorWithData_struct(bandersnatchErrors.ErrCannotDeserializeNaP, "%w. Received affine coordinates X=%v{X}, Y=%v{Y}.", &struct{ X, Y FieldElement }{X: *x, Y: *y})
			}
			return

		}
		if !point.isPointOnCurve() {
			err = errorsWithData.NewErrorWithData_struct(bandersnatchErrors.ErrNotOnCurve, "%w. Received affine coordinates X=%v{X}, Y=%v{Y}", &struct{ X, Y FieldElement }{X: *x, Y: *y})
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
//
// Possible error values are (possibly errors wrapping)
// ErrNotOnCurve, ErrCannotDeserializeXYAllZero, ErrCannotDeserializeNaP, ErrNotInSubgroup
//
// Note that ErrCannotDeserializeXYAllZero wraps ErrCannotDeserializeNaP.
func CurvePointFromXYAffine_subgroup(x *FieldElement, y *FieldElement, trustLevel IsInputTrusted) (point Point_axtw_subgroup, err errorsWithData.ErrorWithData[struct{ X, Y FieldElement }]) {
	point_full, err := CurvePointFromXYAffine_full(x, y, trustLevel)
	if err != nil {
		return
	}
	if !point.SetFromSubgroupPoint(&point_full, trustLevel) {
		err = errorsWithData.NewErrorWithData_struct(bandersnatchErrors.ErrNotInSubgroup, "%w. Affine coordinatate are X=%v{X}, Y=%v{Y}", &struct{ X, Y FieldElement }{X: *x, Y: *y})
	}
	return
}

// NOTE: For the current implementation of FullCurvePointFromXAndSigny, trustLevel actually does not influence whether we perform checks.
// We always check if the x coordinate corresponds to a curve point.
// However, for trustedInput, we panic on failure rather than return an error.

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
//
// Possible errors are (errors possibly wrapping)
//
// bandersnatchErrors.ErrInvalidSign, ErrXNotOnCurve, ErrXNotInSubgroup,
func CurvePointFromXAndSignY_full(x *FieldElement, signY int, trustLevel IsInputTrusted) (point Point_axtw_full, err errorsWithData.ErrorWithData[struct {
	X     FieldElement
	SignY int
}]) {

	// Note that this is (and has to be) a type alias to an anonymous struct type, not a type definition.
	type retData = struct {
		X     FieldElement
		SignY int
	}

	// Unsure if we shouldn't outright panic if sign is not +/- 1

	// Note that we generally define Sign(0) == 0, so signs usually are from -1,0,+1. Since Y cannot be zero, we only
	// accept signs from +/- 1 in this function.
	if signY != 1 && signY != -1 {
		err = errorsWithData.NewErrorWithData_struct(bandersnatchErrors.ErrInvalidSign,
			ErrorPrefix_CurveFieldElementSerializers+"CurvePointFromXAndSignY_full expects the sign argument be either -1 or +1. Got %v{signY} instead. The X coordinate given was %v{X}",
			&retData{X: *x, SignY: signY},
		)

		// If the caller claimed the input was trusted and we detected it's invalid, we panic.
		if trustLevel.Bool() {
			panic(err)
		}
		return
	}

	// Use recoverYFromXAffine to get some y coordinate
	point.x = *x
	var errWithX errorsWithData.ErrorWithData[struct{ X FieldElement }] // returned error has wrong type
	point.y, errWithX = recoverYFromXAffine(x, false)

	if errWithX != nil {
		err = errorsWithData.AddDataToError_params[retData](errWithX, "%w", "SignY", signY)
		// On trusted input, we panic on error.
		if trustLevel.Bool() {
			panic(fmt.Errorf(ErrorPrefix_CurveFieldElementSerializers+"CurvePointFromXAndSignY_full encountered error on trusted input. Error was %w", err))
		}
		point = Point_axtw_full{}
		return
	}
	// Note: recoverYFromXAffine fails if x did not correspond to a point on the curve.
	// This is currently the case irrespectively of trustLevel. In particular, y!=0 is guaranteed.

	// Adjust y for sign.
	if point.y.Sign() != signY {
		point.y.NegEq()
	}
	point.t.Mul(&point.x, &point.y)
	return
}

// CurvePointFromXAndSignY constructs an elliptic curve point on the prime-order subgroup from the given (affine) x coordinate and the sign (+1 or -1) of the y coordinate.
// trustLevel should be one of TrustedInput or UntrustedInput.
//
// It returns an error if the provided x coordinate or sign is invalid (this includes points not on the subgroup). In this case, the returned point must not be used.
// If trustLevel is TrustedInput, you *MUST* call this only with valid inputs; we are free to skip some tests.
// The library makes no guarantees whatsoever about what happens if you violate this.
//
// Possible errors returned are errors possibly wrapping
// ErrInvalidSign, ErrXNotOnCurve, ErrXNotInSubgroup, ErrNotInSubgroup
func CurvePointFromXAndSignY_subgroup(x *FieldElement, signY int, trustLevel IsInputTrusted) (point Point_axtw_subgroup, err errorsWithData.ErrorWithData[struct {
	X     FieldElement
	SignY int
}]) {
	// Note that this is (and has to be) a type alias to an anonymous struct type, not a type definition.
	type retData = struct {
		X     FieldElement
		SignY int
	}

	// Unsure if we shouldn't outright panic if sign is not +/- 1

	// Note that we generally define Sign(0) == 0, so signs usually are from -1,0,+1. Since Y cannot be zero, we only
	// accept signs from +/- 1 in this function.
	if signY != 1 && signY != -1 {
		err = errorsWithData.NewErrorWithData_struct(bandersnatchErrors.ErrInvalidSign,
			ErrorPrefix_CurveFieldElementSerializers+"CurvePointFromXAndSignY_subgroup expects the sign argument be either -1 or +1. Got %v{signY} instead. The X coordinate given was %v{X}",
			&retData{X: *x, SignY: signY},
		)

		// If the caller claimed the input was trusted and we detected it's invalid, we panic.
		if trustLevel.Bool() {
			panic(err)
		}
		return

	}

	// We just call the _full variant and convert.
	// Note that we cannot really do (much) better apart from sharing computations between subgroup check and computing y from x, but that gets messy and seems hardly worth it.

	var point_full Point_axtw_full
	point_full, err = CurvePointFromXAndSignY_full(x, signY, trustLevel)

	if err != nil {

		// Note: trustLevel == TrustedInput ought to be impossible, because CurvePointFromXAndSignY_full should have panicked already. Still included for robustness.
		if trustLevel.Bool() {

			panic(fmt.Errorf(ErrorPrefix_CurveFieldElementSerializers+"error encountered upon trusted construction of curve point with SubgroupCurvePointFromXAndSignY. Error was %w", err))
		}
		return
	}
	ok := point.SetFromSubgroupPoint(&point_full, trustLevel)
	if !ok {

		err = errorsWithData.NewErrorWithData_struct(bandersnatchErrors.ErrNotInSubgroup,
			ErrorPrefix_CurveFieldElementSerializers+"Called CurvePointFromXAndSignY_subgroup with inputs that define a point outside the prime-order subgroup. The received X was %v{X} and SignY was %v{SignY}.",
			&retData{X: *x, SignY: signY})
		point = Point_axtw_subgroup{} // This is redundant, actually, but added for clarity

		// It should actually not be possible to trigger this with trustLevel ==trustedInput, even with crafted input.
		// The reason is SetFromSubgroupPoint does not perform checks for trusted input apart from Non-NaP-ness (NaPs cannot happen here).
		// So we always have ok == true for trusted input (even it's wrong).
		// So if the input is not in the subgroup and trustLevel is TrustedInput, we actually DO output garbage.
		// It is entirely the fault of the caller if that happens.
		if trustLevel.Bool() {
			panic(err)
		}

	}
	return
}

// TODO: Special-case Point at infinity? After all, these have a meaningful Y/Z coo.
// (As in: Either give specific error message or allow constructing points of infinity -- the latter means changing the return type, which is annoying)

// CurvePointFromYAndSignX_full constructs an elliptic curve point from the given (affine) y coordinate and the sign (0, +1 or -1) of the x coordinate.
// trustLevel should be one of TrustedInput or UntrustedInput.
//
// x = 0 can only happen for y = +/- 1. In this case, the function accepts any sign from {-1,0,1} as valid for the sign of X.
// Conversely, a zero sign for X is accepted only for y = +/-1
//
// It returns an error if the provided input is invalid. In this case, the returned point must not be used.
// If trustLevel is TrustedInput, you *MUST* call this only with valid input; we are free to skip some tests.
// The library makes no guarantees whatsoever about what happens if you violate this.
//
// Note that it is currently impossible to construct a point at infinity with this function.
// In the likely case that you want to ensure that the constructed point is on the prime-order subgroup, use CurvePointFromYAndSignX_subgroup instead.
//
// Possible errors returned are (errors possibly wrapping)
//
// ErrInvalidZeroSignX, ErrInvalidSign, ErrYNotOnCurve
func CurvePointFromYAndSignX_full(y *FieldElement, signX int, trustLevel IsInputTrusted) (point Point_axtw_full, err errorsWithData.ErrorWithData[struct {
	Y     FieldElement
	SignX int
}]) {

	type errData = struct {
		Y     FieldElement
		SignX int
	}

	// We need to special-case signX == 0. This means X==0, so we only accept Y == +/- 1
	if signX == 0 {
		if ok, sign := y.CmpAbs(&fieldElementOne); ok {
			if sign {
				point = NeutralElement_axtw_full
				return
			} else {
				point = AffineOrderTwoPoint_axtw
				return
			}
		} else {
			point = Point_axtw_full{}
			err = errorsWithData.NewErrorWithData_struct(bandersnatchErrors.ErrInvalidZeroSignX, "%w, Provided Y coordinate was %v{Y}", &errData{Y: *y, SignX: 0})
			if trustLevel.Bool() {
				panic(err)
			}
			return
		}

	}
	if signX != +1 && signX != -1 {
		err = errorsWithData.NewErrorWithData_struct(bandersnatchErrors.ErrInvalidSign,
			ErrorPrefix_CurveFieldElementSerializers+"CurvePointFromYAndSignX_full and CurvePointFromYAndSignX_subgroup expect signX from {-1,0,+1}. Got: %v{SignX}. The provided Y coorindate was %v{Y}",
			&errData{Y: *y, SignX: signX})
		if trustLevel.Bool() {
			panic(err)
		}
		return
	}

	// We are sure that sign is in {-1,0,+1} now

	// Get some valid X coo, if it exists.
	point.y = *y
	var errFromRecoverX error // due to different extension of the error interface type
	point.x, errFromRecoverX = recoverXFromYAffine(y)
	if errFromRecoverX != nil {
		err = errorsWithData.NewErrorWithData_params[errData](errFromRecoverX, ErrorPrefix_CurveFieldElementSerializers+"Error in CurvePointFromYAndSignX_full: %w, signX = %v{SignX}",
			"SignX", signX)
		if trustLevel.Bool() {
			panic(err)
		}
		point = Point_axtw_full{}
		return
	}

	// point is guaranteed to be on the curve (expect that the value of t is not correct yet) if we get here.
	// New now adjust the sign of the x coo according to signX.
	// Note that if signX == 0, we are guaranteed that point.x == 0 (due the checks above)
	// Conversely, if point.x.Sign() == 0, we accept any sign, since as point.x.NegEq() does not do anything.
	if point.x.Sign() != signX {
		point.x.NegEq()
	}

	// Fix value of t. This is done at the end to avoid having to potentially negate it.
	point.t.Mul(&point.x, &point.y)
	return
}

// CurvePointFromYAndSignX_subgroup constructs an elliptic curve point on the prime-order subgroup from the given (affine) y coordinate and the sign (0, +1 or -1) of the x coordinate.
// trustLevel should be one of TrustedInput or UntrustedInput.
//
// x = 0 can only happen for y = +1. In this case, the function accepts any sign from {-1,0,1} as valid for the sign of X.
// Conversely, a zero sign for X is accepted only for y = +1
//
// It returns an error if the provided input is invalid (this included point not on the subgroup). In this case, the returned point must not be used.
// If trustLevel is TrustedInput, you *MUST* call this only with valid input; we are free to skip some tests.
// The library makes no guarantees whatsoever about what happens if you violate this.
//
// Possible errors returned are errors (possibly wrapping)
// ErrInvalidZeroSignX, ErrInvalidSign, ErrYNotOnCurve, ErrNotInSubgroup
func CurvePointFromYAndSignX_subgroup(y *FieldElement, signX int, trustLevel IsInputTrusted) (point Point_axtw_subgroup, err errorsWithData.ErrorWithData[struct {
	Y     FieldElement
	SignX int
}]) {

	// We just run the _full version and convert
	point_full, err := CurvePointFromYAndSignX_full(y, signX, trustLevel)
	if err != nil {
		if trustLevel.Bool() {
			panic(err) // cannot happen, because the _full - variant already panicked. Added for clarity
		}
		point = Point_axtw_subgroup{} // just for clarity. This is the case anyway.
		return
	}
	ok := point.SetFromSubgroupPoint(&point_full, trustLevel)
	if !ok {
		err = errorsWithData.NewErrorWithData_struct(bandersnatchErrors.ErrNotInSubgroup,
			ErrorPrefix_CurveFieldElementSerializers+"Called CurvePointFromYAndSignX_subgroup with parameters that are not in the prime order subgroup. Y was %v{Y}. signX was %v{SignX}.",
			&struct {
				Y     FieldElement
				SignX int
			}{})
		point = Point_axtw_subgroup{} // just for clarity. This is the case anyway.
		if trustLevel.Bool() {
			panic(err)
		}
	}
	return
}

// This does not have a Full variant: The fact that the curve point is on the subgroup is actually required to reconstruct the point!
//
// Note that this function only requires 1 Legendre symbol computation for untrusted input rather than 2.

// CurvePointFromXTimesSignY_subgroup constructs an elliptic curve point on the prime-order subgroup from the product of the X coordinate and the sign (+1 or -1) of the y coordinate.
// trustLevel should be one of TrustedInput or UntrustedInput.
// Note that the information that the point needs to be on the subgroup is neccessary to uniquely determine the point.
//
// It returns an error if the provided input is invalid. In this case, the returned point must not be used.
// If trustLevel is TrustedInput, you *MUST* call this only with valid input; we are free to skip some tests.
// The library makes no guarantees whatsoever about what happens if you violate this.
//
// Possible errors are (errors possibly wrapping)
//
// ErrXNotOnCurve, ErrXNotInSubgroup; the returned error parameter X equals the value of xSignY up to sign -- note that errors only depend on |X|.
func CurvePointFromXTimesSignY_subgroup(xSignY *FieldElement, trustLevel IsInputTrusted) (point Point_axtw_subgroup, err errorsWithData.ErrorWithData[struct{ X FieldElement }]) {
	point.x = *xSignY // this is only correct up to sign, but point.x is only defined up to sign anyway for our Point_axtw_subgroup implementation.

	// Note that recoverYFromXAffine only depends on the square of x, so the sign of xSignY does not matter.
	point.y, err = recoverYFromXAffine(xSignY, !trustLevel.Bool())
	if err != nil {
		// update error message.
		err = errorsWithData.NewErrorWithData_params[struct{ X FieldElement }](err, ErrorPrefix_CurveFieldElementSerializers+"Error in CurvePointFromXTimesSignY_subgroup: %w. Note that this error only depends on the absolute value |X|")
		point = Point_axtw_subgroup{}
		if trustLevel.Bool() {
			panic(err)
		}
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

// CurvePointFromXYTimesSignY_subgroup constructs an elliptic curve point on the prime order subgroup
// from the pair (X*sign(Y), Y*sign(Y)), where X,Y are affine coordinates and the sign is {-1,+1}-valued.
// trustLevel should be one of TrustedInput or UntrustedInput.
// Note that the information that the point needs to be on the subgroup is neccessary to uniquely determine the point.
//
// It returns an error if the provided input is invalid. In this case, the returned point must not be used.
// If trustLevel is TrustedInput, you *MUST* call this only with valid input; we are free to skip some tests.
// The library makes no guarantees whatsoever about what happens if you violate this.
// Possible errors are (errors possibly wrapping)
//
// ErrWrongSignY, ErrNotInSubgroup, ErrNotOnCurve
func CurvePointFromXYTimesSignY_subgroup(xSignY *FieldElement, ySignY *FieldElement, trustlevel IsInputTrusted) (point Point_axtw_subgroup, err errorsWithData.ErrorWithData[struct {
	XSignY, YSignY FieldElement
}]) {

	// Note: The currently implementation does not make any error checks at all for trusted input.

	type errData = struct {
		XSignY, YSignY FieldElement
	}

	if !trustlevel.Bool() {
		// y * Sign(Y) must have Sign > 0. This also check that y!=0
		if ySignY.Sign() <= 0 {
			err = errorsWithData.NewErrorWithData_struct(bandersnatchErrors.ErrWrongSignY, "%w. X*SignY was %v{XSignY}. The offending Y*SignY was %v{YSignY}", &errData{XSignY: *xSignY, YSignY: *ySignY})
			point = Point_axtw_subgroup{} // no-op, but we prefer to be explicit.
			return
		}
	}

	// If Sign(Y) == 1, then the following is obviously correct (provided the input is good);
	// However, if Sign(Y) == -1, this just differs by + A, which we do not care about, since the internal representation of Point_axtw_subgroup works modulo A.
	point.x = *xSignY
	point.y = *ySignY
	point.t.Mul(xSignY, ySignY)

	// We may need to check that the input was correct.
	if !trustlevel.Bool() {

		// We compute 1-ax^2 - y^2 + dt^2, which is 0 iff the point is on the curve (and finite).
		// Observe that the subexpression 1-ax^2 is also used in the subgroup check, so we do that along the way.

		var accumulator, temp FieldElement

		accumulator.Square(xSignY) // x^2

		accumulator.MulEqFive()             // 5x^2 == -ax^2
		accumulator.AddEq(&fieldElementOne) // 1+5x^2 == 1-ax^2

		if accumulator.Jacobi() < 0 {
			err = errorsWithData.NewErrorWithData_struct(bandersnatchErrors.ErrNotInSubgroup, "%w. The received X*SignY and Y*SignY were %v{XSignY} and %v{YSignY} respectively.", &errData{XSignY: *xSignY, YSignY: *ySignY})
			// no return here. We continue computing.
			// This way, if we have both "not on curve" and "not in subgroup", we get "not on curve", which is more informative.
			// We also do not yet set point to a NaP, because we use point.t in the "not on curve" check.
		}

		temp.Square(&point.y)           // y^2
		accumulator.SubEq(&temp)        // 1-ax^2 - y^2
		temp.Square(&point.t)           // t^2 == x^2y^2
		temp.MulEq(&CurveParameterD_fe) // dt^2
		accumulator.AddEq(&temp)        // 1 - ax^2 - y^2 + dt^2
		if !accumulator.IsZero() {
			err = errorsWithData.NewErrorWithData_struct(bandersnatchErrors.ErrNotOnCurve, "%w. The received X*SignY and Y*SignY were %v{XSignY} and %v{YSignY} respectively.", &errData{XSignY: *xSignY, YSignY: *ySignY})
		}
		if err != nil {
			point = Point_axtw_subgroup{}
		}
	}
	return
}

// Y*Sign(Y), X*Sign(Y) in this order feels less natural, but this is the order of serialization.
// The reason is that in the serialization format(s) we want to read the information that distinguishes long from short format early, so we can autodetect that
// while reading, while at the same time having the short format be a substring of the long format.

// CurvePointFromYXTimesSignY_subgroup constructs an elliptic curve point on the prime order subgroup
// from the pair (Y*sign(Y), X*sign(Y)), where (X,Y) are affine coordinates and the sign is {-1,+1}-valued.
// trustLevel should be one of TrustedInput or UntrustedInput.
// Note that the information that the point needs to be on the subgroup is neccessary to uniquely determine the point.
//
// This is identical except for the order of parameters to CurvePointFromXYTimesSignY_subgroup and provided for consistency with the
// actual order of the Banderwagon serializer (which writes in this order)
//
// It returns an error if the provided input is invalid. In this case, the returned point must not be used.
// If trustLevel is TrustedInput, you *MUST* call this only with valid input; we are free to skip some tests.
// The library makes no guarantees whatsoever about what happens if you violate this.
// Possible errors are (errors possibly wrapping)
//
// ErrWrongSignY, ErrNotInSubgroup, ErrNotOnCurve
func CurvePointFromYXTimesSignY_subgroup(ySignY *FieldElement, xSignY *FieldElement, trustLevel IsInputTrusted) (point Point_axtw_subgroup, err errorsWithData.ErrorWithData[struct{ XSignY, YSignY FieldElement }]) {
	return CurvePointFromXYTimesSignY_subgroup(xSignY, ySignY, trustLevel)
}
