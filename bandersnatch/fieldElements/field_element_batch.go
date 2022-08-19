package fieldElements

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

// MultiInvertEq replaces every argument by its multiplicative inverse.
// If any arguments are zero, returns an error satisfying MultiInversionError without modifying any of the args.
//
// Use MultiInvertEqSlice instead if the arguments are contained in a slice.
//
// NOTE: For now, we do not guarantee any kind of correct or consistent behaviour (even for the non-aliasing elements) if any args alias.
//
// If non-nil, the returned error satisfies the interface MultiInversionError and wraps ErrDivisionByZero.
// The MultiInversionError extends error by allowing to retrieve which and how many args were 0.
func MultiInvertEq(args ...*bsFieldElement_64) (err MultiInversionError) {
	L := len(args)

	// handle special cases L==0, L==1
	// Having L >= 2 guaranteed avoids special cases.
	if L == 0 {
		return
	}
	if L == 1 {
		if args[0].IsZero() {
			err = generateMultiDivisionByZeroError(args, ErrorPrefix+"Division by zero when calling MultiInvertEq on single element")
			return
		}
		args[0].InvEq()
		return
	}

	// Multi-Inversion algorithm: We compute P = args[0] * ... * args[L-1]
	// via a multiplication tree with leaves args[i] and root P.
	// We then invert all nodes, starting from P (inverting a node is cheap if
	// the parent was inverted, so we only need to invert the root P) until the leaves.
	// While this actually works for any tree structure, we use the common
	// version of the algorithm corresponding to the usual
	// left-associative multiplication tree
	// ((((...(args[0]*args[1]) * args[2]) *  ... ) * args[L-1]

	var productOfFirstN []bsFieldElement_64 = make([]bsFieldElement_64, L-1)

	// Set productOfFirstN[i] == args[0] * args[1] * ... * args[i+1]
	productOfFirstN[0].Mul(args[0], args[1])
	for i := 1; i < len(args)-1; i++ {
		productOfFirstN[i].Mul(&productOfFirstN[i-1], args[i+1])
	}

	// productOfFirstN[L-2] is the products of all inputs.
	// We can check whether any input was zero by just looking at that value.
	if productOfFirstN[L-2].IsZero() {
		err = generateMultiDivisionByZeroError(args, ErrorPrefix+"Division by zero when calling MultiInvertEq")
		return
	}

	// actually invert the args[i] in order i==L-1 down to i==0
	// We need to handle i==0 and i==1 specially
	// (because we don't have explicit productOfFirstN - values for
	// the empty product and the product of just arg[0])

	var temp1, temp2 bsFieldElement_64
	temp1.Inv(&productOfFirstN[L-2])

	for i := L - 1; i >= 2; i-- {
		// invariant: temp1 == 1 / (args[0] * args[1] * ... * args[i]) at the beginning of the loop
		temp2.Mul(&temp1, args[i])                 // value of temp1 for next iteration
		args[i].Mul(&temp1, &productOfFirstN[i-2]) // final value for args[i]
		temp1 = temp2
	}

	// temp1 == 1 / args[0] * args[1] at this point
	temp2.Mul(&temp1, args[0])
	args[0].Mul(&temp1, args[1])
	*args[1] = temp2
	return nil // no error
}

// MultiInvertEq replaces every element in args by its multiplicative inverse.
// If any arguments are zero, returns a non-nil error without modifying any of the args.
//
// If non-nil, the returned error satisfies the interface MultiInversionError and wraps ErrDivisionByZero.
// The MultiInversionError extends error by allowing to retrieve which and how many args were 0.
func MultiInvertEqSlice(args []bsFieldElement_64) (err MultiInversionError) {
	L := len(args)
	// special case L==0, L==1 to allow optimizing the initial cases
	//  (this avoids having to set some elements to 1 and then multiplyting by it)
	if L == 0 {
		return
	}
	if L == 1 {
		if args[0].IsZero() {
			err = generateMultiDivisionByZeroError([]*bsFieldElement_64{&args[0]}, "bandersnatch / field elements: Division by zero when calling MultiInvertSliceEq on single element")
			return
		}

		args[0].InvEq()
		return
	}

	// Same algorithm as MultiInvertEq

	var productOfFirstN []bsFieldElement_64 = make([]bsFieldElement_64, L-1)

	// Set productOfFirstN[i] == args[0] * args[1] * ... * args[i+1]
	productOfFirstN[0].Mul(&args[0], &args[1])
	for i := 1; i < L-1; i++ {
		productOfFirstN[i].Mul(&productOfFirstN[i-1], &args[i+1])
	}

	// check if product of all args is zero. If yes, we need to handle some errors.
	if productOfFirstN[L-2].IsZero() {
		var argPtrs []*bsFieldElement_64 = make([]*bsFieldElement_64, len(args))
		for i := 0; i < L; i++ {
			argPtrs[i] = &args[i]
		}
		err = generateMultiDivisionByZeroError(argPtrs, "bandersnatch / field elements: Division by zero when calling MultiInvertEq")
		return
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
	return nil // no error
}

// MultiInvertEqSliceSkipZeros bulk-replaces every field element in args by its multiplicative inverse.
// Any zero field element is left unchanged.
//
// If no field element among args was zero, zeroIndices is nil; otherwise, zeroIndices is a list of (0-based) indices of all field elements that were zero.
func MultiInvertEqSliceSkipZeros(args []bsFieldElement_64) (zeroIndices []int) {
	L := len(args)

	// We build a list of pointers to all elements that need inverting (i.e. all that are non-zero)
	// We may also optionally skip all +/- 1 entries (for efficiency)
	Ptrs := make([]*FieldElement, L)

	// Build Ptrs InsertionPos is the index for Ptrs, i is the index for args.
	InsertionPos := 0
	// We always increment i in the loop, but not always InsertionPos
	for i := 0; i < L; i++ {
		Ptr := &args[i]
		if Ptr.IsZero() {

			if zeroIndices == nil {
				zeroIndices = make([]int, 1, L)
				zeroIndices[0] = i
				continue
			}
			zeroIndices = append(zeroIndices, i)
			continue
		}
		Ptrs[InsertionPos] = Ptr
		InsertionPos++
	}
	err := MultiInvertEq(Ptrs[0:InsertionPos]...)
	if err != nil {
		panic(ErrorPrefix + " Internal error: Division by zero, even though zero field elements were supposed to be skipped. This is supposed to be impossible to happen.")
	}
	return
}

// MultiInvertEqSkipZeros replaces every non-zero argument by its multiplicative inverse.
// zero arguments are unmodified.
//
// The returned zeroIndices is nil if none of the args were zero. Otherwise, it is a slice of 0-based indices indicating which args were zero.
//
// NOTE: For now, we do not guarantee any kind of correct or consistent behaviour (even for the non-aliasing elements) if any args alias.
func MultiInvertEqSkipZeros(args ...*bsFieldElement_64) (zeroIndices []int) {

	// Almost identical to the above. Note that we could avoid copying pointers by just swapping pointer-to-zero args to the end and undoing that after inversion.
	// However, that's error-prone (due to the need to get zeroIndices right for indices that were swapped from the end)

	L := len(args)

	// We build a list of pointers to all elements that need inverting (i.e. all that are non-zero)
	// We may also optionally skip all +/- 1 entries (for efficiency)
	Ptrs := make([]*FieldElement, L)

	// Build Ptrs InsertionPos is the index for Ptrs, i is the index for args.
	InsertionPos := 0
	// We always increment i in the loop, but not always InsertionPos
	for i := 0; i < L; i++ {
		Ptr := args[i]
		if Ptr.IsZero() {

			if zeroIndices == nil {
				zeroIndices = make([]int, 1, L)
				zeroIndices[0] = i
				continue
			}
			zeroIndices = append(zeroIndices, i)
			continue
		}
		Ptrs[InsertionPos] = Ptr
		InsertionPos++
	}
	err := MultiInvertEq(Ptrs[0:InsertionPos]...)
	if err != nil {
		panic(ErrorPrefix + " Internal error: Division by zero, even though zero field elements were supposed to be skipped. This is supposed to be impossible to happen.")
	}
	return
}
