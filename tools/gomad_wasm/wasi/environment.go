package wasi

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"math"
	"path"
	"slices"
	"strings"
	"unicode/utf8"

	"go.temporal.io/server/tools/gomad3/deterministicio/readonlymount"
)

const StockProfile = "gomad3.wasi-stock-exploratory/v1"
const entropyProfile = "gomad3.wasi-entropy-sha256-counter/v1"
const frameLimit = 8 << 20
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
	CapturedInputs      readonlymount.CapturedInputs
	Stdin               []byte
	EntropyKey          [32]byte
	Clock               ClockPolicy
	Limits              Limits
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
	config         Config
	identity       string
	namespace      *namespace
	nextCall       uint64
	entropyOffset  uint64
	offeredOutput  uint64
	now            uint64
	transcript     []byte
	stdout, stderr []byte
	runtime        *runtimeSession
}

func NewEnvironment(config Config) (*Environment, error) {
	if config.Profile != StockProfile && config.Profile != CooperativeProfile {
		return nil, invalid("profile")
	}
	limits := config.Limits
	if limits.OutputBytes == 0 || limits.OutputBytes > 1<<30 || limits.TranscriptBytes == 0 || limits.TranscriptBytes > 1<<30 || limits.Calls == 0 || limits.Calls > 1000000 || limits.Descriptors < 5 || limits.Descriptors > 1<<20 || limits.Files == 0 || limits.Files > 1<<20 || limits.FilesystemBytes == 0 || limits.FilesystemBytes > 1<<30 || limits.PendingEvents == 0 || limits.PendingEvents > 4096 {
		return nil, invalid("limits")
	}
	if !absoluteClean(config.WorkingDirectory) || config.Clock.EpochNanos > math.MaxInt64 || config.Clock.ReadStepNanos == 0 && config.Profile != CooperativeProfile || config.Clock.ReadStepNanos > 1000000000 {
		return nil, invalid("working directory or clock policy")
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
	data, err := json.Marshal(struct {
		Profile, EntropyProfile string
		Args, Environment       []string
		WorkingDirectory        string
		WritableDirectories     []string
		CapturedManifest        readonlymount.CapturedInputsManifest
		Stdin                   []byte
		EntropyKey              [32]byte
		Clock                   ClockPolicy
		Limits                  Limits
		FrameBytes, BufferBytes uint64
	}{config.Profile, entropyProfile, config.Args, config.Environment, config.WorkingDirectory, config.WritableDirectories, config.CapturedInputs.Manifest, config.Stdin, config.EntropyKey, config.Clock, config.Limits, frameLimit, bufferLimit})
	if err != nil {
		return nil, err
	}
	digest := sha256.Sum256(data)
	namespace, err := newNamespace(config)
	if err != nil {
		return nil, err
	}
	config.CapturedInputs = readonlymount.CapturedInputs{}
	config.Stdin = nil
	e := &Environment{config: config, namespace: namespace, identity: hex.EncodeToString(digest[:]), nextCall: 1}
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
func (e *Environment) Identity() string   { return e.identity }
func (e *Environment) Transcript() []byte { return slices.Clone(e.transcript) }
func (e *Environment) Output() ([]byte, []byte) {
	return slices.Clone(e.stdout), slices.Clone(e.stderr)
}
func (e *Environment) Handle(call Call) (Reply, error) {
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
		if e.runtime == nil || e.config.Profile != CooperativeProfile {
			return Reply{}, invalid("runtime import requires cooperative profile")
		}
		var result bytesOutput
		if call.Op == "runtime_idle" {
			result, err = e.runtimeIdle(call.Input)
		} else {
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
	reservation, err := e.responseReservation(call)
	if err != nil {
		return Reply{}, err
	}
	if reservation+1 > frameLimit {
		return Reply{}, capacity("reply-frame")
	}
	if uint64(len(e.transcript))+uint64(len(callBytes))+reservation+2 > e.config.Limits.TranscriptBytes {
		return Reply{}, capacity("transcript-bytes")
	}
	reply := Reply{Type: "reply", ID: call.ID, Output: json.RawMessage(`{}`)}
	result, errno, unsupported, err := e.dispatch(call)
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
	e.transcript = append(e.transcript, callBytes...)
	e.transcript = append(e.transcript, '\n')
	e.transcript = append(e.transcript, replyBytes...)
	e.transcript = append(e.transcript, '\n')
	e.nextCall++
	return reply, nil
}
func (e *Environment) responseReservation(call Call) (uint64, error) {
	if len(call.Input) > frameLimit {
		return 0, capacity("input-frame")
	}
	if err := validateJSON(call.Input); err != nil {
		return 0, err
	}
	// Reserving the maximum buffer before dispatch prevents a capacity result from committing a modeled mutation.
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(call.Input, &fields); err != nil || fields == nil {
		return 0, invalid("operation input")
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
func (e *Environment) dispatch(call Call) (any, uint16, string, error) {
	switch call.Op {
	case "args_sizes_get", "environ_sizes_get", "args_get", "environ_get", "sched_yield":
		if err := decodeInput(call.Input, &struct{}{}); err != nil {
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
		if err := decodeInput(call.Input, &input, "length"); err != nil {
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
		return e.namespaceCall(call)
	}
}
