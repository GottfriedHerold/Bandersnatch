package bandersnatch

import (
	"github.com/GottfriedHerold/Bandersnatch/bandersnatch/common"
	"github.com/GottfriedHerold/Bandersnatch/bandersnatch/fieldElements"
)

type FieldElement = fieldElements.FieldElement

type IsInputTrusted = common.IsInputTrusted

var (
	TrustedInput   = common.TrustedInput
	UntrustedInput = common.UntrustedInput
)
