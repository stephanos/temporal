package wasi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	hostexec "go.temporal.io/server/tools/gomad3/internal/host/exec"
)

func TestTerminalExitMeasurementControls(t *testing.T) {
	for _, test := range []struct {
		name, before, terminal, after string
		want                          bool
	}{
		{"prompt-exit", "", terminalExitSuccess, "", true},
		{"slow-preparation", "/bin/sleep 1.1\n", terminalExitSuccess, "", true},
		{"delayed-exit", "", terminalExitSuccess, "/bin/sleep 1.1\n", false},
		{"missing-terminal", "", "", "", false},
		{"invalid-terminal", "", `{"type":"result","termination":"exit"}`, "", false},
		{"helper-death", "", terminalExitSuccess, "kill -KILL $$\n", false},
		{"post-terminal-watchdog", "", terminalExitSuccess, "/bin/sleep 30\n", false},
	} {
		t.Run(test.name, func(t *testing.T) {
			request := terminalExitSyntheticRequest(t)
			if test.name == "post-terminal-watchdog" {
				request.Timeout = 200 * time.Millisecond
			}
			script := "#!/bin/sh\nread request\n" + test.before + "printf '%s\\n' '" + startedJSON(t, request) + "'\nread authorization\n"
			if test.terminal != "" {
				script += "printf '%s\\n' '" + test.terminal + "'\n"
			}
			script += test.after
			if err := os.WriteFile(request.HelperPath, []byte(script), 0700); err != nil {
				t.Fatal(err)
			}
			request.HelperSHA256 = fileHash(t, request.HelperPath)
			observation := measureTerminalExit(t, request, false)
			if got := observation.within(time.Second); got != test.want {
				t.Fatalf("within bound = %v, want %v: %+v", got, test.want, observation)
			}
			if !observation.Host.GroupGone {
				t.Fatalf("helper was not reaped: %+v", observation)
			}
			t.Logf("terminal-to-reap=%dns accepted=%v watchdog=%v", observation.TerminalToReapNanos, !observation.TerminalReceivedAt.IsZero(), observation.Host.WatchdogTimeout)
			if test.name == "delayed-exit" && observation.TerminalToReapNanos < uint64(time.Second) {
				t.Fatalf("post-terminal exit delay missing: %+v", observation)
			}
		})
	}
}

func TestTerminalExitObserverReceivesCompleteAcceptedFrameBeforeValidation(t *testing.T) {
	request := terminalExitSyntheticRequest(t)
	environment, err := NewEnvironment(request.Config)
	if err != nil {
		t.Fatal(err)
	}
	var validationFinished time.Time
	sink := &frameSink{request: request, environment: environment, input: &bytes.Buffer{}, cancel: func() {}, closeInput: func() {
		<-time.After(20 * time.Millisecond)
		validationFinished = time.Now()
	}}
	observer := &terminalExitSink{sink: sink}
	started := startedJSON(t, request)
	if _, err := observer.Write([]byte(started + "\n" + terminalExitSuccess)); err != nil {
		t.Fatal(err)
	}
	if !observer.receivedAt.IsZero() {
		t.Fatal("partial terminal frame started the clock")
	}
	if _, err := observer.Write([]byte("\n")); err != nil {
		t.Fatal(err)
	}
	if observer.receivedAt.IsZero() || validationFinished.Sub(observer.receivedAt) < 15*time.Millisecond {
		t.Fatal("clock started after terminal validation")
	}
	receivedAt := observer.receivedAt
	if _, err := observer.Write([]byte(terminalExitSuccess + "\n")); err != nil {
		t.Fatal(err)
	}
	if sink.err == nil || !observer.receivedAt.Equal(receivedAt) {
		t.Fatal("duplicate result changed terminal receipt or escaped real transport rejection")
	}
}

func TestTerminalExitAuthorizationMismatchAfterStarted(t *testing.T) {
	request := terminalExitSyntheticRequest(t)
	script := "#!/bin/sh\nread request\nprintf '%s\\n' '" + startedJSON(t, request) + "'\nread authorization\ncase \"$authorization\" in\n*intentional-mismatch*) printf '%s\\n' '{\"type\":\"result\",\"termination\":\"invalid\",\"message\":\"execution authorization identity mismatch\",\"fuel_remaining\":0,\"peak_memory_bytes\":0,\"output_bytes\":0}' ;;\n*) exit 9 ;;\nesac\n"
	if err := os.WriteFile(request.HelperPath, []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	request.HelperSHA256 = fileHash(t, request.HelperPath)
	observation := measureTerminalExit(t, request, true)
	if !observation.within(time.Second) || !observation.AuthorizationChanged || observation.Result.Started == nil || observation.Result.Termination != "invalid" || observation.Result.Message != "execution authorization identity mismatch" {
		t.Fatalf("post-load authorization control: %+v", observation)
	}
}

const terminalExitSuccess = `{"type":"result","termination":"exit","exit_code":0,"message":"","fuel_remaining":0,"peak_memory_bytes":0,"output_bytes":0}`

func terminalExitSyntheticRequest(t *testing.T) Request {
	t.Helper()
	root := t.TempDir()
	module := filepath.Join(root, "guest.wat")
	if err := os.WriteFile(module, []byte(`(module (memory (export "memory") 1) (func (export "_start")))`), 0600); err != nil {
		t.Fatal(err)
	}
	return Request{HelperPath: filepath.Join(root, "helper"), ModulePath: module, ModuleSHA256: fileHash(t, module), Engine: PinnedEngine(), InitialMemoryPages: 1, MemoryBytes: 128 << 20, Fuel: 100000000, Config: testConfig(), Timeout: 5 * time.Second}
}

func TestTemporalTerminalToProcessExit(t *testing.T) {
	root := os.Getenv("GOMAD_WASM_TERMINAL_EXIT_ROOT")
	if root == "" {
		t.Skip("set GOMAD_WASM_TERMINAL_EXIT_ROOT for release-helper Temporal terminal-to-reap measurements")
	}
	if !filepath.IsAbs(root) {
		t.Fatal("terminal-exit evidence root must be absolute")
	}
	if !filepath.IsAbs(os.Getenv("GOMAD_WASM_TEST_HELPER")) {
		t.Fatal("GOMAD_WASM_TEST_HELPER must explicitly name the candidate release helper")
	}
	evidence, err := diagnosticRunRoot(root, "terminal-exit-")
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("GOMAD_WASM_PROTOTYPE_ROOT", evidence)
	repo, err := filepath.Abs("../../../../../..")
	if err != nil {
		t.Fatal(err)
	}
	engineSource := filepath.Join(repo, "tools/gomad3/internal/backends/wasm/wasmhost/src/engine.rs")
	engineBytes, err := os.ReadFile(engineSource)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(engineBytes, []byte("std::mem::forget")) || !bytes.Contains(engineBytes, []byte("native_unwind_info(false)")) {
		t.Fatal("measure a release helper built from the no-unwind candidate without the module/store leak")
	}
	engineSHA256 := fileHash(t, engineSource)
	if os.Getenv("GOMAD_WASM_TERMINAL_EXIT_ENGINE_SHA256") != engineSHA256 {
		t.Fatal("GOMAD_WASM_TERMINAL_EXIT_ENGINE_SHA256 must bind the engine source recorded by the release-helper build")
	}
	module := os.Getenv("GOMAD_WASM_TERMINAL_EXIT_MODULE")
	if !filepath.IsAbs(module) {
		t.Fatal("GOMAD_WASM_TERMINAL_EXIT_MODULE must name the current ./tests cooperative module")
	}
	config := prototypeTemporalConfig(t, repo, "/workspace/tests", "-test.run=^TestWorkflowTypeEncodingSuite$/^TestPlainASCII$/^Succeeds$", "-persistenceType=sql", "-persistenceDriver=sqlite")
	request := prototypeRequest(t, module, config, 0, 30*time.Minute)
	var observations []terminalExitObservation
	failed := false
	for _, mismatch := range []bool{false, true} {
		name := "success"
		if mismatch {
			name = "post-load-authorization-mismatch"
		}
		observation := measureTerminalExit(t, request, mismatch)
		observations = append(observations, observation)
		t.Logf("%s terminal-to-reap=%dns termination=%s reaped=%v error=%s", name, observation.TerminalToReapNanos, observation.Result.Termination, observation.Host.GroupGone, observation.Error)
		temporalWrite(t, filepath.Join(evidence, name+"-helper-stderr.txt"), observation.Host.Stderr.RawBytes)
		if !observation.within(time.Second) {
			failed = true
		}
		if mismatch {
			if !observation.AuthorizationChanged || observation.Result.Termination != "invalid" || observation.Result.Message != "execution authorization identity mismatch" || observation.Result.Started == nil || observation.Result.Telemetry == nil || observation.Result.Telemetry.InstantiateNanos != 0 || observation.Result.Telemetry.ExecuteNanos != 0 {
				failed = true
			}
		} else if observation.Result.Termination != "exit" || observation.Result.ExitCode == nil || *observation.Result.ExitCode != 0 || !strings.Contains(string(observation.Result.Stdout), "--- PASS: TestWorkflowTypeEncodingSuite") || !strings.Contains(string(observation.Result.Stdout), "TestPlainASCII/Succeeds") {
			failed = true
		}
	}
	info, err := os.Stat(module)
	if err != nil {
		t.Fatal(err)
	}
	temporalJSON(t, filepath.Join(evidence, "terminal-exit.json"), map[string]any{
		"schema": "gomad3.wasm-terminal-exit-measurement/v1", "host": PinnedEngine(), "bound_nanos": uint64(time.Second),
		"module_sha256": request.ModuleSHA256, "module_bytes": info.Size(), "helper_sha256": request.HelperSHA256,
		"engine_source_sha256": engineSHA256,
		"selection":            config.Args, "observations": observations,
		"boundary": "receipt of complete actual validated terminal frame through completed host supervisor wait, capture drain and group reaping; conservative upper bound on exit time",
	})
	if failed {
		t.Fatalf("Temporal terminal-to-process-exit proof failed; raw measurements retained in %s", evidence)
	}
}

func measureTerminalExit(t *testing.T, request Request, mismatch bool) terminalExitObservation {
	t.Helper()
	if err := verifyFile(request.HelperPath, request.HelperSHA256); err != nil {
		t.Fatal(err)
	}
	if err := validateInputIdentity(request.ModulePath, request.ModuleSHA256); err != nil {
		t.Fatal(err)
	}
	environment, err := NewEnvironment(request.Config)
	if err != nil {
		t.Fatal(err)
	}
	if request.Runtime != nil {
		environment.runtime, err = newRuntimeSession(request.Runtime)
		if err != nil {
			t.Fatal(err)
		}
	}
	if request.Broker != nil || request.Config.Storage != nil || request.StorageImage != nil || request.Config.NetworkV2 != nil || request.guestProfile != nil || request.diagnosticConfigurationRevision != "" || request.Engine != PinnedEngine() {
		t.Fatal("terminal-exit measurement requires the release engine and standalone Temporal model")
	}
	execute := executeFrame{Schema: "gomad3.wasm-host/v1", Type: "execute", ModulePath: request.ModulePath, ModuleSHA256: request.ModuleSHA256, MemoryBytes: request.MemoryBytes, Fuel: request.Fuel, OutputBytes: request.Config.Limits.OutputBytes, CompileCache: request.CompileCache}
	if environment.runtime != nil {
		execute.LocalModel = helperConfig(environment, request.DigestTranscript)
		environment.helperOwned = true
	}
	initial, err := json.Marshal(execute)
	if err != nil {
		t.Fatal(err)
	}
	input, err := newHelperInput()
	if err != nil {
		t.Fatal(err)
	}
	input.start(append(initial, '\n'))
	ctx, cancel := context.WithTimeout(context.Background(), request.Timeout)
	defer cancel()
	finished, watcherDone := make(chan struct{}), make(chan struct{})
	go func() {
		defer close(watcherDone)
		select {
		case <-ctx.Done():
			input.closeInput()
		case <-finished:
		}
	}()
	writer := &terminalExitAuthorization{Writer: input, mismatch: mismatch}
	sink := &frameSink{request: request, environment: environment, input: writer, cancel: cancel, closeInput: input.closeInput, context: ctx}
	observer := &terminalExitSink{sink: sink}
	host, hostErr := hostexec.Run(ctx, hostexec.Request{Command: []string{request.HelperPath}, Dir: filepath.Dir(request.HelperPath), Env: []string{}, Stdin: input.reader, StdoutSink: observer, StdoutDone: input.closeInput, Timeout: request.Timeout, TerminateGrace: 100 * time.Millisecond, OutputLimit: 64 << 10, CollectResourceUsage: true})
	reapedAt := time.Now()
	close(finished)
	<-watcherDone
	closeErr := input.finish()
	observation := terminalExitObservation{TerminalReceivedAt: observer.receivedAt, SupervisorReturnedAt: reapedAt, TerminalFrame: observer.terminalFrame, Host: host, AuthorizationChanged: writer.changed}
	if !observer.receivedAt.IsZero() {
		observation.TerminalToReapNanos = uint64(reapedAt.Sub(observer.receivedAt).Nanoseconds())
	}
	result := Result{Started: sink.started, Reaped: host.GroupGone, EnvironmentIdentity: environment.Identity(), AuthorizationToTerminalNanos: sink.authorizationToTerminalNanos}
	result.Stdout, result.Stderr = environment.Output()
	result.TranscriptSHA256 = environment.TranscriptSHA256()
	if terminal := sink.terminal; terminal != nil {
		result.Termination, result.ExitCode, result.Message, result.Telemetry = terminal.Termination, terminal.ExitCode, terminal.Message, terminal.Metrics
		if terminal.Termination == "exit" && environment.helperOwned {
			if err := sink.collectHelper(&result); err != nil {
				hostErr = errors.Join(hostErr, err)
			}
		}
	}
	observation.Result = result
	if err := errors.Join(hostErr, closeErr, sink.err, ctx.Err()); err != nil {
		observation.Error = err.Error()
	}
	return observation
}

type terminalExitObservation struct {
	TerminalReceivedAt, SupervisorReturnedAt time.Time
	TerminalToReapNanos                      uint64
	TerminalFrame                            []byte
	Host                                     hostexec.Result
	Result                                   Result
	AuthorizationChanged                     bool
	Error                                    string
}

func (observation terminalExitObservation) within(bound time.Duration) bool {
	return observation.Error == "" && !observation.TerminalReceivedAt.IsZero() && observation.SupervisorReturnedAt.After(observation.TerminalReceivedAt) && observation.TerminalToReapNanos <= uint64(bound) && observation.Host.Termination == hostexec.TerminationExit && observation.Host.ExitCode == 0 && observation.Host.GroupGone && !observation.Host.Cancelled && !observation.Host.WatchdogTimeout
}

type terminalExitSink struct {
	sink          *frameSink
	receivedAt    time.Time
	terminalFrame []byte
}

func (observer *terminalExitSink) Write(data []byte) (int, error) {
	written := len(data)
	for len(data) > 0 {
		count := bytes.IndexByte(data, '\n')
		if count < 0 {
			count = len(data)
		} else {
			count++
		}
		frameBytes := len(observer.sink.buffer) + count
		receivedAt := time.Now()
		if _, err := observer.sink.Write(data[:count]); err != nil {
			return 0, err
		}
		if observer.receivedAt.IsZero() && observer.sink.terminal != nil && observer.sink.err == nil {
			observer.receivedAt = receivedAt
			observer.terminalFrame = bytes.Clone(observer.sink.buffer[:frameBytes])
		}
		data = data[count:]
	}
	return written, nil
}

type terminalExitAuthorization struct {
	io.Writer
	mismatch, changed bool
}

func (writer *terminalExitAuthorization) Write(data []byte) (int, error) {
	if !writer.mismatch {
		return writer.Writer.Write(data)
	}
	var frame struct {
		Type          string `json:"type"`
		ModuleSHA256  string `json:"module_sha256"`
		Configuration string `json:"configuration"`
	}
	if err := json.Unmarshal(data, &frame); err != nil {
		return 0, err
	}
	if frame.Type != "authorize" {
		return writer.Writer.Write(data)
	}
	frame.Configuration += "-intentional-mismatch"
	changed, err := json.Marshal(frame)
	if err != nil {
		return 0, err
	}
	changed = append(changed, '\n')
	count, err := writer.Writer.Write(changed)
	if err != nil {
		return 0, err
	}
	if count != len(changed) {
		return 0, io.ErrShortWrite
	}
	writer.changed = true
	return len(data), nil
}
