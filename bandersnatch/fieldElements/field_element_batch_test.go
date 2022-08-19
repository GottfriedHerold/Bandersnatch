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

	// Test on empty lists / slices
	MultiInvertEq()
	MultiInvertEqSlice(empty)
	MultiInvertEqSliceSkipZeros(empty)
	MultiInvertEqSkipZeros()

	// prepare random non-zero elements and individually invert them (for testing)
	var numsArray, numsArrayInv [MAXSIZE]bsFieldElement_64
	for i := 0; i < MAXSIZE; i++ {
		numsArray[i].SetRandomUnsafeNonZero(drng)
		numsArrayInv[i].Inv(&numsArray[i])
	}

	// Set some elements to 1. This is becauce we might have special code that skips ones.
	testutils.Assert(MAXSIZE > 5)
	numsArray[4].SetOne()
	numsArray[5].SetOne()
	numsArrayInv[4].SetOne()
	numsArrayInv[5].SetOne()

	// Test whether MultiInvertEqSlice matches individual inversion results
	// NOTE: We check this for every sub-slice [0:size] for 0<=size < MAXSIZE
	for size := 0; size < MAXSIZE; size++ {
		var numsArrayCopy [MAXSIZE]bsFieldElement_64 = numsArray // deep copy, because this is array, not slice!
		err := MultiInvertEqSlice(numsArrayCopy[0:size])
		if err != nil {
			t.Fatalf("Error during Multi-Invert (Slice): %v", err)
		}
		for i := 0; i < size; i++ {
			if !numsArrayCopy[i].IsEqual(&numsArrayInv[i]) {
				t.Fatal("Multi-Inversion does not give the same result as individual inversion")
			}
		}
		// redo check with MultiInvertEqSlice
		numsArrayCopy = numsArray
		MultiInvertEqSliceSkipZeros(numsArrayCopy[0:size])
		for i := 0; i < size; i++ {
			if !numsArrayCopy[i].IsEqual(&numsArrayInv[i]) {
				t.Fatal("Multi-Inversion (with Skipped zeros) does not give the same result as individual inversion")
			}
		}
	}

	// Same test with MultiInvertEq and MultiInvertEqSkipZeros.
	for size := 0; size < MAXSIZE; size++ {
		var numsArrayCopy [MAXSIZE]bsFieldElement_64 = numsArray
		// We need to make a slice of Pointer first to work with the variadic functions.
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
				t.Fatal("Multi-Inversion (variadic) does not give the same result as individual inversion")
			}
		}
		// redo check with MultiInvertEqSkipZeros
		numsArrayCopy = numsArray

		// NOTE: No need to reset Ptrs
		if size > 0 {
			testutils.Assert(Ptrs[0] == &numsArrayCopy[0])
		}

		MultiInvertEqSkipZeros(Ptrs...)
		for i := 0; i < size; i++ {
			if !numsArrayCopy[i].IsEqual(&numsArrayInv[i]) {
				t.Fatal("Multi-Inversion (skipped zeros, variadic) does not give the same result as individual inversion")
			}
		}
	}

	// Check behaviour on single zero element:
	var zero bsFieldElement_64 = FieldElementZero
	SizeOneZeroSlice := []bsFieldElement_64{zero}
	err := MultiInvertEq(&zero)
	err2 := MultiInvertEqSlice(SizeOneZeroSlice)
	if !zero.IsZero() {
		t.Fatalf("Inverting zero changed the element")
	}
	if !SizeOneZeroSlice[0].IsZero() {
		t.Fatalf("Inverting zero changed the element")
	}
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

	if !utils.CompareSlices(data.ZeroIndices, []int{0}) {
		t.Fatalf("Wrong data reported when inverting single 0")
	}
	if !utils.CompareSlices(data2.ZeroIndices, []int{0}) {
		t.Fatalf("Wrong data reported when inverting single 0")
	}
	if !zero.IsZero() {
		t.Fatalf("Inverting single zero did not leave values untouched.")
	}

	// Test SkipZero alternatives for single zero element:
	list1 := MultiInvertEqSkipZeros(&zero)
	list2 := MultiInvertEqSliceSkipZeros(SizeOneZeroSlice)
	if !zero.IsZero() {
		t.Fatalf("Inverting zero changed the element (zero-skipped version)")
	}
	if !SizeOneZeroSlice[0].IsZero() {
		t.Fatalf("Inverting zero changed the element (zero-skipped, slice version)")
	}
	if list1 == nil {
		t.Fatalf("MultiInvertEqSkipZeros returned nil on 1 zero element")
	}
	if list2 == nil {
		t.Fatalf("MultiInvertEqSliceSkipZeros returned nil 1 zero element")
	}
	if !utils.CompareSlices(list1, []int{0}) {
		t.Fatalf("MultiInverEqSkipZeros unexpectedly retuned %v", list1)
	}
	if !utils.CompareSlices(list2, []int{0}) {
		t.Fatalf("MultiInverEqSliceSkipZeros unexpectedly retuned %v", list2)
	}

	// Make slice where every second element is 0
	var ExpectedZeroIndices []int = make([]int, 0)
	for i := 0; i < MAXSIZE; i++ {
		numsArray[i].SetRandomUnsafeNonZero(drng)
		switch i % 2 {
		case 0:
			numsArray[i].SetZero()
			numsArrayInv[i].SetZero()
			ExpectedZeroIndices = append(ExpectedZeroIndices, i)
		case 1:
			numsArrayInv[i].Inv(&numsArray[i])
		}
	}
	// make sure some element is 1
	numsArray[5].SetOne()
	numsArrayInv[5].SetOne()

	// make a copy of the above
	var numsArrayCopy [MAXSIZE]bsFieldElement_64 = numsArray

	err = MultiInvertEqSlice(numsArray[:])
	if err == nil {
		t.Fatalf("Inverting slice with 0s did not report error")
	}
	data = err.GetData()
	if data.NumberOfZeroIndices != (MAXSIZE+1)/2 {
		t.Fatalf("Reported error had number of 0 indices wrong")
	}
	if !utils.CompareSlices(data.ZeroIndices, ExpectedZeroIndices) {
		t.Fatalf("MultiInvertEqSlice reported unexpected zero indices: %v", data.ZeroIndices)
	}

	// NOTE: This assuments that on error, we do not make any normalizations.
	if numsArray != numsArrayCopy {
		t.Fatalf("MultiInvertEqSlice modified data on error")
	}

	var ArrPtrs [MAXSIZE]*bsFieldElement_64
	for i := 0; i < MAXSIZE; i++ {
		ArrPtrs[i] = &numsArray[i]
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
	if numsArrayCopy != numsArray {
		t.Fatalf("MultiInvertEq modified elements on error")
	}

	// Test SkipZeros variants:
	list1 = MultiInvertEqSliceSkipZeros(numsArray[:])
	if !utils.CompareSlices(list1, ExpectedZeroIndices) {
		t.Fatalf("MultiInvertEqSliceSkipZeros did not report zero indices correctly")
	}
	for i := 0; i < MAXSIZE; i++ {
		if !numsArray[i].IsEqual(&numsArrayInv[i]) {
			t.Fatalf("MultiplyEqSliceSkipZeros did not modify args as expected")
		}
	}
	numsArray = numsArrayCopy
	list2 = MultiInvertEqSkipZeros(ArrPtrs[:]...)

	if !utils.CompareSlices(list2, ExpectedZeroIndices) {
		t.Fatalf("MultiInvertEqSkipZeros did not report zero indices correctly")
	}
	for i := 0; i < MAXSIZE; i++ {
		if !ArrPtrs[i].IsEqual(&numsArrayInv[i]) {
			t.Fatalf("MultiplyEqSkipZeros did not modify args as expected")
		}
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
