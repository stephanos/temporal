package campaign

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomad/internal/evidence/artifact"
	"go.temporal.io/server/tools/gomad/internal/evidence/record"
	"go.temporal.io/server/tools/gomad/internal/runner/internal/logstore"
)

// frontierScript is a deterministic seed campaign: each ordinal's actual
// observation, keyed by ordinal, under one failure policy.
type frontierScript struct {
	policy   string
	budget   uint64
	parallel uint64
	failures map[uint64]string
	count    uint64
}

func (script frontierScript) options(ctx context.Context) failureCampaignOptions {
	return failureCampaignOptions{ctx: ctx, selection: fmt.Sprintf("7-%d", 7+script.count-1), count: script.count, parallel: script.parallel, policy: script.policy, budget: script.budget}
}

// frontierRun is a started campaign with one published failure artifact per
// failure reason of its script.
type frontierRun struct {
	fixture   failureCampaign
	script    frontierScript
	artifacts map[string]ExecutionRecord
}

func startFrontierRun(t *testing.T, script frontierScript, ctx context.Context) frontierRun {
	t.Helper()
	fixture := newFailureCampaignWith(t, script.options(ctx))
	run := frontierRun{fixture: fixture, script: script, artifacts: map[string]ExecutionRecord{}}
	for ordinal, reason := range script.failures {
		if _, published := run.artifacts[reason]; !published {
			run.artifacts[reason] = fixture.publishFailure(t, ordinal, 7+ordinal, reason)
		}
	}
	if err := fixture.journal.StartExecutions(); err != nil {
		t.Fatal(err)
	}
	return run
}

// observation is ordinal's actual completion; cancelled reports an attempt
// that a first-failure stop cancelled before it completed.
func (run frontierRun) observation(ordinal uint64, cancelled bool) Receipt {
	seed := record.Uint64String(7 + ordinal)
	if cancelled {
		return Receipt{Observation: ExecutionRecord{SelectionOrdinal: record.Uint64String(ordinal), Seed: seed, Domain: "runner", Reason: "runner_cancelled", Termination: "none"}}
	}
	if reason, failed := run.script.failures[ordinal]; failed {
		failure := run.artifacts[reason]
		failure.SelectionOrdinal, failure.Seed = record.Uint64String(ordinal), seed
		return Receipt{Observation: failure, Artifacts: []string{*failure.Artifact}}
	}
	return Receipt{Observation: successExecution(ordinal, 7+ordinal)}
}

func outcomeOf(receipt Receipt) ExecutionRecord {
	return receipt.Observation
}

// drive advances the campaign from its durable frontier to completion, one
// record at a time, exactly as an ordered scheduler would: receipts complete
// out of order (highest outstanding first), the next ordered receipt commits
// with its refill, and a pending stop drains into one stop group. It returns
// the first error.
func (run frontierRun) drive(journal *CampaignJournal) error {
	next := uint64(0)
	committed := func() map[uint64]struct{} {
		done := map[uint64]struct{}{}
		for _, execution := range journal.state.executions {
			done[uint64(execution.SelectionOrdinal)] = struct{}{}
		}
		return done
	}
	refill := func(available uint64, frontier Frontier) []SeedJob {
		done := committed()
		admitted := map[uint64]struct{}{}
		for _, job := range frontier.Outstanding {
			admitted[job.Ordinal] = struct{}{}
		}
		var jobs []SeedJob
		for ; next < run.script.count && uint64(len(jobs)) < available; next++ {
			_, isDone := done[next]
			_, isAdmitted := admitted[next]
			if !isDone && !isAdmitted {
				jobs = append(jobs, SeedJob{Ordinal: next, Seed: 7 + next})
			}
		}
		return jobs
	}
	for {
		frontier := journal.Frontier()
		if frontier.Stopped != "" {
			return nil
		}
		if frontier.PendingStop != "" {
			retry := frontier.Retry()
			if len(retry) != 0 {
				job := nextReceipt(retry)
				if err := journal.RecordReceipt(run.observation(job.Ordinal, frontier.PendingStop == StopFirstFailure)); err != nil {
					return err
				}
				continue
			}
			var group []ExecutionRecord
			for _, job := range frontier.Outstanding {
				group = append(group, outcomeOf(frontier.Receipts[job.Ordinal]))
			}
			if err := journal.CommitStopGroup(group); err != nil {
				return err
			}
			continue
		}
		if len(frontier.Outstanding) == 0 {
			jobs := refill(frontier.Available, frontier)
			if len(jobs) == 0 {
				return nil
			}
			if err := journal.Admit(jobs); err != nil {
				return err
			}
			continue
		}
		first := frontier.Outstanding[0]
		if receipt, received := frontier.Receipts[first.Ordinal]; received {
			if err := journal.CommitOutcome(outcomeOf(receipt), refill(frontier.Available+1, frontier)); err != nil {
				return err
			}
			continue
		}
		job := nextReceipt(frontier.Retry())
		if err := journal.RecordReceipt(run.observation(job.Ordinal, false)); err != nil {
			return err
		}
	}
}

// nextReceipt chooses which receipt-less attempt completes next from the
// durable frontier alone, so a resumed run makes the same choices: the
// highest when its ordinal is even, otherwise the lowest.
func nextReceipt(retry []SeedJob) SeedJob {
	if highest := retry[len(retry)-1]; highest.Ordinal%2 == 0 {
		return highest
	}
	return retry[0]
}

func committedOutcomes(t *testing.T, path string) []ExecutionRecord {
	t.Helper()
	committed, err := ReadCommittedState(path)
	if err != nil {
		t.Fatal(err)
	}
	outcomes := committed.Executions
	slices.SortFunc(outcomes, func(left, right ExecutionRecord) int {
		return compareUint64(uint64(left.SelectionOrdinal), uint64(right.SelectionOrdinal))
	})
	return outcomes
}

func outcomeSummary(outcomes []ExecutionRecord) []string {
	summary := make([]string, len(outcomes))
	for index, outcome := range outcomes {
		summary[index] = fmt.Sprintf("%d:%s:%s", outcome.SelectionOrdinal, outcome.Domain, outcome.Reason)
	}
	return summary
}

var frontierScripts = map[string]frontierScript{
	"first failure cancels the drain": {policy: "first", budget: 1, parallel: 3, count: 7, failures: map[uint64]string{3: "first_reason"}},
	"budget lets the drain finish":    {policy: "budget", budget: 2, parallel: 2, count: 7, failures: map[uint64]string{1: "first_reason", 3: "second_reason", 4: "first_reason"}},
	"all failures run every seed":     {policy: "all", budget: 1, parallel: 2, count: 5, failures: map[uint64]string{0: "first_reason", 2: "first_reason"}},
}

func TestFrontierScriptsCommitTheirOrderedOutcomes(t *testing.T) {
	want := map[string][]string{
		"first failure cancels the drain": {"0:success:success", "1:success:success", "2:success:success", "3:target:first_reason", "4:runner:runner_cancelled", "5:runner:runner_cancelled"},
		"budget lets the drain finish":    {"0:success:success", "1:target:first_reason", "2:success:success", "3:target:second_reason", "4:target:first_reason"},
		"all failures run every seed":     {"0:target:first_reason", "1:success:success", "2:target:first_reason", "3:success:success", "4:success:success"},
	}
	for name, script := range frontierScripts {
		t.Run(name, func(t *testing.T) {
			run := startFrontierRun(t, script, context.Background())
			if err := run.drive(run.fixture.journal); err != nil {
				t.Fatal(err)
			}
			if got := outcomeSummary(committedOutcomes(t, run.fixture.journal.Path())); !slices.Equal(got, want[name]) {
				t.Fatalf("committed outcomes = %v, want %v", got, want[name])
			}
		})
	}
}

// Every append, partial write, sync and acknowledged sync of every admission,
// receipt, ordered outcome and stop group is interrupted once. The writer
// stops at the fault; a resume from the durable log then finishes the
// campaign, which must commit exactly the uninterrupted logical outcomes.
func TestFrontierFaultsAtEveryPointResumeToTheUninterruptedOutcomes(t *testing.T) {
	for name, script := range frontierScripts {
		t.Run(name, func(t *testing.T) {
			reference := startFrontierRun(t, script, context.Background())
			var observed []mutationPoint
			controller := &mutationController{inject: func(point mutationPoint) error {
				observed = append(observed, point)
				return nil
			}}
			hooked := startFrontierRun(t, script, withMutationHook(context.Background(), controller.observe))
			controller.enabled = true
			if err := hooked.drive(hooked.fixture.journal); err != nil {
				t.Fatal(err)
			}
			if err := reference.drive(reference.fixture.journal); err != nil {
				t.Fatal(err)
			}
			want := outcomeSummary(committedOutcomes(t, reference.fixture.journal.Path()))
			var points []int
			for index, point := range observed {
				if strings.HasPrefix(string(point.Kind), "log-") {
					points = append(points, index)
				}
			}
			if len(points) < 3*10 {
				t.Fatalf("frontier run observed only %d log points", len(points))
			}
			for _, index := range points {
				for _, partial := range []int{0, 1, 17} {
					expected := observed[index]
					if partial != 0 && expected.Kind != "log-write" {
						continue
					}
					t.Run(fmt.Sprintf("%03d-%s-partial%d", index, expected.Kind, partial), func(t *testing.T) {
						plan := &faultPlan{faultIndex: index, partial: partial, expected: expected}
						controller := &mutationController{inject: plan.observe(t)}
						run := startFrontierRun(t, script, withMutationHook(context.Background(), controller.observe))
						controller.enabled = true
						err := run.drive(run.fixture.journal)
						if err == nil || !errors.Is(err, logstore.ErrPoisoned) {
							t.Fatalf("drive() at %#v error = %v, want a poisoned writer", expected, err)
						}
						if err := run.fixture.journal.writer.Append(kindLifecycle, []byte(`{"state":"running"}`)); !errors.Is(err, logstore.ErrPoisoned) {
							t.Fatalf("append after an uncertain append error = %v, want poisoned", err)
						}
						if err := run.fixture.journal.Close(); err != nil {
							t.Fatal(err)
						}
						for attempt := range 2 {
							resumed, state, err := ResumeCampaignJournal(context.Background(), run.fixture.journal.Path())
							if err != nil {
								t.Fatalf("resume %d after %#v: %v", attempt, expected, err)
							}
							for _, outcome := range state.Executions {
								if _, admitted := state.Frontier.Receipts[uint64(outcome.SelectionOrdinal)]; admitted {
									t.Fatalf("ordinal %d is both committed and pending", outcome.SelectionOrdinal)
								}
							}
							if attempt == 1 {
								if err := run.drive(resumed); err != nil {
									t.Fatalf("finish after %#v: %v", expected, err)
								}
							}
							if err := resumed.Close(); err != nil {
								t.Fatal(err)
							}
						}
						if got := outcomeSummary(committedOutcomes(t, run.fixture.journal.Path())); !slices.Equal(got, want) {
							t.Fatalf("fault at %#v committed %v, want %v", expected, got, want)
						}
					})
				}
			}
		})
	}
}

func TestFrontierAdmitsWithinParallelAndRefillsIncrementally(t *testing.T) {
	run := startFrontierRun(t, frontierScript{policy: "all", budget: 1, parallel: 2, count: 5}, context.Background())
	journal := run.fixture.journal
	if err := journal.Admit([]SeedJob{{Ordinal: 0, Seed: 7}, {Ordinal: 1, Seed: 8}}); err != nil {
		t.Fatal(err)
	}
	before := logFileSize(t, journal.Path())
	var capacityErr *JournalCapacityError
	if err := journal.Admit([]SeedJob{{Ordinal: 2, Seed: 9}}); !errors.As(err, &capacityErr) || capacityErr.Limit != journalLimitPartialExecutions {
		t.Fatalf("Admit() over parallel error = %v", err)
	}
	for _, invalid := range [][]SeedJob{{{Ordinal: 1, Seed: 8}}, {{Ordinal: 9, Seed: 16}}} {
		if err := journal.CommitOutcome(successExecution(0, 7), invalid); err == nil {
			t.Fatalf("CommitOutcome() refill %v was accepted", invalid)
		}
	}
	if err := journal.RecordReceipt(run.observation(1, false)); err != nil {
		t.Fatal(err)
	}
	if err := journal.CommitOutcome(successExecution(1, 8), nil); err == nil || !strings.Contains(err.Error(), "next ordered") {
		t.Fatalf("out-of-order outcome error = %v", err)
	}
	if after := logFileSize(t, journal.Path()); after-before == 0 {
		t.Fatal("receipt was not appended")
	}
	if err := journal.RecordReceipt(run.observation(0, false)); err != nil {
		t.Fatal(err)
	}
	if err := journal.CommitOutcome(successExecution(0, 7), []SeedJob{{Ordinal: 2, Seed: 9}, {Ordinal: 3, Seed: 10}}); !errors.As(err, &capacityErr) {
		t.Fatalf("refill beyond the released slot error = %v", err)
	}
	if err := journal.CommitOutcome(successExecution(0, 7), []SeedJob{{Ordinal: 2, Seed: 9}}); err != nil {
		t.Fatal(err)
	}
	frontier := journal.Frontier()
	if !reflect.DeepEqual(frontier.Outstanding, []SeedJob{{Ordinal: 1, Seed: 8}, {Ordinal: 2, Seed: 9}}) || frontier.Available != 0 || len(frontier.Receipts) != 1 {
		t.Fatalf("frontier after ordered refill = %#v", frontier)
	}
	if err := journal.Close(); err != nil {
		t.Fatal(err)
	}
	resumed, state, err := ResumeCampaignJournal(context.Background(), journal.Path())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := resumed.Close(); err != nil {
			t.Error(err)
		}
	})
	if !reflect.DeepEqual(state.Frontier, frontier) || !reflect.DeepEqual(state.Frontier.Retry(), []SeedJob{{Ordinal: 2, Seed: 9}}) || len(state.Executions) != 1 {
		t.Fatalf("resumed frontier = %#v, executions %v", state.Frontier, state.Executions)
	}
	if err := resumed.AppendExecution(successExecution(3, 10)); err == nil {
		t.Fatal("a campaign that admits ordinals committed an unadmitted outcome")
	}
}

func TestFrontierReceiptsRejectConflictsAndRetentionDecisions(t *testing.T) {
	run := startFrontierRun(t, frontierScript{policy: "first", budget: 1, parallel: 2, count: 4, failures: map[uint64]string{0: "first_reason"}}, context.Background())
	journal := run.fixture.journal
	if err := journal.RecordReceipt(run.observation(0, false)); err == nil {
		t.Fatal("a receipt without an admission was accepted")
	}
	if err := journal.Admit([]SeedJob{{Ordinal: 0, Seed: 7}, {Ordinal: 1, Seed: 8}}); err != nil {
		t.Fatal(err)
	}
	retained := successExecution(1, 8)
	reference, bytes := "successes/sha256-0123456789abcdef0123456789abcdef", record.Uint64String(1)
	retained.SuccessArtifact, retained.SuccessArtifactBytes = &reference, &bytes
	wrongSeed := successExecution(1, 9)
	missingEvidence := run.observation(0, false)
	missingEvidence.Artifacts = nil
	for name, receipt := range map[string]Receipt{
		"retention decision":     {Observation: retained, Artifacts: []string{reference}},
		"another seed":           {Observation: wrongSeed},
		"unpublished evidence":   missingEvidence,
		"premature cancellation": run.observation(1, true),
		"foreign payload":        {Observation: successExecution(1, 8), Artifacts: []string{"../escape"}},
		"too many pending":       {Observation: successExecution(1, 8), Artifacts: []string{reference, reference, reference}},
	} {
		if err := journal.RecordReceipt(receipt); err == nil {
			t.Fatalf("%s receipt was accepted", name)
		}
	}
	if err := journal.RecordReceipt(run.observation(1, false)); err != nil {
		t.Fatal(err)
	}
	if err := journal.RecordReceipt(run.observation(1, false)); err == nil || !strings.Contains(err.Error(), "conflicts") {
		t.Fatalf("duplicate receipt error = %v", err)
	}
	if err := journal.RecordReceipt(run.observation(0, false)); err != nil {
		t.Fatal(err)
	}
	relabelled := run.observation(1, true).Observation
	if err := journal.CommitStopGroup([]ExecutionRecord{outcomeOf(run.observation(0, false)), relabelled}); err == nil || !strings.Contains(err.Error(), "differs from its durable receipt") {
		t.Fatalf("relabelled stop group member error = %v", err)
	}
	if err := journal.CommitStopGroup([]ExecutionRecord{outcomeOf(run.observation(0, false))}); err == nil {
		t.Fatal("a stop group without every admitted member was accepted")
	}
	if err := journal.CommitOutcome(outcomeOf(run.observation(0, false)), nil); err == nil || !strings.Contains(err.Error(), "stop group") {
		t.Fatalf("stopping outcome outside its group error = %v", err)
	}
	unpublished := successExecution(1, 8)
	unpublished.SuccessArtifact, unpublished.SuccessArtifactBytes = &reference, &bytes
	if err := journal.CommitStopGroup([]ExecutionRecord{outcomeOf(run.observation(0, false)), unpublished}); err == nil || !strings.Contains(err.Error(), "did not publish") {
		t.Fatalf("retaining unpublished evidence error = %v", err)
	}
}

func TestFrontierStopGroupKeepsActualDrainOutcomesAndFreezesAdmission(t *testing.T) {
	for name, test := range map[string]struct {
		script frontierScript
		stop   ControllerStopReason
		want   []string
	}{
		"first failure": {
			script: frontierScript{policy: "first", budget: 1, parallel: 3, count: 5, failures: map[uint64]string{0: "first_reason"}},
			stop:   StopFirstFailure, want: []string{"0:target:first_reason", "1:runner:runner_cancelled", "2:success:success"},
		},
		"failure budget": {
			script: frontierScript{policy: "budget", budget: 1, parallel: 3, count: 5, failures: map[uint64]string{0: "first_reason", 1: "second_reason"}},
			stop:   StopFailureBudget, want: []string{"0:target:first_reason", "1:target:second_reason", "2:success:success"},
		},
	} {
		t.Run(name, func(t *testing.T) {
			run := startFrontierRun(t, test.script, context.Background())
			journal := run.fixture.journal
			if err := journal.Admit([]SeedJob{{Ordinal: 0, Seed: 7}, {Ordinal: 1, Seed: 8}, {Ordinal: 2, Seed: 9}}); err != nil {
				t.Fatal(err)
			}
			// A real success buffered behind the stopping failure.
			if err := journal.RecordReceipt(run.observation(2, false)); err != nil {
				t.Fatal(err)
			}
			if err := journal.RecordReceipt(run.observation(0, false)); err != nil {
				t.Fatal(err)
			}
			frontier := journal.Frontier()
			if frontier.PendingStop != test.stop || frontier.Available != 0 || !reflect.DeepEqual(frontier.Retry(), []SeedJob{{Ordinal: 1, Seed: 8}}) {
				t.Fatalf("pending frontier = %#v", frontier)
			}
			if err := journal.Admit([]SeedJob{{Ordinal: 3, Seed: 10}}); err == nil || !strings.Contains(err.Error(), "freezes admission") {
				t.Fatalf("admission during a pending stop error = %v", err)
			}
			cancelled := test.stop == StopFirstFailure
			if !cancelled {
				if err := journal.RecordReceipt(run.observation(1, true)); err == nil {
					t.Fatal("a budget stop accepted a policy cancellation")
				}
			}
			if err := journal.RecordReceipt(run.observation(1, cancelled)); err != nil {
				t.Fatal(err)
			}
			if err := journal.Close(); err != nil {
				t.Fatal(err)
			}
			resumed, state, err := ResumeCampaignJournal(context.Background(), journal.Path())
			if err != nil {
				t.Fatal(err)
			}
			if state.Frontier.PendingStop != test.stop || len(state.Frontier.Retry()) != 0 || len(state.Frontier.Receipts) != 3 {
				t.Fatalf("resumed pending stop = %#v", state.Frontier)
			}
			var group []ExecutionRecord
			for _, job := range state.Frontier.Outstanding {
				group = append(group, outcomeOf(state.Frontier.Receipts[job.Ordinal]))
			}
			if err := resumed.CommitStopGroup(group); err != nil {
				t.Fatal(err)
			}
			if err := resumed.CommitStopGroup(group); err == nil {
				t.Fatal("a committed stop group committed again")
			}
			if err := resumed.Admit([]SeedJob{{Ordinal: 3, Seed: 10}}); err == nil || !strings.Contains(err.Error(), "admits no ordinal") {
				t.Fatalf("admission after a committed stop error = %v", err)
			}
			if err := resumed.Close(); err != nil {
				t.Fatal(err)
			}
			for range 2 {
				again, state, err := ResumeCampaignJournal(context.Background(), journal.Path())
				if err != nil {
					t.Fatal(err)
				}
				if got := outcomeSummary(state.Executions); !slices.Equal(got, test.want) || state.Frontier.Stopped != test.stop || len(state.Frontier.Outstanding) != 0 || state.Frontier.Available != 0 {
					t.Fatalf("resumed stopped campaign = %v %#v, want %v", got, state.Frontier, test.want)
				}
				if err := again.Close(); err != nil {
					t.Fatal(err)
				}
			}
		})
	}
}

func TestFrontierPendingPayloadsStayLiveUntilTheirOutcomeAdoptsOrRetiresThem(t *testing.T) {
	run := startFrontierRun(t, frontierScript{policy: "all", budget: 1, parallel: 2, count: 3, failures: map[uint64]string{0: "first_reason"}}, context.Background())
	journal := run.fixture.journal
	if err := journal.Admit([]SeedJob{{Ordinal: 0, Seed: 7}, {Ordinal: 1, Seed: 8}}); err != nil {
		t.Fatal(err)
	}
	if err := journal.RecordReceipt(run.observation(0, false)); err != nil {
		t.Fatal(err)
	}
	// Ordinal 1 published a success candidate that its outcome will not
	// retain.
	candidate := run.fixture.input
	candidate.Manifest.ArtifactKind, candidate.Manifest.SelectionOrdinal, candidate.Manifest.Seed = record.ArtifactSuccess, 1, 8
	candidate.Manifest.Environment = append([]record.Environment(nil), candidate.Manifest.Environment...)
	candidate.Manifest.Environment[1].Value = "8"
	exitCode := record.Uint64String(0)
	candidate.Manifest.Outcome = record.Outcome{Domain: "success", Reason: "success", Termination: "exit", ExitCode: &exitCode}
	published, err := artifact.PublishArtifact(artifact.Store{Root: journal.SuccessesPath(), Key: artifact.StoreKeyRecord}, candidate)
	if err != nil {
		t.Fatal(err)
	}
	successReference := "successes/" + filepath.Base(published.Path)
	diagnostic, err := artifact.PublishOwnedFile(context.Background(), filepath.Join(journal.Path(), "diagnostics"), "00000000000000000001-trace.bin", []byte("trace"))
	if err != nil {
		t.Fatal(err)
	}
	if err := journal.RecordReceipt(Receipt{Observation: successExecution(1, 8), Artifacts: []string{successReference}, Diagnostics: []artifact.FileReference{diagnostic}}); err != nil {
		t.Fatal(err)
	}
	if err := journal.Close(); err != nil {
		t.Fatal(err)
	}
	failurePath := filepath.Join(journal.Path(), filepath.FromSlash(*run.artifacts["first_reason"].Artifact))
	successPath := filepath.Join(journal.Path(), filepath.FromSlash(successReference))
	recovered, err := RecoverCampaign(context.Background(), journal.Path())
	if err != nil || recovered.Changed {
		t.Fatalf("recovery with only live pending payloads = %#v, %v", recovered, err)
	}
	for _, path := range []string{failurePath, successPath} {
		if _, err := os.Lstat(path); err != nil {
			t.Fatalf("pending payload %s was reclaimed: %v", path, err)
		}
	}
	if err := os.Rename(successPath, successPath+".moved"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := ResumeCampaignJournal(context.Background(), journal.Path()); err == nil || !IsIntegrityError(err) || !errors.Is(err, artifact.ErrCorruptReference) {
		t.Fatalf("resume with a missing pending payload error = %v", err)
	}
	if err := os.Rename(successPath+".moved", successPath); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(journal.Path(), "diagnostics", diagnostic.Name), []byte("changed"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := RecoverCampaign(context.Background(), journal.Path()); err == nil || !errors.Is(err, artifact.ErrCorruptReference) {
		t.Fatalf("recovery with a corrupt pending diagnostic error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(journal.Path(), "diagnostics", diagnostic.Name), []byte("trace"), 0o600); err != nil {
		t.Fatal(err)
	}
	resumed, _, err := ResumeCampaignJournal(context.Background(), journal.Path())
	if err != nil {
		t.Fatal(err)
	}
	if err := resumed.CommitOutcome(outcomeOf(run.observation(0, false)), nil); err != nil {
		t.Fatal(err)
	}
	if err := resumed.CommitOutcome(successExecution(1, 8), nil); err != nil {
		t.Fatal(err)
	}
	if err := resumed.Close(); err != nil {
		t.Fatal(err)
	}
	// A sidecar no receipt or outcome references is an orphan of an attempt
	// that never recorded its receipt.
	orphan := filepath.Join(journal.Path(), "diagnostics", "00000000000000000002-orphan.bin")
	if err := os.WriteFile(orphan, []byte("orphan"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := RecoverCampaign(context.Background(), journal.Path()); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(failurePath); err != nil {
		t.Fatalf("adopted failure evidence was reclaimed: %v", err)
	}
	if _, err := os.Lstat(successPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("retired success candidate survived recovery: %v", err)
	}
	if _, err := os.Lstat(filepath.Join(journal.Path(), "diagnostics", diagnostic.Name)); err != nil {
		t.Fatalf("diagnostic adopted by its outcome was reclaimed: %v", err)
	}
	if _, err := os.Lstat(orphan); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("orphaned diagnostic survived recovery: %v", err)
	}
	if err := os.WriteFile(filepath.Join(journal.Path(), "diagnostics", diagnostic.Name), []byte("changed"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := RecoverCampaign(context.Background(), journal.Path()); err == nil || !errors.Is(err, artifact.ErrCorruptReference) {
		t.Fatalf("recovery with a corrupt adopted diagnostic error = %v", err)
	}
}

func TestFrontierBoundsDeriveFromTheCampaignEnvelope(t *testing.T) {
	run := startFrontierRun(t, frontierScript{policy: "all", budget: 1, parallel: 2, count: 3}, context.Background())
	journal := run.fixture.journal
	if err := journal.Admit([]SeedJob{{Ordinal: 0, Seed: 7}}); err != nil {
		t.Fatal(err)
	}
	oversized := successExecution(0, 7)
	oversized.Reason = strings.Repeat("x", defaultExecutionSegmentBytes)
	var capacityErr *JournalCapacityError
	if err := journal.RecordReceipt(Receipt{Observation: oversized}); !errors.As(err, &capacityErr) || capacityErr.Limit != journalLimitSegmentBytes {
		t.Fatalf("oversized receipt error = %v", err)
	}
	if bound := campaignLogKinds[kindReceipt]; bound < maximumExecutionSegmentBytes {
		t.Fatalf("receipt frame bound %d is below the per-execution envelope", bound)
	}
	if bound := campaignLogKinds[kindOutcome]; bound != logstore.MaxPayloadBytes {
		t.Fatalf("outcome frame bound %d is a new aggregate cap", bound)
	}
	if err := journal.append(kindOutcome, outcomeRecord{Executions: []ExecutionRecord{successExecution(0, 7), successExecution(1, 8), successExecution(2, 9)}}); err == nil {
		t.Fatal("an outcome transaction larger than the partial executions was accepted")
	}
}
