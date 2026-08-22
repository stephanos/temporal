package interceptcaller

import "gomadv3.test/intercept"

func Function(value int) (int, string) {
	return intercept.Function(value)
}

func Add(value *intercept.Value, delta int) int {
	return value.Add(delta)
}
