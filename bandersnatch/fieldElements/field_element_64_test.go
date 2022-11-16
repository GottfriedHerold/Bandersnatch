package fieldElements

import (
	"fmt"
	"math/big"
	"math/rand"
	"testing"
)

var _ fmt.Formatter = bsFieldElement_64{}
var _ fmt.Stringer = bsFieldElement_64{}

func TestInit_64(t *testing.T) {
	var x, y, z bsFieldElement_64
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
		x.SetRandomUnsafe(drng)
		y = x
		z.Sub(&y, &x)
		if !z.IsZero() {
			t.Error("x-x != 0")
			break
		}
	}
	for i := 0; i < iterations; i++ {
		x.SetRandomUnsafe(drng)
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
	if !bsFieldElement_64_one.IsOne() {
		t.Fatalf("1 is not 1")
	}
}

func TestOps_8_vs_64(t *testing.T) {
	var drng *rand.Rand = rand.New(rand.NewSource(134124))

	const iterations = 100

	for i := 0; i < iterations; i++ {
		var x *big.Int = new(big.Int).Rand(drng, baseFieldSize_Int)
		var y *big.Int = new(big.Int).Rand(drng, baseFieldSize_Int)

		var x_8, y_8, z_8 bsFieldElement_8
		var x_64, y_64, z_64 bsFieldElement_64
		var result_8, result_64 *big.Int
		x_8.SetBigInt(x)
		y_8.SetBigInt(y)
		x_64.SetBigInt(x)
		y_64.SetBigInt(y)

		z_8.Mul(&x_8, &y_8)
		z_64.Mul(&x_64, &y_64)
		result_8 = z_8.ToBigInt()
		result_64 = z_64.ToBigInt()

		if result_8.Cmp(result_64) != 0 {
			t.Fatal("Multiplication differs between bsFieldElement_8 and bsFieldElement64")
		}

		z_8.Add(&x_8, &y_8)
		z_64.Add(&x_64, &y_64)
		result_8 = z_8.ToBigInt()
		result_64 = z_64.ToBigInt()

		if result_8.Cmp(result_64) != 0 {
			t.Fatal("Addition differs between bsFieldElement_8 and bsFieldElement64")
		}

		z_8.Sub(&x_8, &y_8)
		z_64.Sub(&x_64, &y_64)
		result_8 = z_8.ToBigInt()
		result_64 = z_64.ToBigInt()

		if result_8.Cmp(result_64) != 0 {
			t.Fatal("Subtraction differs between bsFieldElement_8 and bsFieldElement64")
		}

		if x_8.IsZero() {
			t.Fatal("Random 256 bit number is zero")
		}

		z_8.Inv(&x_8)
		z_64.Inv(&x_64)
		result_8 = z_8.ToBigInt()
		result_64 = z_64.ToBigInt()

		if result_8.Cmp(result_64) != 0 {
			t.Fatal("Inversion differs between bsFieldElement_8 and bsFieldElement64", *result_8, *result_64)
		}

		if x_8.Sign() != x_64.Sign() {
			t.Fatalf("Sign differs between bsFieldElement_8 and bsFieldElement_64")
		}

	}
}

func TestDivision(t *testing.T) {
	var drng *rand.Rand = rand.New(rand.NewSource(13513))
	const iterations = 50
	for i := 0; i < iterations; i++ {
		var num, denom, result bsFieldElement_64
		num.SetRandomUnsafe(drng)
		denom.SetRandomUnsafe(drng)
		result.Divide(&num, &denom)
		result.MulEq(&denom)
		if !num.IsEqual(&result) {
			t.Fatal("(x/y) * y != x for random x,y")
		}
		num.DivideEq(&num)
		if !num.IsOne() {
			t.Fatal("x/x != 1 for random x")
		}
	}
}

func TestAssign_64(t *testing.T) {
	var drng *rand.Rand = rand.New(rand.NewSource(123523))
	var x, y, z bsFieldElement_64
	x.SetRandomUnsafe(drng)
	y.SetOne()
	z = x
	z.Add(&x, &y)
	if z.IsEqual(&x) {
		t.Fatal("Assignment seems shallow")
	}

}

func TestOpsOnRandomValues_64(t *testing.T) {
	var drng *rand.Rand = rand.New(rand.NewSource(555))
	const iterations = 1000

	var x, y, z, res1, res2 bsFieldElement_64

	for i := 0; i < iterations; i++ {
		x.SetRandomUnsafe(drng)
		y.SetRandomUnsafe(drng)
		res1.Add(&x, &y)
		res2.Add(&y, &x)
		if !res1.IsEqual(&res2) {
			t.Error("Addition does not commute for 64-bit version")
			break
		}
	}

	for i := 0; i < iterations; i++ {
		x.SetRandomUnsafe(drng)
		y.SetRandomUnsafe(drng)
		res1.Mul(&x, &y)
		res2.Mul(&y, &x)
		if !res1.IsEqual(&res2) {
			t.Error("Multiplication does not commute in 64-bit version")
			break
		}
	}

	for i := 0; i < iterations; i++ {
		x.SetRandomUnsafe(drng)
		y.SetRandomUnsafe(drng)
		z.SetRandomUnsafe(drng)
		res1.Add(&x, &y)
		res1.Add(&res1, &z)
		res2.Add(&y, &z)
		res2.Add(&x, &res2)
		if !res1.IsEqual(&res2) {
			t.Error("Addition non assiciative (64-bit version)")
			break
		}
	}

	for i := 0; i < iterations; i++ {
		x.SetRandomUnsafe(drng)
		y.SetRandomUnsafe(drng)
		z.SetRandomUnsafe(drng)
		res1.Mul(&x, &y)
		res1.Mul(&res1, &z)
		res2.Mul(&y, &z)
		res2.Mul(&x, &res2)
		if !res1.IsEqual(&res2) {
			t.Error("Multiplication non assiciative (64-bit version)")
			break
		}
	}
}

func TestSign(t *testing.T) {
	// Testing random x and comparing against _8 is done already.
	// We only check special values.

	var x FieldElement
	x.SetZero()
	if x.Sign() != 0 {
		t.Fatalf("Sign(0) != 0")
	}
	x = bsFieldElement_64_zero_alt
	if x.Sign() != 0 {
		t.Fatalf("Sign(0_alt) != 0")
	}
	x = bsFieldElement_64_minusone
	if x.Sign() != -1 {
		t.Fatalf("Sign(-1) != -1")
	}
	x = FieldElementTwo
	x.InvEq() // x = 1/2 == (1-BaseFieldSize)/2. This is the point where the sign switches
	if x.Sign() != -1 {
		t.Fatalf("Sign(1/2) != -1")
	}
	x.SubEq(&FieldElementOne)
	if x.Sign() != +1 {
		t.Fatalf("Sign(1/2 - 1) != +1")
	}
}

func TestSerializeInt_64(t *testing.T) {
	var drng *rand.Rand = rand.New(rand.NewSource(123523))
	const iterations = 1000
	var x bsFieldElement_64
	for i := 0; i < iterations; i++ {
		// Try zero in first case
		if i != 0 {
			x.SetRandomUnsafe(drng)
		}
		var y bsFieldElement_64 = x
		var xInt *big.Int = x.ToBigInt()
		x.SetBigInt(xInt)
		if !y.IsEqual(&x) {
			t.Fatal("Serialization roundtrip fails", i, *(x.ToBigInt()), *(y.ToBigInt()))
		}
	}
}

func TestSetUIunt(t *testing.T) {
	var drng *rand.Rand = rand.New(rand.NewSource(444))
	const iterations = 10000
	for i := 0; i < iterations; i++ {
		var x uint64 = drng.Uint64()
		xInt := big.NewInt(0)
		xInt.SetUint64(x)
		var a, b bsFieldElement_64
		a.SetBigInt(xInt)
		b.SetUInt64(x)

		y, err := b.ToUInt64()
		if err != nil {
			t.Fatal("Conversion back to Uint reports too big number")
		}
		if x != y {
			t.Fatal("Roundtrip uint64 -> FieldElement -> uint64 does not work")
		}

		if !a.IsEqual(&b) {
			t.Fatal("Setting from UInt and Int is inconsistent")
		}

	}
}

func TestMultiplyByFive(t *testing.T) {
	var drng *rand.Rand = rand.New(rand.NewSource(444))
	const iterations = 10000

	var five, x, y bsFieldElement_64
	five.SetUInt64(5)

	for i := 0; i < iterations; i++ {
		x.SetRandomUnsafe(drng)
		y.Mul(&x, &five)
		x.Multiply_by_five()
		if !x.IsEqual(&y) {
			t.Fatal("Multiplication by five does not work", i, x, y)
		}
	}
}

func TestConstants(t *testing.T) {
	// Note that IsEqual can internally call Normalize(), hence the need to work on a copy.
	var altzero = bsFieldElement_64_zero_alt
	if !bsFieldElement_64_zero.IsEqual(&altzero) {
		t.Fatal("Different representations of zero do not compare equal")
	}
	var temp bsFieldElement_64 = bsFieldElement_64_zero
	if !temp.IsZero() {
		t.Fatal("Zero is not recognized as zero")
	}
	temp = bsFieldElement_64_zero_alt
	if !temp.IsZero() {
		t.Fatal("Alternative representation of zero is not recognized as zero")
	}
	temp.Add(&bsFieldElement_64_minusone, &bsFieldElement_64_one)
	if !temp.IsZero() {
		t.Fatal("Representation of one or minus one are inconsistent: They do not add to zero")
	}
}
