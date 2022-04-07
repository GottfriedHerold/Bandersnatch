package pointserializer

import (
	"encoding/binary"

	"github.com/GottfriedHerold/Bandersnatch/bandersnatch"
	// . "github.com/GottfriedHerold/Bandersnatch/bandersnatch"
)

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

var defaultEndianness fieldElementEndianness = fieldElementEndianness{byteOrder: binary.LittleEndian}

// TODO: Move this to main utils? Should be used by field element serializers.

// bitHeader is a "header" consisting of a prefixLen many extra bits that are included inside a field element as a form of compression.
type bitHeader struct {
	prefixBits bandersnatch.PrefixBits
	prefixLen  uint8
}

func (bh *bitHeader) SetBitHeader(newBitHeader bitHeader) {
	if newBitHeader.prefixLen > 8 {
		panic("bandersnatch / serialization: trying to set bit-prefix of length > 8")
	}
	bitFilter := (1 << newBitHeader.prefixLen) - 1 // bitmask of the form 0b0..01..1 ending with prefixLen 1s
	if bitFilter&int(newBitHeader.prefixBits) != int(newBitHeader.prefixBits) {
		panic("bandersnatch / serialization: trying to set bitHeader with a prefix and length, where the prefix has bits set that are not among the length many lsb")
	}
	*bh = newBitHeader
}

func (bh *bitHeader) GetBitHeader() bitHeader {
	return *bh
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
