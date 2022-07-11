package common

import "encoding/binary"

// This file defines a type for endianness to serialize FieldElements.
// NOTE: Our *internal* representations use a fixed order; endianness choice is only relevant for I/O.

// FieldElementEndianness really has only 2 possible values (little and big endian), so we could use a bool;
// However, forcing users to write $Packagename.BigEndian or $Packagename.LittleEndian is better, since it's self-documenting.
// It's also more extendible.

// We use FieldElementEndianness to determine the byteOrder or 256-bit field elements.
// It is part (usually via struct embedding) of serializers to control the endianness choices of field elements.
// The reason for the restriction to just 2 possible values is that the binary.ByteOrder interface is restricted
// to the default integer types and the interface lacks any general way to meaningfully extend it to 256-bit ints for field elements:
// We have no idea what e.g. SetEndianness(MixedEndian) for some custom MixedEndian type satisfying binary.ByteOrder should do.
// If FieldElementEndianness becomes an interface, this restriction is lifted.

type byteOrder = binary.ByteOrder // type alias to allow private struct embedding

// FieldElementEndianness is a struct satisfying binary.ByteOrder, extended to also allow (de)serializing
// 256-bit numbers, as required to serialize field elements.
//
// This is currently a struct, but may be changed to an interface.
// The implementation is currently limited to little endian and big endian (byte-wise).
//
// The zero value of FieldElementEndianness is not a usable object.
// We provide BigEndian, LittleEndian or DefaultEndian instances of this type.
type FieldElementEndianness struct {
	byteOrder
}

// GetEndianness is a simple getter. This exists for the sake of struct-embedding.
func (s FieldElementEndianness) GetEndianness() FieldElementEndianness {
	return s
}

// SetEndianness sets FieldElementEndianess by wrapping e.
// We only accept (literal) binary.LittleEndian or binary.BigEndian or any FieldElementEndianness.
// Other values will cause a panic.
func (s *FieldElementEndianness) SetEndianness(e binary.ByteOrder) {
	if e_fe, ok := e.(FieldElementEndianness); ok {
		s.byteOrder = e_fe.byteOrder
		s.validate()
	} else {
		s.byteOrder = e
		s.validate()
	}
}

// The implementation assumes that s.byteOrder == binary.LittleEndian or == binary.BigEndian.
// There is no reason to make this assumption from an API point of view, so we just don't
// and regard this fact as an implementation detail.
// We use this to mark places where we make that assumption.
const onlyLittleAndBigEndianByteOrder = true

// validate checks the FieldElementEndianness for Validity.
// This is called internally be setters.
func (s FieldElementEndianness) validate() {
	if s.byteOrder == nil {
		panic("bandersnatch / serialize: cannot set FieldElementEndianness to a nil binary.ByteOrder")
	}
	if !onlyLittleAndBigEndianByteOrder {
		panic("Needs to change")
	}
	if s.byteOrder != binary.BigEndian && s.byteOrder != binary.LittleEndian {
		panic("bandersnatch / serialize: we currently only support binary.BigEndian and binary.LittleEndian from the standard library as possible endianness")
	}
}

// Validate is required to satisfy certain (internal) cross-package interfaces.
func (s FieldElementEndianness) Validate() {
	if s.byteOrder == nil {
		panic("bandersnatch / serialize: uinitialized Field Element Endianness detected.")
	}
	s.validate() // actually, there ought to be no way to trigger an error from this.
}

// Note: Renamed from IsBigEndian to make it possible in theory to extend from only 2 possible endiannesses without breaking the API.
// We only need this to determine whether we can "peek" at the start of a stream to get a BitHeader.

// StartsWithMSB returns true if the first byte output/input correspond to the most significant bits.
func (s FieldElementEndianness) StartsWithMSB() bool {
	if !onlyLittleAndBigEndianByteOrder {
		panic("Needs to change")
	}
	return s.byteOrder == binary.BigEndian
}

// Annoying: I would want to write UInt256 rather than Uint256, but binary.ByteOrder used Uint64 etc.

// Should we require the given byte slices to have a length of exactly 32?
// The current interface is chosen to be consistent with binary.byteOrder's.

// PutUint256 writes the given uint256 (given as 4 uint64's in little endian order) to out with given byte endianness.
//
// out must have a length (not only capacity) of at least 32.
//
// Note that the endianness choice of s only affects the output stream, not the input.
func (s FieldElementEndianness) PutUint256(out []byte, little_endian_words [4]uint64) {
	// We test for length rather than capacity here (Note that insufficient capacity would cause a panic anyway by Go's runtime).
	// The case of insufficient length, but sufficient capacity would work in principle, but this is more likely a bug on the caller's side than not.
	if len(out) < 32 {
		if cap(out) < 32 {
			panic("bandersnatch / serialization: PutUint256 called on a slice of insufficient capacity")
		}
		panic("bandersnatch / serialization: PutUint256 called on a slice of insufficient length (but sufficient capacity)")
	}
	if !onlyLittleAndBigEndianByteOrder {
		panic("Needs to change") // code below assumes there is only two options.
	}
	if s.byteOrder == binary.BigEndian {
		for i := 0; i < 4; i++ {
			s.byteOrder.PutUint64(out[i*8:(i+1)*8], little_endian_words[3-i])
		}
	} else {
		for i := 0; i < 4; i++ {
			s.byteOrder.PutUint64(out[i*8:(i+1)*8], little_endian_words[i])
		}
	}
}

// Uint256 reads the first 32 bytes from in, interprets them according to the endianness choice and
// returns a 256-bit integer, encoded as 4 uint64's in little endian order.
//
// Note that the endianness choice of s only affects the input stream, not the output.
func (s FieldElementEndianness) Uint256(in []byte) (little_endian_ret [4]uint64) {
	if len(in) < 32 {
		panic("bandersnatch / serialization: Uint256 called on a slice of insufficient length")
	}
	if !onlyLittleAndBigEndianByteOrder {
		panic("Needs to change") // code below assumes there is only two options.
	}
	if s.byteOrder == binary.BigEndian {
		for i := 0; i < 4; i++ {
			little_endian_ret[3-i] = s.byteOrder.Uint64(in[i*8 : (i+1)*8])
		}
	} else {
		for i := 0; i < 4; i++ {
			little_endian_ret[i] = s.byteOrder.Uint64(in[i*8 : (i+1)*8])
		}
	}
	return
}

// DefaultEndian is the default setting, we use in our serializers unless overridden.
// NOTE: You must not modify DefaultEndian; if you want to deviate from the default, create a new serializer with modified endianness.
var DefaultEndian FieldElementEndianness = FieldElementEndianness{byteOrder: binary.LittleEndian}
var LittleEndian FieldElementEndianness = FieldElementEndianness{byteOrder: binary.LittleEndian}
var BigEndian FieldElementEndianness = FieldElementEndianness{byteOrder: binary.BigEndian}

func init() {
	DefaultEndian.validate()
	LittleEndian.validate()
	BigEndian.validate()
}
