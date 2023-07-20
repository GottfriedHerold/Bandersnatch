package errorsWithData

import (
	"testing"

	"github.com/GottfriedHerold/Bandersnatch/internal/utils"
)

func BenchmarkTokenizeFormatString(b *testing.B) {
	f := func(b *testing.B, s string) {
		prepareBenchmarkErrorsWithData(b)
		for n := 0; n < b.N; n++ {
			Dump_TokenList[n%benchS] = tokenizeInterpolationString(s)
		}
	}

	b.Run("plain String", utils.Bind2(f, `some String with escape chars \% \\ %%`))
	b.Run("with specials", utils.Bind2(f, `a %w %{Param} ${Param} $!m{$w}`))
}
