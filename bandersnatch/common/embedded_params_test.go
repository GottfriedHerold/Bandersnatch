package common

import "testing"

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
	bh.Validate()
}
