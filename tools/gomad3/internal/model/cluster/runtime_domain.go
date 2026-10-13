package cluster

import (
	"errors"
	"go.temporal.io/server/tools/gomad3/internal/model/cluster/internal/wire"
	_ "unsafe"
)

var ErrRuntimeUnavailable = errors.New("gomad3 simulation runtime is unavailable")

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

//go:linkname gomadSimulationEnabled runtime.gomadDeterministicEnabled
func gomadSimulationEnabled() bool

//go:linkname gomadSimulationBegin internal/gomadsim.Begin
func gomadSimulationBegin(uint64, uint64) uint64

//go:linkname gomadSimulationRegister internal/gomadsim.Register
func gomadSimulationRegister(uint64, string, string, uint64) uint64

//go:linkname gomadSimulationEnter internal/gomadsim.Enter
func gomadSimulationEnter(uint64) (uint64, bool)

//go:linkname gomadSimulationLeave internal/gomadsim.Leave
func gomadSimulationLeave(uint64)

//go:linkname gomadSimulationRevoke internal/gomadsim.Revoke
func gomadSimulationRevoke(uint64) bool

//go:linkname gomadSimulationFinish internal/gomadsim.Finish
func gomadSimulationFinish(uint64) ([]byte, bool)

func gomadInterceptRuntimeDomainAvailable() (bool, bool) {
	return gomadSimulationEnabled(), true
}

func gomadInterceptRuntimeDomainBegin(observationBytes, maximumDomains uint64) (uint64, error, bool) {
	token := gomadSimulationBegin(observationBytes, maximumDomains)
	if token == 0 {
		return 0, errors.New("gomad3 simulation runtime run capacity exhausted"), true
	}
	return token, nil, true
}

func gomadInterceptRuntimeDomainRegister(run uint64, node NodeID, address string, incarnation uint64) (uint64, error, bool) {
	token := gomadSimulationRegister(run, string(node), address, incarnation)
	if token == 0 {
		return 0, errors.New("gomad3 simulation runtime domain capacity exhausted"), true
	}
	return token, nil, true
}

func gomadInterceptRuntimeDomainEnter(token uint64) (uint64, error, bool) {
	previous, ok := gomadSimulationEnter(token)
	if !ok {
		return 0, ErrStaleIncarnation, true
	}
	return previous, nil, true
}

func gomadInterceptRuntimeDomainLeave(previous uint64) bool {
	gomadSimulationLeave(previous)
	return true
}

func gomadInterceptRuntimeDomainRevoke(token uint64) (error, bool) {
	if !gomadSimulationRevoke(token) {
		return ErrStaleIncarnation, true
	}
	return nil, true
}

func gomadInterceptRuntimeDomainFinish(run uint64) ([]OutputObservation, error, bool) {
	encoded, ok := gomadSimulationFinish(run)
	if !ok {
		return nil, errors.New("finish gomad3 simulation runtime run"), true
	}
	outputs, err := wire.DecodeOutputs(encoded)
	return outputs, err, true
}
