package trace

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"slices"

	"go.temporal.io/server/tools/umpire3/checker/finite"
	protocolcatalog "go.temporal.io/server/tools/umpire3/protocol/catalog"
	protocolchecker "go.temporal.io/server/tools/umpire3/protocol/checker"
	protocolexperiment "go.temporal.io/server/tools/umpire3/protocol/experiment"
)

const liveTraceReplayInputFormatVersion = "umpire3/live-trace-replay-input/v1"

func Validate(trace protocolchecker.SemanticTrace) error {
	if err := trace.Validate(); err != nil {
		return err
	}
	switch trace.Kind {
	case protocolchecker.SemanticTraceFinite:
		return validateFinite(trace)
	case protocolchecker.SemanticTraceTemporal:
		return validateTemporal(trace)
	case protocolchecker.SemanticTraceLive:
		return validateLive(trace)
	default:
		return fmt.Errorf("unknown semantic trace kind %q", trace.Kind)
	}
}

func validateFinite(trace protocolchecker.SemanticTrace) error {
	view, found, err := finite.DefaultFirstOrderView(trace.Target, trace.Variant)
	if err != nil {
		return err
	}
	if !found || view.Property != trace.Property || view.World != trace.World ||
		view.SemanticHash != trace.SemanticHash || !slices.Equal(view.Resources, trace.Resources) ||
		!bindingsEqual(trace.Binding, bindingFromRelation(view.Relation)) {
		return errors.New("finite semantic trace does not match its generated exact view")
	}
	return nil
}

func validateTemporal(trace protocolchecker.SemanticTrace) error {
	view, found, err := protocolchecker.DefaultTemporalView(trace.Variant)
	if err != nil {
		return err
	}
	if !found || view.Target != trace.Target || view.Property != trace.Property ||
		view.World != trace.World || view.SemanticHash != trace.SemanticHash ||
		!slices.Equal(view.Resources, trace.Resources) ||
		!bindingsEqual(trace.Binding, bindingFromRelation(view.Relation)) {
		return errors.New("temporal semantic trace does not match its generated view")
	}
	return nil
}

func validateLive(trace protocolchecker.SemanticTrace) error {
	view, found, err := finite.DefaultAttemptExecutionView(*trace.Experiment)
	if err != nil {
		return err
	}
	if !found || view.Target() != trace.Target || view.Property() != trace.Property ||
		view.Variant() != trace.Variant || view.World() != trace.World ||
		view.SemanticHash() != trace.SemanticHash ||
		!bindingsEqual(trace.Binding, bindingFromAttemptExecutionView(view)) {
		return errors.New("live semantic trace does not match its generated attempt view")
	}
	attempts := make([]finite.ObservedAttempt, len(trace.Steps))
	for index, step := range trace.Steps {
		attempts[index] = finite.ObservedAttempt{Action: step.Action, Outcome: step.Outcome}
	}
	replayed, err := view.ReplayObserved(attempts)
	if err != nil {
		return err
	}
	if !replayed.Accepted {
		return fmt.Errorf("live semantic trace rejects action %q outcome %q",
			replayed.RejectedAction, replayed.RejectedOutcome)
	}
	return nil
}

func FromBackendResult(result protocolchecker.BackendResult) (protocolchecker.SemanticTrace, error) {
	if err := result.Validate(); err != nil {
		return protocolchecker.SemanticTrace{}, err
	}
	if result.ResultClass != protocolcatalog.ResultClassTraceWitness || result.Trace == nil {
		return protocolchecker.SemanticTrace{}, errors.New("backend result is not a checked trace witness")
	}
	actions := make([]protocolcatalog.ActionKind, len(result.Trace.Steps))
	for index, step := range result.Trace.Steps {
		actions[index] = step.Action
	}
	trace, err := FromTraceReplayReceipt(protocolchecker.SemanticTraceProducerVeil, protocolchecker.TraceReplayReceipt{
		FormatVersion: protocolchecker.TraceReplayReceiptFormatVersion,
		TraceDigest:   result.Trace.Replay.TraceDigest,
		Target:        result.Target, Property: result.Property, World: result.World,
		Variant: result.Variant, SemanticHash: result.SemanticHash, Actions: actions,
		Status: result.Trace.Replay.Status, TrustBadge: result.Trace.Replay.TrustBadge,
		Axioms: append([]string{}, result.Trace.Replay.Axioms...),
	})
	if err != nil {
		return protocolchecker.SemanticTrace{}, err
	}
	trace.Omissions = append([]string{}, result.Omissions...)
	if err := Validate(trace); err != nil {
		return protocolchecker.SemanticTrace{}, err
	}
	return trace, nil
}

func FromTraceReplayReceipt(
	producer protocolchecker.SemanticTraceProducer,
	receipt protocolchecker.TraceReplayReceipt,
) (protocolchecker.SemanticTrace, error) {
	if producer != protocolchecker.SemanticTraceProducerExact && producer != protocolchecker.SemanticTraceProducerNative &&
		producer != protocolchecker.SemanticTraceProducerVeil {
		return protocolchecker.SemanticTrace{}, fmt.Errorf("finite semantic trace has incompatible producer %q", producer)
	}
	if err := receipt.Validate(); err != nil {
		return protocolchecker.SemanticTrace{}, err
	}
	view, found, err := finite.DefaultFirstOrderView(receipt.Target, receipt.Variant)
	if err != nil {
		return protocolchecker.SemanticTrace{}, err
	}
	if !found || view.Property != receipt.Property || view.World != receipt.World ||
		view.SemanticHash != receipt.SemanticHash {
		return protocolchecker.SemanticTrace{}, errors.New("finite semantic trace does not match a generated exact view")
	}
	steps := make([]protocolchecker.SemanticTraceStep, len(receipt.Actions))
	for index, action := range receipt.Actions {
		steps[index] = protocolchecker.SemanticTraceStep{Action: action}
	}
	trace := protocolchecker.SemanticTrace{
		FormatVersion: protocolchecker.SemanticTraceFormatVersion,
		Kind:          protocolchecker.SemanticTraceFinite, Producer: producer,
		Target: receipt.Target, Property: receipt.Property, World: receipt.World,
		Variant: receipt.Variant, SemanticHash: receipt.SemanticHash,
		Resources: append([]protocolchecker.FirstOrderResource{}, view.Resources...),
		Steps:     steps, States: []string{}, LoopStart: -1,
		Binding: bindingFromRelation(view.Relation),
		Replay: protocolchecker.SemanticTraceReplay{
			Digest: receipt.TraceDigest, Status: receipt.Status, TrustBadge: receipt.TrustBadge,
			Axioms: append([]string{}, receipt.Axioms...),
		},
		Omissions: []string{},
	}
	if err := Validate(trace); err != nil {
		return protocolchecker.SemanticTrace{}, err
	}
	return trace, nil
}

func FromTemporalLassoReplayReceipt(
	producer protocolchecker.SemanticTraceProducer,
	receipt protocolchecker.TemporalLassoReplayReceipt,
) (protocolchecker.SemanticTrace, error) {
	if producer != protocolchecker.SemanticTraceProducerLeanTemporal {
		return protocolchecker.SemanticTrace{}, fmt.Errorf("temporal semantic trace has incompatible producer %q", producer)
	}
	if err := receipt.Validate(); err != nil {
		return protocolchecker.SemanticTrace{}, err
	}
	view, found, err := protocolchecker.DefaultTemporalView(receipt.Variant)
	if err != nil {
		return protocolchecker.SemanticTrace{}, err
	}
	if !found || view.Target != receipt.Target || view.Property != receipt.Property ||
		view.World != receipt.World || view.SemanticHash != receipt.SemanticHash {
		return protocolchecker.SemanticTrace{}, errors.New("temporal semantic trace does not match a generated view")
	}
	steps := make([]protocolchecker.SemanticTraceStep, len(receipt.Lasso.Actions))
	for index, action := range receipt.Lasso.Actions {
		steps[index] = protocolchecker.SemanticTraceStep{Action: action}
	}
	trace := protocolchecker.SemanticTrace{
		FormatVersion: protocolchecker.SemanticTraceFormatVersion,
		Kind:          protocolchecker.SemanticTraceTemporal, Producer: producer,
		Target: receipt.Target, Property: receipt.Property, World: receipt.World,
		Variant: receipt.Variant, SemanticHash: receipt.SemanticHash,
		Resources: append([]protocolchecker.FirstOrderResource{}, view.Resources...),
		Steps:     steps, States: append([]string{}, receipt.Lasso.States...),
		LoopStart: receipt.Lasso.LoopStart,
		Binding:   bindingFromRelation(view.Relation),
		Replay: protocolchecker.SemanticTraceReplay{
			Digest: receipt.LassoDigest, Status: receipt.Status, TrustBadge: receipt.TrustBadge,
			Axioms: append([]string{}, receipt.Axioms...),
		},
		Omissions: []string{},
	}
	if err := Validate(trace); err != nil {
		return protocolchecker.SemanticTrace{}, err
	}
	return trace, nil
}

func NewLive(
	experiment protocolexperiment.Experiment,
	view finite.AttemptExecutionView,
	attempts []finite.ObservedAttempt,
) (protocolchecker.SemanticTrace, error) {
	if err := experiment.Validate(); err != nil {
		return protocolchecker.SemanticTrace{}, err
	}
	expected, found, err := finite.DefaultAttemptExecutionView(experiment)
	if err != nil {
		return protocolchecker.SemanticTrace{}, err
	}
	if !found || expected.Target() != view.Target() || expected.Property() != view.Property() ||
		expected.World() != view.World() || expected.Variant() != view.Variant() ||
		expected.SemanticHash() != view.SemanticHash() {
		return protocolchecker.SemanticTrace{}, errors.New("live semantic trace view does not match the experiment")
	}
	replayed, err := view.ReplayObserved(attempts)
	if err != nil {
		return protocolchecker.SemanticTrace{}, err
	}
	if !replayed.Accepted {
		return protocolchecker.SemanticTrace{}, fmt.Errorf("live semantic trace rejects action %q outcome %q",
			replayed.RejectedAction, replayed.RejectedOutcome)
	}
	experimentDigest, err := experiment.Digest()
	if err != nil {
		return protocolchecker.SemanticTrace{}, err
	}
	resources := make([]protocolchecker.FirstOrderResource, len(experiment.Resources))
	for index, resource := range experiment.Resources {
		resources[index] = protocolchecker.FirstOrderResource{
			Identifier: resource.Identifier, Kind: protocolcatalog.EntityKind(resource.Kind),
		}
	}
	steps := make([]protocolchecker.SemanticTraceStep, len(attempts))
	for index, attempt := range attempts {
		steps[index] = protocolchecker.SemanticTraceStep{Action: attempt.Action, Outcome: attempt.Outcome}
	}
	encodedExperiment, err := experiment.CanonicalJSON()
	if err != nil {
		return protocolchecker.SemanticTrace{}, err
	}
	clonedExperiment, err := protocolexperiment.DecodeExperiment(bytes.NewReader(encodedExperiment), protocolexperiment.DefaultDecodeLimit)
	if err != nil {
		return protocolchecker.SemanticTrace{}, err
	}
	trace := protocolchecker.SemanticTrace{
		FormatVersion: protocolchecker.SemanticTraceFormatVersion,
		Kind:          protocolchecker.SemanticTraceLive, Producer: protocolchecker.SemanticTraceProducerLive,
		Target: view.Target(), Property: view.Property(), World: view.World(),
		Variant: view.Variant(), SemanticHash: view.SemanticHash(),
		ExperimentDigest: experimentDigest, Experiment: &clonedExperiment, Resources: resources,
		Steps: steps, States: []string{}, LoopStart: -1,
		Binding:   bindingFromAttemptExecutionView(view),
		Omissions: []string{},
	}
	digest, err := liveDigest(trace)
	if err != nil {
		return protocolchecker.SemanticTrace{}, err
	}
	trace.Replay = protocolchecker.SemanticTraceReplay{
		Digest: digest, Status: protocolchecker.TraceReplayAccepted, TrustBadge: protocolcatalog.TrustBadgeCheckedCertificate,
		Axioms: append([]string{}, trace.Binding.Axioms...),
	}
	if err := Validate(trace); err != nil {
		return protocolchecker.SemanticTrace{}, err
	}
	return trace, nil
}

func bindingFromRelation(relation protocolchecker.FirstOrderRelation) protocolchecker.SemanticTraceBinding {
	return newBinding(relation.Declaration, relation.Axioms, relation.TrustBadge)
}

func bindingFromAttemptExecutionView(view finite.AttemptExecutionView) protocolchecker.SemanticTraceBinding {
	if view.Finite != nil {
		return newBinding(
			view.Finite.Relation.Declaration,
			view.Finite.Relation.Axioms,
			view.Finite.Relation.TrustBadge,
		)
	}
	return bindingFromRelation(view.Attempts.Relation)
}

func newBinding(
	declaration string,
	axiomInventory []string,
	trustBadge protocolcatalog.TrustBadge,
) protocolchecker.SemanticTraceBinding {
	axioms := append([]string{}, axiomInventory...)
	slices.Sort(axioms)
	return protocolchecker.SemanticTraceBinding{
		Declaration: declaration,
		Axioms:      axioms,
		TrustBadge:  trustBadge,
	}
}

func bindingsEqual(left, right protocolchecker.SemanticTraceBinding) bool {
	return left.Declaration == right.Declaration && left.TrustBadge == right.TrustBadge &&
		slices.Equal(left.Axioms, right.Axioms)
}

func liveDigest(trace protocolchecker.SemanticTrace) (string, error) {
	input := struct {
		FormatVersion    string                               `json:"formatVersion"`
		Target           protocolcatalog.TargetID             `json:"target"`
		Property         protocolcatalog.PropertyID           `json:"property"`
		World            string                               `json:"world"`
		Variant          string                               `json:"variant"`
		SemanticHash     string                               `json:"semanticHash"`
		ExperimentDigest string                               `json:"experimentDigest"`
		Resources        []protocolchecker.FirstOrderResource `json:"resources"`
		Steps            []protocolchecker.SemanticTraceStep  `json:"steps"`
	}{
		FormatVersion: liveTraceReplayInputFormatVersion,
		Target:        trace.Target, Property: trace.Property, World: trace.World, Variant: trace.Variant,
		SemanticHash: trace.SemanticHash, ExperimentDigest: trace.ExperimentDigest,
		Resources: append([]protocolchecker.FirstOrderResource{}, trace.Resources...),
		Steps:     append([]protocolchecker.SemanticTraceStep{}, trace.Steps...),
	}
	encoded, err := json.Marshal(input)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(encoded)
	return "sha256:" + hex.EncodeToString(digest[:]), nil
}
