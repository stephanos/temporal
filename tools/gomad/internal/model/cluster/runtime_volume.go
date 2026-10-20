package cluster

import (
	"errors"

	"go.temporal.io/server/tools/gomad/internal/model/cluster/internal/history"
	"go.temporal.io/server/tools/gomad/internal/model/cluster/internal/wire"
)

func runtimeVolumeBegin(uint64, []byte) error {
	return ErrRuntimeUnavailable
}

func runtimeVolumeRegister(uint64) error {
	return ErrRuntimeUnavailable
}

func runtimeVolumeRevoke(uint64, bool, bool) error {
	return ErrRuntimeUnavailable
}

func runtimeVolumeEnumerate(uint64, VolumeID, VolumeCrashEnumerationLimits, *VolumeCrashFrontier) (VolumeCrashEnumeration, error) {
	return VolumeCrashEnumeration{}, ErrRuntimeUnavailable
}

func runtimeVolumeFinish(uint64) (VolumeRecord, error) {
	return VolumeRecord{}, ErrRuntimeUnavailable
}

type runtimeVolumeError = history.RuntimeVolumeError

type runtimeVolumeReplayDivergenceCarrier interface {
	GomadSimulationVolumeReplayDivergence() []byte
}

func runtimeVolumeReplayDivergence(source error) error {
	var carrier runtimeVolumeReplayDivergenceCarrier
	if !errors.As(source, &carrier) {
		return nil
	}
	encoded := carrier.GomadSimulationVolumeReplayDivergence()
	if len(encoded) == 0 {
		return nil
	}
	return decodeRuntimeVolumeError(encoded)
}

func decodeRuntimeVolumeError(encoded []byte) error {
	runtimeErr, err := wire.DecodeVolumeError(encoded)
	if err != nil {
		return errors.New("simulation runtime volume returned an invalid error")
	}
	return runtimeVolumeErrorValue(runtimeErr)
}

func runtimeVolumeErrorValue(runtimeErr runtimeVolumeError) error {
	if runtimeErr.Kind == "replay" {
		return &ReplayDivergenceError{Divergence: ReplayDivergence{
			Dimension: ReplayDimensionVolume, Ordinal: runtimeErr.Ordinal,
			ExpectedSHA256: runtimeErr.ExpectedSHA256, ActualSHA256: runtimeErr.ActualSHA256,
			ExpectedVolume: runtimeErr.Expected, ActualVolume: runtimeErr.Actual,
		}}
	}
	return errors.New(runtimeErr.Message)
}
