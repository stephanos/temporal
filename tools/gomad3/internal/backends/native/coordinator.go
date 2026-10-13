package native

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"time"

	"go.temporal.io/server/tools/gomad3/internal/backends"
	hostexec "go.temporal.io/server/tools/gomad3/internal/host/exec"
)

const maximumCoordinatorMessageBytes = 16 << 20

func decodeCoordinatorFields(input []byte) (map[string]json.RawMessage, error) {
	if len(input) > maximumCoordinatorMessageBytes {
		return nil, errors.New("coordinator request exceeds its byte limit")
	}
	decoder := json.NewDecoder(bytes.NewReader(input))
	var fields map[string]json.RawMessage
	if err := decoder.Decode(&fields); err != nil {
		return nil, fmt.Errorf("decode coordinator request: %w", err)
	}
	if token, err := decoder.Token(); err != io.EOF {
		return nil, fmt.Errorf("trailing coordinator request %v: %w", token, err)
	}
	if fields == nil {
		return nil, errors.New("coordinator request must be an object")
	}
	return fields, nil
}

func CoordinatorInput(input []byte) (Config, []byte, error) {
	fields, err := decodeCoordinatorFields(input)
	if err != nil {
		return Config{}, nil, err
	}
	var config Config
	if err := json.Unmarshal(fields["SupervisorCommand"], &config.SupervisorCommand); err != nil {
		return Config{}, nil, fmt.Errorf("decode coordinator supervisor command: %w", err)
	}
	if err := json.Unmarshal(fields["RunnerBuild"], &config.RunnerBuild); err != nil {
		return Config{}, nil, fmt.Errorf("decode coordinator Runner build: %w", err)
	}
	var target struct{ ToolchainRoot string }
	if err := json.Unmarshal(fields["Target"], &target); err != nil {
		return Config{}, nil, fmt.Errorf("decode coordinator target: %w", err)
	}
	config.ToolchainRoot = target.ToolchainRoot
	delete(fields, "SupervisorCommand")
	delete(fields, "RunnerBuild")
	shared, err := json.Marshal(fields)
	return config, shared, err
}

func (provider *Provider) coordinatorInput(input []byte) ([]byte, error) {
	fields, err := decodeCoordinatorFields(input)
	if err != nil {
		return nil, err
	}
	command, err := json.Marshal(provider.config.SupervisorCommand)
	if err != nil {
		return nil, err
	}
	build, err := json.Marshal(provider.config.RunnerBuild)
	if err != nil {
		return nil, err
	}
	fields["SupervisorCommand"] = command
	fields["RunnerBuild"] = build
	if provider.config.ToolchainRoot != "" {
		var targetFields map[string]json.RawMessage
		if err := json.Unmarshal(fields["Target"], &targetFields); err != nil {
			return nil, fmt.Errorf("decode coordinator target: %w", err)
		}
		if targetFields == nil {
			return nil, errors.New("coordinator target must be an object")
		}
		var root string
		if encodedRoot := targetFields["ToolchainRoot"]; encodedRoot != nil {
			if err := json.Unmarshal(encodedRoot, &root); err != nil {
				return nil, fmt.Errorf("decode coordinator toolchain root: %w", err)
			}
		}
		if root == "" {
			encodedRoot, err := json.Marshal(provider.config.ToolchainRoot)
			if err != nil {
				return nil, err
			}
			targetFields["ToolchainRoot"] = encodedRoot
			fields["Target"], err = json.Marshal(targetFields)
			if err != nil {
				return nil, err
			}
		}
	}
	encoded, err := json.Marshal(fields)
	if err != nil {
		return nil, err
	}
	if len(encoded) > maximumCoordinatorMessageBytes {
		return nil, errors.New("coordinator request exceeds its byte limit")
	}
	return encoded, nil
}

func (provider *Provider) executeCoordinator(ctx context.Context, request backends.CoordinatorRequest) (backends.ExecutionResult, error) {
	failure := func(reason string, err error) (backends.ExecutionResult, error) {
		if request.Stdout != nil {
			err = errors.Join(err, request.Stdout.Close())
		}
		return backends.ExecutionResult{CoordinatorErrorReason: reason}, err
	}
	if err := ctx.Err(); err != nil {
		return failure("coordinator_timeout", err)
	}
	if len(provider.config.CoordinatorCommand) == 0 || provider.config.CoordinatorCommand[0] == "" {
		return failure("coordinator_start", errors.New("coordinator command is required"))
	}
	if request.Stdout == nil {
		return failure("coordinator_pipe", errors.New("coordinator output pipe is required"))
	}
	if request.Deadline.IsZero() {
		return failure("coordinator_timeout", errors.New("coordinator deadline is required"))
	}
	input, err := provider.coordinatorInput(request.Input)
	if err != nil {
		return failure("coordinator_encode", err)
	}
	stderr, err := hostexec.New(4096)
	if err != nil {
		return failure("coordinator_capture", err)
	}
	command := exec.Command(provider.config.CoordinatorCommand[0], provider.config.CoordinatorCommand[1:]...)
	command.Env = append(os.Environ(), "GOMAD3_RUNNER_COORDINATOR=1")
	command.Stdin = bytes.NewReader(input)
	command.Stdout = request.Stdout
	command.Stderr = stderr
	configureCoordinatorCommand(command)
	if err := command.Start(); err != nil {
		return failure("coordinator_start", err)
	}
	waited := make(chan error, 1)
	go func() { waited <- command.Wait() }()
	if err := request.Stdout.Close(); err != nil {
		return backends.ExecutionResult{CoordinatorErrorReason: "coordinator_pipe"}, errors.Join(err, terminateCoordinator(command, waited, request.Deadline))
	}
	deadline := request.Deadline
	reserve := min(250*time.Millisecond, max(time.Until(deadline)/5, time.Nanosecond))
	timer := time.NewTimer(max(time.Until(deadline)-reserve, 0))
	defer timer.Stop()
	poll := time.NewTicker(5 * time.Millisecond)
	defer poll.Stop()
	var waitErr error
	reaped := false
	for {
		if reaped {
			gone, err := coordinatorGroupGone(command)
			if err != nil {
				return backends.ExecutionResult{CoordinatorErrorReason: "coordinator_exit"}, errors.Join(err, terminateCoordinator(command, completedCoordinatorWait(waitErr), deadline))
			}
			if gone {
				result := backends.ExecutionResult{Captured: true, GroupGone: true, PID: command.Process.Pid, PGID: command.Process.Pid, Stderr: backends.Output(stderr.Result()), Termination: backends.TerminationExit}
				if waitErr != nil {
					result.CoordinatorErrorReason = "coordinator_exit"
				}
				return result, waitErr
			}
		}
		select {
		case waitErr = <-waited:
			reaped = true
			waited = nil
		case <-ctx.Done():
			if reaped {
				waited = completedCoordinatorWait(waitErr)
			}
			cleanupErr := terminateCoordinator(command, waited, deadline)
			result := backends.ExecutionResult{Cancelled: true, CoordinatorErrorReason: "coordinator_timeout"}
			if cleanupErr == nil {
				result.Stderr = backends.Output(stderr.Result())
				result.GroupGone = true
			}
			return result, errors.Join(ctx.Err(), cleanupErr)
		case <-timer.C:
			if reaped {
				waited = completedCoordinatorWait(waitErr)
			}
			cleanupErr := terminateCoordinator(command, waited, deadline)
			result := backends.ExecutionResult{WatchdogTimeout: true, CoordinatorErrorReason: "coordinator_timeout"}
			if cleanupErr == nil {
				result.Stderr = backends.Output(stderr.Result())
				result.GroupGone = true
			}
			return result, errors.Join(context.DeadlineExceeded, cleanupErr)
		case <-poll.C:
		}
	}
}

func completedCoordinatorWait(waitErr error) chan error {
	waited := make(chan error, 1)
	waited <- waitErr
	return waited
}
