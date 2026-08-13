package artifact

import (
	"errors"
	"fmt"
	"path/filepath"

	"go.temporal.io/server/tools/gomadv3/internal/record"
)

type RetainedEvidence struct {
	Path        string
	Manifest    record.Manifest
	StoredBytes uint64
}

func ResolveRetainedEvidence(batchPath, batchID string, run RunRecord) (RetainedEvidence, error) {
	reference, success, err := retainedReference(run)
	if err != nil {
		return RetainedEvidence{}, err
	}
	path := filepath.Join(batchPath, filepath.FromSlash(reference))
	opened, err := Open(path)
	if err != nil {
		return RetainedEvidence{}, err
	}
	evidence := RetainedEvidence{Path: path, Manifest: opened.Manifest, StoredBytes: opened.StoredBytes}
	if err := opened.Close(); err != nil {
		return RetainedEvidence{}, fmt.Errorf("close retained artifact: %w", err)
	}
	manifest := evidence.Manifest
	if manifest.BatchID != batchID {
		return RetainedEvidence{}, retainedMismatchError(success)
	}
	if success {
		if manifest.SelectionOrdinal != run.SelectionOrdinal || manifest.Seed != run.Seed || manifest.ArtifactKind != record.ArtifactSuccess || manifest.Outcome.Domain != "success" || evidence.StoredBytes != uint64(*run.SuccessArtifactBytes) {
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
	return evidence, nil
}

func retainedReference(run RunRecord) (string, bool, error) {
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
		return errors.New("retained success artifact does not match its batch run")
	}
	return errors.New("retained failure artifact does not match its batch run")
}
