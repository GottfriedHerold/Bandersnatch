package bandersnatch

import (
	"testing"
)

func TestSanity(t *testing.T) {

	// This is to prevent users from shooting themselves in the foot by changing BaseFieldSize_untyped
	// and believing everything keeps working fine.
	// (Note that field_element_64 relies on even more special properties of the modulus)
	if BaseFieldSize.ProbablyPrime(10) == false {
		t.Fatal("Modulus is not prime")
	}
	if BaseFieldBitLength > 64*len(BaseFieldSize_64) {
		t.Fatal("BaseFieldSize_64 too small")
	}
	// The next 3 tests really tests if BaseFieldSize == BaseFieldSize_untyped:
	// (at least in bitsize. Due to language restrictions, we need an intermediate const)
	if BaseFieldSize.BitLen() != BaseFieldBitLength {
		t.Fatal("Modulus bitlength inconsistent between untyped and big.Int")
	}

	if BaseFieldSize_untyped>>BaseFieldBitLength != 0 {
		t.Fatal("BaseFieldBitLength shorter than actual untyped value")
	}

	if BaseFieldSize_untyped>>(BaseFieldBitLength-1) != 1 {
		t.Fatal("BaseFieldBitLength larger than actual untyped value")
	}

	if BaseFieldBitLength > 256 {
		t.Error("BaseFieldSize_untyped > 256 bits is not portable")
	}

}
