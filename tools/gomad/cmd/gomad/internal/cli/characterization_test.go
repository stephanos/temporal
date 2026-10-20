package cli

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"

	"go.temporal.io/server/tools/gomad/internal/backends/target"
	qualificationworkload "go.temporal.io/server/tools/gomad/internal/backends/wasm/qualification"
	"go.temporal.io/server/tools/gomad/internal/evidence/artifact"
	"go.temporal.io/server/tools/gomad/internal/evidence/qualification"
	"go.temporal.io/server/tools/gomad/internal/evidence/record"
	"go.temporal.io/server/tools/gomad/internal/runner"
	campaignoptions "go.temporal.io/server/tools/gomad/internal/runner/options"
)

// Characterization of the gomad command grammar, defaults, output routing and
// exit statuses. These tests pin observable CLI behavior before application
// construction is refactored, and must keep passing unchanged afterwards.
// Dependencies are built only through the fake installation in this file.

type commandResult struct {
	status int
	stdout string
	stderr string
}

func (result commandResult) String() string {
	return fmt.Sprintf("status=%d\n--- stdout\n%s--- stderr\n%s", result.status, result.stdout, result.stderr)
}

func runCommand(command func(stdout, stderr *bytes.Buffer) int) commandResult {
	var stdout, stderr bytes.Buffer
	status := command(&stdout, &stderr)
	return commandResult{status: status, stdout: stdout.String(), stderr: stderr.String()}
}

func runGomad(arguments ...string) commandResult {
	return runCommand(func(stdout, stderr *bytes.Buffer) int { return Run(arguments, stdout, stderr) })
}

// TestCharacterizeInputRejection pins malformed input, explicit zero values and
// flags irrelevant to the selected mode. Every case also names an unusable
// --toolchain-root, so it pins that input validation precedes installation
// resolution.

// TestCharacterizeInstallationResolution pins how each command reports an
// installation it cannot resolve, from the flag and from the environment.

func TestCharacterizeExploreOutputAndStatus(t *testing.T) {
	success := runner.CampaignResult{CampaignPath: "/artifacts/v1/campaign", SelectionCount: 2, Attempted: 2, Succeeded: 2, StopReason: runner.StopSeedsExhausted}
	failure := runner.CampaignResult{CampaignPath: "/artifacts/v1/campaign", SelectionCount: 2, Attempted: 2, Succeeded: 1, Failures: 1, DistinctFailures: 1, StopReason: runner.StopFirstFailure, Artifacts: []string{"/artifacts/v1/failure one"}}
	for _, test := range []struct {
		name      string
		arguments []string
		result    runner.CampaignResult
		err       error
		want      commandResult
	}{
		{"text success", nil, success, nil, commandResult{status: 0, stdout: "gomad: classification=success attempted=2 succeeded=2 failures=0 watchdogs=0 replay-divergences=0 distinct=0 retained-successes=0 retained-success-bytes=0 stop=seeds_exhausted artifact=/artifacts/v1/campaign\n"}},
		{"JSON success", []string{"--json"}, success, nil, commandResult{status: 0, stdout: `{"schema":"gomad.explore-event/v3","type":"result","classification":"success","campaign_path":"/artifacts/v1/campaign","selected":2,"attempted":2,"succeeded":2,"stop_reason":"seeds_exhausted"}` + "\n"}},
		{"text failure", nil, failure, nil, commandResult{status: 1, stdout: "gomad: classification=target_failure attempted=2 succeeded=1 failures=1 watchdogs=0 replay-divergences=0 distinct=1 retained-successes=0 retained-success-bytes=0 stop=first_failure artifact=/artifacts/v1/campaign\ngomad: retained failure: /artifacts/v1/failure one\ngomad: replay: gomad replay '/artifacts/v1/failure one'\n"}},
		{"JSON failure", []string{"--json"}, failure, nil, commandResult{status: 1, stdout: `{"schema":"gomad.explore-event/v3","type":"result","classification":"target_failure","campaign_path":"/artifacts/v1/campaign","selected":2,"attempted":2,"succeeded":1,"failures":1,"novelty":1,"stop_reason":"first_failure"}` + "\n" + `{"schema":"gomad.explore-event/v3","type":"artifact","classification":"target_failure","path":"/artifacts/v1/failure one","replay_command":"gomad replay '/artifacts/v1/failure one'"}` + "\n"}},
		{"invalid input error", nil, runner.CampaignResult{}, errors.New("bad request"), commandResult{status: 2, stderr: "gomad: invalid_input: bad request\n"}},
		{"runner failure", nil, runner.CampaignResult{}, &runner.HostError{Reason: "prepare", Err: errors.New("disk")}, commandResult{status: 3, stderr: "gomad: runner_failure: " + (&runner.HostError{Reason: "prepare", Err: errors.New("disk")}).Error() + "\n"}},
		{"cancelled", []string{"--json"}, runner.CampaignResult{}, &runner.HostError{Reason: "cancelled"}, commandResult{status: 3, stdout: `{"schema":"gomad.explore-event/v3","type":"error","classification":"cancelled","message":"gomad Runner/host failure: cancelled"}` + "\n"}},
		{"semantic coverage failure", nil, runner.CampaignResult{}, &record.MissingSemanticProbesError{Probes: []string{"probe"}}, commandResult{status: 1, stderr: "gomad: semantic_coverage_failure: required semantic probes were not observed: probe\n"}},
		{"unsupported target", nil, runner.CampaignResult{}, &target.UnsupportedCapabilityError{ImportPath: "example.com/p", Capability: "network"}, commandResult{status: 2, stderr: "gomad: unsupported_target: " + (&target.UnsupportedCapabilityError{ImportPath: "example.com/p", Capability: "network"}).Error() + "\n"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			dependencies := newFakeInstallation().exploreDependencies(func(context.Context, runner.CampaignSpec) (runner.CampaignResult, error) {
				return test.result, test.err
			}, nil)
			got := runCommand(func(stdout, stderr *bytes.Buffer) int {
				return runExploreWith(append(append([]string(nil), test.arguments...), "go-run", "./cmd"), stdout, stderr, dependencies)
			})
			if got != test.want {
				t.Fatalf("explore:\n%s\nwant:\n%s", got, test.want)
			}
		})
	}
	installationFailure := newFakeInstallation().failing(errFakeInstallation)
	if got, want := runCommand(func(stdout, stderr *bytes.Buffer) int {
		return runExploreWith([]string{"go-run", "./cmd"}, stdout, stderr, installationFailure.exploreDependencies(nil, nil))
	}), (commandResult{status: 3, stderr: "gomad: runner_failure: " + errFakeInstallation.Error() + "\n"}); got != want {
		t.Fatalf("explore installation failure = %s, want %s", got, want)
	}
}

func TestCharacterizePlanRequestAndOutput(t *testing.T) {
	var observed []runner.CampaignPlanSpec
	planned := runner.CampaignPlanResult{Path: "/plans/plan.json", BundlePath: "/plans/bundle", SHA256: "sha256:plan", SelectionCount: 4, TargetSHA256: "sha256:target"}
	plan := func(_ context.Context, spec runner.CampaignPlanSpec) (runner.CampaignPlanResult, error) {
		spec.Campaign.Progress = nil
		observed = append(observed, spec)
		return planned, nil
	}
	dependencies := newFakeInstallation().exploreDependencies(nil, plan)
	if got, want := runCommand(func(stdout, stderr *bytes.Buffer) int {
		return runPlanWith([]string{"--output=/plans/plan.json", "--count=4", "--env=A=1", "--build-tag=tag", "go-run", "./cmd", "--", "argument"}, stdout, stderr, dependencies)
	}), (commandResult{status: 0, stdout: "gomad plan: path=/plans/plan.json bundle=/plans/bundle sha256=sha256:plan selected=4 target=sha256:target\n"}); got != want {
		t.Fatalf("plan = %s, want %s", got, want)
	}
	if got, want := runCommand(func(stdout, stderr *bytes.Buffer) int {
		return runPlanWith([]string{"--json", "--output=/plans/plan.json", "--on-failure=first", "go-run", "./cmd"}, stdout, stderr, dependencies)
	}), (commandResult{status: 0, stdout: `{"path":"/plans/plan.json","bundle_path":"/plans/bundle","sha256":"sha256:plan","selection_count":4,"target_sha256":"sha256:target"}` + "\n"}); got != want {
		t.Fatalf("plan JSON = %s, want %s", got, want)
	}
	if len(observed) != 2 {
		t.Fatalf("plan requests = %d", len(observed))
	}
	first := observed[0]
	if first.Output != "/plans/plan.json" || first.Campaign.OnFailure != runner.PolicyAll || first.Campaign.Seeds != "0-3" || !reflect.DeepEqual(first.Campaign.Environment, []string{"A=1"}) ||
		!reflect.DeepEqual(first.Campaign.Target.BuildTags, []string{"tag"}) || !reflect.DeepEqual(first.Campaign.Target.Args, []string{"argument"}) ||
		first.Campaign.RunnerBuild != "sha256:runner" || first.Campaign.Target.ToolchainRoot != "/toolchain" {
		t.Fatalf("plan request = %#v", first)
	}
	// An explicit --on-failure follows plan's implicit --on-failure=all.
	if observed[1].Campaign.OnFailure != runner.PolicyFirst {
		t.Fatalf("plan failure policy = %q", observed[1].Campaign.OnFailure)
	}
	// Plan requires --output only after the installation is resolved.
	if got, want := runCommand(func(stdout, stderr *bytes.Buffer) int {
		return runPlanWith([]string{"go-run", "./cmd"}, stdout, stderr, dependencies)
	}), (commandResult{status: 2, stderr: "gomad: invalid_input: gomad plan requires --output FILE\n"}); got != want {
		t.Fatalf("plan without output = %s, want %s", got, want)
	}
	failing := newFakeInstallation().exploreDependencies(nil, func(context.Context, runner.CampaignPlanSpec) (runner.CampaignPlanResult, error) {
		return runner.CampaignPlanResult{}, &runner.HostError{Reason: "plan", Err: errors.New("write")}
	})
	if got := runCommand(func(stdout, stderr *bytes.Buffer) int {
		return runPlanWith([]string{"--json", "--output=/plan", "go-run", "./cmd"}, stdout, stderr, failing)
	}); got.status != 3 || got.stderr != "" || !strings.Contains(got.stdout, `"classification":"runner_failure"`) {
		t.Fatalf("plan failure = %s", got)
	}
}

func TestCharacterizeReplayRequestOutputAndStatus(t *testing.T) {
	installation := newFakeInstallation()
	var observed []runner.ReplaySpec
	result := func(value runner.ReplayResult, err error) replayDependencies {
		return installation.replayDependencies(func(_ context.Context, spec runner.ReplaySpec) (runner.ReplayResult, error) {
			observed = append(observed, spec)
			return value, err
		})
	}
	succeeded := runner.ReplayResult{Artifact: artifact.Artifact{Path: "/artifact", Manifest: record.ExecutionRecord{Outcome: record.Outcome{Domain: "success"}}}, Match: true, ChoiceReplayStatus: runner.ChoiceReplayExact}
	for _, test := range []struct {
		name      string
		arguments []string
		result    runner.ReplayResult
		err       error
		want      commandResult
	}{
		{"verify only", []string{"--verify-only", "/artifact"}, succeeded, nil, commandResult{status: 0, stdout: "gomad: verified /artifact\n"}},
		{"reproduced success", []string{"/artifact"}, succeeded, nil, commandResult{status: 0, stdout: "gomad: reproduced=true diagnostic=false result=success choice-replay=exact\n"}},
		{"divergence", []string{"/artifact"}, runner.ReplayResult{Divergence: "stdout", ChoiceReplayStatus: runner.ChoiceReplayExact}, nil, commandResult{status: 1, stdout: "gomad: reproduced=false divergence=stdout choice-replay=exact\n"}},
		{"preflight", []string{"/artifact"}, runner.ReplayResult{}, &runner.ReplayPreflightError{Err: errors.New("toolchain changed")}, commandResult{status: 2, stderr: "incompatible replay artifact: toolchain changed\n"}},
		{"replay failure", []string{"/artifact"}, runner.ReplayResult{}, errors.New("supervisor failed"), commandResult{status: 3, stderr: "supervisor failed\n"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := runCommand(func(stdout, stderr *bytes.Buffer) int {
				return runReplayWith(test.arguments, stdout, stderr, result(test.result, test.err))
			}); got != test.want {
				t.Fatalf("replay:\n%s\nwant:\n%s", got, test.want)
			}
		})
	}
	observed = nil
	if got := runCommand(func(stdout, stderr *bytes.Buffer) int {
		return runReplayWith([]string{"--toolchain-root=/bundle", "--observed=/observed", "--verify-only", "/artifact"}, stdout, stderr, result(succeeded, nil))
	}); got.status != 0 {
		t.Fatalf("replay = %s", got)
	}
	want := runner.ReplaySpec{ArtifactPath: "/artifact", VerifyOnly: true, ToolchainRoot: "/toolchain", ObservedDir: "/observed"}
	if len(observed) != 1 || !reflect.DeepEqual(observed[0], want) || (*installation.requested)[len(*installation.requested)-1] != "/bundle" {
		t.Fatalf("replay request = %#v requested=%q", observed, *installation.requested)
	}
	if got, want := runCommand(func(stdout, stderr *bytes.Buffer) int {
		return runReplayWith([]string{"/artifact"}, stdout, stderr, newFakeInstallation().failing(errFakeInstallation).replayDependencies(nil))
	}), (commandResult{status: 3, stderr: errFakeInstallation.Error() + "\n"}); got != want {
		t.Fatalf("replay installation failure = %s, want %s", got, want)
	}
}

func TestCharacterizeExecuteShardRequestOutputAndStatus(t *testing.T) {
	var observed []runner.CampaignShardSpec
	dependencies := func(result runner.CampaignResult, err error) campaignShardDependencies {
		return newFakeInstallation().campaignShardDependencies(func(_ context.Context, spec runner.CampaignShardSpec) (runner.CampaignResult, error) {
			if spec.Progress == nil {
				t.Fatal("shard request has no progress reporter")
			}
			spec.Progress = nil
			observed = append(observed, spec)
			return result, err
		})
	}
	success := runner.CampaignResult{CampaignPath: "/artifacts/v1/shard", SelectionCount: 1, Attempted: 1, Succeeded: 1, StopReason: runner.StopSeedsExhausted}
	if got, want := runCommand(func(stdout, stderr *bytes.Buffer) int {
		return runCampaignShardWith([]string{"--json", "--artifacts=/artifacts", "--toolchain-root=/bundle", "--shard=1/2", "/plan"}, stdout, stderr, dependencies(success, nil))
	}), (commandResult{status: 0, stdout: `{"schema":"gomad.explore-event/v3","type":"result","classification":"success","campaign_path":"/artifacts/v1/shard","selected":1,"attempted":1,"succeeded":1,"stop_reason":"seeds_exhausted"}` + "\n"}); got != want {
		t.Fatalf("execute-shard = %s, want %s", got, want)
	}
	want := runner.CampaignShardSpec{PlanPath: "/plan", Shard: runner.CampaignShard{Index: 1, Count: 2}, Artifacts: "/artifacts", ToolchainRoot: "/toolchain", RunnerBuild: "sha256:runner", ProgressInterval: 5 * time.Second}
	if len(observed) != 1 || !reflect.DeepEqual(observed[0], want) {
		t.Fatalf("execute-shard request = %#v", observed)
	}
	failed := runner.CampaignResult{CampaignPath: "/artifacts/v1/shard", Attempted: 1, Failures: 1}
	if got := runCommand(func(stdout, stderr *bytes.Buffer) int {
		return runCampaignShardWith([]string{"--shard=0/1", "/plan"}, stdout, stderr, dependencies(failed, nil))
	}); got.status != 1 || !strings.HasPrefix(got.stdout, "gomad: classification=target_failure ") {
		t.Fatalf("failed shard = %s", got)
	}
	if got, want := runCommand(func(stdout, stderr *bytes.Buffer) int {
		return runCampaignShardWith([]string{"--shard=0/1", "/plan"}, stdout, stderr, dependencies(runner.CampaignResult{}, errors.New("plan changed")))
	}), (commandResult{status: 2, stderr: "gomad: invalid_input: plan changed\n"}); got != want {
		t.Fatalf("invalid shard plan = %s, want %s", got, want)
	}
	if got, want := runCommand(func(stdout, stderr *bytes.Buffer) int {
		return runCampaignShardWith([]string{"--shard=0/1", "/plan"}, stdout, stderr, newFakeInstallation().failing(errFakeInstallation).campaignShardDependencies(nil))
	}), (commandResult{status: 3, stderr: errFakeInstallation.Error() + "\n"}); got != want {
		t.Fatalf("shard installation failure = %s, want %s", got, want)
	}
}

// TestCharacterizeInstallationWiring pins how resume, minimize, qualify and
// analyze pass the resolved installation and child-mode commands to their
// operations, and how each reports an unresolvable installation.
func TestCharacterizeInstallationWiring(t *testing.T) {
	installation := newFakeInstallation()
	var resumed runner.ResumeSpec
	resume := installation.resumeDependencies(func(_ context.Context, spec runner.ResumeSpec) (runner.CampaignResult, error) {
		spec.Progress = nil
		resumed = spec
		return runner.CampaignResult{CampaignPath: "/campaign"}, nil
	})
	if got := runCommand(func(stdout, stderr *bytes.Buffer) int {
		return runResumeWith([]string{"--toolchain-root=/bundle", "--guide-regression=false", "/campaign"}, stdout, stderr, resume)
	}); got.status != 0 {
		t.Fatalf("resume = %s", got)
	}
	regression := false
	if want := (runner.ResumeSpec{CampaignPath: "/campaign", ToolchainRoot: "/toolchain", RunnerBuild: "sha256:runner", ProgressInterval: 5 * time.Second, GuideRegression: &regression}); !reflect.DeepEqual(resumed, want) {
		t.Fatalf("resume request = %#v, want %#v", resumed, want)
	}

	var minimized runner.MinimizeSpec
	minimize := installation.minimizeDependencies(func(_ context.Context, spec runner.MinimizeSpec) (runner.MinimizeResult, error) {
		minimized = spec
		return runner.MinimizeResult{Artifact: artifact.Artifact{Path: "/minimized"}, Attempts: 1, AttemptBudget: 64, StopReason: "minimal"}, nil
	})
	if got, want := runCommand(func(stdout, stderr *bytes.Buffer) int {
		return runMinimizeWith([]string{"--toolchain-root=/bundle", "--artifacts=/artifacts", "/artifact"}, stdout, stderr, minimize)
	}), (commandResult{status: 0, stdout: "gomad minimize: changed=false attempts=1/64 accepted=0 stop=minimal artifact=/minimized\n"}); got != want {
		t.Fatalf("minimize = %s, want %s", got, want)
	}
	if want := (runner.MinimizeSpec{ArtifactPath: "/artifact", OutputRoot: "/artifacts/minimized", AttemptBudget: 64, ToolchainRoot: "/toolchain"}); !reflect.DeepEqual(minimized, want) {
		t.Fatalf("minimize request = %#v, want %#v", minimized, want)
	}

	var qualified []runner.CampaignSpec
	var replayed []runner.ReplaySpec
	qualify := installation.qualifyDependencies(func(_ context.Context, spec runner.CampaignSpec) (runner.CampaignResult, error) {
		spec.Progress = nil
		qualified = append(qualified, spec)
		evidence := characterizationEvidence()
		return runner.CampaignResult{CampaignPath: fmt.Sprintf("/campaign-%d", len(qualified)), SelectionCount: 1, Attempted: 1, Failures: 1, Artifacts: []string{fmt.Sprintf("/failure-%d", len(qualified))}, ExecutionEvidence: &evidence}, nil
	}, func(_ context.Context, spec runner.ReplaySpec) (runner.ReplayResult, error) {
		replayed = append(replayed, spec)
		return runner.ReplayResult{Match: true, ChoiceReplayStatus: runner.ChoiceReplayExact}, nil
	}, func(string, qualification.QualificationReport) (string, error) { return "/report.json", nil })
	if got := runCommand(func(stdout, stderr *bytes.Buffer) int {
		return runQualifyWith([]string{"--toolchain-root=/bundle", "--env=A=1", "--build-tag=tag", "go-test", "./pkg", "--", "-test.run=X"}, stdout, stderr, qualify)
	}); got.status != 1 || !strings.Contains(got.stdout, "gomad: qualification qualified=false") {
		t.Fatalf("qualify = %s", got)
	}
	if len(qualified) != 2 || len(replayed) != 2 {
		t.Fatalf("qualify requests = %d replays = %d", len(qualified), len(replayed))
	}
	for _, spec := range qualified {
		if spec.RunnerBuild != "sha256:runner" ||
			spec.Target.ToolchainRoot != "/toolchain" || !reflect.DeepEqual(spec.Environment, []string{"A=1"}) || !reflect.DeepEqual(spec.Target.BuildTags, []string{"tag"}) || !reflect.DeepEqual(spec.Target.Args, []string{"-test.run=X"}) {
			t.Fatalf("qualify request = %#v", spec)
		}
	}
	for _, spec := range replayed {
		if spec.ToolchainRoot != "/toolchain" {
			t.Fatalf("qualify replay request = %#v", spec)
		}
	}
	if got := *installation.requested; !reflect.DeepEqual(got, []string{"/bundle", "/bundle", "/bundle"}) {
		t.Fatalf("requested toolchain roots = %q", got)
	}

	unavailable := newFakeInstallation().failing(errFakeInstallation)
	for _, test := range []struct {
		name    string
		command func(stdout, stderr *bytes.Buffer) int
		want    commandResult
	}{
		{"resume", func(stdout, stderr *bytes.Buffer) int {
			return runResumeWith([]string{"/campaign"}, stdout, stderr, unavailable.resumeDependencies(nil))
		}, commandResult{status: 3, stderr: "gomad: runner_failure: " + errFakeInstallation.Error() + "\n"}},
		{"minimize", func(stdout, stderr *bytes.Buffer) int {
			return runMinimizeWith([]string{"/artifact"}, stdout, stderr, unavailable.minimizeDependencies(nil))
		}, commandResult{status: 3, stderr: errFakeInstallation.Error() + "\n"}},
		{"qualify", func(stdout, stderr *bytes.Buffer) int {
			return runQualifyWith([]string{"--json", "go-run", "./cmd"}, stdout, stderr, unavailable.qualifyDependencies(nil, nil, nil))
		}, commandResult{status: 3, stdout: `{"schema":"gomad.qualify-event/v1","type":"error","classification":"runner_failure","message":"` + errFakeInstallation.Error() + `"}` + "\n"}},
	} {
		if got := runCommand(test.command); got != test.want {
			t.Fatalf("%s installation failure:\n%s\nwant:\n%s", test.name, got, test.want)
		}
	}
}

func characterizationEvidence() record.ExecutionEvidence {
	return record.ExecutionEvidence{
		Schema: record.ExecutionEvidenceSchema, Seed: record.Uint64String(1), RunnerBuild: "sha256:runner",
		Toolchain:   record.Toolchain{GoVersion: "go1.27.1", BuildKey: "build", TargetGOOS: "darwin", TargetGOARCH: "arm64"},
		Target:      record.Target{Kind: "go-test", Source: "./pkg", SHA256: "sha256:target", Size: 1, Argv: []string{"gomad-target"}},
		IOProfile:   record.IOContract{Name: "deterministic", ImplementationSHA256: "sha256:io", InventorySHA256: "sha256:inventory"},
		Environment: []record.Environment{{Name: "GOMADSEED", Value: "1"}, {Name: "TZ", Value: "UTC"}},
		Outcome:     record.OutcomeEvidence{Domain: "target_failure", Reason: "exit_status", Termination: "exit"}, GroupGone: true,
		Stdout: record.Stream{FullSHA256: "sha256:stdout"}, Stderr: record.Stream{FullSHA256: "sha256:stderr"},
		IOTranscriptSHA256: "sha256:transcript", IOTranscriptRecords: 1, IOTranscriptComplete: true,
		SemanticCoverage: record.SemanticCoverage{Schema: record.SemanticCoverageSchema, Digest: "sha256:coverage"},
	}
}

// TestCharacterizeOutputWriterFailures pins the status each command returns
// when the writer that carries its result or its error fails.
func TestCharacterizeOutputWriterFailures(t *testing.T) {
	t.Setenv("GOMAD_TOOLCHAIN_DIR", "")
	installation := newFakeInstallation()
	success := runner.CampaignResult{CampaignPath: "/campaign", Attempted: 1, Succeeded: 1}
	explore := installation.exploreDependencies(func(context.Context, runner.CampaignSpec) (runner.CampaignResult, error) { return success, nil }, func(context.Context, runner.CampaignPlanSpec) (runner.CampaignPlanResult, error) {
		return runner.CampaignPlanResult{Path: "/plan"}, nil
	})
	replay := installation.replayDependencies(func(context.Context, runner.ReplaySpec) (runner.ReplayResult, error) {
		return runner.ReplayResult{Artifact: artifact.Artifact{Path: "/artifact"}, Match: true}, nil
	})
	shard := installation.campaignShardDependencies(func(context.Context, runner.CampaignShardSpec) (runner.CampaignResult, error) { return success, nil })
	resume := installation.resumeDependencies(func(context.Context, runner.ResumeSpec) (runner.CampaignResult, error) { return success, nil })
	minimize := installation.minimizeDependencies(func(context.Context, runner.MinimizeSpec) (runner.MinimizeResult, error) {
		return runner.MinimizeResult{}, nil
	})
	for _, test := range []struct {
		name    string
		command func(stdout, stderr io.Writer) int
		status  int
	}{
		{"explore text result", func(stdout, stderr io.Writer) int {
			return runExploreWith([]string{"go-run", "./cmd"}, stdout, stderr, explore)
		}, 3},
		{"explore JSON result", func(stdout, stderr io.Writer) int {
			return runExploreWith([]string{"--json", "go-run", "./cmd"}, stdout, stderr, explore)
		}, 3},
		{"explore text input error", func(stdout, stderr io.Writer) int {
			return runExploreWith([]string{"--count=0", "go-run", "./cmd"}, stdout, stderr, explore)
		}, 3},
		{"explore JSON input error", func(stdout, stderr io.Writer) int {
			return runExploreWith([]string{"--json", "--count=0", "go-run", "./cmd"}, stdout, stderr, explore)
		}, 3},
		{"plan text result", func(stdout, stderr io.Writer) int {
			return runPlanWith([]string{"--output=/plan", "go-run", "./cmd"}, stdout, stderr, explore)
		}, 3},
		{"plan JSON result", func(stdout, stderr io.Writer) int {
			return runPlanWith([]string{"--json", "--output=/plan", "go-run", "./cmd"}, stdout, stderr, explore)
		}, 3},
		{"replay result", func(stdout, stderr io.Writer) int {
			return runReplayWith([]string{"/artifact"}, stdout, stderr, replay)
		}, 3},
		{"replay verification", func(stdout, stderr io.Writer) int {
			return runReplayWith([]string{"--verify-only", "/artifact"}, stdout, stderr, replay)
		}, 3},
		{"execute-shard result", func(stdout, stderr io.Writer) int {
			return runCampaignShardWith([]string{"--shard=0/1", "/plan"}, stdout, stderr, shard)
		}, 3},
		{"resume result", func(stdout, stderr io.Writer) int {
			return runResumeWith([]string{"/campaign"}, stdout, stderr, resume)
		}, 3},
		{"minimize text result", func(stdout, stderr io.Writer) int {
			return runMinimizeWith([]string{"/artifact"}, stdout, stderr, minimize)
		}, 3},
		{"minimize JSON result", func(stdout, stderr io.Writer) int {
			return runMinimizeWith([]string{"--json", "/artifact"}, stdout, stderr, minimize)
		}, 3},
		{"minimize usage", func(stdout, stderr io.Writer) int { return runMinimizeWith(nil, stdout, stderr, minimize) }, 3},
		{"recover usage", func(stdout, stderr io.Writer) int { return Run([]string{"recover"}, stdout, stderr) }, 3},
		{"inspect usage", func(stdout, stderr io.Writer) int { return Run([]string{"inspect"}, stdout, stderr) }, 3},
		{"doctor report", func(stdout, stderr io.Writer) int {
			return Run([]string{"doctor", "--json"}, stdout, stderr)
		}, 3},
		{"replay installation error", func(stdout, stderr io.Writer) int {
			return runReplayWith([]string{"/artifact"}, stdout, stderr, installation.failing(errFakeInstallation).replayDependencies(nil))
		}, 3},
	} {
		t.Run(test.name, func(t *testing.T) {
			if status := test.command(failingWriter{}, failingWriter{}); status != test.status {
				t.Fatalf("status = %d, want %d", status, test.status)
			}
		})
	}
}

// fakeInstallation is the resolved installation a characterization test hands
// to a command through its dependency seam. It records every explicit
// --toolchain-root the command asked to resolve, so tests can pin both what a
// command requests and how it wires the resolved result into its operation.
// Only this file knows the seam's shape; the characterization tests in
// characterization_test.go build their dependencies through it.
type fakeInstallation struct {
	toolchainRoot string
	executable    string
	runnerBuild   string
	err           error
	requested     *[]string
}

func newFakeInstallation() fakeInstallation {
	return fakeInstallation{toolchainRoot: "/toolchain", executable: "/bin/gomad", runnerBuild: "sha256:runner", requested: new([]string)}
}

func (fake fakeInstallation) failing(err error) fakeInstallation {
	fake.err = err
	return fake
}

func (fake fakeInstallation) install(explicitToolchainRoot string) (installation, error) {
	*fake.requested = append(*fake.requested, explicitToolchainRoot)
	if fake.err != nil {
		return installation{}, fake.err
	}
	return installation{toolchainRoot: fake.toolchainRoot, executable: fake.executable, runnerBuild: fake.runnerBuild}, nil
}

func (fake fakeInstallation) exploreDependencies(explore func(context.Context, runner.CampaignSpec) (runner.CampaignResult, error), plan func(context.Context, runner.CampaignPlanSpec) (runner.CampaignPlanResult, error)) exploreDependencies {
	return exploreDependencies{install: fake.install, workingDirectory: func() (string, error) { return "/workspace", nil }, explore: explore, plan: plan}
}

func (fake fakeInstallation) replayDependencies(replay func(context.Context, runner.ReplaySpec) (runner.ReplayResult, error)) replayDependencies {
	return replayDependencies{install: fake.install, replay: replay}
}

func (fake fakeInstallation) campaignShardDependencies(run func(context.Context, runner.CampaignShardSpec) (runner.CampaignResult, error)) campaignShardDependencies {
	return campaignShardDependencies{install: fake.install, run: run}
}

func (fake fakeInstallation) resumeDependencies(run func(context.Context, runner.ResumeSpec) (runner.CampaignResult, error)) resumeDependencies {
	return resumeDependencies{install: fake.install, run: run}
}

func (fake fakeInstallation) minimizeDependencies(minimize func(context.Context, runner.MinimizeSpec) (runner.MinimizeResult, error)) minimizeDependencies {
	return minimizeDependencies{install: fake.install, minimize: minimize}
}

func (fake fakeInstallation) qualifyDependencies(run func(context.Context, runner.CampaignSpec) (runner.CampaignResult, error), replay func(context.Context, runner.ReplaySpec) (runner.ReplayResult, error), write func(string, qualification.QualificationReport) (string, error)) qualifyDependencies {
	return qualifyDependencies{install: fake.install, workingDirectory: func() (string, error) { return "/workspace", nil }, run: run, replay: replay, write: write}
}

// analyzeToolchain is the toolchain-root seam gomad analyze resolves through.
func (fake fakeInstallation) analyzeToolchain(explicitToolchainRoot string) (string, error) {
	resolved, err := fake.install(explicitToolchainRoot)
	return resolved.toolchainRoot, err
}

var errFakeInstallation = errors.New("resolve Gomad installation: fake installation is unavailable")

func TestQualifyGrammarPreservesEnvironmentTagsAndTargetArguments(t *testing.T) {
	for _, jsonOutput := range []bool{false, true} {
		var observed qualificationworkload.WorkloadSpec
		dependencies := qualifyDependencies{
			install: func(string) (installation, error) {
				return installation{toolchainRoot: "/toolchain", executable: "/bin/gomad", runnerBuild: "sha256:runner"}, nil
			},
			workingDirectory: func() (string, error) { return "/workspace", nil },
			workload: func(_ context.Context, spec qualificationworkload.WorkloadSpec) (qualificationworkload.WorkloadResult, error) {
				observed = spec
				return qualificationworkload.WorkloadResult{
					Report:     qualification.QualificationReport{Qualified: true, Deterministic: true, TargetSuccess: true},
					ReportPath: "/report.json",
				}, nil
			},
		}
		arguments := []string{"--env=FOO=bar", "--build-tag=gomad_fixture", "go-test", "./pkg", "--", "-test.run=Test Name", "literal;$value"}
		if jsonOutput {
			arguments = append([]string{"--json"}, arguments...)
		}
		var stdout, stderr bytes.Buffer
		status := runQualifyWith(arguments, &stdout, &stderr, dependencies)
		if status != 0 || stderr.Len() != 0 {
			t.Fatalf("json=%t status=%d stdout=%q stderr=%q", jsonOutput, status, stdout.String(), stderr.String())
		}
		got := observed.Campaign
		if len(got.Environment) != 1 || got.Environment[0] != "FOO=bar" || len(got.Target.BuildTags) != 1 || got.Target.BuildTags[0] != "gomad_fixture" || len(got.Target.Args) != 2 || got.Target.Args[0] != "-test.run=Test Name" || got.Target.Args[1] != "literal;$value" {
			t.Fatalf("json=%t campaign=%#v", jsonOutput, got)
		}
		want := "gomad: qualification qualified=true"
		if jsonOutput {
			want = `"classification":"qualified"`
		}
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("json=%t output=%q, want %q", jsonOutput, stdout.String(), want)
		}
	}
}

// Characterization of the grammar gomad plan shares with gomad explore: the
// plan request it builds, the explore validation it inherits in the same
// first-error order, the explore-only and plan-only flags, and the routing of
// its errors and results. Dependencies are built only through the fake
// installation in this file.

// TestCharacterizePlanRequestDefaults pins the whole campaign a default plan
// freezes: explore's defaults with the failure policy fixed to all.
func TestCharacterizePlanRequestDefaults(t *testing.T) {
	installation := newFakeInstallation()
	var observed []runner.CampaignPlanSpec
	plan := func(_ context.Context, spec runner.CampaignPlanSpec) (runner.CampaignPlanResult, error) {
		if spec.Campaign.Progress == nil {
			t.Fatal("plan request has no progress reporter")
		}
		spec.Campaign.Progress = nil
		observed = append(observed, spec)
		return runner.CampaignPlanResult{Path: "/plan"}, nil
	}
	dependencies := installation.exploreDependencies(nil, plan)
	if got := runCommand(func(stdout, stderr *bytes.Buffer) int {
		return runPlanWith([]string{"--output=relative/plan.json", "go-run", "./cmd"}, stdout, stderr, dependencies)
	}); got.status != 0 || got.stderr != "" {
		t.Fatalf("default plan = %s", got)
	}
	want := runner.CampaignPlanSpec{
		Output: "relative/plan.json",
		Campaign: runner.CampaignSpec{
			Strategy: runner.StrategySeed, Seeds: "1", Parallel: min(runtime.NumCPU(), 8), ExecutionTimeout: 30 * time.Second, OverallTimeout: 10 * time.Minute, TerminateGrace: 2 * time.Second,
			OnFailure: runner.PolicyAll, FailureBudget: 1, OutputLimit: 8 << 20, WorldTransitionLimit: 64 << 20, ClockTick: record.ClockTickStrict, IOTranscriptLimit: 64 << 20,
			Artifacts: ".gomad/artifacts", RunnerBuild: "sha256:runner",
			Coverage: campaignoptions.CoverageNone, KeepSuccesses: runner.KeepSuccessesNone, ProgressInterval: 5 * time.Second,
			Target: target.Spec{Kind: target.KindGoRun, Source: "./cmd", WorkingDir: "/workspace", ToolchainRoot: "/toolchain", CapabilityMode: target.CapabilityModeClosure},
		},
	}
	if len(observed) != 1 || !reflect.DeepEqual(observed[0], want) {
		t.Fatalf("default plan request =\n%#v\nwant\n%#v", observed, want)
	}
	if !reflect.DeepEqual(*installation.requested, []string{""}) {
		t.Fatalf("requested toolchain roots = %q", *installation.requested)
	}

	// The CLI forwards every explore option to the plan operation and leaves
	// the portable-plan restrictions (seed strategy, on-failure=all) to it.
	observed = nil
	arguments := []string{
		"--strategy=choice-exploration", "--seeds=5", "--max-executions=2", "--max-choice-depth=3", "--max-exploration-bytes=1MiB", "--choice-start-ordinal=4",
		"--guide-regression=false", "--output=/plans/choice.json", "--json", "exec", "--provenance", "/target.provenance.json", "--", "/target", "--flag",
	}
	if got := runCommand(func(stdout, stderr *bytes.Buffer) int {
		return runPlanWith(arguments, stdout, stderr, dependencies)
	}); got.status != 0 || got.stderr != "" {
		t.Fatalf("choice-exploration plan = %s", got)
	}
	if len(observed) != 1 {
		t.Fatalf("choice-exploration plan requests = %d", len(observed))
	}
	campaign := observed[0].Campaign
	if observed[0].Output != "/plans/choice.json" || campaign.Strategy != runner.StrategyChoiceExploration || campaign.OnFailure != runner.PolicyAll || campaign.Seeds != "5" ||
		campaign.MaxExecutions != 2 || campaign.MaxChoiceDepth != 3 || campaign.MaxExplorationBytes != 1<<20 || campaign.ChoiceStartOrdinal != 4 || campaign.ChoiceTraceLimit != 8<<20 ||
		!reflect.DeepEqual(campaign.Target, target.Spec{Kind: target.KindExec, Source: "/target", Provenance: "/target.provenance.json", Args: []string{"--flag"}, WorkingDir: "/workspace", ToolchainRoot: "/toolchain", CapabilityMode: target.CapabilityModeClosure}) {
		t.Fatalf("choice-exploration plan request = %#v", observed[0])
	}
}

// TestCharacterizePlanInputRejection pins that plan reports explore's input
// errors with the same messages and in the same first-error order, before it
// resolves the installation and before it requires --output.

// TestCharacterizePlanFlagErrors pins flag-parse failures of plan in text and
// JSON, and that an explicit --on-failure follows plan's fixed policy only
// when it parses.

// TestCharacterizePlanOperationErrors pins how plan classifies the errors of
// its operation.
func TestCharacterizePlanOperationErrors(t *testing.T) {
	for _, test := range []struct {
		name string
		err  error
		json bool
		want commandResult
	}{
		{"portable restriction", errors.New("portable campaign plans require a seed campaign with on-failure=all"), false, commandResult{status: 2, stderr: "gomad: invalid_input: portable campaign plans require a seed campaign with on-failure=all\n"}},
		{"runner failure", &runner.HostError{Reason: "plan", Err: errors.New("write")}, false, commandResult{status: 3, stderr: "gomad: runner_failure: " + (&runner.HostError{Reason: "plan", Err: errors.New("write")}).Error() + "\n"}},
		{"cancelled JSON", &runner.HostError{Reason: "cancelled"}, true, commandResult{status: 3, stdout: `{"schema":"gomad.explore-event/v3","type":"error","classification":"cancelled","message":"gomad Runner/host failure: cancelled"}` + "\n"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			dependencies := newFakeInstallation().exploreDependencies(nil, func(context.Context, runner.CampaignPlanSpec) (runner.CampaignPlanResult, error) {
				return runner.CampaignPlanResult{}, test.err
			})
			arguments := []string{"--output=/plan", "go-run", "./cmd"}
			if test.json {
				arguments = append([]string{"--json"}, arguments...)
			}
			if got := runCommand(func(stdout, stderr *bytes.Buffer) int { return runPlanWith(arguments, stdout, stderr, dependencies) }); got != test.want {
				t.Fatalf("plan:\n%s\nwant:\n%s", got, test.want)
			}
		})
	}
	// Plan reports a missing --output after installation and working-directory
	// resolution, as invalid input in JSON too.
	if got, want := runCommand(func(stdout, stderr *bytes.Buffer) int {
		return runPlanWith([]string{"--json", "go-run", "./cmd"}, stdout, stderr, newFakeInstallation().exploreDependencies(nil, nil))
	}), (commandResult{status: 2, stdout: `{"schema":"gomad.explore-event/v3","type":"error","classification":"invalid_input","message":"gomad plan requires --output FILE"}` + "\n"}); got != want {
		t.Fatalf("plan JSON without output = %s, want %s", got, want)
	}
	if got, want := runCommand(func(stdout, stderr *bytes.Buffer) int {
		return runPlanWith([]string{"--working-dir=module", "go-run", "./cmd"}, stdout, stderr, newFakeInstallation().exploreDependencies(nil, nil))
	}), (commandResult{status: 2, stderr: "gomad: invalid_input: working directory \"module\" must be an absolute, clean path\n"}); got != want {
		t.Fatalf("plan relative working directory without output = %s, want %s", got, want)
	}
}

// TestCharacterizeExploreErrorWriteRouting pins where explore and plan report
// a failure to write a JSON error event: most input errors also print the
// write failure on stderr, while the diagnostics-strategy rejection does not.
func TestCharacterizeExploreErrorWriteRouting(t *testing.T) {
	dependencies := newFakeInstallation().exploreDependencies(nil, nil)
	for _, test := range []struct {
		name      string
		command   func(stdout, stderr io.Writer) int
		wantError bool
	}{
		{"explore count", func(stdout, stderr io.Writer) int {
			return runExploreWith([]string{"--json", "--count=0", "go-run", "./cmd"}, stdout, stderr, dependencies)
		}, true},
		{"plan count", func(stdout, stderr io.Writer) int {
			return runPlanWith([]string{"--json", "--count=0", "go-run", "./cmd"}, stdout, stderr, dependencies)
		}, true},
		{"explore diagnostics", func(stdout, stderr io.Writer) int {
			return runExploreWith([]string{"--json", "--strategy=choice-exploration", "--max-executions=1", "--max-choice-depth=1", "--max-exploration-bytes=1MiB", "--diagnostics", "go-run", "./cmd"}, stdout, stderr, dependencies)
		}, false},
		{"plan missing output", func(stdout, stderr io.Writer) int {
			return runPlanWith([]string{"--json", "go-run", "./cmd"}, stdout, stderr, dependencies)
		}, true},
		{"explore output", func(stdout, stderr io.Writer) int {
			return runExploreWith([]string{"--json", "--output=/plan", "go-run", "./cmd"}, stdout, stderr, dependencies)
		}, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			var stderr bytes.Buffer
			if status := test.command(failingWriter{}, &stderr); status != 3 || (stderr.Len() != 0) != test.wantError {
				t.Fatalf("status = %d, stderr = %q, want status 3 and stderr written = %t", status, stderr.String(), test.wantError)
			}
		})
	}
}

// TestCharacterizeHiddenPlanFlag pins that the hidden --__plan route is gone
// (fn-109 R6): explore and plan reject --__plan as an unknown flag, before
// resolving the installation, and never reach the plan operation.

// preMoveWASMArtifact is a stock-profile success retained before the backend
// move and before artifacts declared a storage lineage. The native fn-113
// pre-move artifact is no longer read: historical artifacts are not reopened
// by the new build.
const preMoveWASMArtifact = "../../../../internal/backends/wasm/backend/testdata/premove-stock-artifact"

// stageRetainedArtifact copies a checked-in artifact with the private modes
// that OpenArtifact requires and version control does not keep.
func stageRetainedArtifact(t *testing.T, source string) string {
	t.Helper()
	destination := filepath.Join(t.TempDir(), "artifact")
	err := filepath.WalkDir(source, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		target := filepath.Join(destination, relative)
		if entry.IsDir() {
			return os.Mkdir(target, 0o700)
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		mode := os.FileMode(0o600)
		if info.Mode()&0o100 != 0 {
			mode = 0o700
		}
		return os.WriteFile(target, data, mode)
	})
	if err != nil {
		t.Fatal(err)
	}
	return destination
}

const preLineageRefusal = `unsupported artifact storage format "pre-lineage": the artifact has no storage.json; this Gomad reads only "gomad.artifact-storage/v1" artifacts and does not migrate others`

// The pre-move artifact is a negative control: inspect refuses it with the
// named unsupported storage format as invalid input (status 2), in human and
// JSON form, without printing a report.
func TestInspectPreMoveArtifactIsRefusedAsUnsupportedStorageFormat(t *testing.T) {
	path := stageRetainedArtifact(t, preMoveWASMArtifact)
	for _, arguments := range [][]string{{"inspect", path}, {"inspect", "--json", path}} {
		t.Run(strings.Join(arguments[:len(arguments)-1], " "), func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			status := Run(arguments, &stdout, &stderr)
			want := "inspect " + path + ": " + preLineageRefusal + "\n"
			if status != 2 || stdout.Len() != 0 || stderr.String() != want {
				t.Fatalf("status=%d stdout=%q stderr=%q; want status 2 and %q", status, stdout.String(), stderr.String(), want)
			}
		})
	}
}

// Replay preflight refuses the pre-move artifact with the same named format
// as an incompatible replay artifact (status 2) before any toolchain is
// resolved.
func TestReplayPreMoveArtifactIsRefusedAsUnsupportedStorageFormat(t *testing.T) {
	var stdout, stderr bytes.Buffer
	status := Run([]string{"replay", "--verify-only", stageRetainedArtifact(t, preMoveWASMArtifact)}, &stdout, &stderr)
	want := "incompatible replay artifact: " + preLineageRefusal + "\n"
	if status != 2 || stdout.Len() != 0 || stderr.String() != want {
		t.Fatalf("status=%d stdout=%q stderr=%q; want status 2 and %q", status, stdout.String(), stderr.String(), want)
	}
}

func TestPublicCommandGrammarRejectsMissingOperandsBeforeInstallation(t *testing.T) {
	t.Setenv("GOMAD_TOOLCHAIN_DIR", "relative-invalid-root")
	for _, command := range []string{
		"plan", "execute-shard", "merge", "explore", "qualify", "qualify-set",
		"merge-set", "compare-support", "analyze", "resume", "recover", "replay",
		"minimize", "doctor", "inspect",
	} {
		t.Run(command, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			arguments := []string{command}
			if command == "doctor" {
				arguments = append(arguments, "unexpected")
			}
			if status := Run(arguments, &stdout, &stderr); status != 2 {
				t.Fatalf("Run(%q) status=%d stdout=%q stderr=%q", arguments, status, stdout.String(), stderr.String())
			}
			if strings.Contains(stdout.String()+stderr.String(), "relative-invalid-root") {
				t.Fatalf("Run(%q) resolved installation before rejecting syntax: stdout=%q stderr=%q", arguments, stdout.String(), stderr.String())
			}
		})
	}
}

func TestPublicCommandWriterFailuresPreserveStatus(t *testing.T) {
	arguments := []string{"inspect", "/missing/artifact"}
	var stdout bytes.Buffer
	if status := Run(arguments, &stdout, failingWriter{}); status != 3 {
		t.Fatalf("Run(%q) status=%d stdout=%q", arguments, status, stdout.String())
	}
}
