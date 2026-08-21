package campaignstore

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"go.temporal.io/server/tools/gomadv3/evidence"
	"go.temporal.io/server/tools/gomadv3/runner/internal/combinedfrontier"
)

const (
	combinedFrontierPlanSchema       = "gomadv3.combined-frontier-plan/v2"
	maximumCombinedFrontierPlanBytes = 1 << 20
)

type combinedFrontierPlan struct {
	Schema              string                  `json:"schema"`
	Config              combinedfrontier.Config `json:"config"`
	InitialStateSHA256  evidence.SHA256         `json:"initial_state_sha256"`
	MaximumSegmentBytes evidence.Uint64String   `json:"maximum_segment_bytes"`
}

type CombinedFrontierJournal struct {
	ctx                 context.Context
	batchPath           string
	state               combinedfrontier.State
	maximumSegmentBytes uint64
	runs                []ExecutionRecord
	chainSHA256         evidence.SHA256
}

type CombinedFrontierRoundJournal struct {
	journal *CombinedFrontierJournal
	path    string
	index   uint64
	runs    []ExecutionRecord
	runSet  []bool
}

type CombinedFrontierInspection struct {
	Summary              combinedfrontier.Summary
	ImplementationSHA256 evidence.SHA256
	ChainSHA256          evidence.SHA256
	Pending              []combinedfrontier.Candidate
	StagedRound          *CombinedFrontierStagedRound
}

type CombinedFrontierStagedRound struct {
	Index      uint64
	Candidates uint64
	Attempted  uint64
}

type reconstructedCombinedFrontier struct {
	plan         combinedFrontierPlan
	state        combinedfrontier.State
	runs         []ExecutionRecord
	chainSHA256  evidence.SHA256
	completeRuns bool
}

func InspectCombinedFrontier(batchPath string) (CombinedFrontierInspection, error) {
	batchPath, err := filepath.Abs(batchPath)
	if err != nil {
		return CombinedFrontierInspection{}, fmt.Errorf("resolve combined frontier batch path: %w", err)
	}
	if err := validatePrivateDirectory(batchPath, "combined frontier batch"); err != nil {
		return CombinedFrontierInspection{}, err
	}
	reconstructed, err := reconstructCombinedFrontier(batchPath)
	if err != nil {
		return CombinedFrontierInspection{}, err
	}
	staged, err := inspectIncompleteCombinedFrontierRound(batchPath, reconstructed.state)
	if err != nil {
		return CombinedFrontierInspection{}, err
	}
	return CombinedFrontierInspection{
		Summary: reconstructed.state.Summary(), ImplementationSHA256: reconstructed.plan.Config.ControllerSHA256,
		ChainSHA256: reconstructed.chainSHA256, Pending: append([]combinedfrontier.Candidate(nil), reconstructed.state.Queue...), StagedRound: staged,
	}, nil
}

func reconstructCombinedFrontier(batchPath string) (reconstructedCombinedFrontier, error) {
	frontierPath := filepath.Join(batchPath, "combined-frontier")
	planBytes, err := readFrontierFile(filepath.Join(frontierPath, "plan.json"), maximumCombinedFrontierPlanBytes)
	if err != nil {
		return reconstructedCombinedFrontier{}, fmt.Errorf("read combined frontier plan: %w", err)
	}
	var plan combinedFrontierPlan
	if err := evidence.DecodeCanonicalJSON(planBytes, &plan); err != nil {
		return reconstructedCombinedFrontier{}, fmt.Errorf("decode combined frontier plan: %w", err)
	}
	if plan.Schema != combinedFrontierPlanSchema || plan.MaximumSegmentBytes == 0 {
		return reconstructedCombinedFrontier{}, errors.New("combined frontier plan identity or bounds are invalid")
	}
	state, err := combinedfrontier.New(plan.Config)
	if err != nil {
		return reconstructedCombinedFrontier{}, err
	}
	initialIdentity, err := combinedfrontier.StateSHA256(state)
	if err != nil || initialIdentity != plan.InitialStateSHA256 {
		return reconstructedCombinedFrontier{}, errors.Join(errors.New("combined frontier initial state identity changed"), err)
	}
	roundsPath := filepath.Join(frontierPath, "rounds")
	if err := validatePrivateDirectory(roundsPath, "combined frontier rounds"); err != nil {
		return reconstructedCombinedFrontier{}, err
	}
	entries, err := os.ReadDir(roundsPath)
	if err != nil {
		return reconstructedCombinedFrontier{}, fmt.Errorf("read combined frontier rounds: %w", err)
	}
	if uint64(len(entries)) > plan.Config.MaxRuns {
		return reconstructedCombinedFrontier{}, errors.New("combined frontier round count exceeds its run bound")
	}
	chainSHA256 := initialIdentity
	journalRuns := []ExecutionRecord{}
	completeRuns := true
	for index, entry := range entries {
		name := fmt.Sprintf("%020d", index)
		if entry.Name() != name {
			return reconstructedCombinedFrontier{}, fmt.Errorf("combined frontier round sequence has a gap at %s", name)
		}
		roundPath := filepath.Join(roundsPath, name)
		if err := validateCombinedFrontierRoundDirectory(roundPath, true); err != nil {
			return reconstructedCombinedFrontier{}, err
		}
		roundBytes, err := readFrontierFile(filepath.Join(roundPath, "round.json"), plan.Config.MaxFrontierBytes)
		if err != nil {
			return reconstructedCombinedFrontier{}, fmt.Errorf("read combined frontier round %d: %w", index, err)
		}
		var storedRound combinedfrontier.Round
		if err := evidence.DecodeCanonicalJSON(roundBytes, &storedRound); err != nil {
			return reconstructedCombinedFrontier{}, fmt.Errorf("decode combined frontier round %d: %w", index, err)
		}
		expectedRound, ok := state.NextRound()
		equal, equalErr := canonicalEqual(storedRound, expectedRound)
		if equalErr != nil || !ok || !equal {
			return reconstructedCombinedFrontier{}, errors.Join(fmt.Errorf("combined frontier round %d does not match reconstructed state", index), equalErr)
		}
		segmentBytes, err := readFrontierFile(filepath.Join(roundPath, "segment.json"), uint64(plan.MaximumSegmentBytes))
		if err != nil {
			return reconstructedCombinedFrontier{}, fmt.Errorf("read combined frontier segment %d: %w", index, err)
		}
		var segment combinedfrontier.RoundSegment
		if err := evidence.DecodeCanonicalJSON(segmentBytes, &segment); err != nil {
			return reconstructedCombinedFrontier{}, fmt.Errorf("decode combined frontier segment %d: %w", index, err)
		}
		logicalStart := state.LogicalExecutions
		state, err = combinedfrontier.ReplaySegment(state, segment)
		if err != nil {
			return reconstructedCombinedFrontier{}, fmt.Errorf("replay combined frontier segment %d: %w", index, err)
		}
		runs, err := readFrontierRoundExecutions(roundPath, len(storedRound.Candidates), uint64(plan.MaximumSegmentBytes))
		if err != nil {
			return reconstructedCombinedFrontier{}, fmt.Errorf("read combined frontier run records %d: %w", index, err)
		}
		if len(runs) == 0 {
			completeRuns = false
		} else {
			if err := validateCombinedFrontierRoundExecutions(storedRound, segment, runs, logicalStart, plan.Config.BaseSeed); err != nil {
				return reconstructedCombinedFrontier{}, fmt.Errorf("validate combined frontier run records %d: %w", index, err)
			}
			journalRuns = append(journalRuns, runs...)
		}
		chainSHA256 = segment.SHA256
	}
	return reconstructedCombinedFrontier{plan: plan, state: state, runs: journalRuns, chainSHA256: chainSHA256, completeRuns: completeRuns}, nil
}

func NewCombinedFrontierJournal(ctx context.Context, batchPath string, initial combinedfrontier.State, maximumSegmentBytes uint64) (_ *CombinedFrontierJournal, retErr error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if maximumSegmentBytes == 0 {
		return nil, errors.New("combined frontier segment capacity must be positive")
	}
	batchPath, err := filepath.Abs(batchPath)
	if err != nil {
		return nil, fmt.Errorf("resolve combined frontier batch path: %w", err)
	}
	if err := validatePrivateDirectory(batchPath, "combined frontier batch"); err != nil {
		return nil, err
	}
	expected, err := combinedfrontier.New(initial.Config)
	if err != nil {
		return nil, fmt.Errorf("validate initial combined frontier state: %w", err)
	}
	providedIdentity, err := combinedfrontier.StateSHA256(initial)
	if err != nil {
		return nil, err
	}
	expectedIdentity, err := combinedfrontier.StateSHA256(expected)
	if err != nil {
		return nil, err
	}
	if providedIdentity != expectedIdentity {
		return nil, errors.New("combined frontier journal requires the canonical initial state")
	}
	frontierPath := filepath.Join(batchPath, "combined-frontier")
	partialPath := filepath.Join(batchPath, ".partial", "combined-frontier")
	for _, path := range []string{frontierPath, partialPath} {
		if _, err := os.Lstat(path); err == nil {
			return nil, errors.New("combined frontier journal already exists")
		} else if !os.IsNotExist(err) {
			return nil, err
		}
	}
	defer func() {
		if retErr == nil {
			return
		}
		retErr = errors.Join(retErr, removeCompletedPartialContext(ctx, partialPath, "combined-frontier-journal"))
		retErr = errors.Join(retErr, removeCompletedPartialContext(ctx, frontierPath, "combined-frontier-journal"))
	}()
	if err := makePrivateDirectoriesContext(ctx, filepath.Join(frontierPath, "rounds")); err != nil {
		return nil, err
	}
	if err := makePrivateDirectoriesContext(ctx, partialPath); err != nil {
		return nil, err
	}
	plan := combinedFrontierPlan{
		Schema: combinedFrontierPlanSchema, Config: initial.Config, InitialStateSHA256: expectedIdentity,
		MaximumSegmentBytes: evidence.Uint64String(maximumSegmentBytes),
	}
	encoded, err := evidence.CanonicalJSON(plan)
	if err != nil {
		return nil, err
	}
	if err := atomicWriteContext(ctx, filepath.Join(frontierPath, "plan.json"), encoded); err != nil {
		return nil, fmt.Errorf("write combined frontier plan: %w", err)
	}
	if err := syncDirectoryContext(ctx, frontierPath); err != nil {
		return nil, fmt.Errorf("sync combined frontier plan directory: %w", err)
	}
	return &CombinedFrontierJournal{
		ctx: ctx, batchPath: batchPath, state: initial, maximumSegmentBytes: maximumSegmentBytes, chainSHA256: expectedIdentity,
	}, nil
}

func ResumeCombinedFrontierJournal(ctx context.Context, batchPath string, config combinedfrontier.Config, maximumSegmentBytes uint64) (*CombinedFrontierJournal, combinedfrontier.State, uint64, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	batchPath, err := filepath.Abs(batchPath)
	if err != nil {
		return nil, combinedfrontier.State{}, 0, fmt.Errorf("resolve combined frontier batch path: %w", err)
	}
	if err := validatePrivateDirectory(batchPath, "combined frontier batch"); err != nil {
		return nil, combinedfrontier.State{}, 0, err
	}
	frontierPath := filepath.Join(batchPath, "combined-frontier")
	planBytes, err := readFrontierFile(filepath.Join(frontierPath, "plan.json"), maximumCombinedFrontierPlanBytes)
	if err != nil {
		return nil, combinedfrontier.State{}, 0, fmt.Errorf("read combined frontier plan: %w", err)
	}
	var plan combinedFrontierPlan
	if err := evidence.DecodeCanonicalJSON(planBytes, &plan); err != nil {
		return nil, combinedfrontier.State{}, 0, fmt.Errorf("decode combined frontier plan: %w", err)
	}
	if plan.Schema != combinedFrontierPlanSchema || plan.Config != config || uint64(plan.MaximumSegmentBytes) != maximumSegmentBytes || maximumSegmentBytes == 0 {
		return nil, combinedfrontier.State{}, 0, errors.New("combined frontier plan identity or bounds changed")
	}
	state, err := combinedfrontier.New(config)
	if err != nil {
		return nil, combinedfrontier.State{}, 0, err
	}
	initialIdentity, err := combinedfrontier.StateSHA256(state)
	if err != nil || initialIdentity != plan.InitialStateSHA256 {
		return nil, combinedfrontier.State{}, 0, errors.Join(errors.New("combined frontier initial state identity changed"), err)
	}
	roundsPath := filepath.Join(frontierPath, "rounds")
	if err := validatePrivateDirectory(roundsPath, "combined frontier rounds"); err != nil {
		return nil, combinedfrontier.State{}, 0, err
	}
	entries, err := os.ReadDir(roundsPath)
	if err != nil {
		return nil, combinedfrontier.State{}, 0, fmt.Errorf("read combined frontier rounds: %w", err)
	}
	if uint64(len(entries)) > config.MaxRuns {
		return nil, combinedfrontier.State{}, 0, errors.New("combined frontier round count exceeds its run bound")
	}
	chainSHA256 := initialIdentity
	journalRuns := []ExecutionRecord{}
	for index, entry := range entries {
		name := fmt.Sprintf("%020d", index)
		if entry.Name() != name {
			return nil, combinedfrontier.State{}, 0, fmt.Errorf("combined frontier round sequence has a gap at %s", name)
		}
		roundPath := filepath.Join(roundsPath, name)
		if err := validateCombinedFrontierRoundDirectory(roundPath, true); err != nil {
			return nil, combinedfrontier.State{}, 0, err
		}
		roundBytes, err := readFrontierFile(filepath.Join(roundPath, "round.json"), config.MaxFrontierBytes)
		if err != nil {
			return nil, combinedfrontier.State{}, 0, fmt.Errorf("read combined frontier round %d: %w", index, err)
		}
		var storedRound combinedfrontier.Round
		if err := evidence.DecodeCanonicalJSON(roundBytes, &storedRound); err != nil {
			return nil, combinedfrontier.State{}, 0, fmt.Errorf("decode combined frontier round %d: %w", index, err)
		}
		expectedRound, ok := state.NextRound()
		equal, equalErr := canonicalEqual(storedRound, expectedRound)
		if equalErr != nil || !ok || !equal {
			return nil, combinedfrontier.State{}, 0, errors.Join(fmt.Errorf("combined frontier round %d does not match reconstructed state", index), equalErr)
		}
		segmentBytes, err := readFrontierFile(filepath.Join(roundPath, "segment.json"), maximumSegmentBytes)
		if err != nil {
			return nil, combinedfrontier.State{}, 0, fmt.Errorf("read combined frontier segment %d: %w", index, err)
		}
		var segment combinedfrontier.RoundSegment
		if err := evidence.DecodeCanonicalJSON(segmentBytes, &segment); err != nil {
			return nil, combinedfrontier.State{}, 0, fmt.Errorf("decode combined frontier segment %d: %w", index, err)
		}
		logicalStart := state.LogicalExecutions
		state, err = combinedfrontier.ReplaySegment(state, segment)
		if err != nil {
			return nil, combinedfrontier.State{}, 0, fmt.Errorf("replay combined frontier segment %d: %w", index, err)
		}
		runs, err := readFrontierRoundExecutions(roundPath, len(storedRound.Candidates), maximumSegmentBytes)
		if err != nil {
			return nil, combinedfrontier.State{}, 0, fmt.Errorf("read combined frontier run records %d: %w", index, err)
		}
		if len(runs) != 0 {
			if err := validateCombinedFrontierRoundExecutions(storedRound, segment, runs, logicalStart, config.BaseSeed); err != nil {
				return nil, combinedfrontier.State{}, 0, fmt.Errorf("validate combined frontier run records %d: %w", index, err)
			}
			journalRuns = append(journalRuns, runs...)
		}
		chainSHA256 = segment.SHA256
	}
	recoveryExecutions, err := discardIncompleteCombinedFrontierRound(ctx, batchPath, state)
	if err != nil {
		return nil, combinedfrontier.State{}, 0, err
	}
	journal := &CombinedFrontierJournal{
		ctx: ctx, batchPath: batchPath, state: state, maximumSegmentBytes: maximumSegmentBytes, runs: journalRuns, chainSHA256: chainSHA256,
	}
	return journal, state, recoveryExecutions, nil
}

func (journal *CombinedFrontierJournal) State() combinedfrontier.State {
	if journal == nil {
		return combinedfrontier.State{}
	}
	return journal.state
}

func (journal *CombinedFrontierJournal) ChainSHA256() evidence.SHA256 {
	if journal == nil {
		return ""
	}
	return journal.chainSHA256
}

func ValidatePublishedCombinedFrontier(batchPath string, expectedSummary combinedfrontier.Summary, expectedImplementation, expectedChain evidence.SHA256, expectedRuns []ExecutionRecord) error {
	planBytes, err := readFrontierFile(filepath.Join(batchPath, "combined-frontier", "plan.json"), maximumCombinedFrontierPlanBytes)
	if err != nil {
		return fmt.Errorf("read published combined frontier plan: %w", err)
	}
	var plan combinedFrontierPlan
	if err := evidence.DecodeCanonicalJSON(planBytes, &plan); err != nil {
		return fmt.Errorf("decode published combined frontier plan: %w", err)
	}
	if plan.Schema != combinedFrontierPlanSchema || plan.Config.ControllerSHA256 != expectedImplementation || expectedImplementation != combinedfrontier.ImplementationSHA256() || plan.MaximumSegmentBytes == 0 {
		return errors.New("published combined frontier plan identity is invalid")
	}
	state, err := combinedfrontier.New(plan.Config)
	if err != nil {
		return err
	}
	initialIdentity, err := combinedfrontier.StateSHA256(state)
	if err != nil || initialIdentity != plan.InitialStateSHA256 {
		return errors.Join(errors.New("published combined frontier initial state identity changed"), err)
	}
	chain := initialIdentity
	roundsPath := filepath.Join(batchPath, "combined-frontier", "rounds")
	entries, err := os.ReadDir(roundsPath)
	if err != nil {
		return err
	}
	committedRuns := []ExecutionRecord{}
	for index, entry := range entries {
		name := fmt.Sprintf("%020d", index)
		if entry.Name() != name {
			return fmt.Errorf("published combined frontier round sequence has a gap at %s", name)
		}
		roundPath := filepath.Join(roundsPath, name)
		if err := validateCombinedFrontierRoundDirectory(roundPath, true); err != nil {
			return err
		}
		roundBytes, err := readFrontierFile(filepath.Join(roundPath, "round.json"), plan.Config.MaxFrontierBytes)
		if err != nil {
			return err
		}
		var round combinedfrontier.Round
		if err := evidence.DecodeCanonicalJSON(roundBytes, &round); err != nil {
			return err
		}
		expectedRound, ok := state.NextRound()
		equal, equalErr := canonicalEqual(round, expectedRound)
		if equalErr != nil || !ok || !equal {
			return errors.Join(fmt.Errorf("published combined frontier round %d does not match its state", index), equalErr)
		}
		segmentBytes, err := readFrontierFile(filepath.Join(roundPath, "segment.json"), uint64(plan.MaximumSegmentBytes))
		if err != nil {
			return err
		}
		var segment combinedfrontier.RoundSegment
		if err := evidence.DecodeCanonicalJSON(segmentBytes, &segment); err != nil {
			return err
		}
		logicalStart := state.LogicalExecutions
		state, err = combinedfrontier.ReplaySegment(state, segment)
		if err != nil {
			return err
		}
		runs, err := readFrontierRoundExecutions(roundPath, len(round.Candidates), uint64(plan.MaximumSegmentBytes))
		if err != nil || len(runs) == 0 {
			return errors.Join(errors.New("published combined frontier run records are unavailable"), err)
		}
		if err := validateCombinedFrontierRoundExecutions(round, segment, runs, logicalStart, plan.Config.BaseSeed); err != nil {
			return err
		}
		committedRuns = append(committedRuns, runs...)
		chain = segment.SHA256
	}
	equal, err := canonicalEqual(state.Summary(), expectedSummary)
	if err != nil || !equal || chain != expectedChain {
		return errors.Join(errors.New("published combined frontier summary or chain does not match its batch"), err)
	}
	if len(committedRuns) != len(expectedRuns) {
		return errors.New("published combined frontier run projection count does not match its batch")
	}
	for index := range committedRuns {
		equal, err := canonicalEqual(committedRuns[index], expectedRuns[index])
		if err != nil || !equal {
			return errors.Join(fmt.Errorf("published combined frontier run projection diverges at ordinal %d", index), err)
		}
	}
	partialEntries, err := os.ReadDir(filepath.Join(batchPath, ".partial", "combined-frontier"))
	if err != nil || len(partialEntries) != 0 {
		return errors.Join(errors.New("published combined frontier retains incomplete round state"), err)
	}
	return nil
}

func (journal *CombinedFrontierJournal) StageRound(round combinedfrontier.Round) (_ *CombinedFrontierRoundJournal, retErr error) {
	if journal == nil {
		return nil, errors.New("combined frontier journal is required")
	}
	expected, ok := journal.state.NextRound()
	equal, err := canonicalEqual(round, expected)
	if err != nil {
		return nil, err
	}
	if !ok || !equal {
		return nil, errors.New("combined frontier staged round does not match current state")
	}
	path := filepath.Join(journal.batchPath, ".partial", "combined-frontier", fmt.Sprintf("%020d", round.Index))
	if _, err := os.Lstat(path); err == nil {
		return nil, errors.New("combined frontier round is already staged")
	} else if !os.IsNotExist(err) {
		return nil, err
	}
	if err := observeMutation(journal.ctx, mutationCreate, "combined-frontier-round-directory"); err != nil {
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
		retErr = errors.Join(retErr, removeCompletedPartialContext(journal.ctx, path, "combined-frontier-round"))
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
	encoded, err := evidence.CanonicalJSON(round)
	if err != nil {
		return nil, err
	}
	if uint64(len(encoded)) > journal.state.Config.MaxFrontierBytes {
		return nil, errors.New("combined frontier round exceeds its frontier byte capacity")
	}
	if err := atomicWriteContext(journal.ctx, filepath.Join(path, "round.json"), encoded); err != nil {
		return nil, fmt.Errorf("write staged combined frontier round: %w", err)
	}
	if err := syncDirectoryContext(journal.ctx, path); err != nil {
		return nil, fmt.Errorf("sync staged combined frontier round: %w", err)
	}
	owned = false
	return &CombinedFrontierRoundJournal{
		journal: journal, path: path, index: round.Index,
		runs: make([]ExecutionRecord, len(round.Candidates)), runSet: make([]bool, len(round.Candidates)),
	}, nil
}

func (staged *CombinedFrontierRoundJournal) RecordExecution(index int, run ExecutionRecord) error {
	if staged == nil || staged.journal == nil || index < 0 || index >= len(staged.runs) {
		return errors.New("combined frontier run record index is invalid")
	}
	if staged.runSet[index] {
		return errors.New("combined frontier run record is already staged")
	}
	path := filepath.Join(staged.path, "runs")
	if err := makePrivateDirectoriesContext(staged.journal.ctx, path); err != nil {
		return err
	}
	encoded, err := evidence.CanonicalJSON(run)
	if err != nil {
		return err
	}
	if uint64(len(encoded)) > staged.journal.maximumSegmentBytes {
		return errors.New("combined frontier run record exceeds its byte capacity")
	}
	if err := atomicWriteContext(staged.journal.ctx, filepath.Join(path, fmt.Sprintf("%020d.json", index)), encoded); err != nil {
		return fmt.Errorf("write combined frontier run record: %w", err)
	}
	staged.runs[index] = run
	staged.runSet[index] = true
	return nil
}

func (journal *CombinedFrontierJournal) CommitRound(staged *CombinedFrontierRoundJournal, segment combinedfrontier.RoundSegment) error {
	if journal == nil || staged == nil || staged.journal != journal || staged.index != segment.Index {
		return errors.New("combined frontier staged round does not match its segment")
	}
	next, err := combinedfrontier.ReplaySegment(journal.state, segment)
	if err != nil {
		return fmt.Errorf("validate combined frontier segment before commit: %w", err)
	}
	encoded, err := evidence.CanonicalJSON(segment)
	if err != nil {
		return err
	}
	if uint64(len(encoded)) > journal.maximumSegmentBytes {
		return fmt.Errorf("combined frontier segment requires %d bytes, exceeding its %d-byte capacity", len(encoded), journal.maximumSegmentBytes)
	}
	if err := atomicWriteContext(journal.ctx, filepath.Join(staged.path, "segment.json"), encoded); err != nil {
		return fmt.Errorf("write staged combined frontier segment: %w", err)
	}
	if slices.Contains(staged.runSet, true) && slices.Contains(staged.runSet, false) {
		return errors.New("combined frontier round run records are incomplete")
	}
	if err := removeCompletedPartialContext(journal.ctx, filepath.Join(staged.path, "candidates"), "combined-frontier-candidate-work"); err != nil {
		return fmt.Errorf("remove completed combined frontier candidate work: %w", err)
	}
	if err := removeCompletedPartialContext(journal.ctx, filepath.Join(staged.path, "attempts"), "combined-frontier-attempts"); err != nil {
		return fmt.Errorf("remove completed combined frontier attempt records: %w", err)
	}
	if err := syncDirectoryContext(journal.ctx, staged.path); err != nil {
		return fmt.Errorf("sync staged combined frontier segment: %w", err)
	}
	finalPath := filepath.Join(journal.batchPath, "combined-frontier", "rounds", fmt.Sprintf("%020d", staged.index))
	if _, err := os.Lstat(finalPath); err == nil {
		return errors.New("combined frontier segment is already committed")
	} else if !os.IsNotExist(err) {
		return err
	}
	if err := renameContext(journal.ctx, staged.path, finalPath, "combined-frontier-publish"); err != nil {
		return fmt.Errorf("publish combined frontier segment: %w", err)
	}
	if err := syncDirectoryContext(journal.ctx, filepath.Dir(finalPath)); err != nil {
		return fmt.Errorf("sync combined frontier segment sequence: %w", err)
	}
	journal.state = next
	journal.chainSHA256 = segment.SHA256
	if slices.Contains(staged.runSet, true) {
		journal.runs = append(journal.runs, staged.runs...)
	}
	return nil
}

func (journal *CombinedFrontierJournal) CommittedExecutions() []ExecutionRecord {
	if journal == nil {
		return nil
	}
	return append([]ExecutionRecord(nil), journal.runs...)
}

func (staged *CombinedFrontierRoundJournal) Path() string {
	if staged == nil {
		return ""
	}
	return staged.path
}

func (staged *CombinedFrontierRoundJournal) BeginExecution(ordinal, seed uint64) (*ExecutionJournal, error) {
	if staged == nil || staged.journal == nil {
		return nil, errors.New("combined frontier staged round is required")
	}
	attempt, err := evidence.CanonicalJSON(frontierAttempt{Ordinal: evidence.Uint64String(ordinal), Seed: evidence.Uint64String(seed)})
	if err != nil {
		return nil, err
	}
	if err := atomicWriteContext(staged.journal.ctx, filepath.Join(staged.path, "attempts", fmt.Sprintf("%020d.json", ordinal)), attempt); err != nil {
		return nil, fmt.Errorf("record combined frontier candidate attempt: %w", err)
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

func validateCombinedFrontierRoundDirectory(path string, complete bool) error {
	if err := validatePrivateDirectory(path, "combined frontier round"); err != nil {
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
		case "runs", "failures", "successes":
			if !entry.IsDir() {
				return errors.New("combined frontier round artifact entry is not a directory")
			}
		case "candidates", "attempts":
			if complete || !entry.IsDir() {
				return errors.New("combined frontier round work entry is invalid")
			}
		default:
			return errors.New("combined frontier round contains an unexpected entry")
		}
	}
	if !seenRound || complete != seenSegment {
		return errors.New("combined frontier round directory contents are invalid")
	}
	return nil
}

func validateCombinedFrontierRoundExecutions(round combinedfrontier.Round, segment combinedfrontier.RoundSegment, runs []ExecutionRecord, logicalStart, baseSeed uint64) error {
	if len(runs) != len(round.Candidates) || len(runs) != len(segment.Results) {
		return errors.New("combined frontier round run records do not match its results")
	}
	for index, run := range runs {
		candidate := round.Candidates[index]
		result := segment.Results[index]
		if run.Strategy != "combined-frontier" || run.SelectionOrdinal != evidence.Uint64String(logicalStart+uint64(index)) || run.Seed != evidence.Uint64String(baseSeed) || run.Round == nil || *run.Round != evidence.Uint64String(round.Index) || run.CandidateSHA256 != candidate.SHA256 || run.ParentCandidateSHA256 != candidate.ParentSHA256 || run.PrefixSHA256 != "" || run.ForcedDepth == nil || *run.ForcedDepth != evidence.Uint64String(len(candidate.Overrides)) || run.OutcomeSHA256 != result.OutcomeSHA256 {
			return fmt.Errorf("combined frontier run %d provenance does not match its segment", index)
		}
		failed := run.Domain == "target" || run.Domain == "watchdog"
		if result.Failed != failed || result.Failed && (run.FailureSignature == nil || *run.FailureSignature != result.FailureSHA256) {
			return fmt.Errorf("combined frontier run %d outcome does not match its segment", index)
		}
	}
	return nil
}

func inspectIncompleteCombinedFrontierRound(batchPath string, state combinedfrontier.State) (*CombinedFrontierStagedRound, error) {
	partialPath := filepath.Join(batchPath, ".partial", "combined-frontier")
	if err := validatePrivateDirectory(partialPath, "partial combined frontier"); err != nil {
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
		return nil, errors.New("partial combined frontier contains multiple rounds")
	}
	expected, ok := state.NextRound()
	name := fmt.Sprintf("%020d", expected.Index)
	if !ok || entries[0].Name() != name {
		return nil, errors.New("partial combined frontier round does not match reconstructed state")
	}
	path := filepath.Join(partialPath, name)
	if err := validatePrivateDirectory(path, "partial combined frontier round"); err != nil {
		return nil, err
	}
	roundBytes, err := readFrontierFile(filepath.Join(path, "round.json"), state.Config.MaxFrontierBytes)
	if err != nil {
		return nil, err
	}
	var round combinedfrontier.Round
	if err := evidence.DecodeCanonicalJSON(roundBytes, &round); err != nil {
		return nil, err
	}
	equal, err := canonicalEqual(round, expected)
	if err != nil || !equal {
		return nil, errors.Join(errors.New("partial combined frontier round changed"), err)
	}
	attempts, err := countCombinedFrontierAttempts(path, state, round)
	if err != nil {
		return nil, err
	}
	return &CombinedFrontierStagedRound{Index: round.Index, Candidates: uint64(len(round.Candidates)), Attempted: attempts}, nil
}

func discardIncompleteCombinedFrontierRound(ctx context.Context, batchPath string, state combinedfrontier.State) (uint64, error) {
	staged, err := inspectIncompleteCombinedFrontierRound(batchPath, state)
	if err != nil || staged == nil {
		return 0, err
	}
	partialPath := filepath.Join(batchPath, ".partial", "combined-frontier")
	path := filepath.Join(partialPath, fmt.Sprintf("%020d", staged.Index))
	if err := removeCompletedPartialContext(ctx, path, "combined-frontier-incomplete-round"); err != nil {
		return 0, err
	}
	if err := syncDirectoryContext(ctx, partialPath); err != nil {
		return 0, err
	}
	return staged.Attempted, nil
}

func countCombinedFrontierAttempts(roundPath string, state combinedfrontier.State, round combinedfrontier.Round) (uint64, error) {
	attemptsPath := filepath.Join(roundPath, "attempts")
	if err := validatePrivateDirectory(attemptsPath, "partial combined frontier attempts"); err != nil {
		return 0, err
	}
	entries, err := os.ReadDir(attemptsPath)
	if err != nil {
		return 0, err
	}
	if len(entries) > len(round.Candidates) {
		return 0, errors.New("partial combined frontier contains too many candidate attempts")
	}
	for index, entry := range entries {
		ordinal := state.LogicalExecutions + uint64(index)
		name := fmt.Sprintf("%020d.json", ordinal)
		if entry.Name() != name || entry.IsDir() {
			return 0, errors.New("partial combined frontier candidate attempts are not contiguous")
		}
		contents, err := readFrontierFile(filepath.Join(attemptsPath, name), maximumCombinedFrontierPlanBytes)
		if err != nil {
			return 0, err
		}
		var attempt frontierAttempt
		if err := evidence.DecodeCanonicalJSON(contents, &attempt); err != nil || attempt.Ordinal != evidence.Uint64String(ordinal) || attempt.Seed != evidence.Uint64String(state.Config.BaseSeed) {
			return 0, errors.Join(errors.New("partial combined frontier candidate attempt is invalid"), err)
		}
	}
	return uint64(len(entries)), nil
}
