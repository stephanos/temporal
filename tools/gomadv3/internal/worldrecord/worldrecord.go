package worldrecord

import (
	"bytes"
	"fmt"

	"go.temporal.io/server/tools/gomadv3/internal/record"
	"go.temporal.io/server/tools/gomadv3/world"
	"go.temporal.io/server/tools/gomadv3/world/mailbox"
)

type Bundle struct {
	Manifest record.World
	Payloads record.WorldPayloads
}

func Validate(manifest record.World, payloads record.WorldPayloads) (world.Snapshot, world.Snapshot, error) {
	if manifest.Initial.Schema == "gomadv3.world.snapshot/none" && manifest.Transitions.Schema == "gomadv3.world.transitions/none" && manifest.Final.Schema == "gomadv3.world.snapshot/none" {
		expectedManifest, expectedPayloads := record.NoneWorld()
		if !bytes.Equal(payloads.Initial, expectedPayloads.Initial) || !bytes.Equal(payloads.Transitions, expectedPayloads.Transitions) || !bytes.Equal(payloads.Final, expectedPayloads.Final) {
			return world.Snapshot{}, world.Snapshot{}, fmt.Errorf("none World payloads changed")
		}
		if !sameManifest(manifest, expectedManifest) {
			return world.Snapshot{}, world.Snapshot{}, fmt.Errorf("none World manifest changed")
		}
		return world.Snapshot{}, world.Snapshot{}, nil
	}
	if manifest.Initial.Schema != "gomadv3.world.snapshot/v1" || manifest.Transitions.Schema != "gomadv3.world.transitions/v1" || manifest.Final.Schema != "gomadv3.world.snapshot/v1" {
		return world.Snapshot{}, world.Snapshot{}, fmt.Errorf("incompatible World schema combination")
	}
	initial, err := world.DecodeSnapshot(payloads.Initial)
	if err != nil {
		return world.Snapshot{}, world.Snapshot{}, fmt.Errorf("decode initial World snapshot: %w", err)
	}
	final, err := world.DecodeSnapshot(payloads.Final)
	if err != nil {
		return world.Snapshot{}, world.Snapshot{}, fmt.Errorf("decode final World snapshot: %w", err)
	}
	transitions, err := record.StrictDecodeJSONLines[world.Transition](payloads.Transitions)
	if err != nil {
		return world.Snapshot{}, world.Snapshot{}, fmt.Errorf("decode World transitions: %w", err)
	}
	if uint64(len(transitions)) != uint64(manifest.Transitions.Count) {
		return world.Snapshot{}, world.Snapshot{}, fmt.Errorf("World transition count mismatch")
	}
	limit := uint64(len(payloads.Transitions))
	if limit == 0 {
		limit = 1
	}
	recomposed, err := compose(initial, final, world.Terminal{Kind: world.TerminalKind(manifest.Terminal.Kind), Detail: manifest.Terminal.Detail}, limit)
	if err != nil {
		return world.Snapshot{}, world.Snapshot{}, fmt.Errorf("recompose World record: %w", err)
	}
	if !bytes.Equal(recomposed.Payloads.Transitions, payloads.Transitions) || !sameManifest(recomposed.Manifest, manifest) {
		return world.Snapshot{}, world.Snapshot{}, fmt.Errorf("World semantic or transition identity mismatch")
	}
	return initial, final, nil
}

func sameManifest(left, right record.World) bool {
	leftBytes, leftErr := record.CanonicalJSON(left)
	rightBytes, rightErr := record.CanonicalJSON(right)
	return leftErr == nil && rightErr == nil && bytes.Equal(leftBytes, rightBytes)
}

func Compose(initial, final world.Snapshot, transitionLimit uint64) (Bundle, error) {
	terminal := world.Terminal{Kind: world.TerminalNone}
	if len(final.Transitions) > len(initial.Transitions) {
		last := final.Transitions[len(final.Transitions)-1]
		if last.Quiesce != nil {
			terminal.Kind = world.TerminalKind(last.Quiesce.Result.Kind)
		}
	}
	return compose(initial, final, terminal, transitionLimit)
}

func ComposeRecording(recording world.Recording, transitionLimit uint64) (Bundle, error) {
	return compose(recording.Initial, recording.Final, recording.Terminal, transitionLimit)
}

func compose(initial, final world.Snapshot, terminal world.Terminal, transitionLimit uint64) (Bundle, error) {
	if transitionLimit == 0 {
		return Bundle{}, fmt.Errorf("World transition limit must be positive")
	}
	if err := validateTerminal(terminal); err != nil {
		return Bundle{}, err
	}
	if err := world.ValidateRecording(world.Recording{Initial: initial, Final: final, Terminal: terminal}); err != nil {
		return Bundle{}, err
	}
	if _, err := world.Restore(initial, nil); err != nil {
		return Bundle{}, fmt.Errorf("validate initial World snapshot: %w", err)
	}
	if _, err := world.Restore(final, nil); err != nil {
		return Bundle{}, fmt.Errorf("validate final World snapshot: %w", err)
	}
	initialConfig, err := record.CanonicalJSON(initial.Config)
	if err != nil {
		return Bundle{}, fmt.Errorf("encode initial World config: %w", err)
	}
	finalConfig, err := record.CanonicalJSON(final.Config)
	if err != nil {
		return Bundle{}, fmt.Errorf("encode final World config: %w", err)
	}
	if !bytes.Equal(initialConfig, finalConfig) {
		return Bundle{}, fmt.Errorf("initial and final World configs differ")
	}
	if len(final.Transitions) < len(initial.Transitions) {
		return Bundle{}, fmt.Errorf("final World transition history regressed")
	}
	for index := range initial.Transitions {
		initialTransition, encodeErr := record.CanonicalJSON(initial.Transitions[index])
		if encodeErr != nil {
			return Bundle{}, fmt.Errorf("encode initial World transition %d: %w", index, encodeErr)
		}
		finalTransition, encodeErr := record.CanonicalJSON(final.Transitions[index])
		if encodeErr != nil {
			return Bundle{}, fmt.Errorf("encode final World transition %d: %w", index, encodeErr)
		}
		if !bytes.Equal(initialTransition, finalTransition) {
			return Bundle{}, fmt.Errorf("final World transition history diverges at %d", index)
		}
	}
	delta := final.Transitions[len(initial.Transitions):]
	plan := world.ReplayPlan{SchemaVersion: world.SchemaVersion, InitialDigest: initial.StateDigest, Transitions: delta, FinalDigest: final.StateDigest}
	if _, err := world.Restore(initial, &plan); err != nil {
		return Bundle{}, fmt.Errorf("validate World transition continuation: %w", err)
	}
	if final.Replay.Expected != 0 && final.Replay.Cursor != final.Replay.Expected {
		return Bundle{}, fmt.Errorf("final World replay is incomplete")
	}

	initialBytes, err := world.EncodeSnapshot(initial)
	if err != nil {
		return Bundle{}, fmt.Errorf("encode initial World snapshot: %w", err)
	}
	finalBytes, err := world.EncodeSnapshot(final)
	if err != nil {
		return Bundle{}, fmt.Errorf("encode final World snapshot: %w", err)
	}
	transitionBytes, err := world.EncodeTransitions(delta)
	if err != nil {
		return Bundle{}, fmt.Errorf("encode World transitions: %w", err)
	}
	if uint64(len(transitionBytes)) > transitionLimit {
		return Bundle{}, fmt.Errorf("World transition payload requires %d bytes, limit is %d", len(transitionBytes), transitionLimit)
	}
	payloads := record.WorldPayloads{Initial: initialBytes, Transitions: transitionBytes, Final: finalBytes}
	manifest := record.World{
		Initial: record.WorldPayload{
			Schema: "gomadv3.world.snapshot/v1", File: "world/snapshot.json", RawSHA256: record.HashBytes(initialBytes),
			SemanticDigest: record.SHA256("sha256:" + string(initial.StateDigest)),
		},
		Transitions: record.WorldTransitions{
			Schema: "gomadv3.world.transitions/v1", File: "world/transitions.jsonl", RawSHA256: record.HashBytes(transitionBytes),
			Count: record.Uint64String(len(delta)), TranscriptDigest: record.SHA256("sha256:" + string(final.TranscriptDigest)),
		},
		Final: record.WorldPayload{
			Schema: "gomadv3.world.snapshot/v1", File: "world/final-snapshot.json", RawSHA256: record.HashBytes(finalBytes),
			SemanticDigest: record.SHA256("sha256:" + string(final.StateDigest)),
		},
		Adapters: []record.WorldAdapter{},
		Terminal: record.WorldTerminal{Kind: string(terminal.Kind), Detail: terminal.Detail},
	}
	if usesMailbox(initial) || usesMailbox(final) {
		initialMailbox, err := mailbox.DeriveSnapshot(initial)
		if err != nil {
			return Bundle{}, fmt.Errorf("derive initial mailbox snapshot: %w", err)
		}
		finalMailbox, err := mailbox.DeriveSnapshot(final)
		if err != nil {
			return Bundle{}, fmt.Errorf("derive final mailbox snapshot: %w", err)
		}
		manifest.Adapters = append(manifest.Adapters, record.WorldAdapter{
			Schema:        "gomadv3.world.adapter/mailbox/v1",
			InitialDigest: record.SHA256("sha256:" + string(initialMailbox.Digest)),
			FinalDigest:   record.SHA256("sha256:" + string(finalMailbox.Digest)),
		})
	}
	return Bundle{Manifest: manifest, Payloads: payloads}, nil
}

func validateTerminal(terminal world.Terminal) error {
	switch terminal.Kind {
	case world.TerminalNone, world.TerminalDelivered, world.TerminalIdle, world.TerminalDeadlock:
		if terminal.Detail != "" {
			return fmt.Errorf("World quiescence terminal has detail")
		}
	case world.TerminalCapacity, world.TerminalReplayDivergence, world.TerminalInvalidInput:
		if terminal.Detail == "" {
			return fmt.Errorf("World error terminal omitted detail")
		}
	default:
		return fmt.Errorf("invalid World terminal kind %q", terminal.Kind)
	}
	return nil
}

func usesMailbox(snapshot world.Snapshot) bool {
	for _, request := range snapshot.Requests {
		if request.Request.Resource.Adapter == "mailbox" {
			return true
		}
	}
	return false
}
