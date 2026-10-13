package campaign

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"

	"go.temporal.io/server/tools/gomad3/record"
	"go.temporal.io/server/tools/gomad3/runner/internal/exploration"
	"go.temporal.io/server/tools/gomad3/runner/internal/logstore"
)

// Choice and simulation exploration campaigns commit whole rounds through
// one owner on the campaign log, parameterized by the strategy's pure engine.
// An exploration record freezes the engine's configuration and the identity
// of its initial state. A round start records that the next round's
// candidates are about to run. A round record is the round's one outcome
// transaction: the round's candidate order, the engine's segment (its
// controller transition and replay evidence) and every candidate's execution
// outcome, which no further record repeats. Replay validates a whole round
// against the replayed engine state before it applies any of it. A started
// round that never committed is discarded by the next resume, which counts
// its candidates as recovery executions; its candidates then run again.
const (
	kindExploration logstore.Kind = 8
	kindRoundStart  logstore.Kind = 9
	kindRound       logstore.Kind = 10

	maximumExplorationRecordBytes = 1 << 20
	maximumRoundStartBytes        = 4 << 10
)

// explorationRecord freezes an exploration campaign's engine configuration.
// MaximumSegmentBytes bounds the encoded segment of one round, derived from
// the round's component envelopes.
type explorationRecord struct {
	Strategy            string              `json:"strategy"`
	Config              json.RawMessage     `json:"config"`
	InitialStateSHA256  record.SHA256       `json:"initial_state_sha256"`
	MaximumSegmentBytes record.Uint64String `json:"maximum_segment_bytes"`
}

type roundStartRecord struct {
	Round      record.Uint64String `json:"round"`
	Candidates record.Uint64String `json:"candidates"`
}

type roundRecord struct {
	Round      json.RawMessage   `json:"round"`
	Segment    json.RawMessage   `json:"segment"`
	Executions []ExecutionRecord `json:"executions"`
}

// RoundEngine is one exploration strategy's pure engine as the round owner
// drives it: C is its configuration, S its state, R a round and G a round's
// committed segment.
type RoundEngine[C, S, R, G any] struct {
	Strategy           string
	New                func(C) (S, error)
	Identity           func(S) (record.SHA256, error)
	NextRound          func(S) (R, bool)
	Replay             func(S, G) (S, error)
	LogicalExecutions  func(S) uint64
	RoundIndex         func(R) uint64
	Candidates         func(R) int
	SegmentSHA256      func(G) record.SHA256
	Implementation     func(C) record.SHA256
	BaseSeed           func(C) uint64
	MaxRoundBytes      func(C) uint64
	Summary            func(S) any
	ValidateExecutions func(round R, segment G, runs []ExecutionRecord, logicalStart, baseSeed uint64) error
}

// ChoiceRounds is the choice-exploration engine.
var ChoiceRounds = &RoundEngine[exploration.ChoiceConfig, exploration.ChoiceState, exploration.ChoiceRound, exploration.ChoiceRoundSegment]{
	Strategy: "choice-exploration", New: exploration.NewChoice, Identity: exploration.ChoiceStateSHA256,
	NextRound: func(state exploration.ChoiceState) (exploration.ChoiceRound, bool) { return state.NextRound() },
	Replay:    exploration.ChoiceReplaySegment, LogicalExecutions: func(state exploration.ChoiceState) uint64 { return state.LogicalExecutions },
	RoundIndex: func(round exploration.ChoiceRound) uint64 { return round.Index }, Candidates: func(round exploration.ChoiceRound) int { return len(round.Candidates) },
	SegmentSHA256:      func(segment exploration.ChoiceRoundSegment) record.SHA256 { return segment.SHA256 },
	Implementation:     func(config exploration.ChoiceConfig) record.SHA256 { return config.ControllerSHA256 },
	BaseSeed:           func(config exploration.ChoiceConfig) uint64 { return config.BaseSeed },
	MaxRoundBytes:      func(config exploration.ChoiceConfig) uint64 { return config.MaxExplorationBytes },
	Summary:            func(state exploration.ChoiceState) any { return state.Summary() },
	ValidateExecutions: validateChoiceRoundExecutions,
}

// SimulationRounds is the simulation-exploration engine.
var SimulationRounds = &RoundEngine[exploration.SimulationConfig, exploration.SimulationState, exploration.SimulationRound, exploration.SimulationRoundSegment]{
	Strategy: "simulation-exploration", New: exploration.NewSimulation, Identity: exploration.SimulationStateSHA256,
	NextRound: func(state exploration.SimulationState) (exploration.SimulationRound, bool) { return state.NextRound() },
	Replay:    exploration.SimulationReplaySegment, LogicalExecutions: func(state exploration.SimulationState) uint64 { return state.LogicalExecutions },
	RoundIndex: func(round exploration.SimulationRound) uint64 { return round.Index }, Candidates: func(round exploration.SimulationRound) int { return len(round.Candidates) },
	SegmentSHA256:      func(segment exploration.SimulationRoundSegment) record.SHA256 { return segment.SHA256 },
	Implementation:     func(config exploration.SimulationConfig) record.SHA256 { return config.ControllerSHA256 },
	BaseSeed:           func(config exploration.SimulationConfig) uint64 { return config.BaseSeed },
	MaxRoundBytes:      func(config exploration.SimulationConfig) uint64 { return config.MaxExplorationBytes },
	Summary:            func(state exploration.SimulationState) any { return state.Summary() },
	ValidateExecutions: validateSimulationRoundExecutions,
}

// CommittedRound describes one committed round of an exploration campaign.
// RoundBytes is the length of its encoded round payload, the measure the
// exploration byte bound applies to.
type CommittedRound struct {
	Index         uint64
	RoundBytes    uint64
	RoundSHA256   record.SHA256
	SegmentSHA256 record.SHA256
	FirstOrdinal  uint64
	Executions    uint64
}

// StagedRound is a started round without a committed outcome.
type StagedRound struct {
	Index      uint64
	Candidates uint64
}

// ExplorationInspection is the replayed exploration of a campaign log. State
// is the strategy engine's state: an exploration.ChoiceState or an
// exploration.SimulationState.
type ExplorationInspection struct {
	Strategy             string
	ImplementationSHA256 record.SHA256
	ChainSHA256          record.SHA256
	State                any
	Summary              any
	StagedRound          *StagedRound
	RecoveryExecutions   uint64
	Rounds               []CommittedRound
}

// roundReplay is the exploration part of campaignState.
type roundReplay interface {
	prepareExploration(explorationRecord) (func(), error)
	prepareStart(roundStartRecord) (func(), error)
	prepareRound(roundRecord) (func(), error)
	// abandon discards a started round when a campaign resumes.
	abandon()
	checkPublished(CampaignRecord) error
	inspection() ExplorationInspection
}

func newRoundReplay(strategy string) roundReplay {
	switch strategy {
	case ChoiceRounds.Strategy:
		return &roundState[exploration.ChoiceConfig, exploration.ChoiceState, exploration.ChoiceRound, exploration.ChoiceRoundSegment]{engine: ChoiceRounds}
	case SimulationRounds.Strategy:
		return &roundState[exploration.SimulationConfig, exploration.SimulationState, exploration.SimulationRound, exploration.SimulationRoundSegment]{engine: SimulationRounds}
	default:
		return nil
	}
}

type roundState[C, S, R, G any] struct {
	engine              *RoundEngine[C, S, R, G]
	opened              bool
	config              C
	configBytes         []byte
	maximumSegmentBytes uint64
	state               S
	chain               record.SHA256
	started             *roundStartRecord
	recovery            uint64
	rounds              []CommittedRound
}

func sameEncoding(left, right any) bool {
	leftBytes, leftErr := logstore.EncodeJSON(left)
	rightBytes, rightErr := logstore.EncodeJSON(right)
	return leftErr == nil && rightErr == nil && bytes.Equal(leftBytes, rightBytes)
}

func (rounds *roundState[C, S, R, G]) prepareExploration(open explorationRecord) (func(), error) {
	if rounds.opened {
		return nil, errors.New("exploration record repeats")
	}
	if open.Strategy != rounds.engine.Strategy || open.MaximumSegmentBytes == 0 {
		return nil, errors.New("exploration record does not match its campaign strategy")
	}
	var config C
	if err := logstore.DecodeJSON(open.Config, &config); err != nil {
		return nil, fmt.Errorf("decode exploration configuration: %w", err)
	}
	initial, err := rounds.engine.New(config)
	if err != nil {
		return nil, fmt.Errorf("validate exploration configuration: %w", err)
	}
	identity, err := rounds.engine.Identity(initial)
	if err != nil {
		return nil, err
	}
	if identity != open.InitialStateSHA256 {
		return nil, errors.New("exploration initial state identity changed")
	}
	configBytes, err := logstore.EncodeJSON(config)
	if err != nil {
		return nil, err
	}
	return func() {
		rounds.opened, rounds.config, rounds.configBytes = true, config, configBytes
		rounds.maximumSegmentBytes, rounds.state, rounds.chain = uint64(open.MaximumSegmentBytes), initial, identity
	}, nil
}

func (rounds *roundState[C, S, R, G]) prepareStart(start roundStartRecord) (func(), error) {
	if !rounds.opened {
		return nil, errors.New("exploration round starts before its exploration record")
	}
	if rounds.started != nil {
		return nil, fmt.Errorf("exploration round %d starts while round %d is uncommitted", start.Round, rounds.started.Round)
	}
	round, available := rounds.engine.NextRound(rounds.state)
	if !available || uint64(start.Round) != rounds.engine.RoundIndex(round) || uint64(start.Candidates) != uint64(rounds.engine.Candidates(round)) {
		return nil, errors.New("exploration round start does not match the next round")
	}
	return func() { rounds.started = &start }, nil
}

// prepareRound validates a whole round against the replayed engine state:
// the round is the engine's next round within the exploration byte bound,
// its segment replays within its segment bound, and every candidate's
// execution matches the segment's result.
func (rounds *roundState[C, S, R, G]) prepareRound(committed roundRecord) (func(), error) {
	engine := rounds.engine
	if !rounds.opened || rounds.started == nil {
		return nil, errors.New("exploration round commits without its round start")
	}
	var round R
	if err := logstore.DecodeJSON(committed.Round, &round); err != nil {
		return nil, fmt.Errorf("decode exploration round: %w", err)
	}
	expected, available := engine.NextRound(rounds.state)
	if !available || !sameEncoding(round, expected) || engine.RoundIndex(round) != uint64(rounds.started.Round) {
		return nil, fmt.Errorf("exploration round %d does not match the replayed state", rounds.started.Round)
	}
	if limit := engine.MaxRoundBytes(rounds.config); uint64(len(committed.Round)) > limit {
		return nil, fmt.Errorf("exploration round requires %d bytes, exceeding its %d-byte exploration bound", len(committed.Round), limit)
	}
	if uint64(len(committed.Segment)) > rounds.maximumSegmentBytes {
		return nil, fmt.Errorf("exploration segment requires %d bytes, exceeding its %d-byte capacity", len(committed.Segment), rounds.maximumSegmentBytes)
	}
	var segment G
	if err := logstore.DecodeJSON(committed.Segment, &segment); err != nil {
		return nil, fmt.Errorf("decode exploration segment: %w", err)
	}
	next, err := engine.Replay(rounds.state, segment)
	if err != nil {
		return nil, fmt.Errorf("replay exploration segment: %w", err)
	}
	logicalStart := engine.LogicalExecutions(rounds.state)
	if err := engine.ValidateExecutions(round, segment, committed.Executions, logicalStart, engine.BaseSeed(rounds.config)); err != nil {
		return nil, fmt.Errorf("validate exploration round %d executions: %w", engine.RoundIndex(round), err)
	}
	view := CommittedRound{
		Index: engine.RoundIndex(round), RoundBytes: uint64(len(committed.Round)), RoundSHA256: record.HashBytes(committed.Round),
		SegmentSHA256: engine.SegmentSHA256(segment), FirstOrdinal: logicalStart, Executions: uint64(len(committed.Executions)),
	}
	return func() {
		rounds.state, rounds.chain, rounds.started = next, engine.SegmentSHA256(segment), nil
		rounds.rounds = append(rounds.rounds, view)
	}, nil
}

func (rounds *roundState[C, S, R, G]) abandon() {
	if rounds.started != nil {
		rounds.recovery += uint64(rounds.started.Candidates)
		rounds.started = nil
	}
}

// checkPublished requires a published summary to report exactly the replayed
// exploration: its engine summary, implementation, chain and recovery count.
func (rounds *roundState[C, S, R, G]) checkPublished(published CampaignRecord) error {
	if !rounds.opened || rounds.started != nil {
		return errors.New("exploration campaign is published without its exploration or with an uncommitted round")
	}
	summary, implementation, chain := any(published.ChoiceExploration), published.ChoiceExplorationImplementationSHA256, published.ChoiceExplorationChainSHA256
	if rounds.engine.Strategy == SimulationRounds.Strategy {
		summary, implementation, chain = published.SimulationExploration, published.SimulationExplorationImplementationSHA256, published.SimulationExplorationChainSHA256
	}
	if !sameEncoding(summary, rounds.engine.Summary(rounds.state)) || implementation != rounds.engine.Implementation(rounds.config) || chain != rounds.chain {
		return errors.New("published exploration summary or chain does not match its campaign")
	}
	if uint64(published.RecoveryExecutions) != rounds.recovery {
		return errors.New("published recovery executions do not match the abandoned rounds")
	}
	return nil
}

func (rounds *roundState[C, S, R, G]) inspection() ExplorationInspection {
	inspected := ExplorationInspection{
		Strategy: rounds.engine.Strategy, ChainSHA256: rounds.chain, RecoveryExecutions: rounds.recovery,
		Rounds: append([]CommittedRound(nil), rounds.rounds...),
	}
	if rounds.opened {
		inspected.ImplementationSHA256 = rounds.engine.Implementation(rounds.config)
		inspected.State, inspected.Summary = rounds.state, rounds.engine.Summary(rounds.state)
	}
	if rounds.started != nil {
		inspected.StagedRound = &StagedRound{Index: uint64(rounds.started.Round), Candidates: uint64(rounds.started.Candidates)}
	}
	return inspected
}

func (state *campaignState) prepareExplorationRecord(kind logstore.Kind, payload []byte) (func(), error) {
	if state.rounds == nil {
		return nil, fmt.Errorf("%s campaign records no exploration round", state.open.Strategy)
	}
	if state.lifecycle != lifecycleRunning {
		return nil, fmt.Errorf("exploration record is committed in lifecycle %q", state.lifecycle)
	}
	switch kind {
	case kindExploration:
		var open explorationRecord
		if err := logstore.DecodeJSON(payload, &open); err != nil {
			return nil, err
		}
		return state.rounds.prepareExploration(open)
	case kindRoundStart:
		var start roundStartRecord
		if err := logstore.DecodeJSON(payload, &start); err != nil {
			return nil, err
		}
		return state.rounds.prepareStart(start)
	default:
		var committed roundRecord
		if err := logstore.DecodeJSON(payload, &committed); err != nil {
			return nil, err
		}
		sizes, err := state.checkExecutions(committed.Executions)
		if err != nil {
			return nil, err
		}
		apply, err := state.rounds.prepareRound(committed)
		if err != nil {
			return nil, err
		}
		return func() {
			apply()
			state.applyExecutions(committed.Executions, sizes)
		}, nil
	}
}

// Rounds is an exploration campaign's round owner, bound to one strategy's
// engine. It commits rounds to the campaign log and reads their replayed
// state.
type Rounds[C, S, R, G any] struct {
	journal *CampaignJournal
	engine  *RoundEngine[C, S, R, G]
}

// OpenRounds binds a running campaign's exploration to config. A campaign
// records its exploration once; a resumed campaign requires the recorded
// configuration and segment bound.
func OpenRounds[C, S, R, G any](journal *CampaignJournal, engine *RoundEngine[C, S, R, G], config C, maximumSegmentBytes uint64) (*Rounds[C, S, R, G], error) {
	if !journal.started {
		return nil, errors.New("campaign execution journal is not open")
	}
	rounds := &Rounds[C, S, R, G]{journal: journal, engine: engine}
	journal.mu.Lock()
	replayed, err := rounds.replayLocked()
	opened, configBytes, bound := false, []byte(nil), uint64(0)
	if err == nil {
		opened, configBytes, bound = replayed.opened, replayed.configBytes, replayed.maximumSegmentBytes
	}
	journal.mu.Unlock()
	if err != nil {
		return nil, err
	}
	encoded, err := logstore.EncodeJSON(config)
	if err != nil {
		return nil, err
	}
	if opened {
		if !bytes.Equal(encoded, configBytes) || bound != maximumSegmentBytes {
			return nil, errors.New("exploration plan identity or bounds changed")
		}
		return rounds, nil
	}
	initial, err := engine.New(config)
	if err != nil {
		return nil, fmt.Errorf("validate initial exploration state: %w", err)
	}
	identity, err := engine.Identity(initial)
	if err != nil {
		return nil, err
	}
	if err := journal.append(kindExploration, explorationRecord{
		Strategy: engine.Strategy, Config: encoded, InitialStateSHA256: identity, MaximumSegmentBytes: record.Uint64String(maximumSegmentBytes),
	}); err != nil {
		return nil, err
	}
	return rounds, nil
}

func (rounds *Rounds[C, S, R, G]) replayLocked() (*roundState[C, S, R, G], error) {
	replayed, ok := rounds.journal.state.rounds.(*roundState[C, S, R, G])
	if !ok || replayed.engine != rounds.engine {
		return nil, fmt.Errorf("%s campaign has no %s rounds", rounds.journal.state.open.Strategy, rounds.engine.Strategy)
	}
	return replayed, nil
}

func (rounds *Rounds[C, S, R, G]) replayed() *roundState[C, S, R, G] {
	rounds.journal.mu.Lock()
	defer rounds.journal.mu.Unlock()
	replayed, _ := rounds.replayLocked()
	return replayed
}

// State is the engine state after the last committed round.
func (rounds *Rounds[C, S, R, G]) State() S {
	return rounds.replayed().state
}

// ChainSHA256 is the segment chain identity after the last committed round.
func (rounds *Rounds[C, S, R, G]) ChainSHA256() record.SHA256 {
	return rounds.replayed().chain
}

// RecoveryExecutions counts the candidates of every round a resume
// discarded uncommitted.
func (rounds *Rounds[C, S, R, G]) RecoveryExecutions() uint64 {
	return rounds.replayed().recovery
}

// Start durably records that round's candidates are about to run.
func (rounds *Rounds[C, S, R, G]) Start(round R) error {
	return rounds.journal.append(kindRoundStart, roundStartRecord{
		Round: record.Uint64String(rounds.engine.RoundIndex(round)), Candidates: record.Uint64String(rounds.engine.Candidates(round)),
	})
}

// Commit commits a started round, its segment and every candidate's
// execution outcome as one transaction.
func (rounds *Rounds[C, S, R, G]) Commit(round R, segment G, runs []ExecutionRecord) error {
	encodedRound, err := logstore.EncodeJSON(round)
	if err != nil {
		return err
	}
	encodedSegment, err := logstore.EncodeJSON(segment)
	if err != nil {
		return err
	}
	return rounds.journal.append(kindRound, roundRecord{Round: encodedRound, Segment: encodedSegment, Executions: runs})
}

// InspectExploration replays a campaign log read-only and reports its
// exploration rounds. It reports false for a seed campaign.
func InspectExploration(path string) (ExplorationInspection, bool, error) {
	state, _, err := replayCampaign(path)
	if err != nil {
		return ExplorationInspection{}, false, classifyIntegrityError(err)
	}
	if state.rounds == nil {
		return ExplorationInspection{}, false, nil
	}
	return state.rounds.inspection(), true, nil
}

func validateChoiceRoundExecutions(round exploration.ChoiceRound, segment exploration.ChoiceRoundSegment, runs []ExecutionRecord, logicalStart, baseSeed uint64) error {
	if len(runs) != len(round.Candidates) || len(runs) != len(segment.Results) {
		return errors.New("exploration round execution records do not match its results")
	}
	candidates := make(map[record.SHA256]struct{}, len(runs))
	for index, run := range runs {
		candidate := round.Candidates[index]
		result := segment.Results[index]
		if run.SelectionOrdinal != record.Uint64String(logicalStart+uint64(index)) || run.Seed != record.Uint64String(baseSeed) || run.Round == nil || *run.Round != record.Uint64String(round.Index) || run.CandidateSHA256 != candidate.SHA256 || run.ParentCandidateSHA256 != candidate.ParentSHA256 || run.PrefixSHA256 != candidate.PrefixSHA256 || run.ForcedDepth == nil || *run.ForcedDepth != record.Uint64String(candidate.ForcedDepth) || run.OutcomeSHA256 != result.OutcomeSHA256 {
			return fmt.Errorf("exploration execution %d provenance does not match its segment", index)
		}
		if err := validateExplorationExecutionSummary(run, candidates); err != nil {
			return err
		}
		if !reflect.DeepEqual(run.Divergence, result.Divergence) {
			return fmt.Errorf("exploration execution %d divergence does not match its segment", index)
		}
		if result.Failed != (run.Domain == "target" || run.Domain == "watchdog") || result.Failed && (run.FailureSignature == nil || *run.FailureSignature != result.FailureSHA256) {
			return fmt.Errorf("exploration execution %d outcome does not match its segment", index)
		}
	}
	return nil
}

func validateSimulationRoundExecutions(round exploration.SimulationRound, segment exploration.SimulationRoundSegment, runs []ExecutionRecord, logicalStart, baseSeed uint64) error {
	if len(runs) != len(round.Candidates) || len(runs) != len(segment.Results) {
		return errors.New("simulation exploration round execution records do not match its results")
	}
	candidates := make(map[record.SHA256]struct{}, len(runs))
	for index, run := range runs {
		candidate := round.Candidates[index]
		result := segment.Results[index]
		if run.SelectionOrdinal != record.Uint64String(logicalStart+uint64(index)) || run.Seed != record.Uint64String(baseSeed) || run.Round == nil || *run.Round != record.Uint64String(round.Index) || run.CandidateSHA256 != candidate.SHA256 || run.ParentCandidateSHA256 != candidate.ParentSHA256 || run.ForcedDepth == nil || *run.ForcedDepth != record.Uint64String(len(candidate.Overrides)) || run.OutcomeSHA256 != result.OutcomeSHA256 {
			return fmt.Errorf("simulation exploration execution %d provenance does not match its segment", index)
		}
		if err := validateSimulationExplorationExecutionSummary(run, candidates); err != nil {
			return err
		}
		failed := run.Domain == "target" || run.Domain == "watchdog"
		if result.Failed != failed || result.Failed && (run.FailureSignature == nil || *run.FailureSignature != result.FailureSHA256) {
			return fmt.Errorf("simulation exploration execution %d outcome does not match its segment", index)
		}
	}
	return nil
}
