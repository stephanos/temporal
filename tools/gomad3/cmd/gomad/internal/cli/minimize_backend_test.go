package cli

import (
	"bytes"
	"context"
	"errors"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomad3/artifact"
	"go.temporal.io/server/tools/gomad3/runner"
)

func TestExternalMinimizeForwardsProviderBoundsResumeAndSharedOutput(t *testing.T) {
	defaultRoot, err := filepath.Abs(".gomad/artifacts")
	if err != nil {
		t.Fatal(err)
	}
	provider := commandBackend{}
	for _, test := range []struct {
		name      string
		arguments []string
		want      runner.MinimizeSpec
		output    string
	}{
		{
			name: "defaults", arguments: []string{"/failure"},
			want:   runner.MinimizeSpec{Backend: provider, ArtifactPath: "/failure", OutputRoot: filepath.Join(defaultRoot, "minimized"), AttemptBudget: 64},
			output: "gomad minimize: changed=true attempts=2/64 accepted=0 stop=minimal artifact=/minimized\n",
		},
		{
			name: "resume bounded root", arguments: []string{"--resume", "--artifacts=/artifacts", "--attempt-budget=4", "--max-bytes=2MiB", "/failure"},
			want:   runner.MinimizeSpec{Backend: provider, ArtifactPath: "/failure", OutputRoot: "/artifacts/minimized", AttemptBudget: 4, MaximumBytes: 2 << 20, Resume: true},
			output: "gomad minimize: changed=true attempts=2/4 accepted=0 stop=minimal artifact=/minimized\n",
		},
		{
			name: "shared JSON", arguments: []string{"--json", "--artifacts=/artifacts", "--attempt-budget=4", "/failure"},
			want:   runner.MinimizeSpec{Backend: provider, ArtifactPath: "/failure", OutputRoot: "/artifacts/minimized", AttemptBudget: 4},
			output: "{\"artifact_path\":\"/minimized\",\"record_hash\":\"\",\"changed\":true,\"attempts\":2,\"attempt_budget\":4,\"accepted\":null,\"stop_reason\":\"minimal\"}\n",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			var observed []runner.MinimizeSpec
			var stdout, stderr bytes.Buffer
			status := runMinimizeWith(test.arguments, &stdout, &stderr, minimizeDependencies{
				install: func(root string) (installation, error) {
					if root != "" {
						t.Fatalf("external minimization selected native toolchain: %s", root)
					}
					return installation{backend: provider, backendName: "wasm"}, nil
				},
				minimize: func(_ context.Context, spec runner.MinimizeSpec) (runner.MinimizeResult, error) {
					observed = append(observed, spec)
					return runner.MinimizeResult{Artifact: artifact.Artifact{Path: "/minimized"}, Changed: true, Attempts: 2, AttemptBudget: test.want.AttemptBudget, StopReason: "minimal"}, nil
				},
			})
			if status != 0 || stderr.Len() != 0 || len(observed) != 1 || !reflect.DeepEqual(observed[0], test.want) || stdout.String() != test.output {
				t.Fatalf("status=%d requests=%#v want=%#v stdout=%q stderr=%q", status, observed, test.want, stdout.String(), stderr.String())
			}
		})
	}
}

func TestExternalMinimizePreservesErrorClassification(t *testing.T) {
	for _, test := range []struct {
		name   string
		err    error
		status int
	}{
		{"preflight", &runner.ReplayPreflightError{Err: errors.New("incompatible retained choice failure")}, 2},
		{"runner", errors.New("minimization infrastructure failure"), 3},
	} {
		t.Run(test.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			status := runMinimizeWith([]string{"/failure"}, &stdout, &stderr, minimizeDependencies{
				install: func(string) (installation, error) {
					return installation{backend: commandBackend{}, backendName: "wasm"}, nil
				},
				minimize: func(_ context.Context, spec runner.MinimizeSpec) (runner.MinimizeResult, error) {
					if spec.Backend == nil || spec.ToolchainRoot != "" {
						t.Fatalf("external backend lost or native launch configured: %+v", spec)
					}
					return runner.MinimizeResult{}, test.err
				},
			})
			if status != test.status || stdout.Len() != 0 || !strings.Contains(stderr.String(), test.err.Error()) {
				t.Fatalf("status=%d stdout=%q stderr=%q", status, stdout.String(), stderr.String())
			}
		})
	}
}
