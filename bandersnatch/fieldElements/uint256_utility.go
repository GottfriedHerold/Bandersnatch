package fieldElements

import "github.com/GottfriedHerold/Bandersnatch/internal/utils"

// InitFieldElementFromString initializes a Uint256 from a given string.
// This internally uses big.Int's SetString and understands exactly those string formats.
// In particular, the given string can be a decimal, hex, octal or binary representation, but needs to be prefixed if not decimal.
//
// This function panics on failure, which is appropriate for its use case:
// It is supposed to be used to initialize package-level variables (probably intendend to be constant) from constant string literals.
//
// The input string must represent a number in [0, 2^256).
func InitUint256FromString(input string) (output Uint256) {
	inputInt := utils.InitIntFromString(input)
	output.SetBigInt(inputInt)
	return
}
