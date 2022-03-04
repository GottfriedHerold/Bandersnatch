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
