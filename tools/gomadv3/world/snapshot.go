package world

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"sort"
)

type RequestSnapshot struct {
	ID      RequestID    `json:"id"`
	Request Request      `json:"request"`
	State   RequestState `json:"state"`
	EventID EventID      `json:"eventId,omitempty"`
}

type EventSnapshot struct {
	ID        EventID    `json:"id"`
	Readiness Readiness  `json:"readiness"`
	State     EventState `json:"state"`
}

type Snapshot struct {
	SchemaVersion    uint32            `json:"schemaVersion"`
	Config           Config            `json:"config"`
	Now              LogicalTime       `json:"now"`
	NextRequestID    RequestID         `json:"nextRequestId"`
	NextEventID      EventID           `json:"nextEventId"`
	NextTransition   Sequence          `json:"nextTransition"`
	PayloadBytes     uint64            `json:"payloadBytes"`
	Requests         []RequestSnapshot `json:"requests"`
	Events           []EventSnapshot   `json:"events"`
	Transitions      []Transition      `json:"transitions"`
	TranscriptDigest Digest            `json:"transcriptDigest"`
	Replay           ReplayProgress    `json:"replay"`
	StateDigest      Digest            `json:"stateDigest"`
}

func (w *Model) Snapshot() Snapshot {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.snapshotLocked()
}

func (w *Model) snapshotLocked() Snapshot {
	snapshot := Snapshot{
		SchemaVersion: SchemaVersion, Config: w.config, Now: w.now, NextRequestID: w.nextRequestID, NextEventID: w.nextEventID,
		NextTransition: w.nextTransition, PayloadBytes: w.payloadBytes, TranscriptDigest: w.transcript,
	}
	if w.replay != nil {
		snapshot.Replay = ReplayProgress{Cursor: w.replay.cursor, Expected: uint64(len(w.replay.plan.Transitions))}
	}
	snapshot.Requests = make([]RequestSnapshot, 0, len(w.requests))
	for _, request := range w.requests {
		snapshot.Requests = append(snapshot.Requests, RequestSnapshot{ID: request.id, Request: copyRequest(request.request), State: request.state, EventID: request.eventID})
	}
	sort.Slice(snapshot.Requests, func(i, j int) bool { return snapshot.Requests[i].ID < snapshot.Requests[j].ID })
	snapshot.Events = make([]EventSnapshot, 0, len(w.events))
	for _, event := range w.events {
		snapshot.Events = append(snapshot.Events, EventSnapshot{ID: event.id, Readiness: copyReadiness(event.readiness), State: event.state})
	}
	sort.Slice(snapshot.Events, func(i, j int) bool { return snapshot.Events[i].ID < snapshot.Events[j].ID })
	snapshot.Transitions = copyTransitions(w.history)
	snapshot.StateDigest = stateDigest(snapshot)
	return snapshot
}

func Restore(snapshot Snapshot, replay *ReplayPlan) (*Model, error) {
	if snapshot.SchemaVersion != SchemaVersion {
		return nil, invalidSnapshot("schemaVersion")
	}
	if err := validateConfig(snapshot.Config); err != nil {
		return nil, invalidSnapshot("config")
	}
	if snapshot.Now < 0 || !validDigest(snapshot.TranscriptDigest) || !validDigest(snapshot.StateDigest) {
		return nil, invalidSnapshot("digest-or-time")
	}
	if snapshot.Replay.Cursor > snapshot.Replay.Expected {
		return nil, invalidSnapshot("replay.cursor")
	}
	if uint64(len(snapshot.Requests)) > snapshot.Config.Limits.MaxRequests || uint64(len(snapshot.Events)) > snapshot.Config.Limits.MaxEvents || uint64(len(snapshot.Transitions)) > snapshot.Config.Limits.MaxTransitions {
		return nil, invalidSnapshot("lifetime-count")
	}
	for index, request := range snapshot.Requests {
		if request.ID == 0 || index > 0 && request.ID <= snapshot.Requests[index-1].ID {
			return nil, invalidSnapshot("requests.order")
		}
	}
	for index, event := range snapshot.Events {
		if event.ID == 0 || index > 0 && event.ID <= snapshot.Events[index-1].ID {
			return nil, invalidSnapshot("events.order")
		}
	}
	for index, transition := range snapshot.Transitions {
		if transition.Sequence == 0 || index > 0 && transition.Sequence <= snapshot.Transitions[index-1].Sequence {
			return nil, invalidSnapshot("transitions.order")
		}
	}
	if err := validateSnapshotPayloads(snapshot); err != nil {
		return nil, err
	}
	if stateDigest(snapshot) != snapshot.StateDigest {
		return nil, invalidSnapshot("stateDigest")
	}

	reconstructed := newModel(snapshot.Config)
	for index, expected := range snapshot.Transitions {
		if err := replaySnapshotTransition(reconstructed, expected); err != nil {
			return nil, invalidSnapshot(fmt.Sprintf("transitions[%d]: %v", index, err))
		}
		actual := reconstructed.history[len(reconstructed.history)-1]
		if string(transitionBytes(actual, true)) != string(transitionBytes(expected, true)) {
			return nil, invalidSnapshot(fmt.Sprintf("transitions[%d].result", index))
		}
	}
	actual := reconstructed.Snapshot()
	if actual.StateDigest != snapshot.StateDigest {
		return nil, invalidSnapshot("state")
	}
	if replay != nil {
		if err := reconstructed.attachReplay(snapshot, *replay); err != nil {
			return nil, err
		}
	}
	return reconstructed, nil
}

func validateSnapshotPayloads(snapshot Snapshot) error {
	limit := snapshot.Config.Limits.MaxPayloadBytes
	var used uint64
	add := func(size uint64) error {
		if size > limit-used {
			return invalidSnapshot("payloadBytes")
		}
		used += size
		return nil
	}
	for _, request := range snapshot.Requests {
		if err := add(uint64(len(request.Request.Payload))); err != nil {
			return err
		}
	}
	for _, event := range snapshot.Events {
		if err := add(uint64(len(event.Readiness.Payload))); err != nil {
			return err
		}
	}
	for _, transition := range snapshot.Transitions {
		size, ok := checkedTransitionPayloadSize(transition)
		if !ok {
			return invalidSnapshot("payloadBytes")
		}
		if err := add(size); err != nil {
			return err
		}
	}
	if used != snapshot.PayloadBytes {
		return invalidSnapshot("payloadBytes")
	}
	return nil
}

func replaySnapshotTransition(w *Model, transition Transition) error {
	if err := validateTransitionShape(transition); err != nil {
		return err
	}
	switch transition.Kind {
	case "register":
		id, err := w.Register(transition.Register.Request)
		if err != nil || id != transition.Register.RequestID {
			return fmt.Errorf("register result")
		}
	case "ready":
		id, err := w.Ready(transition.Ready.Readiness)
		if err != nil || id != transition.Ready.EventID {
			return fmt.Errorf("ready result")
		}
	case "cancel":
		result, err := w.Cancel(transition.Cancel.RequestID)
		if err != nil || result != transition.Cancel.Cancellation {
			return fmt.Errorf("cancel result")
		}
	case "quiesce":
		result, err := w.Quiesce()
		if err != nil || string(appendQuiescence(nil, result)) != string(appendQuiescence(nil, transition.Quiesce.Result)) {
			return fmt.Errorf("quiesce result")
		}
	}
	return nil
}

func invalidSnapshot(field string) error {
	return fmt.Errorf("%w: %s", ErrInvalidSnapshot, field)
}

func stateDigest(snapshot Snapshot) Digest {
	data := []byte("gomadv3/world/state/v1\x00")
	data = appendUint32(data, snapshot.SchemaVersion)
	data = appendConfig(data, snapshot.Config)
	data = appendInt64(data, int64(snapshot.Now))
	data = appendUint64(data, uint64(snapshot.NextRequestID))
	data = appendUint64(data, uint64(snapshot.NextEventID))
	data = appendUint64(data, uint64(snapshot.NextTransition))
	data = appendUint64(data, snapshot.PayloadBytes)
	data = appendUint64(data, uint64(len(snapshot.Requests)))
	for _, request := range snapshot.Requests {
		data = appendUint64(data, uint64(request.ID))
		data = appendRequest(data, request.Request)
		data = appendString(data, string(request.State))
		data = appendUint64(data, uint64(request.EventID))
	}
	data = appendUint64(data, uint64(len(snapshot.Events)))
	for _, event := range snapshot.Events {
		data = appendUint64(data, uint64(event.ID))
		data = appendReadiness(data, event.Readiness)
		data = appendString(data, string(event.State))
	}
	data = appendUint64(data, uint64(len(snapshot.Transitions)))
	for _, transition := range snapshot.Transitions {
		data = appendBytes(data, transitionBytes(transition, true))
	}
	data = appendString(data, string(snapshot.TranscriptDigest))
	digest := sha256.Sum256(data)
	return Digest(hex.EncodeToString(digest[:]))
}

func appendConfig(data []byte, config Config) []byte {
	data = appendUint64(data, uint64(config.Seed))
	data = appendUint64(data, config.Limits.MaxRequests)
	data = appendUint64(data, config.Limits.MaxEvents)
	data = appendUint64(data, config.Limits.MaxQueuedEvents)
	data = appendUint64(data, config.Limits.MaxTransitions)
	data = appendUint64(data, config.Limits.MaxPayloadBytes)
	return appendUint32(data, config.Limits.MaxStringBytes)
}

func appendRequest(data []byte, request Request) []byte {
	data = appendString(data, request.Kind)
	data = appendString(data, request.Resource.Adapter)
	data = appendString(data, request.Resource.Kind)
	data = appendString(data, request.Resource.Key)
	data = appendUint32(data, uint32(request.Priority))
	return appendBytes(data, request.Payload)
}

func appendReadiness(data []byte, readiness Readiness) []byte {
	data = appendUint64(data, uint64(readiness.RequestID))
	data = appendInt64(data, int64(readiness.At))
	data = appendString(data, readiness.Kind)
	data = appendBytes(data, readiness.Payload)
	return appendString(data, readiness.EquivalenceClass)
}

func appendCancellation(data []byte, cancellation Cancellation) []byte {
	data = appendUint64(data, uint64(cancellation.RequestID))
	data = appendUint64(data, uint64(cancellation.EventID))
	return appendString(data, string(cancellation.Status))
}

func appendQuiescence(data []byte, quiescence Quiescence) []byte {
	data = appendString(data, string(quiescence.Kind))
	data = appendInt64(data, int64(quiescence.Before))
	data = appendInt64(data, int64(quiescence.After))
	data = appendUint64(data, uint64(len(quiescence.Deliveries)))
	for _, delivery := range quiescence.Deliveries {
		data = appendUint64(data, uint64(delivery.RequestID))
		data = appendUint64(data, uint64(delivery.EventID))
		data = appendInt64(data, int64(delivery.At))
		data = appendString(data, delivery.Kind)
		data = appendBytes(data, delivery.Payload)
	}
	data = appendUint64(data, uint64(len(quiescence.Blocked)))
	for _, blocked := range quiescence.Blocked {
		data = appendUint64(data, uint64(blocked))
	}
	return data
}

func appendUint64(data []byte, value uint64) []byte {
	var encoded [8]byte
	binary.BigEndian.PutUint64(encoded[:], value)
	return append(data, encoded[:]...)
}

func appendInt64(data []byte, value int64) []byte {
	return appendUint64(data, uint64(value))
}

func appendUint32(data []byte, value uint32) []byte {
	var encoded [4]byte
	binary.BigEndian.PutUint32(encoded[:], value)
	return append(data, encoded[:]...)
}

func appendString(data []byte, value string) []byte {
	return appendBytes(data, []byte(value))
}

func appendBytes(data, value []byte) []byte {
	data = appendUint64(data, uint64(len(value)))
	return append(data, value...)
}

func validDigest(value Digest) bool {
	if len(value) != sha256.Size*2 {
		return false
	}
	for _, character := range value {
		if character < '0' || character > '9' && character < 'a' || character > 'f' {
			return false
		}
	}
	_, err := hex.DecodeString(string(value))
	return err == nil
}
