package checker

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"slices"
)

const (
	SemanticTraceFormatVersion        = "umpire3/semantic-trace/v1"
	liveTraceReplayInputFormatVersion = "umpire3/live-trace-replay-input/v1"
)

type SemanticTraceKind string

const (
	SemanticTraceFinite   SemanticTraceKind = "finite"
	SemanticTraceTemporal SemanticTraceKind = "temporal"
	SemanticTraceLive     SemanticTraceKind = "live"
)

type SemanticTraceProducer string

const (
	SemanticTraceProducerExact        SemanticTraceProducer = "exact"
	SemanticTraceProducerNative       SemanticTraceProducer = "native"
	SemanticTraceProducerVeil         SemanticTraceProducer = "veil"
	SemanticTraceProducerLeanTemporal SemanticTraceProducer = "lean-temporal"
	SemanticTraceProducerLive         SemanticTraceProducer = "live"
)

type SemanticTraceStep struct {
	Action  ActionKind    `json:"action"`
	Outcome ActionOutcome `json:"outcome,omitempty"`
}

type SemanticTraceBinding struct {
	Declaration string     `json:"declaration"`
	Axioms      []string   `json:"axioms"`
	TrustBadge  TrustBadge `json:"trustBadge"`
}

type SemanticTraceReplay struct {
	Digest     string            `json:"digest"`
	Status     TraceReplayStatus `json:"status"`
	TrustBadge TrustBadge        `json:"trustBadge"`
	Axioms     []string          `json:"axioms"`
}

type SemanticTrace struct {
	FormatVersion    string                `json:"formatVersion"`
	Kind             SemanticTraceKind     `json:"kind"`
	Producer         SemanticTraceProducer `json:"producer"`
	Target           TargetID              `json:"target"`
	Property         PropertyID            `json:"property"`
	World            string                `json:"world"`
	Variant          string                `json:"variant"`
	SemanticHash     string                `json:"semanticHash"`
	ExperimentDigest string                `json:"experimentDigest,omitempty"`
	Experiment       *Experiment           `json:"experiment,omitempty"`
	Resources        []FirstOrderResource  `json:"resources"`
	Steps            []SemanticTraceStep   `json:"steps"`
	States           []string              `json:"states"`
	LoopStart        int                   `json:"loopStart"`
	Binding          SemanticTraceBinding  `json:"binding"`
	Replay           SemanticTraceReplay   `json:"replay"`
	Omissions        []string              `json:"omissions"`
}

type liveTraceReplayInput struct {
	FormatVersion    string               `json:"formatVersion"`
	Target           TargetID             `json:"target"`
	Property         PropertyID           `json:"property"`
	World            string               `json:"world"`
	Variant          string               `json:"variant"`
	SemanticHash     string               `json:"semanticHash"`
	ExperimentDigest string               `json:"experimentDigest"`
	Resources        []FirstOrderResource `json:"resources"`
	Steps            []SemanticTraceStep  `json:"steps"`
}

func DecodeSemanticTrace(reader io.Reader, limit int64) (SemanticTrace, error) {
	var trace SemanticTrace
	if err := decodeStrictJSON(reader, limit, "semantic trace", &trace); err != nil {
		return SemanticTrace{}, err
	}
	if err := trace.Validate(); err != nil {
		return SemanticTrace{}, err
	}
	return trace, nil
}

func (t SemanticTrace) Validate() error {
	if t.FormatVersion != SemanticTraceFormatVersion || t.Target == "" || t.Property == "" ||
		t.World == "" || t.Variant == "" || !validHash(t.SemanticHash) ||
		t.Resources == nil || len(t.Resources) == 0 || t.Steps == nil || len(t.Steps) == 0 ||
		t.States == nil || t.Omissions == nil {
		return errors.New("complete semantic trace identity, scope, steps, and omissions are required")
	}
	if err := t.Binding.validate(); err != nil {
		return err
	}
	if t.Replay.Status != TraceReplayAccepted || t.Replay.TrustBadge != TrustBadgeCheckedCertificate ||
		!validHash(t.Replay.Digest) || t.Replay.Axioms == nil {
		return errors.New("semantic trace requires an accepted checked replay")
	}
	if err := validateOrderedStrings("semantic trace replay axiom", t.Replay.Axioms); err != nil {
		return err
	}
	if err := validateOrderedStrings("semantic trace omission", t.Omissions); err != nil {
		return err
	}
	if err := validateTraceResources(t.Resources); err != nil {
		return err
	}
	switch t.Kind {
	case SemanticTraceFinite:
		return t.validateFinite()
	case SemanticTraceTemporal:
		return t.validateTemporal()
	case SemanticTraceLive:
		return t.validateLive()
	default:
		return fmt.Errorf("unknown semantic trace kind %q", t.Kind)
	}
}

func (t SemanticTrace) CanonicalJSON() ([]byte, error) {
	if err := t.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(t)
}

func (t SemanticTrace) validateFinite() error {
	if t.Producer != SemanticTraceProducerExact && t.Producer != SemanticTraceProducerNative &&
		t.Producer != SemanticTraceProducerVeil {
		return fmt.Errorf("finite semantic trace has incompatible producer %q", t.Producer)
	}
	if t.ExperimentDigest != "" || t.Experiment != nil || len(t.States) != 0 || t.LoopStart != -1 {
		return errors.New("finite semantic trace cannot carry live or lasso state")
	}
	actions := make([]ActionKind, len(t.Steps))
	for index, step := range t.Steps {
		if step.Action == "" || step.Outcome != "" {
			return fmt.Errorf("finite semantic trace step %d must contain only an action", index)
		}
		actions[index] = step.Action
	}
	receipt := TraceReplayReceipt{
		FormatVersion: TraceReplayReceiptFormatVersion,
		TraceDigest:   t.Replay.Digest, Target: t.Target, Property: t.Property,
		World: t.World, Variant: t.Variant, SemanticHash: t.SemanticHash,
		Actions: actions, Status: t.Replay.Status, TrustBadge: t.Replay.TrustBadge,
		Axioms: append([]string{}, t.Replay.Axioms...),
	}
	return receipt.Validate()
}

func (t SemanticTrace) validateTemporal() error {
	if t.Producer != SemanticTraceProducerLeanTemporal || t.ExperimentDigest != "" || t.Experiment != nil ||
		len(t.States) != len(t.Steps) {
		return errors.New("temporal semantic trace requires a Lean temporal lasso")
	}
	actions := make([]ActionKind, len(t.Steps))
	for index, step := range t.Steps {
		if step.Outcome != "" {
			return fmt.Errorf("temporal semantic trace step %d cannot carry a live outcome", index)
		}
		actions[index] = step.Action
	}
	receipt := TemporalLassoReplayReceipt{
		FormatVersion: TemporalLassoReplayReceiptFormatVersion,
		LassoDigest:   t.Replay.Digest, Target: t.Target, Property: t.Property,
		World: t.World, Variant: t.Variant, SemanticHash: t.SemanticHash,
		Lasso: TemporalLasso{
			States: append([]string{}, t.States...), Actions: actions, LoopStart: t.LoopStart,
		},
		Status: t.Replay.Status, TrustBadge: t.Replay.TrustBadge,
		Axioms: append([]string{}, t.Replay.Axioms...),
	}
	return receipt.Validate()
}

func (t SemanticTrace) validateLive() error {
	if t.Producer != SemanticTraceProducerLive || !validHash(t.ExperimentDigest) || t.Experiment == nil ||
		len(t.States) != 0 || t.LoopStart != -1 {
		return errors.New("live semantic trace requires experiment identity and no lasso state")
	}
	for index, step := range t.Steps {
		if step.Action == "" || !validActionOutcome(step.Outcome) {
			return fmt.Errorf("live semantic trace step %d requires an action and observed outcome", index)
		}
	}
	if err := t.Experiment.Validate(); err != nil {
		return fmt.Errorf("validate live semantic trace experiment: %w", err)
	}
	experimentDigest, err := t.Experiment.Digest()
	if err != nil || experimentDigest != t.ExperimentDigest {
		return errors.New("live semantic trace experiment digest does not match its source")
	}
	if len(t.Steps) > len(t.Experiment.Actions) {
		return errors.New("live semantic trace has more attempts than its source experiment")
	}
	for index, step := range t.Steps {
		if string(step.Action) != t.Experiment.Actions[index].Kind ||
			!slices.Contains(t.Experiment.Actions[index].AllowedOutcomes, step.Outcome) {
			return fmt.Errorf("live semantic trace step %d does not match its source experiment", index)
		}
	}
	experimentResources := make([]FirstOrderResource, len(t.Experiment.Resources))
	for index, resource := range t.Experiment.Resources {
		experimentResources[index] = FirstOrderResource{
			Identifier: resource.Identifier, Kind: EntityKind(resource.Kind),
		}
	}
	if !slices.Equal(experimentResources, t.Resources) {
		return errors.New("live semantic trace resources do not match its source experiment")
	}
	digest, err := t.liveDigest()
	if err != nil || digest != t.Replay.Digest {
		return errors.New("live semantic trace replay digest does not match its observed attempts")
	}
	if !slices.Equal(t.Replay.Axioms, t.Binding.Axioms) {
		return errors.New("live semantic trace replay axioms do not match its model binding")
	}
	return nil
}

func (t SemanticTrace) liveDigest() (string, error) {
	input := liveTraceReplayInput{
		FormatVersion: liveTraceReplayInputFormatVersion,
		Target:        t.Target, Property: t.Property, World: t.World, Variant: t.Variant,
		SemanticHash: t.SemanticHash, ExperimentDigest: t.ExperimentDigest,
		Resources: append([]FirstOrderResource{}, t.Resources...),
		Steps:     append([]SemanticTraceStep{}, t.Steps...),
	}
	encoded, err := json.Marshal(input)
	if err != nil {
		return "", err
	}
	return digestBytes(encoded), nil
}

func (b SemanticTraceBinding) validate() error {
	if b.Declaration == "" || b.Axioms == nil || !b.TrustBadge.Valid() {
		return errors.New("semantic trace requires a model binding and axiom inventory")
	}
	if err := validateOrderedStrings("semantic trace binding axiom", b.Axioms); err != nil {
		return err
	}
	expected := TrustBadgeKernel
	if len(b.Axioms) != 0 {
		expected = TrustBadgeKernelWithDeclaredAxioms
	}
	if b.TrustBadge != expected {
		return errors.New("semantic trace binding trust does not match its axiom inventory")
	}
	return nil
}

func validateTraceResources(resources []FirstOrderResource) error {
	catalog, err := DefaultCatalog()
	if err != nil {
		return err
	}
	knownKinds := make(map[EntityKind]struct{}, len(catalog.Entities))
	for _, entity := range catalog.Entities {
		knownKinds[EntityKind(entity.Identifier)] = struct{}{}
	}
	seen := make(map[string]struct{}, len(resources))
	for _, resource := range resources {
		if resource.Identifier == "" {
			return errors.New("semantic trace resource identifier is required")
		}
		if _, known := knownKinds[resource.Kind]; !known {
			return fmt.Errorf("semantic trace resource %q has unknown kind %q",
				resource.Identifier, resource.Kind)
		}
		if _, duplicate := seen[resource.Identifier]; duplicate {
			return fmt.Errorf("semantic trace has duplicate resource %q", resource.Identifier)
		}
		seen[resource.Identifier] = struct{}{}
	}
	return nil
}
