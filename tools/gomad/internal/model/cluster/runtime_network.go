package cluster

import (
	"errors"

	"go.temporal.io/server/tools/gomad/internal/model/cluster/internal/history"
	"go.temporal.io/server/tools/gomad/internal/model/cluster/internal/wire"
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

type runtimeNetworkError = history.RuntimeNetworkError

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
	runtimeErr, err := wire.DecodeNetworkError(encoded)
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
