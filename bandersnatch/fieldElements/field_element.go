package fieldElements

import (
	"fmt"
	"io"
	"math/big"
	"math/rand"

	"github.com/GottfriedHerold/Bandersnatch/bandersnatch/bandersnatchErrors"
	"github.com/GottfriedHerold/Bandersnatch/bandersnatch/common"
)

// TODO: Package DOC

// FieldElementInterface_common is the non-generic part of the [FieldElementInterface] interface satisfied by the field element implementation
// for the field of definition of the Bandersnatch curve.
// Note that there is usually little reason to use an interface rather than a concrete type for field elements and the library is designed with using concrete types in mind.
type FieldElementInterface_common interface {
	IsZero() bool // Checks whether the given field element is 0
	IsOne() bool  // Checks whether the given field element is 1
	SetZero()     // Sets the given field element to 0
	SetOne()      // Sets the given field element to 1

	Sign() int             // Returns the sign in {-1, 0, +1} of the representation of minimal absolute value
	Jacobi() int           // z.Jacobi() in {-1,0,+1} returns the Jacobi/Legendre Symbol of (z/BaseFieldSize).
	SquareEq()             // z.SquareEq sets z = z*z
	NegEq()                // z.NegEq sets z = -z
	InvEq()                // z.InvEq() sets z = 1/z. Panics for z==0
	SetBigInt(x *big.Int)  // z.SetBigInt(x) sets z to the value given by x. The value of x does not have to be in [0, BaseFieldSize).
	SetUint256(x *Uint256) // z.SetUint256(x) sets z to the value given by x. The value of x does not have to be in [0, BaseFieldSize).
	SetUint64(x uint64)    // z.SetUint64(x) sets z to the value given by the uint64 x.
	SetInt64(x int64)      // z.SetInt64(x) sets z to the value given by the int64 x. x may be negative.
	ToBigInt() *big.Int    // z.ToBigInt() returns a new [*big.Int] with a reprentation of z in [0, BaseFieldSize)
	ToUint256(x *Uint256)  // z.ToUint256(x) modifies x, setting it to a representation of z in [0, BaseFieldSize). NOTE: The weird API (not returning Uint256) is for efficiency -- Go seems to have a hard time creating the returned value in the callers stack frame.

	// If z is not in the allowed range, we return an error and the first returned value is meaningless not be used.
	// The returned error wraps [ErrCannotRepresentFieldElement] and the actual failing field element can be retrieved from the error with errorWithData.GetParameterFromError(err, "FieldElement")
	ToUint64() (uint64, error) // z.ToUint64() converts a field element in [0,2^64) to uint64.
	ToInt64() (int64, error)   // z.ToInt64() converts a field element in [-2^64,2^63) to int64.

	MulEqFive() // z.MulEqFive sets z = z * 5.
	DoubleEq()  // z.DoubleEq() sets z = z + z == 2*z

	SetRandomUnsafe(rnd *rand.Rand) // DEPRECATED

	fmt.Formatter // allows formatted output of field elements. -- Note that fmt.Formatter should be defined on value receivers TODO: Specify minimal accepted format verbs
	fmt.Stringer  // allows output as string. -- Note that fmt.Stringer (i.e interface{String() string}) should be defined on value receivers.
	// TODO: fmt.Scanner

	// NOTE: These are low-level conversions to []byte, mostly for internal usage to facilitate accessing the internal representation in tests. Users should rarely use those.
	ToBytes(buf []byte)                    // z.ToBytes(buf) writes the internal representation of z to buf, using z.BytesLength() many bytes. This MUST NOT be used for portable serialization.
	SetBytes(buf []byte)                   // z.FromBytes(buf) restores z's internal representation from buf, reading z.BytesLength() many bytes. Note: The stored internal format is not guaranteed to be stable across library versions, Go versions, architecture or anything. We only guarantee internal roundtrip.
	BytesLength() int                      // z.BytesLength() returns the length of buffer needed for ToBytes or SetBytes. Can be called on nil receiver.
	Normalize()                            // z.Normalize() sets the internal representation of z to a default one without changing the value (as field element). We guarantee that if x.IsEqual(&y), then after normalizing both x and y, they have the same internal representation.
	RerandomizeRepresentation(seed uint64) // z.RerandomizeRepresentation(seed) rerandomizes the internal representation of z. Does not change the value as field element. The resulting internal representation must only depend on seed and z as field element (i.e we normalize internally before rerandomizing). This method is useless outside of testing. Note that we give no guarantee about the quality of randomness.
}

// FieldElementInterface is the (generic, due to argument types depending on the receiver type)
// interface satisfied by implementations of the field of definition of the Bandersnatch curve.
//
// Note that there is usually little reason for users to use this interface rather than concrete types and the library has been designed with using concrete types in mind
// (if for no other reason than to avoid the runtime overhead that any kind of polymorphism entails in the Go language).
// The main usage is to facilitate testing. Note that Field elements are always passed as pointers to arithmetic operations. Aliasing is allowed.
type FieldElementInterface[FieldElementPointer any] interface {
	FieldElementInterface_common // contains all methods that don't depend on the generic parameter.

	Add(x, y FieldElementPointer)          // z.Add(&x,&y) performs z = x + y
	Sub(x, y FieldElementPointer)          // z.Sub(&x,&y) performs z = x - y
	Mul(x, y FieldElementPointer)          // z.Mul(&x,&y) performs z = x * y
	Divide(x, y FieldElementPointer)       // z.Divide(&x, &y) performs z = x / y. Panics for y == 0 (including 0/0)
	Double(x FieldElementPointer)          // z.Double(&x) performs z = 2*x = x + x
	Square(x FieldElementPointer)          // z.Square(&x) performs z = x*x
	SquareRoot(x FieldElementPointer) bool // z.SquareRoot(&x) sets z to a square root of x. If no such square root exists, returns false without modifying z. There are no guarantees about the choice of square root (repeated calls with same x may differ).

	MulFive(x FieldElementPointer) // z.MulFive(&x) performs z = 5*x
	Neg(x FieldElementPointer)     // z.Neg(&x) performs z = -x
	Inv(x FieldElementPointer)     // z.Inv(&x) performs z = 1/x. Panics for x == 0

	AddEq(y FieldElementPointer)    // z.AddEq(&y) performs z+=y
	SubEq(y FieldElementPointer)    // z.SubEq(&y) performs z-=y
	MulEq(y FieldElementPointer)    // z.MulEq(&y) performs z*=y
	DivideEq(y FieldElementPointer) // z.DivideEq(&y) performs z/=y. Panics for y == 0

	IsEqual(other FieldElementPointer) bool                                    // x.IsEqual(&y) tests whether x == y (as field elements, not equality of pointers, of course).
	CmpAbs(other FieldElementPointer) (absValuesEqual bool, exactlyEqual bool) // x.CmpAbs(&y) tests whether x == +/-y (first return argument) and whether x==y (second returned argument)

	// Versions for adding / multiplying with "small" numbers. Note that these are mostly convenience functions.
	//
	// Regarding efficiency:
	//   For addition and subtraction,
	//   If a given y is used for multiple field element operations and if the internal representation is in Montgomery form,
	//   it may actually be (much) better to convert to field element once and then use the general methods.
	//   If the internal representation is not in Montgomery form, the special Addition and subtraction methods are slightly faster than the general methods.
	//
	//   For multiplication and division, these special-purpose function may be optimized to be considerably more efficient than the general methods.
	AddInt64(x FieldElementPointer, y int64)      // z.AddInt64(&x, y) performs z = x + y, where y is an int64
	AddUint64(x FieldElementPointer, y uint64)    // z.AddUint64(&x, y) performs z = x + y, where y is an uint64
	SubInt64(x FieldElementPointer, y int64)      // z.SubInt64(&x, y) performs z = x - y, where y is an int64
	SubUint64(x FieldElementPointer, y uint64)    // z.SubUint64(&x, y) performs z = x - y, where y is an uint64
	MulInt64(x FieldElementPointer, y int64)      // z.MulInt64(&x, y) performs z = x * y, where y is an int64. Note that this may be considerably faster than converting y to a field element if y is only used once.
	MulUint64(x FieldElementPointer, y uint64)    // z.MulUint64(&x, y) performs z = x * y, where y is an uint64. Note that this may be considerably faster than converting y to a field element if y is only used once.
	DivideInt64(x FieldElementPointer, y int64)   // z.DivideInt64(&x, y) performs z = x / y, where y is an int64
	DivideUint64(x FieldElementPointer, y uint64) // z.DivideIUint64(&x, y) performs z = x / y, where y is an uint64

	Exp(base FieldElementPointer, exponent *Uint256) // z.Exp(&base,&exponent) performs z = base^exponent. Note that we only support exponent >=0 for simplicity. 0^0 == 1.
}

// TODO: MulInt64, MulUint64, DivideInt64, DivideUint64 not really optimized for main field element implementation at the moment.\
// The issue is that we would want to use mixed Montgomery multiplication (which boils down to NOT doing Montgomery multiplication) for this case.

// NOTE: Serialization is currently provided both as methods and as free functions.

// temporary interface, may be changed.
// NOTE: We have free functions as well that do essentially the same.
type FieldElementSerializeMethods interface {
	Serialize(io.Writer, FieldElementEndianness) (int, bandersnatchErrors.SerializationError)
	Deserialize(io.Reader, FieldElementEndianness) (int, bandersnatchErrors.DeserializationError)
	SerializeWithPrefix(io.Writer, BitHeader, FieldElementEndianness) (int, bandersnatchErrors.SerializationError)
	DeserializeAndGetPrefix(io.Reader, uint8, FieldElementEndianness) (int, common.PrefixBits, bandersnatchErrors.DeserializationError)
	DeserializeWithExpectedPrefix(io.Reader, BitHeader, FieldElementEndianness) (int, bandersnatchErrors.DeserializationError)
}
