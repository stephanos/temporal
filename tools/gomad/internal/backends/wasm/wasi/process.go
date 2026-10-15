package wasi

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"sync"
	"time"

	"go.temporal.io/server/tools/gomad/internal/backends/target"
	hostexec "go.temporal.io/server/tools/gomad/internal/host/exec"
	hostfs "go.temporal.io/server/tools/gomad/internal/host/fs"
	"go.temporal.io/server/tools/gomad/internal/model/choice"
	"go.temporal.io/server/tools/gomad/internal/model/volume"
)

type helperInput struct {
	reader, writer *os.File
	ready          chan struct{}
	bootstrapErr   error
	closeOnce      sync.Once
	closeErr       error
	finishOnce     sync.Once
	finishErr      error
}

func newHelperInput() (*helperInput, error) {
	reader, writer, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	return &helperInput{reader: reader, writer: writer, ready: make(chan struct{})}, nil
}

func (input *helperInput) start(initial []byte) {
	go func() {
		written, err := input.writer.Write(initial)
		if err == nil && written != len(initial) {
			err = io.ErrShortWrite
		}
		input.bootstrapErr = helperInputError(err)
		close(input.ready)
	}()
}

func (input *helperInput) Write(data []byte) (int, error) {
	<-input.ready
	if input.bootstrapErr != nil {
		return 0, input.bootstrapErr
	}
	written, err := input.writer.Write(data)
	return written, helperInputError(err)
}

func helperInputError(err error) error {
	if errors.Is(err, os.ErrClosed) {
		return io.ErrClosedPipe
	}
	return err
}

func (input *helperInput) closeInput() {
	input.closeOnce.Do(func() { input.closeErr = input.writer.Close() })
}

func (input *helperInput) finish() error {
	input.finishOnce.Do(func() {
		input.closeInput()
		<-input.ready
		input.finishErr = errors.Join(input.closeErr, input.reader.Close())
		if input.bootstrapErr != io.ErrClosedPipe {
			input.finishErr = errors.Join(input.bootstrapErr, input.finishErr)
		}
	})
	return input.finishErr
}

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
	WorkloadKind                                       target.Kind `json:",omitempty"`
	HelperPath, HelperSHA256, ModulePath, ModuleSHA256 string
	Engine                                             EngineIdentity
	Imports                                            []Import
	InitialMemoryPages, MemoryBytes, Fuel              uint64
	Config                                             Config
	Timeout                                            time.Duration
	Runtime                                            *RuntimeControl
	StorageImage                                       *StorageImage
	StorageCrashSelection                              map[string][]uint64
	StorageCrashLimits                                 *volume.CrashEnumerationLimits
	// ReturnStorageImage asks a helper-owned storage run to return the
	// recovery image blob; its identity is always returned.
	ReturnStorageImage bool
	Broker             *BrokerSession
	// StdoutMirror receives a live copy of guest stdout for progress observation.
	StdoutMirror io.Writer
	// CompileCache names a host-local directory where the helper keeps compiled
	// modules across fresh executions. It is an optimization, never evidence.
	CompileCache string
	// DigestTranscript retains only the WASI transcript's SHA-256, bounding host
	// memory for whole-program guests; Result.Transcript is then empty.
	DigestTranscript                bool
	remoteModel                     bool
	diagnosticConfigurationRevision string
	guestProfile                    *guestProfileConfig
}
type guestProfileConfig struct {
	Path           string `json:"path"`
	IntervalMicros uint64 `json:"interval_micros"`
	MaxSamples     uint64 `json:"max_samples"`
}
type Result struct {
	Termination                                        string
	ExitCode                                           *uint32
	Message                                            string
	Stdout, Stderr, Transcript, NetworkTranscript      []byte
	EnvironmentIdentity                                string
	Started                                            *Started
	Reaped, Cancelled, WatchdogTimeout                 bool
	FuelRemaining, PeakMemoryBytes, OfferedOutputBytes uint64
	Calls                                              uint64
	GoFilesystemCalls                                  map[string]uint64
	PartialHelperTranscript, PartialHelperChoice       []byte
	PartialHelperDiagnostic                            []byte
	TranscriptSHA256                                   [32]byte
	TimeAdvances                                       uint64
	ChoiceTrace                                        choice.Trace
	DiagnosticTrace                                    choice.DiagnosticTrace
	ChoiceDivergence                                   *choice.Divergence
	RuntimeInterrupted                                 bool
	InterruptedChoiceBytes, InterruptedDiagnosticBytes []byte
	VirtualTimeElapsedNanos                            uint64
	StorageImage                                       *StorageImage
	StorageTranscript                                  []byte
	StorageSnapshots                                   []volume.VolumeSnapshot
	StorageOperations                                  map[string][]volume.Operation
	StorageCrashEnumeration                            map[string]volume.CrashEnumeration
	Telemetry                                          *Telemetry
	HelperResourceUsage                                *hostexec.ResourceUsage
	// AuthorizationToTerminalNanos spans authorization through the validated
	// terminal frame, including fresh instantiation and evidence transfer.
	AuthorizationToTerminalNanos uint64
	runtimeTerminal              [16]byte
	helperStderr                 []byte
}

// Telemetry describes real helper costs, never inputs to modeled execution.
// CPU and RSS describe the entire helper process, not guest-only execution.
type Telemetry struct {
	PrepareNanos           uint64 `json:"prepare_nanos"`
	CompileNanos           uint64 `json:"compile_nanos"`
	CacheLoadNanos         uint64 `json:"cache_load_nanos"`
	CacheWriteNanos        uint64 `json:"cache_write_nanos"`
	CacheHit               bool   `json:"cache_hit"`
	InstantiateNanos       uint64 `json:"instantiate_nanos"`
	ExecuteNanos           uint64 `json:"execute_nanos"`
	ProcessUserNanos       uint64 `json:"process_user_nanos"`
	ProcessSystemNanos     uint64 `json:"process_system_nanos"`
	PeakRSSBytes           uint64 `json:"peak_rss_bytes"`
	ResourceUsageAvailable bool   `json:"resource_usage_available"`
}
type executeFrame struct {
	Schema       string                `json:"schema"`
	Type         string                `json:"type"`
	ModulePath   string                `json:"module_path"`
	ModuleSHA256 string                `json:"module_sha256"`
	MemoryBytes  uint64                `json:"memory_bytes"`
	Fuel         uint64                `json:"fuel"`
	OutputBytes  uint64                `json:"output_bytes"`
	CompileCache string                `json:"compile_cache,omitempty"`
	LocalModel   *helperModelConfig    `json:"local_model,omitempty"`
	Namespace    *namespaceDeclaration `json:"namespace,omitempty"`
	Storage      *helperStorageRequest `json:"storage,omitempty"`
	GuestProfile *guestProfileConfig   `json:"guest_profile,omitempty"`
}
type resultFrame struct {
	Type            string             `json:"type"`
	Termination     string             `json:"termination"`
	ExitCode        *uint32            `json:"exit_code,omitempty"`
	Message         string             `json:"message"`
	FuelRemaining   uint64             `json:"fuel_remaining"`
	PeakMemoryBytes uint64             `json:"peak_memory_bytes"`
	OutputBytes     uint64             `json:"output_bytes"`
	LocalModel      *helperModelResult `json:"local_model,omitempty"`
	Divergence      *choice.Divergence `json:"divergence,omitempty"`
	Metrics         *Telemetry         `json:"metrics,omitempty"`
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
	return EngineIdentity{"wasmtime", "47.0.3", "cranelift-fuel-nan-canonical-nounwind-v2", os, arch}
}
func validateInputIdentity(name, digest string) error {
	if !filepath.IsAbs(name) || len(digest) != 64 {
		return invalid("module/helper path or digest")
	}
	decoded, err := hex.DecodeString(digest)
	if err != nil || hex.EncodeToString(decoded) != digest {
		return invalid("module/helper digest")
	}
	return nil
}
func verifyFile(name, digest string) error {
	if err := validateInputIdentity(name, digest); err != nil {
		return err
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
func Run(ctx context.Context, request Request) (result Result, runErr error) {
	pinned := PinnedEngine()
	diagnostic := func(revision string) bool {
		return revision == pinned.Configuration+"-guest-profile" || revision == pinned.Configuration+"-fuel-off" || revision == pinned.Configuration+"-guest-profile-fuel-off"
	}
	if request.diagnosticConfigurationRevision != "" {
		if request.Engine != pinned || !diagnostic(request.diagnosticConfigurationRevision) {
			return Result{}, invalid("diagnostic engine configuration")
		}
		request.Engine.Configuration = request.diagnosticConfigurationRevision
	} else if diagnostic(request.Engine.Configuration) {
		return Result{}, invalid("diagnostic engine configuration requires test opt-in")
	}
	if profile := request.guestProfile; profile != nil {
		if (request.diagnosticConfigurationRevision != pinned.Configuration+"-guest-profile" && request.diagnosticConfigurationRevision != pinned.Configuration+"-guest-profile-fuel-off") || !filepath.IsAbs(profile.Path) || profile.IntervalMicros == 0 || profile.IntervalMicros > 1000000 || profile.MaxSamples == 0 || profile.MaxSamples > 100000 {
			return Result{}, invalid("guest profile configuration")
		}
	}
	if request.Config.Storage == nil && (request.StorageCrashSelection != nil || request.StorageCrashLimits != nil) {
		return Result{}, invalid("storage crash controls without storage model")
	}
	for id := range request.StorageCrashSelection {
		if !slices.ContainsFunc(request.Config.Storage.Volumes, func(config volume.VolumeConfig) bool { return config.ID == id }) {
			return Result{}, invalid("storage crash selection volume")
		}
	}
	if request.Broker != nil {
		defer func() {
			if err := request.Broker.Finished(); err != nil {
				result.Termination = "infrastructure"
				result.Message = errors.Join(errors.New(result.Message), err).Error()
			}
		}()
	}
	if request.Timeout <= 0 || request.Engine.Name == "" || request.Engine.Version == "" || request.Engine.Configuration == "" || request.Engine.HostOS == "" || request.Engine.HostArch == "" || request.InitialMemoryPages == 0 || request.MemoryBytes == 0 || request.MemoryBytes > 1<<32 || request.MemoryBytes%65536 != 0 || request.Fuel == 0 {
		return Result{}, invalid("execution configuration")
	}
	var environment *Environment
	var err error
	if request.StorageImage != nil {
		image := request.StorageImage
		if helperStorageRun(request) && image.data != nil {
			// The helper loads the recovery blob; the Go namespace stays empty.
			image = &StorageImage{data: image.data, identity: image.identity, model: image.model, occurrences: image.occurrences}
		}
		environment, err = NewRecoveredEnvironment(request.Config, image)
	} else {
		environment, err = NewEnvironment(request.Config)
	}
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
		if (imported.Module == "gomad_wasm_v1" || imported.Module == "gomad_wasm_v2") && environment.runtime == nil {
			return Result{}, invalid("cooperative import in stock profile")
		}
	}
	if environment.runtime != nil {
		remaining := map[string]bool{"config": true, "decision": true, "observation": true, "finish": true, "idle": true}
		for _, imported := range request.Imports {
			if imported.Module == "gomad_wasm_v2" {
				if imported.Name != "net_stream" || !slices.Equal(imported.Params, []string{"i32", "i32"}) || !slices.Equal(imported.Results, []string{"i32"}) {
					return Result{}, invalid("network v2 import signature")
				}
				continue
			}
			if imported.Module != "gomad_wasm_v1" {
				continue
			}
			if !remaining[imported.Name] {
				return Result{}, invalid("cooperative runtime import signature")
			}
			if !slices.Equal(imported.Params, []string{"i32", "i32"}) || !slices.Equal(imported.Results, []string{"i32"}) {
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
	if err := validateInputIdentity(request.ModulePath, request.ModuleSHA256); err != nil {
		return Result{}, err
	}
	environment.mirror = request.StdoutMirror
	if request.DigestTranscript {
		environment.digestOnly, environment.transcriptHash = true, sha256.New()
	}
	result = Result{EnvironmentIdentity: environment.Identity()}
	if err := ctx.Err(); err != nil {
		result.Termination = "infrastructure"
		result.Message = err.Error()
		result.Cancelled = true
		return result, nil
	}
	if request.CompileCache != "" && !filepath.IsAbs(request.CompileCache) {
		return Result{}, invalid("compile cache directory")
	}
	execute := executeFrame{Schema: "gomad3.wasm-host/v1", Type: "execute", ModulePath: request.ModulePath, ModuleSHA256: request.ModuleSHA256, MemoryBytes: request.MemoryBytes, Fuel: request.Fuel, OutputBytes: request.Config.Limits.OutputBytes, CompileCache: request.CompileCache, GuestProfile: request.guestProfile}
	switch ownership(request) {
	case helperOwnership:
		execute.LocalModel = helperConfig(environment, request.DigestTranscript)
		environment.helperOwned = true
	case splitOwnership:
		// Broker storage runs split ownership: the broker keeps the
		// runtime session and clock, the helper the namespace and volume.
		execute.LocalModel = helperConfig(environment, request.DigestTranscript)
		execute.LocalModel.Mode, execute.LocalModel.TapeChunks, execute.LocalModel.Diagnostics, execute.LocalModel.Split = choice.ModeRecord, 0, false, true
		environment.splitOwned = true
		request.Broker.localCalls = true
		// The broker may restart any incarnation, so split runs always
		// return the recovery image blob.
		request.ReturnStorageImage = true
	}
	var namespaceImage *namespaceImage
	if helperOwnsNamespace(environment) {
		environment.namespaceOwned = true
		namespaceImage, execute.Namespace, err = declareNamespaceImage(environment)
		if err != nil {
			return Result{}, err
		}
	}
	var storageImage []byte
	if environment.namespaceOwned && request.Config.Storage != nil {
		execute.Storage, storageImage, err = helperStorageConfig(environment, request)
		if err != nil {
			return Result{}, err
		}
	} else if recovery := environment.storageRecovery; recovery != nil && recovery.fs == nil {
		return Result{}, invalid("durable storage image")
	}
	initial, err := json.Marshal(execute)
	if err != nil {
		return Result{}, err
	}
	initial = append(initial, '\n')
	if execute.LocalModel != nil {
		initial, err = appendHelperInputs(initial, environment)
		if err != nil {
			return Result{}, err
		}
	}
	// Preparation waits share the execution watchdog, without sharing a Store.
	executionContext, cancel := context.WithTimeout(ctx, request.Timeout)
	defer cancel()
	if request.Broker != nil {
		if err := request.Broker.Prepare(executionContext); err != nil {
			result.Termination, result.Message = "infrastructure", err.Error()
			result.Cancelled = ctx.Err() != nil
			result.WatchdogTimeout = !result.Cancelled && errors.Is(executionContext.Err(), context.DeadlineExceeded)
			return result, nil
		}
		defer request.Broker.Prepared()
	}
	input, err := newHelperInput()
	if err != nil {
		return Result{}, err
	}
	input.start(initial)
	closeInput := input.closeInput
	// The same deadline closes the stdin pump and bounds compilation, guest work, and engine teardown.
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
	stream := &frameSink{request: request, environment: environment, input: input, cancel: cancel, closeInput: closeInput, context: executionContext, namespaceImage: namespaceImage, storageImage: storageImage}
	host, hostErr := hostexec.Run(executionContext, hostexec.Request{Command: []string{request.HelperPath}, Dir: filepath.Dir(request.HelperPath), Env: []string{}, Stdin: input.reader, StdoutSink: stream, StdoutDone: closeInput, Timeout: request.Timeout, TerminateGrace: 100 * time.Millisecond, OutputLimit: 64 << 10, CollectResourceUsage: true})
	close(finished)
	<-watcherDone
	closeErr := input.finish()
	defer func() {
		if environment.namespaceOwned && result.Termination != "exit" {
			stream.retainPartialHelperEvidence(&result)
		}
	}()
	result.HelperResourceUsage = host.ResourceUsage
	result.helperStderr = host.Stderr.RawBytes
	result.Stdout, result.Stderr = stream.guestOutput()
	result.GoFilesystemCalls = environment.filesystemServed()
	result.Transcript = environment.Transcript()
	var networkEvidenceErr error
	if request.Config.NetworkV2 != nil {
		result.NetworkTranscript, networkEvidenceErr = environment.NetworkV2Evidence()
	}
	result.Calls = environment.nextCall - 1
	result.TranscriptSHA256 = environment.TranscriptSHA256()
	result.TimeAdvances = environment.advances
	if request.Config.Storage != nil && !environment.namespaceOwned {
		result.StorageTranscript = environment.StorageTranscript()
		result.StorageSnapshots = environment.StorageSnapshots()
		result.StorageOperations = environment.StorageOperations()
		if request.StorageCrashLimits != nil {
			result.StorageCrashEnumeration = make(map[string]volume.CrashEnumeration)
			for _, configured := range request.Config.Storage.Volumes {
				enumeration, enumerateErr := environment.EnumerateStorageCrashStates(configured.ID, *request.StorageCrashLimits, nil)
				if enumerateErr != nil {
					result.Termination, result.Message = "invalid", enumerateErr.Error()
					return result, nil
				}
				result.StorageCrashEnumeration[configured.ID] = enumeration
			}
		}
		result.StorageImage, err = environment.CrashStorage(request.StorageCrashSelection)
		if err != nil {
			// Image capacity keeps its classification; other failures are
			// infrastructure.
			result.Termination, result.Message = "infrastructure", err.Error()
			var boundary *BoundaryError
			if errors.As(err, &boundary) && boundary.Kind == "capacity" {
				result.Termination = "capacity"
			}
			return result, nil
		}
	}
	result.Started = stream.started
	result.AuthorizationToTerminalNanos = stream.authorizationToTerminalNanos
	result.Reaped = host.GroupGone
	result.OfferedOutputBytes = environment.offeredOutput
	result.Cancelled = ctx.Err() != nil
	result.WatchdogTimeout = !result.Cancelled && (errors.Is(executionContext.Err(), context.DeadlineExceeded) || host.WatchdogTimeout)
	if networkEvidenceErr != nil {
		var boundary *BoundaryError
		result.Termination = "invalid"
		if errors.As(networkEvidenceErr, &boundary) {
			result.Termination = boundary.Kind
		}
		result.Message = networkEvidenceErr.Error()
		return result, nil
	}
	if request.Broker != nil && modeledReapedCancellation(stream.err, request.Broker.Crashed() || request.Broker.Stopped(), host.GroupGone, result.Cancelled, result.WatchdogTimeout) {
		result.Termination, result.Message = "crash", "modeled broker incarnation crash"
		if request.Broker.Stopped() {
			result.Termination, result.Message = "stop", "modeled broker incarnation stop"
		}
		result.collectInterrupted(environment.runtime)
		return result, nil
	}
	if hostErr != nil || closeErr != nil || !host.GroupGone {
		result.Termination = "infrastructure"
		result.Message = fmt.Sprint(errors.Join(hostErr, closeErr))
		return result, nil
	}
	if stream.err != nil {
		var boundary *BoundaryError
		var divergence *RuntimeReplayError
		var storageCrash *StorageCrash
		if errors.As(stream.err, &storageCrash) {
			result.Termination = "storage_crash"
			if !result.collectInterrupted(environment.runtime) {
				return result, nil
			}
		} else if errors.As(stream.err, &divergence) {
			result.Termination = "divergence"
			result.ChoiceDivergence = &divergence.Divergence
		} else if errors.As(stream.err, &boundary) {
			result.Termination = boundary.Kind
			if request.Broker != nil && (boundary.Kind == "crash" && request.Broker.Crashed() || boundary.Kind == "stop" && request.Broker.Stopped()) && !result.collectInterrupted(environment.runtime) {
				return result, nil
			}
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
	if stream.started == nil && terminal.Termination == "invalid" && terminal.Message == "module SHA-256 mismatch" {
		return Result{}, invalid("module/helper bytes changed")
	}
	result.Termination = terminal.Termination
	result.ExitCode = terminal.ExitCode
	result.Message = terminal.Message
	result.FuelRemaining = terminal.FuelRemaining
	result.PeakMemoryBytes = terminal.PeakMemoryBytes
	result.OfferedOutputBytes = terminal.OutputBytes
	result.Telemetry = terminal.Metrics
	if environment.splitOwned {
		if err := stream.collectSplit(&result); err != nil {
			result.Termination, result.Message = "invalid", err.Error()
			return result, nil
		}
	}
	if environment.namespaceOwned && request.Config.Storage != nil {
		if err := stream.collectHelperStorage(&result); err != nil {
			result.Termination, result.Message = "invalid", err.Error()
			return result, nil
		}
	}
	if terminal.Termination == "crash" || terminal.Termination == "stop" || terminal.Termination == "storage_crash" && !environment.helperOwned {
		result.collectInterrupted(environment.runtime)
		return result, nil
	}
	if environment.helperOwned {
		if err := stream.collectHelper(&result); err != nil {
			var divergence *RuntimeReplayError
			if errors.As(err, &divergence) {
				result.Termination = "divergence"
				result.ChoiceDivergence = &divergence.Divergence
			} else {
				result.Termination = "invalid"
			}
			result.Message = err.Error()
		}
		return result, nil
	}
	if environment.runtime != nil && terminal.Termination == "exit" {
		result.ChoiceTrace, result.DiagnosticTrace, err = environment.runtime.collect()
		result.runtimeTerminal = environment.runtime.terminal
		result.VirtualTimeElapsedNanos = environment.runtime.elapsed
		if err != nil {
			result.Termination = "invalid"
			result.Message = err.Error()
		}
	}
	return result, nil
}

type runOwnership int

const (
	// goOwnership keeps namespace, volume and model in Go (stock profile,
	// non-storage broker runs and the test-only Go reference).
	goOwnership runOwnership = iota
	// helperOwnership is the cooperative helper-owned model, namespace and
	// volume of a run without a broker.
	helperOwnership
	// splitOwnership is a broker storage run: the helper owns namespace,
	// volume and transcript; the broker keeps runtime controls and clock.
	splitOwnership
)

// ownership is the single ownership predicate of a run.
func ownership(request Request) runOwnership {
	if request.Config.Profile != CooperativeProfile || request.remoteModel {
		return goOwnership
	}
	if request.Broker == nil && request.Config.NetworkV2 == nil {
		return helperOwnership
	}
	// Network v2 streams are runtime_net_stream calls, which split ownership
	// keeps with the broker and its Go network owner.
	if request.Broker != nil && request.Config.Storage != nil {
		return splitOwnership
	}
	return goOwnership
}

// helperStorageRun reports a storage run whose namespace and volume the helper
// owns: helper-owned (no broker) or split-owned (broker) cooperative runs.
func helperStorageRun(request Request) bool {
	return request.Config.Storage != nil && ownership(request) != goOwnership
}

// SplitOwnsStorage reports a broker storage run under split ownership. Its
// crash/stop revocations must be delivered in band so the helper can flush
// its storage evidence and recovery image before it is reaped.
func SplitOwnsStorage(request Request) bool { return ownership(request) == splitOwnership }

func (result *Result) collectInterrupted(session *runtimeSession) bool {
	if session == nil {
		return true
	}
	var err error
	result.InterruptedChoiceBytes, result.InterruptedDiagnosticBytes, err = session.interrupted()
	if err != nil {
		var divergence *RuntimeReplayError
		result.Termination = "invalid"
		if errors.As(err, &divergence) {
			result.Termination, result.ChoiceDivergence = "divergence", &divergence.Divergence
		}
		result.Message = err.Error()
		return false
	}
	result.RuntimeInterrupted = true
	return true
}

func modeledReapedCancellation(err error, revoked, reaped, cancelled, watchdog bool) bool {
	return revoked && reaped && cancelled && !watchdog && (err == nil || err == context.Canceled || err == io.ErrClosedPipe)
}

type frameSink struct {
	namespaceImage               *namespaceImage
	storageImage                 []byte
	request                      Request
	environment                  *Environment
	input                        io.Writer
	cancel                       context.CancelFunc
	closeInput                   func()
	buffer                       []byte
	started                      *Started
	terminal                     *resultFrame
	err                          error
	helperEvidence               helperEvidence
	context                      context.Context
	storageInterrupted           bool
	interruption                 string
	revokedAfterReply            *BoundaryError
	syncAllowed                  bool
	authorizedAt                 time.Time
	authorizationToTerminalNanos uint64
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
func (sink *frameSink) interrupt(termination, message string) error {
	return sink.interruptWith(termination, message, nil)
}

func (sink *frameSink) interruptWith(termination, message string, record []byte) error {
	if sink.interruption != "" || len(message) > 65536 || len(record) > frameLimit {
		return invalid("modeled interruption")
	}
	sink.interruption = termination
	sink.storageInterrupted = termination == "storage_crash"
	var inline *string
	if record != nil {
		// Records over 64 KiB travel as remote_transcript chunks, as replies do.
		encoded := ""
		if len(record) > 64<<10 {
			for start := 0; start < len(record); start += 64 << 10 {
				if err := sink.send(struct {
					Type string `json:"type"`
					Data []byte `json:"data_base64"`
				}{"remote_transcript", record[start:min(start+64<<10, len(record))]}); err != nil {
					return err
				}
			}
		} else {
			encoded = base64.StdEncoding.EncodeToString(record)
		}
		inline = &encoded
	}
	return sink.send(struct {
		Type        string  `json:"type"`
		Termination string  `json:"termination"`
		Message     string  `json:"message"`
		Transcript  *string `json:"transcript_base64,omitempty"`
	}{"interrupt", termination, message, inline})
}
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

func (sink *frameSink) sendHelperTape() error {
	if !sink.environment.helperOwned || sink.environment.runtime.tape == nil {
		return nil
	}
	decisions := sink.environment.runtime.tape.Decisions
	for start := 0; start < len(decisions); start += helperTapeChunk {
		if err := sink.send(struct {
			Type      string            `json:"type"`
			Decisions []choice.Decision `json:"decisions"`
		}{"tape", decisions[start:min(start+helperTapeChunk, len(decisions))]}); err != nil {
			return err
		}
	}
	return nil
}

func (sink *frameSink) helperCallback(owned validatedCall, state helperLocalState) error {
	handle := sink.environment.handleValidated
	broker := sink.request.Broker
	if sink.environment.splitOwned {
		handle = func(owned validatedCall) (Reply, error) {
			return broker.Handle(sink.context, sink.environment, owned.call)
		}
	}
	sink.syncAllowed = false
	payload, err := sink.environment.helperCallbackReplyWith(owned, state, handle)
	if err != nil {
		var crash *StorageCrash
		var boundary *BoundaryError
		if errors.As(err, &crash) {
			return sink.interrupt("storage_crash", crash.Error())
		}
		if sink.environment.splitOwned && errors.As(err, &boundary) && (boundary.Kind == "crash" && broker.Crashed() || boundary.Kind == "stop" && broker.Stopped()) {
			// An output boundary revokes after Go completed and recorded the
			// call; the helper appends that record as the Go owner keeps it.
			record := sink.environment.transcript
			sink.environment.transcript = nil
			return sink.interruptWith(boundary.Kind, boundary.Error(), record)
		}
		return err
	}
	// The broker evaluates delivery after the reply; the helper confirms
	// before its next local call that no revocation crossed it.
	payload.Revocable = sink.environment.splitOwned
	if len(payload.Transcript) > 64<<10 {
		for start := 0; start < len(payload.Transcript); start += 64 << 10 {
			if err := sink.send(struct {
				Type string `json:"type"`
				Data []byte `json:"data_base64"`
			}{"remote_transcript", payload.Transcript[start:min(start+64<<10, len(payload.Transcript))]}); err != nil {
				return err
			}
		}
		payload.Transcript = []byte{}
	}
	if err := sink.send(payload); err != nil {
		return err
	}
	if !sink.environment.splitOwned {
		return nil
	}
	sink.syncAllowed = true
	return sink.delivered(owned.call)
}

// delivered reports a sent reply to the broker. A crash or stop revocation
// found here answers the helper's next call or sync with the interrupt.
func (sink *frameSink) delivered(call Call) error {
	broker := sink.request.Broker
	err := broker.Delivered(call)
	var boundary *BoundaryError
	if errors.As(err, &boundary) && (boundary.Kind == "crash" && broker.Crashed() || boundary.Kind == "stop" && broker.Stopped()) {
		sink.revokedAfterReply = boundary
		return nil
	}
	return err
}

// revocationSync answers a split-owned helper's confirmation before a local
// call: `continue`, or the interrupt of a revocation that crossed the reply.
func (sink *frameSink) revocationSync(frame *validatedFrame) error {
	var sync struct {
		Type string `json:"type"`
		ID   uint64 `json:"id"`
	}
	if err := frame.decode(&sync, "type", "id"); err != nil {
		return err
	}
	if !sink.environment.splitOwned || !sink.syncAllowed || sink.interruption != "" || sync.ID+1 != sink.environment.nextCall {
		return invalid("unexpected revocation sync")
	}
	sink.syncAllowed = false
	if revoked := sink.revokedAfterReply; revoked != nil {
		return sink.interrupt(revoked.Kind, revoked.Error())
	}
	// The Go-owned path raises the clock to broker time at every call.
	e := sink.environment
	e.now = max(e.now, sink.request.Broker.broker.Current())
	return sink.send(struct {
		Type string `json:"type"`
		Now  uint64 `json:"now"`
	}{"continue", e.now})
}

func (sink *frameSink) frame(data []byte) error {
	if sink.terminal != nil {
		return invalid("frame after terminal")
	}
	frame, err := newValidatedFrame(data)
	if err != nil {
		return err
	}
	fields := frame.fields
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
		if err := frame.decode(&started, "schema", "type", "engine", "module_sha256", "imports", "exports", "initial_memory_pages"); err != nil {
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
		// A split-owned helper confirms the broker clock before its first
		// local call.
		sink.syncAllowed = sink.environment.splitOwned
		if sink.request.Broker != nil {
			sink.request.Broker.Prepared()
			if err := sink.request.Broker.Activate(sink.context); err != nil {
				return err
			}
		}
		sink.authorizedAt = time.Now()
		if err := sink.send(struct {
			Type          string `json:"type"`
			Module        string `json:"module_sha256"`
			Configuration string `json:"configuration"`
		}{"authorize", started.ModuleSHA256, started.Engine.Configuration}); err != nil {
			return err
		}
		if err := sink.sendHelperTape(); err != nil {
			return err
		}
		return sink.sendHelperNamespace()
	case "call":
		if sink.started == nil {
			return invalid("callback before authorization")
		}
		if sink.interruption != "" {
			return invalid("callback after modeled interruption")
		}
		owned, state, err := frame.decodeCall(sink.environment.helperOwned || sink.environment.splitOwned)
		if err != nil {
			return err
		}
		if revoked := sink.revokedAfterReply; revoked != nil {
			// The helper may issue this call before it could observe the
			// revocation; it receives the interrupt in place of a reply.
			return sink.interrupt(revoked.Kind, revoked.Error())
		}
		call := owned.call
		if len(call.Op) > 128 {
			return invalid("operation name")
		}
		if state != nil {
			return sink.helperCallback(owned, *state)
		}
		var reply Reply
		if sink.request.Broker != nil {
			reply, err = sink.request.Broker.Handle(sink.context, sink.environment, call)
		} else {
			reply, err = sink.environment.handleValidated(owned)
		}
		if err != nil {
			var crash *StorageCrash
			var boundary *BoundaryError
			if errors.As(err, &crash) {
				return sink.interrupt("storage_crash", crash.Error())
			}
			if errors.As(err, &boundary) && sink.request.Broker != nil && (boundary.Kind == "crash" && sink.request.Broker.Crashed() || boundary.Kind == "stop" && sink.request.Broker.Stopped()) {
				return sink.interrupt(boundary.Kind, boundary.Error())
			}
			return err
		}
		if err := sink.send(reply); err != nil {
			return err
		}
		if sink.request.Broker != nil {
			// The helper already holds this reply and reads frames only
			// while awaiting one, so the interrupt answers its next call.
			return sink.delivered(call)
		}
		return nil
	case "evidence":
		return sink.receiveHelperEvidence(frame)
	case "sync":
		return sink.revocationSync(frame)
	case "result":
		var result resultFrame
		keys := []string{"type", "termination", "message", "fuel_remaining", "peak_memory_bytes", "output_bytes"}
		if _, found := fields["exit_code"]; found {
			keys = append(keys, "exit_code")
		}
		for _, key := range []string{"local_model", "divergence", "metrics"} {
			if _, found := fields[key]; found {
				keys = append(keys, key)
			}
		}
		if err := frame.decode(&result, keys...); err != nil {
			return err
		}
		switch result.Termination {
		case "exit":
			if result.ExitCode == nil || sink.started == nil {
				return invalid("exit terminal")
			}
		case "capacity", "unsupported", "invalid", "infrastructure", "divergence", "storage_crash", "crash", "stop":
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
		if result.Termination == "divergence" && (!sink.environment.helperOwned || result.Divergence == nil) || result.Termination != "divergence" && result.Divergence != nil || !sink.environment.helperOwned && !sink.environment.splitOwned && result.LocalModel != nil {
			return invalid("helper model terminal classification")
		}
		if result.Termination == "storage_crash" && !sink.storageInterrupted && (!sink.environment.namespaceOwned || !storageCrashConfigured(sink.request.Config.Storage, result.Message)) {
			return invalid("unsolicited storage crash terminal")
		}
		if (sink.interruption != "" || sink.revokedAfterReply != nil) && (result.Termination == "exit" || result.Termination == "trap") {
			return invalid("guest completion after modeled interruption")
		}
		if (result.Termination == "crash" || result.Termination == "stop") && (result.Termination != sink.interruption || sink.request.Broker == nil || result.Termination == "crash" && !sink.request.Broker.Crashed() || result.Termination == "stop" && !sink.request.Broker.Stopped()) {
			return invalid("unsolicited broker interruption terminal")
		}
		outputValid := result.OutputBytes == sink.environment.offeredOutput
		if sink.environment.namespaceOwned && !sink.environment.splitOwned {
			outputValid = result.OutputBytes >= uint64(len(sink.helperEvidence.stdout))+uint64(len(sink.helperEvidence.stderr))
		}
		if len(result.Message) > 65536 || result.FuelRemaining > sink.request.Fuel || result.PeakMemoryBytes > sink.request.MemoryBytes || result.OutputBytes > sink.request.Config.Limits.OutputBytes || !outputValid {
			return invalid("terminal limits")
		}
		sink.terminal = &result
		if !sink.authorizedAt.IsZero() {
			sink.authorizationToTerminalNanos = uint64(time.Since(sink.authorizedAt).Nanoseconds())
		}
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
