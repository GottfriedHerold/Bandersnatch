package fieldElements

import (
	//"fmt"
	mrand "math/rand"
	"testing"
)

var(
testval                           [1024][4]uint64
)

func init(){
	//initialize testval to some interesting values
	i := 0
	//zero
	testval[i] = [4]uint64{0,0,0,0}
	i++
	//single bit set
	for j:=0; j<64; j++ {
		testval[i] = [4]uint64{1 << j, 0, 0, 0}
		i++
	}

	for j:=0; j<64; j++ {
		testval[i] = [4]uint64{0, 1 << j, 0, 0}
		i++
	}

	for j:=0; j<64; j++ {
		testval[i] = [4]uint64{0, 0, 1 << j, 0}
		i++
	}

	for j:=0; j<64; j++ {
		testval[i] = [4]uint64{0, 0, 0, 1 << j}
		i++
	}

	//single unset bit
	for j:=0; j<256; j++ {  
		tmp := testval[i+j] //For those coming from other languages, ^ is the XOR operator for ints.
		tmp[0] = ^tmp[0]    //But on the unsigned 64, this is the bitwise negation operator. 
		tmp[1] = ^tmp[1]
		tmp[2] = ^tmp[2]
		tmp[3] = ^tmp[3]
		testval[i] = tmp
		i++
	}

	//randoms
	for i < cap(testval){
		testval[i][0] = mrand.Uint64()
		testval[i][1] = mrand.Uint64()
		testval[i][2] = mrand.Uint64()
		testval[i][3] = mrand.Uint64()
		i++
	}

}

func TestModulus(t *testing.T){
	//check if the current constants used for the modulus match the computated values

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
	count:=0

	for _, val := range(testval){
		var x uint256 = val
		var y uint256 = x

		x.AddEq_ReduceWeak(&zero)

		if x.ToUint64() != y.ToUint64() {
			t.Fatalf("TestAdditiveIdentity failed! %v != %v", x.ToBigInt(), y.ToBigInt())
		}
		count++
	}
}

//a*1 == 1*a == a
func TestMultiplicativeIdentity(t *testing.T) {
	var one = uint256{1,0,0,0}
	count:=0

	for _, val := range(testval){
		var x uint256 = val
		y := x

		x.MulEq(&one)

		if x.ToUint64() != y.ToUint64() {
			t.Fatalf("TestMultiplicativeIdentity failed! %v != %v", x.ToUint64(), y.ToUint64())
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
	for _,val := range(testval){
		a=val
		b = a.Neg()
		u=b
		v=a

		a.AddEq_ReduceWeak(&b)
		u.AddEq_ReduceWeak(&v)

		if a.ToUint64() != u.ToUint64(){
			t.Fatalf("Aditive inverse is not cumutative! %v != %v", a, u)
		}
	}

	// a == -(-(a))
	for _,val := range(testval){
		a = val
		b = a.Neg()
		b = b.Neg()

		if a.ToUint64() != b.ToUint64(){
			t.Fatalf("Double inverse does not cancel! %v != %v", a, b)
		}
	}

	// a-b == (-b)+a == -(b-a)
	for _,_a := range(testval){
		for _,_b := range(testval){
			a = _a
			b = _b

			u = a
			u.SubEq_ReduceWeak(&b) //u=a-b

			v = b.Neg()
			v.AddEq_ReduceWeak(&a) //v=(-b)+a

			w=b
			w.SubEq_ReduceWeak(&a) 
			w=w.Neg() //w=-(b-a)

			if (u.ToUint64() != v.ToUint64()) || (u.ToUint64()!=w.ToUint64()) || (v.ToUint64()!=w.ToUint64()){
				t.Errorf("a-b = %v", u)
				t.Errorf("-b+a = %v", v)
				t.Errorf("-(b-a) = %v", w)
				t.Errorf("a=%v b=%v", a, b)
				t.Fatalf("Additive inverse is invalid!")
			}

		}
	}

}

//a*(1/a) == (1/a)*a == 1
func TestMultiplicativeInverse(t *testing.T) {
	var a, b, one uint256
	one = [4]uint64{1,0,0,0}
	var noninvertible int
	for _, val := range(testval){
		a = val
		invertible := b.Inv(&a)
		if invertible==false{
			noninvertible++
			continue
		}

		a.MulEq(&b)

		if a.ToUint64() != one{
			t.Fatalf("Multiplicative inverse failed with a=%v b=%v", a, b)
		}
	}
	t.Logf("%v non inv in test", noninvertible)

}

//a+b == b+a
func TestCummutativeAddition(t *testing.T) {
	var a, b, u, v uint256

	for _, v1 := range(testval){
		a = v1
		for _, v2 := range(testval){
			b = v2

			u=a
			u.AddEq_ReduceWeak(&b)

			v=b
			v.AddEq_ReduceWeak(&a)

			if v.ToUint64() != u.ToUint64(){
				t.Fatalf("Addition does not commute a=%v b=%v", a, b)
			}
		}
	}
}

//a*b == b*a
func TestCummutativeMultiplication(t *testing.T) {
	var a, b, u, v uint256

	for _, v1 := range(testval){
		a = v1
		for _, v2 := range(testval){
			b = v2

			u=a
			u.MulEq(&b)

			v=b
			v.MulEq(&a)

			if v.ToUint64() != u.ToUint64(){
				t.Fatalf("Multiplication does not commute a=%v b=%v", a, b)
			}
		}
	}
}

// (a+b)+c == a+(b+c)
func TestAssociativeAddition(t *testing.T) {
	var a, b, c, u, v uint256

	for j, _a := range(testval){
		a = _a
		for k, _b := range(testval[:j]){
			b = _b
			for _, _c := range(testval[:k]){
				c = _c

				u=a; u.AddEq_ReduceWeak(&b); u.AddEq_ReduceWeak(&c)
				v=c; v.AddEq_ReduceWeak(&b); v.AddEq_ReduceWeak(&a)

				if u.ToUint64() != v.ToUint64(){
					t.Fatalf("Addition fails associative property %v != %v", u, v)
				}
			}
		}
	}
}

// (a*b)*c == a*(b*c)
func TestAssociativeMultiplication(t *testing.T) {
	var a, b, c, u, v uint256

	for j, _a := range(testval){
		a = _a
		for k, _b := range(testval[:j]){
			b = _b
			for _, _c := range(testval[:k]){
				c = _c

				u=a; u.MulEq(&b); u.MulEq(&c)
				v=c; v.MulEq(&b); v.MulEq(&a)

				if u.ToUint64() != v.ToUint64(){
					t.Fatalf("Addition fails associative property %v != %v", u, v)
				}
			}
		}
	}
}

// a(b+c) == ab+ac
func TestDistributiveLeft(t *testing.T) {
	var a, b, c, u, v, w uint256

	for j, _a := range(testval){
		a = _a
		for k, _b := range(testval[:j]){
			b = _b
			for _, _c := range(testval[:k]){
				c = _c

				//u = ab+ac
				u=a; u.MulEq(&b);
				v=a; v.MulEq(&c);
				u.AddEq_ReduceWeak(&v)

				//v = a(b+c)
				v=a; w=b
				w.AddEq_ReduceWeak(&c)
				v.MulEq(&w)

				if u.ToUint64() != v.ToUint64(){
					t.Fatalf("Failed left distributive property (a(b+c) == ab+ac) %v != %v", u, v)
				}
			}
		}
	}

}

// (a+b)c == ac+bc
func TestDistributiveRight(t *testing.T) {
	var a, b, c, u, v uint256

	for j, _a := range(testval){
		a = _a
		for k, _b := range(testval[:j]){
			b = _b
			for _, _c := range(testval[:k]){
				c = _c

				//u = ac+bc
				u=a; u.MulEq(&c);
				v=b; v.MulEq(&c);
				u.AddEq_ReduceWeak(&v)

				//v = (a+b)c
				v=a; 
				v.AddEq_ReduceWeak(&b)
				v.MulEq(&c)

				if u.ToUint64() != v.ToUint64(){
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
	for _, _a := range(testval){
		a = _a
		b = _a

		//2a
		a.DoubleEq()
		//a+a
		b.AddEq_ReduceWeak(&b)

		if a.ToUint64() != b.ToUint64(){
			t.Fatalf("Failed doubling test (2a = a+a) %v != %v", a, b)
		}

	}

	for _, _a := range(testval){
		a = _a
		for _, _b := range(testval){
			b = _b
	
			//2(a+b)
			u = a; u.AddEq_ReduceWeak(&b); u.DoubleEq()

			//2a+2b
			v = a; v.DoubleEq()
			b.DoubleEq()
			v.AddEq_ReduceWeak(&b)

			if v.ToUint64() != u.ToUint64(){
				t.Fatalf("Failed distributive in doubling test (2(a+b) == 2a + 2b) %v != %v", v, u)
			}
		}
	}
}



/*

BENCHMARK

*/
func Benchmark_uint256(b *testing.B){
	b.Run("Neg", benchmarkNegEq)
	b.Run("Double", benchmarkDoubleEq)
	b.Run("Sub", benchmarkSubEq_ReduceWeak)
	b.Run("Add", benchmarkAddEq_ReduceWeak)

	b.Run("Square", benchmarkSquareEq)
	b.Run("Multiply", benchmarkMulEq)
	b.Run("Invert", benchmarkInv)

}


func benchmarkAddEq_ReduceWeak(b *testing.B){
	x := uint256{257, 479, 487, 491}
    y := uint256{997, 499, 503, 509}

	for i :=0; i<b.N; i+=2{
		x.AddEq_ReduceWeak(&y)
		y.AddEq_ReduceWeak(&x)
	}

}

func benchmarkSubEq_ReduceWeak(b *testing.B){
	x := uint256{257, 479, 487, 491}
    y := uint256{997, 499, 503, 509}

	for i :=0; i<b.N; i+=2{
		x.SubEq_ReduceWeak(&y)
		y.SubEq_ReduceWeak(&x)
	}
}

func benchmarkInv(b *testing.B){
	var a uint256
	count := 0
	//runs over the test values
	OL:
	for{
		for _, _a := range testval{
			a = _a
			a.Inv(&a)
			a.Inv(&a)

			count+=2

			if count >= b.N{
				break OL
			}
		}
	}

}

func benchmarkMulEq(b *testing.B){
	x := uint256{257, 479, 487, 491}
    y := uint256{997, 499, 503, 509}

	for i :=0; i<b.N; i+=2{
		x.MulEq(&y)
		y.MulEq(&x)
	}
}

func benchmarkSquareEq(b *testing.B){
	x := uint256{257, 479, 487, 491}
    y := uint256{997, 499, 503, 509}

	for i :=0; i<b.N; i+=2{
		x.SquareEq()
		y.SquareEq()
	}
}

func benchmarkNegEq(b *testing.B){
	x := uint256{257, 479, 487, 491}
    y := uint256{997, 499, 503, 509}

	for i :=0; i<b.N; i+=2{
		x.Neg()
		y.Neg()
	}
}

func benchmarkDoubleEq(b *testing.B){
	x := uint256{257, 479, 487, 491}
    y := uint256{997, 499, 503, 509}

	for i :=0; i<b.N; i+=2{
		x.DoubleEq()
		y.DoubleEq()
	}
}
