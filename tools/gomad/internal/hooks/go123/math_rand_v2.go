package go123

import "github.com/temporalio/gomad/gomadruntime"

func MathRandV2_runtime_rand() uint64 {
	return gomadruntime.Fastrand64()
}
