package bandersnatch

import (
	"encoding/binary"
	"errors"
	"io"
)

/*
	This file contains the code to serialize and deserialize curve points.
	We support a short and a long serialization format that are chosen specifically to benefit from the structure of the Bandersnatch curve and have the following desiderata:

	- The short serialization format consists of a single (serialized) field element, i.e. 256 bit
	- the long serialization format contains the short serialzation format as an exact substring
	- the format is prefix-free, even when short and long formats are mixed: this means that we can automatically from the input stream which format was used.
	- serializing and deserializing is an exact round-trip (modulo P=P+A; in fact the P=P+A identification was done precisely to make things work out nicely here), without needing to clear cofactors.
	- Serializing a point in a given format (short or long) gives a unique bitstring. -- No two valid bitstrings of a given format deserialize to the same point.
	- Serializing a point is very efficient (The most expensive part is a simple conversion to affine coordinates, meaning 1 division -- this seems hard to avoid if we want a unique bitstring)
	- Verifying untrusted input upon deserializing is very efficient: The most expensive part is a single Legendre-Symbol computation. No square roots are required for the check!
	- Deserializing is very efficient: Deserializing from short format requires 1 Square root operation (which is essentially the minimum possible -- you cannot have a rational function here), deserializing from long format is basically free (apart from potential subgroup check)
	- Certain leading bits of the serialization format are always 0 (Notably: If it starts (counting bits within bytes high-endian) with with 1..., then the next bit must be 0, the format is long and the 257th bit is 0). This allows (if desired) to add other formats while retaining prefix-freeness.

	Also note that serializing the neutral element in short format results in an all-zero bit string (this may or may not be desired)

	We achieve this by using (essentially), given affine coordinates X,Y:
	X * Sign(Y) as the short serialization format (where sign is +1 if the number is in 1..(p-1)/2, and -1 otherwise -- Note that Y is never 0 and anything +/-1 -valued with f(-v) = -f(v) for v!=0 would work for Sign)
	More precisely, we take X*Sign(Y) and serialize this as a high-endian sequence of bytes. The choice of high-endianness is because X*Sign(Y) fits into 255 bits, so the highest bit (which now comes in the first byte) is zero.
	This helps with prefix-freeness and we can use this bit (which is 0 for the short format) to signal the format early during the scanning process.

	For the long serialization, we use (essentially)
	Y*Sign(Y), X*Sign(Y).
	Note that Y*Sign(Y) fits into 254 bits. We serialize this as (writing the bits within a byte high-endian as well)
	0b10 (2 bits) concatenated HIGHENDIAN_254(Y*Sign(Y)) concatenated HIGHENDIAN_256(X*Sign(Y)), where HIGHENDIAN_foo is the canonical (i.e. smallest non-negative) high-endian foo-bit representation of that number.
	The first 2 prefix bits are used to signal that this is in long format.

	The trick here is that X*Sign(Y) is 2:1 on the full curve, with preimages exactly of the form {P, P+A}.
	Furthermore, for any affine curve point P = (x,y), one of P, P+A is in the p253 subgroup iff
	1-ax^2 is a square (which is equivalent to 1-dx^2 being for x's that appear as rational points)

	Note: This is far from being obvious and I (Gottfried) am not aware whether this particular way of doing Legendre-Checks for cofactor-4 incomplete Edwards curves is known. -- the easiest way is to prove the above is that P is in the subgroup iff P = Q+Q for some rational curve point Q.
	The doubling formula then relates the coordinates of P and Q and this gives that 1-ax^2 must be a square for P in the subgroup. That it is both sufficient and neccessary (modulo A) can the be derived by proving that
	L(P) == L(P+A) and L(P+E1) == L(P+E2) == -L(P) for L(P) := Jacobi symbol of (1-ax^2)
*/

/*
	In addition to short / long serialization formats, there is also the map Curve -> BaseField given by X/Y
	This map is indeed injective on p253, compatible with our P = P+A identification, so it could also be used as a serialization format.
	However, it requires 2 square-root to deserialize from (unless you give 2 field elements), so we did not choose it, but it has the nice property that it avoids the (non-algebraic) Sign
	Still, in situations where we only need an injective map Curve -> BaseField without actually computing the inverse, this map is actually nicer (in particular when there is any chance that we ever want to use NIZKs or SNARKS to prove something about that map).
	So we also provide this as a MapToFieldElement function (this is defined as a function with a CurvePoint as a non-receiver argument).
*/

// These are the errors that can appear in the functions defined in this file, but be aware that in addition to those errors, we can also return
// - errors from field element deserialization such as
//  	ErrNonNormalizedDeserialization = errors.New("during FieldElement deserialization, the read number was not the minimal representative modulo BaseFieldSize")
// - whatever error the given io.Reader / io.Writer returns.
var (
	ErrCannotSerializePointAtInfinity = errors.New("serialization: cannot serialize point at infinity")
	ErrCannotSerializeNaP             = errors.New("serialization: cannot serialize NaP")
)

// Note: If X/Z is not on the curve, we might get either a "not on curve" or "not in subgroup" error. Should we clarify the wording to reflect that?
var (
	ErrXNotInSubgroup     = errors.New("deserialization: received affine X coordinate does not correspond to any point in the p253 subgroup of the Bandersnatch curve")
	ErrXNotOnCurve        = errors.New("deserialization: received affine X coordinate does not correspond to any (finite, rational) point of the Bandersnatch curve")
	ErrNotInSubgroup      = errors.New("deserialization: received affine X and Y coordinates do not correspond to a point in the p253 subgroup of the Bandersnatch curve")
	ErrNotOnCurve         = errors.New("deserialization: received affine X and Y corrdinates do not correspond to a point on the Bandersnatch curve")
	ErrWrongSignY         = errors.New("deserialization: encountered affine Y coordinate with unexpected Sign bit")
	ErrUnrecognizedFormat = errors.New("deserialization: could not automatically detect serialization format")
)

// Default implementation of DeserializeShort in terms of Point_axtw::DeserializeShort
func default_DeserializeShort(receiver CurvePointPtrInterfaceWrite, input io.Reader, trusted IsPointTrusted) (bytes_read int, err error) {
	var result Point_axtw
	bytes_read, err = result.DeserializeShort(input, trusted)
	if err == nil || err == ErrNonNormalizedDeserialization {
		receiver.SetFrom(&result)
	}
	return
}

// Default implementation of DeserializeLong in terms of Point_axtw::DeserializeLong
func default_DeserializeLong(receiver CurvePointPtrInterfaceWrite, input io.Reader, trusted IsPointTrusted) (bytes_read int, err error) {
	var result Point_axtw
	bytes_read, err = result.DeserializeLong(input, trusted)
	if err == nil || err == ErrNonNormalizedDeserialization {
		receiver.SetFrom(&result)
	}
	return
}

// Default implementation of DeserializeAuto in terms of Point_axtw::DeserializeAuto
func default_DeserializeAuto(receiver CurvePointPtrInterfaceWrite, input io.Reader, trusted IsPointTrusted) (bytes_read int, err error) {
	var result Point_axtw
	bytes_read, err = result.DeserializeAuto(input, trusted)
	if err == nil || err == ErrNonNormalizedDeserialization {
		receiver.SetFrom(&result)
	}
	return
}

// Default implementation of SerializeShort in terms of Point_axtw::SerializeShort
func default_SerializeShort(receiver CurvePointPtrInterfaceRead, output io.Writer) (bytes_written int, err error) {
	if receiver.CanRepresentInfinity() {
		if receiver.(CurvePointPtrInterfaceRead_FullCurve).IsAtInfinity() {
			return 0, ErrCannotSerializePointAtInfinity
		}
	}
	if receiver.IsNaP() {
		napEncountered("trying to serialize NaP in short format", false, receiver)
		return 0, ErrCannotSerializeNaP
	}
	var receiver_copy Point_axtw = receiver.AffineExtended()
	bytes_written, err = receiver_copy.SerializeShort(output)
	return
}

// Default implementation of SerializeLong in terms of Point_axtw::SerializeLong
func default_SerializeLong(receiver CurvePointPtrInterfaceRead, output io.Writer) (bytes_written int, err error) {
	if receiver.CanRepresentInfinity() {
		if receiver.(CurvePointPtrInterfaceRead_FullCurve).IsAtInfinity() {
			return 0, ErrCannotSerializePointAtInfinity
		}
	}
	if receiver.IsNaP() {
		napEncountered("trying to serialize NaP in short format", false, receiver)
		return 0, ErrCannotSerializeNaP
	}
	var receiver_copy Point_axtw = receiver.AffineExtended()
	bytes_written, err = receiver_copy.SerializeLong(output)
	return
}

// getXSignY returns X*Sign(Y), which is exactly what we use for our short serialization format.
func (p *Point_axtw) getXSignY() (ret FieldElement) {
	ret = p.x
	if p.y.Sign() < 0 {
		ret.NegEq()
	}
	return
}

// getYSignY returns Y*Sign(Y), which is essentially the second component of our long serialization format.
func (p *Point_axtw) getYSignY() (ret FieldElement) {
	ret = p.y
	if ret.Sign() < 0 {
		ret.NegEq()
	}
	return
}

// MapToFieldElement maps a CurvePoint to a FieldElement as X/Y. Note that for p253, Y is never 0 and this function is actually injective.
// We provide it as a free function with input being a non-receiver argument to avoid having to write it down several times.
func MapToFieldElement(input CurvePointPtrInterfaceRead) (ret FieldElement) {
	ret = input.Y_projective()
	ret.InvEq()
	temp := input.X_projective()
	ret.MulEq(&temp)
	return
}

// SerializeShort serializes the given point in short serialization format. err == nil iff no error occurred.
func (p *Point_axtw) SerializeShort(output io.Writer) (bytes_written int, err error) {
	if p.IsNaP() {
		napEncountered("trying to serialize NaP in short format", false, p)
		return 0, ErrCannotSerializeNaP
	}
	xSigny := p.getXSignY()
	bytes_written, err = xSigny.Serialize(output, binary.BigEndian)
	return
}

// SerializeLong serializes the given point in long serialization format. err == nil iff no error occurred.
func (p *Point_axtw) SerializeLong(output io.Writer) (bytes_written int, err error) {
	if p.IsNaP() {
		napEncountered("trying to serialize NaP in long format", false, p)
		return 0, ErrCannotSerializeNaP
	}
	ySignY := p.getYSignY()
	bytes_written, err = ySignY.SerializeWithPrefix(output, PrefixBits(0b10), 2, binary.BigEndian)
	if err != nil {
		return
	}
	bytes_just_written, err := p.SerializeShort(output)
	bytes_written += bytes_just_written
	return
}

// affineFromXSignY is used during deserialization. It constructs an affine Point_axtw from xSignY with is supposed to hold x * Sign(Y), which uniquely determines the point up to P=P+A.
// if trusted is false, we verify whether the given input actually corresponds to a point on the curve and in the correct subgroup.
func affineFromXSignY(xSignY *FieldElement, trusted bool) (ret Point_axtw, err error) {
	ret.x = *xSignY // xSignY is x * Sign(y), which is correct for ret.x up to sign.

	// Note that recoverYFromXAffine only depends on the square of x, so the sign of xSignY does not matter.
	ret.y, err = recoverYFromXAffine(xSignY, !trusted)
	if err != nil {
		return
	}

	// p.x, p.y are now guaranteed to satisfy the curve equation (pretend that we set p.t := p.x * p.y, which we will do later).
	// The +/- ambiguity of both p.x and p.y corresponds to the set of 4 points of the form {P, -P, P+A, -P+A} for the affine 2-torsion point A.
	// Due to working mod A, we just need to fix the sign:
	if ret.y.Sign() < 0 {
		ret.y.NegEq() // p.x.NegEq() would work just as well, giving a point that differs by +A
	}

	// Set t coordinate correctly:
	ret.t.Mul(xSignY, &ret.y)
	return
}

/*
	Note: The code paths for the checks on untrusted point on the short and long deserialization are rather separate.
	This is just because on the short format deserialization path, we do not need to perform a check whether the point is on the curve (because we constructed the y coo ourselves)
	and on the long format path, the checks share subexpressions, which are only computed once.
*/

// affineFromXYSignY is used during deserialization. It constructs an affine Point_axtw from xTemp=x Sign(Y) and yTemp = y Sign(Y).
// This function is just an internal helper function; note that it actually modifies xTemp and yTemp, which are no longer needed by the caller anyway.
func affineFromXYSignY(xTemp *FieldElement, yTemp *FieldElement, trusted bool) (ret Point_axtw, err error) {
	// If Sign(Y) == 1, then this is obviously correct (provided the input is good); However, if Sign(Y) == -1, this just differs by + A, which we do not care about.
	ret.x = *xTemp
	ret.y = *yTemp
	ret.t.Mul(xTemp, yTemp)
	if !trusted {
		// y * Sign(Y) must have Sign > 0.
		if yTemp.Sign() <= 0 {
			err = ErrWrongSignY
			return
		}

		// We compute 1-ax^2 - y^2 + dt^2, which is 0 iff the point is on the curve (and finite).
		// Observe that the subexpression 1-ax^2 is also used in the subgroup check, so we do that along the way.
		// We reuse xTemp and yTemp as temporaries, using yTemp as accumulator.
		yTemp.Square(xTemp) // x^2

		yTemp.multiply_by_five()      // 5x^2 == -ax^2
		yTemp.AddEq(&FieldElementOne) // 1+5x^2 == 1-ax^2

		if yTemp.Jacobi() < 0 {
			err = ErrNotInSubgroup
			// no return. This way, if we have both "not on curve" and "not in subgroup", we get "not on curve", which is more informative.
		}

		xTemp.Square(&ret.y) // y^2
		yTemp.SubEq(xTemp)   // 1-ax^2 - y^2

		xTemp.Square(&ret.t)             // t^2 == x^2y^2
		xTemp.MulEq(&CurveParameterD_fe) // dt^2
		yTemp.AddEq(xTemp)               // 1 - ax^2 - y^2 + dt^2
		if !yTemp.IsZero() {
			err = ErrNotOnCurve
		}
	}
	return
}

/*
	NOTE: The current behaviour is that p.DeserializeShort/Long/Auto will not overwrite p on error. We could make an exception (some old version did this) to this rule for ErrNonNormalizedDeserialization,
	i.e. if the field element that was read was not in 0 <= . < BaseFieldSize. However, that seems to complicate things needlessly and makes DeserializeAuto behave differently from DeserializeShort/Long, becaue
	we definitely do not want this behaviour for the Auto variant (s)
*/

// DeserialzeShort deserializes from the given input byte stream (expecting it to start with a point in short serialization format) and store the result in the receiver.
// err==nil iff no error occured. trusted should be one of the constants TrustedInput or UntrustedInput.
// For UntrustedInput, we perform a specially-tailored efficient curve and subgroup membership tests.
// Note that long format is considerably more efficient to deserialize.
func (p *Point_axtw) DeserializeShort(input io.Reader, trusted IsPointTrusted) (bytes_read int, err error) {

	// TODO/Q: Should we treat NonNormalized as a hard error instead and keep p untouched?

	// var NonNormalized bool = false // special error flag for reading inputs that are not in the range 0<=. < BaseFieldSize. This error needs special treatment.

	var xTemp FieldElement
	// Read from input. Note that Deserialization gives x * Sign(y), so p.x is only correct up to sign.
	bytes_read, err = xTemp.DeserializeWithPrefix(input, PrefixBits(0), 1, binary.BigEndian)
	if err != nil {
		// If we get a ErrNonNormalizedDeserialization, we continue as if no error had occurred, but remember the error to return it in the end (if no other error happens).
		// if err == ErrNonNormalizedDeserialization {
		// 	// err = nil -- err will be overwritten below anyway
		// 	NonNormalized = true
		// } else {
		return
		// }
	}

	// We write to temp instead of directly to p. This way, p is untouched on errors others than ErrNonNormalizedDeserialization.
	temp, err := affineFromXSignY(&xTemp, trusted.V())
	if err == nil {
		*p = temp
		// if NonNormalized {
		// 			err = ErrNonNormalizedDeserialization
		//	}
	}

	// If NonNormalized was set, we return ErrNonNormalizedDeserializtion as error, but the point is actually correct.
	return
}

// DeserialzeLong deserializes from the given input byte stream (expecting it to start with a point in long serialization format) and store the result in the receiver.
// err==nil iff no error occured. trusted should be one of the constants TrustedInput or UntrustedInput.
// For UntrustedInput, we perform a specially-tailored efficient curve and subgroup membership tests.
// Note that long format is considerably more efficient to deserialize.
func (p *Point_axtw) DeserializeLong(input io.Reader, trusted IsPointTrusted) (bytes_read int, err error) {
	// var NonNormalized bool = false // special error flag for reading inputs that are not in the range 0<=. < BaseFieldSize. This error needs special treatment

	var ySignY, xSignY FieldElement
	bytes_read, err = ySignY.DeserializeWithPrefix(input, PrefixBits(0b10), 2, binary.BigEndian)

	// Abort if error was encountered, unless the error was NonNormalizedDeserialization.
	if err != nil {
		// if err == ErrNonNormalizedDeserialization {
		// 	NonNormalized = true
		// } else {
		return
		// }
	}

	bytes_just_read, err := xSignY.DeserializeWithPrefix(input, PrefixBits(0b0), 1, binary.BigEndian)
	bytes_read += bytes_just_read
	if err != nil {
		// if err == ErrNonNormalizedDeserialization {
		// 	NonNormalized = true
		// } else {
		return
		// }
	}

	// If we get here, we got no error other than ErrNonNormalizedDeserialization so far.
	// We write to temp instead of directly to p, since we only overwrite p if there is no error.
	temp, err := affineFromXYSignY(&xSignY, &ySignY, trusted.V())
	if err == nil {
		*p = temp
		// if NonNormalized {
		// 	err = ErrNonNormalizedDeserialization
		// }
	}
	return
}

// DeserializeAuto deserializes from the given input byte stream (expecting it to start with a point in either short or long serialization format -- it autodetects that) and store the result in the receiver.
// err==nil iff no error occured. trusted should be one of the constants TrustedInput or UntrustedInput.
// For UntrustedInput, we perform a specially-tailored efficient curve and subgroup membership tests.
// Note that long format is considerably more efficient to deserialize.
func (p *Point_axtw) DeserializeAuto(input io.Reader, trusted IsPointTrusted) (bytes_read int, err error) {
	var fieldElement_read FieldElement
	var prefix_read PrefixBits
	var temp Point_axtw
	bytes_read, prefix_read, err = fieldElement_read.deserializeAndGetPrefix(input, 1, binary.BigEndian)

	// The point here is that in long deserialization format, the bit-stream starts with 10..., because
	// the first element (as an integer) has sign >=0, hence is actually at most 254 bits.
	// The second bit after reading a 1 might signal some extension this library does not understand.
	// We want to abort and alert the user with a more meaningful error rather than treat the number just mod BaseFieldSize.
	if err == ErrNonNormalizedDeserialization {
		err = ErrUnrecognizedFormat
	}
	if err != nil {
		return
	}
	if prefix_read == PrefixBits(0b0) {
		// short serialization format
		temp, err = affineFromXSignY(&fieldElement_read, trusted.V())
		if err == nil {
			*p = temp
		}
		return
	} else if prefix_read == PrefixBits(0b1) {
		// long serialization format.
		// Note: We only checked that the uppermost-bit was 1 and the rest was interpreted as number that was checked to be in 0<=.<BaseFieldSize.
		// If the uppermost 2 bits were 11, we would get a field element with Sign < 0; we perform this check even for trusted input.
		// TODO: This can be improved if it turns out to matter: Computing Sign involves changing from Montgomery representation to "standard", so it is actually not as cheap as it might seem.
		// However, we could just get that bit directly from the input. The only reason we do things the current way is due to modularity and the way FieldElement::deserializeAndGetPrefix was designed.
		// Furthermore, we needlessly repeat this check on untrusted input in affineFromXYSignY.
		if fieldElement_read.Sign() < 0 {
			err = ErrUnrecognizedFormat
			return
		}
		// If we get here, the prefix must have beein 0b10, since otherwise we would either hit ErrNonNormalizedDeserialization or the Sign() < 0 above.
		var fieldElement2_read FieldElement
		var bytes_just_read int
		bytes_just_read, err = fieldElement2_read.DeserializeWithPrefix(input, PrefixBits(0b0), 1, binary.BigEndian)
		bytes_read += bytes_just_read
		if err == ErrNonNormalizedDeserialization {
			err = ErrUnrecognizedFormat
		}
		if err != nil {
			return
		}
		temp, err = affineFromXYSignY(&fieldElement2_read, &fieldElement_read, trusted.V())
		if err == nil {
			*p = temp
		}
		return
	} else {
		panic("This cannot happen") // prefix_read must be either 0b0 or 0b1
	}
}

// recoverYFromXAffine computes y from x such that (x,y) is on the curve. Note that the result only depends on x up to sign.
// For valid input x, for which some y exists in the first place, there are always exatly two possible y which differ by sign.
// recoverYFromXAffine makes no guarantees about the choice of y. It need not even be consistent for multiple calls with the same x.
// If checkSubgroup is set to true, we also check that the resulting (+/-x,+/-y) is on the subgroup for some choice of signs.
// Returns err==nil if no error occurred (meaning that some y existed and the subgroup check, if requrested, did not fail).
func recoverYFromXAffine(x *FieldElement, checkSubgroup bool) (y FieldElement, err error) {

	// We have y^2 = (1-ax^2) / (1-dx^2)
	// So, we first compute (1-ax^2) / 1-dx^2
	var num, denom FieldElement

	num.Square(x)                        // x^2, only compute this once
	denom.Mul(&num, &CurveParameterD_fe) // dx^2
	num.multiply_by_five()               // 5x^2 = -ax^2
	num.AddEq(&FieldElementOne)          // 1 - ax^2
	denom.Sub(&FieldElementOne, &denom)  // 1 - dx^2
	// Note that x is in the correct subgroup iff *both* num and denom are squares
	if checkSubgroup {
		if num.Jacobi() < 0 {
			err = ErrXNotInSubgroup
			return
		}
	}
	num.DivideEq(&denom) // (1-ax^2)/(1-dx^2). Note that 1-dx^2 cannot be 0, as d is a non-square.
	if !y.SquareRoot(&num) {
		err = ErrXNotOnCurve
		return
	}
	err = nil // err is nil at this point anyway, but we prefer to be explicit.
	return
}

// isPointOnCurve checks whether the given point is actually on the curve.
// Note: This does NOT verify that the point is in the correct subgroup.
// Note2: On encountering singular values (0:0:0:0), we just return false *without* calling any error handler.
// Note3: This function is only provided for xtw and only used internally.
func (p *Point_xtw) isPointOnCurve() bool {

	// Singular points are not on the curve
	if p.IsNaP() {
		return false
	}

	// check whether x*y == t*z
	var u, v FieldElement
	u.Mul(&p.x, &p.y)
	v.Mul(&p.t, &p.z)
	if !u.IsEqual(&v) {
		return false
	}

	// We now check the main curve equation, i.e. whether ax^2 + y^2 == z^2 + dt^2
	u.Mul(&p.t, &p.t)
	u.MulEq(&CurveParameterD_fe) // u = d*t^2
	v.Mul(&p.z, &p.z)
	u.AddEq(&v) // u= dt^2 + z^2
	v.Mul(&p.y, &p.y)
	u.SubEq(&v) // u = z^2 + dt^2 - y^2
	v.Mul(&p.x, &p.x)
	v.multiply_by_five()
	u.AddEq(&v) // u = z^2 + dt^2 - y^2 + 5x^2 ==  z^2 + dt^2 - y^2 - ax^2
	return u.IsZero()
}

// checkLegendreX(X/Z) checks whether the provided x=X/Z value may be the x-coordinate of a point in the subgroup spanned by p253 and A, assuming the curve equation has a rational solution for the given X/Z.
func checkLegendreX(x FieldElement) bool {
	// x is passed by value. We use it as a temporary.
	x.SquareEq()
	x.multiply_by_five()
	x.AddEq(&FieldElementOne) // 1 + 5x^2 = 1-ax^2
	return x.Jacobi() >= 0    // cannot be ==0, since a is a non-square
}

// checkLegendreX2(x) == checkLegendreX iff an rational y-coo satisfying the curve equation exists.
func checkLegendreX2(x FieldElement) bool {
	x.SquareEq()
	x.MulEq(&CurveParameterD_fe)
	x.Sub(&FieldElementOne, &x) // 1 - dx^2
	return x.Jacobi() >= 0      // cannot be ==0, since d is a non-square
}

// This checks whether the X/Z coordinate may be in the subgroup spanned by p253 and A.
// Note that since this is called on a Point_xtw, we assume that y is set correctly (we do not use y, but in order for the test to be sufficient, we need that some rational y for which the curve equation is satisfied *exists*)
func (p *Point_xtw) legendre_check_point() bool {
	var temp FieldElement
	/// p.MakeAffine()  -- removed in favour of homogenous formula
	temp.Square(&p.x)
	temp.multiply_by_five()
	var zz FieldElement
	zz.Square(&p.z)
	temp.AddEq(&zz) // temp = z^2 + 5x^2 = z^2-ax^2
	result := temp.Jacobi()
	if result == 0 {
		panic("Jacobi symbol of z^2-ax^2 is 0") // Cannot happen, because a is a non-square.
	}
	return result > 0
}
