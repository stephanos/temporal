package duplicate_target

// The manifest lists this Target twice; the package itself is valid.
func Target(value int) int { return value }

func Hook(value int) (int, bool) { return value, false }
