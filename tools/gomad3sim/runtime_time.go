//go:build !gomad3_toolchain

package gomad3sim

func runtimeProcessTimeAdvance(int64) error {
	return ErrRuntimeUnavailable
}

func runtimeProcessTimeCurrent() int64 {
	return 0
}

func runtimeProcessTimeArrivals() uint32 {
	return 0
}
