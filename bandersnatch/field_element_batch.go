package bandersnatch

import (
	"errors"

	"github.com/GottfriedHerold/Bandersnatch/bandersnatch/errorsWithData"
)

/*
	This file contains field element operations that can operate on multiple field elements.
*/

// MultiplySlice sets the receiver to the product of all the given elements.
//
// An empty product results in a result of 1. Note: Use MultiplyMany for a variadic version.
func (z *bsFieldElement_64) MultiplySlice(factors []bsFieldElement_64) {
	L := len(factors)
	if L == 0 {
		z.SetOne()
		return
	}

	// We need to store the eventual result in a temporary rather than z directly, due to potential aliasing of z with a factor.
	// Since L >= 1, we can initialize with factors[0] and start the following loop from i==1; this means we only have L-1 multiplications.
	var result bsFieldElement_64 = factors[0]
	for i := 1; i < L; i++ {
		result.MulEq(&factors[i])
	}
	*z = result
}

// MultiplyMany sets the receiver to the product of the factors.
//
// An empty product gives a result of 1. Note: Use MultiplySlice if you have the non-pointer factors stored in a slice.
func (z *bsFieldElement_64) MultiplyMany(factors ...*bsFieldElement_64) {
	L := len(factors)
	if L == 0 {
		z.SetOne()
		return
	}
	var result bsFieldElement_64 = *factors[0] // due to potential aliasing of z with a factor, we cannot directly write into z yet.
	for i := 1; i < len(factors); i++ {
		result.MulEq(factors[i])
	}
	*z = result
}

// SummationSlice sets the receiver to the sum the values contained in summands.
//
// An empty sum gives a result of 0. Note: Use SummationMany for a variadic version.
func (z *bsFieldElement_64) SummationSlice(summands []bsFieldElement_64) {
	var result bsFieldElement_64 // due to potential aliasing of z with a factor.
	L := len(summands)
	if L == 0 {
		z.SetZero()
		return
	}
	result = summands[0]
	for i := 1; i < L; i++ {
		result.AddEq(&summands[i])
	}
	*z = result
}

// SummationMany sets the receiver to the sum of the given summands.
//
// An empty sum gives a result of 0. Note: UseSummationSlice if you have the non-pointer summands stored in a slice.
func (z *bsFieldElement_64) SummationMany(summands ...*bsFieldElement_64) {
	var result bsFieldElement_64 // due to potential aliasing of z with a factor.
	L := len(summands)
	if L == 0 {
		z.SetZero()
		return
	}
	result = *summands[0]
	for i := 1; i < L; i++ {
		result.AddEq(summands[i])
	}
	*z = result
}

// NOTE: Some callers actually recover the panic and rely on the fact that no changes to args are made on panic.

// MultiInvertEq replaces every argument by its multiplicative inverse.
// If any arguments are zero, panics with *ErrMultiInversionEncounteredZero without modifying any of the args.
//
// Use MultiInvertEqSlice instead if the arguments are contained in a slice.
//
// NOTE: For now, we do not guarantee any kind of correct or consistent behaviour (even for the non-aliasing elements) if any args alias.
func MultiInvertEq(args ...*bsFieldElement_64) {
	L := len(args)
	// special case L==0, L==1
	// Having L >= 2 guaranteed avoids special cases.
	if L == 0 {
		return
	}
	if L == 1 {
		if args[0].IsZero() {
			err := generateMultiDivisionByZeroPanic(args, "bandersnatch / field elements: Division by zero when calling MultiInvertEq on single element")
			panic(err)
		}
		args[0].InvEq()
		return
	}

	// Mutli-Inversion algorithm: We compute P = args[0] * ... * args[L-1] via a multiplication tree with leaves args[i] and root P.
	// then invert all nodes, starting from P (inverting a node is cheap if the parent was inverted, so we only need to invert P) until the leaves.
	// We use the common version of the algorithm corresponding to the usual left-associative multiplication tree ((((...(args[0]*args[1]) * args[2]) *  ...

	var productOfFirstN []bsFieldElement_64 = make([]bsFieldElement_64, L-1)

	// Set productOfFirstN[i] == args[0] * args[1] * ... * args[i+1]
	productOfFirstN[0].Mul(args[0], args[1])
	for i := 1; i < len(args)-1; i++ {
		productOfFirstN[i].Mul(&productOfFirstN[i-1], args[i+1])
	}

	var temp1, temp2 bsFieldElement_64 // temp2 is just needed to swap
	if productOfFirstN[L-2].IsZero() {
		err := generateMultiDivisionByZeroPanic(args, "bandersnatch / field elements: Division by zero when calling MultiInvertEq")
		panic(err)
	}
	temp1.Inv(&productOfFirstN[L-2])

	for i := L - 1; i >= 2; i-- {
		// invariant: temp1 == 1 / (args[0] * args[1] * ... * args[i]) at the beginning of the lopp
		temp2.Mul(&temp1, args[i])
		args[i].Mul(&temp1, &productOfFirstN[i-2])
		temp1 = temp2
	}
	// temp1 == 1 / args[0] * args[1] at this point
	temp2.Mul(&temp1, args[0])
	args[0].Mul(&temp1, args[1])
	*args[1] = temp2
}

// MultiInvertEq replaces every element in args by its multiplicative inverse.
// If any arguments are zero, panics with *ErrMultiInversionEncounteredZero without modifying any of the args.
func MultiInvertEqSlice(args []bsFieldElement_64) {
	L := len(args)
	// special case L==0, L==1 to allow optimizing the initial cases (this avoids having to set some elements to 1 and then multiplyting by it)
	if L == 0 {
		return
	}
	if L == 1 {
		if args[0].IsZero() {
			err := generateMultiDivisionByZeroPanic([]*bsFieldElement_64{&args[0]}, "bandersnatch / field elements: Division by zero when calling MultiInvertSliceEq on single element")
			panic(err)
		}

		args[0].InvEq()
		return
	}

	// Multi-Inversion algorithm: We compute P = args[0] * ... * args[L-1] via a multiplication tree with leaves args[i] and root P.
	// then invert all nodes, starting from P (inverting a node is cheap if the parent was inverted, so we only need to invert P) until the leaves.
	// We use the common version of the algorithm corresponding to the usual left-associative multiplication tree ((((...(args[0]*args[1]) * args[2]) *  ...

	var productOfFirstN []bsFieldElement_64 = make([]bsFieldElement_64, L-1)

	// Set productOfFirstN[i] == args[0] * args[1] * ... * args[i+1]
	productOfFirstN[0].Mul(&args[0], &args[1])
	for i := 1; i < len(args)-1; i++ {
		productOfFirstN[i].Mul(&productOfFirstN[i-1], &args[i+1])
	}

	// check if product of all args is zero. If yes, we need to handle some errors.
	if productOfFirstN[L-2].IsZero() {
		var argPtrs []*bsFieldElement_64 = make([]*bsFieldElement_64, len(args))
		for i := 0; i < len(args); i++ {
			argPtrs[i] = &args[i]
		}
		err := generateMultiDivisionByZeroPanic(argPtrs, "bandersnatch / field elements: Division by zero when calling MultiInvertEq")
		panic(err)
	}

	var temp1, temp2 bsFieldElement_64 // temp2 is just needed to swap
	temp1.Inv(&productOfFirstN[L-2])

	for i := L - 1; i >= 2; i-- {
		// invariant: temp1 == 1 / (args[0] * args[1] * ... * args[i]) at the beginning of the lopp
		temp2.Mul(&temp1, &args[i])
		args[i].Mul(&temp1, &productOfFirstN[i-2])
		temp1 = temp2
	}
	// temp1 == 1 / args[0] * args[1] at this point
	temp2.Mul(&temp1, &args[0])
	args[0].Mul(&temp1, &args[1])
	args[1] = temp2
}

/*
// TODO: Base on ErrorWithData

// ErrMultiInversionEncounteredZero is a (stateful) error either returned by or provided as argument to panic by functions that perform
// simulutaneous inversion of multiple field elements.
//
// It contains information about which elements were zero.
type ErrMultiInversionEncounteredZero struct {
	ZeroIndices         []bool // indices (starting from 0) of the field elements that were zero, i.e. in a call (ignoring argument types) MultiInvertEq(0, 1, 2, 0, 0), we would have ZeroIndices = [0, 3, 4]
	NumberOfZeroIndices int    // number of field elements that were zero when multi-inversion was requested. In the above example, would be 3
	s                   string // internal: (static) error string that is to be displayed by Error(). Note that Error() also outputs additional information about ZeroIndices etc.
}

// Error is provided to satisfy the error interface (for pointer receivers). We report the stored string s together with information about ZeroIndices.
func (err *ErrMultiInversionEncounteredZero) Error() string {
	var b strings.Builder
	b.WriteString(err.s)
	if err.NumberOfZeroIndices <= 0 {
		fmt.Fprintf(&b, "\nThe number of zero indices stored as metadata is %v <= 0. This should only occur if you are creating uninitialized ErrMultiInversionEncounteredZero errors manually.", err.NumberOfZeroIndices)
		return b.String()
	}
	if err.NumberOfZeroIndices == 1 {
		for i := 0; i < len(err.ZeroIndices); i++ {
			if err.ZeroIndices[i] {
				fmt.Fprintf(&b, "\nThe %v'th argument (counting from 0) was the only one that was zero.", i)
				return b.String()
			}
		}
		fmt.Fprintf(&b, "\nInternal bug: the number of zero indices stored as metadata is 1, but no zero index was contained in the metadata.")
		return b.String()
	}
	var indices []int = make([]int, 0, err.NumberOfZeroIndices)
	for i := 0; i < len(err.ZeroIndices); i++ {
		if err.ZeroIndices[i] {
			indices = append(indices, i)
		}
	}
	if len(indices) != err.NumberOfZeroIndices {
		fmt.Fprintf(&b, "\nInternal bug: the number of zero indices stored as metadata does not match the number of field elements that were reported as zero. Error reporting may be unreliable.")
	}
	fmt.Fprintf(&b, "\nThere were %v numbers (starting indexing with 0) that were 0 in the call. Those are:\n %v", err.NumberOfZeroIndices, indices)
	return b.String()
}

// generateMultiDivisionByZeroPanic is a helper function for MultiInvertEq and MultiInvertSliceEq.
//
// It creates the actual non-nil error that includes diagnostics which field Elements were zero.
func generateMultiDivisionByZeroPanic(fieldElements []*bsFieldElement_64, s string) *ErrMultiInversionEncounteredZero {
	var err ErrMultiInversionEncounteredZero
	err.s = s
	err.NumberOfZeroIndices = 0
	err.ZeroIndices = make([]bool, len(fieldElements))
	for i, fe := range fieldElements {
		if fe.IsZero() {
			err.ZeroIndices[i] = true
			err.NumberOfZeroIndices++
		}
	}
	return &err
}
*/

type MultiInversionErrorData struct {
}

// ErrMultiInversionEncounteredZero is the (base) error reported when a division by zero occurs during a multi-inversion.
// Note that we actually always return an error *wrapping* ErrMultiInversionEncounteredZero;
// this wrapping error always fully overrides the error message by a more detailed one.
var ErrMultiInversionEncounteredZero = errors.New("Division by zero in Multi-Inversion") // actual message string is irrelevant.

func generateMultiDivisionByZeroError(fieldElements []*bsFieldElement_64, baseErrorMessage string) errorsWithData.ErrorWithParameters[MultiInversionErrorData] {
	
}
