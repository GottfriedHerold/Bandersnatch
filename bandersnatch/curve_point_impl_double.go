package bandersnatch

/*
	Note: Suffixes like _ttt or _tta refer to the type of input point (with order output, input1 [,input2] )
	t denote extended projective,
	a denotes extended affine (i.e. Z==1)
	s denotes double-projective
*/

func (out *Point_xtw) double_tt(input1 *Point_xtw) {
	// TODO: Use https://www.hyperelliptic.org/EFD/g1p/auto-twisted-extended.html#doubling-dbl-2008-hwcd.
	// Note we need to ensure that this formula gives the same result as add_xxx (modulo ax^2 + y^2 = z^2 + dt^2 and a global sign), even for z==0
	out.add_ttt(input1, input1)
}

func (out *Point_xtw) double_ta(input *Point_axtw) {
	out.add_taa(input, input)
}

func default_Double(receiver CurvePointPtrInterfaceWrite, input CurvePointPtrInterfaceRead) {
	receiver.Add(input, input)
}

func default_DoubleEq(receiver CurvePointPtrInterface) {
	receiver.Double(receiver)
}
