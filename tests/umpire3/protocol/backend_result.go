package protocol

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"slices"
)

const (
	BackendResultFormatVersion      = "umpire3/backend-result/v1"
	TraceReplayInputFormatVersion   = "umpire3/trace-replay-input/v1"
	TraceReplayReceiptFormatVersion = "umpire3/trace-replay-receipt/v1"
	VeilBackendRevision             = "300c305e945750ab3fb62de4a79c23161b24da39"
	VeilConcreteCollisionOmission   = "veil-concrete-fingerprint-collisions-not-ruled-out"
	VeilTraceStateOmission          = "veil-trace-state-values-unrepresentable"
)

type Backend string

const BackendVeil Backend = "veil"

type BackendJob string

const (
	BackendJobConcrete      BackendJob = "concrete"
	BackendJobSymbolicTrace BackendJob = "symbolic-trace"
	BackendJobInvariant     BackendJob = "invariant"
)

type BackendTermination string

const (
	BackendTerminationViolationFound    BackendTermination = "violation-found"
	BackendTerminationExhaustedInstance BackendTermination = "exhausted-instance"
	BackendTerminationBoundedSafe       BackendTermination = "bounded-safe"
	BackendTerminationGoalsClosed       BackendTermination = "goals-closed"
	BackendTerminationUnknown           BackendTermination = "unknown"
	BackendTerminationResourceLimit     BackendTermination = "resource-limit"
)

type TraceReplayStatus string

const (
	TraceReplayRequired TraceReplayStatus = "required"
	TraceReplayAccepted TraceReplayStatus = "accepted"
	TraceReplayRejected TraceReplayStatus = "rejected"
)

type BackendBounds struct {
	Depth              int `json:"depth,omitempty"`
	ConcreteStateLimit int `json:"concreteStateLimit,omitempty"`
}

type TraceStep struct {
	Action      ActionKind `json:"action"`
	StateDigest string     `json:"stateDigest,omitempty"`
}

type TraceSource struct {
	Action        ActionKind `json:"action"`
	BackendAction string     `json:"backendAction"`
}

type TraceReplayResult struct {
	TraceDigest string            `json:"traceDigest"`
	Status      TraceReplayStatus `json:"status"`
	TrustBadge  TrustBadge        `json:"trustBadge"`
	Axioms      []string          `json:"axioms"`
}

type TraceReplayInput struct {
	FormatVersion string       `json:"formatVersion"`
	Target        TargetID     `json:"target"`
	Property      PropertyID   `json:"property"`
	World         string       `json:"world"`
	Variant       string       `json:"variant"`
	SemanticHash  string       `json:"semanticHash"`
	Actions       []ActionKind `json:"actions"`
}

type TraceReplayReceipt struct {
	FormatVersion string            `json:"formatVersion"`
	TraceDigest   string            `json:"traceDigest"`
	Target        TargetID          `json:"target"`
	Property      PropertyID        `json:"property"`
	World         string            `json:"world"`
	Variant       string            `json:"variant"`
	SemanticHash  string            `json:"semanticHash"`
	Actions       []ActionKind      `json:"actions"`
	Status        TraceReplayStatus `json:"status"`
	TrustBadge    TrustBadge        `json:"trustBadge"`
	Axioms        []string          `json:"axioms"`
}

type ModelTrace struct {
	World              string            `json:"world"`
	InitialStateDigest string            `json:"initialStateDigest,omitempty"`
	Steps              []TraceStep       `json:"steps"`
	Property           PropertyID        `json:"property"`
	Violation          bool              `json:"violation"`
	Assumptions        []string          `json:"assumptions"`
	Bounds             BackendBounds     `json:"bounds"`
	SourceMap          []TraceSource     `json:"sourceMap"`
	Replay             TraceReplayResult `json:"replay"`
}

type BackendResult struct {
	FormatVersion     string             `json:"formatVersion"`
	Backend           Backend            `json:"backend"`
	BackendRevision   string             `json:"backendRevision"`
	ViewFormatVersion string             `json:"viewFormatVersion"`
	Target            TargetID           `json:"target"`
	Property          PropertyID         `json:"property"`
	World             string             `json:"world"`
	Variant           string             `json:"variant"`
	SemanticHash      string             `json:"semanticHash"`
	Job               BackendJob         `json:"job"`
	ResultClass       ResultClass        `json:"resultClass"`
	TrustBadge        TrustBadge         `json:"trustBadge"`
	Exact             bool               `json:"exact"`
	Termination       BackendTermination `json:"termination"`
	Bounds            BackendBounds      `json:"bounds"`
	ExploredStates    int                `json:"exploredStates,omitempty"`
	Options           []string           `json:"options"`
	Axioms            []string           `json:"axioms"`
	Omissions         []string           `json:"omissions"`
	Trace             *ModelTrace        `json:"trace,omitempty"`
}

func DecodeBackendResult(reader io.Reader, limit int64) (BackendResult, error) {
	var result BackendResult
	if err := decodeStrictJSON(reader, limit, "backend result", &result); err != nil {
		return BackendResult{}, err
	}
	if err := result.Validate(); err != nil {
		return BackendResult{}, err
	}
	return result, nil
}

func DecodeTraceReplayReceipt(reader io.Reader, limit int64) (TraceReplayReceipt, error) {
	var receipt TraceReplayReceipt
	if err := decodeStrictJSON(reader, limit, "trace replay receipt", &receipt); err != nil {
		return TraceReplayReceipt{}, err
	}
	if err := receipt.Validate(); err != nil {
		return TraceReplayReceipt{}, err
	}
	return receipt, nil
}

func (i TraceReplayInput) Validate() error {
	if i.FormatVersion != TraceReplayInputFormatVersion || i.Target == "" || i.Property == "" ||
		i.World == "" || i.Variant == "" || !validHash(i.SemanticHash) || len(i.Actions) == 0 {
		return errors.New("complete trace replay input identity, provenance, and actions are required")
	}
	if err := validateFirstOrderTarget(i.Target, i.Property); err != nil {
		return err
	}
	catalog, err := DefaultCatalog()
	if err != nil {
		return err
	}
	for _, action := range i.Actions {
		if _, found := catalog.Action(string(action)); !found {
			return fmt.Errorf("unknown trace replay action %q", action)
		}
	}
	return nil
}

func (i TraceReplayInput) CanonicalJSON() ([]byte, error) {
	if err := i.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(i)
}

func (i TraceReplayInput) Digest() (string, error) {
	encoded, err := i.CanonicalJSON()
	if err != nil {
		return "", err
	}
	return digestBytes(encoded), nil
}

func (r TraceReplayReceipt) Validate() error {
	if r.FormatVersion != TraceReplayReceiptFormatVersion || !validHash(r.TraceDigest) ||
		r.Status != TraceReplayAccepted || r.TrustBadge != TrustBadgeCheckedCertificate ||
		r.Axioms == nil {
		return errors.New("trace replay receipt requires an accepted checked-certificate bound digest")
	}
	input := TraceReplayInput{
		FormatVersion: TraceReplayInputFormatVersion,
		Target:        r.Target, Property: r.Property, World: r.World, Variant: r.Variant,
		SemanticHash: r.SemanticHash, Actions: append([]ActionKind(nil), r.Actions...),
	}
	digest, err := input.Digest()
	if err != nil || digest != r.TraceDigest {
		return errors.New("trace replay receipt digest does not match its checked trace")
	}
	return validateOrderedStrings("trace replay receipt axiom", r.Axioms)
}

func (r BackendResult) Validate() error {
	if r.FormatVersion != BackendResultFormatVersion || r.Backend != BackendVeil ||
		r.BackendRevision != VeilBackendRevision || r.ViewFormatVersion != FirstOrderViewFormatVersion ||
		r.Target == "" || r.Property == "" || r.World == "" || r.Variant == "" ||
		!validHash(r.SemanticHash) {
		return errors.New("complete pinned backend result identity and provenance are required")
	}
	if err := validateFirstOrderTarget(r.Target, r.Property); err != nil {
		return err
	}
	if !r.ResultClass.valid() || !r.TrustBadge.valid() {
		return errors.New("backend result requires a known result class and trust badge")
	}
	if r.Options == nil || r.Axioms == nil || r.Omissions == nil {
		return errors.New("backend options, axioms, and omissions must be explicit")
	}
	if err := validateOrderedStrings("backend option", r.Options); err != nil {
		return err
	}
	if err := validateOrderedStrings("backend axiom", r.Axioms); err != nil {
		return err
	}
	if err := validateOrderedStrings("backend omission", r.Omissions); err != nil {
		return err
	}
	if r.Job == BackendJobConcrete && (r.Bounds.ConcreteStateLimit <= 0 ||
		r.ExploredStates < 0 || r.ExploredStates > r.Bounds.ConcreteStateLimit) {
		return errors.New("concrete backend result exceeds or omits its declared state limit")
	}
	if r.Job == BackendJobConcrete && r.Termination == BackendTerminationExhaustedInstance &&
		r.ResultClass != ResultClassExternalNoCounterexample {
		return errors.New("concrete no-counterexample result cannot claim finite completeness")
	}

	switch r.ResultClass {
	case ResultClassTraceWitness:
		if !r.Exact || r.TrustBadge != TrustBadgeCheckedCertificate ||
			r.Termination != BackendTerminationViolationFound || r.Trace == nil {
			return errors.New("trace witness requires exact checked-certificate trust and a violation trace")
		}
		if err := r.Trace.validate(r); err != nil {
			return err
		}
	case ResultClassExternalNoCounterexample:
		if r.Job != BackendJobConcrete || r.Exact || r.TrustBadge != TrustBadgeTestedInstance ||
			r.Termination != BackendTerminationExhaustedInstance ||
			r.Bounds.Depth != 0 || r.Bounds.ConcreteStateLimit <= 0 || r.ExploredStates <= 0 ||
			r.Trace != nil || !slices.Contains(r.Omissions, VeilConcreteCollisionOmission) {
			return errors.New("concrete no-counterexample result must remain collision-qualified tested-instance evidence")
		}
	case ResultClassBoundedSafe:
		if r.Job != BackendJobSymbolicTrace || !r.Exact ||
			r.TrustBadge != TrustBadgeTrustedSolver ||
			r.Termination != BackendTerminationBoundedSafe || r.Bounds.Depth <= 0 || r.Trace != nil {
			return errors.New("bounded-safe result requires an exact symbolic bound and trusted-solver disclosure")
		}
	case ResultClassInvariantProved:
		if r.Job != BackendJobInvariant || !r.Exact ||
			(r.TrustBadge != TrustBadgeReconstructedSolverProof && r.TrustBadge != TrustBadgeTrustedSolver) ||
			r.Termination != BackendTerminationGoalsClosed || r.Trace != nil {
			return errors.New("invariant proof requires closed goals and disclosed solver trust")
		}
		if r.TrustBadge == TrustBadgeReconstructedSolverProof && slices.Contains(r.Axioms, "sorryAx") {
			return errors.New("reconstructed invariant proof cannot depend on sorryAx")
		}
		if r.TrustBadge == TrustBadgeTrustedSolver && !slices.Contains(r.Axioms, "sorryAx") {
			return errors.New("trusted Veil invariant proof must disclose sorryAx")
		}
	case ResultClassUnknown:
		if r.Exact || (r.Termination != BackendTerminationUnknown &&
			r.Termination != BackendTerminationResourceLimit) || r.Trace != nil {
			return errors.New("unknown backend result cannot claim exactness or carry a trace")
		}
	default:
		return fmt.Errorf("backend result class %q is not supported", r.ResultClass)
	}
	return nil
}

func (r BackendResult) CanonicalJSON() ([]byte, error) {
	if err := r.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(r)
}

func (t ModelTrace) validate(result BackendResult) error {
	if t.World != result.World || t.Property != result.Property || !t.Violation ||
		len(t.Steps) == 0 || t.Assumptions == nil || t.SourceMap == nil || t.Bounds != result.Bounds {
		return errors.New("trace identity, violation, assumptions, and bounds must match the backend result")
	}
	if t.Replay.Status != TraceReplayAccepted || t.Replay.TrustBadge != TrustBadgeCheckedCertificate ||
		!validHash(t.Replay.TraceDigest) || t.Replay.Axioms == nil {
		return errors.New("trace witness requires accepted canonical replay with checked-certificate trust")
	}
	actions := make([]ActionKind, len(t.Steps))
	for index, step := range t.Steps {
		actions[index] = step.Action
	}
	replayInput := TraceReplayInput{
		FormatVersion: TraceReplayInputFormatVersion,
		Target:        result.Target, Property: result.Property, World: result.World, Variant: result.Variant,
		SemanticHash: result.SemanticHash, Actions: actions,
	}
	digest, err := replayInput.Digest()
	if err != nil || digest != t.Replay.TraceDigest {
		return errors.New("trace witness replay digest does not match result identity and actions")
	}
	if !slices.Equal(result.Axioms, t.Replay.Axioms) {
		return errors.New("trace witness axiom inventory must match canonical replay")
	}
	if err := validateOrderedStrings("trace assumption", t.Assumptions); err != nil {
		return err
	}
	if err := validateOrderedStrings("trace replay axiom", t.Replay.Axioms); err != nil {
		return err
	}
	missingStateDigest := t.InitialStateDigest == ""
	for _, step := range t.Steps {
		missingStateDigest = missingStateDigest || step.StateDigest == ""
	}
	if missingStateDigest != slices.Contains(result.Omissions, VeilTraceStateOmission) {
		return errors.New("trace state digest omission must match unavailable state values")
	}
	catalog, err := DefaultCatalog()
	if err != nil {
		return err
	}
	sources := make(map[ActionKind]string, len(t.SourceMap))
	for _, source := range t.SourceMap {
		if source.Action == "" || source.BackendAction == "" {
			return errors.New("complete trace source mapping is required")
		}
		if _, known := catalog.Action(string(source.Action)); !known {
			return fmt.Errorf("unknown trace action %q", source.Action)
		}
		if _, duplicate := sources[source.Action]; duplicate {
			return fmt.Errorf("duplicate trace source mapping for %q", source.Action)
		}
		sources[source.Action] = source.BackendAction
	}
	for _, step := range t.Steps {
		if _, known := catalog.Action(string(step.Action)); !known {
			return fmt.Errorf("unknown trace action %q", step.Action)
		}
		if _, mapped := sources[step.Action]; !mapped {
			return fmt.Errorf("trace action %q has no backend source mapping", step.Action)
		}
		if step.StateDigest != "" && !validHash(step.StateDigest) {
			return fmt.Errorf("trace action %q has invalid state digest", step.Action)
		}
	}
	return nil
}

func validateOrderedStrings(kind string, values []string) error {
	for index, value := range values {
		if value == "" {
			return fmt.Errorf("%s cannot be empty", kind)
		}
		if index > 0 && values[index-1] >= value {
			return fmt.Errorf("%ss must be sorted and unique", kind)
		}
	}
	return nil
}
