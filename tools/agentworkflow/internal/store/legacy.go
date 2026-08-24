package store

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type legacyStageResult struct {
	Schema         string `json:"schema"`
	RunID          string `json:"run_id"`
	Stage          string `json:"stage"`
	Status         string `json:"status"`
	ThreadID       string `json:"thread_id,omitempty"`
	FinalOutput    string `json:"final_output"`
	EventCount     int    `json:"event_count"`
	StdoutSHA256   string `json:"stdout_sha256"`
	StderrSHA256   string `json:"stderr_sha256"`
	StdoutBytes    uint64 `json:"stdout_bytes"`
	StderrBytes    uint64 `json:"stderr_bytes"`
	RunDirectory   string `json:"run_directory"`
	StageDirectory string `json:"stage_directory"`
}

func inspectLegacyRun(directory, runID string, maxBytes int64) (Inspection, error) {
	stagesDirectory := filepath.Join(directory, "stages")
	entries, err := os.ReadDir(stagesDirectory)
	if err != nil {
		return Inspection{}, fmt.Errorf("read legacy stage directory: %w", err)
	}
	started, updated := legacyRunTimes(directory)
	inspection := Inspection{Manifest: Manifest{
		Schema: "agentworkflow.run/v1", RunID: runID, State: "legacy", Phase: "legacy", Outcome: "inconclusive",
		StartedAt: started, UpdatedAt: updated,
	}}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		attempt, stageUpdated, err := inspectLegacyStage(stagesDirectory, runID, entry.Name(), len(inspection.Attempts)+1, maxBytes)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return Inspection{}, err
		}
		if stageUpdated.After(inspection.Manifest.UpdatedAt) {
			inspection.Manifest.UpdatedAt = stageUpdated
		}
		inspection.Attempts = append(inspection.Attempts, attempt)
	}
	return inspection, nil
}

func inspectLegacyStage(stagesDirectory, runID, stage string, attemptNumber int, maxBytes int64) (AttemptManifest, time.Time, error) {
	stageDirectory := filepath.Join(stagesDirectory, stage)
	encoded, err := readBounded(filepath.Join(stageDirectory, "stage.json"), maxBytes)
	if err != nil {
		return AttemptManifest{}, time.Time{}, err
	}
	var record legacyStageResult
	if err := strictDecode(encoded, &record); err != nil {
		return AttemptManifest{}, time.Time{}, fmt.Errorf("%w: decode legacy stage %q: %v", ErrCorrupt, stage, err)
	}
	if record.Schema != "agentworkflow.stage-result/v1" || record.RunID != runID || record.Stage != stage || record.Status != "completed" {
		return AttemptManifest{}, time.Time{}, fmt.Errorf("%w: inconsistent legacy stage %q", ErrCorrupt, stage)
	}
	if record.EventCount < 0 {
		return AttemptManifest{}, time.Time{}, fmt.Errorf("%w: legacy stage %q has a negative event count", ErrCorrupt, stage)
	}
	if err := verifyLegacyArtifact(filepath.Join(stageDirectory, "events.jsonl"), record.StdoutBytes, record.StdoutSHA256, "agentworkflow.stdout/v1", maxBytes); err != nil {
		return AttemptManifest{}, time.Time{}, fmt.Errorf("legacy stage %q events: %w", stage, err)
	}
	if err := verifyLegacyArtifact(filepath.Join(stageDirectory, "stderr.log"), record.StderrBytes, record.StderrSHA256, "agentworkflow.stderr/v1", maxBytes); err != nil {
		return AttemptManifest{}, time.Time{}, fmt.Errorf("legacy stage %q stderr: %w", stage, err)
	}
	info, err := os.Stat(filepath.Join(stageDirectory, "stage.json"))
	if err != nil {
		return AttemptManifest{}, time.Time{}, err
	}
	finished := info.ModTime().UTC()
	return AttemptManifest{
		Schema: "agentworkflow.stage-result/v1", RunID: runID, Attempt: attemptNumber, Stage: stage,
		Status: "completed", Session: record.ThreadID, StartedAt: finished, FinishedAt: finished,
		EventPath: filepath.ToSlash(filepath.Join("stages", stage, "events.jsonl")), EventCount: record.EventCount,
		EventBytes: int64(record.StdoutBytes), EventDigest: record.StdoutSHA256,
		OutputPath: filepath.ToSlash(filepath.Join("stages", stage, "stage.json")), OutputBytes: int64(len(encoded)),
	}, finished, nil
}

func verifyLegacyArtifact(path string, declaredBytes uint64, expectedDigest, domain string, maxBytes int64) error {
	if declaredBytes > uint64(maxBytes) {
		return fmt.Errorf("%w: artifact has invalid declared size", ErrCorrupt)
	}
	data, err := readBounded(path, maxBytes)
	if err != nil {
		return fmt.Errorf("%w: read artifact: %v", ErrCorrupt, err)
	}
	if uint64(len(data)) != declaredBytes || digest(domain, data) != expectedDigest {
		return fmt.Errorf("%w: artifact failed integrity validation", ErrCorrupt)
	}
	return nil
}

func legacyRunTimes(directory string) (started time.Time, updated time.Time) {
	info, err := os.Stat(directory)
	if err != nil {
		return time.Time{}, time.Time{}
	}
	modified := info.ModTime().UTC()
	return modified, modified
}
