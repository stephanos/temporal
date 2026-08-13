package body_mismatch

func Target() int { return 42 }

func Hook() (int, bool) { return 0, false }
