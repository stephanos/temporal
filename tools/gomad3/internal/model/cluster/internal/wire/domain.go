package wire

import (
	"errors"
	"fmt"
	"go.temporal.io/server/tools/gomad3/internal/model/cluster/internal/history"
	"sort"
)

func DecodeOutputs(encoded []byte) ([]history.OutputObservation, error) {
	const headerBytes = 16
	const recordBytes = 72
	if len(encoded) < headerBytes || string(encoded[:8]) != "GOMADO1\x00" {
		return nil, errors.New("simulation runtime output has an invalid header")
	}
	count := readRuntimeUint64(encoded[8:16])
	if count > 4096*2 {
		return nil, errors.New("simulation runtime output count exceeds the execution limit")
	}
	outputs := make([]history.OutputObservation, 0, count)
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
		var stream history.OutputStream
		switch record[0] {
		case 1:
			stream = "stdout"
		case 2:
			stream = "stderr"
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
		node := history.NodeID(encoded[offset : offset+nodeBytes])
		offset += nodeBytes
		retained := append([]byte(nil), encoded[offset:offset+retainedBytes]...)
		offset += retainedBytes
		discardedBytes := totalBytes - retainedBytes
		outputs = append(outputs, history.OutputObservation{
			Handle:         history.NodeHandle{Node: node, Incarnation: incarnation},
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
		return history.OutputBefore(outputs[left], outputs[right])
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
