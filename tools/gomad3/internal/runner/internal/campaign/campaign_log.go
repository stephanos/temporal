package campaign

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"unicode/utf8"

	"go.temporal.io/server/tools/gomad3/internal/evidence/record"
	"go.temporal.io/server/tools/gomad3/internal/runner/internal/logstore"
)

// A campaign directory keeps all of its state in one append-only log under
// store/: the envelope that names the storage format and the campaign
// identity, then lifecycle transitions, the frozen campaign plan, committed
// execution outcomes and the final publication. Replaying that log in order
// is the only way to learn a campaign's state. Artifacts, the prepared
// target and execution work directories are payload files the log
// references; none of them is read as campaign state.
const (
	// CampaignLogFormat is the campaign storage lineage this build writes and
	// the only one it opens.
	CampaignLogFormat = "gomad3.campaign-log/v1"
	// MergedCampaignLogFormat is the merged-campaign storage lineage.
	MergedCampaignLogFormat = "gomad3.merged-campaign-log/v1"
	// LegacyCampaignFormat names the campaign directories earlier Gomad builds
	// wrote (campaign.json, segmented execution journals and lifecycle files).
	// They are refused, never decoded or migrated.
	LegacyCampaignFormat = "gomad3.campaign-directory/v1"
	// LegacyMergedCampaignFormat names the merged campaigns earlier builds
	// wrote (merge.json beside a segmented execution journal).
	LegacyMergedCampaignFormat = "gomad3.merged-campaign-directory/v1"

	storeDirectory = "store"
)

// Record kinds of campaign and merged-campaign logs. One record is one
// transaction; replay validates a whole record before applying any of it.
// Kind 17 held a copied merged execution in the earlier merged layout and is
// retired: a merged log now concatenates its sources' own records.
const (
	kindCampaignOpen   logstore.Kind = 1
	kindLifecycle      logstore.Kind = 2
	kindPlan           logstore.Kind = 3
	kindOutcome        logstore.Kind = 4
	kindPublish        logstore.Kind = 5
	kindAdmission      logstore.Kind = 6
	kindReceipt        logstore.Kind = 7
	kindMergedOpen     logstore.Kind = 16
	kindMergedSource   logstore.Kind = 18
	kindMergedEvidence logstore.Kind = 19
)

const (
	maximumLifecycleBytes = 64 << 10
	maximumLifecycleText  = 4 << 10
	maximumEnvelopeBytes  = maximumCampaignPlanBytes + 64<<10
	// maximumAdmissionBytes bounds one admission: at most the campaign's
	// partial executions, each an ordinal and a seed.
	maximumAdmissionBytes        = maximumExecutionJournalExecutions * 64
	maximumReceiptReferenceBytes = 64 << 10
)

// campaignLogKinds bounds each record kind before its payload is allocated.
// An outcome transaction may hold every in-flight attempt of a stop group, so
// its static bound is the store maximum; replay checks each execution against
// the campaign's per-execution bound and the transaction against the
// campaign's partial-execution bound. A round transaction's static bound is
// the store maximum too; replay bounds its round by the exploration byte
// bound, its segment by the recorded segment bound and each execution by the
// per-execution bound.
var campaignLogKinds = map[logstore.Kind]uint32{
	kindCampaignOpen:   maximumEnvelopeBytes,
	kindLifecycle:      maximumLifecycleBytes,
	kindPlan:           maximumCampaignPlanBytes,
	kindOutcome:        logstore.MaxPayloadBytes,
	kindPublish:        maximumManifestBytes,
	kindAdmission:      maximumAdmissionBytes,
	kindReceipt:        maximumExecutionSegmentBytes + maximumReceiptReferenceBytes,
	kindExploration:    maximumExplorationRecordBytes,
	kindRoundStart:     maximumRoundStartBytes,
	kindRound:          logstore.MaxPayloadBytes,
	kindMergedOpen:     maximumMergedManifestBytes,
	kindMergedSource:   maximumMergedSourceBytes,
	kindMergedEvidence: maximumMergedEvidenceBytes,
}

// EnvelopeKinds returns the record kinds and bounds that may begin a campaign
// or merged campaign log. A store's first record is its envelope, and the
// envelope names the store's format.
func EnvelopeKinds() map[logstore.Kind]uint32 {
	return map[logstore.Kind]uint32{kindCampaignOpen: campaignLogKinds[kindCampaignOpen], kindMergedOpen: campaignLogKinds[kindMergedOpen]}
}

func campaignLogOptions(hook logstore.Hook, create bool) logstore.Options {
	return logstore.Options{Kinds: campaignLogKinds, Create: create, Hook: hook}
}

// ErrUnsupportedFormat matches every UnsupportedFormatError.
var ErrUnsupportedFormat = errors.New("campaign storage format is unsupported")

// UnsupportedFormatError refuses a campaign or merged campaign whose storage
// lineage is not the one this build writes: a directory an earlier Gomad
// wrote, or a log whose envelope names another or a malformed format. Nothing
// of it is decoded or migrated.
type UnsupportedFormatError struct {
	Format    string
	Malformed bool
}

func (err *UnsupportedFormatError) Error() string {
	if err.Malformed {
		return "unsupported campaign storage format: the log envelope does not name a format"
	}
	return fmt.Sprintf("unsupported campaign storage format %q: this Gomad opens only %q and %q, and no migration exists", err.Format, CampaignLogFormat, MergedCampaignLogFormat)
}

func (err *UnsupportedFormatError) Is(target error) bool {
	return target == ErrUnsupportedFormat
}

// checkLogFormat reads only the format of an envelope payload, so a future or
// malformed envelope is refused by name before it is decoded strictly.
func checkLogFormat(payload []byte, want string) error {
	var envelope struct {
		Format *string `json:"format"`
	}
	if err := json.NewDecoder(bytes.NewReader(payload)).Decode(&envelope); err != nil || envelope.Format == nil {
		return &UnsupportedFormatError{Malformed: true}
	}
	if *envelope.Format != want {
		return &UnsupportedFormatError{Format: *envelope.Format}
	}
	return nil
}

// campaignStorePath returns the log store of a campaign or merged campaign
// directory. A directory with the files an earlier Gomad wrote and no store
// is refused by name.
func campaignStorePath(path string) (string, error) {
	store := filepath.Join(path, storeDirectory)
	if _, err := os.Lstat(store); err == nil || !errors.Is(err, os.ErrNotExist) {
		return store, nil
	}
	if legacy := legacyCampaignFormat(path); legacy != "" {
		return "", &UnsupportedFormatError{Format: legacy}
	}
	return store, nil
}

func legacyCampaignFormat(path string) string {
	if _, err := os.Lstat(filepath.Join(path, "merge.json")); err == nil {
		return LegacyMergedCampaignFormat
	}
	for _, marker := range []string{"campaign.json", "executions", filepath.Join(".partial", "campaign")} {
		if _, err := os.Lstat(filepath.Join(path, marker)); err == nil {
			return LegacyCampaignFormat
		}
	}
	return ""
}

// RefuseLegacyLayout fails with an UnsupportedFormatError when path holds a
// campaign or merged campaign an earlier Gomad wrote and no log store. It
// reads only directory entries and mutates nothing.
func RefuseLegacyLayout(path string) error {
	_, err := campaignStorePath(path)
	return err
}

// campaignOpenRecord is the envelope of a campaign log: its storage format
// and the identity the campaign was created with. A later plan record may
// refine the selection, as guided campaigns choose their seeds after
// preparing the target.
type campaignOpenRecord struct {
	Format         string              `json:"format"`
	CampaignID     string              `json:"campaign_id"`
	Strategy       string              `json:"strategy"`
	PlanSHA256     record.SHA256       `json:"plan_sha256,omitempty"`
	Shard          *CampaignShard      `json:"shard,omitempty"`
	Selection      string              `json:"selection"`
	SelectionCount record.Uint64String `json:"selection_count"`
	Guidance       *GuidancePlan       `json:"guidance,omitempty"`
	MaxExecutions  record.Uint64String `json:"max_executions,omitempty"`
	Parallel       record.Uint64String `json:"parallel,omitempty"`
}

func (open campaignOpenRecord) config() CampaignConfig {
	return CampaignConfig{
		CampaignID: open.CampaignID, PlanSHA256: open.PlanSHA256, Shard: cloneCampaignShard(open.Shard), Strategy: open.Strategy,
		Selection: open.Selection, SelectionCount: uint64(open.SelectionCount), Guidance: open.Guidance,
		MaxExecutions: uint64(open.MaxExecutions), Parallel: uint64(open.Parallel),
	}
}

func validateCampaignConfig(config CampaignConfig) error {
	if config.Strategy != "seed" && config.Strategy != "choice-exploration" && config.Strategy != "simulation-exploration" {
		return errors.New("campaign journal strategy is invalid")
	}
	if config.CampaignID == "" || (config.Selection == "") != (config.SelectionCount == 0) || config.SelectionCount == 0 && !isFullyAnsweredGuidance(config.Guidance) {
		return errors.New("campaign journal root, campaign ID, selection, and selection count are required")
	}
	if (config.PlanSHA256 == "") != (config.Shard == nil) {
		return errors.New("campaign journal portable plan identity is incomplete")
	}
	if config.Shard != nil && (!validRecordSHA256(config.PlanSHA256) || config.Shard.Count == 0 || config.Shard.Index >= config.Shard.Count || config.Strategy != "seed") {
		return errors.New("campaign journal shard identity is invalid")
	}
	return nil
}

type lifecycleState string

const (
	lifecyclePlanned            lifecycleState = "planned"
	lifecyclePrepared           lifecycleState = "prepared"
	lifecycleRunning            lifecycleState = "running"
	lifecyclePublished          lifecycleState = "published"
	lifecycleRecoverableFailure lifecycleState = "recoverable-failure"
)

// lifecycleRecord is one lifecycle transition. A transition into running
// from any state but prepared resumes an interrupted campaign.
type lifecycleRecord struct {
	State  lifecycleState `json:"state"`
	Reason string         `json:"reason,omitempty"`
	Detail string         `json:"detail,omitempty"`
}

// outcomeRecord commits execution outcomes as one transaction. A campaign
// that admits ordinals commits either one ordered outcome with the refill it
// reserves (Admit), or a stop group: the trigger and every admitted ordinal's
// actual outcome, in ordinal order, under Stop.
type outcomeRecord struct {
	Executions []ExecutionRecord    `json:"executions"`
	Admit      []admittedJob        `json:"admit,omitempty"`
	Stop       ControllerStopReason `json:"stop,omitempty"`
}

// campaignState is the replayed state of a campaign log. prepare validates a
// whole record against it and returns the change; nothing is applied until
// the caller applies it, which a writer does only after the record is
// durable.
type campaignState struct {
	open           campaignOpenRecord
	opened         bool
	limits         executionJournalLimits
	lifecycle      lifecycleState
	stable         lifecycleState
	reason         string
	detail         string
	plan           *CampaignPlan
	executions     []ExecutionRecord
	executionBytes []uint64
	ordinals       map[uint64]struct{}
	totalBytes     uint64
	published      *CampaignRecord
	frontier       frontierState
	// rounds is the exploration of a choice or simulation exploration
	// campaign, nil for a seed campaign.
	rounds roundReplay
}

func newCampaignState() *campaignState {
	return &campaignState{ordinals: make(map[uint64]struct{}), frontier: newFrontierState()}
}

func (state *campaignState) consume(record logstore.Record) error {
	apply, err := state.prepare(record.Kind, record.Payload)
	if err != nil {
		return err
	}
	apply()
	return nil
}

func (state *campaignState) prepare(kind logstore.Kind, payload []byte) (func(), error) {
	if kind == kindCampaignOpen {
		return state.prepareOpen(payload)
	}
	if !state.opened {
		return nil, errors.New("campaign log does not begin with its envelope")
	}
	if state.published != nil {
		return nil, errors.New("published campaign log has a later record")
	}
	switch kind {
	case kindLifecycle:
		return state.prepareLifecycle(payload)
	case kindPlan:
		return state.preparePlan(payload)
	case kindOutcome:
		return state.prepareOutcome(payload)
	case kindPublish:
		return state.preparePublish(payload)
	case kindAdmission:
		return state.prepareAdmission(payload)
	case kindReceipt:
		return state.prepareReceipt(payload)
	case kindExploration, kindRoundStart, kindRound:
		return state.prepareExplorationRecord(kind, payload)
	default:
		return nil, fmt.Errorf("campaign log record kind %d is not a campaign record", kind)
	}
}

func (state *campaignState) prepareOpen(payload []byte) (func(), error) {
	if state.opened {
		return nil, errors.New("campaign log envelope repeats")
	}
	if err := checkLogFormat(payload, CampaignLogFormat); err != nil {
		return nil, err
	}
	var open campaignOpenRecord
	if err := logstore.DecodeJSON(payload, &open); err != nil {
		return nil, err
	}
	config := open.config()
	if err := validateCampaignConfig(config); err != nil {
		return nil, err
	}
	limits, err := normalizeExecutionJournalLimits(config)
	if err != nil {
		return nil, err
	}
	return func() {
		state.open, state.opened, state.limits = open, true, limits
		state.lifecycle, state.stable = lifecyclePlanned, lifecyclePlanned
		state.rounds = newRoundReplay(open.Strategy)
	}, nil
}

func (state *campaignState) prepareLifecycle(payload []byte) (func(), error) {
	var transition lifecycleRecord
	if err := logstore.DecodeJSON(payload, &transition); err != nil {
		return nil, err
	}
	if err := state.checkTransition(transition); err != nil {
		return nil, err
	}
	return func() { state.applyTransition(transition) }, nil
}

func (state *campaignState) checkTransition(transition lifecycleRecord) error {
	current, stable := state.lifecycle, state.stable
	allowed := false
	switch transition.State {
	case lifecyclePrepared:
		allowed = current == lifecyclePlanned
	case lifecycleRunning:
		allowed = current == lifecyclePrepared || current == lifecycleRunning ||
			current == lifecycleRecoverableFailure && (stable == lifecyclePrepared || stable == lifecycleRunning)
	case lifecycleRecoverableFailure:
		allowed = current == lifecyclePlanned || current == lifecyclePrepared || current == lifecycleRunning || current == lifecycleRecoverableFailure
	case lifecyclePlanned, lifecyclePublished:
		// The envelope begins a campaign planned and only a publication
		// record publishes it; no lifecycle record names either state.
	}
	if !allowed {
		return fmt.Errorf("invalid campaign lifecycle transition %q -> %q", current, transition.State)
	}
	if transition.State == lifecycleRecoverableFailure {
		if transition.Reason == "" || !validLifecycleText(transition.Reason) || !validLifecycleText(transition.Detail) {
			return errors.New("campaign lifecycle failure evidence is invalid")
		}
	} else if transition.Reason != "" || transition.Detail != "" {
		return errors.New("stable campaign lifecycle contains failure evidence")
	}
	return nil
}

func (state *campaignState) applyTransition(transition lifecycleRecord) {
	if transition.State == lifecycleRunning && state.lifecycle != lifecyclePrepared {
		state.discardUnreusableOutcomes()
		if state.rounds != nil {
			state.rounds.abandon()
		}
	}
	state.lifecycle, state.reason, state.detail = transition.State, transition.Reason, transition.Detail
	if transition.State != lifecycleRecoverableFailure {
		state.stable = transition.State
	}
}

// discardUnreusableOutcomes applies a resume: Runner outcomes other than a
// choice-exploration replay divergence or a committed stop group's
// cancellation are not reusable, so a resumed campaign executes their
// ordinals again.
func (state *campaignState) discardUnreusableOutcomes() {
	kept := state.executions[:0]
	keptBytes := state.executionBytes[:0]
	for index, run := range state.executions {
		if _, grouped := state.frontier.stopGroup[uint64(run.SelectionOrdinal)]; grouped || reusableOutcome(state.open.Strategy, run) {
			kept = append(kept, run)
			keptBytes = append(keptBytes, state.executionBytes[index])
			continue
		}
		delete(state.ordinals, uint64(run.SelectionOrdinal))
		state.totalBytes -= state.executionBytes[index]
	}
	clear(state.executions[len(kept):])
	state.executions, state.executionBytes = kept, keptBytes
}

func reusableOutcome(strategy string, run ExecutionRecord) bool {
	return run.Domain != "runner" || strategy == "choice-exploration" && run.Divergence != nil
}

func validLifecycleText(value string) bool {
	return len(value) <= maximumLifecycleText && utf8.ValidString(value)
}

func (state *campaignState) preparePlan(payload []byte) (func(), error) {
	if state.lifecycle != lifecyclePlanned || state.plan != nil {
		return nil, errors.New("campaign plan is recorded outside preparation")
	}
	var plan CampaignPlan
	if err := logstore.DecodeJSON(payload, &plan); err != nil {
		return nil, err
	}
	if err := state.checkPlan(plan); err != nil {
		return nil, err
	}
	limits := executionJournalLimitsFromPlan(*plan.Journal)
	return func() {
		state.plan, state.limits = &plan, limits
	}, nil
}

func (state *campaignState) checkPlan(plan CampaignPlan) error {
	if err := validateCampaignPlan(plan); err != nil {
		return err
	}
	open := state.open
	if plan.PlanSHA256 != open.PlanSHA256 || !equalCampaignShard(plan.Shard, open.Shard) {
		return errors.New("campaign plan shard identity does not match journal")
	}
	if plan.Strategy != open.Strategy {
		return errors.New("campaign plan does not match its campaign")
	}
	derived, err := DeriveExecutionJournalPlan(plan.Strategy, uint64(plan.SelectionCount), uint64(open.MaxExecutions), uint64(open.Parallel))
	if err != nil || *plan.Journal != derived {
		return errors.Join(errors.New("campaign plan journal limits do not match journal"), err)
	}
	return nil
}

func (state *campaignState) prepareOutcome(payload []byte) (func(), error) {
	if state.lifecycle != lifecycleRunning {
		return nil, fmt.Errorf("campaign outcome is committed in lifecycle %q", state.lifecycle)
	}
	if state.rounds != nil {
		return nil, errors.New("an exploration campaign commits its outcomes only in rounds")
	}
	var outcome outcomeRecord
	if err := logstore.DecodeJSON(payload, &outcome); err != nil {
		return nil, err
	}
	if len(outcome.Executions) == 0 || uint64(len(outcome.Executions)) > max(state.limits.MaximumPartialExecutions, 1) {
		return nil, errors.New("campaign outcome transaction must commit between one and the campaign's partial executions")
	}
	sizes, err := state.checkExecutions(outcome.Executions)
	if err != nil {
		return nil, err
	}
	if !state.frontier.admitted {
		if len(outcome.Executions) != 1 || len(outcome.Admit) != 0 || outcome.Stop != "" {
			return nil, errors.New("a campaign that admits no ordinal commits one outcome per transaction")
		}
		return func() { state.applyExecutions(outcome.Executions, sizes) }, nil
	}
	releases, err := state.checkFrontierOutcome(outcome)
	if err != nil {
		return nil, err
	}
	return func() {
		state.applyExecutions(outcome.Executions, sizes)
		state.applyFrontierOutcome(outcome, releases)
	}, nil
}

// checkExecutions validates new committed outcomes against the campaign's
// identity and execution limits, and returns each outcome's encoded size.
func (state *campaignState) checkExecutions(runs []ExecutionRecord) ([]uint64, error) {
	limits := state.limits
	if uint64(len(runs)) > limits.MaximumExecutions-uint64(len(state.executions)) {
		return nil, &JournalCapacityError{Limit: journalLimitExecutions, Required: uint64(len(state.executions)) + uint64(len(runs)), Maximum: limits.MaximumExecutions, Outcome: CapacityInfrastructureFailure}
	}
	ordinalLimit := state.selectionCount()
	if state.open.Strategy != "seed" {
		ordinalLimit = limits.MaximumExecutions
	}
	seen := make(map[uint64]struct{}, len(runs))
	sizes := make([]uint64, len(runs))
	total := state.totalBytes
	for index, run := range runs {
		size, err := state.checkExecutionSize(run)
		if err != nil {
			return nil, err
		}
		if size > limits.MaximumBytes-total {
			return nil, &JournalCapacityError{Limit: journalLimitBytes, Required: total + size, Maximum: limits.MaximumBytes, Outcome: CapacityInfrastructureFailure}
		}
		total += size
		sizes[index] = size
		ordinal := uint64(run.SelectionOrdinal)
		if ordinal >= ordinalLimit {
			return nil, fmt.Errorf("campaign outcome selection ordinal %d is out of range", ordinal)
		}
		if shard := state.open.Shard; shard != nil && ordinal%uint64(shard.Count) != uint64(shard.Index) {
			return nil, fmt.Errorf("campaign outcome ordinal %d is outside its shard assignment", ordinal)
		}
		if _, duplicate := state.ordinals[ordinal]; duplicate {
			return nil, fmt.Errorf("campaign selection ordinal is duplicated: %d", ordinal)
		}
		if _, duplicate := seen[ordinal]; duplicate {
			return nil, fmt.Errorf("campaign selection ordinal is duplicated: %d", ordinal)
		}
		seen[ordinal] = struct{}{}
		if run.Domain == "" || run.Reason == "" {
			return nil, fmt.Errorf("campaign outcome %d identity is incomplete", ordinal)
		}
	}
	return sizes, nil
}

// checkExecutionSize bounds one execution's encoded bytes by the campaign's
// per-execution payload bound.
func (state *campaignState) checkExecutionSize(run ExecutionRecord) (uint64, error) {
	encoded, err := json.Marshal(run)
	if err != nil {
		return 0, err
	}
	size := uint64(len(encoded))
	if size > state.limits.SegmentBytes {
		return 0, &JournalCapacityError{Limit: journalLimitSegmentBytes, Required: size, Maximum: state.limits.SegmentBytes, Outcome: CapacityInfrastructureFailure}
	}
	return size, nil
}

func (state *campaignState) applyExecutions(runs []ExecutionRecord, sizes []uint64) {
	for index, run := range runs {
		state.executions = append(state.executions, run)
		state.executionBytes = append(state.executionBytes, sizes[index])
		state.ordinals[uint64(run.SelectionOrdinal)] = struct{}{}
		state.totalBytes += sizes[index]
	}
}

func (state *campaignState) selectionCount() uint64 {
	if state.plan != nil {
		return uint64(state.plan.SelectionCount)
	}
	return uint64(state.open.SelectionCount)
}

func (state *campaignState) preparePublish(payload []byte) (func(), error) {
	if state.lifecycle != lifecycleRunning {
		return nil, fmt.Errorf("campaign is published in lifecycle %q", state.lifecycle)
	}
	var published CampaignRecord
	if err := logstore.DecodeJSON(payload, &published); err != nil {
		return nil, err
	}
	expected := state.publicationIdentity()
	if published.CampaignID != expected.CampaignID || published.Strategy != expected.Strategy || published.Selection != expected.Selection || published.SelectionCount != expected.SelectionCount ||
		published.PlanSHA256 != expected.PlanSHA256 || !equalCampaignShard(published.Shard, expected.Shard) || !reflect.DeepEqual(published.Guidance, expected.Guidance) ||
		!reflect.DeepEqual(published.Artifacts, expected.Artifacts) || !reflect.DeepEqual(published.Journal, expected.Journal) {
		return nil, errors.New("published campaign summary does not match its log")
	}
	if published.Attempted != expected.Journal.Records {
		return nil, errors.New("published campaign summary does not count its committed outcomes")
	}
	if err := validateCampaign(published, state.executions, executionSuccessBytesBounded); err != nil {
		return nil, fmt.Errorf("published campaign summary is invalid: %w", err)
	}
	if err := state.checkPublishedFrontier(published); err != nil {
		return nil, err
	}
	if state.rounds != nil {
		if err := state.rounds.checkPublished(published); err != nil {
			return nil, err
		}
	}
	return func() {
		state.published = &published
	}, nil
}

// publicationIdentity is the part of a campaign record the log, not the
// summary, decides.
func (state *campaignState) publicationIdentity() CampaignRecord {
	identity := CampaignRecord{
		CampaignID: state.open.CampaignID, Strategy: state.open.Strategy, PlanSHA256: state.open.PlanSHA256, Shard: cloneCampaignShard(state.open.Shard),
		Selection: state.open.Selection, SelectionCount: state.open.SelectionCount, Guidance: state.open.Guidance,
		Journal: &executionJournalReference{Schema: CampaignLogFormat, Records: record.Uint64String(len(state.executions)), Bytes: record.Uint64String(state.totalBytes)},
	}
	if plan := state.plan; plan != nil {
		identity.Selection, identity.SelectionCount, identity.Guidance = plan.Selection, plan.SelectionCount, plan.Guidance
		artifacts := *plan.Artifacts
		identity.Artifacts = &artifacts
	}
	return identity
}

// status projects the replayed lifecycle.
func (state *campaignState) status() LifecycleStatus {
	if state.published != nil {
		return LifecycleStatus{State: lifecyclePublished, LastStableState: lifecyclePublished, Published: true}
	}
	return LifecycleStatus{State: state.lifecycle, LastStableState: state.stable, Reason: state.reason, Detail: state.detail}
}

// resumable reports whether the replayed lifecycle may continue executing.
func (state *campaignState) resumable() error {
	if state.published != nil {
		return errors.New("published campaign cannot be resumed")
	}
	if state.plan == nil || state.lifecycle == lifecyclePlanned || state.stable == lifecyclePlanned {
		return errors.New("campaign preparation did not complete")
	}
	return nil
}

func (state *campaignState) journalPlan() ExecutionJournalPlan {
	if state.plan != nil && state.plan.Journal != nil {
		return *state.plan.Journal
	}
	return recordExecutionJournalLimits(state.limits)
}
