package backend

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"go.temporal.io/server/tools/gomad3/artifact"
	"go.temporal.io/server/tools/gomad3/deterministicio/readonlymount"
	"go.temporal.io/server/tools/gomad3/record"
	"go.temporal.io/server/tools/gomad3/runner"
	runnerbackend "go.temporal.io/server/tools/gomad3/runner/backend"
	"go.temporal.io/server/tools/gomad3/target"
)

type countedProvider struct {
	*Provider
	calls        int
	preparedRoot string
}

func (p *countedProvider) Run(ctx context.Context, request runnerbackend.Request) (runnerbackend.Result, error) {
	p.calls++
	return p.Provider.Run(ctx, request)
}
func (p *countedProvider) Prepare(ctx context.Context, spec target.Spec) (target.Prepared, error) {
	prepared, err := p.Provider.Prepare(ctx, spec)
	if err != nil {
		return prepared, err
	}
	if p.preparedRoot != "" {
		data, err := os.ReadFile(prepared.Path)
		if err != nil {
			return target.Prepared{}, err
		}
		if err := os.MkdirAll(p.preparedRoot, 0700); err != nil {
			return target.Prepared{}, err
		}
		if err := os.WriteFile(filepath.Join(p.preparedRoot, "module.wasm"), data, 0500); err != nil {
			return target.Prepared{}, err
		}
		for _, payload := range prepared.BackendPayloads {
			if err := os.WriteFile(filepath.Join(p.preparedRoot, filepath.Base(payload.Reference.File)), payload.Data, 0600); err != nil {
				return target.Prepared{}, err
			}
		}
	}
	return prepared, nil
}

func TestIntegratedFailureArtifact100FreshObservedReplays(t *testing.T) {
	root := os.Getenv("GOMAD_WASM_TASK4_EVIDENCE_ROOT")
	if root == "" {
		t.Skip("explicit retained evidence root selects the full 100-fresh artifact gate")
	}
	if !filepath.IsAbs(root) {
		t.Fatal("evidence root must be absolute")
	}
	if err := os.MkdirAll(root, 0700); err != nil {
		t.Fatal(err)
	}
	base := integrationProvider(t)
	options := base.options
	options.CacheRoot = filepath.Join(root, "cache")
	originalInputs := filepath.Join(t.TempDir(), "captured")
	if err := os.Mkdir(originalInputs, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(originalInputs, "input.txt"), []byte("immutable captured control"), 0600); err != nil {
		t.Fatal(err)
	}
	captured, err := readonlymount.CaptureReadOnlyMountInputs([]readonlymount.Mapping{{Source: originalInputs, Target: "/inputs"}}, readonlymount.DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	options.Config.CapturedInputs = captured
	provider, err := New(options)
	if err != nil {
		t.Fatal(err)
	}
	counted := &countedProvider{Provider: provider, preparedRoot: filepath.Join(root, "prepared")}
	fixture, err := filepath.Abs("../../gomad3/internal/gomadtool/conformance/testdata")
	if err != nil {
		t.Fatal(err)
	}
	campaign, err := runner.Explore(t.Context(), runner.CampaignSpec{Backend: counted, Seeds: "7", Parallel: 1, ExecutionTimeout: time.Minute, OverallTimeout: 3 * time.Minute, TerminateGrace: 100 * time.Millisecond, OnFailure: runner.PolicyFirst, FailureBudget: 1, OutputLimit: 1 << 20, WorldTransitionLimit: 1 << 20, Artifacts: filepath.Join(root, "artifacts"), RunnerBuild: "task4-integration", Coverage: runner.CoverageNone, Target: target.Spec{Backend: Name, Kind: target.KindGoTest, Source: "./io_failure", WorkingDir: fixture, BuildTags: []string{"gomad_fixture", "test_dep"}, Args: []string{"-test.run=^TestDeterministicIOFailure$", "-test.v"}}})
	if err != nil || campaign.Failures != 1 || len(campaign.Artifacts) != 1 {
		t.Fatalf("intentional failure: %#v %v", campaign, err)
	}
	if err := os.RemoveAll(originalInputs); err != nil {
		t.Fatal(err)
	}
	options.CompilerPath = filepath.Join(root, "compiler-deliberately-unavailable")
	replayProvider, err := New(options)
	if err != nil {
		t.Fatal(err)
	}
	fresh := &countedProvider{Provider: replayProvider}
	opened, err := artifact.OpenArtifact(campaign.Artifacts[0])
	if err != nil {
		t.Fatal(err)
	}
	manifest := opened.Manifest()
	expectedStdout, err := opened.ReadPayload("stdout", 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	expectedStderr, err := opened.ReadPayload("stderr", 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	reference := manifest.Target.Backend.Evidence
	expectedEvidence, err := opened.ReadPayload(reference.File, uint64(reference.Bytes))
	if err != nil {
		t.Fatal(err)
	}
	if err := opened.Close(); err != nil {
		t.Fatal(err)
	}
	if manifest.ReplayMode != record.ReplayObserved || manifest.IOProfile.Transcript != nil {
		t.Fatal("stock observed artifact claimed native exact transcript")
	}
	receipts := []map[string]any{}
	for i := 0; i < 100; i++ {
		observed := filepath.Join(root, "observed", fmt.Sprintf("%03d", i+1))
		result, err := runner.Replay(t.Context(), runner.ReplaySpec{Backend: fresh, ArtifactPath: campaign.Artifacts[0], ObservedDir: observed})
		if err != nil || !result.Match || !result.ObservedRepetition || result.ChoiceReplayStatus != runner.ChoiceReplayNone {
			t.Fatalf("fresh replay %d: %#v %v", i+1, result, err)
		}
		for _, payload := range []struct {
			name     string
			expected []byte
		}{{"stdout", expectedStdout}, {"stderr", expectedStderr}, {"backend-evidence.bin", expectedEvidence}} {
			actual, err := os.ReadFile(filepath.Join(observed, payload.name))
			if err != nil || !bytes.Equal(actual, payload.expected) {
				t.Fatalf("fresh replay %d %s bytes differ: %v", i+1, payload.name, err)
			}
		}
		receipts = append(receipts, map[string]any{"fresh": i + 1, "stdout_sha256": record.HashBytes(expectedStdout), "stderr_sha256": record.HashBytes(expectedStderr), "evidence_sha256": record.HashBytes(expectedEvidence), "match": true, "capability": record.ReplayObserved})
		t.Logf("fresh replay %d/100 exact raw output/evidence equality; capability=%s", i+1, record.ReplayObserved)
	}
	if counted.calls != 1 || fresh.calls != 100 {
		t.Fatalf("actual helper calls discovery=%d fresh=%d", counted.calls, fresh.calls)
	}
	receipt := map[string]any{"campaign": campaign.CampaignPath, "artifact": campaign.Artifacts[0], "discovery_executions": counted.calls, "fresh_replay_executions": fresh.calls, "record_hash": manifest.RecordHash, "failure_signature": manifest.Outcome.FailureSignature, "module_sha256": manifest.Target.SHA256, "provenance_sha256": manifest.Target.Backend.Provenance.SHA256, "helper_sha256": options.HelperSHA256, "compiler_sha256": options.CompilerSHA256, "model_sha256": options.ModelSHA256, "captured_input_sha256": captured.Manifest.SHA256, "original_inputs_removed": true, "compiler_unavailable": true, "replays": receipts}
	encoded, err := json.MarshalIndent(receipt, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "receipt.json"), append(encoded, '\n'), 0600); err != nil {
		t.Fatal(err)
	}
}
