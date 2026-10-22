package wasi

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"strings"

	"go.temporal.io/server/tools/gomad/internal/model/choice"
)

const helperReplaySchema = "gomad.replay-trace/v1"

type helperReplayRequest struct {
	Schema            string                   `json:"schema"`
	ExecutionIdentity choice.ExecutionIdentity `json:"execution_identity"`
	TraceSHA256       string                   `json:"trace_sha256"`
	TapeSHA256        string                   `json:"tape_sha256"`
	Decisions         uint64                   `json:"decisions"`
	TraceBytes        uint64                   `json:"trace_bytes"`
	TraceFrames       uint64                   `json:"trace_frames"`
	PrefixControl     *choice.PrefixControl    `json:"prefix_control,omitempty"`
}

type helperModelConfig struct {
	Seed                   uint64               `json:"seed"`
	Mode                   choice.Mode          `json:"mode"`
	Limit                  uint64               `json:"limit"`
	Diagnostics            bool                 `json:"diagnostics"`
	Replay                 *helperReplayRequest `json:"replay,omitempty"`
	EpochNanos             uint64               `json:"epoch_nanos"`
	ReadStepNanos          uint64               `json:"read_step_nanos"`
	Now                    uint64               `json:"now"`
	Limits                 helperLimits         `json:"limits"`
	DigestOnly             bool                 `json:"digest_only"`
	EntropyKey             [32]byte             `json:"entropy_key"`
	WorkingDirectorySHA256 string               `json:"working_directory_sha256"`
	Arguments              helperStrings        `json:"arguments"`
	Environment            helperStrings        `json:"environment"`
	Split                  bool                 `json:"split,omitempty"`
	Inspection             json.RawMessage      `json:"inspection,omitempty"`
}
type helperStrings struct {
	Bytes  uint64 `json:"bytes"`
	Count  uint64 `json:"count"`
	Chunks uint64 `json:"chunks"`
}

func packedHelperStrings(values []string) []byte {
	var packed []byte
	for _, value := range values {
		packed = append(packed, value...)
		packed = append(packed, 0)
	}
	return packed
}
func helperStringDeclaration(values []string) helperStrings {
	var size uint64
	for _, value := range values {
		size += uint64(len(value)) + 1
	}
	return helperStrings{size, uint64(len(values)), (size + (64 << 10) - 1) / (64 << 10)}
}
func appendHelperInputs(initial []byte, e *Environment) ([]byte, error) {
	for index, values := range [][]string{e.config.Args, e.config.Environment} {
		stream := "arguments"
		if index == 1 {
			stream = "environment"
		}
		packed := packedHelperStrings(values)
		for sequence := uint64(0); len(packed) > 0; sequence++ {
			count := min(len(packed), 64<<10)
			frame, err := json.Marshal(struct {
				Type     string `json:"type"`
				Stream   string `json:"stream"`
				Sequence uint64 `json:"sequence"`
				Data     []byte `json:"data_base64"`
			}{"model_input", stream, sequence, packed[:count]})
			if err != nil {
				return nil, err
			}
			initial = append(initial, frame...)
			initial = append(initial, '\n')
			packed = packed[count:]
		}
	}
	initial = append(initial, []byte("{\"type\":\"model_inputs_end\"}\n")...)
	return initial, nil
}

type helperLimits struct {
	Calls           uint64 `json:"calls"`
	TranscriptBytes uint64 `json:"transcript_bytes"`
	PendingEvents   uint64 `json:"pending_events"`
}
type helperLocalState struct {
	Now                uint64 `json:"now"`
	Advances           uint64 `json:"advances"`
	TranscriptBytes    uint64 `json:"transcript_bytes"`
	OfferedOutputBytes uint64 `json:"offered_output_bytes,omitempty"`
}
type helperCall struct {
	Call
	LocalState helperLocalState `json:"local_state"`
}
type helperModelResult struct {
	Now              uint64           `json:"now"`
	Advances         uint64           `json:"advances"`
	Calls            uint64           `json:"calls"`
	Records          uint64           `json:"records"`
	TraceSHA256      string           `json:"trace_sha256"`
	BranchingRecords uint64           `json:"branching_records"`
	TapeRecords      uint64           `json:"tape_records"`
	TapeSHA256       string           `json:"tape_sha256"`
	NextExpected     *choice.Decision `json:"next_expected"`
	Cursor           uint64           `json:"cursor"`
	Configured       bool             `json:"configured"`
	Finished         bool             `json:"finished"`
	Peak             uint32           `json:"peak"`
	Elapsed          uint64           `json:"elapsed"`
	Terminal         []byte           `json:"terminal_base64"`
	TranscriptBytes  uint64           `json:"transcript_bytes"`
	TranscriptSHA256 string           `json:"transcript_sha256"`
	InspectionSealed bool             `json:"inspection_sealed"`
}
type helperEvidence struct {
	choice, diagnostic, transcript, stdout, stderr []byte
	storage, storageState, storageImage            []byte
	inspection, debug                              []byte
}

type helperReply struct {
	Reply
	LocalState struct {
		Now      uint64 `json:"now"`
		Advances uint64 `json:"advances"`
	} `json:"local_state"`
	Transcript []byte `json:"transcript_base64"`
	Revocable  bool   `json:"revocable,omitempty"`
}

func helperConfig(e *Environment, digestOnly bool) *helperModelConfig {
	r := e.runtime
	var replay *helperReplayRequest
	if request := r.control.Choice; request.Mode != choice.ModeRecord {
		frames := uint64(len(request.Trace)) / 65536
		if len(request.Trace)%65536 != 0 {
			frames++
		}
		replay = &helperReplayRequest{Schema: helperReplaySchema, ExecutionIdentity: request.ExecutionIdentity, TraceSHA256: hex.EncodeToString(request.TraceSHA256[:]), TapeSHA256: hex.EncodeToString(request.TapeSHA256[:]), Decisions: request.Decisions, TraceBytes: uint64(len(request.Trace)), TraceFrames: frames, PrefixControl: request.PrefixControl}
	}
	directory := sha256.Sum256([]byte(e.config.WorkingDirectory))
	return &helperModelConfig{Seed: r.control.Seed, Mode: r.control.Choice.Mode, Limit: r.control.Choice.Limit, Diagnostics: r.control.Diagnostics, Replay: replay, EpochNanos: e.config.Clock.EpochNanos, ReadStepNanos: e.config.Clock.ReadStepNanos, Now: e.now, Limits: helperLimits{e.config.Limits.Calls, e.config.Limits.TranscriptBytes, e.config.Limits.PendingEvents}, DigestOnly: digestOnly, EntropyKey: e.config.EntropyKey, WorkingDirectorySHA256: hex.EncodeToString(directory[:]), Arguments: helperStringDeclaration(e.config.Args), Environment: helperStringDeclaration(e.config.Environment), Inspection: r.control.Inspection}
}

func (e *Environment) helperCallbackReply(owned validatedCall, state helperLocalState) (helperReply, error) {
	return e.helperCallbackReplyWith(owned, state, e.handleValidated)
}

// guestOutputWrite reports a guest write to fd 1 or 2, which split ownership
// keeps remote so broker output boundaries observe it in order.
func guestOutputWrite(call Call) bool {
	if call.Op != "fd_write" && call.Op != "fd_pwrite" {
		return false
	}
	var input struct {
		FD uint32 `json:"fd"`
	}
	return json.Unmarshal(call.Input, &input) == nil && (input.FD == 1 || input.FD == 2)
}

func (e *Environment) helperCallbackReplyWith(owned validatedCall, state helperLocalState, handle func(validatedCall) (Reply, error)) (helperReply, error) {
	call := owned.call
	if e.namespaceOwned && (call.Op == "poll_oneoff" || filesystemCall(call)) && (!e.splitOwned || !guestOutputWrite(call)) {
		return helperReply{}, invalid("helper-owned namespace callback")
	}
	if e.splitOwned && !strings.HasPrefix(call.Op, "runtime_") && !guestOutputWrite(call) {
		return helperReply{}, invalid("split-owned local callback")
	}
	if call.ID < e.nextCall || call.ID > e.config.Limits.Calls || state.Now < e.now || state.Now > math.MaxInt64-e.config.Clock.EpochNanos || state.Advances < e.advances || state.TranscriptBytes < e.transcriptLength {
		return helperReply{}, invalid("helper callback state")
	}
	if e.splitOwned {
		if !e.namespaceOwned || state.OfferedOutputBytes < e.offeredOutput || state.OfferedOutputBytes > e.config.Limits.OutputBytes {
			return helperReply{}, invalid("helper callback output state")
		}
	} else if state.OfferedOutputBytes != 0 {
		return helperReply{}, invalid("helper callback output state")
	}
	e.nextCall, e.now, e.advances, e.transcriptLength = call.ID, state.Now, state.Advances, state.TranscriptBytes
	if e.splitOwned {
		e.offeredOutput = state.OfferedOutputBytes
	}
	e.transcript = nil
	reply, err := handle(owned)
	if err != nil {
		return helperReply{}, err
	}
	e.closeClockRun()
	canonical := e.transcript
	if strings.HasPrefix(call.Op, "runtime_") {
		canonical = []byte{}
	}
	e.transcript = nil
	return helperReply{Reply: reply, LocalState: struct {
		Now      uint64 `json:"now"`
		Advances uint64 `json:"advances"`
	}{e.now, e.advances}, Transcript: canonical}, nil
}

func (sink *frameSink) receiveHelperEvidence(input *validatedFrame) error {
	if !sink.environment.helperOwned && !sink.environment.splitOwned || sink.started == nil {
		return invalid("unexpected helper evidence")
	}
	var frame struct {
		Type   string `json:"type"`
		Stream string `json:"stream"`
		Data   []byte `json:"data_base64"`
	}
	if err := input.decode(&frame, "type", "stream", "data_base64"); err != nil {
		return err
	}
	if len(frame.Data) == 0 || len(frame.Data) > 64<<10 {
		return invalid("helper evidence chunk")
	}
	var target *[]byte
	var limit uint64
	switch frame.Stream {
	case "inspection":
		target, limit = &sink.helperEvidence.inspection, 64<<20
	case "debug":
		if sink.request.Runtime == nil || len(sink.request.Runtime.Inspection) == 0 {
			return invalid("debug output during recording")
		}
		target, limit = &sink.helperEvidence.debug, 8<<20
	case "choice":
		if sink.environment.splitOwned {
			return invalid("unexpected split-owned choice evidence")
		}
		target, limit = &sink.helperEvidence.choice, sink.request.Runtime.Choice.Limit-64
	case "diagnostic":
		if !sink.request.Runtime.Diagnostics || sink.environment.splitOwned {
			return invalid("unexpected helper diagnostics")
		}
		target, limit = &sink.helperEvidence.diagnostic, sink.request.Runtime.Choice.Limit-64
	case "transcript":
		if sink.request.DigestTranscript {
			return invalid("unexpected retained transcript")
		}
		target, limit = &sink.helperEvidence.transcript, sink.request.Config.Limits.TranscriptBytes
	case "stdout", "stderr":
		if !sink.environment.namespaceOwned || sink.environment.splitOwned {
			return invalid("unexpected helper namespace output")
		}
		target, limit = &sink.helperEvidence.stdout, sink.request.Config.Limits.OutputBytes
		other := len(sink.helperEvidence.stderr)
		if frame.Stream == "stderr" {
			target, other = &sink.helperEvidence.stderr, len(sink.helperEvidence.stdout)
		}
		// Earlier appends keep the combined streams within the limit.
		limit -= uint64(other)
	case "storage", "storage_state", "storage_image":
		if !sink.environment.namespaceOwned || sink.request.Config.Storage == nil {
			return invalid("unexpected helper storage evidence")
		}
		switch frame.Stream {
		case "storage":
			target, limit = &sink.helperEvidence.storage, sink.request.Config.Limits.TranscriptBytes
		case "storage_state":
			target = &sink.helperEvidence.storageState
			limit = storageStateStreamLimit(sink.request, *target, frame.Data)
		default:
			if !sink.request.ReturnStorageImage {
				return invalid("unrequested helper storage image")
			}
			target, limit = &sink.helperEvidence.storageImage, storageImageLimit(sink.request.Config)
		}
	default:
		return invalid("helper evidence stream")
	}
	if uint64(len(*target))+uint64(len(frame.Data)) > limit {
		if frame.Stream == "stdout" || frame.Stream == "stderr" {
			return capacity("offered-output-bytes")
		}
		return capacity("helper evidence bytes")
	}
	*target = append(*target, frame.Data...)
	if frame.Stream == "stdout" && sink.environment.mirror != nil {
		_, _ = sink.environment.mirror.Write(frame.Data)
	}
	return nil
}

func (sink *frameSink) guestOutput() ([]byte, []byte) {
	if sink.environment.namespaceOwned && !sink.environment.splitOwned {
		return append([]byte(nil), sink.helperEvidence.stdout...), append([]byte(nil), sink.helperEvidence.stderr...)
	}
	return sink.environment.Output()
}

func (sink *frameSink) retainPartialHelperEvidence(result *Result) {
	result.PartialHelperTranscript = append([]byte(nil), sink.helperEvidence.transcript...)
	result.PartialHelperChoice = append([]byte(nil), sink.helperEvidence.choice...)
	result.PartialHelperDiagnostic = append([]byte(nil), sink.helperEvidence.diagnostic...)
}

// collectSplit takes the WASI transcript and call accounting of a split-owned
// broker run from the helper; runtime evidence stays with the Go session.
func (sink *frameSink) collectSplit(result *Result) error {
	model := sink.terminal.LocalModel
	if model == nil {
		if sink.started != nil {
			return invalid("missing helper model terminal")
		}
		return nil
	}
	e := sink.environment
	// A broker call interrupted after adopting helper state may already have
	// raised Go's clock to broker time; the helper never observed it.
	interrupted := sink.terminal.Termination == "crash" || sink.terminal.Termination == "stop"
	if model.Calls < e.nextCall-1 || model.Calls > e.config.Limits.Calls || model.Now < e.now && !interrupted || model.Now > math.MaxInt64-e.config.Clock.EpochNanos || model.Advances < e.advances || model.TranscriptBytes < e.transcriptLength || model.Records != 0 || model.Configured || model.Finished || len(sink.helperEvidence.choice) != 0 {
		return invalid(fmt.Sprintf("split helper terminal counts calls=%d/%d now=%d/%d advances=%d/%d transcript=%d/%d records=%d", model.Calls, e.nextCall, model.Now, e.now, model.Advances, e.advances, model.TranscriptBytes, e.transcriptLength, model.Records))
	}
	digest, err := hex.DecodeString(model.TranscriptSHA256)
	if err != nil || len(digest) != sha256.Size || hex.EncodeToString(digest) != model.TranscriptSHA256 {
		return invalid("helper transcript digest")
	}
	if !sink.request.DigestTranscript {
		if model.TranscriptBytes != uint64(len(sink.helperEvidence.transcript)) || sha256.Sum256(sink.helperEvidence.transcript) != [32]byte(digest) {
			return invalid("helper transcript bytes")
		}
		if _, err := DecodeTranscript(sink.helperEvidence.transcript); err != nil {
			return err
		}
	}
	result.Calls, result.TimeAdvances = model.Calls, model.Advances
	result.Transcript = sink.helperEvidence.transcript
	copy(result.TranscriptSHA256[:], digest)
	return nil
}

func inspectionTermination(termination string) bool {
	switch termination {
	case "inspected_prefix", "instrumentation_divergence", "debug_error", "probe_fuel_limit":
		return true
	default:
		return false
	}
}

func validateHelperInputTape(model *helperModelResult, request *choice.Request) error {
	if request.Mode == choice.ModeRecord {
		if model.TapeRecords != 0 || model.TapeSHA256 != "" || model.Cursor != 0 || model.NextExpected != nil {
			return invalid("record terminal has replay state")
		}
		return nil
	}
	if model.TapeRecords != request.Decisions || model.TapeSHA256 != hex.EncodeToString(request.TapeSHA256[:]) || model.Cursor > model.TapeRecords {
		return invalid("helper input tape identity/count/cursor")
	}
	if model.Cursor == model.TapeRecords {
		if model.NextExpected != nil {
			return invalid("helper next decision after tape end")
		}
		return nil
	}
	next := model.NextExpected
	if next == nil || next.Ordinal != model.Cursor || next.Kind != choice.KindRunnable && next.Kind != choice.KindSelectPoll || next.Alternatives < 2 || next.Selected >= next.Alternatives || next.SiteMissing && next.SiteOffset != 0 || next.AlternativeSetDigest == ([32]byte{}) {
		return invalid("helper next unconsumed decision")
	}
	var override *choice.RankOverride
	if control := request.PrefixControl; control != nil {
		override = control.Override
	}
	wantOverride := override != nil && override.Ordinal == model.Cursor
	if next.RankOverride != wantOverride || next.RankOverride && (next.SelectedIdentity != ([32]byte{}) || next.Selected != override.Rank) || !next.RankOverride && next.SelectedIdentity == ([32]byte{}) {
		return invalid("helper next decision identity/rank")
	}
	return nil
}

func (sink *frameSink) collectHelper(result *Result) error {
	result.InspectionMetadata = append([]byte(nil), sink.helperEvidence.inspection...)
	result.DebugOutput = append([]byte(nil), sink.helperEvidence.debug...)
	model := sink.terminal.LocalModel
	if model == nil {
		if sink.started != nil {
			return invalid("missing helper model terminal")
		}
		return nil
	}
	e := sink.environment
	if inspectionTermination(result.Termination) {
		if sink.request.Runtime == nil || len(sink.request.Runtime.Inspection) == 0 || model.Finished || result.Termination == "inspected_prefix" && !model.InspectionSealed {
			return invalid("unsolicited or unsealed inspected-prefix terminal")
		}
		result.PartialHelperChoice = append([]byte(nil), sink.helperEvidence.choice...)
		result.PartialHelperTranscript = append([]byte(nil), sink.helperEvidence.transcript...)
		result.VirtualTimeElapsedNanos = model.Now
		return nil
	}
	if model.Calls < e.nextCall-1 || model.Calls > e.config.Limits.Calls || model.Now < e.now || model.Now > math.MaxInt64-e.config.Clock.EpochNanos || model.Advances < e.advances || model.TranscriptBytes < e.transcriptLength || model.Records != uint64(len(sink.helperEvidence.choice))/96 || len(sink.helperEvidence.choice)%96 != 0 {
		return invalid("helper model terminal counts")
	}
	traceDigest, err := hex.DecodeString(model.TraceSHA256)
	if err != nil || len(traceDigest) != sha256.Size || hex.EncodeToString(traceDigest) != model.TraceSHA256 || sha256.Sum256(sink.helperEvidence.choice) != [32]byte(traceDigest) || model.BranchingRecords > model.Records {
		return invalid("helper choice trace identity/counts")
	}
	input := e.runtime.control.Choice
	if e.splitOwned {
		if err := validateHelperInputTape(model, &choice.Request{Mode: choice.ModeRecord}); err != nil {
			return err
		}
	} else if err := validateHelperInputTape(model, input); err != nil {
		return err
	}
	if input.Mode != choice.ModeRecord {
		result.InputTapeSHA256, result.InputDecisions = input.TapeSHA256, input.Decisions
	}
	digest, err := hex.DecodeString(model.TranscriptSHA256)
	if err != nil || len(digest) != sha256.Size || hex.EncodeToString(digest) != model.TranscriptSHA256 {
		return invalid("helper transcript digest")
	}
	if !sink.request.DigestTranscript {
		if model.TranscriptBytes != uint64(len(sink.helperEvidence.transcript)) || sha256.Sum256(sink.helperEvidence.transcript) != [32]byte(digest) {
			return invalid("helper transcript bytes")
		}
		if _, err := DecodeTranscript(sink.helperEvidence.transcript); err != nil {
			return err
		}
	}
	result.Calls, result.TimeAdvances = model.Calls, model.Advances
	result.Transcript = sink.helperEvidence.transcript
	copy(result.TranscriptSHA256[:], digest)
	result.ChoiceDivergence = sink.terminal.Divergence
	if result.Termination == "storage_crash" {
		if model.Finished {
			return invalid("completed runtime on storage interruption")
		}
		if !e.splitOwned && input.Mode != choice.ModeRecord && model.Cursor < model.TapeRecords {
			return &RuntimeReplayError{Divergence: choice.Divergence{Ordinal: model.Cursor, Reason: choice.DivergenceTapeUnconsumed, Expected: model.NextExpected, TapeRecords: model.TapeRecords}}
		}
		if e.splitOwned && e.runtime.tape != nil && e.runtime.cursor < uint64(len(e.runtime.tape.Decisions)) {
			expected := e.runtime.tape.Decisions[e.runtime.cursor]
			return &RuntimeReplayError{Divergence: choice.Divergence{Ordinal: e.runtime.cursor, Reason: choice.DivergenceTapeUnconsumed, Expected: &expected, TapeRecords: uint64(len(e.runtime.tape.Decisions))}}
		}
		if _, err := choice.ProjectInterruptedReplayPlan(sink.helperEvidence.choice, sink.request.Runtime.Choice.Limit, sink.request.Runtime.Choice.ExecutionIdentity); err != nil {
			return err
		}
		result.RuntimeInterrupted = true
		result.InterruptedChoiceBytes = sink.helperEvidence.choice
		result.InterruptedDiagnosticBytes = sink.helperEvidence.diagnostic
	}
	if result.Termination != "exit" {
		return nil
	}
	if !model.Configured || !model.Finished || len(model.Terminal) != 16 {
		return invalid("helper runtime unfinished")
	}
	copy(result.runtimeTerminal[:], model.Terminal)
	result.VirtualTimeElapsedNanos = model.Elapsed
	result.ChoiceTrace = choice.Trace{Version: choice.Version3, Bytes: sink.helperEvidence.choice, SHA256: [32]byte(traceDigest), Summary: choice.Summary{Terminal: choice.TerminalComplete, Records: model.Records, Branching: model.BranchingRecords, PeakGoroutines: model.Peak}}
	if input.Mode == choice.ModeRecord {
		result.ChoiceTrace, err = choice.DecodeStoredTrace(choice.Profile, sink.helperEvidence.choice, choice.TerminalMetadata{State: choice.TerminalComplete, Limit: input.Limit, Records: model.Records, SHA256: [32]byte(traceDigest)})
		if err != nil {
			return err
		}
		if result.ChoiceTrace.Summary.Branching != model.BranchingRecords {
			return invalid("helper record branching count")
		}
		result.ChoiceTrace.Summary.PeakGoroutines = model.Peak
	}
	if sink.request.Runtime.Diagnostics {
		if len(sink.helperEvidence.diagnostic) != len(sink.helperEvidence.choice) {
			return invalid("helper diagnostic count")
		}
		limit, err := choice.DiagnosticLimit(sink.request.Runtime.Choice.Limit)
		if err != nil {
			return err
		}
		records := make([]choice.DiagnosticRecord, model.Records)
		for i := range records {
			records[i].Ordinal = uint64(i)
		}
		trace, err := choice.BuildDiagnosticTrace(records, limit)
		if err != nil {
			return err
		}
		copy(trace.Bytes[len(trace.Bytes)-len(sink.helperEvidence.diagnostic):], sink.helperEvidence.diagnostic)
		result.DiagnosticTrace, err = choice.DecodeDiagnosticTrace(trace.Bytes)
		if err != nil {
			return err
		}
	}
	return nil
}
