//go:build !gomadv3_toolchain

package gomadv3sim

func runtimeProcessTimeAdvance(int64) error {
	return ErrRuntimeUnavailable
}

func runtimeProcessTimeArrivals() uint32 {
	return 0
}
