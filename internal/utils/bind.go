package utils

// various functions to bind arguments to functions.

func Bind2[Arg1 any, Arg2 any](f func(arg1 Arg1, arg2 Arg2), arg2 Arg2) func(arg1 Arg1) {
	return func(arg1 Arg1) { f(arg1, arg2) }
}

func Bind23[Arg1 any, Arg2 any, Arg3 any](f func(arg1 Arg1, arg2 Arg2, arg3 Arg3), arg2 Arg2, arg3 Arg3) func(arg1 Arg1) {
	return func(arg1 Arg1) { f(arg1, arg2, arg3) }
}

func Bind234[Arg1 any, Arg2 any, Arg3 any, Arg4 any](f func(arg1 Arg1, arg2 Arg2, arg3 Arg3, arg4 Arg4), arg2 Arg2, arg3 Arg3, arg4 Arg4) func(arg1 Arg1) {
	return func(arg1 Arg1) { f(arg1, arg2, arg3, arg4) }
}

func Bind2345[Arg1 any, Arg2 any, Arg3 any, Arg4 any, Arg5 any](f func(arg1 Arg1, arg2 Arg2, arg3 Arg3, arg4 Arg4, arg5 Arg5), arg2 Arg2, arg3 Arg3, arg4 Arg4, arg5 Arg5) func(arg1 Arg1) {
	return func(arg1 Arg1) { f(arg1, arg2, arg3, arg4, arg5) }
}
