//go:build !gomad3_toolchain

package gomad3sim

func runtimeProcessTimeAdvance(int64) error {
	return ErrRuntimeUnavailable
}

func runtimeProcessTimeArrivals() uint32 {
	return 0
}
