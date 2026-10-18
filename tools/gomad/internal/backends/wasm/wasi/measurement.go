package wasi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	hostexec "go.temporal.io/server/tools/gomad/internal/host/exec"
)

type GuestProfileConfig = guestProfileConfig

const MaxCooperativeCalls = maxCooperativeCalls

func ValidateMeasurementJSON(data []byte) error { return validateJSON(data) }

func (result *Result) SetHelperStderr(data []byte) { result.helperStderr = data }

func (request *Request) SetRemoteModel(remote bool) { request.remoteModel = remote }

func (request Request) GuestProfile() *GuestProfileConfig { return request.guestProfile }

func (request *Request) SetGuestProfile(profile *GuestProfileConfig) { request.guestProfile = profile }

func (request Request) DiagnosticConfigurationRevision() string {
	return request.diagnosticConfigurationRevision
}

func (request *Request) SetDiagnosticConfigurationRevision(revision string) {
	request.diagnosticConfigurationRevision = revision
}

func (result Result) RuntimeTerminal() [16]byte { return result.runtimeTerminal }

func (result *Result) SetRuntimeTerminal(terminal [16]byte) { result.runtimeTerminal = terminal }

func (result Result) HelperStderr() []byte { return result.helperStderr }

func MeasureTerminalExit(parent context.Context, request Request, mismatch bool) (TerminalExitObservation, error) {
	if err := verifyFile(request.HelperPath, request.HelperSHA256); err != nil {
		return TerminalExitObservation{}, err
	}
	if err := validateInputIdentity(request.ModulePath, request.ModuleSHA256); err != nil {
		return TerminalExitObservation{}, err
	}
	environment, err := NewEnvironment(request.Config)
	if err != nil {
		return TerminalExitObservation{}, err
	}
	if request.Runtime != nil {
		environment.runtime, err = newRuntimeSession(request.Runtime)
		if err != nil {
			return TerminalExitObservation{}, err
		}
	}
	if request.Broker != nil || request.Config.Storage != nil || request.StorageImage != nil || request.Config.NetworkV2 != nil || request.guestProfile != nil || request.diagnosticConfigurationRevision != "" || request.Engine != PinnedEngine() {
		return TerminalExitObservation{}, errors.New("terminal-exit measurement requires the release engine and standalone Temporal model")
	}
	execute := executeFrame{Schema: "gomad.wasm-host/v1", Type: "execute", ModulePath: request.ModulePath, ModuleSHA256: request.ModuleSHA256, MemoryBytes: request.MemoryBytes, Fuel: request.Fuel, OutputBytes: request.Config.Limits.OutputBytes, CompileCache: request.CompileCache}
	var namespaceImage *namespaceImage
	if environment.runtime != nil {
		execute.LocalModel = helperConfig(environment, request.DigestTranscript)
		environment.helperOwned = true
	}
	if helperOwnsNamespace(environment) {
		environment.namespaceOwned = true
		namespaceImage, execute.Namespace, err = declareNamespaceImage(environment)
		if err != nil {
			return TerminalExitObservation{}, err
		}
	}
	environment.mirror = request.StdoutMirror
	initial, err := json.Marshal(execute)
	if err != nil {
		return TerminalExitObservation{}, err
	}
	initial = append(initial, '\n')
	if execute.LocalModel != nil {
		initial, err = appendHelperInputs(initial, environment)
		if err != nil {
			return TerminalExitObservation{}, err
		}
	}
	input, err := newHelperInput()
	if err != nil {
		return TerminalExitObservation{}, err
	}
	input.start(initial)
	ctx, cancel := context.WithTimeout(parent, request.Timeout)
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
	sink := &frameSink{request: request, environment: environment, input: writer, cancel: cancel, closeInput: input.closeInput, context: ctx, namespaceImage: namespaceImage}
	observer := &terminalExitSink{sink: sink}
	host, hostErr := hostexec.Run(ctx, hostexec.Request{Command: []string{request.HelperPath}, Dir: filepath.Dir(request.HelperPath), Env: []string{}, Stdin: input.reader, StdoutSink: observer, StdoutDone: input.closeInput, Timeout: request.Timeout, TerminateGrace: 100 * time.Millisecond, OutputLimit: 64 << 10, CollectResourceUsage: true})
	reapedAt := time.Now()
	close(finished)
	<-watcherDone
	closeErr := input.finish()
	observation := TerminalExitObservation{TerminalReceivedAt: observer.receivedAt, SupervisorReturnedAt: reapedAt, TerminalFrame: observer.terminalFrame, Host: host, AuthorizationChanged: writer.changed}
	if !observer.receivedAt.IsZero() {
		observation.TerminalToReapNanos = uint64(reapedAt.Sub(observer.receivedAt).Nanoseconds())
	}
	result := Result{Started: sink.started, Reaped: host.GroupGone, EnvironmentIdentity: environment.Identity(), AuthorizationToTerminalNanos: sink.authorizationToTerminalNanos}
	result.Stdout, result.Stderr = sink.guestOutput()
	result.GoFilesystemCalls = environment.filesystemServed()
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
	err = errors.Join(hostErr, closeErr, sink.err, ctx.Err())
	if err != nil {
		observation.Error = err.Error()
	}
	return observation, err
}

type TerminalExitObservation struct {
	TerminalReceivedAt, SupervisorReturnedAt time.Time
	TerminalToReapNanos                      uint64
	TerminalFrame                            []byte
	Host                                     hostexec.Result
	Result                                   Result
	AuthorizationChanged                     bool
	Error                                    string
}

func (observation TerminalExitObservation) Within(bound time.Duration) bool {
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

func MeasurementLegacyTranscript(data []byte, config Config) ([]byte, error) {
	environment, err := NewEnvironment(config)
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, nil
	}
	if uint64(len(data)) > environment.config.Limits.TranscriptBytes || data[len(data)-1] != '\n' {
		return nil, fmt.Errorf("historical transcript bounds or framing")
	}
	rows := bytes.Split(data[:len(data)-1], []byte{'\n'})
	if len(rows)%2 != 0 {
		return nil, fmt.Errorf("unpaired historical transcript")
	}
	var previous uint64
	i := 0
	for ; i < len(rows); i += 2 {
		if len(rows[i]) > frameLimit || len(rows[i+1]) > frameLimit {
			return nil, fmt.Errorf("historical frame bound")
		}
		var call Call
		err := decodeInput(rows[i], &call, "type", "id", "op", "input")
		if err != nil {
			return nil, err
		}
		frame, err := newValidatedFrame(rows[i+1])
		if err != nil {
			return nil, err
		}
		keys := []string{"type", "id", "errno", "output"}
		for _, optional := range []string{"unsupported", "message"} {
			_, ok := frame.fields[optional]
			if ok {
				keys = append(keys, optional)
			}
		}
		var reply Reply
		err = frame.decode(&reply, keys...)
		if err != nil {
			return nil, err
		}
		if call.Type != "call" || reply.Type != "reply" || call.ID == 0 || call.ID <= previous || call.ID != reply.ID || strings.HasPrefix(call.Op, "runtime_") || !reply.Unsupported && reply.Message != "" {
			return nil, fmt.Errorf("historical call/reply identity or error evidence")
		}
		previous = call.ID
		tag, input, err := transcriptInput(call, environment.entropyOffset)
		if err != nil {
			return nil, err
		}
		if reply.Unsupported || reply.Errno != 0 {
			err := decodeInput(reply.Output, &struct{}{})
			if err != nil {
				return nil, err
			}
		} else if tag >= 3 && tag <= 7 {
			output, errno, message, err := environment.dispatch(call, nil)
			if err != nil {
				return nil, err
			}
			if errno != 0 || message != "" {
				return nil, fmt.Errorf("historical derived output owner refused")
			}
			keys := []string{"values"}
			switch tag {
			case 3, 5:
				keys = []string{"count", "bytes"}
			case 7:
				keys = []string{"data_base64"}
			}
			decoded := reflect.New(reflect.TypeOf(output))
			err = decodeInput(reply.Output, decoded.Interface(), keys...)
			if err != nil {
				return nil, err
			}
			if !reflect.DeepEqual(output, decoded.Elem().Interface()) {
				return nil, fmt.Errorf("historical derived output differs from captured inputs")
			}
		}
		if tag == 1 && reply.Errno == 0 && !reply.Unsupported {
			var clock struct {
				ID        uint32 `json:"clock_id"`
				Precision uint64 `json:"precision"`
			}
			var output struct {
				Timestamp uint64 `json:"timestamp"`
			}
			err := decodeInput(call.Input, &clock, "clock_id", "precision")
			if err != nil {
				return nil, err
			}
			err = decodeInput(reply.Output, &output, "timestamp")
			if err != nil {
				return nil, err
			}
			if clock.ID > 1 || clock.ID == 0 && output.Timestamp < config.Clock.EpochNanos {
				return nil, fmt.Errorf("historical clock domain")
			}
			now := output.Timestamp
			if clock.ID == 0 {
				now -= config.Clock.EpochNanos
			}
			if now > math.MaxInt64 {
				return nil, fmt.Errorf("historical clock overflow")
			}
			environment.appendClockRead(call.ID, clock.ID, clock.Precision, now)
			continue
		}
		record, err := encodeTranscriptRecord(tag, call.ID, input, reply)
		if err != nil {
			return nil, err
		}
		environment.appendTranscript(record)
	}
	converted := environment.Transcript()
	_, err = DecodeTranscript(converted)
	if err != nil {
		return nil, err
	}
	return converted, nil
}
