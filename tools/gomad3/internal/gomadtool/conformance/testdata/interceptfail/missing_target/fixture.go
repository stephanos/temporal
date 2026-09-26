package missing_target

// Hook has no Target to attach to.
func Hook(value int) (int, bool) { return value, false }
