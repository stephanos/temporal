package body_mismatch

// The manifest pins a zero fingerprint for Target, so any declaration mismatches.
func Target(value int) int { return value }

func Hook(value int) (int, bool) { return value, false }
