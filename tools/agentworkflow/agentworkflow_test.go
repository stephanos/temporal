package agentworkflow

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestExecuteStagePersistsBoundedCodexEvents(t *testing.T) {
	root := t.TempDir()
	workspace := t.TempDir()
	runner := &fakeRunner{result: commandResult{
		stdout: []byte("{\"type\":\"thread.started\",\"thread_id\":\"thread-1\"}\n" +
			"{\"type\":\"item.completed\",\"item\":{\"type\":\"agent_message\",\"text\":\"implemented\"}}\n"),
		stderr: []byte("diagnostic"),
	}}
	engine, err := newEngine(Config{
		Root:           root,
		Command:        []string{"codex"},
		Timeout:        time.Minute,
		MaxOutputBytes: 1024,
		MaxEvents:      10,
	}, runner)
	if err != nil {
		t.Fatal(err)
	}

	result, err := engine.ExecuteStage(context.Background(), StageRequest{
		RunID:     "run-1",
		Stage:     "implement",
		Workspace: workspace,
		Prompt:    "Implement the accepted semantic lock.",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != StageCompleted || result.ThreadID != "thread-1" || result.FinalOutput != "implemented" {
		t.Fatalf("stage result = %#v", result)
	}
	if len(runner.requests) != 1 {
		t.Fatalf("runner calls = %d, want 1", len(runner.requests))
	}
	request := runner.requests[0]
	if request.directory != workspace || request.stdin != "Implement the accepted semantic lock." {
		t.Fatalf("command request = %#v", request)
	}
	if !containsSequence(request.arguments, []string{"exec", "--json", "--sandbox", "workspace-write", "-"}) {
		t.Fatalf("command arguments = %v", request.arguments)
	}

	stagePath := filepath.Join(root, "run-1", "stages", "implement", "stage.json")
	encoded, err := os.ReadFile(stagePath)
	if err != nil {
		t.Fatal(err)
	}
	var stored StageResult
	if err := json.Unmarshal(encoded, &stored); err != nil {
		t.Fatal(err)
	}
	if stored.FinalOutput != result.FinalOutput || stored.StdoutSHA256 != result.StdoutSHA256 {
		t.Fatalf("stored stage = %#v, want %#v", stored, result)
	}
}

func TestExecuteStageRejectsOutputBeforePublishingCompletion(t *testing.T) {
	root := t.TempDir()
	engine, err := newEngine(Config{
		Root:           root,
		Command:        []string{"codex"},
		Timeout:        time.Minute,
		MaxOutputBytes: 8,
		MaxEvents:      10,
	}, &fakeRunner{result: commandResult{stdout: []byte("123456789")}})
	if err != nil {
		t.Fatal(err)
	}

	_, err = engine.ExecuteStage(context.Background(), StageRequest{
		RunID: "run-1", Stage: "review", Workspace: t.TempDir(), Prompt: "review",
	})
	if !errors.Is(err, ErrOutputLimit) {
		t.Fatalf("ExecuteStage() error = %v, want ErrOutputLimit", err)
	}
	if _, statErr := os.Stat(filepath.Join(root, "run-1", "stages", "review", "stage.json")); !os.IsNotExist(statErr) {
		t.Fatalf("terminal stage record exists after rejected output: %v", statErr)
	}
}

func TestExecuteReviewGroupKeepsReviewResultsInDeclaredOrder(t *testing.T) {
	root := t.TempDir()
	runner := &fakeRunner{results: []commandResult{
		{stdout: []byte("{\"type\":\"item.completed\",\"item\":{\"type\":\"agent_message\",\"text\":\"first\"}}\n")},
		{stdout: []byte("{\"type\":\"item.completed\",\"item\":{\"type\":\"agent_message\",\"text\":\"second\"}}\n")},
	}}
	engine, err := newEngine(Config{
		Root: root, Command: []string{"codex"}, Timeout: time.Minute, MaxOutputBytes: 1024, MaxEvents: 10,
	}, runner)
	if err != nil {
		t.Fatal(err)
	}
	workspace := t.TempDir()

	result, err := engine.ExecuteReviewGroup(context.Background(), ReviewGroupRequest{
		RunID: "run-1",
		Reviews: []ReviewRequest{
			{Name: "semantics", Workspace: workspace, Prompt: "review semantics"},
			{Name: "tests", Workspace: workspace, Prompt: "review tests"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Reviews) != 2 || result.Reviews[0].Stage != "review-semantics" || result.Reviews[1].Stage != "review-tests" {
		t.Fatalf("review results = %#v", result.Reviews)
	}
}

func TestResumeReadsOnlyCompletedStageRecords(t *testing.T) {
	root := t.TempDir()
	engine, err := newEngine(Config{
		Root: root, Command: []string{"codex"}, Timeout: time.Minute, MaxOutputBytes: 1024, MaxEvents: 10,
	}, &fakeRunner{result: commandResult{stdout: []byte("{\"type\":\"item.completed\",\"item\":{\"type\":\"agent_message\",\"text\":\"done\"}}\n")}})
	if err != nil {
		t.Fatal(err)
	}
	workspace := t.TempDir()
	if _, err := engine.ExecuteStage(context.Background(), StageRequest{RunID: "run-1", Stage: "plan", Workspace: workspace, Prompt: "plan"}); err != nil {
		t.Fatal(err)
	}

	result, err := engine.Resume(context.Background(), filepath.Join(root, "run-1"))
	if err != nil {
		t.Fatal(err)
	}
	if result.RunID != "run-1" || len(result.Stages) != 1 || result.Stages[0].Stage != "plan" {
		t.Fatalf("resume result = %#v", result)
	}
}

type fakeRunner struct {
	result   commandResult
	results  []commandResult
	requests []commandRequest
}

func (runner *fakeRunner) run(_ context.Context, request commandRequest) (commandResult, error) {
	runner.requests = append(runner.requests, request)
	if len(runner.results) == 0 {
		return runner.result, nil
	}
	result := runner.results[0]
	runner.results = runner.results[1:]
	return result, nil
}

func containsSequence(values, sequence []string) bool {
	if len(sequence) > len(values) {
		return false
	}
	for start := 0; start <= len(values)-len(sequence); start++ {
		matched := true
		for index := range sequence {
			if values[start+index] != sequence[index] {
				matched = false
				break
			}
		}
		if matched {
			return true
		}
	}
	return false
}
