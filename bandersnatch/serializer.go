package bandersnatch

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

type CurvePointDeserializer interface {
	Deserialize(outputPoint CurvePointPtrInterfaceWrite, inputStream io.Reader, trustLevel IsPointTrusted) (bytesRead int, err error)
	// DeserializeManyIntoBuffer(outputPoints []CurvePointPtrInterfaceWrite, inputStream io.Reader, trustLevel IsPointTrusted) (bytesRead int, err error, pointsWritten int)
	// DeserializeMany(outputPoints []CurvePointPtrInterfaceWrite, inputStream io.Reader, trustLevel IsPointTrusted, maxPoints int) (readPoints []CurvePointPtrInterface, bytesRead int, err error)
}

type CurvePointSerializer interface {
	CurvePointDeserializer
	Serialize(inputPoint CurvePointPtrInterfaceRead, outputStream io.Writer) (bytesWritten int, err error)
	// SerializeMany(inputPoints interface{}, outputStream io.Writer) (bytesWritten int, err error)
	// SerializeSlice(inputPoints []CurvePointPtrInterfaceRead)
}

var (
	ErrCannotSerializePointAtInfinity = errors.New("bandersnatch / point serialization: cannot serialize point at infinity")
	ErrCannotSerializeNaP             = errors.New("bandersnatch / point serialization: cannot serialize NaP")
	ErrCannotDeserializeXYAllZero     = errors.New("bandersnatch / point deserialization: trying to deserialize a point with coordinates x==y==0")
)

var ErrWillNotSerializePointOutsideSubgroup = errors.New("bandersnatch / point serialization: trying to serialize point outside subgroup while serializer is subgroup-only")

// Note: If X/Z is not on the curve, we might get either a "not on curve" or "not in subgroup" error. Should we clarify the wording to reflect that?

var (
	ErrXNotInSubgroup = errors.New("bandersnatch / point deserialization: received affine X coordinate does not correspond to any point in the p253 subgroup of the Bandersnatch curve")
	ErrXNotOnCurve    = errors.New("bandersnatch / point deserialization: received affine X coordinate does not correspond to any (finite, rational) point of the Bandersnatch curve")
	ErrYNotOnCurve    = errors.New("bandersnatch / point deserialization: encountered affine Y coordinate that does not correspond to any (finite, rational) point of the Bandersnatch curve")
	ErrNotInSubgroup  = errors.New("bandersnatch / point deserialization: received affine X and Y coordinates do not correspond to a point in the p253 subgroup of the Bandersnatch curve")
	ErrNotOnCurve     = errors.New("bandersnatch / point deserialization: received affine X and Y coordinates do not correspond to a point on the Bandersnatch curve")
	ErrWrongSignY     = errors.New("bandersnatch / point deserialization: encountered affine Y coordinate with unexpected Sign bit")
	// ErrUnrecognizedFormat = errors.New("bandersnatch / point deserialization: could not automatically detect serialization format")
)

var ErrInvalidZeroSignX = errors.New("bandersnatch / point deserialization: When constructing curve point from Y and the sign of X, the sign of X was 0, but X==0 is not compatible with the given Y")
var ErrInvalidSign = errors.New("bandersnatch / point deserialization: impossible sign encountered")

// Bandersnatch-specific MapToFieldElement. We prefer this over more common choices such as X/Z or Y/Z, because
// a) it maps the neutral element to 0.
// b) preimages are a coset wrt a subgroup of the curve; in particular, it is injective on the prime-order subgroup.
// c) it can be computed working modulo A.

// MapToFieldElement maps a CurvePoint to a FieldElement as X/Y. Note that for the prime-order subgroup, Y is never 0 and this function is actually injective.
//
// In general, preimages of MapToFieldElement have the form {P, P+A} with A the affine two-torsion point.
func MapToFieldElement(input CurvePointPtrInterfaceRead) (ret FieldElement) {
	if input.IsAtInfinity() {
		panic("Called MapToFieldElement on point at infinity")
	}
	// Note: IsAtInfinity should already have detected NaPs.
	// Still, if the nap-handler ignores it, we prefer to panic right now with a more meaningul error message rather than divide by zero later.
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
// An optimized implementation for hardwired field size probably is not, but a generic one for field size mod 8 = 1 is reasonably likely randomized.
// We do not wish to depend on particularities of the base field implementation.

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
// we at least have the chance that our NaP guards will catch it when the point is used later.
// We do not announce this as part of the interface, because
// a) We do not want to guarantee this behaviour
// b) Not checking the err value is a serious bug on the user's side. We do not want to be even close to encouraging it.
// c) This is not reliable: Not everything inside the library checks for NaPs (in particular, coordinate functions don't)

// CurvePointFromXYAffine_full constructs a curve point with the given affine x and y coordinates. trustLevel should be one of
// TrustedInput or UntrustedInput.
//
// It returns an error, if the provided x and y coordinates are invalid. In this case, the returned point must not be used.
// If trustLevel is TrustedInput, you *MUST* call this only with valid x and y coordinates; the library has the liberty to skip checks.
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
		// We explicitly check the special case X==Y==0 to give a more informative error message.
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
		err = fmt.Errorf("%w. FullCurvePointFromXAndSignY expects the sign argument to be either +1 or -1. Got: %v", ErrInvalidSign, signY)
		if trustLevel.Bool() {
			panic(err)
		}
		return
	}
	point.x = *x
	point.y, err = recoverYFromXAffine(x, false)
	if err != nil {
		if trustLevel.Bool() {
			err = fmt.Errorf("bandersnatch: CurvePointFromXAndSignY_full encountered error on trusted input. Error was %w", err)
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

// CurvePointFromXAndSignY constructs an elliptic curve point on the prime-order subgroup from the given (affine) x coordinate and the sign (+1 or -1) of the y coordinate.
// trustLevel should be one of TrustedInput or UntrustedInput.
//
// It returns an error if the provided x coordinate or sign is invalid (this includes points not on the subgroup). In this case, the returned point must not be used.
// If trustLevel is TrustedInput, you *MUST* call this only with valid inputs; we are free to skip some tests.
// The library makes no guarantees whatsoever about what happens if you violate this.
func CurvePointFromXAndSignY_subgroup(x *FieldElement, signY int, trustLevel IsPointTrusted) (point Point_axtw_subgroup, err error) {
	signValid := (signY == 1 || signY == -1)
	if !signValid {
		// Unsure if we shouldn't outright panic. This is as likely to be a bug in the calling code as it is malicious input.
		// TODO: write warning to stderr?
		// Q: Consider treating signY == 0 specially (after all, sign(0)==0, so this is reasonably an ErrNotOnCurve error)
		err = fmt.Errorf("%w. CurvePointFromXAndSignY_subgroup expects the sign argument to be either +1 or -1. Got %v", ErrInvalidSign, signY)
		if trustLevel.Bool() {
			panic(err)
		}
		return
	}
	if trustLevel.Bool() {
		// trusted input case:
		var point_full Point_axtw_full
		point_full, err = CurvePointFromXAndSignY_full(x, signY, trustLevel)
		// err==nil at this point, because FullCurvePointFromXAndSignY panics on error for trusted input.
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
// Note that it is impossible to construct a point at infinity with this function.
// In the likely case that you want to ensure that the constructed point is on the prime-order subgroup, use CurvePointFromYAndSignX_subgroup instead.
func CurvePointFromYAndSignX_full(y *FieldElement, signX int, trustLevel IsPointTrusted) (point Point_axtw_full, err error) {
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
		err = fmt.Errorf("%w. CurvePointFromYAndSignX_full and CurvePointFromYAndSignX_subgroup expect signX from {-1,0,+1}. Got: %v", ErrInvalidSign, signX)
		return
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

// CurvePointFromYAndSignX_subgroup constructs an elliptic curve point on the prime-order subgroup from the given (affine) y coordinate and the sign (0, +1 or -1) of the x coordinate.
// trustLevel should be one of TrustedInput or UntrustedInput.
//
// x = 0 can only happen for y = +1. In this case, the function accepts any sign from {-1,0,1} as valid for the sign of X.
// Conversely, a zero sign for X is accepted only for y = +1
//
// It returns an error if the provided input is invalid (this included point not on the subgroup). In this case, the returned point must not be used.
// If trustLevel is TrustedInput, you *MUST* call this only with valid input; we are free to skip some tests.
// The library makes no guarantees whatsoever about what happens if you violate this.
func CurvePointFromYAndSignX_subgroup(y *FieldElement, signX int, trustLevel IsPointTrusted) (point Point_axtw_subgroup, err error) {
	point_full, err := CurvePointFromYAndSignX_full(y, signX, trustLevel)
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

// CurvePointFromXTimesSignY_subgroup constructs an elliptic curve point on the prime-order subgroup from the product of the X coordinate and the sign (+1 or -1) of the y coordinate.
// trustLevel should be one of TrustedInput or UntrustedInput.
// Note that the information that the point needs to be on the subgroup is neccessary to uniquely determine the point.
//
// It returns an error if the provided input is invalid. In this case, the returned point must not be used.
// If trustLevel is TrustedInput, you *MUST* call this only with valid input; we are free to skip some tests.
// The library makes no guarantees whatsoever about what happens if you violate this.
func CurvePointFromXTimesSignY_subgroup(xSignY *FieldElement, trustLevel IsPointTrusted) (point Point_axtw_subgroup, err error) {
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

// CurvePointFromXYTimesSignY_subgroup constructs an elliptic curve point on the prime order subgroup
// from the pair (X*sign(Y), Y*sign(Y)), where X,Y are affine coordinates and the sign is {-1,+1}-valued.
// trustLevel should be one of TrustedInput or UntrustedInput.
// Note that the information that the point needs to be on the subgroup is neccessary to uniquely determine the point.
//
// It returns an error if the provided input is invalid. In this case, the returned point must not be used.
// If trustLevel is TrustedInput, you *MUST* call this only with valid input; we are free to skip some tests.
// The library makes no guarantees whatsoever about what happens if you violate this.
func CurvePointFromXYTimesSignY_subgroup(xSignY *FieldElement, ySignY *FieldElement, trustlevel IsPointTrusted) (point Point_axtw_subgroup, err error) {
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
			// We also do not yet set point to a NaP, because we use point.t to continue the "not on curve" check.
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

// consumeExpectRead reads and consumes len(expectToRead) bytes from input and reports an error if the read bytes differ from expectToRead.
func consumeExpectRead(input io.Reader, expectToRead []byte) (bytes_read int, err error) {
	if len(expectToRead) == 0 {
		return 0, nil
	}
	var buf []byte = make([]byte, len(expectToRead))
	bytes_read, err = io.ReadFull(input, buf)
	if err != nil {
		return
	}
	if !bytes.Equal(expectToRead, buf) {
		err = fmt.Errorf("bandersnatch / deserialization: Unexpected Header encountered upon deserialization. Expected 0x%x, got 0x%x", expectToRead, buf)
	}
	return
}

type endianness struct {
	byteOrder binary.ByteOrder
}

func (s *endianness) getEndianness() binary.ByteOrder {
	return s.byteOrder
}

func (s *endianness) setEndianness(e binary.ByteOrder) {
	s.byteOrder = e
}

type valuesSerializerFeFe struct {
	endianness
}

func (s *valuesSerializerFeFe) deserializeValues(input io.Reader) (bytesRead int, err error, fieldElement1, fieldElement2 FieldElement) {
	bytesRead, err = fieldElement1.Deserialize(input, s.byteOrder)
	// Note: This aborts on ErrNonNormalizedDeserialization
	if err != nil {
		return
	}
	bytesJustRead, err := fieldElement2.Deserialize(input, s.byteOrder)
	bytesRead += bytesJustRead
	return
}

func (s *valuesSerializerFeFe) serializeValues(output io.Writer, fieldElement1, fieldElement2 *FieldElement) (bytesWritten int, err error) {
	bytesWritten, err = fieldElement1.Serialize(output, s.byteOrder)
	if err != nil {
		return
	}
	bytesJustWritten, err := fieldElement2.Serialize(output, s.byteOrder)
	bytesWritten += bytesJustWritten
	return
}

type valuesSerializerFe struct {
	endianness
}

func (s *valuesSerializerFe) deserializeValues(input io.Reader) (bytesRead int, err error, fieldElement FieldElement) {
	bytesRead, err = fieldElement.Deserialize(input, s.byteOrder)
	return
}

func (s *valuesSerializerFe) serializeValues(output io.Writer, fieldElement *FieldElement) (bytesWritten int, err error) {
	bytesWritten, err = fieldElement.Serialize(output, s.byteOrder)
	return
}

type valuesSerializerFeCompressedBit struct {
	endianness
}

func (s *valuesSerializerFeCompressedBit) deserializeValues(input io.Reader) (bytesRead int, err error, fieldElement FieldElement, bit bool) {
	var prefix PrefixBits
	bytesRead, prefix, err = fieldElement.DeserializeAndGetPrefix(input, 1, s.byteOrder)
	bit = (prefix == 0b1)
	return
}

func (s *valuesSerializerFeCompressedBit) serializeValues(output io.Writer, fieldElement *FieldElement, bit bool) (bytesWritten int, err error) {
	var embeddedPrefix PrefixBits
	if bit {
		embeddedPrefix = PrefixBits(0b1)
	} else {
		embeddedPrefix = PrefixBits(0b0)
	}
	bytesWritten, err = fieldElement.SerializeWithPrefix(output, embeddedPrefix, 1, s.byteOrder)
	return
}

type pointSerializerInterface interface {
	serializeCurvePoint(output io.Writer, point CurvePointPtrInterfaceRead) (bytesWritten int, err error)
	deserializeCurvePoint(input io.Reader, point CurvePointPtrInterfaceWrite, trustLevel IsPointTrusted) (bytesRead int, err error)
	clone() pointSerializerInterface
	getEndianness() binary.ByteOrder
	setEndianness(binary.ByteOrder)
}

func checkPointSerializability(point CurvePointPtrInterfaceRead, subgroupCheck bool) (err error) {
	if point.IsNaP() {
		err = ErrCannotSerializeNaP
		return
	}
	if point.IsAtInfinity() {
		err = ErrCannotSerializePointAtInfinity
		return
	}
	if subgroupCheck {
		if !point.IsInSubgroup() {
			err = ErrWillNotSerializePointOutsideSubgroup
			return
		}
	}
	return nil
}

type pointSerializerXY struct {
	valuesSerializerFeFe
	subgroupOnly bool
}

func (s *pointSerializerXY) serializeCurvePoint(output io.Writer, point CurvePointPtrInterfaceRead) (bytesWritten int, err error) {
	err = checkPointSerializability(point, s.subgroupOnly)
	if err != nil {
		return
	}
	X, Y := point.XY_affine()
	bytesWritten, err = s.valuesSerializerFeFe.serializeValues(output, &X, &Y)
	return
}

func (s *pointSerializerXY) deserializeCurvePoint(input io.Reader, point CurvePointPtrInterfaceWrite, trustLevel IsPointTrusted) (bytesRead int, err error) {
	var X, Y FieldElement
	bytesRead, err, X, Y = s.valuesSerializerFeFe.deserializeValues(input)
	if err != nil {
		return
	}
	if s.subgroupOnly || point.CanOnlyRepresentSubgroup() {
		var P Point_axtw_subgroup
		P, err = CurvePointFromXYAffine_subgroup(&X, &Y, trustLevel)
		if err != nil {
			return
		}
		ok := point.SetFromSubgroupPoint(&P, TrustedInput) // P is trusted at this point
		if !ok {
			// This is supposed to be impossible to happen (unless the user lied wrt trusted-ness of input)
			panic("bandersnatch: when deserializing a curve Point from X,Y-coordinates, conversion to the requested point type failed.")
		}
	} else {
		var P Point_axtw_full
		P, err = CurvePointFromXYAffine_full(&X, &Y, trustLevel)
		if err != nil {
			return
		}
		point.SetFrom(&P)
	}
	return
}

func (s *pointSerializerXY) clone() (ret pointSerializerInterface) {
	var sCopy pointSerializerXY
	sCopy.endianness = s.endianness
	sCopy.subgroupOnly = s.subgroupOnly
	ret = &sCopy
	return
}

type pointSerializerXAndSignY struct {
	valuesSerializerFeCompressedBit
	subgroupOnly bool
}

func (s *pointSerializerXAndSignY) serializeCurvePoint(output io.Writer, point CurvePointPtrInterfaceRead) (bytesWritten int, err error) {
	err = checkPointSerializability(point, s.subgroupOnly)
	if err != nil {
		return
	}
	X, Y := point.XY_affine()
	var SignY bool = Y.Sign() < 0
	bytesWritten, err = s.valuesSerializerFeCompressedBit.serializeValues(output, &X, SignY)
	return
}

func (s *pointSerializerXAndSignY) deserializeCurvePoint(input io.Reader, point CurvePointPtrInterfaceWrite, trustLevel IsPointTrusted) (bytesRead int, err error) {
	var X FieldElement
	var signBit bool
	bytesRead, err, X, signBit = s.valuesSerializerFeCompressedBit.deserializeValues(input)
	if err != nil {
		return
	}
	var signInt int
	if signBit {
		signInt = -1
	} else {
		signInt = +1
	}
	if s.subgroupOnly || point.CanOnlyRepresentSubgroup() {
		var P Point_axtw_subgroup
		P, err = CurvePointFromXAndSignY_subgroup(&X, signInt, trustLevel)
		if err != nil {
			return
		}
		ok := point.SetFromSubgroupPoint(&P, TrustedInput) // P is trusted at this point
		if !ok {
			// This is supposed to be impossible to happen (unless the user lied wrt trusted-ness of input)
			panic("bandersnatch: when deserializing a curve Point from X,Y-coordinates, conversion to the requested point type failed.")
		}
	} else {
		var P Point_axtw_full
		P, err = CurvePointFromXAndSignY_full(&X, signInt, trustLevel)
		if err != nil {
			return
		}
		point.SetFrom(&P)
	}
	return
}

func (s *pointSerializerXAndSignY) clone() (ret pointSerializerInterface) {
	var sCopy pointSerializerXAndSignY
	sCopy.endianness = s.endianness
	sCopy.subgroupOnly = s.subgroupOnly
	ret = &sCopy
	return
}

type pointSerializerYAndSignX struct {
	valuesSerializerFeCompressedBit
	subgroupOnly bool
}

func (s *pointSerializerYAndSignX) serializeCurvePoint(output io.Writer, point CurvePointPtrInterfaceRead) (bytesWritten int, err error) {
	err = checkPointSerializability(point, s.subgroupOnly)
	if err != nil {
		return
	}
	X, Y := point.XY_affine()
	var SignX bool = X.Sign() < 0 // for X==0, we want the sign bit to be NOT set.
	bytesWritten, err = s.valuesSerializerFeCompressedBit.serializeValues(output, &Y, SignX)
	return
}

func (s *pointSerializerYAndSignX) deserializeCurvePoint(input io.Reader, point CurvePointPtrInterfaceWrite, trustLevel IsPointTrusted) (bytesRead int, err error) {
	var Y FieldElement
	var signBit bool
	bytesRead, err, Y, signBit = s.valuesSerializerFeCompressedBit.deserializeValues(input)
	if err != nil {
		return
	}
	var signInt int
	if signBit {
		signInt = -1
	} else {
		signInt = +1
	}
	if s.subgroupOnly || point.CanOnlyRepresentSubgroup() {
		var P Point_axtw_subgroup
		P, err = CurvePointFromYAndSignX_subgroup(&Y, signInt, trustLevel)
		if err != nil {
			return
		}
		ok := point.SetFromSubgroupPoint(&P, TrustedInput) // P is trusted at this point
		if !ok {
			// This is supposed to be impossible to happen (unless the user lied wrt trusted-ness of input)
			panic("bandersnatch: when deserializing a curve Point from X,Y-coordinates, conversion to the requested point type failed.")
		}
	} else {
		var P Point_axtw_full
		P, err = CurvePointFromYAndSignX_full(&Y, signInt, trustLevel)
		if err != nil {
			return
		}
		point.SetFrom(&P)
	}
	return
}

func (s *pointSerializerYAndSignX) clone() (ret pointSerializerInterface) {
	var sCopy pointSerializerYAndSignX
	sCopy.endianness = s.endianness
	sCopy.subgroupOnly = s.subgroupOnly
	ret = &sCopy
	return
}

type pointSerializerXTimesSignY struct {
	valuesSerializerFe
	// subgroup only == true (implicit)
}

func (s *pointSerializerXTimesSignY) serializeCurvePoint(output io.Writer, point CurvePointPtrInterfaceRead) (bytesWritten int, err error) {
	err = checkPointSerializability(point, true)
	if err != nil {
		return
	}
	X := point.X_decaf_affine()
	Y := point.Y_decaf_affine()
	var SignY int = Y.Sign()
	if SignY < 0 {
		X.NegEq()
	}
	bytesWritten, err = s.valuesSerializerFe.serializeValues(output, &X)
	return
}

func (s *pointSerializerXTimesSignY) deserializeCurvePoint(input io.Reader, point CurvePointPtrInterfaceWrite, trustLevel IsPointTrusted) (bytesRead int, err error) {
	var XSignY FieldElement
	bytesRead, err, XSignY = s.valuesSerializerFe.deserializeValues(input)
	if err != nil {
		return
	}
	var P Point_axtw_subgroup
	P, err = CurvePointFromXTimesSignY_subgroup(&XSignY, trustLevel)
	if err != nil {
		return
	}
	ok := point.SetFromSubgroupPoint(&P, TrustedInput) // P is trusted at this point
	if !ok {
		// This is supposed to be impossible to happen (unless the user lied wrt trusted-ness of input)
		panic("bandersnatch: when deserializing a curve Point from X,Y-coordinates, conversion to the requested point type failed.")
	}
	return
}

type pointSerializerXYTimesSignY struct {
	valuesSerializerFeFe
	// subgroup only == true (implicit)
}

func (s *pointSerializerXYTimesSignY) serializeCurvePoint(output io.Writer, point CurvePointPtrInterfaceRead) (bytesWritten int, err error) {
	err = checkPointSerializability(point, true)
	if err != nil {
		return
	}
	X := point.X_decaf_affine()
	Y := point.Y_decaf_affine()
	var SignY int = Y.Sign()
	if SignY < 0 {
		X.NegEq()
		Y.NegEq()
	}
	bytesWritten, err = s.valuesSerializerFeFe.serializeValues(output, &X, &Y)
	return
}

func (s *pointSerializerXYTimesSignY) deserializeCurvePoint(input io.Reader, point CurvePointPtrInterfaceWrite, trustLevel IsPointTrusted) (bytesRead int, err error) {
	var XSignY, YSignY FieldElement
	bytesRead, err, XSignY, YSignY = s.valuesSerializerFeFe.deserializeValues(input)
	if err != nil {
		return
	}
	var P Point_axtw_subgroup
	P, err = CurvePointFromXYTimesSignY_subgroup(&XSignY, &YSignY, trustLevel)
	if err != nil {
		return
	}
	ok := point.SetFromSubgroupPoint(&P, TrustedInput) // P is trusted at this point
	if !ok {
		// This is supposed to be impossible to happen (unless the user lied wrt trusted-ness of input)
		panic("bandersnatch: when deserializing a curve Point from X,Y-coordinates, conversion to the requested point type failed.")
	}
	return
}

func (s *pointSerializerXYTimesSignY) clone() (ret pointSerializerInterface) {
	var sCopy pointSerializerXYTimesSignY
	sCopy.endianness = s.endianness
	ret = &sCopy
	return
}

type headerSerializer struct {
	// headerAll           []byte
	headerPerCurvePoint []byte
	// footer              []byte
	// headerAllReader     func(input io.Reader) (bytes_read int, err error, curvePointsToRead int, extra interface{})
	// headerPointReader   func(input io.Reader) (bytes_read int, err error, extra interface{})
}

func (hs *headerSerializer) clone() (ret headerSerializer) {
	if hs.headerPerCurvePoint == nil {
		ret.headerPerCurvePoint = nil
	} else {
		ret.headerPerCurvePoint = make([]byte, len(hs.headerPerCurvePoint))
		copy(ret.headerPerCurvePoint, hs.headerPerCurvePoint)
	}
	return
}

type simpleDeserializer struct {
	headerSerializer
	pointSerializer pointSerializerInterface
}

func (s *simpleDeserializer) Deserialize(outputPoint CurvePointPtrInterfaceWrite, inputStream io.Reader, trustLevel IsPointTrusted) (bytesRead int, err error) {
	var bytesJustRead int
	if s.headerSerializer.headerPerCurvePoint != nil {
		bytesRead, err = consumeExpectRead(inputStream, s.headerSerializer.headerPerCurvePoint)
		if err != nil {
			return
		}
	}
	bytesJustRead, err = s.pointSerializer.deserializeCurvePoint(inputStream, outputPoint, trustLevel)
	bytesRead += bytesJustRead
	return
}

func (s *simpleDeserializer) Serialize(inputPoint CurvePointPtrInterfaceRead, outputStream io.Writer) (bytesWritten int, err error) {
	var bytesJustWritten int
	if s.headerSerializer.headerPerCurvePoint != nil {
		bytesWritten, err = outputStream.Write(s.headerSerializer.headerPerCurvePoint)
		if err != nil {
			return
		}
	}
	bytesJustWritten, err = s.pointSerializer.serializeCurvePoint(outputStream, inputPoint)
	bytesWritten += bytesJustWritten
	return
}

func (s *simpleDeserializer) Clone() (ret simpleDeserializer) {
	ret.headerSerializer = s.headerSerializer.clone()
	ret.pointSerializer = s.pointSerializer.clone()
	return
}

func (s *simpleDeserializer) WithEndianness(e binary.ByteOrder) (ret simpleDeserializer) {
	ret = s.Clone()
	ret.pointSerializer.setEndianness(e)
	return
}

func (s *simpleDeserializer) WithHeader(perPointHeader []byte) (ret simpleDeserializer) {
	ret = s.Clone()
	if perPointHeader == nil {
		s.headerPerCurvePoint = nil
	} else {
		s.headerPerCurvePoint = make([]byte, len(perPointHeader))
		copy(s.headerPerCurvePoint, perPointHeader)
	}
	return
}
