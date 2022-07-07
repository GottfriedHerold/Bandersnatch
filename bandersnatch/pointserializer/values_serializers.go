package pointserializer

import (
	"io"

	"github.com/GottfriedHerold/Bandersnatch/bandersnatch"
	"github.com/GottfriedHerold/Bandersnatch/bandersnatch/bandersnatchErrors"
	"github.com/GottfriedHerold/Bandersnatch/bandersnatch/common"
	"github.com/GottfriedHerold/Bandersnatch/bandersnatch/errorsWithData"
)

type FieldElementEndianness = common.FieldElementEndianness

// aliases to make struct-embedding non-exported.
type fieldElementEndianness = FieldElementEndianness
type bitHeader = common.BitHeader

// Due to insufficency of generics, we separate our serializers depending on whether the internal object that actually gets serialized consists of
// a field element, two field element, field element+bit etc.
// These (de)serializiers all have serializeValues and DeserializeValues methods, which differ in their arguments.

// ValuesSerializers are serializers for field elements and bits. These have the following "interface", defined on pointer receivers
//
// DeserializeValues(input io.Reader) (bytesRead int, err error, ...)
// SerializeValues(output io.Writer, ...) (bytesWritten int, err error)
// Clone() receiver(pointer) [NOTE: Returned type is concrete, not interface type]
// VerifyPtr
//
// The parameter types of DeserializeValues and SerializeValues need to match. The return type of Clone() is the same as the pointer receiver.
// Go's interfaces cannot express this. We use reflection.

// Note: These are internal structs that only serve to modularize the (de)serializers.
// The methods are exported in order to make them callable via reflection.

// valuesSerializerFeFe is a simple serializer for a pair of field elements
type valuesSerializerFeFe struct {
	fieldElementEndianness // meaning the endianness for fieldElementSerialization
}

// updateReadError is used to update the metadata contained in the error;
// bytesRead is passed via pointer because this function is called via defer and arguments to deferred functions
// get evaluated at the time of defer, not when the function is actually run.
func updateReadError(errPtr *bandersnatchErrors.DeserializationError, bytesReadPtr *int, expectToRead int) {
	if *errPtr != nil {
		bytesRead := *bytesReadPtr
		*errPtr = errorsWithData.IncludeGuaranteedParametersInError[bandersnatchErrors.ReadErrorData](*errPtr,
			PARTIAL_READ, bytesRead != 0 && bytesRead != expectToRead,
			bandersnatchErrors.BYTES_READ, bytesRead, // NOTE: This might change
		)
	}
}

// updateWriteError is used to update the metadata contained in the error;
// bytesWritten is passed via pointer because this function is called via defer and arguments to deferred functions
// get evaluated at the time of defer, not when the functon is run.
func updateWriteError(errPtr *bandersnatchErrors.SerializationError, bytesWrittenPtr *int, expectToWrite int) {
	if *errPtr != nil {
		bytesWritten := *bytesWrittenPtr
		*errPtr = errorsWithData.IncludeGuaranteedParametersInError[bandersnatchErrors.WriteErrorData](*errPtr,
			PARTIAL_WRITE, bytesWritten != 0 && bytesWritten != expectToWrite,
			bandersnatchErrors.BYTES_WRITTEN, bytesWritten,
		)
	}
}

// NOTE: DeserializeValues has err as second (rather than last) parameter, triggering warnings from static style-checkers.
// This choice is because it simplifies some reflection-using code using these methods, which is written for methods returning (int, error, ...) - tuples.
// Having the unknown-length part at the end makes things simpler.

func (s *valuesSerializerFeFe) DeserializeValues(input io.Reader) (bytesRead int, err bandersnatchErrors.DeserializationError, fieldElement1, fieldElement2 bandersnatch.FieldElement) {
	defer updateReadError(&err, &bytesRead, int(s.OutputLength()))
	bytesRead, err = fieldElement1.Deserialize(input, s.fieldElementEndianness)
	// Note: This aborts on ErrNonNormalizedDeserialization
	if err != nil {
		return
	}
	bytesJustRead, err := fieldElement2.Deserialize(input, s.fieldElementEndianness)
	bytesRead += bytesJustRead
	// We treat EOF like UnexpectedEOF at this point. The reason is that we treat the PAIR of field elements as a unit.
	bandersnatchErrors.UnexpectEOF2(&err)
	return
}

func (s *valuesSerializerFeFe) SerializeValues(output io.Writer, fieldElement1, fieldElement2 *bandersnatch.FieldElement) (bytesWritten int, err bandersnatchErrors.SerializationError) {
	defer updateWriteError(&err, &bytesWritten, int(s.OutputLength()))

	bytesWritten, err = fieldElement1.Serialize(output, s.fieldElementEndianness)
	if err != nil {
		return
	}
	bytesJustWritten, err := fieldElement2.Serialize(output, s.fieldElementEndianness)
	bytesWritten += bytesJustWritten
	// We treat EOF like UnexpectedEOF at this point. The reason is that we treat the PAIR of field elements as a unit.
	bandersnatchErrors.UnexpectEOF2(&err)
	return
}

func (s *valuesSerializerFeFe) Clone() *valuesSerializerFeFe {
	return &valuesSerializerFeFe{fieldElementEndianness: s.fieldElementEndianness}
}

// not needed (due to struct embedinng, but added for consistency)
func (s *valuesSerializerFeFe) Validate() {
	s.fieldElementEndianness.Validate()
}

func (s *valuesSerializerFeFe) OutputLength() int32 { return 64 }

// valuesSerializerHeaderFeHeaderFe is a serializer for a pair of field elements, where each of the two field elements has a prefix (of sub-byte length) contained in the
// msbs. These prefixes are fixed headers for the serializer and not part of the individual output/input field elements.
type valuesSerializerHeaderFeHeaderFe struct {
	fieldElementEndianness
	bitHeader  //bitHeader for the first field element. This is embedded, so we don't have to forward setters/getters.
	bitHeader2 bitHeader
}

func (s *valuesSerializerHeaderFeHeaderFe) DeserializeValues(input io.Reader) (bytesRead int, err bandersnatchErrors.DeserializationError, fieldElement1, fieldElement2 bandersnatch.FieldElement) {
	defer updateReadError(&err, &bytesRead, int(s.OutputLength()))

	bytesRead, err = fieldElement1.DeserializeWithPrefix(input, s.bitHeader, s.fieldElementEndianness)
	// Note: This aborts on ErrNonNormalizedDeserialization
	if err != nil {
		return
	}
	bytesJustRead, err := fieldElement2.DeserializeWithPrefix(input, s.bitHeader2, s.fieldElementEndianness)
	bytesRead += bytesJustRead
	// We treat EOF like UnexpectedEOF at this point. The reason is that we treat the PAIR of field elements as a unit.
	bandersnatchErrors.UnexpectEOF2(&err)
	return
}

func (s *valuesSerializerHeaderFeHeaderFe) SerializeValues(output io.Writer, fieldElement1, fieldElement2 *bandersnatch.FieldElement) (bytesWritten int, err bandersnatchErrors.SerializationError) {
	defer updateWriteError(&err, &bytesWritten, int(s.OutputLength()))

	bytesWritten, err = fieldElement1.SerializeWithPrefix(output, s.bitHeader, s.fieldElementEndianness)
	if err != nil {
		return
	}

	bytesJustWritten, err := fieldElement2.SerializeWithPrefix(output, s.bitHeader2, s.fieldElementEndianness)
	bytesWritten += bytesJustWritten
	// We treat EOF like UnexpectedEOF at this point. The reason is that we treat the PAIR of field elements as a unit.
	bandersnatchErrors.UnexpectEOF2(&err)
	return
}

func (s *valuesSerializerHeaderFeHeaderFe) SetBitHeader2(bh bitHeader) {
	s.bitHeader2.SetBitHeaderFromBitHeader(bh)
}

func (s *valuesSerializerHeaderFeHeaderFe) GetBitHeader2() bitHeader {
	return s.bitHeader2.GetBitHeader()
}

func (s *valuesSerializerHeaderFeHeaderFe) Clone() *valuesSerializerHeaderFeHeaderFe {
	copy := *s
	return &copy
}

func (s *valuesSerializerHeaderFeHeaderFe) Validate() {
	s.fieldElementEndianness.Validate()
	s.bitHeader.Validate()
	s.bitHeader2.Validate()
}

func (s *valuesSerializerHeaderFeHeaderFe) OutputLength() int32 { return 64 }

// valuesSerializerFe is a simple serializer for a single field element.
type valuesSerializerFe struct {
	fieldElementEndianness
}

func (s *valuesSerializerFe) DeserializeValues(input io.Reader) (bytesRead int, err bandersnatchErrors.DeserializationError, fieldElement bandersnatch.FieldElement) {
	// no need for defer updateReadError(...)
	bytesRead, err = fieldElement.Deserialize(input, s.fieldElementEndianness)
	return
}

func (s *valuesSerializerFe) SerializeValues(output io.Writer, fieldElement *bandersnatch.FieldElement) (bytesWritten int, err bandersnatchErrors.SerializationError) {
	// no need for defer updateWriteError(...)
	bytesWritten, err = fieldElement.Serialize(output, s.fieldElementEndianness)
	return
}

func (s *valuesSerializerFe) Clone() *valuesSerializerFe {
	return &valuesSerializerFe{fieldElementEndianness: s.fieldElementEndianness}
}

// Technically not needed due to struct embedding.

func (s *valuesSerializerFe) Validate() {
	s.fieldElementEndianness.Validate()
}

func (s *valuesSerializerFe) OutputLength() int32 { return 32 }

// valuesSerializerHeaderFe is a simple serializer for a single field element with sub-byte header
type valuesSerializerHeaderFe struct {
	fieldElementEndianness
	bitHeader
}

func (s *valuesSerializerHeaderFe) DeserializeValues(input io.Reader) (bytesRead int, err bandersnatchErrors.DeserializationError, fieldElement bandersnatch.FieldElement) {
	bytesRead, err = fieldElement.DeserializeWithPrefix(input, s.bitHeader, s.fieldElementEndianness)
	return
}

func (s *valuesSerializerHeaderFe) SerializeValues(output io.Writer, fieldElement *bandersnatch.FieldElement) (bytesWritten int, err bandersnatchErrors.SerializationError) {
	bytesWritten, err = fieldElement.SerializeWithPrefix(output, s.bitHeader, s.fieldElementEndianness)
	return
}

func (s *valuesSerializerHeaderFe) Clone() *valuesSerializerHeaderFe {
	copy := *s
	return &copy
}

func (s *valuesSerializerHeaderFe) Validate() {
	s.fieldElementEndianness.Validate()
	s.bitHeader.Validate()
}

func (s *valuesSerializerHeaderFe) OutputLength() int32 { return 32 }

// valuesSerializerFeCompressedBit is a simple serializer for a field element + 1 extra bit. The extra bit is squeezed into the field element.
type valuesSerializerFeCompressedBit struct {
	fieldElementEndianness
}

func (s *valuesSerializerFeCompressedBit) DeserializeValues(input io.Reader) (bytesRead int, err bandersnatchErrors.DeserializationError, fieldElement bandersnatch.FieldElement, bit bool) {
	var prefix common.PrefixBits
	bytesRead, prefix, err = fieldElement.DeserializeAndGetPrefix(input, 1, s.fieldElementEndianness)
	bit = (prefix == 0b1)
	return
}

func (s *valuesSerializerFeCompressedBit) SerializeValues(output io.Writer, fieldElement *bandersnatch.FieldElement, bit bool) (bytesWritten int, err bandersnatchErrors.SerializationError) {
	var embeddedPrefix common.BitHeader
	if bit {
		embeddedPrefix = common.MakeBitHeader(common.PrefixBits(0b1), 1)
	} else {
		embeddedPrefix = common.MakeBitHeader(common.PrefixBits(0b0), 1)
	}
	bytesWritten, err = fieldElement.SerializeWithPrefix(output, embeddedPrefix, s.fieldElementEndianness)
	return
}

func (s *valuesSerializerFeCompressedBit) Clone() *valuesSerializerFeCompressedBit {
	return &valuesSerializerFeCompressedBit{fieldElementEndianness: s.fieldElementEndianness}
}

func (s *valuesSerializerFeCompressedBit) Validate() {
	s.fieldElementEndianness.Validate()
}

func (s *valuesSerializerFeCompressedBit) OutputLength() int32 { return 32 }
