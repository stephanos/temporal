package campaign

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"

	"go.temporal.io/server/tests/umpire3/artifact"
	"go.temporal.io/server/tests/umpire3/protocol"
	"go.temporal.io/server/tests/umpire3/replay"
	umpire3runtime "go.temporal.io/server/tests/umpire3/runtime"
)

type ApprovedMutation struct {
	Identifier string       `json:"identifier"`
	Layer      string       `json:"layer"`
	Kind       MutationKind `json:"kind"`
	Path       string       `json:"path"`
}

type MutationGateRequest struct {
	Mutation MutationRequest
	Approved []ApprovedMutation
	Executor umpire3runtime.ExecuteCandidate
}

type MutationGateReport struct {
	Seed               int64               `json:"seed"`
	Examined           []string            `json:"examined"`
	Discovered         ApprovedMutation    `json:"discovered"`
	OriginalDigest     string              `json:"originalDigest"`
	Minimized          protocol.Experiment `json:"minimized"`
	MinimizedDigest    string              `json:"minimizedDigest"`
	ReplayBundleDigest string              `json:"replayBundleDigest"`
	Replay             replay.Report       `json:"replay"`
	PromotionSource    string              `json:"promotionSource"`
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
	mutations, err := Mutate(request.Mutation)
	if err != nil {
		return MutationGateReport{}, err
	}
	report := MutationGateReport{Seed: request.Mutation.Seed}
	var discovered Mutation
	found := false
	for _, mutation := range mutations.Selected {
		report.Examined = append(report.Examined, string(mutation.Kind)+":"+mutation.Path)
		result, executeErr := request.Executor(ctx, mutation.Experiment)
		if executeErr != nil {
			return MutationGateReport{}, fmt.Errorf("execute mutation %s at %s: %w", mutation.Kind, mutation.Path, executeErr)
		}
		approvedMutation, known := approved[string(mutation.Kind)+"\x00"+mutation.Path]
		if result.Claim.Kind != umpire3runtime.ClaimViolating {
			continue
		}
		if !known {
			return MutationGateReport{}, fmt.Errorf("unapproved mutation %s at %s produced a violation", mutation.Kind, mutation.Path)
		}
		report.Discovered = approvedMutation
		discovered = mutation
		found = true
		break
	}
	if !found {
		return MutationGateReport{}, errors.New("campaign did not discover an approved cross-layer mutation")
	}
	report.OriginalDigest = discovered.Digest
	minimized, err := umpire3runtime.MinimizeExperiment(ctx, discovered.Experiment, request.Executor)
	if err != nil {
		return MutationGateReport{}, fmt.Errorf("minimize discovered mutation: %w", err)
	}
	minimizedResult, err := request.Executor(ctx, minimized)
	if err != nil {
		return MutationGateReport{}, fmt.Errorf("execute minimized mutation: %w", err)
	}
	minimizedDigest, err := minimized.Digest()
	if err != nil {
		return MutationGateReport{}, err
	}
	encoded, err := artifact.Encode(minimized, minimizedResult, minimized.Retention.MaxArtifactBytes)
	if err != nil {
		return MutationGateReport{}, fmt.Errorf("encode minimized replay bundle: %w", err)
	}
	bundle, err := artifact.Decode(encoded, minimized.Retention.MaxArtifactBytes)
	if err != nil {
		return MutationGateReport{}, fmt.Errorf("decode minimized replay bundle: %w", err)
	}
	replayed, err := replay.Run(ctx, bundle, replay.Executor(request.Executor))
	if err != nil {
		return MutationGateReport{}, err
	}
	if !replayed.Reproduced {
		return MutationGateReport{}, errors.New("minimized replay did not reproduce the qualified violation")
	}
	promotion, err := promotionSource(minimized)
	if err != nil {
		return MutationGateReport{}, fmt.Errorf("promote minimized mutation: %w", err)
	}
	bundleHash := sha256.Sum256(encoded)
	report.Minimized = minimized
	report.MinimizedDigest = minimizedDigest
	report.ReplayBundleDigest = "sha256:" + hex.EncodeToString(bundleHash[:])
	report.Replay = replayed
	report.PromotionSource = promotion
	return report, nil
}
