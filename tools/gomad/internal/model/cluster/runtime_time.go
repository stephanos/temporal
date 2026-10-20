package cluster

func runtimeProcessTimeAdvance(int64) error {
	return ErrRuntimeUnavailable
}

func runtimeProcessTimeCurrent() int64 {
	return 0
}

func runtimeProcessTimeArrivals() uint32 {
	return 0
}
