package hooks

import "github.com/temporalio/gomad/gomadruntime"

func MathRand_runtime_rand() uint64 {
	return gomadruntime.Fastrand64()
}
