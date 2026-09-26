package variadic

func Target(values ...int) int { return len(values) }

func Hook(values []int) (int, bool) { return len(values), false }
