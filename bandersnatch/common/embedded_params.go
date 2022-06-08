package common

import (
	"encoding/binary"
)

// This file defines wrappers (essentially just a data element with a setter/getter method) that are needed because
// either we have some validity constraints that we need to check in the setter or that are struct-embedded in serializers.
//
// For the latter (structs embedded in serializers), note that parameter setting actually goes through reflection, which is a bit easier
// (and consistent, since at least for some parameters we need validity checks)
// with getters/setters, so we want to always have those.

// Note: FieldElementEndianness has only 2 possible values, so we could use a bool;
// However, forcing users to write binary.BigEndian or binary.LittleEndian is better, since it's self-documenting.

// FieldElementEndianness is just a wrapper around binary.ByteOrder, restricted to Little and Big Endian.
// Note that we ONLY support the predefined standard library constants binary.BigEndian and binary.LittleEndian.
// Trying to set to any other value will panic.
//
// It is used to determine the byteOrder or 256-bit field elements.
// It is part (usually via struct embedding) of serializers to control the FieldElementEndianness of field elements.
// The reason for the restriction to just 2 possible values is that the binary.ByteOrder interface is restricted to the default integer types and the interface lacks any general way to meaningfully extend it to 256-bit ints for field elements.
type FieldElementEndianness struct {
	byteOrder binary.ByteOrder
}

// GetEndianness unwraps FieldElementEndianness to binary.ByteOrder
func (s *FieldElementEndianness) GetEndianness() binary.ByteOrder {
	return s.byteOrder
}

// SetEndianness sets FieldElementEndianess by wrapping e. We only accept (literal) binary.LittleEndian or binary.BigEndian or any *FieldElementEndianness.
// Other values will cause a panic.
func (s *FieldElementEndianness) SetEndianness(e binary.ByteOrder) {
	if wrapping, ok := e.(*FieldElementEndianness); ok {
		s.byteOrder = wrapping.byteOrder
		s.Validate()
	} else {
		s.byteOrder = e
		s.Validate()
	}
}

// Validate checks the FieldElementEndianness for Validity.
func (s *FieldElementEndianness) Validate() {
	if s.byteOrder == nil {
		panic("bandersnatch / serialize: cannot set FieldElementEndianness to a nil binary.ByteOrder")
	}
	if s.byteOrder != binary.BigEndian && s.byteOrder != binary.LittleEndian {
		panic("bandersnatch / serialize: we only support binary.BigEndian and binary.LittleEndian from the standard library as possible endianness")
	}
}

func (s *FieldElementEndianness) IsLittleEndian() bool {
	return s.byteOrder == binary.LittleEndian
}

func (s *FieldElementEndianness) IsBigEndian() bool {
	return s.byteOrder == binary.BigEndian
}

// forward function from binary.ByteOrder, so FieldElementEndianness actually satisfies binary.ByteOrder

func (s *FieldElementEndianness) Uint64(in []byte) uint64 {
	return s.byteOrder.Uint64(in)
}

func (s *FieldElementEndianness) Uint32(in []byte) uint32 {
	return s.byteOrder.Uint32(in)
}

func (s *FieldElementEndianness) Uint16(in []byte) uint16 {
	return s.byteOrder.Uint16(in)
}

func (s *FieldElementEndianness) PutUint64(out []byte, in uint64) {
	s.byteOrder.PutUint64(out, in)
}

func (s *FieldElementEndianness) PutUint32(out []byte, in uint32) {
	s.byteOrder.PutUint32(out, in)
}

func (s *FieldElementEndianness) PutUint16(out []byte, in uint16) {
	s.byteOrder.PutUint16(out, in)
}

func (s FieldElementEndianness) String() string {
	return s.byteOrder.String()
}

func (s *FieldElementEndianness) PutUInt256(out []byte, low_endian_words [4]uint64) {
	if cap(out) < 32 {
		panic("bandersnatch / serialization: PutUInt256 called on a slice of insufficient capacity")
	}
	if s.IsBigEndian() {
		for i := 0; i < 4; i++ {
			s.byteOrder.PutUint64(out[i*8:(i+1)*8], low_endian_words[3-i])
		}
	} else {
		for i := 0; i < 4; i++ {
			s.byteOrder.PutUint64(out[i*8:(i+1)*8], low_endian_words[i])
		}
	}
}

func (s *FieldElementEndianness) UInt256(in []byte) (ret [4]uint64) {
	if len(in) < 32 {
		panic("bandersnatch / serialization: UInt256 called on a slice of insufficient length")
	}
	if s.IsBigEndian() {
		for i := 0; i < 4; i++ {
			ret[3-i] = s.byteOrder.Uint64(in[i*8 : (i+1)*8])
		}
	} else {
		for i := 0; i < 4; i++ {
			ret[i] = s.byteOrder.Uint64(in[i*8 : (i+1)*8])
		}
	}
	return
}

// DefaultEndian is the default setting, we use in our serializers unless overridden.
// NOTE: Do not modify DefaultEndian; if you want to deviate from the default, create a new serializer with modified endianness.
var DefaultEndian FieldElementEndianness = FieldElementEndianness{byteOrder: binary.LittleEndian}
var LittleEndian FieldElementEndianness = FieldElementEndianness{byteOrder: binary.LittleEndian}
var BigEndian FieldElementEndianness = FieldElementEndianness{byteOrder: binary.BigEndian}

func init() {
	DefaultEndian.Validate()
	LittleEndian.Validate()
	BigEndian.Validate()
}

// BitHeader is a "header" consisting of a prefixLen < 8 many extra bits that are included inside a field element as a form of compression.
// The zero value of BitHeader is a valid, but useless length-0 bit header.
type BitHeader struct {
	prefixBits PrefixBits // based on byte. We use a different type to avoid mistaking parameter orders.
	prefixLen  uint8
}

// PrefixBits is a type based on byte
type PrefixBits byte

// MaxLengthPrefixBits is the maximal length of a BitHeader. Since it needs to fit in a byte, this value is 8.
const MaxLengthPrefixBits = 8

// SetBitHeaderFromBitHeader and GetBitHeader are an internal function that
// need to be exported for cross-package and reflect usage:
// our (rather generic) parameter-update functions for serializers go through reflection
// and always require some form of possibly trivial getter / setter methods.

// SetBitHeaderFromBitHeader copies a BitHeader into another.
//
// This function is only exported (and needed) for internal cross-package and reflect usage.
// Plain assignment works just fine.
func (bh *BitHeader) SetBitHeaderFromBitHeader(newBitHeader BitHeader) {
	*bh = newBitHeader
	bh.Validate() // not needed, technically. newBitHeader is guaranteed to satisfy this in the first place.
}

// GetBitHeader returns a copy of the given BitHeader.
//
// This function is only exported (and needed) for internal cross-package and reflect usage.
// Plain assignment works just fine.
func (bh *BitHeader) GetBitHeader() BitHeader {
	// Note: No need to make a copy, since we return a value.
	return *bh
}

// SetBitHeader sets the BitHeader to the given prefixBits and prefixLen.
// It panics if the input is invalid.
//
// Note: PrefixBits is based on uint8 == byte.
// You are supposed to write bh.SetBitHeader(PrefixBits(0b101), 4)
// with explicit type conversion to PrefixBits in order to not mess up the order of parameters.
func (bh *BitHeader) SetBitHeader(prefixBits PrefixBits, prefixLen uint8) {
	bh.prefixBits = prefixBits
	bh.prefixLen = prefixLen
	bh.Validate()
}

// MakeBitHeader creates a new BitHeader with the given prefixBits and prefixLen.
// It panics for invalid inputs.
//
// Note: PrefixBits is based on uint8 == byte.
// You are supposed to write MakeBitHeader(PrefixBits(0b101), 4)
// with explicit type conversion to PrefixBits in order to not mess up the order of parameters.
func MakeBitHeader(prefixBits PrefixBits, prefixLen uint8) BitHeader {
	var ret BitHeader = BitHeader{prefixBits: prefixBits, prefixLen: prefixLen}
	ret.Validate()
	return ret
}

// PrefixBits obtains the PrefixBits of a BitHeader
func (bh *BitHeader) PrefixBits() PrefixBits {
	return bh.prefixBits
}

// PrefixLen obtains the prefix length of a BitHeader
func (bh *BitHeader) PrefixLen() uint8 {
	return bh.prefixLen
}

// Validate ensures the BitHeader is valid. It panics if not.
func (bh *BitHeader) Validate() {
	if bh.prefixLen > MaxLengthPrefixBits {
		panic("bandersnatch / serialization: trying to set bit-prefix of length > 8")
	}
	bitFilter := (1 << bh.prefixLen) - 1 // bitmask of the form 0b0..01..1 ending with prefixLen 1s
	if bitFilter&int(bh.prefixBits) != int(bh.prefixBits) {
		panic("bandersnatch / serialization: trying to set bitHeader with a prefix and length, where the prefix has bits set that are not among the length many least significant bits")
	}
}

// implicit interface with methods SetSubgroupRestriction(bool) and IsSubgroupOnly() bool defined in tests only.
// Since we use reflection, we don't need the explicit interface here.

// SubgroupRestriction is a type (intended for struct embedding into serializers) wrapping a bool
// that determines whether the serializer only works for subgroup elements.
// The purpose is to have getters and setters.
type SubgroupRestriction struct {
	subgroupOnly bool
}

func (sr *SubgroupRestriction) SetSubgroupRestriction(restrict bool) {
	sr.subgroupOnly = restrict
}

func (sr *SubgroupRestriction) IsSubgroupOnly() bool {
	return sr.subgroupOnly
}

func (sr *SubgroupRestriction) Validate() {}

// SubgroupOnly is a type wrapping a bool constant true that indicates that the serializer only works for subgroup elements. Used as embedded struct to forward setter and getter methods to reflect.
type SubgroupOnly struct {
}

func (sr *SubgroupOnly) IsSubgroupOnly() bool {
	return true
}

func (sr *SubgroupOnly) SetSubgroupRestriction(restrict bool) {
	if !restrict {
		panic("bandersnatch / serialization: Trying to unset restriction to subgroup points for a serializer that does not support this")
	}
}

func (sr *SubgroupOnly) Validate() {}
