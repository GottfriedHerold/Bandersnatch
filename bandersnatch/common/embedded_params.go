package common

// This file defines wrappers (essentially just a data element with a setter/getter method) that are needed because
// either we have some validity constraints that we need to check in the setter or that are struct-embedded in serializers.
//
// For the latter (structs embedded in serializers), note that parameter setting actually goes through reflection, which is a bit easier
// (and consistent, since at least for some parameters we need validity checks)
// with getters/setters, so we want to always have those.

// BitHeader is a "header" consisting of a prefixLen < 8 many extra bits that are included inside a field element as a form of compression.
// The zero value of BitHeader is a valid, but useless length-0 bit header.
type BitHeader struct {
	prefixBits PrefixBits // based on byte. We use a different type to avoid mistaking parameter orders.
	prefixLen  uint8
}

// PrefixBits is a type based on byte
type PrefixBits byte

// MaxLengthPrefixBits is the maximal length of a BitHeader. Since it needs to fit in a byte, this value is 8.
const MaxLengthPrefixBits = 8

// SetBitHeaderFromBitHeader and GetBitHeader are an internal function that
// need to be exported for cross-package and reflect usage:
// our (rather generic) parameter-update functions for serializers go through reflection
// and always require some form of possibly trivial getter / setter methods.

// SetBitHeaderFromBitHeader copies a BitHeader into another.
//
// This function is only exported (and needed) for internal cross-package and reflect usage.
// Plain assignment works just fine.
func (bh *BitHeader) SetBitHeaderFromBitHeader(newBitHeader BitHeader) {
	*bh = newBitHeader
	bh.validate() // not needed, technically. newBitHeader is guaranteed to satisfy this in the first place.
}

// GetBitHeader returns a copy of the given BitHeader.
//
// This function is only exported (and needed) for internal cross-package and reflect usage.
// Plain assignment works just fine.
func (bh *BitHeader) GetBitHeader() BitHeader {
	// Note: No need to make a copy, since we return a value.
	return *bh
}

// SetBitHeader sets the BitHeader to the given prefixBits and prefixLen.
// Only the prefixLen lsbs of prefixBits may be non-zero.
// It panics if the input is invalid.
//
// Note: PrefixBits is based on uint8 == byte.
// You are supposed to write e.g. bh.SetBitHeader(PrefixBits(0b0101), 4)
// with explicit type conversion to PrefixBits in order to not mess up the order of parameters.
func (bh *BitHeader) SetBitHeader(prefixBits PrefixBits, prefixLen uint8) {
	bh.prefixBits = prefixBits
	bh.prefixLen = prefixLen
	bh.validate()
}

// MakeBitHeader creates a new BitHeader with the given prefixBits and prefixLen.
// Only the prefixLen lsbs of prefixBits may be non-zero.
// It panics for invalid inputs.
//
// Note: PrefixBits is based on uint8 == byte.
// You are supposed to write e.g. MakeBitHeader(PrefixBits(0b0101), 4)
// with explicit type conversion to PrefixBits in order to not mess up the order of parameters.
func MakeBitHeader(prefixBits PrefixBits, prefixLen uint8) BitHeader {
	var ret BitHeader = BitHeader{prefixBits: prefixBits, prefixLen: prefixLen}
	ret.validate()
	return ret
}

// PrefixBits obtains the PrefixBits of a BitHeader
func (bh *BitHeader) PrefixBits() PrefixBits {
	return bh.prefixBits
}

// PrefixLen obtains the prefix length of a BitHeader
func (bh *BitHeader) PrefixLen() uint8 {
	return bh.prefixLen
}

// validate ensures the BitHeader is valid. It panics if not.
func (bh *BitHeader) validate() {
	if bh.prefixLen > MaxLengthPrefixBits {
		panic("bandersnatch / serialization: trying to set bit-prefix of length > 8")
	}
	bitFilter := (1 << bh.prefixLen) - 1 // bitmask of the form 0b0..01..1 ending with prefixLen 1s
	if bitFilter&int(bh.prefixBits) != int(bh.prefixBits) {
		panic("bandersnatch / serialization: trying to set BitHeader with a prefix and length, where the prefix has bits set that are not among the length many least significant bits")
	}
}

// Note: The exported Validate should actually never fail, because all Setters run the non-exported validate to ensure consistency; the zero value is valid.

// Validate ensures the BitHeader is valid. This can actually never fail and is provided to satisfy (internal) interfaces.
func (bh *BitHeader) Validate() {
	bh.validate() // just double-checking.
}

// implicit interface with methods SetSubgroupRestriction(bool) and IsSubgroupOnly() bool defined in tests only.
// Since we use reflection, we don't need the explicit interface here.

// SubgroupRestriction is a type (intended for struct embedding into serializers) wrapping a bool
// that determines whether the serializer only works for subgroup elements.
// The purpose is to have getters and setters.
type SubgroupRestriction struct {
	subgroupOnly bool
}

func (sr *SubgroupRestriction) SetSubgroupRestriction(restrict bool) {
	sr.subgroupOnly = restrict
}

func (sr *SubgroupRestriction) IsSubgroupOnly() bool {
	return sr.subgroupOnly
}

func (sr *SubgroupRestriction) Validate() {}

func (sr *SubgroupRestriction) RecognizedParameters() []string {
	return []string{"SubgroupOnly"}
}

// SubgroupOnly is a type wrapping a bool constant true that indicates that the serializer only works for subgroup elements. Used as embedded struct to forward setter and getter methods to reflect.
type SubgroupOnly struct {
}

func (sr *SubgroupOnly) IsSubgroupOnly() bool {
	return true
}

func (sr *SubgroupOnly) SetSubgroupRestriction(restrict bool) {
	if !restrict {
		panic("bandersnatch / serialization: Trying to unset restriction to subgroup points for a serializer that does not support this")
	}
}

func (sr *SubgroupOnly) Validate() {}

func (sr *SubgroupOnly) RecognizedParameters() []string {
	return []string{"SubgroupOnly"}
}
