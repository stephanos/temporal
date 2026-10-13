package backend

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"go.temporal.io/server/tools/gomad3/backends"
	"go.temporal.io/server/tools/gomad3/record"
	"go.temporal.io/server/tools/gomad3/runner"
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
	workingDir, err := filepath.Abs("../../../../..")
	if err != nil {
		t.Fatal(err)
	}
	for _, mode := range []string{"own", "late"} {
		t.Run(mode, func(t *testing.T) {
			for _, reportOnly := range []bool{false, true} {
				campaign, err := runner.Explore(t.Context(), runner.CampaignSpec{
					Backend: provider, Seeds: "7,19", Parallel: 1, ExecutionTimeout: time.Minute, OverallTimeout: 3 * time.Minute,
					TerminateGrace: 100 * time.Millisecond, OnFailure: runner.PolicyAll, FailureBudget: 1, OutputLimit: 1 << 20, WorldTransitionLimit: 1 << 20,
					Artifacts: t.TempDir(), RunnerBuild: "soft-assertion-fixture", Coverage: runner.CoverageNone,
					KeepSuccesses: runner.KeepSuccessesAll, SuccessArtifactLimit: 2, SuccessBytesLimit: 128 << 20,
					Target: target.Spec{Backend: Name, Kind: target.KindGoTest, Source: "./tools/gomad3sim/testdata/softassert", WorkingDir: workingDir,
						SoftAssertionReportOnly: reportOnly, BuildTags: []string{"test_dep"}, Args: []string{"-test.v", "-assertion-mode=" + mode}},
				})
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
