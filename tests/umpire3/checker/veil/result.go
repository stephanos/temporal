package veil

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"

	protocolcatalog "go.temporal.io/server/tests/umpire3/protocol/catalog"
	protocolchecker "go.temporal.io/server/tests/umpire3/protocol/checker"
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

type concreteReceipt struct {
	Binding CompiledBinding `json:"binding"`
	Result  concreteResult  `json:"result"`
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
	view protocolchecker.FirstOrderView,
	binding BindingArtifact,
	reader io.Reader,
	limit int64,
	executionLimits protocolchecker.BackendExecutionLimits,
	receipt *protocolchecker.TraceReplayReceipt,
) (protocolchecker.BackendResult, error) {
	raw, err := readConcreteOutput(view, binding, reader, limit)
	if err != nil {
		return protocolchecker.BackendResult{}, err
	}
	bounds := protocolchecker.BackendBounds{ConcreteStateLimit: view.Bounds.ConcreteStateLimit}
	result := protocolchecker.BackendResult{
		FormatVersion:         protocolchecker.BackendResultFormatVersion,
		Backend:               protocolchecker.BackendVeil,
		BackendRevision:       protocolchecker.VeilBackendRevision,
		ViewFormatVersion:     view.FormatVersion,
		Target:                view.Target,
		Property:              view.Property,
		World:                 view.World,
		Variant:               view.Variant,
		SemanticHash:          view.SemanticHash,
		BindingArtifactDigest: binding.ArtifactDigest,
		Job:                   protocolchecker.BackendJobConcrete,
		Bounds:                bounds,
		ExecutionLimits:       executionLimits,
		ExploredStates:        raw.ExploredStates,
		Options:               []string{"sequential"},
		Axioms:                []string{},
		Omissions:             []string{},
	}

	switch raw.Result {
	case veilResultNoViolationFound:
		if err := validateNoViolationResult(raw, len(view.Oracle.States)); err != nil {
			return protocolchecker.BackendResult{}, err
		}
		result.ResultClass = protocolcatalog.ResultClassExternalNoCounterexample
		result.TrustBadge = protocolcatalog.TrustBadgeTestedInstance
		result.Termination = protocolchecker.BackendTerminationExhaustedInstance
		result.Omissions = []string{protocolchecker.VeilConcreteCollisionOmission}
	case veilResultFoundViolation:
		trace, err := normalizeViolationTrace(view, binding, raw)
		if err != nil {
			return protocolchecker.BackendResult{}, err
		}
		replayInput := traceReplayInput(view, trace)
		digest, err := replayInput.Digest()
		if err != nil {
			return protocolchecker.BackendResult{}, err
		}
		if receipt == nil || receipt.Validate() != nil {
			return protocolchecker.BackendResult{}, errors.New("trace witness requires accepted canonical replay")
		}
		if receipt.TraceDigest != digest {
			return protocolchecker.BackendResult{}, errors.New("canonical replay receipt is not bound to normalized trace")
		}
		trace.Bounds = bounds
		trace.Replay = protocolchecker.TraceReplayResult{
			TraceDigest: receipt.TraceDigest, Status: receipt.Status, TrustBadge: receipt.TrustBadge,
			Axioms: append([]string{}, receipt.Axioms...),
		}
		result.Axioms = append([]string{}, receipt.Axioms...)
		result.ResultClass = protocolcatalog.ResultClassTraceWitness
		result.TrustBadge = protocolcatalog.TrustBadgeCheckedCertificate
		result.Exact = true
		result.Termination = protocolchecker.BackendTerminationViolationFound
		result.Omissions = []string{protocolchecker.VeilTraceStateOmission}
		result.Trace = &trace
	default:
		return protocolchecker.BackendResult{}, fmt.Errorf("unknown Veil concrete result %q", raw.Result)
	}
	if err := result.Validate(); err != nil {
		return protocolchecker.BackendResult{}, err
	}
	return result, nil
}

func ConcreteReplayInput(
	view protocolchecker.FirstOrderView,
	binding BindingArtifact,
	reader io.Reader,
	limit int64,
) (*protocolchecker.TraceReplayInput, error) {
	raw, err := readConcreteOutput(view, binding, reader, limit)
	if err != nil {
		return nil, err
	}
	switch raw.Result {
	case veilResultNoViolationFound:
		if err := validateNoViolationResult(raw, len(view.Oracle.States)); err != nil {
			return nil, err
		}
		return nil, nil
	case veilResultFoundViolation:
		trace, err := normalizeViolationTrace(view, binding, raw)
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
	view protocolchecker.FirstOrderView,
	binding BindingArtifact,
	reader io.Reader,
	limit int64,
) (concreteResult, error) {
	if err := binding.ValidateAgainst(view); err != nil {
		return concreteResult{}, err
	}
	var receipt concreteReceipt
	if err := decodeConcreteOutput(reader, limit, &receipt); err != nil {
		return concreteResult{}, err
	}
	if err := receipt.Binding.Validate(); err != nil || !receipt.Binding.equal(binding.Binding) {
		return concreteResult{}, errors.New("veil concrete result is not bound to the compiled Veil binding")
	}
	raw := receipt.Result
	if raw.ExploredStates > view.Bounds.ConcreteStateLimit {
		return concreteResult{}, fmt.Errorf("veil explored %d states beyond the declared limit %d",
			raw.ExploredStates, view.Bounds.ConcreteStateLimit)
	}
	return raw, nil
}

func validateNoViolationResult(raw concreteResult, expectedStates int) error {
	if raw.ExploredStates != expectedStates || raw.TerminationReason == nil ||
		raw.TerminationReason.Kind != veilExploredAllStates || raw.StateFingerprint != "" ||
		raw.Trace != nil || raw.Violation != nil {
		return errors.New("invalid Veil no-violation result")
	}
	return nil
}

func traceReplayInput(view protocolchecker.FirstOrderView, trace protocolchecker.ModelTrace) protocolchecker.TraceReplayInput {
	actions := make([]protocolcatalog.ActionKind, len(trace.Steps))
	for index, step := range trace.Steps {
		actions[index] = step.Action
	}
	return protocolchecker.TraceReplayInput{
		FormatVersion: protocolchecker.TraceReplayInputFormatVersion,
		Target:        view.Target,
		Property:      view.Property,
		World:         view.World,
		Variant:       view.Variant,
		SemanticHash:  view.SemanticHash,
		Actions:       actions,
	}
}

func normalizeViolationTrace(
	view protocolchecker.FirstOrderView,
	binding BindingArtifact,
	raw concreteResult,
) (protocolchecker.ModelTrace, error) {
	if raw.StateFingerprint == "" || raw.Trace == nil || raw.Violation == nil ||
		raw.TerminationReason != nil || raw.ExploredStates < 0 {
		return protocolchecker.ModelTrace{}, errors.New("invalid Veil violation result")
	}
	if _, err := strconv.ParseUint(raw.StateFingerprint, 10, 64); err != nil {
		return protocolchecker.ModelTrace{}, fmt.Errorf("invalid Veil state fingerprint: %w", err)
	}
	if raw.Violation.Kind != veilSafetyFailure || len(raw.Violation.Violates) != 1 ||
		raw.Violation.Violates[0] != binding.Binding.PropertyLabel {
		return protocolchecker.ModelTrace{}, errors.New("veil violation does not match first-order property")
	}
	if len(raw.Trace.States) < 2 || raw.Trace.States[0].Index != 0 ||
		raw.Trace.States[0].Transition != veilAfterInit {
		return protocolchecker.ModelTrace{}, errors.New("veil violation trace requires an indexed initial state")
	}
	if raw.Trace.Theory != veilUnrepresentable {
		return protocolchecker.ModelTrace{}, errors.New("unexpected Veil trace state representation")
	}
	for _, state := range raw.Trace.States {
		if state.Fields != veilUnrepresentable {
			return protocolchecker.ModelTrace{}, errors.New("unexpected Veil trace state representation")
		}
	}
	backendActions := make(map[string]protocolcatalog.ActionKind, len(binding.Binding.ActionLabels))
	sourceMap := append([]protocolchecker.TraceSource(nil), binding.Binding.ActionLabels...)
	for _, label := range sourceMap {
		backendActions[label.BackendAction] = label.Action
	}
	steps := make([]protocolchecker.TraceStep, 0, len(raw.Trace.States)-1)
	for index, state := range raw.Trace.States[1:] {
		if state.Index != index+1 || state.Transition == veilAfterInit {
			return protocolchecker.ModelTrace{}, errors.New("veil violation trace indices and transitions must be ordered")
		}
		action, found := backendActions[state.Transition]
		if !found {
			return protocolchecker.ModelTrace{}, fmt.Errorf("unknown Veil transition %q", state.Transition)
		}
		steps = append(steps, protocolchecker.TraceStep{Action: action})
	}
	return protocolchecker.ModelTrace{
		World:       view.World,
		Steps:       steps,
		Property:    view.Property,
		Violation:   true,
		Assumptions: []string{},
		SourceMap:   sourceMap,
	}, nil
}

func decodeConcreteOutput(reader io.Reader, limit int64, destination *concreteReceipt) error {
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
