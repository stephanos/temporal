package bodyless_target

// Target has no body and no assembly, which the compiler rejects.
func Target(value int) int

func Hook(value int) (int, bool) { return value, false }
