package utils

import (
	"encoding/binary"
	"math/big"
)

const ErrorPrefix = "bandersnatch / internal / utils: "

// UintarrayToInt converts a low-endian [4]uint64 array to big.Int, without any Montgomery conversions
func UIntarrayToInt(z *[4]uint64) *big.Int {
	var big_endian_byte_slice [32]byte
	binary.BigEndian.PutUint64(big_endian_byte_slice[0:8], z[3])
	binary.BigEndian.PutUint64(big_endian_byte_slice[8:16], z[2])
	binary.BigEndian.PutUint64(big_endian_byte_slice[16:24], z[1])
	binary.BigEndian.PutUint64(big_endian_byte_slice[24:32], z[0])
	return new(big.Int).SetBytes(big_endian_byte_slice[:])
}

// BigIntToUIntArray converts a big.Int to a low-endian [4]uint64 array without Montgomery conversions.
// We assume 0 <= x < 2^256
func BigIntToUIntArray(x *big.Int) (result [4]uint64) {
	// As this is an internal function, panic is OK for error handling.
	if x.Sign() < 0 {
		panic(ErrorPrefix + "bigIntToUIntArray: Trying to convert negative big Int")
	}
	if x.BitLen() > 256 {
		panic(ErrorPrefix + "bigIntToUIntArray: big Int too large to fit into 32 bytes.")
	}
	var big_endian_byte_slice [32]byte
	x.FillBytes(big_endian_byte_slice[:])
	result[0] = binary.BigEndian.Uint64(big_endian_byte_slice[24:32])
	result[1] = binary.BigEndian.Uint64(big_endian_byte_slice[16:24])
	result[2] = binary.BigEndian.Uint64(big_endian_byte_slice[8:16])
	result[3] = binary.BigEndian.Uint64(big_endian_byte_slice[0:8])
	return
}

