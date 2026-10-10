package wasi

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"runtime"
	"slices"
	"sync"
	"time"

	"go.temporal.io/server/tools/gomad3/choice"
	"go.temporal.io/server/tools/gomad3/hostexec"
	"go.temporal.io/server/tools/gomad3/hostfs"
)

type EngineIdentity struct {
	Name          string `json:"name"`
	Version       string `json:"version"`
	Configuration string `json:"configuration"`
	HostOS        string `json:"host_os"`
	HostArch      string `json:"host_arch"`
}
type Import struct {
	Module  string   `json:"module"`
	Name    string   `json:"name"`
	Params  []string `json:"params"`
	Results []string `json:"results"`
}
type Export struct {
	Name string `json:"name"`
	Kind string `json:"kind"`
}
type Started struct {
	Schema             string         `json:"schema"`
	Type               string         `json:"type"`
	Engine             EngineIdentity `json:"engine"`
	ModuleSHA256       string         `json:"module_sha256"`
	Imports            []Import       `json:"imports"`
	Exports            []Export       `json:"exports"`
	InitialMemoryPages uint64         `json:"initial_memory_pages"`
}
type Request struct {
	HelperPath, HelperSHA256, ModulePath, ModuleSHA256 string
	Engine                                             EngineIdentity
	Imports                                            []Import
	InitialMemoryPages, MemoryBytes, Fuel              uint64
	Config                                             Config
	Timeout                                            time.Duration
	Runtime                                            *RuntimeControl
}
type Result struct {
	Termination                                        string
	ExitCode                                           *uint32
	Message                                            string
	Stdout, Stderr, Transcript                         []byte
	EnvironmentIdentity                                string
	Started                                            *Started
	Reaped, Cancelled, WatchdogTimeout                 bool
	FuelRemaining, PeakMemoryBytes, OfferedOutputBytes uint64
	ChoiceTrace                                        choice.Trace
	DiagnosticTrace                                    choice.DiagnosticTrace
	ChoiceDivergence                                   *choice.Divergence
	runtimeTerminal                                    [16]byte
}
type executeFrame struct {
	Schema       string `json:"schema"`
	Type         string `json:"type"`
	ModulePath   string `json:"module_path"`
	ModuleSHA256 string `json:"module_sha256"`
	MemoryBytes  uint64 `json:"memory_bytes"`
	Fuel         uint64 `json:"fuel"`
	OutputBytes  uint64 `json:"output_bytes"`
}
type resultFrame struct {
	Type            string  `json:"type"`
	Termination     string  `json:"termination"`
	ExitCode        *uint32 `json:"exit_code,omitempty"`
	Message         string  `json:"message"`
	FuelRemaining   uint64  `json:"fuel_remaining"`
	PeakMemoryBytes uint64  `json:"peak_memory_bytes"`
	OutputBytes     uint64  `json:"output_bytes"`
}

func PinnedEngine() EngineIdentity {
	os := runtime.GOOS
	arch := runtime.GOARCH
	if os == "darwin" {
		os = "macos"
	}
	if arch == "arm64" {
		arch = "aarch64"
	}
	if arch == "amd64" {
		arch = "x86_64"
	}
	return EngineIdentity{"wasmtime", "47.0.3", "cranelift-fuel-nan-canonical-v1", os, arch}
}
func verifyFile(name, digest string) error {
	if !filepath.IsAbs(name) || len(digest) != 64 {
		return invalid("module/helper path or digest")
	}
	decoded, err := hex.DecodeString(digest)
	if err != nil || hex.EncodeToString(decoded) != digest {
		return invalid("module/helper digest")
	}
	data, err := hostfs.ReadBounded(name, 512<<20)
	if err != nil {
		return fmt.Errorf("read execution input: %w", err)
	}
	hash := sha256.Sum256(data)
	if hex.EncodeToString(hash[:]) != digest {
		return invalid("module/helper bytes changed")
	}
	return nil
}
func Run(ctx context.Context, request Request) (Result, error) {
	if request.Timeout <= 0 || request.Engine.Name == "" || request.Engine.Version == "" || request.Engine.Configuration == "" || request.Engine.HostOS == "" || request.Engine.HostArch == "" || request.InitialMemoryPages == 0 || request.MemoryBytes == 0 || request.MemoryBytes > 1<<32 || request.MemoryBytes%65536 != 0 || request.Fuel == 0 {
		return Result{}, invalid("execution configuration")
	}
	environment, err := NewEnvironment(request.Config)
	if err != nil {
		return Result{}, err
	}
	if request.Config.Profile == CooperativeProfile {
		environment.runtime, err = newRuntimeSession(request.Runtime)
		if err != nil {
			return Result{}, err
		}
	} else if request.Runtime != nil {
		return Result{}, invalid("stock profile has cooperative runtime control")
	}
	for _, imported := range request.Imports {
		if imported.Module == "gomad_wasm_v1" && environment.runtime == nil {
			return Result{}, invalid("cooperative import in stock profile")
		}
	}
	if environment.runtime != nil {
		remaining := map[string]bool{"config": true, "decision": true, "observation": true, "finish": true, "idle": true}
		for _, imported := range request.Imports {
			if imported.Module != "gomad_wasm_v1" {
				continue
			}
			if !remaining[imported.Name] || !slices.Equal(imported.Params, []string{"i32", "i32"}) || !slices.Equal(imported.Results, []string{"i32"}) {
				return Result{}, invalid("cooperative runtime import signature")
			}
			delete(remaining, imported.Name)
		}
		if len(remaining) != 0 {
			return Result{}, invalid("incomplete cooperative runtime imports")
		}
	}
	if err := verifyFile(request.HelperPath, request.HelperSHA256); err != nil {
		return Result{}, err
	}
	if err := verifyFile(request.ModulePath, request.ModuleSHA256); err != nil {
		return Result{}, err
	}
	result := Result{EnvironmentIdentity: environment.Identity()}
	if err := ctx.Err(); err != nil {
		result.Termination = "infrastructure"
		result.Message = err.Error()
		result.Cancelled = true
		return result, nil
	}
	execute := executeFrame{"gomad3.wasm-host/v1", "execute", request.ModulePath, request.ModuleSHA256, request.MemoryBytes, request.Fuel, request.Config.Limits.OutputBytes}
	initial, err := json.Marshal(execute)
	if err != nil {
		return Result{}, err
	}
	initial = append(initial, '\n')
	reader, writer := io.Pipe()
	var closeOnce sync.Once
	var closeErr error
	closeInput := func() { closeOnce.Do(func() { closeErr = writer.Close() }) }
	defer closeInput()
	// The same deadline closes the stdin pump and bounds compilation, guest work, and engine teardown.
	executionContext, cancel := context.WithTimeout(ctx, request.Timeout)
	defer cancel()
	finished := make(chan struct{})
	watcherDone := make(chan struct{})
	go func() {
		defer close(watcherDone)
		select {
		case <-executionContext.Done():
			closeInput()
		case <-finished:
		}
	}()
	stream := &frameSink{request: request, environment: environment, input: writer, cancel: cancel, closeInput: closeInput}
	host, hostErr := hostexec.Run(executionContext, hostexec.Request{Command: []string{request.HelperPath}, Dir: filepath.Dir(request.HelperPath), Env: []string{}, Stdin: io.MultiReader(bytes.NewReader(initial), reader), StdoutSink: stream, StdoutDone: closeInput, Timeout: request.Timeout, TerminateGrace: 100 * time.Millisecond, OutputLimit: 64 << 10})
	close(finished)
	<-watcherDone
	closeInput()
	closeErr = errors.Join(closeErr, reader.Close())
	result.Stdout, result.Stderr = environment.Output()
	result.Transcript = environment.Transcript()
	result.Started = stream.started
	result.Reaped = host.GroupGone
	result.Cancelled = ctx.Err() != nil
	result.WatchdogTimeout = !result.Cancelled && (errors.Is(executionContext.Err(), context.DeadlineExceeded) || host.WatchdogTimeout)
	if hostErr != nil || closeErr != nil || !host.GroupGone {
		result.Termination = "infrastructure"
		result.Message = fmt.Sprint(errors.Join(hostErr, closeErr))
		return result, nil
	}
	if stream.err != nil {
		var boundary *BoundaryError
		var divergence *RuntimeReplayError
		if errors.As(stream.err, &divergence) {
			result.Termination = "divergence"
			result.ChoiceDivergence = &divergence.Divergence
		} else if errors.As(stream.err, &boundary) {
			result.Termination = boundary.Kind
		} else {
			result.Termination = "infrastructure"
		}
		result.Message = stream.err.Error()
		return result, nil
	}
	if ctx.Err() != nil || executionContext.Err() != nil || host.Cancelled || host.WatchdogTimeout {
		result.Termination = "infrastructure"
		result.Message = "helper cancelled or exceeded wall watchdog"
		return result, nil
	}
	if host.Termination != hostexec.TerminationExit || host.ExitCode != 0 || stream.terminal == nil || len(stream.buffer) != 0 {
		result.Termination = "infrastructure"
		result.Message = "helper died or ended without a complete terminal frame"
		return result, nil
	}
	terminal := stream.terminal
	result.Termination = terminal.Termination
	result.ExitCode = terminal.ExitCode
	result.Message = terminal.Message
	result.FuelRemaining = terminal.FuelRemaining
	result.PeakMemoryBytes = terminal.PeakMemoryBytes
	result.OfferedOutputBytes = terminal.OutputBytes
	if environment.runtime != nil && terminal.Termination == "exit" {
		result.ChoiceTrace, result.DiagnosticTrace, err = environment.runtime.collect()
		result.runtimeTerminal = environment.runtime.terminal
		if err != nil {
			result.Termination = "invalid"
			result.Message = err.Error()
		}
	}
	return result, nil
}

type frameSink struct {
	request     Request
	environment *Environment
	input       io.Writer
	cancel      context.CancelFunc
	closeInput  func()
	buffer      []byte
	started     *Started
	terminal    *resultFrame
	err         error
}

func (sink *frameSink) Write(data []byte) (int, error) {
	written := len(data)
	if sink.err != nil {
		return written, nil
	}
	for len(data) > 0 {
		count := bytes.IndexByte(data, '\n')
		complete := count >= 0
		if complete {
			count++
		} else {
			count = len(data)
		}
		if len(sink.buffer)+count > frameLimit {
			sink.fail(capacity("protocol-frame"))
			return written, nil
		}
		sink.buffer = append(sink.buffer, data[:count]...)
		data = data[count:]
		if complete {
			if err := sink.frame(sink.buffer); err != nil {
				sink.fail(err)
				return written, nil
			}
			sink.buffer = sink.buffer[:0]
		}
	}
	return written, nil
}
func (sink *frameSink) fail(err error) { sink.err = err; sink.closeInput(); sink.cancel() }
func (sink *frameSink) send(value any) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	data = append(data, '\n')
	if len(data) > frameLimit {
		return capacity("reply-frame")
	}
	written, err := sink.input.Write(data)
	if err != nil {
		return err
	}
	if written != len(data) {
		return io.ErrShortWrite
	}
	return nil
}
func (sink *frameSink) frame(data []byte) error {
	if sink.terminal != nil {
		return invalid("frame after terminal")
	}
	if err := validateJSON(data); err != nil {
		return err
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return invalid("protocol object")
	}
	var kind string
	if err := json.Unmarshal(fields["type"], &kind); err != nil {
		return invalid("protocol type")
	}
	switch kind {
	case "started":
		if sink.started != nil {
			return invalid("duplicate started")
		}
		var started Started
		if err := decodeInput(data, &started, "schema", "type", "engine", "module_sha256", "imports", "exports", "initial_memory_pages"); err != nil {
			return err
		}
		if started.Schema != "gomad3.wasm-host/v1" || started.Engine != sink.request.Engine || started.ModuleSHA256 != sink.request.ModuleSHA256 || started.InitialMemoryPages != sink.request.InitialMemoryPages || !sameImports(started.Imports, sink.request.Imports) {
			return invalid("engine/module/import identity mismatch before authorization")
		}
		memory, start := false, false
		for _, export := range started.Exports {
			if export.Name == "memory" && export.Kind == "memory" {
				memory = true
			}
			if export.Name == "_start" && export.Kind == "func" {
				start = true
			}
		}
		if !memory || !start {
			return invalid("guest exports")
		}
		sink.started = &started
		return sink.send(struct {
			Type          string `json:"type"`
			Module        string `json:"module_sha256"`
			Configuration string `json:"configuration"`
		}{"authorize", started.ModuleSHA256, started.Engine.Configuration})
	case "call":
		if sink.started == nil {
			return invalid("callback before authorization")
		}
		var call Call
		if err := decodeInput(data, &call, "type", "id", "op", "input"); err != nil {
			return err
		}
		if len(call.Op) > 128 {
			return invalid("operation name")
		}
		reply, err := sink.environment.Handle(call)
		if err != nil {
			return err
		}
		return sink.send(reply)
	case "result":
		var result resultFrame
		keys := []string{"type", "termination", "message", "fuel_remaining", "peak_memory_bytes", "output_bytes"}
		if _, found := fields["exit_code"]; found {
			keys = append(keys, "exit_code")
		}
		if err := decodeInput(data, &result, keys...); err != nil {
			return err
		}
		switch result.Termination {
		case "exit":
			if result.ExitCode == nil || sink.started == nil {
				return invalid("exit terminal")
			}
		case "capacity", "unsupported", "invalid", "infrastructure":
			if result.ExitCode != nil {
				return invalid("non-exit code")
			}
		case "trap":
			if result.ExitCode != nil || sink.started == nil {
				return invalid("trap terminal")
			}
		default:
			return invalid("terminal classification")
		}
		if len(result.Message) > 65536 || result.FuelRemaining > sink.request.Fuel || result.PeakMemoryBytes > sink.request.MemoryBytes || result.OutputBytes > sink.request.Config.Limits.OutputBytes || result.OutputBytes != sink.environment.offeredOutput {
			return invalid("terminal limits")
		}
		sink.terminal = &result
		sink.closeInput()
		return nil
	default:
		return invalid("unknown protocol frame")
	}
}
func sameImports(left, right []Import) bool {
	return slices.EqualFunc(left, right, func(a, b Import) bool {
		return a.Module == b.Module && a.Name == b.Name && slices.Equal(a.Params, b.Params) && slices.Equal(a.Results, b.Results)
	})
}
