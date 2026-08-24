package campaign

import (
	"errors"

	"go.temporal.io/server/tools/gomad3/deterministicio"
	"go.temporal.io/server/tools/gomad3/record"
	"go.temporal.io/server/tools/gomad3/world"
)

const artifactManifestCapacity = uint64(16 << 20)

type ArtifactCapacityPlan struct {
	FailureArtifacts record.Uint64String    `json:"failure_artifacts"`
	FailureBytes     record.Uint64String    `json:"failure_bytes"`
	SuccessArtifacts record.Uint64String    `json:"success_artifacts"`
	SuccessBytes     record.Uint64String    `json:"success_bytes"`
	TotalBytes       record.Uint64String    `json:"total_bytes"`
	TranscriptBytes  record.Uint64String    `json:"transcript_bytes"`
	FailureOutcome   JournalCapacityOutcome `json:"failure_outcome"`
	SuccessOutcome   JournalCapacityOutcome `json:"success_outcome"`
}

func DeriveArtifactCapacityPlan(plan CampaignPlan) (ArtifactCapacityPlan, error) {
	maximumRuns := uint64(plan.SelectionCount)
	if plan.Strategy == "choice-exploration" || plan.Strategy == "simulation-exploration" {
		maximumRuns = uint64(plan.MaxExecutions)
	}
	choiceBytes := uint64(0)
	if plan.ChoiceProfile != nil {
		choiceBytes = uint64(plan.ChoiceProfile.Limit)
	}
	mountReferences, err := checkedArtifactCapacityAdd(uint64(plan.IOROMountLimits.Requests), uint64(plan.IOROMountLimits.Files), uint64(plan.IOROMountLimits.DirectoryEntries))
	if err != nil {
		return ArtifactCapacityPlan{}, err
	}
	mountDescriptorBytes, err := checkedArtifactCapacityMultiply(uint64(plan.IOROMountLimits.PathBytes), mountReferences)
	if err != nil {
		return ArtifactCapacityPlan{}, err
	}
	perArtifact := uint64(plan.Prepared.Target.Size)
	values := []uint64{
		uint64(plan.Prepared.Target.Size), uint64(plan.OutputBytes), uint64(plan.OutputBytes), deterministicio.MaximumTranscriptBytes,
		uint64(plan.IOROMountLimits.TotalBytes), mountDescriptorBytes, 2 * world.MaximumSnapshotJSONBytes,
		uint64(plan.WorldTransitionBytes), choiceBytes, artifactManifestCapacity,
	}
	for _, value := range values {
		if value > ^uint64(0)-perArtifact {
			return ArtifactCapacityPlan{}, errors.New("per-artifact capacity overflows")
		}
		perArtifact += value
	}
	if maximumRuns != 0 && perArtifact > ^uint64(0)/maximumRuns {
		return ArtifactCapacityPlan{}, errors.New("failure artifact capacity overflows")
	}
	failureBytes := perArtifact * maximumRuns
	successBytes := uint64(plan.SuccessBytesLimit)
	if successBytes > ^uint64(0)-failureBytes {
		return ArtifactCapacityPlan{}, errors.New("aggregate artifact capacity overflows")
	}
	return ArtifactCapacityPlan{
		FailureArtifacts: record.Uint64String(maximumRuns), FailureBytes: record.Uint64String(failureBytes),
		SuccessArtifacts: plan.SuccessArtifactLimit, SuccessBytes: plan.SuccessBytesLimit,
		TotalBytes: record.Uint64String(failureBytes + successBytes), TranscriptBytes: deterministicio.MaximumTranscriptBytes,
		FailureOutcome: CapacityInfrastructureFailure, SuccessOutcome: CapacityInfrastructureFailure,
	}, nil
}

func checkedArtifactCapacityAdd(values ...uint64) (uint64, error) {
	var total uint64
	for _, value := range values {
		if value > ^uint64(0)-total {
			return 0, errors.New("artifact capacity overflows")
		}
		total += value
	}
	return total, nil
}

func checkedArtifactCapacityMultiply(left, right uint64) (uint64, error) {
	if left != 0 && right > ^uint64(0)/left {
		return 0, errors.New("artifact capacity overflows")
	}
	return left * right, nil
}
