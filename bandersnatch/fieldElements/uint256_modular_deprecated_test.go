//go:build ignore

package fieldElements

// Test by Luan for his Barret-Reduction-based uint256 implementation.
// These tests do work, but are too inefficient to be run on my machine -- Gotti

import (
	mrand "math/rand"
	"testing"
)

var (
	testval [1024][4]uint64
)

func init() {
	//initialize testval to some interesting values
	i := 0
	//zero
	testval[i] = [4]uint64{0, 0, 0, 0}
	i++
	//single bit set
	for j := 0; j < 64; j++ {
		testval[i] = [4]uint64{1 << j, 0, 0, 0}
		i++
	}

	for j := 0; j < 64; j++ {
		testval[i] = [4]uint64{0, 1 << j, 0, 0}
		i++
	}

	for j := 0; j < 64; j++ {
		testval[i] = [4]uint64{0, 0, 1 << j, 0}
		i++
	}

	for j := 0; j < 64; j++ {
		testval[i] = [4]uint64{0, 0, 0, 1 << j}
		i++
	}

	//single unset bit
	for j := 0; j < 256; j++ {
		tmp := testval[i+j] //For those coming from other languages, ^ is the XOR operator for ints.
		tmp[0] = ^tmp[0]    //But on the unsigned 64, this is the bitwise negation operator.
		tmp[1] = ^tmp[1]
		tmp[2] = ^tmp[2]
		tmp[3] = ^tmp[3]
		testval[i] = tmp
		i++
	}

	//randoms
	for i < cap(testval) {
		testval[i][0] = mrand.Uint64()
		testval[i][1] = mrand.Uint64()
		testval[i][2] = mrand.Uint64()
		testval[i][3] = mrand.Uint64()
		i++
	}

}

func TestModulus(t *testing.T) {
	//check if the current constants used for the modulus match the computed values

	var mod modulus
	mod.FromUint256(uint256{m_0, m_1, m_2, m_3})

	if mod.re != [5]uint64{re_0, re_1, re_2, re_3, re_4} {
		t.Fatalf("The reciprocal of the modulus does not match")
	}

	if mod.mmu0 != [4]uint64{mmu0_0, mmu0_1, mmu0_2, mmu0_3} {
		t.Fatalf("The precomputed multiple0 does not match")
	}

	if mod.mmu1 != [4]uint64{mmu1_0, mmu1_1, mmu1_2, mmu1_3} {
		t.Fatalf("The precomputed multiple1 does not match")
	}
}

/*

PROPERTY TESTS

*/

// a+0 == 0+a == 0
func TestAdditiveIdentity(t *testing.T) {

	var zero uint256
	count := 0

	for _, val := range testval {
		var x uint256 = val
		var y uint256 = x

		x.AddEqAndReduce_a(&zero)

		if x.toUint64() != y.toUint64() {
			t.Fatalf("TestAdditiveIdentity failed! %v != %v", x.ToBigInt(), y.ToBigInt())
		}
		count++
	}
}

// a*1 == 1*a == a
func TestMultiplicativeIdentity(t *testing.T) {
	var one = uint256{1, 0, 0, 0}
	count := 0

	for _, val := range testval {
		var x uint256 = val
		y := x

		x.MulEqAndReduce_a(&one)

		if x.toUint64() != y.toUint64() {
			t.Fatalf("TestMultiplicativeIdentity failed! %v != %v", x.toUint64(), y.toUint64())
		}
		count++
	}

}

// a+(-a) == (-a)+a == 0
// a == -(-(a))
// a-b == (-b)+a == -(b-a)
func TestAdditiveInverse(t *testing.T) {
	var a, b, u, v, w uint256

	// a+(-a) == (-a)+a == 0
	for _, val := range testval {
		a = val
		b = a.ComputeModularNegative_Weak_a()
		u = b
		v = a

		a.AddEqAndReduce_a(&b)
		u.AddEqAndReduce_a(&v)

		if a.toUint64() != u.toUint64() {
			t.Fatalf("Aditive inverse is not cumutative! %v != %v", a, u)
		}
	}

	// a == -(-(a))
	for _, val := range testval {
		a = val
		b = a.ComputeModularNegative_Weak_a()
		b = b.ComputeModularNegative_Weak_a()

		if a.toUint64() != b.toUint64() {
			t.Fatalf("Double inverse does not cancel! %v != %v", a, b)
		}
	}

	// a-b == (-b)+a == -(b-a)
	for _, _a := range testval {
		for _, _b := range testval {
			a = _a
			b = _b

			u = a
			u.SubEqAndReduce_a(&b) //u=a-b

			v = b.ComputeModularNegative_Weak_a()
			v.AddEqAndReduce_a(&a) //v=(-b)+a

			w = b
			w.SubEqAndReduce_a(&a)
			w = w.ComputeModularNegative_Weak_a() //w=-(b-a)

			if (u.toUint64() != v.toUint64()) || (u.toUint64() != w.toUint64()) || (v.toUint64() != w.toUint64()) {
				t.Errorf("a-b = %v", u)
				t.Errorf("-b+a = %v", v)
				t.Errorf("-(b-a) = %v", w)
				t.Errorf("a=%v b=%v", a, b)
				t.Fatalf("Additive inverse is invalid!")
			}

		}
	}

}

// a*(1/a) == (1/a)*a == 1
func TestMultiplicativeInverse(t *testing.T) {
	var a, b, one uint256
	one = [4]uint64{1, 0, 0, 0}
	var noninvertible int
	for _, val := range testval {
		a = val
		invertible := b.ModularInverse_a_NAIVEHAC(&a)
		if invertible == false {
			noninvertible++
			continue
		}

		a.MulEqAndReduce_a(&b)

		if a.toUint64() != one {
			t.Fatalf("Multiplicative inverse failed with a=%v b=%v", a, b)
		}
	}
	t.Logf("%v non inv in test", noninvertible)

}

// a+b == b+a
func TestCummutativeAddition(t *testing.T) {
	var a, b, u, v uint256

	for _, v1 := range testval {
		a = v1
		for _, v2 := range testval {
			b = v2

			u = a
			u.AddEqAndReduce_a(&b)

			v = b
			v.AddEqAndReduce_a(&a)

			if v.toUint64() != u.toUint64() {
				t.Fatalf("Addition does not commute a=%v b=%v", a, b)
			}
		}
	}
}

// a*b == b*a
func TestCummutativeMultiplication(t *testing.T) {
	var a, b, u, v uint256

	for _, v1 := range testval {
		a = v1
		for _, v2 := range testval {
			b = v2

			u = a
			u.MulEqAndReduce_a(&b)

			v = b
			v.MulEqAndReduce_a(&a)

			if v.toUint64() != u.toUint64() {
				t.Fatalf("Multiplication does not commute a=%v b=%v", a, b)
			}
		}
	}
}

// (a+b)+c == a+(b+c)
func TestAssociativeAddition(t *testing.T) {
	var a, b, c, u, v uint256

	for j, _a := range testval {
		a = _a
		for k, _b := range testval[:j] {
			b = _b
			for _, _c := range testval[:k] {
				c = _c

				u = a
				u.AddEqAndReduce_a(&b)
				u.AddEqAndReduce_a(&c)
				v = c
				v.AddEqAndReduce_a(&b)
				v.AddEqAndReduce_a(&a)

				if u.toUint64() != v.toUint64() {
					t.Fatalf("Addition fails associative property %v != %v", u, v)
				}
			}
		}
	}
}

// Commented out, because it's too slow -- I'm getting timeout errors on my machine.

/*

// (a*b)*c == a*(b*c)
func TestAssociativeMultiplication(t *testing.T) {
	var a, b, c, u, v uint256

	for j, _a := range testval {
		a = _a
		for k, _b := range testval[:j] {
			b = _b
			for _, _c := range testval[:k] {
				c = _c

				u = a
				u.MulEqAndReduce_a(&b)
				u.MulEqAndReduce_a(&c)
				v = c
				v.MulEqAndReduce_a(&b)
				v.MulEqAndReduce_a(&a)

				if u.toUint64() != v.toUint64() {
					t.Fatalf("Addition fails associative property %v != %v", u, v)
				}
			}
		}
	}
}

*/

// a(b+c) == ab+ac
func TestDistributiveLeft(t *testing.T) {
	var a, b, c, u, v, w uint256

	for j, _a := range testval {
		a = _a
		for k, _b := range testval[:j] {
			b = _b
			for _, _c := range testval[:k] {
				c = _c

				//u = ab+ac
				u = a
				u.MulEqAndReduce_a(&b)
				v = a
				v.MulEqAndReduce_a(&c)
				u.AddEqAndReduce_a(&v)

				//v = a(b+c)
				v = a
				w = b
				w.AddEqAndReduce_a(&c)
				v.MulEqAndReduce_a(&w)

				if u.toUint64() != v.toUint64() {
					t.Fatalf("Failed left distributive property (a(b+c) == ab+ac) %v != %v", u, v)
				}
			}
		}
	}

}

// (a+b)c == ac+bc
func TestDistributiveRight(t *testing.T) {
	var a, b, c, u, v uint256

	for j, _a := range testval {
		a = _a
		for k, _b := range testval[:j] {
			b = _b
			for _, _c := range testval[:k] {
				c = _c

				//u = ac+bc
				u = a
				u.MulEqAndReduce_a(&c)
				v = b
				v.MulEqAndReduce_a(&c)
				u.AddEqAndReduce_a(&v)

				//v = (a+b)c
				v = a
				v.AddEqAndReduce_a(&b)
				v.MulEqAndReduce_a(&c)

				if u.toUint64() != v.toUint64() {
					t.Fatalf("Failed right distributive property ((a+b)c == ac+bc) %v != %v", u, v)
				}
			}
		}
	}
}

// 2a = a+a
// 2(a+b) == 2a + 2b
func TestDoubling(t *testing.T) {
	var a, b, u, v uint256
	// 2a = a+a
	for _, _a := range testval {
		a = _a
		b = _a

		//2a
		a.DoubleEqAndReduce_a()
		//a+a
		b.AddEqAndReduce_a(&b)

		if a.toUint64() != b.toUint64() {
			t.Fatalf("Failed doubling test (2a = a+a) %v != %v", a, b)
		}

	}

	for _, _a := range testval {
		a = _a
		for _, _b := range testval {
			b = _b

			//2(a+b)
			u = a
			u.AddEqAndReduce_a(&b)
			u.DoubleEqAndReduce_a()

			//2a+2b
			v = a
			v.DoubleEqAndReduce_a()
			b.DoubleEqAndReduce_a()
			v.AddEqAndReduce_a(&b)

			if v.toUint64() != u.toUint64() {
				t.Fatalf("Failed distributive in doubling test (2(a+b) == 2a + 2b) %v != %v", v, u)
			}
		}
	}
}
