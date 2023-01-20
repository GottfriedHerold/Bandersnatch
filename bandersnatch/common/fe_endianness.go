package common

import "encoding/binary"

const v_littleEndian = false
const v_bigEndian = true

type FieldElementEndianness struct {
	v bool
	byteOrder
}

func (s *FieldElementEndianness) validate() {
	if s.v == v_bigEndian && s.byteOrder != binary.BigEndian {
		panic("Invalid")
	}
	if s.v == v_littleEndian && s.byteOrder != binary.LittleEndian {
		panic("Invalid")
	}
}

func (s *FieldElementEndianness) Validate() { s.validate() }

func (s *FieldElementEndianness) SetEndianness(e binary.ByteOrder) {
	if e_fe, ok := e.(FieldElementEndianness); ok {
		*s = e_fe
		s.validate()
	} else {
		switch e {
		case binary.LittleEndian:
			s.v = v_littleEndian
			s.byteOrder = binary.LittleEndian
		case binary.BigEndian:
			s.v = v_bigEndian
			s.byteOrder = binary.BigEndian
		default:
			panic("")
		}
	}
}

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

func (s FieldElementEndianness) Uint256(in []byte) (little_endian_ret [4]uint64) {
	if len(in) < 32 {
		panic(ErrorPrefix + "Uint256 called on a slice of insufficient length")
	}
	if !onlyLittleAndBigEndianByteOrder {
		panic("Needs to change") // code below assumes there is only two options.
	}
	if s.v == v_bigEndian {
		for i := 0; i < 4; i++ {
			little_endian_ret[3-i] = binary.BigEndian.Uint64(in[i*8 : (i+1)*8])
		}
	} else {
		for i := 0; i < 4; i++ {
			little_endian_ret[i] = binary.LittleEndian.Uint64(in[i*8 : (i+1)*8])
		}
	}
	return
}

var (
	LittleEndian FieldElementEndianness = FieldElementEndianness{v: v_littleEndian, byteOrder: binary.LittleEndian}
	BigEndian    FieldElementEndianness = FieldElementEndianness{v: v_bigEndian, byteOrder: binary.BigEndian}
)

var (
	DefaultEndian = LittleEndian
)

func init() {
	DefaultEndian.validate()
	LittleEndian.validate()
	BigEndian.validate()
}
