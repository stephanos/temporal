package agentworkflow

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"
)

var (
	ErrOutputLimit = errors.New("agent workflow output limit exceeded")
	ErrEventLimit  = errors.New("agent workflow event limit exceeded")
	ErrStageExists = errors.New("agent workflow stage already exists")
)

type Config struct {
	Root           string
	Command        []string
	Timeout        time.Duration
	MaxOutputBytes uint64
	MaxEvents      int
}

type Engine struct {
	config Config
	runner commandRunner
}

type StageRequest struct {
	RunID        string
	Stage        string
	Workspace    string
	Prompt       string
	OutputSchema json.RawMessage
}

type StageStatus string

const StageCompleted StageStatus = "completed"

type StageResult struct {
	Schema         string      `json:"schema"`
	RunID          string      `json:"run_id"`
	Stage          string      `json:"stage"`
	Status         StageStatus `json:"status"`
	ThreadID       string      `json:"thread_id,omitempty"`
	FinalOutput    string      `json:"final_output"`
	EventCount     int         `json:"event_count"`
	StdoutSHA256   string      `json:"stdout_sha256"`
	StderrSHA256   string      `json:"stderr_sha256"`
	StdoutBytes    uint64      `json:"stdout_bytes"`
	StderrBytes    uint64      `json:"stderr_bytes"`
	RunDirectory   string      `json:"run_directory"`
	StageDirectory string      `json:"stage_directory"`
}

type ReviewRequest struct {
	Name         string
	Workspace    string
	Prompt       string
	OutputSchema json.RawMessage
}

type ReviewGroupRequest struct {
	RunID   string
	Reviews []ReviewRequest
}

type ReviewGroupResult struct {
	RunID   string        `json:"run_id"`
	Reviews []StageResult `json:"reviews"`
}

type RunResult struct {
	RunID        string        `json:"run_id"`
	RunDirectory string        `json:"run_directory"`
	Stages       []StageResult `json:"stages"`
}

func Open(config Config) (*Engine, error) {
	return newEngine(config, processRunner{})
}

func newEngine(config Config, runner commandRunner) (*Engine, error) {
	if runner == nil {
		return nil, errors.New("agent workflow command runner is required")
	}
	root, err := filepath.Abs(config.Root)
	if err != nil || root == string(filepath.Separator) {
		return nil, errors.Join(errors.New("agent workflow root must be an absolute non-root directory"), err)
	}
	if len(config.Command) == 0 || strings.TrimSpace(config.Command[0]) == "" {
		return nil, errors.New("agent workflow command is required")
	}
	if config.Timeout <= 0 || config.MaxOutputBytes == 0 || config.MaxEvents <= 0 {
		return nil, errors.New("agent workflow bounds must be positive")
	}
	if config.MaxOutputBytes > uint64(^uint(0)>>1) {
		return nil, errors.New("agent workflow output limit exceeds addressable memory")
	}
	config.Root = root
	config.Command = append([]string(nil), config.Command...)
	return &Engine{config: config, runner: runner}, nil
}

func (engine *Engine) ExecuteStage(ctx context.Context, request StageRequest) (StageResult, error) {
	if err := validateComponent("run identity", request.RunID); err != nil {
		return StageResult{}, err
	}
	if err := validateComponent("stage", request.Stage); err != nil {
		return StageResult{}, err
	}
	workspace, err := existingDirectory(request.Workspace)
	if err != nil {
		return StageResult{}, err
	}
	if strings.TrimSpace(request.Prompt) == "" {
		return StageResult{}, errors.New("agent workflow prompt is required")
	}
	runDirectory := filepath.Join(engine.config.Root, request.RunID)
	stageDirectory := filepath.Join(runDirectory, "stages", request.Stage)
	if err := os.MkdirAll(stageDirectory, 0o700); err != nil {
		return StageResult{}, fmt.Errorf("create agent workflow stage directory: %w", err)
	}
	lockPath := filepath.Join(stageDirectory, "running.lock")
	lock, err := os.OpenFile(lockPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		if errors.Is(err, os.ErrExist) {
			return StageResult{}, ErrStageExists
		}
		return StageResult{}, fmt.Errorf("lock agent workflow stage: %w", err)
	}
	if closeErr := lock.Close(); closeErr != nil {
		return StageResult{}, fmt.Errorf("close agent workflow stage lock: %w", closeErr)
	}
	defer func() { _ = os.Remove(lockPath) }()
	if _, err := os.Stat(filepath.Join(stageDirectory, "stage.json")); err == nil {
		return StageResult{}, ErrStageExists
	} else if !errors.Is(err, os.ErrNotExist) {
		return StageResult{}, fmt.Errorf("inspect agent workflow stage: %w", err)
	}

	arguments := append([]string(nil), engine.config.Command[1:]...)
	arguments = append(arguments, "exec", "--json", "--sandbox", "workspace-write")
	if len(request.OutputSchema) > 0 {
		if !json.Valid(request.OutputSchema) {
			return StageResult{}, errors.New("agent workflow output schema is not valid JSON")
		}
		schemaPath := filepath.Join(stageDirectory, "output-schema.json")
		if err := atomicWrite(schemaPath, request.OutputSchema); err != nil {
			return StageResult{}, err
		}
		arguments = append(arguments, "--output-schema", schemaPath)
	}
	arguments = append(arguments, "-")

	stageContext, cancel := context.WithTimeout(ctx, engine.config.Timeout)
	defer cancel()
	commandResult, err := engine.runner.run(stageContext, commandRequest{
		executable: engine.config.Command[0],
		arguments:  arguments,
		directory:  workspace,
		stdin:      request.Prompt,
		limit:      engine.config.MaxOutputBytes,
	})
	if err != nil {
		if errors.Is(err, ErrOutputLimit) {
			return StageResult{}, err
		}
		if stageContext.Err() != nil {
			return StageResult{}, fmt.Errorf("execute agent workflow stage: %w", stageContext.Err())
		}
		return StageResult{}, fmt.Errorf("execute agent workflow stage: %w", err)
	}
	if uint64(len(commandResult.stdout)) > engine.config.MaxOutputBytes || uint64(len(commandResult.stderr)) > engine.config.MaxOutputBytes {
		return StageResult{}, ErrOutputLimit
	}
	parsed, err := parseEvents(commandResult.stdout, engine.config.MaxEvents)
	if err != nil {
		return StageResult{}, err
	}
	if err := atomicWrite(filepath.Join(stageDirectory, "events.jsonl"), commandResult.stdout); err != nil {
		return StageResult{}, err
	}
	if err := atomicWrite(filepath.Join(stageDirectory, "stderr.log"), commandResult.stderr); err != nil {
		return StageResult{}, err
	}
	result := StageResult{
		Schema:         "agentworkflow.stage-result/v1",
		RunID:          request.RunID,
		Stage:          request.Stage,
		Status:         StageCompleted,
		ThreadID:       parsed.threadID,
		FinalOutput:    parsed.finalOutput,
		EventCount:     parsed.count,
		StdoutSHA256:   digest("agentworkflow.stdout/v1", commandResult.stdout),
		StderrSHA256:   digest("agentworkflow.stderr/v1", commandResult.stderr),
		StdoutBytes:    uint64(len(commandResult.stdout)),
		StderrBytes:    uint64(len(commandResult.stderr)),
		RunDirectory:   runDirectory,
		StageDirectory: stageDirectory,
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		return StageResult{}, fmt.Errorf("encode agent workflow stage result: %w", err)
	}
	if err := atomicWrite(filepath.Join(stageDirectory, "stage.json"), encoded); err != nil {
		return StageResult{}, err
	}
	return result, nil
}

func (engine *Engine) ExecuteReviewGroup(ctx context.Context, request ReviewGroupRequest) (ReviewGroupResult, error) {
	if err := validateComponent("run identity", request.RunID); err != nil {
		return ReviewGroupResult{}, err
	}
	if len(request.Reviews) == 0 {
		return ReviewGroupResult{}, errors.New("agent workflow review group is empty")
	}
	result := ReviewGroupResult{RunID: request.RunID, Reviews: make([]StageResult, len(request.Reviews))}
	errorsByIndex := make([]error, len(request.Reviews))
	seen := make(map[string]struct{}, len(request.Reviews))
	for _, review := range request.Reviews {
		if err := validateComponent("review name", review.Name); err != nil {
			return ReviewGroupResult{}, err
		}
		if _, found := seen[review.Name]; found {
			return ReviewGroupResult{}, fmt.Errorf("agent workflow review %q is duplicated", review.Name)
		}
		seen[review.Name] = struct{}{}
	}
	var group sync.WaitGroup
	for index, review := range request.Reviews {
		group.Add(1)
		go func() {
			defer group.Done()
			result.Reviews[index], errorsByIndex[index] = engine.ExecuteStage(ctx, StageRequest{
				RunID: request.RunID, Stage: "review-" + review.Name, Workspace: review.Workspace,
				Prompt: review.Prompt, OutputSchema: review.OutputSchema,
			})
		}()
	}
	group.Wait()
	for index, err := range errorsByIndex {
		if err != nil {
			return result, fmt.Errorf("execute agent workflow review %q: %w", request.Reviews[index].Name, err)
		}
	}
	return result, nil
}

func (engine *Engine) Resume(_ context.Context, runDirectory string) (RunResult, error) {
	directory, err := filepath.Abs(runDirectory)
	if err != nil {
		return RunResult{}, fmt.Errorf("resolve agent workflow run directory: %w", err)
	}
	relative, err := filepath.Rel(engine.config.Root, directory)
	if err != nil || relative == "." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) || relative == ".." {
		return RunResult{}, errors.Join(errors.New("agent workflow run directory is outside its root"), err)
	}
	if strings.ContainsRune(relative, filepath.Separator) {
		return RunResult{}, errors.New("agent workflow run directory must be a direct child of its root")
	}
	entries, err := os.ReadDir(filepath.Join(directory, "stages"))
	if err != nil {
		return RunResult{}, fmt.Errorf("read agent workflow stages: %w", err)
	}
	result := RunResult{RunID: relative, RunDirectory: directory, Stages: []StageResult{}}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		encoded, readErr := os.ReadFile(filepath.Join(directory, "stages", entry.Name(), "stage.json"))
		if errors.Is(readErr, os.ErrNotExist) {
			continue
		}
		if readErr != nil {
			return RunResult{}, fmt.Errorf("read agent workflow stage %q: %w", entry.Name(), readErr)
		}
		var stage StageResult
		if err := json.Unmarshal(encoded, &stage); err != nil {
			return RunResult{}, fmt.Errorf("decode agent workflow stage %q: %w", entry.Name(), err)
		}
		if stage.Schema != "agentworkflow.stage-result/v1" || stage.RunID != relative || stage.Stage != entry.Name() || stage.Status != StageCompleted {
			return RunResult{}, fmt.Errorf("agent workflow stage %q record is inconsistent", entry.Name())
		}
		result.Stages = append(result.Stages, stage)
	}
	slices.SortFunc(result.Stages, func(left, right StageResult) int {
		return strings.Compare(left.Stage, right.Stage)
	})
	return result, nil
}

type commandRequest struct {
	executable string
	arguments  []string
	directory  string
	stdin      string
	limit      uint64
}

type commandResult struct {
	stdout []byte
	stderr []byte
}

type commandRunner interface {
	run(context.Context, commandRequest) (commandResult, error)
}

type processRunner struct{}

func (processRunner) run(ctx context.Context, request commandRequest) (commandResult, error) {
	stdout := newLimitBuffer(request.limit)
	stderr := newLimitBuffer(request.limit)
	command := exec.CommandContext(ctx, request.executable, request.arguments...)
	command.Dir = request.directory
	command.Stdin = strings.NewReader(request.stdin)
	command.Stdout = stdout
	command.Stderr = stderr
	err := command.Run()
	if stdout.exceeded || stderr.exceeded {
		return commandResult{}, ErrOutputLimit
	}
	if err != nil {
		return commandResult{stdout: stdout.bytes(), stderr: stderr.bytes()}, err
	}
	return commandResult{stdout: stdout.bytes(), stderr: stderr.bytes()}, nil
}

type limitBuffer struct {
	buffer   bytes.Buffer
	limit    uint64
	exceeded bool
}

func newLimitBuffer(limit uint64) *limitBuffer {
	return &limitBuffer{limit: limit}
}

func (buffer *limitBuffer) Write(data []byte) (int, error) {
	remaining := buffer.limit - uint64(buffer.buffer.Len())
	if uint64(len(data)) > remaining {
		_, _ = buffer.buffer.Write(data[:int(remaining)])
		buffer.exceeded = true
		return len(data), nil
	}
	return buffer.buffer.Write(data)
}

func (buffer *limitBuffer) bytes() []byte {
	return append([]byte(nil), buffer.buffer.Bytes()...)
}

type parsedEvents struct {
	threadID    string
	finalOutput string
	count       int
}

func parseEvents(data []byte, limit int) (parsedEvents, error) {
	result := parsedEvents{}
	scanner := bufio.NewScanner(bytes.NewReader(data))
	scanner.Buffer(make([]byte, 64*1024), len(data)+1)
	for scanner.Scan() {
		if len(bytes.TrimSpace(scanner.Bytes())) == 0 {
			continue
		}
		result.count++
		if result.count > limit {
			return parsedEvents{}, ErrEventLimit
		}
		var event struct {
			Type     string `json:"type"`
			ThreadID string `json:"thread_id"`
			Item     struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"item"`
		}
		if err := json.Unmarshal(scanner.Bytes(), &event); err != nil {
			return parsedEvents{}, fmt.Errorf("decode agent workflow JSONL event %d: %w", result.count, err)
		}
		if event.Type == "thread.started" {
			result.threadID = event.ThreadID
		}
		if event.Type == "item.completed" && event.Item.Type == "agent_message" {
			result.finalOutput = event.Item.Text
		}
	}
	if err := scanner.Err(); err != nil && !errors.Is(err, io.EOF) {
		return parsedEvents{}, fmt.Errorf("read agent workflow JSONL: %w", err)
	}
	return result, nil
}

func existingDirectory(path string) (string, error) {
	directory, err := filepath.Abs(path)
	if err != nil || directory == string(filepath.Separator) {
		return "", errors.Join(errors.New("agent workflow workspace must be an absolute non-root directory"), err)
	}
	info, err := os.Stat(directory)
	if err != nil || !info.IsDir() {
		return "", errors.Join(errors.New("agent workflow workspace is not a directory"), err)
	}
	return directory, nil
}

func validateComponent(kind, value string) error {
	if value == "" || value == "." || value == ".." {
		return fmt.Errorf("agent workflow %s is invalid", kind)
	}
	for _, character := range value {
		if (character >= 'a' && character <= 'z') || (character >= 'A' && character <= 'Z') ||
			(character >= '0' && character <= '9') || character == '-' || character == '_' || character == '.' {
			continue
		}
		return fmt.Errorf("agent workflow %s %q contains an invalid character", kind, value)
	}
	return nil
}

func atomicWrite(path string, data []byte) error {
	directory := filepath.Dir(path)
	temporary, err := os.CreateTemp(directory, ".write-*")
	if err != nil {
		return fmt.Errorf("create agent workflow temporary file: %w", err)
	}
	temporaryPath := temporary.Name()
	defer func() { _ = os.Remove(temporaryPath) }()
	if err := temporary.Chmod(0o600); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("set agent workflow file permissions: %w", err)
	}
	if _, err := temporary.Write(data); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("write agent workflow file: %w", err)
	}
	if err := temporary.Sync(); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("sync agent workflow file: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close agent workflow file: %w", err)
	}
	if err := os.Rename(temporaryPath, path); err != nil {
		return fmt.Errorf("publish agent workflow file: %w", err)
	}
	directoryHandle, err := os.Open(directory)
	if err != nil {
		return fmt.Errorf("open agent workflow directory: %w", err)
	}
	defer directoryHandle.Close()
	if err := directoryHandle.Sync(); err != nil {
		return fmt.Errorf("sync agent workflow directory: %w", err)
	}
	return nil
}

func digest(domain string, data []byte) string {
	hasher := sha256.New()
	_, _ = hasher.Write([]byte(domain))
	_, _ = hasher.Write([]byte{0})
	_, _ = hasher.Write(data)
	return "sha256:" + hex.EncodeToString(hasher.Sum(nil))
}
