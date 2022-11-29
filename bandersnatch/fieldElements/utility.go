package fieldElements

import (
	"fmt"
	"math/big"
	"math/rand"
)

// InitFieldElementFromString initializes a field element from a given string.
// The ouput is guaranteed to be Normalized.
// This internally uses big.Int's SetString and understands exactly those string formats.
// In particular, the given string can be a decimal, hex, octal or binary representation, but needs to be prefixed if not decimal.
//
// This function panics on failure, which is appropriate for its use case:
// It is supposed to be used to initialize package-level variables (probably intendend to be constant) from constant string literals.
//
// The input string does not have to represent a number in [0, BaseFieldSize). It may represent any integer, possibly negative.
func InitFieldElementFromString[FE any, FEPtr interface {
	*FE
	FieldElementInterface[*FE]
}](input string) (output FE) {
	var t *big.Int = new(big.Int)
	var success bool
	t, success = t.SetString(input, 0)
	if !success {
		panic(fmt.Errorf(ErrorPrefix+"String %v used to initialize field element was not recognized as a valid number", input))
	}
	FEPtr(&output).SetBigInt(t)
	FEPtr(&output).Normalize() // not needed actually, because of current implementation of SetBigInt for all our field element types, but we want to be 100% sure.
	return
}

// CreateRandomFieldElement_Unsafe creates a random field element
//
// NOTE: The randomness quality is *NOT* sufficient for cryptographic purposes, hence the "unsafe". This function is merely used for unit tests.
// We do not even guarantee that it is close to uniform, reasonably random, or that the output sequence is preserved across library releases.
//
// NOTE2: Neither the value of the created field element nor the amound of randomness consumed depend on the field element type.
// This is intentional and allows differential testing.
func CreateRandomFieldElement_Unsafe[FE any, FEPtr interface {
	*FE
	FieldElementInterface[FEPtr]
}](rnd *rand.Rand) (fe FE) {
	var randInt *big.Int = new(big.Int).Rand(rnd, baseFieldSize_Int)
	FEPtr(&fe).SetBigInt(randInt)
	FEPtr(&fe).RerandomizeRepresentation(rnd.Uint64())
	return
}

// CreateNonZeroRandomFieldElement_Unsafe creates a random field element
//
// NOTE: The randomness quality is *NOT* sufficient for cryptographic purposes, hence the "unsafe". This function is merely used for unit tests.
// We do not even guarantee that it is close to uniform, reasonably random, or that the output sequence is preserved across library releases.
//
// NOTE2: Neither the value of the created field element nor the amound of randomness consumed depend on the field element type.
// This is intentional and allows differential testing.
func CreateRandomNonZeroFieldElement_Unsafe[FE any, FEPtr interface {
	*FE
	FieldElementInterface[FEPtr]
}](rnd *rand.Rand) (fe FE) {
	var randInt *big.Int = new(big.Int)
	for {
		randInt.Rand(rnd, baseFieldSize_Int)
		if randInt.Sign() == 0 {
			continue
		}
		FEPtr(&fe).SetBigInt(randInt)
		FEPtr(&fe).RerandomizeRepresentation(rnd.Uint64())
		return
	}
}
