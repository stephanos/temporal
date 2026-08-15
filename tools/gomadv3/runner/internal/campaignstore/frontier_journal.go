package campaignstore

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"go.temporal.io/server/tools/gomadv3/evidence"
	"go.temporal.io/server/tools/gomadv3/runner/internal/frontier"
)

const (
	frontierPlanSchema       = "gomadv3.frontier-plan/v2"
	maximumFrontierPlanBytes = 1 << 20
)

type frontierPlan struct {
	Schema              string                `json:"schema"`
	Config              frontier.Config       `json:"config"`
	InitialStateSHA256  evidence.SHA256       `json:"initial_state_sha256"`
	MaximumSegmentBytes evidence.Uint64String `json:"maximum_segment_bytes"`
}

type FrontierJournal struct {
	ctx                 context.Context
	batchPath           string
	state               frontier.State
	maximumSegmentBytes uint64
	runs                []ExecutionRecord
	chainSHA256         evidence.SHA256
}

type FrontierRoundJournal struct {
	journal *FrontierJournal
	path    string
	index   uint64
	runs    []ExecutionRecord
	runSet  []bool
}

type frontierAttempt struct {
	Ordinal evidence.Uint64String `json:"ordinal"`
	Seed    evidence.Uint64String `json:"seed"`
}

func NewFrontierJournal(ctx context.Context, batchPath string, initial frontier.State, maximumSegmentBytes uint64) (*FrontierJournal, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if maximumSegmentBytes == 0 {
		return nil, errors.New("frontier segment capacity must be positive")
	}
	batchPath, err := filepath.Abs(batchPath)
	if err != nil {
		return nil, fmt.Errorf("resolve frontier batch path: %w", err)
	}
	if err := validatePrivateDirectory(batchPath, "frontier batch"); err != nil {
		return nil, err
	}
	expected, err := frontier.New(initial.Config)
	if err != nil {
		return nil, fmt.Errorf("validate initial frontier state: %w", err)
	}
	providedIdentity, err := frontier.StateSHA256(initial)
	if err != nil {
		return nil, err
	}
	expectedIdentity, err := frontier.StateSHA256(expected)
	if err != nil {
		return nil, err
	}
	if providedIdentity != expectedIdentity {
		return nil, errors.New("frontier journal requires the canonical initial state")
	}
	frontierPath := filepath.Join(batchPath, "frontier")
	for _, directory := range []string{frontierPath, filepath.Join(frontierPath, "rounds"), filepath.Join(batchPath, ".partial"), filepath.Join(batchPath, ".partial", "frontier")} {
		if err := makePrivateDirectories(directory); err != nil {
			return nil, err
		}
	}
	plan := frontierPlan{
		Schema: frontierPlanSchema, Config: initial.Config, InitialStateSHA256: expectedIdentity,
		MaximumSegmentBytes: evidence.Uint64String(maximumSegmentBytes),
	}
	encoded, err := evidence.CanonicalJSON(plan)
	if err != nil {
		return nil, err
	}
	planPath := filepath.Join(frontierPath, "plan.json")
	if _, err := os.Lstat(planPath); err == nil {
		return nil, errors.New("frontier journal already exists")
	} else if !os.IsNotExist(err) {
		return nil, err
	}
	if err := atomicWriteContext(ctx, planPath, encoded); err != nil {
		return nil, fmt.Errorf("write frontier plan: %w", err)
	}
	if err := syncDirectoryContext(ctx, frontierPath); err != nil {
		return nil, fmt.Errorf("sync frontier plan directory: %w", err)
	}
	return &FrontierJournal{ctx: ctx, batchPath: batchPath, state: initial, maximumSegmentBytes: maximumSegmentBytes, chainSHA256: expectedIdentity}, nil
}

func ResumeFrontierJournal(ctx context.Context, batchPath string, config frontier.Config, maximumSegmentBytes uint64) (*FrontierJournal, frontier.State, uint64, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	batchPath, err := filepath.Abs(batchPath)
	if err != nil {
		return nil, frontier.State{}, 0, fmt.Errorf("resolve frontier batch path: %w", err)
	}
	if err := validatePrivateDirectory(batchPath, "frontier batch"); err != nil {
		return nil, frontier.State{}, 0, err
	}
	planBytes, err := readFrontierFile(filepath.Join(batchPath, "frontier", "plan.json"), maximumFrontierPlanBytes)
	if err != nil {
		return nil, frontier.State{}, 0, fmt.Errorf("read frontier plan: %w", err)
	}
	var plan frontierPlan
	if err := evidence.DecodeCanonicalJSON(planBytes, &plan); err != nil {
		return nil, frontier.State{}, 0, fmt.Errorf("decode frontier plan: %w", err)
	}
	if plan.Schema != frontierPlanSchema || plan.Config != config || uint64(plan.MaximumSegmentBytes) != maximumSegmentBytes || maximumSegmentBytes == 0 {
		return nil, frontier.State{}, 0, errors.New("frontier plan identity or bounds changed")
	}
	state, err := frontier.New(config)
	if err != nil {
		return nil, frontier.State{}, 0, err
	}
	initialIdentity, err := frontier.StateSHA256(state)
	if err != nil || initialIdentity != plan.InitialStateSHA256 {
		return nil, frontier.State{}, 0, errors.Join(errors.New("frontier initial state identity changed"), err)
	}
	roundsPath := filepath.Join(batchPath, "frontier", "rounds")
	if err := validatePrivateDirectory(roundsPath, "frontier rounds"); err != nil {
		return nil, frontier.State{}, 0, err
	}
	entries, err := os.ReadDir(roundsPath)
	if err != nil {
		return nil, frontier.State{}, 0, fmt.Errorf("read frontier rounds: %w", err)
	}
	if uint64(len(entries)) > config.MaxRuns {
		return nil, frontier.State{}, 0, errors.New("frontier round count exceeds its run bound")
	}
	journalRuns := []ExecutionRecord{}
	chainSHA256 := initialIdentity
	for index, entry := range entries {
		name := fmt.Sprintf("%020d", index)
		if entry.Name() != name {
			return nil, frontier.State{}, 0, fmt.Errorf("frontier round sequence has a gap at %s", name)
		}
		roundPath := filepath.Join(roundsPath, name)
		if err := validateFrontierRoundDirectory(roundPath); err != nil {
			return nil, frontier.State{}, 0, err
		}
		roundBytes, err := readFrontierFile(filepath.Join(roundPath, "round.json"), config.MaxFrontierBytes)
		if err != nil {
			return nil, frontier.State{}, 0, fmt.Errorf("read frontier round %d: %w", index, err)
		}
		var storedRound frontier.Round
		if err := evidence.DecodeCanonicalJSON(roundBytes, &storedRound); err != nil {
			return nil, frontier.State{}, 0, fmt.Errorf("decode frontier round %d: %w", index, err)
		}
		expectedRound, ok := state.NextRound()
		equal, equalErr := canonicalEqual(storedRound, expectedRound)
		if equalErr != nil {
			return nil, frontier.State{}, 0, fmt.Errorf("compare frontier round %d: %w", index, equalErr)
		}
		if !ok || !equal {
			return nil, frontier.State{}, 0, fmt.Errorf("frontier round %d does not match reconstructed state", index)
		}
		segmentBytes, err := readFrontierFile(filepath.Join(roundPath, "segment.json"), maximumSegmentBytes)
		if err != nil {
			return nil, frontier.State{}, 0, fmt.Errorf("read frontier segment %d: %w", index, err)
		}
		var segment frontier.RoundSegment
		if err := evidence.DecodeCanonicalJSON(segmentBytes, &segment); err != nil {
			return nil, frontier.State{}, 0, fmt.Errorf("decode frontier segment %d: %w", index, err)
		}
		logicalStart := state.LogicalExecutions
		state, err = frontier.ReplaySegment(state, segment)
		if err != nil {
			return nil, frontier.State{}, 0, fmt.Errorf("replay frontier segment %d: %w", index, err)
		}
		chainSHA256 = segment.SHA256
		roundRuns, err := readFrontierRoundExecutions(roundPath, len(storedRound.Candidates), maximumSegmentBytes)
		if err != nil {
			return nil, frontier.State{}, 0, fmt.Errorf("read frontier round %d run records: %w", index, err)
		}
		if len(roundRuns) != 0 {
			if err := validateFrontierRoundExecutions(storedRound, segment, roundRuns, logicalStart, config.BaseSeed); err != nil {
				return nil, frontier.State{}, 0, fmt.Errorf("validate frontier round %d run records: %w", index, err)
			}
		}
		journalRuns = append(journalRuns, roundRuns...)
	}
	recoveryExecutions, err := archiveIncompleteFrontier(batchPath, state, config.MaxFrontierBytes)
	if err != nil {
		return nil, frontier.State{}, 0, err
	}
	journal := &FrontierJournal{ctx: ctx, batchPath: batchPath, state: state, maximumSegmentBytes: maximumSegmentBytes, runs: journalRuns, chainSHA256: chainSHA256}
	return journal, state, recoveryExecutions, nil
}

func (journal *FrontierJournal) State() frontier.State {
	return journal.state
}

func (journal *FrontierJournal) StageRound(round frontier.Round) (*FrontierRoundJournal, error) {
	if journal == nil {
		return nil, errors.New("frontier journal is required")
	}
	expected, ok := journal.state.NextRound()
	equal, err := canonicalEqual(round, expected)
	if err != nil {
		return nil, err
	}
	if !ok || !equal {
		return nil, errors.New("frontier staged round does not match current state")
	}
	path := filepath.Join(journal.batchPath, ".partial", "frontier", fmt.Sprintf("%020d", round.Index))
	if _, err := os.Lstat(path); err == nil {
		return nil, errors.New("frontier round is already staged")
	} else if !os.IsNotExist(err) {
		return nil, err
	}
	if err := makePrivateDirectories(path); err != nil {
		return nil, err
	}
	if err := makePrivateDirectories(filepath.Join(path, "candidates")); err != nil {
		return nil, err
	}
	if err := makePrivateDirectories(filepath.Join(path, "attempts")); err != nil {
		return nil, err
	}
	encoded, err := evidence.CanonicalJSON(round)
	if err != nil {
		return nil, err
	}
	if uint64(len(encoded)) > journal.state.Config.MaxFrontierBytes {
		return nil, errors.New("frontier round exceeds its frontier byte capacity")
	}
	if err := atomicWriteContext(journal.ctx, filepath.Join(path, "round.json"), encoded); err != nil {
		return nil, fmt.Errorf("write staged frontier round: %w", err)
	}
	if err := syncDirectoryContext(journal.ctx, path); err != nil {
		return nil, fmt.Errorf("sync staged frontier round: %w", err)
	}
	return &FrontierRoundJournal{journal: journal, path: path, index: round.Index, runs: make([]ExecutionRecord, len(round.Candidates)), runSet: make([]bool, len(round.Candidates))}, nil
}

func (staged *FrontierRoundJournal) RecordExecution(index int, run ExecutionRecord) error {
	if staged == nil || staged.journal == nil || index < 0 || index >= len(staged.runs) {
		return errors.New("frontier run record index is invalid")
	}
	if staged.runSet[index] {
		return errors.New("frontier run record is already staged")
	}
	path := filepath.Join(staged.path, "runs")
	if err := makePrivateDirectories(path); err != nil {
		return err
	}
	encoded, err := evidence.CanonicalJSON(run)
	if err != nil {
		return err
	}
	if uint64(len(encoded)) > staged.journal.maximumSegmentBytes {
		return errors.New("frontier run record exceeds its byte capacity")
	}
	if err := atomicWriteContext(staged.journal.ctx, filepath.Join(path, fmt.Sprintf("%020d.json", index)), encoded); err != nil {
		return fmt.Errorf("write frontier run record: %w", err)
	}
	staged.runs[index] = run
	staged.runSet[index] = true
	return nil
}

func (journal *FrontierJournal) CommitRound(staged *FrontierRoundJournal, segment frontier.RoundSegment) error {
	if journal == nil || staged == nil || staged.journal != journal || staged.index != segment.Index {
		return errors.New("frontier staged round does not match its segment")
	}
	next, err := frontier.ReplaySegment(journal.state, segment)
	if err != nil {
		return fmt.Errorf("validate frontier segment before commit: %w", err)
	}
	encoded, err := evidence.CanonicalJSON(segment)
	if err != nil {
		return err
	}
	if uint64(len(encoded)) > journal.maximumSegmentBytes {
		return fmt.Errorf("frontier segment requires %d bytes, exceeding its %d-byte capacity", len(encoded), journal.maximumSegmentBytes)
	}
	if err := atomicWriteContext(journal.ctx, filepath.Join(staged.path, "segment.json"), encoded); err != nil {
		return fmt.Errorf("write staged frontier segment: %w", err)
	}
	if slices.Contains(staged.runSet, true) && slices.Contains(staged.runSet, false) {
		return errors.New("frontier round run records are incomplete")
	}
	if err := os.Remove(filepath.Join(staged.path, "candidates")); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove completed frontier candidate work: %w", err)
	}
	if err := os.RemoveAll(filepath.Join(staged.path, "attempts")); err != nil {
		return fmt.Errorf("remove completed frontier attempt records: %w", err)
	}
	if err := syncDirectoryContext(journal.ctx, staged.path); err != nil {
		return fmt.Errorf("sync staged frontier segment: %w", err)
	}
	finalPath := filepath.Join(journal.batchPath, "frontier", "rounds", fmt.Sprintf("%020d", staged.index))
	if _, err := os.Lstat(finalPath); err == nil {
		return errors.New("frontier segment is already committed")
	} else if !os.IsNotExist(err) {
		return err
	}
	if err := os.Rename(staged.path, finalPath); err != nil {
		return fmt.Errorf("publish frontier segment: %w", err)
	}
	if err := syncDirectoryContext(journal.ctx, filepath.Dir(finalPath)); err != nil {
		return fmt.Errorf("sync frontier segment sequence: %w", err)
	}
	journal.state = next
	journal.chainSHA256 = segment.SHA256
	if slices.Contains(staged.runSet, true) {
		journal.runs = append(journal.runs, staged.runs...)
	}
	return nil
}

func (journal *FrontierJournal) CommittedExecutions() []ExecutionRecord {
	if journal == nil {
		return nil
	}
	return append([]ExecutionRecord(nil), journal.runs...)
}

func (journal *FrontierJournal) ChainSHA256() evidence.SHA256 {
	if journal == nil {
		return ""
	}
	return journal.chainSHA256
}

func ValidatePublishedFrontier(batchPath string, expectedSummary frontier.Summary, expectedImplementation, expectedChain evidence.SHA256, expectedRuns []ExecutionRecord) error {
	planBytes, err := readFrontierFile(filepath.Join(batchPath, "frontier", "plan.json"), maximumFrontierPlanBytes)
	if err != nil {
		return fmt.Errorf("read published frontier plan: %w", err)
	}
	var plan frontierPlan
	if err := evidence.DecodeCanonicalJSON(planBytes, &plan); err != nil {
		return fmt.Errorf("decode published frontier plan: %w", err)
	}
	if plan.Schema != frontierPlanSchema || plan.Config.ControllerSHA256 != expectedImplementation || expectedImplementation != frontier.ImplementationSHA256() || plan.MaximumSegmentBytes == 0 {
		return errors.New("published frontier plan identity is invalid")
	}
	state, err := frontier.New(plan.Config)
	if err != nil {
		return err
	}
	initialIdentity, err := frontier.StateSHA256(state)
	if err != nil || initialIdentity != plan.InitialStateSHA256 {
		return errors.Join(errors.New("published frontier initial state identity changed"), err)
	}
	chain := initialIdentity
	roundsPath := filepath.Join(batchPath, "frontier", "rounds")
	entries, err := os.ReadDir(roundsPath)
	if err != nil {
		return err
	}
	committedRuns := []ExecutionRecord{}
	for index, entry := range entries {
		name := fmt.Sprintf("%020d", index)
		if entry.Name() != name {
			return fmt.Errorf("published frontier round sequence has a gap at %s", name)
		}
		roundPath := filepath.Join(roundsPath, name)
		if err := validateFrontierRoundDirectory(roundPath); err != nil {
			return err
		}
		roundBytes, err := readFrontierFile(filepath.Join(roundPath, "round.json"), plan.Config.MaxFrontierBytes)
		if err != nil {
			return err
		}
		var round frontier.Round
		if err := evidence.DecodeCanonicalJSON(roundBytes, &round); err != nil {
			return err
		}
		expectedRound, ok := state.NextRound()
		equal, equalErr := canonicalEqual(round, expectedRound)
		if equalErr != nil || !ok || !equal {
			return errors.Join(fmt.Errorf("published frontier round %d does not match its state", index), equalErr)
		}
		segmentBytes, err := readFrontierFile(filepath.Join(roundPath, "segment.json"), uint64(plan.MaximumSegmentBytes))
		if err != nil {
			return err
		}
		var segment frontier.RoundSegment
		if err := evidence.DecodeCanonicalJSON(segmentBytes, &segment); err != nil {
			return err
		}
		logicalStart := state.LogicalExecutions
		state, err = frontier.ReplaySegment(state, segment)
		if err != nil {
			return err
		}
		runs, err := readFrontierRoundExecutions(roundPath, len(round.Candidates), uint64(plan.MaximumSegmentBytes))
		if err != nil || len(runs) == 0 {
			return errors.Join(errors.New("published frontier round run records are unavailable"), err)
		}
		if err := validateFrontierRoundExecutions(round, segment, runs, logicalStart, plan.Config.BaseSeed); err != nil {
			return err
		}
		committedRuns = append(committedRuns, runs...)
		chain = segment.SHA256
	}
	equal, err := canonicalEqual(state.Summary(), expectedSummary)
	if err != nil || !equal || chain != expectedChain {
		return errors.Join(errors.New("published frontier summary or chain does not match its batch"), err)
	}
	if len(committedRuns) != len(expectedRuns) {
		return errors.New("published frontier run projection count does not match its batch")
	}
	for index := range committedRuns {
		equal, err := canonicalEqual(committedRuns[index], expectedRuns[index])
		if err != nil || !equal {
			return errors.Join(fmt.Errorf("published frontier run projection diverges at ordinal %d", index), err)
		}
	}
	partialEntries, err := os.ReadDir(filepath.Join(batchPath, ".partial", "frontier"))
	if err != nil || len(partialEntries) != 0 {
		return errors.Join(errors.New("published frontier retains incomplete round state"), err)
	}
	return nil
}

func (staged *FrontierRoundJournal) Path() string {
	if staged == nil {
		return ""
	}
	return staged.path
}

func (staged *FrontierRoundJournal) BeginExecution(ordinal, seed uint64) (*ExecutionJournal, error) {
	if staged == nil || staged.journal == nil {
		return nil, errors.New("frontier staged round is required")
	}
	attempt, err := evidence.CanonicalJSON(frontierAttempt{Ordinal: evidence.Uint64String(ordinal), Seed: evidence.Uint64String(seed)})
	if err != nil {
		return nil, err
	}
	if err := atomicWriteContext(staged.journal.ctx, filepath.Join(staged.path, "attempts", fmt.Sprintf("%020d.json", ordinal)), attempt); err != nil {
		return nil, fmt.Errorf("record frontier candidate attempt: %w", err)
	}
	path := filepath.Join(staged.path, "candidates", fmt.Sprintf("%020d", ordinal))
	if err := makePrivateDirectories(path); err != nil {
		return nil, err
	}
	run := &ExecutionJournal{path: path, ordinal: ordinal, seed: seed, state: ExecutionStaging}
	if err := run.writeState(ExecutionStaging); err != nil {
		return run, err
	}
	if err := makePrivateDirectories(run.WorkPath()); err != nil {
		return run, err
	}
	return run, nil
}

func archiveIncompleteFrontier(batchPath string, state frontier.State, maximumRoundBytes uint64) (uint64, error) {
	partialRoot := filepath.Join(batchPath, ".partial")
	frontierRoot := filepath.Join(partialRoot, "frontier")
	if err := validatePrivateDirectory(frontierRoot, "partial frontier"); err != nil {
		return 0, err
	}
	entries, err := os.ReadDir(frontierRoot)
	if err != nil {
		return 0, fmt.Errorf("read partial frontier: %w", err)
	}
	if len(entries) == 0 {
		return 0, nil
	}
	if len(entries) != 1 {
		return 0, errors.New("partial frontier contains multiple rounds")
	}
	entry := entries[0]
	if entry.Name() != fmt.Sprintf("%020d", state.CommittedRounds) {
		return 0, errors.New("partial frontier round does not follow committed state")
	}
	path := filepath.Join(frontierRoot, entry.Name())
	if err := validatePrivateDirectory(path, "partial frontier round"); err != nil {
		return 0, err
	}
	roundBytes, err := readFrontierFile(filepath.Join(path, "round.json"), maximumRoundBytes)
	if err != nil {
		return 0, fmt.Errorf("read partial frontier round: %w", err)
	}
	var round frontier.Round
	if err := evidence.DecodeCanonicalJSON(roundBytes, &round); err != nil {
		return 0, fmt.Errorf("decode partial frontier round: %w", err)
	}
	expected, ok := state.NextRound()
	equal, err := canonicalEqual(round, expected)
	if err != nil {
		return 0, err
	}
	if !ok || !equal {
		return 0, errors.New("partial frontier round does not match committed state")
	}
	attempts, err := countFrontierAttempts(path, state, round)
	if err != nil {
		return 0, err
	}
	resumeRoot := filepath.Join(partialRoot, "resume")
	if err := makePrivateDirectories(resumeRoot); err != nil {
		return 0, err
	}
	attempt, err := nextResumeAttempt(resumeRoot)
	if err != nil {
		return 0, err
	}
	destinationRoot := filepath.Join(resumeRoot, fmt.Sprintf("%06d", attempt), "partials", "frontier")
	if err := makePrivateDirectories(destinationRoot); err != nil {
		return 0, err
	}
	if err := os.Rename(path, filepath.Join(destinationRoot, entry.Name())); err != nil {
		return 0, fmt.Errorf("archive partial frontier round: %w", err)
	}
	if err := syncDirectory(partialRoot); err != nil {
		return 0, fmt.Errorf("sync partial frontier archive: %w", err)
	}
	return attempts, nil
}

func countFrontierAttempts(roundPath string, state frontier.State, round frontier.Round) (uint64, error) {
	attemptsPath := filepath.Join(roundPath, "attempts")
	if err := validatePrivateDirectory(attemptsPath, "partial frontier attempts"); err != nil {
		return 0, err
	}
	entries, err := os.ReadDir(attemptsPath)
	if err != nil {
		return 0, err
	}
	if len(entries) > len(round.Candidates) {
		return 0, errors.New("partial frontier contains too many candidate attempts")
	}
	for index, entry := range entries {
		ordinal := state.LogicalExecutions + uint64(index)
		name := fmt.Sprintf("%020d.json", ordinal)
		if entry.Name() != name || entry.IsDir() {
			return 0, errors.New("partial frontier candidate attempts are not contiguous")
		}
		contents, err := readFrontierFile(filepath.Join(attemptsPath, name), maximumFrontierPlanBytes)
		if err != nil {
			return 0, err
		}
		var attempt frontierAttempt
		if err := evidence.DecodeCanonicalJSON(contents, &attempt); err != nil || attempt.Ordinal != evidence.Uint64String(ordinal) || attempt.Seed != evidence.Uint64String(state.Config.BaseSeed) {
			return 0, errors.Join(errors.New("partial frontier candidate attempt is invalid"), err)
		}
	}
	return uint64(len(entries)), nil
}

func validateFrontierRoundDirectory(path string) error {
	if err := validatePrivateDirectory(path, "frontier round"); err != nil {
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
		case "runs", "failures", "successes":
			if !entry.IsDir() {
				return errors.New("frontier round artifact entry is not a directory")
			}
		default:
			return errors.New("frontier round contains unexpected entries")
		}
	}
	if !seenRound || !seenSegment {
		return errors.New("frontier round is incomplete")
	}
	return nil
}

func readFrontierRoundExecutions(roundPath string, count int, maximumBytes uint64) ([]ExecutionRecord, error) {
	runsPath := filepath.Join(roundPath, "runs")
	if _, err := os.Lstat(runsPath); os.IsNotExist(err) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	if err := validatePrivateDirectory(runsPath, "frontier round runs"); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(runsPath)
	if err != nil {
		return nil, err
	}
	if len(entries) != count {
		return nil, errors.New("frontier round run record count does not match its candidates")
	}
	runs := make([]ExecutionRecord, count)
	var total uint64
	for index, entry := range entries {
		name := fmt.Sprintf("%020d.json", index)
		if entry.Name() != name || entry.IsDir() {
			return nil, errors.New("frontier round run record sequence is invalid")
		}
		contents, err := readFrontierFile(filepath.Join(runsPath, name), maximumBytes-total)
		if err != nil {
			return nil, err
		}
		total += uint64(len(contents))
		if err := evidence.DecodeCanonicalJSON(contents, &runs[index]); err != nil {
			return nil, err
		}
	}
	return runs, nil
}

func validateFrontierRoundExecutions(round frontier.Round, segment frontier.RoundSegment, runs []ExecutionRecord, logicalStart, baseSeed uint64) error {
	if len(runs) != len(round.Candidates) || len(runs) != len(segment.Results) {
		return errors.New("frontier round run records do not match its results")
	}
	candidates := make(map[evidence.SHA256]struct{}, len(runs))
	for index, run := range runs {
		candidate := round.Candidates[index]
		result := segment.Results[index]
		if run.SelectionOrdinal != evidence.Uint64String(logicalStart+uint64(index)) || run.Seed != evidence.Uint64String(baseSeed) || run.Round == nil || *run.Round != evidence.Uint64String(round.Index) || run.CandidateSHA256 != candidate.SHA256 || run.ParentCandidateSHA256 != candidate.ParentSHA256 || run.PrefixSHA256 != candidate.PrefixSHA256 || run.ForcedDepth == nil || *run.ForcedDepth != evidence.Uint64String(candidate.ForcedDepth) || run.OutcomeSHA256 != result.OutcomeSHA256 {
			return fmt.Errorf("frontier run %d provenance does not match its segment", index)
		}
		if err := validateFrontierExecutionSummary(run, candidates); err != nil {
			return err
		}
		if result.Failed != (run.Domain == "target" || run.Domain == "watchdog") || result.Failed && (run.FailureSignature == nil || *run.FailureSignature != result.FailureSHA256) {
			return fmt.Errorf("frontier run %d outcome does not match its segment", index)
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

func readFrontierFile(path string, maximum uint64) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm() != 0o600 || info.Size() < 0 || uint64(info.Size()) > maximum {
		return nil, errors.Join(errors.New("frontier file metadata or capacity is invalid"), err)
	}
	contents, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if uint64(len(contents)) != uint64(info.Size()) {
		return nil, errors.New("frontier file changed while reading")
	}
	return contents, nil
}

func canonicalEqual(left, right any) (bool, error) {
	leftBytes, err := evidence.CanonicalJSON(left)
	if err != nil {
		return false, err
	}
	rightBytes, err := evidence.CanonicalJSON(right)
	if err != nil {
		return false, err
	}
	return string(leftBytes) == string(rightBytes), nil
}
