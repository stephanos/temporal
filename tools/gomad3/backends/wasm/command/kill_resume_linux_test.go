//go:build linux

package command

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"go.temporal.io/server/tools/gomad3/backends/wasm/internal/stockgo"
	"go.temporal.io/server/tools/gomad3/backends/wasm/wasi"
	"go.temporal.io/server/tools/gomad3/runner"
)

// TestInstalledOperatorKillResume proves the supported Linux WASM path through
// the built CLI: committed-log readiness, abrupt coordinator death, lock-free
// JSON inspection, and semantically equivalent continuation.
func TestInstalledOperatorKillResume(t *testing.T) {
	compiler := os.Getenv("GOMAD3_STOCK_GO")
	if compiler == "" {
		t.Skip("explicit stock Go compiler selects the installed operator gate")
	}
	root := t.TempDir()
	binary := filepath.Join(root, "gomad")
	build := exec.CommandContext(t.Context(), compiler, "build", "-p=1", "-trimpath", "-o", binary, "../../../cmd/gomad")
	build.Env = operatorEnvironment()
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build operator: %v\n%s", err, output)
	}

	helperPath := os.Getenv("GOMAD_WASM_TEST_HELPER")
	if helperPath == "" {
		helperPath = "../wasmhost/target/release/gomad3-wasmhost"
	}
	helper, err := filepath.Abs(helperPath)
	if err != nil {
		t.Fatal(err)
	}
	runtimeRoot, err := filepath.Abs("../../../runtimesemantics")
	if err != nil {
		t.Fatal(err)
	}
	installedCompiler := filepath.Join(stockgo.Clone(t, filepath.Join(root, "go")), "bin", "go")
	manifest := filepath.Join(root, "installation.json")
	invokeOperator(t, binary, 0,
		"install", "--output", manifest, "--compiler", installedCompiler, "--helper", helper,
		"--cache", filepath.Join(root, "cache"), "--compile-cache", filepath.Join(root, "engine-cache"),
		"--profile", wasi.CooperativeProfile, "--runtime-root", runtimeRoot,
	)

	source := filepath.Join(root, "source")
	if err := os.Mkdir(source, 0o700); err != nil {
		t.Fatal(err)
	}
	for name, contents := range map[string]string{
		"go.mod": "module gomad.wasm.killresume.fixture\n\ngo 1.27.1\n",
		"work_test.go": `package fixture

import "testing"

func TestWASMKillResumeWork(t *testing.T) {}
`,
	} {
		if err := os.WriteFile(filepath.Join(source, name), []byte(contents), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	baselineRoot := filepath.Join(root, "baseline")
	invokeOperator(t, binary, 0, wasmExploreArguments(manifest, source, baselineRoot)...)
	baselinePath := wasmCampaignPath(t, baselineRoot)
	baseline := wasmInspectCampaign(t, binary, baselinePath)
	if baseline.Campaign == nil || baseline.Campaign.Attempted != 12 || baseline.Campaign.Succeeded != 12 {
		t.Fatalf("baseline campaign = %#v", baseline.Campaign)
	}

	interruptedRoot := filepath.Join(root, "interrupted")
	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Minute)
	defer cancel()
	command := exec.CommandContext(ctx, binary, wasmExploreArguments(manifest, source, interruptedRoot)...)
	command.Env = operatorEnvironment()
	command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	var output bytes.Buffer
	command.Stdout, command.Stderr = &output, &output
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	waited := make(chan error, 1)
	go func() { waited <- command.Wait() }()

	owned := map[int]bool{command.Process.Pid: true}
	t.Cleanup(func() {
		wasmDiscoverOwnedProcesses(owned, wasmCLIProcesses(t))
		for pid := range owned {
			if err := syscall.Kill(pid, syscall.SIGKILL); err != nil && !errors.Is(err, syscall.ESRCH) {
				t.Errorf("cleanup process %d: %v", pid, err)
			}
		}
	})
	var interruptedPath string
	var committed uint64
	wasmPoll(t, func() bool {
		paths, err := filepath.Glob(filepath.Join(interruptedRoot, "v1", "campaign-*", "store", "log"))
		if err != nil {
			t.Fatal(err)
		}
		if len(paths) != 1 {
			return false
		}
		interruptedPath = filepath.Dir(filepath.Dir(paths[0]))
		report, ready := wasmInspectLiveCampaign(t, binary, interruptedPath)
		if !ready || report.Lifecycle == nil || report.Committed == nil {
			return false
		}
		committed = report.Committed.Outcomes
		return report.Lifecycle.State == "running" && report.Lifecycle.Resumable && committed >= 1
	})

	if err := syscall.Kill(-command.Process.Pid, syscall.SIGSTOP); err != nil {
		t.Fatal(err)
	}
	wasmPoll(t, func() bool {
		for _, process := range wasmCLIProcesses(t) {
			if process.pid == command.Process.Pid {
				return strings.Contains(process.state, "T")
			}
		}
		return false
	})
	// The stopped CLI still owns the writer. Inspect must remain lock-free, and
	// this second read fixes the durable prefix before descendant discovery.
	frozen := wasmInspectCampaign(t, binary, interruptedPath)
	if frozen.Lifecycle == nil || frozen.Committed == nil || frozen.Lifecycle.State != "running" || !frozen.Lifecycle.Resumable || frozen.Committed.Outcomes < committed {
		t.Fatalf("stopped committed state = lifecycle %#v committed %#v, want at least %d outcomes", frozen.Lifecycle, frozen.Committed, committed)
	}
	committed = frozen.Committed.Outcomes
	wasmDiscoverOwnedProcesses(owned, wasmCLIProcesses(t))
	if err := syscall.Kill(-command.Process.Pid, syscall.SIGKILL); err != nil {
		t.Fatal(err)
	}
	for pid := range owned {
		if pid == command.Process.Pid {
			continue
		}
		if err := syscall.Kill(pid, syscall.SIGKILL); err != nil && !errors.Is(err, syscall.ESRCH) {
			t.Fatal(err)
		}
	}
	select {
	case err := <-waited:
		var exited *exec.ExitError
		if !errors.As(err, &exited) || !exited.ProcessState.Sys().(syscall.WaitStatus).Signaled() || exited.ProcessState.Sys().(syscall.WaitStatus).Signal() != syscall.SIGKILL {
			t.Fatalf("SIGKILLed explore CLI exit = %v\n%s", err, output.Bytes())
		}
	case <-ctx.Done():
		t.Fatalf("SIGKILLed explore CLI did not exit: %v", ctx.Err())
	}
	delete(owned, command.Process.Pid)
	wasmPoll(t, func() bool {
		for _, process := range wasmCLIProcesses(t) {
			if owned[process.pid] {
				return false
			}
		}
		return true
	})
	clear(owned)

	afterKill := wasmInspectCampaign(t, binary, interruptedPath)
	if afterKill.Lifecycle == nil || afterKill.Committed == nil || afterKill.Lifecycle.State != "running" || !afterKill.Lifecycle.Resumable || afterKill.Committed.Outcomes != committed {
		t.Fatalf("post-kill committed state = lifecycle %#v committed %#v, want %d outcomes", afterKill.Lifecycle, afterKill.Committed, committed)
	}
	invokeOperator(t, binary, 0, "--installation", manifest, "resume", "--json", interruptedPath)
	resumed := wasmInspectCampaign(t, binary, interruptedPath)
	if resumed.Campaign == nil || resumed.Lifecycle == nil || !resumed.Lifecycle.Published {
		t.Fatalf("resumed campaign = campaign %#v lifecycle %#v", resumed.Campaign, resumed.Lifecycle)
	}
	if left, right := wasmCampaignSemantics(*baseline.Campaign), wasmCampaignSemantics(*resumed.Campaign); !reflect.DeepEqual(left, right) {
		t.Fatalf("campaign semantics differ:\n baseline %#v\n resumed %#v", left, right)
	}
	t.Logf("built WASM CLI SIGKILL after %d committed outcomes; post-kill inspect remained resumable and resume matched 12-outcome baseline", committed)
}

func wasmExploreArguments(manifest, source, artifacts string) []string {
	return []string{
		"--installation", manifest, "explore", "--json", "--seeds=1-12", "--parallel=1",
		"--on-failure=all", "--execution-timeout=30s", "--overall-timeout=2m", "--terminate-grace=100ms",
		"--choices", "--diagnostics", "--artifacts", artifacts, "--working-dir", source,
		"go-test", ".", "--", "-test.run=^TestWASMKillResumeWork$",
	}
}

func wasmCampaignPath(t *testing.T, root string) string {
	t.Helper()
	paths, err := filepath.Glob(filepath.Join(root, "v1", "campaign-*"))
	if err != nil || len(paths) != 1 {
		t.Fatalf("campaign paths = %v: %v", paths, err)
	}
	return paths[0]
}

func wasmInspectLiveCampaign(t *testing.T, binary, path string) (runner.Inspection, bool) {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, binary, "inspect", "--json", path)
	command.Env = operatorEnvironment()
	var stdout, stderr bytes.Buffer
	command.Stdout, command.Stderr = &stdout, &stderr
	err := command.Run()
	var exited *exec.ExitError
	if errors.As(err, &exited) && exited.ExitCode() == 2 {
		return runner.Inspection{}, false
	}
	if err != nil {
		t.Fatalf("inspect %s: %v\n%s", path, err, stderr.Bytes())
	}
	var report runner.Inspection
	decoder := json.NewDecoder(&stdout)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&report); err != nil {
		t.Fatalf("inspect %s: %v\n%s", path, err, stdout.Bytes())
	}
	return report, true
}

func wasmInspectCampaign(t *testing.T, binary, path string) runner.Inspection {
	t.Helper()
	report, ready := wasmInspectLiveCampaign(t, binary, path)
	if !ready {
		t.Fatalf("campaign %s is not inspectable", path)
	}
	return report
}

func wasmCampaignSemantics(campaign runner.CampaignInspection) runner.CampaignInspection {
	campaign.CampaignID = ""
	if campaign.Journal != nil {
		journal := *campaign.Journal
		journal.IndexSHA256, journal.Segments, journal.Bytes = "", 0, 0
		campaign.Journal = &journal
	}
	campaign.RetainedSuccessBytes = 0
	campaign.FailureArtifacts = nil
	campaign.SuccessArtifacts = nil
	campaign.Executions = append([]runner.ExecutionInspection(nil), campaign.Executions...)
	for index := range campaign.Executions {
		execution := &campaign.Executions[index]
		execution.ElapsedNanos = 0
		execution.Artifact = nil
		execution.SuccessArtifact = nil
		execution.SuccessArtifactBytes = nil
	}
	return campaign
}

type wasmCLIProcess struct {
	pid, parent, group int
	state, arguments   string
}

func wasmCLIProcesses(t *testing.T) []wasmCLIProcess {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	output, err := exec.CommandContext(ctx, "ps", "-axo", "pid=,ppid=,pgid=,stat=,args=").Output()
	if err != nil {
		t.Fatal(err)
	}
	var processes []wasmCLIProcess
	for line := range strings.SplitSeq(string(output), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 5 {
			continue
		}
		pid, e1 := strconv.Atoi(fields[0])
		parent, e2 := strconv.Atoi(fields[1])
		group, e3 := strconv.Atoi(fields[2])
		if err := errors.Join(e1, e2, e3); err != nil {
			t.Fatal(err)
		}
		processes = append(processes, wasmCLIProcess{pid, parent, group, fields[3], strings.Join(fields[4:], " ")})
	}
	return processes
}

func wasmDiscoverOwnedProcesses(owned map[int]bool, processes []wasmCLIProcess) {
	for changed := true; changed; {
		changed = false
		for _, process := range processes {
			if owned[process.parent] && !owned[process.pid] {
				owned[process.pid], changed = true, true
			}
		}
	}
}

func wasmPoll(t *testing.T, condition func() bool) {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Minute)
	defer cancel()
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for !condition() {
		select {
		case <-ctx.Done():
			t.Fatalf("WASM CLI condition timed out: %v", ctx.Err())
		case <-ticker.C:
		}
	}
}

func (process wasmCLIProcess) String() string {
	return fmt.Sprintf("pid=%d ppid=%d pgid=%d state=%q args=%q", process.pid, process.parent, process.group, process.state, process.arguments)
}
