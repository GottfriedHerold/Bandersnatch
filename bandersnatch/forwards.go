package bandersnatch

import (
	"github.com/GottfriedHerold/Bandersnatch/bandersnatch/common"
	"github.com/GottfriedHerold/Bandersnatch/bandersnatch/fieldElement"
)

type FieldElement = fieldElement.FieldElement

type IsPointTrusted = common.IsPointTrusted

var (
	TrustedInput   = common.TrustedInput
	UntrustedInput = common.UntrustedInput
)
