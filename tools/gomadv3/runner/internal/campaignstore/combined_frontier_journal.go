package campaignstore

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"go.temporal.io/server/tools/gomadv3/evidence"
	"go.temporal.io/server/tools/gomadv3/runner/internal/combinedfrontier"
)

const (
	combinedFrontierPlanSchema       = "gomadv3.combined-frontier-plan/v1"
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
	chainSHA256         evidence.SHA256
}

type CombinedFrontierRoundJournal struct {
	journal *CombinedFrontierJournal
	path    string
	index   uint64
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
		state, err = combinedfrontier.ReplaySegment(state, segment)
		if err != nil {
			return nil, combinedfrontier.State{}, 0, fmt.Errorf("replay combined frontier segment %d: %w", index, err)
		}
		chainSHA256 = segment.SHA256
	}
	recoveryExecutions, err := discardIncompleteCombinedFrontierRound(ctx, batchPath, state)
	if err != nil {
		return nil, combinedfrontier.State{}, 0, err
	}
	journal := &CombinedFrontierJournal{
		ctx: ctx, batchPath: batchPath, state: state, maximumSegmentBytes: maximumSegmentBytes, chainSHA256: chainSHA256,
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
	return &CombinedFrontierRoundJournal{journal: journal, path: path, index: round.Index}, nil
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
	return nil
}

func (staged *CombinedFrontierRoundJournal) Path() string {
	if staged == nil {
		return ""
	}
	return staged.path
}

func validateCombinedFrontierRoundDirectory(path string, complete bool) error {
	if err := validatePrivateDirectory(path, "combined frontier round"); err != nil {
		return err
	}
	entries, err := os.ReadDir(path)
	if err != nil {
		return err
	}
	want := 1
	if complete {
		want = 2
	}
	if len(entries) != want || entries[0].Name() != "round.json" || complete && entries[1].Name() != "segment.json" {
		return errors.New("combined frontier round directory contents are invalid")
	}
	return nil
}

func discardIncompleteCombinedFrontierRound(ctx context.Context, batchPath string, state combinedfrontier.State) (uint64, error) {
	partialPath := filepath.Join(batchPath, ".partial", "combined-frontier")
	if err := validatePrivateDirectory(partialPath, "partial combined frontier"); err != nil {
		return 0, err
	}
	entries, err := os.ReadDir(partialPath)
	if err != nil {
		return 0, err
	}
	if len(entries) == 0 {
		return 0, nil
	}
	if len(entries) != 1 {
		return 0, errors.New("partial combined frontier contains multiple rounds")
	}
	expected, ok := state.NextRound()
	name := fmt.Sprintf("%020d", expected.Index)
	if !ok || entries[0].Name() != name {
		return 0, errors.New("partial combined frontier round does not match reconstructed state")
	}
	path := filepath.Join(partialPath, name)
	if err := validatePrivateDirectory(path, "partial combined frontier round"); err != nil {
		return 0, err
	}
	roundBytes, err := readFrontierFile(filepath.Join(path, "round.json"), state.Config.MaxFrontierBytes)
	if err != nil {
		return 0, err
	}
	var round combinedfrontier.Round
	if err := evidence.DecodeCanonicalJSON(roundBytes, &round); err != nil {
		return 0, err
	}
	equal, err := canonicalEqual(round, expected)
	if err != nil || !equal {
		return 0, errors.Join(errors.New("partial combined frontier round changed"), err)
	}
	if err := removeCompletedPartialContext(ctx, path, "combined-frontier-incomplete-round"); err != nil {
		return 0, err
	}
	if err := syncDirectoryContext(ctx, partialPath); err != nil {
		return 0, err
	}
	return uint64(len(round.Candidates)), nil
}
