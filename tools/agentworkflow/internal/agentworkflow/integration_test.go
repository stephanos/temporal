package agentworkflow_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"go.temporal.io/server/tools/agentworkflow/internal/agentworkflow"
	"go.temporal.io/server/tools/agentworkflow/internal/backend/claude"
	"go.temporal.io/server/tools/agentworkflow/internal/backend/codex"
)

func TestRealAdaptersRunTheSameEndToEndWorkflow(t *testing.T) {
	backends := map[string]func([]string) (agentworkflow.Backend, error){
		"codex": func(command []string) (agentworkflow.Backend, error) {
			return codex.New(codex.Config{Command: command})
		},
		"claude": func(command []string) (agentworkflow.Backend, error) {
			return claude.New(claude.Config{Command: command})
		},
	}
	for name, constructor := range backends {
		t.Run(name, func(t *testing.T) {
			projectRoot := t.TempDir()
			command := []string{os.Args[0], "-test.run=TestEndToEndProviderCLI", "--", name}
			backend, err := constructor(command)
			if err != nil {
				t.Fatal(err)
			}
			limits := agentworkflow.DefaultLimits()
			limits.InvocationTimeout = time.Minute
			limits.CheckTimeout = time.Minute
			limits.MaxOutputBytes = 1 << 20
			limits.MaxSourceBytes = 16 << 20
			limits.MaxSourceFiles = 1_000
			engine, err := agentworkflow.Open(agentworkflow.Config{Root: t.TempDir(), Backend: backend, Limits: limits})
			if err != nil {
				t.Fatal(err)
			}
			result, err := engine.Run(context.Background(), agentworkflow.Request{
				Task: agentworkflow.Task{Objective: "write result.txt", SuccessCriteria: []string{"result.txt contains good"}},
				Project: agentworkflow.Project{
					Root: projectRoot, Source: agentworkflow.SourcePolicy{Mode: agentworkflow.SourceDirectoryCopy},
					Checks: []agentworkflow.Check{{Name: "test", Command: []string{os.Args[0], "-test.run=TestEndToEndCheck", "--", "result.txt"}, Required: true}},
				},
				Policy: agentworkflow.Policy{Assurance: agentworkflow.AssuranceStandard, MaxRepairs: 1},
			})
			if err != nil {
				t.Fatal(err)
			}
			if result.Outcome != agentworkflow.OutcomeSucceeded || result.Backend.Name != name {
				t.Fatalf("result = %#v", result)
			}
			if _, err := os.Stat(filepath.Join(projectRoot, "result.txt")); !os.IsNotExist(err) {
				t.Fatalf("adapter mutated original workspace: %v", err)
			}
		})
	}
}

//nolint:errcheck,revive // Writes and exits intentionally model a real provider subprocess.
func TestEndToEndProviderCLI(t *testing.T) {
	separator := argumentIndex(os.Args, "--")
	if separator < 0 || separator+1 >= len(os.Args) {
		return
	}
	provider := os.Args[separator+1]
	arguments := os.Args[separator+2:]
	if containsArgument(arguments, "--version") {
		if provider == "codex" {
			fmt.Fprintln(os.Stdout, "codex-cli e2e-1")
		} else {
			fmt.Fprintln(os.Stdout, "2.1.999 (Claude Code)")
		}
		os.Exit(0)
	}
	var prompt string
	var schema []byte
	var outputPath string
	if provider == "codex" {
		promptData, _ := io.ReadAll(os.Stdin)
		prompt = string(promptData)
		schemaPath := argumentValue(arguments, "--output-schema")
		schema, _ = os.ReadFile(schemaPath)
		outputPath = argumentValue(arguments, "--output-last-message")
	} else {
		promptData, _ := io.ReadAll(os.Stdin)
		prompt = string(promptData)
		schema = []byte(argumentValue(arguments, "--json-schema"))
	}
	if !json.Valid(schema) {
		fmt.Fprintln(os.Stderr, "invalid schema")
		os.Exit(20)
	}
	output, err := structuredOutput(prompt)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(21)
	}
	if strings.HasPrefix(prompt, "Agentworkflow phase: implement\n") {
		if err := os.WriteFile("result.txt", []byte("good"), 0o600); err != nil {
			os.Exit(22)
		}
	}
	if provider == "codex" {
		if err := os.WriteFile(outputPath, output, 0o600); err != nil {
			os.Exit(23)
		}
		fmt.Fprintln(os.Stdout, `{"type":"thread.started","thread_id":"session-e2e"}`)
		fmt.Fprintln(os.Stdout, `{"type":"turn.started"}`)
		fmt.Fprintln(os.Stdout, `{"type":"item.completed","item":{"type":"agent_message","text":"done"}}`)
		fmt.Fprintln(os.Stdout, `{"type":"turn.completed","usage":{"input_tokens":1,"output_tokens":1}}`)
	} else {
		var structured any
		_ = json.Unmarshal(output, &structured)
		payload, _ := json.Marshal(map[string]any{
			"type": "result", "subtype": "success", "is_error": false, "result": "done",
			"session_id": "session-e2e", "structured_output": structured,
		})
		fmt.Fprintln(os.Stdout, string(payload))
	}
	os.Exit(0)
}

//nolint:revive // os.Exit is the protocol used by this subprocess fixture.
func TestEndToEndCheck(t *testing.T) {
	separator := argumentIndex(os.Args, "--")
	if separator < 0 || separator+1 >= len(os.Args) {
		return
	}
	data, err := os.ReadFile(os.Args[separator+1])
	if err != nil || string(data) != "good" {
		os.Exit(1)
	}
	os.Exit(0)
}

func structuredOutput(prompt string) ([]byte, error) {
	switch {
	case strings.HasPrefix(prompt, "Agentworkflow phase: discover\n"):
		return json.Marshal(map[string]any{"summary": "fixture", "languages": []string{"text"}, "build_systems": []string{}, "architecture": []string{}, "risks": []string{}})
	case strings.HasPrefix(prompt, "Agentworkflow phase: plan\n"):
		return json.Marshal(map[string]any{
			"understanding": "write the file", "assumptions": []string{}, "risks": []string{}, "tradeoffs": []string{},
			"steps": []any{map[string]any{"description": "write result", "files": []string{"result.txt"}, "criteria": []int{1}, "verification": []string{"run test"}}},
		})
	case strings.HasPrefix(prompt, "Agentworkflow phase: implement\n"):
		return json.Marshal(map[string]any{"summary": "implemented", "files": []string{"result.txt"}, "tests": []string{}})
	case strings.HasPrefix(prompt, "Agentworkflow phase: review\n"):
		lens := "correctness"
		begin := strings.Index(prompt, "BEGIN_CONTEXT\n")
		end := strings.Index(prompt, "\nEND_CONTEXT")
		if begin >= 0 && end > begin {
			var contextValue map[string]any
			_ = json.Unmarshal([]byte(prompt[begin+len("BEGIN_CONTEXT\n"):end]), &contextValue)
			if value, ok := contextValue["lens"].(string); ok {
				lens = value
			}
		}
		return json.Marshal(map[string]any{"lens": lens, "summary": "clean", "findings": []any{}})
	default:
		return nil, fmt.Errorf("unknown prompt phase: %.80s", prompt)
	}
}

func argumentIndex(values []string, target string) int {
	for index, value := range values {
		if value == target {
			return index
		}
	}
	return -1
}

func containsArgument(values []string, target string) bool {
	return argumentIndex(values, target) >= 0
}

func argumentValue(values []string, name string) string {
	index := argumentIndex(values, name)
	if index < 0 || index+1 >= len(values) {
		return ""
	}
	return values[index+1]
}
