package fieldElement

import (
	"math/big"
)

// InitFieldElementFromString initializes a field element from a given string.
// The ouput is guaranteed to be Normalized.
// This internally uses big.Int's SetString and understands exactly those string formats.
// In particular, the given string can be a decimal, hex, octal or binary representation, but needs to be prefixed if not decimal.
//
// This function panics on failure, which is appropriate for its use case:
// It is supposed to be used to initialize package-level variables (probably intendend to be constant) from constant string literals.
func InitFieldElementFromString(input string) (output bsFieldElement_64) {
	var t *big.Int = big.NewInt(0)
	var success bool
	t, success = t.SetString(input, 0)
	if !success {
		panic(ErrorPrefix + "String used to initialize field element not recognized as a valid number")
	}
	output.SetBigInt(t)
	output.Normalize() // not needed actually, because of current implementation of SetBigInt, but we want to be 100% sure.
	return
}
