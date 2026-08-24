package veil

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"slices"

	"go.temporal.io/server/tools/umpire3/internal/subprocess"
	protocolcatalog "go.temporal.io/server/tools/umpire3/protocol/catalog"
	protocolchecker "go.temporal.io/server/tools/umpire3/protocol/checker"
	protocolexperiment "go.temporal.io/server/tools/umpire3/protocol/experiment"
)

const veilJobReceiptFormatVersion = "umpire3/veil-job-receipt/v1"

type jobReceipt struct {
	FormatVersion   string                             `json:"formatVersion"`
	BackendRevision string                             `json:"backendRevision"`
	Binding         CompiledBinding                    `json:"binding"`
	Job             protocolchecker.BackendJob         `json:"job"`
	Status          protocolchecker.BackendTermination `json:"status"`
	Depth           int                                `json:"depth,omitempty"`
	TrustBadge      protocolcatalog.TrustBadge         `json:"trustBadge"`
	Options         []string                           `json:"options"`
	Axioms          []string                           `json:"axioms"`
}

func RunJob(
	ctx context.Context,
	command []string,
	view protocolchecker.FirstOrderView,
	binding BindingArtifact,
	job protocolchecker.BackendJob,
) (protocolchecker.BackendResult, error) {
	if len(command) == 0 {
		return protocolchecker.BackendResult{}, errors.New("veil job receipt command is required")
	}
	if job != protocolchecker.BackendJobSymbolicTrace && job != protocolchecker.BackendJobInvariant {
		return protocolchecker.BackendResult{}, fmt.Errorf("unsupported Veil job %q", job)
	}
	if err := binding.ValidateAgainst(view); err != nil {
		return protocolchecker.BackendResult{}, err
	}
	result, err := subprocess.Run(ctx, subprocess.Request{
		Command: append(append([]string(nil), command...), view.SemanticHash, string(job)),
		Timeout: canonicalReplayTimeout, MaxOutputBytes: protocolexperiment.DefaultDecodeLimit,
		Limits: backendProcessLimits,
	})
	if err != nil {
		return protocolchecker.BackendResult{}, fmt.Errorf("run Veil job receipt: %w", err)
	}
	return normalizeJobReceipt(view, binding, job, bytes.NewReader(result.Output),
		protocolexperiment.DefaultDecodeLimit)
}

func normalizeJobReceipt(
	view protocolchecker.FirstOrderView,
	binding BindingArtifact,
	job protocolchecker.BackendJob,
	reader io.Reader,
	limit int64,
) (protocolchecker.BackendResult, error) {
	if err := binding.ValidateAgainst(view); err != nil {
		return protocolchecker.BackendResult{}, err
	}
	var receipt jobReceipt
	if err := decodeStrictJSON(reader, limit, "Veil job receipt", &receipt); err != nil {
		return protocolchecker.BackendResult{}, err
	}
	if receipt.FormatVersion != veilJobReceiptFormatVersion ||
		receipt.BackendRevision != protocolchecker.VeilBackendRevision {
		return protocolchecker.BackendResult{}, errors.New("veil job receipt has an unsupported identity")
	}
	if err := receipt.Binding.Validate(); err != nil || !receipt.Binding.equal(binding.Binding) {
		return protocolchecker.BackendResult{}, errors.New("veil job receipt is not bound to the compiled Veil binding")
	}
	if receipt.Job != job {
		return protocolchecker.BackendResult{}, errors.New("veil job receipt does not match requested job")
	}
	expectedTrust, expectedTrustOption, err := jobTrust(job, binding.Binding.TrustMode)
	if err != nil {
		return protocolchecker.BackendResult{}, err
	}
	if receipt.TrustBadge != expectedTrust ||
		!slices.Equal(receipt.Options, []string{"grind+smt", "sequential", expectedTrustOption}) {
		return protocolchecker.BackendResult{}, errors.New("veil job receipt does not match the compiled Veil trust mode")
	}
	if receipt.Axioms == nil || !slices.IsSorted(receipt.Axioms) ||
		len(slices.Compact(append([]string(nil), receipt.Axioms...))) != len(receipt.Axioms) {
		return protocolchecker.BackendResult{}, errors.New("veil job receipt axiom inventory must be sorted and unique")
	}
	if receipt.Job == protocolchecker.BackendJobSymbolicTrace && len(receipt.Axioms) != 0 {
		return protocolchecker.BackendResult{}, errors.New("veil symbolic receipt cannot claim reconstructed proof axioms")
	}
	if receipt.Job == protocolchecker.BackendJobInvariant {
		if binding.Binding.TrustMode == ReconstructedSMT && slices.Contains(receipt.Axioms, "sorryAx") {
			return protocolchecker.BackendResult{}, errors.New("reconstructed Veil invariant contains sorryAx")
		}
		if binding.Binding.TrustMode == TrustedSMT && !slices.Contains(receipt.Axioms, "sorryAx") {
			return protocolchecker.BackendResult{}, errors.New("trusted Veil invariant must disclose sorryAx")
		}
	}
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
		Job:                   receipt.Job,
		TrustBadge:            receipt.TrustBadge,
		Exact:                 true,
		Termination:           receipt.Status,
		ExecutionLimits:       canonicalExecutionLimits(),
		Options:               append([]string{}, receipt.Options...),
		Axioms:                append([]string{}, receipt.Axioms...),
		Omissions:             []string{},
	}
	switch receipt.Job {
	case protocolchecker.BackendJobSymbolicTrace:
		if receipt.Status != protocolchecker.BackendTerminationBoundedSafe ||
			receipt.Depth != view.Bounds.SymbolicDepth {
			return protocolchecker.BackendResult{}, errors.New("veil symbolic receipt does not match the first-order depth")
		}
		result.ResultClass = protocolcatalog.ResultClassBoundedSafe
		result.Bounds.Depth = receipt.Depth
	case protocolchecker.BackendJobInvariant:
		if receipt.Status != protocolchecker.BackendTerminationGoalsClosed || receipt.Depth != 0 {
			return protocolchecker.BackendResult{}, errors.New("veil invariant receipt does not record closed unbounded goals")
		}
		result.ResultClass = protocolcatalog.ResultClassInvariantProved
	default:
		return protocolchecker.BackendResult{}, fmt.Errorf("unknown Veil job receipt %q", receipt.Job)
	}
	if err := result.Validate(); err != nil {
		return protocolchecker.BackendResult{}, err
	}
	return result, nil
}

func jobTrust(job protocolchecker.BackendJob, mode SMTTrustMode) (protocolcatalog.TrustBadge, string, error) {
	if job != protocolchecker.BackendJobSymbolicTrace && job != protocolchecker.BackendJobInvariant {
		return "", "", fmt.Errorf("veil job cannot record unsupported job %q", job)
	}
	switch mode {
	case ReconstructedSMT:
		if job == protocolchecker.BackendJobSymbolicTrace {
			return protocolcatalog.TrustBadgeTrustedSolver, "smt-trust=false", nil
		}
		return protocolcatalog.TrustBadgeReconstructedSolverProof, "smt-trust=false", nil
	case TrustedSMT:
		return protocolcatalog.TrustBadgeTrustedSolver, "smt-trust=true", nil
	default:
		return "", "", fmt.Errorf("veil job cannot record trust mode %q", mode)
	}
}
