package bandersnatch

/*
	Note: Suffixes like _ttt or _tta refer to the type of input point (with order output, input1 [,input2] )
	t denote extended projective,
	a denotes extended affine (i.e. Z==1)
	s denotes double-projective
*/

// https://www.hyperelliptic.org/EFD/g1p/auto-twisted-extended.html#addition-add-2008-hwcd, due to Hisil–Wong–Carter–Dawson 2008, http://eprint.iacr.org/2008/522, Section 3.1.
func (out *Point_xtw) add_ttt(input1, input2 *Point_xtw) {
	var A, B, C, D, E, F, G, H FieldElement // We follow the notation of the link above

	A.Mul(&input1.x, &input2.x) // A = X1 * X2
	B.Mul(&input1.y, &input2.y) // B = Y1 * Y2
	C.Mul(&input1.t, &input2.t)
	C.MulEq(&CurveParameterD_fe) // C = d * T1 * T2
	D.Mul(&input1.z, &input2.z)  // D = Z1 * Z2
	E.Add(&input1.x, &input1.y)
	F.Add(&input2.x, &input2.y) // F serves as temporary
	E.MulEq(&F)
	E.SubEq(&A)
	E.SubEq(&B)   // E = (X1 + Y1) * (X2 + Y2) - A - B == X1*Y2 + Y1*X2
	F.Sub(&D, &C) // F = D - C
	G.Add(&D, &C) // G = D + C

	A.multiply_by_five()
	H.Add(&B, &A) // H = B + 5X1 * X2 = Y1*Y2 - a*X1*X2  (a=-5 is a parameter of the curve)

	out.x.Mul(&E, &F) // X3 = E * F
	out.y.Mul(&G, &H) // Y3 = G * H
	out.t.Mul(&E, &H) // T3 = E * H
	out.z.Mul(&F, &G) // Z3 = F * G
}

func (out *Point_xtw) sub_ttt(input1, input2 *Point_xtw) {
	var temp Point_xtw // needed because of potential aliasing
	temp.neg_tt(input2)
	out.add_ttt(&temp, input1)
}

func (out *Point_efgh) add_stt(input1, input2 *Point_xtw) {
	var A, B, C, D FieldElement

	// same as add_ttt. Note that we can directly write to out, since input1, input2 cannot alias out (due to type).
	A.Mul(&input1.x, &input2.x) // A = X1 * X2
	B.Mul(&input1.y, &input2.y) // B = Y1 * Y2
	C.Mul(&input1.t, &input2.t)
	C.MulEq(&CurveParameterD_fe) // C = d * T1 * T2
	D.Mul(&input1.z, &input2.z)  // D = Z1 * Z2
	out.e.Add(&input1.x, &input1.y)
	out.f.Add(&input2.x, &input2.y) // F serves as temporary
	out.e.MulEq(&out.f)
	out.e.SubEq(&A)
	out.e.SubEq(&B)   // E = (X1 + Y1) * (X2 + Y2) - A - B == X1*Y2 + Y1*X2
	out.f.Sub(&D, &C) // F = D - C
	out.g.Add(&D, &C) // G = D + C

	A.multiply_by_five()
	out.h.Add(&B, &A) // H = B + 5X1 * X2 = Y1*Y2 - a*X1*X2  (a=-5 is a parameter of the curve)
}

func (out *Point_efgh) sub_stt(input1, input2 *Point_xtw) {
	var temp Point_xtw
	temp.neg_tt(input2)
	out.add_stt(input1, &temp)
}

// same as above, but with z2==1
func (out *Point_xtw) add_tta(input1 *Point_xtw, input2 *Point_axtw) {
	var A, B, C, E, F, G, H FieldElement

	A.Mul(&input1.x, &input2.x) // A = X1 * X2
	B.Mul(&input1.y, &input2.y) // B = Y1 * Y2
	C.Mul(&input1.t, &input2.t)
	C.MulEq(&CurveParameterD_fe) // C = d * T1 * T2
	// D = Z1 D.Mul(&input1.z, &input2.z)  // D = Z1 * Z2
	E.Add(&input1.x, &input1.y)
	F.Add(&input2.x, &input2.y) // F serves as temporary
	E.MulEq(&F)
	E.SubEq(&A)
	E.SubEq(&B)          // E = (X1 + Y1) * (X2 + Y2) - A - B == X1*Y2 + Y1*X2
	F.Sub(&input1.z, &C) // F = D - C
	G.Add(&input1.z, &C) // G = D + C

	A.multiply_by_five()
	H.Add(&B, &A) // H = B + 5X1 * X2 = Y1*Y2 - a*X1*X2  (a=-5 is a parameter of the curve)

	out.x.Mul(&E, &F) // X3 = E * F
	out.y.Mul(&G, &H) // Y3 = G * H
	out.t.Mul(&E, &H) // T3 = E * H
	out.z.Mul(&F, &G) // Z3 = F * G
}

func (out *Point_xtw) sub_tta(input1 *Point_xtw, input2 *Point_axtw) {
	var temp2 Point_axtw
	temp2.neg_aa(input2)
	out.add_tta(input1, &temp2)
}

func (out *Point_xtw) sub_tat(input1 *Point_axtw, input2 *Point_xtw) {
	var temp2 Point_xtw
	temp2.neg_tt(input2)
	out.add_tta(&temp2, input1)
}

func (out *Point_efgh) add_sta(input1 *Point_xtw, input2 *Point_axtw) {
	var A, B, C FieldElement
	A.Mul(&input1.x, &input2.x) // A = X1 * X2
	B.Mul(&input1.y, &input2.y) // B = Y1 * Y2
	C.Mul(&input1.t, &input2.t)
	C.MulEq(&CurveParameterD_fe) // C = d * T1 * T2
	// D = Z1 D.Mul(&input1.z, &input2.z)  // D = Z1 * Z2
	out.e.Add(&input1.x, &input1.y)
	out.f.Add(&input2.x, &input2.y) // F serves as temporary
	out.e.MulEq(&out.f)
	out.e.SubEq(&A)
	out.e.SubEq(&B)          // E = (X1 + Y1) * (X2 + Y2) - A - B == X1*Y2 + Y1*X2
	out.f.Sub(&input1.z, &C) // F = D - C
	out.g.Add(&input1.z, &C) // G = D + C

	A.multiply_by_five()
	out.h.Add(&B, &A) // H = B + 5X1 * X2 = Y1*Y2 - a*X1*X2  (a=-5 is a parameter of the curve)
}

func (out *Point_efgh) sub_sta(input1 *Point_xtw, input2 *Point_axtw) {
	var temp2 Point_axtw
	temp2.neg_aa(input2)
	out.add_sta(input1, &temp2)
}

func (out *Point_efgh) sub_sat(input1 *Point_axtw, input2 *Point_xtw) {
	var temp2 Point_xtw
	temp2.neg_tt(input2)
	out.add_sta(&temp2, input1)
}

// same as above, but with z1==z2==1
func (out *Point_xtw) add_taa(input1 *Point_axtw, input2 *Point_axtw) {
	var A, B, C, E, F, G, H FieldElement

	A.Mul(&input1.x, &input2.x) // A = X1 * X2
	B.Mul(&input1.y, &input2.y) // B = Y1 * Y2
	C.Mul(&input1.t, &input2.t)
	C.MulEq(&CurveParameterD_fe) // C = d * T1 * T2
	// D = 1 == Z1 * Z2
	E.Add(&input1.x, &input1.y)
	F.Add(&input2.x, &input2.y) // F serves as temporary
	E.MulEq(&F)
	E.SubEq(&A)
	E.SubEq(&B)                 // E = (X1 + Y1) * (X2 + Y2) - A - B == X1*Y2 + Y1*X2
	F.Sub(&FieldElementOne, &C) // F = D - C == 1 - C
	G.Add(&FieldElementOne, &C) // G = D + C == 1 + C

	A.multiply_by_five()
	H.Add(&B, &A) // H = B + 5X1 * X2 = Y1*Y2 - a*X1*X2  (a=-5 is a parameter of the curve)

	out.x.Mul(&E, &F) // X3 = E * F
	out.y.Mul(&G, &H) // Y3 = G * H
	out.t.Mul(&E, &H) // T3 = E * H
	H.Square(&C)
	out.z.Sub(&FieldElementOne, &H) // Z3 = F * G == 1 - C^2
}

func (out *Point_xtw) sub_taa(input1 *Point_axtw, input2 *Point_axtw) {
	var temp2 Point_axtw
	temp2.neg_aa(input2)
	out.add_taa(input1, &temp2)
}

func (out *Point_efgh) add_saa(input1 *Point_axtw, input2 *Point_axtw) {
	var A, B, C FieldElement

	A.Mul(&input1.x, &input2.x) // A = X1 * X2
	B.Mul(&input1.y, &input2.y) // B = Y1 * Y2
	C.Mul(&input1.t, &input2.t)
	C.MulEq(&CurveParameterD_fe) // C = d * T1 * T2
	// D = 1 == Z1 * Z2
	out.e.Add(&input1.x, &input1.y)
	out.f.Add(&input2.x, &input2.y) // F serves as temporary
	out.e.MulEq(&out.f)
	out.e.SubEq(&A)
	out.e.SubEq(&B)                 // E = (X1 + Y1) * (X2 + Y2) - A - B == X1*Y2 + Y1*X2
	out.f.Sub(&FieldElementOne, &C) // F = D - C == 1 - C
	out.g.Add(&FieldElementOne, &C) // G = D + C == 1 + C

	A.multiply_by_five()
	out.h.Add(&B, &A) // H = B + 5X1 * X2 = Y1*Y2 - a*X1*X2  (a=-5 is a parameter of the curve)
}

func (out *Point_efgh) sub_saa(input1 *Point_axtw, input2 *Point_axtw) {
	var temp2 Point_axtw
	temp2.neg_aa(input2)
	out.add_saa(input1, &temp2)
}
