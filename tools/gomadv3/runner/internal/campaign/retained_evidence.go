package campaign

import (
	"errors"
	"fmt"
	"path/filepath"

	"go.temporal.io/server/tools/gomadv3/artifact"
	"go.temporal.io/server/tools/gomadv3/record"
)

type RetainedEvidence struct {
	Path        string
	Manifest    record.ExecutionRecord
	StoredBytes uint64
}

func ResolveRetainedEvidence(batchPath, batchID string, run ExecutionRecord) (RetainedEvidence, error) {
	reference, success, err := retainedReference(run)
	if err != nil {
		return RetainedEvidence{}, err
	}
	path := filepath.Join(batchPath, filepath.FromSlash(reference))
	opened, err := artifact.OpenArtifact(path)
	if err != nil {
		return RetainedEvidence{}, err
	}
	retained := RetainedEvidence{Path: path, Manifest: opened.Manifest, StoredBytes: opened.StoredBytes}
	if err := opened.Close(); err != nil {
		return RetainedEvidence{}, fmt.Errorf("close retained artifact: %w", err)
	}
	manifest := retained.Manifest
	if manifest.CampaignID != batchID {
		return RetainedEvidence{}, retainedMismatchError(success)
	}
	if !retainedChoiceMatches(run, manifest) {
		return RetainedEvidence{}, retainedMismatchError(success)
	}
	if success {
		if manifest.SelectionOrdinal != run.SelectionOrdinal || manifest.Seed != run.Seed || manifest.ArtifactKind != record.ArtifactSuccess || manifest.Outcome.Domain != "success" || retained.StoredBytes != uint64(*run.SuccessArtifactBytes) {
			return RetainedEvidence{}, retainedMismatchError(true)
		}
	} else {
		if manifest.ArtifactKind != record.ArtifactTargetFailure && manifest.ArtifactKind != record.ArtifactWatchdogTimeout && manifest.ArtifactKind != record.ArtifactRunnerFailure {
			return RetainedEvidence{}, retainedMismatchError(false)
		}
		if manifest.Outcome.FailureSignature != *run.FailureSignature || manifest.Outcome.Domain != run.Domain || manifest.Outcome.Reason != run.Reason || manifest.Outcome.Termination != run.Termination {
			return RetainedEvidence{}, retainedMismatchError(false)
		}
	}
	return retained, nil
}

func retainedChoiceMatches(run ExecutionRecord, manifest record.ExecutionRecord) bool {
	present := run.ChoiceTraceSHA256 != nil || run.ChoiceTraceRecords != nil || run.ChoiceTraceBranchingRecords != nil || run.ChoiceTraceTerminalState != nil || run.ChoiceTapeSHA256 != nil || run.ChoiceDecisions != nil
	if !present {
		return manifest.ChoiceProfile == nil
	}
	if manifest.ChoiceProfile == nil || run.ChoiceTraceSHA256 == nil || run.ChoiceTraceRecords == nil || run.ChoiceTraceBranchingRecords == nil || run.ChoiceTraceTerminalState == nil {
		return false
	}
	trace := manifest.ChoiceProfile.Trace
	if trace.SHA256 != *run.ChoiceTraceSHA256 || trace.Records != *run.ChoiceTraceRecords || trace.BranchingRecords != *run.ChoiceTraceBranchingRecords || trace.TerminalState != *run.ChoiceTraceTerminalState {
		return false
	}
	if trace.TerminalState == "overflow" {
		return run.ChoiceTapeSHA256 == nil && run.ChoiceDecisions == nil && trace.TapeSHA256 == "" && trace.Decisions == 0
	}
	return run.ChoiceTapeSHA256 != nil && run.ChoiceDecisions != nil && trace.TapeSHA256 == *run.ChoiceTapeSHA256 && trace.Decisions == *run.ChoiceDecisions
}

func retainedReference(run ExecutionRecord) (string, bool, error) {
	if run.SuccessArtifact != nil {
		if run.Artifact != nil || run.SuccessArtifactBytes == nil || *run.SuccessArtifactBytes == 0 || !validSuccessArtifactReference(*run.SuccessArtifact) {
			return "", false, errors.New("retained success artifact reference is invalid")
		}
		return *run.SuccessArtifact, true, nil
	}
	if run.Artifact == nil || run.FailureSignature == nil || !validRecordSHA256(*run.FailureSignature) || !validArtifactReference(*run.Artifact) {
		return "", false, errors.New("failure artifact reference is invalid")
	}
	return *run.Artifact, false, nil
}

func retainedMismatchError(success bool) error {
	if success {
		return errors.New("retained success artifact does not match its campaign execution")
	}
	return errors.New("retained failure artifact does not match its campaign execution")
}
