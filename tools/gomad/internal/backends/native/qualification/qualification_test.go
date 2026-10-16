package qualification

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

func TestRootMakeWrapperInputs(t *testing.T) {
	repositoryRoot, err := filepath.Abs(filepath.Join("..", "..", "..", "..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name       string
		arguments  []string
		wantStatus int
		wantOutput string
	}{
		{name: "run requires seed", arguments: []string{"gomad-run"}, wantStatus: 2, wantOutput: "GOMADSEED is required"},
		{name: "run requires target", arguments: []string{"gomad-run", "GOMADSEED=1"}, wantStatus: 2, wantOutput: "GOMAD_RUN is required"},
		{name: "test requires packages", arguments: []string{"gomad-test", "GOMADSEED=1"}, wantStatus: 2, wantOutput: "GOMAD_PACKAGES is required"},
	} {
		t.Run(test.name, func(t *testing.T) {
			output, status := runMake(t, repositoryRoot, test.arguments...)
			if status != test.wantStatus || !strings.Contains(output, test.wantOutput) {
				t.Fatalf("status=%d output=%q", status, output)
			}
		})
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
