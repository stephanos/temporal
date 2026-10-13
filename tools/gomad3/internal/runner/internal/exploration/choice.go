package exploration

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"unicode/utf8"

	"go.temporal.io/server/tools/gomad3/internal/evidence/record"
	"go.temporal.io/server/tools/gomad3/internal/model/choice"
)

const (
	choiceCandidateDomain    = "gomad3-choice-exploration-candidate/v1"
	choiceStateDomain        = "gomad3-choice-exploration-state/v1"
	choiceRoundSegmentDomain = "gomad3-choice-exploration-round-segment/v2"
	ChoiceRoundSegmentSchema = "gomad3.choice-exploration-round/v2"
	// controllerIdentity names the expansion rule, so a journal written under
	// another rule is refused on resume rather than continued under this one.
	// v3 leaves the select-poll decisions of a no-op shape unexpanded
	// (select_readiness.go).
	choiceControllerIdentity = "deterministic-rounds/breadth-first-rank-prefix/v3"
)

func ChoiceImplementationSHA256() record.SHA256 {
	return record.DomainHash("gomad3-choice-exploration-controller/v1", []byte(choiceControllerIdentity))
}

type FailurePolicy string

const (
	PolicyFirst  FailurePolicy = "first"
	PolicyBudget FailurePolicy = "budget"
	PolicyAll    FailurePolicy = "all"
)

type ChoiceStopReason string

const (
	ChoiceStopExhausted           ChoiceStopReason = "exploration_exhausted"
	ChoiceStopDepthComplete       ChoiceStopReason = "choice_depth_complete"
	ChoiceStopMaxExecutions       ChoiceStopReason = "max_executions"
	ChoiceStopExplorationCapacity ChoiceStopReason = "exploration_capacity"
	ChoiceStopFirstFailure        ChoiceStopReason = "first_failure"
	ChoiceStopFailureBudget       ChoiceStopReason = "failure_budget"
	ChoiceStopStartUnreached      ChoiceStopReason = "choice_start_unreached"
)

type ChoiceConfig struct {
	Execution           choice.ExecutionIdentity `json:"execution"`
	ControllerSHA256    record.SHA256            `json:"controller_sha256"`
	BaseSeed            uint64                   `json:"base_seed"`
	Parallel            int                      `json:"parallel"`
	MaxExecutions       uint64                   `json:"max_executions"`
	MaxChoiceDepth      uint64                   `json:"max_choice_depth"`
	StartOrdinal        uint64                   `json:"start_ordinal,omitempty"`
	MaxExplorationBytes uint64                   `json:"max_exploration_bytes"`
	FailurePolicy       FailurePolicy            `json:"failure_policy"`
	FailureBudget       uint64                   `json:"failure_budget"`
}

type ChoiceCandidate struct {
	SHA256            record.SHA256 `json:"sha256"`
	ParentSHA256      record.SHA256 `json:"parent_sha256,omitempty"`
	SourceTraceSHA256 record.SHA256 `json:"source_trace_sha256,omitempty"`
	ForcedDepth       uint64        `json:"forced_depth"`
	PrefixSHA256      record.SHA256 `json:"prefix_sha256,omitempty"`
	PrefixBytes       []byte        `json:"prefix_bytes,omitempty"`
}

type ChoiceResult struct {
	CandidateSHA256 record.SHA256
	OutcomeSHA256   record.SHA256
	Failed          bool
	FailureSHA256   record.SHA256
	Divergence      *choice.Divergence
	Trace           *choice.ReplayPlan
}

type ChoiceState struct {
	Config                  ChoiceConfig      `json:"config"`
	Queue                   []ChoiceCandidate `json:"queue"`
	Seen                    []record.SHA256   `json:"seen"`
	Outcomes                []record.SHA256   `json:"outcomes"`
	FailureSignatures       []record.SHA256   `json:"failure_signatures"`
	LogicalExecutions       uint64            `json:"logical_executions"`
	CommittedRounds         uint64            `json:"committed_rounds"`
	PendingBytes            uint64            `json:"pending_bytes"`
	DeepestPrefix           uint64            `json:"deepest_prefix"`
	OmittedByExecutionBound uint64            `json:"omitted_by_execution_bound"`
	OmittedByDepth          uint64            `json:"omitted_by_depth"`
	OmittedByCapacity       uint64            `json:"omitted_by_capacity"`
	// OmittedBySelectReadiness counts the alternatives of no-op select-poll
	// decisions. Unlike the bounds above it leaves nothing unexplored, so it
	// does not make a stop reason.
	OmittedBySelectReadiness uint64           `json:"omitted_by_select_readiness"`
	StopReason               ChoiceStopReason `json:"stop_reason,omitempty"`
}

type ChoiceRound struct {
	Index      uint64            `json:"index"`
	Candidates []ChoiceCandidate `json:"candidates"`
}

type ChoiceSegmentResult struct {
	CandidateSHA256   record.SHA256              `json:"candidate_sha256"`
	OutcomeSHA256     record.SHA256              `json:"outcome_sha256,omitempty"`
	Failed            bool                       `json:"failed"`
	FailureSHA256     record.SHA256              `json:"failure_sha256,omitempty"`
	Divergence        *choice.DivergenceEvidence `json:"divergence,omitempty"`
	TraceSHA256       record.SHA256              `json:"trace_sha256,omitempty"`
	TraceSourceSHA256 record.SHA256              `json:"trace_source_sha256,omitempty"`
	TraceBytes        []byte                     `json:"trace_bytes,omitempty"`
}

type ChoiceRoundSegment struct {
	Schema       string                `json:"schema"`
	Index        uint64                `json:"index"`
	BeforeSHA256 record.SHA256         `json:"before_sha256"`
	AfterSHA256  record.SHA256         `json:"after_sha256"`
	Results      []ChoiceSegmentResult `json:"results"`
	SHA256       record.SHA256         `json:"sha256,omitempty"`
}

type ChoiceSummary struct {
	Parallel                 int              `json:"parallel"`
	MaxExecutions            uint64           `json:"max_executions"`
	MaxChoiceDepth           uint64           `json:"max_choice_depth"`
	StartOrdinal             uint64           `json:"start_ordinal,omitempty"`
	MaxExplorationBytes      uint64           `json:"max_exploration_bytes"`
	LogicalExecutions        uint64           `json:"logical_executions"`
	CommittedRounds          uint64           `json:"committed_rounds"`
	Pending                  uint64           `json:"pending"`
	PendingBytes             uint64           `json:"pending_bytes"`
	SeenPrefixes             uint64           `json:"seen_prefixes"`
	DeduplicatedOutcomes     uint64           `json:"deduplicated_outcomes"`
	DeepestPrefix            uint64           `json:"deepest_prefix"`
	OmittedByExecutionBound  uint64           `json:"omitted_by_execution_bound"`
	OmittedByDepth           uint64           `json:"omitted_by_depth"`
	OmittedByCapacity        uint64           `json:"omitted_by_capacity"`
	OmittedBySelectReadiness uint64           `json:"omitted_by_select_readiness"`
	StopReason               ChoiceStopReason `json:"stop_reason,omitempty"`
	BoundedComplete          bool             `json:"bounded_complete"`
}

func NewChoice(config ChoiceConfig) (ChoiceState, error) {
	if config.Parallel <= 0 || config.MaxExecutions == 0 || config.MaxChoiceDepth == 0 || config.MaxExplorationBytes == 0 {
		return ChoiceState{}, errors.New("choice exploration requires positive parallel, execution, depth, and byte bounds")
	}
	if err := choice.ValidateExecutionIdentity(config.Execution); err != nil {
		//gomad:dynamic reviewed 2026-10-12 reviewed codec and validator errors have memory-only methods
		return ChoiceState{}, fmt.Errorf("choice exploration execution identity: %w", err)
	}
	if config.ControllerSHA256 != ChoiceImplementationSHA256() {
		return ChoiceState{}, errors.New("choice exploration controller identity does not match this implementation")
	}
	switch config.FailurePolicy {
	case PolicyFirst, PolicyAll:
		if config.FailureBudget != 1 {
			return ChoiceState{}, errors.New("choice exploration failure budget is only configurable in budget mode")
		}
	case PolicyBudget:
		if config.FailureBudget == 0 {
			return ChoiceState{}, errors.New("choice exploration failure budget must be positive")
		}
	default:
		return ChoiceState{}, fmt.Errorf("unknown choice exploration failure policy %q", config.FailurePolicy)
	}
	root, err := choiceNewCandidate(config, nil, "", "")
	if err != nil {
		return ChoiceState{}, err
	}
	pendingBytes, err := choiceQueueBytes([]ChoiceCandidate{root})
	if err != nil {
		return ChoiceState{}, err
	}
	if pendingBytes > config.MaxExplorationBytes {
		return ChoiceState{}, fmt.Errorf("choice exploration root requires %d bytes, exceeding the %d-byte exploration bound", pendingBytes, config.MaxExplorationBytes)
	}
	return ChoiceState{Config: config, Queue: []ChoiceCandidate{root}, Seen: []record.SHA256{root.SHA256}, Outcomes: []record.SHA256{}, FailureSignatures: []record.SHA256{}, PendingBytes: pendingBytes}, nil
}

func (state ChoiceState) NextRound() (ChoiceRound, bool) {
	candidates, ok := NextRound(state.Queue, state.Config.Parallel, state.StopReason != "", choiceCloneCandidate)
	return ChoiceRound{Index: state.CommittedRounds, Candidates: candidates}, ok
}

func (state ChoiceState) Summary() ChoiceSummary {
	return ChoiceSummary{
		Parallel: state.Config.Parallel, MaxExecutions: state.Config.MaxExecutions, MaxChoiceDepth: state.Config.MaxChoiceDepth, StartOrdinal: state.Config.StartOrdinal, MaxExplorationBytes: state.Config.MaxExplorationBytes,
		LogicalExecutions: state.LogicalExecutions, CommittedRounds: state.CommittedRounds,
		Pending: uint64(len(state.Queue)), PendingBytes: state.PendingBytes, SeenPrefixes: uint64(len(state.Seen)),
		DeduplicatedOutcomes: uint64(len(state.Outcomes)), DeepestPrefix: state.DeepestPrefix,
		OmittedByExecutionBound: state.OmittedByExecutionBound, OmittedByDepth: state.OmittedByDepth, OmittedByCapacity: state.OmittedByCapacity, OmittedBySelectReadiness: state.OmittedBySelectReadiness,
		StopReason: state.StopReason, BoundedComplete: state.StopReason == ChoiceStopExhausted || state.StopReason == ChoiceStopDepthComplete,
	}
}

func (candidate ChoiceCandidate) PrefixReplayPlan(identity choice.ExecutionIdentity) (choice.ReplayPlan, error) {
	if candidate.ForcedDepth == 0 {
		if candidate.PrefixSHA256 != "" || len(candidate.PrefixBytes) != 0 {
			return choice.ReplayPlan{}, errors.New("root choice candidate contains a prefix")
		}
		return choice.ReplayPlan{}, nil
	}
	digest, err := candidate.PrefixSHA256.Bytes()
	if err != nil {
		//gomad:dynamic reviewed 2026-10-12 reviewed codec and validator errors have memory-only methods
		return choice.ReplayPlan{}, fmt.Errorf("decode choice candidate prefix identity: %w", err)
	}
	tape := choice.ReplayPlan{Identity: identity, Bytes: append([]byte(nil), candidate.PrefixBytes...), SHA256: digest}
	return choice.ValidatePrefixReplayPlan(tape, identity)
}

func CommitChoiceRound(state ChoiceState, round ChoiceRound, results []ChoiceResult) (ChoiceState, ChoiceRoundSegment, error) {
	before, err := choiceStateIdentity(state)
	if err != nil {
		return ChoiceState{}, ChoiceRoundSegment{}, err
	}
	expected, ok := state.NextRound()
	if !ok || round.Index != expected.Index || !choiceSameCandidates(round.Candidates, expected.Candidates) {
		return ChoiceState{}, ChoiceRoundSegment{}, errors.New("choice exploration round does not match the current state")
	}
	if len(results) != len(round.Candidates) {
		return ChoiceState{}, ChoiceRoundSegment{}, errors.New("choice exploration result count does not match its round")
	}
	next := choiceCloneState(state)
	next.Queue = next.Queue[len(round.Candidates):]
	next.LogicalExecutions += uint64(len(results))
	next.CommittedRounds++
	segmentResults := make([]ChoiceSegmentResult, len(results))
	newChildren := make(map[record.SHA256]ChoiceCandidate)
	policyStopped := false
	startUnreached := false
	for index, result := range results {
		candidate := round.Candidates[index]
		if result.CandidateSHA256 != candidate.SHA256 {
			return ChoiceState{}, ChoiceRoundSegment{}, fmt.Errorf("choice exploration result %d does not match candidate", index)
		}
		if result.Divergence != nil {
			if result.OutcomeSHA256 != "" || result.Failed || result.FailureSHA256 != "" || result.Trace != nil {
				return ChoiceState{}, ChoiceRoundSegment{}, fmt.Errorf("choice exploration result %d divergence contains outcome evidence", index)
			}
			if err := ValidateCandidateDivergence(candidate, state.Config.Execution, *result.Divergence); err != nil {
				//gomad:dynamic reviewed 2026-10-12 reviewed codec and validator errors have memory-only methods
				return ChoiceState{}, ChoiceRoundSegment{}, fmt.Errorf("choice exploration result %d divergence: %w", index, err)
			}
			divergence := choice.ProjectDivergenceEvidence(*result.Divergence)
			segmentResults[index] = ChoiceSegmentResult{CandidateSHA256: candidate.SHA256, Divergence: &divergence}
			if next.Config.FailurePolicy == PolicyFirst {
				next.StopReason = ChoiceStopFirstFailure
				policyStopped = true
			}
			continue
		}
		if _, err := result.OutcomeSHA256.Bytes(); err != nil {
			//gomad:dynamic reviewed 2026-10-12 reviewed codec and validator errors have memory-only methods
			return ChoiceState{}, ChoiceRoundSegment{}, fmt.Errorf("choice exploration result %d outcome: %w", index, err)
		}
		next.Outcomes = choiceInsertIdentity(next.Outcomes, result.OutcomeSHA256)
		if result.Failed {
			if _, err := result.FailureSHA256.Bytes(); err != nil {
				//gomad:dynamic reviewed 2026-10-12 reviewed codec and validator errors have memory-only methods
				return ChoiceState{}, ChoiceRoundSegment{}, fmt.Errorf("choice exploration result %d failure signature: %w", index, err)
			}
			next.FailureSignatures = choiceInsertIdentity(next.FailureSignatures, result.FailureSHA256)
			if next.Config.FailurePolicy == PolicyFirst {
				next.StopReason = ChoiceStopFirstFailure
				policyStopped = true
			} else if next.Config.FailurePolicy == PolicyBudget && uint64(len(next.FailureSignatures)) >= next.Config.FailureBudget {
				next.StopReason = ChoiceStopFailureBudget
				policyStopped = true
			}
		}
		segmentResult := ChoiceSegmentResult{CandidateSHA256: result.CandidateSHA256, OutcomeSHA256: result.OutcomeSHA256, Failed: result.Failed, FailureSHA256: result.FailureSHA256}
		if candidate.ForcedDepth == 0 && next.Config.StartOrdinal != 0 && (result.Trace == nil || uint64(len(result.Trace.Decisions)) <= next.Config.StartOrdinal) {
			startUnreached = true
		}
		if result.Trace != nil {
			trace, validateErr := choice.ValidateReplayPlan(*result.Trace, next.Config.Execution)
			if validateErr != nil {
				//gomad:dynamic reviewed 2026-10-12 reviewed codec and validator errors have memory-only methods
				return ChoiceState{}, ChoiceRoundSegment{}, fmt.Errorf("validate choice exploration result %d trace: %w", index, validateErr)
			}
			if validateErr := validateCandidateTrace(candidate, trace, next.Config.Execution); validateErr != nil {
				//gomad:dynamic reviewed 2026-10-12 reviewed codec and validator errors have memory-only methods
				return ChoiceState{}, ChoiceRoundSegment{}, fmt.Errorf("validate choice exploration result %d prefix: %w", index, validateErr)
			}
			segmentResult.TraceSHA256 = record.SHA256FromSum(trace.SHA256)
			segmentResult.TraceSourceSHA256 = record.SHA256FromSum(trace.SourceTraceSHA256)
			segmentResult.TraceBytes = append([]byte(nil), trace.Bytes...)
			if !policyStopped {
				if err := choiceExpandCandidate(&next, candidate, trace, newChildren); err != nil {
					return ChoiceState{}, ChoiceRoundSegment{}, err
				}
			}
		}
		segmentResults[index] = segmentResult
	}
	children := make([]ChoiceCandidate, 0, len(newChildren))
	for _, candidate := range newChildren {
		children = append(children, candidate)
	}
	choiceSortCandidates(children)
	if err := choiceAdmitChildren(&next, children); err != nil {
		return ChoiceState{}, ChoiceRoundSegment{}, err
	}
	if next.StopReason == "" && len(next.Queue) == 0 {
		switch {
		case startUnreached:
			next.StopReason = ChoiceStopStartUnreached
		case next.OmittedByExecutionBound != 0 || next.LogicalExecutions >= next.Config.MaxExecutions:
			next.StopReason = ChoiceStopMaxExecutions
		case next.OmittedByDepth != 0:
			next.StopReason = ChoiceStopDepthComplete
		default:
			next.StopReason = ChoiceStopExhausted
		}
	}
	next.PendingBytes, err = choiceQueueBytes(next.Queue)
	if err != nil {
		return ChoiceState{}, ChoiceRoundSegment{}, err
	}
	after, err := choiceStateIdentity(next)
	if err != nil {
		return ChoiceState{}, ChoiceRoundSegment{}, err
	}
	segment := ChoiceRoundSegment{Schema: ChoiceRoundSegmentSchema, Index: round.Index, BeforeSHA256: before, AfterSHA256: after, Results: segmentResults}
	segment.SHA256, err = choiceSegmentIdentity(segment)
	if err != nil {
		return ChoiceState{}, ChoiceRoundSegment{}, err
	}
	return next, segment, nil
}

func ChoiceReplaySegment(state ChoiceState, segment ChoiceRoundSegment) (ChoiceState, error) {
	if segment.Schema != ChoiceRoundSegmentSchema {
		return ChoiceState{}, fmt.Errorf("choice exploration segment schema %q is incompatible with %q", segment.Schema, ChoiceRoundSegmentSchema)
	}
	identity, err := choiceSegmentIdentity(segment)
	if err != nil || identity != segment.SHA256 {
		return ChoiceState{}, errors.Join(errors.New("choice exploration segment identity does not match"), err)
	}
	before, err := choiceStateIdentity(state)
	if err != nil || before != segment.BeforeSHA256 || segment.Schema != ChoiceRoundSegmentSchema {
		return ChoiceState{}, errors.Join(errors.New("choice exploration segment does not link to its state"), err)
	}
	round, ok := state.NextRound()
	if !ok || round.Index != segment.Index || len(round.Candidates) != len(segment.Results) {
		return ChoiceState{}, errors.New("choice exploration segment round is unavailable")
	}
	results := make([]ChoiceResult, len(segment.Results))
	for index, stored := range segment.Results {
		result := ChoiceResult{CandidateSHA256: stored.CandidateSHA256, OutcomeSHA256: stored.OutcomeSHA256, Failed: stored.Failed, FailureSHA256: stored.FailureSHA256}
		if stored.Divergence != nil {
			divergence := stored.Divergence.Divergence()
			result.Divergence = &divergence
		}
		if len(stored.TraceBytes) != 0 {
			digest, decodeErr := stored.TraceSHA256.Bytes()
			if decodeErr != nil {
				return ChoiceState{}, decodeErr
			}
			tape := choice.ReplayPlan{Identity: state.Config.Execution, Bytes: append([]byte(nil), stored.TraceBytes...), SHA256: digest}
			validated, validateErr := choice.ValidateReplayPlan(tape, state.Config.Execution)
			if validateErr != nil || record.SHA256FromSum(validated.SourceTraceSHA256) != stored.TraceSourceSHA256 {
				return ChoiceState{}, errors.Join(errors.New("choice exploration segment trace is invalid"), validateErr)
			}
			result.Trace = &validated
		}
		results[index] = result
	}
	next, regenerated, err := CommitChoiceRound(state, round, results)
	if err != nil {
		return ChoiceState{}, err
	}
	left, err := json.Marshal(segment)
	if err != nil {
		return ChoiceState{}, err
	}
	right, err := json.Marshal(regenerated)
	if err != nil {
		return ChoiceState{}, err
	}
	after, err := choiceStateIdentity(next)
	if err != nil || !bytes.Equal(left, right) || after != segment.AfterSHA256 {
		return ChoiceState{}, errors.Join(errors.New("choice exploration segment replay changed its recorded result"), err)
	}
	return next, nil
}

func ValidateCandidateDivergence(candidate ChoiceCandidate, identity choice.ExecutionIdentity, divergence choice.Divergence) error {
	if candidate.ForcedDepth == 0 {
		return errors.New("choice divergence requires a forced-prefix candidate")
	}
	if divergence.Ordinal >= candidate.ForcedDepth || divergence.Expected == nil {
		return errors.New("choice divergence must reference its forced prefix")
	}
	switch divergence.Reason {
	case choice.DivergenceKind, choice.DivergenceSite, choice.DivergenceAlternatives, choice.DivergenceSelected, choice.DivergenceAlternativeSet, choice.DivergenceTapeUnconsumed, choice.DivergenceObservation:
	default:
		return errors.New("choice divergence is not a forced-prefix mismatch")
	}
	prefix, err := candidate.PrefixReplayPlan(identity)
	if err != nil {
		return err
	}
	return choice.ValidatePrefixReplayDivergence(prefix, divergence)
}

func choiceExpandCandidate(state *ChoiceState, parent ChoiceCandidate, trace choice.ReplayPlan, children map[record.SHA256]ChoiceCandidate) error {
	noOp := noOpSelectPolls(trace)
	for ordinal, decision := range trace.Decisions {
		if uint64(ordinal) < max(parent.ForcedDepth, state.Config.StartOrdinal) {
			continue
		}
		if decision.Alternatives <= 1 {
			continue
		}
		if noOp[ordinal] {
			state.OmittedBySelectReadiness += uint64(decision.Alternatives - 1)
			continue
		}
		depth := uint64(ordinal) - state.Config.StartOrdinal + 1
		if depth > state.Config.MaxChoiceDepth {
			state.OmittedByDepth += uint64(decision.Alternatives - 1)
			continue
		}
		for rank := uint32(0); rank < decision.Alternatives; rank++ {
			if rank == decision.Selected {
				continue
			}
			prefix, err := choice.BuildRankPrefix(trace, uint64(ordinal), rank)
			if err != nil {
				//gomad:dynamic reviewed 2026-10-12 reviewed codec and validator errors have memory-only methods
				return fmt.Errorf("build choice exploration child at decision %d rank %d: %w", ordinal, rank, err)
			}
			child, err := choiceNewCandidate(state.Config, &prefix, parent.SHA256, record.SHA256FromSum(trace.SourceTraceSHA256))
			if err != nil {
				return err
			}
			if choiceContainsIdentity(state.Seen, child.SHA256) {
				continue
			}
			if existing, found := children[child.SHA256]; found && !choiceSameCandidate(existing, child) {
				return errors.New("choice exploration candidate identity collision")
			}
			children[child.SHA256] = child
		}
	}
	return nil
}

func choiceAdmitChildren(state *ChoiceState, children []ChoiceCandidate) error {
	pendingBytes, err := choiceQueueBytes(state.Queue)
	if err != nil {
		return err
	}
	for index, candidate := range children {
		if uint64(len(state.Seen)) >= state.Config.MaxExecutions {
			state.OmittedByExecutionBound += uint64(len(children) - index)
			break
		}
		entryBytes, err := choiceCandidateBytes(candidate)
		if err != nil {
			return err
		}
		if entryBytes > ^uint64(0)-pendingBytes {
			return errors.New("choice exploration byte accounting overflow")
		}
		if pendingBytes+entryBytes > state.Config.MaxExplorationBytes {
			state.OmittedByCapacity += uint64(len(children) - index)
			if state.StopReason == "" {
				state.StopReason = ChoiceStopExplorationCapacity
			}
			break
		}
		pendingBytes += entryBytes
		state.Queue = append(state.Queue, candidate)
		state.Seen = choiceInsertIdentity(state.Seen, candidate.SHA256)
		state.DeepestPrefix = max(state.DeepestPrefix, candidate.ForcedDepth)
	}
	choiceSortCandidates(state.Queue)
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

func choiceNewCandidate(config ChoiceConfig, prefix *choice.ReplayPlan, parent, source record.SHA256) (ChoiceCandidate, error) {
	if !utf8.ValidString(config.Execution.ToolchainBuildKey) || !utf8.ValidString(config.Execution.GOOS) || !utf8.ValidString(config.Execution.GOARCH) {
		return ChoiceCandidate{}, errors.New("choice exploration execution identity is not valid UTF-8")
	}
	decisions := []choice.Decision{}
	candidate := ChoiceCandidate{ParentSHA256: parent, SourceTraceSHA256: source}
	if prefix != nil {
		validated, err := choice.ValidatePrefixReplayPlan(*prefix, config.Execution)
		if err != nil {
			return ChoiceCandidate{}, err
		}
		decisions = validated.Decisions
		if uint64(len(decisions)) <= config.StartOrdinal {
			return ChoiceCandidate{}, fmt.Errorf("choice exploration candidate of forced depth %d alters a decision before start ordinal %d", len(decisions), config.StartOrdinal)
		}
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
	encoded, err := json.Marshal(projection)
	if err != nil {
		return ChoiceCandidate{}, err
	}
	candidate.SHA256 = record.DomainHash(choiceCandidateDomain, encoded)
	return candidate, nil
}

func validateCandidateTrace(candidate ChoiceCandidate, trace choice.ReplayPlan, identity choice.ExecutionIdentity) error {
	if candidate.ForcedDepth == 0 {
		return nil
	}
	prefix, err := candidate.PrefixReplayPlan(identity)
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

func choiceStateIdentity(state ChoiceState) (record.SHA256, error) {
	encoded, err := json.Marshal(state)
	if err != nil {
		return "", err
	}
	return record.DomainHash(choiceStateDomain, encoded), nil
}

func ChoiceStateSHA256(state ChoiceState) (record.SHA256, error) {
	return choiceStateIdentity(state)
}

func choiceSegmentIdentity(segment ChoiceRoundSegment) (record.SHA256, error) {
	copy := segment
	copy.SHA256 = ""
	encoded, err := json.Marshal(copy)
	if err != nil {
		return "", err
	}
	return record.DomainHash(choiceRoundSegmentDomain, encoded), nil
}

func choiceCandidateBytes(candidate ChoiceCandidate) (uint64, error) {
	encoded, err := json.Marshal(candidate)
	if err != nil {
		return 0, err
	}
	return uint64(len(encoded)) + 1, nil
}

func choiceQueueBytes(queue []ChoiceCandidate) (uint64, error) {
	return SumBytes(queue, choiceCandidateBytes)
}

func choiceInsertIdentity(values []record.SHA256, value record.SHA256) []record.SHA256 {
	return InsertIdentity(values, value)
}

func choiceContainsIdentity(values []record.SHA256, value record.SHA256) bool {
	return ContainsIdentity(values, value)
}

func choiceSortCandidates(candidates []ChoiceCandidate) {
	slices.SortFunc(candidates, func(left, right ChoiceCandidate) int {
		if left.ForcedDepth < right.ForcedDepth {
			return -1
		}
		if left.ForcedDepth > right.ForcedDepth {
			return 1
		}
		return bytes.Compare([]byte(left.SHA256), []byte(right.SHA256))
	})
}

func choiceSameCandidates(left, right []ChoiceCandidate) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if !choiceSameCandidate(left[index], right[index]) {
			return false
		}
	}
	return true
}

func choiceSameCandidate(left, right ChoiceCandidate) bool {
	return left.SHA256 == right.SHA256 && left.ParentSHA256 == right.ParentSHA256 && left.SourceTraceSHA256 == right.SourceTraceSHA256 && left.ForcedDepth == right.ForcedDepth && left.PrefixSHA256 == right.PrefixSHA256 && bytes.Equal(left.PrefixBytes, right.PrefixBytes)
}

func choiceCloneCandidate(candidate ChoiceCandidate) ChoiceCandidate {
	candidate.PrefixBytes = append([]byte(nil), candidate.PrefixBytes...)
	return candidate
}

func choiceCloneState(state ChoiceState) ChoiceState {
	copy := state
	copy.Queue = make([]ChoiceCandidate, len(state.Queue))
	for index := range state.Queue {
		copy.Queue[index] = choiceCloneCandidate(state.Queue[index])
	}
	copy.Seen = append([]record.SHA256(nil), state.Seen...)
	copy.Outcomes = append([]record.SHA256(nil), state.Outcomes...)
	copy.FailureSignatures = append([]record.SHA256(nil), state.FailureSignatures...)
	return copy
}
