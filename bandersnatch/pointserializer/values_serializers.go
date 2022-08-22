package pointserializer

import (
	"io"

	"github.com/GottfriedHerold/Bandersnatch/bandersnatch/bandersnatchErrors"
	"github.com/GottfriedHerold/Bandersnatch/bandersnatch/common"
	"github.com/GottfriedHerold/Bandersnatch/bandersnatch/errorsWithData"
	"github.com/GottfriedHerold/Bandersnatch/bandersnatch/fieldElements"
	"github.com/GottfriedHerold/Bandersnatch/internal/utils"
)

// This file contains the serializers that are responsible for (de)serializing (uninterpreted) sequences of field elements, bits and sub-byte headers.
// We call these valuesSerializers, values referring to bits and field elements.

// Note that sub-byte headers are fixed constants (stored in the serializer object) written/consumed upon serializing.
// They are not considered values and are invisible to the SerializeValues and DeserializeValues methods.

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
// Validate()
// RecognizedParameters() []string
// OutputLength() int32
//
// The parameter types of DeserializeValues and SerializeValues need to match. The return type of Clone() is the same as the pointer receiver.
// Go's interfaces cannot express this (even with generics). We use reflection to express this.
// (note that reflection is mostly used to unify the testing code -- the actual usage does not use reflection, since the code paths for the different
// realisations of this "interface" are separate anyway)

// valuesSerializer is the part of the interface satisfied by all valuesSerializer that is actually expressible via Go interfaces.
type valuesSerializer interface {
	Validate()
	RecognizedParameters() []string
	HasParameter(parameterName string) bool
	OutputLength() int32
}

// ValuesSerializers are serializers for field elements and bits. These have the following "interface", defined on pointer receivers
//
// DeserializeValues(input io.Reader) (bytesRead int, err error, ...)
// SerializeValues(output io.Writer, ...) (bytesWritten int, err error)
// Clone() receiver(pointer) [NOTE: Returned type is concrete, not interface type]
// Validate()
// RecognizedParameters() []string
// OutputLength() int32
//
// The parameter types of DeserializeValues and SerializeValues need to match. The return type of Clone() is the same as the pointer receiver.
// Go's interfaces cannot express this. We use reflection.

// updateReadError and updateWriteError are helper functions to avoid code duplication:

// updateReadError is used to update the metadata contained in the error;
// bytesRead is passed via pointer because this function is called via defer and arguments to deferred functions
// get evaluated at the time of defer, not when the function is actually run.
func updateReadError(errPtr *bandersnatchErrors.DeserializationError, bytesReadPtr *int, expectToRead int) {
	if *errPtr != nil {
		bytesRead := *bytesReadPtr
		*errPtr = errorsWithData.IncludeGuaranteedParametersInError[bandersnatchErrors.ReadErrorData](*errPtr,
			FIELDNAME_PARTIAL_READ, bytesRead != 0 && bytesRead != expectToRead,
			// bandersnatchErrors.BYTES_READ, bytesRead, // NOTE: This might change
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
			FIELDNAME_PARTIAL_WRITE, bytesWritten != 0 && bytesWritten != expectToWrite,
			// bandersnatchErrors.BYTES_WRITTEN, bytesWritten,
		)
	}
}

//*******************************************************************************************************************************

// Note: These are internal structs that only serve to modularize the (de)serializers.
// The methods are exported in order to make them callable via reflection.

// valuesSerializerFeFe is a simple serializer for a pair of field elements
type valuesSerializerFeFe struct {
	fieldElementEndianness // meaning the endianness for fieldElementSerialization
}

// DeserializeValues reads from input and returns values.
//
// For valuesSerializerFeFe, it returns 2 field elements
//
// Note the err is returned as second rather than last return value. This may trigger linters warnings.
// This choice is because it simplifies some reflection-using code using these methods, which is written for methods returning (int, error, ...) - tuples.
// Having the unknown-length part at the end makes things simpler.
func (s *valuesSerializerFeFe) DeserializeValues(input io.Reader) (bytesRead int, err bandersnatchErrors.DeserializationError, fieldElement1, fieldElement2 fieldElements.FieldElement) {
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

// SerializeValues writes the given values (and possibly header) to output.
//
// For valuesSerializerFeFe, it writes 2 field elements.
func (s *valuesSerializerFeFe) SerializeValues(output io.Writer, fieldElement1, fieldElement2 *fieldElements.FieldElement) (bytesWritten int, err bandersnatchErrors.SerializationError) {
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

// Clone returns a copy of the valuesSerializer (of the same type as the receiver) as a pointer.
func (s *valuesSerializerFeFe) Clone() *valuesSerializerFeFe {
	return &valuesSerializerFeFe{fieldElementEndianness: s.fieldElementEndianness}
}

// Validate is not needed due to struct embedding, but added for consistency.

// Validate checks the internal parameter of the valuesSerializer for consistency.
func (s *valuesSerializerFeFe) Validate() {
	s.fieldElementEndianness.Validate()
}

// OutputLength returns the number of bytes written/read by this valuesSerialzer.
//
// For valuesSerializerFeFe, it always outputs 64
func (s *valuesSerializerFeFe) OutputLength() int32 { return 64 }

// RecognizedParameters outputs a slice of strings listing valid parameters that can be queried/set via
// the hasParameter / makeCopyWithParameters generic methods.
//
// For a valuesSerializerFeFe, contains "Endianness" (the endianness used to (de)serialize FieldElements)
func (s *valuesSerializerFeFe) RecognizedParameters() []string {
	return []string{"Endianness"}
}

func (s *valuesSerializerFeFe) HasParameter(parameterName string) bool {
	return normalizeParameter(parameterName) == normalizeParameter("Endianness")
}

//*******************************************************************************************************************************

// valuesSerializerHeaderFeHeaderFe is a serializer for a pair of field elements, where each of the two field elements has a prefix (of sub-byte length) contained in the
// msbs. These prefixes are fixed headers for the serializer and not part of the individual output/input field elements.
type valuesSerializerHeaderFeHeaderFe struct {
	fieldElementEndianness
	bitHeader  //bitHeader for the first field element. This is embedded, so we don't have to forward setters/getters.
	bitHeader2 bitHeader
}

// DeserializeValues reads from input and returns values.
//
// For valuesSerializerHeaderFeHeaderFe, it returns 2 field elements.
//
// Note the err is returned as second rather than last return value. This may trigger linters warnings.
// This choice is because it simplifies some reflection-using code using these methods, which is written for methods returning (int, error, ...) - tuples.
// Having the unknown-length part at the end makes things simpler.
func (s *valuesSerializerHeaderFeHeaderFe) DeserializeValues(input io.Reader) (bytesRead int, err bandersnatchErrors.DeserializationError, fieldElement1, fieldElement2 fieldElements.FieldElement) {
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

// SerializeValues writes the given values (and possibly header) to output.
//
// For valuesSerializerHeaderFeHeaderFe, it writes 2 field elements and headers.
func (s *valuesSerializerHeaderFeHeaderFe) SerializeValues(output io.Writer, fieldElement1, fieldElement2 *fieldElements.FieldElement) (bytesWritten int, err bandersnatchErrors.SerializationError) {
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

// SetBitHeader2 is a setter for the second bitHeader. We cannot use struct embedding here, because we have 2 separate BitHeaders.
func (s *valuesSerializerHeaderFeHeaderFe) SetBitHeader2(bh bitHeader) {
	s.bitHeader2.SetBitHeaderFromBitHeader(bh)
}

// GetBitHeader2 is a getter for the second bitHeader. We cannot use struct embedding here, because we have 2 separate BitHeaders.
func (s *valuesSerializerHeaderFeHeaderFe) GetBitHeader2() bitHeader {
	return s.bitHeader2.GetBitHeader()
}

// Clone returns a copy of the valuesSerializer (of the same type as the receiver) as a pointer.
func (s *valuesSerializerHeaderFeHeaderFe) Clone() *valuesSerializerHeaderFeHeaderFe {
	copy := *s
	return &copy
}

// Validate checks the internal parameter of the valuesSerializer for consistency.
func (s *valuesSerializerHeaderFeHeaderFe) Validate() {
	s.fieldElementEndianness.Validate()
	s.bitHeader.Validate()
	s.bitHeader2.Validate()
}

// RecognizedParameters outputs a slice of strings listing valid parameters that can be queried/set via
// the hasParameter / makeCopyWithParameters generic methods.
//
// For a valuesSerializerHeaderFeHeaderFe, the list contains "Endianness" (the endianness used to (de)serialize FieldElements),
// "BitHeader", "BitHeader2" (the Headers used for the first and second field element)
func (s *valuesSerializerHeaderFeHeaderFe) RecognizedParameters() []string {
	return []string{"Endianness", "BitHeader", "BitHeader2"}
}

func (s *valuesSerializerHeaderFeHeaderFe) HasParameter(parameterName string) bool {
	return utils.ElementInList(parameterName, s.RecognizedParameters(), normalizeParameter)
}

// OutputLength returns the number of bytes written/read by this valuesSerialzer.
//
// For valuesSerializerHeaderFeHeaderFe, it always outputs 64
func (s *valuesSerializerHeaderFeHeaderFe) OutputLength() int32 { return 64 }

//*******************************************************************************************************************************

// valuesSerializerFe is a simple serializer for a single field element.
type valuesSerializerFe struct {
	fieldElementEndianness
}

// DeserializeValues reads from input and returns values.
//
// For valuesSerializerFe, it returns 1 field element.
//
// Note the err is returned as second rather than last return value. This may trigger linters warnings.
// This choice is because it simplifies some reflection-using code using these methods, which is written for methods returning (int, error, ...) - tuples.
// Having the unknown-length part at the end makes things simpler.
func (s *valuesSerializerFe) DeserializeValues(input io.Reader) (bytesRead int, err bandersnatchErrors.DeserializationError, fieldElement fieldElements.FieldElement) {
	// no need for defer updateReadError(...)
	bytesRead, err = fieldElement.Deserialize(input, s.fieldElementEndianness)
	return
}

// SerializeValues writes the given values (and possibly header) to output.
//
// For valuesSerializerFe, it writes 1field element.
func (s *valuesSerializerFe) SerializeValues(output io.Writer, fieldElement *fieldElements.FieldElement) (bytesWritten int, err bandersnatchErrors.SerializationError) {
	// no need for defer updateWriteError(...)
	bytesWritten, err = fieldElement.Serialize(output, s.fieldElementEndianness)
	return
}

// Clone returns a copy of the valuesSerializer (of the same type as the receiver) as a pointer.
func (s *valuesSerializerFe) Clone() *valuesSerializerFe {
	return &valuesSerializerFe{fieldElementEndianness: s.fieldElementEndianness}
}

// Not needed to due struct embedding, but added for consistency.

// Validate checks the internal parameter of the valuesSerializer for consistency.
func (s *valuesSerializerFe) Validate() {
	s.fieldElementEndianness.Validate()
}

// OutputLength returns the number of bytes written/read by this valuesSerialzer.
//
// For valuesSerializerFe, it always outputs 32
func (s *valuesSerializerFe) OutputLength() int32 { return 32 }

// RecognizedParameters outputs a slice of strings listing valid parameters that can be queried/set via
// the hasParameter / makeCopyWithParameters generic methods.
//
// For a valuesSerializerFe, the list contains "Endianness" (the endianness used to (de)serialize FieldElements)
func (s *valuesSerializerFe) RecognizedParameters() []string {
	return []string{"Endianness"}
}

func (s *valuesSerializerFe) HasParameter(parameterName string) bool {
	return utils.ElementInList(parameterName, s.RecognizedParameters(), normalizeParameter)
}

//*******************************************************************************************************************************

// valuesSerializerHeaderFe is a simple serializer for a single field element with sub-byte header
type valuesSerializerHeaderFe struct {
	fieldElementEndianness
	bitHeader
}

// DeserializeValues reads from input and returns values.
//
// For valuesSerializerHeaderFe, it returns 1 field element.
//
// Note the err is returned as second rather than last return value. This may trigger linters warnings.
// This choice is because it simplifies some reflection-using code using these methods, which is written for methods returning (int, error, ...) - tuples.
// Having the unknown-length part at the end makes things simpler.
func (s *valuesSerializerHeaderFe) DeserializeValues(input io.Reader) (bytesRead int, err bandersnatchErrors.DeserializationError, fieldElement fieldElements.FieldElement) {
	bytesRead, err = fieldElement.DeserializeWithPrefix(input, s.bitHeader, s.fieldElementEndianness)
	return
}

// SerializeValues writes the given values (and possibly header) to output.
//
// For valuesSerializerHeaderFe, it writes 1 field element with header.
func (s *valuesSerializerHeaderFe) SerializeValues(output io.Writer, fieldElement *fieldElements.FieldElement) (bytesWritten int, err bandersnatchErrors.SerializationError) {
	bytesWritten, err = fieldElement.SerializeWithPrefix(output, s.bitHeader, s.fieldElementEndianness)
	return
}

// Clone returns a copy of the valuesSerializer (of the same type as the receiver) as a pointer.
func (s *valuesSerializerHeaderFe) Clone() *valuesSerializerHeaderFe {
	copy := *s
	return &copy
}

// Validate checks the internal parameter of the valuesSerializer for consistency.
func (s *valuesSerializerHeaderFe) Validate() {
	s.fieldElementEndianness.Validate()
	s.bitHeader.Validate()
}

// OutputLength returns the number of bytes written/read by this valuesSerialzer.
//
// For valuesSerializerHeaderFe, it always outputs 32
func (s *valuesSerializerHeaderFe) OutputLength() int32 { return 32 }

// RecognizedParameters outputs a slice of strings listing valid parameters that can be queried/set via
// the hasParameter / makeCopyWithParameters generic methods.
//
// For a valuesSerializerHeaderFe, the list contains "Endianness" (the endianness used to (de)serialize FieldElements) and "BitHeader" (the sub-byte header)
func (s *valuesSerializerHeaderFe) RecognizedParameters() []string {
	return []string{"Endianness", "BitHeader"}
}

func (s *valuesSerializerHeaderFe) HasParameter(parameterName string) bool {
	return utils.ElementInList(parameterName, s.RecognizedParameters(), normalizeParameter)
}

//*******************************************************************************************************************************

// valuesSerializerFeCompressedBit is a simple serializer for a field element + 1 extra bit. The extra bit is squeezed into the field element.
type valuesSerializerFeCompressedBit struct {
	fieldElementEndianness
}

// DeserializeValues reads from input and returns values.
//
// For valuesSerializerFeCompressedBit, it returns 1 field element and 1 Bit.
//
// Note the err is returned as second rather than last return value. This may trigger linters warnings.
// This choice is because it simplifies some reflection-using code using these methods, which is written for methods returning (int, error, ...) - tuples.
// Having the unknown-length part at the end makes things simpler.
func (s *valuesSerializerFeCompressedBit) DeserializeValues(input io.Reader) (bytesRead int, err bandersnatchErrors.DeserializationError, fieldElement fieldElements.FieldElement, bit bool) {
	var prefix common.PrefixBits
	bytesRead, prefix, err = fieldElement.DeserializeAndGetPrefix(input, 1, s.fieldElementEndianness)
	bit = (prefix == 0b1)
	return
}

// SerializeValues writes the given values (and possibly header) to output.
//
// For valuesSerializerFeFe, it writes 1 field element and 1 Bit.
func (s *valuesSerializerFeCompressedBit) SerializeValues(output io.Writer, fieldElement *fieldElements.FieldElement, bit bool) (bytesWritten int, err bandersnatchErrors.SerializationError) {
	var embeddedPrefix common.BitHeader
	if bit {
		embeddedPrefix = common.MakeBitHeader(common.PrefixBits(0b1), 1)
	} else {
		embeddedPrefix = common.MakeBitHeader(common.PrefixBits(0b0), 1)
	}
	bytesWritten, err = fieldElement.SerializeWithPrefix(output, embeddedPrefix, s.fieldElementEndianness)
	return
}

// Clone returns a copy of the valuesSerializer (of the same type as the receiver) as a pointer.
func (s *valuesSerializerFeCompressedBit) Clone() *valuesSerializerFeCompressedBit {
	return &valuesSerializerFeCompressedBit{fieldElementEndianness: s.fieldElementEndianness}
}

// Not needed due to struct embedding, but added for consistency.

// Validate checks the internal parameter of the valuesSerializer for consistency.
func (s *valuesSerializerFeCompressedBit) Validate() {
	s.fieldElementEndianness.Validate()
}

// OutputLength returns the number of bytes written/read by this valuesSerialzer.
//
// For valuesSerializerFeCompressedBit, it always outputs 32
func (s *valuesSerializerFeCompressedBit) OutputLength() int32 { return 32 }

// RecognizedParameters outputs a slice of strings listing valid parameters that can be queried/set via
// the hasParameter / makeCopyWithParameters generic methods.
//
// For a valuesSerializerFeCompressedBit, the list contains "Endianness" (the endianness used to (de)serialize FieldElements)
func (s *valuesSerializerFeCompressedBit) RecognizedParameters() []string {
	return []string{"Endianness"}
}

func (s *valuesSerializerFeCompressedBit) HasParameter(parameterName string) bool {
	return utils.ElementInList(parameterName, s.RecognizedParameters(), normalizeParameter)
}
