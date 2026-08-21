package campaign

import (
	"context"
	"errors"
	"fmt"

	umpire3runtime "go.temporal.io/server/tests/umpire3/execution"
	"go.temporal.io/server/tests/umpire3/protocol"
	"go.temporal.io/server/tests/umpire3/replay"
)

type ApprovedMutation struct {
	Identifier string       `json:"identifier"`
	Layer      string       `json:"layer"`
	Kind       MutationKind `json:"kind"`
	Path       string       `json:"path"`
}

type MutationGateRequest struct {
	Mutation       MutationRequest
	Approved       []ApprovedMutation
	MaxExecutions  int
	CorpusCoverage []CoveragePoint
	RiskFocus      []CoveragePoint
	Executor       ExecuteCandidate
}

type MutationGateReport struct {
	FormatVersion      string               `json:"formatVersion"`
	ResultClass        protocol.ResultClass `json:"resultClass"`
	TrustBadge         protocol.TrustBadge  `json:"trustBadge"`
	SourceDigest       string               `json:"sourceDigest"`
	ArtifactDigest     string               `json:"artifactDigest"`
	Seed               int64                `json:"seed"`
	ExecutionBudget    int                  `json:"executionBudget"`
	CandidateCount     int                  `json:"candidateCount"`
	BudgetDrops        int                  `json:"budgetDrops"`
	CoverageBefore     []CoveragePoint      `json:"coverageBefore"`
	CoverageDelta      []CoveragePoint      `json:"coverageDelta"`
	Examined           []string             `json:"examined"`
	Discovered         ApprovedMutation     `json:"discovered"`
	OriginalDigest     string               `json:"originalDigest"`
	Minimized          protocol.Experiment  `json:"minimized"`
	MinimizedDigest    string               `json:"minimizedDigest"`
	ReplayBundleDigest string               `json:"replayBundleDigest"`
	Replay             replay.Report        `json:"replay"`
	PromotionSource    string               `json:"promotionSource"`
}

func RunMutationGate(ctx context.Context, request MutationGateRequest) (MutationGateReport, error) {
	if request.Executor == nil || len(request.Approved) == 0 {
		return MutationGateReport{}, errors.New("mutation gate requires an executor and approved mutations")
	}
	approved := make(map[string]ApprovedMutation, len(request.Approved))
	for _, mutation := range request.Approved {
		if mutation.Identifier == "" || mutation.Layer == "" || mutation.Kind == "" || mutation.Path == "" {
			return MutationGateReport{}, errors.New("approved mutation identity, layer, kind, and path are required")
		}
		key := string(mutation.Kind) + "\x00" + mutation.Path
		if _, duplicate := approved[key]; duplicate {
			return MutationGateReport{}, fmt.Errorf("duplicate approved mutation %q", mutation.Identifier)
		}
		approved[key] = mutation
	}
	executionBudget := request.MaxExecutions
	if executionBudget == 0 {
		executionBudget = request.Mutation.MaxCandidates
	}
	canonical, err := Run(ctx, Request{
		Mutation: &request.Mutation, Workers: 1, MaxExecutions: executionBudget,
		MinimizeAttempts: max(64, request.Mutation.MaxCandidates*16),
		CorpusCoverage:   request.CorpusCoverage, RiskFocus: request.RiskFocus,
		Executor: func(ctx context.Context, experiment protocol.Experiment) (umpire3runtime.Result, []CoveragePoint, error) {
			result, err := request.Executor(ctx, experiment)
			return result, nil, err
		},
	})
	if err != nil {
		return MutationGateReport{}, err
	}
	report := MutationGateReport{
		Seed: request.Mutation.Seed, ExecutionBudget: executionBudget,
		CoverageBefore: canonical.CoverageBefore, CoverageDelta: canonical.CoverageDelta,
	}
	if canonical.Mutation != nil {
		report.CandidateCount = len(canonical.Mutation.Selected)
	}
	for _, dropped := range canonical.Dropped {
		if dropped.Reason == DropBudget {
			report.BudgetDrops++
		}
	}
	for _, execution := range canonical.Executions {
		report.Examined = append(report.Examined, mutationGateCandidateIdentity(execution))
	}
	for _, execution := range canonical.Executions {
		if execution.Result.Claim.Kind != umpire3runtime.ClaimViolating {
			continue
		}
		approvedMutation, known := approved[string(execution.Mutation)+"\x00"+execution.Path]
		if !known {
			return MutationGateReport{}, fmt.Errorf("unapproved mutation %s at %s produced a violation", execution.Mutation, execution.Path)
		}
		var discovery *Discovery
		for index := range canonical.Discoveries {
			candidate := &canonical.Discoveries[index]
			if candidate.Mutation == execution.Mutation && candidate.Path == execution.Path {
				discovery = candidate
				break
			}
		}
		if discovery == nil {
			return MutationGateReport{}, errors.New("qualified mutation has no canonical campaign discovery")
		}
		if discovery.PromotionBlock != "" {
			return MutationGateReport{}, fmt.Errorf("qualify mutation discovery: %s", discovery.PromotionBlock)
		}
		report.Discovered = approvedMutation
		report.OriginalDigest = execution.Digest
		report.Minimized = discovery.Minimized
		report.MinimizedDigest, err = discovery.Minimized.Digest()
		if err != nil {
			return MutationGateReport{}, err
		}
		report.ReplayBundleDigest = discovery.BundleDigest
		report.Replay = discovery.Replay
		report.PromotionSource = discovery.Promotion.Source
		return sealMutationGateReport(report, request.Mutation.Experiment)
	}
	if len(report.Examined) == 0 {
		return MutationGateReport{}, errors.New("campaign did not discover an approved cross-layer mutation")
	}
	return MutationGateReport{}, errors.New("campaign did not discover an approved cross-layer mutation")
}

func mutationGateCandidateIdentity(execution Execution) string {
	return string(execution.Mutation) + ":" + execution.Path + "@" + execution.Digest
}
