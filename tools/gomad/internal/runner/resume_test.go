package runner

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	backend "go.temporal.io/server/tools/gomad/internal/backends"
	"go.temporal.io/server/tools/gomad/internal/backends/target"
	"go.temporal.io/server/tools/gomad/internal/evidence/record"
	"go.temporal.io/server/tools/gomad/internal/model/choice"
	"go.temporal.io/server/tools/gomad/internal/runner/inspect"
	"go.temporal.io/server/tools/gomad/internal/runner/internal/campaign"
	"go.temporal.io/server/tools/gomad/internal/runner/internal/logstore"
	campaignoptions "go.temporal.io/server/tools/gomad/internal/runner/options"
)

type resumeProfileProvider struct {
	backend.Provider
	description backend.Description
	options     string
	validations int
}

type resumeProfileProvenance struct {
	Options      string
	TargetSHA256 record.SHA256
	Args         []string
	Scenario     []byte
}

func (provider *resumeProfileProvider) Describe(_ context.Context, config backend.Configuration) (backend.Description, error) {
	if config.IOProfile != provider.description.IOProfile.Name {
		return backend.Description{}, errors.New("configured provider profile changed")
	}
	return provider.description, nil
}

func (provider *resumeProfileProvider) ValidatePrepared(spec target.Spec, prepared target.Prepared, environment []string) error {
	provider.validations++
	if err := prepared.ValidateBackendPayloads(); err != nil {
		return err
	}
	var provenance resumeProfileProvenance
	if err := json.Unmarshal(prepared.BackendPayloads[0].Data, &provenance); err != nil {
		return err
	}
	if provenance.Options != provider.options {
		return errors.New("retained provider options changed")
	}
	if !bytes.Equal(spec.Scenario, provenance.Scenario) {
		return errors.New("retained scenario changed")
	}
	if spec.Backend != prepared.Backend.Name || spec.Kind != prepared.Kind || spec.Source != prepared.Source || provenance.TargetSHA256 != record.SHA256(prepared.SHA256) || !reflect.DeepEqual(prepared.Argv, append([]string{"gomad3-target"}, provenance.Args...)) || !reflect.DeepEqual(spec.Args, provenance.Args) || !reflect.DeepEqual(environment, []string{"MODE=retained"}) {
		return errors.New("retained target provenance changed")
	}
	return prepared.Verify()
}

func TestResumeProfileUsesValidatedBackendProvenance(t *testing.T) {
	for _, test := range []struct {
		name        string
		change      func(*resumeProfileProvider, *campaign.CampaignPlan, string)
		wantError   string
		validations int
	}{
		{name: "retained provenance", validations: 1},
		{name: "changed current options", change: func(provider *resumeProfileProvider, _ *campaign.CampaignPlan, _ string) {
			provider.options = "changed"
		}, wantError: "retained provider options changed", validations: 1},
		{name: "changed scenario", change: func(_ *resumeProfileProvider, plan *campaign.CampaignPlan, _ string) {
			plan.Prepared.Scenario = []byte(`{"nodes":["changed"]}`)
		}, wantError: "retained scenario changed", validations: 1},
		{name: "changed provenance bytes", change: func(_ *resumeProfileProvider, _ *campaign.CampaignPlan, path string) {
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			data[0] ^= 1
			if err := os.WriteFile(path, data, 0o600); err != nil {
				t.Fatal(err)
			}
		}, wantError: "external backend payload bytes do not match their reference"},
		{name: "changed recorded profile", change: func(_ *resumeProfileProvider, plan *campaign.CampaignPlan, _ string) {
			plan.IOProfile.ImplementationSHA256 = record.HashBytes([]byte("changed"))
		}, wantError: "recorded I/O profile identity does not match the external backend"},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			module := []byte("retained executable")
			scenario := []byte(`{"nodes":["a","b"]}`)
			provenance, err := json.Marshal(resumeProfileProvenance{Options: "retained", TargetSHA256: record.HashBytes(module), Args: []string{"fixed"}, Scenario: scenario})
			if err != nil {
				t.Fatal(err)
			}
			provenancePath := filepath.Join(root, "prepared", "backend", "provenance.json")
			if err := os.MkdirAll(filepath.Dir(provenancePath), 0o700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(provenancePath, provenance, 0o600); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(root, "prepared", "target"), module, 0o500); err != nil {
				t.Fatal(err)
			}
			metadata := &record.BackendMetadata{Name: "fixture-resume", ReplayMode: record.ReplayObserved, Provenance: record.BackendPayload{Schema: "fixture.resume-provenance/v1", File: "backend/provenance.json", SHA256: record.HashBytes(provenance), Bytes: record.Uint64String(len(provenance))}}
			profile := record.BackendIOProfile(*metadata)
			plan := campaign.CampaignPlan{
				RunnerBuild: "runner", Selection: "7", SelectionCount: 1,
				IOProfile:   record.IOContract{Name: profile.Name, ImplementationSHA256: profile.ImplementationSHA256, InventorySHA256: profile.InventorySHA256},
				Prepared:    campaign.PreparedTargetPlan{Path: "prepared/target", Scenario: scenario, Target: record.Target{Backend: metadata, Kind: "go-run", Source: "example.com/workload", SHA256: record.HashBytes(module), Size: record.Uint64String(len(module)), Argv: []string{"gomad3-target", "fixed"}}},
				Toolchain:   record.Toolchain{GoVersion: "go1.27.1", BuildKey: "retained-build", TargetGOOS: "wasip1", TargetGOARCH: "wasm"},
				Environment: []record.Environment{{Name: "MODE", Value: "retained"}}, IOROMountLimits: record.ReadOnlyMountLimits{PathBytes: 4096, Requests: 1, Files: 1, DirectoryEntries: 1, SingleFileBytes: 1024, TotalBytes: 1024},
			}
			provider := &resumeProfileProvider{options: "retained", description: backend.Description{IOProfile: record.BackendIOProfile(record.BackendMetadata{Name: metadata.Name, Provenance: record.BackendPayload{SHA256: record.HashBytes([]byte("provider model"))}})}}
			if provider.description.IOProfile.ImplementationSHA256 == profile.ImplementationSHA256 {
				t.Fatal("fixture lacks the material provenance and static description distinction")
			}
			if test.change != nil {
				test.change(provider, &plan, provenancePath)
			}
			config, selection, environment, _, prepared, err := resumeConfiguration(campaignRequest{campaignOptions: campaignOptions{campaignIdentityOptions: campaignIdentityOptions{ResumeCampaign: root}}, campaignRuntime: campaignRuntime{Backend: provider, RunnerBuild: "runner"}}, plan)
			if test.wantError != "" {
				if err == nil || err.Error() != test.wantError || provider.validations != test.validations {
					t.Fatalf("resume error = %v, validations = %d; want %q, %d", err, provider.validations, test.wantError, test.validations)
				}
				return
			}
			if err != nil || config.Backend != provider || config.Target.Backend != metadata.Name || selection.String() != "7" || !reflect.DeepEqual(environment, plan.Environment) || len(prepared.BackendPayloads) != 1 || !bytes.Equal(prepared.BackendPayloads[0].Data, provenance) || provider.validations != 1 {
				t.Fatalf("resume = %#v, selection = %s, prepared = %#v, error = %v, validations = %d", config, selection.String(), prepared, err, provider.validations)
			}
			if !bytes.Equal(config.Target.Scenario, scenario) {
				t.Fatalf("resumed scenario = %s", config.Target.Scenario)
			}
		})
	}
}

func TestResumeNativeStaticProfileFailsBeforeOtherPreflight(t *testing.T) {
	provider := &resumeProfileProvider{description: backend.Description{IOProfile: record.IOProfile{Name: "native-profile", ImplementationSHA256: record.HashBytes([]byte("current")), InventorySHA256: record.HashBytes([]byte("inventory"))}}}
	plan := campaign.CampaignPlan{RunnerBuild: "runner", Selection: "invalid", IOProfile: record.IOContract{Name: "native-profile", ImplementationSHA256: record.HashBytes([]byte("changed")), InventorySHA256: provider.description.IOProfile.InventorySHA256}}
	_, _, _, _, _, err := resumeConfiguration(campaignRequest{campaignRuntime: campaignRuntime{Backend: provider, RunnerBuild: "runner"}}, plan)
	if err == nil || err.Error() != "recorded I/O profile identity does not match the backend provider" || provider.validations != 0 {
		t.Fatalf("native profile error precedence = %v, validations = %d", err, provider.validations)
	}
}

func TestRunResumesVerifiedBatchAndSkipsCompletedOrdinals(t *testing.T) {
	preparer := newFakePreparer(t)
	interrupted := &resumeInterruptExecutor{}
	config, configDependencies := testConfig(t, preparer, interrupted, "7-9", PolicyAll, 1)
	ctx := cancelOnProgress(t, &config, func(progress CampaignEvent) bool { return progress.Succeeded == 1 })
	config.TerminateGrace = 10 * time.Millisecond
	partial, err := exploreWith(ctx, config, configDependencies)
	var hostError *HostError
	if !errors.As(err, &hostError) || hostError.Reason != "cancelled" {
		t.Fatalf("interrupted exploreWith() error = %v", err)
	}
	if seeds := interrupted.seeds(); !slices.Equal(seeds, []uint64{7, 8}) {
		t.Fatalf("interrupted seeds = %v", seeds)
	}

	resumedExecutor := &fakeExecutor{}
	resumed, err := exploreWith(context.Background(), CampaignSpec{
		ResumeCampaign: partial.CampaignPath, RunnerBuild: config.RunnerBuild, Backend: scriptProvider(),
	}, executionDependencies{executor: resumedExecutor},
	)
	if err != nil {
		t.Fatal(err)
	}
	if seeds := executorSeeds(resumedExecutor); !slices.Equal(seeds, []uint64{8, 9}) {
		t.Fatalf("resumed seeds = %v", seeds)
	}
	if resumed.CampaignPath != partial.CampaignPath || resumed.SelectionCount != 3 || resumed.Attempted != 3 || resumed.Succeeded != 3 || resumed.Failures != 0 || resumed.StopReason != StopSeedsExhausted || preparer.calls != 1 {
		t.Fatalf("resumed summary = %#v, preparation calls = %d", resumed, preparer.calls)
	}
	batch, err := campaign.OpenCampaign(resumed.CampaignPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(batch.Executions) != 3 || batch.Executions[0].Seed != 7 || batch.Executions[1].Seed != 8 || batch.Executions[2].Seed != 9 {
		t.Fatalf("batch runs = %#v", batch.Executions)
	}
}

func TestRunResumeRestoresSeenChoiceFeaturesBeforeNovelRetention(t *testing.T) {
	preparer := newFakePreparer(t)
	limit := choiceTraceLimit(t, 1)
	trace := completeChoiceTrace(t, preparer.prepared.BuildKey, limit, []choice.Record{{
		Ordinal: 0, Kind: choice.KindRunnable, Flags: choice.FlagDecision, SiteOffset: 24, Alternatives: 2, Selected: 0,
	}})
	interrupted := &choiceResumeInterruptExecutor{trace: trace}
	config, configDependencies := testConfig(t, preparer, interrupted, "7-8", PolicyAll, 1)
	ctx := cancelOnProgress(t, &config, func(progress CampaignEvent) bool { return progress.RetainedSuccesses == 1 })
	config.TerminateGrace = 10 * time.Millisecond
	config.Coverage = campaignoptions.CoverageChoice
	config.ChoiceTraceLimit = limit
	config.KeepSuccesses = KeepSuccessesNovel
	config.SuccessArtifactLimit = 2
	config.SuccessBytesLimit = 64 << 20
	partial, err := exploreWith(ctx, config, configDependencies)
	if err == nil || partial.RetainedSuccesses != 1 {
		t.Fatalf("interrupted summary = %#v, error = %v", partial, err)
	}

	resumedExecutor := &fakeExecutor{result: func(uint64) backend.ExecutionResult {
		result := processResult(0, "", "")
		result.IOTranscript = completeEmptyTranscript()
		result.ChoiceTrace = trace
		return result
	}}
	resumed, err := exploreWith(context.Background(), CampaignSpec{
		ResumeCampaign: partial.CampaignPath, RunnerBuild: config.RunnerBuild, Backend: scriptProvider(),
	}, executionDependencies{executor: resumedExecutor},
	)
	if err != nil {
		t.Fatal(err)
	}
	if resumed.RetainedSuccesses != 1 || len(resumed.SuccessArtifacts) != 1 {
		t.Fatalf("resumed summary = %#v", resumed)
	}
	batch, err := campaign.OpenCampaign(resumed.CampaignPath)
	if err != nil {
		t.Fatal(err)
	}
	if batch.Executions[1].SuccessArtifact != nil || len(batch.Executions[1].ChoiceFeatures) == 0 {
		t.Fatalf("resumed run = %#v", batch.Executions[1])
	}
}

func TestRunResumesGuidedBatchWithoutReselectingSeeds(t *testing.T) {
	corpus := filepath.Join(t.TempDir(), "corpus")
	replayer := &matchingReplayer{}
	interrupted := &resumeInterruptExecutor{}
	config, configDependencies := testConfig(t, newFakePreparer(t), interrupted, "7-8", PolicyAll, 1)
	ctx := cancelOnProgress(t, &config, func(progress CampaignEvent) bool { return progress.CorpusEntries == 1 })
	config.TerminateGrace = 10 * time.Millisecond
	config.Coverage = campaignoptions.CoverageSemantic
	config.Guide = true
	config.Corpus = corpus
	config.Replayer = replayer
	partial, err := exploreWith(ctx, config, configDependencies)
	if err == nil || partial.CorpusEntries != 1 {
		t.Fatalf("interrupted guided summary = %#v, error = %v", partial, err)
	}

	resumedExecutor := &fakeExecutor{result: func(uint64) backend.ExecutionResult {
		result := processResult(0, "", "")
		result.IOTranscript = completeEmptyTranscript()
		return result
	}}
	resumed, err := exploreWith(context.Background(), CampaignSpec{
		ResumeCampaign: partial.CampaignPath, RunnerBuild: config.RunnerBuild, Backend: scriptProvider(), Replayer: replayer,
	}, executionDependencies{executor: resumedExecutor},
	)
	if err != nil {
		t.Fatal(err)
	}
	if seeds := executorSeeds(resumedExecutor); !slices.Equal(seeds, []uint64{8}) || resumed.SelectionCount != 2 || resumed.CorpusEntries != 1 {
		t.Fatalf("resumed guided seeds = %v, summary = %#v", executorSeeds(resumedExecutor), resumed)
	}
}

// TestRunResumeRejectsChangedEvidence pins that resuming an interrupted
// Campaign fails when the evidence it recorded no longer matches.
func TestRunResumeRejectsChangedEvidence(t *testing.T) {
	for _, test := range []struct {
		name string
		// interrupt starts the Campaign and returns once it is interrupted.
		interrupt func(t *testing.T) (CampaignResult, CampaignSpec)
		// change alters the interrupted Campaign or the resume request.
		change  func(t *testing.T, partial CampaignResult, resume *CampaignSpec)
		wantErr string
	}{
		{
			name: "runner build",
			interrupt: func(t *testing.T) (CampaignResult, CampaignSpec) {
				config, configDependencies := testConfig(t, newFakePreparer(t), blockingExecutor{}, "1", PolicyAll, 1)
				ctx := cancelOnProgress(t, &config, func(progress CampaignEvent) bool { return progress.Running == 1 })
				config.TerminateGrace = 10 * time.Millisecond
				partial, err := exploreWith(ctx, config, configDependencies)
				if err == nil {
					t.Fatal("exploreWith() did not leave an interrupted batch")
				}
				return partial, config
			},
			change:  func(_ *testing.T, _ CampaignResult, resume *CampaignSpec) { resume.RunnerBuild = "sha256:changed" },
			wantErr: "Runner build identity",
		},
		{
			name: "tampered retained success",
			interrupt: func(t *testing.T) (CampaignResult, CampaignSpec) {
				interrupted := &resumeInterruptExecutor{}
				config, configDependencies := testConfig(t, newFakePreparer(t), interrupted, "7-8", PolicyAll, 1)
				ctx := cancelOnProgress(t, &config, func(progress CampaignEvent) bool { return progress.RetainedSuccesses == 1 })
				config.TerminateGrace = 10 * time.Millisecond
				config.KeepSuccesses = KeepSuccessesAll
				config.SuccessArtifactLimit = 2
				config.SuccessBytesLimit = 64 << 20
				partial, err := exploreWith(ctx, config, configDependencies)
				if err == nil || len(partial.SuccessArtifacts) != 1 {
					t.Fatalf("interrupted summary = %#v, error = %v", partial, err)
				}
				return partial, config
			},
			change: func(t *testing.T, partial CampaignResult, _ *CampaignSpec) {
				if err := os.WriteFile(filepath.Join(partial.SuccessArtifacts[0], "stdout"), []byte("tampered"), 0o600); err != nil {
					t.Fatal(err)
				}
			},
			wantErr: "retained success",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			partial, config := test.interrupt(t)
			resume := CampaignSpec{ResumeCampaign: partial.CampaignPath, RunnerBuild: config.RunnerBuild, Backend: scriptProvider()}
			test.change(t, partial, &resume)
			_, err := exploreWith(context.Background(), resume, executionDependencies{executor: &fakeExecutor{}})
			if err == nil || !strings.Contains(err.Error(), test.wantErr) {
				t.Fatalf("resume error = %v, want %q", err, test.wantErr)
			}
		})
	}
}

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
	report, err := inspect.Inspect(path, inspect.InspectOptions{})
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
							explored, _, err := inspectedRounds(interrupted.CampaignPath)
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
		explored, _, err := inspectedRounds(config.ResumeCampaign)
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

// seedStep scripts one seed's attempt.
type seedStep struct {
	kind  string // success, failure, watchdog, block (until cancelled), or finish (unless cancelled first)
	probe string
	delay time.Duration
}

// scriptedSeedAttempts runs each seed's step and records which seeds ran.
type scriptedSeedAttempts struct {
	t      *testing.T
	script map[uint64]seedStep
	mu     sync.Mutex
	ran    []uint64
}

func (executor *scriptedSeedAttempts) Run(ctx context.Context, request backend.ExecutionRequest) (backend.ExecutionResult, error) {
	seed := seedFromEnvironment(request.Env)
	executor.mu.Lock()
	executor.ran = append(executor.ran, seed)
	executor.mu.Unlock()
	step := executor.script[seed]
	cancelled := func() backend.ExecutionResult {
		result := processResult(0, "", "")
		result.Cancelled, result.Termination, result.Signal = true, backend.TerminationSignal, "killed"
		return result
	}
	if step.delay != 0 {
		<-time.After(step.delay)
	}
	switch step.kind {
	case "block":
		<-ctx.Done()
		return cancelled(), nil
	case "finish":
		select {
		case <-ctx.Done():
			return cancelled(), nil
		case <-time.After(20 * time.Millisecond):
		}
	case "failure":
		result := processResult(1, step.probe, "")
		result.IOTranscript = completeEmptyTranscript()
		return result, nil
	case "watchdog":
		result := processResult(0, "", "")
		result.Termination, result.Signal, result.WatchdogTimeout = backend.TerminationSignal, "SIGKILL", true
		result.IOTranscript = completeEmptyTranscript()
		return result, nil
	}
	result := processResult(0, "", "")
	result.IOTranscript = semanticTranscript(executor.t, step.probe)
	return result, nil
}

func (executor *scriptedSeedAttempts) seeds() []uint64 {
	executor.mu.Lock()
	defer executor.mu.Unlock()
	return slices.Clone(executor.ran)
}

// seedScenario is one scripted campaign whose logical outcome must not
// depend on where it is interrupted.
type seedScenario struct {
	name     string
	policy   FailurePolicy
	budget   uint64
	parallel int
	script   []seedStep
	want     []string
	counts   string
}

func seedScenarios() []seedScenario {
	return []seedScenario{
		{
			name: "all policy keeps every classification", policy: PolicyAll, parallel: 3,
			script: []seedStep{{kind: "success", probe: probeA}, {kind: "failure", probe: "x"}, {kind: "watchdog"}, {kind: "success", probe: probeB}, {kind: "failure", probe: "x"}, {kind: "success", probe: probeA}},
			want: []string{
				"0 success/success/exit kept novel=" + probeA, "1 target/nonzero_exit/exit failure=1", "2 watchdog/watchdog_timeout/timeout failure=2",
				"3 success/success/exit kept novel=" + probeB, "4 target/nonzero_exit/exit failure=1", "5 success/success/exit",
			},
			counts: "attempted=6 succeeded=3 failures=3 watchdogs=1 cancelled=0 distinct=2 retained=2 stop=seeds_exhausted",
		},
		{
			// A first failure at parallel three keeps its one real failure and
			// cancels the two attempts admitted beside it.
			name: "first failure cancels its admitted attempts", policy: PolicyFirst, parallel: 3,
			script: []seedStep{{kind: "success", probe: probeA}, {kind: "failure", probe: "x"}, {kind: "block"}, {kind: "block"}, {kind: "block"}, {kind: "block"}},
			want: []string{
				"0 success/success/exit kept novel=" + probeA, "1 target/nonzero_exit/exit failure=1",
				"2 runner/runner_cancelled/none", "3 runner/runner_cancelled/none",
			},
			counts: "attempted=4 succeeded=1 failures=1 watchdogs=0 cancelled=2 distinct=1 retained=1 stop=first_failure",
		},
		{
			// The later attempts finish before the lower failure that stops the
			// campaign; their real success and failure join its stop group.
			name: "first failure keeps buffered real outcomes", policy: PolicyFirst, parallel: 3,
			script: []seedStep{{kind: "failure", probe: "x", delay: 30 * time.Millisecond}, {kind: "success", probe: probeA}, {kind: "failure", probe: "y"}, {kind: "block"}, {kind: "block"}},
			want: []string{
				"0 target/nonzero_exit/exit failure=0", "1 success/success/exit kept novel=" + probeA, "2 target/nonzero_exit/exit failure=2",
			},
			counts: "attempted=3 succeeded=1 failures=2 watchdogs=0 cancelled=0 distinct=2 retained=1 stop=first_failure",
		},
		{
			// A failure budget lets the attempt admitted beside its trigger
			// finish; nothing is cancelled by the policy.
			name: "failure budget drains naturally", policy: PolicyBudget, budget: 2, parallel: 2,
			script: []seedStep{{kind: "failure", probe: "x"}, {kind: "success", probe: probeA}, {kind: "failure", probe: "x"}, {kind: "failure", probe: "y"}, {kind: "finish", probe: probeB}, {kind: "success", probe: probeC}},
			want: []string{
				"0 target/nonzero_exit/exit failure=0", "1 success/success/exit kept novel=" + probeA, "2 target/nonzero_exit/exit failure=0",
				"3 target/nonzero_exit/exit failure=3", "4 success/success/exit kept novel=" + probeB,
			},
			counts: "attempted=5 succeeded=2 failures=3 watchdogs=0 cancelled=0 distinct=2 retained=2 stop=failure_budget",
		},
	}
}

func (scenario seedScenario) config(t *testing.T) (CampaignSpec, *scriptedSeedAttempts) {
	t.Helper()
	executor := &scriptedSeedAttempts{t: t, script: map[uint64]seedStep{}}
	for index, step := range scenario.script {
		executor.script[uint64(index+1)] = step
	}
	config, _ := testConfig(t, newFakePreparer(t), executor, fmt.Sprintf("1-%d", len(scenario.script)), scenario.policy, scenario.parallel)
	config.Coverage = campaignoptions.CoverageSemantic
	keepSuccesses(&config, KeepSuccessesNovel)
	if scenario.budget != 0 {
		config.FailureBudget = scenario.budget
	}
	return config, executor
}

func (scenario seedScenario) resumeExecutor(t *testing.T) *scriptedSeedAttempts {
	_, executor := scenario.config(t)
	return executor
}

// observeLogicalSeedCampaign projects a published seed campaign onto what an
// interruption must not change: each ordinal's classification, retention and
// novelty, which earlier ordinal first had its failure signature, and the
// published counters.
func observeLogicalSeedCampaign(t *testing.T, path string) ([]string, string) {
	t.Helper()
	batch, err := campaign.OpenCampaign(path)
	if err != nil {
		t.Fatal(err)
	}
	runs := slices.Clone(batch.Executions)
	sort.Slice(runs, func(i, j int) bool { return runs[i].SelectionOrdinal < runs[j].SelectionOrdinal })
	first := map[record.SHA256]uint64{}
	var observed []string
	for _, run := range runs {
		line := fmt.Sprintf("%d %s/%s/%s", run.SelectionOrdinal, run.Domain, run.Reason, run.Termination)
		if run.SuccessArtifact != nil {
			line += " kept"
		}
		if len(run.NovelSemanticProbes) != 0 {
			line += " novel=" + strings.Join(run.NovelSemanticProbes, ",")
		}
		if run.FailureSignature != nil {
			if _, found := first[*run.FailureSignature]; !found {
				first[*run.FailureSignature] = uint64(run.SelectionOrdinal)
			}
			line += fmt.Sprintf(" failure=%d", first[*run.FailureSignature])
		}
		observed = append(observed, line)
	}
	summary := batch.Record
	counts := fmt.Sprintf("attempted=%d succeeded=%d failures=%d watchdogs=%d cancelled=%d distinct=%d retained=%d stop=%s",
		summary.Attempted, summary.Succeeded, summary.Failures, summary.Watchdogs, summary.Cancelled, summary.DistinctFailures, summary.RetainedSuccesses, summary.StopReason)
	return observed, counts
}

// campaignLogKinds reads the record kinds of a campaign log in order.
func campaignLogKinds(t *testing.T, path string) []logstore.Kind {
	t.Helper()
	kinds := map[logstore.Kind]uint32{}
	for kind := logstore.Kind(1); kind <= 17; kind++ {
		kinds[kind] = logstore.MaxPayloadBytes
	}
	var observed []logstore.Kind
	if _, err := logstore.Replay(filepath.Join(path, "store"), logstore.Options{Kinds: kinds}, func(entry logstore.Record) error {
		observed = append(observed, entry.Kind)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	return observed
}

const campaignAdmissionKind logstore.Kind = 6

// appendFault fails the index-th campaign log append (counted from one) at
// one of its boundaries: before the write, midway through it, after the
// complete frame before its sync, or after the sync before the append
// returns. "cancel" instead cancels the campaign's context once that append
// is durable.
type appendFault struct {
	index  int
	mode   string
	cancel context.CancelFunc

	mu       sync.Mutex
	appends  int
	injected bool
	late     int
}

var errInjectedAppend = errors.New("injected campaign log append failure")

func (fault *appendFault) hook(kind, operation string) error {
	if operation != "append" {
		return nil
	}
	fault.mu.Lock()
	defer fault.mu.Unlock()
	if kind == "log-write" {
		if fault.injected && fault.mode != "cancel" {
			fault.late++
		}
		fault.appends++
	}
	if fault.appends != fault.index || fault.injected {
		return nil
	}
	switch {
	case kind == "log-write" && fault.mode == "write":
		fault.injected = true
		return errInjectedAppend
	case kind == "log-write" && fault.mode == "partial":
		fault.injected = true
		return &logstore.PartialWrite{Bytes: 9}
	case kind == "log-sync" && fault.mode == "sync":
		fault.injected = true
		return errInjectedAppend
	case kind == "log-synced" && fault.mode == "synced":
		fault.injected = true
		return errInjectedAppend
	case kind == "log-synced" && fault.mode == "cancel":
		fault.injected = true
		fault.cancel()
	}
	return nil
}

// durableOrdinals are the ordinals a resumed campaign must not run again:
// those with a committed outcome other than host failure evidence, and those
// with a durable receipt.
func durableOrdinals(t *testing.T, path string) (map[uint64]struct{}, campaign.CommittedState) {
	t.Helper()
	state, err := campaign.ReadCommittedState(path)
	if err != nil {
		t.Fatal(err)
	}
	requireInspectedFrontier(t, path, state)
	durable := map[uint64]struct{}{}
	for _, run := range state.Executions {
		if run.Domain != "runner" || run.Reason == "runner_cancelled" {
			durable[uint64(run.SelectionOrdinal)] = struct{}{}
		}
	}
	for ordinal := range state.Frontier.Receipts {
		durable[ordinal] = struct{}{}
	}
	return durable, state
}

// requireInspectedFrontier checks that public inspection reports the
// committed outcomes and the admission frontier the campaign log replays to.
func requireInspectedFrontier(t *testing.T, path string, state campaign.CommittedState) {
	t.Helper()
	report, err := inspect.Inspect(path, inspect.InspectOptions{})
	if err != nil {
		t.Fatal(err)
	}
	committed := report.Committed
	if committed == nil || committed.Outcomes != uint64(len(state.Executions)) || len(committed.Admitted) != len(state.Frontier.Outstanding) ||
		committed.Available != state.Frontier.Available || committed.PendingStop != string(state.Frontier.PendingStop) || committed.Stopped != string(state.Frontier.Stopped) {
		t.Fatalf("inspected committed state = %#v, log replays to %#v", committed, state.Frontier)
	}
	for index, admitted := range committed.Admitted {
		job := state.Frontier.Outstanding[index]
		receipt, received := state.Frontier.Receipts[job.Ordinal]
		if admitted.Ordinal != job.Ordinal || admitted.Seed != job.Seed || (admitted.Receipt != nil) != received || received && (admitted.Receipt.Domain != receipt.Observation.Domain || len(admitted.Receipt.Artifacts) != len(receipt.Artifacts)) {
			t.Fatalf("inspected admission %d = %#v, log replays to %#v %#v", index, admitted, job, receipt)
		}
	}
}

// classifiedOutcomes counts the durable successes and failures. A host
// failure's cancelled attempts are counted as cancelled without committing a
// classified outcome, so cancellations are not compared.
func classifiedOutcomes(state campaign.CommittedState) uint64 {
	var classified uint64
	for _, run := range state.Executions {
		if run.Domain != "runner" {
			classified++
		}
	}
	return classified
}

// resumeSeedCampaign resumes an interrupted campaign until it publishes. No
// resumed attempt reruns an ordinal whose outcome or receipt was durable,
// and a committed stop launches nothing.
func resumeSeedCampaign(t *testing.T, scenario seedScenario, config CampaignSpec, path string) {
	t.Helper()
	for range 3 {
		durable, state := durableOrdinals(t, path)
		if state.Lifecycle.Published {
			return
		}
		executor := scenario.resumeExecutor(t)
		_, err := exploreWith(context.Background(), CampaignSpec{ResumeCampaign: path, RunnerBuild: config.RunnerBuild, Backend: scriptProvider()}, executionDependencies{executor: executor})
		ran := executor.seeds()
		for _, seed := range ran {
			if _, found := durable[seed-1]; found {
				t.Fatalf("resume reran ordinal %d whose outcome or receipt was durable (ran %v)", seed-1, ran)
			}
		}
		if state.Frontier.Stopped != "" && len(ran) != 0 {
			t.Fatalf("resume after a committed %s stop ran %v", state.Frontier.Stopped, ran)
		}
		if err == nil {
			return
		}
		t.Logf("resume failed: %v", err)
	}
	t.Fatal("campaign did not publish after three resumes")
}

// Every seed campaign log append from the first admission through
// publication fails in turn at each of its boundaries, or has the host
// cancel the campaign once it is durable. Each interrupted campaign resumes
// to the uninterrupted logical outcome, and a failed append neither advances
// the reported committed progress nor lets another append through.
func TestSeedCampaignResumesEveryAppendFaultToTheUninterruptedOutcome(t *testing.T) {
	for _, scenario := range seedScenarios() {
		t.Run(scenario.name, func(t *testing.T) {
			t.Parallel()
			config, executor := scenario.config(t)
			reference, err := exploreWith(context.Background(), config, executionDependencies{executor: executor})
			if err != nil {
				t.Fatal(err)
			}
			want, wantCounts := observeLogicalSeedCampaign(t, reference.CampaignPath)
			if !reflect.DeepEqual(want, scenario.want) || wantCounts != scenario.counts {
				t.Fatalf("uninterrupted campaign = %q %s, want %q %s", want, wantCounts, scenario.want, scenario.counts)
			}
			kinds := campaignLogKinds(t, reference.CampaignPath)
			first := slices.Index(kinds, campaignAdmissionKind) + 1
			for index := first; index <= len(kinds); index++ {
				for _, mode := range []string{"write", "partial", "sync", "synced", "cancel"} {
					t.Run(fmt.Sprintf("append %d %s", index, mode), func(t *testing.T) {
						config, executor := scenario.config(t)
						ctx, cancel := context.WithCancel(context.Background())
						defer cancel()
						fault := &appendFault{index: index, mode: mode, cancel: cancel}
						config.TerminateGrace = 10 * time.Millisecond
						interrupted, err := exploreWith(campaign.WithStoreFaults(ctx, fault.hook), config, executionDependencies{executor: executor})
						if !fault.injected {
							t.Fatalf("append %d was not reached (%d appends)", index, fault.appends)
						}
						if err == nil {
							t.Fatal("interrupted campaign succeeded")
						}
						if fault.late != 0 {
							t.Fatalf("%d appends reached the log after the failed append", fault.late)
						}
						_, state := durableOrdinals(t, interrupted.CampaignPath)
						// A partial append leaves its torn bytes for the next
						// writer; inspection observes them without repair.
						report, err := inspect.Inspect(interrupted.CampaignPath, inspect.InspectOptions{})
						if err != nil {
							t.Fatal(err)
						}
						wantTail, wantTailBytes := "none", uint64(0)
						if mode == "partial" {
							wantTail, wantTailBytes = "short-frame", 9
						}
						if report.Log.Tail != wantTail || report.Log.TailBytes != wantTailBytes || report.Log.RepairedBytes != 0 {
							t.Fatalf("interrupted log inspection = %#v", report.Log)
						}
						reported := interrupted.Succeeded + interrupted.Failures
						durable := classifiedOutcomes(state)
						if reported > durable || (mode == "write" || mode == "partial") && reported != durable {
							t.Fatalf("reported %d classified outcomes; %d are durable", reported, durable)
						}
						resumeSeedCampaign(t, scenario, config, interrupted.CampaignPath)
						got, gotCounts := observeLogicalSeedCampaign(t, interrupted.CampaignPath)
						if !reflect.DeepEqual(got, want) || gotCounts != wantCounts {
							t.Fatalf("resumed campaign = %q %s, want %q %s", got, gotCounts, want, wantCounts)
						}
					})
				}
			}
		})
	}
}

// A resumed campaign whose first resume is interrupted too still converges:
// each resume reuses what the previous ones made durable.
func TestSeedCampaignRepeatedResumeConverges(t *testing.T) {
	scenario := seedScenarios()[1]
	config, executor := scenario.config(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	fault := &appendFault{index: 7, mode: "sync", cancel: cancel}
	interrupted, err := exploreWith(campaign.WithStoreFaults(ctx, fault.hook), config, executionDependencies{executor: executor})
	if err == nil || !fault.injected {
		t.Fatalf("interrupted campaign error = %v, injected = %t", err, fault.injected)
	}
	for _, index := range []int{2, 3} {
		resumeFault := &appendFault{index: index, mode: "partial"}
		_, err := exploreWith(campaign.WithStoreFaults(context.Background(), resumeFault.hook), CampaignSpec{ResumeCampaign: interrupted.CampaignPath, RunnerBuild: config.RunnerBuild, Backend: scriptProvider()}, executionDependencies{executor: scenario.resumeExecutor(t)})
		if err == nil && resumeFault.injected {
			t.Fatalf("resume interrupted at append %d succeeded", index)
		}
	}
	resumeSeedCampaign(t, scenario, config, interrupted.CampaignPath)
	got, counts := observeLogicalSeedCampaign(t, interrupted.CampaignPath)
	if !reflect.DeepEqual(got, scenario.want) || counts != scenario.counts {
		t.Fatalf("resumed campaign = %q %s, want %q %s", got, counts, scenario.want, scenario.counts)
	}
}

// A durable receipt whose published payload is missing is corruption: resume
// fails before it reclaims anything and never runs the attempt again.
func TestSeedCampaignResumeRefusesAMissingReceiptPayload(t *testing.T) {
	scenario := seedScenarios()[2]
	config, executor := scenario.config(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var received campaign.Receipt
	var path string
	fault := &appendFault{mode: "none"}
	hook := func(kind, operation string) error {
		if kind != "log-synced" || operation != "append" || path == "" {
			return fault.hook(kind, operation)
		}
		state, err := campaign.ReadCommittedState(path)
		if err == nil {
			for _, receipt := range state.Frontier.Receipts {
				if receipt.Observation.Artifact != nil {
					received = receipt
					cancel()
				}
			}
		}
		return nil
	}
	config.Progress = func(event CampaignEvent) error {
		if path == "" {
			path = event.CampaignPath
		}
		return nil
	}
	interrupted, err := exploreWith(campaign.WithStoreFaults(ctx, hook), config, executionDependencies{executor: executor})
	if err == nil || received.Observation.Artifact == nil {
		t.Fatalf("campaign was not interrupted after a failure receipt: %v", err)
	}
	missing := filepath.Join(interrupted.CampaignPath, filepath.FromSlash(*received.Observation.Artifact))
	if err := os.RemoveAll(missing); err != nil {
		t.Fatal(err)
	}
	before := ownedPayloadNames(t, interrupted.CampaignPath)
	resumed := scenario.resumeExecutor(t)
	_, err = exploreWith(context.Background(), CampaignSpec{ResumeCampaign: interrupted.CampaignPath, RunnerBuild: config.RunnerBuild, Backend: scriptProvider()}, executionDependencies{executor: resumed})
	if err == nil || !campaign.IsIntegrityError(err) {
		t.Fatalf("resume with a missing receipt payload error = %v", err)
	}
	if ran := resumed.seeds(); len(ran) != 0 {
		t.Fatalf("resume ran %v", ran)
	}
	if after := ownedPayloadNames(t, interrupted.CampaignPath); !reflect.DeepEqual(after, before) {
		t.Fatalf("resume reclaimed payloads before failing: %v -> %v", before, after)
	}
}

func ownedPayloadNames(t *testing.T, path string) []string {
	t.Helper()
	var names []string
	for _, root := range []string{"failures", "successes", "diagnostics"} {
		entries, err := os.ReadDir(filepath.Join(path, root))
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			t.Fatal(err)
		}
		for _, entry := range entries {
			names = append(names, root+"/"+entry.Name())
		}
	}
	return names
}
