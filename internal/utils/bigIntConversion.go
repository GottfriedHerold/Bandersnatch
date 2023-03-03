package utils

import (
	"encoding/binary"
	"math/big"
)

// TODO: Move to uint256 type

// ErrorPrefix is prepended to all errors messages originating from this package.
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

// InitIntFromString initializes a [*big.Int] from a given string similar to InitFieldElementFromString.
// This internally uses [*big.Int]'s SetString and understands exactly those string formats.
// This implies that the given string can be decimal, hex, octal or binary, but needs to be prefixed if not decimal.
//
// This essentially is equivalent to [*big.Int]'s SetString method, except that it panics on error.
//
// The use-case for this function is initializing (global constant, really) *big.Int's from string constants. As such, panic on failure is appropriate.
func InitIntFromString(input string) *big.Int {
	var t *big.Int = big.NewInt(0)
	var success bool
	t, success = t.SetString(input, 0)
	// Note: panic is the appropriate error handling here. Also, since this code is only run during package import, there is actually no way to catch it.
	if !success {
		panic("String used to initialized big.Int not recognized as a valid number")
	}
	return t
}

type ToIntConvertible interface {
	ToBigInt() *big.Int
}

func IsEqualAsBigInt(x, y ToIntConvertible) bool {
	return x.ToBigInt().Cmp(y.ToBigInt()) == 0
}
