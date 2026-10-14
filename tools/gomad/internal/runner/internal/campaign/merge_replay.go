package campaign

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"

	"go.temporal.io/server/tools/gomad/internal/evidence/record"
	"go.temporal.io/server/tools/gomad/internal/runner/internal/logstore"
)

type mergedCampaignReplay struct {
	campaign *mergedCampaignRecord
	scopes   []*sourceScope
	evidence []mergedEvidenceRecord
}

func (replay *mergedCampaignReplay) consume(entry logstore.Record) error {
	if entry.Kind != kindMergedOpen && replay.campaign == nil {
		return errors.New("merged campaign log does not begin with its envelope")
	}
	switch entry.Kind {
	case kindMergedOpen:
		if replay.campaign != nil {
			return errors.New("merged campaign envelope repeats")
		}
		if err := checkLogFormat(entry.Payload, MergedCampaignLogFormat); err != nil {
			return err
		}
		var campaign mergedCampaignRecord
		if err := logstore.DecodeJSON(entry.Payload, &campaign); err != nil {
			return err
		}
		if campaign.Schema != mergedCampaignSchema || campaign.SchemaVersion != record.SchemaVersion || !validRecordSHA256(campaign.PlanSHA256) || (campaign.Selection == "") != (campaign.SelectionCount == 0) || len(campaign.Shards) == 0 {
			return errors.New("merged campaign record is invalid")
		}
		replay.campaign = &campaign
	case kindMergedSource:
		if len(replay.evidence) != 0 {
			return errors.New("merged source scope follows the merged evidence")
		}
		var source mergedSourceRecord
		if err := logstore.DecodeJSON(entry.Payload, &source); err != nil {
			return err
		}
		next := len(replay.scopes)
		if next >= len(replay.campaign.Shards) || source.Index != replay.campaign.Shards[next].Index {
			return fmt.Errorf("merged source scope %d is not the next shard of the envelope", source.Index)
		}
		replay.scopes = append(replay.scopes, newSourceScope())
	case kindMergedEvidence:
		if uint64(len(replay.evidence)) >= uint64(replay.campaign.Attempted) {
			return errors.New("merged campaign has more evidence records than executions")
		}
		var evidence mergedEvidenceRecord
		if err := logstore.DecodeJSON(entry.Payload, &evidence); err != nil {
			return err
		}
		if count := len(replay.evidence); count != 0 && replay.evidence[count-1].Ordinal >= evidence.Ordinal {
			return errors.New("merged evidence is not in ordinal order")
		}
		replay.evidence = append(replay.evidence, evidence)
	default:
		if len(replay.scopes) == 0 || len(replay.evidence) != 0 {
			return fmt.Errorf("record kind %d is outside a merged source scope", entry.Kind)
		}
		return replay.scopes[len(replay.scopes)-1].consume(entry)
	}
	return nil
}

// executions validates every replayed source scope against its recorded
// provenance and the envelope, resolves its borrowed references under the
// source path, and returns the merged executions in ordinal order with
// their evidence attributions.
func (replay *mergedCampaignReplay) executions() ([]mergedExecution, error) {
	campaign := replay.campaign
	limits := executionJournalLimitsFromPlan(campaign.Limits)
	if _, err := normalizeExecutionJournalLimits(CampaignConfig{Journal: limits}); err != nil || campaign.Limits.CapacityOutcome != CapacityInfrastructureFailure || limits.MaximumExecutions != uint64(campaign.SelectionCount) {
		return nil, mergedLogCorruption(errors.Join(errors.New("merged campaign limits are invalid"), err))
	}
	identity := mergeIdentity{
		planSHA256: campaign.PlanSHA256, selection: campaign.Selection, selectionCount: uint64(campaign.SelectionCount), guidance: campaign.Guidance,
		journal: campaign.Limits, artifacts: campaign.Artifacts,
	}
	runs := make([]mergedExecution, 0, uint64(campaign.Attempted))
	var journalBytes uint64
	for index, scope := range replay.scopes {
		recorded := campaign.Shards[index]
		if !filepath.IsAbs(recorded.Path) || filepath.Clean(recorded.Path) != recorded.Path {
			return nil, mergedLogCorruption(fmt.Errorf("merged source %d path is not a clean absolute path", recorded.Index))
		}
		if err := scope.validatePublishedLog(); err != nil {
			return nil, mergedLogCorruption(fmt.Errorf("merged source %s: %w", recorded.Path, err))
		}
		opened, err := scope.published(recorded.Path)
		if err != nil {
			return nil, fmt.Errorf("merged source %s: %w", recorded.Path, err)
		}
		if err := identity.checkSource(recorded.Path, opened, scope.state); err != nil {
			return nil, mergedLogCorruption(err)
		}
		provenance, err := scope.provenance(recorded.Path, opened)
		if err != nil {
			return nil, err
		}
		if provenance != recorded {
			return nil, mergedLogCorruption(fmt.Errorf("merged source %s does not match its recorded provenance", recorded.Path))
		}
		if uint64(recorded.JournalBytes) > limits.MaximumBytes-journalBytes {
			return nil, mergedLogCorruption(errors.New("merged campaign outcomes exceed the plan's byte bound"))
		}
		journalBytes += uint64(recorded.JournalBytes)
		for _, run := range opened.Executions {
			runs = append(runs, mergedExecution{SourceCampaignID: recorded.CampaignID, Execution: run})
		}
	}
	if uint64(len(runs)) > limits.MaximumExecutions || journalBytes != uint64(campaign.Journal.Bytes) {
		return nil, mergedLogCorruption(errors.New("merged campaign outcomes do not match the plan's execution bounds"))
	}
	sort.Slice(runs, func(i, j int) bool { return runs[i].Execution.SelectionOrdinal < runs[j].Execution.SelectionOrdinal })
	attributed := 0
	for index := range runs {
		if index > 0 && runs[index-1].Execution.SelectionOrdinal == runs[index].Execution.SelectionOrdinal {
			return nil, mergedLogCorruption(fmt.Errorf("campaign selection ordinal is duplicated: %d", runs[index].Execution.SelectionOrdinal))
		}
		if !retainsEvidence(runs[index].Execution) {
			continue
		}
		if attributed >= len(replay.evidence) || replay.evidence[attributed].Ordinal != runs[index].Execution.SelectionOrdinal {
			return nil, mergedLogCorruption(fmt.Errorf("merged execution %d has no evidence record", runs[index].Execution.SelectionOrdinal))
		}
		runs[index].EvidenceSHA256 = replay.evidence[attributed].EvidenceSHA256
		runs[index].Evidence = replay.evidence[attributed].Evidence
		attributed++
	}
	if attributed != len(replay.evidence) {
		return nil, mergedLogCorruption(errors.New("merged evidence record has no execution that retains evidence"))
	}
	return runs, nil
}

// ErrIncompleteLog reports a merged campaign log whose replay ends before
// its envelope and every source scope it declares: a merge publishes its log
// complete, so a shorter or torn log is an invalid store.
var ErrIncompleteLog = errors.New("merged campaign log is incomplete")

type mergedLogCorruptionError struct {
	err error
}

func (err *mergedLogCorruptionError) Error() string {
	return err.err.Error()
}

func (err *mergedLogCorruptionError) Unwrap() error {
	return err.err
}

func (err *mergedLogCorruptionError) Is(target error) bool {
	return target == logstore.ErrCorrupt
}

func mergedLogCorruption(err error) error {
	if err == nil {
		return nil
	}
	var corruption *mergedLogCorruptionError
	if errors.As(err, &corruption) {
		return err
	}
	return &mergedLogCorruptionError{err: err}
}

// IsMergedLogCorruption reports semantic inconsistency between the merged log
// envelope and the complete records it contains. The envelope is the rejected
// record, so inspection reports an empty validated prefix for this class.
func IsMergedLogCorruption(err error) bool {
	var corruption *mergedLogCorruptionError
	return errors.As(err, &corruption)
}

func OpenMergedCampaign(path string) (MergedCampaign, error) {
	opened, _, err := InspectMergedCampaign(path)
	return opened, err
}

// InspectMergedCampaign replays a merged campaign log once without its lock,
// validates it and its borrowed source evidence, and returns the replay
// report with the campaign. It creates, repairs and removes nothing. A failed
// replay returns the report of its validated prefix with the error.
func InspectMergedCampaign(path string) (MergedCampaign, logstore.Report, error) {
	rootInfo, err := os.Lstat(path)
	if err != nil || !rootInfo.IsDir() || rootInfo.Mode()&os.ModeSymlink != 0 || rootInfo.Mode().Perm() != 0o700 {
		return MergedCampaign{}, logstore.Report{}, errors.Join(errors.New("merged campaign path is not a private directory"), err)
	}
	if legacy := legacyCampaignFormat(path); legacy != "" {
		return MergedCampaign{}, logstore.Report{}, newIntegrityError(&UnsupportedFormatError{Format: legacy})
	}
	entries, err := os.ReadDir(path)
	if err != nil {
		return MergedCampaign{}, logstore.Report{}, err
	}
	if len(entries) != 1 || entries[0].Name() != storeDirectory {
		return MergedCampaign{}, logstore.Report{}, newIntegrityError(errors.New("merged campaign inventory is invalid"))
	}
	var replay mergedCampaignReplay
	report, err := logstore.Replay(filepath.Join(path, storeDirectory), campaignLogOptions(nil, false), replay.consume)
	if err != nil {
		return MergedCampaign{}, report, classifyIntegrityError(fmt.Errorf("replay merged campaign log: %w", err))
	}
	if report.Tail != logstore.TailNone || replay.campaign == nil || len(replay.scopes) != len(replay.campaign.Shards) {
		return MergedCampaign{}, report, newIntegrityError(ErrIncompleteLog)
	}
	campaign := *replay.campaign
	runs, err := replay.executions()
	if err != nil {
		return MergedCampaign{}, report, classifyIntegrityError(err)
	}
	if err := validateMergedCampaign(campaign, runs); err != nil {
		return MergedCampaign{}, report, newIntegrityError(mergedLogCorruption(err))
	}
	if err := verifyMergedEvidence(campaign, runs); err != nil {
		return MergedCampaign{}, report, classifyIntegrityError(err)
	}
	return MergedCampaign{Path: path, Record: campaign, Executions: runs}, report, nil
}

// verifyMergedEvidence resolves the evidence the aggregate counts against the
// borrowed artifact in its source shard.
func verifyMergedEvidence(campaign mergedCampaignRecord, runs []mergedExecution) error {
	sources := make(map[string]string, len(campaign.Shards))
	for _, shard := range campaign.Shards {
		sources[shard.CampaignID] = shard.Path
	}
	for _, run := range runs {
		if run.Evidence == nil {
			continue
		}
		resolved, err := retainedMergedEvidence(sources[run.SourceCampaignID], run.SourceCampaignID, run.Execution)
		if err != nil {
			return fmt.Errorf("resolve merged evidence of ordinal %d: %w", run.Execution.SelectionOrdinal, err)
		}
		if resolved != *run.Evidence {
			return fmt.Errorf("merged evidence of ordinal %d does not match its source artifact", run.Execution.SelectionOrdinal)
		}
	}
	return nil
}

func validateMergedCampaign(campaign mergedCampaignRecord, runs []mergedExecution) error {
	missing := missingOrdinalRanges(uint64(campaign.SelectionCount), runs)
	if uint64(len(runs)) != uint64(campaign.Attempted) || !reflect.DeepEqual(missing, campaign.Missing) || campaign.Partial != (len(missing) != 0) || campaign.Journal.Records != campaign.Attempted || campaign.Journal.Schema != MergedCampaignLogFormat || campaign.Format != MergedCampaignLogFormat {
		return errors.New("merged campaign summary is inconsistent")
	}
	if !sort.SliceIsSorted(campaign.Shards, func(i, j int) bool { return campaign.Shards[i].Index < campaign.Shards[j].Index }) {
		return errors.New("merged campaign shards are not sorted")
	}
	campaigns := make(map[string]struct{}, len(campaign.Shards))
	assignments := make(map[uint64]struct{}, len(campaign.Shards))
	var shardCount, attempted uint64
	for _, shard := range campaign.Shards {
		if shard.Count == 0 || shard.Index >= shard.Count || shard.CampaignID == "" || !validRecordSHA256(shard.CampaignSHA256) || !validRecordSHA256(shard.LogSHA256) {
			return errors.New("merged campaign shard identity is invalid")
		}
		if shardCount == 0 {
			shardCount = uint64(shard.Count)
		}
		if uint64(shard.Count) != shardCount {
			return errors.New("merged campaign shard counts differ")
		}
		if _, found := assignments[uint64(shard.Index)]; found {
			return errors.New("merged campaign shard assignment is duplicated")
		}
		if _, found := campaigns[shard.CampaignID]; found {
			return errors.New("merged campaign source is duplicated")
		}
		assignments[uint64(shard.Index)] = struct{}{}
		campaigns[shard.CampaignID] = struct{}{}
		if uint64(shard.Attempted) > ^uint64(0)-attempted {
			return errors.New("merged campaign shard attempts overflow")
		}
		attempted += uint64(shard.Attempted)
	}
	if attempted != uint64(campaign.Attempted) {
		return errors.New("merged campaign shard attempts are inconsistent")
	}
	failures := make(map[record.SHA256]struct{})
	executions := make([]ExecutionRecord, len(runs))
	for index, run := range runs {
		if _, found := campaigns[run.SourceCampaignID]; !found {
			return errors.New("merged execution source campaign is unknown")
		}
		if run.Execution.FailureSignature != nil {
			failures[*run.Execution.FailureSignature] = struct{}{}
			if run.EvidenceSHA256 != *run.Execution.FailureSignature {
				return errors.New("merged failure evidence identity does not match its signature")
			}
		}
		if retainsEvidence(run.Execution) != (run.EvidenceSHA256 != "") || run.EvidenceSHA256 != "" && !validRecordSHA256(run.EvidenceSHA256) {
			return errors.New("merged execution evidence identity is invalid")
		}
		executions[index] = run.Execution
	}
	failureSignatures := make([]record.SHA256, 0, len(failures))
	for signature := range failures {
		failureSignatures = append(failureSignatures, signature)
	}
	sort.Slice(failureSignatures, func(i, j int) bool { return failureSignatures[i] < failureSignatures[j] })
	validationRecord := CampaignRecord{
		Guidance:      campaign.Guidance,
		SchemaVersion: record.SchemaVersion, Schema: "gomad3.campaign/v1", CampaignID: "merged-validation", Strategy: "seed", Selection: campaign.Selection, SelectionCount: campaign.SelectionCount,
		Attempted: campaign.Attempted, Succeeded: campaign.Succeeded, Failures: campaign.Failures, Watchdogs: campaign.Watchdogs, Cancelled: campaign.Cancelled, DistinctFailures: campaign.DistinctFailures,
		RetainedSuccesses: campaign.RetainedSuccesses, RetainedSuccessBytes: campaign.RetainedSuccessBytes, StopReason: "seeds_exhausted", Journal: &campaign.Journal, Artifacts: &campaign.Artifacts, FailureSignatures: failureSignatures,
	}
	// The success limit of a campaign bounds the success bytes its executions
	// report, where every artifact counts in full. A merged campaign bounds its
	// retained evidence below instead, where a shared target counts once, so
	// here that sum only has to match its executions.
	if err := validateCampaign(validationRecord, executions, executionSuccessBytesUnbounded); err != nil {
		return fmt.Errorf("validate merged executions: %w", err)
	}
	if uint64(campaign.DistinctFailures) != uint64(len(failures)) {
		return errors.New("merged evidence summary is invalid")
	}
	evidenceByIdentity := make(map[record.SHA256]struct{}, uint64(campaign.RetainedEvidence))
	var failureEvidence, successEvidence mergedEvidenceBytes
	for _, mergedRun := range runs {
		if mergedRun.Evidence == nil {
			continue
		}
		retained := *mergedRun.Evidence
		if !validRecordSHA256(retained.SHA256) || !validRecordSHA256(retained.RecordSHA256) || retained.SourceCampaignID == "" || retained.Reference == "" || retained.StoredBytes == 0 || retained.TargetSHA256 != "" && !validRecordSHA256(retained.TargetSHA256) {
			return errors.New("merged evidence identity is invalid")
		}
		if retained.SHA256 != mergedRun.EvidenceSHA256 || retained.SourceCampaignID != mergedRun.SourceCampaignID {
			return errors.New("merged evidence is attached to the wrong execution")
		}
		if retained.Reference != retainedReferenceOf(mergedRun.Execution) {
			return errors.New("merged evidence reference does not match its execution")
		}
		if _, duplicate := evidenceByIdentity[retained.SHA256]; duplicate {
			return errors.New("merged evidence identity is duplicated")
		}
		evidenceByIdentity[retained.SHA256] = struct{}{}
		if retained.Kind == record.ArtifactSuccess {
			if !validSuccessArtifactReference(retained.Reference) {
				return errors.New("merged success evidence reference is invalid")
			}
			if err := successEvidence.add(retained, artifactLimitSuccessBytes, uint64(campaign.Artifacts.SuccessBytes)); err != nil {
				return fmt.Errorf("merged success evidence bytes: %w", err)
			}
		} else {
			if retained.Kind != record.ArtifactTargetFailure && retained.Kind != record.ArtifactWatchdogTimeout && retained.Kind != record.ArtifactRunnerFailure || !validArtifactReference(retained.Reference) {
				return errors.New("merged failure evidence reference is invalid")
			}
			if err := failureEvidence.add(retained, ArtifactLimitFailureBytes, uint64(campaign.Artifacts.FailureBytes)); err != nil {
				return fmt.Errorf("merged failure evidence bytes: %w", err)
			}
		}
	}
	failureCount, failureBytes := failureEvidence.count, failureEvidence.retained.Total()
	successCount, successBytes := successEvidence.count, successEvidence.retained.Total()
	if uint64(len(evidenceByIdentity)) != uint64(campaign.RetainedEvidence) {
		return errors.New("merged retained evidence count is inconsistent")
	}
	for _, run := range runs {
		if run.EvidenceSHA256 != "" {
			if _, found := evidenceByIdentity[run.EvidenceSHA256]; !found {
				return errors.New("merged execution evidence is not retained")
			}
		}
	}
	if err := validateMergedArtifactCapacity(campaign.Artifacts, failureCount, failureBytes, successCount, successBytes); err != nil {
		return err
	}
	if successBytes > ^uint64(0)-failureBytes || failureBytes+successBytes != uint64(campaign.EvidenceBytes) {
		return errors.New("merged evidence byte summary is inconsistent")
	}
	return nil
}
