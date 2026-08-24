package store

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

func (run *Run) RecoverAttempts(maxBytes int64, maxEvents int, now time.Time) error {
	if maxBytes <= 0 || maxEvents <= 0 {
		return errors.New("attempt recovery bounds must be positive")
	}
	run.mu.Lock()
	defer run.mu.Unlock()
	if run.closed {
		return errors.New("store run is closed")
	}
	entries, err := os.ReadDir(filepath.Join(run.dir, "attempts"))
	if err != nil {
		return fmt.Errorf("read attempt directory during recovery: %w", err)
	}
	changed := false
	for _, entry := range entries {
		if !entry.IsDir() {
			return fmt.Errorf("%w: unexpected attempt entry %q", ErrCorrupt, entry.Name())
		}
		attemptDirectory := filepath.Join(run.dir, "attempts", entry.Name())
		attempt, err := readAttemptManifest(attemptDirectory, maxBytes)
		if err != nil {
			return err
		}
		if attempt.RunID != run.id || attempt.Schema != "agentworkflow.attempt/v2" {
			return fmt.Errorf("%w: inconsistent attempt %q", ErrCorrupt, entry.Name())
		}
		if attempt.Status != "running" {
			continue
		}
		if err := recoverAttempt(attemptDirectory, &attempt, maxBytes, maxEvents, now); err != nil {
			return fmt.Errorf("recover attempt %q: %w", entry.Name(), err)
		}
		changed = true
	}
	if !changed {
		return nil
	}
	run.manifest.UpdatedAt = now.UTC()
	return run.writeManifest()
}

func (run *Run) ReadCompletedAttempt(stage string, maxBytes int64) (AttemptManifest, []byte, bool, error) {
	if err := validComponent(stage); err != nil {
		return AttemptManifest{}, nil, false, err
	}
	run.mu.Lock()
	defer run.mu.Unlock()
	entries, err := os.ReadDir(filepath.Join(run.dir, "attempts"))
	if err != nil {
		return AttemptManifest{}, nil, false, fmt.Errorf("read attempt directory: %w", err)
	}
	var selected AttemptManifest
	var selectedDirectory string
	for _, entry := range entries {
		if !entry.IsDir() {
			return AttemptManifest{}, nil, false, fmt.Errorf("%w: unexpected attempt entry %q", ErrCorrupt, entry.Name())
		}
		attempt, err := inspectAttempt(run.dir, run.id, entry.Name(), maxBytes)
		if err != nil {
			return AttemptManifest{}, nil, false, err
		}
		if attempt.Stage == stage && attempt.Status == "completed" && attempt.Attempt > selected.Attempt {
			selected = attempt
			selectedDirectory = filepath.Join(run.dir, "attempts", entry.Name())
		}
	}
	if selected.Attempt == 0 {
		return AttemptManifest{}, nil, false, nil
	}
	output, err := readBounded(filepath.Join(selectedDirectory, filepath.FromSlash(selected.OutputPath)), maxBytes)
	if err != nil {
		return AttemptManifest{}, nil, false, fmt.Errorf("read completed attempt output: %w", err)
	}
	return selected, output, true, nil
}

func recoverAttempt(directory string, attempt *AttemptManifest, maxBytes int64, maxEvents int, now time.Time) error {
	if attempt.EventPath != "events.jsonl" || attempt.OutputPath != "" {
		return fmt.Errorf("%w: running attempt paths are inconsistent", ErrCorrupt)
	}
	events, err := readBounded(filepath.Join(directory, attempt.EventPath), maxBytes)
	if err != nil {
		return fmt.Errorf("%w: read event prefix: %v", ErrCorrupt, err)
	}
	prefix, err := validateEventPrefix(events, maxEvents)
	if err != nil {
		return err
	}
	attempt.EventCount = prefix.count
	attempt.EventBytes = int64(len(events))
	hasher := newHashWriter("agentworkflow.events/v2")
	_, _ = hasher.Write(events)
	attempt.EventDigest = hasher.Digest()
	attempt.FinishedAt = now.UTC()
	outputPath := filepath.Join(directory, "output.json")
	output, outputErr := readBounded(outputPath, maxBytes)
	switch {
	case outputErr == nil:
		if !prefix.successful {
			return fmt.Errorf("%w: recovered output lacks a successful event lifecycle", ErrCorrupt)
		}
		if !json.Valid(output) {
			return fmt.Errorf("%w: recovered output is not valid JSON", ErrCorrupt)
		}
		attempt.Status = "completed"
		attempt.OutputPath = "output.json"
		attempt.OutputBytes = int64(len(output))
		attempt.OutputDigest = digest("agentworkflow.output/v2", output)
	case errors.Is(outputErr, os.ErrNotExist):
		attempt.Status = "interrupted"
		attempt.Error = "process ended before attempt output publication"
	default:
		return fmt.Errorf("%w: inspect recovered output: %v", ErrCorrupt, outputErr)
	}
	encoded, err := json.Marshal(attempt)
	if err != nil {
		return err
	}
	return atomicWrite(filepath.Join(directory, "attempt.json"), encoded)
}

type eventPrefix struct {
	count      int
	started    bool
	terminal   bool
	successful bool
}

func validateEventPrefix(events []byte, maxEvents int) (eventPrefix, error) {
	result := eventPrefix{}
	for _, line := range bytes.Split(events, []byte{'\n'}) {
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}
		result.count++
		if result.count > maxEvents {
			return eventPrefix{}, fmt.Errorf("%w: recovered event count exceeds its bound", ErrCorrupt)
		}
		var event struct {
			Kind string `json:"kind"`
		}
		if err := json.Unmarshal(line, &event); err != nil {
			return eventPrefix{}, fmt.Errorf("%w: recovered event %d is not valid JSON", ErrCorrupt, result.count)
		}
		if result.terminal {
			return eventPrefix{}, fmt.Errorf("%w: recovered event follows a terminal event", ErrCorrupt)
		}
		switch event.Kind {
		case "invocation-started":
			if result.started {
				return eventPrefix{}, fmt.Errorf("%w: recovered lifecycle has duplicate starts", ErrCorrupt)
			}
			result.started = true
		case "invocation-completed":
			result.terminal = true
			result.successful = result.started
		case "invocation-failed":
			result.terminal = true
		default:
		}
	}
	return result, nil
}

func readAttemptManifest(directory string, maxBytes int64) (AttemptManifest, error) {
	encoded, err := readBounded(filepath.Join(directory, "attempt.json"), maxBytes)
	if err != nil {
		return AttemptManifest{}, fmt.Errorf("read attempt manifest: %w", err)
	}
	var attempt AttemptManifest
	if err := strictDecode(encoded, &attempt); err != nil {
		return AttemptManifest{}, fmt.Errorf("%w: decode attempt manifest: %v", ErrCorrupt, err)
	}
	return attempt, nil
}
