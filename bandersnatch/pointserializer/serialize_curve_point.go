package curveserialize

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strings"

	. "github.com/GottfriedHerold/Bandersnatch/bandersnatch"
	"github.com/GottfriedHerold/Bandersnatch/bandersnatch/bandersnatchErrors"
)

type curvePointDeserializer_basic interface {
	DeserializeCurvePoint(inputStream io.Reader, trustLevel IsPointTrusted, outputPoint CurvePointPtrInterfaceWrite) (bytesRead int, err error)
}

// intended additional interface
// WithEndianness(binary.ByteOrder) SELF
// WithSubgroupOnly(bool) SELF
//
// using reflectection, because Go's expressiveness for types is ... lacking, to say the least

/*
type curvePointDeserializer_basicModifyable interface {
	curvePointDeserializer_basic
	WithEndianness(e binary.ByteOrder) curvePointDeserializer_basicModifyable
	WithSubgroupOnly(bool) curvePointDeserializer_basicModifyable
}
*/

type curvePointSerializer_basic interface {
	SerializeCurvePoint(outputStream io.Writer, inputPoint CurvePointPtrInterfaceRead) (bytesWritten int, err error)
}

const (
	basicSerializerCloneFun            = "Clone"
	basicSerializerNewEndiannessFun    = "WithEndianness"
	basicSerializerNewSubgroupRestrict = "WithSubgroupOnly"
)

type CurvePointDeserializer interface {
	curvePointDeserializer_basic // TODO: Copy definition for godoc
	DeserializePoints(inputStream io.Reader, outputPoints CurvePointSlice) (bytesRead int, err bandersnatchErrors.BatchSerializationError)
	DeserializeBatch(inputStream io.Reader, outputPoints ...CurvePointPtrInterfaceWrite) (bytesRead int, err bandersnatchErrors.BatchSerializationError)

	// Matches SerializeSlice
	DeserializeSlice(inputStream io.Reader) (outputPoints CurvePointSlice, bytesRead int, err bandersnatchErrors.BatchSerializationError)
	DeserializeSliceToBuffer(inputStream io.Reader, outputPoints CurvePointSlice) (bytesRead int, pointsRead int, err bandersnatchErrors.BatchSerializationError)
}

type CurvePointSerializer interface {
	CurvePointDeserializer
	curvePointSerializer_basic
	SerializePoints(outputStream io.Writer, inputPoints CurvePointSlice) (bytesWritten int, err bandersnatchErrors.BatchSerializationError) // SerializeBatch(os, points) is equivalent (if no error occurs) to calling Serialize(os, point[i]) for all i. NOTE: This provides the same functionality as SerializePoints, but with a different argument type.
	SerializeBatch(outputStream io.Writer, inputPoints ...CurvePointPtrInterfaceRead) (bytesWritten int, err error)                         // SerializePoints(os, &x1, &x2, ...) is equivalent (if not error occurs, at least) to Serialize(os, &x1), Serialize(os, &x1), ... NOTE: Using SerializePoints(os, points...) with ...-notation might not work due to the need to convert []concrete Point type to []CurvePointPtrInterface. Use SerializeBatch to avoid this.
	SerializeSlice(outputStream io.Writer, inputSlice CurvePointSlice) (bytesWritten int, err bandersnatchErrors.BatchSerializationError)   // SerializeSlice(os, points) serializes a slice of points to outputStream. As opposed to SerializeBatch and SerializePoints, the number of points written is stored in the output stream and can NOT be read back individually, but only by DeserializeSlice
}

// consumeExpectRead reads and consumes len(expectToRead) bytes from input and reports an error if the read bytes differ from expectToRead.
// This is intended to read headers. Remember to use errors.Is to check the returned errors rather than == due to error wrapping.
//
// NOTES:
// Returns an error wrapping io.ErrUnexpectedEOF or io.EOF on end-of-file (io.EOF if the io.Reader was in EOF file to start with, io.ErrUnexpectedEOF if we encounter EOF after reading >0 bytes)
// On mismatch of expectToRead vs. actually read values, returns an error wrapping ErrDidNotReadExpectedString
func consumeExpectRead(input io.Reader, expectToRead []byte) (bytes_read int, err error) {
	if len(expectToRead) == 0 {
		return 0, nil
	}
	var buf []byte = make([]byte, len(expectToRead))
	bytes_read, err = io.ReadFull(input, buf)
	if err != nil {
		if errors.Is(err, io.ErrUnexpectedEOF) {
			err = fmt.Errorf("bandersnatch / deserialization: Unexpected EOF after reading %v out of %v bytes when reading header.\nReported error was %w.\nBytes expected were 0x%x, got 0x%x", bytes_read, len(expectToRead), err, expectToRead, buf[0:bytes_read])
		}
		if errors.Is(err, io.EOF) {
			err = bandersnatchErrors.NewWrappedError(err, fmt.Sprintf("bandersnatch / deserialization: Unexpected EOF when trying to read buffer.\nExpected to read 0x%x, got EOF instead", expectToRead))
		}
		return
	}
	if !bytes.Equal(expectToRead, buf) {
		err = bandersnatchErrors.NewWrappedError(bandersnatchErrors.ErrDidNotReadExpectedString, fmt.Sprintf("bandersnatch / deserialization: Unexpected Header encountered upon deserialization. Expected 0x%x, got 0x%x", expectToRead, buf))
	}
	return
}

// .With(...) forwards to this

//type serializerParamsEn

var serializerParams = map[string]struct {
	getter  string
	setter  string
	vartype reflect.Type
}{
	"endianness": {getter: "GetEndianness", setter: "SetEndianness", vartype: reflect.TypeOf(fieldElementEndianness{})},
	"bitheader":  {getter: "GetBitHeader", setter: "SetBitHeader", vartype: reflect.TypeOf(bitHeader{})},
}

func makeCopyWithParams(serializer interface{}, param string, newParam interface{}) interface{} {
	param = strings.ToLower(param) // make params case-insensitive
	paramInfo, ok := serializerParams[param]
	if !ok {
		panic("bandersnatch / serialization: makeCopyWithParams called with unrecognized parameter name")
	}

	serializerValue := reflect.ValueOf(serializer)
	// serializerType := reflect.TypeOf(serializer)
	cloneMethod := serializerValue.MethodByName("Clone")
	if !cloneMethod.IsValid() {
		panic("bandersnatch / serialization: makeCopyWithParams called with non-clonable serializer")
	}
	cloneMethodType := cloneMethod.Type()
	if cloneMethodType.NumIn() != 0 {
		panic("bandersnatch / serialization: makeCopyWithParams called with type whose Clone() method has >0 args")
	}
	if cloneMethodType.NumOut() != 1 {
		panic("bandersnatch / serialization: makeCopyWithParams called with type whose Clone() method returns != 1 args")
	}
	serializerClone := cloneMethod.Call([]reflect.Value{})[0]
	// serializerClone.Type() ought to be the same as serializerValue.Type(), up to pointer indirection. We care only about the result of clone(), which should be
	// a pointer (since we need to modify the result)
	serializerType := serializerClone.Type()
	if serializerType.Kind() != reflect.Ptr {
		// We could take the adress and work with values as well, but none of our serializers does that
		panic("bandersnatch / serialization: makeCopyWithParams calles with type whole Clone() method returns non-pointer type")
	}
	setterMethod := serializerClone.MethodByName(paramInfo.setter)
	if !setterMethod.IsValid() {
		panic(fmt.Errorf("bandersnatch / serialization: makeCopyWithParams called with type lacking a setter method %v for the requested parameter %v", paramInfo.setter, param))
	}
	newParamValue := reflect.ValueOf(newParam)
	newParamType := newParamValue.Type()
	if newParamType != paramInfo.vartype {
		panic(fmt.Errorf("bandersnatch / serialization: makeCopyWithParams called with wrong type of argument. Expected argument type was %v", paramInfo.vartype.Name()))
	}
	setterMethod.Call([]reflect.Value{newParamValue})
	return serializerClone.Elem().Interface()
}

// fieldElementEndianness is just a wrapper around binary.ByteOrder. It is part of serializers to control the fieldElementEndianness of field elements.
// Note that we ONLY support the predefined standard library constants binary.BigEndian and binary.LittleEndian.
// the reason is that the binary.ByteOrder interface is restricted to the default integer types and the interface lacks any general way to meaningfull extend it to 256-bit ints for field elements.
type fieldElementEndianness struct {
	byteOrder binary.ByteOrder
}

func (s *fieldElementEndianness) GetEndianness() binary.ByteOrder {
	return s.byteOrder
}

func (s *fieldElementEndianness) SetEndianness(e binary.ByteOrder) {
	if e != binary.BigEndian && e != binary.LittleEndian {
		panic("bandersnatch / serialize: we only support binary.BigEndian and binary.LittleEndian from the standard library as possible endianness")
	}
	s.byteOrder = e
}

// bitHeader is a "header" consisting of a prefixLen many extra bits that are included inside a field element as a form of compression.
type bitHeader struct {
	prefixBits PrefixBits
	prefixLen  uint8
}

func (bh *bitHeader) SetBitHeader(prefixBits PrefixBits, prefixLen uint8) {
	if prefixLen > 8 {
		panic("bandersnatch / serialization: trying to set bit-prefix of length > 8")
	}
	bitFilter := (1 << prefixLen) - 1 // bitmask of the form 0b0..01..1 ending with prefixLen 1s
	if bitFilter&int(prefixBits) != int(prefixBits) {
		panic("bandersnatch / serialization: trying to set bitHeader with a prefix and length, where the prefix has bits set that are not among the length many lsb")
	}
	bh.prefixBits = prefixBits
	bh.prefixLen = prefixLen
}

func (bh *bitHeader) GetBitHeader() (prefixBits PrefixBits, prefixLen uint8) {
	prefixBits = bh.prefixBits
	prefixLen = bh.prefixLen
	return
}

// implicit interface with methods SetSubgroupRestriction(bool) and IsSubgroupOnly() bool defined in tests only. Since we use reflection, we don't need the explicit interface here.

// subgroupRestriction is a type wrapping a bool that determines whether the serializer only works for subgroup elements (to use struct embedding in order to forward getter and setters to be found be reflect)
type subgroupRestriction struct {
	subgroupOnly bool
}

func (sr *subgroupRestriction) SetSubgroupRestriction(restrict bool) {
	sr.subgroupOnly = restrict
}

func (sr *subgroupRestriction) IsSubgroupOnly() bool {
	return sr.subgroupOnly
}

// subgroupOnly is a type wrapping a bool constant true that indicates that the serializer only works for subgroup elements. Used as embedded struct to forward setter and getter methods to reflect.
type subgroupOnly struct {
}

func (sr *subgroupOnly) IsSubgroupOnly() bool {
	return true
}

func (sr *subgroupOnly) SetSubgroupRestriction(restrict bool) {
	if !restrict {
		panic("bandersnatch / serialization: Trying to unset restriction to subgroup points for a serializer that does not support this")
	}
}

// Due to lack of generics, we separate our serializers depending on whether the internal object that actually gets serialized consists of
// a field element, two field element, field element+bit etc.
// These (de)serializiers all have serializeValues and DeserializeValues methods, which differ in their arguments.

// valuesSerializerFeFe is a simple serializer for a pair of field elements
type valuesSerializerFeFe struct {
	fieldElementEndianness // meaning the endianness for fieldElementSerialization
}

func (s *valuesSerializerFeFe) DeserializeValues(input io.Reader) (bytesRead int, err error, fieldElement1, fieldElement2 FieldElement) {
	bytesRead, err = fieldElement1.Deserialize(input, s.byteOrder)
	// Note: This aborts on ErrNonNormalizedDeserialization
	if err != nil {
		return
	}
	bytesJustRead, err := fieldElement2.Deserialize(input, s.byteOrder)
	// We treat EOF like UnexpectedEOF at this point. The reason is that we treat the PAIR of field elements as a unit.
	if errors.Is(err, io.EOF) {
		err = io.ErrUnexpectedEOF
	}
	bytesRead += bytesJustRead
	return
}

func (s *valuesSerializerFeFe) SerializeValues(output io.Writer, fieldElement1, fieldElement2 *FieldElement) (bytesWritten int, err error) {
	bytesWritten, err = fieldElement1.Serialize(output, s.byteOrder)
	if err != nil {
		return
	}
	bytesJustWritten, err := fieldElement2.Serialize(output, s.byteOrder)
	// We treat EOF like UnexpectedEOF at this point. The reason is that we treat the PAIR of field elements as a unit.
	if errors.Is(err, io.EOF) {
		err = io.ErrUnexpectedEOF
	}
	bytesWritten += bytesJustWritten
	return
}

func (s *valuesSerializerFeFe) Clone() *valuesSerializerFeFe {
	return &valuesSerializerFeFe{fieldElementEndianness: s.fieldElementEndianness}
}

// valuesSerializerHeaderFeHeaderFe is a serializer for a pair of field elements, where each of the two field elements has a prefix (of sub-byte length) contained in the
// msbs. These prefixes are fixed headers for the serializer and not part of the individual output/input field elements.
type valuesSerializerHeaderFeHeaderFe struct {
	fieldElementEndianness
	bitHeader  //bitHeader for the first field element. This is embedded, so we don't have to forward setters/getters.
	bitHeader2 bitHeader
}

func (s *valuesSerializerHeaderFeHeaderFe) DeserializeValues(input io.Reader) (bytesRead int, err error, fieldElement1, fieldElement2 FieldElement) {
	bytesRead, err = fieldElement1.DeserializeWithPrefix(input, s.prefixBits, s.prefixLen, s.byteOrder)
	// Note: This aborts on ErrNonNormalizedDeserialization
	if err != nil {
		return
	}
	bytesJustRead, err := fieldElement2.DeserializeWithPrefix(input, s.bitHeader2.prefixBits, s.bitHeader2.prefixLen, s.byteOrder)
	// We treat EOF like UnexpectedEOF at this point. The reason is that we treat the PAIR of field elements as a unit.
	if errors.Is(err, io.EOF) {
		err = io.ErrUnexpectedEOF
	}
	bytesRead += bytesJustRead
	return
}

func (s *valuesSerializerHeaderFeHeaderFe) serializeValues(output io.Writer, fieldElement1, fieldElement2 *FieldElement) (bytesWritten int, err error) {
	bytesWritten, err = fieldElement1.SerializeWithPrefix(output, s.prefixBits, s.prefixLen, s.byteOrder)
	if err != nil {
		return
	}
	bytesJustWritten, err := fieldElement2.SerializeWithPrefix(output, s.bitHeader2.prefixBits, s.bitHeader2.prefixLen, s.byteOrder)
	// We treat EOF like UnexpectedEOF at this point. The reason is that we treat the PAIR of field elements as a unit.
	if errors.Is(err, io.EOF) {
		err = io.ErrUnexpectedEOF
	}
	bytesWritten += bytesJustWritten
	return
}

func (s *valuesSerializerHeaderFeHeaderFe) SetBitHeader2(prefixBits PrefixBits, prefixLen uint8) {
	s.bitHeader2.SetBitHeader(prefixBits, prefixLen)
}

func (s *valuesSerializerHeaderFeHeaderFe) GetBitHeader2() (prefixBits PrefixBits, prefixLen uint8) {
	return s.bitHeader2.GetBitHeader()
}

func (s *valuesSerializerHeaderFeHeaderFe) Clone() *valuesSerializerHeaderFeHeaderFe {
	copy := *s
	return &copy
}

// valuesSerializerFe is a simple serializer for a single field element.
type valuesSerializerFe struct {
	fieldElementEndianness
}

func (s *valuesSerializerFe) DeserializeValues(input io.Reader) (bytesRead int, err error, fieldElement FieldElement) {
	bytesRead, err = fieldElement.Deserialize(input, s.byteOrder)
	return
}

func (s *valuesSerializerFe) SerializeValues(output io.Writer, fieldElement *FieldElement) (bytesWritten int, err error) {
	bytesWritten, err = fieldElement.Serialize(output, s.byteOrder)
	return
}

func (s *valuesSerializerFe) Clone() *valuesSerializerFe {
	return &valuesSerializerFe{fieldElementEndianness: s.fieldElementEndianness}
}

// valuesSerializerHeaderFe is a simple serializer for a single field element with sub-byte header
type valuesSerializerHeaderFe struct {
	fieldElementEndianness
	bitHeader
}

func (s *valuesSerializerHeaderFe) DeserializeValues(input io.Reader) (bytesRead int, err error, fieldElement FieldElement) {
	bytesRead, err = fieldElement.DeserializeWithPrefix(input, s.prefixBits, s.prefixLen, s.byteOrder)
	return
}

func (s *valuesSerializerHeaderFe) serializeValues(output io.Writer, fieldElement *FieldElement) (bytesWritten int, err error) {
	bytesWritten, err = fieldElement.SerializeWithPrefix(output, s.prefixBits, s.prefixLen, s.byteOrder)
	return
}

func (s *valuesSerializerHeaderFe) Clone() *valuesSerializerHeaderFe {
	copy := *s
	return &copy
}

// valuesSerializerFeCompressedBit is a simple serializer for a field element + 1 extra bit. The extra bit is squeezed into the field element.
type valuesSerializerFeCompressedBit struct {
	fieldElementEndianness
}

func (s *valuesSerializerFeCompressedBit) DeserializeValues(input io.Reader) (bytesRead int, err error, fieldElement FieldElement, bit bool) {
	var prefix PrefixBits
	bytesRead, prefix, err = fieldElement.DeserializeAndGetPrefix(input, 1, s.byteOrder)
	bit = (prefix == 0b1)
	return
}

func (s *valuesSerializerFeCompressedBit) SerializeValues(output io.Writer, fieldElement *FieldElement, bit bool) (bytesWritten int, err error) {
	var embeddedPrefix PrefixBits
	if bit {
		embeddedPrefix = PrefixBits(0b1)
	} else {
		embeddedPrefix = PrefixBits(0b0)
	}
	bytesWritten, err = fieldElement.SerializeWithPrefix(output, embeddedPrefix, 1, s.byteOrder)
	return
}

func (s *valuesSerializerFeCompressedBit) Clone() *valuesSerializerFeCompressedBit {
	return &valuesSerializerFeCompressedBit{fieldElementEndianness: s.fieldElementEndianness}
}

// TODO: Separate into different checks?

// checkPointSerializability verifies that the point is not a NaP or infinite. If subgroupCheck is set to true, also ensures that the point is in the p253-prime order subgroup.
// If everything is fine, returns nil. These correspond to the points that we usually want to serialize.
//
// Note: This function is is typically called before serializing (not for deserializing), where we do not have a trustLevel argument.
// This means that we always check whether the point is in the subgroup for any writes if the serializer is subgroup-only. Note for efficiency that this check is actually
// trivial if the type of point can only represent subgroup elements; we assume that this is the most common usage scenario.
func checkPointSerializability(point CurvePointPtrInterfaceRead, subgroupCheck bool) (err error) {
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
// They also do not allow headers of any sort.

// pointSerializerXY is a simple serializer that works by just writing / reading both the affine X and Y coordinates.
// If subgroupOnly is set to true, it will only work for points in the subgroup.
//
// NOTE: This cannot serialize points at infinity atm, even if subgroupRestriction is set to false
type pointSerializerXY struct {
	valuesSerializerFeFe
	subgroupRestriction // wraps a bool
}

func (s *pointSerializerXY) SerializeCurvePoint(output io.Writer, point CurvePointPtrInterfaceRead) (bytesWritten int, err error) {
	err = checkPointSerializability(point, s.IsSubgroupOnly())
	if err != nil {
		return
	}
	X, Y := point.XY_affine()
	bytesWritten, err = s.valuesSerializerFeFe.SerializeValues(output, &X, &Y)
	return
}

func (s *pointSerializerXY) DeserializeCurvePoint(input io.Reader, trustLevel IsPointTrusted, point CurvePointPtrInterfaceWrite) (bytesRead int, err error) {
	var X, Y FieldElement
	bytesRead, err, X, Y = s.DeserializeValues(input)
	if err != nil {
		return
	}
	if s.IsSubgroupOnly() || point.CanOnlyRepresentSubgroup() {
		var P Point_axtw_subgroup
		P, err = CurvePointFromXYAffine_subgroup(&X, &Y, trustLevel)
		if err != nil {
			return
		}
		point.SetFrom(&P)
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

func (s *pointSerializerXY) Clone() (ret *pointSerializerXY) {
	var sCopy pointSerializerXY = *s
	ret = &sCopy
	return
}

// pointSerializerXAndSignY is a Serialializer that serializes the affine X coordinate and the sign of the Y coordinate. (Note that the latter is never 0)
//
// More precisely, we write a 1 bit into the msb of the output (if interpreteed as 256bit-number) if the sign of Y is negative.
type pointSerializerXAndSignY struct {
	valuesSerializerFeCompressedBit
	subgroupRestriction
}

func (s *pointSerializerXAndSignY) SerializeCurvePoint(output io.Writer, point CurvePointPtrInterfaceRead) (bytesWritten int, err error) {
	err = checkPointSerializability(point, s.subgroupOnly)
	if err != nil {
		return
	}
	X, Y := point.XY_affine()
	var SignY bool = Y.Sign() < 0 // canot be == 0
	bytesWritten, err = s.SerializeValues(output, &X, SignY)
	return
}

func (s *pointSerializerXAndSignY) DeserializeCurvePoint(input io.Reader, trustLevel IsPointTrusted, point CurvePointPtrInterfaceWrite) (bytesRead int, err error) {
	var X FieldElement
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
		var P Point_axtw_subgroup
		P, err = CurvePointFromXAndSignY_subgroup(&X, signInt, trustLevel)
		if err != nil {
			return
		}
		point.SetFrom(&P)
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

func (s *pointSerializerXAndSignY) Clone() (ret *pointSerializerXAndSignY) {
	var sCopy pointSerializerXAndSignY = *s
	ret = &sCopy
	return
}

// pointSerializerYAndSignX serializes a point via its Y coordinate and the sign of X. (For X==0, we do not set the sign bit)
type pointSerializerYAndSignX struct {
	valuesSerializerFeCompressedBit
	subgroupRestriction
}

func (s *pointSerializerYAndSignX) SerializeCurvePoint(output io.Writer, point CurvePointPtrInterfaceRead) (bytesWritten int, err error) {
	err = checkPointSerializability(point, s.IsSubgroupOnly())
	if err != nil {
		return
	}
	X, Y := point.XY_affine()
	var SignX bool = X.Sign() < 0 // for X==0, we want the sign bit to be NOT set.
	bytesWritten, err = s.SerializeValues(output, &Y, SignX)
	return
}

func (s *pointSerializerYAndSignX) DeserializeCurvePoint(input io.Reader, trustLevel IsPointTrusted, point CurvePointPtrInterfaceWrite) (bytesRead int, err error) {
	var Y FieldElement
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

	// Note: CurvePointFromYAndSignX_* accept any sign for Y=+/-1. We need to correct this to ensure uniqueness of serialized representation.

	if s.subgroupOnly || point.CanOnlyRepresentSubgroup() {
		var P Point_axtw_subgroup
		P, err = CurvePointFromYAndSignX_subgroup(&Y, signInt, trustLevel)
		if err != nil {
			return
		}

		// This can only happen if Y = +1. In this case, we only accept signBit = false, as that's what we write when serializing.
		if P.IsNeutralElement() && signBit {
			err = bandersnatchErrors.ErrUnexpectedNegativeZero
			return
		}

		point.SetFrom(&P) // P is trusted at this point
	} else {
		var P Point_axtw_full
		P, err = CurvePointFromYAndSignX_full(&Y, signInt, trustLevel)
		if err != nil {
			return
		}

		// Special case: If Y = +/-1, we have X=0. In that case, we only accept signBit = false, as that's what we write when serializing.
		{
			var X FieldElement = P.X_decaf_affine()
			if X.IsZero() && signBit {
				err = bandersnatchErrors.ErrUnexpectedNegativeZero
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

/*
func (s *pointSerializerYAndSignX) WithEndianness(e binary.ByteOrder) (ret pointSerializerYAndSignX) {
	ret = *s.Clone()
	ret.SetEndianness(e)
	return
}
*/

/*
func (s *pointSerializerYAndSignX) WithSubgroupOnly(b bool) (ret pointSerializerYAndSignX) {
	ret = *s.Clone()
	ret.subgroupOnly = b
	return
}
*/

// pointSerializerXTimesSignY is a basic serializer that serializes via X * Sign(Y). Note that this only works for points in the subgroup, as the information of being in the subgroup is needed to deserialize uniquely.
type pointSerializerXTimesSignY struct {
	valuesSerializerFe
	subgroupOnly
}

func (s *pointSerializerXTimesSignY) SerializeCurvePoint(output io.Writer, point CurvePointPtrInterfaceRead) (bytesWritten int, err error) {
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
	bytesWritten, err = s.valuesSerializerFe.SerializeValues(output, &X)
	return
}

func (s *pointSerializerXTimesSignY) DeserializeCurvePoint(input io.Reader, trustLevel IsPointTrusted, point CurvePointPtrInterfaceWrite) (bytesRead int, err error) {
	var XSignY FieldElement
	bytesRead, err, XSignY = s.valuesSerializerFe.DeserializeValues(input)
	if err != nil {
		return
	}
	var P Point_axtw_subgroup
	P, err = CurvePointFromXTimesSignY_subgroup(&XSignY, trustLevel)
	if err != nil {
		return
	}
	point.SetFrom(&P)
	return
}

func (s *pointSerializerXTimesSignY) Clone() (ret *pointSerializerXTimesSignY) {
	var sCopy pointSerializerXTimesSignY = *s
	return &sCopy
}

/*
func (s *pointSerializerXTimesSignY) WithEndianness(e binary.ByteOrder) (ret pointSerializerXTimesSignY) {
	ret = *s.Clone()
	ret.fieldElementEndianness.SetEndianness(e)
	return
}
*/

/*
func (s *pointSerializerXTimesSignY) WithSubgroupOnly(b bool) (ret pointSerializerXTimesSignY) {
	ret = *s.Clone()
	if !b {
		panic("bandersnatch / serialization: point serialization via X * Sign(Y) only works for subgroup. Trying to construct a serializer without that restriction is a bug.")
	}
	return
}
*/

// pointSerializerXYTimesSignY is a serializer that used X*Sign(Y), Y*Sign(Y). This serializer only works for subgroup elements, since
type pointSerializerXYTimesSignY struct {
	valuesSerializerFeFe
	subgroupOnly
}

func (s *pointSerializerXYTimesSignY) SerializeCurvePoint(output io.Writer, point CurvePointPtrInterfaceRead) (bytesWritten int, err error) {
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
	bytesWritten, err = s.valuesSerializerFeFe.SerializeValues(output, &X, &Y)
	return
}

func (s *pointSerializerXYTimesSignY) DeserializeCurvePoint(input io.Reader, trustLevel IsPointTrusted, point CurvePointPtrInterfaceWrite) (bytesRead int, err error) {
	var XSignY, YSignY FieldElement
	bytesRead, err, XSignY, YSignY = s.valuesSerializerFeFe.DeserializeValues(input)
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

func (s *pointSerializerXYTimesSignY) Clone() (ret *pointSerializerXYTimesSignY) {
	var sCopy pointSerializerXYTimesSignY = *s
	ret = &sCopy
	return
}

/*
type headerSerializer struct {
	// headerAll           []byte
	headerPerCurvePoint []byte
	// footer              []byte
	// headerAllReader     func(input io.Reader) (bytes_read int, err error, curvePointsToRead int, extra interface{})
	// headerPointReader   func(input io.Reader) (bytes_read int, err error, extra interface{})
}
*/

/*
func (hs *headerSerializer) clone() (ret headerSerializer) {
	if hs.headerPerCurvePoint == nil {
		ret.headerPerCurvePoint = nil
	} else {
		ret.headerPerCurvePoint = make([]byte, len(hs.headerPerCurvePoint))
		copy(ret.headerPerCurvePoint, hs.headerPerCurvePoint)
	}
	return
}
*/

/*
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

*/
