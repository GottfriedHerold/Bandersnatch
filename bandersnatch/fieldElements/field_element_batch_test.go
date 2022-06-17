package fieldElements

import (
	"errors"
	"math/rand"
	"testing"

	"github.com/GottfriedHerold/Bandersnatch/internal/testutils"
	"github.com/GottfriedHerold/Bandersnatch/internal/utils"
)

func TestMultiInvert(t *testing.T) {
	const MAXSIZE = 20
	var drng *rand.Rand = rand.New(rand.NewSource(87))
	empty := make([]bsFieldElement_64, 0)
	MultiInvertEq()
	MultiInvertEqSlice(empty)
	var numsArray, numsArrayInv [MAXSIZE]bsFieldElement_64
	for i := 0; i < MAXSIZE; i++ {
		numsArray[i].SetRandomUnsafeNonZero(drng)
		numsArrayInv[i].Inv(&numsArray[i])
	}
	for size := 0; size < MAXSIZE; size++ {
		var numsArrayCopy [MAXSIZE]bsFieldElement_64 = numsArray
		err := MultiInvertEqSlice(numsArrayCopy[0:size])
		if err != nil {
			t.Fatalf("Error during Multi-Invert (Slice): %v", err)
		}
		for i := 0; i < size; i++ {
			if !numsArrayCopy[i].IsEqual(&numsArrayInv[i]) {
				t.Fatal("Multi-Inversion does not give the same result as indivdual inversion")
			}
		}
	}
	for size := 0; size < MAXSIZE; size++ {
		var numsArrayCopy [MAXSIZE]bsFieldElement_64 = numsArray
		Ptrs := make([]*bsFieldElement_64, size)
		for i := 0; i < size; i++ {
			Ptrs[i] = &numsArrayCopy[i]
		}
		err := MultiInvertEq(Ptrs...)
		if err != nil {
			t.Fatalf("Error during MultiInvert: %v", err)
		}
		for i := 0; i < size; i++ {
			if !numsArrayCopy[i].IsEqual(&numsArrayInv[i]) {
				t.Fatal("Multi-Inversion does not give the same result as indivdual inversion")
			}
		}
	}
	var zero bsFieldElement_64 = FieldElementZero
	err := MultiInvertEq(&zero)
	err2 := MultiInvertEqSlice([]bsFieldElement_64{zero})
	if err == nil || err2 == nil {
		t.Fatalf("Inverting Zero did not produce error")
	}
	if !errors.Is(err, ErrDivisionByZero) {
		t.Fatalf("Inverting zero did not produce division by Zero error")
	}
	if !errors.Is(err2, ErrDivisionByZero) {
		t.Fatalf("Inverting zero did not produce division by Zero error")
	}
	data := err.GetData()
	data2 := err2.GetData()
	if data.NumberOfZeroIndices != 1 {
		t.Fatalf("Wrong data reported when inverting single 0")
	}
	if data2.NumberOfZeroIndices != 1 {
		t.Fatalf("Wrong data reported when inverting single 0")
	}

	if !utils.CompareSlices(data.ZeroIndices, []bool{true}) {
		t.Fatalf("Wrong data reported when inverting single 0")
	}
	if !utils.CompareSlices(data2.ZeroIndices, []bool{true}) {
		t.Fatalf("Wrong data reported when inverting single 0")
	}
	if !zero.IsZero() {
		t.Fatalf("Inverting single zero did not leave values untouched.")
	}

	var numsArrayCopy [MAXSIZE]bsFieldElement_64
	for i := 0; i < MAXSIZE; i++ {
		numsArray[i].SetRandomUnsafeNonZero(drng)
		switch i % 2 {
		case 0:
			numsArrayCopy[i].SetZero() // not needed, but for clarity
		case 1:
			numsArrayCopy[i] = numsArray[i]
		}
	}
	err = MultiInvertEqSlice(numsArrayCopy[:])
	if err == nil {
		t.Fatalf("Inverting slice with 0s did not report error")
	}
	data = err.GetData()
	if data.NumberOfZeroIndices != (MAXSIZE+1)/2 {
		t.Fatalf("Reported error had number of 0 indices wrong")
	}
	for i := 0; i < MAXSIZE; i++ {
		// Check that MultiInvertEqSlice did not modify the slice.
		switch i % 2 {
		case 0:
			if !numsArrayCopy[i].IsZero() {
				t.Fatalf("MultiInvertEqSlice modified args on error")
			}
			if !data.ZeroIndices[i] {
				t.Fatalf("MultiInvertEqSlice's error did not correctly report 0 indices")
			}
		case 1:
			if numsArrayCopy[i] != numsArray[i] {
				t.Fatalf("MultiInvertEqSlice modified args on error")
			}
			if data.ZeroIndices[i] {
				t.Fatalf("MultiInvertEqSlice's error did not correctly report 0 indices")
			}
		default:
			panic("Cannot happen")
		}
	}
	var ArrPtrs [MAXSIZE]*bsFieldElement_64
	for i := 0; i < MAXSIZE; i++ {
		ArrPtrs[i] = &numsArrayCopy[i]
	}
	err2 = MultiInvertEq(ArrPtrs[:]...)
	if err2 == nil {
		t.Fatalf("MultInvertEq did not report error on 0 args")
	}
	data2 = err2.GetData()
	if data.NumberOfZeroIndices != data2.NumberOfZeroIndices {
		t.Fatalf("MultiInvertEq did not report same error as MultiInvertEqSlice")
	}
	if !utils.CompareSlices(data.ZeroIndices, data2.ZeroIndices) {
		t.Fatalf("MultiInvertEq did not report same error as MultiInvertEqSlice")
	}

}

func TestSummationSlice(t *testing.T) {
	const size = 20
	var drng *rand.Rand = rand.New(rand.NewSource(100))
	empty := make([]bsFieldElement_64, 0)
	var result bsFieldElement_64
	var a, b, c bsFieldElement_64
	result.SetRandomUnsafe(drng) // arbitrary value, really.
	a.SetRandomUnsafe(drng)
	b.SetRandomUnsafe(drng)
	c.SetRandomUnsafe(drng)
	result.SummationSlice(empty)
	if !result.IsZero() {
		t.Fatal("SummationSlice with zero-length slice does not result in 0")
	}
	result.SetRandomUnsafe(drng)
	result.SummationMany()
	if !result.IsZero() {
		t.Fatal("SummationMany with 0 arguments does not result in 0")
	}
	result.SummationMany(&a)
	if !result.IsEqual(&a) {
		t.Fatal("SummationMany with 1 argument does not copy")
	}
	result.SummationMany(&a, &b, &c)
	result.SubEq(&a)
	result.SubEq(&b)
	result.SubEq(&c)
	if !result.IsZero() {
		t.Fatal("SummationMany with 3 arguments does not match expected result")
	}
	var summands [size]bsFieldElement_64
	var acc bsFieldElement_64
	var Ptrs [size]*bsFieldElement_64
	for i := 0; i < size; i++ {
		summands[i].SetRandomUnsafe(drng)
		Ptrs[i] = &summands[i]
	}
	acc.SetZero()
	for i := 0; i < size; i++ {
		acc.AddEq(&summands[i])
	}
	result.SummationSlice(summands[:])
	if !result.IsEqual(&acc) {
		t.Fatal("SummationSlice does not match result of manual addition")
	}
	result.SummationMany(Ptrs[:]...)
	if !result.IsEqual(&acc) {
		t.Fatal("SummationMany does not match result of manual addition")
	}
	summandsCopy := summands
	testutils.Assert((size >= 2))
	summandsCopy[1].SummationSlice(summandsCopy[:])
	if !summandsCopy[1].IsEqual(&result) {
		t.Fatal("SummationSlice does not work when result aliases an input")
	}
	summandsCopy = summands
	for i := 0; i < size; i++ {
		Ptrs[i] = &summandsCopy[i]
	}
	summandsCopy[1].SummationMany(Ptrs[:]...)
	if !summandsCopy[1].IsEqual(&result) {
		t.Fatal("SummationMany does not work when result aliases an input")
	}
	a.SetRandomUnsafe(drng)
	b.SetUInt64(size)
	result.Mul(&b, &a)
	for i := 0; i < size; i++ {
		Ptrs[i] = &a
	}
	a.SummationMany(Ptrs[:]...)
	if !a.IsEqual(&result) {
		t.Fatal("SummationMany does not work when results and all inputs alias")
	}
}

func TestMultiplySlice(t *testing.T) {
	const size = 20
	var drng *rand.Rand = rand.New(rand.NewSource(100))
	empty := make([]bsFieldElement_64, 0)
	var result bsFieldElement_64
	var a, b, c bsFieldElement_64
	result.SetRandomUnsafe(drng) // arbitrary value, really.
	a.SetRandomUnsafe(drng)
	b.SetRandomUnsafe(drng)
	c.SetRandomUnsafe(drng)
	result.MultiplySlice(empty)
	if !result.IsOne() {
		t.Fatal("MultiplySlice with zero-length slice does not result in 1")
	}
	result.SetRandomUnsafe(drng)
	result.MultiplyMany()
	if !result.IsOne() {
		t.Fatal("MultiplyMany with 0 arguments does not result in 0")
	}
	result.MultiplyMany(&a)
	if !result.IsEqual(&a) {
		t.Fatal("MultiplyMany with 1 argument does not copy")
	}
	result.MultiplyMany(&a, &b, &c)
	var result2 bsFieldElement_64
	result2.Mul(&a, &b)
	result2.MulEq(&c)
	if !result.IsEqual(&result2) {
		t.Fatal("MultiplyMany with 3 arguments does not match expected result")
	}
	var factors [size]bsFieldElement_64
	var acc bsFieldElement_64
	var Ptrs [size]*bsFieldElement_64
	for i := 0; i < size; i++ {
		factors[i].SetRandomUnsafe(drng)
		Ptrs[i] = &factors[i]
	}
	acc.SetOne()
	for i := 0; i < size; i++ {
		acc.MulEq(&factors[i])
	}
	result.MultiplySlice(factors[:])
	if !result.IsEqual(&acc) {
		t.Fatal("MultiplySlice does not match result of manual multiplication")
	}
	result.MultiplyMany(Ptrs[:]...)
	if !result.IsEqual(&acc) {
		t.Fatal("MultiplyMany does not match result of manual multiplication")
	}
	factorsCopy := factors
	testutils.Assert((size >= 2))
	factorsCopy[1].MultiplySlice(factorsCopy[:])
	if !factorsCopy[1].IsEqual(&result) {
		t.Fatal("MultiplySlice does not work when result aliases an input")
	}
	factorsCopy = factors
	for i := 0; i < size; i++ {
		Ptrs[i] = &factorsCopy[i]
	}
	factorsCopy[1].MultiplyMany(Ptrs[:]...)
	if !factorsCopy[1].IsEqual(&result) {
		t.Fatal("MultiplyMany does not work when result aliases an input")
	}
	a.SetRandomUnsafe(drng)
	result.SetOne()
	for i := 0; i < size; i++ {
		result.MulEq(&a)
	}
	for i := 0; i < size; i++ {
		Ptrs[i] = &a
	}
	a.MultiplyMany(Ptrs[:]...)
	if !a.IsEqual(&result) {
		t.Fatal("MultiplyMany does not work when results and all inputs alias")
	}
}
