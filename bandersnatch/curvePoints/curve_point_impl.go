package curvePoints

/*
	Note: Suffixes like _ttt or _tta refer to the type of input point (with order output, input1 [,input2] )
	t denote extended projective,
	a denotes extended affine (i.e. Z==1)
	s denotes double-projective
*/

func (out *point_xtw_base) neg_tt(input *point_xtw_base) {
	out.x.Neg(&input.x)
	out.y = input.y
	out.t.Neg(&input.t)
	out.z = input.z
}

func (out *point_axtw_base) neg_aa(input *point_axtw_base) {
	out.x.Neg(&input.x)
	out.y = input.y
	out.t.Neg(&input.t)
}

func (out *point_efgh_base) neg_ss(input *point_efgh_base) {
	// Only need to negate e (or equivalently, g)
	out.e.Neg(&input.e)
	out.f = input.f
	out.g = input.g
	out.h = input.h
}

func (out *point_xtw_base) neg_ta(input *point_axtw_base) {
	out.x.Neg(&input.x)
	out.y = input.y
	out.t.Neg(&input.t)
	out.z.SetOne()
}
