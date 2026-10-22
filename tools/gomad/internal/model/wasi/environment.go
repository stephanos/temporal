package wasi

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"hash"
	"io"
	"math"
	"path"
	"slices"
	"strings"
	"unicode/utf8"

	captured "go.temporal.io/server/tools/gomad/internal/model/capture"
)

const StockProfile = "gomad.wasi-stock-exploratory/v1"
const entropyProfile = "gomad.wasi-entropy-sha256-counter/v1"
const frameLimit = 8 << 20

// maxCooperativeCalls admits whole Temporal workloads, whose scheduler decisions are host calls.
const maxCooperativeCalls = 1 << 28
const bufferLimit = 4 << 20

type Limits struct {
	OutputBytes, TranscriptBytes, Calls, Descriptors, Files, FilesystemBytes, PendingEvents uint64
}
type ClockPolicy struct{ EpochNanos, ReadStepNanos uint64 }
type Config struct {
	Profile             string
	Args, Environment   []string
	WorkingDirectory    string
	WritableDirectories []string
	CapturedInputs      captured.CapturedInputs
	Stdin               []byte
	EntropyKey          [32]byte
	Clock               ClockPolicy
	Limits              Limits
	Storage             *StorageModel   `json:"storage,omitempty"`
	NetworkV2           *NetworkModelV2 `json:"network_v2,omitempty"`
}
type Call struct {
	Type  string          `json:"type"`
	ID    uint64          `json:"id"`
	Op    string          `json:"op"`
	Input json.RawMessage `json:"input"`
}
type Reply struct {
	Type        string          `json:"type"`
	ID          uint64          `json:"id"`
	Errno       uint16          `json:"errno"`
	Output      json.RawMessage `json:"output"`
	Unsupported bool            `json:"unsupported,omitempty"`
	Message     string          `json:"message,omitempty"`
}
type valuesOutput struct {
	Values [][]byte `json:"values"`
}
type sizesOutput struct {
	Count uint32 `json:"count"`
	Bytes uint32 `json:"bytes"`
}
type bytesOutput struct {
	Data []byte `json:"data_base64"`
}
type timestampOutput struct {
	Timestamp uint64 `json:"timestamp"`
}
type offsetOutput struct {
	Offset uint64 `json:"offset"`
}
type writtenOutput struct {
	Written uint32 `json:"written"`
}

type BoundaryError struct{ Kind, Dimension string }

func (e *BoundaryError) Error() string { return "WASI " + e.Kind + ": " + e.Dimension }
func capacity(dimension string) error  { return &BoundaryError{Kind: "capacity", Dimension: dimension} }
func invalid(dimension string) error   { return &BoundaryError{Kind: "invalid", Dimension: dimension} }

type Environment struct {
	config        Config
	identity      string
	namespace     *namespace
	nextCall      uint64
	entropyOffset uint64
	offeredOutput uint64
	now           uint64
	transcript    []byte
	// With digestOnly the transcript is hashed as it grows instead of retained.
	digestOnly       bool
	transcriptHash   hash.Hash
	transcriptLength uint64
	transcriptWriter transcriptEncoder
	stdout, stderr   []byte
	runtime          *runtimeSession
	mirror           io.Writer // observes guest stdout; never feeds back into the guest
	advances         uint64    // idle wakeups that moved the clock
	helperOwned      bool
	// splitOwned is broker split ownership: the helper owns namespace,
	// volume and transcript; runtime controls and fd 1/2 writes stay here.
	splitOwned bool
	// storageRecovery is the image this environment recovers from.
	storageRecovery *StorageImage
	namespaceOwned  bool
	filesystemCalls map[string]uint64
	networkV2       *networkV2State
	brokerTime      *uint64
}

func NewEnvironment(config Config) (*Environment, error) {
	if config.Profile != StockProfile && config.Profile != CooperativeProfile {
		return nil, invalid("profile")
	}
	limits := config.Limits
	if limits.OutputBytes == 0 || limits.OutputBytes > 1<<30 || limits.TranscriptBytes == 0 || limits.TranscriptBytes > 1<<30 || limits.Calls == 0 || limits.Calls > 1000000 && (config.Profile != CooperativeProfile || limits.Calls > maxCooperativeCalls) || limits.Descriptors < 5 || limits.Descriptors > 1<<20 || limits.Files == 0 || limits.Files > 1<<20 || limits.FilesystemBytes == 0 || limits.FilesystemBytes > 1<<30 || limits.PendingEvents == 0 || limits.PendingEvents > 4096 {
		return nil, invalid("limits")
	}
	if !absoluteClean(config.WorkingDirectory) || config.Clock.EpochNanos > math.MaxInt64 || config.Clock.ReadStepNanos == 0 && config.Profile != CooperativeProfile || config.Clock.ReadStepNanos > 1000000000 {
		return nil, invalid("working directory or clock policy")
	}
	if config.NetworkV2 != nil && (config.Profile != CooperativeProfile || config.Clock.ReadStepNanos != 0) {
		return nil, invalid("network v2 requires cooperative frozen-clock profile")
	}
	if err := validateNetworkModelV2(config.NetworkV2); err != nil {
		return nil, err
	}
	if _, err := stringSizes(config.Args); err != nil {
		return nil, err
	}
	if _, err := stringSizes(config.Environment); err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	pwd := ""
	for _, value := range config.Environment {
		name, content, found := strings.Cut(value, "=")
		if !found || name == "" || seen[name] {
			return nil, invalid("environment entries")
		}
		seen[name] = true
		if name == "PWD" {
			pwd = content
		}
	}
	if pwd != config.WorkingDirectory {
		return nil, invalid("captured PWD must match working directory")
	}
	if uint64(len(config.Stdin)) > limits.FilesystemBytes {
		return nil, capacity("stdin-bytes")
	}
	for i, directory := range config.WritableDirectories {
		if !absoluteClean(directory) || directory == "/" || slices.Contains(config.WritableDirectories[:i], directory) {
			return nil, invalid("writable directory")
		}
	}
	manifest := config.CapturedInputs.Manifest
	for _, value := range append([]string{manifest.Schema, manifest.File, string(manifest.SHA256)}, manifest.Mappings...) {
		if !utf8.ValidString(value) {
			return nil, invalid("UTF-8 in captured manifest")
		}
	}
	config.Args = slices.Clone(config.Args)
	config.Environment = slices.Clone(config.Environment)
	config.WritableDirectories = slices.Clone(config.WritableDirectories)
	config.Stdin = slices.Clone(config.Stdin)
	if config.NetworkV2 != nil {
		network := *config.NetworkV2
		network.Faults = slices.Clone(network.Faults)
		config.NetworkV2 = &network
	}
	if config.Storage != nil {
		storage := *config.Storage
		storage.Volumes = slices.Clone(storage.Volumes)
		storage.Faults = slices.Clone(storage.Faults)
		storage.SQLiteLockDatabases = slices.Clone(storage.SQLiteLockDatabases)
		config.Storage = &storage
	}
	data, err := json.Marshal(struct {
		Profile, EntropyProfile string
		Args, Environment       []string
		WorkingDirectory        string
		WritableDirectories     []string
		CapturedManifest        captured.CapturedInputsManifest
		Stdin                   []byte
		EntropyKey              [32]byte
		Clock                   ClockPolicy
		Limits                  Limits
		FrameBytes, BufferBytes uint64
		Storage                 *StorageModel   `json:"storage,omitempty"`
		NetworkV2               *NetworkModelV2 `json:"network_v2,omitempty"`
	}{config.Profile, entropyProfile, config.Args, config.Environment, config.WorkingDirectory, config.WritableDirectories, config.CapturedInputs.Manifest, config.Stdin, config.EntropyKey, config.Clock, config.Limits, frameLimit, bufferLimit, config.Storage, config.NetworkV2})
	if err != nil {
		return nil, err
	}
	digest := sha256.Sum256(data)
	namespace, err := newNamespace(config)
	if err != nil {
		return nil, err
	}
	config.CapturedInputs = captured.CapturedInputs{}
	config.Stdin = nil
	e := &Environment{config: config, namespace: namespace, identity: hex.EncodeToString(digest[:]), nextCall: 1}
	e.networkV2, err = newNetworkV2State(config.NetworkV2, config.Clock.EpochNanos)
	if err != nil {
		return nil, err
	}
	if config.Profile == CooperativeProfile && config.Clock.ReadStepNanos == 0 {
		e.now = 1
	}
	return e, nil
}
func absoluteClean(value string) bool {
	return utf8.ValidString(value) && strings.HasPrefix(value, "/") && path.Clean(value) == value && strings.IndexByte(value, 0) < 0
}
func stringSizes(values []string) (sizesOutput, error) {
	if len(values) > bufferLimit {
		return sizesOutput{}, capacity("strings-count")
	}
	var bytes uint64
	for _, value := range values {
		if !utf8.ValidString(value) {
			return sizesOutput{}, invalid("UTF-8 in argument or environment")
		}
		if strings.IndexByte(value, 0) >= 0 {
			return sizesOutput{}, invalid("NUL in argument or environment")
		}
		bytes += uint64(len(value)) + 1
		if bytes > bufferLimit {
			return sizesOutput{}, capacity("strings-bytes")
		}
	}
	return sizesOutput{uint32(len(values)), uint32(bytes)}, nil
}
func (e *Environment) Identity() string { return e.identity }
func (e *Environment) Transcript() []byte {
	data := slices.Clone(e.transcript)
	if !e.digestOnly && e.transcriptWriter.clock != nil {
		data = append(data, e.transcriptWriter.clock.bytes()...)
	}
	return data
}

// TranscriptSHA256 digests the complete transcript in either retention mode.
func (e *Environment) TranscriptSHA256() [32]byte {
	if e.digestOnly {
		var sum [32]byte
		digest := e.transcriptHash
		if e.transcriptWriter.clock != nil {
			cloned, err := digest.(hash.Cloner).Clone()
			if err != nil {
				panic(err) //nolint:forbidigo // The private pinned SHA-256 must clone; no digest can be returned safely if that invariant fails.
			}
			cloned.Write(e.transcriptWriter.clock.bytes())
			digest = cloned
		}
		copy(sum[:], digest.Sum(nil))
		return sum
	}
	return sha256.Sum256(e.Transcript())
}
func (e *Environment) Output() ([]byte, []byte) {
	return slices.Clone(e.stdout), slices.Clone(e.stderr)
}
func (e *Environment) filesystemServed() map[string]uint64 {
	counts := make(map[string]uint64, len(e.filesystemCalls))
	for op, count := range e.filesystemCalls {
		counts[op] = count
	}
	return counts
}
func filesystemCall(call Call) bool {
	if strings.HasPrefix(call.Op, "fd_") || strings.HasPrefix(call.Op, "path_") {
		return true
	}
	if call.Op != "poll_oneoff" {
		return false
	}
	var input struct {
		Subscriptions []struct {
			Type uint8 `json:"type"`
		} `json:"subscriptions"`
	}
	if json.Unmarshal(call.Input, &input) != nil {
		return false
	}
	for _, sub := range input.Subscriptions {
		if sub.Type == 1 || sub.Type == 2 {
			return true
		}
	}
	return false
}
func (e *Environment) Handle(call Call) (Reply, error) {
	return e.handle(call, nil)
}

func (e *Environment) handle(call Call, validated *validatedInput) (Reply, error) {
	if call.Type != "call" || call.ID != e.nextCall {
		return Reply{}, invalid("callback sequence")
	}
	if e.nextCall > e.config.Limits.Calls {
		return Reply{}, capacity("callbacks")
	}
	callBytes, err := json.Marshal(call)
	if err != nil {
		return Reply{}, invalid("callback JSON")
	}
	if len(callBytes)+1 > frameLimit {
		return Reply{}, capacity("callback-frame")
	}
	if strings.HasPrefix(call.Op, "runtime_") {
		e.closeClockRun()
		if e.runtime == nil || e.config.Profile != CooperativeProfile {
			return Reply{}, invalid("runtime import requires cooperative profile")
		}
		var result bytesOutput
		switch call.Op {
		case "runtime_idle":
			result, err = e.runtimeIdle(call.Input)
		case "runtime_net_stream":
			result, err = e.runtimeNetworkStream(call.Input)
		default:
			result, err = e.runtime.handle(call.Op, call.Input)
		}
		if err != nil {
			return Reply{}, err
		}
		encoded, err := json.Marshal(result)
		if err != nil {
			return Reply{}, err
		}
		e.nextCall++
		return Reply{Type: "reply", ID: call.ID, Output: encoded}, nil
	}
	reservation, err := e.replyFrameReservation(call, validated)
	if err != nil {
		return Reply{}, err
	}
	if reservation+1 > frameLimit {
		return Reply{}, capacity("reply-frame")
	}
	minimum := uint64(transcriptRecordHeader)
	if call.Op == "clock_time_get" && e.transcriptWriter.clock != nil {
		minimum = 0
	}
	if e.transcriptLength == 0 {
		minimum += uint64(len(transcriptHeader))
	}
	if !e.digestOnly && e.transcriptLength+minimum > e.config.Limits.TranscriptBytes {
		return Reply{}, capacity("transcript-bytes")
	}
	tag, input, err := transcriptInput(call, e.entropyOffset)
	if err != nil {
		return Reply{}, err
	}
	transcriptReservation, err := e.responseReservation(tag, input, call)
	if err != nil {
		return Reply{}, err
	}
	if !e.digestOnly && e.transcriptLength+transcriptReservation > e.config.Limits.TranscriptBytes {
		return Reply{}, capacity("transcript-bytes")
	}
	reply := Reply{Type: "reply", ID: call.ID, Output: json.RawMessage(`{}`)}
	clockNow := e.now
	if tag != 1 {
		e.closeClockRun()
	}
	if filesystemCall(call) {
		if e.filesystemCalls == nil {
			e.filesystemCalls = make(map[string]uint64)
		}
		e.filesystemCalls[call.Op]++
	}
	result, errno, unsupported, err := e.dispatch(call, validated)
	if err != nil {
		return Reply{}, err
	}
	reply.Errno = errno
	if unsupported != "" {
		reply.Unsupported = true
		reply.Message = unsupported
	}
	if errno == 0 && result != nil {
		reply.Output, err = json.Marshal(result)
		if err != nil {
			return Reply{}, err
		}
	}
	replyBytes, err := json.Marshal(reply)
	if err != nil {
		return Reply{}, err
	}
	if uint64(len(replyBytes)) > reservation {
		return Reply{}, invalid("internal transcript reservation")
	}
	if tag == 1 && !reply.Unsupported {
		e.appendClockRead(call.ID, binary.LittleEndian.Uint32(input), binary.LittleEndian.Uint64(input[4:]), clockNow)
	} else {
		data, err := encodeTranscriptRecord(tag, call.ID, input, reply)
		if err != nil {
			return Reply{}, err
		}
		e.appendTranscript(data)
	}

	e.nextCall++
	return reply, nil
}
func (e *Environment) replyFrameReservation(call Call, validated *validatedInput) (uint64, error) {
	if len(call.Input) > frameLimit {
		return 0, capacity("input-frame")
	}
	// Reserving the maximum buffer before dispatch prevents a capacity result from committing a modeled mutation.
	var fields map[string]json.RawMessage
	if validated == nil {
		if err := validateJSON(call.Input); err != nil {
			return 0, err
		}
		if err := json.Unmarshal(call.Input, &fields); err != nil || fields == nil {
			return 0, invalid("operation input")
		}
	} else {
		var err error
		fields, err = validated.fields()
		if err != nil {
			return 0, err
		}
	}
	var length uint32
	if raw, ok := fields["length"]; ok {
		if err := json.Unmarshal(raw, &length); err != nil {
			return 0, invalid("operation length")
		}
		if length > bufferLimit {
			return 0, capacity("buffer-bytes")
		}
	}
	extra := uint64(length)*4/3 + 4
	if call.Op == "args_get" || call.Op == "environ_get" {
		values := e.config.Args
		if call.Op == "environ_get" {
			values = e.config.Environment
		}
		for _, value := range values {
			extra += uint64(len(value))*4/3 + 8
		}
	}
	if call.Op == "poll_oneoff" {
		extra += uint64(len(call.Input)) * 4
	}
	return 1024 + extra, nil
}
func (e *Environment) dispatch(call Call, validated *validatedInput) (any, uint16, string, error) {
	switch call.Op {
	case "args_sizes_get", "environ_sizes_get", "args_get", "environ_get", "sched_yield":
		if err := decodeOperationInput(call.Input, validated, &struct{}{}); err != nil {
			return nil, 0, "", err
		}
		values := e.config.Args
		if strings.HasPrefix(call.Op, "environ") {
			values = e.config.Environment
		}
		if call.Op == "sched_yield" {
			return struct{}{}, 0, "", nil
		}
		if strings.HasSuffix(call.Op, "sizes_get") {
			sizes, err := stringSizes(values)
			return sizes, 0, "", err
		}
		bytes := make([][]byte, len(values))
		for i, value := range values {
			bytes[i] = []byte(value)
		}
		return valuesOutput{bytes}, 0, "", nil
	case "random_get":
		var input struct {
			Length uint32 `json:"length"`
		}
		if err := decodeOperationInput(call.Input, validated, &input, "length"); err != nil {
			return nil, 0, "", err
		}
		if input.Length > bufferLimit || math.MaxUint64-e.entropyOffset < uint64(input.Length) {
			return nil, 0, "", capacity("entropy-bytes")
		}
		data := make([]byte, input.Length)
		for i := range data {
			offset := e.entropyOffset + uint64(i)
			var counter [8]byte
			binary.LittleEndian.PutUint64(counter[:], offset/32)
			material := append([]byte(entropyProfile), e.config.EntropyKey[:]...)
			material = append(material, counter[:]...)
			digest := sha256.Sum256(material)
			data[i] = digest[offset%32]
		}
		e.entropyOffset += uint64(input.Length)
		return bytesOutput{data}, 0, "", nil
	default:
		return e.namespaceCall(call, validated)
	}
}
