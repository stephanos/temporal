package campaign

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"go.temporal.io/server/tools/gomadv3/internal/canonicaljson"
	"go.temporal.io/server/tools/gomadv3/record"
	simulationengine "go.temporal.io/server/tools/gomadv3/runner/internal/exploration/simulation"
)

const (
	simulationExplorationPlanSchema       = "gomadv3.simulation-exploration-plan/v2"
	maximumSimulationExplorationPlanBytes = 1 << 20
)

type simulationExplorationPlan struct {
	Schema              string                  `json:"schema"`
	Config              simulationengine.Config `json:"config"`
	InitialStateSHA256  record.SHA256           `json:"initial_state_sha256"`
	MaximumSegmentBytes record.Uint64String     `json:"maximum_segment_bytes"`
}

type SimulationExplorationJournal struct {
	ctx                 context.Context
	batchPath           string
	state               simulationengine.State
	maximumSegmentBytes uint64
	runs                []ExecutionRecord
	chainSHA256         record.SHA256
}

type SimulationExplorationRoundJournal struct {
	journal *SimulationExplorationJournal
	path    string
	index   uint64
	runs    []ExecutionRecord
	runSet  []bool
}

type SimulationExplorationInspection struct {
	Summary              simulationengine.Summary
	ImplementationSHA256 record.SHA256
	ChainSHA256          record.SHA256
	Pending              []simulationengine.Candidate
	StagedRound          *SimulationExplorationStagedRound
}

type SimulationExplorationStagedRound struct {
	Index      uint64
	Candidates uint64
	Attempted  uint64
}

type reconstructedSimulationExploration struct {
	plan         simulationExplorationPlan
	state        simulationengine.State
	runs         []ExecutionRecord
	chainSHA256  record.SHA256
	completeRuns bool
}

func InspectSimulationExploration(batchPath string) (SimulationExplorationInspection, error) {
	batchPath, err := filepath.Abs(batchPath)
	if err != nil {
		return SimulationExplorationInspection{}, fmt.Errorf("resolve simulation exploration campaign path: %w", err)
	}
	if err := validatePrivateDirectory(batchPath, "simulation exploration campaign"); err != nil {
		return SimulationExplorationInspection{}, err
	}
	reconstructed, err := reconstructSimulationExploration(batchPath)
	if err != nil {
		return SimulationExplorationInspection{}, err
	}
	staged, err := inspectIncompleteSimulationExplorationRound(batchPath, reconstructed.state)
	if err != nil {
		return SimulationExplorationInspection{}, err
	}
	return SimulationExplorationInspection{
		Summary: reconstructed.state.Summary(), ImplementationSHA256: reconstructed.plan.Config.ControllerSHA256,
		ChainSHA256: reconstructed.chainSHA256, Pending: append([]simulationengine.Candidate(nil), reconstructed.state.Queue...), StagedRound: staged,
	}, nil
}

func reconstructSimulationExploration(batchPath string) (reconstructedSimulationExploration, error) {
	explorationPath := filepath.Join(batchPath, "simulation-exploration")
	planBytes, err := readExplorationFile(filepath.Join(explorationPath, "plan.json"), maximumSimulationExplorationPlanBytes)
	if err != nil {
		return reconstructedSimulationExploration{}, fmt.Errorf("read simulation exploration plan: %w", err)
	}
	var plan simulationExplorationPlan
	if err := canonicaljson.DecodeCanonicalJSON(planBytes, &plan); err != nil {
		return reconstructedSimulationExploration{}, fmt.Errorf("decode simulation exploration plan: %w", err)
	}
	if plan.Schema != simulationExplorationPlanSchema || plan.MaximumSegmentBytes == 0 {
		return reconstructedSimulationExploration{}, errors.New("simulation exploration plan identity or bounds are invalid")
	}
	state, err := simulationengine.New(plan.Config)
	if err != nil {
		return reconstructedSimulationExploration{}, err
	}
	initialIdentity, err := simulationengine.StateSHA256(state)
	if err != nil || initialIdentity != plan.InitialStateSHA256 {
		return reconstructedSimulationExploration{}, errors.Join(errors.New("simulation exploration initial state identity changed"), err)
	}
	roundsPath := filepath.Join(explorationPath, "rounds")
	if err := validatePrivateDirectory(roundsPath, "simulation exploration rounds"); err != nil {
		return reconstructedSimulationExploration{}, err
	}
	entries, err := os.ReadDir(roundsPath)
	if err != nil {
		return reconstructedSimulationExploration{}, fmt.Errorf("read simulation exploration rounds: %w", err)
	}
	if uint64(len(entries)) > plan.Config.MaxExecutions {
		return reconstructedSimulationExploration{}, errors.New("simulation exploration round count exceeds its execution bound")
	}
	chainSHA256 := initialIdentity
	journalRuns := []ExecutionRecord{}
	completeRuns := true
	for index, entry := range entries {
		name := fmt.Sprintf("%020d", index)
		if entry.Name() != name {
			return reconstructedSimulationExploration{}, fmt.Errorf("simulation exploration round sequence has a gap at %s", name)
		}
		roundPath := filepath.Join(roundsPath, name)
		if err := validateSimulationExplorationRoundDirectory(roundPath, true); err != nil {
			return reconstructedSimulationExploration{}, err
		}
		roundBytes, err := readExplorationFile(filepath.Join(roundPath, "round.json"), plan.Config.MaxExplorationBytes)
		if err != nil {
			return reconstructedSimulationExploration{}, fmt.Errorf("read simulation exploration round %d: %w", index, err)
		}
		var storedRound simulationengine.Round
		if err := canonicaljson.DecodeCanonicalJSON(roundBytes, &storedRound); err != nil {
			return reconstructedSimulationExploration{}, fmt.Errorf("decode simulation exploration round %d: %w", index, err)
		}
		expectedRound, ok := state.NextRound()
		equal, equalErr := canonicalEqual(storedRound, expectedRound)
		if equalErr != nil || !ok || !equal {
			return reconstructedSimulationExploration{}, errors.Join(fmt.Errorf("simulation exploration round %d does not match reconstructed state", index), equalErr)
		}
		segmentBytes, err := readExplorationFile(filepath.Join(roundPath, "segment.json"), uint64(plan.MaximumSegmentBytes))
		if err != nil {
			return reconstructedSimulationExploration{}, fmt.Errorf("read simulation exploration segment %d: %w", index, err)
		}
		var segment simulationengine.RoundSegment
		if err := canonicaljson.DecodeCanonicalJSON(segmentBytes, &segment); err != nil {
			return reconstructedSimulationExploration{}, fmt.Errorf("decode simulation exploration segment %d: %w", index, err)
		}
		logicalStart := state.LogicalExecutions
		state, err = simulationengine.ReplaySegment(state, segment)
		if err != nil {
			return reconstructedSimulationExploration{}, fmt.Errorf("replay simulation exploration segment %d: %w", index, err)
		}
		runs, err := readExplorationRoundExecutions(roundPath, len(storedRound.Candidates), uint64(plan.MaximumSegmentBytes))
		if err != nil {
			return reconstructedSimulationExploration{}, fmt.Errorf("read simulation exploration execution records %d: %w", index, err)
		}
		if len(runs) == 0 {
			completeRuns = false
		} else {
			if err := validateSimulationExplorationRoundExecutions(storedRound, segment, runs, logicalStart, plan.Config.BaseSeed); err != nil {
				return reconstructedSimulationExploration{}, fmt.Errorf("validate simulation exploration execution records %d: %w", index, err)
			}
			journalRuns = append(journalRuns, runs...)
		}
		chainSHA256 = segment.SHA256
	}
	return reconstructedSimulationExploration{plan: plan, state: state, runs: journalRuns, chainSHA256: chainSHA256, completeRuns: completeRuns}, nil
}

func NewSimulationExplorationJournal(ctx context.Context, batchPath string, initial simulationengine.State, maximumSegmentBytes uint64) (_ *SimulationExplorationJournal, retErr error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if maximumSegmentBytes == 0 {
		return nil, errors.New("simulation exploration segment capacity must be positive")
	}
	batchPath, err := filepath.Abs(batchPath)
	if err != nil {
		return nil, fmt.Errorf("resolve simulation exploration campaign path: %w", err)
	}
	if err := validatePrivateDirectory(batchPath, "simulation exploration campaign"); err != nil {
		return nil, err
	}
	expected, err := simulationengine.New(initial.Config)
	if err != nil {
		return nil, fmt.Errorf("validate initial simulation exploration state: %w", err)
	}
	providedIdentity, err := simulationengine.StateSHA256(initial)
	if err != nil {
		return nil, err
	}
	expectedIdentity, err := simulationengine.StateSHA256(expected)
	if err != nil {
		return nil, err
	}
	if providedIdentity != expectedIdentity {
		return nil, errors.New("simulation exploration journal requires the canonical initial state")
	}
	explorationPath := filepath.Join(batchPath, "simulation-exploration")
	partialPath := filepath.Join(batchPath, ".partial", "simulation-exploration")
	for _, path := range []string{explorationPath, partialPath} {
		if _, err := os.Lstat(path); err == nil {
			return nil, errors.New("simulation exploration journal already exists")
		} else if !os.IsNotExist(err) {
			return nil, err
		}
	}
	defer func() {
		if retErr == nil {
			return
		}
		retErr = errors.Join(retErr, removeCompletedPartialContext(ctx, partialPath, "simulation-exploration-journal"))
		retErr = errors.Join(retErr, removeCompletedPartialContext(ctx, explorationPath, "simulation-exploration-journal"))
	}()
	if err := makePrivateDirectoriesContext(ctx, filepath.Join(explorationPath, "rounds")); err != nil {
		return nil, err
	}
	if err := makePrivateDirectoriesContext(ctx, partialPath); err != nil {
		return nil, err
	}
	plan := simulationExplorationPlan{
		Schema: simulationExplorationPlanSchema, Config: initial.Config, InitialStateSHA256: expectedIdentity,
		MaximumSegmentBytes: record.Uint64String(maximumSegmentBytes),
	}
	encoded, err := canonicaljson.CanonicalJSON(plan)
	if err != nil {
		return nil, err
	}
	if err := atomicWriteContext(ctx, filepath.Join(explorationPath, "plan.json"), encoded); err != nil {
		return nil, fmt.Errorf("write simulation exploration plan: %w", err)
	}
	if err := syncDirectoryContext(ctx, explorationPath); err != nil {
		return nil, fmt.Errorf("sync simulation exploration plan directory: %w", err)
	}
	return &SimulationExplorationJournal{
		ctx: ctx, batchPath: batchPath, state: initial, maximumSegmentBytes: maximumSegmentBytes, chainSHA256: expectedIdentity,
	}, nil
}

func ResumeSimulationExplorationJournal(ctx context.Context, batchPath string, config simulationengine.Config, maximumSegmentBytes uint64) (*SimulationExplorationJournal, simulationengine.State, uint64, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	batchPath, err := filepath.Abs(batchPath)
	if err != nil {
		return nil, simulationengine.State{}, 0, fmt.Errorf("resolve simulation exploration campaign path: %w", err)
	}
	if err := validatePrivateDirectory(batchPath, "simulation exploration campaign"); err != nil {
		return nil, simulationengine.State{}, 0, err
	}
	explorationPath := filepath.Join(batchPath, "simulation-exploration")
	planBytes, err := readExplorationFile(filepath.Join(explorationPath, "plan.json"), maximumSimulationExplorationPlanBytes)
	if err != nil {
		return nil, simulationengine.State{}, 0, fmt.Errorf("read simulation exploration plan: %w", err)
	}
	var plan simulationExplorationPlan
	if err := canonicaljson.DecodeCanonicalJSON(planBytes, &plan); err != nil {
		return nil, simulationengine.State{}, 0, fmt.Errorf("decode simulation exploration plan: %w", err)
	}
	if plan.Schema != simulationExplorationPlanSchema || plan.Config != config || uint64(plan.MaximumSegmentBytes) != maximumSegmentBytes || maximumSegmentBytes == 0 {
		return nil, simulationengine.State{}, 0, errors.New("simulation exploration plan identity or bounds changed")
	}
	state, err := simulationengine.New(config)
	if err != nil {
		return nil, simulationengine.State{}, 0, err
	}
	initialIdentity, err := simulationengine.StateSHA256(state)
	if err != nil || initialIdentity != plan.InitialStateSHA256 {
		return nil, simulationengine.State{}, 0, errors.Join(errors.New("simulation exploration initial state identity changed"), err)
	}
	roundsPath := filepath.Join(explorationPath, "rounds")
	if err := validatePrivateDirectory(roundsPath, "simulation exploration rounds"); err != nil {
		return nil, simulationengine.State{}, 0, err
	}
	entries, err := os.ReadDir(roundsPath)
	if err != nil {
		return nil, simulationengine.State{}, 0, fmt.Errorf("read simulation exploration rounds: %w", err)
	}
	if uint64(len(entries)) > config.MaxExecutions {
		return nil, simulationengine.State{}, 0, errors.New("simulation exploration round count exceeds its execution bound")
	}
	chainSHA256 := initialIdentity
	journalRuns := []ExecutionRecord{}
	for index, entry := range entries {
		name := fmt.Sprintf("%020d", index)
		if entry.Name() != name {
			return nil, simulationengine.State{}, 0, fmt.Errorf("simulation exploration round sequence has a gap at %s", name)
		}
		roundPath := filepath.Join(roundsPath, name)
		if err := validateSimulationExplorationRoundDirectory(roundPath, true); err != nil {
			return nil, simulationengine.State{}, 0, err
		}
		roundBytes, err := readExplorationFile(filepath.Join(roundPath, "round.json"), config.MaxExplorationBytes)
		if err != nil {
			return nil, simulationengine.State{}, 0, fmt.Errorf("read simulation exploration round %d: %w", index, err)
		}
		var storedRound simulationengine.Round
		if err := canonicaljson.DecodeCanonicalJSON(roundBytes, &storedRound); err != nil {
			return nil, simulationengine.State{}, 0, fmt.Errorf("decode simulation exploration round %d: %w", index, err)
		}
		expectedRound, ok := state.NextRound()
		equal, equalErr := canonicalEqual(storedRound, expectedRound)
		if equalErr != nil || !ok || !equal {
			return nil, simulationengine.State{}, 0, errors.Join(fmt.Errorf("simulation exploration round %d does not match reconstructed state", index), equalErr)
		}
		segmentBytes, err := readExplorationFile(filepath.Join(roundPath, "segment.json"), maximumSegmentBytes)
		if err != nil {
			return nil, simulationengine.State{}, 0, fmt.Errorf("read simulation exploration segment %d: %w", index, err)
		}
		var segment simulationengine.RoundSegment
		if err := canonicaljson.DecodeCanonicalJSON(segmentBytes, &segment); err != nil {
			return nil, simulationengine.State{}, 0, fmt.Errorf("decode simulation exploration segment %d: %w", index, err)
		}
		logicalStart := state.LogicalExecutions
		state, err = simulationengine.ReplaySegment(state, segment)
		if err != nil {
			return nil, simulationengine.State{}, 0, fmt.Errorf("replay simulation exploration segment %d: %w", index, err)
		}
		runs, err := readExplorationRoundExecutions(roundPath, len(storedRound.Candidates), maximumSegmentBytes)
		if err != nil {
			return nil, simulationengine.State{}, 0, fmt.Errorf("read simulation exploration execution records %d: %w", index, err)
		}
		if len(runs) != 0 {
			if err := validateSimulationExplorationRoundExecutions(storedRound, segment, runs, logicalStart, config.BaseSeed); err != nil {
				return nil, simulationengine.State{}, 0, fmt.Errorf("validate simulation exploration execution records %d: %w", index, err)
			}
			journalRuns = append(journalRuns, runs...)
		}
		chainSHA256 = segment.SHA256
	}
	recoveryExecutions, err := discardIncompleteSimulationExplorationRound(ctx, batchPath, state)
	if err != nil {
		return nil, simulationengine.State{}, 0, err
	}
	journal := &SimulationExplorationJournal{
		ctx: ctx, batchPath: batchPath, state: state, maximumSegmentBytes: maximumSegmentBytes, runs: journalRuns, chainSHA256: chainSHA256,
	}
	return journal, state, recoveryExecutions, nil
}

func (journal *SimulationExplorationJournal) State() simulationengine.State {
	if journal == nil {
		return simulationengine.State{}
	}
	return journal.state
}

func (journal *SimulationExplorationJournal) ChainSHA256() record.SHA256 {
	if journal == nil {
		return ""
	}
	return journal.chainSHA256
}

func ValidatePublishedSimulationExploration(batchPath string, expectedSummary simulationengine.Summary, expectedImplementation, expectedChain record.SHA256, expectedRuns []ExecutionRecord) error {
	planBytes, err := readExplorationFile(filepath.Join(batchPath, "simulation-exploration", "plan.json"), maximumSimulationExplorationPlanBytes)
	if err != nil {
		return fmt.Errorf("read published simulation exploration plan: %w", err)
	}
	var plan simulationExplorationPlan
	if err := canonicaljson.DecodeCanonicalJSON(planBytes, &plan); err != nil {
		return fmt.Errorf("decode published simulation exploration plan: %w", err)
	}
	if plan.Schema != simulationExplorationPlanSchema || plan.Config.ControllerSHA256 != expectedImplementation || expectedImplementation != simulationengine.ImplementationSHA256() || plan.MaximumSegmentBytes == 0 {
		return errors.New("published simulation exploration plan identity is invalid")
	}
	state, err := simulationengine.New(plan.Config)
	if err != nil {
		return err
	}
	initialIdentity, err := simulationengine.StateSHA256(state)
	if err != nil || initialIdentity != plan.InitialStateSHA256 {
		return errors.Join(errors.New("published simulation exploration initial state identity changed"), err)
	}
	chain := initialIdentity
	roundsPath := filepath.Join(batchPath, "simulation-exploration", "rounds")
	entries, err := os.ReadDir(roundsPath)
	if err != nil {
		return err
	}
	committedRuns := []ExecutionRecord{}
	for index, entry := range entries {
		name := fmt.Sprintf("%020d", index)
		if entry.Name() != name {
			return fmt.Errorf("published simulation exploration round sequence has a gap at %s", name)
		}
		roundPath := filepath.Join(roundsPath, name)
		if err := validateSimulationExplorationRoundDirectory(roundPath, true); err != nil {
			return err
		}
		roundBytes, err := readExplorationFile(filepath.Join(roundPath, "round.json"), plan.Config.MaxExplorationBytes)
		if err != nil {
			return err
		}
		var round simulationengine.Round
		if err := canonicaljson.DecodeCanonicalJSON(roundBytes, &round); err != nil {
			return err
		}
		expectedRound, ok := state.NextRound()
		equal, equalErr := canonicalEqual(round, expectedRound)
		if equalErr != nil || !ok || !equal {
			return errors.Join(fmt.Errorf("published simulation exploration round %d does not match its state", index), equalErr)
		}
		segmentBytes, err := readExplorationFile(filepath.Join(roundPath, "segment.json"), uint64(plan.MaximumSegmentBytes))
		if err != nil {
			return err
		}
		var segment simulationengine.RoundSegment
		if err := canonicaljson.DecodeCanonicalJSON(segmentBytes, &segment); err != nil {
			return err
		}
		logicalStart := state.LogicalExecutions
		state, err = simulationengine.ReplaySegment(state, segment)
		if err != nil {
			return err
		}
		runs, err := readExplorationRoundExecutions(roundPath, len(round.Candidates), uint64(plan.MaximumSegmentBytes))
		if err != nil || len(runs) == 0 {
			return errors.Join(errors.New("published simulation exploration execution records are unavailable"), err)
		}
		if err := validateSimulationExplorationRoundExecutions(round, segment, runs, logicalStart, plan.Config.BaseSeed); err != nil {
			return err
		}
		committedRuns = append(committedRuns, runs...)
		chain = segment.SHA256
	}
	equal, err := canonicalEqual(state.Summary(), expectedSummary)
	if err != nil || !equal || chain != expectedChain {
		return errors.Join(errors.New("published simulation exploration summary or chain does not match its campaign"), err)
	}
	if len(committedRuns) != len(expectedRuns) {
		return errors.New("published simulation exploration execution projection count does not match its campaign")
	}
	for index := range committedRuns {
		equal, err := canonicalEqual(committedRuns[index], expectedRuns[index])
		if err != nil || !equal {
			return errors.Join(fmt.Errorf("published simulation exploration execution projection diverges at ordinal %d", index), err)
		}
	}
	partialEntries, err := os.ReadDir(filepath.Join(batchPath, ".partial", "simulation-exploration"))
	if err != nil || len(partialEntries) != 0 {
		return errors.Join(errors.New("published simulation exploration retains incomplete round state"), err)
	}
	return nil
}

func (journal *SimulationExplorationJournal) StageRound(round simulationengine.Round) (_ *SimulationExplorationRoundJournal, retErr error) {
	if journal == nil {
		return nil, errors.New("simulation exploration journal is required")
	}
	expected, ok := journal.state.NextRound()
	equal, err := canonicalEqual(round, expected)
	if err != nil {
		return nil, err
	}
	if !ok || !equal {
		return nil, errors.New("simulation exploration staged round does not match current state")
	}
	path := filepath.Join(journal.batchPath, ".partial", "simulation-exploration", fmt.Sprintf("%020d", round.Index))
	if _, err := os.Lstat(path); err == nil {
		return nil, errors.New("simulation exploration round is already staged")
	} else if !os.IsNotExist(err) {
		return nil, err
	}
	if err := observeMutation(journal.ctx, mutationCreate, "simulation-exploration-round-directory"); err != nil {
		return nil, err
	}
	if err := os.Mkdir(path, 0o700); err != nil {
		return nil, err
	}
	owned := true
	defer func() {
		if retErr == nil || !owned {
			return
		}
		retErr = errors.Join(retErr, removeCompletedPartialContext(journal.ctx, path, "simulation-exploration-round"))
		retErr = errors.Join(retErr, syncDirectoryContext(journal.ctx, filepath.Dir(path)))
	}()
	if err := os.Chmod(path, 0o700); err != nil {
		return nil, err
	}
	if err := makePrivateDirectoriesContext(journal.ctx, filepath.Join(path, "candidates")); err != nil {
		return nil, err
	}
	if err := makePrivateDirectoriesContext(journal.ctx, filepath.Join(path, "attempts")); err != nil {
		return nil, err
	}
	encoded, err := canonicaljson.CanonicalJSON(round)
	if err != nil {
		return nil, err
	}
	if uint64(len(encoded)) > journal.state.Config.MaxExplorationBytes {
		return nil, errors.New("simulation exploration round exceeds its exploration byte capacity")
	}
	if err := atomicWriteContext(journal.ctx, filepath.Join(path, "round.json"), encoded); err != nil {
		return nil, fmt.Errorf("write staged simulation exploration round: %w", err)
	}
	if err := syncDirectoryContext(journal.ctx, path); err != nil {
		return nil, fmt.Errorf("sync staged simulation exploration round: %w", err)
	}
	owned = false
	return &SimulationExplorationRoundJournal{
		journal: journal, path: path, index: round.Index,
		runs: make([]ExecutionRecord, len(round.Candidates)), runSet: make([]bool, len(round.Candidates)),
	}, nil
}

func (staged *SimulationExplorationRoundJournal) RecordExecution(index int, run ExecutionRecord) error {
	if staged == nil || staged.journal == nil || index < 0 || index >= len(staged.runs) {
		return errors.New("simulation exploration execution record index is invalid")
	}
	if staged.runSet[index] {
		return errors.New("simulation exploration execution record is already staged")
	}
	path := filepath.Join(staged.path, "executions")
	if err := makePrivateDirectoriesContext(staged.journal.ctx, path); err != nil {
		return err
	}
	encoded, err := canonicaljson.CanonicalJSON(run)
	if err != nil {
		return err
	}
	if uint64(len(encoded)) > staged.journal.maximumSegmentBytes {
		return errors.New("simulation exploration execution record exceeds its byte capacity")
	}
	if err := atomicWriteContext(staged.journal.ctx, filepath.Join(path, fmt.Sprintf("%020d.json", index)), encoded); err != nil {
		return fmt.Errorf("write simulation exploration execution record: %w", err)
	}
	staged.runs[index] = run
	staged.runSet[index] = true
	return nil
}

func (journal *SimulationExplorationJournal) CommitRound(staged *SimulationExplorationRoundJournal, segment simulationengine.RoundSegment) error {
	if journal == nil || staged == nil || staged.journal != journal || staged.index != segment.Index {
		return errors.New("simulation exploration staged round does not match its segment")
	}
	next, err := simulationengine.ReplaySegment(journal.state, segment)
	if err != nil {
		return fmt.Errorf("validate simulation exploration segment before commit: %w", err)
	}
	encoded, err := canonicaljson.CanonicalJSON(segment)
	if err != nil {
		return err
	}
	if uint64(len(encoded)) > journal.maximumSegmentBytes {
		return fmt.Errorf("simulation exploration segment requires %d bytes, exceeding its %d-byte capacity", len(encoded), journal.maximumSegmentBytes)
	}
	if err := atomicWriteContext(journal.ctx, filepath.Join(staged.path, "segment.json"), encoded); err != nil {
		return fmt.Errorf("write staged simulation exploration segment: %w", err)
	}
	if slices.Contains(staged.runSet, true) && slices.Contains(staged.runSet, false) {
		return errors.New("simulation exploration round execution records are incomplete")
	}
	if err := removeCompletedPartialContext(journal.ctx, filepath.Join(staged.path, "candidates"), "simulation-exploration-candidate-work"); err != nil {
		return fmt.Errorf("remove completed simulation exploration candidate work: %w", err)
	}
	if err := removeCompletedPartialContext(journal.ctx, filepath.Join(staged.path, "attempts"), "simulation-exploration-attempts"); err != nil {
		return fmt.Errorf("remove completed simulation exploration attempt records: %w", err)
	}
	if err := syncDirectoryContext(journal.ctx, staged.path); err != nil {
		return fmt.Errorf("sync staged simulation exploration segment: %w", err)
	}
	finalPath := filepath.Join(journal.batchPath, "simulation-exploration", "rounds", fmt.Sprintf("%020d", staged.index))
	if _, err := os.Lstat(finalPath); err == nil {
		return errors.New("simulation exploration segment is already committed")
	} else if !os.IsNotExist(err) {
		return err
	}
	if err := renameContext(journal.ctx, staged.path, finalPath, "simulation-exploration-publish"); err != nil {
		return fmt.Errorf("publish simulation exploration segment: %w", err)
	}
	if err := syncDirectoryContext(journal.ctx, filepath.Dir(finalPath)); err != nil {
		return fmt.Errorf("sync simulation exploration segment sequence: %w", err)
	}
	journal.state = next
	journal.chainSHA256 = segment.SHA256
	if slices.Contains(staged.runSet, true) {
		journal.runs = append(journal.runs, staged.runs...)
	}
	return nil
}

func (journal *SimulationExplorationJournal) CommittedExecutions() []ExecutionRecord {
	if journal == nil {
		return nil
	}
	return append([]ExecutionRecord(nil), journal.runs...)
}

func (staged *SimulationExplorationRoundJournal) Path() string {
	if staged == nil {
		return ""
	}
	return staged.path
}

func (staged *SimulationExplorationRoundJournal) BeginExecution(ordinal, seed uint64) (*ExecutionJournal, error) {
	if staged == nil || staged.journal == nil {
		return nil, errors.New("simulation exploration staged round is required")
	}
	attempt, err := canonicaljson.CanonicalJSON(explorationAttempt{Ordinal: record.Uint64String(ordinal), Seed: record.Uint64String(seed)})
	if err != nil {
		return nil, err
	}
	if err := atomicWriteContext(staged.journal.ctx, filepath.Join(staged.path, "attempts", fmt.Sprintf("%020d.json", ordinal)), attempt); err != nil {
		return nil, fmt.Errorf("record simulation exploration candidate attempt: %w", err)
	}
	path := filepath.Join(staged.path, "candidates", fmt.Sprintf("%020d", ordinal))
	if err := makePrivateDirectoriesContext(staged.journal.ctx, path); err != nil {
		return nil, err
	}
	run := &ExecutionJournal{ctx: staged.journal.ctx, path: path, ordinal: ordinal, seed: seed, state: ExecutionStaging}
	if err := run.writeState(ExecutionStaging); err != nil {
		return run, err
	}
	if err := makePrivateDirectoriesContext(staged.journal.ctx, run.WorkPath()); err != nil {
		return run, err
	}
	return run, nil
}

func validateSimulationExplorationRoundDirectory(path string, complete bool) error {
	if err := validatePrivateDirectory(path, "simulation exploration round"); err != nil {
		return err
	}
	entries, err := os.ReadDir(path)
	if err != nil {
		return err
	}
	seenRound, seenSegment := false, false
	for _, entry := range entries {
		switch entry.Name() {
		case "round.json":
			seenRound = !entry.IsDir()
		case "segment.json":
			seenSegment = !entry.IsDir()
		case "executions", "failures", "successes":
			if !entry.IsDir() {
				return errors.New("simulation exploration round artifact entry is not a directory")
			}
		case "candidates", "attempts":
			if complete || !entry.IsDir() {
				return errors.New("simulation exploration round work entry is invalid")
			}
		default:
			return errors.New("simulation exploration round contains an unexpected entry")
		}
	}
	if !seenRound || complete != seenSegment {
		return errors.New("simulation exploration round directory contents are invalid")
	}
	return nil
}

func validateSimulationExplorationRoundExecutions(round simulationengine.Round, segment simulationengine.RoundSegment, runs []ExecutionRecord, logicalStart, baseSeed uint64) error {
	if len(runs) != len(round.Candidates) || len(runs) != len(segment.Results) {
		return errors.New("simulation exploration round execution records do not match its results")
	}
	for index, run := range runs {
		candidate := round.Candidates[index]
		result := segment.Results[index]
		if run.Strategy != "simulation-exploration" || run.SelectionOrdinal != record.Uint64String(logicalStart+uint64(index)) || run.Seed != record.Uint64String(baseSeed) || run.Round == nil || *run.Round != record.Uint64String(round.Index) || run.CandidateSHA256 != candidate.SHA256 || run.ParentCandidateSHA256 != candidate.ParentSHA256 || run.PrefixSHA256 != "" || run.ForcedDepth == nil || *run.ForcedDepth != record.Uint64String(len(candidate.Overrides)) || run.OutcomeSHA256 != result.OutcomeSHA256 {
			return fmt.Errorf("simulation exploration execution %d provenance does not match its segment", index)
		}
		failed := run.Domain == "target" || run.Domain == "watchdog"
		if result.Failed != failed || result.Failed && (run.FailureSignature == nil || *run.FailureSignature != result.FailureSHA256) {
			return fmt.Errorf("simulation exploration execution %d outcome does not match its segment", index)
		}
	}
	return nil
}

func inspectIncompleteSimulationExplorationRound(batchPath string, state simulationengine.State) (*SimulationExplorationStagedRound, error) {
	partialPath := filepath.Join(batchPath, ".partial", "simulation-exploration")
	if err := validatePrivateDirectory(partialPath, "partial simulation exploration"); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(partialPath)
	if err != nil {
		return nil, err
	}
	if len(entries) == 0 {
		return nil, nil
	}
	if len(entries) != 1 {
		return nil, errors.New("partial simulation exploration contains multiple rounds")
	}
	expected, ok := state.NextRound()
	name := fmt.Sprintf("%020d", expected.Index)
	if !ok || entries[0].Name() != name {
		return nil, errors.New("partial simulation exploration round does not match reconstructed state")
	}
	path := filepath.Join(partialPath, name)
	if err := validatePrivateDirectory(path, "partial simulation exploration round"); err != nil {
		return nil, err
	}
	roundBytes, err := readExplorationFile(filepath.Join(path, "round.json"), state.Config.MaxExplorationBytes)
	if err != nil {
		return nil, err
	}
	var round simulationengine.Round
	if err := canonicaljson.DecodeCanonicalJSON(roundBytes, &round); err != nil {
		return nil, err
	}
	equal, err := canonicalEqual(round, expected)
	if err != nil || !equal {
		return nil, errors.Join(errors.New("partial simulation exploration round changed"), err)
	}
	attempts, err := countSimulationExplorationAttempts(path, state, round)
	if err != nil {
		return nil, err
	}
	return &SimulationExplorationStagedRound{Index: round.Index, Candidates: uint64(len(round.Candidates)), Attempted: attempts}, nil
}

func discardIncompleteSimulationExplorationRound(ctx context.Context, batchPath string, state simulationengine.State) (uint64, error) {
	staged, err := inspectIncompleteSimulationExplorationRound(batchPath, state)
	if err != nil || staged == nil {
		return 0, err
	}
	partialPath := filepath.Join(batchPath, ".partial", "simulation-exploration")
	path := filepath.Join(partialPath, fmt.Sprintf("%020d", staged.Index))
	if err := removeCompletedPartialContext(ctx, path, "simulation-exploration-incomplete-round"); err != nil {
		return 0, err
	}
	if err := syncDirectoryContext(ctx, partialPath); err != nil {
		return 0, err
	}
	return staged.Attempted, nil
}

func countSimulationExplorationAttempts(roundPath string, state simulationengine.State, round simulationengine.Round) (uint64, error) {
	attemptsPath := filepath.Join(roundPath, "attempts")
	if err := validatePrivateDirectory(attemptsPath, "partial simulation exploration attempts"); err != nil {
		return 0, err
	}
	entries, err := os.ReadDir(attemptsPath)
	if err != nil {
		return 0, err
	}
	if len(entries) > len(round.Candidates) {
		return 0, errors.New("partial simulation exploration contains too many candidate attempts")
	}
	for index, entry := range entries {
		ordinal := state.LogicalExecutions + uint64(index)
		name := fmt.Sprintf("%020d.json", ordinal)
		if entry.Name() != name || entry.IsDir() {
			return 0, errors.New("partial simulation exploration candidate attempts are not contiguous")
		}
		contents, err := readExplorationFile(filepath.Join(attemptsPath, name), maximumSimulationExplorationPlanBytes)
		if err != nil {
			return 0, err
		}
		var attempt explorationAttempt
		if err := canonicaljson.DecodeCanonicalJSON(contents, &attempt); err != nil || attempt.Ordinal != record.Uint64String(ordinal) || attempt.Seed != record.Uint64String(state.Config.BaseSeed) {
			return 0, errors.Join(errors.New("partial simulation exploration candidate attempt is invalid"), err)
		}
	}
	return uint64(len(entries)), nil
}
