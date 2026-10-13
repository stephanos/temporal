package campaign

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"

	"go.temporal.io/server/tools/gomad3/artifact"
	"go.temporal.io/server/tools/gomad3/internal/canonicaljson"
	"go.temporal.io/server/tools/gomad3/record"
	"go.temporal.io/server/tools/gomad3/runner/internal/logstore"
)

const (
	mergedCampaignSchema       = "gomad3.merged-campaign/v1"
	maximumMergedManifestBytes = 16 << 20
)

type expectedSeedFunc func(uint64) (uint64, bool)

type MergeSpec struct {
	Output         string
	PlanSHA256     record.SHA256
	Selection      string
	SelectionCount uint64
	Guidance       *GuidancePlan
	Journal        ExecutionJournalPlan
	Artifacts      ArtifactCapacityPlan
	Partial        bool
	ShardPaths     []string
	SeedAt         expectedSeedFunc
}

type mergedShard struct {
	Index          record.Uint64String `json:"index"`
	Count          record.Uint64String `json:"count"`
	CampaignID     string              `json:"campaign_id"`
	CampaignSHA256 record.SHA256       `json:"campaign_sha256"`
	Attempted      record.Uint64String `json:"attempted"`
	JournalBytes   record.Uint64String `json:"journal_bytes"`
}

type mergedExecution struct {
	SourceCampaignID string          `json:"source_campaign_id"`
	EvidenceSHA256   record.SHA256   `json:"evidence_sha256,omitempty"`
	Evidence         *mergedEvidence `json:"evidence,omitempty"`
	Execution        ExecutionRecord `json:"execution"`
}

type mergedEvidence struct {
	SHA256           record.SHA256       `json:"sha256"`
	Kind             string              `json:"kind"`
	SourceCampaignID string              `json:"source_campaign_id"`
	Reference        string              `json:"reference"`
	RecordSHA256     record.SHA256       `json:"record_sha256"`
	StoredBytes      record.Uint64String `json:"stored_bytes"`
	// TargetSHA256 and TargetBytes name the part of StoredBytes that is the
	// target. A merged campaign accounts for its evidence as one store holds
	// it: under the stored-bytes rule of artifact.RetainedBytes, each distinct
	// target counts once in the failure bytes and once in the success bytes,
	// whichever shard store the evidence came from. A record written before
	// targets were shared has neither field and counts every artifact in full.
	TargetSHA256 record.SHA256       `json:"target_sha256,omitempty"`
	TargetBytes  record.Uint64String `json:"target_bytes,omitempty"`
}

func (evidence mergedEvidence) sharedTarget() artifact.SharedTarget {
	return artifact.SharedTarget{SHA256: evidence.TargetSHA256, Bytes: uint64(evidence.TargetBytes)}
}

// mergedEvidenceBytes sums the retained evidence of one kind.
type mergedEvidenceBytes struct {
	count    uint64
	retained artifact.RetainedBytes
}

func (bytes *mergedEvidenceBytes) add(evidence mergedEvidence, limit artifactLimit, maximum uint64) error {
	cost, err := bytes.retained.Cost(uint64(evidence.StoredBytes), evidence.sharedTarget())
	if err != nil {
		return err
	}
	if _, err := checkedMergedEvidenceBytes(bytes.retained.Total(), cost, limit, maximum); err != nil {
		return err
	}
	if _, err := bytes.retained.Add(uint64(evidence.StoredBytes), evidence.sharedTarget()); err != nil {
		return err
	}
	bytes.count++
	return nil
}

type ordinalRange struct {
	Start record.Uint64String `json:"start"`
	End   record.Uint64String `json:"end"`
}

// mergedCampaignRecord is the envelope of a merged campaign log. Its executions
// follow it, one record each, in ordinal order.
type mergedCampaignRecord struct {
	Format               string                    `json:"format"`
	Guidance             *GuidancePlan             `json:"guidance,omitempty"`
	Schema               string                    `json:"schema"`
	SchemaVersion        uint32                    `json:"schema_version"`
	PlanSHA256           record.SHA256             `json:"plan_sha256"`
	Selection            string                    `json:"selection"`
	SelectionCount       record.Uint64String       `json:"selection_count"`
	Partial              bool                      `json:"partial"`
	Missing              []ordinalRange            `json:"missing"`
	Attempted            record.Uint64String       `json:"attempted"`
	Succeeded            record.Uint64String       `json:"succeeded"`
	Failures             record.Uint64String       `json:"failures"`
	Watchdogs            record.Uint64String       `json:"watchdogs"`
	Cancelled            record.Uint64String       `json:"cancelled"`
	DistinctFailures     record.Uint64String       `json:"distinct_failures"`
	RetainedSuccesses    record.Uint64String       `json:"retained_successes"`
	RetainedSuccessBytes record.Uint64String       `json:"retained_success_bytes"`
	RetainedEvidence     record.Uint64String       `json:"retained_evidence"`
	EvidenceBytes        record.Uint64String       `json:"evidence_bytes"`
	Journal              executionJournalReference `json:"journal"`
	Artifacts            ArtifactCapacityPlan      `json:"artifacts"`
	Shards               []mergedShard             `json:"shards"`
}

type MergedCampaign struct {
	Path       string
	Record     mergedCampaignRecord
	Executions []mergedExecution
}

func MergeCampaigns(ctx context.Context, spec MergeSpec) (_ MergedCampaign, retErr error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if spec.Output == "" || (spec.Selection == "") != (spec.SelectionCount == 0) || spec.SelectionCount == 0 && !isFullyAnsweredGuidance(spec.Guidance) || spec.SeedAt == nil || !validRecordSHA256(spec.PlanSHA256) || len(spec.ShardPaths) == 0 {
		return MergedCampaign{}, errors.New("campaign merge input is incomplete")
	}
	limits := executionJournalLimitsFromPlan(spec.Journal)
	if _, err := normalizeExecutionJournalLimits(CampaignConfig{Journal: limits}); err != nil || limits.MaximumExecutions != spec.SelectionCount {
		return MergedCampaign{}, errors.Join(errors.New("campaign merge journal capacity is invalid"), err)
	}
	output, err := filepath.Abs(spec.Output)
	if err != nil {
		return MergedCampaign{}, fmt.Errorf("resolve merged campaign output: %w", err)
	}
	if _, err := os.Lstat(output); err == nil {
		return MergedCampaign{}, fmt.Errorf("merged campaign output already exists")
	} else if !errors.Is(err, os.ErrNotExist) {
		return MergedCampaign{}, err
	}
	shards, runs, summary, err := collectMergedCampaign(spec)
	if err != nil {
		return MergedCampaign{}, err
	}
	missing := missingOrdinalRanges(spec.SelectionCount, runs)
	if len(missing) != 0 && !spec.Partial {
		return MergedCampaign{}, fmt.Errorf("campaign merge is missing %d ordinal ranges", len(missing))
	}
	if err := os.MkdirAll(filepath.Dir(output), 0o700); err != nil {
		return MergedCampaign{}, fmt.Errorf("create merged campaign parent: %w", err)
	}
	staging, err := os.MkdirTemp(filepath.Dir(output), ".gomad-merge-")
	if err != nil {
		return MergedCampaign{}, fmt.Errorf("create merged campaign staging directory: %w", err)
	}
	defer func() { retErr = errors.Join(retErr, os.RemoveAll(staging)) }()
	if err := os.Chmod(staging, 0o700); err != nil {
		return MergedCampaign{}, err
	}
	reference, err := mergedJournalReference(limits, runs)
	if err != nil {
		return MergedCampaign{}, err
	}
	record := mergedCampaignRecord{
		Format: MergedCampaignLogFormat, Guidance: spec.Guidance,
		Schema: mergedCampaignSchema, SchemaVersion: record.SchemaVersion, PlanSHA256: spec.PlanSHA256, Selection: spec.Selection, SelectionCount: record.Uint64String(spec.SelectionCount),
		Partial: len(missing) != 0, Missing: missing, Attempted: record.Uint64String(len(runs)), Succeeded: record.Uint64String(summary.succeeded), Failures: record.Uint64String(summary.failures), Watchdogs: record.Uint64String(summary.watchdogs), Cancelled: record.Uint64String(summary.cancelled),
		DistinctFailures: record.Uint64String(len(summary.failuresSeen)), RetainedSuccesses: record.Uint64String(summary.retainedSuccesses), RetainedSuccessBytes: record.Uint64String(summary.retainedSuccessBytes), RetainedEvidence: record.Uint64String(summary.retainedEvidence), EvidenceBytes: record.Uint64String(summary.evidenceBytes),
		Journal: reference, Artifacts: spec.Artifacts, Shards: shards,
	}
	manifest, err := logstore.EncodeJSON(record)
	if err != nil {
		return MergedCampaign{}, err
	}
	if len(manifest) > maximumMergedManifestBytes {
		return MergedCampaign{}, &JournalCapacityError{Limit: journalLimitManifestBytes, Required: uint64(len(manifest)), Maximum: maximumMergedManifestBytes, Outcome: CapacityInfrastructureFailure}
	}
	if err := writeMergedCampaignLog(ctx, filepath.Join(staging, storeDirectory), record, runs); err != nil {
		return MergedCampaign{}, err
	}
	if err := syncDirectoryContext(ctx, staging); err != nil {
		return MergedCampaign{}, err
	}
	if err := observeMutation(ctx, mutationRename, "merged-campaign"); err != nil {
		return MergedCampaign{}, err
	}
	if err := artifact.RenameNoReplace(staging, output); err != nil {
		return MergedCampaign{}, fmt.Errorf("publish merged campaign: %w", err)
	}
	if err := syncDirectoryContext(ctx, filepath.Dir(output)); err != nil {
		return MergedCampaign{}, err
	}
	return MergedCampaign{Path: output, Record: record, Executions: runs}, nil
}

type mergedSummary struct {
	succeeded, failures, watchdogs, cancelled uint64
	retainedSuccesses, retainedSuccessBytes   uint64
	retainedEvidence                          uint64
	evidenceBytes                             uint64
	failuresSeen                              map[record.SHA256]struct{}
}

func collectMergedCampaign(spec MergeSpec) ([]mergedShard, []mergedExecution, mergedSummary, error) {
	paths := append([]string(nil), spec.ShardPaths...)
	sort.Strings(paths)
	shards := make([]mergedShard, 0, len(paths))
	runs := make([]mergedExecution, 0)
	retainedByIdentity := make(map[record.SHA256]mergedEvidence)
	assignments := make(map[uint64]struct{}, len(paths))
	ordinals := make(map[uint64]struct{})
	summary := mergedSummary{failuresSeen: make(map[record.SHA256]struct{})}
	var shardCount uint64
	for _, path := range paths {
		opened, err := OpenCampaign(path)
		if err != nil {
			return nil, nil, mergedSummary{}, err
		}
		batch := opened.Record
		if batch.Schema != "gomad3.campaign/v1" || batch.PlanSHA256 != spec.PlanSHA256 || batch.Selection != spec.Selection || uint64(batch.SelectionCount) != spec.SelectionCount || batch.Shard == nil || !reflect.DeepEqual(batch.Guidance, spec.Guidance) {
			return nil, nil, mergedSummary{}, fmt.Errorf("shard campaign %s does not match the campaign plan", path)
		}
		count := uint64(batch.Shard.Count)
		index := uint64(batch.Shard.Index)
		if shardCount == 0 {
			shardCount = count
		}
		if count != shardCount {
			return nil, nil, mergedSummary{}, errors.New("campaign shards use different partition counts")
		}
		if _, duplicate := assignments[index]; duplicate {
			return nil, nil, mergedSummary{}, fmt.Errorf("campaign shard %d/%d is duplicated", index, count)
		}
		assignments[index] = struct{}{}
		batchBytes, err := logstore.EncodeJSON(batch)
		if err != nil {
			return nil, nil, mergedSummary{}, err
		}
		journalBytes := uint64(0)
		if opened.Journal != nil {
			journalBytes = opened.Journal.Bytes
		}
		shards = append(shards, mergedShard{Index: batch.Shard.Index, Count: batch.Shard.Count, CampaignID: batch.CampaignID, CampaignSHA256: record.HashBytes(batchBytes), Attempted: batch.Attempted, JournalBytes: record.Uint64String(journalBytes)})
		for _, run := range opened.Executions {
			ordinal := uint64(run.SelectionOrdinal)
			seed, found := spec.SeedAt(ordinal)
			if !found || seed != uint64(run.Seed) || ordinal%count != index {
				return nil, nil, mergedSummary{}, fmt.Errorf("shard execution ordinal %d does not match the campaign plan", ordinal)
			}
			if _, duplicate := ordinals[ordinal]; duplicate {
				return nil, nil, mergedSummary{}, fmt.Errorf("campaign selection ordinal is duplicated: %d", ordinal)
			}
			ordinals[ordinal] = struct{}{}
			merged := mergedExecution{SourceCampaignID: batch.CampaignID, Execution: run}
			if run.FailureSignature != nil || run.SuccessArtifact != nil {
				resolved, err := ResolveRetainedEvidence(opened.Path, batch.CampaignID, run)
				if err != nil {
					return nil, nil, mergedSummary{}, err
				}
				identity, err := mergedEvidenceIdentity(resolved.Manifest)
				if err != nil {
					return nil, nil, mergedSummary{}, err
				}
				merged.EvidenceSHA256 = identity
				var reference string
				if run.SuccessArtifact != nil {
					reference = *run.SuccessArtifact
				} else {
					reference = *run.Artifact
				}
				target := artifact.TargetOf(resolved.Manifest)
				candidate := mergedEvidence{
					SHA256: identity, Kind: resolved.Manifest.ArtifactKind, SourceCampaignID: batch.CampaignID, Reference: reference, RecordSHA256: resolved.Manifest.RecordHash, StoredBytes: record.Uint64String(resolved.StoredBytes),
					TargetSHA256: target.SHA256, TargetBytes: record.Uint64String(target.Bytes),
				}
				if prior, found := retainedByIdentity[identity]; !found || mergedEvidenceLess(candidate, prior) {
					retainedByIdentity[identity] = candidate
				}
			}
			switch run.Domain {
			case "success":
				summary.succeeded++
				if run.SuccessArtifact != nil {
					summary.retainedSuccesses++
					updated, err := checkedMergedEvidenceBytes(summary.retainedSuccessBytes, uint64(*run.SuccessArtifactBytes), artifactLimitSuccessBytes, uint64(spec.Artifacts.SuccessBytes))
					if err != nil {
						return nil, nil, mergedSummary{}, err
					}
					summary.retainedSuccessBytes = updated
				}
			case "target", "watchdog":
				summary.failures++
				if run.Domain == "watchdog" {
					summary.watchdogs++
				}
				summary.failuresSeen[*run.FailureSignature] = struct{}{}
			case "runner":
				summary.cancelled++
			}
			runs = append(runs, merged)
		}
	}
	sort.Slice(shards, func(i, j int) bool { return shards[i].Index < shards[j].Index })
	sort.Slice(runs, func(i, j int) bool { return runs[i].Execution.SelectionOrdinal < runs[j].Execution.SelectionOrdinal })
	retained := make([]mergedEvidence, 0, len(retainedByIdentity))
	var failures, successes mergedEvidenceBytes
	var err error
	for _, item := range retainedByIdentity {
		retained = append(retained, item)
		if item.Kind == record.ArtifactSuccess {
			err = successes.add(item, artifactLimitSuccessBytes, uint64(spec.Artifacts.SuccessBytes))
		} else {
			err = failures.add(item, ArtifactLimitFailureBytes, uint64(spec.Artifacts.FailureBytes))
		}
		if err != nil {
			return nil, nil, mergedSummary{}, err
		}
	}
	failureBytes, successBytes := failures.retained.Total(), successes.retained.Total()
	sort.Slice(retained, func(i, j int) bool { return retained[i].SHA256 < retained[j].SHA256 })
	if err := validateMergedArtifactCapacity(spec.Artifacts, failures.count, failureBytes, successes.count, successBytes); err != nil {
		return nil, nil, mergedSummary{}, err
	}
	for index := range runs {
		retainedEvidence, found := retainedByIdentity[runs[index].EvidenceSHA256]
		if !found || runs[index].SourceCampaignID != retainedEvidence.SourceCampaignID {
			continue
		}
		reference := runs[index].Execution.Artifact
		if runs[index].Execution.SuccessArtifact != nil {
			reference = runs[index].Execution.SuccessArtifact
		}
		if reference != nil && *reference == retainedEvidence.Reference {
			value := retainedEvidence
			runs[index].Evidence = &value
		}
	}
	summary.retainedEvidence = uint64(len(retained))
	summary.evidenceBytes, err = checkedMergedEvidenceBytes(failureBytes, successBytes, ArtifactLimitTotalBytes, uint64(spec.Artifacts.TotalBytes))
	if err != nil {
		return nil, nil, mergedSummary{}, err
	}
	return shards, runs, summary, nil
}

func checkedMergedEvidenceBytes(current, additional uint64, limit artifactLimit, maximum uint64) (uint64, error) {
	if additional > ^uint64(0)-current {
		return 0, &ArtifactCapacityError{Limit: limit, Required: ^uint64(0), Maximum: maximum, Outcome: CapacityInfrastructureFailure}
	}
	return current + additional, nil
}

func mergedEvidenceIdentity(manifest record.ExecutionRecord) (record.SHA256, error) {
	if manifest.Outcome.FailureSignature != "" {
		return manifest.Outcome.FailureSignature, nil
	}
	projection := manifest
	projection.CreatedAt = ""
	projection.CampaignID = ""
	projection.SelectionOrdinal = 0
	projection.Seed = 0
	projection.RecordHash = ""
	encoded, err := canonicaljson.CanonicalJSON(projection)
	if err != nil {
		return "", err
	}
	return record.HashBytes(encoded), nil
}

func mergedEvidenceLess(left, right mergedEvidence) bool {
	if left.SourceCampaignID != right.SourceCampaignID {
		return left.SourceCampaignID < right.SourceCampaignID
	}
	return left.Reference < right.Reference
}

func validateMergedArtifactCapacity(limits ArtifactCapacityPlan, failureCount, failureBytes, successCount, successBytes uint64) error {
	checks := []struct {
		limit    artifactLimit
		required uint64
		maximum  uint64
	}{
		{ArtifactLimitFailureCount, failureCount, uint64(limits.FailureArtifacts)},
		{ArtifactLimitFailureBytes, failureBytes, uint64(limits.FailureBytes)},
		{artifactLimitSuccessCount, successCount, uint64(limits.SuccessArtifacts)},
		{artifactLimitSuccessBytes, successBytes, uint64(limits.SuccessBytes)},
	}
	for _, check := range checks {
		if check.required > check.maximum {
			return &ArtifactCapacityError{Limit: check.limit, Required: check.required, Maximum: check.maximum, Outcome: CapacityInfrastructureFailure}
		}
	}
	total, err := checkedMergedEvidenceBytes(failureBytes, successBytes, ArtifactLimitTotalBytes, uint64(limits.TotalBytes))
	if err != nil {
		return err
	}
	if total > uint64(limits.TotalBytes) {
		return &ArtifactCapacityError{Limit: ArtifactLimitTotalBytes, Required: total, Maximum: uint64(limits.TotalBytes), Outcome: CapacityInfrastructureFailure}
	}
	return nil
}

func missingOrdinalRanges(total uint64, runs []mergedExecution) []ordinalRange {
	missing := make([]ordinalRange, 0)
	runIndex := 0
	for ordinal := uint64(0); ordinal < total; {
		if runIndex < len(runs) && uint64(runs[runIndex].Execution.SelectionOrdinal) == ordinal {
			runIndex++
			ordinal++
			continue
		}
		start := ordinal
		for ordinal < total && (runIndex >= len(runs) || uint64(runs[runIndex].Execution.SelectionOrdinal) != ordinal) {
			ordinal++
		}
		missing = append(missing, ordinalRange{Start: record.Uint64String(start), End: record.Uint64String(ordinal - 1)})
	}
	return missing
}

// mergedJournalReference checks the merged executions against the plan's
// execution limits and summarizes them.
func mergedJournalReference(limits executionJournalLimits, runs []mergedExecution) (executionJournalReference, error) {
	if uint64(len(runs)) > limits.MaximumExecutions {
		return executionJournalReference{}, &JournalCapacityError{Limit: journalLimitExecutions, Required: uint64(len(runs)), Maximum: limits.MaximumExecutions, Outcome: CapacityInfrastructureFailure}
	}
	var total uint64
	for _, run := range runs {
		encoded, err := logstore.EncodeJSON(run)
		if err != nil {
			return executionJournalReference{}, err
		}
		size := uint64(len(encoded))
		if size > limits.SegmentBytes {
			return executionJournalReference{}, &JournalCapacityError{Limit: journalLimitSegmentBytes, Required: size, Maximum: limits.SegmentBytes, Outcome: CapacityInfrastructureFailure}
		}
		if size > limits.MaximumBytes-total {
			return executionJournalReference{}, &JournalCapacityError{Limit: journalLimitBytes, Required: total + size, Maximum: limits.MaximumBytes, Outcome: CapacityInfrastructureFailure}
		}
		total += size
	}
	return executionJournalReference{Schema: MergedCampaignLogFormat, Records: record.Uint64String(len(runs)), Bytes: record.Uint64String(total)}, nil
}

// writeMergedCampaignLog writes a complete merged campaign log into the
// staging directory that publication renames into place.
func writeMergedCampaignLog(ctx context.Context, path string, campaign mergedCampaignRecord, runs []mergedExecution) (retErr error) {
	if err := makePrivateDirectoriesContext(ctx, path); err != nil {
		return err
	}
	writer, _, err := logstore.OpenWriter(path, campaignLogOptions(logStoreHook(ctx), true), nil)
	if err != nil {
		return fmt.Errorf("create merged campaign log: %w", err)
	}
	defer func() {
		retErr = errors.Join(retErr, writer.Close())
	}()
	if err := writer.AppendJSON(kindMergedOpen, campaign); err != nil {
		return fmt.Errorf("append merged campaign: %w", err)
	}
	for _, run := range runs {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := writer.AppendJSON(kindMergedExecution, run); err != nil {
			return fmt.Errorf("append merged execution: %w", err)
		}
	}
	return nil
}

// mergedCampaignReplay validates a merged campaign log record by record.
type mergedCampaignReplay struct {
	campaign *mergedCampaignRecord
	runs     []mergedExecution
}

func (replay *mergedCampaignReplay) consume(entry logstore.Record) error {
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
		if campaign.Schema != mergedCampaignSchema || campaign.SchemaVersion != record.SchemaVersion || !validRecordSHA256(campaign.PlanSHA256) || (campaign.Selection == "") != (campaign.SelectionCount == 0) {
			return errors.New("merged campaign record is invalid")
		}
		replay.campaign = &campaign
	case kindMergedExecution:
		if replay.campaign == nil {
			return errors.New("merged campaign log does not begin with its envelope")
		}
		if uint64(len(replay.runs)) >= uint64(replay.campaign.Attempted) {
			return errors.New("merged campaign has more executions than it attempted")
		}
		var run mergedExecution
		if err := logstore.DecodeJSON(entry.Payload, &run); err != nil {
			return err
		}
		if count := len(replay.runs); count != 0 && replay.runs[count-1].Execution.SelectionOrdinal >= run.Execution.SelectionOrdinal {
			return errors.New("merged executions are not in canonical ordinal order")
		}
		replay.runs = append(replay.runs, run)
	default:
		return fmt.Errorf("record kind %d is not a merged campaign record", entry.Kind)
	}
	return nil
}

func OpenMergedCampaign(path string) (_ MergedCampaign, retErr error) {
	rootInfo, err := os.Lstat(path)
	if err != nil || !rootInfo.IsDir() || rootInfo.Mode()&os.ModeSymlink != 0 || rootInfo.Mode().Perm() != 0o700 {
		return MergedCampaign{}, errors.Join(errors.New("merged campaign path is not a private directory"), err)
	}
	if legacy := legacyCampaignFormat(path); legacy != "" {
		return MergedCampaign{}, newIntegrityError(&UnsupportedFormatError{Format: legacy})
	}
	entries, err := os.ReadDir(path)
	if err != nil {
		return MergedCampaign{}, err
	}
	if len(entries) != 1 || entries[0].Name() != storeDirectory {
		return MergedCampaign{}, newIntegrityError(errors.New("merged campaign inventory is invalid"))
	}
	var replay mergedCampaignReplay
	report, err := logstore.Replay(filepath.Join(path, storeDirectory), campaignLogOptions(nil, false), replay.consume)
	if err != nil {
		return MergedCampaign{}, classifyIntegrityError(fmt.Errorf("replay merged campaign log: %w", err))
	}
	if report.Tail != logstore.TailNone || replay.campaign == nil {
		return MergedCampaign{}, newIntegrityError(errors.New("merged campaign log is incomplete"))
	}
	campaign := *replay.campaign
	runs := replay.runs
	if runs == nil {
		runs = []mergedExecution{}
	}
	if err := validateMergedCampaign(campaign, runs); err != nil {
		return MergedCampaign{}, newIntegrityError(err)
	}
	return MergedCampaign{Path: path, Record: campaign, Executions: runs}, nil
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
		if shard.Count == 0 || shard.Index >= shard.Count || shard.CampaignID == "" || !validRecordSHA256(shard.CampaignSHA256) {
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
		hasRetained := run.Execution.FailureSignature != nil || run.Execution.SuccessArtifact != nil
		if hasRetained != (run.EvidenceSHA256 != "") || run.EvidenceSHA256 != "" && !validRecordSHA256(run.EvidenceSHA256) {
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
		reference := mergedRun.Execution.Artifact
		if mergedRun.Execution.SuccessArtifact != nil {
			reference = mergedRun.Execution.SuccessArtifact
		}
		if reference == nil || retained.Reference != *reference {
			return errors.New("merged evidence reference does not match its execution")
		}
		if _, duplicate := evidenceByIdentity[retained.SHA256]; duplicate {
			return errors.New("merged evidence identity is duplicated")
		}
		if _, found := campaigns[retained.SourceCampaignID]; !found {
			return errors.New("merged evidence source campaign is unknown")
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
