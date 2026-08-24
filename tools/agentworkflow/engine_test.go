package agentworkflow

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tools/agentworkflow/internal/workspace"
)

func TestRunQualifiesIsolatedCandidateAndAppliesExplicitly(t *testing.T) {
	projectRoot := t.TempDir()
	writeTestFile(t, filepath.Join(projectRoot, "original.txt"), "untouched")
	backend := &fakeBackend{name: "fake-one", implementation: "good"}
	engine := newTestEngine(t, backend, nil)

	result, err := engine.Run(context.Background(), testRequest(projectRoot, true))
	if err != nil {
		t.Fatal(err)
	}
	if result.Outcome != OutcomeSucceeded || result.Backend.Name != "fake-one" || len(result.Reviews) != 2 {
		t.Fatalf("result = %#v", result)
	}
	if _, err := os.Stat(filepath.Join(projectRoot, "result.txt")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("original workspace changed before apply: %v", err)
	}
	candidateDirectory := filepath.Join(engine.store.Root(), string(result.RunID), "workspaces", "candidate")
	if data, err := os.ReadFile(filepath.Join(candidateDirectory, "result.txt")); err != nil || string(data) != "good" {
		t.Fatalf("candidate result = %q, %v", data, err)
	}
	changes, err := engine.Diff(context.Background(), result.RunID)
	if err != nil {
		t.Fatal(err)
	}
	if !containsChange(changes, "result.txt", "added") {
		t.Fatalf("changes = %#v", changes)
	}
	if err := engine.Apply(context.Background(), result.RunID); err != nil {
		t.Fatal(err)
	}
	if data, err := os.ReadFile(filepath.Join(projectRoot, "result.txt")); err != nil || string(data) != "good" {
		t.Fatalf("applied result = %q, %v", data, err)
	}
}

func TestRunRepairsFailedDirectCheckAndRerunsEvidence(t *testing.T) {
	projectRoot := t.TempDir()
	backend := &fakeBackend{name: "fake", implementation: "bad", repairContent: "good"}
	engine := newTestEngine(t, backend, nil)

	result, err := engine.Run(context.Background(), testRequest(projectRoot, true))
	if err != nil {
		t.Fatal(err)
	}
	if result.Outcome != OutcomeSucceeded || result.Repairs != 1 {
		t.Fatalf("result = %#v", result)
	}
	if len(result.Checks) != 1 || result.Checks[0].Outcome != "passed" {
		t.Fatalf("checks = %#v", result.Checks)
	}
	if backend.callCount("repair-1") != 1 {
		t.Fatalf("repair calls = %d", backend.callCount("repair-1"))
	}
}

func TestRunUsesFreshRepairSessionForNonResumableBackend(t *testing.T) {
	backend := &fakeBackend{name: "fake", implementation: "bad", repairContent: "good", noResume: true}
	result, err := newTestEngine(t, backend, nil).Run(context.Background(), testRequest(t.TempDir(), true))
	if err != nil || result.Outcome != OutcomeSucceeded {
		t.Fatalf("result=%#v error=%v", result, err)
	}
	invocation, found := backend.invocation("repair-1")
	if !found || invocation.Session != "" || invocation.RetainSession {
		t.Fatalf("fresh repair invocation=%#v found=%v", invocation, found)
	}
}

func TestRunRepairsBlockingReviewFinding(t *testing.T) {
	projectRoot := t.TempDir()
	backend := &fakeBackend{name: "fake", implementation: "bad", repairContent: "good", findingUntilRepair: true}
	engine := newTestEngine(t, backend, nil)
	request := testRequest(projectRoot, false)
	request.Project.Checks = []Check{{
		Name: "exists", Command: checkCommand("result.txt", "exists"), Directory: ".", Required: true,
	}}

	result, err := engine.Run(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if result.Outcome != OutcomeSucceeded || result.Repairs != 1 || len(result.Reviews) != 2 {
		t.Fatalf("result = %#v", result)
	}
	if len(result.Reviews[0].Findings) != 0 {
		t.Fatalf("final review retained repaired finding: %#v", result.Reviews)
	}
	if len(result.Findings) != 1 || !slicesContainsDisposition(result.Findings[0].Dispositions, FindingConfirmed) || !slicesContainsDisposition(result.Findings[0].Dispositions, FindingRepaired) {
		t.Fatalf("finding ledger = %#v", result.Findings)
	}
}

func TestRunRetainsUnresolvedFindingDisposition(t *testing.T) {
	backend := &fakeBackend{name: "fake", implementation: "bad", findingUntilRepair: true}
	request := testRequest(t.TempDir(), false)
	request.Policy.MaxRepairs = 0
	request.Project.Checks = []Check{{Name: "exists", Command: checkCommand("result.txt", "exists"), Directory: ".", Required: true}}
	result, err := newTestEngine(t, backend, nil).Run(context.Background(), request)
	if err != nil || result.Outcome != OutcomeNeedsChanges {
		t.Fatalf("result=%#v error=%v", result, err)
	}
	if len(result.Findings) != 1 || !slicesContainsDisposition(result.Findings[0].Dispositions, FindingConfirmed) || !slicesContainsDisposition(result.Findings[0].Dispositions, FindingUnresolved) {
		t.Fatalf("finding ledger = %#v", result.Findings)
	}
}

func TestRunIsInconclusiveWithoutDirectChecks(t *testing.T) {
	projectRoot := t.TempDir()
	engine := newTestEngine(t, &fakeBackend{name: "fake", implementation: "good"}, nil)
	request := testRequest(projectRoot, false)
	request.Project.Checks = nil

	result, err := engine.Run(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if result.Outcome != OutcomeInconclusive {
		t.Fatalf("outcome = %s", result.Outcome)
	}
}

func TestRunRejectsMutationFromOptionalCheck(t *testing.T) {
	request := testRequest(t.TempDir(), false)
	request.Policy.MaxRepairs = 0
	request.Project.Checks = []Check{{
		Name: "generator", Command: checkCommand("generated-by-check.txt", "mutate"), Directory: ".", Required: false,
	}}
	result, err := newTestEngine(t, &fakeBackend{name: "fake", implementation: "good"}, nil).Run(context.Background(), request)
	if err != nil || result.Outcome != OutcomeProjectFailed || len(result.Checks) != 1 || result.Checks[0].Outcome != "mutated" {
		t.Fatalf("result=%#v error=%v", result, err)
	}
}

func TestRunHonorsDisabledWorkflowStages(t *testing.T) {
	t.Run("discover", func(t *testing.T) {
		backend := &fakeBackend{name: "fake", implementation: "good"}
		request := testRequest(t.TempDir(), true)
		disableStage(&request, StageDiscover)
		result, err := newTestEngine(t, backend, nil).Run(context.Background(), request)
		if err != nil || result.Outcome != OutcomeSucceeded || backend.callCount("discover") != 0 {
			t.Fatalf("result=%#v discover calls=%d error=%v", result, backend.callCount("discover"), err)
		}
	})

	t.Run("plan", func(t *testing.T) {
		backend := &fakeBackend{name: "fake", implementation: "good"}
		request := testRequest(t.TempDir(), true)
		request.Policy.Assurance = AssuranceHigh
		disableStage(&request, StagePlan)
		result, err := newTestEngine(t, backend, nil).Run(context.Background(), request)
		if err != nil || result.Outcome != OutcomeSucceeded || backend.callCount("plan") != 0 || backend.callCount("plan-review") != 0 {
			t.Fatalf("result=%#v plan calls=%d review calls=%d error=%v", result, backend.callCount("plan"), backend.callCount("plan-review"), err)
		}
	})

	t.Run("implement", func(t *testing.T) {
		backend := &fakeBackend{name: "fake", implementation: "good"}
		request := testRequest(t.TempDir(), true)
		disableStage(&request, StageImplement)
		result, err := newTestEngine(t, backend, nil).Run(context.Background(), request)
		if err != nil || result.Outcome != OutcomeInconclusive || backend.callCount("implement") != 0 {
			t.Fatalf("result=%#v implement calls=%d error=%v", result, backend.callCount("implement"), err)
		}
	})

	t.Run("check", func(t *testing.T) {
		backend := &fakeBackend{name: "fake", implementation: "good"}
		request := testRequest(t.TempDir(), false)
		request.Project.Checks = []Check{{Name: "must-not-run", Command: []string{"missing-agentworkflow-check"}, Required: true}}
		disableStage(&request, StageCheck)
		result, err := newTestEngine(t, backend, nil).Run(context.Background(), request)
		if err != nil || result.Outcome != OutcomeInconclusive || len(result.Checks) != 0 {
			t.Fatalf("result=%#v error=%v", result, err)
		}
	})

	t.Run("review", func(t *testing.T) {
		backend := &fakeBackend{name: "fake", implementation: "good"}
		request := testRequest(t.TempDir(), true)
		disableStage(&request, StageReview)
		result, err := newTestEngine(t, backend, nil).Run(context.Background(), request)
		if err != nil || result.Outcome != OutcomeInconclusive || backend.callCount("review-0-correctness") != 0 {
			t.Fatalf("result=%#v review calls=%d error=%v", result, backend.callCount("review-0-correctness"), err)
		}
	})

	t.Run("repair", func(t *testing.T) {
		backend := &fakeBackend{name: "fake", implementation: "bad", repairContent: "good"}
		request := testRequest(t.TempDir(), true)
		disableStage(&request, StageRepair)
		result, err := newTestEngine(t, backend, nil).Run(context.Background(), request)
		if err != nil || result.Outcome != OutcomeProjectFailed || backend.callCount("repair-1") != 0 {
			t.Fatalf("result=%#v repair calls=%d error=%v", result, backend.callCount("repair-1"), err)
		}
	})

	t.Run("apply", func(t *testing.T) {
		backend := &fakeBackend{name: "fake", implementation: "good"}
		request := testRequest(t.TempDir(), true)
		disableStage(&request, StageApply)
		engine := newTestEngine(t, backend, nil)
		result, err := engine.Run(context.Background(), request)
		if err != nil || result.Outcome != OutcomeSucceeded {
			t.Fatalf("result=%#v error=%v", result, err)
		}
		if err := engine.Apply(context.Background(), result.RunID); !errors.Is(err, ErrUnsupported) {
			t.Fatalf("Apply() error = %v, want unsupported", err)
		}
	})

	t.Run("apply remains gated for an inconclusive run", func(t *testing.T) {
		backend := &fakeBackend{name: "fake", implementation: "good"}
		request := testRequest(t.TempDir(), true)
		disableStage(&request, StageApply)
		for index := range request.Workflow.Stages {
			if request.Workflow.Stages[index].Kind == StageCheck {
				request.Workflow.Stages[index].Enabled = false
			}
		}
		engine := newTestEngine(t, backend, nil)
		result, err := engine.Run(context.Background(), request)
		if err != nil || result.Outcome != OutcomeInconclusive {
			t.Fatalf("result=%#v error=%v", result, err)
		}
		if err := engine.Apply(context.Background(), result.RunID); !errors.Is(err, ErrUnsupported) {
			t.Fatalf("Apply() error = %v, want unsupported", err)
		}
	})
}

func TestRunAlwaysProtectsAgentworkflowAndInstructions(t *testing.T) {
	for _, protected := range []string{".agentworkflow/task.md", "GUIDE.md"} {
		t.Run(protected, func(t *testing.T) {
			projectRoot := t.TempDir()
			writeTestFile(t, filepath.Join(projectRoot, filepath.FromSlash(protected)), "human input")
			backend := &fakeBackend{name: "fake", implementation: "agent output", implementationPath: protected}
			request := testRequest(projectRoot, false)
			if protected == "GUIDE.md" {
				request.Project.Instructions = []string{protected}
			}
			result, err := newTestEngine(t, backend, nil).Run(context.Background(), request)
			if err != nil || result.Outcome != OutcomeNeedsChanges || !strings.Contains(result.Message, "forbidden") {
				t.Fatalf("result=%#v error=%v", result, err)
			}
			data, readErr := os.ReadFile(filepath.Join(projectRoot, filepath.FromSlash(protected)))
			if readErr != nil || string(data) != "human input" {
				t.Fatalf("protected source = %q, %v", data, readErr)
			}
		})
	}

	backend := &fakeBackend{name: "fake", implementation: "good"}
	request := testRequest(t.TempDir(), true)
	request.Project.Source.Exclude = []string{".agentworkflow"}
	if _, err := newTestEngine(t, backend, nil).Run(context.Background(), request); err == nil || len(backend.calls) != 0 {
		t.Fatalf("excluding .agentworkflow error=%v calls=%#v", err, backend.calls)
	}
}

func TestRunUsesConfiguredStagePrompts(t *testing.T) {
	projectRoot := t.TempDir()
	backend := &fakeBackend{name: "fake", implementation: "bad", repairContent: "good", rejectFirstPlan: true}
	request := testRequest(projectRoot, true)
	request.Policy.Assurance = AssuranceHigh
	request.Workflow = DefaultWorkflow()
	markers := map[StageKind]string{
		StageDiscover: "CUSTOM DISCOVER", StageImplement: "CUSTOM IMPLEMENT",
		StageReview: "CUSTOM REVIEW", StageRepair: "CUSTOM REPAIR",
	}
	for index := range request.Workflow.Stages {
		stage := &request.Workflow.Stages[index]
		if marker := markers[stage.Kind]; marker != "" {
			stage.Prompt = marker
		}
		if stage.Kind == StagePlan {
			stage.Prompt = "CUSTOM PLAN"
			stage.ReviewPrompt = "CUSTOM PLAN REVIEW"
			stage.RevisionPrompt = "CUSTOM PLAN REVISION"
		}
	}
	result, err := newTestEngine(t, backend, nil).Run(context.Background(), request)
	if err != nil || result.Outcome != OutcomeSucceeded {
		t.Fatalf("result=%#v error=%v", result, err)
	}
	for phase, marker := range map[string]string{
		"discover": "CUSTOM DISCOVER", "plan": "CUSTOM PLAN", "plan-review": "CUSTOM PLAN REVIEW",
		"plan-revise": "CUSTOM PLAN REVISION", "implement": "CUSTOM IMPLEMENT",
		"review-1-correctness": "CUSTOM REVIEW", "repair-1": "CUSTOM REPAIR",
	} {
		invocation, found := backend.invocation(phase)
		if !found || !strings.Contains(invocation.Prompt, marker) {
			t.Fatalf("phase %s prompt = %q, found=%v, want marker %q", phase, invocation.Prompt, found, marker)
		}
	}
}

func TestRunUsesSelectedBackendModelForEveryLogicalStage(t *testing.T) {
	for _, provider := range []string{"codex", "claude"} {
		t.Run(provider, func(t *testing.T) {
			backend := &fakeBackend{name: provider, implementation: "bad", repairContent: "good", rejectFirstPlan: true}
			request := testRequest(t.TempDir(), true)
			request.Policy.Assurance = AssuranceHigh
			request.Policy.Reviewers = []string{"correctness", "tests"}
			request.Workflow = DefaultWorkflow()
			for index := range request.Workflow.Stages {
				stage := &request.Workflow.Stages[index]
				if stage.Kind == StageCheck || stage.Kind == StageApply {
					continue
				}
				chosen := provider + "-" + string(stage.Kind)
				other := map[string]string{"codex": "claude", "claude": "codex"}[provider]
				stage.Models = Models{provider: chosen, other: other + "-" + string(stage.Kind)}
			}

			result, err := newTestEngine(t, backend, nil).Run(context.Background(), request)
			require.NoError(t, err)
			require.Equal(t, OutcomeSucceeded, result.Outcome)
			for phase, want := range map[string]string{
				"discover": "discover", "plan": "plan", "plan-review": "plan", "plan-revise": "plan",
				"plan-review-2": "plan", "implement": "implement", "repair-1": "repair",
				"review-1-correctness": "review", "review-1-tests": "review",
			} {
				invocation, found := backend.invocation(phase)
				require.True(t, found, "missing invocation for %s", phase)
				require.Equal(t, provider+"-"+want, invocation.Model, "phase %s", phase)
			}
		})
	}
}

func TestNormalizeWorkflowCopiesStageModels(t *testing.T) {
	workflow := DefaultWorkflow()
	workflow.Stages[0].Models = Models{"codex": "discover-model"}

	normalized, err := normalizeWorkflow(workflow)
	require.NoError(t, err)
	workflow.Stages[0].Models["codex"] = "changed"

	require.Equal(t, "discover-model", normalized.Stages[0].Models["codex"])
}

func TestRunRejectsInvalidStructuredPlanAndRetainsFailure(t *testing.T) {
	projectRoot := t.TempDir()
	backend := &fakeBackend{name: "fake", implementation: "good", invalidPhase: "plan"}
	engine := newTestEngine(t, backend, nil)

	result, err := engine.Run(context.Background(), testRequest(projectRoot, true))
	if err != nil {
		t.Fatal(err)
	}
	if result.Outcome != OutcomeAgentFailed || result.Phase != "plan" {
		t.Fatalf("result = %#v", result)
	}
	status, err := engine.Inspect(context.Background(), result.RunID)
	if err != nil {
		t.Fatal(err)
	}
	if status.Result == nil || status.Result.Outcome != OutcomeAgentFailed {
		t.Fatalf("status = %#v", status)
	}
}

func TestRunRejectsBackendMutationDuringReadOnlyPhase(t *testing.T) {
	projectRoot := t.TempDir()
	backend := &fakeBackend{name: "fake", implementation: "good", mutateReadOnly: true}
	engine := newTestEngine(t, backend, nil)
	result, err := engine.Run(context.Background(), testRequest(projectRoot, true))
	if err != nil {
		t.Fatal(err)
	}
	if result.Outcome != OutcomeAgentFailed || !strings.Contains(result.Message, "mutated read-only") {
		t.Fatalf("result = %#v", result)
	}
	if _, err := os.Stat(filepath.Join(projectRoot, "intrusion")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("original source was mutated: %v", err)
	}
}

func TestRunRequiresNormalizedBackendTerminalLifecycle(t *testing.T) {
	projectRoot := t.TempDir()
	backend := &fakeBackend{name: "fake", implementation: "good", omitTerminal: true}
	engine := newTestEngine(t, backend, nil)
	result, err := engine.Run(context.Background(), testRequest(projectRoot, true))
	if err != nil {
		t.Fatal(err)
	}
	if result.Outcome != OutcomeAgentFailed || !strings.Contains(result.Message, "normalized lifecycle") {
		t.Fatalf("result = %#v", result)
	}
}

func TestRunPropagatesOversizedBackendEvidenceAsCapacity(t *testing.T) {
	projectRoot := t.TempDir()
	backend := &fakeBackend{name: "fake", implementation: "good", oversizedOutput: true}
	engine := newTestEngine(t, backend, nil)
	result, err := engine.Run(context.Background(), testRequest(projectRoot, true))
	if err != nil {
		t.Fatal(err)
	}
	if result.Outcome != OutcomeCapacityExhausted {
		t.Fatalf("result = %#v", result)
	}
}

func TestInspectDetectsCorruptAttemptEvidence(t *testing.T) {
	projectRoot := t.TempDir()
	engine := newTestEngine(t, &fakeBackend{name: "fake", implementation: "good"}, nil)
	result, err := engine.Run(context.Background(), testRequest(projectRoot, true))
	if err != nil {
		t.Fatal(err)
	}
	events, err := filepath.Glob(filepath.Join(engine.store.Root(), string(result.RunID), "attempts", "*", "events.jsonl"))
	if err != nil || len(events) == 0 {
		t.Fatalf("events = %v, %v", events, err)
	}
	file, err := os.OpenFile(events[0], os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.WriteString("{}\n"); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	status, err := engine.Inspect(context.Background(), result.RunID)
	if !errors.Is(err, ErrCorrupt) || status.Outcome != OutcomeCorrupt {
		t.Fatalf("status = %#v, error = %v", status, err)
	}
}

func TestInspectReadsLegacyV1StageRun(t *testing.T) {
	engine := newTestEngine(t, &fakeBackend{name: "fake", implementation: "good"}, nil)
	runDirectory := filepath.Join(engine.store.Root(), "legacy-run")
	stageDirectory := filepath.Join(runDirectory, "stages", "plan")
	writeTestFile(t, filepath.Join(stageDirectory, "events.jsonl"), "")
	writeTestFile(t, filepath.Join(stageDirectory, "stderr.log"), "")
	record := map[string]any{
		"schema": "agentworkflow.stage-result/v1", "run_id": "legacy-run", "stage": "plan", "status": "completed",
		"final_output": "plan", "event_count": 0,
		"stdout_sha256": legacyDigest("agentworkflow.stdout/v1", nil), "stderr_sha256": legacyDigest("agentworkflow.stderr/v1", nil),
		"stdout_bytes": 0, "stderr_bytes": 0, "run_directory": runDirectory, "stage_directory": stageDirectory,
	}
	encoded, _ := json.Marshal(record)
	if err := os.WriteFile(filepath.Join(stageDirectory, "stage.json"), encoded, 0o600); err != nil {
		t.Fatal(err)
	}
	status, err := engine.Inspect(context.Background(), "legacy-run")
	if err != nil || status.State != "legacy" || status.Phase != "legacy" || status.Outcome != OutcomeInconclusive {
		t.Fatalf("legacy status=%#v error=%v", status, err)
	}
}

func TestApplyRefusesSourceDrift(t *testing.T) {
	projectRoot := t.TempDir()
	writeTestFile(t, filepath.Join(projectRoot, "original.txt"), "base")
	engine := newTestEngine(t, &fakeBackend{name: "fake", implementation: "good"}, nil)
	result, err := engine.Run(context.Background(), testRequest(projectRoot, true))
	if err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, filepath.Join(projectRoot, "original.txt"), "changed")
	if err := engine.Apply(context.Background(), result.RunID); !errors.Is(err, ErrSourceDrift) {
		t.Fatalf("Apply() error = %v, want source drift", err)
	}
	if _, err := os.Stat(filepath.Join(projectRoot, "result.txt")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("drifted source was partially applied: %v", err)
	}
}

func TestBackendIdentityIsSwappableWithoutWorkflowChanges(t *testing.T) {
	for _, name := range []string{"codex-compatible", "claude-compatible"} {
		t.Run(name, func(t *testing.T) {
			projectRoot := t.TempDir()
			engine := newTestEngine(t, &fakeBackend{name: name, implementation: "good"}, nil)
			result, err := engine.Run(context.Background(), testRequest(projectRoot, true))
			if err != nil {
				t.Fatal(err)
			}
			if result.Outcome != OutcomeSucceeded || result.Backend.Name != name {
				t.Fatalf("result = %#v", result)
			}
		})
	}
}

func TestBackendIdentityIncludesResolvedConfiguration(t *testing.T) {
	left := BackendInfo{Name: "fake", Version: "v1", ConfigurationDigest: "sha256:one", Capabilities: []Capability{CapabilityReadOnly}}
	right := BackendInfo{Name: "fake", Version: "v1", ConfigurationDigest: "sha256:two", Capabilities: []Capability{CapabilityReadOnly}}
	if sameBackend(left, right) {
		t.Fatal("different resolved backend configurations had the same identity")
	}
}

func TestHighAssuranceRevisesAndRereviewsRejectedPlan(t *testing.T) {
	projectRoot := t.TempDir()
	backend := &fakeBackend{name: "fake", implementation: "good", rejectFirstPlan: true}
	engine := newTestEngine(t, backend, nil)
	request := testRequest(projectRoot, true)
	request.Policy.Assurance = AssuranceHigh
	request.Policy.Reviewers = []string{"correctness", "tests"}
	result, err := engine.Run(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if result.Outcome != OutcomeSucceeded || backend.callCount("plan-review") != 1 || backend.callCount("plan-revise") != 1 || backend.callCount("plan-review-2") != 1 {
		t.Fatalf("result = %#v, calls = %#v", result, backend.calls)
	}
}

func TestOpenAndRequestValidationFailBeforeBackendExecution(t *testing.T) {
	if _, err := Open(Config{}); err == nil {
		t.Fatal("missing backend was accepted")
	}
	backend := &fakeBackend{name: "fake", implementation: "good"}
	limits := DefaultLimits()
	limits.MaxEvents = 0
	if _, err := Open(Config{Root: t.TempDir(), Backend: backend, Limits: limits}); err == nil {
		t.Fatal("invalid limits were accepted")
	}
	engine := newTestEngine(t, backend, nil)
	request := testRequest(t.TempDir(), true)
	request.Task.Objective = ""
	if _, err := engine.Run(context.Background(), request); err == nil {
		t.Fatal("empty objective was accepted")
	}
	request = testRequest(t.TempDir(), true)
	request.Project.Source.Mode = "unknown"
	if _, err := engine.Run(context.Background(), request); !errors.Is(err, ErrUnsupported) {
		t.Fatalf("unsupported source error = %v", err)
	}
	request = testRequest(t.TempDir(), true)
	request.Project.Instructions = []string{"missing.md"}
	result, err := engine.Run(context.Background(), request)
	if err == nil || result.RunID != "" {
		t.Fatalf("missing instruction result=%#v error=%v", result, err)
	}
	projectRoot := t.TempDir()
	outside := filepath.Join(t.TempDir(), "outside.md")
	writeTestFile(t, outside, "outside")
	if err := os.Symlink(outside, filepath.Join(projectRoot, "GUIDE.md")); err == nil {
		request = testRequest(projectRoot, true)
		request.Project.Instructions = []string{"GUIDE.md"}
		result, err = engine.Run(context.Background(), request)
		if err == nil || result.RunID != "" {
			t.Fatalf("escaping instruction result=%#v error=%v", result, err)
		}
	}
	if len(backend.calls) != 0 {
		t.Fatalf("backend ran during invalid preflight: %#v", backend.calls)
	}
}

func TestRequestNormalizationPreservesExplicitEmptyEnvironmentAndZeroRepairs(t *testing.T) {
	request := testRequest(t.TempDir(), true)
	request.Project.Environment.Allow = []string{}
	request.Policy.MaxRepairs = 0
	normalized, err := normalizeRequest(request, DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	if len(normalized.Project.Environment.Allow) != 0 || normalized.Policy.MaxRepairs != 0 {
		t.Fatalf("normalized environment=%v repairs=%d", normalized.Project.Environment.Allow, normalized.Policy.MaxRepairs)
	}
}

func TestReviewConcurrencyIsRaceCleanAndResultOrderIsStable(t *testing.T) {
	projectRoot := t.TempDir()
	started := make(chan struct{}, 2)
	release := make(chan struct{})
	backend := &fakeBackend{name: "fake", implementation: "good", reviewStarted: started, reviewRelease: release}
	engine := newTestEngine(t, backend, nil)
	resultChannel := make(chan Result, 1)
	errorChannel := make(chan error, 1)
	go func() {
		result, err := engine.Run(context.Background(), testRequest(projectRoot, true))
		resultChannel <- result
		errorChannel <- err
	}()
	<-started
	<-started
	close(release)
	result := <-resultChannel
	if err := <-errorChannel; err != nil {
		t.Fatal(err)
	}
	if result.Outcome != OutcomeSucceeded || result.Reviews[0].Lens != "correctness" || result.Reviews[1].Lens != "tests" {
		t.Fatalf("reviews = %#v", result.Reviews)
	}
}

func TestResumeContinuesIdentityBoundImplementationSession(t *testing.T) {
	projectRoot := t.TempDir()
	backend := &fakeBackend{name: "codex", implementation: "good"}
	engine := newTestEngine(t, backend, nil)
	original := testRequest(projectRoot, true)
	original.Workflow = DefaultWorkflow()
	for index := range original.Workflow.Stages {
		if original.Workflow.Stages[index].Kind == StageImplement {
			original.Workflow.Stages[index].Models = Models{"codex": "stored-implement-model"}
		}
	}
	prepareImplementationResume(t, engine, "resume-run", original, "session-implement")

	result, err := engine.Resume(context.Background(), "resume-run")
	if err != nil {
		t.Fatal(err)
	}
	if result.Outcome != OutcomeSucceeded {
		t.Fatalf("result = %#v", result)
	}
	invocation, found := backend.invocation("implement")
	if !found || invocation.Session != "session-implement" || invocation.Model != "stored-implement-model" {
		t.Fatalf("implementation invocation = %#v, found=%v", invocation, found)
	}
}

func TestResumeRejectsDifferentBackendConfigurationBeforeInvocation(t *testing.T) {
	initialBackend := &fakeBackend{name: "codex", implementation: "good", configurationDigest: "sha256:old-model"}
	initialEngine := newTestEngine(t, initialBackend, nil)
	prepareImplementationResume(t, initialEngine, "resume-model-conflict", testRequest(t.TempDir(), true), "session-implement")

	conflictingBackend := &fakeBackend{name: "codex", implementation: "good", configurationDigest: "sha256:new-model"}
	conflictingEngine, err := Open(Config{
		Root: initialEngine.store.Root(), Backend: conflictingBackend, Limits: initialEngine.limits,
	})
	require.NoError(t, err)
	_, err = conflictingEngine.Resume(context.Background(), "resume-model-conflict")
	require.ErrorIs(t, err, ErrUnsupported)
	require.ErrorContains(t, err, "backend identity changed")
	require.Empty(t, conflictingBackend.calls)
}

func TestResumeConsumesCompletedAttemptBeforeCheckpointPublication(t *testing.T) {
	projectRoot := t.TempDir()
	backend := &fakeBackend{name: "fake", implementation: "should-not-run"}
	engine := newTestEngine(t, backend, nil)
	request, err := normalizeRequest(testRequest(projectRoot, true), engine.limits)
	if err != nil {
		t.Fatal(err)
	}
	info, err := engine.describe(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	requestData, _ := json.Marshal(request)
	run, err := engine.store.Create("completed-attempt-run", requestData, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	prepared, err := workspace.Prepare(context.Background(), projectRoot, run.Directory(), workspaceOptions(request.Project, engine.limits))
	if err != nil {
		t.Fatal(err)
	}
	checkpoint := checkpoint{
		Schema: "agentworkflow.checkpoint/v2", Request: request, Backend: info, Limits: engine.limits,
		StartedAt: time.Now().UTC(), Phase: "implement", Completed: map[string]bool{"prepare": true, "discover": true, "plan": true},
		SourceDigest: prepared.Digest, CandidateDigest: prepared.Digest,
		Brief: projectBrief{Summary: "fixture"},
		Plan:  planArtifact{Understanding: "fixture", Steps: []planStep{{Description: "change", Criteria: []int{1}, Verification: []string{"check"}}}},
	}
	if err := saveCheckpoint(run, &checkpoint, "running", "implement", ""); err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, filepath.Join(prepared.Candidate, "result.txt"), "good")
	recorder, err := run.StartAttempt("implement", engine.limits.MaxOutputBytes, engine.limits.MaxEvents, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	output := []byte(`{"summary":"implemented","files":["result.txt"],"tests":[]}`)
	if err := recorder.Finish("completed", "session-implement", output, nil, time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	if err := run.Close(); err != nil {
		t.Fatal(err)
	}

	result, err := engine.Resume(context.Background(), "completed-attempt-run")
	if err != nil {
		t.Fatal(err)
	}
	if result.Outcome != OutcomeSucceeded || backend.callCount("implement") != 0 {
		t.Fatalf("result=%#v implement calls=%d", result, backend.callCount("implement"))
	}
}

func TestProgressReportsDurablePhaseTransitions(t *testing.T) {
	projectRoot := t.TempDir()
	var mu sync.Mutex
	var progress []Progress
	engine := newTestEngine(t, &fakeBackend{name: "fake", implementation: "good"}, func(value Progress) {
		mu.Lock()
		defer mu.Unlock()
		progress = append(progress, value)
	})
	result, err := engine.Run(context.Background(), testRequest(projectRoot, true))
	if err != nil {
		t.Fatal(err)
	}
	mu.Lock()
	defer mu.Unlock()
	if !containsProgress(progress, "implement") || !containsProgress(progress, "final-checks-0") || progress[len(progress)-1].Outcome != OutcomeSucceeded {
		t.Fatalf("progress = %#v, result = %#v", progress, result)
	}
}

//nolint:revive // os.Exit is the protocol used by this subprocess fixture.
func TestProjectCheckHelper(t *testing.T) {
	separator := slicesIndex(os.Args, "--")
	if separator < 0 || separator+2 >= len(os.Args) {
		return
	}
	path := os.Args[separator+1]
	expected := os.Args[separator+2]
	if expected == "mutate" {
		if err := os.WriteFile(path, []byte("generated"), 0o600); err != nil {
			os.Exit(2)
		}
		os.Exit(0)
	}
	data, err := os.ReadFile(path)
	if expected == "exists" && err == nil {
		os.Exit(0)
	}
	if err != nil || string(data) != expected {
		fmt.Fprintf(os.Stderr, "got %q (%v), want %q", data, err, expected)
		os.Exit(1)
	}
	os.Exit(0)
}

type fakeBackend struct {
	name                string
	implementation      string
	implementationPath  string
	repairContent       string
	findingUntilRepair  bool
	invalidPhase        string
	reviewStarted       chan struct{}
	reviewRelease       chan struct{}
	mutateReadOnly      bool
	omitTerminal        bool
	oversizedOutput     bool
	rejectFirstPlan     bool
	noResume            bool
	configurationDigest string

	mu       sync.Mutex
	calls    []Invocation
	repaired bool
}

func (backend *fakeBackend) Describe(context.Context) (BackendInfo, error) {
	capabilities := []Capability{CapabilityReadOnly, CapabilityWorkspaceWrite, CapabilityStructuredOutput, CapabilityCancellation}
	if !backend.noResume {
		capabilities = append(capabilities, CapabilityResume)
	}
	configurationDigest := backend.configurationDigest
	if configurationDigest == "" {
		configurationDigest = "sha256:fake-configuration"
	}
	return BackendInfo{
		Name: backend.name, Version: "fake/v1", ConfigurationDigest: configurationDigest,
		Capabilities: capabilities,
	}, nil
}

func prepareImplementationResume(t *testing.T, engine *Engine, id string, original Request, session string) {
	t.Helper()
	request, err := normalizeRequest(original, engine.limits)
	require.NoError(t, err)
	info, err := engine.describe(context.Background())
	require.NoError(t, err)
	requestData, err := json.Marshal(request)
	require.NoError(t, err)
	run, err := engine.store.Create(id, requestData, time.Now().UTC())
	require.NoError(t, err)
	prepared, err := workspace.Prepare(context.Background(), request.Project.Root, run.Directory(), workspaceOptions(request.Project, engine.limits))
	require.NoError(t, err)
	checkpoint := checkpoint{
		Schema: "agentworkflow.checkpoint/v2", Request: request, Backend: info, Limits: engine.limits,
		StartedAt: time.Now().UTC(), Phase: "implement", Completed: map[string]bool{"prepare": true, "discover": true, "plan": true},
		SourceDigest: prepared.Digest, CandidateDigest: prepared.Digest,
		Brief:                 projectBrief{Summary: "fixture"},
		Plan:                  planArtifact{Understanding: "fixture", Steps: []planStep{{Description: "change", Criteria: []int{1}, Verification: []string{"check"}}}},
		ImplementationSession: session,
	}
	require.NoError(t, saveCheckpoint(run, &checkpoint, "running", "implement", ""))
	require.NoError(t, run.Close())
}

func (backend *fakeBackend) Execute(ctx context.Context, invocation Invocation, sink EventSink) (InvocationResult, error) {
	if err := ctx.Err(); err != nil {
		return InvocationResult{}, err
	}
	if backend.noResume && (invocation.Session != "" || invocation.RetainSession) {
		return InvocationResult{}, errors.New("non-resumable backend received session controls")
	}
	backend.mu.Lock()
	backend.calls = append(backend.calls, invocation)
	repaired := backend.repaired
	backend.mu.Unlock()
	session := invocation.Session
	if session == "" {
		session = "session-" + invocation.Phase
	}
	for _, event := range []Event{{Kind: EventInvocationStarted}, {Kind: EventSessionIdentified, Session: session}} {
		if err := sink.Emit(event); err != nil {
			return InvocationResult{}, err
		}
	}
	var output any
	switch {
	case invocation.Phase == "discover":
		output = projectBrief{Summary: "test project", Languages: []string{"text"}, BuildSystems: []string{"test"}, Architecture: []string{"single fixture"}, Risks: []string{}}
		if backend.mutateReadOnly {
			if err := os.WriteFile(filepath.Join(invocation.Workspace, "intrusion"), []byte("bad"), 0o600); err != nil {
				return InvocationResult{}, err
			}
		}
	case invocation.Phase == "plan" || invocation.Phase == "plan-revise":
		output = planArtifact{Understanding: "implement fixture", Assumptions: []string{}, Risks: []string{}, Tradeoffs: []string{}, Steps: []planStep{{Description: "write result", Files: []string{"result.txt"}, Criteria: []int{1}, Verification: []string{"required check"}}}}
	case strings.HasPrefix(invocation.Phase, "plan-review"):
		if backend.rejectFirstPlan && invocation.Phase == "plan-review" {
			output = planReview{Approved: false, Summary: "plan needs revision", Issues: []string{"add failure analysis"}}
		} else {
			output = planReview{Approved: true, Summary: "plan covers the task", Issues: []string{}}
		}
	case invocation.Phase == "implement":
		path := backend.implementationPath
		if path == "" {
			path = "result.txt"
		}
		if err := os.WriteFile(filepath.Join(invocation.Workspace, filepath.FromSlash(path)), []byte(backend.implementation), 0o600); err != nil {
			return InvocationResult{}, err
		}
		output = changeSummary{Summary: "implemented", Files: []string{path}, Tests: []string{}}
	case strings.HasPrefix(invocation.Phase, "repair-"):
		content := backend.repairContent
		if content == "" {
			content = "good"
		}
		if err := os.WriteFile(filepath.Join(invocation.Workspace, "result.txt"), []byte(content), 0o600); err != nil {
			return InvocationResult{}, err
		}
		backend.mu.Lock()
		backend.repaired = true
		backend.mu.Unlock()
		output = changeSummary{Summary: "repaired", Files: []string{"result.txt"}, Tests: []string{}}
	case strings.HasPrefix(invocation.Phase, "review-"):
		if backend.reviewStarted != nil {
			backend.reviewStarted <- struct{}{}
			select {
			case <-backend.reviewRelease:
			case <-ctx.Done():
				return InvocationResult{}, ctx.Err()
			}
		}
		lens := reviewLens(invocation.Phase)
		findings := []Finding{}
		if backend.findingUntilRepair && !repaired && lens == "correctness" {
			findings = append(findings, Finding{
				ID: "incorrect-result", Lens: lens, Severity: SeverityHigh, Confidence: "high",
				Requirement: "correct result", Location: "result.txt:1", Claim: "result remains bad",
				Evidence: "file contains bad", Reproduction: "read result.txt", Impact: "task fails", ProposedFix: "write good",
			})
		}
		output = reviewArtifact{Lens: lens, Summary: "review complete", Findings: findings}
	default:
		return InvocationResult{}, fmt.Errorf("unexpected phase %s", invocation.Phase)
	}
	encoded, err := json.Marshal(output)
	if err != nil {
		return InvocationResult{}, err
	}
	if backend.invalidPhase == invocation.Phase {
		encoded = []byte(`{"understanding":"bad","unknown":true}`)
	}
	if backend.oversizedOutput {
		encoded, _ = json.Marshal(projectBrief{Summary: strings.Repeat("x", (1<<20)+1), Languages: []string{}, BuildSystems: []string{}, Architecture: []string{}, Risks: []string{}})
	}
	if err := sink.Emit(Event{Kind: EventAgentMessage, Message: "done"}); err != nil {
		return InvocationResult{}, err
	}
	if !backend.omitTerminal {
		if err := sink.Emit(Event{Kind: EventInvocationCompleted}); err != nil {
			return InvocationResult{}, err
		}
	}
	return InvocationResult{Session: session, Output: encoded}, nil
}

func (backend *fakeBackend) callCount(phase string) int {
	backend.mu.Lock()
	defer backend.mu.Unlock()
	count := 0
	for _, invocation := range backend.calls {
		if invocation.Phase == phase {
			count++
		}
	}
	return count
}

func (backend *fakeBackend) invocation(phase string) (Invocation, bool) {
	backend.mu.Lock()
	defer backend.mu.Unlock()
	for _, invocation := range backend.calls {
		if invocation.Phase == phase {
			return invocation, true
		}
	}
	return Invocation{}, false
}

func newTestEngine(t *testing.T, backend Backend, observe func(Progress)) *Engine {
	t.Helper()
	limits := DefaultLimits()
	limits.InvocationTimeout = time.Minute
	limits.CheckTimeout = time.Minute
	limits.MaxOutputBytes = 1 << 20
	limits.MaxEvents = 100
	limits.MaxSourceBytes = 32 << 20
	limits.MaxSourceFiles = 1_000
	engine, err := Open(Config{Root: t.TempDir(), Backend: backend, Limits: limits, Observe: observe})
	if err != nil {
		t.Fatal(err)
	}
	return engine
}

func testRequest(root string, check bool) Request {
	request := Request{
		Task:    Task{Objective: "write a good result", SuccessCriteria: []string{"result.txt contains good"}},
		Project: Project{Root: root, Source: SourcePolicy{Mode: SourceDirectoryCopy}},
		Policy:  Policy{Assurance: AssuranceStandard, MaxRepairs: 1},
	}
	if check {
		request.Project.Checks = []Check{{
			Name: "test", Command: checkCommand("result.txt", "good"), Directory: ".", Required: true,
		}}
	}
	return request
}

func disableStage(request *Request, kind StageKind) {
	request.Workflow = DefaultWorkflow()
	for index := range request.Workflow.Stages {
		if request.Workflow.Stages[index].Kind == kind {
			request.Workflow.Stages[index].Enabled = false
			return
		}
	}
}

func checkCommand(path, expected string) []string {
	return []string{os.Args[0], "-test.run=TestProjectCheckHelper", "--", path, expected}
}

func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func legacyDigest(domain string, data []byte) string {
	hasher := sha256.New()
	_, _ = hasher.Write([]byte(domain))
	_, _ = hasher.Write([]byte{0})
	_, _ = hasher.Write(data)
	return "sha256:" + hex.EncodeToString(hasher.Sum(nil))
}

func reviewLens(phase string) string {
	value := strings.TrimPrefix(phase, "review-")
	separator := strings.IndexByte(value, '-')
	if separator < 0 {
		return value
	}
	return value[separator+1:]
}

func containsChange(changes []Change, path, kind string) bool {
	for _, change := range changes {
		if change.Path == path && change.Kind == kind {
			return true
		}
	}
	return false
}

func slicesContainsDisposition(values []FindingDisposition, target FindingDisposition) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func containsProgress(progress []Progress, phase string) bool {
	for _, value := range progress {
		if value.Phase == phase {
			return true
		}
	}
	return false
}

func slicesIndex(values []string, target string) int {
	for index, value := range values {
		if value == target {
			return index
		}
	}
	return -1
}
