package scenario

import (
	"errors"
	"fmt"
	"strings"

	"go.temporal.io/server/tests/umpire3/protocol"
)

func FromBackendResult(identifier string, result protocol.BackendResult) (Scenario, error) {
	if identifier == "" {
		return Scenario{}, errors.New("model trace scenario identifier is required")
	}
	if err := result.Validate(); err != nil {
		return Scenario{}, fmt.Errorf("validate model-checker result: %w", err)
	}
	if result.ResultClass != protocol.ResultClassTraceWitness || result.Trace == nil {
		return Scenario{}, errors.New("model-checker result is not a checked trace witness")
	}
	actions := make([]protocol.ActionKind, len(result.Trace.Steps))
	for index, step := range result.Trace.Steps {
		actions[index] = step.Action
	}
	return FromTraceReplayReceipt(identifier, protocol.TraceReplayReceipt{
		FormatVersion: protocol.TraceReplayReceiptFormatVersion,
		TraceDigest:   result.Trace.Replay.TraceDigest,
		Target:        result.Target,
		Property:      result.Property,
		World:         result.World,
		Variant:       result.Variant,
		SemanticHash:  result.SemanticHash,
		Actions:       actions,
		Status:        result.Trace.Replay.Status,
		TrustBadge:    result.Trace.Replay.TrustBadge,
		Axioms:        append([]string(nil), result.Trace.Replay.Axioms...),
	})
}

func FromTraceReplayReceipt(identifier string, receipt protocol.TraceReplayReceipt) (Scenario, error) {
	if identifier == "" {
		return Scenario{}, errors.New("model trace scenario identifier is required")
	}
	if err := receipt.Validate(); err != nil {
		return Scenario{}, fmt.Errorf("validate checked trace receipt: %w", err)
	}
	view, found, err := protocol.DefaultFirstOrderView(receipt.Target, receipt.Variant)
	if err != nil {
		return Scenario{}, fmt.Errorf("load trace executable view: %w", err)
	}
	if !found {
		return Scenario{}, fmt.Errorf("no executable view for model trace %q/%q", receipt.Target, receipt.Variant)
	}
	if receipt.Property != view.Property || receipt.World != view.World || receipt.SemanticHash != view.SemanticHash {
		return Scenario{}, errors.New("checked trace receipt does not match generated executable view")
	}
	resources := make([]Resource, len(view.Resources))
	for index, resource := range view.Resources {
		resources[index] = Resource{Identifier: resource.Identifier, Kind: resource.Kind}
	}
	actions := make([]Term, len(receipt.Actions))
	for index, action := range receipt.Actions {
		actions[index] = Action(fmt.Sprintf("model-step-%03d", index+1), action)
	}
	body := OnePath(actions...)
	for index := len(view.ActivatingFaults) - 1; index >= 0; index-- {
		body = During(Fault(fmt.Sprintf("model-fault-%03d", index+1), view.ActivatingFaults[index]), body)
	}
	return Scenario{
		Identifier: identifier,
		Target:     receipt.Target,
		Resources:  resources,
		Root:       OnePath(body, Require(receipt.Property)),
	}, nil
}

func ModelTraceIdentifier(result protocol.BackendResult) string {
	if result.Trace == nil {
		return traceIdentifier("")
	}
	return traceIdentifier(result.Trace.Replay.TraceDigest)
}

func TraceReceiptIdentifier(receipt protocol.TraceReplayReceipt) string {
	return traceIdentifier(receipt.TraceDigest)
}

func FromTemporalLassoReplayReceipt(
	identifier string,
	receipt protocol.TemporalLassoReplayReceipt,
) (Scenario, error) {
	if identifier == "" {
		return Scenario{}, errors.New("temporal lasso scenario identifier is required")
	}
	if err := receipt.Validate(); err != nil {
		return Scenario{}, fmt.Errorf("validate checked temporal lasso receipt: %w", err)
	}
	view, found, err := protocol.DefaultTemporalView(receipt.Variant)
	if err != nil {
		return Scenario{}, fmt.Errorf("load temporal lasso view: %w", err)
	}
	if !found || view.Target != receipt.Target || view.Property != receipt.Property ||
		view.World != receipt.World || view.SemanticHash != receipt.SemanticHash {
		return Scenario{}, errors.New("checked temporal lasso receipt does not match generated view")
	}
	resources := make([]Resource, len(view.Resources))
	for index, resource := range view.Resources {
		resources[index] = Resource{Identifier: resource.Identifier, Kind: resource.Kind}
	}
	actions := make([]Term, 0, len(receipt.Lasso.Actions))
	for index, action := range receipt.Lasso.Actions {
		if action == "" {
			continue
		}
		actions = append(actions, Action(fmt.Sprintf("lasso-step-%03d", index+1), action))
	}
	if len(actions) == 0 {
		return Scenario{}, errors.New("temporal lasso has no executable non-stutter actions")
	}
	return Scenario{
		Identifier: identifier,
		Target:     receipt.Target,
		Resources:  resources,
		Root:       OnePath(OnePath(actions...), Require(receipt.Property)),
	}, nil
}

func TemporalLassoIdentifier(receipt protocol.TemporalLassoReplayReceipt) string {
	return "temporal-" + traceIdentifier(receipt.LassoDigest)
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
