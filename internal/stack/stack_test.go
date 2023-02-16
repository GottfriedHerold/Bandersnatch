package stack

import (
	"testing"

	"github.com/GottfriedHerold/Bandersnatch/internal/testutils"
)

func TestStack(t *testing.T) {
	type S = Stack[int] // shortcut

	var s S = MakeStack[int]()
	testutils.FatalUnless(t, s.Len() == 0, "")
	didPanic := testutils.CheckPanic((*S).Pop, &s)
	testutils.FatalUnless(t, didPanic, "")
	didPanic = testutils.CheckPanic((*S).Top, &s)
	testutils.FatalUnless(t, didPanic, "")
	testutils.FatalUnless(t, s.IsEmpty(), "")

	s.Push(10)
	x := 20
	s.PushPtr(&x)
	testutils.FatalUnless(t, s.Len() == 2, "")
	top := s.Top()
	testutils.FatalUnless(t, *top == 20, "")
	testutils.FatalUnless(t, top != &x, "")
	*top = 19
	testutils.FatalUnless(t, s.Pop() == 19, "")
	testutils.FatalUnless(t, s.Pop() == 10, "")
	testutils.FatalUnless(t, s.IsEmpty(), "")
}
