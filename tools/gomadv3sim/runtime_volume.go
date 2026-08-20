package gomadv3sim

import (
	"errors"
	_ "unsafe"
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

//go:linkname gomadVolumeBegin internal/gomadfs.BeginSimulationVolumes
func gomadVolumeBegin(uint64, []byte) ([]byte, bool)

//go:linkname gomadInitializeVolumeFilesystem os.gomadInitializeSimulationFilesystem
func gomadInitializeVolumeFilesystem()

//go:linkname gomadVolumeRegister internal/gomadfs.RegisterSimulationVolumes
func gomadVolumeRegister(uint64) ([]byte, bool)

//go:linkname gomadVolumeRevoke internal/gomadfs.RevokeSimulationVolumes
func gomadVolumeRevoke(uint64, bool, bool) ([]byte, bool)

//go:linkname gomadVolumeEnumerate internal/gomadfs.EnumerateSimulationVolume
func gomadVolumeEnumerate(uint64, string, uint64, uint64, uint64, uint64, uint64, []byte) ([]byte, bool)

//go:linkname gomadVolumeFinish internal/gomadfs.FinishSimulationVolumes
func gomadVolumeFinish(uint64) ([]byte, bool)

func gomadInterceptRuntimeVolumeBegin(run uint64, config []byte) (error, bool) {
	gomadInitializeVolumeFilesystem()
	encoded, ok := gomadVolumeBegin(run, config)
	if !ok {
		return decodeRuntimeVolumeError(encoded), true
	}
	return nil, true
}

func gomadInterceptRuntimeVolumeRegister(domain uint64) (error, bool) {
	encoded, ok := gomadVolumeRegister(domain)
	if !ok {
		return decodeRuntimeVolumeError(encoded), true
	}
	return nil, true
}

func gomadInterceptRuntimeVolumeRevoke(domain uint64, graceful, persistedOnly bool) (error, bool) {
	encoded, ok := gomadVolumeRevoke(domain, graceful, persistedOnly)
	if !ok {
		return decodeRuntimeVolumeError(encoded), true
	}
	return nil, true
}

func gomadInterceptRuntimeVolumeEnumerate(domain uint64, volume VolumeID, limits VolumeCrashEnumerationLimits, frontier *VolumeCrashFrontier) (VolumeCrashEnumeration, error, bool) {
	encodedFrontier, err := encodeRuntimeVolumeFrontier(frontier)
	if err != nil {
		return VolumeCrashEnumeration{}, err, true
	}
	encoded, ok := gomadVolumeEnumerate(domain, string(volume), limits.States, limits.Operations, limits.Depth, limits.Bytes, limits.WallNanos, encodedFrontier)
	if !ok {
		return VolumeCrashEnumeration{}, decodeRuntimeVolumeError(encoded), true
	}
	page, err := decodeRuntimeVolumeEnumeration(encoded)
	return page, err, true
}

func gomadInterceptRuntimeVolumeFinish(run uint64) (VolumeRecord, error, bool) {
	encoded, ok := gomadVolumeFinish(run)
	if !ok {
		return VolumeRecord{}, decodeRuntimeVolumeError(encoded), true
	}
	record, runtimeErr, err := decodeRuntimeVolumeFinish(encoded)
	if err != nil {
		return VolumeRecord{}, err, true
	}
	if runtimeErr != nil {
		return record, runtimeVolumeErrorValue(*runtimeErr), true
	}
	return record, nil, true
}

type runtimeVolumeError struct {
	Kind           string
	Message        string
	Ordinal        uint64
	ExpectedSHA256 string
	ActualSHA256   string
	Expected       *VolumeTransition
	Actual         *VolumeTransition
}

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
	runtimeErr, err := decodeRuntimeVolumeErrorWire(encoded)
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
