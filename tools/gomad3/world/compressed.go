package world

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"hash"
	"io"
	"sort"
	"sync"
)

const CompressedSnapshotSchema = "gomad3/world/compressed-snapshot/v2"

const compressedEnvelopeHeaderBytes = 64

var compressedEnvelopeMagic = [8]byte{'G', 'M', 'W', 'O', 'R', 'L', 'D', '2'}

type ProfileLimits struct {
	MaxExpandedBytes   uint64
	MaxCompressedBytes uint64
	Limits             Limits
}

func CompressedSnapshotProfile() ProfileLimits {
	return ProfileLimits{
		MaxExpandedBytes: 384 << 20, MaxCompressedBytes: 64 << 20,
		Limits: Limits{MaxRequests: 250000, MaxEvents: 250000, MaxQueuedEvents: 250000,
			MaxTransitions: 750000, MaxPayloadBytes: 64 << 20, MaxStringBytes: 1024},
	}
}

type CompressedSnapshotSummary struct {
	Config           Config
	Now              LogicalTime
	RequestCount     uint64
	EventCount       uint64
	TransitionCount  uint64
	PayloadBytes     uint64
	TranscriptDigest Digest
	StateDigest      Digest
	Replay           ReplayProgress
	ExpandedBytes    uint64
	CompressedBytes  uint64
	ExpandedSHA256   Digest
}

type compressedSnapshotEnvelope struct {
	ExpandedBytes   uint64
	CompressedBytes uint64
	ExpandedSHA256  Digest
	Gzip            []byte
}

// DetachedSnapshot copies share the consumption state.
type DetachedSnapshot struct {
	state *detachedSnapshotState
}

type detachedSnapshotState struct {
	mu       sync.Mutex
	model    *Model
	limits   ProfileLimits
	consumed bool
}

// ValidateCompressedSnapshotConfiguration checks the closed profile before a
// producer starts. It neither changes configured capacities nor creates state.
func ValidateCompressedSnapshotConfiguration(config Config, limits ProfileLimits) error {
	if err := validateCompressedProfile(limits); err != nil {
		return err
	}
	return validateCompressedConfig(config, limits)
}

func (w *Model) DetachCompressedSnapshot(limits ProfileLimits) (*DetachedSnapshot, error) {
	if err := validateCompressedProfile(limits); err != nil {
		return nil, err
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.detached {
		return nil, invalidSnapshot("model.detached")
	}
	if w.recording != nil {
		return nil, invalidSnapshot("model.recording")
	}
	if err := validateCompressedConfig(w.config, limits); err != nil {
		return nil, err
	}
	owned := &Model{
		config: w.config, now: w.now, nextRequestID: w.nextRequestID, nextEventID: w.nextEventID,
		nextTransition: w.nextTransition, payloadBytes: w.payloadBytes, requests: w.requests,
		events: w.events, queue: w.queue, history: w.history, transcript: w.transcript, replay: w.replay,
	}
	w.detached = true
	w.detachedCardinality = DiagnosticCardinality{
		Now: w.now, Requests: uint64(len(w.requests)), Events: uint64(len(w.events)), QueuedEvents: uint64(len(w.queue)),
		Transitions: uint64(len(w.history)), PayloadBytes: w.payloadBytes, TranscriptDigest: w.transcript,
	}
	w.requests, w.events, w.queue, w.history, w.replay = nil, nil, nil, nil, nil
	w.replayPayloadBytes = 0
	return &DetachedSnapshot{state: &detachedSnapshotState{model: owned, limits: limits}}, nil
}

func (detached *DetachedSnapshot) Encode() ([]byte, error) {
	if detached == nil || detached.state == nil {
		return nil, invalidSnapshot("detached.invalid")
	}
	state := detached.state
	state.mu.Lock()
	if state.consumed || state.model == nil {
		state.mu.Unlock()
		return nil, invalidSnapshot("detached.consumed")
	}
	state.consumed = true
	model, limits := state.model, state.limits
	state.model = nil
	state.mu.Unlock()
	snapshot := ownedSnapshot(model)
	model = nil
	return encodeCompressedSnapshot(snapshot, limits)
}

func encodeCompressedSnapshot(snapshot Snapshot, limits ProfileLimits) ([]byte, error) {
	if err := validateCompressedProfile(limits); err != nil {
		return nil, err
	}
	if err := validateCompressedConfig(snapshot.Config, limits); err != nil {
		return nil, err
	}
	if err := validateSnapshotStrings(snapshot); err != nil {
		return nil, err
	}
	var compressed bytes.Buffer
	compressedOutput := &compressedBoundedWriter{writer: &compressed, limit: limits.MaxCompressedBytes, dimension: "compressed-snapshot-gzip-bytes"}
	//gomad:dynamic reviewed 2026-10-12 reviewed model or record values and in-memory streams constrain callback methods
	compressor := gzip.NewWriter(compressedOutput)
	hasher := sha256.New()
	//gomad:dynamic reviewed 2026-10-12 reviewed model or record values and in-memory streams constrain callback methods
	expandedOutput := &compressedBoundedWriter{writer: io.MultiWriter(compressor, hasher), limit: limits.MaxExpandedBytes, dimension: "compressed-snapshot-expanded-bytes"}
	if err := writeCompressedSnapshotWire(expandedOutput, snapshot); err != nil {
		closeErr := compressor.Close()
		if closeErr != nil {
			//gomad:dynamic reviewed 2026-10-12 model codec and gzip errors expose memory-only messages
			return nil, modelContextError("compressed.encode (close: "+closeErr.Error()+")", err)
		}
		return nil, modelContextError("compressed.encode", err)
	}
	if err := compressor.Close(); err != nil {
		return nil, modelContextError("compressed.close", err)
	}
	encoded := make([]byte, compressedEnvelopeHeaderBytes+compressed.Len())
	copy(encoded[:8], compressedEnvelopeMagic[:])
	binary.BigEndian.PutUint32(encoded[8:12], compressedEnvelopeHeaderBytes)
	// Bytes 12:16 are reserved flags and must remain zero.
	binary.BigEndian.PutUint64(encoded[16:24], expandedOutput.used)
	binary.BigEndian.PutUint64(encoded[24:32], compressedOutput.used)
	//gomad:dynamic reviewed 2026-10-12 receiver is the locally constructed SHA256 or HMAC memory hash
	copy(encoded[32:64], hasher.Sum(nil))
	copy(encoded[compressedEnvelopeHeaderBytes:], compressed.Bytes())
	return encoded, nil
}

// ownedSnapshot borrows payloads only from a model that no caller can mutate.
func ownedSnapshot(w *Model) Snapshot {
	snapshot := Snapshot{
		SchemaVersion: SchemaVersion, Config: w.config, Now: w.now, NextRequestID: w.nextRequestID,
		NextEventID: w.nextEventID, NextTransition: w.nextTransition, PayloadBytes: w.payloadBytes,
		TranscriptDigest: w.transcript, Transitions: w.history,
		Requests: make([]RequestSnapshot, 0, len(w.requests)), Events: make([]EventSnapshot, 0, len(w.events)),
	}
	if w.replay != nil {
		snapshot.Replay = ReplayProgress{Cursor: w.replay.cursor, Expected: uint64(len(w.replay.plan.Transitions))}
	}
	if snapshot.Transitions == nil {
		snapshot.Transitions = []Transition{}
	}
	for _, request := range w.requests {
		snapshot.Requests = append(snapshot.Requests, RequestSnapshot{ID: request.id, Request: request.request, State: request.state, EventID: request.eventID})
	}
	for _, event := range w.events {
		snapshot.Events = append(snapshot.Events, EventSnapshot{ID: event.id, Readiness: event.readiness, State: event.state})
	}
	sort.Slice(snapshot.Requests, func(i, j int) bool { return snapshot.Requests[i].ID < snapshot.Requests[j].ID })
	sort.Slice(snapshot.Events, func(i, j int) bool { return snapshot.Events[i].ID < snapshot.Events[j].ID })
	snapshot.StateDigest = stateDigest(snapshot)
	return snapshot
}

func validateCompressedProfile(limits ProfileLimits) error {
	maximum := CompressedSnapshotProfile()
	if limits.MaxExpandedBytes == 0 || limits.MaxExpandedBytes > maximum.MaxExpandedBytes ||
		limits.MaxCompressedBytes == 0 || limits.MaxCompressedBytes > maximum.MaxCompressedBytes {
		return invalidSnapshot("compressed.profile.bytes")
	}
	return validateCompressedConfig(Config{Limits: limits.Limits}, maximum)
}

func validateCompressedConfig(config Config, limits ProfileLimits) error {
	if err := validateConfig(config); err != nil {
		return invalidSnapshot("compressed.config")
	}
	actual, maximum := config.Limits, limits.Limits
	for _, capacity := range []struct {
		dimension     string
		actual, limit uint64
	}{
		{"requests", actual.MaxRequests, maximum.MaxRequests},
		{"events", actual.MaxEvents, maximum.MaxEvents},
		{"queued-events", actual.MaxQueuedEvents, maximum.MaxQueuedEvents},
		{"transitions", actual.MaxTransitions, maximum.MaxTransitions},
		{"payload-bytes", actual.MaxPayloadBytes, maximum.MaxPayloadBytes},
		{"string-bytes", uint64(actual.MaxStringBytes), uint64(maximum.MaxStringBytes)},
	} {
		if capacity.actual > capacity.limit {
			return &CapacityError{Dimension: "compressed-snapshot-config-" + capacity.dimension, Limit: capacity.limit, Delta: capacity.actual}
		}
	}
	return nil
}

type compressedBoundedWriter struct {
	writer    io.Writer
	limit     uint64
	used      uint64
	dimension string
}

func (writer *compressedBoundedWriter) Write(data []byte) (int, error) {
	if uint64(len(data)) > writer.limit-writer.used {
		return 0, &CapacityError{Dimension: writer.dimension, Limit: writer.limit, Used: writer.used, Delta: uint64(len(data))}
	}
	//gomad:dynamic reviewed 2026-10-12 compressed model stream wraps memory readers writers and SHA256 only
	written, err := writer.writer.Write(data)
	writer.used += uint64(written)
	return written, err
}

func compressedSnapshotWireDigest(snapshot Snapshot, limit uint64) (Digest, uint64, error) {
	hasher := sha256.New()
	writer := &compressedBoundedWriter{writer: hasher, limit: limit, dimension: "compressed-snapshot-wire-bytes"}
	if err := writeCompressedSnapshotWire(writer, snapshot); err != nil {
		return "", 0, err
	}
	//gomad:dynamic reviewed 2026-10-12 receiver is the locally constructed SHA256 or HMAC memory hash
	return Digest(hex.EncodeToString(hasher.Sum(nil))), writer.used, nil
}

func compressedGzipPass(envelope compressedSnapshotEnvelope, read func(io.Reader) error) error {
	source := bytes.NewReader(envelope.Gzip)
	reader, err := gzip.NewReader(source)
	if err != nil {
		//gomad:dynamic reviewed 2026-10-12 model codec and gzip errors expose memory-only messages
		return invalidSnapshot("compressed.gzip: " + err.Error())
	}
	reader.Multistream(false)
	hasher := sha256.New()
	bounded := &compressedHashReader{reader: io.LimitReader(reader, int64(envelope.ExpandedBytes)+1), hash: hasher, limit: envelope.ExpandedBytes}
	//gomad:dynamic reviewed 2026-10-12 local model closure or reviewed caller-supplied callback performs memory-only work
	readErr := read(bounded)
	closeErr := reader.Close()
	if readErr != nil {
		//gomad:dynamic reviewed 2026-10-12 model codec and gzip errors expose memory-only messages
		return invalidSnapshot("compressed.expanded: " + readErr.Error())
	}
	if closeErr != nil {
		//gomad:dynamic reviewed 2026-10-12 model codec and gzip errors expose memory-only messages
		return invalidSnapshot("compressed.gzip.close: " + closeErr.Error())
	}
	//gomad:dynamic reviewed 2026-10-12 receiver is the locally constructed SHA256 or HMAC memory hash
	if source.Len() != 0 || bounded.used != envelope.ExpandedBytes || Digest(hex.EncodeToString(hasher.Sum(nil))) != envelope.ExpandedSHA256 {
		return invalidSnapshot("compressed.expanded.identity")
	}
	return nil
}

type compressedHashReader struct {
	reader io.Reader
	hash   hash.Hash
	limit  uint64
	used   uint64
}

func (reader *compressedHashReader) Read(data []byte) (int, error) {
	//gomad:dynamic reviewed 2026-10-12 compressed model stream wraps memory readers writers and SHA256 only
	n, err := reader.reader.Read(data)
	if uint64(n) > reader.limit-reader.used {
		return 0, fmt.Errorf("expanded bytes exceed declared bound")
	}
	reader.used += uint64(n)
	//gomad:dynamic reviewed 2026-10-12 compressed model stream wraps memory readers writers and SHA256 only
	if _, hashErr := reader.hash.Write(data[:n]); hashErr != nil {
		return 0, hashErr
	}
	return n, err
}
