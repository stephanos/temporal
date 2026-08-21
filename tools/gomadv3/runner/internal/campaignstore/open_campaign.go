package campaignstore

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"go.temporal.io/server/tools/gomadv3/evidence"
	"go.temporal.io/server/tools/gomadv3/runner/internal/combinedfrontier"
	"go.temporal.io/server/tools/gomadv3/runner/internal/frontier"
)

const maximumRunsBytes = 64 << 20

type Campaign struct {
	Path    string
	Record  CampaignRecord
	Runs    []ExecutionRecord
	Journal *RunJournalInfo
}

type RunJournalInfo struct {
	Schema      string
	IndexSHA256 evidence.SHA256
	Segments    uint64
	Records     uint64
	Bytes       uint64
	Limits      RunJournalPlan
}

func OpenCampaign(path string) (Campaign, error) {
	rootInfo, err := os.Lstat(path)
	if err != nil {
		return Campaign{}, fmt.Errorf("open batch directory: %w", err)
	}
	if !rootInfo.IsDir() || rootInfo.Mode()&os.ModeSymlink != 0 {
		return Campaign{}, fmt.Errorf("batch path is not a directory")
	}
	if rootInfo.Mode().Perm() != 0o700 {
		return Campaign{}, fmt.Errorf("batch directory mode is %#o, want 0700", rootInfo.Mode().Perm())
	}
	root, err := os.OpenRoot(path)
	if err != nil {
		return Campaign{}, fmt.Errorf("pin batch directory: %w", err)
	}
	defer root.Close()
	pinnedInfo, err := root.Stat(".")
	if err != nil || !os.SameFile(rootInfo, pinnedInfo) {
		return Campaign{}, errors.Join(fmt.Errorf("batch directory changed while opening"), err)
	}
	batchBytes, err := readValidatedFile(root, "batch.json", 0o600, maximumManifestBytes)
	if err != nil {
		return Campaign{}, fmt.Errorf("read batch record: %w", err)
	}
	var batch CampaignRecord
	if err := evidence.DecodeCanonicalJSON(batchBytes, &batch); err != nil {
		return Campaign{}, fmt.Errorf("decode batch record: %w", err)
	}
	var runs []ExecutionRecord
	var journal *RunJournalInfo
	if batch.Schema == "gomadv3.batch/v3" || batch.Schema == "gomadv3.batch/v4" || batch.Schema == "gomadv3.batch/v5" {
		runs, journal, err = readPublishedRunJournal(root, batch)
	} else {
		runs, err = readLegacyPublishedRuns(root, batch)
	}
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
	if batch.Strategy == "choice-frontier" {
		if err := ValidatePublishedFrontier(path, *batch.Frontier, batch.FrontierImplementationSHA256, batch.FrontierChainSHA256, runs); err != nil {
			return Campaign{}, fmt.Errorf("validate published frontier: %w", err)
		}
	}
	if batch.Strategy == "combined-frontier" {
		if err := ValidatePublishedCombinedFrontier(path, *batch.CombinedFrontier, batch.CombinedFrontierImplementationSHA256, batch.CombinedFrontierChainSHA256, runs); err != nil {
			return Campaign{}, fmt.Errorf("validate published combined frontier: %w", err)
		}
	}
	if batch.Schema == "gomadv3.batch/v1" {
		batch.Strategy = "seed"
	}
	return Campaign{Path: path, Record: batch, Runs: runs, Journal: journal}, nil
}

func decodeExecutions(contents []byte) ([]ExecutionRecord, error) {
	if len(contents) == 0 {
		return []ExecutionRecord{}, nil
	}
	if contents[len(contents)-1] != '\n' {
		return nil, fmt.Errorf("batch runs journal is not newline terminated")
	}
	lines := bytes.Split(contents[:len(contents)-1], []byte{'\n'})
	runs := make([]ExecutionRecord, len(lines))
	for index, line := range lines {
		if len(line) == 0 {
			return nil, fmt.Errorf("batch runs journal has an empty record at line %d", index+1)
		}
		if err := evidence.DecodeCanonicalJSON(line, &runs[index]); err != nil {
			return nil, fmt.Errorf("decode batch run %d: %w", index+1, err)
		}
	}
	return runs, nil
}

func validateCampaign(batch CampaignRecord, runs []ExecutionRecord) error {
	legacy := batch.Schema == "gomadv3.batch/v1"
	segmented := batch.Schema == "gomadv3.batch/v3" || batch.Schema == "gomadv3.batch/v4" || batch.Schema == "gomadv3.batch/v5"
	if batch.SchemaVersion != evidence.SchemaVersion || !legacy && batch.Schema != "gomadv3.batch/v2" && !segmented || batch.CampaignID == "" || batch.Selection == "" || batch.SelectionCount == 0 {
		return fmt.Errorf("batch record identity is invalid")
	}
	if batch.Schema == "gomadv3.batch/v4" {
		if !validRecordSHA256(batch.PlanSHA256) || batch.Shard == nil || batch.Shard.Count == 0 || batch.Shard.Index >= batch.Shard.Count || batch.Strategy != "seed" {
			return errors.New("sharded batch identity is invalid")
		}
	} else if batch.PlanSHA256 != "" || batch.Shard != nil {
		return errors.New("historical batch contains shard identity")
	}
	if batch.Schema == "gomadv3.batch/v5" && batch.Strategy != "combined-frontier" {
		return errors.New("combined frontier batch schema has another strategy")
	}
	if segmented {
		if batch.Journal == nil || batch.RunsSHA256 != "" {
			return errors.New("segmented batch journal identity is invalid")
		}
	} else if batch.Journal != nil || batch.Artifacts != nil || !validRecordSHA256(batch.RunsSHA256) {
		return errors.New("legacy batch journal identity is invalid")
	}
	if legacy {
		if batch.Strategy != "" || batch.Frontier != nil || batch.FrontierImplementationSHA256 != "" || batch.FrontierChainSHA256 != "" || batch.CombinedFrontier != nil || batch.CombinedFrontierImplementationSHA256 != "" || batch.CombinedFrontierChainSHA256 != "" || batch.RecoveryExecutions != 0 {
			return fmt.Errorf("legacy batch record contains strategy evidence")
		}
		batch.Strategy = "seed"
	}
	choiceEvidence := batch.Frontier != nil || batch.FrontierImplementationSHA256 != "" || batch.FrontierChainSHA256 != ""
	combinedEvidence := batch.CombinedFrontier != nil || batch.CombinedFrontierImplementationSHA256 != "" || batch.CombinedFrontierChainSHA256 != ""
	if batch.Strategy != "seed" && batch.Strategy != "choice-frontier" && batch.Strategy != "combined-frontier" ||
		batch.Strategy == "seed" && (choiceEvidence || combinedEvidence) ||
		batch.Strategy == "choice-frontier" && (batch.Frontier == nil || batch.FrontierImplementationSHA256 != frontier.ImplementationSHA256() || !validRecordSHA256(batch.FrontierChainSHA256) || combinedEvidence) ||
		batch.Strategy == "combined-frontier" && (batch.CombinedFrontier == nil || batch.CombinedFrontierImplementationSHA256 != combinedfrontier.ImplementationSHA256() || !validRecordSHA256(batch.CombinedFrontierChainSHA256) || choiceEvidence) {
		return fmt.Errorf("batch strategy evidence is invalid")
	}
	if uint64(batch.Attempted) != uint64(len(runs)) || uint64(batch.Succeeded)+uint64(batch.Failures)+uint64(batch.Cancelled) != uint64(batch.Attempted) || batch.Watchdogs > batch.Failures || batch.RetainedSuccesses > batch.Succeeded || batch.RetainedSuccesses == 0 && batch.RetainedSuccessBytes != 0 {
		return fmt.Errorf("batch summary counts are inconsistent")
	}
	if limits := batch.Artifacts; limits != nil {
		failureBytes := uint64(limits.FailureBytes)
		successBytes := uint64(limits.SuccessBytes)
		if limits.FailureOutcome != CapacityInfrastructureFailure || limits.SuccessOutcome != CapacityInfrastructureFailure || limits.FailureArtifacts == 0 || failureBytes == 0 || limits.TranscriptBytes == 0 || uint64(limits.TotalBytes) != failureBytes+successBytes || uint64(limits.TotalBytes) < failureBytes ||
			batch.DistinctFailures > limits.FailureArtifacts || batch.RetainedSuccesses > limits.SuccessArtifacts || batch.RetainedSuccessBytes > limits.SuccessBytes {
			return errors.New("batch artifact capacity is invalid")
		}
	}
	if batch.StopReason != "seeds_exhausted" && batch.StopReason != "first_failure" && batch.StopReason != "failure_budget" && batch.StopReason != "frontier_exhausted" && batch.StopReason != "choice_depth_complete" && batch.StopReason != "combined_depth_complete" && batch.StopReason != "dimension_depth_complete" && batch.StopReason != "max_runs" && batch.StopReason != "frontier_capacity" {
		return fmt.Errorf("batch stop reason is invalid: %s", batch.StopReason)
	}
	ordinals := make(map[uint64]struct{}, len(runs))
	candidates := make(map[evidence.SHA256]struct{}, len(runs))
	failures := make(map[evidence.SHA256]struct{})
	var succeeded, failed, watchdogs, cancelled, retainedSuccesses, retainedSuccessBytes uint64
	for index, run := range runs {
		ordinal := uint64(run.SelectionOrdinal)
		ordinalLimit := uint64(batch.SelectionCount)
		if batch.Strategy == "choice-frontier" || batch.Strategy == "combined-frontier" {
			ordinalLimit = uint64(batch.Attempted)
		}
		if ordinal >= ordinalLimit {
			return fmt.Errorf("batch run %d selection ordinal is out of range", index+1)
		}
		if batch.Shard != nil && ordinal%uint64(batch.Shard.Count) != uint64(batch.Shard.Index) {
			return fmt.Errorf("batch run %d is outside shard assignment", index+1)
		}
		if _, duplicate := ordinals[ordinal]; duplicate {
			return fmt.Errorf("batch selection ordinal is duplicated: %d", ordinal)
		}
		if batch.Strategy == "choice-frontier" {
			baseSeed, parseErr := strconv.ParseUint(batch.Selection, 10, 64)
			if parseErr != nil || uint64(run.Seed) != baseSeed || ordinal != uint64(index) {
				return fmt.Errorf("batch frontier run %d seed or logical ordinal is invalid", index+1)
			}
			if err := validateFrontierExecutionSummary(run, candidates); err != nil {
				return fmt.Errorf("batch frontier run %d: %w", index+1, err)
			}
		} else if batch.Strategy == "combined-frontier" {
			baseSeed, parseErr := strconv.ParseUint(batch.Selection, 10, 64)
			if parseErr != nil || uint64(run.Seed) != baseSeed || ordinal != uint64(index) {
				return fmt.Errorf("batch combined frontier run %d seed or logical ordinal is invalid", index+1)
			}
			if err := validateCombinedFrontierExecutionSummary(run, candidates); err != nil {
				return fmt.Errorf("batch combined frontier run %d: %w", index+1, err)
			}
		} else if run.Strategy != "" {
			return fmt.Errorf("seed batch run %d contains strategy evidence", index+1)
		}
		ordinals[ordinal] = struct{}{}
		if run.Reason == "" {
			return fmt.Errorf("batch run %d identity is invalid", index+1)
		}
		if (run.IOTranscriptSHA256 == nil) != (run.IOTranscriptRecords == nil) {
			return fmt.Errorf("batch run %d transcript identity is incomplete", index+1)
		}
		if run.IOTranscriptSHA256 != nil {
			if _, err := evidence.ParseSHA256(string(*run.IOTranscriptSHA256)); err != nil {
				return fmt.Errorf("batch run %d transcript digest is invalid", index+1)
			}
		}
		if err := validateChoiceExecutionSummary(run); err != nil {
			return fmt.Errorf("batch run %d: %w", index+1, err)
		}
		if err := validateSemanticProbeLists(run.SemanticProbes, run.NovelSemanticProbes); err != nil {
			return fmt.Errorf("batch run %d: %w", index+1, err)
		}
		if err := validateChoiceFeatureLists(run.ChoiceFeatures, run.NovelChoiceFeatures); err != nil {
			return fmt.Errorf("batch run %d: %w", index+1, err)
		}
		switch run.Domain {
		case "success":
			succeeded++
			if run.Termination != "exit" || run.FailureSignature != nil || run.Artifact != nil {
				return fmt.Errorf("successful batch run %d has failure evidence", index+1)
			}
			if (run.SuccessArtifact == nil) != (run.SuccessArtifactBytes == nil) {
				return fmt.Errorf("successful batch run %d has incomplete retained evidence", index+1)
			}
			if run.SuccessArtifact == nil {
				if len(run.NovelSemanticProbes) != 0 || len(run.NovelChoiceFeatures) != 0 {
					return fmt.Errorf("unretained successful batch run %d has novelty reasons", index+1)
				}
			} else if !validSuccessArtifactReference(*run.SuccessArtifact) || *run.SuccessArtifactBytes == 0 {
				return fmt.Errorf("successful batch run %d has invalid retained evidence", index+1)
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
				return fmt.Errorf("failed batch run %d has invalid artifact evidence", index+1)
			}
			failures[*run.FailureSignature] = struct{}{}
			if run.SuccessArtifact != nil || run.SuccessArtifactBytes != nil || len(run.NovelSemanticProbes) != 0 || len(run.NovelChoiceFeatures) != 0 {
				return fmt.Errorf("failed batch run %d has successful-run evidence", index+1)
			}
		case "runner":
			cancelled++
			if run.Reason != "runner_cancelled" || run.Termination != "none" || run.FailureSignature != nil || run.Artifact != nil {
				return fmt.Errorf("cancelled batch run %d is invalid", index+1)
			}
			if run.SuccessArtifact != nil || run.SuccessArtifactBytes != nil || len(run.NovelSemanticProbes) != 0 || len(run.NovelChoiceFeatures) != 0 {
				return fmt.Errorf("cancelled batch run %d has successful-run evidence", index+1)
			}
		default:
			return fmt.Errorf("batch run %d domain is invalid: %s", index+1, run.Domain)
		}
	}
	if succeeded != uint64(batch.Succeeded) || failed != uint64(batch.Failures) || watchdogs != uint64(batch.Watchdogs) || cancelled != uint64(batch.Cancelled) {
		return fmt.Errorf("batch run counts do not match the summary")
	}
	if retainedSuccesses != uint64(batch.RetainedSuccesses) || retainedSuccessBytes != uint64(batch.RetainedSuccessBytes) {
		return fmt.Errorf("batch retained success counts do not match the summary")
	}
	if uint64(len(failures)) != uint64(batch.DistinctFailures) || len(batch.FailureSignatures) != len(failures) {
		return fmt.Errorf("batch distinct failure count is inconsistent")
	}
	if !sort.SliceIsSorted(batch.FailureSignatures, func(i, j int) bool { return batch.FailureSignatures[i] < batch.FailureSignatures[j] }) {
		return fmt.Errorf("batch failure signatures are not sorted")
	}
	for index, signature := range batch.FailureSignatures {
		if !validRecordSHA256(signature) || index > 0 && batch.FailureSignatures[index-1] == signature {
			return fmt.Errorf("batch failure signatures are invalid")
		}
		if _, found := failures[signature]; !found {
			return fmt.Errorf("batch failure signature has no run: %s", signature)
		}
	}
	return nil
}

func validatePublishedArtifactCapacity(batchPath string, batch CampaignRecord, runs []ExecutionRecord) error {
	limits := *batch.Artifacts
	seenFailures := make(map[evidence.SHA256]struct{}, uint64(batch.DistinctFailures))
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
		return errors.New("published artifacts exceed the batch capacity")
	}
	return nil
}

func validateFrontierExecutionSummary(run ExecutionRecord, candidates map[evidence.SHA256]struct{}) error {
	if run.Strategy != "choice-frontier" || run.Round == nil || run.ForcedDepth == nil || !validRecordSHA256(run.CandidateSHA256) || !validRecordSHA256(run.OutcomeSHA256) {
		return errors.New("frontier identity is incomplete")
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

func validateCombinedFrontierExecutionSummary(run ExecutionRecord, candidates map[evidence.SHA256]struct{}) error {
	if run.Strategy != "combined-frontier" || run.Round == nil || run.ForcedDepth == nil || !validRecordSHA256(run.CandidateSHA256) || !validRecordSHA256(run.OutcomeSHA256) || run.PrefixSHA256 != "" {
		return errors.New("combined frontier identity is incomplete")
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
	return clean == reference && (strings.HasPrefix(reference, "failures/sha256-") || validFrontierArtifactReference(reference, "failures")) && !strings.Contains(reference, "..")
}

func validSuccessArtifactReference(reference string) bool {
	clean := filepath.ToSlash(filepath.Clean(filepath.FromSlash(reference)))
	return clean == reference && (strings.HasPrefix(reference, "successes/sha256-") || validFrontierArtifactReference(reference, "successes")) && !strings.Contains(reference, "..")
}

func validFrontierArtifactReference(reference, kind string) bool {
	parts := strings.Split(reference, "/")
	if len(parts) != 5 || parts[0] != "frontier" && parts[0] != "combined-frontier" || parts[1] != "rounds" || len(parts[2]) != 20 || parts[3] != kind || !strings.HasPrefix(parts[4], "sha256-") {
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
			return fmt.Errorf("novel %s %q was not observed by the run", item, value)
		}
	}
	return nil
}

func validRecordSHA256(value evidence.SHA256) bool {
	_, err := evidence.ParseSHA256(string(value))
	return err == nil
}
