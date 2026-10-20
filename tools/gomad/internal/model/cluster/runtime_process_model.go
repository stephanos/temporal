package cluster

func runtimeProcessNetworkOperation(uint64, []byte) ([]byte, error) {
	return nil, ErrRuntimeUnavailable
}

func runtimeProcessVolumeOperation(uint64, []byte) ([]byte, error) {
	return nil, ErrRuntimeUnavailable
}
