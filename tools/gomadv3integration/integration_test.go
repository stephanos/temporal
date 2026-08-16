//go:build gomadv3_integration

package gomadv3integration

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

func TestQualificationManifestsUsePortableV3(t *testing.T) {
	for _, test := range []struct {
		path      string
		module    string
		seeds     []uint64
		workloads int
	}{
		{path: filepath.Join("..", "gomadv3", "qualification", "core.json"), module: "gomadv3.core.corpus", seeds: []uint64{17}, workloads: 5},
		{path: filepath.Join("qualification", "temporal.json"), module: "go.temporal.io/server", seeds: []uint64{11, 17}, workloads: 16},
	} {
		contents, err := os.ReadFile(test.path)
		if err != nil {
			t.Fatal(err)
		}
		var manifest struct {
			Schema string   `json:"schema"`
			Module string   `json:"module"`
			Seeds  []uint64 `json:"seeds"`
			Suites []struct {
				ID              string `json:"id"`
				Tier            uint64 `json:"tier"`
				Invariant       string `json:"invariant"`
				ReplaySuccesses bool   `json:"replay_successes"`
			} `json:"suites"`
		}
		if err := json.Unmarshal(contents, &manifest); err != nil {
			t.Fatal(err)
		}
		if manifest.Schema != "gomadv3.qualification-set/v3" || manifest.Module != test.module || !slices.Equal(manifest.Seeds, test.seeds) || len(manifest.Suites) != test.workloads {
			t.Fatalf("manifest %s = %#v", test.path, manifest)
		}
		for _, suite := range manifest.Suites {
			if suite.ID == "" || suite.Tier == 0 || suite.Invariant == "" || !suite.ReplaySuccesses {
				t.Fatalf("manifest %s suite = %#v", test.path, suite)
			}
		}
	}
}

func TestPublicWrappers(t *testing.T) {
	repositoryRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name       string
		arguments  []string
		wantStatus int
		wantOutput string
	}{
		{name: "run requires seed", arguments: []string{"gomadv3-run"}, wantStatus: 2, wantOutput: "GOMADSEED is required"},
		{name: "run requires target", arguments: []string{"gomadv3-run", "GOMADSEED=1"}, wantStatus: 2, wantOutput: "GOMADV3_RUN is required"},
		{name: "test requires packages", arguments: []string{"gomadv3-test", "GOMADSEED=1"}, wantStatus: 2, wantOutput: "GOMADV3_PACKAGES is required"},
	} {
		t.Run(test.name, func(t *testing.T) {
			output, status := runMake(t, repositoryRoot, test.arguments...)
			if status != test.wantStatus || !strings.Contains(output, test.wantOutput) {
				t.Fatalf("status=%d output=%q", status, output)
			}
		})
	}

	output, status := runMake(t, repositoryRoot, "gomadv3-run", "GOMADSEED=103", "GOMADV3_RUN=./tools/gomadv3/toolchain/internal/conformance/testdata/clock/main.go", "GOMADV3_ARGS=initial")
	if status != 0 || !strings.HasSuffix(strings.TrimSpace(output), "clock initial ok") {
		t.Fatalf("gomadv3-run status=%d output=%q", status, output)
	}

	first, status := runMake(t, repositoryRoot, "gomadv3-test", "GOMADSEED=101", "GOMADV3_PACKAGES=./tools/gomadv3integration/testdata/tagged", "GOMADV3_ARGS=-run=TestTargetRequiresTestDep")
	if status != 0 {
		t.Fatalf("first gomadv3-test status=%d output=%q", status, first)
	}
	second, status := runMake(t, repositoryRoot, "gomadv3-test", "GOMADSEED=102", "GOMADV3_PACKAGES=./tools/gomadv3integration/testdata/tagged", "GOMADV3_ARGS=-run=TestTargetRequiresTestDep")
	if status != 0 || strings.Contains(first, "(cached)") || strings.Contains(second, "(cached)") || strings.Contains(first, "[no test files]") || strings.Contains(second, "[no test files]") {
		t.Fatalf("second gomadv3-test status=%d first=%q second=%q", status, first, second)
	}

	output, status = runMake(t, repositoryRoot, "gomadv3-test", "GOMADSEED=", "GOMADV3_PACKAGES=./tools/gomadv3integration/testdata/tagged", "GOMADV3_ARGS=-run=TestTargetRequiresTestDep")
	if status != 2 || !strings.Contains(output, "runtime: invalid GOMADSEED") {
		t.Fatalf("empty-seed gomadv3-test status=%d output=%q", status, output)
	}
}

func runMake(t *testing.T, repositoryRoot string, arguments ...string) (string, int) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	command := exec.CommandContext(ctx, "make", append([]string{"--no-print-directory", "-C", repositoryRoot}, arguments...)...)
	var output bytes.Buffer
	command.Stdout = &output
	command.Stderr = &output
	err := command.Run()
	if ctx.Err() != nil {
		t.Fatal(ctx.Err())
	}
	if err == nil {
		return output.String(), 0
	}
	var exitError *exec.ExitError
	if !errors.As(err, &exitError) {
		t.Fatal(err)
	}
	return output.String(), exitError.ExitCode()
}
