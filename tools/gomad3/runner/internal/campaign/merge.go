package campaign

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"hash"
	"os"
	"path/filepath"
	"reflect"
	"sort"

	"go.temporal.io/server/tools/gomad3/artifact"
	"go.temporal.io/server/tools/gomad3/record"
	"go.temporal.io/server/tools/gomad3/runner/internal/logstore"
)

// A merged campaign is one log that concatenates the published logs of one
// portable plan's shards. Its envelope names the plan, the aggregate summary
// and every source's provenance. Each source's records follow in shard order,
// opened by a source-scope record and replayed in isolation, so one shard's
// initialization and publication records never touch another shard's
// lifecycle. Last, one evidence record attributes the retained evidence of
// each merged execution that references any.
//
// The aggregate borrows its sources' artifacts, diagnostics and target pools
// by reference under each source's recorded path. It copies none of them,
// owns none of them and sweeps nothing: they stay owned by their shards.
const (
	mergedCampaignSchema       = "gomad3.merged-campaign/v1"
	maximumMergedManifestBytes = 16 << 20
	maximumMergedSourceBytes   = 4 << 10
	maximumMergedEvidenceBytes = 64 << 10
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

// mergedShard is the immutable provenance of one source shard.
type mergedShard struct {
	Index          record.Uint64String `json:"index"`
	Count          record.Uint64String `json:"count"`
	CampaignID     string              `json:"campaign_id"`
	CampaignSHA256 record.SHA256       `json:"campaign_sha256"`
	Attempted      record.Uint64String `json:"attempted"`
	JournalBytes   record.Uint64String `json:"journal_bytes"`
	// Path is the absolute source campaign directory under which the
	// shard's borrowed artifact and diagnostic references resolve.
	Path string `json:"path"`
	// Records and LogSHA256 identify exactly the source records the
	// aggregate concatenates in this shard's scope.
	Records   record.Uint64String `json:"records"`
	LogSHA256 record.SHA256       `json:"log_sha256"`
}

// mergedExecution is one execution of a merged campaign: a source shard's
// committed outcome and the evidence identity the aggregate attributes to it.
type mergedExecution struct {
	SourceCampaignID string
	EvidenceSHA256   record.SHA256
	Evidence         *mergedEvidence
	Execution        ExecutionRecord
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
	// whichever shard store the evidence came from. Evidence without a target
	// identity counts in full.
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

// mergedCampaignRecord is the envelope of a merged campaign log.
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
	// Limits are the plan's execution limits; every source shard ran under
	// them and the aggregate's committed outcomes are bounded by them.
	Limits    ExecutionJournalPlan `json:"limits"`
	Artifacts ArtifactCapacityPlan `json:"artifacts"`
	Shards    []mergedShard        `json:"shards"`
}

// mergedSourceRecord opens the scope of one source shard's records.
type mergedSourceRecord struct {
	Index record.Uint64String `json:"index"`
}

// mergedEvidenceRecord attributes evidence to one merged execution that
// references a retained artifact. Evidence is set on the execution whose
// artifact the aggregate counts for that evidence identity.
type mergedEvidenceRecord struct {
	Ordinal        record.Uint64String `json:"ordinal"`
	EvidenceSHA256 record.SHA256       `json:"evidence_sha256"`
	Evidence       *mergedEvidence     `json:"evidence,omitempty"`
}

type MergedCampaign struct {
	Path       string
	Record     mergedCampaignRecord
	Executions []mergedExecution
}

// mergeIdentity is what every source shard of one merge must share.
type mergeIdentity struct {
	planSHA256     record.SHA256
	selection      string
	selectionCount uint64
	guidance       *GuidancePlan
	journal        ExecutionJournalPlan
	artifacts      ArtifactCapacityPlan
}

func (identity mergeIdentity) checkSource(path string, opened Campaign, state *campaignState) error {
	batch := opened.Record
	if batch.Schema != "gomad3.campaign/v1" || batch.PlanSHA256 != identity.planSHA256 || batch.Selection != identity.selection || uint64(batch.SelectionCount) != identity.selectionCount || batch.Shard == nil ||
		!reflect.DeepEqual(batch.Guidance, identity.guidance) || state.journalPlan() != identity.journal || batch.Artifacts != nil && *batch.Artifacts != identity.artifacts {
		return fmt.Errorf("shard campaign %s does not match the campaign plan", path)
	}
	return nil
}

// sourceScope replays one source shard's records in isolation and computes
// the identity of exactly those records.
type sourceScope struct {
	state   *campaignState
	digest  hash.Hash
	records uint64
}

func newSourceScope() *sourceScope {
	return &sourceScope{state: newCampaignState(), digest: sha256.New()}
}

func (scope *sourceScope) consume(entry logstore.Record) error {
	if err := checkSourceRecordKind(entry.Kind); err != nil {
		return err
	}
	if err := scope.state.consume(entry); err != nil {
		return err
	}
	scope.add(entry)
	return nil
}

// checkSourceRecordKind admits the records of a seed shard: its lifecycle,
// plan, admissions, receipts, outcomes and publication. A seed shard never
// holds an exploration round, and a source never nests a merged campaign.
func checkSourceRecordKind(kind logstore.Kind) error {
	switch kind {
	case kindCampaignOpen, kindLifecycle, kindPlan, kindOutcome, kindPublish, kindAdmission, kindReceipt:
		return nil
	case kindExploration, kindRoundStart, kindRound:
		return fmt.Errorf("shard campaign record kind %d is an exploration record, which no seed shard holds", kind)
	default:
		return fmt.Errorf("record kind %d is not a shard campaign record", kind)
	}
}

func (scope *sourceScope) add(entry logstore.Record) {
	var header [10]byte
	binary.BigEndian.PutUint16(header[:2], uint16(entry.Kind))
	binary.BigEndian.PutUint64(header[2:], uint64(len(entry.Payload)))
	scope.digest.Write(header[:])
	scope.digest.Write(entry.Payload)
	scope.records++
}

func (scope *sourceScope) identity() record.SHA256 {
	var sum [sha256.Size]byte
	copy(sum[:], scope.digest.Sum(nil))
	return record.SHA256FromSum(sum)
}

// published validates a replayed source as a mergeable shard: a published
// campaign whose admitted work and receipts are all retired, whose summary
// matches its outcomes, and whose every live artifact and diagnostic sidecar
// under path is present and intact. It reads and changes nothing else.
func (scope *sourceScope) published(path string) (Campaign, error) {
	state := scope.state
	if !state.opened {
		return Campaign{}, newIntegrityError(errors.New("campaign log has no envelope"))
	}
	if state.published == nil {
		return Campaign{}, newIntegrityError(errors.New("campaign is not published"))
	}
	if len(state.frontier.outstanding) != 0 || len(state.frontier.receipts) != 0 {
		return Campaign{}, newIntegrityError(errors.New("published shard campaign has unretired admissions or receipts"))
	}
	opened, err := openPublishedState(path, state)
	if err != nil {
		return Campaign{}, err
	}
	if err := validateLiveReferences(path, state); err != nil {
		return Campaign{}, err
	}
	return opened, nil
}

// provenance describes a validated source for the aggregate envelope.
func (scope *sourceScope) provenance(path string, opened Campaign) (mergedShard, error) {
	batch := opened.Record
	encoded, err := logstore.EncodeJSON(batch)
	if err != nil {
		return mergedShard{}, err
	}
	journalBytes := uint64(0)
	if opened.Journal != nil {
		journalBytes = opened.Journal.Bytes
	}
	return mergedShard{
		Index: batch.Shard.Index, Count: batch.Shard.Count, CampaignID: batch.CampaignID, CampaignSHA256: record.HashBytes(encoded), Attempted: batch.Attempted, JournalBytes: record.Uint64String(journalBytes),
		Path: path, Records: record.Uint64String(scope.records), LogSHA256: scope.identity(),
	}, nil
}

// mergeSource is one source shard validated by merge preflight.
type mergeSource struct {
	path   string
	scope  *sourceScope
	opened Campaign
	shard  mergedShard
}

// readMergeSource replays a source shard read-only and validates it
// completely. A torn final record is refused: the aggregate concatenates
// only complete source logs.
func readMergeSource(path string) (mergeSource, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return mergeSource{}, fmt.Errorf("resolve shard campaign path: %w", err)
	}
	if err := validateCampaignDirectory(absolute); err != nil {
		return mergeSource{}, err
	}
	store, err := campaignStorePath(absolute)
	if err != nil {
		return mergeSource{}, newIntegrityError(err)
	}
	scope := newSourceScope()
	report, err := logstore.Replay(store, campaignLogOptions(nil, false), scope.consume)
	if err != nil {
		return mergeSource{}, classifyIntegrityError(fmt.Errorf("replay shard campaign %s: %w", absolute, err))
	}
	if report.Tail != logstore.TailNone {
		return mergeSource{}, newIntegrityError(fmt.Errorf("shard campaign %s log ends in a torn %s record", absolute, report.Tail))
	}
	opened, err := scope.published(absolute)
	if err != nil {
		return mergeSource{}, classifyIntegrityError(fmt.Errorf("shard campaign %s: %w", absolute, err))
	}
	if opened.Record.Shard == nil {
		return mergeSource{}, fmt.Errorf("shard campaign %s does not match the campaign plan", absolute)
	}
	shard, err := scope.provenance(absolute, opened)
	if err != nil {
		return mergeSource{}, err
	}
	return mergeSource{path: absolute, scope: scope, opened: opened, shard: shard}, nil
}

// MergeCampaigns validates every source shard, its records and the evidence
// they reference, the plan's ordinal coverage and the aggregate bounds before
// it creates any output. Only then does it write the aggregate log in a
// private staging directory and publish it by a no-replace rename. A failed
// merge leaves no output, and no merge changes a source shard.
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
	sources, runs, summary, err := collectMergedCampaign(spec)
	if err != nil {
		return MergedCampaign{}, err
	}
	missing := missingOrdinalRanges(spec.SelectionCount, runs)
	if len(missing) != 0 && !spec.Partial {
		return MergedCampaign{}, fmt.Errorf("campaign merge is missing %d ordinal ranges", len(missing))
	}
	shards := make([]mergedShard, len(sources))
	for index, source := range sources {
		shards[index] = source.shard
	}
	reference, err := mergedJournalReference(limits, shards, uint64(len(runs)))
	if err != nil {
		return MergedCampaign{}, err
	}
	aggregate := mergedCampaignRecord{
		Format: MergedCampaignLogFormat, Guidance: spec.Guidance,
		Schema: mergedCampaignSchema, SchemaVersion: record.SchemaVersion, PlanSHA256: spec.PlanSHA256, Selection: spec.Selection, SelectionCount: record.Uint64String(spec.SelectionCount),
		Partial: len(missing) != 0, Missing: missing, Attempted: record.Uint64String(len(runs)), Succeeded: record.Uint64String(summary.succeeded), Failures: record.Uint64String(summary.failures), Watchdogs: record.Uint64String(summary.watchdogs), Cancelled: record.Uint64String(summary.cancelled),
		DistinctFailures: record.Uint64String(len(summary.failuresSeen)), RetainedSuccesses: record.Uint64String(summary.retainedSuccesses), RetainedSuccessBytes: record.Uint64String(summary.retainedSuccessBytes), RetainedEvidence: record.Uint64String(summary.retainedEvidence), EvidenceBytes: record.Uint64String(summary.evidenceBytes),
		Journal: reference, Limits: spec.Journal, Artifacts: spec.Artifacts, Shards: shards,
	}
	manifest, err := logstore.EncodeJSON(aggregate)
	if err != nil {
		return MergedCampaign{}, err
	}
	if len(manifest) > maximumMergedManifestBytes {
		return MergedCampaign{}, &JournalCapacityError{Limit: journalLimitManifestBytes, Required: uint64(len(manifest)), Maximum: maximumMergedManifestBytes, Outcome: CapacityInfrastructureFailure}
	}
	if err := ctx.Err(); err != nil {
		return MergedCampaign{}, err
	}
	// Preflight is complete: every source, its records, its live evidence,
	// the ordinal coverage and every bound are validated. Output starts here.
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
	if err := writeMergedCampaignLog(ctx, filepath.Join(staging, storeDirectory), aggregate, sources, runs); err != nil {
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
	return MergedCampaign{Path: output, Record: aggregate, Executions: runs}, nil
}

type mergedSummary struct {
	succeeded, failures, watchdogs, cancelled uint64
	retainedSuccesses, retainedSuccessBytes   uint64
	retainedEvidence                          uint64
	evidenceBytes                             uint64
	failuresSeen                              map[record.SHA256]struct{}
}

// collectMergedCampaign is merge preflight: it validates every source and
// returns them in shard order with their executions in ordinal order.
func collectMergedCampaign(spec MergeSpec) ([]mergeSource, []mergedExecution, mergedSummary, error) {
	paths := append([]string(nil), spec.ShardPaths...)
	sort.Strings(paths)
	identity := mergeIdentity{planSHA256: spec.PlanSHA256, selection: spec.Selection, selectionCount: spec.SelectionCount, guidance: spec.Guidance, journal: spec.Journal, artifacts: spec.Artifacts}
	sources := make([]mergeSource, 0, len(paths))
	runs := make([]mergedExecution, 0)
	retainedByIdentity := make(map[record.SHA256]mergedEvidence)
	assignments := make(map[uint64]struct{}, len(paths))
	ordinals := make(map[uint64]struct{})
	summary := mergedSummary{failuresSeen: make(map[record.SHA256]struct{})}
	var shardCount uint64
	for _, path := range paths {
		source, err := readMergeSource(path)
		if err != nil {
			return nil, nil, mergedSummary{}, err
		}
		if err := identity.checkSource(source.path, source.opened, source.scope.state); err != nil {
			return nil, nil, mergedSummary{}, err
		}
		batch := source.opened.Record
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
		sources = append(sources, source)
		for _, run := range source.opened.Executions {
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
			if retainsEvidence(run) {
				candidate, err := retainedMergedEvidence(source.path, batch.CampaignID, run)
				if err != nil {
					return nil, nil, mergedSummary{}, classifyIntegrityError(fmt.Errorf("shard campaign %s: %w", source.path, err))
				}
				merged.EvidenceSHA256 = candidate.SHA256
				if prior, found := retainedByIdentity[candidate.SHA256]; !found || mergedEvidenceLess(candidate, prior) {
					retainedByIdentity[candidate.SHA256] = candidate
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
	sort.Slice(sources, func(i, j int) bool { return sources[i].shard.Index < sources[j].shard.Index })
	sort.Slice(runs, func(i, j int) bool { return runs[i].Execution.SelectionOrdinal < runs[j].Execution.SelectionOrdinal })
	var failures, successes mergedEvidenceBytes
	var err error
	for _, item := range retainedByIdentity {
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
	if err := validateMergedArtifactCapacity(spec.Artifacts, failures.count, failureBytes, successes.count, successBytes); err != nil {
		return nil, nil, mergedSummary{}, err
	}
	// The counted evidence attaches to the lowest ordinal that references its
	// artifact: executions of one shard that share a failure signature share
	// one artifact, and the aggregate counts it once.
	attached := make(map[record.SHA256]struct{}, len(retainedByIdentity))
	for index := range runs {
		retainedEvidence, found := retainedByIdentity[runs[index].EvidenceSHA256]
		if !found || runs[index].SourceCampaignID != retainedEvidence.SourceCampaignID {
			continue
		}
		if _, done := attached[retainedEvidence.SHA256]; done {
			continue
		}
		if retainedReferenceOf(runs[index].Execution) == retainedEvidence.Reference {
			value := retainedEvidence
			runs[index].Evidence = &value
			attached[retainedEvidence.SHA256] = struct{}{}
		}
	}
	summary.retainedEvidence = uint64(len(retainedByIdentity))
	summary.evidenceBytes, err = checkedMergedEvidenceBytes(failureBytes, successBytes, ArtifactLimitTotalBytes, uint64(spec.Artifacts.TotalBytes))
	if err != nil {
		return nil, nil, mergedSummary{}, err
	}
	return sources, runs, summary, nil
}

func retainsEvidence(run ExecutionRecord) bool {
	return run.FailureSignature != nil || run.SuccessArtifact != nil
}

// retainedReferenceOf is the campaign-relative artifact reference of an
// execution's retained evidence, or "" when it retains none.
func retainedReferenceOf(run ExecutionRecord) string {
	switch {
	case run.SuccessArtifact != nil:
		return *run.SuccessArtifact
	case run.Artifact != nil:
		return *run.Artifact
	default:
		return ""
	}
}

// retainedMergedEvidence resolves the evidence a source execution retains
// under its source campaign directory and describes it for the aggregate.
func retainedMergedEvidence(sourcePath, campaignID string, run ExecutionRecord) (mergedEvidence, error) {
	resolved, err := ResolveRetainedEvidence(sourcePath, campaignID, run)
	if err != nil {
		return mergedEvidence{}, err
	}
	identity, err := mergedEvidenceIdentity(resolved.Manifest)
	if err != nil {
		return mergedEvidence{}, err
	}
	target := artifact.TargetOf(resolved.Manifest)
	return mergedEvidence{
		SHA256: identity, Kind: resolved.Manifest.ArtifactKind, SourceCampaignID: campaignID, Reference: retainedReferenceOf(run), RecordSHA256: resolved.Manifest.RecordHash, StoredBytes: record.Uint64String(resolved.StoredBytes),
		TargetSHA256: target.SHA256, TargetBytes: record.Uint64String(target.Bytes),
	}, nil
}

func checkedMergedEvidenceBytes(current, additional uint64, limit artifactLimit, maximum uint64) (uint64, error) {
	if additional > ^uint64(0)-current {
		return 0, &ArtifactCapacityError{Limit: limit, Required: ^uint64(0), Maximum: maximum, Outcome: CapacityInfrastructureFailure}
	}
	return current + additional, nil
}

// mergedEvidenceIdentity is a failure's signature, or the digest of a
// success's manifest without the members that differ between executions of
// one outcome.
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
	encoded, err := logstore.EncodeJSON(projection)
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

// mergedJournalReference bounds the aggregate's committed outcomes by the
// plan's execution limits and summarizes them. Each source already bounds
// each of its outcomes by the plan's per-execution bound; the aggregate
// bounds their count and summed encoded bytes.
func mergedJournalReference(limits executionJournalLimits, shards []mergedShard, executions uint64) (executionJournalReference, error) {
	if executions > limits.MaximumExecutions {
		return executionJournalReference{}, &JournalCapacityError{Limit: journalLimitExecutions, Required: executions, Maximum: limits.MaximumExecutions, Outcome: CapacityInfrastructureFailure}
	}
	var total uint64
	for _, shard := range shards {
		size := uint64(shard.JournalBytes)
		if size > limits.MaximumBytes-total {
			required := ^uint64(0)
			if size <= ^uint64(0)-total {
				required = total + size
			}
			return executionJournalReference{}, &JournalCapacityError{Limit: journalLimitBytes, Required: required, Maximum: limits.MaximumBytes, Outcome: CapacityInfrastructureFailure}
		}
		total += size
	}
	return executionJournalReference{Schema: MergedCampaignLogFormat, Records: record.Uint64String(executions), Bytes: record.Uint64String(total)}, nil
}

// writeMergedCampaignLog writes a complete merged campaign log into the
// staging directory that publication renames into place: the envelope, each
// source's scope and records in shard order, then the evidence attributions
// in ordinal order.
func writeMergedCampaignLog(ctx context.Context, path string, aggregate mergedCampaignRecord, sources []mergeSource, runs []mergedExecution) (retErr error) {
	writer, _, err := logstore.OpenWriter(path, campaignLogOptions(logStoreHook(ctx), true), nil)
	if err != nil {
		return fmt.Errorf("create merged campaign log: %w", err)
	}
	defer func() {
		retErr = errors.Join(retErr, writer.Close())
	}()
	if err := writer.AppendJSON(kindMergedOpen, aggregate); err != nil {
		return fmt.Errorf("append merged campaign: %w", err)
	}
	for _, source := range sources {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := writer.AppendJSON(kindMergedSource, mergedSourceRecord{Index: source.shard.Index}); err != nil {
			return fmt.Errorf("append merged source scope: %w", err)
		}
		if err := copySourceRecords(ctx, writer, source); err != nil {
			return err
		}
	}
	for _, run := range runs {
		if run.EvidenceSHA256 == "" {
			continue
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := writer.AppendJSON(kindMergedEvidence, mergedEvidenceRecord{Ordinal: run.Execution.SelectionOrdinal, EvidenceSHA256: run.EvidenceSHA256, Evidence: run.Evidence}); err != nil {
			return fmt.Errorf("append merged evidence: %w", err)
		}
	}
	return nil
}

var errSourceCopyStopped = errors.New("merged source copy stopped")

// copySourceRecords appends a source shard's records unchanged. They must be
// exactly the records preflight validated; a source that changed since then
// fails the merge.
func copySourceRecords(ctx context.Context, writer *logstore.Writer, source mergeSource) error {
	copied := &sourceScope{digest: sha256.New()}
	var stopped error
	report, err := logstore.Replay(filepath.Join(source.path, storeDirectory), campaignLogOptions(nil, false), func(entry logstore.Record) error {
		if err := ctx.Err(); err != nil {
			stopped = err
			return errSourceCopyStopped
		}
		copied.add(entry)
		if copied.records > source.scope.records {
			stopped = fmt.Errorf("shard campaign %s changed during merge", source.path)
			return errSourceCopyStopped
		}
		if err := writer.Append(entry.Kind, entry.Payload); err != nil {
			stopped = fmt.Errorf("append merged source record: %w", err)
			return errSourceCopyStopped
		}
		return nil
	})
	if stopped != nil {
		return stopped
	}
	if err != nil {
		return fmt.Errorf("copy shard campaign %s: %w", source.path, err)
	}
	if report.Tail != logstore.TailNone || copied.records != source.scope.records || copied.identity() != source.scope.identity() {
		return fmt.Errorf("shard campaign %s changed during merge", source.path)
	}
	return nil
}

// mergedCampaignReplay validates a merged campaign log record by record. Each
// source scope replays into its own campaign state.
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
		return nil, errors.Join(errors.New("merged campaign limits are invalid"), err)
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
			return nil, fmt.Errorf("merged source %d path is not a clean absolute path", recorded.Index)
		}
		opened, err := scope.published(recorded.Path)
		if err != nil {
			return nil, fmt.Errorf("merged source %s: %w", recorded.Path, err)
		}
		if err := identity.checkSource(recorded.Path, opened, scope.state); err != nil {
			return nil, err
		}
		provenance, err := scope.provenance(recorded.Path, opened)
		if err != nil {
			return nil, err
		}
		if provenance != recorded {
			return nil, fmt.Errorf("merged source %s does not match its recorded provenance", recorded.Path)
		}
		if uint64(recorded.JournalBytes) > limits.MaximumBytes-journalBytes {
			return nil, errors.New("merged campaign outcomes exceed the plan's byte bound")
		}
		journalBytes += uint64(recorded.JournalBytes)
		for _, run := range opened.Executions {
			runs = append(runs, mergedExecution{SourceCampaignID: recorded.CampaignID, Execution: run})
		}
	}
	if uint64(len(runs)) > limits.MaximumExecutions || journalBytes != uint64(campaign.Journal.Bytes) {
		return nil, errors.New("merged campaign outcomes do not match the plan's execution bounds")
	}
	sort.Slice(runs, func(i, j int) bool { return runs[i].Execution.SelectionOrdinal < runs[j].Execution.SelectionOrdinal })
	attributed := 0
	for index := range runs {
		if index > 0 && runs[index-1].Execution.SelectionOrdinal == runs[index].Execution.SelectionOrdinal {
			return nil, fmt.Errorf("campaign selection ordinal is duplicated: %d", runs[index].Execution.SelectionOrdinal)
		}
		if !retainsEvidence(runs[index].Execution) {
			continue
		}
		if attributed >= len(replay.evidence) || replay.evidence[attributed].Ordinal != runs[index].Execution.SelectionOrdinal {
			return nil, fmt.Errorf("merged execution %d has no evidence record", runs[index].Execution.SelectionOrdinal)
		}
		runs[index].EvidenceSHA256 = replay.evidence[attributed].EvidenceSHA256
		runs[index].Evidence = replay.evidence[attributed].Evidence
		attributed++
	}
	if attributed != len(replay.evidence) {
		return nil, errors.New("merged evidence record has no execution that retains evidence")
	}
	return runs, nil
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
	if report.Tail != logstore.TailNone || replay.campaign == nil || len(replay.scopes) != len(replay.campaign.Shards) {
		return MergedCampaign{}, newIntegrityError(errors.New("merged campaign log is incomplete"))
	}
	campaign := *replay.campaign
	runs, err := replay.executions()
	if err != nil {
		return MergedCampaign{}, classifyIntegrityError(err)
	}
	if err := validateMergedCampaign(campaign, runs); err != nil {
		return MergedCampaign{}, newIntegrityError(err)
	}
	if err := verifyMergedEvidence(campaign, runs); err != nil {
		return MergedCampaign{}, classifyIntegrityError(err)
	}
	return MergedCampaign{Path: path, Record: campaign, Executions: runs}, nil
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
