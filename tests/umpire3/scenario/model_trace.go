package scenario

import (
	"errors"
	"fmt"
	"strings"

	"go.temporal.io/server/tests/umpire3/protocol"
)

func FromSemanticTrace(identifier string, trace protocol.SemanticTrace) (Scenario, error) {
	if identifier == "" {
		return Scenario{}, errors.New("semantic trace scenario identifier is required")
	}
	if err := trace.Validate(); err != nil {
		return Scenario{}, fmt.Errorf("validate semantic trace: %w", err)
	}
	resources := make([]Resource, len(trace.Resources))
	for index, resource := range trace.Resources {
		resources[index] = Resource{Identifier: resource.Identifier, Kind: resource.Kind}
	}
	actions := make([]Term, 0, len(trace.Steps))
	for index, step := range trace.Steps {
		if step.Action == "" {
			continue
		}
		options := []ActionOption{}
		if step.Outcome != "" {
			options = append(options, WithOutcomes(step.Outcome))
		}
		actions = append(actions,
			Action(fmt.Sprintf("semantic-trace-step-%03d", index+1), step.Action, options...))
	}
	if len(actions) == 0 {
		return Scenario{}, errors.New("semantic trace has no executable non-stutter actions")
	}
	body := OnePath(actions...)
	if trace.Kind == protocol.SemanticTraceFinite {
		view, found, err := protocol.DefaultFirstOrderView(trace.Target, trace.Variant)
		if err != nil {
			return Scenario{}, fmt.Errorf("load semantic trace executable view: %w", err)
		}
		if !found {
			return Scenario{}, fmt.Errorf("no executable view for semantic trace %q/%q",
				trace.Target, trace.Variant)
		}
		for index := len(view.ActivatingFaults) - 1; index >= 0; index-- {
			body = During(Fault(fmt.Sprintf("semantic-trace-fault-%03d", index+1),
				view.ActivatingFaults[index]), body)
		}
	}
	return Scenario{
		Identifier: identifier,
		Target:     trace.Target,
		Resources:  resources,
		Root:       OnePath(body, Require(trace.Property)),
	}, nil
}

func SemanticTraceIdentifier(trace protocol.SemanticTrace) string {
	return traceIdentifier(trace.Replay.Digest)
}

func FromBackendResult(identifier string, result protocol.BackendResult) (Scenario, error) {
	trace, err := protocol.SemanticTraceFromBackendResult(result)
	if err != nil {
		return Scenario{}, fmt.Errorf("validate model-checker result: %w", err)
	}
	return FromSemanticTrace(identifier, trace)
}

func FromTraceReplayReceipt(identifier string, receipt protocol.TraceReplayReceipt) (Scenario, error) {
	trace, err := protocol.SemanticTraceFromTraceReplayReceipt(
		protocol.SemanticTraceProducerExact, receipt)
	if err != nil {
		return Scenario{}, fmt.Errorf(
			"checked trace receipt does not match generated executable view: %w", err)
	}
	return FromSemanticTrace(identifier, trace)
}

func ModelTraceIdentifier(result protocol.BackendResult) string {
	trace, err := protocol.SemanticTraceFromBackendResult(result)
	if err != nil {
		return traceIdentifier("")
	}
	return SemanticTraceIdentifier(trace)
}

func TraceReceiptIdentifier(receipt protocol.TraceReplayReceipt) string {
	return traceIdentifier(receipt.TraceDigest)
}

func FromTemporalLassoReplayReceipt(
	identifier string,
	receipt protocol.TemporalLassoReplayReceipt,
) (Scenario, error) {
	trace, err := protocol.SemanticTraceFromTemporalLassoReplayReceipt(
		protocol.SemanticTraceProducerLeanTemporal, receipt)
	if err != nil {
		return Scenario{}, fmt.Errorf("checked temporal lasso receipt does not match generated view: %w", err)
	}
	return FromSemanticTrace(identifier, trace)
}

func TemporalLassoIdentifier(receipt protocol.TemporalLassoReplayReceipt) string {
	return traceIdentifier(receipt.LassoDigest)
}

func traceIdentifier(traceDigest string) string {
	digest := strings.TrimPrefix(traceDigest, "sha256:")
	if digest == "" {
		digest = "trace"
	}
	if len(digest) > 16 {
		digest = digest[:16]
	}
	return "model-trace-" + digest
}
