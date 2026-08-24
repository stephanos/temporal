package gomad3sim

import (
	"errors"
	"fmt"
	"sort"
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
	outputs, err := decodeRuntimeOutputs(encoded)
	return outputs, err, true
}

func decodeRuntimeOutputs(encoded []byte) ([]OutputObservation, error) {
	const headerBytes = 16
	const recordBytes = 72
	if len(encoded) < headerBytes || string(encoded[:8]) != "GOMADO1\x00" {
		return nil, errors.New("simulation runtime output has an invalid header")
	}
	count := readRuntimeUint64(encoded[8:16])
	if count > MaximumScenarioActions*2 {
		return nil, errors.New("simulation runtime output count exceeds the execution limit")
	}
	outputs := make([]OutputObservation, 0, count)
	offset := uint64(headerBytes)
	for range count {
		if uint64(len(encoded))-offset < recordBytes {
			return nil, errors.New("simulation runtime output record is truncated")
		}
		record := encoded[offset : offset+recordBytes]
		for _, reserved := range record[1:8] {
			if reserved != 0 {
				return nil, errors.New("simulation runtime output reserved bytes are nonzero")
			}
		}
		var stream OutputStream
		switch record[0] {
		case 1:
			stream = OutputStdout
		case 2:
			stream = OutputStderr
		default:
			return nil, fmt.Errorf("simulation runtime output stream = %d", record[0])
		}
		incarnation := readRuntimeUint64(record[8:16])
		nodeBytes := readRuntimeUint64(record[16:24])
		retainedBytes := readRuntimeUint64(record[24:32])
		totalBytes := readRuntimeUint64(record[32:40])
		offset += recordBytes
		if nodeBytes > uint64(len(encoded))-offset || retainedBytes > uint64(len(encoded))-offset-nodeBytes || totalBytes < retainedBytes {
			return nil, errors.New("simulation runtime output metadata is inconsistent")
		}
		node := NodeID(encoded[offset : offset+nodeBytes])
		offset += nodeBytes
		retained := append([]byte(nil), encoded[offset:offset+retainedBytes]...)
		offset += retainedBytes
		discardedBytes := totalBytes - retainedBytes
		outputs = append(outputs, OutputObservation{
			Handle:         NodeHandle{Node: node, Incarnation: incarnation},
			Stream:         stream,
			Bytes:          retained,
			FullSHA256:     fmt.Sprintf("sha256:%x", record[40:72]),
			TotalBytes:     totalBytes,
			RetainedBytes:  retainedBytes,
			DiscardedBytes: discardedBytes,
			Truncated:      discardedBytes != 0,
		})
	}
	if offset != uint64(len(encoded)) {
		return nil, errors.New("simulation runtime output contains trailing data")
	}
	sort.Slice(outputs, func(left, right int) bool {
		return outputBefore(outputs[left], outputs[right])
	})
	return outputs, nil
}

func readRuntimeUint64(source []byte) uint64 {
	return uint64(source[0]) |
		uint64(source[1])<<8 |
		uint64(source[2])<<16 |
		uint64(source[3])<<24 |
		uint64(source[4])<<32 |
		uint64(source[5])<<40 |
		uint64(source[6])<<48 |
		uint64(source[7])<<56
}
