package pointserializer

import (
	"io"

	"github.com/GottfriedHerold/Bandersnatch/bandersnatch/bandersnatchErrors"
	"github.com/GottfriedHerold/Bandersnatch/bandersnatch/common"
	"github.com/GottfriedHerold/Bandersnatch/bandersnatch/errorsWithData"
	"github.com/GottfriedHerold/Bandersnatch/bandersnatch/fieldElements"
	"github.com/GottfriedHerold/Bandersnatch/internal/utils"
)

// This file is part of the serialization-for-curve-points package.
// This package defines types that act as (de)serializers. These types hold metadata (such as e.g. endianness) about the serialization format.
// (De)serializers then have methods that are called with the actual curve point(s) as arguments to (de)serialize them.

// This file contains the serializers that are responsible for (de)serializing (uninterpreted) sequences of field elements, bits and sub-byte headers.
// We call these valuesSerializers, values referring to bits and field elements.

// Note that sub-byte headers are fixed constants (stored in the serializer object) written/consumed upon (de)serializing.
// They are not considered values and are invisible to the SerializeValues and DeserializeValues methods.

type FieldElementEndianness = common.FieldElementEndianness

// aliases to make struct-embedding non-exported.
type fieldElementEndianness = FieldElementEndianness
type bitHeader = common.BitHeader

// Due to insufficency of generics, we separate our serializers depending on whether the internal object that actually gets serialized consists of
// a field element, two field element, field element+bit etc.
// These (de)serializiers all have serializeValues and DeserializeValues methods, which differ in their arguments (including number of arguments)

// TODO: Move this to *_test.go? I prefer it here for clarity.

// valuesSerializer is the part of the interface satisfied by all valuesSerializer that is actually expressible via Go interfaces.
// It is only used in testing to unify some tests.
type valuesSerializer interface {
	Validate()                              // performs a (possibly trivial) internal consistency check. It panics on failure. This is called after all setters by embedding structs.
	RecognizedParameters() []string         // returns a list of parameters that can be set/queried for this particular serializer (type). For all our realizations, can be called with nil receivers.
	HasParameter(parameterName string) bool // Checks whether a given parameter can be set/queried for this particular serializer (type). For all our realizations, can be called with nil receivers.
	OutputLength() int32                    // Returns that number of bytes that this valuesSerializers will read/write when calling SerializeValues or DeserializeValues. For all our realizations, can be called with nil receivers.
	// Functions not well-expressible via interface:
	// Clone() PointerReceiver -- Returns an independent copy of itself.
	// SerializeValues(output io.Writer, [...]) (bytesWritten int, err bandersnatchErrors.SerializationError)
	// DeserializeValues(input io.Reader) (bytesRead int, err bandersnatchErrors.DeserializationError, [...])
}

// The [...] parameter types and number of DeserializeValues and SerializeValues need to match.
// The return type of Clone() is the same as the pointer receiver.
// Go's interfaces cannot express this (even with generics). We use reflection to express this.
// Note that reflection is mostly used to unify the testing code -- the actual usage does not use reflection, since the code paths for the different
// realisations of this "interface" are separate anyway.

// We could add Clone here if we made the valuesSerializer interface generic, but it's only used in testing anyway and we already have a Clonable[Foo] generic.

// **********************************************************************************************************************************
// Utility functions:

// updateReadError and updateWriteError are helper functions to avoid code duplication:

// updateReadError is used to update "PartialRead" in the metadata contained in the error;
// bytesRead is passed via pointer because this function is called via defer and arguments to deferred functions
// get evaluated at the time of defer, not when the function is actually run.
func updateReadError(errPtr *bandersnatchErrors.DeserializationError, bytesReadPtr *int, expectToRead int) {
	if *errPtr != nil {
		var bytesRead int = *bytesReadPtr
		*errPtr = errorsWithData.IncludeGuaranteedParametersInError[bandersnatchErrors.ReadErrorData](*errPtr,
			FIELDNAME_PARTIAL_READ, bytesRead != 0 && bytesRead != expectToRead,
			// NOTE: We do not update the "BytesRead" metadata in the error. This is intentional: "BytesRead" refers to the failing sub-call.
		)
	}
}

// updateWriteError is used to update the "PartialWrite" metadata contained in the error;
// bytesWritten is passed via pointer because this function is called via defer and arguments to deferred functions
// get evaluated at the time of defer, not when the functon is run.
func updateWriteError(errPtr *bandersnatchErrors.SerializationError, bytesWrittenPtr *int, expectToWrite int) {
	if *errPtr != nil {
		var bytesWritten int = *bytesWrittenPtr
		*errPtr = errorsWithData.IncludeGuaranteedParametersInError[bandersnatchErrors.WriteErrorData](*errPtr,
			FIELDNAME_PARTIAL_WRITE, (bytesWritten != 0) && (bytesWritten != expectToWrite),
			// NOTE: We do not update the "BytesWritten" metadata in the error. This is intentional: "BytesWritten" refers to the failing sub-call.
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
	defer updateReadError(&err, &bytesRead, int(s.OutputLength())) // ensures PartialRead is correctly set on error.
	bytesRead, err = fieldElement1.Deserialize(input, s.fieldElementEndianness)
	// Note: This aborts on ErrNonNormalizedDeserialization. I.e. if the first read field element is not in normalized form, we don't even read the second.
	if err != nil {
		return
	}
	bytesJustRead, err := fieldElement2.Deserialize(input, s.fieldElementEndianness)
	bytesRead += bytesJustRead
	// We treat EOF like UnexpectedEOF at this point. The reason is that we treat the PAIR of field elements as a unit.
	bandersnatchErrors.UnexpectEOF2(&err) // transforms EOF -> UnexpectedEOF
	return
}

// SerializeValues writes the given values (and possibly header) to output.
//
// For valuesSerializerFeFe, it writes 2 field elements.
func (s *valuesSerializerFeFe) SerializeValues(output io.Writer, fieldElement1, fieldElement2 *fieldElements.FieldElement) (bytesWritten int, err bandersnatchErrors.SerializationError) {
	defer updateWriteError(&err, &bytesWritten, int(s.OutputLength())) // ensures PartialWrite is correctly set on error.

	bytesWritten, err = fieldElement1.Serialize(output, s.fieldElementEndianness)
	if err != nil {
		return
	}
	bytesJustWritten, err := fieldElement2.Serialize(output, s.fieldElementEndianness)
	bytesWritten += bytesJustWritten
	// We treat EOF like UnexpectedEOF at this point. The reason is that we treat the PAIR of field elements as a unit.
	bandersnatchErrors.UnexpectEOF2(&err) // transforms EOF -> UnexpectedEOF
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
// For valuesSerializerFeFe, it always outputs 64 and works with nil receivers.
func (s *valuesSerializerFeFe) OutputLength() int32 { return 64 }

// RecognizedParameters outputs a slice of strings listing valid parameters that can be queried/set via
// the hasParameter / makeCopyWithParameters generic methods.
//
// For a valuesSerializerFeFe, contains "Endianness" (the endianness used to (de)serialize FieldElements)
// This method works with nil receivers.
func (s *valuesSerializerFeFe) RecognizedParameters() []string {
	return []string{"Endianness"}
}

// Has Parameter checks whether the given Serializer has a parameter with parameterName that can be set queried with the respective Setters/Getters.
// This method works with nil receivers.
func (s *valuesSerializerFeFe) HasParameter(parameterName string) bool {
	return normalizeParameter(parameterName) == normalizeParameter("Endianness") // Only endianness, from embedded field.
}

//*******************************************************************************************************************************

// valuesSerializerHeaderFeHeaderFe is a serializer for a pair of field elements, where each of the two field elements
// has a prefix (of sub-byte length) contained in the msbs.
// These prefixes are fixed headers for the serializer and not part of the individual output/input field elements.
type valuesSerializerHeaderFeHeaderFe struct {
	fieldElementEndianness           // endianness for (de)serializing both field elements.
	bitHeader                        // bitHeader for the first field element. This is embedded, so we don't have to forward setters/getters.
	bitHeader2             bitHeader // bitHeader for the second field element.
}

// DeserializeValues reads from input and returns values.
//
// For valuesSerializerHeaderFeHeaderFe, it returns 2 field elements.
//
// Note the err is returned as second rather than last return value. This may trigger linters warnings.
// This choice is because it simplifies some reflection-using code using these methods, which is written for methods returning (int, error, ...) - tuples.
// Having the unknown-length part at the end makes things simpler.
func (s *valuesSerializerHeaderFeHeaderFe) DeserializeValues(input io.Reader) (bytesRead int, err bandersnatchErrors.DeserializationError, fieldElement1, fieldElement2 fieldElements.FieldElement) {
	defer updateReadError(&err, &bytesRead, int(s.OutputLength())) // Ensure correctess of PartialRead flag on error.

	bytesRead, err = fieldElement1.DeserializeWithPrefix(input, s.bitHeader, s.fieldElementEndianness)
	// Note: This aborts on ErrNonNormalizedDeserialization. I.e. if the first field element is not in normalized form, we do not even read the second.
	if err != nil {
		return
	}
	bytesJustRead, err := fieldElement2.DeserializeWithPrefix(input, s.bitHeader2, s.fieldElementEndianness)
	bytesRead += bytesJustRead
	// We treat EOF like UnexpectedEOF at this point. The reason is that we treat the PAIR of field elements as a unit.
	bandersnatchErrors.UnexpectEOF2(&err) // transforms EOF -> UnexpectedEOF
	return
}

// SerializeValues writes the given values (and possibly header) to output.
//
// For valuesSerializerHeaderFeHeaderFe, it writes 2 field elements and headers.
func (s *valuesSerializerHeaderFeHeaderFe) SerializeValues(output io.Writer, fieldElement1, fieldElement2 *fieldElements.FieldElement) (bytesWritten int, err bandersnatchErrors.SerializationError) {
	defer updateWriteError(&err, &bytesWritten, int(s.OutputLength())) // Ensure correctness of PartialWrite flag on error

	bytesWritten, err = fieldElement1.SerializeWithPrefix(output, s.bitHeader, s.fieldElementEndianness)
	if err != nil {
		return
	}

	bytesJustWritten, err := fieldElement2.SerializeWithPrefix(output, s.bitHeader2, s.fieldElementEndianness)
	bytesWritten += bytesJustWritten
	// We treat EOF like UnexpectedEOF at this point. The reason is that we treat the PAIR of field elements as a unit.
	bandersnatchErrors.UnexpectEOF2(&err) // transform EOF -> UnexpectedEOF
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
	// Validate each component individually. Note that there actually should be now way of Validate failing through this code path.
	s.fieldElementEndianness.Validate()
	s.bitHeader.Validate()
	s.bitHeader2.Validate()
}

// RecognizedParameters outputs a slice of strings listing valid parameters that can be queried/set via
// the hasParameter / makeCopyWithParameters generic methods.
//
// For a valuesSerializerHeaderFeHeaderFe, the list contains "Endianness" (the endianness used to (de)serialize FieldElements),
// "BitHeader", "BitHeader2" (the Headers used for the first and second field element)
// This method works with nil receivers.
func (s *valuesSerializerHeaderFeHeaderFe) RecognizedParameters() []string {
	return []string{"Endianness", "BitHeader", "BitHeader2"}
}

// Has Parameter checks whether the given Serializer has a parameter with parameterName that can be set queried with the respective Setters/Getters.
// This method works with nil receivers.
func (s *valuesSerializerHeaderFeHeaderFe) HasParameter(parameterName string) bool {
	return utils.ElementInList(parameterName, s.RecognizedParameters(), normalizeParameter)
}

// OutputLength returns the number of bytes written/read by this valuesSerialzer.
//
// For valuesSerializerHeaderFeHeaderFe, it always outputs 64.
// This method works with nil receivers.
func (s *valuesSerializerHeaderFeHeaderFe) OutputLength() int32 { return 64 }

//*******************************************************************************************************************************

// valuesSerializerFe is a simple serializer for a single field element.
type valuesSerializerFe struct {
	fieldElementEndianness // endianness for field element (de)serialization, embedded.
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
	// It should actually be impossible for this to fail through this code path.
	s.fieldElementEndianness.Validate()
}

// OutputLength returns the number of bytes written/read by this valuesSerialzer.
//
// For valuesSerializerFe, it always outputs 32. This can be called on nil receivers.
func (s *valuesSerializerFe) OutputLength() int32 { return 32 }

// RecognizedParameters outputs a slice of strings listing valid parameters that can be queried/set via
// the hasParameter / makeCopyWithParameters generic methods.
//
// For a valuesSerializerFe, the list contains "Endianness" (the endianness used to (de)serialize FieldElements).
// This method works with nil receivers.
func (s *valuesSerializerFe) RecognizedParameters() []string {
	return []string{"Endianness"}
}

// Has Parameter checks whether the given Serializer has a parameter with parameterName that can be set queried with the respective Setters/Getters.
// This method works with nil receivers.
func (s *valuesSerializerFe) HasParameter(parameterName string) bool {
	return utils.ElementInList(parameterName, s.RecognizedParameters(), normalizeParameter)
}

//*******************************************************************************************************************************

// valuesSerializerHeaderFe is a simple serializer for a single field element with sub-byte header
type valuesSerializerHeaderFe struct {
	fieldElementEndianness // endianness for field element (de)serialization
	bitHeader              // fixed sub-byte-sized header that is serialized/consumed at each write/read
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
	// Note: Should should be impossible to fail via this code path.
	s.fieldElementEndianness.Validate()
	s.bitHeader.Validate()
}

// OutputLength returns the number of bytes written/read by this valuesSerialzer.
//
// For valuesSerializerHeaderFe, it always outputs 32 and can be called on nil receivers.
func (s *valuesSerializerHeaderFe) OutputLength() int32 { return 32 }

// RecognizedParameters outputs a slice of strings listing valid parameters that can be queried/set via
// the hasParameter / makeCopyWithParameters generic methods.
//
// For a valuesSerializerHeaderFe, the list contains "Endianness" (the endianness used to (de)serialize FieldElements) and "BitHeader" (the sub-byte header).
// It can be called on nil receivers.
func (s *valuesSerializerHeaderFe) RecognizedParameters() []string {
	return []string{"Endianness", "BitHeader"}
}

// Has Parameter checks whether the given Serializer has a parameter with parameterName that can be set queried with the respective Setters/Getters.
// This method works with nil receivers.
func (s *valuesSerializerHeaderFe) HasParameter(parameterName string) bool {
	return utils.ElementInList(parameterName, s.RecognizedParameters(), normalizeParameter)
}

//*******************************************************************************************************************************

// valuesSerializerFeCompressedBit is a simple serializer for a field element + 1 extra bit. The extra bit is squeezed into the field element.
type valuesSerializerFeCompressedBit struct {
	fieldElementEndianness // endianness for field element (de)serialization.
}

// the extra bit b is encoded as a lenght-1 BitHeader (with value 1 iff b==true, i.e. the "obvious encoding"). This is just to get the types right:
const falsePrefix = common.PrefixBits(0b0)
const truePrefix = common.PrefixBits(0b1)

var falsePrefixBitHeader = common.MakeBitHeader(falsePrefix, 1)
var truePrefixBitHeader = common.MakeBitHeader(truePrefix, 1)

// DeserializeValues reads from input and returns values.
//
// For valuesSerializerFeCompressedBit, it returns 1 field element and 1 Bit.
//
// Note the err is returned as second rather than last return value. This may trigger linters warnings.
// This choice is because it simplifies some reflection-using code using these methods, which is written for methods returning (int, error, ...) - tuples.
// Having the unknown-length part at the end makes things simpler.
func (s *valuesSerializerFeCompressedBit) DeserializeValues(input io.Reader) (bytesRead int, err bandersnatchErrors.DeserializationError, fieldElement fieldElements.FieldElement, bit bool) {
	var prefix common.PrefixBits
	bytesRead, prefix, err = fieldElement.DeserializeAndGetPrefix(input, 1, s.fieldElementEndianness) // Get one prefix bit and deserialize the rest as field element.
	bit = (prefix != falsePrefix)
	return
}

// SerializeValues writes the given values (and possibly header) to output.
//
// For valuesSerializerFeFe, it writes 1 field element and 1 Bit.
func (s *valuesSerializerFeCompressedBit) SerializeValues(output io.Writer, fieldElement *fieldElements.FieldElement, bit bool) (bytesWritten int, err bandersnatchErrors.SerializationError) {
	var embeddedPrefix common.BitHeader
	if bit {
		embeddedPrefix = truePrefixBitHeader
	} else {
		embeddedPrefix = falsePrefixBitHeader
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
// For valuesSerializerFeCompressedBit, it always outputs 32 and can be called on nil receivers.
func (s *valuesSerializerFeCompressedBit) OutputLength() int32 { return 32 }

// RecognizedParameters outputs a slice of strings listing valid parameters that can be queried/set via
// the hasParameter / makeCopyWithParameters generic methods.
//
// For a valuesSerializerFeCompressedBit, the list contains "Endianness" (the endianness used to (de)serialize FieldElements).
// This method can be called on nil receivers.
func (s *valuesSerializerFeCompressedBit) RecognizedParameters() []string {
	return []string{"Endianness"}
}

// Has Parameter checks whether the given Serializer has a parameter with parameterName that can be set queried with the respective Setters/Getters.
// This method works with nil receivers.
func (s *valuesSerializerFeCompressedBit) HasParameter(parameterName string) bool {
	return utils.ElementInList(parameterName, s.RecognizedParameters(), normalizeParameter)
}
