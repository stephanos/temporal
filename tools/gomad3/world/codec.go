package world

import (
	"bytes"
	"encoding/json"
)

const (
	MaximumSnapshotBytes     = compressedEnvelopeHeaderBytes + 64<<20
	MaximumSnapshotJSONBytes = MaximumSnapshotBytes
)

type worldJSONValue interface {
	Snapshot | ReplayPlan | Transition | Terminal | RequestSnapshot | EventSnapshot | Quiescence | Cancellation | LogicalTime
}

func encodeWorldJSON[T worldJSONValue](value T) ([]byte, error) {
	//gomad:dynamic reviewed 2026-10-13 closed World wire types marshal only caller-owned memory
	initial, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	decoder := json.NewDecoder(bytes.NewReader(initial))
	decoder.UseNumber()
	var normalized any
	//gomad:dynamic reviewed 2026-10-13 stdlib JSON normalization decodes only the just-marshaled World value
	if err := decoder.Decode(&normalized); err != nil {
		return nil, err
	}
	var output bytes.Buffer
	encoder := json.NewEncoder(&output)
	encoder.SetEscapeHTML(false)
	//gomad:dynamic reviewed 2026-10-13 stdlib JSON normalization encodes only its memory-owned intermediate value
	if err := encoder.Encode(normalized); err != nil {
		return nil, err
	}
	return bytes.TrimSuffix(output.Bytes(), []byte{'\n'}), nil
}

func EncodeSnapshot(snapshot Snapshot) ([]byte, error) {
	if _, err := Restore(snapshot, nil); err != nil {
		return nil, err
	}
	if err := validateSnapshotStrings(snapshot); err != nil {
		return nil, err
	}
	return encodeCompressedSnapshot(snapshot, CompressedSnapshotProfile())
}

func DecodeSnapshot(data []byte) (Snapshot, error) {
	snapshot, _, _, _, _, err := decodeCompressedSnapshot(data, CompressedSnapshotProfile())
	return snapshot, err
}
