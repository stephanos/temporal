package gomad3sim

import (
	"errors"
	_ "unsafe"
)

func runtimeProcessNetworkOperation(uint64, []byte) ([]byte, error) {
	return nil, ErrRuntimeUnavailable
}

func runtimeProcessVolumeOperation(uint64, []byte) ([]byte, error) {
	return nil, ErrRuntimeUnavailable
}

//go:linkname gomadProcessNetworkOperation internal/gomadio.ProcessSimulationNetworkOperation
func gomadProcessNetworkOperation(uint64, []byte) ([]byte, bool)

//go:linkname gomadProcessVolumeOperation internal/gomadfs.ProcessSimulationVolumeOperation
func gomadProcessVolumeOperation(uint64, []byte) ([]byte, bool)

func gomadInterceptRuntimeProcessNetworkOperation(domain uint64, request []byte) ([]byte, error, bool) {
	response, ok := gomadProcessNetworkOperation(domain, request)
	if !ok {
		return nil, errors.New("apply process simulation network operation"), true
	}
	return response, nil, true
}

func gomadInterceptRuntimeProcessVolumeOperation(domain uint64, request []byte) ([]byte, error, bool) {
	response, ok := gomadProcessVolumeOperation(domain, request)
	if !ok {
		return nil, errors.New("apply process simulation volume operation"), true
	}
	return response, nil, true
}
