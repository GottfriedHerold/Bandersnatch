package bandersnatch

import "math/big"

// initFieldElementFromString initializes a field element from a given string. The given string can be a decimal or hex representation, but needs to be prefixed if hex.
// Since we only use it internally to initialize package-level variables (intendend to be constant) from (compile-time!) constant strings, panic() on error is appropriate.
func initFieldElementFromString(input string) (output bsFieldElement_64) {
	var t *big.Int = big.NewInt(0)
	var success bool
	t, success = t.SetString(input, 0)
	if !success {
		panic("String used to initialize field element not recognized as a valid number")
	}
	output.SetInt(t)
	return
}

// initIntFromString initializes a big.Int from a given string similar to initFieldElementFromString. The given string can be decimal or hex, but needs to be prefixed if hex.
// This essentially is equivalent to big.Int's SetString method, except that it panics on error (which is appropriate for initialization from compile-time constant strings).
func initIntFromString(input string) *big.Int {
	var t *big.Int = big.NewInt(0)
	var success bool
	t, success = t.SetString(input, 0)
	// Note: panic is the appropriate error handling here. Also, since this code is only run during package import, there is actually no way to catch it.
	if !success {
		panic("String used to initialized big.Int not recognized as a valid number")
	}
	return t
}
