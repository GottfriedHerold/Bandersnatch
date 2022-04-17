package pointserializer

type subgroupRestrictionInterface interface {
	SetSubgroupRestriction(bool)
	IsSubgroupOnly() bool
}

var _ subgroupRestrictionInterface = &subgroupRestriction{}
var _ subgroupRestrictionInterface = &subgroupOnly{}
