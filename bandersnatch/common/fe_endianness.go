package common

import "encoding/binary"

// This file defines a type for endianness to serialize FieldElements.
// NOTE: Our *internal* representations use a fixed order; endianness choice is only relevant for I/O.

// FieldElementEndianness really has only 2 possible values (little and big endian), so we could use a bool;
// However, forcing users to write $Packagename.BigEndian or $Packagename.LittleEndian is better, since it's self-documenting.
// It's also more extendible.

// We use FieldElementEndianness to determine the byteOrder or 256-bit field elements.
// It is part (possibly via struct embedding) of serializers to control the endianness choices of field elements.
// The current implementation is restriction to just 2 possible values (big and little Endian):
//
// From an API point of view, it would make much more sense to make FieldElementEndianness an interface that extends binary.ByteOrder
// and provide implementations for LittleEndian and BigEndian (as well as allowing conversion from at least the two standard binary.Byte order literals)
// In fact, this implementation was refactored away from an interface.
//
// The issue is that with an interface, there is an order of magnitude of performance loss, due to the fact that
// with var x FieldElementEndianness, calling x.PutUint256(arg) through an interface forces heap-allocation of arg.

// is this still used?
type byteOrder = binary.ByteOrder // type alias to allow private struct embedding

// The implementation assumes that s.byteOrder == binary.LittleEndian or == binary.BigEndian.
// There is no reason to make this assumption from an API point of view, so we just don't
// and regard this fact as an implementation detail.
// We use this to mark places where we make that assumption.
const onlyLittleAndBigEndianByteOrder = true

// FieldElementEndianness is a struct or interface (users should account for the option to change this) that satisfies/extends the
// [binary.ByteOrder] to 256-bit numbers.
type FieldElementEndianness struct {
	v bool // value indicating whether we are little Endian or big Endian
}

// The bool stored inside FieldElementEndianness is compared against these constants (essentially an enum-type based on bool)
// The choice is such that the zero-value of bool (i.e. false) corresponds to the default endianness.
const (
	v_littleEndian = false
	v_bigEndian    = true
)

// Validate checks the FieldElementEndianness for Validity.
//
// TODO: Note that there is no way for this to fail at the moment; it is required due to some generic code.
func (s *FieldElementEndianness) Validate() {}

// SetEndianness sets FieldElementEndianess by wrapping e.
// We only accept (literal) e==binary.LittleEndian or e==binary.BigEndian or any FieldElementEndianness e.
// Other values for e will cause a panic.
func (s *FieldElementEndianness) SetEndianness(e binary.ByteOrder) {
	if e_fe, ok := e.(FieldElementEndianness); ok {
		*s = e_fe
	} else {
		switch e {
		case binary.LittleEndian:
			s.v = v_littleEndian
		case binary.BigEndian:
			s.v = v_bigEndian
		default:
			panic(ErrorPrefix + "SetEndianness only accepts values of type FieldElementEndianness or literal binary.LittleEndian or binary.BigEndian")
		}
	}
}

// GetEndianness is a simple getter (in fact, the identity map). This exists for the sake of struct-embedding.
func (s FieldElementEndianness) GetEndianness() FieldElementEndianness {
	return s
}

// StartsWithMSB returns true if the first byte output/input correspond to the most significant bits.
func (s FieldElementEndianness) StartsWithMSB() bool {
	if !onlyLittleAndBigEndianByteOrder {
		panic("Needs to change")
	}
	return s.v == v_bigEndian
}

// Interface consistent with PutUint16, etc.
// NOTE: We would like to have a PutUint256([]byte, Uint256) for the actual Uint256 type

// PutUint256 writes the given uint256 (given as 4 uint64's in little endian order) to out with given byte endianness.
// Note that values of type [Uint256] need to be converted to [4]uint64 (which [Uint256] is based on).
//
// out must have a length (not only capacity) of at least 32.
//
// Note that the endianness choice determined by s only affects the output stream, not the input.
//
// This function is provided solely for API consistency with [PutUint16], [PutUint32] and [PutUint64].
// Users should use [PutUint256_ptr] or [PutUint256_array], which is much faster.
func (s FieldElementEndianness) PutUint256(out []byte, little_endian_words [4]uint64) {

	if len(out) < 32 {
		if cap(out) < 32 {
			panic(ErrorPrefix + "PutUint256 called on a slice of insufficient capacity")
		}
		panic(ErrorPrefix + "PutUint256 called on a slice of insufficient length (but sufficient capacity)")
	}

	if s.v == v_littleEndian {
		binary.LittleEndian.PutUint64(out[0:8], little_endian_words[0])
		binary.LittleEndian.PutUint64(out[8:16], little_endian_words[1])
		binary.LittleEndian.PutUint64(out[16:24], little_endian_words[2])
		binary.LittleEndian.PutUint64(out[24:32], little_endian_words[3])
	} else {
		binary.BigEndian.PutUint64(out[0:8], little_endian_words[3])
		binary.BigEndian.PutUint64(out[8:16], little_endian_words[2])
		binary.BigEndian.PutUint64(out[16:24], little_endian_words[1])
		binary.BigEndian.PutUint64(out[24:32], little_endian_words[0])
	}
}

// PutUint256_ptr writes the given uint256 (given as 4 uint64's in little endian order) to out with given byte endianness.
// Note that values of type [Uint256] need to be converted to [4]uint64 (which [Uint256] is based on).
//
// out must have a length (not only capacity) of at least 32.
//
// Note that the endianness choice determined by s only affects the output stream, not the input.
//
// This method is similar to [PutUint256], but more efficient.
func (s FieldElementEndianness) PutUint256_ptr(out []byte, little_endian_words *[4]uint64) {

	if len(out) < 32 {
		if cap(out) < 32 {
			panic(ErrorPrefix + "PutUint256 called on a slice of insufficient capacity")
		}
		panic(ErrorPrefix + "PutUint256 called on a slice of insufficient length (but sufficient capacity)")
	}

	if s.v == v_littleEndian {
		binary.LittleEndian.PutUint64(out[0:8], little_endian_words[0])
		binary.LittleEndian.PutUint64(out[8:16], little_endian_words[1])
		binary.LittleEndian.PutUint64(out[16:24], little_endian_words[2])
		binary.LittleEndian.PutUint64(out[24:32], little_endian_words[3])
	} else {
		binary.BigEndian.PutUint64(out[0:8], little_endian_words[3])
		binary.BigEndian.PutUint64(out[8:16], little_endian_words[2])
		binary.BigEndian.PutUint64(out[16:24], little_endian_words[1])
		binary.BigEndian.PutUint64(out[24:32], little_endian_words[0])
	}
}

// PutUint256_array writes the given uint256 (given as 4 uint64's in little endian order) to out with given byte endianness.
// Note that values of type [Uint256] need to be converted to [4]uint64 (which [Uint256] is based on).
//
// Note that the endianness choice determined by s only affects the output stream, not the input.
//
// This method is similar to [PutUint256] or [PutUint256_ptr], but more efficient.
func (s FieldElementEndianness) PutUint256_array(out *[32]byte, little_endian_words *[4]uint64) {
	if s.v == v_littleEndian {
		binary.LittleEndian.PutUint64(out[0:8], little_endian_words[0])
		binary.LittleEndian.PutUint64(out[8:16], little_endian_words[1])
		binary.LittleEndian.PutUint64(out[16:24], little_endian_words[2])
		binary.LittleEndian.PutUint64(out[24:32], little_endian_words[3])
	} else {
		binary.BigEndian.PutUint64(out[0:8], little_endian_words[3])
		binary.BigEndian.PutUint64(out[8:16], little_endian_words[2])
		binary.BigEndian.PutUint64(out[16:24], little_endian_words[1])
		binary.BigEndian.PutUint64(out[24:32], little_endian_words[0])
	}
}

// Uint256 reads the first 32 bytes from in, interprets them according to the endianness choice and
// returns a 256-bit integer, encoded as 4 uint64's in little endian order.
//
// Note that the endianness choice of s only affects the input stream, not the output.
func (s FieldElementEndianness) Uint256(in []byte) (little_endian_ret [4]uint64) {
	if len(in) < 32 {
		panic(ErrorPrefix + "Uint256 called on a slice of insufficient length")
	}
	if s.v == v_littleEndian {
		little_endian_ret[0] = binary.LittleEndian.Uint64(in[0:8])
		little_endian_ret[1] = binary.LittleEndian.Uint64(in[8:16])
		little_endian_ret[2] = binary.LittleEndian.Uint64(in[16:24])
		little_endian_ret[3] = binary.LittleEndian.Uint64(in[24:32])
	} else {
		little_endian_ret[3] = binary.BigEndian.Uint64(in[0:8])
		little_endian_ret[2] = binary.BigEndian.Uint64(in[8:16])
		little_endian_ret[1] = binary.BigEndian.Uint64(in[16:24])
		little_endian_ret[0] = binary.BigEndian.Uint64(in[24:32])
	}
	return
}

// same as above, but instead of returning a [4]uint64, we provide the address where to put them.

// Uint256_indirect is similar to [Uint256], but writes to a given [4]uint64 rather than returnig it.
func (s FieldElementEndianness) Uint256_indirect(in []byte, little_endian_ret *[4]uint64) {
	if len(in) < 32 {
		panic(ErrorPrefix + "Uint256 called on a slice of insufficient length")
	}
	if s.v == v_littleEndian {
		little_endian_ret[0] = binary.LittleEndian.Uint64(in[0:8])
		little_endian_ret[1] = binary.LittleEndian.Uint64(in[8:16])
		little_endian_ret[2] = binary.LittleEndian.Uint64(in[16:24])
		little_endian_ret[3] = binary.LittleEndian.Uint64(in[24:32])
	} else {
		little_endian_ret[3] = binary.BigEndian.Uint64(in[0:8])
		little_endian_ret[2] = binary.BigEndian.Uint64(in[8:16])
		little_endian_ret[1] = binary.BigEndian.Uint64(in[16:24])
		little_endian_ret[0] = binary.BigEndian.Uint64(in[24:32])
	}
}

// same as above, but the input is a pointer-to-array (of the correct size) instead of a slice.

// Uint256_array is similar to [Uint256_indirect], but the input is given as pointer-to-array (of correct size) rather than as a slice.
func (s FieldElementEndianness) Uint256_array(in *[32]byte, little_endian_ret *[4]uint64) {
	if s.v == v_littleEndian {
		little_endian_ret[0] = binary.LittleEndian.Uint64(in[0:8])
		little_endian_ret[1] = binary.LittleEndian.Uint64(in[8:16])
		little_endian_ret[2] = binary.LittleEndian.Uint64(in[16:24])
		little_endian_ret[3] = binary.LittleEndian.Uint64(in[24:32])
	} else {
		little_endian_ret[3] = binary.BigEndian.Uint64(in[0:8])
		little_endian_ret[2] = binary.BigEndian.Uint64(in[8:16])
		little_endian_ret[1] = binary.BigEndian.Uint64(in[16:24])
		little_endian_ret[0] = binary.BigEndian.Uint64(in[24:32])
	}
}

// Uint16 is provided to satisfy the [binary.ByteOrder] interface
func (s FieldElementEndianness) Uint16(in []byte) uint16 {
	if s.v == v_littleEndian {
		return binary.LittleEndian.Uint16(in)
	} else {
		return binary.BigEndian.Uint16(in)
	}
}

// Uint32 is provided to satisfy the [binary.ByteOrder] interface
func (s FieldElementEndianness) Uint32(in []byte) uint32 {
	if s.v == v_littleEndian {
		return binary.LittleEndian.Uint32(in)
	} else {
		return binary.BigEndian.Uint32(in)
	}
}

// Uint64 is provided to satisfy the [binary.ByteOrder] interface
func (s FieldElementEndianness) Uint64(in []byte) uint64 {
	if s.v == v_littleEndian {
		return binary.LittleEndian.Uint64(in)
	} else {
		return binary.BigEndian.Uint64(in)
	}
}

// String is provided to satisfy the [binary.ByteOrder] interface
func (s FieldElementEndianness) String() string {
	if s.v == v_littleEndian {
		return "LittleEndian"
	} else {
		return "BigEndian"
	}
}

// PutUint16 is provided to satisfy the [binary.ByteOrder] interface
func (s FieldElementEndianness) PutUint16(out []byte, in uint16) {
	if s.v == v_littleEndian {
		binary.LittleEndian.PutUint16(out, in)
	} else {
		binary.BigEndian.PutUint16(out, in)
	}
}

// PutUint32 is provided to satisfy the [binary.ByteOrder] interface
func (s FieldElementEndianness) PutUint32(out []byte, in uint32) {
	if s.v == v_littleEndian {
		binary.LittleEndian.PutUint32(out, in)
	} else {
		binary.BigEndian.PutUint32(out, in)
	}
}

// PutUint64 is provided to satisfy the [binary.ByteOrder] interface
func (s FieldElementEndianness) PutUint64(out []byte, in uint64) {
	if s.v == v_littleEndian {
		binary.LittleEndian.PutUint64(out, in)
	} else {
		binary.BigEndian.PutUint64(out, in)
	}
}

// LittleEndian and BigEndian of type FieldElementEndianness are used for big resp. little endian serialization of field elements.
var (
	LittleEndian FieldElementEndianness = FieldElementEndianness{v: v_littleEndian}
	BigEndian    FieldElementEndianness = FieldElementEndianness{v: v_bigEndian}
)

// NOTE: DefaultEndian is the zero value of FieldElementEndianness (by design).
// While intentional, we do not promise that as part of the API, becaue it would preclude us from turning FieldElementEndianness into an interface
// (maybe in same later version of Go)

// DefaultEndian is the default setting we use in our serializers unless overridden.
// NOTE: Users should not modify DefaultEndian; if you want to deviate from the default, create a new serializer with modified endianness.
var (
	DefaultEndian = LittleEndian
)

func init() {
	DefaultEndian.Validate()
	LittleEndian.Validate()
	BigEndian.Validate()
}
