package backend

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"testing"

	"go.temporal.io/server/tools/gomad3/artifact"
	"go.temporal.io/server/tools/gomad3/backends/wasm/wasi"
	"go.temporal.io/server/tools/gomad3/choice"
	"go.temporal.io/server/tools/gomad3/runner"
	runnerbackend "go.temporal.io/server/tools/gomad3/runner/backend"
)

const minimizationGuest = `package main
import ("fmt"; "os")
func choose() int {
	a, b := make(chan int, 1), make(chan int, 1)
	a <- 0; b <- 1
	select { case n := <-a: return n; case n := <-b: return n }
}
func main() {
	failure := choose() == 1
	choose(); choose()
	if failure { fmt.Println("essential early choice failed"); os.Exit(17) }
	fmt.Println("success")
}
`

var errMinimizeInterrupted = errors.New("minimization integration interruption")

type minimizationProvider struct {
	*Provider
	controls []int
}

func (p *minimizationProvider) Run(ctx context.Context, request runnerbackend.Request) (runnerbackend.Result, error) {
	if request.Choice == nil {
		return runnerbackend.Result{}, errors.New("minimizer omitted choice request")
	}
	if request.Choice.Mode != choice.ModeReplay {
		length := 0
		if request.Choice.Tape != nil {
			length = len(request.Choice.Tape.Decisions)
		}
		p.controls = append(p.controls, length)
	}
	return p.Provider.Run(ctx, request)
}

type minimizationPublicationReplayer struct {
	provider  *Provider
	interrupt bool
}

func (r *minimizationPublicationReplayer) Replay(ctx context.Context, spec runner.ReplaySpec) (runner.ReplayResult, error) {
	if spec.Backend != r.provider {
		return runner.ReplayResult{}, errors.New("injected replay lost backend provider")
	}
	opened, err := artifact.OpenArtifact(spec.ArtifactPath)
	if err != nil {
		return runner.ReplayResult{}, err
	}
	final := opened.Manifest().Minimization != nil
	if err := opened.Close(); err != nil {
		return runner.ReplayResult{}, err
	}
	if final && r.interrupt {
		return runner.ReplayResult{}, errMinimizeInterrupted
	}
	return runner.Replay(ctx, spec)
}

func TestWASMMinimizeEssentialPrefixDetachedReplayRepeatAndResume(t *testing.T) {
	provider := explorationProvider(t)
	provider.options.Config.Network = &wasi.NetworkModel{Seed: 11, MaxDelayNanos: 1000, Faults: []wasi.NetworkFault{{AtNanos: 5_000_000_000, Action: "reset", Port: 9000}}}
	spec := explorationSpec(t, provider, minimizationGuest)
	spec.Parallel = 1
	seeded := spec
	seeded.Strategy, seeded.Seeds, seeded.Artifacts = runner.StrategySeed, "0-7", t.TempDir()
	seeded.MaxExecutions, seeded.MaxChoiceDepth, seeded.MaxExplorationBytes = 0, 0, 0
	seededCampaign, err := runner.Explore(t.Context(), seeded)
	if err != nil || len(seededCampaign.SuccessArtifacts) == 0 {
		t.Fatalf("canary needs a seed where unforced execution succeeds: %+v err=%v", seededCampaign, err)
	}
	success, err := artifact.OpenArtifact(seededCampaign.SuccessArtifacts[0])
	if err != nil {
		t.Fatal(err)
	}
	spec.Seeds = strconv.FormatUint(uint64(success.Manifest().Seed), 10)
	if err := success.Close(); err != nil {
		t.Fatal(err)
	}
	campaign, err := runner.Explore(t.Context(), spec)
	if err != nil || campaign.Succeeded == 0 || len(campaign.Artifacts) == 0 {
		t.Fatalf("essential canary discovery: %+v err=%v", campaign, err)
	}
	parent, err := artifact.OpenArtifact(campaign.Artifacts[0])
	if err != nil {
		t.Fatal(err)
	}
	manifest := parent.Manifest()
	if manifest.ChoiceProfile == nil || manifest.ChoiceProfile.Trace.Decisions < 3 {
		t.Fatalf("canary needs redundant trailing choices: %+v", manifest.ChoiceProfile)
	}
	if err := parent.Close(); err != nil {
		t.Fatal(err)
	}
	for _, directory := range []string{spec.Target.WorkingDir, provider.options.CacheRoot} {
		if err := os.Rename(directory, directory+"-unavailable"); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			if err := os.Rename(directory+"-unavailable", directory); err != nil {
				t.Error(err)
			}
		})
	}
	provider.options.CompilerPath = filepath.Join(t.TempDir(), "unavailable-go")
	provider.options.RuntimeRoot = filepath.Join(t.TempDir(), "unavailable-runtime")
	observed := &minimizationProvider{Provider: provider}
	minimize := runner.MinimizeSpec{Backend: observed, ArtifactPath: campaign.Artifacts[0], OutputRoot: t.TempDir(), AttemptBudget: 64}
	baseline, err := runner.Minimize(t.Context(), minimize)
	if err != nil {
		t.Fatal(err)
	}
	evidence := baseline.Artifact.Manifest.Minimization
	if !baseline.Changed || len(baseline.Accepted) == 0 || evidence == nil || evidence.Schema != "gomad3.minimization/choice-prefix/v1" || evidence.FinalForcedDecisions == 0 || evidence.FinalForcedDecisions >= evidence.OriginalForcedDecisions || evidence.Predicate.SimulationReplay != "not_present" || baseline.Artifact.Manifest.Outcome.FailureSignature != manifest.Outcome.FailureSignature {
		t.Fatalf("effective forcing not reduced with essential prefix preserved: %+v evidence=%+v", baseline, evidence)
	}
	if baseline.Artifact.Manifest.Target.Backend.Provenance.SHA256 != manifest.Target.Backend.Provenance.SHA256 || baseline.Artifact.Manifest.Target.Backend.Evidence.SHA256 != manifest.Target.Backend.Evidence.SHA256 {
		t.Fatal("minimization changed fixed model/fault inputs or complete backend observations")
	}
	if len(observed.controls) == 0 || observed.controls[0] != 0 {
		t.Fatalf("missing rejected no-forcing attempt: %v", observed.controls)
	}
	for _, reduction := range baseline.Accepted {
		if reduction.Kind != "schedule_suffix" {
			t.Fatalf("invalid prefix reduction: %+v", reduction)
		}
	}
	replay, err := runner.Replay(t.Context(), runner.ReplaySpec{Backend: provider, ArtifactPath: baseline.Artifact.Path})
	if err != nil || !replay.Match || replay.ChoiceReplayStatus != runner.ChoiceReplayExact {
		t.Fatalf("source-less fresh replay: %+v err=%v", replay, err)
	}
	for _, interruption := range []string{"repeat", "after accepted", "after publication"} {
		t.Run(interruption, func(t *testing.T) {
			tracked := &minimizationProvider{Provider: provider}
			request := minimize
			request.Backend, request.OutputRoot = tracked, t.TempDir()
			blockedPublication := filepath.Join(request.OutputRoot, filepath.Base(baseline.Artifact.Path))
			if interruption == "after accepted" {
				if err := os.WriteFile(blockedPublication, []byte("publication interruption"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			if interruption == "after publication" {
				request.Backend = provider
				request.Replayer = &minimizationPublicationReplayer{provider: provider, interrupt: true}
			}
			result, err := runner.Minimize(t.Context(), request)
			if interruption != "repeat" {
				if interruption == "after publication" && !errors.Is(err, errMinimizeInterrupted) || interruption == "after accepted" && err == nil {
					t.Fatalf("missing interruption: %v", err)
				}
				if interruption == "after accepted" {
					if err := os.Rename(blockedPublication, blockedPublication+"-unavailable"); err != nil {
						t.Fatal(err)
					}
				}
				request.Resume = true
				if request.Replayer != nil {
					request.Replayer = &minimizationPublicationReplayer{provider: provider}
				}
				result, err = runner.Minimize(t.Context(), request)
			}
			if err != nil || result.Attempts != baseline.Attempts || result.Artifact.Manifest.RecordHash != baseline.Artifact.Manifest.RecordHash || !reflect.DeepEqual(result.Accepted, baseline.Accepted) {
				t.Fatalf("repeat/resume differs: %+v err=%v", result, err)
			}
			if interruption != "after publication" && !reflect.DeepEqual(tracked.controls, observed.controls) {
				t.Fatalf("committed attempts repeated: got=%v want=%v", tracked.controls, observed.controls)
			}
		})
	}
	parent, err = artifact.OpenArtifact(campaign.Artifacts[0])
	if err != nil {
		t.Fatal(err)
	}
	if parent.Manifest().RecordHash != manifest.RecordHash {
		t.Fatal("minimization mutated parent")
	}
	if err := parent.Close(); err != nil {
		t.Fatal(err)
	}
	t.Logf("parent=%s final=%s attempts=%d original-forcing=%d final-forcing=%d controls=%v backend-evidence=%s", manifest.RecordHash, baseline.Artifact.Manifest.RecordHash, baseline.Attempts, evidence.OriginalForcedDecisions, evidence.FinalForcedDecisions, observed.controls, baseline.Artifact.Manifest.Target.Backend.Evidence.SHA256)
}
