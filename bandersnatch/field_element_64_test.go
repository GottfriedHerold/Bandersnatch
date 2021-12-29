package bandersnatch

import (
	"bytes"
	"encoding/binary"
	"math/big"
	"math/bits"
	"math/rand"
	"testing"
)

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
	if !bsFieldElement_64_one.IsOne() {
		t.Fatalf("1 is not 1")
	}
}

func TestOps_8_vs_64(t *testing.T) {
	var drng *rand.Rand = rand.New(rand.NewSource(134124))

	const iterations = 100

	for i := 0; i < iterations; i++ {
		var x *big.Int = new(big.Int).Rand(drng, BaseFieldSize)
		var y *big.Int = new(big.Int).Rand(drng, BaseFieldSize)

		var x_8, y_8, z_8 bsFieldElement_8
		var x_64, y_64, z_64 bsFieldElement_64
		var result_8, result_64 *big.Int
		x_8.SetInt(x)
		y_8.SetInt(y)
		x_64.SetInt(x)
		y_64.SetInt(y)

		z_8.Mul(&x_8, &y_8)
		z_64.Mul(&x_64, &y_64)
		result_8 = z_8.ToInt()
		result_64 = z_64.ToInt()

		if result_8.Cmp(result_64) != 0 {
			t.Fatal("Multiplication differs between bsFieldElement_8 and bsFieldElement64")
		}

		z_8.Add(&x_8, &y_8)
		z_64.Add(&x_64, &y_64)
		result_8 = z_8.ToInt()
		result_64 = z_64.ToInt()

		if result_8.Cmp(result_64) != 0 {
			t.Fatal("Addition differs between bsFieldElement_8 and bsFieldElement64")
		}

		z_8.Sub(&x_8, &y_8)
		z_64.Sub(&x_64, &y_64)
		result_8 = z_8.ToInt()
		result_64 = z_64.ToInt()

		if result_8.Cmp(result_64) != 0 {
			t.Fatal("Subtraction differs between bsFieldElement_8 and bsFieldElement64")
		}

		if x_8.IsZero() {
			t.Fatal("Random 256 bit number is zero")
		}

		z_8.Inv(&x_8)
		z_64.Inv(&x_64)
		result_8 = z_8.ToInt()
		result_64 = z_64.ToInt()

		if result_8.Cmp(result_64) != 0 {
			t.Fatal("Inversion differs between bsFieldElement_8 and bsFieldElement64", *result_8, *result_64)
		}
	}
}

func TestDivision(t *testing.T) {
	var drng *rand.Rand = rand.New(rand.NewSource(13513))
	const iterations = 50
	for i := 0; i < iterations; i++ {
		var num, denom, result bsFieldElement_64
		num.setRandomUnsafe(drng)
		denom.setRandomUnsafe(drng)
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
	x.setRandomUnsafe(drng)
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
		x.setRandomUnsafe(drng)
		y.setRandomUnsafe(drng)
		res1.Add(&x, &y)
		res2.Add(&y, &x)
		if !res1.IsEqual(&res2) {
			t.Error("Addition does not commute for 64-bit version")
			break
		}
	}

	for i := 0; i < iterations; i++ {
		x.setRandomUnsafe(drng)
		y.setRandomUnsafe(drng)
		res1.Mul(&x, &y)
		res2.Mul(&y, &x)
		if !res1.IsEqual(&res2) {
			t.Error("Multiplication does not commute in 64-bit version")
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
			t.Error("Addition non assiciative (64-bit version)")
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
			t.Error("Multiplication non assiciative (64-bit version)")
			break
		}
	}
}

func TestMulHelpers(testing_instance *testing.T) {
	var drng *rand.Rand = rand.New(rand.NewSource(11141))
	const iterations = 1000
	bound := big.NewInt(1)
	bound.Lsh(bound, 256) // bound = 2^256

	R := big.NewInt(1)
	R.Lsh(R, 64) // R = 2^64

	oneInt := big.NewInt(1)

	// Test mul_four_one_64 by comparing to big.Int computation on random inputs x, y
	for i := 1; i < iterations; i++ {
		xInt := new(big.Int).Rand(drng, bound)
		var x [4]uint64 = intTouintarray(xInt)

		var y uint64 = drng.Uint64()
		yInt := new(big.Int).SetUint64(y)

		// x*y as computed via big.Int.Mul
		resultInt := new(big.Int).Mul(xInt, yInt)

		low, high := mul_four_one_64(&x, y)
		lowInt := new(big.Int).SetUint64(low)
		highInt := uintarrayToInt(&high)
		resultInt2 := new(big.Int).Mul(highInt, R)

		// x*y as computed using mul_four_one
		resultInt2.Add(resultInt2, lowInt)

		if resultInt.Cmp(resultInt2) != 0 {
			testing_instance.Error("mul_four_one is incorrect")
			break
		}
	}

	// Test montgomery_step_64
	for i := 1; i < iterations; i++ {
		tInt := new(big.Int).Rand(drng, bound)
		var t [4]uint64 = intTouintarray(tInt)

		var q uint64 = drng.Uint64()
		qInt := new(big.Int).SetUint64(q)

		qInt.Mul(qInt, BaseFieldSize)
		qInt.Div(qInt, R)
		tInt.Add(tInt, qInt)
		tInt.Add(tInt, oneInt)
		if tInt.BitLen() > 256 {
			// In case of overflow, we do not guarantee anything anyway.
			continue
		}
		montgomery_step_64(&t, q)
		tInt2 := uintarrayToInt(&t)
		if tInt.Cmp(tInt2) != 0 {
			testing_instance.Error("montgomery_step_64 is incorrect", *tInt, *tInt2)
			break
		}

	}

	// Test add_mul_shift_64
	for i := 1; i < iterations; i++ {
		targetInt := new(big.Int).Rand(drng, bound)
		var target [4]uint64 = intTouintarray(targetInt)

		xInt := new(big.Int).Rand(drng, bound)
		var x [4]uint64 = intTouintarray(xInt)

		var y uint64 = drng.Uint64()
		yInt := new(big.Int).SetUint64(y)

		// compute using big.Int (result_low1, result_target2 return value/new target)
		resultInt := new(big.Int)
		resultInt.Mul(xInt, yInt)
		resultInt.Add(resultInt, targetInt)
		resultlowInt := new(big.Int).Mod(resultInt, R)
		var result_low1 uint64 = resultlowInt.Uint64()
		resultInt.Rsh(resultInt, 64)
		result_target1 := intTouintarray(resultInt)

		result_low2 := add_mul_shift_64(&target, &x, y)
		if target != result_target1 {
			testing_instance.Error("add_mul_shift_64 is wrong (target)")
			break
		}
		if result_low1 != result_low2 {
			testing_instance.Error("add_mul_shift_64 is wrong (low)", result_low1, result_low2)
			break
		}
	}

}

func TestSerializeInt_64(t *testing.T) {
	var drng *rand.Rand = rand.New(rand.NewSource(123523))
	const iterations = 1000
	var x bsFieldElement_64
	for i := 0; i < iterations; i++ {
		// Try zero in first case
		if i != 0 {
			x.setRandomUnsafe(drng)
		}
		var y bsFieldElement_64 = x
		var xInt *big.Int = x.ToInt()
		x.SetInt(xInt)
		if !y.IsEqual(&x) {
			t.Fatal("Serialization roundtrip fails", i, *(x.ToInt()), *(y.ToInt()))
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
		a.SetInt(xInt)
		b.SetUInt64(x)

		y, err := b.ToUint64()
		if err {
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
		x.setRandomUnsafe(drng)
		y.Mul(&x, &five)
		x.multiply_by_five()
		if !x.IsEqual(&y) {
			t.Fatal("Multiplication by five does not work", i, x, y)
		}
	}
}

func TestConstants(t *testing.T) {
	// Note that IsEqual can internally call Normalize(). This is not a problem, because we do not export it.
	var oldaltzero = bsFieldElement_64_zero_alt
	if !bsFieldElement_64_zero.IsEqual(&bsFieldElement_64_zero_alt) {
		t.Fatal("Different representations of zero do not compare equal")
	}
	bsFieldElement_64_zero_alt = oldaltzero
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

func TestSerializeFieldElements(t *testing.T) {
	const iterations = 100
	var drng *rand.Rand = rand.New(rand.NewSource(87))
	for i := 0; i < iterations; i++ {
		var buf bytes.Buffer
		var fe bsFieldElement_64
		fe.setRandomUnsafe(drng)
		// do little endian and big endian half the time
		var byteOrder binary.ByteOrder = binary.LittleEndian
		if i%2 == 0 {
			byteOrder = binary.BigEndian
		}
		bytes_written, err := fe.Serialize(&buf, byteOrder)
		if err != nil {
			t.Fatal("Serialization of field element failed with error ", err)
		}
		if bytes_written != BaseFieldByteLength {
			t.Fatal("Serialization of field element did not write exptected number of bytes")
		}
		var fe2 bsFieldElement_64
		bytes_read, err := fe2.Deserialize(&buf, byteOrder)
		if err != nil {
			t.Fatal("Deserialization of field element failed with error ", err)
		}
		if bytes_read != BaseFieldByteLength {
			t.Fatal("Deserialization of field element did not read expceted number of bytes")
		}
		if !fe.IsEqual(&fe2) {
			t.Fatal("Deserializing of field element did not reproduce what was serialized")

		}
	}
	for i := 0; i < iterations; i++ {
		var buf bytes.Buffer
		var fe, fe2 bsFieldElement_64
		fe.setRandomUnsafe(drng)
		if fe.Sign() < 0 {
			fe.NegEq()
		}
		if fe.Sign() < 0 {
			t.Fatal("Sign does not work as expected")
		}
		if bits.LeadingZeros64(fe.undoMontgomery()[3]) < 2 {
			t.Fatal("Positive sign field elements do not start with 00")
		}
		var random_prefix prefixBits = (prefixBits(i) / 2) % 4
		var byteOrder binary.ByteOrder = binary.LittleEndian
		if i%2 == 0 {
			byteOrder = binary.BigEndian
		}

		bytes_written, err := fe.SerializeWithPrefix(&buf, PrefixBits(random_prefix), 2, byteOrder)
		if err != nil || bytes_written != BaseFieldByteLength {
			t.Fatal("Serialization of field element failed with long prefix: ", err)
		}
		bytes_read, err := fe2.DeserializeWithPrefix(&buf, PrefixBits(random_prefix), 2, byteOrder)
		if err != nil || bytes_read != BaseFieldByteLength {
			t.Fatal("Deserialization of field element failed with long prefix: ", err)
		}
		if !fe.IsEqual(&fe2) {
			t.Fatal("Roundtripping field elements failed with long prefix")
		}
		buf.Reset() // not really needed
		bytes_written, err = fe.SerializeWithPrefix(&buf, PrefixBits(1), 1, byteOrder)
		if bytes_written != BaseFieldByteLength || err != nil {
			t.Fatal("Serialization of field elements failed on resetted buffer")
		}
		_, err = fe2.DeserializeWithPrefix(&buf, PrefixBits(0), 1, byteOrder)
		if err != ErrPrefixMismatch {
			t.Fatal("Prefix mismatch was not detected in deserialization of field elements")
		}
		buf.Reset()
		fe.Serialize(&buf, binary.BigEndian)
		buf.Bytes()[0] |= 0x80
		bytes_read, err = fe2.Deserialize(&buf, binary.BigEndian)
		if bytes_read != BaseFieldByteLength || err != ErrNonNormalizedDeserialization {
			t.Fatal("Non-normalized field element not recognized as such during deserialization")
		}
	}
}
