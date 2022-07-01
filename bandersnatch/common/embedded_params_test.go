package common

import (
	"testing"

	"github.com/GottfriedHerold/Bandersnatch/internal/testutils"
)

type subgroupRestrictionInterface interface {
	SetSubgroupRestriction(bool)
	IsSubgroupOnly() bool
}

var _ subgroupRestrictionInterface = &SubgroupRestriction{}
var _ subgroupRestrictionInterface = &SubgroupOnly{}

func TestValidiyOfZeroValues(t *testing.T) {
	var sr SubgroupRestriction
	sr.Validate()
	var sronly SubgroupOnly
	sronly.Validate()
	var bh BitHeader
	bh.validate()
}

func TestBitHeader(t *testing.T) {
	var bh BitHeader
	bh.validate() // validate zero value

	bh.SetBitHeader(PrefixBits(0b0101), 4)
	bh2 := MakeBitHeader(PrefixBits(0b0101), 4)
	if bh != bh2 {
		t.Fatalf("Bit headers unexpectedly mismatch")
	}
	if bh.PrefixBits() != 0b101 {
		t.Fatalf("Unexpected Prefix bits returned")
	}
	if bh.PrefixLen() != 4 {
		t.Fatalf("Unexpected Prefix length returned")
	}
	bh.validate()
	bh2.validate()
	bh.SetBitHeader(PrefixBits(0b0101), 3)
	if bh == bh2 {
		t.Fatalf("Bit header unexpectedly match")
	}
	bh.validate()
	bh2.SetBitHeaderFromBitHeader(bh)
	// Note: CheckPanic requires exact match for types of arguments
	if !testutils.CheckPanic(bh.SetBitHeader, PrefixBits(0), uint8(9)) {
		t.Fatalf("No panic when calling SetBitHeader with too long Prefix length")
	}
	if !testutils.CheckPanic(MakeBitHeader, PrefixBits(0), uint8(9)) {
		t.Fatalf("No panic when calling SetBitHeader with too long Prefix length")
	}
	if !testutils.CheckPanic(bh.SetBitHeader, PrefixBits(0b11), uint8(1)) {
		t.Fatalf("No panic when calling SetBitHeader with Prefix length incompatible with prefix bits")
	}
	if !testutils.CheckPanic(MakeBitHeader, PrefixBits(0b11), uint8(1)) {
		t.Fatalf("No panic when calling SetBitHeader with Prefix length incompatible with prefix bits")
	}

}
