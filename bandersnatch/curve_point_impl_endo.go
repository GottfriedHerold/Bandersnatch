package bandersnatch

/*
	Note: Suffixes like _ttt or _tta refer to the type of input point (with order output, input1 [,input2] )
	t denote extended projective,
	a denotes extended affine (i.e. Z==1)
	s denotes double-projective
*/

// computeEndomorphism_xx computes the Endomorphism from the Bandersnatch paper (degree-2 isogeny with kernel {Neutral, Affine oder-2}) on a given input point.
// The formula given is valid unless the input is a two-torsion point. This means that Neutral and Affine need to be handled explicitly. For points at infinity, we
// do not care, as they are not in the subgroup anyway. Use Endo_FullCurve for that (which handles the case of infinite points)
// Note that our identification of P with P+A is taken care of automatically, as Psi(P) == Psi(P+A) anyway
// Note2: There are two such endomorphisms with that kernel, which are dual to each other and differ by composition with negation. We consistently choose the one from the paper.

func (output *Point_xtw) computeEndomorphism_tt(input *Point_xtw) {
	// The formula used below is valid unless for the input xy==zt is zero, which happens iff the input has order 2 or 1.
	if input.x.IsZero() {
		// Endo(Neutral) == Endo(Affine-order-two) == Neutral element.

		// To avoid problems, we verify that the input is valid, as otherwise, we would "heal" an invalid point -> neutral element, which could hide errors.
		if input.IsNaP() {
			napEncountered("Computing endomorphism of NaP of type xtw", false, input)
			// output NaP on purpose if the check above did not panic. This means Endo(NaP) = NaP
			*output = Point_xtw{}
			return
		}
		*output = NeutralElement_xtw
		return
	}
	var bzz, yy, E, F, G, H FieldElement
	bzz.Square(&input.z)
	yy.Square(&input.y)
	E.Sub(&bzz, &yy)
	E.MulEq(&endo_c_fe) // E = c*(z^2 - y^2)

	bzz.MulEq(&endo_b_fe)
	F.Sub(&yy, &bzz) // F = y^2 - bz^2

	H.Add(&yy, &bzz)
	H.MulEq(&endo_b_fe) // H = b(y^2 + bz^2)

	G.Mul(&input.t, &input.z) // G = t*z == x*y

	output.x.Mul(&E, &F)
	output.y.Mul(&H, &G)
	output.t.Mul(&E, &H)
	output.z.Mul(&F, &G)
}

// same as above, but simpler, since output is efgh directly.
func (output *Point_efgh) computeEndomorphism_st(input *Point_xtw) {
	// The formula used below is valid unless for the input xy==zt is zero, which happens iff the input has order 2 or 1.
	if input.x.IsZero() {
		// Endo(Neutral) == Endo(Affine-order-two) == Neutral element.

		// To avoid problems, we verify that the input is valid, as otherwise, we would "heal" an invalid point -> neutral element, which could hide errors.
		if input.IsNaP() {
			napEncountered("Computing endomorphism of NaP of type xtw", false, input)
			// output NaP on purpose if the check above did not panic. This means Endo(NaP) = NaP
			*output = Point_efgh{}
			return
		}
		*output = NeutralElement_efgh
		return
	}
	var bzz, yy FieldElement
	bzz.Square(&input.z)
	yy.Square(&input.y)
	output.e.Sub(&bzz, &yy)
	output.e.MulEq(&endo_c_fe) // E = c*(z^2 - y^2)

	bzz.MulEq(&endo_b_fe)
	output.f.Sub(&yy, &bzz) // F = y^2 - bz^2

	output.h.Add(&yy, &bzz)
	output.h.MulEq(&endo_b_fe) // H = b(y^2 + bz^2)

	output.g.Mul(&input.t, &input.z) // G = t*z == x*y
}

// same as above, but with z == 1 for the input
func (output *Point_xtw) computeEndomorphism_ta(input *Point_axtw) {
	// The formula used below is valid unless for the input xy==zt is zero, which happens iff the input has order 2 or 1.
	if input.x.IsZero() {
		// Endo(Neutral) == Endo(Affine-order-two) == Neutral element.

		// To avoid problems, we verify that the input is not singular.
		if input.IsNaP() {
			napEncountered("Computing endomorphism of NaP of type axtw", false, input)
			*output = Point_xtw{}
			return
		}
		*output = NeutralElement_xtw
		return
	}
	var yy, E, F, H FieldElement
	// bzz.Square(&input.z)
	yy.Square(&input.y)
	E.Sub(&FieldElementOne, &yy)
	E.MulEq(&endo_c_fe) // E = c*(z^2 - y^2) == c*(1-y^2)

	// bzz.MulEq(&endo_b_fe)
	F.Sub(&yy, &endo_b_fe) // F = y^2 - bz^2 == y^2 - b

	H.Add(&yy, &endo_b_fe)
	H.MulEq(&endo_b_fe) // H = b(y^2 + bz^2) == b (y^2 + b)

	// G == t
	// G.Mul(&input.t, &input.z) // G = t*z == x*y

	output.x.Mul(&E, &F)
	output.y.Mul(&H, &input.t)
	output.t.Mul(&E, &H)
	output.z.Mul(&F, &input.t)
}

// same as above, but with z==1 for input and output being efgh.
func (output *Point_efgh) computeEndomorphism_sa(input *Point_axtw) {
	// The formula used below is valid unless for the input xy==zt is zero, which happens iff the input has order 2 or 1.
	if input.x.IsZero() {
		// Endo(Neutral) == Endo(Affine-order-two) == Neutral element

		// To avoid problems, we verify that the input is not singular.
		if input.IsNaP() {
			napEncountered("Computing endomorphism of NaP of type axtw", false, input)
			*output = Point_efgh{}
			return
		}
		*output = NeutralElement_efgh
		return
	}

	var yy FieldElement
	// bzz.Square(&input.z)
	yy.Square(&input.y)
	output.e.Sub(&FieldElementOne, &yy)
	output.e.MulEq(&endo_c_fe)    // E = c*(z^2 - y^2) == c*(1-y^2)
	output.f.Sub(&yy, &endo_b_fe) // F = y^2 - bz^2 == y^2 - b
	output.g = input.t            // G = t
	output.h.Add(&yy, &endo_b_fe) //
	output.h.MulEq(&endo_b_fe)    // H = b(y^2 + bz^2) == b (y^2 + b)
}

/*
	Computing endomorphism on *input* points with efgh coordinates can be done faster than converting the input to axtw first, because we can skip computing input.t (which we do not need)
	However, we can actually do better by observing that the output's E,F,H are multiples of input.G^2 and output.G is a multiple of input.G.
	We clear powers of G and work in EFGH coos directly, saving a further multiplication and reducing the number of exceptional cases.
	Furthermore, we can also rewrite z^2 - y^2 = ax^2 -dt^2 and rewrite formulas a bit. This has the effect that the sole remaining exceptional case is now at infinity (rather than at N and A).
*/

func (output *Point_efgh) computeEndomorphism_ss(input *Point_efgh) {
	var hh, hhByb, ff, bff, fh FieldElement
	hh.Square(&input.h)               // h^2
	hhByb.Mul(&hh, &endo_binverse_fe) // 1/b * h^2
	ff.Square(&input.f)               // f^2
	bff.Mul(&ff, &endo_b_fe)          // bf^2
	fh.Mul(&input.f, &input.h)        //fh  -- Note: Could do (f+h)^2 - f^2 - h^2 == 2fh to replace the Multiplaction by a Squaring + 2 Subtractions. The factor 2 can be accounted for by scaling the precomputed c constant cost-free.

	// Note that c*(z^2 - y^2) == c(ax^2 -dt^2) == c(ae^2f^2 - de^2h^2) == bcde^2(bf^2 - 1/b h^2)  [We used b^2 == a/d for the last equation]
	// Clearing input.e from output.e and output.g and clearing b * input.g^2 from output.f and output.h gives:
	var temp FieldElement // required since input and output might alias
	temp.Sub(&bff, &hhByb)
	temp.MulEq(&input.e)
	output.e.Mul(&endo_bcd_fe, &temp) // bcd * e * (bf^2 - 1/b h^2) == 1/e * c * (z^2 - y^2)
	output.f.Sub(&hhByb, &ff)         // 1/b h^2 - f^2 == 1/b * 1/g^2 (g^2h^2 - b g^2f^2) == 1/b * 1/g^2 * (y^2- bz^2)
	output.g.Mul(&input.g, &fh)       // fgh == 1/e * efgh == 1/e * xy == 1/e * tz
	output.h.Add(&hh, &bff)           // h^2 + bf^2 = 1/b * 1/g^2 * b * (g^2h^2 + bf^2g^2) == 1/b * 1/g^2 * b(y^2 + bz^2)
}

// Basically equivalent to computeEndomorphism_ss, followed by a converstion to xtw - coordinates
func (output *Point_xtw) computeEndomorphism_ts(input *Point_efgh) {
	var hh, hhByb, ff, bff, fh FieldElement
	hh.Square(&input.h)               // h^2
	hhByb.Mul(&hh, &endo_binverse_fe) // 1/b * h^2
	ff.Square(&input.f)               // f^2
	bff.Mul(&ff, &endo_b_fe)          // bf^2
	fh.Mul(&input.f, &input.h)        //fh  -- Note: Could do (f+h)^2 - f^2 - h^2 == 2fh (the factor 2 can be accounted for by scaling the precomputed c constant) to replace Mult by Square + 3Adds

	// Note that c*(z^2 - y^2) == c(ax^2 -dt^2) == c(ae^2f^2 - de^2h^2) == bcde^2(bf^2 - 1/b h^2)  [Note that b^2 == a/d for the last equation]
	// Clearing input.e from output.e and output.g and clearing b * input.g^2 from output.f and output.h gives:
	var E, F, G, H FieldElement

	// As opposed to computeEndomorphism_ss, we do not need a temporary, because input and output cannot alias, since their types differ.
	E.Sub(&bff, &hhByb)
	E.MulEq(&input.e)
	E.MulEq(&endo_bcd_fe) // bcd * e * (bf^2 - 1/b h^2) == 1/e * c * (z^2 - y^2)
	F.Sub(&hhByb, &ff)    // 1/b h^2 - f^2 == 1/b * 1/g^2 (g^2h^2 - b g^2f^2) == 1/b * 1/g^2 * (y^2- bz^2)
	G.Mul(&input.g, &fh)  // fgh == 1/e * efgh == 1/e * xy == 1/e * tz
	H.Add(&hh, &bff)      // h^2 + bf^2 = 1/b * 1/g^2 * b * (g^2h^2 + bf^2g^2) == 1/b * 1/g^2 * b(y^2 + bz^2)

	output.x.Mul(&E, &F)
	output.y.Mul(&G, &H)
	output.t.Mul(&E, &H)
	output.z.Mul(&F, &G)
}
