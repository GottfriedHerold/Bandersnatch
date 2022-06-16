package common

import "encoding/binary"

// Note: FieldElementEndianness has only 2 possible values, so we could use a bool;
// However, forcing users to write $Packagename.BigEndian or $Packagename.LittleEndian is better, since it's self-documenting.

// FieldElementEndianness is just a wrapper around binary.ByteOrder, restricted to Little and Big Endian.
// Note that we ONLY support the predefined standard library constants binary.BigEndian and binary.LittleEndian for now.
// Trying to set to any other value will panic.
//
// We might eventually upgrade this to an interface type.
//
// It is used to determine the byteOrder or 256-bit field elements.
// It is part (usually via struct embedding) of serializers to control the FieldElementEndianness of field elements.
// This usage neccessitates Setters and Getters.
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
