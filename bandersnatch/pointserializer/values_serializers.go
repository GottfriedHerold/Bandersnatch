package pointserializer

import (
	"errors"
	"io"

	"github.com/GottfriedHerold/Bandersnatch/bandersnatch"
	"github.com/GottfriedHerold/Bandersnatch/internal/utils"
)

// Due to insufficency of generics, we separate our serializers depending on whether the internal object that actually gets serialized consists of
// a field element, two field element, field element+bit etc.
// These (de)serializiers all have serializeValues and DeserializeValues methods, which differ in their arguments.

// ValuesSerializers are serializers for field elements and bits. These have the following "interface", defined on pointer receivers
//
// DeserializeValues(input io.Reader) (bytesRead int, err error, ...)
// SerializeValues(output io.Writer, ...) (bytesWritten int, err error)
// Clone() receiver(pointer) [NOTE: Returned type is concrete, not interface type]
//
// The parameter types of DeserializeValues and SerializeValues need to match. The return type of Clone() is the same as the pointer receiver.
// Go's interfaces cannot express this. We use reflection.

// Note: These are internal struct that only serve to modularize the (de)serializers.
// The methods are exported in order to make them callable via reflection.

// valuesSerializerFeFe is a simple serializer for a pair of field elements
type valuesSerializerFeFe struct {
	fieldElementEndianness // meaning the endianness for fieldElementSerialization
}

func (s *valuesSerializerFeFe) DeserializeValues(input io.Reader) (bytesRead int, err error, fieldElement1, fieldElement2 bandersnatch.FieldElement) {
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

func (s *valuesSerializerFeFe) SerializeValues(output io.Writer, fieldElement1, fieldElement2 *bandersnatch.FieldElement) (bytesWritten int, err error) {
	bytesWritten, err = fieldElement1.Serialize(output, s.byteOrder)
	if err != nil {
		return
	}
	bytesJustWritten, err := fieldElement2.Serialize(output, s.byteOrder)
	// We treat EOF like UnexpectedEOF at this point. The reason is that we treat the PAIR of field elements as a unit.
	utils.UnexpectEOF(&err)
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

func (s *valuesSerializerHeaderFeHeaderFe) DeserializeValues(input io.Reader) (bytesRead int, err error, fieldElement1, fieldElement2 bandersnatch.FieldElement) {
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

func (s *valuesSerializerHeaderFeHeaderFe) SerializeValues(output io.Writer, fieldElement1, fieldElement2 *bandersnatch.FieldElement) (bytesWritten int, err error) {
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

func (s *valuesSerializerHeaderFeHeaderFe) SetBitHeader2(bh bitHeader) {
	s.bitHeader2.SetBitHeader(bh)
}

func (s *valuesSerializerHeaderFeHeaderFe) GetBitHeader2() bitHeader {
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

func (s *valuesSerializerFe) DeserializeValues(input io.Reader) (bytesRead int, err error, fieldElement bandersnatch.FieldElement) {
	bytesRead, err = fieldElement.Deserialize(input, s.byteOrder)
	return
}

func (s *valuesSerializerFe) SerializeValues(output io.Writer, fieldElement *bandersnatch.FieldElement) (bytesWritten int, err error) {
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

func (s *valuesSerializerHeaderFe) DeserializeValues(input io.Reader) (bytesRead int, err error, fieldElement bandersnatch.FieldElement) {
	bytesRead, err = fieldElement.DeserializeWithPrefix(input, s.prefixBits, s.prefixLen, s.byteOrder)
	return
}

func (s *valuesSerializerHeaderFe) SerializeValues(output io.Writer, fieldElement *bandersnatch.FieldElement) (bytesWritten int, err error) {
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

func (s *valuesSerializerFeCompressedBit) DeserializeValues(input io.Reader) (bytesRead int, err error, fieldElement bandersnatch.FieldElement, bit bool) {
	var prefix bandersnatch.PrefixBits
	bytesRead, prefix, err = fieldElement.DeserializeAndGetPrefix(input, 1, s.byteOrder)
	bit = (prefix == 0b1)
	return
}

func (s *valuesSerializerFeCompressedBit) SerializeValues(output io.Writer, fieldElement *bandersnatch.FieldElement, bit bool) (bytesWritten int, err error) {
	var embeddedPrefix bandersnatch.PrefixBits
	if bit {
		embeddedPrefix = bandersnatch.PrefixBits(0b1)
	} else {
		embeddedPrefix = bandersnatch.PrefixBits(0b0)
	}
	bytesWritten, err = fieldElement.SerializeWithPrefix(output, embeddedPrefix, 1, s.byteOrder)
	return
}

func (s *valuesSerializerFeCompressedBit) Clone() *valuesSerializerFeCompressedBit {
	return &valuesSerializerFeCompressedBit{fieldElementEndianness: s.fieldElementEndianness}
}
