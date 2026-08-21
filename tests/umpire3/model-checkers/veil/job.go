package veil

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"maps"
	"slices"

	"go.temporal.io/server/tests/umpire3/process"
	"go.temporal.io/server/tests/umpire3/protocol"
)

const veilJobReceiptFormatVersion = "umpire3/veil-job-receipt/v1"

type jobReceipt struct {
	FormatVersion      string                      `json:"formatVersion"`
	BackendRevision    string                      `json:"backendRevision"`
	ViewFormatVersion  string                      `json:"viewFormatVersion"`
	Target             protocol.TargetID           `json:"target"`
	Property           protocol.PropertyID         `json:"property"`
	World              string                      `json:"world"`
	Variant            string                      `json:"variant"`
	SemanticHash       string                      `json:"semanticHash"`
	GeneratedModelHash string                      `json:"generatedModelHash"`
	Job                protocol.BackendJob         `json:"job"`
	Status             protocol.BackendTermination `json:"status"`
	Depth              int                         `json:"depth,omitempty"`
	TrustBadge         protocol.TrustBadge         `json:"trustBadge"`
	Options            []string                    `json:"options"`
	Axioms             []string                    `json:"axioms"`
}

func RunJob(
	ctx context.Context,
	command []string,
	view protocol.FirstOrderView,
	generated GeneratedModule,
	job protocol.BackendJob,
) (protocol.BackendResult, error) {
	if len(command) == 0 {
		return protocol.BackendResult{}, errors.New("veil job receipt command is required")
	}
	if job != protocol.BackendJobSymbolicTrace && job != protocol.BackendJobInvariant {
		return protocol.BackendResult{}, fmt.Errorf("unsupported Veil job %q", job)
	}
	if err := validateInteractiveModule(view, generated); err != nil {
		return protocol.BackendResult{}, err
	}
	result, err := process.Run(ctx, process.Request{
		Command: append(append([]string(nil), command...), string(job)),
		Timeout: canonicalReplayTimeout, MaxOutputBytes: protocol.DefaultDecodeLimit,
		Limits: backendProcessLimits,
	})
	if err != nil {
		return protocol.BackendResult{}, fmt.Errorf("run Veil job receipt: %w", err)
	}
	return normalizeJobReceipt(view, generated, job, bytes.NewReader(result.Output),
		protocol.DefaultDecodeLimit)
}

func normalizeJobReceipt(
	view protocol.FirstOrderView,
	generated GeneratedModule,
	job protocol.BackendJob,
	reader io.Reader,
	limit int64,
) (protocol.BackendResult, error) {
	if err := validateInteractiveModule(view, generated); err != nil {
		return protocol.BackendResult{}, err
	}
	var receipt jobReceipt
	if err := decodeStrictJSON(reader, limit, "Veil job receipt", &receipt); err != nil {
		return protocol.BackendResult{}, err
	}
	if receipt.FormatVersion != veilJobReceiptFormatVersion ||
		receipt.BackendRevision != protocol.VeilBackendRevision ||
		receipt.ViewFormatVersion != view.FormatVersion || receipt.Target != view.Target ||
		receipt.Property != view.Property || receipt.World != view.World ||
		receipt.Variant != view.Variant || receipt.SemanticHash != view.SemanticHash {
		return protocol.BackendResult{}, errors.New("veil job receipt is not bound to the first-order view")
	}
	if receipt.GeneratedModelHash != generated.ModelHash {
		return protocol.BackendResult{}, errors.New("veil job receipt is not bound to the generated Veil module")
	}
	if receipt.Job != job {
		return protocol.BackendResult{}, errors.New("veil job receipt does not match requested job")
	}
	expectedTrust, expectedTrustOption, err := jobTrust(job, generated.TrustMode)
	if err != nil {
		return protocol.BackendResult{}, err
	}
	if receipt.TrustBadge != expectedTrust ||
		!slices.Equal(receipt.Options, []string{"grind+smt", "sequential", expectedTrustOption}) {
		return protocol.BackendResult{}, errors.New("veil job receipt does not match generated Veil trust mode")
	}
	if receipt.Axioms == nil || !slices.IsSorted(receipt.Axioms) ||
		len(slices.Compact(append([]string(nil), receipt.Axioms...))) != len(receipt.Axioms) {
		return protocol.BackendResult{}, errors.New("veil job receipt axiom inventory must be sorted and unique")
	}
	if receipt.Job == protocol.BackendJobSymbolicTrace && len(receipt.Axioms) != 0 {
		return protocol.BackendResult{}, errors.New("veil symbolic receipt cannot claim reconstructed proof axioms")
	}
	if receipt.Job == protocol.BackendJobInvariant {
		if generated.TrustMode == ReconstructedSMT && slices.Contains(receipt.Axioms, "sorryAx") {
			return protocol.BackendResult{}, errors.New("reconstructed Veil invariant contains sorryAx")
		}
		if generated.TrustMode == TrustedSMT && !slices.Contains(receipt.Axioms, "sorryAx") {
			return protocol.BackendResult{}, errors.New("trusted Veil invariant must disclose sorryAx")
		}
	}
	result := protocol.BackendResult{
		FormatVersion:           protocol.BackendResultFormatVersion,
		Backend:                 protocol.BackendVeil,
		BackendRevision:         protocol.VeilBackendRevision,
		ViewFormatVersion:       view.FormatVersion,
		Target:                  view.Target,
		Property:                view.Property,
		World:                   view.World,
		Variant:                 view.Variant,
		SemanticHash:            view.SemanticHash,
		GeneratedArtifactDigest: generated.ModelHash,
		Job:                     receipt.Job,
		TrustBadge:              receipt.TrustBadge,
		Exact:                   true,
		Termination:             receipt.Status,
		ExecutionLimits:         canonicalExecutionLimits(),
		Options:                 append([]string{}, receipt.Options...),
		Axioms:                  append([]string{}, receipt.Axioms...),
		Omissions:               []string{},
	}
	switch receipt.Job {
	case protocol.BackendJobSymbolicTrace:
		if receipt.Status != protocol.BackendTerminationBoundedSafe ||
			receipt.Depth != view.Bounds.SymbolicDepth {
			return protocol.BackendResult{}, errors.New("veil symbolic receipt does not match the first-order depth")
		}
		result.ResultClass = protocol.ResultClassBoundedSafe
		result.Bounds.Depth = receipt.Depth
	case protocol.BackendJobInvariant:
		if receipt.Status != protocol.BackendTerminationGoalsClosed || receipt.Depth != 0 {
			return protocol.BackendResult{}, errors.New("veil invariant receipt does not record closed unbounded goals")
		}
		result.ResultClass = protocol.ResultClassInvariantProved
	default:
		return protocol.BackendResult{}, fmt.Errorf("unknown Veil job receipt %q", receipt.Job)
	}
	if err := result.Validate(); err != nil {
		return protocol.BackendResult{}, err
	}
	return result, nil
}

func validateInteractiveModule(view protocol.FirstOrderView, generated GeneratedModule) error {
	expected, err := GenerateWithTrust(view, Interactive, generated.TrustMode)
	if err != nil {
		return err
	}
	if generated.Module != expected.Module || !bytes.Equal(generated.Source, expected.Source) ||
		!maps.Equal(generated.ActionLabels, expected.ActionLabels) || generated.ExportsModelChecker ||
		generated.TrustMode != expected.TrustMode {
		return errors.New("veil job receipt is not bound to the generated interactive module")
	}
	return nil
}

func jobTrust(job protocol.BackendJob, mode SMTTrustMode) (protocol.TrustBadge, string, error) {
	if job != protocol.BackendJobSymbolicTrace && job != protocol.BackendJobInvariant {
		return "", "", fmt.Errorf("veil job cannot record unsupported job %q", job)
	}
	switch mode {
	case ReconstructedSMT:
		if job == protocol.BackendJobSymbolicTrace {
			return protocol.TrustBadgeTrustedSolver, "smt-trust=false", nil
		}
		return protocol.TrustBadgeReconstructedSolverProof, "smt-trust=false", nil
	case TrustedSMT:
		return protocol.TrustBadgeTrustedSolver, "smt-trust=true", nil
	default:
		return "", "", fmt.Errorf("veil job cannot record trust mode %q", mode)
	}
}
