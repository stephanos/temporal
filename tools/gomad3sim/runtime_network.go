package gomad3sim

import (
	"errors"
	_ "unsafe"
)

func runtimeNetworkBegin(uint64, []byte) error {
	return ErrRuntimeUnavailable
}

func runtimeNetworkPartition(uint64, NodeID, NodeID, bool) error {
	return ErrRuntimeUnavailable
}

func runtimeNetworkHeal(uint64, NodeID, NodeID, bool) error {
	return ErrRuntimeUnavailable
}

func runtimeNetworkDelay(uint64, NodeID, NodeID, uint64, bool) error {
	return ErrRuntimeUnavailable
}

func runtimeNetworkGroup(uint64, []NodeID, []NodeID, bool) error {
	return ErrRuntimeUnavailable
}

func runtimeNetworkRevoke(uint64, bool) error {
	return ErrRuntimeUnavailable
}

func runtimeNetworkFinish(uint64) (NetworkRecord, error) {
	return NetworkRecord{}, ErrRuntimeUnavailable
}

//go:linkname gomadNetworkBegin internal/gomadio.BeginSimulation
func gomadNetworkBegin(uint64, []byte) ([]byte, bool)

//go:linkname gomadNetworkPartition internal/gomadio.PartitionSimulation
func gomadNetworkPartition(uint64, string, string, bool) ([]byte, bool)

//go:linkname gomadNetworkHeal internal/gomadio.HealSimulation
func gomadNetworkHeal(uint64, string, string, bool) ([]byte, bool)

//go:linkname gomadNetworkDelay internal/gomadio.DelaySimulation
func gomadNetworkDelay(uint64, string, string, uint64, bool) ([]byte, bool)

//go:linkname gomadNetworkGroup internal/gomadio.ChangeSimulationGroup
func gomadNetworkGroup(uint64, []string, []string, bool) ([]byte, bool)

//go:linkname gomadNetworkRevoke internal/gomadio.RevokeSimulation
func gomadNetworkRevoke(uint64, bool) ([]byte, bool)

//go:linkname gomadNetworkFinish internal/gomadio.FinishSimulation
func gomadNetworkFinish(uint64) ([]byte, bool)

func gomadInterceptRuntimeNetworkBegin(run uint64, config []byte) (error, bool) {
	encoded, ok := gomadNetworkBegin(run, config)
	if !ok {
		return decodeRuntimeNetworkError(encoded), true
	}
	return nil, true
}

func gomadInterceptRuntimeNetworkPartition(run uint64, left, right NodeID, symmetric bool) (error, bool) {
	encoded, ok := gomadNetworkPartition(run, string(left), string(right), symmetric)
	if !ok {
		return decodeRuntimeNetworkError(encoded), true
	}
	return nil, true
}

func gomadInterceptRuntimeNetworkHeal(run uint64, left, right NodeID, symmetric bool) (error, bool) {
	encoded, ok := gomadNetworkHeal(run, string(left), string(right), symmetric)
	if !ok {
		return decodeRuntimeNetworkError(encoded), true
	}
	return nil, true
}

func gomadInterceptRuntimeNetworkDelay(run uint64, left, right NodeID, delayNanos uint64, symmetric bool) (error, bool) {
	encoded, ok := gomadNetworkDelay(run, string(left), string(right), delayNanos, symmetric)
	if !ok {
		return decodeRuntimeNetworkError(encoded), true
	}
	return nil, true
}

func gomadInterceptRuntimeNetworkGroup(run uint64, left, right []NodeID, enabled bool) (error, bool) {
	leftNodes := make([]string, len(left))
	for index, node := range left {
		leftNodes[index] = string(node)
	}
	rightNodes := make([]string, len(right))
	for index, node := range right {
		rightNodes[index] = string(node)
	}
	encoded, ok := gomadNetworkGroup(run, leftNodes, rightNodes, enabled)
	if !ok {
		return decodeRuntimeNetworkError(encoded), true
	}
	return nil, true
}

func gomadInterceptRuntimeNetworkRevoke(domain uint64, graceful bool) (error, bool) {
	encoded, ok := gomadNetworkRevoke(domain, graceful)
	if !ok {
		return decodeRuntimeNetworkError(encoded), true
	}
	return nil, true
}

func gomadInterceptRuntimeNetworkFinish(run uint64) (NetworkRecord, error, bool) {
	encoded, ok := gomadNetworkFinish(run)
	if !ok {
		return NetworkRecord{}, decodeRuntimeNetworkError(encoded), true
	}
	record, runtimeErr, err := decodeRuntimeNetworkFinish(encoded)
	if err != nil {
		return NetworkRecord{}, err, true
	}
	if runtimeErr != nil {
		return record, runtimeNetworkErrorValue(*runtimeErr), true
	}
	return record, nil, true
}

type runtimeNetworkError struct {
	Kind           string             `json:"kind"`
	Message        string             `json:"message"`
	Ordinal        uint64             `json:"ordinal,omitempty"`
	ExpectedSHA256 string             `json:"expected_sha256,omitempty"`
	ActualSHA256   string             `json:"actual_sha256,omitempty"`
	Expected       *NetworkTransition `json:"expected,omitempty"`
	Actual         *NetworkTransition `json:"actual,omitempty"`
}

type runtimeNetworkReplayDivergenceCarrier interface {
	GomadSimulationNetworkReplayDivergence() []byte
}

func runtimeNetworkReplayDivergence(source error) error {
	var carrier runtimeNetworkReplayDivergenceCarrier
	if !errors.As(source, &carrier) {
		return nil
	}
	encoded := carrier.GomadSimulationNetworkReplayDivergence()
	if len(encoded) == 0 {
		return nil
	}
	return decodeRuntimeNetworkError(encoded)
}

func decodeRuntimeNetworkError(encoded []byte) error {
	runtimeErr, err := decodeRuntimeNetworkErrorWire(encoded)
	if err != nil {
		return errors.New("simulation runtime network returned an invalid error")
	}
	return runtimeNetworkErrorValue(runtimeErr)
}

func runtimeNetworkErrorValue(runtimeErr runtimeNetworkError) error {
	if runtimeErr.Kind == "replay" {
		return &ReplayDivergenceError{Divergence: ReplayDivergence{
			Dimension: ReplayDimensionNetwork, Ordinal: runtimeErr.Ordinal,
			ExpectedSHA256: runtimeErr.ExpectedSHA256, ActualSHA256: runtimeErr.ActualSHA256,
			ExpectedNetwork: runtimeErr.Expected, ActualNetwork: runtimeErr.Actual,
		}}
	}
	return errors.New(runtimeErr.Message)
}
