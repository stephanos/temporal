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

	"go.temporal.io/server/tools/gomad/internal/backends/target"
	simulation "go.temporal.io/server/tools/gomad/internal/backends/wasm/cluster"
	"go.temporal.io/server/tools/gomad/internal/backends/wasm/installation"
	"go.temporal.io/server/tools/gomad/internal/backends/wasm/internal/stockgo"
	"go.temporal.io/server/tools/gomad/internal/backends/wasm/wasi"
	"go.temporal.io/server/tools/gomad/internal/evidence/artifact"
	"go.temporal.io/server/tools/gomad/internal/runner/inspect"
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
	build := exec.CommandContext(t.Context(), compiler, "build", "-p=1", "-trimpath", "-o", binary, "../../../../cmd/gomad")
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
	runtimeRoot, err := filepath.Abs("../../../model/runtimesource")
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

	wasmRunKillResume(t, root, binary, manifest, 12, false, func(artifacts string) []string {
		return wasmExploreArguments(manifest, source, artifacts)
	}, func() {})
}

func TestInstalledOperatorTypedScenarioKillResume(t *testing.T) {
	f := newTypedOperator(t)
	scenario := filepath.Join(f.root, "mixed.json")
	writeTypedJSON(t, scenario, typedScenario(t, "durable", "mixed"))
	wasmRunKillResume(t, f.root, f.binary, f.manifest, 24, true, func(artifacts string) []string {
		return f.explore(scenario, artifacts, "1-24")
	}, func() { f.detach(t, scenario) })
}

func wasmRunKillResume(t *testing.T, root, binary, manifest string, total uint64, mixed bool, explore func(string) []string, detach func()) {
	t.Helper()
	baselineRoot := filepath.Join(root, "baseline")
	want := 0
	if mixed {
		want = 1
	}
	invokeOperator(t, binary, want, explore(baselineRoot)...)
	baselinePath := wasmCampaignPath(t, baselineRoot)
	baseline := wasmInspectCampaign(t, binary, baselinePath)
	if baseline.Campaign == nil || baseline.Campaign.Attempted != total || baseline.Campaign.Succeeded+baseline.Campaign.Failures != total || mixed && (baseline.Campaign.Succeeded == 0 || baseline.Campaign.Failures == 0) || !mixed && baseline.Campaign.Succeeded != total {
		t.Fatalf("baseline campaign = %#v", baseline.Campaign)
	}
	minimumCommitted := uint64(1)
	if mixed {
		seen := map[string]bool{}
		for _, execution := range baseline.Campaign.Executions {
			seen[execution.Domain] = true
			if seen["success"] && seen["target"] {
				minimumCommitted = execution.SelectionOrdinal + 1
				break
			}
		}
		if minimumCommitted >= total {
			t.Fatal("mixed control has no interruptible completed success/failure prefix")
		}
	}

	interruptedRoot := filepath.Join(root, "interrupted")
	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Minute)
	defer cancel()
	command := exec.CommandContext(ctx, binary, explore(interruptedRoot)...)
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
		if report.Lifecycle.State != "running" || !report.Lifecycle.Resumable || committed < minimumCommitted || committed >= total {
			return false
		}
		if mixed {
			for _, admitted := range report.Committed.Admitted {
				if admitted.Receipt == nil {
					return true
				}
			}
			return false
		}
		return true
	})

	if err := syscall.Kill(-command.Process.Pid, syscall.SIGSTOP); err != nil {
		t.Fatal(err)
	}
	wasmDiscoverOwnedProcesses(owned, wasmCLIProcesses(t))
	for pid := range owned {
		if err := syscall.Kill(pid, syscall.SIGSTOP); err != nil && !errors.Is(err, syscall.ESRCH) {
			t.Fatal(err)
		}
	}
	wasmPoll(t, func() bool {
		for _, process := range wasmCLIProcesses(t) {
			if owned[process.pid] && !strings.ContainsAny(process.state, "TZ") {
				return false
			}
		}
		return true
	})
	// The stopped CLI still owns the writer. Inspect must remain lock-free, and
	// this second read fixes the durable prefix before descendant discovery.
	frozen := wasmInspectCampaign(t, binary, interruptedPath)
	if frozen.Lifecycle == nil || frozen.Committed == nil || frozen.Lifecycle.State != "running" || !frozen.Lifecycle.Resumable || frozen.Committed.Outcomes < committed {
		t.Fatalf("stopped committed state = lifecycle %#v committed %#v, want at least %d outcomes", frozen.Lifecycle, frozen.Committed, committed)
	}
	committed = frozen.Committed.Outcomes
	if committed >= total {
		t.Fatalf("campaign completed before freeze: %d/%d", committed, total)
	}
	frozenLog, err := os.ReadFile(filepath.Join(interruptedPath, "store", "log"))
	if err != nil {
		t.Fatal(err)
	}
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
	postKillLog, err := os.ReadFile(filepath.Join(interruptedPath, "store", "log"))
	if err != nil || !bytes.Equal(frozenLog, postKillLog) {
		t.Fatalf("frozen durable log changed after killing descendants: %v", err)
	}
	var frozenArtifacts map[string]string
	if mixed {
		frozenArtifacts = wasmRetainedArtifactHashes(t, interruptedPath, committed)
		detach()
		manifestBytes, err := os.ReadFile(manifest)
		if err != nil {
			t.Fatal(err)
		}
		var incompatible installation.Manifest
		if err := json.Unmarshal(manifestBytes, &incompatible); err != nil {
			t.Fatal(err)
		}
		incompatible.Options.ModelSHA256 = "sha256:" + strings.Repeat("0", 64)
		incompatiblePath := filepath.Join(root, "incompatible-installation.json")
		writeTypedJSON(t, incompatiblePath, incompatible)
		invokeOperator(t, binary, 3, "--installation", incompatiblePath, "resume", "--json", interruptedPath)
		if got := wasmRetainedArtifactHashes(t, interruptedPath, committed); !reflect.DeepEqual(got, frozenArtifacts) {
			t.Fatal("refused resume overwrote completed evidence")
		}
		for _, damage := range []string{"missing-module", "corrupt-log"} {
			copyPath := filepath.Join(root, damage)
			copyOperatorTree(t, copyPath, interruptedPath)
			before := wasmRetainedArtifactHashes(t, copyPath, committed)
			if damage == "missing-module" {
				removed := false
				if err := filepath.WalkDir(filepath.Join(copyPath, ".prepared"), func(path string, entry os.DirEntry, err error) error {
					if err != nil {
						return err
					}
					if entry.Name() == "module.wasm" && !entry.IsDir() {
						removed = true
						return os.Remove(path)
					}
					return nil
				}); err != nil || !removed {
					t.Fatalf("remove retained prepared module: removed=%t err=%v", removed, err)
				}
				invokeOperator(t, binary, 2, "--installation", manifest, "resume", "--json", copyPath)
			} else {
				logPath := filepath.Join(copyPath, "store", "log")
				log, err := os.ReadFile(logPath)
				if err != nil || len(log) == 0 {
					t.Fatalf("read campaign log: %v", err)
				}
				log[0] ^= 1
				if err := os.WriteFile(logPath, log, 0o600); err != nil {
					t.Fatal(err)
				}
				invokeOperator(t, binary, 2, "recover", "--json", copyPath)
			}
			if after := wasmRetainedArtifactHashes(t, copyPath, committed); !reflect.DeepEqual(before, after) {
				t.Fatalf("%s refusal overwrote valid completed artifacts", damage)
			}
		}
		invokeOperator(t, binary, 0, "recover", "--json", interruptedPath)
		recovered := wasmInspectCampaign(t, binary, interruptedPath)
		if recovered.Committed == nil || recovered.Committed.Outcomes != committed || recovered.Lifecycle == nil || !recovered.Lifecycle.Resumable || recovered.Lifecycle.Published {
			t.Fatalf("recovery changed committed prefix or published incomplete work: %+v", recovered)
		}
	}
	invokeOperator(t, binary, want, "--installation", manifest, "resume", "--json", interruptedPath)
	resumed := wasmInspectCampaign(t, binary, interruptedPath)
	if resumed.Campaign == nil || resumed.Lifecycle == nil || !resumed.Lifecycle.Published {
		t.Fatalf("resumed campaign = campaign %#v lifecycle %#v", resumed.Campaign, resumed.Lifecycle)
	}
	if left, right := wasmCampaignSemantics(*baseline.Campaign), wasmCampaignSemantics(*resumed.Campaign); !reflect.DeepEqual(left, right) {
		t.Fatalf("campaign semantics differ:\n baseline %#v\n resumed %#v", left, right)
	}
	if mixed {
		preserved := wasmRetainedArtifactHashes(t, interruptedPath, committed)
		for path, hash := range frozenArtifacts {
			if preserved[path] != hash {
				t.Fatalf("resume changed frozen completed artifact %s", path)
			}
		}
		if left, right := wasmClusterRecordIdentities(t, baselinePath), wasmClusterRecordIdentities(t, interruptedPath); !reflect.DeepEqual(left, right) {
			t.Fatalf("retained lifecycle/network/storage logical records changed: baseline=%v resumed=%v", left, right)
		}
	}
	t.Logf("built WASM CLI SIGKILL after %d committed outcomes; post-kill inspect remained resumable and resume matched %d-outcome baseline", committed, total)
}

func wasmRetainedArtifactHashes(t *testing.T, campaign string, committed uint64) map[string]string {
	t.Helper()
	hashes := map[string]string{}
	domains := map[string]bool{}
	for _, path := range typedArtifactPaths(t, campaign) {
		opened, err := artifact.OpenArtifact(path)
		if err != nil {
			t.Fatal(err)
		}
		manifest := opened.Manifest()
		if uint64(manifest.SelectionOrdinal) < committed {
			hashes[path] = string(manifest.RecordHash)
			domains[manifest.Outcome.Domain] = true
		}
		if err := opened.Close(); err != nil {
			t.Fatal(err)
		}
	}
	if len(hashes) == 0 || !domains["success"] || !domains["target"] {
		t.Fatalf("committed prefix lacks retained success/failure evidence: outcomes=%d artifacts=%d domains=%v", committed, len(hashes), domains)
	}
	return hashes
}

func wasmClusterRecordIdentities(t *testing.T, campaign string) map[uint64]string {
	t.Helper()
	identities := map[uint64]string{}
	for _, path := range typedArtifactPaths(t, campaign) {
		opened, err := artifact.OpenArtifact(path)
		if err != nil {
			t.Fatal(err)
		}
		reference := *opened.Manifest().Target.Backend.Evidence
		data, err := opened.ReadPayload(reference.File, 128<<20)
		if err != nil {
			t.Fatal(err)
		}
		_, record, err := simulation.VerifyRetained(target.BackendPayload{Reference: reference, Data: data})
		if err != nil {
			t.Fatal(err)
		}
		identities[record.Seed] = record.Identity
		if err := opened.Close(); err != nil {
			t.Fatal(err)
		}
	}
	return identities
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

func wasmInspectLiveCampaign(t *testing.T, binary, path string) (inspect.Inspection, bool) {
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
		return inspect.Inspection{}, false
	}
	if err != nil {
		t.Fatalf("inspect %s: %v\n%s", path, err, stderr.Bytes())
	}
	var report inspect.Inspection
	decoder := json.NewDecoder(&stdout)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&report); err != nil {
		t.Fatalf("inspect %s: %v\n%s", path, err, stdout.Bytes())
	}
	return report, true
}

func wasmInspectCampaign(t *testing.T, binary, path string) inspect.Inspection {
	t.Helper()
	report, ready := wasmInspectLiveCampaign(t, binary, path)
	if !ready {
		t.Fatalf("campaign %s is not inspectable", path)
	}
	return report
}

func wasmCampaignSemantics(campaign inspect.CampaignInspection) inspect.CampaignInspection {
	campaign.CampaignID = ""
	if campaign.Journal != nil {
		journal := *campaign.Journal
		journal.IndexSHA256, journal.Segments, journal.Bytes = "", 0, 0
		campaign.Journal = &journal
	}
	campaign.RetainedSuccessBytes = 0
	campaign.FailureArtifacts = nil
	campaign.SuccessArtifacts = nil
	campaign.Executions = append([]inspect.ExecutionInspection(nil), campaign.Executions...)
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
