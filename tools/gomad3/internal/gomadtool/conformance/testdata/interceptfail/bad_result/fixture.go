package bad_result

func Target(value int) int { return value }

func Hook(value int) (string, bool) { return "", false }
