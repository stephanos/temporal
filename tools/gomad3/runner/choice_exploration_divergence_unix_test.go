//go:build unix

package runner

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"syscall"
	"testing"
	"time"

	"go.temporal.io/server/tools/gomad3/runner/internal/campaign"
)

func TestChoiceExplorationDivergenceCrashHelper(t *testing.T) {
	root := os.Getenv("GOMAD3_DIVERGENCE_CRASH_ARTIFACTS")
	if root == "" {
		return
	}
	config, configDependencies := divergenceCampaignConfig(t, PolicyAll, 1)
	config.Artifacts = root
	config.Progress = func(event CampaignEvent) error {
		if event.ChoiceExploration != nil && event.ChoiceExploration.CommittedRounds == 2 {
			if _, err := fmt.Fprintln(os.Stdout, "COMMITTED_DIVERGENCE "+event.CampaignPath); err != nil {
				return err
			}
			_, err := io.ReadFull(os.Stdin, make([]byte, 1))
			return err
		}
		return nil
	}
	if _, err := exploreWith(t.Context(), config, configDependencies); err != nil {
		t.Fatal(err)
	}
	t.Fatal("crash helper escaped the committed-round barrier")
}

func TestRunChoiceExplorationKilledAfterDivergenceResumesSameState(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 45*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestChoiceExplorationDivergenceCrashHelper$")
	command.Env = append(os.Environ(), "GOMAD3_DIVERGENCE_CRASH_ARTIFACTS="+t.TempDir())
	output, err := command.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	input, err := command.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	defer input.Close()
	var stderr bytes.Buffer
	command.Stderr = &stderr
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	waited := make(chan error, 1)
	reaped := false
	go func() { waited <- command.Wait() }()
	t.Cleanup(func() {
		if !reaped {
			if err := command.Process.Kill(); err != nil && !errors.Is(err, os.ErrProcessDone) {
				t.Error(err)
			}
			<-waited
			reaped = true
		}
	})
	ready := make(chan string, 1)
	go func() {
		scanner := bufio.NewScanner(output)
		for scanner.Scan() {
			if path, ok := strings.CutPrefix(scanner.Text(), "COMMITTED_DIVERGENCE "); ok {
				ready <- path
				return
			}
		}
		ready <- ""
	}()
	var path string
	select {
	case path = <-ready:
	case <-ctx.Done():
		t.Fatal("committed-round barrier timed out")
	}
	if path == "" {
		select {
		case waitErr := <-waited:
			reaped = true
			t.Fatalf("crash helper failed: %v: %s", waitErr, stderr.String())
		case <-ctx.Done():
			t.Fatal("crash helper did not exit after closing stdout")
		}
	}
	committedPath := filepath.Join(path, "choice-exploration", "rounds", "00000000000000000001", "segment.json")
	if _, err := os.Stat(committedPath); err != nil {
		t.Fatal(err)
	}
	if err := syscall.Kill(command.Process.Pid, syscall.SIGKILL); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-waited:
		reaped = true
		var exited *exec.ExitError
		if !errors.As(err, &exited) || exited.Sys().(syscall.WaitStatus).Signal() != syscall.SIGKILL {
			t.Fatalf("helper kill = %v", err)
		}
	case <-ctx.Done():
		t.Fatal("killed helper did not exit")
	}
	config, configDependencies := divergenceCampaignConfig(t, PolicyAll, 1)
	base := configDependencies.executor.(*candidateDivergenceExecutor).base
	resumed, err := exploreWith(t.Context(), CampaignSpec{ResumeCampaign: path, RunnerBuild: config.RunnerBuild, SupervisorCommand: []string{"unused"}}, executionDependencies{executor: base})
	if err != nil {
		t.Fatal(err)
	}
	uninterrupted, err := exploreWith(t.Context(), config, configDependencies)
	if err != nil {
		t.Fatal(err)
	}
	resumedBatch, err := campaign.OpenCampaign(path)
	if err != nil {
		t.Fatal(err)
	}
	uninterruptedBatch, err := campaign.OpenCampaign(uninterrupted.CampaignPath)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(resumed.ChoiceExploration, uninterrupted.ChoiceExploration) || resumed.ReplayDivergences != 1 || resumed.Attempted != 4 || resumed.RecoveryExecutions != 0 || resumedBatch.Record.ChoiceExplorationChainSHA256 != uninterruptedBatch.Record.ChoiceExplorationChainSHA256 {
		t.Fatalf("crash/resume differs: %#v, %#v", resumed, uninterrupted)
	}
	t.Logf("SIGKILL pid=%d after committed divergent round1; resumed canonical chain=%s equals uninterrupted", command.Process.Pid, resumedBatch.Record.ChoiceExplorationChainSHA256)
}
