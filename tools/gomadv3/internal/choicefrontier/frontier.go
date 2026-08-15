package choicefrontier

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"slices"

	"go.temporal.io/server/tools/gomadv3/internal/choicewire"
	"go.temporal.io/server/tools/gomadv3/internal/record"
)

const (
	candidateDomain    = "gomadv3-choice-frontier-candidate/v1"
	stateDomain        = "gomadv3-choice-frontier-state/v1"
	roundSegmentDomain = "gomadv3-choice-frontier-round-segment/v1"
	RoundSegmentSchema = "gomadv3.choice-frontier-round/v1"
	controllerIdentity = "deterministic-rounds/breadth-first-rank-prefix/v1"
)

func ImplementationSHA256() record.SHA256 {
	return record.DomainHash("gomadv3-choice-frontier-controller/v1", []byte(controllerIdentity))
}

type FailurePolicy string

const (
	PolicyFirst  FailurePolicy = "first"
	PolicyBudget FailurePolicy = "budget"
	PolicyAll    FailurePolicy = "all"
)

type StopReason string

const (
	StopExhausted        StopReason = "frontier_exhausted"
	StopDepthComplete    StopReason = "choice_depth_complete"
	StopMaxRuns          StopReason = "max_runs"
	StopFrontierCapacity StopReason = "frontier_capacity"
	StopFirstFailure     StopReason = "first_failure"
	StopFailureBudget    StopReason = "failure_budget"
)

type Config struct {
	Execution        choicewire.ExecutionIdentity `json:"execution"`
	ControllerSHA256 record.SHA256                `json:"controller_sha256"`
	BaseSeed         uint64                       `json:"base_seed"`
	Parallel         int                          `json:"parallel"`
	MaxRuns          uint64                       `json:"max_runs"`
	MaxChoiceDepth   uint64                       `json:"max_choice_depth"`
	MaxFrontierBytes uint64                       `json:"max_frontier_bytes"`
	FailurePolicy    FailurePolicy                `json:"failure_policy"`
	FailureBudget    uint64                       `json:"failure_budget"`
}

type Candidate struct {
	SHA256            record.SHA256 `json:"sha256"`
	ParentSHA256      record.SHA256 `json:"parent_sha256,omitempty"`
	SourceTraceSHA256 record.SHA256 `json:"source_trace_sha256,omitempty"`
	ForcedDepth       uint64        `json:"forced_depth"`
	PrefixSHA256      record.SHA256 `json:"prefix_sha256,omitempty"`
	PrefixBytes       []byte        `json:"prefix_bytes,omitempty"`
}

type Result struct {
	CandidateSHA256 record.SHA256
	OutcomeSHA256   record.SHA256
	Failed          bool
	FailureSHA256   record.SHA256
	Trace           *choicewire.Tape
}

type State struct {
	Config            Config          `json:"config"`
	Queue             []Candidate     `json:"queue"`
	Seen              []record.SHA256 `json:"seen"`
	Outcomes          []record.SHA256 `json:"outcomes"`
	FailureSignatures []record.SHA256 `json:"failure_signatures"`
	LogicalExecutions uint64          `json:"logical_executions"`
	CommittedRounds   uint64          `json:"committed_rounds"`
	PendingBytes      uint64          `json:"pending_bytes"`
	DeepestPrefix     uint64          `json:"deepest_prefix"`
	OmittedByRunBound uint64          `json:"omitted_by_run_bound"`
	OmittedByDepth    uint64          `json:"omitted_by_depth"`
	OmittedByCapacity uint64          `json:"omitted_by_capacity"`
	StopReason        StopReason      `json:"stop_reason,omitempty"`
}

type Round struct {
	Index      uint64      `json:"index"`
	Candidates []Candidate `json:"candidates"`
}

type SegmentResult struct {
	CandidateSHA256   record.SHA256 `json:"candidate_sha256"`
	OutcomeSHA256     record.SHA256 `json:"outcome_sha256"`
	Failed            bool          `json:"failed"`
	FailureSHA256     record.SHA256 `json:"failure_sha256,omitempty"`
	TraceSHA256       record.SHA256 `json:"trace_sha256,omitempty"`
	TraceSourceSHA256 record.SHA256 `json:"trace_source_sha256,omitempty"`
	TraceBytes        []byte        `json:"trace_bytes,omitempty"`
}

type RoundSegment struct {
	Schema       string          `json:"schema"`
	Index        uint64          `json:"index"`
	BeforeSHA256 record.SHA256   `json:"before_sha256"`
	AfterSHA256  record.SHA256   `json:"after_sha256"`
	Results      []SegmentResult `json:"results"`
	SHA256       record.SHA256   `json:"sha256,omitempty"`
}

type Summary struct {
	LogicalExecutions    uint64     `json:"logical_executions"`
	CommittedRounds      uint64     `json:"committed_rounds"`
	Pending              uint64     `json:"pending"`
	PendingBytes         uint64     `json:"pending_bytes"`
	SeenPrefixes         uint64     `json:"seen_prefixes"`
	DeduplicatedOutcomes uint64     `json:"deduplicated_outcomes"`
	DeepestPrefix        uint64     `json:"deepest_prefix"`
	OmittedByRunBound    uint64     `json:"omitted_by_run_bound"`
	OmittedByDepth       uint64     `json:"omitted_by_depth"`
	OmittedByCapacity    uint64     `json:"omitted_by_capacity"`
	StopReason           StopReason `json:"stop_reason,omitempty"`
	BoundedComplete      bool       `json:"bounded_complete"`
}

func New(config Config) (State, error) {
	if config.Parallel <= 0 || config.MaxRuns == 0 || config.MaxChoiceDepth == 0 || config.MaxFrontierBytes == 0 {
		return State{}, errors.New("choice frontier requires positive parallel, run, depth, and byte bounds")
	}
	if err := choicewire.ValidateExecutionIdentity(config.Execution); err != nil {
		return State{}, fmt.Errorf("choice frontier execution identity: %w", err)
	}
	if config.ControllerSHA256 != ImplementationSHA256() {
		return State{}, errors.New("choice frontier controller identity does not match this implementation")
	}
	switch config.FailurePolicy {
	case PolicyFirst, PolicyAll:
		if config.FailureBudget != 1 {
			return State{}, errors.New("choice frontier failure budget is only configurable in budget mode")
		}
	case PolicyBudget:
		if config.FailureBudget == 0 {
			return State{}, errors.New("choice frontier failure budget must be positive")
		}
	default:
		return State{}, fmt.Errorf("unknown choice frontier failure policy %q", config.FailurePolicy)
	}
	root, err := newCandidate(config, nil, "", "")
	if err != nil {
		return State{}, err
	}
	pendingBytes, err := queueBytes([]Candidate{root})
	if err != nil {
		return State{}, err
	}
	if pendingBytes > config.MaxFrontierBytes {
		return State{}, fmt.Errorf("choice frontier root requires %d bytes, exceeding the %d-byte frontier bound", pendingBytes, config.MaxFrontierBytes)
	}
	return State{Config: config, Queue: []Candidate{root}, Seen: []record.SHA256{root.SHA256}, Outcomes: []record.SHA256{}, FailureSignatures: []record.SHA256{}, PendingBytes: pendingBytes}, nil
}

func (state State) NextRound() (Round, bool) {
	if state.StopReason != "" || len(state.Queue) == 0 {
		return Round{}, false
	}
	count := min(state.Config.Parallel, len(state.Queue))
	candidates := make([]Candidate, count)
	for index := range candidates {
		candidates[index] = cloneCandidate(state.Queue[index])
	}
	return Round{Index: state.CommittedRounds, Candidates: candidates}, true
}

func (state State) Summary() Summary {
	return Summary{
		LogicalExecutions: state.LogicalExecutions, CommittedRounds: state.CommittedRounds,
		Pending: uint64(len(state.Queue)), PendingBytes: state.PendingBytes, SeenPrefixes: uint64(len(state.Seen)),
		DeduplicatedOutcomes: uint64(len(state.Outcomes)), DeepestPrefix: state.DeepestPrefix,
		OmittedByRunBound: state.OmittedByRunBound, OmittedByDepth: state.OmittedByDepth, OmittedByCapacity: state.OmittedByCapacity,
		StopReason: state.StopReason, BoundedComplete: state.StopReason == StopExhausted || state.StopReason == StopDepthComplete,
	}
}

func (candidate Candidate) PrefixTape(identity choicewire.ExecutionIdentity) (choicewire.Tape, error) {
	if candidate.ForcedDepth == 0 {
		if candidate.PrefixSHA256 != "" || len(candidate.PrefixBytes) != 0 {
			return choicewire.Tape{}, errors.New("root choice candidate contains a prefix")
		}
		return choicewire.Tape{}, nil
	}
	digest, err := candidate.PrefixSHA256.Bytes()
	if err != nil {
		return choicewire.Tape{}, fmt.Errorf("decode choice candidate prefix identity: %w", err)
	}
	tape := choicewire.Tape{Identity: identity, Bytes: append([]byte(nil), candidate.PrefixBytes...), SHA256: digest}
	return choicewire.ValidatePrefixTape(tape, identity)
}

func CommitRound(state State, round Round, results []Result) (State, RoundSegment, error) {
	before, err := stateIdentity(state)
	if err != nil {
		return State{}, RoundSegment{}, err
	}
	expected, ok := state.NextRound()
	if !ok || round.Index != expected.Index || !sameCandidates(round.Candidates, expected.Candidates) {
		return State{}, RoundSegment{}, errors.New("choice frontier round does not match the current state")
	}
	if len(results) != len(round.Candidates) {
		return State{}, RoundSegment{}, errors.New("choice frontier result count does not match its round")
	}
	next := cloneState(state)
	next.Queue = next.Queue[len(round.Candidates):]
	next.LogicalExecutions += uint64(len(results))
	next.CommittedRounds++
	segmentResults := make([]SegmentResult, len(results))
	newChildren := make(map[record.SHA256]Candidate)
	policyStopped := false
	for index, result := range results {
		candidate := round.Candidates[index]
		if result.CandidateSHA256 != candidate.SHA256 {
			return State{}, RoundSegment{}, fmt.Errorf("choice frontier result %d does not match candidate", index)
		}
		if _, err := result.OutcomeSHA256.Bytes(); err != nil {
			return State{}, RoundSegment{}, fmt.Errorf("choice frontier result %d outcome: %w", index, err)
		}
		next.Outcomes = insertIdentity(next.Outcomes, result.OutcomeSHA256)
		if result.Failed {
			if _, err := result.FailureSHA256.Bytes(); err != nil {
				return State{}, RoundSegment{}, fmt.Errorf("choice frontier result %d failure signature: %w", index, err)
			}
			next.FailureSignatures = insertIdentity(next.FailureSignatures, result.FailureSHA256)
			switch next.Config.FailurePolicy {
			case PolicyFirst:
				next.StopReason = StopFirstFailure
				policyStopped = true
			case PolicyBudget:
				if uint64(len(next.FailureSignatures)) >= next.Config.FailureBudget {
					next.StopReason = StopFailureBudget
					policyStopped = true
				}
			}
		}
		segmentResult := SegmentResult{CandidateSHA256: result.CandidateSHA256, OutcomeSHA256: result.OutcomeSHA256, Failed: result.Failed, FailureSHA256: result.FailureSHA256}
		if result.Trace != nil {
			trace, validateErr := choicewire.ValidateDecisionTape(*result.Trace, next.Config.Execution)
			if validateErr != nil {
				return State{}, RoundSegment{}, fmt.Errorf("validate choice frontier result %d trace: %w", index, validateErr)
			}
			if validateErr := validateCandidateTrace(candidate, trace, next.Config.Execution); validateErr != nil {
				return State{}, RoundSegment{}, fmt.Errorf("validate choice frontier result %d prefix: %w", index, validateErr)
			}
			segmentResult.TraceSHA256 = record.SHA256FromSum(trace.SHA256)
			segmentResult.TraceSourceSHA256 = record.SHA256FromSum(trace.SourceTraceSHA256)
			segmentResult.TraceBytes = append([]byte(nil), trace.Bytes...)
			if !policyStopped {
				if err := expandCandidate(&next, candidate, trace, newChildren); err != nil {
					return State{}, RoundSegment{}, err
				}
			}
		}
		segmentResults[index] = segmentResult
	}
	children := make([]Candidate, 0, len(newChildren))
	for _, candidate := range newChildren {
		children = append(children, candidate)
	}
	sortCandidates(children)
	if err := admitChildren(&next, children); err != nil {
		return State{}, RoundSegment{}, err
	}
	if next.StopReason == "" && len(next.Queue) == 0 {
		switch {
		case next.OmittedByRunBound != 0 || next.LogicalExecutions >= next.Config.MaxRuns:
			next.StopReason = StopMaxRuns
		case next.OmittedByDepth != 0:
			next.StopReason = StopDepthComplete
		default:
			next.StopReason = StopExhausted
		}
	}
	next.PendingBytes, err = queueBytes(next.Queue)
	if err != nil {
		return State{}, RoundSegment{}, err
	}
	after, err := stateIdentity(next)
	if err != nil {
		return State{}, RoundSegment{}, err
	}
	segment := RoundSegment{Schema: RoundSegmentSchema, Index: round.Index, BeforeSHA256: before, AfterSHA256: after, Results: segmentResults}
	segment.SHA256, err = segmentIdentity(segment)
	if err != nil {
		return State{}, RoundSegment{}, err
	}
	return next, segment, nil
}

func ReplaySegment(state State, segment RoundSegment) (State, error) {
	identity, err := segmentIdentity(segment)
	if err != nil || identity != segment.SHA256 {
		return State{}, errors.Join(errors.New("choice frontier segment identity does not match"), err)
	}
	before, err := stateIdentity(state)
	if err != nil || before != segment.BeforeSHA256 || segment.Schema != RoundSegmentSchema {
		return State{}, errors.Join(errors.New("choice frontier segment does not link to its state"), err)
	}
	round, ok := state.NextRound()
	if !ok || round.Index != segment.Index || len(round.Candidates) != len(segment.Results) {
		return State{}, errors.New("choice frontier segment round is unavailable")
	}
	results := make([]Result, len(segment.Results))
	for index, stored := range segment.Results {
		result := Result{CandidateSHA256: stored.CandidateSHA256, OutcomeSHA256: stored.OutcomeSHA256, Failed: stored.Failed, FailureSHA256: stored.FailureSHA256}
		if len(stored.TraceBytes) != 0 {
			digest, decodeErr := stored.TraceSHA256.Bytes()
			if decodeErr != nil {
				return State{}, decodeErr
			}
			tape := choicewire.Tape{Identity: state.Config.Execution, Bytes: append([]byte(nil), stored.TraceBytes...), SHA256: digest}
			validated, validateErr := choicewire.ValidateDecisionTape(tape, state.Config.Execution)
			if validateErr != nil || record.SHA256FromSum(validated.SourceTraceSHA256) != stored.TraceSourceSHA256 {
				return State{}, errors.Join(errors.New("choice frontier segment trace is invalid"), validateErr)
			}
			result.Trace = &validated
		}
		results[index] = result
	}
	next, regenerated, err := CommitRound(state, round, results)
	if err != nil {
		return State{}, err
	}
	left, err := record.CanonicalJSON(segment)
	if err != nil {
		return State{}, err
	}
	right, err := record.CanonicalJSON(regenerated)
	if err != nil {
		return State{}, err
	}
	after, err := stateIdentity(next)
	if err != nil || !bytes.Equal(left, right) || after != segment.AfterSHA256 {
		return State{}, errors.Join(errors.New("choice frontier segment replay changed its canonical result"), err)
	}
	return next, nil
}

func expandCandidate(state *State, parent Candidate, trace choicewire.Tape, children map[record.SHA256]Candidate) error {
	for ordinal, decision := range trace.Decisions {
		if uint64(ordinal) < parent.ForcedDepth {
			continue
		}
		if decision.Alternatives <= 1 {
			continue
		}
		depth := uint64(ordinal + 1)
		if depth > state.Config.MaxChoiceDepth {
			state.OmittedByDepth += uint64(decision.Alternatives - 1)
			continue
		}
		for rank := uint32(0); rank < decision.Alternatives; rank++ {
			if rank == decision.Selected {
				continue
			}
			prefix, err := choicewire.BuildRankPrefix(trace, uint64(ordinal), rank)
			if err != nil {
				return fmt.Errorf("build choice frontier child at decision %d rank %d: %w", ordinal, rank, err)
			}
			child, err := newCandidate(state.Config, &prefix, parent.SHA256, record.SHA256FromSum(trace.SourceTraceSHA256))
			if err != nil {
				return err
			}
			if containsIdentity(state.Seen, child.SHA256) {
				continue
			}
			if existing, found := children[child.SHA256]; found && !sameCandidate(existing, child) {
				return errors.New("choice frontier candidate identity collision")
			}
			children[child.SHA256] = child
		}
	}
	return nil
}

func admitChildren(state *State, children []Candidate) error {
	pendingBytes, err := queueBytes(state.Queue)
	if err != nil {
		return err
	}
	for index, candidate := range children {
		if uint64(len(state.Seen)) >= state.Config.MaxRuns {
			state.OmittedByRunBound += uint64(len(children) - index)
			break
		}
		entryBytes, err := candidateBytes(candidate)
		if err != nil {
			return err
		}
		if entryBytes > ^uint64(0)-pendingBytes {
			return errors.New("choice frontier byte accounting overflow")
		}
		if pendingBytes+entryBytes > state.Config.MaxFrontierBytes {
			state.OmittedByCapacity += uint64(len(children) - index)
			if state.StopReason == "" {
				state.StopReason = StopFrontierCapacity
			}
			break
		}
		pendingBytes += entryBytes
		state.Queue = append(state.Queue, candidate)
		state.Seen = insertIdentity(state.Seen, candidate.SHA256)
		state.DeepestPrefix = max(state.DeepestPrefix, candidate.ForcedDepth)
	}
	sortCandidates(state.Queue)
	state.PendingBytes = pendingBytes
	return nil
}

type candidateIdentityProjection struct {
	TargetSHA256         record.SHA256                `json:"target_sha256"`
	ToolchainBuildKey    string                       `json:"toolchain_build_key"`
	GOOS                 string                       `json:"goos"`
	GOARCH               string                       `json:"goarch"`
	ImplementationSHA256 record.SHA256                `json:"implementation_sha256"`
	ControllerSHA256     record.SHA256                `json:"controller_sha256"`
	BaseSeed             record.Uint64String          `json:"base_seed"`
	Decisions            []decisionIdentityProjection `json:"decisions"`
}

type decisionIdentityProjection struct {
	Ordinal              record.Uint64String `json:"ordinal"`
	Kind                 uint8               `json:"kind"`
	SiteOffset           record.Uint64String `json:"site_offset"`
	SiteMissing          bool                `json:"site_missing"`
	RankOverride         bool                `json:"rank_override"`
	Alternatives         uint32              `json:"alternatives"`
	Selected             uint32              `json:"selected"`
	Data                 uint32              `json:"data"`
	SelectedIdentity     record.SHA256       `json:"selected_identity,omitempty"`
	AlternativeSetDigest record.SHA256       `json:"alternative_set_digest"`
}

func newCandidate(config Config, prefix *choicewire.Tape, parent, source record.SHA256) (Candidate, error) {
	decisions := []choicewire.Decision{}
	candidate := Candidate{ParentSHA256: parent, SourceTraceSHA256: source}
	if prefix != nil {
		validated, err := choicewire.ValidatePrefixTape(*prefix, config.Execution)
		if err != nil {
			return Candidate{}, err
		}
		decisions = validated.Decisions
		candidate.ForcedDepth = uint64(len(decisions))
		candidate.PrefixSHA256 = record.SHA256FromSum(validated.SHA256)
		candidate.PrefixBytes = append([]byte(nil), validated.Bytes...)
	}
	projection := candidateIdentityProjection{
		TargetSHA256: record.SHA256FromSum(config.Execution.TargetSHA256), ToolchainBuildKey: config.Execution.ToolchainBuildKey,
		GOOS: config.Execution.GOOS, GOARCH: config.Execution.GOARCH, ImplementationSHA256: record.SHA256FromSum(config.Execution.ImplementationSHA256),
		ControllerSHA256: config.ControllerSHA256,
		BaseSeed:         record.Uint64String(config.BaseSeed), Decisions: make([]decisionIdentityProjection, len(decisions)),
	}
	for index, decision := range decisions {
		selected := record.SHA256("")
		if decision.SelectedIdentity != ([sha256.Size]byte{}) {
			selected = record.SHA256FromSum(decision.SelectedIdentity)
		}
		projection.Decisions[index] = decisionIdentityProjection{
			Ordinal: record.Uint64String(index), Kind: uint8(decision.Kind), SiteOffset: record.Uint64String(decision.SiteOffset),
			SiteMissing: decision.SiteMissing, RankOverride: decision.RankOverride, Alternatives: decision.Alternatives,
			Selected: decision.Selected, Data: decision.Data, SelectedIdentity: selected, AlternativeSetDigest: record.SHA256FromSum(decision.AlternativeSetDigest),
		}
	}
	encoded, err := record.CanonicalJSON(projection)
	if err != nil {
		return Candidate{}, err
	}
	candidate.SHA256 = record.DomainHash(candidateDomain, encoded)
	return candidate, nil
}

func validateCandidateTrace(candidate Candidate, trace choicewire.Tape, identity choicewire.ExecutionIdentity) error {
	if candidate.ForcedDepth == 0 {
		return nil
	}
	prefix, err := candidate.PrefixTape(identity)
	if err != nil {
		return err
	}
	if len(trace.Decisions) < len(prefix.Decisions) {
		return errors.New("choice trace ended before its forced prefix")
	}
	for index, expected := range prefix.Decisions {
		observed := trace.Decisions[index]
		if expected.RankOverride {
			if observed.RankOverride || expected.Kind != observed.Kind || expected.SiteOffset != observed.SiteOffset || expected.SiteMissing != observed.SiteMissing || expected.Alternatives != observed.Alternatives || expected.Selected != observed.Selected || expected.Data != observed.Data || expected.AlternativeSetDigest != observed.AlternativeSetDigest || observed.SelectedIdentity == ([sha256.Size]byte{}) {
				return fmt.Errorf("choice rank override diverged at decision %d", index)
			}
			continue
		}
		if expected != observed {
			return fmt.Errorf("choice prefix diverged at decision %d", index)
		}
	}
	return nil
}

func stateIdentity(state State) (record.SHA256, error) {
	encoded, err := record.CanonicalJSON(state)
	if err != nil {
		return "", err
	}
	return record.DomainHash(stateDomain, encoded), nil
}

func StateSHA256(state State) (record.SHA256, error) {
	return stateIdentity(state)
}

func segmentIdentity(segment RoundSegment) (record.SHA256, error) {
	copy := segment
	copy.SHA256 = ""
	encoded, err := record.CanonicalJSON(copy)
	if err != nil {
		return "", err
	}
	return record.DomainHash(roundSegmentDomain, encoded), nil
}

func candidateBytes(candidate Candidate) (uint64, error) {
	encoded, err := record.CanonicalJSON(candidate)
	if err != nil {
		return 0, err
	}
	return uint64(len(encoded)) + 1, nil
}

func queueBytes(queue []Candidate) (uint64, error) {
	var total uint64
	for _, candidate := range queue {
		bytes, err := candidateBytes(candidate)
		if err != nil {
			return 0, err
		}
		if bytes > ^uint64(0)-total {
			return 0, errors.New("choice frontier byte accounting overflow")
		}
		total += bytes
	}
	return total, nil
}

func insertIdentity(values []record.SHA256, value record.SHA256) []record.SHA256 {
	index, found := slices.BinarySearch(values, value)
	if found {
		return values
	}
	values = append(values, "")
	copy(values[index+1:], values[index:])
	values[index] = value
	return values
}

func containsIdentity(values []record.SHA256, value record.SHA256) bool {
	_, found := slices.BinarySearch(values, value)
	return found
}

func sortCandidates(candidates []Candidate) {
	slices.SortFunc(candidates, func(left, right Candidate) int {
		if left.ForcedDepth < right.ForcedDepth {
			return -1
		}
		if left.ForcedDepth > right.ForcedDepth {
			return 1
		}
		return bytes.Compare([]byte(left.SHA256), []byte(right.SHA256))
	})
}

func sameCandidates(left, right []Candidate) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if !sameCandidate(left[index], right[index]) {
			return false
		}
	}
	return true
}

func sameCandidate(left, right Candidate) bool {
	return left.SHA256 == right.SHA256 && left.ParentSHA256 == right.ParentSHA256 && left.SourceTraceSHA256 == right.SourceTraceSHA256 && left.ForcedDepth == right.ForcedDepth && left.PrefixSHA256 == right.PrefixSHA256 && bytes.Equal(left.PrefixBytes, right.PrefixBytes)
}

func cloneCandidate(candidate Candidate) Candidate {
	candidate.PrefixBytes = append([]byte(nil), candidate.PrefixBytes...)
	return candidate
}

func cloneState(state State) State {
	copy := state
	copy.Queue = make([]Candidate, len(state.Queue))
	for index := range state.Queue {
		copy.Queue[index] = cloneCandidate(state.Queue[index])
	}
	copy.Seen = append([]record.SHA256(nil), state.Seen...)
	copy.Outcomes = append([]record.SHA256(nil), state.Outcomes...)
	copy.FailureSignatures = append([]record.SHA256(nil), state.FailureSignatures...)
	return copy
}
