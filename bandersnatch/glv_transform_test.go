package bandersnatch

import (
	"math/big"
	"math/rand"
	"testing"
)

func TestGLVParameters(t *testing.T) {
	// L is defined as the lattice consisting of (u,v) s.t. u*P + v*psi(P) = neutral for points P in subgroup, which is equivalent to
	// u * 1 + v * GLSEigenvalue == 0 mod p253.
	// We check that the rows of the lBasis - matrix are in L.
	var v *big.Int = big.NewInt(0)
	v.Mul(lBasis_12_Int, EndomorphismEigenvalue_Int)
	v.Add(v, lBasis_11_Int)
	v.Mod(v, GroupOrder_Int)
	if v.Sign() != 0 {
		t.Fatal("First basis vector of L does not satisfy definition of L")
	}
	v.Mul(lBasis_22_Int, EndomorphismEigenvalue_Int)
	v.Add(v, lBasis_21_Int)
	v.Mod(v, GroupOrder_Int)
	if v.Sign() != 0 {
		t.Fatal("Second basis vector of L does not satisfy definition of L")
	}

	v.Mul(lBasis_11_Int, lBasis_22_Int)
	var temp *big.Int = big.NewInt(0)
	temp.Mul(lBasis_12_Int, lBasis_21_Int)
	v.Sub(v, temp)
	v.Sub(v, GroupOrder_Int)
	if v.Sign() != 0 {
		t.Fatal("Determinant of LLL basis for L is wrong")
	}

}
func TestGLV(t *testing.T) {
	const iterations = 10000
	var bigrange1 *big.Int = big.NewInt(0)
	var bigrange2 *big.Int = big.NewInt(0)
	var exponent *big.Int = big.NewInt(0)
	var temp *big.Int = big.NewInt(0)
	var temp2 *big.Int = big.NewInt(0)
	bigrange1.Add(CurveOrder_Int, CurveOrder_Int) // 8 * p253
	bigrange2.Add(bigrange1, bigrange1)           // 16 * p253

	var drng *rand.Rand = rand.New(rand.NewSource(141152))
	for i := 0; i < iterations; i++ {
		// Make number from -8*p253 to 8*p253
		exponent.Rand(drng, bigrange2)
		exponent.Sub(exponent, bigrange1)
		u, v := GLV_representation(exponent)
		temp.Sub(u, exponent)
		temp2.Mul(v, EndomorphismEigenvalue_Int)
		temp.Add(temp, temp2)
		temp.Mod(temp, GroupOrder_Int)
		if temp.Sign() != 0 {
			t.Fatal("GLV_representation does not output pair of exponents that gives correct result")
		}
		norm1 := infty_norm(u, v)

		// check whether we can get a smaller infty-norm by adding a Voronoi-relevant vector (of which there are 6 options)
		temp.Add(u, lBasis_11_Int)
		temp2.Add(v, lBasis_12_Int)
		norm2 := infty_norm(temp, temp2)
		if norm2.CmpAbs(norm1) < 0 {
			t.Fatal("GLV_representation does not output smallest element (+b_1)")
		}

		temp.Sub(u, lBasis_11_Int)
		temp2.Sub(v, lBasis_12_Int)
		norm2 = infty_norm(temp, temp2)
		if norm2.CmpAbs(norm1) < 0 {
			t.Fatal("GLV_representation does not output smallest element (-b_1)")
		}

		temp.Sub(u, lBasis_21_Int)
		temp2.Sub(v, lBasis_22_Int)
		norm2 = infty_norm(temp, temp2)
		if norm2.CmpAbs(norm1) < 0 {
			t.Fatal("GLV_representation does not output smallest element (-b_2)")
		}

		temp.Add(u, lBasis_21_Int)
		temp2.Add(v, lBasis_22_Int)
		norm2 = infty_norm(temp, temp2)
		if norm2.CmpAbs(norm1) < 0 {
			t.Fatal("GLV_representation does not output smallest element (+b_2)")
		}

		temp.Add(u, lBasis_11_Int)
		temp.Add(temp, lBasis_21_Int)
		temp2.Add(v, lBasis_12_Int)
		temp2.Add(temp2, lBasis_22_Int)
		norm2 = infty_norm(temp, temp2)
		if norm2.CmpAbs(norm1) < 0 {
			t.Fatal("GLV_representation does not output smallest element (+b_1+b_2)")
		}

		temp.Sub(u, lBasis_11_Int)
		temp.Sub(temp, lBasis_21_Int)
		temp2.Sub(v, lBasis_12_Int)
		temp2.Sub(temp2, lBasis_22_Int)
		norm2 = infty_norm(temp, temp2)
		if norm2.CmpAbs(norm1) < 0 {
			t.Fatal("GLV_representation does not output smallest element (-b_1-b_2)")
		}
	}
}

func test_decomposition_correctness(x *big.Int, decomposition []decompositionCoefficient) bool {
	var accumulator *big.Int = big.NewInt(0)
	var toAdd *big.Int = big.NewInt(0)
	for _, comp := range decomposition {
		toAdd.SetUint64(uint64(comp.coeff))
		toAdd.Lsh(toAdd, comp.position)
		if comp.sign == 1 {
			accumulator.Add(accumulator, toAdd)
		} else if comp.sign == -1 {
			accumulator.Sub(accumulator, toAdd)
		} else {
			panic("decompositionCoefficient::sign not +/- 1")
		}
	}
	if accumulator.Cmp(x) == 0 {
		return true
	}
	return false
}

func TestDecomposition(t *testing.T) {
	const iterations = 10000
	var drng *rand.Rand = rand.New(rand.NewSource(141152))
	var bigrange *big.Int = big.NewInt(0)
	bigrange.Set(CurveOrder_Int)
	var smallrange *big.Int = big.NewInt(1024)
	_ = smallrange
	_ = bigrange
	for i := 0; i < iterations; i++ {
		var x *big.Int = big.NewInt(0)
		x.Rand(drng, bigrange)
		decomp := decomposeUnalignedSignedAdic_Int(x, 5)
		// fmt.Println(i)
		// fmt.Println(decomp)
		// fmt.Printf("%b\n", x)
		if !test_decomposition_correctness(x, decomp) {
			t.Fatal("Signed Decomposition algorithm for sliding window does not work")
		}
	}

}
