package bandersnatch

/*
	Note: Suffixes like _ttt or _tta refer to the type of input point (with order output, input1 [,input2] )
	t denote extended projective,
	a denotes extended affine (i.e. Z==1)
	s denotes double-projective
*/

// exceptional cases: None
// NaP -> NaP[e=g=h=0]
func (out *point_efgh_base) double_st(input *point_xtw_base) {
	var xx, yy, zz2 FieldElement
	xx.Square(&input.x)
	yy.Square(&input.y)
	out.e.Add(&input.x, &input.y)
	out.e.SquareEq()
	out.e.SubEq(&xx)
	out.e.SubEq(&yy)      // E = 2XY
	xx.Multiply_by_five() // 5X^2
	out.g.Sub(&yy, &xx)   // G = Y^2 - 5X^2 = ax^2 + Y^2 = Z^2 + dT^2
	out.h.Add(&yy, &xx)   // H = Y2 + 5x^2 = -ax^2 + Y^2
	zz2.Square(&input.z)
	zz2.DoubleEq()
	out.f.Sub(&zz2, &out.g) // F = 2Z^2-(Z^2+dT^2) = Z^2 - dT^2
}

// exceptional cases: NaP
// NaP -> g=h=0, e=t, f=2
func (out *point_efgh_base) double_sa(input *point_axtw_base) {
	var xx5, yy FieldElement
	xx5.Square(&input.x)
	xx5.Multiply_by_five()
	yy.Square(&input.y)
	out.e.Double(&input.t)              // E = 2T. Strangely, HWCD computes 2*X*Y here. This is more efficient.
	out.g.Sub(&yy, &xx5)                // G = Y^2 - 5X^2 = Z^2 + dT^2 = 1+dT^2
	out.h.Add(&yy, &xx5)                // H = -aX^2 + Y^2
	out.f.Sub(&fieldElementTwo, &out.g) // F = 2 - G = 1 - dT^2 = Z^2 - dT^2
}

// exceptional cases: None
// NaP -> e=g=h=0. We have f=0 for T1 or T2-NaP.
func (out *point_efgh_base) double_ss(input *point_efgh_base) {
	var xx, yy, zz2 FieldElement
	xx.Mul(&input.e, &input.f)  // X
	yy.Mul(&input.g, &input.h)  // Y
	out.e.Add(&xx, &yy)         // X + Y
	out.e.SquareEq()            // (X + Y)^2
	xx.SquareEq()               // X^2
	yy.SquareEq()               // Y^2
	out.e.SubEq(&xx)            // 2XY + Y^2
	out.e.SubEq(&yy)            // E = 2XY
	xx.Multiply_by_five()       // 5X^2
	zz2.Mul(&input.f, &input.g) // Z
	zz2.SquareEq()              // Z^2
	zz2.DoubleEq()              // 2Z^2
	out.g.Sub(&yy, &xx)         // G = Y^2 - 5X^2 = aX^2 + Y^2 = Z^2 + dT^2
	out.h.Add(&yy, &xx)         // H = Y^2 + 5X^2 = -aX^2 + Y^2
	out.f.Sub(&zz2, &out.g)     // F = 2Z^2 - (Z^2 + dT^2) = Z^2 - dT^2
}

/*
func (out *point_xtw_base) double_tt(input1 *point_xtw_base) {
	// TODO: Use https://www.hyperelliptic.org/EFD/g1p/auto-twisted-extended.html#doubling-dbl-2008-hwcd.
	// Note we need to ensure that this formula gives the same result as add_xxx (modulo ax^2 + y^2 = z^2 + dt^2 and a global sign), even for z==0
	out.add_ttt(input1, input1)
}

func (out *point_xtw_base) double_ta(input *point_axtw_base) {
	out.add_taa(input, input)
}

func (out *point_xtw_base) double_ts(input *point_efgh_base) {
	temp := input.ToDecaf_xtw()
	out.add_ttt(&temp, &temp)
}
*/
/*
func default_Double(receiver CurvePointPtrInterfaceWrite, input CurvePointPtrInterfaceRead) {
	receiver.Add(input, input)
}

func default_DoubleEq(receiver CurvePointPtrInterface) {
	receiver.Double(receiver)
}
*/
