package inspect

import (
	"path/filepath"

	"go.temporal.io/server/tools/gomad/internal/evidence/record"
	"go.temporal.io/server/tools/gomad/internal/runner/internal/campaign"
	"go.temporal.io/server/tools/gomad/internal/runner/internal/corpus"
	"go.temporal.io/server/tools/gomad/internal/runner/internal/exploration"
	"go.temporal.io/server/tools/gomad/internal/runner/internal/inspection"
)

func projectCommittedCampaign(committed campaign.CommittedLog) CommittedCampaignInspection {
	frontier := committed.Frontier
	result := CommittedCampaignInspection{
		Outcomes: committed.Outcomes, OutcomeBytes: committed.Bytes, Limits: projectExecutionJournalLimits(committed.Limits),
		Admitted: make([]AdmittedOrdinalInspection, len(frontier.Outstanding)), Available: frontier.Available,
		PendingStop: string(frontier.PendingStop), Stopped: string(frontier.Stopped),
	}
	for index, job := range frontier.Outstanding {
		admitted := AdmittedOrdinalInspection{Ordinal: job.Ordinal, Seed: job.Seed}
		if receipt, received := frontier.Receipts[job.Ordinal]; received {
			admitted.Receipt = &AdmissionReceiptInspection{
				Domain: receipt.Observation.Domain, Reason: receipt.Observation.Reason, Termination: receipt.Observation.Termination,
				Artifacts: append([]string{}, receipt.Artifacts...), Diagnostics: uint64(len(receipt.Diagnostics)),
			}
		}
		result.Admitted[index] = admitted
	}
	return result
}

func projectCorpus(root string, snapshot corpus.Snapshot) CorpusInspection {
	identity := snapshot.Identity
	result := CorpusInspection{
		Schema: snapshot.Schema, Generation: uint64(snapshot.Generation), TargetSHA256: identity.TargetSHA256, EnvironmentSHA256: identity.EnvironmentSHA256,
		ToolchainBuildKey: identity.Toolchain.BuildKey, InstrumentationSchema: identity.InstrumentationSchema,
		CoverageSHA256: snapshot.CoverageSHA256, SnapshotSHA256: snapshot.SnapshotSHA256, Entries: uint64(len(snapshot.Entries)),
		MaximumEntries: corpus.MaximumEntries, MaximumBytes: corpus.MaximumBytes, Cases: make([]CorpusCaseInspection, len(snapshot.Entries)),
	}
	for index, entry := range snapshot.Entries {
		reasons := make([]CorpusFeatureInspection, len(entry.NoveltyReasons))
		for reasonIndex, reason := range entry.NoveltyReasons {
			reasons[reasonIndex] = CorpusFeatureInspection{Kind: reason.Kind, Value: reason.Value}
		}
		result.EntryStoredBytes += uint64(entry.StoredBytes)
		result.Cases[index] = CorpusCaseInspection{
			Seed: uint64(entry.Seed), RecordHash: entry.RecordHash, Path: filepath.Join(root, filepath.FromSlash(entry.Artifact)),
			StoredBytes: uint64(entry.StoredBytes), PayloadBytes: uint64(entry.PayloadBytes), NoveltyReasons: reasons,
		}
	}
	return result
}

func projectExploration(explored campaign.ExplorationInspection) ExplorationInspection {
	result := ExplorationInspection{
		Schema: "gomad3.exploration-inspection/v1", Strategy: explored.Strategy, ImplementationSHA256: explored.ImplementationSHA256,
		ChainSHA256: explored.ChainSHA256, RecoveryExecutions: explored.RecoveryExecutions, Rounds: make([]ExplorationRoundInspection, len(explored.Rounds)),
	}
	if staged := explored.StagedRound; staged != nil {
		result.StagedRound = &ExplorationStagedRound{Index: staged.Index, Candidates: staged.Candidates}
	}
	for index, round := range explored.Rounds {
		result.Rounds[index] = ExplorationRoundInspection(round)
	}
	return result
}

func projectSimulationExploration(explored campaign.ExplorationInspection, state exploration.SimulationState) SimulationExplorationInspection {
	pending := make([]SimulationCandidateInspection, len(state.Queue))
	for candidateIndex, candidate := range state.Queue {
		overrides := make([]SimulationOverrideInspection, len(candidate.Overrides))
		for overrideIndex, override := range candidate.Overrides {
			projected := SimulationOverrideInspection{
				Dimension: string(override.Dimension), Ordinal: override.Ordinal, SiteSHA256: override.SiteSHA256,
				Alternatives: override.Alternatives, AlternativeSetSHA256: override.AlternativeSetSHA256,
				Selected: override.Selected, SelectedSHA256: override.SelectedSHA256, Identity: override.Identity,
			}
			if len(override.Control) != 0 {
				projected.ControlBytes = uint64(len(override.Control))
				projected.ControlSHA256 = record.HashBytes(override.Control)
			}
			overrides[overrideIndex] = projected
		}
		pending[candidateIndex] = SimulationCandidateInspection{SHA256: candidate.SHA256, ParentSHA256: candidate.ParentSHA256, Overrides: overrides}
	}
	result := SimulationExplorationInspection{
		Schema: "gomad3.simulation-exploration-inspection/v1", Summary: SimulationExplorationSummary(inspection.ProjectSimulationSummary(state.Summary())),
		ImplementationSHA256: explored.ImplementationSHA256, ChainSHA256: explored.ChainSHA256, Pending: pending,
	}
	if staged := explored.StagedRound; staged != nil {
		// A round start records its candidates' launch, so every candidate
		// of a staged round was attempted.
		result.StagedRound = &SimulationStagedRoundInspection{Index: staged.Index, Candidates: staged.Candidates, Attempted: staged.Candidates}
	}
	return result
}
