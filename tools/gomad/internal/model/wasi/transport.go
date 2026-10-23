package wasi

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"time"

	"go.temporal.io/server/tools/gomad/internal/model/choice"
	"go.temporal.io/server/tools/gomad/internal/model/volume"
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
	StorageImage                                       *StorageImage
	StorageCrashSelection                              map[string][]uint64
	StorageCrashContext                                map[string]StorageCrashContext `json:",omitempty"`
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
	stressHostEnvironment           []string
}
type guestProfileConfig struct {
	Path           string `json:"path"`
	IntervalMicros uint64 `json:"interval_micros"`
	MaxSamples     uint64 `json:"max_samples"`
}
type Result struct {
	InspectionMetadata                                 []byte
	DebugOutput                                        []byte
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
	InputTapeSHA256                                    [32]byte
	InputDecisions                                     uint64
	RuntimeInterrupted                                 bool
	InterruptedChoiceBytes, InterruptedDiagnosticBytes []byte
	VirtualTimeElapsedNanos                            uint64
	StorageImage                                       *StorageImage
	StorageTranscript                                  []byte
	StorageSnapshots                                   []volume.VolumeSnapshot
	StorageOperations                                  map[string][]volume.Operation
	StorageCrashContext                                map[string]StorageCrashContext `json:",omitempty"`
	StorageCrashEnumeration                            map[string]volume.CrashEnumeration
	Telemetry                                          *Telemetry

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
	Error           string             `json:"error,omitempty"`
	Check           string             `json:"check,omitempty"`
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

// splitCancellationTerminal classifies a split-owned storage incarnation that
// was cancelled before its broker revocation reached the helper.
func splitCancellationTerminal() (string, string) {
	return "infrastructure", "split-owned storage incarnation cancelled before its revocation reached the helper; no recovery image"
}

func modeledReapedCancellation(err error, revoked, reaped, cancelled, watchdog bool) bool {
	return revoked && reaped && cancelled && !watchdog && (err == nil || err == context.Canceled || err == io.ErrClosedPipe)
}

type frameSink struct {
	namespaceImage               *namespaceImage
	storageImage                 []byte
	storageContext               []byte
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
	return sink.interruptWith(termination, message, nil, nil)
}

// interruptedCall is the helper-visible state after a remote call that Go
// completed before the broker revoked the incarnation.
type interruptedCall struct {
	Now      uint64 `json:"now"`
	Advances uint64 `json:"advances"`
}

func (sink *frameSink) interruptWith(termination, message string, completed *interruptedCall, record []byte) error {
	if sink.interruption != "" || len(message) > 65536 || len(record) > frameLimit || completed == nil && record != nil {
		return invalid("modeled interruption")
	}
	sink.interruption = termination
	sink.storageInterrupted = termination == "storage_crash"
	var inline *string
	if len(record) != 0 {
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
		Type        string           `json:"type"`
		Termination string           `json:"termination"`
		Message     string           `json:"message"`
		Completed   *interruptedCall `json:"completed,omitempty"`
		Transcript  *string          `json:"transcript_base64,omitempty"`
	}{"interrupt", termination, message, completed, inline})
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

type ReplayPreflightError struct{ Check, Message string }

func (failure *ReplayPreflightError) Error() string {
	return "replay preflight " + failure.Check + ": " + failure.Message
}
func (failure *ReplayPreflightError) Is(target error) bool {
	return target == ErrChoiceReplayPreflight
}

func helperReplayBlock(request *choice.Request, node string, incarnation uint64) ([]byte, error) {
	if request.Mode == choice.ModeRecord {
		return nil, nil
	}
	if uint64(len(request.Trace)) > request.Limit {
		return nil, &ReplayPreflightError{Check: "framing", Message: "trace exceeds choice limit"}
	}
	frames := uint64(len(request.Trace)) / 65536
	if len(request.Trace)%65536 != 0 {
		frames++
	}
	var block bytes.Buffer
	marker := struct {
		Type        string `json:"type"`
		Schema      string `json:"schema"`
		TraceBytes  uint64 `json:"trace_bytes"`
		TraceFrames uint64 `json:"trace_frames"`
		Instance    string `json:"instance,omitempty"`
		Incarnation uint64 `json:"incarnation,omitempty"`
	}{"replay_trace", helperReplaySchema, uint64(len(request.Trace)), frames, node, incarnation}
	appendMarker := func() error {
		data, err := json.Marshal(marker)
		if err != nil {
			return err
		}
		block.Write(data)
		block.WriteByte('\n')
		return nil
	}
	if err := appendMarker(); err != nil {
		return nil, err
	}
	for start := 0; start < len(request.Trace); start += 65536 {
		chunk := request.Trace[start:min(start+65536, len(request.Trace))]
		var length [4]byte
		binary.BigEndian.PutUint32(length[:], uint32(len(chunk)))
		block.Write(length[:])
		block.Write(chunk)
	}
	marker.Type = "replay_trace_end"
	if err := appendMarker(); err != nil {
		return nil, err
	}
	return block.Bytes(), nil
}

func (sink *frameSink) sendHelperReplay() error {
	if !sink.environment.helperOwned {
		return nil
	}
	block, err := helperReplayBlock(sink.environment.runtime.control.Choice, "", 0)
	if err != nil || len(block) == 0 {
		return err
	}
	written, err := sink.input.Write(block)
	if err == nil && written != len(block) {
		return io.ErrShortWrite
	}
	return err
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
			// A revocation that lands after Go handled the call (an output
			// boundary, or a concurrent crash/stop) leaves the call completed
			// and recorded in the Go-owned path. The helper adopts that call:
			// its clock state and, for WASI calls, its canonical record.
			e := sink.environment
			if e.nextCall <= owned.call.ID {
				return sink.interrupt(boundary.Kind, boundary.Error())
			}
			e.closeClockRun()
			record := e.transcript
			if strings.HasPrefix(owned.call.Op, "runtime_") {
				record = nil
			}
			e.transcript = nil
			return sink.interruptWith(boundary.Kind, boundary.Error(), &interruptedCall{Now: e.now, Advances: e.advances}, record)
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
	now, err := sink.request.Broker.localContinuation()
	if err != nil {
		var boundary *BoundaryError
		if errors.As(err, &boundary) && (boundary.Kind == "crash" || boundary.Kind == "stop") {
			return sink.interrupt(boundary.Kind, boundary.Error())
		}
		return err
	}
	// The Go-owned path raises the clock to broker time at every call.
	e := sink.environment
	e.now = max(e.now, now)
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
		if err := validateEngineConfiguration(started.Engine, sink.request); err != nil {
			return err
		}
		if started.Schema != "gomad.wasm-host/v1" || started.Engine != sink.request.Engine || started.ModuleSHA256 != sink.request.ModuleSHA256 || started.InitialMemoryPages != sink.request.InitialMemoryPages || !sameImports(started.Imports, sink.request.Imports) {
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
		if err := sink.sendHelperReplay(); err != nil {
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
		for _, key := range []string{"local_model", "divergence", "metrics", "error", "check"} {
			if _, found := fields[key]; found {
				keys = append(keys, key)
			}
		}
		if err := frame.decode(&result, keys...); err != nil {
			return err
		}
		if result.Error != "" || result.Check != "" {
			if result.Error != "replay_preflight" || result.Termination != "invalid" || result.Check == "" || result.LocalModel != nil || result.Divergence != nil || result.PeakMemoryBytes != 0 || result.OutputBytes != 0 || result.Metrics != nil && result.Metrics.InstantiateNanos != 0 {
				return invalid("invalid replay preflight terminal")
			}
			return &ReplayPreflightError{Check: result.Check, Message: result.Message}
		}
		switch result.Termination {
		case "exit":
			if result.ExitCode == nil || sink.started == nil {
				return invalid("exit terminal")
			}
		case "capacity", "unsupported", "invalid", "infrastructure", "divergence", "storage_crash", "crash", "stop", "inspected_prefix", "instrumentation_divergence", "debug_error", "probe_fuel_limit":
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
		if sink.environment.splitOwned {
			outputValid = sink.environment.namespaceOwned && result.OutputBytes >= sink.environment.offeredOutput
		}
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
		return &BoundaryError{Kind: "unsupported", Dimension: "unknown protocol frame " + kind}
	}
}

func validateEngineConfiguration(engine EngineIdentity, request Request) error {
	pinned := PinnedEngine().Configuration
	if engine.Configuration != pinned && (request.diagnosticConfigurationRevision == "" || engine.Configuration != request.diagnosticConfigurationRevision) {
		return &BoundaryError{Kind: "unsupported", Dimension: "engine configuration outside pin: " + engine.Configuration}
	}
	return nil
}
func sameImports(left, right []Import) bool {
	return slices.EqualFunc(left, right, func(a, b Import) bool {
		return a.Module == b.Module && a.Name == b.Name && slices.Equal(a.Params, b.Params) && slices.Equal(a.Results, b.Results)
	})
}

var ErrChoiceReplayPreflight = errors.New("choice replay preflight refused")
