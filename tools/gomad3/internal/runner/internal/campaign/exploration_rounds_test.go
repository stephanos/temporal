package campaign

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomad3/internal/evidence/record"
	"go.temporal.io/server/tools/gomad3/internal/model/choice"
	"go.temporal.io/server/tools/gomad3/internal/runner/internal/exploration"
	"go.temporal.io/server/tools/gomad3/internal/runner/internal/logstore"
)

// inspectRounds reads a campaign's exploration rounds through the read-only
// store inspection; it reports false for a seed campaign.
func inspectRounds(path string) (ExplorationInspection, bool, error) {
	inspected, err := InspectCampaignStore(path)
	if err != nil || inspected.Exploration == nil {
		return ExplorationInspection{}, false, err
	}
	return *inspected.Exploration, true, nil
}

func privateDirectory(t *testing.T) string {
	t.Helper()
	path := t.TempDir()
	if err := os.Chmod(path, 0o700); err != nil {
		t.Fatal(err)
	}
	return path
}

// copyRetainedRecords copies a testdata directory into a private directory with
// the modes the Runner publishes records with, which a checkout does not keep.
func copyRetainedRecords(t *testing.T, name string) string {
	t.Helper()
	batchPath := privateDirectory(t)
	source := filepath.Join("testdata", name)
	err := filepath.WalkDir(source, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(source, path)
		if err != nil || relative == "." {
			return err
		}
		destination := filepath.Join(batchPath, relative)
		if entry.IsDir() {
			return os.Mkdir(destination, 0o700)
		}
		contents, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(destination, contents, 0o600)
	})
	if err != nil {
		t.Fatal(err)
	}
	return batchPath
}

func testExplorationState(t *testing.T) exploration.ChoiceState {
	t.Helper()
	state, err := exploration.NewChoice(exploration.ChoiceConfig{
		Execution: choice.ExecutionIdentity{
			TargetSHA256: sha256.Sum256([]byte("target")), ToolchainBuildKey: strings.Repeat("a", 64),
			GOOS: "darwin", GOARCH: "arm64", ImplementationSHA256: sha256.Sum256([]byte("controller")),
		},
		ControllerSHA256: exploration.ChoiceImplementationSHA256(),
		BaseSeed:         7, Parallel: 2, MaxExecutions: 8, MaxChoiceDepth: 4, MaxExplorationBytes: 1 << 20,
		FailurePolicy: exploration.PolicyAll, FailureBudget: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	return state
}

func testSimulationExplorationState(t *testing.T) exploration.SimulationState {
	t.Helper()
	state, err := exploration.NewSimulation(exploration.SimulationConfig{
		ExecutionSHA256: record.HashBytes([]byte("execution")), ControllerSHA256: exploration.SimulationImplementationSHA256(),
		BaseSeed: 7, Parallel: 2, MaxExecutions: 8, MaxForcedDecisions: 4, MaxExplorationBytes: 1 << 20,
		MaxResultBytes: 1 << 20, FailureBudget: 4,
		Limits: exploration.DimensionLimits{Runtime: 4, Scenario: 4, Network: 4, Storage: 4, Fault: 4, Crash: 4},
	})
	if err != nil {
		t.Fatal(err)
	}
	return state
}

// newExplorationCampaign creates a running exploration campaign of strategy
// with a valid frozen plan; ctx carries an optional mutation hook.
func newExplorationCampaign(t *testing.T, ctx context.Context, strategy string) *CampaignJournal {
	t.Helper()
	journal, err := NewCampaignJournal(ctx, CampaignConfig{
		Root: t.TempDir(), CampaignID: "run-" + strategy, Strategy: strategy, Selection: "7", SelectionCount: 1, MaxExecutions: 8, Parallel: 2,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := journal.Close(); err != nil {
			t.Error(err)
		}
	})
	if err := journal.BeginPreparation(); err != nil {
		t.Fatal(err)
	}
	preparedPath := filepath.Join(journal.PreparedPath(), "build", "target")
	if err := os.MkdirAll(filepath.Dir(preparedPath), 0o700); err != nil {
		t.Fatal(err)
	}
	target := []byte("prepared target")
	if err := os.WriteFile(preparedPath, target, 0o500); err != nil {
		t.Fatal(err)
	}
	if err := journal.RecordPlan(testExplorationCampaignPlan(t, journal, strategy, record.HashBytes(target), uint64(len(target)))); err != nil {
		t.Fatal(err)
	}
	if err := journal.StartExecutions(); err != nil {
		t.Fatal(err)
	}
	return journal
}

func testExplorationCampaignPlan(t *testing.T, journal *CampaignJournal, strategy string, digest record.SHA256, size uint64) CampaignPlan {
	t.Helper()
	plan := testBatchPlan(journal, digest, size)
	plan.Strategy, plan.Selection, plan.SelectionCount, plan.MaxExecutions, plan.MaxExplorationBytes = strategy, "7", 1, 8, 1<<20
	plan.Toolchain.BuildKey = strings.Repeat("a", 64)
	implementation, err := choice.ImplementationIdentity(plan.Toolchain.BuildKey)
	if err != nil {
		t.Fatal(err)
	}
	plan.ChoiceProfile = &ChoiceProfilePlan{Name: choice.Profile, ImplementationSHA256: record.SHA256FromSum(implementation), Limit: record.Uint64String(choice.MinimumTraceBytes)}
	if strategy == "choice-exploration" {
		plan.MaxChoiceDepth, plan.ChoiceExplorationImplementationSHA256 = 4, exploration.ChoiceImplementationSHA256()
	} else {
		plan.MaxForcedDecisions, plan.MaxExplorationResultBytes, plan.SimulationExplorationImplementationSHA256 = 4, 1<<20, exploration.SimulationImplementationSHA256()
		plan.SimulationDimensionLimits = exploration.DimensionLimits{Runtime: 4, Scenario: 4, Network: 4, Storage: 4, Fault: 4, Crash: 4}
	}
	artifacts, err := DeriveArtifactCapacityPlan(plan)
	if err != nil {
		t.Fatal(err)
	}
	plan.Artifacts = &artifacts
	return plan
}

// explorationScript drives one strategy's rounds deterministically through
// the round owner of a campaign.
type explorationScript struct {
	strategy string
	// open binds the campaign's rounds.
	open func(*CampaignJournal) error
	// step starts and commits the next round, or reports there is none.
	// interrupt stops after the round start.
	step func(t *testing.T, journal *CampaignJournal, interrupt bool) (bool, error)
	// identity is the replayed engine state's identity.
	identity func(t *testing.T, journal *CampaignJournal) record.SHA256
}

func explorationScripts(t *testing.T) []explorationScript {
	choiceConfig := testExplorationState(t).Config
	simulationConfig := testSimulationExplorationState(t).Config
	return []explorationScript{
		{
			strategy: "choice-exploration",
			open: func(journal *CampaignJournal) error {
				_, err := OpenRounds(journal, ChoiceRounds, choiceConfig, 1<<20)
				return err
			},
			step: func(t *testing.T, journal *CampaignJournal, interrupt bool) (bool, error) {
				rounds, err := OpenRounds(journal, ChoiceRounds, choiceConfig, 1<<20)
				if err != nil {
					return false, err
				}
				state := rounds.State()
				round, available := state.NextRound()
				if !available {
					return false, nil
				}
				if err := rounds.Start(round); err != nil || interrupt {
					return true, err
				}
				results, runs := choiceRoundOutcomes(t, state, round)
				_, segment, err := exploration.CommitChoiceRound(state, round, results)
				if err != nil {
					t.Fatal(err)
				}
				return true, rounds.Commit(round, segment, runs)
			},
			identity: func(t *testing.T, journal *CampaignJournal) record.SHA256 {
				rounds, err := OpenRounds(journal, ChoiceRounds, choiceConfig, 1<<20)
				if err != nil {
					t.Fatal(err)
				}
				identity, err := exploration.ChoiceStateSHA256(rounds.State())
				if err != nil {
					t.Fatal(err)
				}
				return identity
			},
		},
		{
			strategy: "simulation-exploration",
			open: func(journal *CampaignJournal) error {
				_, err := OpenRounds(journal, SimulationRounds, simulationConfig, 1<<20)
				return err
			},
			step: func(t *testing.T, journal *CampaignJournal, interrupt bool) (bool, error) {
				rounds, err := OpenRounds(journal, SimulationRounds, simulationConfig, 1<<20)
				if err != nil {
					return false, err
				}
				state := rounds.State()
				round, available := state.NextRound()
				if !available {
					return false, nil
				}
				if err := rounds.Start(round); err != nil || interrupt {
					return true, err
				}
				results, runs := simulationRoundOutcomes(state, round)
				_, segment, err := exploration.CommitSimulationRound(state, round, results)
				if err != nil {
					t.Fatal(err)
				}
				return true, rounds.Commit(round, segment, runs)
			},
			identity: func(t *testing.T, journal *CampaignJournal) record.SHA256 {
				rounds, err := OpenRounds(journal, SimulationRounds, simulationConfig, 1<<20)
				if err != nil {
					t.Fatal(err)
				}
				identity, err := exploration.SimulationStateSHA256(rounds.State())
				if err != nil {
					t.Fatal(err)
				}
				return identity
			},
		},
	}
}

// choiceRoundOutcomes succeeds every candidate. The root records one
// two-way decision, so the next round forces its other alternative; every
// later candidate records none.
func choiceRoundOutcomes(t *testing.T, state exploration.ChoiceState, round exploration.ChoiceRound) ([]exploration.ChoiceResult, []ExecutionRecord) {
	t.Helper()
	results := make([]exploration.ChoiceResult, len(round.Candidates))
	runs := make([]ExecutionRecord, len(round.Candidates))
	for index, candidate := range round.Candidates {
		outcome := record.HashBytes([]byte(fmt.Sprintf("outcome %d", state.LogicalExecutions+uint64(index))))
		results[index] = exploration.ChoiceResult{CandidateSHA256: candidate.SHA256, OutcomeSHA256: outcome}
		if candidate.ForcedDepth == 0 {
			identities := [][32]byte{sha256.Sum256([]byte("first")), sha256.Sum256([]byte("second"))}
			decision, err := choice.CanonicalDecision(0, choice.KindRunnable, 1, false, identities, identities[0], 0)
			if err != nil {
				t.Fatal(err)
			}
			trace, err := choice.BuildTrace([]choice.Record{decision.Record()}, choice.TerminalComplete)
			if err != nil {
				t.Fatal(err)
			}
			tape, err := choice.ProjectReplayPlan(trace, state.Config.Execution)
			if err != nil {
				t.Fatal(err)
			}
			results[index].Trace = &tape
		}
		roundValue, depth := record.Uint64String(round.Index), record.Uint64String(candidate.ForcedDepth)
		runs[index] = ExecutionRecord{
			Strategy: "choice-exploration", Round: &roundValue, CandidateSHA256: candidate.SHA256, ParentCandidateSHA256: candidate.ParentSHA256,
			PrefixSHA256: candidate.PrefixSHA256, ForcedDepth: &depth, OutcomeSHA256: outcome,
			SelectionOrdinal: record.Uint64String(state.LogicalExecutions + uint64(index)), Seed: 7, Domain: "success", Reason: "success", Termination: "exit",
		}
	}
	return results, runs
}

func simulationRoundOutcomes(state exploration.SimulationState, round exploration.SimulationRound) ([]exploration.SimulationResult, []ExecutionRecord) {
	results := make([]exploration.SimulationResult, len(round.Candidates))
	runs := make([]ExecutionRecord, len(round.Candidates))
	for index, candidate := range round.Candidates {
		outcome := record.HashBytes([]byte(fmt.Sprintf("outcome %d", state.LogicalExecutions+uint64(index))))
		results[index] = exploration.SimulationResult{CandidateSHA256: candidate.SHA256, OutcomeSHA256: outcome}
		roundValue, depth := record.Uint64String(round.Index), record.Uint64String(len(candidate.Overrides))
		runs[index] = ExecutionRecord{
			Strategy: "simulation-exploration", Round: &roundValue, CandidateSHA256: candidate.SHA256, ParentCandidateSHA256: candidate.ParentSHA256,
			ForcedDepth: &depth, OutcomeSHA256: outcome, SelectionOrdinal: record.Uint64String(state.LogicalExecutions + uint64(index)),
			Seed: 7, Domain: "success", Reason: "exit_zero", Termination: "exit",
		}
	}
	return results, runs
}

func runScript(t *testing.T, script explorationScript, journal *CampaignJournal) error {
	t.Helper()
	for {
		more, err := script.step(t, journal, false)
		if err != nil || !more {
			return err
		}
	}
}

// Both strategies commit whole rounds through the one round owner: each round
// is one log transaction holding its round, segment and every execution, and
// replay rebuilds the exact engine state and outcomes.
func TestExplorationRoundsCommitAndReplayWholeRounds(t *testing.T) {
	for _, script := range explorationScripts(t) {
		t.Run(script.strategy, func(t *testing.T) {
			journal := newExplorationCampaign(t, context.Background(), script.strategy)
			if err := runScript(t, script, journal); err != nil {
				t.Fatal(err)
			}
			want := script.identity(t, journal)
			if err := journal.Close(); err != nil {
				t.Fatal(err)
			}
			kinds, _ := campaignLogRecords(t, journal.Path())
			var transactions []logstore.Kind
			for _, kind := range kinds {
				if kind >= kindOutcome {
					transactions = append(transactions, kind)
				}
			}
			inspected, ok, err := inspectRounds(journal.Path())
			if err != nil || !ok {
				t.Fatalf("inspectRounds() = %v, %v", ok, err)
			}
			wantTransactions := []logstore.Kind{kindExploration}
			for range inspected.Rounds {
				wantTransactions = append(wantTransactions, kindRoundStart, kindRound)
			}
			if !slices.Equal(transactions, wantTransactions) {
				t.Fatalf("log transactions = %v, want %v", transactions, wantTransactions)
			}
			state, err := ReadCommittedState(journal.Path())
			if err != nil {
				t.Fatal(err)
			}
			var executions uint64
			for index, round := range inspected.Rounds {
				if round.Index != uint64(index) || round.FirstOrdinal != executions || round.RoundBytes == 0 || !validRecordSHA256(round.RoundSHA256) {
					t.Fatalf("committed round %d = %#v", index, round)
				}
				executions += round.Executions
			}
			if uint64(len(state.Executions)) != executions || inspected.StagedRound != nil || inspected.RecoveryExecutions != 0 || inspected.Strategy != script.strategy {
				t.Fatalf("inspection = %#v, executions = %d", inspected, len(state.Executions))
			}
			resumed, resumeState, err := ResumeCampaignJournal(context.Background(), journal.Path())
			if err != nil {
				t.Fatal(err)
			}
			defer func() {
				if err := resumed.Close(); err != nil {
					t.Error(err)
				}
			}()
			if got := script.identity(t, resumed); got != want || !reflect.DeepEqual(resumeState.Executions, state.Executions) {
				t.Fatalf("resumed identity = %s, want %s", got, want)
			}
			if more, err := script.step(t, resumed, false); err != nil || more {
				t.Fatalf("resumed campaign has another round: %t, %v", more, err)
			}
		})
	}
}

// A round that started but did not commit is discarded when the campaign
// resumes: its candidates count as recovery executions, cumulatively, and
// the round runs again in full.
func TestExplorationRoundsDiscardAnUncommittedRoundOnResume(t *testing.T) {
	for _, script := range explorationScripts(t) {
		t.Run(script.strategy, func(t *testing.T) {
			journal := newExplorationCampaign(t, context.Background(), script.strategy)
			if _, err := script.step(t, journal, true); err != nil {
				t.Fatal(err)
			}
			if err := journal.Close(); err != nil {
				t.Fatal(err)
			}
			inspected, _, err := inspectRounds(journal.Path())
			if err != nil || inspected.StagedRound == nil || inspected.StagedRound.Index != 0 || inspected.StagedRound.Candidates != 1 {
				t.Fatalf("staged inspection = %#v, %v", inspected, err)
			}
			for attempt := uint64(1); attempt <= 2; attempt++ {
				resumed, _, err := ResumeCampaignJournal(context.Background(), journal.Path())
				if err != nil {
					t.Fatal(err)
				}
				inspected, _, err = inspectRounds(journal.Path())
				if err != nil || inspected.StagedRound != nil || inspected.RecoveryExecutions != attempt {
					t.Fatalf("resume %d inspection = %#v, %v", attempt, inspected, err)
				}
				if _, err := script.step(t, resumed, attempt == 1); err != nil {
					t.Fatal(err)
				}
				if attempt == 2 {
					if err := runScript(t, script, resumed); err != nil {
						t.Fatal(err)
					}
				}
				if err := resumed.Close(); err != nil {
					t.Fatal(err)
				}
			}
			reference := newExplorationCampaign(t, context.Background(), script.strategy)
			if err := runScript(t, script, reference); err != nil {
				t.Fatal(err)
			}
			resumed, _, err := ResumeCampaignJournal(context.Background(), journal.Path())
			if err != nil {
				t.Fatal(err)
			}
			defer func() {
				if err := resumed.Close(); err != nil {
					t.Error(err)
				}
			}()
			if script.identity(t, resumed) != script.identity(t, reference) {
				t.Fatal("resumed exploration differs from the uninterrupted one")
			}
		})
	}
}

// Every campaign log append of an exploration, from its exploration record
// through its last round, fails in turn at each of its boundaries. The
// writer refuses further appends; resume decides from the durable log, and
// running the rounds to completion reaches the uninterrupted state with
// every logical ordinal committed once.
func TestExplorationRoundFaultsResumeToTheUninterruptedState(t *testing.T) {
	for _, script := range explorationScripts(t) {
		t.Run(script.strategy, func(t *testing.T) {
			reference := newExplorationCampaign(t, context.Background(), script.strategy)
			if err := runScript(t, script, reference); err != nil {
				t.Fatal(err)
			}
			want := script.identity(t, reference)
			referenceState, err := ReadCommittedState(reference.Path())
			if err != nil {
				t.Fatal(err)
			}
			kinds, _ := campaignLogRecords(t, reference.Path())
			first := slices.Index(kinds, kindExploration) + 1
			for index := first; index <= len(kinds); index++ {
				for _, mode := range []string{"write", "partial", "sync", "synced"} {
					t.Run(fmt.Sprintf("append %d %s", index, mode), func(t *testing.T) {
						appends, late, injected := 0, 0, false
						ctx := withMutationHook(context.Background(), func(point mutationPoint) error {
							if point.Operation != "append" {
								return nil
							}
							if point.Kind == "log-write" {
								if injected {
									late++
								}
								appends++
							}
							if appends != index || injected {
								return nil
							}
							switch {
							case point.Kind == "log-write" && mode == "write", point.Kind == "log-sync" && mode == "sync", point.Kind == "log-synced" && mode == "synced":
								injected = true
								return errors.New("injected round append failure")
							case point.Kind == "log-write" && mode == "partial":
								injected = true
								return &logstore.PartialWrite{Bytes: 11}
							}
							return nil
						})
						journal := newExplorationCampaign(t, ctx, script.strategy)
						err := script.open(journal)
						if err == nil {
							err = runScript(t, script, journal)
						}
						if err == nil || !injected {
							t.Fatalf("faulted exploration = %v, injected %t", err, injected)
						}
						if late != 0 {
							t.Fatalf("%d appends followed the failed append", late)
						}
						if err := journal.Close(); err != nil {
							t.Fatal(err)
						}
						resumed, _, err := ResumeCampaignJournal(context.Background(), journal.Path())
						if err != nil {
							t.Fatal(err)
						}
						defer func() {
							if err := resumed.Close(); err != nil {
								t.Error(err)
							}
						}()
						if err := runScript(t, script, resumed); err != nil {
							t.Fatal(err)
						}
						state, err := ReadCommittedState(journal.Path())
						if err != nil {
							t.Fatal(err)
						}
						if script.identity(t, resumed) != want || !reflect.DeepEqual(state.Executions, referenceState.Executions) {
							t.Fatalf("resumed exploration differs: executions %d, want %d", len(state.Executions), len(referenceState.Executions))
						}
					})
				}
			}
		})
	}
}

// The round owner refuses an exploration that changed its frozen
// configuration or segment bound, rounds out of order or beyond their
// bounds, executions that do not match their round's segment and a seed
// outcome, all before appending anything.
func TestExplorationRoundsRejectInvalidTransactionsWithoutAppending(t *testing.T) {
	state := testExplorationState(t)
	journal := newExplorationCampaign(t, context.Background(), "choice-exploration")
	rounds, err := OpenRounds(journal, ChoiceRounds, state.Config, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	changed := state.Config
	changed.StartOrdinal = 1
	for _, open := range []func() error{
		func() error { _, err := OpenRounds(journal, ChoiceRounds, changed, 1<<20); return err },
		func() error { _, err := OpenRounds(journal, ChoiceRounds, state.Config, 1<<19); return err },
	} {
		if err := open(); err == nil || !strings.Contains(err.Error(), "exploration plan identity or bounds changed") {
			t.Fatalf("changed exploration error = %v", err)
		}
	}
	if _, err := OpenRounds(journal, SimulationRounds, testSimulationExplorationState(t).Config, 1<<20); err == nil {
		t.Fatal("choice campaign opened simulation rounds")
	}
	round, _ := state.NextRound()
	results, runs := choiceRoundOutcomes(t, state, round)
	_, segment, err := exploration.CommitChoiceRound(state, round, results)
	if err != nil {
		t.Fatal(err)
	}
	before, _ := campaignLogRecords(t, journal.Path())
	if err := rounds.Commit(round, segment, runs); err == nil || !strings.Contains(err.Error(), "without its round start") {
		t.Fatalf("unstarted round commit error = %v", err)
	}
	if err := rounds.Start(round); err != nil {
		t.Fatal(err)
	}
	if err := rounds.Start(round); err == nil {
		t.Fatal("round started twice")
	}
	changedOutcome := slices.Clone(runs)
	changedOutcome[0].OutcomeSHA256 = record.HashBytes([]byte("changed"))
	changedOrdinal := slices.Clone(runs)
	changedOrdinal[0].SelectionOrdinal = 1
	divergent := slices.Clone(runs)
	divergent[0].Domain = "target"
	for name, invalid := range map[string][]ExecutionRecord{"missing": nil, "outcome": changedOutcome, "ordinal": changedOrdinal, "classification": divergent} {
		if err := rounds.Commit(round, segment, invalid); err == nil {
			t.Fatalf("%s executions committed", name)
		}
	}
	if err := journal.AppendExecution(runs[0]); err == nil || !strings.Contains(err.Error(), "only in rounds") {
		t.Fatalf("exploration seed outcome error = %v", err)
	}
	after, _ := campaignLogRecords(t, journal.Path())
	if len(after) != len(before)+1 {
		t.Fatalf("rejected transactions appended: %v -> %v", before, after)
	}
	if err := rounds.Commit(round, segment, runs); err != nil {
		t.Fatalf("valid round after rejected transactions: %v", err)
	}
}

// A round and its segment are bounded by their component envelopes: the
// round by the exploration byte bound, the segment by its recorded bound.
// An oversized transaction fails before it is written and leaves the writer
// usable.
func TestExplorationRoundsBoundTheirTransactionBeforeWriting(t *testing.T) {
	state := testExplorationState(t)
	round, _ := state.NextRound()
	encodedRound, err := logstore.EncodeJSON(round)
	if err != nil {
		t.Fatal(err)
	}
	results, runs := choiceRoundOutcomes(t, state, round)
	_, segment, err := exploration.CommitChoiceRound(state, round, results)
	if err != nil {
		t.Fatal(err)
	}
	encodedSegment, err := logstore.EncodeJSON(segment)
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name         string
		roundBytes   uint64
		segmentBytes uint64
		want         string
	}{
		{name: "round", roundBytes: uint64(len(encodedRound)) - 1, segmentBytes: 1 << 20, want: "exploration bound"},
		{name: "segment", roundBytes: 1 << 20, segmentBytes: uint64(len(encodedSegment)) - 1, want: "capacity"},
		{name: "exact", roundBytes: uint64(len(encodedRound)), segmentBytes: uint64(len(encodedSegment))},
	} {
		t.Run(test.name, func(t *testing.T) {
			config := state.Config
			config.MaxExplorationBytes = test.roundBytes
			bounded, err := exploration.NewChoice(config)
			if err != nil {
				t.Fatal(err)
			}
			boundedRound, _ := bounded.NextRound()
			boundedResults, boundedRuns := choiceRoundOutcomes(t, bounded, boundedRound)
			_, boundedSegment, err := exploration.CommitChoiceRound(bounded, boundedRound, boundedResults)
			if err != nil {
				t.Fatal(err)
			}
			journal := newExplorationCampaign(t, context.Background(), "choice-exploration")
			rounds, err := OpenRounds(journal, ChoiceRounds, config, test.segmentBytes)
			if err != nil {
				t.Fatal(err)
			}
			if err := rounds.Start(boundedRound); err != nil {
				t.Fatal(err)
			}
			err = rounds.Commit(boundedRound, boundedSegment, boundedRuns)
			if test.want == "" {
				if err != nil {
					t.Fatal(err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("oversized %s error = %v", test.name, err)
			}
			if err := journal.Fail("probe", nil); err != nil {
				t.Fatalf("writer was poisoned by a rejected transaction: %v", err)
			}
		})
	}
	_ = runs
}

// A campaign published with its exploration reports exactly the replayed
// summary, chain and recovery count; publication refuses anything else.
func TestExplorationPublicationMatchesItsRounds(t *testing.T) {
	for _, script := range explorationScripts(t) {
		t.Run(script.strategy, func(t *testing.T) {
			journal := newExplorationCampaign(t, context.Background(), script.strategy)
			if err := runScript(t, script, journal); err != nil {
				t.Fatal(err)
			}
			inspected, _, err := inspectRounds(journal.Path())
			if err != nil {
				t.Fatal(err)
			}
			committed, err := ReadCommittedState(journal.Path())
			if err != nil {
				t.Fatal(err)
			}
			summary := CampaignSummary{Attempted: uint64(len(committed.Executions)), Succeeded: uint64(len(committed.Executions))}
			if choiceSummary, ok := inspected.Summary.(exploration.ChoiceSummary); ok {
				summary.ChoiceExploration, summary.ChoiceExplorationImplementationSHA256, summary.ChoiceExplorationChainSHA256 = &choiceSummary, inspected.ImplementationSHA256, inspected.ChainSHA256
				summary.StopReason = string(choiceSummary.StopReason)
			} else {
				simulationSummary := inspected.Summary.(exploration.SimulationSummary)
				summary.SimulationExploration, summary.SimulationExplorationImplementationSHA256, summary.SimulationExplorationChainSHA256 = &simulationSummary, inspected.ImplementationSHA256, inspected.ChainSHA256
				summary.StopReason = string(simulationSummary.StopReason)
			}
			for _, change := range []func(*CampaignSummary){
				func(summary *CampaignSummary) { summary.RecoveryExecutions = 1 },
				func(summary *CampaignSummary) {
					summary.ChoiceExplorationChainSHA256, summary.SimulationExplorationChainSHA256 = record.HashBytes([]byte("chain")), record.HashBytes([]byte("chain"))
				},
			} {
				changed := summary
				change(&changed)
				if err := journal.Publish(changed); err == nil {
					t.Fatal("published an exploration summary that differs from its rounds")
				}
			}
			if err := journal.Publish(summary); err != nil {
				t.Fatal(err)
			}
			if _, err := OpenCampaign(journal.Path()); err != nil {
				t.Fatal(err)
			}
		})
	}
}
