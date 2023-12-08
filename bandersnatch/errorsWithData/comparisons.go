package errorsWithData

type EqualityComparisonFunction func(any, any) (result bool)

func comparison_very_naive_old(x, y any) (equal bool, reason error) {
	return x == y, nil
}
