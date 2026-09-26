package bad_handled

func Target(value int) int { return value }

func Hook(value int) (int, int) { return value, 0 }
