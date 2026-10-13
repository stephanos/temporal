package campaign

import (
	"errors"
	"fmt"
	"maps"
	"reflect"
	"slices"
	"strings"

	"go.temporal.io/server/tools/gomad3/artifact"
	"go.temporal.io/server/tools/gomad3/record"
	"go.temporal.io/server/tools/gomad3/runner/internal/logstore"
)

// A seed campaign's admission frontier lives in its log. Admission records
// reserve ordinals before their attempts launch; terminal receipts record an
// attempt's actual completion and the payloads it published, before the
// ordered outcome that applies it; outcome transactions commit in ordinal
// order and atomically reserve the permitted refill. Reserved, running and
// receipted ordinals all occupy a slot until their outcome commits, so the
// admitted set never exceeds the campaign's partial-execution capacity.
//
// When the next ordered receipt satisfies the failure policy, admissions
// freeze and every admitted ordinal forms one stop group. First-failure
// requests cancellation of active attempts; a budget stop lets them finish.
// Once every member has a receipt, the trigger and every member's actual
// outcome commit together in one outcome record. Replay restores the
// frontier and receipts, derives a pending stop from them and the committed
// policy, and leaves only receipt-less admissions to run again.
//
// A campaign whose log never admits commits outcomes directly, one per
// record; exploration strategies always do, and their rounds own their own
// ordering.

// admittedJob is one reserved ordinal.
type admittedJob struct {
	Ordinal record.Uint64String `json:"ordinal"`
	Seed    record.Uint64String `json:"seed"`
}

// admissionRecord reserves ordinals before their attempts launch.
type admissionRecord struct {
	Jobs []admittedJob `json:"jobs"`
}

// pendingReferences are payloads a receipt's attempt durably published
// before the receipt. They stay live until the ordered outcome adopts them
// as its evidence or retires them.
type pendingReferences struct {
	Artifacts   []string                 `json:"artifacts,omitempty"`
	Diagnostics []artifact.FileReference `json:"diagnostics,omitempty"`
}

// receiptRecord is one admitted attempt's actual terminal observation.
type receiptRecord struct {
	Observation ExecutionRecord   `json:"observation"`
	Pending     pendingReferences `json:"pending"`
}

// Receipt is an admitted attempt's actual completion, recorded before its
// ordered outcome. Observation carries the attempt's classification and
// evidence but no retention or novelty decision, which only its ordered
// outcome makes. Artifacts are campaign-relative references to the failure
// or success artifacts the attempt published; Diagnostics are diagnostic
// sidecars it published under diagnostics/.
type Receipt struct {
	Observation ExecutionRecord
	Artifacts   []string
	Diagnostics []artifact.FileReference
}

// Frontier is the replayed admission state of a seed campaign.
type Frontier struct {
	// Outstanding are the admitted ordinals without a committed outcome, in
	// ordinal order.
	Outstanding []SeedJob
	// Receipts are the durable terminal receipts of outstanding ordinals.
	Receipts map[uint64]Receipt
	// PendingStop is the stop the next ordered receipt triggers under the
	// committed policy; while it is set no ordinal may be admitted, and the
	// whole frontier commits as one stop group.
	PendingStop ControllerStopReason
	// Stopped is the committed stop, after which nothing is admitted.
	Stopped ControllerStopReason
	// Available is how many more ordinals may be admitted now.
	Available uint64
}

// Retry reports the outstanding ordinals without a durable receipt: the only
// attempts a resumed campaign runs again.
func (frontier Frontier) Retry() []SeedJob {
	var retry []SeedJob
	for _, job := range frontier.Outstanding {
		if _, received := frontier.Receipts[job.Ordinal]; !received {
			retry = append(retry, job)
		}
	}
	return retry
}

const (
	maximumPendingArtifacts   = 2
	maximumPendingDiagnostics = 1
)

// frontierState is the admission part of campaignState.
type frontierState struct {
	admitted    bool
	outstanding map[uint64]uint64
	receipts    map[uint64]receiptRecord
	stopped     ControllerStopReason
	stopGroup   map[uint64]struct{}
	signatures  map[record.SHA256]struct{}
	// diagnostics are the sidecars committed outcomes adopted from their
	// receipts. They stay live with those outcomes.
	diagnostics map[string]artifact.FileReference
}

func newFrontierState() frontierState {
	return frontierState{
		outstanding: map[uint64]uint64{}, receipts: map[uint64]receiptRecord{}, stopGroup: map[uint64]struct{}{},
		signatures: map[record.SHA256]struct{}{}, diagnostics: map[string]artifact.FileReference{},
	}
}

func (state *campaignState) policy() (FailurePolicy, uint64) {
	if state.plan == nil {
		return failurePolicyAll, 0
	}
	return FailurePolicy(state.plan.OnFailure), uint64(state.plan.FailureBudget)
}

func (state *campaignState) minimumOutstanding() (uint64, bool) {
	if len(state.frontier.outstanding) == 0 {
		return 0, false
	}
	minimum := ^uint64(0)
	for ordinal := range state.frontier.outstanding {
		minimum = min(minimum, ordinal)
	}
	return minimum, true
}

// orderedCompletion classifies a committed or observed outcome for the
// policy, counting its failure signature among the committed distinct ones.
func (state *campaignState) orderedCompletion(run ExecutionRecord) Completion {
	switch run.Domain {
	case "success":
		return CompletedSuccess()
	case "target", "watchdog":
		distinct := uint64(len(state.frontier.signatures))
		if run.FailureSignature != nil {
			if _, seen := state.frontier.signatures[*run.FailureSignature]; !seen {
				distinct++
			}
		}
		return CompletedFailure(run.Domain, run.Reason, distinct)
	case "runner":
		if run.Reason == "runner_cancelled" {
			return CompletedCancelled()
		}
	}
	return CompletedUnclassified()
}

// pendingStop derives the stop the next ordered receipt triggers.
func (state *campaignState) pendingStop() ControllerStopReason {
	if state.frontier.stopped != "" {
		return ""
	}
	next, ok := state.minimumOutstanding()
	if !ok {
		return ""
	}
	receipt, received := state.frontier.receipts[next]
	if !received {
		return ""
	}
	policy, budget := state.policy()
	return PolicyStop(policy, budget, state.orderedCompletion(receipt.Observation))
}

func (state *campaignState) frontierSnapshot() Frontier {
	frontier := Frontier{Receipts: map[uint64]Receipt{}, PendingStop: state.pendingStop(), Stopped: state.frontier.stopped}
	for ordinal, seed := range state.frontier.outstanding {
		frontier.Outstanding = append(frontier.Outstanding, SeedJob{Ordinal: ordinal, Seed: seed})
	}
	slices.SortFunc(frontier.Outstanding, func(left, right SeedJob) int { return compareUint64(left.Ordinal, right.Ordinal) })
	for ordinal, receipt := range state.frontier.receipts {
		frontier.Receipts[ordinal] = Receipt{
			Observation: receipt.Observation, Artifacts: slices.Clone(receipt.Pending.Artifacts), Diagnostics: slices.Clone(receipt.Pending.Diagnostics),
		}
	}
	if frontier.PendingStop == "" && frontier.Stopped == "" {
		frontier.Available = state.limits.MaximumPartialExecutions - uint64(len(state.frontier.outstanding))
	}
	return frontier
}

func compareUint64(left, right uint64) int {
	switch {
	case left < right:
		return -1
	case left > right:
		return 1
	default:
		return 0
	}
}

// checkAdmissions validates reservations against the frontier that remains
// after removing the ordinals in leaving.
func (state *campaignState) checkAdmissions(jobs []admittedJob, leaving map[uint64]struct{}) error {
	if state.open.Strategy != "seed" {
		return errors.New("only seed campaigns admit ordinals")
	}
	if state.frontier.stopped != "" {
		return fmt.Errorf("stopped campaign (%s) admits no ordinal", state.frontier.stopped)
	}
	outstanding := uint64(len(state.frontier.outstanding) - len(leaving))
	if uint64(len(jobs)) > state.limits.MaximumPartialExecutions-outstanding {
		return &JournalCapacityError{Limit: journalLimitPartialExecutions, Required: outstanding + uint64(len(jobs)), Maximum: state.limits.MaximumPartialExecutions, Outcome: CapacityInfrastructureFailure}
	}
	seen := make(map[uint64]struct{}, len(jobs))
	for _, job := range jobs {
		ordinal := uint64(job.Ordinal)
		if ordinal >= state.selectionCount() {
			return fmt.Errorf("admitted ordinal %d is out of range", ordinal)
		}
		if shard := state.open.Shard; shard != nil && ordinal%uint64(shard.Count) != uint64(shard.Index) {
			return fmt.Errorf("admitted ordinal %d is outside its shard assignment", ordinal)
		}
		_, committed := state.ordinals[ordinal]
		_, outstanding := state.frontier.outstanding[ordinal]
		_, repeated := seen[ordinal]
		if _, released := leaving[ordinal]; released {
			outstanding = false
		}
		if committed || outstanding || repeated {
			return fmt.Errorf("ordinal %d is already admitted or committed", ordinal)
		}
		seen[ordinal] = struct{}{}
	}
	return nil
}

func (state *campaignState) applyAdmissions(jobs []admittedJob) {
	state.frontier.admitted = true
	for _, job := range jobs {
		state.frontier.outstanding[uint64(job.Ordinal)] = uint64(job.Seed)
	}
}

func (state *campaignState) prepareAdmission(payload []byte) (func(), error) {
	if state.lifecycle != lifecycleRunning {
		return nil, fmt.Errorf("campaign admits ordinals in lifecycle %q", state.lifecycle)
	}
	var admission admissionRecord
	if err := logstore.DecodeJSON(payload, &admission); err != nil {
		return nil, err
	}
	if len(admission.Jobs) == 0 {
		return nil, errors.New("campaign admission reserves no ordinal")
	}
	if stop := state.pendingStop(); stop != "" {
		return nil, fmt.Errorf("pending %s stop freezes admission", stop)
	}
	if err := state.checkAdmissions(admission.Jobs, nil); err != nil {
		return nil, err
	}
	return func() { state.applyAdmissions(admission.Jobs) }, nil
}

func (state *campaignState) prepareReceipt(payload []byte) (func(), error) {
	if state.lifecycle != lifecycleRunning {
		return nil, fmt.Errorf("campaign records a receipt in lifecycle %q", state.lifecycle)
	}
	var receipt receiptRecord
	if err := logstore.DecodeJSON(payload, &receipt); err != nil {
		return nil, err
	}
	if err := state.checkReceipt(receipt); err != nil {
		return nil, err
	}
	ordinal := uint64(receipt.Observation.SelectionOrdinal)
	return func() { state.frontier.receipts[ordinal] = receipt }, nil
}

func (state *campaignState) checkReceipt(receipt receiptRecord) error {
	run := receipt.Observation
	ordinal := uint64(run.SelectionOrdinal)
	seed, admitted := state.frontier.outstanding[ordinal]
	if !admitted || seed != uint64(run.Seed) {
		return fmt.Errorf("receipt for ordinal %d seed %d has no admission", ordinal, run.Seed)
	}
	if _, duplicate := state.frontier.receipts[ordinal]; duplicate {
		return fmt.Errorf("receipt for ordinal %d conflicts with its durable receipt", ordinal)
	}
	if run.SuccessArtifact != nil || run.SuccessArtifactBytes != nil || len(run.NovelSemanticProbes) != 0 || len(run.NovelChoiceFeatures) != 0 {
		return fmt.Errorf("receipt for ordinal %d carries a retention decision", ordinal)
	}
	if run.Strategy != "" || run.Divergence != nil || run.Reason == "" {
		return fmt.Errorf("receipt for ordinal %d identity is invalid", ordinal)
	}
	if _, err := state.checkExecutionSize(run); err != nil {
		return err
	}
	pending := receipt.Pending
	if len(pending.Artifacts) > maximumPendingArtifacts || len(pending.Diagnostics) > maximumPendingDiagnostics {
		return fmt.Errorf("receipt for ordinal %d references more payloads than one attempt publishes", ordinal)
	}
	for _, reference := range pending.Artifacts {
		if !validArtifactReference(reference) && !validSuccessArtifactReference(reference) {
			return fmt.Errorf("receipt for ordinal %d references invalid payload %q", ordinal, reference)
		}
	}
	for _, file := range pending.Diagnostics {
		if file.Name == "" || file.Name != filepathBase(file.Name) || !validRecordSHA256(file.SHA256) {
			return fmt.Errorf("receipt for ordinal %d references an invalid diagnostic", ordinal)
		}
	}
	switch run.Domain {
	case "success":
		if run.Termination != "exit" || run.FailureSignature != nil || run.Artifact != nil {
			return fmt.Errorf("successful receipt for ordinal %d has failure evidence", ordinal)
		}
	case "target", "watchdog":
		if run.FailureSignature == nil || !validRecordSHA256(*run.FailureSignature) || run.Artifact == nil || !slices.Contains(pending.Artifacts, *run.Artifact) {
			return fmt.Errorf("failed receipt for ordinal %d does not reference its published evidence", ordinal)
		}
	case "runner":
		policy, _ := state.policy()
		if run.Reason != "runner_cancelled" || run.Termination != "none" || run.FailureSignature != nil || run.Artifact != nil {
			return fmt.Errorf("runner receipt for ordinal %d is not a cancellation", ordinal)
		}
		if policy != failurePolicyFirst || state.pendingStop() != StopFirstFailure {
			return fmt.Errorf("cancellation of ordinal %d has no first-failure stop to request it", ordinal)
		}
	default:
		return fmt.Errorf("receipt for ordinal %d domain is invalid: %s", ordinal, run.Domain)
	}
	return nil
}

func filepathBase(name string) string {
	for index := len(name) - 1; index >= 0; index-- {
		if name[index] == '/' || name[index] == '\\' {
			return name[index+1:]
		}
	}
	return name
}

// adoptedFrom reports whether outcome applies receipt's observation, adding
// only the retention and novelty decisions an ordered outcome makes.
func adoptedFrom(receipt receiptRecord, outcome ExecutionRecord) error {
	decided := outcome
	decided.SuccessArtifact, decided.SuccessArtifactBytes, decided.NovelSemanticProbes, decided.NovelChoiceFeatures = nil, nil, nil, nil
	if !reflect.DeepEqual(decided, receipt.Observation) {
		return fmt.Errorf("outcome for ordinal %d differs from its durable receipt", outcome.SelectionOrdinal)
	}
	if outcome.SuccessArtifact != nil && !slices.Contains(receipt.Pending.Artifacts, *outcome.SuccessArtifact) {
		return fmt.Errorf("outcome for ordinal %d retains evidence its receipt did not publish", outcome.SelectionOrdinal)
	}
	return nil
}

// checkFrontierOutcome validates an outcome transaction of a campaign that
// admits ordinals, and returns the ordinals it releases.
func (state *campaignState) checkFrontierOutcome(outcome outcomeRecord) (map[uint64]struct{}, error) {
	releases := make(map[uint64]struct{}, len(outcome.Executions))
	for _, run := range outcome.Executions {
		ordinal := uint64(run.SelectionOrdinal)
		seed, admitted := state.frontier.outstanding[ordinal]
		if !admitted || seed != uint64(run.Seed) {
			return nil, fmt.Errorf("outcome for ordinal %d seed %d has no admission", ordinal, run.Seed)
		}
		releases[ordinal] = struct{}{}
	}
	if len(outcome.Executions) == 1 && outcome.Stop == "" && hostFailureEvidence(outcome.Executions[0]) {
		if len(outcome.Admit) != 0 {
			return nil, errors.New("host failure evidence admits no ordinal")
		}
		return releases, nil
	}
	pending := state.pendingStop()
	if outcome.Stop == "" {
		if pending != "" {
			return nil, fmt.Errorf("pending %s stop must commit as one stop group", pending)
		}
		if len(outcome.Executions) != 1 {
			return nil, errors.New("ordered outcome transaction must commit one execution")
		}
		run := outcome.Executions[0]
		next, _ := state.minimumOutstanding()
		receipt, received := state.frontier.receipts[uint64(run.SelectionOrdinal)]
		if uint64(run.SelectionOrdinal) != next || !received {
			return nil, fmt.Errorf("outcome for ordinal %d is not the next ordered receipt", run.SelectionOrdinal)
		}
		if err := adoptedFrom(receipt, run); err != nil {
			return nil, err
		}
		if err := state.checkAdmissions(outcome.Admit, releases); err != nil {
			return nil, err
		}
		return releases, nil
	}
	if outcome.Stop != pending {
		return nil, fmt.Errorf("stop group %s does not match the pending stop %q", outcome.Stop, pending)
	}
	if len(outcome.Admit) != 0 {
		return nil, errors.New("stop group admits no ordinal")
	}
	if len(outcome.Executions) != len(state.frontier.outstanding) {
		return nil, fmt.Errorf("stop group commits %d of %d admitted ordinals", len(outcome.Executions), len(state.frontier.outstanding))
	}
	for index, run := range outcome.Executions {
		if index > 0 && outcome.Executions[index-1].SelectionOrdinal >= run.SelectionOrdinal {
			return nil, errors.New("stop group outcomes are not in ordinal order")
		}
		receipt, received := state.frontier.receipts[uint64(run.SelectionOrdinal)]
		if !received {
			return nil, fmt.Errorf("stop group member %d has no durable receipt", run.SelectionOrdinal)
		}
		if err := adoptedFrom(receipt, run); err != nil {
			return nil, err
		}
	}
	return releases, nil
}

// hostFailureEvidence is a Runner outcome other than a cancellation: the
// evidence of an attempt that failed the host. It is not a policy outcome,
// needs no receipt or order, and a resumed campaign runs its ordinal again.
func hostFailureEvidence(run ExecutionRecord) bool {
	return run.Domain == "runner" && run.Reason != "runner_cancelled"
}

func (state *campaignState) applyFrontierOutcome(outcome outcomeRecord, releases map[uint64]struct{}) {
	adopts := len(outcome.Executions) != 1 || outcome.Stop != "" || !hostFailureEvidence(outcome.Executions[0])
	for ordinal := range releases {
		if receipt, received := state.frontier.receipts[ordinal]; received && adopts {
			for _, file := range receipt.Pending.Diagnostics {
				state.frontier.diagnostics[file.Name] = file
			}
		}
		delete(state.frontier.outstanding, ordinal)
		delete(state.frontier.receipts, ordinal)
		if outcome.Stop != "" {
			state.frontier.stopGroup[ordinal] = struct{}{}
		}
	}
	for _, run := range outcome.Executions {
		state.countPolicyOutcome(run)
	}
	if outcome.Stop != "" {
		state.frontier.stopped = outcome.Stop
	}
	state.applyAdmissions(outcome.Admit)
}

func (state *campaignState) countPolicyOutcome(run ExecutionRecord) {
	if run.Domain != "target" && run.Domain != "watchdog" {
		return
	}
	if run.FailureSignature != nil {
		state.frontier.signatures[*run.FailureSignature] = struct{}{}
	}
}

// pendingLiveReferences projects the payloads of durable receipts, which stay
// live until their outcome adopts or retires them.
func (state *campaignState) pendingLiveReferences(add func(*string)) {
	for _, receipt := range state.frontier.receipts {
		for _, reference := range receipt.Pending.Artifacts {
			add(&reference)
		}
	}
}

// liveDiagnostics projects the diagnostic sidecars committed outcomes adopted
// and durable receipts still hold, in name order.
func (state *campaignState) liveDiagnostics() []artifact.FileReference {
	live := maps.Clone(state.frontier.diagnostics)
	for _, receipt := range state.frontier.receipts {
		for _, file := range receipt.Pending.Diagnostics {
			live[file.Name] = file
		}
	}
	files := slices.Collect(maps.Values(live))
	slices.SortFunc(files, func(left, right artifact.FileReference) int { return strings.Compare(left.Name, right.Name) })
	return files
}

func jobsOf(jobs []SeedJob) []admittedJob {
	admitted := make([]admittedJob, len(jobs))
	for index, job := range jobs {
		admitted[index] = admittedJob{Ordinal: record.Uint64String(job.Ordinal), Seed: record.Uint64String(job.Seed)}
	}
	return admitted
}

// Admit durably reserves ordinals before their attempts launch.
func (journal *CampaignJournal) Admit(jobs []SeedJob) error {
	if !journal.started {
		return errors.New("campaign execution journal is not open")
	}
	return journal.append(kindAdmission, admissionRecord{Jobs: jobsOf(jobs)})
}

// RecordReceipt durably records an admitted attempt's actual completion and
// the payloads it published, before its ordered outcome.
func (journal *CampaignJournal) RecordReceipt(receipt Receipt) error {
	if !journal.started {
		return errors.New("campaign execution journal is not open")
	}
	return journal.append(kindReceipt, receiptRecord{Observation: receipt.Observation, Pending: pendingReferences{Artifacts: receipt.Artifacts, Diagnostics: receipt.Diagnostics}})
}

// CommitOutcome commits the next ordered outcome, adopting or retiring its
// receipt's payloads, and atomically reserves the permitted refill.
func (journal *CampaignJournal) CommitOutcome(run ExecutionRecord, refill []SeedJob) error {
	if !journal.started {
		return errors.New("campaign execution journal is not open")
	}
	return journal.append(kindOutcome, outcomeRecord{Executions: []ExecutionRecord{run}, Admit: jobsOf(refill)})
}

// CommitStopGroup commits the pending stop: the trigger and every admitted
// ordinal's actual outcome, in ordinal order, as one transaction.
func (journal *CampaignJournal) CommitStopGroup(runs []ExecutionRecord) error {
	if !journal.started {
		return errors.New("campaign execution journal is not open")
	}
	journal.mu.Lock()
	defer journal.mu.Unlock()
	stop := journal.state.pendingStop()
	if stop == "" {
		return errors.New("campaign has no pending stop")
	}
	return journal.appendLocked(kindOutcome, outcomeRecord{Executions: runs, Stop: stop})
}

// Frontier reports the acknowledged admission state.
func (journal *CampaignJournal) Frontier() Frontier {
	journal.mu.Lock()
	defer journal.mu.Unlock()
	return journal.state.frontierSnapshot()
}

// checkPublishedFrontier refuses to publish a campaign that admits ordinals
// while admitted work is outstanding, or under a stop reason other than its
// committed stop.
func (state *campaignState) checkPublishedFrontier(published CampaignRecord) error {
	if !state.frontier.admitted {
		return nil
	}
	if len(state.frontier.outstanding) != 0 {
		return fmt.Errorf("campaign is published with %d admitted ordinals outstanding", len(state.frontier.outstanding))
	}
	stop := state.frontier.stopped
	if stop == "" {
		stop = StopSeedsExhausted
	}
	if published.StopReason != string(stop) {
		return fmt.Errorf("published stop reason %q is not the committed stop %q", published.StopReason, stop)
	}
	return nil
}

func (state *campaignState) stopGroupMember(ordinal uint64) bool {
	_, member := state.frontier.stopGroup[ordinal]
	return member
}
