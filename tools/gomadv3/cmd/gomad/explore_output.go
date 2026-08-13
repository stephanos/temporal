package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"

	"go.temporal.io/server/tools/gomadv3/internal/commandline"
	"go.temporal.io/server/tools/gomadv3/internal/ioprofile"
	"go.temporal.io/server/tools/gomadv3/internal/runner"
	"go.temporal.io/server/tools/gomadv3/internal/target"
)

const exploreEventSchema = "gomadv3.explore-event/v1"

type exploreEvent struct {
	Schema               string                      `json:"schema"`
	Type                 string                      `json:"type"`
	Phase                runner.ProgressPhase        `json:"phase,omitempty"`
	Classification       string                      `json:"classification,omitempty"`
	Message              string                      `json:"message,omitempty"`
	BatchPath            string                      `json:"batch_path,omitempty"`
	Path                 string                      `json:"path,omitempty"`
	ReplayCommand        string                      `json:"replay_command,omitempty"`
	Selected             uint64                      `json:"selected,omitempty"`
	Attempted            uint64                      `json:"attempted,omitempty"`
	Running              uint64                      `json:"running,omitempty"`
	Succeeded            uint64                      `json:"succeeded,omitempty"`
	Failures             uint64                      `json:"failures,omitempty"`
	Watchdogs            uint64                      `json:"watchdogs,omitempty"`
	ReplayDivergences    uint64                      `json:"replay_divergences,omitempty"`
	Cancelled            uint64                      `json:"cancelled,omitempty"`
	Novelty              uint64                      `json:"novelty,omitempty"`
	RetainedSuccesses    uint64                      `json:"retained_successes,omitempty"`
	RetainedSuccessBytes uint64                      `json:"retained_success_bytes,omitempty"`
	StopReason           runner.StopReason           `json:"stop_reason,omitempty"`
	SemanticCoverage     *ioprofile.SemanticCoverage `json:"semantic_coverage,omitempty"`
	ChoiceTrace          *runner.ChoiceTraceSummary  `json:"choice_trace,omitempty"`
	CorpusPath           string                      `json:"corpus_path,omitempty"`
	CorpusEntries        uint64                      `json:"corpus_entries,omitempty"`
	CorpusAdded          uint64                      `json:"corpus_added,omitempty"`
}

type exploreReporter struct {
	json   bool
	stdout io.Writer
	stderr io.Writer
	mu     sync.Mutex
}

func newExploreReporter(jsonOutput bool, stdout, stderr io.Writer) *exploreReporter {
	return &exploreReporter{json: jsonOutput, stdout: stdout, stderr: stderr}
}

func (reporter *exploreReporter) Progress(progress runner.Progress) error {
	reporter.mu.Lock()
	defer reporter.mu.Unlock()
	if reporter.json {
		return reporter.writeEvent(exploreEvent{
			Schema: exploreEventSchema, Type: "progress", Phase: progress.Phase, BatchPath: progress.BatchPath,
			Selected: progress.Selected, Attempted: progress.Attempted, Running: progress.Running, Succeeded: progress.Succeeded,
			Failures: progress.Failures, Watchdogs: progress.Watchdogs, ReplayDivergences: progress.ReplayDivergences, Cancelled: progress.Cancelled, Novelty: progress.DistinctFailures,
			RetainedSuccesses: progress.RetainedSuccesses, RetainedSuccessBytes: progress.RetainedSuccessBytes,
			CorpusPath: progress.CorpusPath, CorpusEntries: progress.CorpusEntries, CorpusAdded: progress.CorpusAdded,
			ChoiceTrace: progress.ChoiceTrace,
		})
	}
	_, err := fmt.Fprintf(reporter.stderr, "gomad: phase=%s selected=%d attempted=%d running=%d succeeded=%d failures=%d watchdogs=%d replay-divergences=%d novelty=%d retained-successes=%d retained-success-bytes=%d artifact=%s%s\n", progress.Phase, progress.Selected, progress.Attempted, progress.Running, progress.Succeeded, progress.Failures, progress.Watchdogs, progress.ReplayDivergences, progress.DistinctFailures, progress.RetainedSuccesses, progress.RetainedSuccessBytes, progress.BatchPath, formatChoiceTrace(progress.ChoiceTrace))
	return err
}

func (reporter *exploreReporter) Result(summary runner.Summary) error {
	reporter.mu.Lock()
	defer reporter.mu.Unlock()
	classification := classifyExploreSummary(summary)
	if reporter.json {
		if err := reporter.writeEvent(exploreEvent{
			Schema: exploreEventSchema, Type: "result", Classification: classification, BatchPath: summary.BatchPath,
			Selected: summary.SelectionCount, Attempted: summary.Attempted, Succeeded: summary.Succeeded, Failures: summary.Failures,
			Watchdogs: summary.Watchdogs, ReplayDivergences: summary.ReplayDivergences, Cancelled: summary.Cancelled, Novelty: summary.DistinctFailures, StopReason: summary.StopReason,
			RetainedSuccesses: summary.RetainedSuccesses, RetainedSuccessBytes: summary.RetainedSuccessBytes, SemanticCoverage: summary.SemanticCoverage,
			CorpusPath: summary.CorpusPath, CorpusEntries: summary.CorpusEntries, CorpusAdded: summary.CorpusAdded,
			ChoiceTrace: summary.ChoiceTrace,
		}); err != nil {
			return err
		}
		for _, path := range summary.Artifacts {
			if err := reporter.writeEvent(exploreEvent{
				Schema: exploreEventSchema, Type: "artifact", Classification: classification, Path: path, ReplayCommand: "gomad replay " + commandline.QuoteArgument(path),
			}); err != nil {
				return err
			}
		}
		for _, path := range summary.SuccessArtifacts {
			if err := reporter.writeEvent(exploreEvent{
				Schema: exploreEventSchema, Type: "artifact", Classification: "success", Path: path, ReplayCommand: "gomad replay " + commandline.QuoteArgument(path),
			}); err != nil {
				return err
			}
		}
		return nil
	}
	if _, err := fmt.Fprintf(reporter.stdout, "gomad: classification=%s attempted=%d succeeded=%d failures=%d watchdogs=%d replay-divergences=%d distinct=%d retained-successes=%d retained-success-bytes=%d stop=%s artifact=%s%s\n", classification, summary.Attempted, summary.Succeeded, summary.Failures, summary.Watchdogs, summary.ReplayDivergences, summary.DistinctFailures, summary.RetainedSuccesses, summary.RetainedSuccessBytes, summary.StopReason, summary.BatchPath, formatChoiceTrace(summary.ChoiceTrace)); err != nil {
		return err
	}
	for _, path := range summary.Artifacts {
		if _, err := fmt.Fprintf(reporter.stdout, "gomad: retained failure: %s\ngomad: replay: gomad replay %s\n", path, commandline.QuoteArgument(path)); err != nil {
			return err
		}
	}
	for _, path := range summary.SuccessArtifacts {
		if _, err := fmt.Fprintf(reporter.stdout, "gomad: retained success: %s\ngomad: replay: gomad replay %s\n", path, commandline.QuoteArgument(path)); err != nil {
			return err
		}
	}
	if summary.SemanticCoverage != nil {
		if _, err := fmt.Fprintf(reporter.stdout, "gomad: semantic-coverage digest=%s probes=%d %s\n", summary.SemanticCoverage.Digest, len(summary.SemanticCoverage.Probes), strings.Join(summary.SemanticCoverage.Probes, ",")); err != nil {
			return err
		}
	}
	if summary.CorpusPath != "" {
		if _, err := fmt.Fprintf(reporter.stdout, "gomad: guided-corpus path=%s entries=%d added=%d\n", summary.CorpusPath, summary.CorpusEntries, summary.CorpusAdded); err != nil {
			return err
		}
	}
	return nil
}

func formatChoiceTrace(trace *runner.ChoiceTraceSummary) string {
	if trace == nil {
		return ""
	}
	return fmt.Sprintf(" choices-seed=%d choices-profile=%s choices-records=%d choices-branching=%d choices-runnable=%d choices-select-poll=%d choices-select-result=%d choices-sha256=%s choices-terminal=%s", trace.Seed, trace.Profile, trace.Records, trace.BranchingRecords, trace.Runnable, trace.SelectPoll, trace.SelectResult, trace.SHA256, trace.TerminalState)
}

func (reporter *exploreReporter) Error(classification string, err error) error {
	reporter.mu.Lock()
	defer reporter.mu.Unlock()
	if reporter.json {
		return reporter.writeEvent(exploreEvent{Schema: exploreEventSchema, Type: "error", Classification: classification, Message: err.Error()})
	}
	_, writeErr := fmt.Fprintf(reporter.stderr, "gomad: %s: %v\n", classification, err)
	return writeErr
}

func (reporter *exploreReporter) writeEvent(event exploreEvent) error {
	encoded, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("encode explore event: %w", err)
	}
	encoded = append(encoded, '\n')
	if _, err := reporter.stdout.Write(encoded); err != nil {
		return fmt.Errorf("write explore event: %w", err)
	}
	return nil
}

func classifyExploreError(err error) string {
	var missing *ioprofile.MissingSemanticProbesError
	if errors.As(err, &missing) {
		return "semantic_coverage_failure"
	}
	var unsupported *target.UnsupportedCapabilityError
	if errors.As(err, &unsupported) {
		return "unsupported_target"
	}
	var hostError *runner.HostError
	if errors.As(err, &hostError) {
		if hostError.Reason == "cancelled" || hostError.Reason == "overall_timeout" {
			return hostError.Reason
		}
		return "runner_failure"
	}
	return "invalid_input"
}

func exploreErrorStatus(classification string) int {
	switch classification {
	case "semantic_coverage_failure":
		return 1
	case "invalid_input", "unsupported_target":
		return 2
	default:
		return 3
	}
}

func classifyExploreSummary(summary runner.Summary) string {
	if summary.Failures == 0 {
		return "success"
	}
	if summary.ReplayDivergences == summary.Failures {
		return "replay_divergence"
	}
	if summary.Watchdogs == summary.Failures {
		return "watchdog_observation"
	}
	if summary.Watchdogs != 0 || summary.ReplayDivergences != 0 {
		return "mixed_failure"
	}
	return "target_failure"
}
