package bandersnatch

import (
	"math/big"
	"math/rand"
	"testing"
)

func TestInit(t *testing.T) {
	var x, y, z bsFieldElement_8
	if !x.IsZero() {
		t.Fatal("Initialization is not Zero")
	}
	y.SetOne()
	if y.IsZero() {
		t.Error("IsZero true after SetOne")
	}
	if !y.IsOne() {
		t.Error("IsOne false after SetOne")
	}
	y.SetZero()
	if !y.IsZero() {
		t.Error("IsZero false after SetZero")
	}
	if !x.IsEqual(&y) {
		t.Error("Zeroes do not compare equal")
	}
	var drng *rand.Rand = rand.New(rand.NewSource(12431254))
	const iterations = 1000
	for i := 0; i < iterations; i++ {
		x.setRandomUnsafe(drng)
		y = x
		z.Sub(&y, &x)
		if !z.IsZero() {
			t.Error("x-x != 0")
			break
		}
	}
	for i := 0; i < iterations; i++ {
		x.setRandomUnsafe(drng)
		if x.IsZero() {
			t.Log("Random Number is Zero")
			continue
		}
		y = x
		y.Inv(&y)
		z.Mul(&x, &y)
		if !z.IsOne() {
			t.Error("x/x != 1")
			break
		}
	}
}

func TestAssign(t *testing.T) {
	var drng *rand.Rand = rand.New(rand.NewSource(123523))
	var x, y, z bsFieldElement_8
	x.setRandomUnsafe(drng)
	y.SetOne()
	z = x
	z.Add(&x, &y)
	if z.IsEqual(&x) {
		t.Fatal("Assignment seems shallow")
	}

}

func TestOpsOnRandomValues(t *testing.T) {
	var drng *rand.Rand = rand.New(rand.NewSource(123523))
	const iterations = 1000

	var x, y, z, res1, res2 bsFieldElement_8

	for i := 0; i < iterations; i++ {
		x.setRandomUnsafe(drng)
		y.setRandomUnsafe(drng)
		res1.Add(&x, &y)
		res2.Add(&y, &x)
		if !res1.IsEqual(&res2) {
			t.Error("Addition does not commute")
			break
		}
	}

	for i := 0; i < iterations; i++ {
		x.setRandomUnsafe(drng)
		y.setRandomUnsafe(drng)
		res1.Mul(&x, &y)
		res2.Mul(&y, &x)
		if !res1.IsEqual(&res2) {
			t.Error("Multiplication does not commute")
			break
		}
	}

	for i := 0; i < iterations; i++ {
		x.setRandomUnsafe(drng)
		y.setRandomUnsafe(drng)
		z.setRandomUnsafe(drng)
		res1.Add(&x, &y)
		res1.Add(&res1, &z)
		res2.Add(&y, &z)
		res2.Add(&x, &res2)
		if !res1.IsEqual(&res2) {
			t.Error("Addition non assiciative")
			break
		}
	}

	for i := 0; i < iterations; i++ {
		x.setRandomUnsafe(drng)
		y.setRandomUnsafe(drng)
		z.setRandomUnsafe(drng)
		res1.Mul(&x, &y)
		res1.Mul(&res1, &z)
		res2.Mul(&y, &z)
		res2.Mul(&x, &res2)
		if !res1.IsEqual(&res2) {
			t.Error("Multiplication non assiciative")
			break
		}
	}
}

func TestSerializeInt(t *testing.T) {
	var drng *rand.Rand = rand.New(rand.NewSource(123523))
	const iterations = 1000
	var x bsFieldElement_8
	for i := 0; i < iterations; i++ {
		// Try zero in first case
		if i != 0 {
			x.setRandomUnsafe(drng)
		}
		var y bsFieldElement_8 = x
		var xInt *big.Int = x.ToInt()
		x.SetInt(xInt)
		if !y.IsEqual(&x) {
			t.Fatal("Serialization roundtrip fails")
		}
	}
}
