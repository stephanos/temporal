package bad_parameter

func Target(value int) int { return value }

func Hook(value string) (int, bool) { return len(value), false }
