//go:build gomadv3_integration

package gomadv3integration

import (
	"bytes"
	"context"
	"errors"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

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

	output, status := runMake(t, repositoryRoot, "gomadv3-run", "GOMADSEED=103", "GOMADV3_RUN=./tools/gomadv3/testdata/clock/main.go", "GOMADV3_ARGS=initial")
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
