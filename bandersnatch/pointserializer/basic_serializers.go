package pointserializer

import (
	"encoding/binary"
	"io"

	"github.com/GottfriedHerold/Bandersnatch/bandersnatch"
	"github.com/GottfriedHerold/Bandersnatch/bandersnatch/bandersnatchErrors"
	"github.com/GottfriedHerold/Bandersnatch/internal/utils"
)

// This file is part of the serialization-for-curve-points package.
// This package defines types that act as (de)serializers. These types hold metadata (such as e.g. endianness) about the serialization format.
// (De)serializers then have methods that are called with the actual curve point(s) as arguments to serialize them.

type errorWithPartialRead = bandersnatchErrors.ErrorWithData[bool]

// This file defines basic serializers that serialize and deserialize a single curve point.

// curvePointDeserializer_basic is a deserializer for single curve points
type curvePointDeserializer_basic interface {
	// DeserializeCurvePoint deserializes a single curve point from the inputStream. The output is written to output point. TrustLevel determines whether we trust the input to be a valid representation of a curve point.
	// (The latter includes subgroup checks if outputPoint can only store subgroup points)
	// On error, outputPoint is kept unchanged.
	DeserializeCurvePoint(inputStream io.Reader, trustLevel bandersnatch.IsPointTrusted, outputPoint bandersnatch.CurvePointPtrInterfaceWrite) (bytesRead int, err error)
	IsSubgroupOnly() bool // Can be called on nil pointers of concrete type, indicates whether the deserializer is only for subgroup points.
	OutputLength() int32  // returns the length in bytes that this serializer will try to read/write per curve point. For deserializers withot serializers, it is an upper bound.

	GetParameter(parameterName string) interface{} // obtains a parameter (such as endianness. parameterName is a case-insensitive.
	GetEndianness() binary.ByteOrder               // returns the endianness used for field element serialization
	Verifier

	// additional required interface (checked and accessed via reflection, because Go's type system is too weak to express this)
	// WithParameter(parameterName string, newParam any) SELF  // returns a (non-pointer) copy of the receiver with parameter paramName replaced by newParam.
	// WithEndianness(binary.ByteOrder) SELF // returns a (non-pointer) copy of the receiver with the desired endianness for field element serialization. Only supports binary.LittleEndian and binary.BigEndian
}

type modifyableSerializer[SelfValue any] interface {
	WithParameter(parameterName string, newParam any) SelfValue
	WithEndianness(newEndianness binary.ByteOrder) SelfValue
	utils.Clonable[*SelfValue]
}

type modifyableDeserializer_basic[SelfValue any] interface {
	curvePointDeserializer_basic
	modifyableSerializer[SelfValue]
}

// curvePointSerializer_basic is a serializer+deserializer for single curve points.
type curvePointSerializer_basic interface {
	curvePointDeserializer_basic
	SerializeCurvePoint(outputStream io.Writer, inputPoint bandersnatch.CurvePointPtrInterfaceRead) (bytesWritten int, err *errorWithPartialRead)
}

type modifyableSerializer_basic[SelfValue any] interface {
	curvePointSerializer_basic
	modifyableSerializer[SelfValue]
}

// TODO: Separate into separate checks?

// checkPointSerializability verifies that the point is not a NaP or infinite. If subgroupCheck is set to true, also ensures that the point is in the p253-prime order subgroup.
// If everything is fine, returns nil. These correspond to the points that we usually want to serialize.
//
// Note: This function is is typically called before serializing (not for deserializing), where we do not have a trustLevel argument.
// This means that we always check whether the point is in the subgroup for any writes if the serializer is subgroup-only. Note for efficiency that this check is actually
// trivial if the type of point can only represent subgroup elements; we assume that this is the most common usage scenario.
func checkPointSerializability(point bandersnatch.CurvePointPtrInterfaceRead, subgroupCheck bool) (err error) {
	if point.IsNaP() {
		err = bandersnatchErrors.ErrCannotSerializeNaP
		return
	}
	if point.IsAtInfinity() {
		err = bandersnatchErrors.ErrCannotSerializePointAtInfinity
		return
	}
	if subgroupCheck {
		if !point.IsInSubgroup() {
			err = bandersnatchErrors.ErrWillNotSerializePointOutsideSubgroup
			return
		}
	}
	return nil
}

// we now define some "basic" serializers, basic being in the sense that they only allow (de)serializing a single point.
// They also do not allow headers (except possibly embedded)

// pointSerializerXY is a simple serializer that works by just writing / reading both the affine X and Y coordinates.
// If subgroupOnly is set to true, it will only work for points in the subgroup.
//
// NOTE: This cannot serialize points at infinity atm, even if subgroupRestriction is set to false
type pointSerializerXY struct {
	valuesSerializerHeaderFeHeaderFe
	subgroupRestriction // wraps a bool
}

func (s *pointSerializerXY) SerializeCurvePoint(output io.Writer, point bandersnatch.CurvePointPtrInterfaceRead) (bytesWritten int, err error) {
	err = checkPointSerializability(point, s.IsSubgroupOnly())
	if err != nil {
		bandersnatchErrors.IncludeParametersInError(&err, PARTIAL_READ_FLAG, false)
		return
	}
	X, Y := point.XY_affine()
	bytesWritten, err = s.valuesSerializerHeaderFeHeaderFe.SerializeValues(output, &X, &Y)
	return
}

func (s *pointSerializerXY) DeserializeCurvePoint(input io.Reader, trustLevel bandersnatch.IsPointTrusted, point bandersnatch.CurvePointPtrInterfaceWrite) (bytesRead int, err error) {
	var X, Y bandersnatch.FieldElement
	// var errPlain error
	bytesRead, err, X, Y = s.DeserializeValues(input)
	if err != nil {
		return
	}
	if s.IsSubgroupOnly() || point.CanOnlyRepresentSubgroup() {
		// using a temporary P here to ensure P is unchanged on error
		var P bandersnatch.Point_axtw_subgroup
		P, err = bandersnatch.CurvePointFromXYAffine_subgroup(&X, &Y, trustLevel)
		if err != nil {
			bandersnatchErrors.IncludeParametersInError(&err, PARTIAL_READ_FLAG, false)
			return
		}
		point.SetFrom(&P)
	} else {
		// using a temporary P here to ensure P is unchanged on error
		var P bandersnatch.Point_axtw_full
		P, err = bandersnatch.CurvePointFromXYAffine_full(&X, &Y, trustLevel)
		if err != nil {
			bandersnatchErrors.IncludeParametersInError(&err, PARTIAL_READ_FLAG, false)
			return
		}
		point.SetFrom(&P)
	}
	return
}

func (s *pointSerializerXY) Verify() {
	s.valuesSerializerHeaderFeHeaderFe.Verify()
	s.subgroupRestriction.Verify()
}

func (s *pointSerializerXY) Clone() (ret *pointSerializerXY) {
	var sCopy pointSerializerXY = *s
	ret = &sCopy
	return
}

func (s *pointSerializerXY) WithParameter(param string, newParam interface{}) (newSerializer pointSerializerXY) {
	return makeCopyWithParamsNew(s, param, newParam)
	// return makeCopyWithParams(s, param, newParam).(pointSerializerXY)
}

func (s *pointSerializerXY) WithEndianness(newEndianness binary.ByteOrder) pointSerializerXY {
	return s.WithParameter("endianness", newEndianness)
}

func (s *pointSerializerXY) OutputLength() int32 { return 64 }

func (s *pointSerializerXY) GetParameter(parameterName string) interface{} {
	return getSerializerParam(s, parameterName)
}

// pointSerializerXAndSignY is a Serialializer that serializes the affine X coordinate and the sign of the Y coordinate. (Note that the latter is never 0)
//
// More precisely, we write a 1 bit into the msb of the output (if interpreteed as 256bit-number) if the sign of Y is negative.
type pointSerializerXAndSignY struct {
	valuesSerializerFeCompressedBit
	subgroupRestriction
}

func (s *pointSerializerXAndSignY) SerializeCurvePoint(output io.Writer, point bandersnatch.CurvePointPtrInterfaceRead) (bytesWritten int, err error) {
	err = checkPointSerializability(point, s.subgroupOnly)
	if err != nil {
		bandersnatchErrors.IncludeParametersInError(&err, PARTIAL_READ_FLAG, false)
		return
	}
	X, Y := point.XY_affine()
	var SignY bool = Y.Sign() < 0 // canot be == 0
	bytesWritten, err = s.SerializeValues(output, &X, SignY)
	return
}

func (s *pointSerializerXAndSignY) DeserializeCurvePoint(input io.Reader, trustLevel bandersnatch.IsPointTrusted, point bandersnatch.CurvePointPtrInterfaceWrite) (bytesRead int, err error) {
	var X bandersnatch.FieldElement
	var signBit bool
	bytesRead, err, X, signBit = s.DeserializeValues(input)
	if err != nil {
		return
	}

	//  convert boolean sign bit to +/-1 - valued sign
	var signInt int
	if signBit {
		signInt = -1
	} else {
		signInt = +1
	}

	if s.IsSubgroupOnly() || point.CanOnlyRepresentSubgroup() {
		var P bandersnatch.Point_axtw_subgroup
		P, err = bandersnatch.CurvePointFromXAndSignY_subgroup(&X, signInt, trustLevel)
		if err != nil {
			bandersnatchErrors.IncludeParametersInError(&err, PARTIAL_READ_FLAG, false)
			return
		}
		point.SetFrom(&P)
	} else {
		var P bandersnatch.Point_axtw_full
		P, err = bandersnatch.CurvePointFromXAndSignY_full(&X, signInt, trustLevel)
		if err != nil {
			bandersnatchErrors.IncludeParametersInError(&err, PARTIAL_READ_FLAG, false)
			return
		}
		point.SetFrom(&P)
	}
	return
}

func (s *pointSerializerXAndSignY) Clone() (ret *pointSerializerXAndSignY) {
	var sCopy pointSerializerXAndSignY = *s
	ret = &sCopy
	return
}

func (s *pointSerializerXAndSignY) WithParameter(param string, newParam interface{}) (newSerializer pointSerializerXAndSignY) {
	return makeCopyWithParamsNew(s, param, newParam)
	// return makeCopyWithParams(s, param, newParam).(pointSerializerXAndSignY)
}

func (s *pointSerializerXAndSignY) WithEndianness(newEndianness binary.ByteOrder) pointSerializerXAndSignY {
	return s.WithParameter("endianness", newEndianness)
}

func (s *pointSerializerXAndSignY) OutputLength() int32 { return 32 }

func (s *pointSerializerXAndSignY) GetParam(parameterName string) interface{} {
	return getSerializerParam(s, parameterName)
}

func (s *pointSerializerXAndSignY) Verify() {
	s.valuesSerializerFeCompressedBit.Verify()
	s.fieldElementEndianness.Verify()
}

// pointSerializerYAndSignX serializes a point via its Y coordinate and the sign of X. (For X==0, we do not set the sign bit)
type pointSerializerYAndSignX struct {
	valuesSerializerFeCompressedBit
	subgroupRestriction
}

func (s *pointSerializerYAndSignX) Verify() {
	s.valuesSerializerFeCompressedBit.Verify()
	s.subgroupRestriction.Verify()
}

func (s *pointSerializerYAndSignX) SerializeCurvePoint(output io.Writer, point bandersnatch.CurvePointPtrInterfaceRead) (bytesWritten int, err error) {
	err = checkPointSerializability(point, s.IsSubgroupOnly())
	if err != nil {
		bandersnatchErrors.IncludeParametersInError(&err, PARTIAL_READ_FLAG, false)
		return
	}
	X, Y := point.XY_affine()
	var SignX bool = X.Sign() < 0 // for X==0, we want the sign bit to be NOT set.
	bytesWritten, err = s.SerializeValues(output, &Y, SignX)
	return
}

func (s *pointSerializerYAndSignX) DeserializeCurvePoint(input io.Reader, trustLevel bandersnatch.IsPointTrusted, point bandersnatch.CurvePointPtrInterfaceWrite) (bytesRead int, err error) {
	var Y bandersnatch.FieldElement
	var signBit bool
	bytesRead, err, Y, signBit = s.DeserializeValues(input)
	if err != nil {
		return
	}
	var signInt int
	if signBit {
		signInt = -1
	} else {
		signInt = +1
	}

	// Note: CurvePointFromYAndSignX_* accepts any sign for Y=+/-1.
	// We need to correct this to ensure uniqueness of the serialized representation.

	if s.subgroupOnly || point.CanOnlyRepresentSubgroup() {
		var P bandersnatch.Point_axtw_subgroup
		P, err = bandersnatch.CurvePointFromYAndSignX_subgroup(&Y, signInt, trustLevel)
		if err != nil {
			bandersnatchErrors.IncludeParametersInError(&err, PARTIAL_READ_FLAG, false)
			return
		}

		// This can only happen if Y = +1. In this case, we only accept signBit = false, as that's what we write when serializing.
		if P.IsNeutralElement() && signBit {
			err = bandersnatchErrors.ErrUnexpectedNegativeZero
			bandersnatchErrors.IncludeParametersInError(&err, PARTIAL_READ_FLAG, false)
			return
		}

		point.SetFrom(&P) // P is trusted at this point
	} else {
		var P bandersnatch.Point_axtw_full
		P, err = bandersnatch.CurvePointFromYAndSignX_full(&Y, signInt, trustLevel)
		if err != nil {
			bandersnatchErrors.IncludeParametersInError(&err, PARTIAL_READ_FLAG, false)
			return
		}

		// Special case: If Y = +/-1, we have X=0. In that case, we only accept signBit = false, as that's what we write when serializing.
		{
			var X bandersnatch.FieldElement = P.X_decaf_affine()
			if X.IsZero() && signBit {
				err = bandersnatchErrors.ErrUnexpectedNegativeZero
				bandersnatchErrors.IncludeParametersInError(&err, PARTIAL_READ_FLAG, false)
				return
			}
		}

		point.SetFrom(&P)
	}
	return
}

func (s *pointSerializerYAndSignX) Clone() (ret *pointSerializerYAndSignX) {
	var sCopy pointSerializerYAndSignX
	sCopy.fieldElementEndianness = s.fieldElementEndianness
	sCopy.subgroupOnly = s.subgroupOnly
	ret = &sCopy
	return
}

func (s *pointSerializerYAndSignX) WithParameter(param string, newParam interface{}) (newSerializer pointSerializerYAndSignX) {
	return makeCopyWithParamsNew(s, param, newParam)
}

func (s *pointSerializerYAndSignX) WithEndianness(newEndianness binary.ByteOrder) pointSerializerYAndSignX {
	return s.WithParameter("endianness", newEndianness)
}

func (s *pointSerializerYAndSignX) OutputLength() int32 { return 32 }

func (s *pointSerializerYAndSignX) GetParam(parameterName string) interface{} {
	return getSerializerParam(s, parameterName)
}

// pointSerializerXTimesSignY is a basic serializer that serializes via X * Sign(Y).
// Note that this only works for points in the subgroup, as the information of being in the subgroup
// is needed to deserialize uniquely.
type pointSerializerXTimesSignY struct {
	valuesSerializerHeaderFe
	subgroupOnly
}

func (s *pointSerializerXTimesSignY) Verify() {
	s.valuesSerializerHeaderFe.Verify()
	s.subgroupOnly.Verify()
}

func (s *pointSerializerXTimesSignY) SerializeCurvePoint(output io.Writer, point bandersnatch.CurvePointPtrInterfaceRead) (bytesWritten int, err error) {
	err = checkPointSerializability(point, true)
	if err != nil {
		bandersnatchErrors.IncludeParametersInError(&err, PARTIAL_READ_FLAG, false)
		return
	}
	X := point.X_decaf_affine()
	Y := point.Y_decaf_affine()
	var SignY int = Y.Sign()
	if SignY < 0 {
		X.NegEq()
	}
	bytesWritten, err = s.SerializeValues(output, &X)
	return
}

func (s *pointSerializerXTimesSignY) DeserializeCurvePoint(input io.Reader, trustLevel bandersnatch.IsPointTrusted, point bandersnatch.CurvePointPtrInterfaceWrite) (bytesRead int, err error) {
	var XSignY bandersnatch.FieldElement
	bytesRead, err, XSignY = s.DeserializeValues(input)
	if err != nil {
		return
	}
	var P bandersnatch.Point_axtw_subgroup
	P, err = bandersnatch.CurvePointFromXTimesSignY_subgroup(&XSignY, trustLevel)
	if err != nil {
		bandersnatchErrors.IncludeParametersInError(&err, PARTIAL_READ_FLAG, false)
		return
	}
	point.SetFrom(&P)
	return
}

func (s *pointSerializerXTimesSignY) Clone() (ret *pointSerializerXTimesSignY) {
	var sCopy pointSerializerXTimesSignY = *s
	return &sCopy
}

func (s *pointSerializerXTimesSignY) WithParameter(param string, newParam interface{}) (newSerializer pointSerializerXTimesSignY) {
	return makeCopyWithParamsNew(s, param, newParam)
}

func (s *pointSerializerXTimesSignY) WithEndianness(newEndianness binary.ByteOrder) pointSerializerXTimesSignY {
	return s.WithParameter("endianness", newEndianness)
}

func (s *pointSerializerXTimesSignY) OutputLength() int32 { return 32 }

func (s *pointSerializerXTimesSignY) GetParam(parameterName string) interface{} {
	return getSerializerParam(s, parameterName)
}

// pointSerializerYXTimesSignY is a serializer that uses X*Sign(Y), Y*Sign(Y).
// This serializer only works for subgroup elements:
// The fact of being in the subgroup is needed to uniquely deserialize.
type pointSerializerYXTimesSignY struct {
	valuesSerializerHeaderFeHeaderFe
	subgroupOnly
}

func (s *pointSerializerYXTimesSignY) Verify() {
	s.valuesSerializerHeaderFeHeaderFe.Verify()
	s.subgroupOnly.Verify()
}

func (s *pointSerializerYXTimesSignY) SerializeCurvePoint(output io.Writer, point bandersnatch.CurvePointPtrInterfaceRead) (bytesWritten int, err error) {
	err = checkPointSerializability(point, true)
	if err != nil {
		bandersnatchErrors.IncludeParametersInError(&err, PARTIAL_READ_FLAG, false)
		return
	}
	X := point.X_decaf_affine()
	Y := point.Y_decaf_affine()
	var SignY int = Y.Sign()
	if SignY < 0 {
		X.NegEq()
		Y.NegEq()
	}
	bytesWritten, err = s.SerializeValues(output, &Y, &X)
	return
}

func (s *pointSerializerYXTimesSignY) DeserializeCurvePoint(input io.Reader, trustLevel bandersnatch.IsPointTrusted, point bandersnatch.CurvePointPtrInterfaceWrite) (bytesRead int, err error) {
	var XSignY, YSignY bandersnatch.FieldElement
	bytesRead, err, YSignY, XSignY = s.DeserializeValues(input)
	if err != nil {
		return
	}
	var P bandersnatch.Point_axtw_subgroup
	P, err = bandersnatch.CurvePointFromXYTimesSignY_subgroup(&XSignY, &YSignY, trustLevel)
	if err != nil {
		bandersnatchErrors.IncludeParametersInError(&err, PARTIAL_READ_FLAG, false)
		return
	}
	ok := point.SetFromSubgroupPoint(&P, bandersnatch.TrustedInput) // P is trusted at this point
	if !ok {
		// This is supposed to be impossible to happen (unless the user lied wrt trusted-ness of input)
		panic("bandersnatch: when deserializing a curve Point from X,Y-coordinates, conversion to the requested point type failed.")
	}
	return
}

func (s *pointSerializerYXTimesSignY) Clone() (ret *pointSerializerYXTimesSignY) {
	var sCopy pointSerializerYXTimesSignY = *s
	ret = &sCopy
	return
}

func (s *pointSerializerYXTimesSignY) WithParameter(param string, newParam interface{}) (newSerializer pointSerializerYXTimesSignY) {
	return makeCopyWithParamsNew(s, param, newParam)
}

func (s *pointSerializerYXTimesSignY) WithEndianness(newEndianness binary.ByteOrder) pointSerializerYXTimesSignY {
	return s.WithParameter("endianness", newEndianness)
}

func (s *pointSerializerYXTimesSignY) OutputLength() int32 { return 64 }

func (s *pointSerializerYXTimesSignY) GetParam(parameterName string) interface{} {
	return getSerializerParam(s, parameterName)
}

// Note: This selection of bitHeaders is not the original spec, but this has the advantage that an all-zeroes input actually causes an error (rather than be interpreted as the neutral element)

var bitHeaderBanderwagonX = bitHeader{prefixLen: 1, prefixBits: 1}
var bitHeaderBanderwagonY = bitHeader{prefixLen: 2, prefixBits: 0b00}

var basicBanderwagonShort = pointSerializerXTimesSignY{valuesSerializerHeaderFe: valuesSerializerHeaderFe{fieldElementEndianness: defaultEndianness, bitHeader: bitHeaderBanderwagonX}, subgroupOnly: subgroupOnly{}}
var basicBanderwagonLong = pointSerializerYXTimesSignY{valuesSerializerHeaderFeHeaderFe: valuesSerializerHeaderFeHeaderFe{fieldElementEndianness: defaultEndianness, bitHeader: bitHeaderBanderwagonY, bitHeader2: bitHeaderBanderwagonX}, subgroupOnly: subgroupOnly{}}

func init() {
	bitHeaderBanderwagonX.Verify()
	bitHeaderBanderwagonY.Verify()
	basicBanderwagonShort.Verify()
	basicBanderwagonLong.Verify()
}
