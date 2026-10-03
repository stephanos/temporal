//go:build unix

package main

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

	"go.temporal.io/server/tools/gomad3/record"
	"go.temporal.io/server/tools/gomad3/runner"
)

var cliBinary, cliModuleRoot, cliToolchainRoot string

func TestMain(m *testing.M) {
	root, err := filepath.Abs("../..")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	cliModuleRoot, cliToolchainRoot = root, filepath.Join(root, ".toolchain")
	directory, err := os.MkdirTemp("", "gomad-cli-e2e-")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	cliBinary = filepath.Join(directory, "gomad")
	build := exec.Command(filepath.Join(cliToolchainRoot, "bin", "go"), "build", "-trimpath", "-o", cliBinary, "./cmd/gomad")
	build.Dir, build.Env = cliModuleRoot, cliEnvironment()
	output, err := build.CombinedOutput()
	status := 1
	if err != nil {
		fmt.Fprintf(os.Stderr, "build CLI: %v\n%s", err, output)
	} else {
		status = m.Run()
	}
	if err := os.RemoveAll(directory); err != nil {
		fmt.Fprintln(os.Stderr, err)
		status = 1
	}
	os.Exit(status)
}

func cliEnvironment() []string {
	var environment []string
	for _, entry := range os.Environ() {
		name, _, _ := strings.Cut(entry, "=")
		switch name {
		case "GOROOT", "GOMADSEED", "GOMAD3_CHILD_SEED", "GOTOOLCHAIN", "GOWORK", "GOFLAGS", "CGO_ENABLED":
			continue
		}
		environment = append(environment, entry)
	}
	return append(environment, "GOTOOLCHAIN=local", "GOWORK=off", "GOFLAGS=-tags=test_dep", "CGO_ENABLED=0")
}

func cliCommand(ctx context.Context, arguments ...string) *exec.Cmd {
	command := exec.CommandContext(ctx, cliBinary, arguments...)
	command.Dir, command.Env = cliModuleRoot, cliEnvironment()
	return command
}

func runCLI(t *testing.T, status int, arguments ...string) []byte {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 90*time.Second)
	defer cancel()
	output, err := cliCommand(ctx, arguments...).CombinedOutput()
	got := 0
	if err != nil {
		var exited *exec.ExitError
		if !errors.As(err, &exited) {
			t.Fatalf("CLI %v: %v\n%s", arguments, err, output)
		}
		got = exited.ExitCode()
	}
	if got != status {
		t.Fatalf("CLI %v status = %d, want %d\n%s", arguments, got, status, output)
	}
	return output
}

func TestPlanTextAndJSONPreserveCampaignIdentityAndTargetArguments(t *testing.T) {
	var identity record.SHA256
	for _, format := range []string{"text", "json"} {
		t.Run(format, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "campaign.plan.json")
			arguments := []string{"plan", "--toolchain-root", cliToolchainRoot, "--parallel=1", "--seeds=2", "--env=FOO=bar", "--build-tag=gomad_fixture"}
			if format == "json" {
				arguments = append(arguments, "--json")
			}
			arguments = append(arguments, "--output", path, "go-run", "./cmd/gomad/testdata/campaign", "--", "work", "literal;$value")
			output := runCLI(t, 0, arguments...)
			encoded, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			var plan struct {
				Campaign struct {
					Selection   string                         `json:"selection"`
					OnFailure   string                         `json:"on_failure"`
					Environment []struct{ Name, Value string } `json:"environment"`
					Prepared    struct {
						Target struct {
							Args      []string `json:"argv"`
							BuildTags []string `json:"build_tags"`
						} `json:"target"`
					} `json:"prepared"`
				} `json:"campaign"`
			}
			if err := json.Unmarshal(encoded, &plan); err != nil {
				t.Fatal(err)
			}
			if plan.Campaign.Selection != "2" || plan.Campaign.OnFailure != "all" || !reflect.DeepEqual(plan.Campaign.Prepared.Target.Args, []string{"gomad3-target", "work", "literal;$value"}) || !reflect.DeepEqual(plan.Campaign.Prepared.Target.BuildTags, []string{"gomad_fixture"}) {
				t.Fatalf("plan = %#v", plan)
			}
			if len(plan.Campaign.Environment) != 2 || plan.Campaign.Environment[0].Name != "FOO" || plan.Campaign.Environment[0].Value != "bar" {
				t.Fatalf("environment = %#v", plan.Campaign.Environment)
			}
			digest := record.HashBytes(encoded)
			if identity != "" && digest != identity {
				t.Fatalf("plan identity = %s, want %s", digest, identity)
			}
			identity = digest
			if format == "json" {
				var result runner.CampaignPlanResult
				if err := json.Unmarshal(output, &result); err != nil || result.SHA256 != digest || result.Path != path {
					t.Fatalf("JSON result = %#v, %v, output=%q", result, err, output)
				}
			} else if !strings.Contains(string(output), "gomad plan: path="+path+" bundle="+path+".bundle sha256="+string(digest)+" selected=1 target=") {
				t.Fatalf("text result = %q", output)
			}
		})
	}
}

func exploreArguments(root, seeds string, work bool) []string {
	arguments := []string{"explore", "--json", "--toolchain-root", cliToolchainRoot, "--artifacts", root,
		"--seeds", seeds, "--parallel", "1", "--on-failure", "all", "--execution-timeout", "30s",
		"--overall-timeout", "1m", "--terminate-grace", "100ms", "--keep-successes", "all", "--success-limit", "3", "--success-bytes", "64MiB",
		"go-run", "./cmd/gomad/testdata/campaign"}
	if work {
		arguments = append(arguments, "--", "work")
	}
	return arguments
}

func campaignPath(t *testing.T, root string) string {
	t.Helper()
	paths, err := filepath.Glob(filepath.Join(root, "v1", "campaign-*"))
	if err != nil || len(paths) != 1 {
		t.Fatalf("campaign paths = %v: %v", paths, err)
	}
	return paths[0]
}

func inspectCLI(t *testing.T, path string) runner.Inspection {
	t.Helper()
	output := runCLI(t, 0, "inspect", "--json", path)
	var report runner.Inspection
	decoder := json.NewDecoder(bytes.NewReader(output))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&report); err != nil {
		t.Fatalf("inspect %s: %v\n%s", path, err, output)
	}
	return report
}

func TestCLIExploreReplay(t *testing.T) {
	root := t.TempDir()
	runCLI(t, 1, exploreArguments(root, "1-3", false)...)
	path := campaignPath(t, root)
	campaign := inspectCLI(t, path).Campaign
	if campaign == nil || campaign.Attempted != 3 || campaign.Succeeded != 2 || campaign.Failures != 1 || campaign.Watchdogs != 0 || campaign.DistinctFailures != 1 || campaign.RetainedSuccesses != 2 {
		t.Fatalf("campaign = %#v", campaign)
	}
	if len(campaign.FailureArtifacts) != 1 || len(campaign.SuccessArtifacts) != 2 {
		t.Fatalf("retained artifacts = %#v", campaign)
	}
	failure := campaign.FailureArtifacts[0].Path
	observed := inspectCLI(t, failure).Artifact
	if observed == nil || observed.Seed != 2 || observed.SelectionOrdinal != 1 || observed.Outcome.Domain != "target" || observed.Transcript == nil || observed.Transcript.Records == 0 {
		t.Fatalf("failure = %#v", observed)
	}
	runCLI(t, 0, "replay", "--toolchain-root", cliToolchainRoot, "--verify-only", failure)
	output := runCLI(t, 1, "replay", "--toolchain-root", cliToolchainRoot, failure)
	if !bytes.Contains(output, []byte("reproduced=true")) {
		t.Fatalf("failure replay did not reproduce: %s", output)
	}
	runCLI(t, 0, "replay", "--toolchain-root", cliToolchainRoot, campaign.SuccessArtifacts[0].Path)
	runCLI(t, 2, "replay", "--toolchain-root", cliToolchainRoot, filepath.Join(root, "missing"))
	output = runCLI(t, 2, "resume", "--json", "--toolchain-root", cliToolchainRoot, path)
	if !bytes.Contains(output, []byte("invalid_input")) {
		t.Fatalf("published resume was not rejected as invalid input: %s", output)
	}
	inspectCLI(t, path)
}

func TestCLIKillResume(t *testing.T) {
	root := t.TempDir()
	runCLI(t, 1, exploreArguments(root, "1-3", true)...)
	baselinePath := campaignPath(t, root)
	baseline := inspectCLI(t, baselinePath).Campaign
	if baseline == nil || baseline.Attempted != 3 || baseline.Watchdogs != 0 || baseline.Succeeded != 2 || baseline.Failures != 1 {
		t.Fatalf("baseline campaign = %#v", baseline)
	}
	t.Run("journal_capacity_negative_control", func(t *testing.T) {
		if baseline.Journal == nil {
			t.Fatal("baseline inspection omitted its execution journal")
		}
		changed := *baseline
		journal := *baseline.Journal
		changed.Journal = &journal
		journal.Limits.MaximumBytes++
		if reflect.DeepEqual(campaignSemantics(*baseline), campaignSemantics(changed)) {
			t.Fatal("campaign comparison discarded a differing journal MaximumBytes")
		}
	})
	for _, completed := range []int{2, 0} {
		t.Run(fmt.Sprintf("completed_%d", completed), func(t *testing.T) {
			path := killCampaign(t, completed)
			runCLI(t, 1, "resume", "--json", "--toolchain-root", cliToolchainRoot, path)
			resumed := inspectCLI(t, path).Campaign
			compareCampaigns(t, baselinePath, baseline, path, resumed)
		})
	}
}

type cliProcess struct {
	pid, parent, group int
	state, arguments   string
}

func cliProcesses(t *testing.T) []cliProcess {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	output, err := exec.CommandContext(ctx, "ps", "-axo", "pid=,ppid=,pgid=,stat=,args=").Output()
	if err != nil {
		t.Fatal(err)
	}
	var processes []cliProcess
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
		processes = append(processes, cliProcess{pid, parent, group, fields[3], strings.Join(fields[4:], " ")})
	}
	return processes
}

func discoverOwnedProcesses(owned map[int]bool, processes []cliProcess) {
	for changed := true; changed; {
		changed = false
		for _, process := range processes {
			if owned[process.parent] && !owned[process.pid] {
				owned[process.pid], changed = true, true
			}
		}
	}
}

func pollCLI(t *testing.T, condition func() bool) {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 45*time.Second)
	defer cancel()
	ticker := time.NewTicker(time.Millisecond)
	defer ticker.Stop()
	for !condition() {
		select {
		case <-ctx.Done():
			t.Fatalf("CLI condition timed out: %v", ctx.Err())
		case <-ticker.C:
		}
	}
}

func journalCount(t *testing.T, path string) int {
	t.Helper()
	count := 0
	for _, directory := range []string{"executions", ".partial/executions"} {
		paths, err := filepath.Glob(filepath.Join(path, directory, "*.jsonl"))
		if err != nil {
			t.Fatal(err)
		}
		for _, path := range paths {
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			count += bytes.Count(data, []byte{'\n'})
		}
	}
	return count
}

func killCampaign(t *testing.T, completed int) string {
	t.Helper()
	root := t.TempDir()
	ctx, cancel := context.WithTimeout(t.Context(), time.Minute)
	defer cancel()
	command := cliCommand(ctx, exploreArguments(root, "1-3", true)...)
	var output bytes.Buffer
	command.Stdout, command.Stderr = &output, &output
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	waited := make(chan error, 1)
	go func() { waited <- command.Wait() }()
	var coordinator cliProcess
	owned := map[int]bool{command.Process.Pid: true}
	t.Cleanup(func() {
		discoverOwnedProcesses(owned, cliProcesses(t))
		for pid := range owned {
			if err := syscall.Kill(pid, syscall.SIGKILL); err != nil && !errors.Is(err, syscall.ESRCH) {
				t.Errorf("cleanup process %d: %v", pid, err)
			}
		}
	})
	pollCLI(t, func() bool {
		for _, process := range cliProcesses(t) {
			if process.parent == command.Process.Pid && process.arguments == cliBinary+" __coordinator" {
				coordinator = process
				owned[process.pid] = true
				return true
			}
		}
		return false
	})
	if coordinator.group != coordinator.pid {
		t.Fatalf("coordinator lacks its own process group: %#v", coordinator)
	}
	var path string
	pollCLI(t, func() bool {
		paths, err := filepath.Glob(filepath.Join(root, "v1", "campaign-*", ".prepared", "plan.json"))
		if err != nil {
			t.Fatal(err)
		}
		if len(paths) != 1 {
			return false
		}
		path = filepath.Dir(filepath.Dir(paths[0]))
		partials, err := filepath.Glob(filepath.Join(path, ".partial", "[0-9]*"))
		if err != nil {
			t.Fatal(err)
		}
		return journalCount(t, path) >= completed && len(partials) != 0
	})
	if err := syscall.Kill(coordinator.pid, syscall.SIGSTOP); err != nil {
		t.Fatal(err)
	}
	pollCLI(t, func() bool {
		for _, process := range cliProcesses(t) {
			if process.pid == coordinator.pid {
				return strings.Contains(process.state, "T")
			}
		}
		return false
	})
	if count := journalCount(t, path); count != completed {
		t.Fatalf("stopped journal has %d completed executions, want %d", count, completed)
	}
	discoverOwnedProcesses(owned, cliProcesses(t))
	if err := syscall.Kill(-coordinator.group, syscall.SIGKILL); err != nil {
		t.Fatal(err)
	}
	for pid := range owned {
		if pid != command.Process.Pid && pid != coordinator.pid {
			if err := syscall.Kill(pid, syscall.SIGKILL); err != nil && !errors.Is(err, syscall.ESRCH) {
				t.Fatal(err)
			}
		}
	}
	select {
	case err := <-waited:
		var exited *exec.ExitError
		if !errors.As(err, &exited) || exited.ExitCode() != 3 {
			t.Fatalf("killed coordinator CLI exit = %v\n%s", err, output.Bytes())
		}
	case <-ctx.Done():
		t.Fatalf("killed coordinator CLI did not exit: %v", ctx.Err())
	}
	delete(owned, command.Process.Pid)
	pollCLI(t, func() bool {
		for _, process := range cliProcesses(t) {
			if owned[process.pid] {
				return false
			}
		}
		return true
	})
	clear(owned)
	t.Logf("killed coordinator CLI output: %s", output.Bytes())
	t.Logf("SIGKILL coordinator pid=%d pgid=%d after %d journaled executions; owned descendants exited", coordinator.pid, coordinator.group, completed)
	return path
}

func campaignSemantics(campaign runner.CampaignInspection) runner.CampaignInspection {
	campaign.CampaignID = ""
	journal := *campaign.Journal
	journal.IndexSHA256, journal.Segments, journal.Bytes = "", 0, 0
	campaign.Journal = &journal
	// Stored byte counts include variable-length host timestamps and deadlines; inspect validates each store.
	campaign.RetainedSuccessBytes = 0
	campaign.FailureArtifacts = nil
	campaign.SuccessArtifacts = nil
	campaign.Executions = nil
	return campaign
}

func compareCampaigns(t *testing.T, baselinePath string, baseline *runner.CampaignInspection, resumedPath string, resumed *runner.CampaignInspection) {
	t.Helper()
	if resumed == nil {
		t.Fatal("resume produced no campaign inspection")
	}
	if baseline.Journal == nil || resumed.Journal == nil {
		t.Fatal("campaign inspection omitted its execution journal")
	}
	left, right := campaignSemantics(*baseline), campaignSemantics(*resumed)
	if !reflect.DeepEqual(left, right) {
		t.Fatalf("campaign semantics differ:\n baseline %#v\n resumed %#v", left, right)
	}
	if len(baseline.Executions) != len(resumed.Executions) {
		t.Fatalf("execution count differs: %d / %d", len(baseline.Executions), len(resumed.Executions))
	}
	for index, execution := range baseline.Executions {
		other := resumed.Executions[index]
		firstPath, secondPath := execution.Artifact, other.Artifact
		if execution.SuccessArtifact != nil {
			firstPath = execution.SuccessArtifact
		}
		if other.SuccessArtifact != nil {
			secondPath = other.SuccessArtifact
		}
		execution.ElapsedNanos, other.ElapsedNanos = 0, 0
		execution.Artifact, other.Artifact = nil, nil
		execution.SuccessArtifact, other.SuccessArtifact = nil, nil
		execution.SuccessArtifactBytes, other.SuccessArtifactBytes = nil, nil
		if !reflect.DeepEqual(execution, other) {
			t.Fatalf("execution %d differs:\n baseline %#v\n resumed %#v", index, execution, other)
		}
		if firstPath == nil || secondPath == nil {
			t.Fatalf("execution %d has no retained evidence", index)
		}
		first := readCLIRecord(t, filepath.Join(baselinePath, *firstPath))
		second := readCLIRecord(t, filepath.Join(resumedPath, *secondPath))
		first.CampaignID, second.CampaignID = "", ""
		first.CreatedAt, second.CreatedAt = "", ""
		first.Host, second.Host = record.Host{}, record.Host{}
		// The coordinator records the remaining host deadline, which also enters the record hash.
		first.Limits.OverallTimeoutNanos, second.Limits.OverallTimeoutNanos = 0, 0
		first.RecordHash, second.RecordHash = "", ""
		if !reflect.DeepEqual(first, second) {
			t.Fatalf("execution %d manifest evidence differs:\n baseline %#v\n resumed %#v", index, first, second)
		}
		t.Logf("ordinal=%d seed=%d outcome=%s/%s stdout=%s transcript=%s world=%s", execution.SelectionOrdinal, execution.Seed, execution.Domain, execution.Reason, first.Streams.Stdout.FullSHA256, first.IOProfile.Transcript.SHA256, first.World.Final.SemanticDigest)
	}
}

func readCLIRecord(t *testing.T, path string) record.ExecutionRecord {
	t.Helper()
	inspectCLI(t, path)
	data, err := os.ReadFile(filepath.Join(path, "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := record.DecodeExecutionRecord(data)
	if err != nil {
		t.Fatal(err)
	}
	if manifest.IOProfile.Transcript == nil || manifest.IOProfile.Transcript.Records == 0 || manifest.World.Final.SemanticDigest == "" {
		t.Fatalf("missing transcript or World evidence: %#v", manifest)
	}
	return manifest
}
