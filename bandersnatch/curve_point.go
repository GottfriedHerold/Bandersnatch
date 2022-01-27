package bandersnatch

import (
	"fmt"
	"math/rand"
)

/*
 	A CurvePointPtrInterface represents a rational point on the bandersnatch curve.
	The interface is split into Read-only and write parts. This is mostly to clarify writing "general" functions.
	The (somewhat verbose) name is to emphasize this is an interface and that this contains *pointers*.

	Note: Interfaces have been consilidated into big ones in order to make godoc less messy.
*/

// CurvePointPtrInterface is the interface satisfied by all types that represent rational curve points
type CurvePointPtrInterface interface {
	CurvePointPtrInterfaceRead  // contains functions that do not semantically modify the receiver. -- Note that the internal representation may change, which is visible via certain methods.
	CurvePointPtrInterfaceWrite // contains functions that do modify the receiver.
}

// CurvePointPtrInterfaceRead is the read-part of the interface satisfied by all rational curve points.
// Note that all types satisfying this must actually also (and are assumed to) satisfy CurvePointPtrInterfaceWrite -- The read/write distinction only exists to clarify data flow in function signatures.
type CurvePointPtrInterfaceRead interface {
	CurvePointPtrInterfaceBaseRead // contains functions that are Decaf-invariant and are concerned with internal storage. Having these in a separate interface is mostly to avoid code-duplication.

	// functions to check for equality and query properties of points.
	// NOTE: All these Is... methods returning bool MUST check for NaPs
	IsNeutralElement() bool                  // checks whether the received point is the neutral element
	IsNaP() bool                             // checks whether the received point is a NaP (unintialized point or result of computation with such)
	IsEqual(CurvePointPtrInterfaceRead) bool // checks whether the point is equal to another
	IsInSubgroup() bool                      // checks whether the point is inside the subgroup
	IsAtInfinity() bool                      // checks whether the point is at infinity
	// Note: IsE1() bool and IsE2() bool are optionally also present (but really belong to CurvePointPtrInterfaceBaseRead)

	// Calls to other functions (even "read-only") are allowed to modify the internal representation to change to an equivalent point (and thereby change coordinates)
	// Subsequent calls to <foo>_projective (with different foos) are guaranteed to be consistent only if there are no intermediate calls to functions other than those of the form <foo>_projective().
	X_projective() FieldElement                                 // gives the X coordinate in projective twisted Edwards coordinates
	Y_projective() FieldElement                                 // gives the Y coordinate in projective twisted Edwards coordinates
	Z_projective() FieldElement                                 // gives the Z coordinate in projective twisted Edwards coordinates
	XYZ_projective() (FieldElement, FieldElement, FieldElement) // gives (X:Y:Z) coordinates in projective twisted Edwards coordinates. This is equivalent (but possibly MUCH more efficient) to calling each one of X_projective, Y_projective, Z_projective.
	// Note: Types may optionally also provide T_projective and XYTZ_projective

	// <foo>_affine give coordinates of the point in affine coordinates.
	X_affine() FieldElement                  // gives the X=X/Z coordinate in affine twisted Edwards coordinates
	Y_affine() FieldElement                  // gives the Y=Y/Z coordinate in affine twisted Edwards coordinates
	XY_affine() (FieldElement, FieldElement) // gives both X and Y coordinates in affine twisted Edwards coordinates. This is equivalent to (but may be more efficient than) calling both X_affine and Y_affine.
	// Note: Types may optionally also provide T_affine and XYT_affine
}

// CurvePointPtrInterfaceRead is the write-part of the interface satisfied by all rational curve points.
// NOTE: All types satisfying this must actually also (and are assumed to) satisfy CurvePointPtrInterfaceRead -- The read/write distinction in the public interface only exists to clarify data flow in function signatures.
// NOTE: The argument types do not have to be the same as the receiver types. However, if the receiver can only represent subgroup elements, then the arguments usually must as well (we panic otherwise).
// NOTE: The arguments generally need to be pointers
type CurvePointPtrInterfaceWrite interface {
	// Note that all arguments are passed as pointers (or interface values containing pointers)
	SetNeutral()                                                // p.SetNeutral() sets p to the neutral element
	Add(CurvePointPtrInterfaceRead, CurvePointPtrInterfaceRead) // p.Add(x,y) sets p to the sum (according to the elliptic curve group law) x + y.
	Sub(CurvePointPtrInterfaceRead, CurvePointPtrInterfaceRead) // p.Sub(x,y) sets p to the difference (according to the elliptic curve group law) x - y
	Neg(CurvePointPtrInterfaceRead)                             // p.Neg(q) sets p to -q
	Double(CurvePointPtrInterfaceRead)                          // p.Double(q) sets p to q+q. As opposed to p.Add(q,q), this function works even if the type of p can only represent subgroup elements and q does not.
	AddEq(CurvePointPtrInterfaceRead)                           // p.AddEq(q) is shorthand for p.Add(p,q)
	SubEq(CurvePointPtrInterfaceRead)                           // p.SubEq(q) is shorthand for p.Sub(p,q)
	NegEq()                                                     // p.NegEq() is shorthand for p.Neg(p)
	DoubleEq()                                                  // p.DoubleEq() is shorthand for p.Double(p)

	Endo(CurvePointPtrInterfaceRead) // p.Endo(q) sets p to the result of applying the efficient degree-2 endomorphism of the Bandersnatch curve on q
	EndoEq()                         // p.EndoEq() is shorthand for p.Endo(p)

	SetFrom(CurvePointPtrInterfaceRead) // p.SetFrom(q) sets p to (a copy of) the value of q. This is also used to convert between types. Note that it cannot be used to convert from types that store arbitrary curve points to types that only store points on the prime-order subgroup.
}

// Point_xtw_Full and Point_xtw_subgroup actually embed a joint Point_xtw_base type (dito with the other coordinate types).
// These "base" types just store coordinates; how to actually interpret this as a curve point is the job of Point_xtw_Full resp. Point_xtw_Subgroup
// Indeed, *_Full and *_Subgroup may (and do!) interpret coordinates differently because *_Subgroup can work e.g. modulo the affine 2-torsion point Decaf-style.
// Still, to avoid duplication (even more) code, some methods are defined on the *_base version.
// The CurvePointPtrInterfaceBaseRead interface contains the methods that can meaningfully be provided on the *_base types.

// CurvePointPtrInterfaceBaseRead is an subinterface of CurvePointPtrInterfaceRead that contains simple read functions that make sense
// even if we do not know whether the internal representation works modulo A or not.
// This interface is used internally to avoid some code duplication.
type CurvePointPtrInterfaceBaseRead interface {
	fmt.Stringer // aka String() string. Used for debugging, mostly. Note that String() is defined on the VALUE receiver, actually.
	Cloneable    // aka Clone() interface{}. Used to make copies of points via pointers.

	// These functions do not depend on the actual receiver argument and work with nil receivers.
	// TODO: replace by test for presence of IsE1?
	CanRepresentInfinity() bool     // returns true if the type can represent and distinguish the points at infinity
	CanOnlyRepresentSubgroup() bool // returns true if the *type* can represent curve points from outside the prime-order subgroup.
	// If CanRepresentInfinity returns true, the type MUST also satisfy CurvePointPtrInterfaceDistinguishInfinity (The converse is not true due to implementation details)

	// <foo>_decaf_projective() give the X,Y,T,Z - coordinates of either the stored point P or of P+A where A is the affine two-torsion point.
	// NOTE: If P = X:Y:T:Z, then P+A = -X:-Y:T:Z == X:Y:-T:-Z (the latter equality is due to projective equivalence)
	// NOTE: Subsequent queries need only be consistent (incl. the choice of P or P+A) if no other methods are called and the point is not used as an argument to anything in between
	// other than <foo>_decaf_projective() queries.
	// NOTE: For point types that can only store subgroup elements, calling <foo>_decaf_projective is *MUCH* more efficient than calling <foo>_projective()
	X_decaf_projective() FieldElement // gives the X coordinate of P or P+A in extended projective twisted Edwards coordinates
	Y_decaf_projective() FieldElement // gives the Y coordinate of P or P+A in extended projective twisted Edwards coordinates
	T_decaf_projective() FieldElement // gives the T coordinate of P or P+A in extended projective twisted Edwards coordinates
	Z_decaf_projective() FieldElement // gives the Z coordinate of P or P+A in extended projective twisted Edwards coordinates

	// <foo>_decaf_affine() give X,Y,T-coordinates of either the stored point P or of P+A where A is the affine two-torsion point.
	// NOTE: If P=X:Y:T:1 then P+A = -X:-Y:T:1
	// NOTE: Subsequent queries need only be consistent (incl. the choice of P or P+A) if no other methods are called and the point is not used as an argument to anything in between
	// other than <foo>_decaf_affine() queries.
	// NOTE: For point types that can only store subgroup elements, calling <foo>_decaf_affine is *MUCH* more efficient than calling <foo>_affine()
	X_decaf_affine() FieldElement // gives either X or -X in extended affine twisted Edwards coordinates
	Y_decaf_affine() FieldElement // gives either Y or -Y in extended affine twisted Edwards coordinates
	T_decaf_affine() FieldElement // gives the T coordinate in extended affine twisted Edwards coordinates -- equivalent to T_affine() except for the comment about calling methods in between

	// OPTIONAL (depending on values of type query functions mandatory):
	// CurvePointPtrInterfaceDistinguishInfinity i.e.:
	//   	IsE1() bool
	// 		IsE2() bool
}

// Cloneable means that the type (intended to be a pointer) has a Clone() function
type Cloneable interface {
	// This needs to return interface{} due to limitations of the Go language. (Notably a lack of covariance of interfaces)
	Clone() interface{} // Clone() is defined on pointer receivers and returns a pointer to a newly allocated copy.
}

// CurvePointPtrInterfaceDistinguishInfinity contains additional query function that check whether a given point is the E1 or the E2 point at infinity.
// NOTE: We only internally call this function (and type-assert to it) after IsAtInfinity returns true
type CurvePointPtrInterfaceDistinguishInfinity interface {
	IsE1() bool
	IsE2() bool
}

// curvePointPtrInterfaceCooReadExtended is the interface satisfied by curve point types that can be queried for extended twisted Edwards coordinates.
// This means that we have an additional T coordinate that satisfies X*Y = T*Z.
type CurvePointPtrInterfaceCooReadExtended interface {
	CurvePointPtrInterfaceRead

	T_affine() FieldElement                                 // gives the T coordinate in extended affine twisted Edwards coordinates. Since Z==1 on affine coordinates, T_affine=X_affine*Y_affine
	XYT_affine() (FieldElement, FieldElement, FieldElement) // gives X,Y,T coordinates in extended affine twisted Edwards coordinates. -- This is equivalent to, but possibly more efficient than, calling X_affine, Y_affine and T_affine

	// Note that the remark from CurvePointInterfaceRead of not interleaving calls to <foo>_projective with other methods still applies
	T_projective() FieldElement                                                // gives the T coordinate in extended projective twisted Edwards coordinates
	XYTZ_projective() (FieldElement, FieldElement, FieldElement, FieldElement) // gives X:Y:T:Z coordinates in extended projective twisted Edwards coordinates. This is equivalent to calling all 4 coordinate functions, but may be (MUCH) more efficient.
}

// CurvePointPtrInterfaceTestSample is the interface that types need to provide in order to make our testing framework work.
// These add some requirements in addition to CurvePointPtrInterface. -- Note that all of those are only used in testing.
type CurvePointPtrInterfaceTestSample interface {
	CurvePointPtrInterface
	HasDecaf() bool // if true, flipDecaf() must exist AND not change semantics.
	testSampleable
	validateable
	rerandomizeable
	sampleableNaP
	// Optional / Mandatory depending on HasDecaf(), CanRepresentInfinity(), CanOnlyRepresentSubgroup:
	// curvePointPtrInterfaceDecaf i.e.
	//		flipDecaf()
	// torsionAdder
	// curvePointPtrInterfaceTestSampleA i.e.
	// 		SetAffineTwoTorsion
	// curvePointPtrInterfaceTestSampleE
	//		SetE1()
	//		SetE2()
}

// curvePointPtrInterfaceDecaf checks for the presence of a flipDecaf method. This method changes the representation P -> P+A.
// Note that flipDecaf is an internal function and might actually change semantics for a given type due to begin define on *_base types.
// Only if the HasDecaf() method returns true for the given type, then flipDecaf must exist and NOT change semantics.
type curvePointPtrInterfaceDecaf interface {
	flipDecaf()
}

// sampleableNaP is the interface satisfied by curve point types that allow sampling random NaPs
type sampleableNaP interface {
	sampleNaP(rnd *rand.Rand, index int) // sample a random NaP. Certain callers call this with sequential index. This may be used to create NaPs of specified types in a more evenly distributed fashion.
}

// testSampleable curve points can be randomly sampled. This is not exported because the randomness is not good enough for cryptographic purposes.
// This is only used in testing.
type testSampleable interface {
	sampleRandomUnsafe(rnd *rand.Rand) // sample a random curve point
}

// torsionAdder is satisfied by curve points that allows adding 2-torsion points.
// This is an optional interface and is used for testing only. It only makes sense for point types that can represent points outside the prime-order subgroup.
// Note that torsionAddE1, torsionAddE2 can be defined efficiently even if the point type can not represent infinite points.
type torsionAdder interface {
	torsionAddA()  // changes the received point P to P+A, where A is the affine two-torsion point
	torsionAddE1() // changes the received point P to P+E1, where E1 is the E1 point at infinity
	torsionAddE2() // changes the received point P to P+E2, where E2 is the E2 point at infinity
}

// rerandomizable types can rerandomize their internal represenation. This is only used in testing.
type rerandomizeable interface {
	rerandomizeRepresentation(rnd *rand.Rand) // changes the curve point to an equivalent one with a possibly different internal representation
}

// validateable types can validate that their internal representation is valid
type validateable interface {
	Validate() bool // checks whether the internal representation actually is a valid curve point.
}

// curvePointPtrInterfaceTestSampleA is satisfied by point types that can be set to the affine 2-torsion point.
// This interface only makes sense for point types that can represent points outside the prime-order subgroup.
// Conversely, types that satisfy CurvePointPtrInterfaceTestSample and can represent points outside the prime-order subgroup must
// satisfy it for testing to actually work.
type curvePointPtrInterfaceTestSampleA interface {
	SetAffineTwoTorsion() // set the received point to the affine two-torsion point.
}

// curvePointPtrInterfaceTestSampleE is satified by point types that can be set to infinite points.
// This interface only makes sense for point types that can represent and distinguish the points at infinity (which are also outside the prime-order subgroup).
// Such types MUST actually satisfy it for CurvePointPtrInterfaceTestSample to actually work properly.
type curvePointPtrInterfaceTestSampleE interface {
	SetE1() // set the received point to the E1 point at infinity
	SetE2() // set the received point to the E2 point at infinity
}

type thisCurvePointCanRepresentInfinity struct{}
type thisCurvePointCannotRepresentInfinity struct{}
type thisCurvePointCanOnlyRepresentSubgroup struct{}
type thisCurvePointCanRepresentFullCurve struct{}

func (thisCurvePointCanRepresentInfinity) CanRepresentInfinity() bool         { return true }
func (thisCurvePointCannotRepresentInfinity) CanRepresentInfinity() bool      { return false }
func (thisCurvePointCanOnlyRepresentSubgroup) CanOnlyRepresentSubgroup() bool { return true }
func (thisCurvePointCanOnlyRepresentSubgroup) IsInSubgroup() bool             { return true }
func (thisCurvePointCanRepresentFullCurve) CanOnlyRepresentSubgroup() bool    { return false }
func (thisCurvePointCanRepresentFullCurve) HasDecaf() bool                    { return false }

func ensureSubgroupOnly(input CurvePointPtrInterfaceBaseRead) {
	if !input.CanOnlyRepresentSubgroup() {
		panic("curve_point: You are trying to assign (via an operation) to a point type that can only store subgroup points, but the operands are general. This is not allowed. Use explicit conversion instead.")
	}
}
