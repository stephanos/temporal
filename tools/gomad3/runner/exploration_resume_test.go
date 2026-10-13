package runner

import (
	"context"
	"fmt"
	"path/filepath"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	backend "go.temporal.io/server/tools/gomad3/backends"
	"go.temporal.io/server/tools/gomad3/record"
	"go.temporal.io/server/tools/gomad3/runner/internal/campaign"
)

// explorationScenario is one scripted exploration campaign whose committed
// rounds must not depend on where it is interrupted.
type explorationScenario struct {
	name     string
	strategy Strategy
	policy   FailurePolicy
	// width is the decision width every execution records; failing makes
	// every execution fail.
	width   int
	failing bool
}

func explorationScenarios() []explorationScenario {
	return []explorationScenario{
		{name: "choice exploration", strategy: StrategyChoiceExploration, policy: PolicyAll, width: 4},
		{name: "choice exploration failures", strategy: StrategyChoiceExploration, policy: PolicyAll, width: 3, failing: true},
		{name: "choice exploration first failure", strategy: StrategyChoiceExploration, policy: PolicyFirst, width: 3, failing: true},
		{name: "simulation exploration", strategy: StrategySimulationExploration, policy: PolicyAll, width: 3},
		{name: "simulation exploration failures", strategy: StrategySimulationExploration, policy: PolicyAll, width: 3, failing: true},
	}
}

// requestRecorder is the executor of one scenario run, recording each
// candidate's logical ordinal.
type requestRecorder interface {
	executionRunner
	ordinals() []uint64
}

type recordedChoiceExecutor struct{ *explorationExecutor }

func (executor recordedChoiceExecutor) ordinals() []uint64 {
	executor.mu.Lock()
	defer executor.mu.Unlock()
	return requestOrdinals(executor.requests)
}

type recordedSimulationExecutor struct {
	*simulationExplorationExecutor
}

func (executor recordedSimulationExecutor) ordinals() []uint64 {
	executor.mu.Lock()
	defer executor.mu.Unlock()
	return requestOrdinals(executor.requests)
}

// requestOrdinals reads each candidate's logical ordinal from its attempt
// work directory, which every candidate of one base seed is distinguished
// by.
func requestOrdinals(requests []backend.ExecutionRequest) []uint64 {
	ordinals := make([]uint64, 0, len(requests))
	for _, request := range requests {
		name, _, _ := strings.Cut(filepath.Base(filepath.Dir(request.Dir)), "-")
		ordinal, err := strconv.ParseUint(name, 10, 64)
		if err != nil {
			panic(err)
		}
		ordinals = append(ordinals, ordinal)
	}
	slices.Sort(ordinals)
	return ordinals
}

func (scenario explorationScenario) executor(t *testing.T, preparer *fakePreparer, limit uint64) requestRecorder {
	if scenario.strategy == StrategyChoiceExploration {
		exitCode := 0
		if scenario.failing {
			exitCode = 1
		}
		return recordedChoiceExecutor{&explorationExecutor{t: t, buildKey: preparer.prepared.BuildKey, limit: limit, exitCode: exitCode, alternatives: uint32(scenario.width)}}
	}
	return recordedSimulationExecutor{&simulationExplorationExecutor{t: t, buildKey: preparer.prepared.BuildKey, limit: limit, fail: scenario.failing, scenarios: scenario.width}}
}

func (scenario explorationScenario) config(t *testing.T) (CampaignSpec, requestRecorder, func() requestRecorder) {
	t.Helper()
	preparer := newFakePreparer(t)
	limit := choiceTraceLimit(t, 1)
	executor := scenario.executor(t, preparer, limit)
	config, _ := testConfig(t, preparer, executor, "7", scenario.policy, 2)
	config.Strategy, config.ChoiceTraceLimit, config.MaxExecutions, config.MaxExplorationBytes = scenario.strategy, limit, 8, 1<<20
	if scenario.strategy == StrategyChoiceExploration {
		config.MaxChoiceDepth = 4
	} else {
		config.MaxForcedDecisions, config.MaxExplorationResultBytes = 2, 1<<20
		config.SimulationDimensionLimits = SimulationDimensionLimits{Runtime: 3, Scenario: 3, Network: 3, Storage: 3, Fault: 3, Crash: 3}
	}
	return config, executor, func() requestRecorder { return scenario.executor(t, preparer, limit) }
}

// observeExploration projects a published exploration campaign onto what an
// interruption must not change: its committed rounds, every logical
// execution's identity and classification, and its published summary.
func observeExploration(t *testing.T, path string) []string {
	t.Helper()
	report, err := Inspect(path, InspectOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if report.Campaign == nil || report.Exploration == nil {
		t.Fatalf("exploration report = %#v", report)
	}
	var observed []string
	for _, round := range report.Exploration.Rounds {
		observed = append(observed, fmt.Sprintf("round %d %s %s from %d x%d", round.Index, round.RoundSHA256, round.SegmentSHA256, round.FirstOrdinal, round.Executions))
	}
	for _, run := range report.Campaign.Executions {
		signature := ""
		if run.FailureSignature != nil {
			signature = string(*run.FailureSignature)
		}
		observed = append(observed, fmt.Sprintf("execution %d %s %s %s/%s %s", run.SelectionOrdinal, run.CandidateSHA256, run.OutcomeSHA256, run.Domain, run.Reason, signature))
	}
	campaignReport := report.Campaign
	observed = append(observed, fmt.Sprintf("attempted=%d succeeded=%d failures=%d distinct=%d retained=%d stop=%s chain=%s",
		campaignReport.Attempted, campaignReport.Succeeded, campaignReport.Failures, campaignReport.DistinctFailures, campaignReport.RetainedSuccesses, campaignReport.StopReason, report.Exploration.ChainSHA256))
	return observed
}

// Every exploration log append from the exploration record through
// publication fails in turn at each of its boundaries, or has the host cancel
// the campaign once it is durable. Each interrupted campaign resumes to the
// uninterrupted rounds, logical executions and summary: a committed round
// never runs again and an interrupted round runs again in full.
func TestExplorationResumesEveryRoundAppendFaultToTheUninterruptedRounds(t *testing.T) {
	for _, scenario := range explorationScenarios() {
		t.Run(scenario.name, func(t *testing.T) {
			t.Parallel()
			config, executor, _ := scenario.config(t)
			reference, err := exploreWith(context.Background(), config, executionDependencies{executor: executor})
			if err != nil {
				t.Fatal(err)
			}
			want := observeExploration(t, reference.CampaignPath)
			kinds := campaignLogKinds(t, reference.CampaignPath)
			first := slices.Index(kinds, 8) + 1
			if first == 0 {
				t.Fatalf("exploration log kinds = %v", kinds)
			}
			for index := first; index <= len(kinds); index++ {
				for _, mode := range []string{"write", "partial", "sync", "synced", "cancel"} {
					t.Run(fmt.Sprintf("append %d %s", index, mode), func(t *testing.T) {
						config, executor, resumeExecutor := scenario.config(t)
						ctx, cancel := context.WithCancel(context.Background())
						defer cancel()
						fault := &appendFault{index: index, mode: mode, cancel: cancel}
						config.TerminateGrace = 10 * time.Millisecond
						interrupted, err := exploreWith(campaign.WithStoreFaults(ctx, fault.hook), config, executionDependencies{executor: executor})
						if !fault.injected || err == nil || fault.late != 0 {
							t.Fatalf("interrupted exploration error = %v, injected = %t, late appends = %d", err, fault.injected, fault.late)
						}
						for range 3 {
							explored, _, err := campaign.InspectExploration(interrupted.CampaignPath)
							if err != nil {
								t.Fatal(err)
							}
							state, err := campaign.ReadCommittedState(interrupted.CampaignPath)
							if err != nil {
								t.Fatal(err)
							}
							if state.Lifecycle.Published {
								break
							}
							resumed := resumeExecutor()
							_, err = exploreWith(context.Background(), CampaignSpec{ResumeCampaign: interrupted.CampaignPath, RunnerBuild: config.RunnerBuild, Backend: scriptProvider()}, executionDependencies{executor: resumed})
							ran := resumed.ordinals()
							committed := uint64(0)
							for _, round := range explored.Rounds {
								committed = round.FirstOrdinal + round.Executions
							}
							if len(ran) != 0 && ran[0] < committed {
								t.Fatalf("resume reran committed ordinals: ran %v, %d committed", ran, committed)
							}
							if staged := explored.StagedRound; staged != nil {
								for ordinal := committed; ordinal < committed+staged.Candidates; ordinal++ {
									if !slices.Contains(ran, ordinal) {
										t.Fatalf("resume did not rerun ordinal %d of interrupted round %d: ran %v", ordinal, staged.Index, ran)
									}
								}
							}
							if err == nil {
								break
							}
							t.Logf("resume failed: %v", err)
						}
						if got := observeExploration(t, interrupted.CampaignPath); !reflect.DeepEqual(got, want) {
							t.Fatalf("resumed exploration = %q\nwant %q", got, want)
						}
					})
				}
			}
		})
	}
}

// Recovery executions count the candidates of every round a resume
// discarded, cumulatively across resumes, and are published.
func TestExplorationRecoveryExecutionsCountEveryDiscardedRound(t *testing.T) {
	scenario := explorationScenarios()[0]
	config, executor, resumeExecutor := scenario.config(t)
	recoveries := []int{}
	for _, interruption := range []int{0, 1} {
		ctx, cancel := context.WithCancel(context.Background())
		var fault *appendFault
		var err error
		var summary CampaignResult
		if interruption == 0 {
			// Fail the root round's commit, after its start.
			fault = &appendFault{index: 7, mode: "write", cancel: cancel}
			summary, err = exploreWith(campaign.WithStoreFaults(ctx, fault.hook), config, executionDependencies{executor: executor})
			config.ResumeCampaign = summary.CampaignPath
		} else {
			// Fail the resumed root round's commit again.
			fault = &appendFault{index: 3, mode: "write", cancel: cancel}
			_, err = exploreWith(campaign.WithStoreFaults(ctx, fault.hook), CampaignSpec{ResumeCampaign: config.ResumeCampaign, RunnerBuild: config.RunnerBuild, Backend: scriptProvider()}, executionDependencies{executor: resumeExecutor()})
		}
		cancel()
		if err == nil || !fault.injected {
			t.Fatalf("interruption %d error = %v, injected = %t", interruption, err, fault.injected)
		}
		explored, _, err := campaign.InspectExploration(config.ResumeCampaign)
		if err != nil || explored.StagedRound == nil || explored.StagedRound.Index != 0 {
			t.Fatalf("interruption %d exploration = %#v, %v", interruption, explored, err)
		}
		recoveries = append(recoveries, int(explored.RecoveryExecutions))
	}
	resumed, err := exploreWith(context.Background(), CampaignSpec{ResumeCampaign: config.ResumeCampaign, RunnerBuild: config.RunnerBuild, Backend: scriptProvider()}, executionDependencies{executor: resumeExecutor()})
	if err != nil {
		t.Fatal(err)
	}
	batch, err := campaign.OpenCampaign(config.ResumeCampaign)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(recoveries, []int{0, 1}) || resumed.RecoveryExecutions != 2 || batch.Record.RecoveryExecutions != record.Uint64String(2) {
		t.Fatalf("recovery executions = %v, resumed %d, published %d", recoveries, resumed.RecoveryExecutions, batch.Record.RecoveryExecutions)
	}
}
