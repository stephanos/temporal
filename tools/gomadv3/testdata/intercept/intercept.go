package intercept

var handled bool

func SetHandled(value bool) {
	handled = value
}

func Function(value int) (int, string) {
	return value + 1, "original"
}

func gomadInterceptFunction(value int) (int, string, bool) {
	if !handled {
		return 0, "", false
	}
	return value + 10, "intercepted", true
}

func Notify(value *int) {
	*value++
}

func gomadInterceptNotify(value *int) bool {
	if !handled {
		return false
	}
	*value += 10
	return true
}

type Value struct {
	Base int
}

func (value *Value) Add(delta int) int {
	return value.Base + delta
}

func gomadInterceptValueAdd(value *Value, delta int) (int, bool) {
	if !handled {
		return 0, false
	}
	if value == nil {
		return -1, true
	}
	return value.Base + delta + 100, true
}
