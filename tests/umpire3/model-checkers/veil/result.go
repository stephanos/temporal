package veil

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"strconv"

	"go.temporal.io/server/tests/umpire3/protocol"
)

const (
	veilResultNoViolationFound = "no_violation_found"
	veilResultFoundViolation   = "found_violation"
	veilAfterInit              = "after_init"
	veilSafetyFailure          = "safety_failure"
	veilExploredAllStates      = "explored_all_reachable_states"
	veilUnrepresentable        = "<unrepresentable>"
)

type concreteResult struct {
	ExploredStates    int                `json:"explored_states,omitempty"`
	Result            string             `json:"result"`
	StateFingerprint  string             `json:"state_fingerprint,omitempty"`
	Trace             *concreteTrace     `json:"trace,omitempty"`
	TerminationReason *terminationReason `json:"termination_reason,omitempty"`
	Violation         *concreteViolation `json:"violation,omitempty"`
}

type concreteTrace struct {
	States []concreteTraceState `json:"states"`
	Theory string               `json:"theory"`
}

type concreteTraceState struct {
	Fields     string `json:"fields"`
	Index      int    `json:"index"`
	Transition string `json:"transition"`
}

type terminationReason struct {
	Kind string `json:"kind"`
}

type concreteViolation struct {
	Kind     string   `json:"kind"`
	Violates []string `json:"violates"`
}

func NormalizeConcreteOutput(
	view protocol.FirstOrderView,
	generated GeneratedModule,
	reader io.Reader,
	limit int64,
	receipt *protocol.TraceReplayReceipt,
) (protocol.BackendResult, error) {
	raw, err := readConcreteOutput(view, generated, reader, limit)
	if err != nil {
		return protocol.BackendResult{}, err
	}
	bounds := protocol.BackendBounds{ConcreteStateLimit: view.Bounds.ConcreteStateLimit}
	result := protocol.BackendResult{
		FormatVersion:     protocol.BackendResultFormatVersion,
		Backend:           protocol.BackendVeil,
		BackendRevision:   protocol.VeilBackendRevision,
		ViewFormatVersion: view.FormatVersion,
		Target:            view.Target,
		Property:          view.Property,
		World:             view.World,
		Variant:           view.Variant,
		SemanticHash:      view.SemanticHash,
		Job:               protocol.BackendJobConcrete,
		Bounds:            bounds,
		ExploredStates:    raw.ExploredStates,
		Options:           []string{"sequential"},
		Axioms:            []string{},
		Omissions:         []string{},
	}

	switch raw.Result {
	case veilResultNoViolationFound:
		if err := validateNoViolationResult(raw); err != nil {
			return protocol.BackendResult{}, err
		}
		result.ResultClass = protocol.ResultClassExternalNoCounterexample
		result.TrustBadge = protocol.TrustBadgeTestedInstance
		result.Termination = protocol.BackendTerminationExhaustedInstance
		result.Omissions = []string{protocol.VeilConcreteCollisionOmission}
	case veilResultFoundViolation:
		trace, err := normalizeViolationTrace(view, generated, raw)
		if err != nil {
			return protocol.BackendResult{}, err
		}
		replayInput := traceReplayInput(view, trace)
		digest, err := replayInput.Digest()
		if err != nil {
			return protocol.BackendResult{}, err
		}
		if receipt == nil || receipt.Validate() != nil {
			return protocol.BackendResult{}, errors.New("trace witness requires accepted canonical replay")
		}
		if receipt.TraceDigest != digest {
			return protocol.BackendResult{}, errors.New("canonical replay receipt is not bound to normalized trace")
		}
		trace.Bounds = bounds
		trace.Replay = protocol.TraceReplayResult{
			Status: receipt.Status, TrustBadge: receipt.TrustBadge,
			Axioms: append([]string{}, receipt.Axioms...),
		}
		result.Axioms = append([]string{}, receipt.Axioms...)
		result.ResultClass = protocol.ResultClassTraceWitness
		result.TrustBadge = protocol.TrustBadgeCheckedCertificate
		result.Exact = true
		result.Termination = protocol.BackendTerminationViolationFound
		result.Omissions = []string{protocol.VeilTraceStateOmission}
		result.Trace = &trace
	default:
		return protocol.BackendResult{}, fmt.Errorf("unknown Veil concrete result %q", raw.Result)
	}
	if err := result.Validate(); err != nil {
		return protocol.BackendResult{}, err
	}
	return result, nil
}

func ConcreteReplayInput(
	view protocol.FirstOrderView,
	generated GeneratedModule,
	reader io.Reader,
	limit int64,
) (*protocol.TraceReplayInput, error) {
	raw, err := readConcreteOutput(view, generated, reader, limit)
	if err != nil {
		return nil, err
	}
	switch raw.Result {
	case veilResultNoViolationFound:
		if err := validateNoViolationResult(raw); err != nil {
			return nil, err
		}
		return nil, nil
	case veilResultFoundViolation:
		trace, err := normalizeViolationTrace(view, generated, raw)
		if err != nil {
			return nil, err
		}
		input := traceReplayInput(view, trace)
		if err := input.Validate(); err != nil {
			return nil, err
		}
		return &input, nil
	default:
		return nil, fmt.Errorf("unknown Veil concrete result %q", raw.Result)
	}
}

func readConcreteOutput(
	view protocol.FirstOrderView,
	generated GeneratedModule,
	reader io.Reader,
	limit int64,
) (concreteResult, error) {
	if err := validateConcreteModule(view, generated); err != nil {
		return concreteResult{}, err
	}
	var raw concreteResult
	if err := decodeConcreteOutput(reader, limit, &raw); err != nil {
		return concreteResult{}, err
	}
	return raw, nil
}

func validateNoViolationResult(raw concreteResult) error {
	if raw.ExploredStates <= 0 || raw.TerminationReason == nil ||
		raw.TerminationReason.Kind != veilExploredAllStates || raw.StateFingerprint != "" ||
		raw.Trace != nil || raw.Violation != nil {
		return errors.New("invalid Veil no-violation result")
	}
	return nil
}

func traceReplayInput(view protocol.FirstOrderView, trace protocol.ModelTrace) protocol.TraceReplayInput {
	actions := make([]protocol.ActionKind, len(trace.Steps))
	for index, step := range trace.Steps {
		actions[index] = step.Action
	}
	return protocol.TraceReplayInput{
		FormatVersion: protocol.TraceReplayInputFormatVersion,
		Target:        view.Target,
		Property:      view.Property,
		World:         view.World,
		Variant:       view.Variant,
		SemanticHash:  view.SemanticHash,
		Actions:       actions,
	}
}

func validateConcreteModule(view protocol.FirstOrderView, generated GeneratedModule) error {
	expected, err := Generate(view, Concrete)
	if err != nil {
		return err
	}
	if generated.Module != expected.Module || !bytes.Equal(generated.Source, expected.Source) ||
		!maps.Equal(generated.ActionLabels, expected.ActionLabels) ||
		generated.ExportsModelChecker != expected.ExportsModelChecker ||
		generated.TrustMode != expected.TrustMode {
		return errors.New("Veil concrete output is not bound to the generated first-order module")
	}
	return nil
}

func normalizeViolationTrace(
	view protocol.FirstOrderView,
	generated GeneratedModule,
	raw concreteResult,
) (protocol.ModelTrace, error) {
	if raw.StateFingerprint == "" || raw.Trace == nil || raw.Violation == nil ||
		raw.TerminationReason != nil || raw.ExploredStates < 0 {
		return protocol.ModelTrace{}, errors.New("invalid Veil violation result")
	}
	if _, err := strconv.ParseUint(raw.StateFingerprint, 10, 64); err != nil {
		return protocol.ModelTrace{}, fmt.Errorf("invalid Veil state fingerprint: %w", err)
	}
	if raw.Violation.Kind != veilSafetyFailure || len(raw.Violation.Violates) != 1 ||
		raw.Violation.Violates[0] != exportedIdentifier(string(view.Property)) {
		return protocol.ModelTrace{}, errors.New("Veil violation does not match first-order property")
	}
	if len(raw.Trace.States) < 2 || raw.Trace.States[0].Index != 0 ||
		raw.Trace.States[0].Transition != veilAfterInit {
		return protocol.ModelTrace{}, errors.New("Veil violation trace requires an indexed initial state")
	}
	if raw.Trace.Theory != veilUnrepresentable {
		return protocol.ModelTrace{}, errors.New("unexpected Veil trace state representation")
	}
	for _, state := range raw.Trace.States {
		if state.Fields != veilUnrepresentable {
			return protocol.ModelTrace{}, errors.New("unexpected Veil trace state representation")
		}
	}
	backendActions := make(map[string]protocol.ActionKind, len(generated.ActionLabels))
	sourceMap := make([]protocol.TraceSource, 0, len(view.Actions))
	for _, action := range view.Actions {
		backendAction, found := generated.ActionLabels[action.Identifier]
		if !found || backendAction == "" {
			return protocol.ModelTrace{}, fmt.Errorf("Veil source map omits action %q", action.Identifier)
		}
		if _, duplicate := backendActions[backendAction]; duplicate {
			return protocol.ModelTrace{}, fmt.Errorf("duplicate Veil transition %q", backendAction)
		}
		canonical := protocol.ActionKind(action.Identifier)
		backendActions[backendAction] = canonical
		sourceMap = append(sourceMap, protocol.TraceSource{Action: canonical, BackendAction: backendAction})
	}
	steps := make([]protocol.TraceStep, 0, len(raw.Trace.States)-1)
	for index, state := range raw.Trace.States[1:] {
		if state.Index != index+1 || state.Transition == veilAfterInit {
			return protocol.ModelTrace{}, errors.New("Veil violation trace indices and transitions must be ordered")
		}
		action, found := backendActions[state.Transition]
		if !found {
			return protocol.ModelTrace{}, fmt.Errorf("unknown Veil transition %q", state.Transition)
		}
		steps = append(steps, protocol.TraceStep{Action: action})
	}
	return protocol.ModelTrace{
		World:       view.World,
		Steps:       steps,
		Property:    view.Property,
		Violation:   true,
		Assumptions: []string{},
		SourceMap:   sourceMap,
	}, nil
}

func decodeConcreteOutput(reader io.Reader, limit int64, destination *concreteResult) error {
	return decodeStrictJSON(reader, limit, "Veil concrete output", destination)
}

func decodeStrictJSON(reader io.Reader, limit int64, kind string, destination any) error {
	if limit <= 0 {
		return fmt.Errorf("%s decode limit must be positive", kind)
	}
	encoded, err := io.ReadAll(io.LimitReader(reader, limit+1))
	if err != nil {
		return fmt.Errorf("read %s: %w", kind, err)
	}
	if int64(len(encoded)) > limit {
		return fmt.Errorf("%s exceeds %d-byte decode limit", kind, limit)
	}
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return fmt.Errorf("decode %s: %w", kind, err)
	}
	var extra json.RawMessage
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return fmt.Errorf("decode %s: multiple JSON values", kind)
		}
		return fmt.Errorf("decode %s trailer: %w", kind, err)
	}
	return nil
}
