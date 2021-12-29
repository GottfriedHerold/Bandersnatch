package bandersnatch

/*
	Note: Suffixes like _ttt or _tta refer to the type of input point (with order output, input1 [,input2] )
	t denote extended projective,
	a denotes extended affine (i.e. Z==1)
	s denotes double-projective
*/

func (out *Point_xtw) neg_tt(input *Point_xtw) {
	out.x.Neg(&input.x)
	out.y = input.y
	out.t.Neg(&input.t)
	out.z = input.z
}

func (out *Point_axtw) neg_aa(input *Point_axtw) {
	out.x.Neg(&input.x)
	out.y = input.y
	out.t.Neg(&input.t)
}

func (out *Point_efgh) neg_ss(input *Point_efgh) {
	// Only need to negate e (or equivalently, g)
	out.e.Neg(&input.e)
	out.f = input.f
	out.g = input.g
	out.h = input.h
}

func (out *Point_xtw) neg_ta(input *Point_axtw) {
	out.x.Neg(&input.x)
	out.y = input.y
	out.t.Neg(&input.t)
	out.z.SetOne()
}
