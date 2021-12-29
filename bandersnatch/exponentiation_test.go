package bandersnatch

import (
	"math/big"
	"math/rand"
	"testing"
)

func TestSimpleExponentiation(t *testing.T) {
	const iterations = 10
	var temp1, temp2, temp3, temp4 Point_xtw
	temp1.exp_naive_xx(&example_generator_xtw, GroupOrder_Int)
	if !temp1.IsNeutralElement_FullCurve() {
		t.Fatal("Either naive exponentiation is wrong or example point not in subgroup")
	}
	var drng *rand.Rand = rand.New(rand.NewSource(1024))
	var exp1 = big.NewInt(0)
	var exp2 = big.NewInt(1)
	var exp3 = big.NewInt(-1)

	temp1 = makeRandomPointOnCurve_t(drng)
	temp2.exp_naive_xx(&temp1, exp2) // exponent is 1
	if !temp2.is_equal_exact_tt(&temp1) {
		t.Error("1 * P != P for naive exponentiation")
	}
	temp2.exp_naive_xx(&temp1, exp1) // exponent is 0
	if !temp2.is_equal_exact_tt(&NeutralElement_xtw) {
		t.Error("0 * P != Neutral element for naive exponentiation")
	}
	temp2.exp_naive_xx(&temp1, exp3)
	temp1.neg_tt(&temp1)
	if !temp1.is_equal_exact_tt(&temp2) {
		t.Error("-1 * P != -P for naive exponentiation")
	}

	var p1, p2, p3 Point_xtw
	for i := 0; i < iterations; i++ {
		p1 = makeRandomPointOnCurve_t(drng)
		p2 = makeRandomPointOnCurve_t(drng)
		p3.add_ttt(&p1, &p2)
		exp1.Rand(drng, CurveOrder_Int)
		exp2.Rand(drng, CurveOrder_Int)
		exp3.Add(exp1, exp2)
		temp1.exp_naive_xx(&p1, exp1)
		temp2.exp_naive_xx(&p2, exp1)
		temp3.exp_naive_xx(&p3, exp1)
		temp4.add_ttt(&temp1, &temp2)
		if !temp3.is_equal_exact_tt(&temp4) {
			t.Error("a * (P+Q) != a*P + a*Q for naive exponentiation")
		}
		temp2.exp_naive_xx(&p1, exp2)
		temp3.exp_naive_xx(&p1, exp3)
		temp4.add_ttt(&temp1, &temp2)
		if !temp3.is_equal_exact_tt(&temp4) {
			t.Error("(a+b) * P != a*P + b*P for naive exponentiation")
		}
	}
}

func TestQuotientGroup(t *testing.T) {
	const iterations = 1000
	var drng *rand.Rand = rand.New(rand.NewSource(1024))
	var temp Point_xtw
	var NumN, NumD, NumE1, NumE2 uint16
	var LegCheck bool
	for i := 0; i < iterations; i++ {
		temp = makeRandomPointOnCurve_t(drng)
		LegCheck = temp.legendre_check_point()
		temp.exp_naive_xx(&temp, GroupOrder_Int)
		var isN, isD, isE1, isE2 int = 0, 0, 0, 0
		if temp.IsNaP() {
			t.Fatal("p253 * random point resulted in singularity")
		}
		if temp.IsNeutralElement_FullCurve() {
			isN = 1
			NumN++
			if !LegCheck {
				t.Fatal("Legendre Check failed in subgroup")
			}
		} else if !temp.z.IsZero() {
			// temp must be the affine order two point.
			if !temp.is_equal_exact_tt(&orderTwoPoint_xtw) {
				t.Fatal("p253 * random point is affine and not neutral, but does not compare equal to the known order-2 point.")
			}
			if !LegCheck {
				t.Fatal("Legendre Check failed in affine oder-2 coset of p253-subgroup ")
			}
			// As is_equal_safe makes some assertions, we double-check that the point is what we expect, avoiding function calls
			// that might expect the point to be in the subgroup.
			temp.z.Inv(&temp.z)
			temp.x.MulEq(&temp.z)
			temp.y.MulEq(&temp.z)
			temp.t.MulEq(&temp.z)
			temp.z.SetOne()
			if !orderTwoPoint_xtw.z.IsOne() {
				t.Fatal("affine order 2 point is not in affine form")
			}
			if !(temp.x.IsEqual(&orderTwoPoint_xtw.x) && temp.y.IsEqual(&orderTwoPoint_xtw.y) && temp.t.IsEqual(&orderTwoPoint_xtw.t)) {
				t.Fatal("p253 * random point equals affine order-2 point according to is_equal_safe, but coos don't match", temp.x.String(), temp.y.String(), temp.z.String())
			}
			isD = 1
			NumD++
		} else {
			// temp.z == 0 is guaranteed
			// temp must be a point of order 2 at infinity.
			if temp.t.IsZero() {
				t.Fatal("p253 * random point resulted in point with t==z==0. This should not happen.")
			}
			if !temp.y.IsZero() {
				t.Fatal("p253 * random point resulted in point with z==0, but y!=0. These do not exist on the curve.")
			}
			if !(temp.is_equal_exact_tt(&exceptionalPoint_1_xtw) || temp.is_equal_exact_tt(&exceptionalPoint_2_xtw)) {
				t.Fatal("p253 * random point is non-affine, but does not compare equal to the known exceptional points.")
			}
			if LegCheck {
				t.Fatal("Legendre Check did not give false result in Exceptional+p253 coset of p253")
			}
			temp.t.Inv(&temp.t)
			temp.x.MulEq(&temp.t)
			temp.t.SetOne()
			// temp.y == temp.z == 0 anyway.
			if temp.is_equal_exact_tt(&exceptionalPoint_1_xtw) {
				isE1 = 1
				NumE1++
			}
			if temp.is_equal_exact_tt(&exceptionalPoint_2_xtw) {
				isE2 = 1
				NumE2++
			}
		}
		if isD+isN+isE1+isE2 != 1 {
			t.Fatal("p253 * random point matches more than one option. Proably something is wrong with the actual testing code.")
		}
	}
	if NumN+NumD+NumE1+NumE2 != iterations {
		t.Fatal("Error in actual testing code logic.")
	}
	// We check statistics now: Each of these cases occurs with probability 25%. Too large deviations are considered failures.
	if NumN < iterations/6 {
		t.Fatal("random point in subgroup rarer than expected")
	}
	if NumN > (iterations*5)/12 {
		t.Fatal("random point in subgroup more often than expected")
	}

	if NumD < iterations/6 {
		t.Fatal("random point in D-coset rarer than expected")
	}
	if NumD > (iterations*5)/12 {
		t.Fatal("random point in D-coset more often than expected")
	}

	if NumE1 < iterations/6 {
		t.Fatal("random point in E1-coset rarer than expected")
	}
	if NumE1 > (iterations*5)/12 {
		t.Fatal("random point in E1-coset more often than expected")
	}

	if NumE2 < iterations/6 {
		t.Fatal("random point in E2-coset rarer than expected")
	}
	if NumE2 > (iterations*5)/12 {
		t.Fatal("random point in E2-coset more often than expected")
	}

}
