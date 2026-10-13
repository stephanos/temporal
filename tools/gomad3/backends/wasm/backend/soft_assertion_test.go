package backend

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"go.temporal.io/server/tools/gomad3/backends"
	"go.temporal.io/server/tools/gomad3/record"
	"go.temporal.io/server/tools/gomad3/runner"
	"go.temporal.io/server/tools/gomad3/runner/inspect"
	"go.temporal.io/server/tools/gomad3/target"
)

func TestSoftAssertionOverridesSuccessfulGuestExit(t *testing.T) {
	for _, reportOnly := range []bool{false, true} {
		result := backends.ExecutionResult{Captured: true, GroupGone: true, Termination: backends.TerminationExit, Stdout: backends.Output{Bytes: []byte("PASS\nfailed assertion: invalid state\n")}}
		result.SoftAssertions = observeSoftAssertions(result, reportOnly)
		classified := backends.Classify(result, false, record.WorldTerminal{})
		want := "soft_assertion"
		if reportOnly {
			want = "success"
		}
		if classified.Reason != want || classified.SoftAssertions == nil || len(classified.SoftAssertions.Messages) != 1 || classified.SoftAssertions.Messages[0] != "invalid state" {
			t.Fatalf("classification = %+v", classified)
		}
	}
}

func TestSoftAssertionActualPassingGuestArtifactReplay(t *testing.T) {
	provider := integrationProvider(t)
	provider.options.CompileCacheRoot = t.TempDir()
	if err := os.Chmod(provider.options.CompileCacheRoot, 0700); err != nil {
		t.Fatal(err)
	}
	workingDir, err := filepath.Abs("../../../../..")
	if err != nil {
		t.Fatal(err)
	}
	for _, mode := range []string{"own", "late"} {
		t.Run(mode, func(t *testing.T) {
			for _, reportOnly := range []bool{false, true} {
				var reports []runner.CampaignEvent
				config := runner.CampaignSpec{
					Backend: provider, Seeds: "7,19", Parallel: 1, ExecutionTimeout: 5 * time.Minute, OverallTimeout: 10 * time.Minute,
					TerminateGrace: 100 * time.Millisecond, OnFailure: runner.PolicyAll, FailureBudget: 1, OutputLimit: 1 << 20, WorldTransitionLimit: 1 << 20,
					Artifacts: t.TempDir(), RunnerBuild: "soft-assertion-fixture", Coverage: runner.CoverageNone,
					KeepSuccesses: runner.KeepSuccessesAll, SuccessArtifactLimit: 2, SuccessBytesLimit: 128 << 20,
					Progress: func(event runner.CampaignEvent) error {
						if event.SoftAssertions != nil {
							reports = append(reports, event)
						}
						return nil
					},
					Target: target.Spec{Backend: Name, Kind: target.KindGoTest, Source: "./tools/gomad3sim/testdata/softassert", WorkingDir: workingDir,
						SoftAssertionReportOnly: reportOnly, BuildTags: []string{"test_dep"}, Args: []string{"-test.v", "-assertion-mode=" + mode}},
				}
				campaign, err := runner.Explore(t.Context(), config)
				if err != nil {
					t.Fatal(err)
				}
				paths := campaign.Artifacts
				if reportOnly {
					if campaign.Failures != 0 {
						t.Fatalf("opted-out campaign failed: %+v", campaign)
					}
					paths = campaign.SuccessArtifacts
				} else {
					if campaign.Failures != 2 || len(paths) != 1 {
						t.Fatalf("static assertions did not deduplicate: %+v", campaign)
					}
				}
				if len(paths) == 0 {
					t.Fatal("campaign did not retain its evidence")
				}
				for _, path := range paths {
					replayed, err := runner.Replay(t.Context(), runner.ReplaySpec{ArtifactPath: path, Backend: provider})
					if err != nil || !replayed.Match {
						t.Fatalf("artifact replay: %+v %v", replayed, err)
					}
					manifest := replayed.Artifact.Manifest
					if manifest.Target.SoftAssertionReportOnly != reportOnly || !reflect.DeepEqual(&record.SoftAssertions{Messages: []string{"invalid fixture state"}, ReportOnly: reportOnly}, manifest.Outcome.SoftAssertions) || !reportOnly && manifest.Outcome.Reason != "soft_assertion" {
						t.Fatalf("oracle identity/observation mismatch: %+v", manifest)
					}
					stdout, err := os.ReadFile(filepath.Join(path, "stdout"))
					if err != nil || !strings.Contains(string(stdout), "PASS") {
						t.Fatalf("guest did not pass: %q %v", stdout, err)
					}
					t.Logf("mode=%s report_only=%t artifact=%s signature=%s replay_match=%t", mode, reportOnly, path, manifest.Outcome.FailureSignature, replayed.Match)
				}
				if reportOnly {
					if len(reports) != 2 || *reports[0].ExecutionSeed != 7 || *reports[1].ExecutionSeed != 19 {
						t.Fatalf("opted-out assertions were not reported in seed order: %+v", reports)
					}
					config.KeepSuccesses, config.SuccessArtifactLimit, config.SuccessBytesLimit = runner.KeepSuccessesNone, 0, 0
					config.Artifacts, reports = t.TempDir(), nil
					discarded, err := runner.Explore(t.Context(), config)
					if err != nil || discarded.Succeeded != 2 || len(discarded.SuccessArtifacts) != 0 || len(reports) != 2 {
						t.Fatalf("unretained opted-out assertions were not reported: %+v %+v %v", discarded, reports, err)
					}
					config.OutputLimit, config.Artifacts = 8, t.TempDir()
					truncated, err := runner.Explore(t.Context(), config)
					var failure *backends.FailureError
					if !errors.As(err, &failure) || failure.Termination != backends.Capacity || failure.SoftAssertions == nil || !failure.SoftAssertions.Inconclusive || !failure.SoftAssertions.ReportOnly || len(truncated.Artifacts) != 0 {
						t.Fatalf("truncated opted-out run lost capacity/inconclusive truth: %+v %v", truncated, err)
					}
				}
			}
		})
	}
}

func TestSoftAssertionReportOnlyUnsupportedWorkloadRefusedBeforePreparation(t *testing.T) {
	p := &Provider{}
	for _, kind := range []target.Kind{target.KindGoRun, target.KindExec} {
		_, err := p.Prepare(t.Context(), target.Spec{Backend: Name, Kind: kind, SoftAssertionReportOnly: true})
		if err == nil || !strings.Contains(err.Error(), "requires a prepared Go test") {
			t.Fatalf("unsupported preparation admitted: %v", err)
		}
		_, err = p.Describe(t.Context(), backends.Configuration{Target: target.Spec{Kind: kind, SoftAssertionReportOnly: true}})
		if err == nil || !strings.Contains(err.Error(), "requires a prepared Go test") {
			t.Fatalf("unsupported execution admitted: %v", err)
		}
	}
}

func TestSoftAssertionChoiceCampaignReportsCommittedRounds(t *testing.T) {
	provider := explorationProvider(t)
	workingDir, err := filepath.Abs("../../../../..")
	if err != nil {
		t.Fatal(err)
	}
	var reports []runner.CampaignEvent
	config := runner.CampaignSpec{Backend: provider, Seeds: "7", Strategy: runner.StrategyChoiceExploration, Parallel: 1,
		ExecutionTimeout: 5 * time.Minute, OverallTimeout: 10 * time.Minute, TerminateGrace: 100 * time.Millisecond, OnFailure: runner.PolicyAll, FailureBudget: 1,
		OutputLimit: 1 << 20, WorldTransitionLimit: 1 << 20, ChoiceTraceLimit: 1 << 20, MaxExecutions: 2, MaxChoiceDepth: 4, MaxExplorationBytes: 8 << 20,
		Artifacts: t.TempDir(), RunnerBuild: "soft-assertion-choice", Coverage: runner.CoverageChoice,
		Target: target.Spec{Backend: Name, Kind: target.KindGoTest, Source: "./tools/gomad3sim/testdata/softassert", WorkingDir: workingDir,
			SoftAssertionReportOnly: true, BuildTags: []string{"test_dep"}, Args: []string{"-test.v", "-assertion-mode=late"}},
		Progress: func(event runner.CampaignEvent) error {
			if event.SoftAssertions == nil {
				return nil
			}
			observed, err := inspect.Inspect(event.CampaignPath, inspect.InspectOptions{})
			if err != nil {
				return err
			}
			if observed.Committed == nil || observed.Committed.Outcomes <= uint64(len(reports)) {
				return errors.New("soft assertion leaked before committed outcome")
			}
			reports = append(reports, event)
			return nil
		},
	}
	result, err := runner.Explore(t.Context(), config)
	if err != nil || result.Attempted == 0 || len(reports) != int(result.Attempted) || result.Failures != 0 || len(result.SuccessArtifacts) != 0 {
		t.Fatalf("choice oracle reporting: %+v reports=%+v %v", result, reports, err)
	}
	observed, err := inspect.Inspect(result.CampaignPath, inspect.InspectOptions{})
	if err != nil || observed.Campaign == nil || len(observed.Campaign.Executions) != len(reports) {
		t.Fatalf("completed choice inspection: %+v %v", observed, err)
	}
	for index, execution := range observed.Campaign.Executions {
		if !reflect.DeepEqual(execution.SoftAssertions, reports[index].SoftAssertions) {
			t.Fatalf("committed report %d was lost: %+v", index, execution)
		}
	}
}

func TestSoftAssertionTruncationIsInconclusiveEvenWithReportOnly(t *testing.T) {
	for _, reportOnly := range []bool{false, true} {
		result := backends.ExecutionResult{Captured: true, GroupGone: true, Termination: backends.TerminationExit, Stdout: backends.Output{Bytes: []byte("PASS\n"), Truncated: true}}
		result.SoftAssertions = observeSoftAssertions(result, reportOnly)
		classified := backends.Classify(result, false, record.WorldTerminal{})
		if classified.Reason != "soft_assertion_inconclusive" || classified.Domain == "success" {
			t.Fatalf("truncated evidence passed: %+v", classified)
		}
	}
}
