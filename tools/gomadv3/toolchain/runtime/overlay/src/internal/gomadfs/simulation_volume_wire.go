// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gomadfs

import "errors"

const (
	maximumSimulationVolumeWireBytes       = 128 << 20
	maximumSimulationVolumeWireStringBytes = 1 << 20
	maximumSimulationVolumeNodes           = 64
	maximumSimulationVolumes               = 64
	maximumSimulationVolumeTransitions     = 1 << 20
	maximumSimulationCrashStates           = 1 << 16
	maximumSimulationSnapshotEntries       = 100_000
)

const (
	simulationVolumeConfigMagic      = "GOMADVC1"
	simulationVolumeFinishMagic      = "GOMADVR1"
	simulationVolumeErrorMagic       = "GOMADVE1"
	simulationVolumeEnumerationMagic = "GOMADVQ1"
	simulationVolumeFrontierMagic    = "GOMADVF1"
)

type simulationVolumeWireEncoder struct {
	data []byte
	err  error
}

func (encoder *simulationVolumeWireEncoder) uint64(value uint64) {
	if encoder.err != nil {
		return
	}
	encoder.data = append(encoder.data,
		byte(value), byte(value>>8), byte(value>>16), byte(value>>24),
		byte(value>>32), byte(value>>40), byte(value>>48), byte(value>>56),
	)
	if len(encoder.data) > maximumSimulationVolumeWireBytes {
		encoder.err = errors.New("simulation volume wire value exceeds its byte limit")
	}
}

func (encoder *simulationVolumeWireEncoder) boolean(value bool) {
	if value {
		encoder.uint64(1)
		return
	}
	encoder.uint64(0)
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
	encoder.data = append(encoder.data, value...)
	if len(encoder.data) > maximumSimulationVolumeWireBytes {
		encoder.err = errors.New("simulation volume wire value exceeds its byte limit")
	}
}

type simulationVolumeWireDecoder struct {
	data   []byte
	offset uint64
	err    error
}

func newSimulationVolumeWireDecoder(data []byte, magic string) *simulationVolumeWireDecoder {
	decoder := &simulationVolumeWireDecoder{data: data}
	if len(data) < len(magic) || string(data[:len(magic)]) != magic {
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
	if decoder.err == nil && value > maximum {
		decoder.err = errors.New("simulation volume wire count exceeds its limit")
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

func decodeSimulationVolumeConfig(encoded []byte) (simulationVolumeConfig, error) {
	if len(encoded) == 0 || len(encoded) > maximumSimulationVolumeWireBytes {
		return simulationVolumeConfig{}, errors.New("simulation volume configuration is invalid")
	}
	decoder := newSimulationVolumeWireDecoder(encoded, simulationVolumeConfigMagic)
	config := simulationVolumeConfig{Seed: decoder.uint64()}
	config.Limits = simulationVolumeLimits{Operations: decoder.uint64(), Transitions: decoder.uint64()}
	for range decoder.count(maximumSimulationVolumeNodes) {
		node := simulationVolumeNodeConfig{Node: decoder.string()}
		for range decoder.count(maximumSimulationVolumes) {
			node.Volumes = append(node.Volumes, VolumeConfig{ID: decoder.string(), Path: decoder.string(), CapacityBytes: decoder.uint64()})
		}
		config.Nodes = append(config.Nodes, node)
	}
	if decoder.boolean() {
		replay := decodeSimulationVolumeRecord(decoder)
		config.Replay = &replay
	}
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
		encodeSimulationVolumeError(encoder, bridge)
	}
	return encoder.data, encoder.err
}

func encodeSimulationVolumeBridgeError(source error) []byte {
	var bridge *simulationVolumeBridgeError
	if !errors.As(source, &bridge) {
		bridge = &simulationVolumeBridgeError{Kind: "runtime", Message: source.Error()}
	}
	encoder := &simulationVolumeWireEncoder{data: []byte(simulationVolumeErrorMagic)}
	encodeSimulationVolumeError(encoder, bridge)
	if encoder.err == nil {
		return encoder.data
	}
	fallback := &simulationVolumeWireEncoder{data: []byte(simulationVolumeErrorMagic)}
	encodeSimulationVolumeError(fallback, &simulationVolumeBridgeError{Kind: "runtime", Message: "encode simulation volume error"})
	return fallback.data
}

func encodeSimulationVolumeRecord(encoder *simulationVolumeWireEncoder, record simulationVolumeRecord) {
	encoder.uint64(uint64(len(record.Transitions)))
	for _, transition := range record.Transitions {
		encodeSimulationVolumeTransition(encoder, transition)
	}
	encodeSimulationVolumeSnapshot(encoder, record.Snapshot)
}

func decodeSimulationVolumeRecord(decoder *simulationVolumeWireDecoder) simulationVolumeRecord {
	record := simulationVolumeRecord{}
	for range decoder.count(maximumSimulationVolumeTransitions) {
		record.Transitions = append(record.Transitions, decodeSimulationVolumeTransition(decoder))
	}
	record.Snapshot = decodeSimulationVolumeSnapshot(decoder)
	return record
}

func encodeSimulationVolumeTransition(encoder *simulationVolumeWireEncoder, transition simulationVolumeTransition) {
	encoder.uint64(transition.Ordinal)
	encoder.string(transition.Kind)
	encoder.string(transition.Handle.Node)
	encoder.uint64(transition.Handle.Incarnation)
	encoder.string(transition.Volume)
	encoder.uint64(transition.Operation)
	encodeSimulationVolumeUint64s(encoder, transition.Dependencies)
	encodeSimulationVolumeUint64s(encoder, transition.SelectedOperations)
	encoder.string(transition.Path)
	encoder.string(transition.Destination)
	encoder.uint64(transition.Inode)
	encoder.uint64(transition.Offset)
	encoder.uint64(transition.Bytes)
	encoder.string(transition.PayloadSHA256)
	encoder.string(transition.EffectSHA256)
	encoder.string(transition.Outcome)
}

func decodeSimulationVolumeTransition(decoder *simulationVolumeWireDecoder) simulationVolumeTransition {
	transition := simulationVolumeTransition{Ordinal: decoder.uint64(), Kind: decoder.string()}
	transition.Handle = simulationVolumeHandle{Node: decoder.string(), Incarnation: decoder.uint64()}
	transition.Volume = decoder.string()
	transition.Operation = decoder.uint64()
	transition.Dependencies = decodeSimulationVolumeUint64s(decoder, maximumSimulationVolumeTransitions)
	transition.SelectedOperations = decodeSimulationVolumeUint64s(decoder, maximumSimulationVolumeTransitions)
	transition.Path = decoder.string()
	transition.Destination = decoder.string()
	transition.Inode = decoder.uint64()
	transition.Offset = decoder.uint64()
	transition.Bytes = decoder.uint64()
	transition.PayloadSHA256 = decoder.string()
	transition.EffectSHA256 = decoder.string()
	transition.Outcome = decoder.string()
	return transition
}

func encodeSimulationVolumeUint64s(encoder *simulationVolumeWireEncoder, values []uint64) {
	encoder.uint64(uint64(len(values)))
	for _, value := range values {
		encoder.uint64(value)
	}
}

func decodeSimulationVolumeUint64s(decoder *simulationVolumeWireDecoder, maximum uint64) []uint64 {
	values := make([]uint64, 0, decoder.count(maximum))
	for index := uint64(0); index < uint64(cap(values)); index++ {
		values = append(values, decoder.uint64())
	}
	return values
}

func encodeSimulationVolumeSnapshot(encoder *simulationVolumeWireEncoder, snapshot simulationVolumeSnapshot) {
	encoder.uint64(uint64(len(snapshot.Volumes)))
	for _, volume := range snapshot.Volumes {
		encoder.string(volume.Volume)
		encoder.string(volume.Node)
		encoder.string(volume.Mount)
		encoder.uint64(volume.CapacityBytes)
		encodeSimulationSnapshotEntries(encoder, volume.Persisted)
		encodeSimulationSnapshotEntries(encoder, volume.Volatile)
		encoder.uint64(volume.PendingOperations)
		encoder.string(volume.PendingSHA256)
		encoder.uint64(volume.NextOperation)
		encoder.string(volume.Identity)
	}
	encoder.string(snapshot.TransitionSHA256)
	encoder.string(snapshot.Identity)
}

func decodeSimulationVolumeSnapshot(decoder *simulationVolumeWireDecoder) simulationVolumeSnapshot {
	snapshot := simulationVolumeSnapshot{}
	for range decoder.count(maximumSimulationVolumes) {
		volume := simulationVolumeStateSnapshot{Volume: decoder.string(), Node: decoder.string(), Mount: decoder.string(), CapacityBytes: decoder.uint64()}
		volume.Persisted = decodeSimulationSnapshotEntries(decoder)
		volume.Volatile = decodeSimulationSnapshotEntries(decoder)
		volume.PendingOperations = decoder.uint64()
		volume.PendingSHA256 = decoder.string()
		volume.NextOperation = decoder.uint64()
		volume.Identity = decoder.string()
		snapshot.Volumes = append(snapshot.Volumes, volume)
	}
	snapshot.TransitionSHA256 = decoder.string()
	snapshot.Identity = decoder.string()
	return snapshot
}

func encodeSimulationSnapshotEntries(encoder *simulationVolumeWireEncoder, entries []SnapshotEntry) {
	encoder.uint64(uint64(len(entries)))
	for _, entry := range entries {
		encoder.string(entry.Path)
		encoder.uint64(uint64(entry.Mode))
		encoder.string(entry.Kind)
		encoder.uint64(uint64(entry.ModTime))
		encoder.uint64(entry.Size)
		encoder.string(entry.DataSHA256)
	}
}

func decodeSimulationSnapshotEntries(decoder *simulationVolumeWireDecoder) []SnapshotEntry {
	entries := make([]SnapshotEntry, 0, decoder.count(maximumSimulationSnapshotEntries))
	for index := uint64(0); index < uint64(cap(entries)); index++ {
		entries = append(entries, SnapshotEntry{
			Path: decoder.string(), Mode: uint32(decoder.uint64()), Kind: decoder.string(), ModTime: int64(decoder.uint64()),
			Size: decoder.uint64(), DataSHA256: decoder.string(),
		})
	}
	return entries
}

func encodeSimulationVolumeError(encoder *simulationVolumeWireEncoder, bridge *simulationVolumeBridgeError) {
	encoder.string(bridge.Kind)
	encoder.string(bridge.Message)
	encoder.uint64(bridge.Ordinal)
	encoder.string(bridge.ExpectedSHA256)
	encoder.string(bridge.ActualSHA256)
	encoder.boolean(bridge.Expected != nil)
	if bridge.Expected != nil {
		encodeSimulationVolumeTransition(encoder, *bridge.Expected)
	}
	encoder.boolean(bridge.Actual != nil)
	if bridge.Actual != nil {
		encodeSimulationVolumeTransition(encoder, *bridge.Actual)
	}
}

func decodeSimulationCrashFrontier(encoded []byte) (*CrashFrontier, error) {
	if len(encoded) == 0 {
		return nil, nil
	}
	decoder := newSimulationVolumeWireDecoder(encoded, simulationVolumeFrontierMagic)
	frontier := decodeSimulationCrashFrontierBody(decoder)
	if err := decoder.finish(); err != nil {
		return nil, err
	}
	return &frontier, nil
}

func decodeSimulationCrashFrontierBody(decoder *simulationVolumeWireDecoder) CrashFrontier {
	frontier := CrashFrontier{Volume: decoder.string(), PendingSHA256: decoder.string()}
	frontier.Cursor = decoder.bytes(maximumSimulationVolumeTransitions)
	for range decoder.count(maximumSimulationCrashStates) {
		frontier.Seen = append(frontier.Seen, decoder.string())
	}
	frontier.Identity = decoder.string()
	return frontier
}

func encodeSimulationCrashFrontierBody(encoder *simulationVolumeWireEncoder, frontier CrashFrontier) {
	encoder.string(frontier.Volume)
	encoder.string(frontier.PendingSHA256)
	encoder.bytes(frontier.Cursor, maximumSimulationVolumeTransitions)
	encoder.uint64(uint64(len(frontier.Seen)))
	for _, identity := range frontier.Seen {
		encoder.string(identity)
	}
	encoder.string(frontier.Identity)
}

func encodeSimulationCrashEnumeration(page CrashEnumeration) ([]byte, error) {
	encoder := &simulationVolumeWireEncoder{data: []byte(simulationVolumeEnumerationMagic)}
	encoder.uint64(uint64(len(page.States)))
	for _, state := range page.States {
		encoder.string(state.Volume)
		encoder.string(state.PendingSHA256)
		encodeSimulationVolumeUint64s(encoder, state.SelectedOperations)
		encoder.uint64(uint64(len(state.Entries)))
		for _, entry := range state.Entries {
			encoder.string(entry.Path)
			encoder.uint64(uint64(entry.Mode))
			encoder.uint64(uint64(entry.Kind))
			encoder.uint64(uint64(entry.ModTime))
			encoder.bytes(entry.Data, MaximumFileBytes)
		}
		encoder.string(state.Identity)
	}
	encoder.boolean(page.Frontier != nil)
	if page.Frontier != nil {
		encodeSimulationCrashFrontierBody(encoder, *page.Frontier)
	}
	encoder.boolean(page.Complete)
	encoder.string(string(page.Capacity))
	return encoder.data, encoder.err
}
