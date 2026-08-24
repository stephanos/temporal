package campaign

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"go.temporal.io/server/tools/gomad3/internal/canonicaljson"
	"go.temporal.io/server/tools/gomad3/record"
	choiceengine "go.temporal.io/server/tools/gomad3/runner/internal/exploration/choice"
	simulationengine "go.temporal.io/server/tools/gomad3/runner/internal/exploration/simulation"
)

type Campaign struct {
	Path       string
	Record     CampaignRecord
	Executions []ExecutionRecord
	Journal    *ExecutionJournalInfo
}

type ExecutionJournalInfo struct {
	Schema      string
	IndexSHA256 record.SHA256
	Segments    uint64
	Records     uint64
	Bytes       uint64
	Limits      ExecutionJournalPlan
}

func OpenCampaign(path string) (Campaign, error) {
	rootInfo, err := os.Lstat(path)
	if err != nil {
		return Campaign{}, fmt.Errorf("open campaign directory: %w", err)
	}
	if !rootInfo.IsDir() || rootInfo.Mode()&os.ModeSymlink != 0 {
		return Campaign{}, fmt.Errorf("campaign path is not a directory")
	}
	if rootInfo.Mode().Perm() != 0o700 {
		return Campaign{}, fmt.Errorf("campaign directory mode is %#o, want 0700", rootInfo.Mode().Perm())
	}
	root, err := os.OpenRoot(path)
	if err != nil {
		return Campaign{}, fmt.Errorf("pin campaign directory: %w", err)
	}
	defer root.Close()
	pinnedInfo, err := root.Stat(".")
	if err != nil || !os.SameFile(rootInfo, pinnedInfo) {
		return Campaign{}, errors.Join(fmt.Errorf("campaign directory changed while opening"), err)
	}
	batchBytes, err := readValidatedFile(root, "campaign.json", 0o600, maximumManifestBytes)
	if err != nil {
		return Campaign{}, fmt.Errorf("read campaign record: %w", err)
	}
	var batch CampaignRecord
	if err := canonicaljson.DecodeCanonicalJSON(batchBytes, &batch); err != nil {
		return Campaign{}, fmt.Errorf("decode campaign record: %w", err)
	}
	runs, journal, err := readPublishedExecutionJournal(root, batch)
	if err != nil {
		return Campaign{}, classifyIntegrityError(err)
	}
	if err := validateCampaign(batch, runs); err != nil {
		return Campaign{}, err
	}
	if batch.Artifacts != nil {
		if err := validatePublishedArtifactCapacity(path, batch, runs); err != nil {
			return Campaign{}, classifyIntegrityError(err)
		}
	}
	if batch.Strategy == "choice-exploration" {
		if err := ValidatePublishedExploration(path, *batch.ChoiceExploration, batch.ChoiceExplorationImplementationSHA256, batch.ChoiceExplorationChainSHA256, runs); err != nil {
			return Campaign{}, fmt.Errorf("validate published exploration: %w", err)
		}
	}
	if batch.Strategy == "simulation-exploration" {
		if err := ValidatePublishedSimulationExploration(path, *batch.SimulationExploration, batch.SimulationExplorationImplementationSHA256, batch.SimulationExplorationChainSHA256, runs); err != nil {
			return Campaign{}, fmt.Errorf("validate published simulation exploration: %w", err)
		}
	}
	return Campaign{Path: path, Record: batch, Executions: runs, Journal: journal}, nil
}

func decodeExecutions(contents []byte) ([]ExecutionRecord, error) {
	if len(contents) == 0 {
		return []ExecutionRecord{}, nil
	}
	if contents[len(contents)-1] != '\n' {
		return nil, fmt.Errorf("campaign execution journal is not newline terminated")
	}
	lines := bytes.Split(contents[:len(contents)-1], []byte{'\n'})
	runs := make([]ExecutionRecord, len(lines))
	for index, line := range lines {
		if len(line) == 0 {
			return nil, fmt.Errorf("campaign execution journal has an empty record at line %d", index+1)
		}
		if err := canonicaljson.DecodeCanonicalJSON(line, &runs[index]); err != nil {
			return nil, fmt.Errorf("decode campaign execution %d: %w", index+1, err)
		}
	}
	return runs, nil
}

func validateCampaign(batch CampaignRecord, runs []ExecutionRecord) error {
	if batch.SchemaVersion != record.SchemaVersion || batch.Schema != "gomad3.campaign/v1" || batch.CampaignID == "" || batch.Selection == "" || batch.SelectionCount == 0 {
		return fmt.Errorf("campaign record identity is invalid")
	}
	if (batch.PlanSHA256 == "") != (batch.Shard == nil) {
		return errors.New("campaign portable plan identity is incomplete")
	}
	if batch.Shard != nil && (!validRecordSHA256(batch.PlanSHA256) || batch.Shard.Count == 0 || batch.Shard.Index >= batch.Shard.Count || batch.Strategy != "seed") {
		return errors.New("sharded campaign identity is invalid")
	}
	if batch.Journal == nil {
		return errors.New("campaign journal identity is missing")
	}
	choiceEvidence := batch.ChoiceExploration != nil || batch.ChoiceExplorationImplementationSHA256 != "" || batch.ChoiceExplorationChainSHA256 != ""
	combinedEvidence := batch.SimulationExploration != nil || batch.SimulationExplorationImplementationSHA256 != "" || batch.SimulationExplorationChainSHA256 != ""
	if batch.Strategy != "seed" && batch.Strategy != "choice-exploration" && batch.Strategy != "simulation-exploration" ||
		batch.Strategy == "seed" && (choiceEvidence || combinedEvidence) ||
		batch.Strategy == "choice-exploration" && (batch.ChoiceExploration == nil || batch.ChoiceExplorationImplementationSHA256 != choiceengine.ImplementationSHA256() || !validRecordSHA256(batch.ChoiceExplorationChainSHA256) || combinedEvidence) ||
		batch.Strategy == "simulation-exploration" && (batch.SimulationExploration == nil || batch.SimulationExplorationImplementationSHA256 != simulationengine.ImplementationSHA256() || !validRecordSHA256(batch.SimulationExplorationChainSHA256) || choiceEvidence) {
		return fmt.Errorf("campaign strategy evidence is invalid")
	}
	if uint64(batch.Attempted) != uint64(len(runs)) || uint64(batch.Succeeded)+uint64(batch.Failures)+uint64(batch.Cancelled) != uint64(batch.Attempted) || batch.Watchdogs > batch.Failures || batch.RetainedSuccesses > batch.Succeeded || batch.RetainedSuccesses == 0 && batch.RetainedSuccessBytes != 0 {
		return fmt.Errorf("campaign summary counts are inconsistent")
	}
	if limits := batch.Artifacts; limits != nil {
		failureBytes := uint64(limits.FailureBytes)
		successBytes := uint64(limits.SuccessBytes)
		if limits.FailureOutcome != CapacityInfrastructureFailure || limits.SuccessOutcome != CapacityInfrastructureFailure || limits.FailureArtifacts == 0 || failureBytes == 0 || limits.TranscriptBytes == 0 || uint64(limits.TotalBytes) != failureBytes+successBytes || uint64(limits.TotalBytes) < failureBytes ||
			batch.DistinctFailures > limits.FailureArtifacts || batch.RetainedSuccesses > limits.SuccessArtifacts || batch.RetainedSuccessBytes > limits.SuccessBytes {
			return errors.New("campaign artifact capacity is invalid")
		}
	}
	if batch.StopReason != "seeds_exhausted" && batch.StopReason != "first_failure" && batch.StopReason != "failure_budget" && batch.StopReason != "exploration_exhausted" && batch.StopReason != "choice_depth_complete" && batch.StopReason != "simulation_depth_complete" && batch.StopReason != "dimension_depth_complete" && batch.StopReason != "max_executions" && batch.StopReason != "exploration_capacity" {
		return fmt.Errorf("campaign stop reason is invalid: %s", batch.StopReason)
	}
	ordinals := make(map[uint64]struct{}, len(runs))
	candidates := make(map[record.SHA256]struct{}, len(runs))
	failures := make(map[record.SHA256]struct{})
	var succeeded, failed, watchdogs, cancelled, retainedSuccesses, retainedSuccessBytes uint64
	for index, run := range runs {
		ordinal := uint64(run.SelectionOrdinal)
		ordinalLimit := uint64(batch.SelectionCount)
		if batch.Strategy == "choice-exploration" || batch.Strategy == "simulation-exploration" {
			ordinalLimit = uint64(batch.Attempted)
		}
		if ordinal >= ordinalLimit {
			return fmt.Errorf("campaign execution %d selection ordinal is out of range", index+1)
		}
		if batch.Shard != nil && ordinal%uint64(batch.Shard.Count) != uint64(batch.Shard.Index) {
			return fmt.Errorf("campaign execution %d is outside shard assignment", index+1)
		}
		if _, duplicate := ordinals[ordinal]; duplicate {
			return fmt.Errorf("campaign selection ordinal is duplicated: %d", ordinal)
		}
		if batch.Strategy == "choice-exploration" {
			baseSeed, parseErr := strconv.ParseUint(batch.Selection, 10, 64)
			if parseErr != nil || uint64(run.Seed) != baseSeed || ordinal != uint64(index) {
				return fmt.Errorf("campaign exploration execution %d seed or logical ordinal is invalid", index+1)
			}
			if err := validateExplorationExecutionSummary(run, candidates); err != nil {
				return fmt.Errorf("campaign exploration execution %d: %w", index+1, err)
			}
		} else if batch.Strategy == "simulation-exploration" {
			baseSeed, parseErr := strconv.ParseUint(batch.Selection, 10, 64)
			if parseErr != nil || uint64(run.Seed) != baseSeed || ordinal != uint64(index) {
				return fmt.Errorf("campaign simulation exploration execution %d seed or logical ordinal is invalid", index+1)
			}
			if err := validateSimulationExplorationExecutionSummary(run, candidates); err != nil {
				return fmt.Errorf("campaign simulation exploration execution %d: %w", index+1, err)
			}
		} else if run.Strategy != "" {
			return fmt.Errorf("seed campaign execution %d contains strategy evidence", index+1)
		}
		ordinals[ordinal] = struct{}{}
		if run.Reason == "" {
			return fmt.Errorf("campaign execution %d identity is invalid", index+1)
		}
		if (run.IOTranscriptSHA256 == nil) != (run.IOTranscriptRecords == nil) {
			return fmt.Errorf("campaign execution %d transcript identity is incomplete", index+1)
		}
		if run.IOTranscriptSHA256 != nil {
			if _, err := record.ParseSHA256(string(*run.IOTranscriptSHA256)); err != nil {
				return fmt.Errorf("campaign execution %d transcript digest is invalid", index+1)
			}
		}
		if err := validateChoiceExecutionSummary(run); err != nil {
			return fmt.Errorf("campaign execution %d: %w", index+1, err)
		}
		if err := validateSemanticProbeLists(run.SemanticProbes, run.NovelSemanticProbes); err != nil {
			return fmt.Errorf("campaign execution %d: %w", index+1, err)
		}
		if err := validateChoiceFeatureLists(run.ChoiceFeatures, run.NovelChoiceFeatures); err != nil {
			return fmt.Errorf("campaign execution %d: %w", index+1, err)
		}
		switch run.Domain {
		case "success":
			succeeded++
			if run.Termination != "exit" || run.FailureSignature != nil || run.Artifact != nil {
				return fmt.Errorf("successful campaign execution %d has failure evidence", index+1)
			}
			if (run.SuccessArtifact == nil) != (run.SuccessArtifactBytes == nil) {
				return fmt.Errorf("successful campaign execution %d has incomplete retained evidence", index+1)
			}
			if run.SuccessArtifact == nil {
				if len(run.NovelSemanticProbes) != 0 || len(run.NovelChoiceFeatures) != 0 {
					return fmt.Errorf("unretained successful campaign execution %d has novelty reasons", index+1)
				}
			} else if !validSuccessArtifactReference(*run.SuccessArtifact) || *run.SuccessArtifactBytes == 0 {
				return fmt.Errorf("successful campaign execution %d has invalid retained evidence", index+1)
			} else {
				retainedSuccesses++
				if uint64(*run.SuccessArtifactBytes) > ^uint64(0)-retainedSuccessBytes {
					return fmt.Errorf("retained success byte count overflows")
				}
				retainedSuccessBytes += uint64(*run.SuccessArtifactBytes)
			}
		case "target", "watchdog":
			failed++
			if run.Domain == "watchdog" {
				watchdogs++
			}
			if run.FailureSignature == nil || !validRecordSHA256(*run.FailureSignature) || run.Artifact == nil || !validArtifactReference(*run.Artifact) {
				return fmt.Errorf("failed campaign execution %d has invalid artifact evidence", index+1)
			}
			failures[*run.FailureSignature] = struct{}{}
			if run.SuccessArtifact != nil || run.SuccessArtifactBytes != nil || len(run.NovelSemanticProbes) != 0 || len(run.NovelChoiceFeatures) != 0 {
				return fmt.Errorf("failed campaign execution %d has retained successful execution evidence", index+1)
			}
		case "runner":
			cancelled++
			if run.Reason != "runner_cancelled" || run.Termination != "none" || run.FailureSignature != nil || run.Artifact != nil {
				return fmt.Errorf("cancelled campaign execution %d is invalid", index+1)
			}
			if run.SuccessArtifact != nil || run.SuccessArtifactBytes != nil || len(run.NovelSemanticProbes) != 0 || len(run.NovelChoiceFeatures) != 0 {
				return fmt.Errorf("cancelled campaign execution %d has retained successful execution evidence", index+1)
			}
		default:
			return fmt.Errorf("campaign execution %d domain is invalid: %s", index+1, run.Domain)
		}
	}
	if succeeded != uint64(batch.Succeeded) || failed != uint64(batch.Failures) || watchdogs != uint64(batch.Watchdogs) || cancelled != uint64(batch.Cancelled) {
		return fmt.Errorf("campaign execution counts do not match the summary")
	}
	if retainedSuccesses != uint64(batch.RetainedSuccesses) || retainedSuccessBytes != uint64(batch.RetainedSuccessBytes) {
		return fmt.Errorf("campaign retained success counts do not match the summary")
	}
	if uint64(len(failures)) != uint64(batch.DistinctFailures) || len(batch.FailureSignatures) != len(failures) {
		return fmt.Errorf("campaign distinct failure count is inconsistent")
	}
	if !sort.SliceIsSorted(batch.FailureSignatures, func(i, j int) bool { return batch.FailureSignatures[i] < batch.FailureSignatures[j] }) {
		return fmt.Errorf("campaign failure signatures are not sorted")
	}
	for index, signature := range batch.FailureSignatures {
		if !validRecordSHA256(signature) || index > 0 && batch.FailureSignatures[index-1] == signature {
			return fmt.Errorf("campaign failure signatures are invalid")
		}
		if _, found := failures[signature]; !found {
			return fmt.Errorf("campaign failure signature has no execution: %s", signature)
		}
	}
	return nil
}

func validatePublishedArtifactCapacity(batchPath string, batch CampaignRecord, runs []ExecutionRecord) error {
	limits := *batch.Artifacts
	seenFailures := make(map[record.SHA256]struct{}, uint64(batch.DistinctFailures))
	var failureBytes, successBytes uint64
	for index, run := range runs {
		if run.SuccessArtifact != nil {
			retained, err := ResolveRetainedEvidence(batchPath, batch.CampaignID, run)
			if err != nil {
				return fmt.Errorf("validate published success artifact %d: %w", index+1, err)
			}
			if retained.StoredBytes > ^uint64(0)-successBytes {
				return errors.New("published success artifact bytes overflow")
			}
			successBytes += retained.StoredBytes
			continue
		}
		if run.FailureSignature == nil {
			continue
		}
		if _, found := seenFailures[*run.FailureSignature]; found {
			continue
		}
		retained, err := ResolveRetainedEvidence(batchPath, batch.CampaignID, run)
		if err != nil {
			return fmt.Errorf("validate published failure artifact %d: %w", index+1, err)
		}
		if retained.StoredBytes > ^uint64(0)-failureBytes {
			return errors.New("published failure artifact bytes overflow")
		}
		failureBytes += retained.StoredBytes
		seenFailures[*run.FailureSignature] = struct{}{}
	}
	if failureBytes > uint64(limits.FailureBytes) || successBytes > uint64(limits.SuccessBytes) || successBytes > ^uint64(0)-failureBytes || failureBytes+successBytes > uint64(limits.TotalBytes) {
		return errors.New("published artifacts exceed the campaign capacity")
	}
	return nil
}

func validateExplorationExecutionSummary(run ExecutionRecord, candidates map[record.SHA256]struct{}) error {
	if run.Strategy != "choice-exploration" || run.Round == nil || run.ForcedDepth == nil || !validRecordSHA256(run.CandidateSHA256) || !validRecordSHA256(run.OutcomeSHA256) {
		return errors.New("exploration identity is incomplete")
	}
	if _, found := candidates[run.CandidateSHA256]; found {
		return errors.New("candidate identity is duplicated")
	}
	candidates[run.CandidateSHA256] = struct{}{}
	if *run.ForcedDepth == 0 {
		if run.ParentCandidateSHA256 != "" || run.PrefixSHA256 != "" || *run.Round != 0 {
			return errors.New("root candidate provenance is invalid")
		}
	} else if !validRecordSHA256(run.ParentCandidateSHA256) || !validRecordSHA256(run.PrefixSHA256) {
		return errors.New("forced candidate provenance is invalid")
	}
	return nil
}

func validateSimulationExplorationExecutionSummary(run ExecutionRecord, candidates map[record.SHA256]struct{}) error {
	if run.Strategy != "simulation-exploration" || run.Round == nil || run.ForcedDepth == nil || !validRecordSHA256(run.CandidateSHA256) || !validRecordSHA256(run.OutcomeSHA256) || run.PrefixSHA256 != "" {
		return errors.New("simulation exploration identity is incomplete")
	}
	if _, found := candidates[run.CandidateSHA256]; found {
		return errors.New("candidate identity is duplicated")
	}
	candidates[run.CandidateSHA256] = struct{}{}
	if *run.ForcedDepth == 0 {
		if run.ParentCandidateSHA256 != "" || *run.Round != 0 {
			return errors.New("root combined candidate provenance is invalid")
		}
	} else if !validRecordSHA256(run.ParentCandidateSHA256) {
		return errors.New("forced combined candidate provenance is invalid")
	}
	return nil
}

func validateChoiceExecutionSummary(run ExecutionRecord) error {
	present := 0
	for _, value := range []bool{
		run.ChoiceTraceSHA256 != nil,
		run.ChoiceTraceRecords != nil,
		run.ChoiceTraceBranchingRecords != nil,
		run.ChoiceTraceTerminalState != nil,
	} {
		if value {
			present++
		}
	}
	tapePresent := run.ChoiceTapeSHA256 != nil || run.ChoiceDecisions != nil
	if present == 0 && !tapePresent {
		return nil
	}
	if present != 4 {
		return errors.New("choice trace identity is incomplete")
	}
	if !validRecordSHA256(*run.ChoiceTraceSHA256) || *run.ChoiceTraceBranchingRecords > *run.ChoiceTraceRecords {
		return errors.New("choice trace summary is invalid")
	}
	if *run.ChoiceTraceTerminalState == "complete" {
		if run.ChoiceTapeSHA256 == nil || run.ChoiceDecisions == nil || !validRecordSHA256(*run.ChoiceTapeSHA256) || *run.ChoiceDecisions > *run.ChoiceTraceRecords || *run.ChoiceTraceBranchingRecords > *run.ChoiceDecisions {
			return errors.New("choice tape summary is invalid")
		}
		return nil
	}
	if *run.ChoiceTraceTerminalState == "overflow" && run.Domain == "runner" && run.Reason == "choice_trace_overflow" && run.ChoiceTapeSHA256 == nil && run.ChoiceDecisions == nil {
		return nil
	}
	return errors.New("choice trace summary is invalid")
}

func validArtifactReference(reference string) bool {
	clean := filepath.ToSlash(filepath.Clean(filepath.FromSlash(reference)))
	return clean == reference && (strings.HasPrefix(reference, "failures/sha256-") || validExplorationArtifactReference(reference, "failures")) && !strings.Contains(reference, "..")
}

func validSuccessArtifactReference(reference string) bool {
	clean := filepath.ToSlash(filepath.Clean(filepath.FromSlash(reference)))
	return clean == reference && (strings.HasPrefix(reference, "successes/sha256-") || validExplorationArtifactReference(reference, "successes")) && !strings.Contains(reference, "..")
}

func validExplorationArtifactReference(reference, kind string) bool {
	parts := strings.Split(reference, "/")
	if len(parts) != 5 || parts[0] != "choice-exploration" && parts[0] != "simulation-exploration" || parts[1] != "rounds" || len(parts[2]) != 20 || parts[3] != kind || !strings.HasPrefix(parts[4], "sha256-") {
		return false
	}
	_, err := strconv.ParseUint(parts[2], 10, 64)
	return err == nil
}

func validateSemanticProbeLists(probes, novel []string) error {
	return validateObservedAndNovel("semantic probes", "semantic probe", probes, novel)
}

func validateChoiceFeatureLists(features, novel []string) error {
	return validateObservedAndNovel("choice features", "choice feature", features, novel)
}

func validateObservedAndNovel(collection, item string, observed, novel []string) error {
	if !sort.StringsAreSorted(observed) || !sort.StringsAreSorted(novel) {
		return fmt.Errorf("%s are not sorted", collection)
	}
	observedSet := make(map[string]struct{}, len(observed))
	for index, value := range observed {
		if value == "" || index > 0 && observed[index-1] == value {
			return fmt.Errorf("%s are invalid", collection)
		}
		observedSet[value] = struct{}{}
	}
	for index, value := range novel {
		if value == "" || index > 0 && novel[index-1] == value {
			return fmt.Errorf("novel %s are invalid", collection)
		}
		if _, found := observedSet[value]; !found {
			return fmt.Errorf("novel %s %q was not observed by the execution", item, value)
		}
	}
	return nil
}

func validRecordSHA256(value record.SHA256) bool {
	_, err := record.ParseSHA256(string(value))
	return err == nil
}
