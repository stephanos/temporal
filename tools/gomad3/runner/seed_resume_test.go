package runner

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	backend "go.temporal.io/server/tools/gomad3/backends"
	"go.temporal.io/server/tools/gomad3/record"
	"go.temporal.io/server/tools/gomad3/runner/internal/campaign"
	"go.temporal.io/server/tools/gomad3/runner/internal/logstore"
)

// seedStep scripts one seed's attempt.
type seedStep struct {
	kind  string // success, failure, watchdog, block (until cancelled), or finish (unless cancelled first)
	probe string
	delay time.Duration
}

// scriptedSeedAttempts runs each seed's step and records which seeds ran.
type scriptedSeedAttempts struct {
	t      *testing.T
	script map[uint64]seedStep
	mu     sync.Mutex
	ran    []uint64
}

func (executor *scriptedSeedAttempts) Run(ctx context.Context, request backend.ExecutionRequest) (backend.ExecutionResult, error) {
	seed := seedFromEnvironment(request.Env)
	executor.mu.Lock()
	executor.ran = append(executor.ran, seed)
	executor.mu.Unlock()
	step := executor.script[seed]
	cancelled := func() backend.ExecutionResult {
		result := processResult(0, "", "")
		result.Cancelled, result.Termination, result.Signal = true, backend.TerminationSignal, "killed"
		return result
	}
	if step.delay != 0 {
		<-time.After(step.delay)
	}
	switch step.kind {
	case "block":
		<-ctx.Done()
		return cancelled(), nil
	case "finish":
		select {
		case <-ctx.Done():
			return cancelled(), nil
		case <-time.After(20 * time.Millisecond):
		}
	case "failure":
		result := processResult(1, step.probe, "")
		result.IOTranscript = completeEmptyTranscript()
		return result, nil
	case "watchdog":
		result := processResult(0, "", "")
		result.Termination, result.Signal, result.WatchdogTimeout = backend.TerminationSignal, "SIGKILL", true
		result.IOTranscript = completeEmptyTranscript()
		return result, nil
	}
	result := processResult(0, "", "")
	result.IOTranscript = semanticTranscript(executor.t, step.probe)
	return result, nil
}

func (executor *scriptedSeedAttempts) seeds() []uint64 {
	executor.mu.Lock()
	defer executor.mu.Unlock()
	return slices.Clone(executor.ran)
}

// seedScenario is one scripted campaign whose logical outcome must not
// depend on where it is interrupted.
type seedScenario struct {
	name     string
	policy   FailurePolicy
	budget   uint64
	parallel int
	script   []seedStep
	want     []string
	counts   string
}

func seedScenarios() []seedScenario {
	return []seedScenario{
		{
			name: "all policy keeps every classification", policy: PolicyAll, parallel: 3,
			script: []seedStep{{kind: "success", probe: probeA}, {kind: "failure", probe: "x"}, {kind: "watchdog"}, {kind: "success", probe: probeB}, {kind: "failure", probe: "x"}, {kind: "success", probe: probeA}},
			want: []string{
				"0 success/success/exit kept novel=" + probeA, "1 target/nonzero_exit/exit failure=1", "2 watchdog/watchdog_timeout/timeout failure=2",
				"3 success/success/exit kept novel=" + probeB, "4 target/nonzero_exit/exit failure=1", "5 success/success/exit",
			},
			counts: "attempted=6 succeeded=3 failures=3 watchdogs=1 cancelled=0 distinct=2 retained=2 stop=seeds_exhausted",
		},
		{
			// A first failure at parallel three keeps its one real failure and
			// cancels the two attempts admitted beside it.
			name: "first failure cancels its admitted attempts", policy: PolicyFirst, parallel: 3,
			script: []seedStep{{kind: "success", probe: probeA}, {kind: "failure", probe: "x"}, {kind: "block"}, {kind: "block"}, {kind: "block"}, {kind: "block"}},
			want: []string{
				"0 success/success/exit kept novel=" + probeA, "1 target/nonzero_exit/exit failure=1",
				"2 runner/runner_cancelled/none", "3 runner/runner_cancelled/none",
			},
			counts: "attempted=4 succeeded=1 failures=1 watchdogs=0 cancelled=2 distinct=1 retained=1 stop=first_failure",
		},
		{
			// The later attempts finish before the lower failure that stops the
			// campaign; their real success and failure join its stop group.
			name: "first failure keeps buffered real outcomes", policy: PolicyFirst, parallel: 3,
			script: []seedStep{{kind: "failure", probe: "x", delay: 30 * time.Millisecond}, {kind: "success", probe: probeA}, {kind: "failure", probe: "y"}, {kind: "block"}, {kind: "block"}},
			want: []string{
				"0 target/nonzero_exit/exit failure=0", "1 success/success/exit kept novel=" + probeA, "2 target/nonzero_exit/exit failure=2",
			},
			counts: "attempted=3 succeeded=1 failures=2 watchdogs=0 cancelled=0 distinct=2 retained=1 stop=first_failure",
		},
		{
			// A failure budget lets the attempt admitted beside its trigger
			// finish; nothing is cancelled by the policy.
			name: "failure budget drains naturally", policy: PolicyBudget, budget: 2, parallel: 2,
			script: []seedStep{{kind: "failure", probe: "x"}, {kind: "success", probe: probeA}, {kind: "failure", probe: "x"}, {kind: "failure", probe: "y"}, {kind: "finish", probe: probeB}, {kind: "success", probe: probeC}},
			want: []string{
				"0 target/nonzero_exit/exit failure=0", "1 success/success/exit kept novel=" + probeA, "2 target/nonzero_exit/exit failure=0",
				"3 target/nonzero_exit/exit failure=3", "4 success/success/exit kept novel=" + probeB,
			},
			counts: "attempted=5 succeeded=2 failures=3 watchdogs=0 cancelled=0 distinct=2 retained=2 stop=failure_budget",
		},
	}
}

func (scenario seedScenario) config(t *testing.T) (CampaignSpec, *scriptedSeedAttempts) {
	t.Helper()
	executor := &scriptedSeedAttempts{t: t, script: map[uint64]seedStep{}}
	for index, step := range scenario.script {
		executor.script[uint64(index+1)] = step
	}
	config, _ := testConfig(t, newFakePreparer(t), executor, fmt.Sprintf("1-%d", len(scenario.script)), scenario.policy, scenario.parallel)
	config.Coverage = CoverageSemantic
	keepSuccesses(&config, KeepSuccessesNovel)
	if scenario.budget != 0 {
		config.FailureBudget = scenario.budget
	}
	return config, executor
}

func (scenario seedScenario) resumeExecutor(t *testing.T) *scriptedSeedAttempts {
	_, executor := scenario.config(t)
	return executor
}

// observeLogicalSeedCampaign projects a published seed campaign onto what an
// interruption must not change: each ordinal's classification, retention and
// novelty, which earlier ordinal first had its failure signature, and the
// published counters.
func observeLogicalSeedCampaign(t *testing.T, path string) ([]string, string) {
	t.Helper()
	batch, err := campaign.OpenCampaign(path)
	if err != nil {
		t.Fatal(err)
	}
	runs := slices.Clone(batch.Executions)
	sort.Slice(runs, func(i, j int) bool { return runs[i].SelectionOrdinal < runs[j].SelectionOrdinal })
	first := map[record.SHA256]uint64{}
	var observed []string
	for _, run := range runs {
		line := fmt.Sprintf("%d %s/%s/%s", run.SelectionOrdinal, run.Domain, run.Reason, run.Termination)
		if run.SuccessArtifact != nil {
			line += " kept"
		}
		if len(run.NovelSemanticProbes) != 0 {
			line += " novel=" + strings.Join(run.NovelSemanticProbes, ",")
		}
		if run.FailureSignature != nil {
			if _, found := first[*run.FailureSignature]; !found {
				first[*run.FailureSignature] = uint64(run.SelectionOrdinal)
			}
			line += fmt.Sprintf(" failure=%d", first[*run.FailureSignature])
		}
		observed = append(observed, line)
	}
	summary := batch.Record
	counts := fmt.Sprintf("attempted=%d succeeded=%d failures=%d watchdogs=%d cancelled=%d distinct=%d retained=%d stop=%s",
		summary.Attempted, summary.Succeeded, summary.Failures, summary.Watchdogs, summary.Cancelled, summary.DistinctFailures, summary.RetainedSuccesses, summary.StopReason)
	return observed, counts
}

// campaignLogKinds reads the record kinds of a campaign log in order.
func campaignLogKinds(t *testing.T, path string) []logstore.Kind {
	t.Helper()
	kinds := map[logstore.Kind]uint32{}
	for kind := logstore.Kind(1); kind <= 17; kind++ {
		kinds[kind] = logstore.MaxPayloadBytes
	}
	var observed []logstore.Kind
	if _, err := logstore.Replay(filepath.Join(path, "store"), logstore.Options{Kinds: kinds}, func(entry logstore.Record) error {
		observed = append(observed, entry.Kind)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	return observed
}

const campaignAdmissionKind logstore.Kind = 6

// appendFault fails the index-th campaign log append (counted from one) at
// one of its boundaries: before the write, midway through it, after the
// complete frame before its sync, or after the sync before the append
// returns. "cancel" instead cancels the campaign's context once that append
// is durable.
type appendFault struct {
	index  int
	mode   string
	cancel context.CancelFunc

	mu       sync.Mutex
	appends  int
	injected bool
	late     int
}

var errInjectedAppend = errors.New("injected campaign log append failure")

func (fault *appendFault) hook(kind, operation string) error {
	if operation != "append" {
		return nil
	}
	fault.mu.Lock()
	defer fault.mu.Unlock()
	if kind == "log-write" {
		if fault.injected && fault.mode != "cancel" {
			fault.late++
		}
		fault.appends++
	}
	if fault.appends != fault.index || fault.injected {
		return nil
	}
	switch {
	case kind == "log-write" && fault.mode == "write":
		fault.injected = true
		return errInjectedAppend
	case kind == "log-write" && fault.mode == "partial":
		fault.injected = true
		return &logstore.PartialWrite{Bytes: 9}
	case kind == "log-sync" && fault.mode == "sync":
		fault.injected = true
		return errInjectedAppend
	case kind == "log-synced" && fault.mode == "synced":
		fault.injected = true
		return errInjectedAppend
	case kind == "log-synced" && fault.mode == "cancel":
		fault.injected = true
		fault.cancel()
	}
	return nil
}

// durableOrdinals are the ordinals a resumed campaign must not run again:
// those with a committed outcome other than host failure evidence, and those
// with a durable receipt.
func durableOrdinals(t *testing.T, path string) (map[uint64]struct{}, campaign.CommittedState) {
	t.Helper()
	state, err := campaign.ReadCommittedState(path)
	if err != nil {
		t.Fatal(err)
	}
	durable := map[uint64]struct{}{}
	for _, run := range state.Executions {
		if run.Domain != "runner" || run.Reason == "runner_cancelled" {
			durable[uint64(run.SelectionOrdinal)] = struct{}{}
		}
	}
	for ordinal := range state.Frontier.Receipts {
		durable[ordinal] = struct{}{}
	}
	return durable, state
}

// classifiedOutcomes counts the durable successes and failures. A host
// failure's cancelled attempts are counted as cancelled without committing a
// classified outcome, so cancellations are not compared.
func classifiedOutcomes(state campaign.CommittedState) uint64 {
	var classified uint64
	for _, run := range state.Executions {
		if run.Domain != "runner" {
			classified++
		}
	}
	return classified
}

// resumeSeedCampaign resumes an interrupted campaign until it publishes. No
// resumed attempt reruns an ordinal whose outcome or receipt was durable,
// and a committed stop launches nothing.
func resumeSeedCampaign(t *testing.T, scenario seedScenario, config CampaignSpec, path string) {
	t.Helper()
	for range 3 {
		durable, state := durableOrdinals(t, path)
		if state.Lifecycle.Published {
			return
		}
		executor := scenario.resumeExecutor(t)
		_, err := exploreWith(context.Background(), CampaignSpec{ResumeCampaign: path, RunnerBuild: config.RunnerBuild, Backend: scriptProvider()}, executionDependencies{executor: executor})
		ran := executor.seeds()
		for _, seed := range ran {
			if _, found := durable[seed-1]; found {
				t.Fatalf("resume reran ordinal %d whose outcome or receipt was durable (ran %v)", seed-1, ran)
			}
		}
		if state.Frontier.Stopped != "" && len(ran) != 0 {
			t.Fatalf("resume after a committed %s stop ran %v", state.Frontier.Stopped, ran)
		}
		if err == nil {
			return
		}
		t.Logf("resume failed: %v", err)
	}
	t.Fatal("campaign did not publish after three resumes")
}

// Every seed campaign log append from the first admission through
// publication fails in turn at each of its boundaries, or has the host
// cancel the campaign once it is durable. Each interrupted campaign resumes
// to the uninterrupted logical outcome, and a failed append neither advances
// the reported committed progress nor lets another append through.
func TestSeedCampaignResumesEveryAppendFaultToTheUninterruptedOutcome(t *testing.T) {
	for _, scenario := range seedScenarios() {
		t.Run(scenario.name, func(t *testing.T) {
			t.Parallel()
			config, executor := scenario.config(t)
			reference, err := exploreWith(context.Background(), config, executionDependencies{executor: executor})
			if err != nil {
				t.Fatal(err)
			}
			want, wantCounts := observeLogicalSeedCampaign(t, reference.CampaignPath)
			if !reflect.DeepEqual(want, scenario.want) || wantCounts != scenario.counts {
				t.Fatalf("uninterrupted campaign = %q %s, want %q %s", want, wantCounts, scenario.want, scenario.counts)
			}
			kinds := campaignLogKinds(t, reference.CampaignPath)
			first := slices.Index(kinds, campaignAdmissionKind) + 1
			for index := first; index <= len(kinds); index++ {
				for _, mode := range []string{"write", "partial", "sync", "synced", "cancel"} {
					t.Run(fmt.Sprintf("append %d %s", index, mode), func(t *testing.T) {
						config, executor := scenario.config(t)
						ctx, cancel := context.WithCancel(context.Background())
						defer cancel()
						fault := &appendFault{index: index, mode: mode, cancel: cancel}
						config.TerminateGrace = 10 * time.Millisecond
						interrupted, err := exploreWith(campaign.WithStoreFaults(ctx, fault.hook), config, executionDependencies{executor: executor})
						if !fault.injected {
							t.Fatalf("append %d was not reached (%d appends)", index, fault.appends)
						}
						if err == nil {
							t.Fatal("interrupted campaign succeeded")
						}
						if fault.late != 0 {
							t.Fatalf("%d appends reached the log after the failed append", fault.late)
						}
						_, state := durableOrdinals(t, interrupted.CampaignPath)
						reported := interrupted.Succeeded + interrupted.Failures
						durable := classifiedOutcomes(state)
						if reported > durable || (mode == "write" || mode == "partial") && reported != durable {
							t.Fatalf("reported %d classified outcomes; %d are durable", reported, durable)
						}
						resumeSeedCampaign(t, scenario, config, interrupted.CampaignPath)
						got, gotCounts := observeLogicalSeedCampaign(t, interrupted.CampaignPath)
						if !reflect.DeepEqual(got, want) || gotCounts != wantCounts {
							t.Fatalf("resumed campaign = %q %s, want %q %s", got, gotCounts, want, wantCounts)
						}
					})
				}
			}
		})
	}
}

// A resumed campaign whose first resume is interrupted too still converges:
// each resume reuses what the previous ones made durable.
func TestSeedCampaignRepeatedResumeConverges(t *testing.T) {
	scenario := seedScenarios()[1]
	config, executor := scenario.config(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	fault := &appendFault{index: 7, mode: "sync", cancel: cancel}
	interrupted, err := exploreWith(campaign.WithStoreFaults(ctx, fault.hook), config, executionDependencies{executor: executor})
	if err == nil || !fault.injected {
		t.Fatalf("interrupted campaign error = %v, injected = %t", err, fault.injected)
	}
	for _, index := range []int{2, 3} {
		resumeFault := &appendFault{index: index, mode: "partial"}
		_, err := exploreWith(campaign.WithStoreFaults(context.Background(), resumeFault.hook), CampaignSpec{ResumeCampaign: interrupted.CampaignPath, RunnerBuild: config.RunnerBuild, Backend: scriptProvider()}, executionDependencies{executor: scenario.resumeExecutor(t)})
		if err == nil && resumeFault.injected {
			t.Fatalf("resume interrupted at append %d succeeded", index)
		}
	}
	resumeSeedCampaign(t, scenario, config, interrupted.CampaignPath)
	got, counts := observeLogicalSeedCampaign(t, interrupted.CampaignPath)
	if !reflect.DeepEqual(got, scenario.want) || counts != scenario.counts {
		t.Fatalf("resumed campaign = %q %s, want %q %s", got, counts, scenario.want, scenario.counts)
	}
}

// A durable receipt whose published payload is missing is corruption: resume
// fails before it reclaims anything and never runs the attempt again.
func TestSeedCampaignResumeRefusesAMissingReceiptPayload(t *testing.T) {
	scenario := seedScenarios()[2]
	config, executor := scenario.config(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var received campaign.Receipt
	var path string
	fault := &appendFault{mode: "none"}
	hook := func(kind, operation string) error {
		if kind != "log-synced" || operation != "append" || path == "" {
			return fault.hook(kind, operation)
		}
		state, err := campaign.ReadCommittedState(path)
		if err == nil {
			for _, receipt := range state.Frontier.Receipts {
				if receipt.Observation.Artifact != nil {
					received = receipt
					cancel()
				}
			}
		}
		return nil
	}
	config.Progress = func(event CampaignEvent) error {
		if path == "" {
			path = event.CampaignPath
		}
		return nil
	}
	interrupted, err := exploreWith(campaign.WithStoreFaults(ctx, hook), config, executionDependencies{executor: executor})
	if err == nil || received.Observation.Artifact == nil {
		t.Fatalf("campaign was not interrupted after a failure receipt: %v", err)
	}
	missing := filepath.Join(interrupted.CampaignPath, filepath.FromSlash(*received.Observation.Artifact))
	if err := os.RemoveAll(missing); err != nil {
		t.Fatal(err)
	}
	before := ownedPayloadNames(t, interrupted.CampaignPath)
	resumed := scenario.resumeExecutor(t)
	_, err = exploreWith(context.Background(), CampaignSpec{ResumeCampaign: interrupted.CampaignPath, RunnerBuild: config.RunnerBuild, Backend: scriptProvider()}, executionDependencies{executor: resumed})
	if err == nil || !campaign.IsIntegrityError(err) {
		t.Fatalf("resume with a missing receipt payload error = %v", err)
	}
	if ran := resumed.seeds(); len(ran) != 0 {
		t.Fatalf("resume ran %v", ran)
	}
	if after := ownedPayloadNames(t, interrupted.CampaignPath); !reflect.DeepEqual(after, before) {
		t.Fatalf("resume reclaimed payloads before failing: %v -> %v", before, after)
	}
}

func ownedPayloadNames(t *testing.T, path string) []string {
	t.Helper()
	var names []string
	for _, root := range []string{"failures", "successes", "diagnostics"} {
		entries, err := os.ReadDir(filepath.Join(path, root))
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			t.Fatal(err)
		}
		for _, entry := range entries {
			names = append(names, root+"/"+entry.Name())
		}
	}
	return names
}
