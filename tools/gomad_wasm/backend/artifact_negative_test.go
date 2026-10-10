package backend

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"go.temporal.io/server/tools/gomad3/artifact"
	"go.temporal.io/server/tools/gomad3/record"
	"go.temporal.io/server/tools/gomad3/runner"
	"go.temporal.io/server/tools/gomad3/target"
)

func TestReplayRejectsChangedArtifactIdentityBeforeGuest(t *testing.T) {
	provider := integrationProvider(t)
	fixture, err := filepath.Abs("../../gomad3/internal/gomadtool/conformance/testdata")
	if err != nil {
		t.Fatal(err)
	}
	campaign, err := runner.Explore(t.Context(), runner.CampaignSpec{Backend: provider, Seeds: "7", Parallel: 1, ExecutionTimeout: time.Minute, OverallTimeout: 3 * time.Minute, TerminateGrace: 100 * time.Millisecond, OnFailure: runner.PolicyFirst, FailureBudget: 1, OutputLimit: 1 << 20, WorldTransitionLimit: 1 << 20, Artifacts: t.TempDir(), RunnerBuild: "task4-integration", Coverage: runner.CoverageNone, Target: target.Spec{Backend: Name, Kind: target.KindGoTest, Source: "./io_failure", WorkingDir: fixture, BuildTags: []string{"gomad_fixture", "test_dep"}, Args: []string{"-test.run=^TestDeterministicIOFailure$", "-test.v"}}})
	if err != nil || len(campaign.Artifacts) != 1 {
		t.Fatalf("discovery: %#v %v", campaign, err)
	}
	opened, err := artifact.OpenArtifact(campaign.Artifacts[0])
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := opened.Close(); err != nil {
			t.Error(err)
		}
	}()
	original := opened.Manifest()
	input := artifact.ArtifactInput{Manifest: original, TargetPath: filepath.Join(campaign.Artifacts[0], "target")}
	input.Stdout, err = opened.ReadPayload("stdout", 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	input.Stderr, err = opened.ReadPayload("stderr", 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	for _, reference := range record.BackendReferences(*original.Target.Backend) {
		data, err := opened.ReadPayload(reference.File, uint64(reference.Bytes))
		if err != nil {
			t.Fatal(err)
		}
		input.BackendPayloads = append(input.BackendPayloads, target.BackendPayload{Reference: reference, Data: data})
	}
	input.World.Initial, err = opened.ReadPayload(original.World.Initial.File, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	input.World.Transitions, err = opened.ReadPayload(original.World.Transitions.File, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	input.World.Final, err = opened.ReadPayload(original.World.Final.File, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	pr, err := decodeProvenance(input.BackendPayloads[0].Data)
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name   string
		change func(*provenance)
	}{
		{"compiler", func(p *provenance) { p.CompilerSHA256 = string(record.HashBytes([]byte("changed"))) }},
		{"helper", func(p *provenance) { p.HelperSHA256 = string(record.HashBytes([]byte("changed"))) }},
		{"engine", func(p *provenance) { p.Engine.Configuration = "changed" }},
		{"model", func(p *provenance) { p.ModelSHA256 = string(record.HashBytes([]byte("changed"))) }},
		{"profile", func(p *provenance) { p.Profile = "changed" }},
		{"imports", func(p *provenance) { p.Imports = nil }},
		{"memory", func(p *provenance) { p.InitialMemoryPages++ }},
		{"limits", func(p *provenance) { p.Fuel++ }},
		{"inputs", func(p *provenance) { p.Config.Stdin = []byte("changed") }},
		{"argv", func(p *provenance) { p.Args = []string{"changed"} }},
	} {
		t.Run(test.name, func(t *testing.T) {
			changed := pr
			test.change(&changed)
			data, err := json.Marshal(changed)
			if err != nil {
				t.Fatal(err)
			}
			modified := input
			modified.Manifest = opened.Manifest()
			modified.Manifest.Target.Backend.Provenance.SHA256 = record.HashBytes(data)
			modified.Manifest.Target.Backend.Provenance.Bytes = record.Uint64String(len(data))
			modified.Manifest.IOProfile = record.BackendIOProfile(*modified.Manifest.Target.Backend)
			modified.BackendPayloads = append([]target.BackendPayload(nil), input.BackendPayloads...)
			modified.BackendPayloads[0] = target.BackendPayload{Reference: modified.Manifest.Target.Backend.Provenance, Data: data}
			published, err := artifact.PublishArtifact(artifact.Store{Root: t.TempDir()}, modified)
			if err != nil {
				t.Fatal(err)
			}
			counted := &countedProvider{Provider: provider}
			if _, err := runner.Replay(t.Context(), runner.ReplaySpec{Backend: counted, ArtifactPath: published.Path}); err == nil {
				t.Fatal("changed identity was accepted")
			}
			if counted.calls != 0 {
				t.Fatal("identity rejection occurred after guest execution")
			}
		})
	}
	if _, err := runner.Replay(t.Context(), runner.ReplaySpec{ArtifactPath: campaign.Artifacts[0]}); err == nil {
		t.Fatal("provider absence selected native fallback")
	}
	for _, file := range []string{"target", original.Target.Backend.Provenance.File, original.Target.Backend.Evidence.File, "stdout"} {
		t.Run("corrupt-"+file, func(t *testing.T) {
			copyRoot := t.TempDir()
			for _, entry := range original.Files {
				source := filepath.Join(campaign.Artifacts[0], filepath.FromSlash(entry.Path))
				data, err := os.ReadFile(source)
				if err != nil {
					t.Fatal(err)
				}
				mode := os.FileMode(0600)
				if entry.Path == "target" {
					mode = 0700
				}
				destination := filepath.Join(copyRoot, filepath.FromSlash(entry.Path))
				if err := os.MkdirAll(filepath.Dir(destination), 0700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(destination, data, mode); err != nil {
					t.Fatal(err)
				}
			}
			data, err := os.ReadFile(filepath.Join(campaign.Artifacts[0], "manifest.json"))
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(copyRoot, "manifest.json"), data, 0600); err != nil {
				t.Fatal(err)
			}
			corrupted := filepath.Join(copyRoot, filepath.FromSlash(file))
			data, err = os.ReadFile(corrupted)
			if err != nil {
				t.Fatal(err)
			}
			data[0] ^= 1
			if err := os.WriteFile(corrupted, data, 0600); err != nil {
				t.Fatal(err)
			}
			counted := &countedProvider{Provider: provider}
			if _, err := runner.Replay(t.Context(), runner.ReplaySpec{Backend: counted, ArtifactPath: copyRoot}); err == nil {
				t.Fatal("corrupt payload accepted")
			}
			if counted.calls != 0 {
				t.Fatal("corruption reached guest")
			}
		})
	}
}
