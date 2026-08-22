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
	choiceengine "go.temporal.io/server/tools/gomadv3/runner/internal/exploration/choice"
)

const (
	explorationPlanSchema       = "gomadv3.exploration-plan/v2"
	maximumExplorationPlanBytes = 1 << 20
)

type explorationPlan struct {
	Schema              string              `json:"schema"`
	Config              choiceengine.Config `json:"config"`
	InitialStateSHA256  record.SHA256       `json:"initial_state_sha256"`
	MaximumSegmentBytes record.Uint64String `json:"maximum_segment_bytes"`
}

type ExplorationJournal struct {
	ctx                 context.Context
	batchPath           string
	state               choiceengine.State
	maximumSegmentBytes uint64
	runs                []ExecutionRecord
	chainSHA256         record.SHA256
}

type ExplorationRoundJournal struct {
	journal *ExplorationJournal
	path    string
	index   uint64
	runs    []ExecutionRecord
	runSet  []bool
}

type explorationAttempt struct {
	Ordinal record.Uint64String `json:"ordinal"`
	Seed    record.Uint64String `json:"seed"`
}

type explorationJournalDirectories struct {
	exploration string
	partial     string
}

func NewExplorationJournal(ctx context.Context, batchPath string, initial choiceengine.State, maximumSegmentBytes uint64) (_ *ExplorationJournal, retErr error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if maximumSegmentBytes == 0 {
		return nil, errors.New("exploration segment capacity must be positive")
	}
	batchPath, err := filepath.Abs(batchPath)
	if err != nil {
		return nil, fmt.Errorf("resolve exploration campaign path: %w", err)
	}
	if err := validatePrivateDirectory(batchPath, "exploration campaign"); err != nil {
		return nil, err
	}
	expected, err := choiceengine.New(initial.Config)
	if err != nil {
		return nil, fmt.Errorf("validate initial exploration state: %w", err)
	}
	providedIdentity, err := choiceengine.StateSHA256(initial)
	if err != nil {
		return nil, err
	}
	expectedIdentity, err := choiceengine.StateSHA256(expected)
	if err != nil {
		return nil, err
	}
	if providedIdentity != expectedIdentity {
		return nil, errors.New("exploration journal requires the canonical initial state")
	}
	directories, err := createExplorationJournalDirectories(ctx, batchPath)
	if err != nil {
		return nil, err
	}
	defer func() {
		if retErr != nil {
			retErr = errors.Join(retErr, directories.remove(ctx, batchPath))
		}
	}()
	plan := explorationPlan{
		Schema: explorationPlanSchema, Config: initial.Config, InitialStateSHA256: expectedIdentity,
		MaximumSegmentBytes: record.Uint64String(maximumSegmentBytes),
	}
	encoded, err := canonicaljson.CanonicalJSON(plan)
	if err != nil {
		return nil, err
	}
	planPath := filepath.Join(directories.exploration, "plan.json")
	if err := atomicWriteContext(ctx, planPath, encoded); err != nil {
		return nil, fmt.Errorf("write exploration plan: %w", err)
	}
	if err := syncDirectoryContext(ctx, directories.exploration); err != nil {
		return nil, fmt.Errorf("sync exploration plan directory: %w", err)
	}
	return &ExplorationJournal{ctx: ctx, batchPath: batchPath, state: initial, maximumSegmentBytes: maximumSegmentBytes, chainSHA256: expectedIdentity}, nil
}

func createExplorationJournalDirectories(ctx context.Context, batchPath string) (_ explorationJournalDirectories, retErr error) {
	directories := explorationJournalDirectories{
		exploration: filepath.Join(batchPath, "choice-exploration"),
		partial:     filepath.Join(batchPath, ".partial", "choice-exploration"),
	}
	for _, path := range []string{directories.exploration, directories.partial} {
		if _, err := os.Lstat(path); err == nil {
			return explorationJournalDirectories{}, errors.New("exploration journal already exists")
		} else if !os.IsNotExist(err) {
			return explorationJournalDirectories{}, err
		}
	}
	explorationOwned := false
	partialOwned := false
	defer func() {
		if retErr == nil {
			return
		}
		if partialOwned {
			retErr = errors.Join(retErr, removeCompletedPartialContext(ctx, directories.partial, "exploration-journal"))
			retErr = errors.Join(retErr, syncDirectoryContext(ctx, filepath.Dir(directories.partial)))
		}
		if explorationOwned {
			retErr = errors.Join(retErr, removeCompletedPartialContext(ctx, directories.exploration, "exploration-journal"))
			retErr = errors.Join(retErr, syncDirectoryContext(ctx, batchPath))
		}
	}()
	if err := observeMutation(ctx, mutationCreate, "exploration-directory"); err != nil {
		return explorationJournalDirectories{}, err
	}
	if err := os.Mkdir(directories.exploration, 0o700); err != nil {
		return explorationJournalDirectories{}, err
	}
	explorationOwned = true
	if err := os.Chmod(directories.exploration, 0o700); err != nil {
		return explorationJournalDirectories{}, err
	}
	if err := makePrivateDirectoriesContext(ctx, filepath.Join(directories.exploration, "rounds")); err != nil {
		return explorationJournalDirectories{}, err
	}
	if err := makePrivateDirectoriesContext(ctx, filepath.Join(batchPath, ".partial")); err != nil {
		return explorationJournalDirectories{}, err
	}
	if err := observeMutation(ctx, mutationCreate, "partial-exploration-directory"); err != nil {
		return explorationJournalDirectories{}, err
	}
	if err := os.Mkdir(directories.partial, 0o700); err != nil {
		return explorationJournalDirectories{}, err
	}
	partialOwned = true
	if err := os.Chmod(directories.partial, 0o700); err != nil {
		return explorationJournalDirectories{}, err
	}
	return directories, nil
}

func (directories explorationJournalDirectories) remove(ctx context.Context, batchPath string) error {
	var result error
	result = errors.Join(result, removeCompletedPartialContext(ctx, directories.partial, "exploration-journal"))
	result = errors.Join(result, syncDirectoryContext(ctx, filepath.Dir(directories.partial)))
	result = errors.Join(result, removeCompletedPartialContext(ctx, directories.exploration, "exploration-journal"))
	return errors.Join(result, syncDirectoryContext(ctx, batchPath))
}

func ResumeExplorationJournal(ctx context.Context, batchPath string, config choiceengine.Config, maximumSegmentBytes uint64) (*ExplorationJournal, choiceengine.State, uint64, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	batchPath, err := filepath.Abs(batchPath)
	if err != nil {
		return nil, choiceengine.State{}, 0, fmt.Errorf("resolve exploration campaign path: %w", err)
	}
	if err := validatePrivateDirectory(batchPath, "exploration campaign"); err != nil {
		return nil, choiceengine.State{}, 0, err
	}
	planBytes, err := readExplorationFile(filepath.Join(batchPath, "choice-exploration", "plan.json"), maximumExplorationPlanBytes)
	if err != nil {
		return nil, choiceengine.State{}, 0, fmt.Errorf("read exploration plan: %w", err)
	}
	var plan explorationPlan
	if err := canonicaljson.DecodeCanonicalJSON(planBytes, &plan); err != nil {
		return nil, choiceengine.State{}, 0, fmt.Errorf("decode exploration plan: %w", err)
	}
	if plan.Schema != explorationPlanSchema || plan.Config != config || uint64(plan.MaximumSegmentBytes) != maximumSegmentBytes || maximumSegmentBytes == 0 {
		return nil, choiceengine.State{}, 0, errors.New("exploration plan identity or bounds changed")
	}
	state, err := choiceengine.New(config)
	if err != nil {
		return nil, choiceengine.State{}, 0, err
	}
	initialIdentity, err := choiceengine.StateSHA256(state)
	if err != nil || initialIdentity != plan.InitialStateSHA256 {
		return nil, choiceengine.State{}, 0, errors.Join(errors.New("exploration initial state identity changed"), err)
	}
	roundsPath := filepath.Join(batchPath, "choice-exploration", "rounds")
	if err := validatePrivateDirectory(roundsPath, "exploration rounds"); err != nil {
		return nil, choiceengine.State{}, 0, err
	}
	entries, err := os.ReadDir(roundsPath)
	if err != nil {
		return nil, choiceengine.State{}, 0, fmt.Errorf("read exploration rounds: %w", err)
	}
	if uint64(len(entries)) > config.MaxExecutions {
		return nil, choiceengine.State{}, 0, errors.New("exploration round count exceeds its execution bound")
	}
	journalRuns := []ExecutionRecord{}
	chainSHA256 := initialIdentity
	for index, entry := range entries {
		name := fmt.Sprintf("%020d", index)
		if entry.Name() != name {
			return nil, choiceengine.State{}, 0, fmt.Errorf("exploration round sequence has a gap at %s", name)
		}
		roundPath := filepath.Join(roundsPath, name)
		if err := validateExplorationRoundDirectory(roundPath); err != nil {
			return nil, choiceengine.State{}, 0, err
		}
		roundBytes, err := readExplorationFile(filepath.Join(roundPath, "round.json"), config.MaxExplorationBytes)
		if err != nil {
			return nil, choiceengine.State{}, 0, fmt.Errorf("read exploration round %d: %w", index, err)
		}
		var storedRound choiceengine.Round
		if err := canonicaljson.DecodeCanonicalJSON(roundBytes, &storedRound); err != nil {
			return nil, choiceengine.State{}, 0, fmt.Errorf("decode exploration round %d: %w", index, err)
		}
		expectedRound, ok := state.NextRound()
		equal, equalErr := canonicalEqual(storedRound, expectedRound)
		if equalErr != nil {
			return nil, choiceengine.State{}, 0, fmt.Errorf("compare exploration round %d: %w", index, equalErr)
		}
		if !ok || !equal {
			return nil, choiceengine.State{}, 0, fmt.Errorf("exploration round %d does not match reconstructed state", index)
		}
		segmentBytes, err := readExplorationFile(filepath.Join(roundPath, "segment.json"), maximumSegmentBytes)
		if err != nil {
			return nil, choiceengine.State{}, 0, fmt.Errorf("read exploration segment %d: %w", index, err)
		}
		var segment choiceengine.RoundSegment
		if err := canonicaljson.DecodeCanonicalJSON(segmentBytes, &segment); err != nil {
			return nil, choiceengine.State{}, 0, fmt.Errorf("decode exploration segment %d: %w", index, err)
		}
		logicalStart := state.LogicalExecutions
		state, err = choiceengine.ReplaySegment(state, segment)
		if err != nil {
			return nil, choiceengine.State{}, 0, fmt.Errorf("replay exploration segment %d: %w", index, err)
		}
		chainSHA256 = segment.SHA256
		roundRuns, err := readExplorationRoundExecutions(roundPath, len(storedRound.Candidates), maximumSegmentBytes)
		if err != nil {
			return nil, choiceengine.State{}, 0, fmt.Errorf("read exploration round %d execution records: %w", index, err)
		}
		if len(roundRuns) != 0 {
			if err := validateExplorationRoundExecutions(storedRound, segment, roundRuns, logicalStart, config.BaseSeed); err != nil {
				return nil, choiceengine.State{}, 0, fmt.Errorf("validate exploration round %d execution records: %w", index, err)
			}
		}
		journalRuns = append(journalRuns, roundRuns...)
	}
	recoveryExecutions, err := archiveIncompleteExploration(ctx, batchPath, state, config.MaxExplorationBytes)
	if err != nil {
		return nil, choiceengine.State{}, 0, err
	}
	journal := &ExplorationJournal{ctx: ctx, batchPath: batchPath, state: state, maximumSegmentBytes: maximumSegmentBytes, runs: journalRuns, chainSHA256: chainSHA256}
	return journal, state, recoveryExecutions, nil
}

func (journal *ExplorationJournal) State() choiceengine.State {
	return journal.state
}

func (journal *ExplorationJournal) StageRound(round choiceengine.Round) (_ *ExplorationRoundJournal, retErr error) {
	if journal == nil {
		return nil, errors.New("exploration journal is required")
	}
	expected, ok := journal.state.NextRound()
	equal, err := canonicalEqual(round, expected)
	if err != nil {
		return nil, err
	}
	if !ok || !equal {
		return nil, errors.New("exploration staged round does not match current state")
	}
	path := filepath.Join(journal.batchPath, ".partial", "choice-exploration", fmt.Sprintf("%020d", round.Index))
	if _, err := os.Lstat(path); err == nil {
		return nil, errors.New("exploration round is already staged")
	} else if !os.IsNotExist(err) {
		return nil, err
	}
	if err := observeMutation(journal.ctx, mutationCreate, "exploration-round-directory"); err != nil {
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
		retErr = errors.Join(retErr, removeCompletedPartialContext(journal.ctx, path, "exploration-round"))
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
		return nil, errors.New("exploration round exceeds its exploration byte capacity")
	}
	if err := atomicWriteContext(journal.ctx, filepath.Join(path, "round.json"), encoded); err != nil {
		return nil, fmt.Errorf("write staged exploration round: %w", err)
	}
	if err := syncDirectoryContext(journal.ctx, path); err != nil {
		return nil, fmt.Errorf("sync staged exploration round: %w", err)
	}
	owned = false
	return &ExplorationRoundJournal{journal: journal, path: path, index: round.Index, runs: make([]ExecutionRecord, len(round.Candidates)), runSet: make([]bool, len(round.Candidates))}, nil
}

func (staged *ExplorationRoundJournal) RecordExecution(index int, run ExecutionRecord) error {
	if staged == nil || staged.journal == nil || index < 0 || index >= len(staged.runs) {
		return errors.New("exploration execution record index is invalid")
	}
	if staged.runSet[index] {
		return errors.New("exploration execution record is already staged")
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
		return errors.New("exploration execution record exceeds its byte capacity")
	}
	if err := atomicWriteContext(staged.journal.ctx, filepath.Join(path, fmt.Sprintf("%020d.json", index)), encoded); err != nil {
		return fmt.Errorf("write exploration execution record: %w", err)
	}
	staged.runs[index] = run
	staged.runSet[index] = true
	return nil
}

func (journal *ExplorationJournal) CommitRound(staged *ExplorationRoundJournal, segment choiceengine.RoundSegment) error {
	if journal == nil || staged == nil || staged.journal != journal || staged.index != segment.Index {
		return errors.New("exploration staged round does not match its segment")
	}
	next, err := choiceengine.ReplaySegment(journal.state, segment)
	if err != nil {
		return fmt.Errorf("validate exploration segment before commit: %w", err)
	}
	encoded, err := canonicaljson.CanonicalJSON(segment)
	if err != nil {
		return err
	}
	if uint64(len(encoded)) > journal.maximumSegmentBytes {
		return fmt.Errorf("exploration segment requires %d bytes, exceeding its %d-byte capacity", len(encoded), journal.maximumSegmentBytes)
	}
	if err := atomicWriteContext(journal.ctx, filepath.Join(staged.path, "segment.json"), encoded); err != nil {
		return fmt.Errorf("write staged exploration segment: %w", err)
	}
	if slices.Contains(staged.runSet, true) && slices.Contains(staged.runSet, false) {
		return errors.New("exploration round execution records are incomplete")
	}
	if err := removeCompletedPartialContext(journal.ctx, filepath.Join(staged.path, "candidates"), "exploration-candidate-work"); err != nil {
		return fmt.Errorf("remove completed exploration candidate work: %w", err)
	}
	if err := removeCompletedPartialContext(journal.ctx, filepath.Join(staged.path, "attempts"), "exploration-attempts"); err != nil {
		return fmt.Errorf("remove completed exploration attempt records: %w", err)
	}
	if err := syncDirectoryContext(journal.ctx, staged.path); err != nil {
		return fmt.Errorf("sync staged exploration segment: %w", err)
	}
	finalPath := filepath.Join(journal.batchPath, "choice-exploration", "rounds", fmt.Sprintf("%020d", staged.index))
	if _, err := os.Lstat(finalPath); err == nil {
		return errors.New("exploration segment is already committed")
	} else if !os.IsNotExist(err) {
		return err
	}
	if err := renameContext(journal.ctx, staged.path, finalPath, "exploration-publish"); err != nil {
		return fmt.Errorf("publish exploration segment: %w", err)
	}
	if err := syncDirectoryContext(journal.ctx, filepath.Dir(finalPath)); err != nil {
		return fmt.Errorf("sync exploration segment sequence: %w", err)
	}
	journal.state = next
	journal.chainSHA256 = segment.SHA256
	if slices.Contains(staged.runSet, true) {
		journal.runs = append(journal.runs, staged.runs...)
	}
	return nil
}

func (journal *ExplorationJournal) CommittedExecutions() []ExecutionRecord {
	if journal == nil {
		return nil
	}
	return append([]ExecutionRecord(nil), journal.runs...)
}

func (journal *ExplorationJournal) ChainSHA256() record.SHA256 {
	if journal == nil {
		return ""
	}
	return journal.chainSHA256
}

func ValidatePublishedExploration(batchPath string, expectedSummary choiceengine.Summary, expectedImplementation, expectedChain record.SHA256, expectedRuns []ExecutionRecord) error {
	planBytes, err := readExplorationFile(filepath.Join(batchPath, "choice-exploration", "plan.json"), maximumExplorationPlanBytes)
	if err != nil {
		return fmt.Errorf("read published exploration plan: %w", err)
	}
	var plan explorationPlan
	if err := canonicaljson.DecodeCanonicalJSON(planBytes, &plan); err != nil {
		return fmt.Errorf("decode published exploration plan: %w", err)
	}
	if plan.Schema != explorationPlanSchema || plan.Config.ControllerSHA256 != expectedImplementation || expectedImplementation != choiceengine.ImplementationSHA256() || plan.MaximumSegmentBytes == 0 {
		return errors.New("published exploration plan identity is invalid")
	}
	state, err := choiceengine.New(plan.Config)
	if err != nil {
		return err
	}
	initialIdentity, err := choiceengine.StateSHA256(state)
	if err != nil || initialIdentity != plan.InitialStateSHA256 {
		return errors.Join(errors.New("published exploration initial state identity changed"), err)
	}
	chain := initialIdentity
	roundsPath := filepath.Join(batchPath, "choice-exploration", "rounds")
	entries, err := os.ReadDir(roundsPath)
	if err != nil {
		return err
	}
	committedRuns := []ExecutionRecord{}
	for index, entry := range entries {
		name := fmt.Sprintf("%020d", index)
		if entry.Name() != name {
			return fmt.Errorf("published exploration round sequence has a gap at %s", name)
		}
		roundPath := filepath.Join(roundsPath, name)
		if err := validateExplorationRoundDirectory(roundPath); err != nil {
			return err
		}
		roundBytes, err := readExplorationFile(filepath.Join(roundPath, "round.json"), plan.Config.MaxExplorationBytes)
		if err != nil {
			return err
		}
		var round choiceengine.Round
		if err := canonicaljson.DecodeCanonicalJSON(roundBytes, &round); err != nil {
			return err
		}
		expectedRound, ok := state.NextRound()
		equal, equalErr := canonicalEqual(round, expectedRound)
		if equalErr != nil || !ok || !equal {
			return errors.Join(fmt.Errorf("published exploration round %d does not match its state", index), equalErr)
		}
		segmentBytes, err := readExplorationFile(filepath.Join(roundPath, "segment.json"), uint64(plan.MaximumSegmentBytes))
		if err != nil {
			return err
		}
		var segment choiceengine.RoundSegment
		if err := canonicaljson.DecodeCanonicalJSON(segmentBytes, &segment); err != nil {
			return err
		}
		logicalStart := state.LogicalExecutions
		state, err = choiceengine.ReplaySegment(state, segment)
		if err != nil {
			return err
		}
		runs, err := readExplorationRoundExecutions(roundPath, len(round.Candidates), uint64(plan.MaximumSegmentBytes))
		if err != nil || len(runs) == 0 {
			return errors.Join(errors.New("published exploration round execution records are unavailable"), err)
		}
		if err := validateExplorationRoundExecutions(round, segment, runs, logicalStart, plan.Config.BaseSeed); err != nil {
			return err
		}
		committedRuns = append(committedRuns, runs...)
		chain = segment.SHA256
	}
	equal, err := canonicalEqual(state.Summary(), expectedSummary)
	if err != nil || !equal || chain != expectedChain {
		return errors.Join(errors.New("published exploration summary or chain does not match its campaign"), err)
	}
	if len(committedRuns) != len(expectedRuns) {
		return errors.New("published exploration execution projection count does not match its campaign")
	}
	for index := range committedRuns {
		equal, err := canonicalEqual(committedRuns[index], expectedRuns[index])
		if err != nil || !equal {
			return errors.Join(fmt.Errorf("published exploration execution projection diverges at ordinal %d", index), err)
		}
	}
	partialEntries, err := os.ReadDir(filepath.Join(batchPath, ".partial", "choice-exploration"))
	if err != nil || len(partialEntries) != 0 {
		return errors.Join(errors.New("published exploration retains incomplete round state"), err)
	}
	return nil
}

func (staged *ExplorationRoundJournal) Path() string {
	if staged == nil {
		return ""
	}
	return staged.path
}

func (staged *ExplorationRoundJournal) BeginExecution(ordinal, seed uint64) (*ExecutionJournal, error) {
	if staged == nil || staged.journal == nil {
		return nil, errors.New("exploration staged round is required")
	}
	attempt, err := canonicaljson.CanonicalJSON(explorationAttempt{Ordinal: record.Uint64String(ordinal), Seed: record.Uint64String(seed)})
	if err != nil {
		return nil, err
	}
	if err := atomicWriteContext(staged.journal.ctx, filepath.Join(staged.path, "attempts", fmt.Sprintf("%020d.json", ordinal)), attempt); err != nil {
		return nil, fmt.Errorf("record exploration candidate attempt: %w", err)
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

func archiveIncompleteExploration(ctx context.Context, batchPath string, state choiceengine.State, maximumRoundBytes uint64) (uint64, error) {
	partialRoot := filepath.Join(batchPath, ".partial")
	explorationRoot := filepath.Join(partialRoot, "choice-exploration")
	if err := validatePrivateDirectory(explorationRoot, "partial exploration"); err != nil {
		return 0, err
	}
	entries, err := os.ReadDir(explorationRoot)
	if err != nil {
		return 0, fmt.Errorf("read partial exploration: %w", err)
	}
	if len(entries) == 0 {
		return 0, nil
	}
	if len(entries) != 1 {
		return 0, errors.New("partial exploration contains multiple rounds")
	}
	entry := entries[0]
	if entry.Name() != fmt.Sprintf("%020d", state.CommittedRounds) {
		return 0, errors.New("partial exploration round does not follow committed state")
	}
	path := filepath.Join(explorationRoot, entry.Name())
	if err := validatePrivateDirectory(path, "partial exploration round"); err != nil {
		return 0, err
	}
	roundBytes, err := readExplorationFile(filepath.Join(path, "round.json"), maximumRoundBytes)
	if err != nil {
		return 0, fmt.Errorf("read partial exploration round: %w", err)
	}
	var round choiceengine.Round
	if err := canonicaljson.DecodeCanonicalJSON(roundBytes, &round); err != nil {
		return 0, fmt.Errorf("decode partial exploration round: %w", err)
	}
	expected, ok := state.NextRound()
	equal, err := canonicalEqual(round, expected)
	if err != nil {
		return 0, err
	}
	if !ok || !equal {
		return 0, errors.New("partial exploration round does not match committed state")
	}
	attempts, err := countExplorationAttempts(path, state, round, maximumRoundBytes)
	if err != nil {
		return 0, err
	}
	resumeRoot := filepath.Join(partialRoot, "resume")
	if err := makePrivateDirectoriesContext(ctx, resumeRoot); err != nil {
		return 0, err
	}
	attempt, err := nextResumeAttempt(resumeRoot)
	if err != nil {
		return 0, err
	}
	destinationRoot := filepath.Join(resumeRoot, fmt.Sprintf("%06d", attempt), "partials", "choice-exploration")
	if err := makePrivateDirectoriesContext(ctx, destinationRoot); err != nil {
		return 0, err
	}
	if err := renameContext(ctx, path, filepath.Join(destinationRoot, entry.Name()), "exploration-archive"); err != nil {
		return 0, fmt.Errorf("archive partial exploration round: %w", err)
	}
	if err := syncDirectoryContext(ctx, partialRoot); err != nil {
		return 0, fmt.Errorf("sync partial exploration archive: %w", err)
	}
	return attempts, nil
}

func countExplorationAttempts(roundPath string, state choiceengine.State, round choiceengine.Round, maximumRoundBytes uint64) (uint64, error) {
	attemptsPath := filepath.Join(roundPath, "attempts")
	if _, err := os.Lstat(attemptsPath); os.IsNotExist(err) {
		segmentBytes, err := readExplorationFile(filepath.Join(roundPath, "segment.json"), maximumRoundBytes)
		if err != nil {
			return 0, fmt.Errorf("read cleaned partial exploration segment: %w", err)
		}
		var segment choiceengine.RoundSegment
		if err := canonicaljson.DecodeCanonicalJSON(segmentBytes, &segment); err != nil {
			return 0, fmt.Errorf("decode cleaned partial exploration segment: %w", err)
		}
		if _, err := choiceengine.ReplaySegment(state, segment); err != nil || len(segment.Results) != len(round.Candidates) {
			return 0, errors.Join(errors.New("cleaned partial exploration segment is invalid"), err)
		}
		return uint64(len(round.Candidates)), nil
	} else if err != nil {
		return 0, err
	}
	if err := validatePrivateDirectory(attemptsPath, "partial exploration attempts"); err != nil {
		return 0, err
	}
	entries, err := os.ReadDir(attemptsPath)
	if err != nil {
		return 0, err
	}
	if len(entries) > len(round.Candidates) {
		return 0, errors.New("partial exploration contains too many candidate attempts")
	}
	for index, entry := range entries {
		ordinal := state.LogicalExecutions + uint64(index)
		name := fmt.Sprintf("%020d.json", ordinal)
		if entry.Name() != name || entry.IsDir() {
			return 0, errors.New("partial exploration candidate attempts are not contiguous")
		}
		contents, err := readExplorationFile(filepath.Join(attemptsPath, name), maximumExplorationPlanBytes)
		if err != nil {
			return 0, err
		}
		var attempt explorationAttempt
		if err := canonicaljson.DecodeCanonicalJSON(contents, &attempt); err != nil || attempt.Ordinal != record.Uint64String(ordinal) || attempt.Seed != record.Uint64String(state.Config.BaseSeed) {
			return 0, errors.Join(errors.New("partial exploration candidate attempt is invalid"), err)
		}
	}
	return uint64(len(entries)), nil
}

func validateExplorationRoundDirectory(path string) error {
	if err := validatePrivateDirectory(path, "exploration round"); err != nil {
		return err
	}
	entries, err := os.ReadDir(path)
	if err != nil {
		return err
	}
	seenRound := false
	seenSegment := false
	for _, entry := range entries {
		switch entry.Name() {
		case "round.json":
			seenRound = !entry.IsDir()
		case "segment.json":
			seenSegment = !entry.IsDir()
		case "executions", "failures", "successes":
			if !entry.IsDir() {
				return errors.New("exploration round artifact entry is not a directory")
			}
		default:
			return errors.New("exploration round contains unexpected entries")
		}
	}
	if !seenRound || !seenSegment {
		return errors.New("exploration round is incomplete")
	}
	return nil
}

func readExplorationRoundExecutions(roundPath string, count int, maximumBytes uint64) ([]ExecutionRecord, error) {
	runsPath := filepath.Join(roundPath, "executions")
	if _, err := os.Lstat(runsPath); os.IsNotExist(err) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	if err := validatePrivateDirectory(runsPath, "exploration round executions"); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(runsPath)
	if err != nil {
		return nil, err
	}
	if len(entries) != count {
		return nil, errors.New("exploration round execution record count does not match its candidates")
	}
	runs := make([]ExecutionRecord, count)
	var total uint64
	for index, entry := range entries {
		name := fmt.Sprintf("%020d.json", index)
		if entry.Name() != name || entry.IsDir() {
			return nil, errors.New("exploration round execution record sequence is invalid")
		}
		contents, err := readExplorationFile(filepath.Join(runsPath, name), maximumBytes-total)
		if err != nil {
			return nil, err
		}
		total += uint64(len(contents))
		if err := canonicaljson.DecodeCanonicalJSON(contents, &runs[index]); err != nil {
			return nil, err
		}
	}
	return runs, nil
}

func validateExplorationRoundExecutions(round choiceengine.Round, segment choiceengine.RoundSegment, runs []ExecutionRecord, logicalStart, baseSeed uint64) error {
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
		if result.Failed != (run.Domain == "target" || run.Domain == "watchdog") || result.Failed && (run.FailureSignature == nil || *run.FailureSignature != result.FailureSHA256) {
			return fmt.Errorf("exploration execution %d outcome does not match its segment", index)
		}
	}
	return nil
}

func validatePrivateDirectory(path, name string) error {
	info, err := os.Lstat(path)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm() != 0o700 {
		return errors.Join(fmt.Errorf("%s directory metadata is invalid", name), err)
	}
	return nil
}

func readExplorationFile(path string, maximum uint64) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm() != 0o600 || info.Size() < 0 || uint64(info.Size()) > maximum {
		return nil, errors.Join(errors.New("exploration file metadata or capacity is invalid"), err)
	}
	contents, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if uint64(len(contents)) != uint64(info.Size()) {
		return nil, errors.New("exploration file changed while reading")
	}
	return contents, nil
}

func canonicalEqual(left, right any) (bool, error) {
	leftBytes, err := canonicaljson.CanonicalJSON(left)
	if err != nil {
		return false, err
	}
	rightBytes, err := canonicaljson.CanonicalJSON(right)
	if err != nil {
		return false, err
	}
	return string(leftBytes) == string(rightBytes), nil
}
