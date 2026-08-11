package world

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

type Transition struct {
	Sequence       Sequence            `json:"sequence"`
	Kind           string              `json:"kind"`
	PreviousDigest Digest              `json:"previousDigest"`
	Digest         Digest              `json:"digest"`
	Register       *RegisterTransition `json:"register,omitempty"`
	Ready          *ReadyTransition    `json:"ready,omitempty"`
	Cancel         *CancelTransition   `json:"cancel,omitempty"`
	Quiesce        *QuiesceTransition  `json:"quiesce,omitempty"`
}

type RegisterTransition struct {
	Request   Request   `json:"request"`
	RequestID RequestID `json:"requestId"`
}

type ReadyTransition struct {
	Readiness Readiness `json:"readiness"`
	EventID   EventID   `json:"eventId"`
}

type CancelTransition struct {
	RequestID    RequestID    `json:"requestId"`
	Cancellation Cancellation `json:"cancellation"`
}

type QuiesceTransition struct {
	Result Quiescence `json:"result"`
}

type ReplayPlan struct {
	SchemaVersion uint32       `json:"schemaVersion"`
	InitialDigest Digest       `json:"initialDigest"`
	Transitions   []Transition `json:"transitions"`
	FinalDigest   Digest       `json:"finalDigest"`
}

func validateTransitionShape(transition Transition) error {
	bodies := 0
	if transition.Register != nil {
		bodies++
	}
	if transition.Ready != nil {
		bodies++
	}
	if transition.Cancel != nil {
		bodies++
	}
	if transition.Quiesce != nil {
		bodies++
	}
	if bodies != 1 || transition.Sequence == 0 || !validDigest(transition.PreviousDigest) || !validDigest(transition.Digest) {
		return invalidSnapshot("transition.shape")
	}
	switch transition.Kind {
	case "register":
		if transition.Register == nil {
			return invalidSnapshot("transition.register")
		}
	case "ready":
		if transition.Ready == nil {
			return invalidSnapshot("transition.ready")
		}
	case "cancel":
		if transition.Cancel == nil {
			return invalidSnapshot("transition.cancel")
		}
	case "quiesce":
		if transition.Quiesce == nil {
			return invalidSnapshot("transition.quiesce")
		}
	default:
		return invalidSnapshot("transition.kind")
	}
	if transitionDigest(transition) != transition.Digest {
		return invalidSnapshot("transition.digest")
	}
	return nil
}

func (w *World) attachReplay(snapshot Snapshot, plan ReplayPlan) error {
	if plan.SchemaVersion != SchemaVersion || plan.InitialDigest != snapshot.StateDigest || !validDigest(plan.FinalDigest) {
		return fmt.Errorf("%w: replay plan identity", ErrInvalidSnapshot)
	}
	if uint64(len(plan.Transitions)) > w.config.Limits.MaxTransitions {
		return fmt.Errorf("%w: replay transition count", ErrInvalidSnapshot)
	}
	previous := w.transcript
	sequence := w.nextTransition
	for index, transition := range plan.Transitions {
		if err := validateTransitionShape(transition); err != nil || transition.Sequence != sequence || transition.PreviousDigest != previous {
			return fmt.Errorf("%w: replay transition %d", ErrInvalidSnapshot, index)
		}
		previous = transition.Digest
		sequence++
	}
	verifier, err := Restore(snapshot, nil)
	if err != nil {
		return err
	}
	for index, transition := range plan.Transitions {
		if err := replaySnapshotTransition(verifier, transition); err != nil {
			return fmt.Errorf("%w: replay transition %d result", ErrInvalidSnapshot, index)
		}
		actual := verifier.history[len(verifier.history)-1]
		if string(transitionBytes(actual, true)) != string(transitionBytes(transition, true)) {
			return fmt.Errorf("%w: replay transition %d identity", ErrInvalidSnapshot, index)
		}
	}
	if verifier.Snapshot().StateDigest != plan.FinalDigest {
		return fmt.Errorf("%w: replay final digest", ErrInvalidSnapshot)
	}
	var replayPayloadBytes uint64
	for _, transition := range plan.Transitions {
		delta, ok := checkedTransitionPayloadSize(transition)
		if !ok {
			return fmt.Errorf("%w: replay payload capacity", ErrInvalidSnapshot)
		}
		if delta > w.config.Limits.MaxPayloadBytes-replayPayloadBytes {
			return fmt.Errorf("%w: replay payload capacity", ErrInvalidSnapshot)
		}
		replayPayloadBytes += delta
	}
	if w.payloadBytes > w.config.Limits.MaxPayloadBytes-replayPayloadBytes {
		return fmt.Errorf("%w: replay payload capacity", ErrInvalidSnapshot)
	}
	w.replay = &replayState{plan: ReplayPlan{SchemaVersion: plan.SchemaVersion, InitialDigest: plan.InitialDigest, Transitions: copyTransitions(plan.Transitions), FinalDigest: plan.FinalDigest}}
	w.replayPayloadBytes = replayPayloadBytes
	return nil
}

func (w *World) nextRegisterTransition(request Request, id RequestID) Transition {
	transition := Transition{Sequence: w.nextTransition, Kind: "register", PreviousDigest: w.transcript, Register: &RegisterTransition{Request: copyRequest(request), RequestID: id}}
	transition.Digest = transitionDigest(transition)
	return transition
}

func (w *World) nextReadyTransition(readiness Readiness, id EventID) Transition {
	transition := Transition{Sequence: w.nextTransition, Kind: "ready", PreviousDigest: w.transcript, Ready: &ReadyTransition{Readiness: copyReadiness(readiness), EventID: id}}
	transition.Digest = transitionDigest(transition)
	return transition
}

func (w *World) nextCancelTransition(id RequestID, cancellation Cancellation) Transition {
	transition := Transition{Sequence: w.nextTransition, Kind: "cancel", PreviousDigest: w.transcript, Cancel: &CancelTransition{RequestID: id, Cancellation: cancellation}}
	transition.Digest = transitionDigest(transition)
	return transition
}

func (w *World) nextQuiesceTransition(result Quiescence) Transition {
	transition := Transition{Sequence: w.nextTransition, Kind: "quiesce", PreviousDigest: w.transcript, Quiesce: &QuiesceTransition{Result: copyQuiescence(result)}}
	transition.Digest = transitionDigest(transition)
	return transition
}

func transitionDigest(transition Transition) Digest {
	data := []byte("gomadv3/world/transcript/v1\x00")
	data = appendString(data, string(transition.PreviousDigest))
	data = append(data, transitionBytes(transition, false)...)
	digest := sha256.Sum256(data)
	return Digest(hex.EncodeToString(digest[:]))
}

func transitionBytes(transition Transition, includeDigest bool) []byte {
	data := appendUint64(nil, uint64(transition.Sequence))
	data = appendString(data, transition.Kind)
	data = appendString(data, string(transition.PreviousDigest))
	if includeDigest {
		data = appendString(data, string(transition.Digest))
	}
	switch transition.Kind {
	case "register":
		data = appendRequest(data, transition.Register.Request)
		data = appendUint64(data, uint64(transition.Register.RequestID))
	case "ready":
		data = appendReadiness(data, transition.Ready.Readiness)
		data = appendUint64(data, uint64(transition.Ready.EventID))
	case "cancel":
		data = appendUint64(data, uint64(transition.Cancel.RequestID))
		data = appendCancellation(data, transition.Cancel.Cancellation)
	case "quiesce":
		data = appendQuiescence(data, transition.Quiesce.Result)
	}
	return data
}

func (w *World) checkReplay(actual Transition) error {
	if w.replay == nil {
		return nil
	}
	if w.replay.cursor >= uint64(len(w.replay.plan.Transitions)) {
		return &ReplayDivergenceError{Index: w.replay.cursor, ExpectedKind: "end", ActualKind: actual.Kind, Field: "operation", ActualDigest: actual.Digest}
	}
	expected := w.replay.plan.Transitions[w.replay.cursor]
	if string(transitionBytes(expected, true)) != string(transitionBytes(actual, true)) {
		field := transitionDifference(expected, actual)
		return &ReplayDivergenceError{Index: w.replay.cursor, ExpectedKind: expected.Kind, ActualKind: actual.Kind, Field: field, ExpectedDigest: expected.Digest, ActualDigest: actual.Digest}
	}
	return nil
}

func transitionDifference(expected, actual Transition) string {
	if expected.Sequence != actual.Sequence {
		return "sequence"
	}
	if expected.Kind != actual.Kind {
		return "kind"
	}
	if expected.PreviousDigest != actual.PreviousDigest {
		return "previousDigest"
	}
	switch expected.Kind {
	case "register":
		if field := requestDifference(expected.Register.Request, actual.Register.Request); field != "" {
			return "register.request." + field
		}
		if expected.Register.RequestID != actual.Register.RequestID {
			return "register.requestId"
		}
	case "ready":
		if field := readinessDifference(expected.Ready.Readiness, actual.Ready.Readiness); field != "" {
			return "ready.readiness." + field
		}
		if expected.Ready.EventID != actual.Ready.EventID {
			return "ready.eventId"
		}
	case "cancel":
		if expected.Cancel.RequestID != actual.Cancel.RequestID {
			return "cancel.requestId"
		}
		if expected.Cancel.Cancellation != actual.Cancel.Cancellation {
			return "cancel.cancellation"
		}
	case "quiesce":
		if string(appendQuiescence(nil, expected.Quiesce.Result)) != string(appendQuiescence(nil, actual.Quiesce.Result)) {
			return "quiesce.result"
		}
	}
	return "digest"
}

func requestDifference(expected, actual Request) string {
	if expected.Kind != actual.Kind {
		return "kind"
	}
	if expected.Resource.Adapter != actual.Resource.Adapter {
		return "resource.adapter"
	}
	if expected.Resource.Kind != actual.Resource.Kind {
		return "resource.kind"
	}
	if expected.Resource.Key != actual.Resource.Key {
		return "resource.key"
	}
	if expected.Priority != actual.Priority {
		return "priority"
	}
	if !bytes.Equal(expected.Payload, actual.Payload) {
		return "payload"
	}
	return ""
}

func readinessDifference(expected, actual Readiness) string {
	if expected.RequestID != actual.RequestID {
		return "requestId"
	}
	if expected.At != actual.At {
		return "at"
	}
	if expected.Kind != actual.Kind {
		return "kind"
	}
	if !bytes.Equal(expected.Payload, actual.Payload) {
		return "payload"
	}
	if expected.EquivalenceClass != actual.EquivalenceClass {
		return "equivalenceClass"
	}
	return ""
}

func (w *World) commitTransition(transition Transition) {
	w.history = append(w.history, transition)
	w.transcript = transition.Digest
	if w.nextTransition != Sequence(^uint64(0)) {
		w.nextTransition++
	}
	if w.recording != nil {
		w.recording.used += w.recording.pending
		w.recording.pending = 0
	}
	if w.replay != nil {
		consumed := &w.replay.plan.Transitions[w.replay.cursor]
		w.replayPayloadBytes -= transitionPayloadSize(*consumed)
		clearTransitionPayload(consumed)
		w.replay.cursor++
	}
}

func EncodeTransitions(transitions []Transition) ([]byte, error) {
	var output bytes.Buffer
	for index, transition := range transitions {
		if err := validateTransitionShape(transition); err != nil {
			return nil, fmt.Errorf("transition %d: %w", index, err)
		}
		encoded, err := canonicalJSON(transition)
		if err != nil {
			return nil, fmt.Errorf("transition %d: %w", index, err)
		}
		output.Write(encoded)
		output.WriteByte('\n')
	}
	return output.Bytes(), nil
}

func transitionPayloadSize(transition Transition) uint64 {
	size, _ := checkedTransitionPayloadSize(transition)
	return size
}

func checkedTransitionPayloadSize(transition Transition) (uint64, bool) {
	switch transition.Kind {
	case "register":
		return uint64(len(transition.Register.Request.Payload)), true
	case "ready":
		return uint64(len(transition.Ready.Readiness.Payload)), true
	case "quiesce":
		var total uint64
		for _, delivery := range transition.Quiesce.Result.Deliveries {
			size := uint64(len(delivery.Payload))
			if size > ^uint64(0)-total {
				return 0, false
			}
			total += size
		}
		return total, true
	default:
		return 0, true
	}
}

func clearTransitionPayload(transition *Transition) {
	switch transition.Kind {
	case "register":
		transition.Register.Request.Payload = nil
	case "ready":
		transition.Ready.Readiness.Payload = nil
	case "quiesce":
		for index := range transition.Quiesce.Result.Deliveries {
			transition.Quiesce.Result.Deliveries[index].Payload = nil
		}
	}
}

func emptyDigest() Digest {
	digest := sha256.Sum256(nil)
	return Digest(hex.EncodeToString(digest[:]))
}

func copyTransitions(input []Transition) []Transition {
	result := make([]Transition, len(input))
	for index, transition := range input {
		result[index] = transition
		if transition.Register != nil {
			body := *transition.Register
			body.Request = copyRequest(body.Request)
			result[index].Register = &body
		}
		if transition.Ready != nil {
			body := *transition.Ready
			body.Readiness = copyReadiness(body.Readiness)
			result[index].Ready = &body
		}
		if transition.Cancel != nil {
			body := *transition.Cancel
			result[index].Cancel = &body
		}
		if transition.Quiesce != nil {
			body := *transition.Quiesce
			body.Result = copyQuiescence(body.Result)
			result[index].Quiesce = &body
		}
	}
	return result
}
