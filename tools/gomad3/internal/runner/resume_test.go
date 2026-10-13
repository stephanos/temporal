package runner

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	campaignoptions "go.temporal.io/server/tools/gomad3/internal/runner/options"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	backend "go.temporal.io/server/tools/gomad3/internal/backends"
	"go.temporal.io/server/tools/gomad3/internal/backends/target"
	"go.temporal.io/server/tools/gomad3/internal/evidence/record"
	"go.temporal.io/server/tools/gomad3/internal/model/choice"
	"go.temporal.io/server/tools/gomad3/internal/runner/internal/campaign"
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
			provenance, err := json.Marshal(resumeProfileProvenance{Options: "retained", TargetSHA256: record.HashBytes(module), Args: []string{"fixed"}})
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
				Prepared:    campaign.PreparedTargetPlan{Path: "prepared/target", Target: record.Target{Backend: metadata, Kind: "go-run", Source: "example.com/workload", SHA256: record.HashBytes(module), Size: record.Uint64String(len(module)), Argv: []string{"gomad3-target", "fixed"}}},
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
