package simulation

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand/v2"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomad/internal/model/world"
)

// The World conformance corpus pins the Go World owner, which stays
// authoritative for the native backend and the runner, against the Rust
// duplicate in the cluster helper (fn-178.3). Inputs are concrete World calls;
// outputs are what Go returned, so `cargo test` replays them without Go.
const (
	worldConformanceDirectory = "../../../../rust/wasmhost/tests/testdata/world_conformance"
	worldConformancePin       = worldConformanceDirectory + "/PINNED_GO_WORLD_SHA256"
	worldConformanceSchema    = "gomad.world-conformance/v1"
	// Steps at this interval, and the last step, also carry the full state digest
	// and identity counters; other steps carry the cheap cardinality summary.
	worldConformanceCheckpoint = 64
)

type worldConformanceCorpus struct {
	Schema               string                    `json:"schema"`
	Name                 string                    `json:"name"`
	Source               string                    `json:"source"`
	Origin               string                    `json:"origin"`
	ImplementationSHA256 string                    `json:"world_implementation_sha256"`
	Config               worldConformanceConfig    `json:"config"`
	CreationError        *worldConformanceError    `json:"creation_error"`
	Steps                []worldConformanceStep    `json:"steps"`
	Final                *worldConformanceSnapshot `json:"final"`
}

type worldConformanceConfig struct {
	Seed            uint64 `json:"seed"`
	MaxRequests     uint64 `json:"max_requests"`
	MaxEvents       uint64 `json:"max_events"`
	MaxQueuedEvents uint64 `json:"max_queued_events"`
	MaxTransitions  uint64 `json:"max_transitions"`
	MaxPayloadBytes uint64 `json:"max_payload_bytes"`
	MaxStringBytes  uint32 `json:"max_string_bytes"`
}

type worldConformanceRequest struct {
	Kind          string `json:"kind"`
	Adapter       string `json:"adapter"`
	ResourceKind  string `json:"resource_kind"`
	Key           string `json:"key"`
	Priority      uint16 `json:"priority"`
	PayloadBase64 string `json:"payload_base64"`
}

type worldConformanceReadiness struct {
	RequestID        uint64 `json:"request_id"`
	At               int64  `json:"at"`
	Kind             string `json:"kind"`
	PayloadBase64    string `json:"payload_base64"`
	EquivalenceClass string `json:"equivalence_class"`
}

type worldConformanceStep struct {
	Op        string                     `json:"op"`
	Request   *worldConformanceRequest   `json:"request,omitempty"`
	Readiness *worldConformanceReadiness `json:"readiness,omitempty"`
	RequestID uint64                     `json:"request_id,omitempty"`
	Expected  worldConformanceOutput     `json:"expected"`
}

type worldConformanceError struct {
	Class   string `json:"class"`
	Message string `json:"message"`
}

type worldConformanceCancellation struct {
	RequestID uint64 `json:"request_id"`
	EventID   uint64 `json:"event_id"`
	Status    string `json:"status"`
}

type worldConformanceDelivery struct {
	RequestID     uint64 `json:"request_id"`
	EventID       uint64 `json:"event_id"`
	At            int64  `json:"at"`
	Kind          string `json:"kind"`
	PayloadBase64 string `json:"payload_base64"`
}

type worldConformanceQuiescence struct {
	Kind       string                     `json:"kind"`
	Before     int64                      `json:"before"`
	After      int64                      `json:"after"`
	Deliveries []worldConformanceDelivery `json:"deliveries"`
	Blocked    []uint64                   `json:"blocked"`
}

type worldConformanceState struct {
	Now              int64  `json:"now"`
	Requests         uint64 `json:"requests"`
	Events           uint64 `json:"events"`
	QueuedEvents     uint64 `json:"queued_events"`
	Transitions      uint64 `json:"transitions"`
	PayloadBytes     uint64 `json:"payload_bytes"`
	TranscriptDigest string `json:"transcript_digest"`
	// Checkpoint fields; empty or zero elsewhere.
	NextRequestID  uint64 `json:"next_request_id,omitempty"`
	NextEventID    uint64 `json:"next_event_id,omitempty"`
	NextTransition uint64 `json:"next_transition,omitempty"`
	StateDigest    string `json:"state_digest,omitempty"`
}

type worldConformanceOutput struct {
	Error        *worldConformanceError        `json:"error"`
	RequestID    uint64                        `json:"request_id,omitempty"`
	EventID      uint64                        `json:"event_id,omitempty"`
	Cancellation *worldConformanceCancellation `json:"cancellation,omitempty"`
	Quiescence   *worldConformanceQuiescence   `json:"quiescence,omitempty"`
	State        worldConformanceState         `json:"state"`
}

// The final expanded GMWSNAP1 wire is what the Rust generated writer must
// reproduce byte for byte (fn-178.4); the envelope digest is Go's GMWORLD2.
type worldConformanceSnapshot struct {
	ExpandedBase64 string `json:"expanded_base64"`
	ExpandedSHA256 string `json:"expanded_sha256"`
	EnvelopeSHA256 string `json:"envelope_sha256"`
}

func (config worldConformanceConfig) world() world.Config {
	return world.Config{Seed: world.Seed(config.Seed), Limits: world.Limits{
		MaxRequests: config.MaxRequests, MaxEvents: config.MaxEvents, MaxQueuedEvents: config.MaxQueuedEvents,
		MaxTransitions: config.MaxTransitions, MaxPayloadBytes: config.MaxPayloadBytes, MaxStringBytes: config.MaxStringBytes,
	}}
}

func worldConformanceConfigOf(config world.Config) worldConformanceConfig {
	limits := config.Limits
	return worldConformanceConfig{Seed: uint64(config.Seed), MaxRequests: limits.MaxRequests, MaxEvents: limits.MaxEvents, MaxQueuedEvents: limits.MaxQueuedEvents, MaxTransitions: limits.MaxTransitions, MaxPayloadBytes: limits.MaxPayloadBytes, MaxStringBytes: limits.MaxStringBytes}
}

func worldConformanceErrorOf(t *testing.T, err error) *worldConformanceError {
	t.Helper()
	if err == nil {
		return nil
	}
	for _, class := range []struct {
		name     string
		sentinel error
	}{
		{"capacity", world.ErrCapacity}, {"invalid-config", world.ErrInvalidConfig}, {"invalid-request", world.ErrInvalidRequest},
		{"unknown-request", world.ErrUnknownRequest}, {"request-state", world.ErrRequestState}, {"time-regression", world.ErrTimeRegression},
		{"invalid-snapshot", world.ErrInvalidSnapshot}, {"replay-divergence", world.ErrReplayDivergence},
	} {
		if errors.Is(err, class.sentinel) {
			return &worldConformanceError{Class: class.name, Message: err.Error()}
		}
	}
	t.Fatalf("unclassified World error: %v", err)
	return nil
}

func base64Std(data []byte) string { return base64.StdEncoding.EncodeToString(data) }

func mustBase64Std(t *testing.T, text string) []byte {
	t.Helper()
	data, err := base64.StdEncoding.DecodeString(text)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

type worldConformanceBuilder struct {
	t      *testing.T
	model  *world.Model
	corpus worldConformanceCorpus
}

func newWorldConformance(t *testing.T, name, source, origin string, config world.Config) *worldConformanceBuilder {
	t.Helper()
	b := &worldConformanceBuilder{t: t, corpus: worldConformanceCorpus{
		Schema: worldConformanceSchema, Name: name, Source: source, Origin: origin,
		ImplementationSHA256: world.ImplementationSHA256(), Config: worldConformanceConfigOf(config), Steps: []worldConformanceStep{},
	}}
	model, err := world.New(config)
	b.corpus.CreationError = worldConformanceErrorOf(t, err)
	b.model = model
	return b
}

func (b *worldConformanceBuilder) state(checkpoint bool) worldConformanceState {
	cardinality := b.model.DiagnosticCardinality()
	state := worldConformanceState{
		Now: int64(cardinality.Now), Requests: cardinality.Requests, Events: cardinality.Events, QueuedEvents: cardinality.QueuedEvents,
		Transitions: cardinality.Transitions, PayloadBytes: cardinality.PayloadBytes, TranscriptDigest: string(cardinality.TranscriptDigest),
	}
	if checkpoint {
		snapshot := b.model.Snapshot()
		state.NextRequestID, state.NextEventID, state.NextTransition = uint64(snapshot.NextRequestID), uint64(snapshot.NextEventID), uint64(snapshot.NextTransition)
		state.StateDigest = string(snapshot.StateDigest)
	}
	return state
}

func (b *worldConformanceBuilder) checkpoint() bool {
	return len(b.corpus.Steps)%worldConformanceCheckpoint == 0
}

func (b *worldConformanceBuilder) register(request world.Request) (world.RequestID, error) {
	b.t.Helper()
	checkpoint := b.checkpoint()
	id, err := b.model.Register(request)
	b.corpus.Steps = append(b.corpus.Steps, worldConformanceStep{Op: "register", Request: &worldConformanceRequest{
		Kind: request.Kind, Adapter: request.Resource.Adapter, ResourceKind: request.Resource.Kind, Key: request.Resource.Key,
		Priority: uint16(request.Priority), PayloadBase64: base64Std(request.Payload),
	}, Expected: worldConformanceOutput{Error: worldConformanceErrorOf(b.t, err), RequestID: uint64(id), State: b.state(checkpoint)}})
	return id, err
}

func (b *worldConformanceBuilder) ready(readiness world.Readiness) (world.EventID, error) {
	b.t.Helper()
	checkpoint := b.checkpoint()
	id, err := b.model.Ready(readiness)
	b.corpus.Steps = append(b.corpus.Steps, worldConformanceStep{Op: "ready", Readiness: &worldConformanceReadiness{
		RequestID: uint64(readiness.RequestID), At: int64(readiness.At), Kind: readiness.Kind,
		PayloadBase64: base64Std(readiness.Payload), EquivalenceClass: readiness.EquivalenceClass,
	}, Expected: worldConformanceOutput{Error: worldConformanceErrorOf(b.t, err), EventID: uint64(id), State: b.state(checkpoint)}})
	return id, err
}

func (b *worldConformanceBuilder) cancel(id world.RequestID) (world.Cancellation, error) {
	b.t.Helper()
	checkpoint := b.checkpoint()
	result, err := b.model.Cancel(id)
	output := worldConformanceOutput{Error: worldConformanceErrorOf(b.t, err)}
	if err == nil {
		output.Cancellation = &worldConformanceCancellation{RequestID: uint64(result.RequestID), EventID: uint64(result.EventID), Status: string(result.Status)}
	}
	output.State = b.state(checkpoint)
	b.corpus.Steps = append(b.corpus.Steps, worldConformanceStep{Op: "cancel", RequestID: uint64(id), Expected: output})
	return result, err
}

func (b *worldConformanceBuilder) quiesce() (world.Quiescence, error) {
	b.t.Helper()
	checkpoint := b.checkpoint()
	result, err := b.model.Quiesce()
	output := worldConformanceOutput{Error: worldConformanceErrorOf(b.t, err)}
	if err == nil {
		q := &worldConformanceQuiescence{Kind: string(result.Kind), Before: int64(result.Before), After: int64(result.After), Deliveries: []worldConformanceDelivery{}, Blocked: []uint64{}}
		for _, delivery := range result.Deliveries {
			q.Deliveries = append(q.Deliveries, worldConformanceDelivery{RequestID: uint64(delivery.RequestID), EventID: uint64(delivery.EventID), At: int64(delivery.At), Kind: delivery.Kind, PayloadBase64: base64Std(delivery.Payload)})
		}
		for _, blocked := range result.Blocked {
			q.Blocked = append(q.Blocked, uint64(blocked))
		}
		output.Quiescence = q
	}
	output.State = b.state(checkpoint)
	b.corpus.Steps = append(b.corpus.Steps, worldConformanceStep{Op: "quiesce", Expected: output})
	return result, err
}

// finish makes the last step a checkpoint and, when the configuration fits the
// compressed profile, records Go's expanded wire and envelope.
func (b *worldConformanceBuilder) finish() worldConformanceCorpus {
	b.t.Helper()
	if b.model == nil {
		return b.corpus
	}
	if last := len(b.corpus.Steps) - 1; last >= 0 && b.corpus.Steps[last].Expected.State.StateDigest == "" {
		b.corpus.Steps[last].Expected.State = b.state(true)
	}
	if world.ValidateCompressedSnapshotConfiguration(b.corpus.Config.world(), world.CompressedSnapshotProfile()) != nil {
		return b.corpus
	}
	detached, err := b.model.DetachCompressedSnapshot(world.CompressedSnapshotProfile())
	if err != nil {
		b.t.Fatal(err)
	}
	envelope, err := detached.Encode()
	if err != nil {
		b.t.Fatal(err)
	}
	expanded := expandWorldEnvelope(b.t, envelope)
	expandedSum, envelopeSum := sha256.Sum256(expanded), sha256.Sum256(envelope)
	b.corpus.Final = &worldConformanceSnapshot{ExpandedBase64: base64Std(expanded), ExpandedSHA256: hex.EncodeToString(expandedSum[:]), EnvelopeSHA256: hex.EncodeToString(envelopeSum[:])}
	b.model = nil
	return b.corpus
}

func expandWorldEnvelope(t *testing.T, envelope []byte) []byte {
	t.Helper()
	if len(envelope) < 64 || string(envelope[:8]) != "GMWORLD2" {
		t.Fatal("World envelope header")
	}
	reader, err := gzip.NewReader(bytes.NewReader(envelope[64:]))
	if err != nil {
		t.Fatal(err)
	}
	expanded, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(expanded)
	if !bytes.Equal(sum[:], envelope[32:64]) {
		t.Fatal("World envelope expanded digest")
	}
	return expanded
}

func worldConformanceTestLimits() world.Limits {
	return world.Limits{MaxRequests: 100, MaxEvents: 100, MaxQueuedEvents: 100, MaxTransitions: 500, MaxPayloadBytes: 1 << 20, MaxStringBytes: 1024}
}

func worldConformanceScripted(t *testing.T) []worldConformanceCorpus {
	var corpora []worldConformanceCorpus
	add := func(name, origin string, config world.Config, run func(*worldConformanceBuilder)) {
		b := newWorldConformance(t, name, "scripted", origin, config)
		if b.model != nil {
			run(b)
		}
		corpora = append(corpora, b.finish())
	}
	limits := worldConformanceTestLimits()
	cell := func(key string) world.ResourceID { return world.ResourceID{Adapter: "memory", Kind: "cell", Key: key} }
	add("cancel-later-instant-retains-payload-accounting", "src/world/tests.rs world_quiesce_delivers_one_instant_in_order_and_cancel_removes_queued", world.Config{Seed: 7, Limits: limits}, func(b *worldConformanceBuilder) {
		for index, key := range []string{"b", "a", "c"} {
			id, _ := b.register(world.Request{Kind: "wait", Resource: cell(key)})
			at := world.InitialTime + 10
			if index == 2 {
				at++
			}
			_, _ = b.ready(world.Readiness{RequestID: id, At: at, Kind: "done", Payload: []byte{1}})
		}
		_, _ = b.cancel(3)
		_, _ = b.quiesce()
		_, _ = b.quiesce()
	})
	for _, seed := range []uint64{41, ^uint64(0)} {
		add(fmt.Sprintf("unicode-equivalence-class-seed-%d", seed), "src/world/tests.rs world_choice_rank_matches_go_vectors UTF-8 class framing", world.Config{Seed: world.Seed(seed), Limits: limits}, func(b *worldConformanceBuilder) {
			for range 8 {
				id, _ := b.register(world.Request{Kind: "receive", Resource: cell("queue")})
				_, _ = b.ready(world.Readiness{RequestID: id, At: world.InitialTime, Kind: "message", EquivalenceClass: "üñî"})
			}
			_, _ = b.quiesce()
		})
	}
	add("lifecycle-whole-instant", "world_test.go TestWorldLifecycleCopiesDataAndDeliversWholeInstant", world.Config{Seed: 7, Limits: limits}, func(b *worldConformanceBuilder) {
		first, _ := b.register(world.Request{Kind: "read", Resource: cell("a"), Priority: 2, Payload: []byte("request")})
		second, _ := b.register(world.Request{Kind: "read", Resource: cell("b"), Priority: 1})
		_, _ = b.ready(world.Readiness{RequestID: first, At: world.InitialTime + 10, Kind: "result", Payload: []byte("one")})
		_, _ = b.ready(world.Readiness{RequestID: second, At: world.InitialTime + 10, Kind: "result", Payload: []byte("two")})
		_, _ = b.quiesce()
	})
	for _, seed := range []uint64{0, 1, 99, ^uint64(0)} {
		add(fmt.Sprintf("ordinary-order-seed-%d", seed), "world_test.go TestWorldOrdinaryOrderingIsLexicographicAndSeedIndependent", world.Config{Seed: world.Seed(seed), Limits: limits}, func(b *worldConformanceBuilder) {
			for _, request := range []world.Request{
				{Kind: "z", Resource: world.ResourceID{Adapter: "z", Kind: "r", Key: "k"}, Priority: 1},
				{Kind: "a", Resource: world.ResourceID{Adapter: "a", Kind: "r", Key: "k"}, Priority: 1},
				{Kind: "a", Resource: world.ResourceID{Adapter: "a", Kind: "r", Key: "k"}, Priority: 0},
			} {
				id, _ := b.register(request)
				_, _ = b.ready(world.Readiness{RequestID: id, At: world.InitialTime, Kind: "done"})
			}
			_, _ = b.quiesce()
		})
	}
	for seed := uint64(0); seed < 16; seed++ {
		add(fmt.Sprintf("equivalent-order-seed-%02d", seed), "world_test.go TestWorldEquivalentOrderingRepeatsAndCanVaryBySeed", world.Config{Seed: world.Seed(seed), Limits: limits}, func(b *worldConformanceBuilder) {
			for range 4 {
				id, _ := b.register(world.Request{Kind: "receive", Resource: world.ResourceID{Adapter: "network", Kind: "queue", Key: "q"}})
				_, _ = b.ready(world.Readiness{RequestID: id, At: world.InitialTime, Kind: "message", EquivalenceClass: "exchangeable"})
			}
			_, _ = b.quiesce()
		})
	}
	add("cancellation-idle-deadlock", "world_test.go TestWorldCancellationIdleAndDeadlock", world.Config{Seed: 1, Limits: limits}, func(b *worldConformanceBuilder) {
		id, _ := b.register(world.Request{Kind: "wait", Resource: cell("a")})
		_, _ = b.quiesce()
		_, _ = b.cancel(id)
		_, _ = b.cancel(id)
		_, _ = b.quiesce()
		delivered, _ := b.register(world.Request{Kind: "wait", Resource: cell("b")})
		_, _ = b.ready(world.Readiness{RequestID: delivered, At: world.InitialTime, Kind: "done"})
		_, _ = b.quiesce()
		_, _ = b.cancel(delivered)
		_, _ = b.cancel(0)
		_, _ = b.cancel(999)
	})
	small := limits
	small.MaxRequests = 1
	add("invalid-and-capacity-no-mutation", "world_test.go TestWorldInvalidAndCapacityErrorsDoNotMutateState", world.Config{Seed: 1, Limits: small}, func(b *worldConformanceBuilder) {
		_, _ = b.register(world.Request{Kind: "", Resource: world.ResourceID{Adapter: "Bad", Kind: "cell", Key: "a"}})
		id, _ := b.register(world.Request{Kind: "wait", Resource: cell("a")})
		_, _ = b.register(world.Request{Kind: "wait", Resource: cell("b")})
		_, _ = b.ready(world.Readiness{RequestID: id, At: world.InitialTime - 1, Kind: "done"})
		_, _ = b.ready(world.Readiness{RequestID: id, At: world.InitialTime, Kind: "done"})
		_, _ = b.ready(world.Readiness{RequestID: id, At: world.InitialTime, Kind: "done"})
		_, _ = b.ready(world.Readiness{RequestID: 0, At: world.InitialTime, Kind: "done"})
	})
	queued := limits
	queued.MaxQueuedEvents = 1
	add("queue-capacity-rollback", "capacity_test.go TestWorldQueueCapacityRollbackPreservesEventSequence", world.Config{Seed: 1, Limits: queued}, func(b *worldConformanceBuilder) {
		first, _ := b.register(world.Request{Kind: "wait", Resource: cell("a")})
		second, _ := b.register(world.Request{Kind: "wait", Resource: cell("b")})
		_, _ = b.ready(world.Readiness{RequestID: first, At: world.InitialTime, Kind: "done"})
		_, _ = b.ready(world.Readiness{RequestID: second, At: world.InitialTime, Kind: "done"})
		_, _ = b.cancel(first)
		_, _ = b.ready(world.Readiness{RequestID: second, At: world.InitialTime, Kind: "done"})
		_, _ = b.quiesce()
	})
	transitions := limits
	transitions.MaxTransitions = 1
	add("transition-capacity-terminal", "capacity_test.go TestWorldTransitionCapacityIsTerminalWithoutMutation", world.Config{Seed: 1, Limits: transitions}, func(b *worldConformanceBuilder) {
		id, _ := b.register(world.Request{Kind: "wait", Resource: cell("a")})
		_, _ = b.cancel(id)
		_, _ = b.quiesce()
	})
	payload := limits
	payload.MaxPayloadBytes = 5
	add("payload-capacity-transcript-copies", "capacity_test.go TestWorldPayloadCapacityIncludesTranscriptCopies", world.Config{Seed: 1, Limits: payload}, func(b *worldConformanceBuilder) {
		_, _ = b.register(world.Request{Kind: "wait", Resource: cell("a"), Payload: []byte("abc")})
		id, _ := b.register(world.Request{Kind: "wait", Resource: cell("a"), Payload: []byte("ab")})
		_, _ = b.ready(world.Readiness{RequestID: id, At: world.InitialTime, Kind: "done", Payload: []byte("x")})
		_, _ = b.quiesce()
	})
	for _, field := range []string{"requests", "events", "queued", "transitions", "payload", "string", "queued-over-events"} {
		zero := limits
		switch field {
		case "requests":
			zero.MaxRequests = 0
		case "events":
			zero.MaxEvents = 0
		case "queued":
			zero.MaxQueuedEvents = 0
		case "transitions":
			zero.MaxTransitions = 0
		case "payload":
			zero.MaxPayloadBytes = 0
		case "string":
			zero.MaxStringBytes = 0
		case "queued-over-events":
			zero.MaxQueuedEvents = zero.MaxEvents + 1
		}
		add("invalid-config-"+field, "capacity_test.go TestWorldRejectsEveryZeroLimit", world.Config{Limits: zero}, func(*worldConformanceBuilder) {})
	}
	strs := limits
	strs.MaxStringBytes = 3
	add("string-limit-resource-components", "capacity_test.go TestWorldStringLimitCoversResourceComponents", world.Config{Limits: strs}, func(b *worldConformanceBuilder) {
		_, _ = b.register(world.Request{Kind: "get", Resource: world.ResourceID{Adapter: "memory", Kind: "key", Key: "a"}})
		_, _ = b.register(world.Request{Kind: "get", Resource: world.ResourceID{Adapter: "mem", Kind: "key", Key: "abcd"}})
		id, _ := b.register(world.Request{Kind: "get", Resource: world.ResourceID{Adapter: "mem", Kind: "key", Key: "abc"}})
		_, _ = b.ready(world.Readiness{RequestID: id, At: world.InitialTime, Kind: "long"})
		_, _ = b.ready(world.Readiness{RequestID: id, At: world.InitialTime, Kind: "ok", EquivalenceClass: "four"})
		_, _ = b.ready(world.Readiness{RequestID: id, At: world.InitialTime, Kind: "ok", EquivalenceClass: "eq"})
		_, _ = b.quiesce()
	})
	add("replay-shaped-history", "replay_test.go TestWorldReplayValidatesEveryTransitionAndFinalDigest", world.Config{Seed: 41, Limits: limits}, func(b *worldConformanceBuilder) {
		id, _ := b.register(world.Request{Kind: "receive", Resource: world.ResourceID{Adapter: "network", Kind: "queue", Key: "inbox"}, Payload: []byte("request")})
		_, _ = b.ready(world.Readiness{RequestID: id, At: world.InitialTime + 3, Kind: "message", Payload: []byte("result")})
		_, _ = b.quiesce()
		_, _ = b.quiesce()
	})
	add("resource-and-kind-ordering", "queue.go lessEvent field order", world.Config{Seed: 5, Limits: limits}, func(b *worldConformanceBuilder) {
		for _, request := range []world.Request{
			{Kind: "b", Resource: world.ResourceID{Adapter: "a", Kind: "k", Key: "2"}},
			{Kind: "a", Resource: world.ResourceID{Adapter: "a", Kind: "k", Key: "2"}},
			{Kind: "a", Resource: world.ResourceID{Adapter: "a", Kind: "k", Key: "1"}},
			{Kind: "a", Resource: world.ResourceID{Adapter: "a", Kind: "j", Key: "9"}},
			{Kind: "a", Resource: world.ResourceID{Adapter: "b", Kind: "a", Key: ""}, Priority: 0},
			{Kind: "a", Resource: world.ResourceID{Adapter: "a", Kind: "k", Key: "1"}, Priority: 3},
		} {
			id, _ := b.register(request)
			_, _ = b.ready(world.Readiness{RequestID: id, At: world.InitialTime + 5, Kind: "y"})
		}
		late, _ := b.register(world.Request{Kind: "a", Resource: world.ResourceID{Adapter: "a", Kind: "k", Key: "1"}})
		_, _ = b.ready(world.Readiness{RequestID: late, At: world.InitialTime + 5, Kind: "x", EquivalenceClass: "c"})
		early, _ := b.register(world.Request{Kind: "z", Resource: world.ResourceID{Adapter: "z", Kind: "z", Key: "z"}, Priority: 9})
		_, _ = b.ready(world.Readiness{RequestID: early, At: world.InitialTime + 4, Kind: "z"})
		_, _ = b.quiesce()
		_, _ = b.quiesce()
		_, _ = b.quiesce()
	})
	add("broker-incarnation-pattern", "wasi/broker.go register/ready/quiesce per visible call", world.Config{Seed: 7, Limits: world.Limits{MaxRequests: 10000, MaxEvents: 10000, MaxQueuedEvents: 1000, MaxTransitions: 40000, MaxPayloadBytes: 32 << 20, MaxStringBytes: 1024}}, func(b *worldConformanceBuilder) {
		for call := range 12 {
			node := []string{"a/1", "b/1"}[call%2]
			op := []string{"runtime_idle", "runtime_net_stream"}[call%3%2]
			digest := sha256.Sum256([]byte(fmt.Sprint(call)))
			id, _ := b.register(world.Request{Kind: op, Resource: world.ResourceID{Adapter: "wasm-broker", Kind: "incarnation", Key: node}, Payload: digest[:]})
			if call == 9 {
				_, _ = b.cancel(id)
				continue
			}
			_, _ = b.ready(world.Readiness{RequestID: id, At: world.InitialTime + 1 + world.LogicalTime(call/4*1000), Kind: op})
			_, _ = b.quiesce()
		}
	})
	return corpora
}

func worldConformanceGenerated(t *testing.T) []worldConformanceCorpus {
	var corpora []worldConformanceCorpus
	for seed := uint64(1); seed <= 24; seed++ {
		random := rand.New(rand.NewPCG(seed, 0x178_0002))
		limits := world.Limits{MaxRequests: 64, MaxEvents: 64, MaxQueuedEvents: 32, MaxTransitions: 400, MaxPayloadBytes: 4096, MaxStringBytes: 64}
		if seed%3 == 0 {
			limits = world.Limits{MaxRequests: 8, MaxEvents: 10, MaxQueuedEvents: 3, MaxTransitions: 90, MaxPayloadBytes: 48, MaxStringBytes: 12}
		}
		worldSeed := random.Uint64()
		if seed%5 == 0 {
			worldSeed = seed
		}
		b := newWorldConformance(t, fmt.Sprintf("generated-%02d", seed), "generated", fmt.Sprintf("PCG(%d, 0x1780002)", seed), world.Config{Seed: world.Seed(worldSeed), Limits: limits})
		states := map[world.RequestID]string{}
		var ids []world.RequestID
		now := world.InitialTime
		pick := func(values ...string) string { return values[random.IntN(len(values))] }
		payload := func(limit int) []byte {
			data := make([]byte, random.IntN(limit+1))
			for i := range data {
				data[i] = byte(random.Uint32())
			}
			return data
		}
		steps := 120 + random.IntN(120)
		for range steps {
			switch weight := random.IntN(100); {
			case weight < 30:
				request := world.Request{
					Kind:     pick("read", "write", "wait", "receive", "runtime_idle", "x"),
					Resource: world.ResourceID{Adapter: pick("memory", "network", "wasm-broker", "a.b-c_d"), Kind: pick("cell", "queue", "incarnation"), Key: pick("", "a", "b", "node-a/1", "node-b/2", "üñî")},
					Priority: world.Priority(random.IntN(4)),
					Payload:  payload(6),
				}
				switch random.IntN(40) {
				case 0:
					request.Kind = ""
				case 1:
					request.Resource.Adapter = "Bad"
				case 2:
					request.Resource.Kind = "9starts-with-digit"
				case 3:
					request.Resource.Key = strings.Repeat("k", 70)
				}
				id, err := b.register(request)
				if err == nil {
					states[id] = "pending"
					ids = append(ids, id)
				}
			case weight < 60:
				var target world.RequestID
				switch choice := random.IntN(10); {
				case choice < 7:
					var pending []world.RequestID
					for _, id := range ids {
						if states[id] == "pending" {
							pending = append(pending, id)
						}
					}
					if len(pending) != 0 {
						target = pending[random.IntN(len(pending))]
					}
				case choice < 9 && len(ids) != 0:
					target = ids[random.IntN(len(ids))]
				default:
					target = world.RequestID(random.IntN(3) * 1000)
				}
				at := now + world.LogicalTime([]int{0, 0, 0, 1, 2, 7, 1000}[random.IntN(7)])
				if random.IntN(20) == 0 {
					at = now - 1
				}
				readiness := world.Readiness{RequestID: target, At: at, Kind: pick("result", "message", "done", "idle"), Payload: payload(5), EquivalenceClass: pick("", "", "x", "y", "exchangeable")}
				if random.IntN(40) == 0 {
					readiness.Kind = ""
				}
				if _, err := b.ready(readiness); err == nil {
					states[target] = "queued"
				}
			case weight < 75:
				target := world.RequestID(random.IntN(4) * 1000)
				if len(ids) != 0 && random.IntN(6) != 0 {
					target = ids[random.IntN(len(ids))]
				}
				if result, err := b.cancel(target); err == nil && result.Status == world.CancelWon {
					states[target] = "canceled"
				}
			default:
				if result, err := b.quiesce(); err == nil {
					now = result.After
					for _, delivery := range result.Deliveries {
						states[delivery.RequestID] = "delivered"
					}
				}
			}
		}
		corpora = append(corpora, b.finish())
	}
	return corpora
}

// worldConformanceRecordedReplay rebuilds a recorded corpus's outputs from its
// own checked-in inputs, so recorded histories also detect Go World drift.
func worldConformanceRecordedReplay(t *testing.T, recorded worldConformanceCorpus) worldConformanceCorpus {
	t.Helper()
	b := newWorldConformance(t, recorded.Name, recorded.Source, recorded.Origin, recorded.Config.world())
	if b.model == nil {
		t.Fatalf("recorded corpus %s has an invalid configuration", recorded.Name)
	}
	for index, step := range recorded.Steps {
		switch step.Op {
		case "register":
			request := step.Request
			_, _ = b.register(world.Request{Kind: request.Kind, Resource: world.ResourceID{Adapter: request.Adapter, Kind: request.ResourceKind, Key: request.Key}, Priority: world.Priority(request.Priority), Payload: mustBase64Std(t, request.PayloadBase64)})
		case "ready":
			readiness := step.Readiness
			_, _ = b.ready(world.Readiness{RequestID: world.RequestID(readiness.RequestID), At: world.LogicalTime(readiness.At), Kind: readiness.Kind, Payload: mustBase64Std(t, readiness.PayloadBase64), EquivalenceClass: readiness.EquivalenceClass})
		case "cancel":
			_, _ = b.cancel(world.RequestID(step.RequestID))
		case "quiesce":
			_, _ = b.quiesce()
		default:
			t.Fatalf("recorded corpus %s step %d: unknown op %q", recorded.Name, index, step.Op)
		}
	}
	return b.finish()
}

// worldConformanceCapture turns the transitions of a retained Go broker World
// evidence envelope into a recorded corpus of its first `limit` transitions.
func worldConformanceCapture(t *testing.T, name, path string, limit int) worldConformanceCorpus {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := world.DecodeCompressedSnapshotState(data, world.CompressedSnapshotProfile())
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(data)
	transitions := snapshot.Transitions[:min(limit, len(snapshot.Transitions))]
	b := newWorldConformance(t, name, "recorded", fmt.Sprintf("first %d of %d transitions of GMWORLD2 sha256:%x (TestTemporalIndependentClustersPartitionHealWholeClusterCrash broker World)", len(transitions), len(snapshot.Transitions), sum), snapshot.Config)
	for _, transition := range transitions {
		switch transition.Kind {
		case "register":
			_, _ = b.register(transition.Register.Request)
		case "ready":
			_, _ = b.ready(transition.Ready.Readiness)
		case "cancel":
			_, _ = b.cancel(transition.Cancel.RequestID)
		case "quiesce":
			_, _ = b.quiesce()
		default:
			t.Fatalf("recorded World transition kind %q", transition.Kind)
		}
	}
	corpus := b.finish()
	if last := corpus.Steps[len(corpus.Steps)-1].Expected.State.TranscriptDigest; last != string(transitions[len(transitions)-1].Digest) {
		t.Fatalf("captured World replay diverged from the recorded transcript: %s != %s", last, transitions[len(transitions)-1].Digest)
	}
	return corpus
}

func worldConformanceCorpora(t *testing.T) []worldConformanceCorpus {
	t.Helper()
	corpora := append(worldConformanceScripted(t), worldConformanceGenerated(t)...)
	entries, err := os.ReadDir(worldConformanceDirectory)
	if err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	if source := os.Getenv("GOMAD_WORLD_CONFORMANCE_RECORDED_SOURCE"); source != "" {
		return append(corpora, worldConformanceCapture(t, "recorded-independent-clusters", source, 4096))
	}
	for _, entry := range entries {
		if !strings.HasPrefix(entry.Name(), "recorded-") || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		recorded := readWorldConformance(t, filepath.Join(worldConformanceDirectory, entry.Name()))
		corpora = append(corpora, worldConformanceRecordedReplay(t, recorded))
	}
	return corpora
}

func readWorldConformance(t *testing.T, path string) worldConformanceCorpus {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var corpus worldConformanceCorpus
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&corpus); err != nil {
		t.Fatalf("%s: %v", path, err)
	}
	return corpus
}

// TestWorldConformanceCorpusIsCurrent fails on any Go World behavior change
// until the corpus is regenerated, which then forces the Rust replay
// (`cargo test --lib world_conformance`) to be rerun against it.
func TestWorldConformanceCorpusIsCurrent(t *testing.T) {
	regenerate := os.Getenv("GOMAD_WORLD_CONFORMANCE_REGENERATE") == "1"
	if regenerate {
		if err := os.MkdirAll(worldConformanceDirectory, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	want := map[string]bool{}
	sources := map[string]int{}
	for _, corpus := range worldConformanceCorpora(t) {
		name := corpus.Name + ".json"
		if want[name] {
			t.Fatalf("duplicate World conformance corpus %s", name)
		}
		want[name] = true
		sources[corpus.Source]++
		data, err := json.Marshal(corpus)
		if err != nil {
			t.Fatal(err)
		}
		data = append(data, '\n')
		path := filepath.Join(worldConformanceDirectory, name)
		if regenerate {
			if err := os.WriteFile(path, data, 0o644); err != nil {
				t.Fatal(err)
			}
			continue
		}
		got, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got, data) {
			t.Errorf("World conformance corpus %s diverges from the Go World at %s; review the Go change, then run GOMAD_WORLD_CONFORMANCE_REGENERATE=1 go test -tags test_dep -run '^TestWorldConformance' ./internal/backends/wasm/cluster and rerun the Rust replay", name, firstWorldConformanceDivergence(got, corpus))
		}
	}
	if regenerate {
		if err := os.WriteFile(worldConformancePin, []byte(world.ImplementationSHA256()+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	for _, source := range []string{"scripted", "generated", "recorded"} {
		if sources[source] == 0 {
			t.Errorf("World conformance corpus lacks %s cases", source)
		}
	}
	entries, err := os.ReadDir(worldConformanceDirectory)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if filepath.Ext(entry.Name()) == ".json" && !want[entry.Name()] {
			t.Errorf("unexpected World conformance corpus %s", entry.Name())
		}
	}
}

// firstWorldConformanceDivergence names the first differing step of a stale
// corpus, with the generator's seed or origin.
func firstWorldConformanceDivergence(checkedIn []byte, current worldConformanceCorpus) string {
	var old worldConformanceCorpus
	if err := json.Unmarshal(checkedIn, &old); err != nil {
		return "undecodable checked-in corpus: " + err.Error()
	}
	hash := ""
	if old.ImplementationSHA256 != current.ImplementationSHA256 {
		hash = fmt.Sprintf("; implementation hash %s -> %s", old.ImplementationSHA256, current.ImplementationSHA256)
	}
	for index := range max(len(old.Steps), len(current.Steps)) {
		if index >= len(old.Steps) || index >= len(current.Steps) {
			return fmt.Sprintf("step count %d -> %d (origin %s)%s", len(old.Steps), len(current.Steps), current.Origin, hash)
		}
		left, _ := json.Marshal(old.Steps[index])
		right, _ := json.Marshal(current.Steps[index])
		if !bytes.Equal(left, right) {
			return fmt.Sprintf("step %d %s (origin %s)%s", index, current.Steps[index].Op, current.Origin, hash)
		}
	}
	if hash != "" {
		return "steps identical (origin " + current.Origin + ")" + hash
	}
	return "final snapshot or header (origin " + current.Origin + ")"
}

func TestWorldConformanceDriftPinIsCurrent(t *testing.T) {
	pin, err := os.ReadFile(worldConformancePin)
	if err != nil {
		t.Fatal(err)
	}
	if want := world.ImplementationSHA256() + "\n"; !bytes.Equal(pin, []byte(want)) {
		t.Fatalf("Go World implementation changed (%s, pinned %s): regenerate the World conformance corpus and rerun the Rust World replay before updating the pin", strings.TrimSpace(want), strings.TrimSpace(string(pin)))
	}
	entries, err := os.ReadDir(worldConformanceDirectory)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		if corpus := readWorldConformance(t, filepath.Join(worldConformanceDirectory, entry.Name())); corpus.ImplementationSHA256+"\n" != string(pin) || corpus.Schema != worldConformanceSchema {
			t.Errorf("%s is pinned to %s, not %s", entry.Name(), corpus.ImplementationSHA256, strings.TrimSpace(string(pin)))
		}
	}
}

// TestWorldConformanceFinalWireRoundTrips proves every recorded expanded wire
// is canonical for the Go decoder and wraps into Go's own envelope bytes.
func TestWorldConformanceFinalWireRoundTrips(t *testing.T) {
	entries, err := os.ReadDir(worldConformanceDirectory)
	if err != nil {
		t.Fatal(err)
	}
	checked := 0
	for _, entry := range entries {
		if filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		corpus := readWorldConformance(t, filepath.Join(worldConformanceDirectory, entry.Name()))
		if corpus.Final == nil {
			continue
		}
		expanded := mustBase64Std(t, corpus.Final.ExpandedBase64)
		envelope, err := world.WrapExpandedSnapshot(expanded, world.CompressedSnapshotProfile())
		if err != nil {
			t.Fatalf("%s: %v", entry.Name(), err)
		}
		if sum := sha256.Sum256(envelope); hex.EncodeToString(sum[:]) != corpus.Final.EnvelopeSHA256 {
			t.Errorf("%s: Go-wrapped envelope differs from the Go-produced envelope", entry.Name())
		}
		snapshot, err := world.DecodeCompressedSnapshotState(envelope, world.CompressedSnapshotProfile())
		if err != nil {
			t.Fatalf("%s: %v", entry.Name(), err)
		}
		if _, err := world.Restore(snapshot, nil); err != nil {
			t.Fatalf("%s: restore: %v", entry.Name(), err)
		}
		if len(corpus.Steps) != 0 && string(snapshot.StateDigest) != corpus.Steps[len(corpus.Steps)-1].Expected.State.StateDigest {
			t.Errorf("%s: final snapshot state digest differs from the last step", entry.Name())
		}
		checked++
	}
	if checked == 0 {
		t.Fatal("no World conformance corpus carries a final wire")
	}
	if _, err := world.WrapExpandedSnapshot([]byte("GMWSNAP1"), world.CompressedSnapshotProfile()); !errors.Is(err, world.ErrInvalidSnapshot) {
		t.Fatalf("truncated expanded wire admitted: %v", err)
	}
	if _, err := world.WrapExpandedSnapshot(nil, world.CompressedSnapshotProfile()); !errors.Is(err, world.ErrInvalidSnapshot) {
		t.Fatalf("empty expanded wire admitted: %v", err)
	}
}
