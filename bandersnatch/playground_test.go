package bandersnatch

import (
	"fmt"
	"testing"

	"github.com/GottfriedHerold/Bandersnatch/internal/callcounters"
)

func TestPlayground(t *testing.T) {
	fmt.Println(callcounters.GetCallCounterStructureReport("--"))
	// printCallCounterStructure()
}
