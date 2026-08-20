package gomadv3sim

import "errors"

const (
	runtimeVolumeConfigMagic      = "GOMADVC1"
	runtimeVolumeFinishMagic      = "GOMADVR1"
	runtimeVolumeErrorMagic       = "GOMADVE1"
	runtimeVolumeEnumerationMagic = "GOMADVQ1"
	runtimeVolumeFrontierMagic    = "GOMADVF1"
)

func encodeRuntimeVolumeConfig(spec Spec) ([]byte, error) {
	capacities := make(map[VolumeID]uint64, len(spec.Volumes))
	for _, volume := range spec.Volumes {
		capacities[volume.ID] = volume.CapacityBytes
	}
	encoder := &runtimeNetworkWireEncoder{data: []byte(runtimeVolumeConfigMagic)}
	encoder.uint64(spec.Seed)
	encoder.uint64(spec.Limits.VolumeOperations)
	encoder.uint64(spec.Limits.VolumeTransitions)
	encoder.uint64(uint64(len(spec.Nodes)))
	for _, node := range spec.Nodes {
		encoder.string(string(node.ID))
		encoder.uint64(uint64(len(node.Volumes)))
		for _, mount := range node.Volumes {
			encoder.string(string(mount.Volume))
			encoder.string(mount.Path)
			encoder.uint64(capacities[mount.Volume])
		}
	}
	encoder.boolean(spec.Replay != nil)
	if spec.Replay != nil {
		encodeRuntimeVolumeRecord(encoder, spec.Replay.Volumes)
	}
	if encoder.err != nil {
		return nil, encoder.err
	}
	return encoder.data, nil
}

func decodeRuntimeVolumeFinish(encoded []byte) (VolumeRecord, *runtimeVolumeError, error) {
	if len(encoded) == 0 || len(encoded) > maximumRuntimeNetworkWireBytes {
		return VolumeRecord{}, nil, errors.New("simulation runtime volume record is invalid")
	}
	decoder := newRuntimeNetworkWireDecoder(encoded, runtimeVolumeFinishMagic)
	record := decodeRuntimeVolumeRecord(decoder)
	var runtimeErr *runtimeVolumeError
	if decoder.boolean() {
		value := decodeRuntimeVolumeErrorBody(decoder)
		runtimeErr = &value
	}
	if err := decoder.finish(); err != nil {
		return VolumeRecord{}, nil, err
	}
	return record, runtimeErr, nil
}

func decodeRuntimeVolumeErrorWire(encoded []byte) (runtimeVolumeError, error) {
	if len(encoded) == 0 || len(encoded) > maximumRuntimeNetworkWireBytes {
		return runtimeVolumeError{}, errors.New("simulation runtime volume error is invalid")
	}
	decoder := newRuntimeNetworkWireDecoder(encoded, runtimeVolumeErrorMagic)
	runtimeErr := decodeRuntimeVolumeErrorBody(decoder)
	if err := decoder.finish(); err != nil {
		return runtimeVolumeError{}, err
	}
	if runtimeErr.Message == "" {
		return runtimeVolumeError{}, errors.New("simulation runtime volume error message is empty")
	}
	return runtimeErr, nil
}

func encodeRuntimeVolumeRecord(encoder *runtimeNetworkWireEncoder, record VolumeRecord) {
	encoder.uint64(uint64(len(record.Transitions)))
	for _, transition := range record.Transitions {
		encodeRuntimeVolumeTransition(encoder, transition)
	}
	encodeRuntimeVolumeSnapshot(encoder, record.Snapshot)
}

func decodeRuntimeVolumeRecord(decoder *runtimeNetworkWireDecoder) VolumeRecord {
	record := VolumeRecord{}
	for range decoder.count(MaximumVolumeTransitions) {
		record.Transitions = append(record.Transitions, decodeRuntimeVolumeTransition(decoder))
	}
	record.Snapshot = decodeRuntimeVolumeSnapshot(decoder)
	return record
}

func encodeRuntimeVolumeTransition(encoder *runtimeNetworkWireEncoder, transition VolumeTransition) {
	encoder.uint64(transition.Ordinal)
	encoder.string(string(transition.Kind))
	encoder.string(string(transition.Handle.Node))
	encoder.uint64(transition.Handle.Incarnation)
	encoder.string(string(transition.Volume))
	encoder.uint64(transition.Operation)
	encodeRuntimeVolumeUint64s(encoder, transition.Dependencies)
	encodeRuntimeVolumeUint64s(encoder, transition.SelectedOperations)
	encoder.string(transition.Path)
	encoder.string(transition.Destination)
	encoder.uint64(transition.Inode)
	encoder.uint64(transition.Offset)
	encoder.uint64(transition.Bytes)
	encoder.string(transition.PayloadSHA256)
	encoder.string(transition.EffectSHA256)
	encoder.string(string(transition.Outcome))
}

func decodeRuntimeVolumeTransition(decoder *runtimeNetworkWireDecoder) VolumeTransition {
	transition := VolumeTransition{Ordinal: decoder.uint64(), Kind: VolumeTransitionKind(decoder.string())}
	transition.Handle = NodeHandle{Node: NodeID(decoder.string()), Incarnation: decoder.uint64()}
	transition.Volume = VolumeID(decoder.string())
	transition.Operation = decoder.uint64()
	transition.Dependencies = decodeRuntimeVolumeUint64s(decoder, MaximumVolumeOperations)
	transition.SelectedOperations = decodeRuntimeVolumeUint64s(decoder, MaximumVolumeOperations)
	transition.Path = decoder.string()
	transition.Destination = decoder.string()
	transition.Inode = decoder.uint64()
	transition.Offset = decoder.uint64()
	transition.Bytes = decoder.uint64()
	transition.PayloadSHA256 = decoder.string()
	transition.EffectSHA256 = decoder.string()
	transition.Outcome = VolumeOutcome(decoder.string())
	return transition
}

func encodeRuntimeVolumeUint64s(encoder *runtimeNetworkWireEncoder, values []uint64) {
	encoder.uint64(uint64(len(values)))
	for _, value := range values {
		encoder.uint64(value)
	}
}

func decodeRuntimeVolumeUint64s(decoder *runtimeNetworkWireDecoder, maximum uint64) []uint64 {
	count := decoder.count(maximum)
	values := make([]uint64, 0, count)
	for range count {
		values = append(values, decoder.uint64())
	}
	return values
}

func encodeRuntimeVolumeSnapshot(encoder *runtimeNetworkWireEncoder, snapshot VolumeSnapshot) {
	encoder.uint64(uint64(len(snapshot.Volumes)))
	for _, volume := range snapshot.Volumes {
		encoder.string(string(volume.Volume))
		encoder.string(string(volume.Node))
		encoder.string(volume.Mount)
		encoder.uint64(volume.CapacityBytes)
		encodeRuntimeVolumeSnapshotEntries(encoder, volume.Persisted)
		encodeRuntimeVolumeSnapshotEntries(encoder, volume.Volatile)
		encoder.uint64(volume.PendingOperations)
		encoder.string(volume.PendingSHA256)
		encoder.uint64(volume.NextOperation)
		encoder.string(volume.Identity)
	}
	encoder.string(snapshot.TransitionSHA256)
	encoder.string(snapshot.Identity)
}

func decodeRuntimeVolumeSnapshot(decoder *runtimeNetworkWireDecoder) VolumeSnapshot {
	snapshot := VolumeSnapshot{}
	for range decoder.count(MaximumVolumes) {
		volume := VolumeStateSnapshot{
			Volume: VolumeID(decoder.string()), Node: NodeID(decoder.string()), Mount: decoder.string(), CapacityBytes: decoder.uint64(),
		}
		volume.Persisted = decodeRuntimeVolumeSnapshotEntries(decoder)
		volume.Volatile = decodeRuntimeVolumeSnapshotEntries(decoder)
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

func encodeRuntimeVolumeSnapshotEntries(encoder *runtimeNetworkWireEncoder, entries []VolumeEntrySnapshot) {
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

func decodeRuntimeVolumeSnapshotEntries(decoder *runtimeNetworkWireDecoder) []VolumeEntrySnapshot {
	count := decoder.count(100_000)
	entries := make([]VolumeEntrySnapshot, 0, count)
	for range count {
		entries = append(entries, VolumeEntrySnapshot{
			Path: decoder.string(), Mode: uint32(decoder.uint64()), Kind: decoder.string(), ModTime: int64(decoder.uint64()),
			Size: decoder.uint64(), DataSHA256: decoder.string(),
		})
	}
	return entries
}

func decodeRuntimeVolumeErrorBody(decoder *runtimeNetworkWireDecoder) runtimeVolumeError {
	runtimeErr := runtimeVolumeError{
		Kind: decoder.string(), Message: decoder.string(), Ordinal: decoder.uint64(),
		ExpectedSHA256: decoder.string(), ActualSHA256: decoder.string(),
	}
	if decoder.boolean() {
		value := decodeRuntimeVolumeTransition(decoder)
		runtimeErr.Expected = &value
	}
	if decoder.boolean() {
		value := decodeRuntimeVolumeTransition(decoder)
		runtimeErr.Actual = &value
	}
	return runtimeErr
}

func encodeRuntimeVolumeFrontier(frontier *VolumeCrashFrontier) ([]byte, error) {
	if frontier == nil {
		return nil, nil
	}
	encoder := &runtimeNetworkWireEncoder{data: []byte(runtimeVolumeFrontierMagic)}
	encodeRuntimeVolumeFrontierBody(encoder, *frontier)
	if encoder.err != nil {
		return nil, encoder.err
	}
	return encoder.data, nil
}

func encodeRuntimeVolumeFrontierBody(encoder *runtimeNetworkWireEncoder, frontier VolumeCrashFrontier) {
	encoder.string(string(frontier.Volume))
	encoder.string(frontier.PendingSHA256)
	encodeRuntimeVolumeBytes(encoder, frontier.Cursor, MaximumVolumeOperations)
	encoder.uint64(uint64(len(frontier.Seen)))
	for _, identity := range frontier.Seen {
		encoder.string(identity)
	}
	encoder.string(frontier.Identity)
}

func decodeRuntimeVolumeFrontierBody(decoder *runtimeNetworkWireDecoder) VolumeCrashFrontier {
	frontier := VolumeCrashFrontier{Volume: VolumeID(decoder.string()), PendingSHA256: decoder.string()}
	frontier.Cursor = decodeRuntimeVolumeBytes(decoder, MaximumVolumeOperations)
	for range decoder.count(MaximumCrashStates) {
		frontier.Seen = append(frontier.Seen, decoder.string())
	}
	frontier.Identity = decoder.string()
	return frontier
}

func decodeRuntimeVolumeEnumeration(encoded []byte) (VolumeCrashEnumeration, error) {
	if len(encoded) == 0 || len(encoded) > maximumRuntimeNetworkWireBytes {
		return VolumeCrashEnumeration{}, errors.New("simulation runtime volume enumeration is invalid")
	}
	decoder := newRuntimeNetworkWireDecoder(encoded, runtimeVolumeEnumerationMagic)
	page := VolumeCrashEnumeration{}
	for range decoder.count(MaximumCrashStates) {
		state := VolumeCrashState{Volume: VolumeID(decoder.string()), PendingSHA256: decoder.string()}
		state.SelectedOperations = decodeRuntimeVolumeUint64s(decoder, MaximumVolumeOperations)
		for range decoder.count(100_000) {
			state.Entries = append(state.Entries, VolumeCrashEntry{
				Path: decoder.string(), Mode: uint32(decoder.uint64()), Kind: volumeCrashKind(decoder.uint64()), ModTime: int64(decoder.uint64()),
				Data: decodeRuntimeVolumeBytes(decoder, MaximumVolumeCapacityBytes),
			})
		}
		state.Identity = decoder.string()
		page.States = append(page.States, state)
	}
	if decoder.boolean() {
		frontier := decodeRuntimeVolumeFrontierBody(decoder)
		page.Frontier = &frontier
	}
	page.Complete = decoder.boolean()
	page.Capacity = VolumeCrashCapacity(decoder.string())
	if err := decoder.finish(); err != nil {
		return VolumeCrashEnumeration{}, err
	}
	return page, nil
}

func volumeCrashKind(value uint64) string {
	switch value {
	case 1:
		return "file"
	case 2:
		return "directory"
	default:
		return ""
	}
}

func encodeRuntimeVolumeBytes(encoder *runtimeNetworkWireEncoder, value []byte, maximum uint64) {
	if encoder.err != nil {
		return
	}
	if uint64(len(value)) > maximum {
		encoder.err = errors.New("simulation runtime volume bytes exceed their limit")
		return
	}
	encoder.uint64(uint64(len(value)))
	encoder.data = append(encoder.data, value...)
	if len(encoder.data) > maximumRuntimeNetworkWireBytes {
		encoder.err = errors.New("simulation runtime volume wire value exceeds its byte limit")
	}
}

func decodeRuntimeVolumeBytes(decoder *runtimeNetworkWireDecoder, maximum uint64) []byte {
	length := decoder.uint64()
	if decoder.err != nil {
		return nil
	}
	if length > maximum || decoder.offset > uint64(len(decoder.data)) || length > uint64(len(decoder.data))-decoder.offset {
		decoder.err = errors.New("simulation runtime volume bytes are invalid")
		return nil
	}
	value := append([]byte(nil), decoder.data[decoder.offset:decoder.offset+length]...)
	decoder.offset += length
	return value
}

func encodeRuntimeVolumeTransitions(transitions []VolumeTransition) ([]byte, error) {
	encoder := &runtimeNetworkWireEncoder{}
	encoder.uint64(uint64(len(transitions)))
	for _, transition := range transitions {
		encodeRuntimeVolumeTransition(encoder, transition)
	}
	if encoder.err != nil {
		return nil, encoder.err
	}
	return encoder.data, nil
}
