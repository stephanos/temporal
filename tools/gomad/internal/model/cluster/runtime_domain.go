package cluster

import "errors"

var ErrRuntimeUnavailable = errors.New("gomad simulation runtime is unavailable")

func runtimeDomainAvailable() bool {
	return false
}

func runtimeDomainBegin(uint64, uint64) (uint64, error) {
	return 0, ErrRuntimeUnavailable
}

func runtimeDomainRegister(uint64, NodeID, string, uint64) (uint64, error) {
	return 0, ErrRuntimeUnavailable
}

func runtimeDomainEnter(uint64) (uint64, error) {
	return 0, ErrRuntimeUnavailable
}

func runtimeDomainLeave(uint64) {}

func runtimeDomainRevoke(uint64) error {
	return ErrRuntimeUnavailable
}

func runtimeDomainFinish(uint64) ([]OutputObservation, error) {
	return nil, ErrRuntimeUnavailable
}
