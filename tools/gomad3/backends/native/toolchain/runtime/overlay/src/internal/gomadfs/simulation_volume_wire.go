// Code generated from simulation/schema/volumewire.json and volumewire.go.tmpl; DO NOT EDIT.
// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gomadfs

import "errors"

const (
	simulationVolumeConfigMagic            = "GOMADVC1"
	simulationVolumeFinishMagic            = "GOMADVR1"
	simulationVolumeErrorMagic             = "GOMADVE1"
	simulationVolumeEnumerationMagic       = "GOMADVQ1"
	simulationVolumeExplorationMagic       = "GOMADVF1"
	maximumSimulationVolumeWireBytes       = 134217728
	maximumSimulationVolumeWireStringBytes = 1048576

	maximumSimulationVolumeNodes       = 64
	maximumSimulationVolumes           = 64
	maximumSimulationVolumeTransitions = 1048576
	maximumSimulationCrashStates       = 65536
	maximumSimulationSnapshotEntries   = 100000
)

type simulationVolumeWireEncoder struct {
	data []byte
	err  error
}

func (encoder *simulationVolumeWireEncoder) uint64(value uint64) {
	if encoder.err != nil {
		return
	}
	if len(encoder.data) > maximumSimulationVolumeWireBytes-8 {
		encoder.err = errors.New("simulation volume wire value exceeds its byte limit")
		return
	}
	encoder.data = append(encoder.data, byte(value), byte(value>>8), byte(value>>16), byte(value>>24),
		byte(value>>32), byte(value>>40), byte(value>>48), byte(value>>56))
}

func (encoder *simulationVolumeWireEncoder) boolean(value bool) {
	if value {
		encoder.uint64(1)
	} else {
		encoder.uint64(0)
	}
}

func (encoder *simulationVolumeWireEncoder) string(value string) {
	encoder.bytes([]byte(value), maximumSimulationVolumeWireStringBytes)
}

func (encoder *simulationVolumeWireEncoder) bytes(value []byte, maximum uint64) {
	if encoder.err != nil {
		return
	}
	if uint64(len(value)) > maximum {
		encoder.err = errors.New("simulation volume wire byte value exceeds its limit")
		return
	}
	encoder.uint64(uint64(len(value)))
	if encoder.err != nil {
		return
	}
	if len(value) > maximumSimulationVolumeWireBytes-len(encoder.data) {
		encoder.err = errors.New("simulation volume wire value exceeds its byte limit")
		return
	}
	encoder.data = append(encoder.data, value...)
}

type simulationVolumeWireDecoder struct {
	data   []byte
	offset uint64
	err    error
}

func newSimulationVolumeWireDecoder(data []byte, magic string) *simulationVolumeWireDecoder {
	decoder := &simulationVolumeWireDecoder{data: data}
	if len(data) > maximumSimulationVolumeWireBytes || len(data) < len(magic) || string(data[:len(magic)]) != magic {
		decoder.err = errors.New("simulation volume wire header is invalid")
		return decoder
	}
	decoder.offset = uint64(len(magic))
	return decoder
}

func (decoder *simulationVolumeWireDecoder) uint64() uint64 {
	if decoder.err != nil {
		return 0
	}
	if decoder.offset > uint64(len(decoder.data)) || uint64(len(decoder.data))-decoder.offset < 8 {
		decoder.err = errors.New("simulation volume wire value is truncated")
		return 0
	}
	source := decoder.data[decoder.offset : decoder.offset+8]
	decoder.offset += 8
	return uint64(source[0]) | uint64(source[1])<<8 | uint64(source[2])<<16 | uint64(source[3])<<24 |
		uint64(source[4])<<32 | uint64(source[5])<<40 | uint64(source[6])<<48 | uint64(source[7])<<56
}

func (decoder *simulationVolumeWireDecoder) boolean() bool {
	value := decoder.uint64()
	if decoder.err == nil && value > 1 {
		decoder.err = errors.New("simulation volume wire boolean is invalid")
	}
	return value == 1
}

func (decoder *simulationVolumeWireDecoder) string() string {
	return string(decoder.bytes(maximumSimulationVolumeWireStringBytes))
}

func (decoder *simulationVolumeWireDecoder) bytes(maximum uint64) []byte {
	length := decoder.uint64()
	if decoder.err != nil {
		return nil
	}
	if length > maximum || decoder.offset > uint64(len(decoder.data)) || length > uint64(len(decoder.data))-decoder.offset {
		decoder.err = errors.New("simulation volume wire byte value is invalid")
		return nil
	}
	value := append([]byte(nil), decoder.data[decoder.offset:decoder.offset+length]...)
	decoder.offset += length
	return value
}

func (decoder *simulationVolumeWireDecoder) count(maximum uint64) uint64 {
	value := decoder.uint64()
	if decoder.err != nil {
		return 0
	}
	if value > maximum {
		decoder.err = errors.New("simulation volume wire count exceeds its limit")
		return 0
	}
	return value
}

func (decoder *simulationVolumeWireDecoder) finish() error {
	if decoder.err != nil {
		return decoder.err
	}
	if decoder.offset != uint64(len(decoder.data)) {
		return errors.New("simulation volume wire value contains trailing data")
	}
	return nil
}

func encodeSimulationVolumeConfigBody(encoder *simulationVolumeWireEncoder, value simulationVolumeConfig) {
	encoder.uint64(value.Seed)
	encoder.uint64(value.Limits.Operations)
	encoder.uint64(value.Limits.Transitions)
	encoder.uint64(uint64(len(value.Nodes)))
	for _, item := range value.Nodes {
		encodeSimulationVolumeNode(encoder, item)
	}
	encoder.boolean(value.Replay != nil)
	if value.Replay != nil {
		encodeSimulationVolumeRecord(encoder, *value.Replay)
	}

}

func decodeSimulationVolumeConfigBody(decoder *simulationVolumeWireDecoder) simulationVolumeConfig {
	value := simulationVolumeConfig{}
	value.Seed = decoder.uint64()
	value.Limits.Operations = decoder.uint64()
	value.Limits.Transitions = decoder.uint64()
	count3 := decoder.count(64)
	for range count3 {
		if decoder.err != nil {
			break
		}
		value.Nodes = append(value.Nodes, decodeSimulationVolumeNode(decoder))
	}
	if decoder.boolean() {
		decoded := decodeSimulationVolumeRecord(decoder)
		value.Replay = &decoded
	}

	return value
}

func encodeSimulationVolumeNode(encoder *simulationVolumeWireEncoder, value simulationVolumeNodeConfig) {
	encoder.string(string(value.Node))
	encoder.uint64(uint64(len(value.Volumes)))
	for _, item := range value.Volumes {
		encodeSimulationVolumeMount(encoder, item)
	}

}

func decodeSimulationVolumeNode(decoder *simulationVolumeWireDecoder) simulationVolumeNodeConfig {
	value := simulationVolumeNodeConfig{}
	value.Node = decoder.string()
	count1 := decoder.count(64)
	for range count1 {
		if decoder.err != nil {
			break
		}
		value.Volumes = append(value.Volumes, decodeSimulationVolumeMount(decoder))
	}

	return value
}

func encodeSimulationVolumeMount(encoder *simulationVolumeWireEncoder, value VolumeConfig) {
	encoder.string(string(value.ID))
	encoder.string(string(value.Path))
	encoder.uint64(value.CapacityBytes)

}

func decodeSimulationVolumeMount(decoder *simulationVolumeWireDecoder) VolumeConfig {
	value := VolumeConfig{}
	value.ID = decoder.string()
	value.Path = decoder.string()
	value.CapacityBytes = decoder.uint64()

	return value
}

func encodeSimulationVolumeRecord(encoder *simulationVolumeWireEncoder, value simulationVolumeRecord) {
	encoder.uint64(uint64(len(value.Transitions)))
	for _, item := range value.Transitions {
		encodeSimulationVolumeTransition(encoder, item)
	}
	encodeSimulationVolumeSnapshot(encoder, value.Snapshot)

}

func decodeSimulationVolumeRecord(decoder *simulationVolumeWireDecoder) simulationVolumeRecord {
	value := simulationVolumeRecord{}
	count0 := decoder.count(1048576)
	for range count0 {
		if decoder.err != nil {
			break
		}
		value.Transitions = append(value.Transitions, decodeSimulationVolumeTransition(decoder))
	}
	value.Snapshot = decodeSimulationVolumeSnapshot(decoder)

	return value
}

func encodeSimulationVolumeTransition(encoder *simulationVolumeWireEncoder, value simulationVolumeTransition) {
	encoder.uint64(value.Ordinal)
	encoder.string(string(value.Kind))
	encoder.string(string(value.Handle.Node))
	encoder.uint64(value.Handle.Incarnation)
	encoder.string(string(value.Volume))
	encoder.uint64(value.Operation)
	encoder.uint64(uint64(len(value.Dependencies)))
	for _, item := range value.Dependencies {
		encoder.uint64(item)
	}
	encoder.uint64(uint64(len(value.SelectedOperations)))
	for _, item := range value.SelectedOperations {
		encoder.uint64(item)
	}
	encoder.string(string(value.Path))
	encoder.string(string(value.Destination))
	encoder.uint64(value.Inode)
	encoder.uint64(value.Offset)
	encoder.uint64(value.Bytes)
	encoder.string(string(value.PayloadSHA256))
	encoder.string(string(value.EffectSHA256))
	encoder.string(string(value.Outcome))

}

func decodeSimulationVolumeTransition(decoder *simulationVolumeWireDecoder) simulationVolumeTransition {
	value := simulationVolumeTransition{}
	value.Ordinal = decoder.uint64()
	value.Kind = decoder.string()
	value.Handle.Node = decoder.string()
	value.Handle.Incarnation = decoder.uint64()
	value.Volume = decoder.string()
	value.Operation = decoder.uint64()
	count6 := decoder.count(1048576)
	value.Dependencies = make([]uint64, 0, count6)
	for range count6 {
		if decoder.err != nil {
			break
		}
		value.Dependencies = append(value.Dependencies, decoder.uint64())
	}
	count7 := decoder.count(1048576)
	value.SelectedOperations = make([]uint64, 0, count7)
	for range count7 {
		if decoder.err != nil {
			break
		}
		value.SelectedOperations = append(value.SelectedOperations, decoder.uint64())
	}
	value.Path = decoder.string()
	value.Destination = decoder.string()
	value.Inode = decoder.uint64()
	value.Offset = decoder.uint64()
	value.Bytes = decoder.uint64()
	value.PayloadSHA256 = decoder.string()
	value.EffectSHA256 = decoder.string()
	value.Outcome = decoder.string()

	return value
}

func encodeSimulationVolumeSnapshot(encoder *simulationVolumeWireEncoder, value simulationVolumeSnapshot) {
	encoder.uint64(uint64(len(value.Volumes)))
	for _, item := range value.Volumes {
		encodeSimulationVolumeStateSnapshot(encoder, item)
	}
	encoder.string(string(value.TransitionSHA256))
	encoder.string(string(value.Identity))

}

func decodeSimulationVolumeSnapshot(decoder *simulationVolumeWireDecoder) simulationVolumeSnapshot {
	value := simulationVolumeSnapshot{}
	count0 := decoder.count(64)
	for range count0 {
		if decoder.err != nil {
			break
		}
		value.Volumes = append(value.Volumes, decodeSimulationVolumeStateSnapshot(decoder))
	}
	value.TransitionSHA256 = decoder.string()
	value.Identity = decoder.string()

	return value
}

func encodeSimulationVolumeStateSnapshot(encoder *simulationVolumeWireEncoder, value simulationVolumeStateSnapshot) {
	encoder.string(string(value.Volume))
	encoder.string(string(value.Node))
	encoder.string(string(value.Mount))
	encoder.uint64(value.CapacityBytes)
	encoder.uint64(uint64(len(value.Persisted)))
	for _, item := range value.Persisted {
		encodeSimulationSnapshotEntry(encoder, item)
	}
	encoder.uint64(uint64(len(value.Volatile)))
	for _, item := range value.Volatile {
		encodeSimulationSnapshotEntry(encoder, item)
	}
	encoder.uint64(value.PendingOperations)
	encoder.string(string(value.PendingSHA256))
	encoder.uint64(value.NextOperation)
	encoder.string(string(value.Identity))

}

func decodeSimulationVolumeStateSnapshot(decoder *simulationVolumeWireDecoder) simulationVolumeStateSnapshot {
	value := simulationVolumeStateSnapshot{}
	value.Volume = decoder.string()
	value.Node = decoder.string()
	value.Mount = decoder.string()
	value.CapacityBytes = decoder.uint64()
	count4 := decoder.count(100000)
	value.Persisted = make([]SnapshotEntry, 0, count4)
	for range count4 {
		if decoder.err != nil {
			break
		}
		value.Persisted = append(value.Persisted, decodeSimulationSnapshotEntry(decoder))
	}
	count5 := decoder.count(100000)
	value.Volatile = make([]SnapshotEntry, 0, count5)
	for range count5 {
		if decoder.err != nil {
			break
		}
		value.Volatile = append(value.Volatile, decodeSimulationSnapshotEntry(decoder))
	}
	value.PendingOperations = decoder.uint64()
	value.PendingSHA256 = decoder.string()
	value.NextOperation = decoder.uint64()
	value.Identity = decoder.string()

	return value
}

func encodeSimulationSnapshotEntry(encoder *simulationVolumeWireEncoder, value SnapshotEntry) {
	encoder.string(string(value.Path))
	encoder.uint64(uint64(value.Mode))
	encoder.string(string(value.Kind))
	encoder.uint64(uint64(value.ModTime))
	encoder.uint64(value.Size)
	encoder.string(string(value.DataSHA256))

}

func decodeSimulationSnapshotEntry(decoder *simulationVolumeWireDecoder) SnapshotEntry {
	value := SnapshotEntry{}
	value.Path = decoder.string()
	value.Mode = uint32(decoder.uint64())
	value.Kind = decoder.string()
	value.ModTime = int64(decoder.uint64())
	value.Size = decoder.uint64()
	value.DataSHA256 = decoder.string()

	return value
}

func encodeSimulationVolumeErrorBody(encoder *simulationVolumeWireEncoder, value simulationVolumeBridgeError) {
	encoder.string(string(value.Kind))
	encoder.string(string(value.Message))
	encoder.uint64(value.Ordinal)
	encoder.string(string(value.ExpectedSHA256))
	encoder.string(string(value.ActualSHA256))
	encoder.boolean(value.Expected != nil)
	if value.Expected != nil {
		encodeSimulationVolumeTransition(encoder, *value.Expected)
	}
	encoder.boolean(value.Actual != nil)
	if value.Actual != nil {
		encodeSimulationVolumeTransition(encoder, *value.Actual)
	}

}

func decodeSimulationVolumeErrorBody(decoder *simulationVolumeWireDecoder) simulationVolumeBridgeError {
	value := simulationVolumeBridgeError{}
	value.Kind = decoder.string()
	value.Message = decoder.string()
	value.Ordinal = decoder.uint64()
	value.ExpectedSHA256 = decoder.string()
	value.ActualSHA256 = decoder.string()
	if decoder.boolean() {
		decoded := decodeSimulationVolumeTransition(decoder)
		value.Expected = &decoded
	}
	if decoder.boolean() {
		decoded := decodeSimulationVolumeTransition(decoder)
		value.Actual = &decoded
	}

	return value
}

func encodeSimulationCrashExplorationBody(encoder *simulationVolumeWireEncoder, value CrashExploration) {
	encoder.string(string(value.Volume))
	encoder.string(string(value.PendingSHA256))
	encoder.bytes(value.Cursor, 1048576)
	encoder.uint64(uint64(len(value.Seen)))
	for _, item := range value.Seen {
		encoder.string(item)
	}
	encoder.string(string(value.Identity))

}

func decodeSimulationCrashExplorationBody(decoder *simulationVolumeWireDecoder) CrashExploration {
	value := CrashExploration{}
	value.Volume = decoder.string()
	value.PendingSHA256 = decoder.string()
	value.Cursor = decoder.bytes(1048576)
	count3 := decoder.count(65536)
	for range count3 {
		if decoder.err != nil {
			break
		}
		value.Seen = append(value.Seen, decoder.string())
	}
	value.Identity = decoder.string()

	return value
}

func encodeSimulationVolumeCrashEntry(encoder *simulationVolumeWireEncoder, value CrashEntry) {
	encoder.string(string(value.Path))
	encoder.uint64(uint64(value.Mode))
	encoder.uint64(uint64(value.Kind))
	encoder.uint64(uint64(value.ModTime))
	encoder.bytes(value.Data, 268435456)

}

func decodeSimulationVolumeCrashEntry(decoder *simulationVolumeWireDecoder) CrashEntry {
	value := CrashEntry{}
	value.Path = decoder.string()
	value.Mode = uint32(decoder.uint64())
	value.Kind = Kind(decoder.uint64())
	value.ModTime = int64(decoder.uint64())
	value.Data = decoder.bytes(268435456)

	return value
}

func encodeSimulationVolumeCrashState(encoder *simulationVolumeWireEncoder, value CrashState) {
	encoder.string(string(value.Volume))
	encoder.string(string(value.PendingSHA256))
	encoder.uint64(uint64(len(value.SelectedOperations)))
	for _, item := range value.SelectedOperations {
		encoder.uint64(item)
	}
	encoder.uint64(uint64(len(value.Entries)))
	for _, item := range value.Entries {
		encodeSimulationVolumeCrashEntry(encoder, item)
	}
	encoder.string(string(value.Identity))

}

func decodeSimulationVolumeCrashState(decoder *simulationVolumeWireDecoder) CrashState {
	value := CrashState{}
	value.Volume = decoder.string()
	value.PendingSHA256 = decoder.string()
	count2 := decoder.count(1048576)
	value.SelectedOperations = make([]uint64, 0, count2)
	for range count2 {
		if decoder.err != nil {
			break
		}
		value.SelectedOperations = append(value.SelectedOperations, decoder.uint64())
	}
	count3 := decoder.count(100000)
	for range count3 {
		if decoder.err != nil {
			break
		}
		value.Entries = append(value.Entries, decodeSimulationVolumeCrashEntry(decoder))
	}
	value.Identity = decoder.string()

	return value
}

func encodeSimulationCrashEnumerationBody(encoder *simulationVolumeWireEncoder, value CrashEnumeration) {
	encoder.uint64(uint64(len(value.States)))
	for _, item := range value.States {
		encodeSimulationVolumeCrashState(encoder, item)
	}
	encoder.boolean(value.ChoiceExploration != nil)
	if value.ChoiceExploration != nil {
		encodeSimulationCrashExplorationBody(encoder, *value.ChoiceExploration)
	}
	encoder.boolean(value.Complete)
	encoder.string(string(value.Capacity))

}

func decodeSimulationCrashEnumerationBody(decoder *simulationVolumeWireDecoder) CrashEnumeration {
	value := CrashEnumeration{}
	count0 := decoder.count(65536)
	for range count0 {
		if decoder.err != nil {
			break
		}
		value.States = append(value.States, decodeSimulationVolumeCrashState(decoder))
	}
	if decoder.boolean() {
		decoded := decodeSimulationCrashExplorationBody(decoder)
		value.ChoiceExploration = &decoded
	}
	value.Complete = decoder.boolean()
	value.Capacity = CrashCapacity(decoder.string())

	return value
}

func decodeSimulationVolumeConfig(encoded []byte) (simulationVolumeConfig, error) {
	decoder := newSimulationVolumeWireDecoder(encoded, simulationVolumeConfigMagic)
	config := decodeSimulationVolumeConfigBody(decoder)
	if err := decoder.finish(); err != nil {
		return simulationVolumeConfig{}, err
	}
	return config, nil
}

func encodeSimulationVolumeFinish(record simulationVolumeRecord, bridge *simulationVolumeBridgeError) ([]byte, error) {
	encoder := &simulationVolumeWireEncoder{data: []byte(simulationVolumeFinishMagic)}
	encodeSimulationVolumeRecord(encoder, record)
	encoder.boolean(bridge != nil)
	if bridge != nil {
		encodeSimulationVolumeErrorBody(encoder, *bridge)
	}
	return encoder.data, encoder.err
}

func encodeSimulationVolumeBridgeError(source error) []byte {
	var bridge *simulationVolumeBridgeError
	if !errors.As(source, &bridge) {
		bridge = &simulationVolumeBridgeError{Kind: "runtime", Message: source.Error()}
	}
	encoder := &simulationVolumeWireEncoder{data: []byte(simulationVolumeErrorMagic)}
	encodeSimulationVolumeErrorBody(encoder, *bridge)
	if encoder.err == nil {
		return encoder.data
	}
	fallback := &simulationVolumeWireEncoder{data: []byte(simulationVolumeErrorMagic)}
	encodeSimulationVolumeErrorBody(fallback, simulationVolumeBridgeError{Kind: "runtime", Message: "encode simulation volume error"})
	return fallback.data
}

func decodeSimulationCrashExploration(encoded []byte) (*CrashExploration, error) {
	if len(encoded) == 0 {
		return nil, nil
	}
	decoder := newSimulationVolumeWireDecoder(encoded, simulationVolumeExplorationMagic)
	value := decodeSimulationCrashExplorationBody(decoder)
	if err := decoder.finish(); err != nil {
		return nil, err
	}
	return &value, nil
}

func encodeSimulationCrashEnumeration(page CrashEnumeration) ([]byte, error) {
	encoder := &simulationVolumeWireEncoder{data: []byte(simulationVolumeEnumerationMagic)}
	encodeSimulationCrashEnumerationBody(encoder, page)
	return encoder.data, encoder.err
}
