package wasi

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"math"
	"strings"

	"go.temporal.io/server/tools/gomad/internal/model/choice"
)

const helperTapeChunk = 512

type helperModelConfig struct {
	Seed                   uint64        `json:"seed"`
	Mode                   choice.Mode   `json:"mode"`
	Limit                  uint64        `json:"limit"`
	Diagnostics            bool          `json:"diagnostics"`
	TapeChunks             uint64        `json:"tape_chunks"`
	EpochNanos             uint64        `json:"epoch_nanos"`
	ReadStepNanos          uint64        `json:"read_step_nanos"`
	Now                    uint64        `json:"now"`
	Limits                 helperLimits  `json:"limits"`
	DigestOnly             bool          `json:"digest_only"`
	EntropyKey             [32]byte      `json:"entropy_key"`
	WorkingDirectorySHA256 string        `json:"working_directory_sha256"`
	Arguments              helperStrings `json:"arguments"`
	Environment            helperStrings `json:"environment"`
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
	Now             uint64 `json:"now"`
	Advances        uint64 `json:"advances"`
	TranscriptBytes uint64 `json:"transcript_bytes"`
}
type helperCall struct {
	Call
	LocalState helperLocalState `json:"local_state"`
}
type helperModelResult struct {
	Now              uint64 `json:"now"`
	Advances         uint64 `json:"advances"`
	Calls            uint64 `json:"calls"`
	Records          uint64 `json:"records"`
	Cursor           uint64 `json:"cursor"`
	Configured       bool   `json:"configured"`
	Finished         bool   `json:"finished"`
	Peak             uint32 `json:"peak"`
	Elapsed          uint64 `json:"elapsed"`
	Terminal         []byte `json:"terminal_base64"`
	TranscriptBytes  uint64 `json:"transcript_bytes"`
	TranscriptSHA256 string `json:"transcript_sha256"`
}
type helperEvidence struct{ choice, diagnostic, transcript, stdout, stderr []byte }

type helperReply struct {
	Reply
	LocalState struct {
		Now      uint64 `json:"now"`
		Advances uint64 `json:"advances"`
	} `json:"local_state"`
	Transcript []byte `json:"transcript_base64"`
}

func helperConfig(e *Environment, digestOnly bool) *helperModelConfig {
	r := e.runtime
	var chunks uint64
	if r.tape != nil {
		chunks = uint64((len(r.tape.Decisions) + helperTapeChunk - 1) / helperTapeChunk)
	}
	directory := sha256.Sum256([]byte(e.config.WorkingDirectory))
	return &helperModelConfig{Seed: r.control.Seed, Mode: r.control.Choice.Mode, Limit: r.control.Choice.Limit, Diagnostics: r.control.Diagnostics, TapeChunks: chunks, EpochNanos: e.config.Clock.EpochNanos, ReadStepNanos: e.config.Clock.ReadStepNanos, Now: e.now, Limits: helperLimits{e.config.Limits.Calls, e.config.Limits.TranscriptBytes, e.config.Limits.PendingEvents}, DigestOnly: digestOnly, EntropyKey: e.config.EntropyKey, WorkingDirectorySHA256: hex.EncodeToString(directory[:]), Arguments: helperStringDeclaration(e.config.Args), Environment: helperStringDeclaration(e.config.Environment)}
}

func (e *Environment) helperCallbackReply(owned validatedCall, state helperLocalState) (helperReply, error) {
	call := owned.call
	if e.namespaceOwned && (call.Op == "poll_oneoff" || filesystemCall(call)) {
		return helperReply{}, invalid("helper-owned namespace callback")
	}
	if call.ID < e.nextCall || call.ID > e.config.Limits.Calls || state.Now < e.now || state.Now > math.MaxInt64-e.config.Clock.EpochNanos || state.Advances < e.advances || state.TranscriptBytes < e.transcriptLength {
		return helperReply{}, invalid("helper callback state")
	}
	e.nextCall, e.now, e.advances, e.transcriptLength = call.ID, state.Now, state.Advances, state.TranscriptBytes
	e.transcript = nil
	reply, err := e.handleValidated(owned)
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
	if !sink.environment.helperOwned || sink.started == nil {
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
	case "choice":
		target, limit = &sink.helperEvidence.choice, sink.request.Runtime.Choice.Limit-64
	case "diagnostic":
		if !sink.request.Runtime.Diagnostics {
			return invalid("unexpected helper diagnostics")
		}
		target, limit = &sink.helperEvidence.diagnostic, sink.request.Runtime.Choice.Limit-64
	case "transcript":
		if sink.request.DigestTranscript {
			return invalid("unexpected retained transcript")
		}
		target, limit = &sink.helperEvidence.transcript, sink.request.Config.Limits.TranscriptBytes
	case "stdout", "stderr":
		if !sink.environment.namespaceOwned {
			return invalid("unexpected helper namespace output")
		}
		target, limit = &sink.helperEvidence.stdout, sink.request.Config.Limits.OutputBytes
		other := len(sink.helperEvidence.stderr)
		if frame.Stream == "stderr" {
			target, other = &sink.helperEvidence.stderr, len(sink.helperEvidence.stdout)
		}
		// Earlier appends keep the combined streams within the limit.
		limit -= uint64(other)
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
	if sink.environment.namespaceOwned {
		return append([]byte(nil), sink.helperEvidence.stdout...), append([]byte(nil), sink.helperEvidence.stderr...)
	}
	return sink.environment.Output()
}

func (sink *frameSink) retainPartialHelperEvidence(result *Result) {
	result.PartialHelperTranscript = append([]byte(nil), sink.helperEvidence.transcript...)
	result.PartialHelperChoice = append([]byte(nil), sink.helperEvidence.choice...)
	result.PartialHelperDiagnostic = append([]byte(nil), sink.helperEvidence.diagnostic...)
}

func (sink *frameSink) collectHelper(result *Result) error {
	model := sink.terminal.LocalModel
	if model == nil {
		if sink.started != nil {
			return invalid("missing helper model terminal")
		}
		return nil
	}
	e := sink.environment
	if model.Calls < e.nextCall-1 || model.Calls > e.config.Limits.Calls || model.Now < e.now || model.Now > math.MaxInt64-e.config.Clock.EpochNanos || model.Advances < e.advances || model.TranscriptBytes < e.transcriptLength || model.Records != uint64(len(sink.helperEvidence.choice))/96 || len(sink.helperEvidence.choice)%96 != 0 {
		return invalid("helper model terminal counts")
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
		if tape := e.runtime.tape; tape != nil && model.Cursor != uint64(len(tape.Decisions)) {
			if model.Cursor > uint64(len(tape.Decisions)) {
				return invalid("interrupted tape cursor")
			}
			expected := tape.Decisions[model.Cursor]
			return &RuntimeReplayError{Divergence: choice.Divergence{Ordinal: model.Cursor, Reason: choice.DivergenceTapeUnconsumed, Expected: &expected, TapeRecords: uint64(len(tape.Decisions))}}
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
	result.ChoiceTrace, err = choice.DecodeStoredTrace(choice.Profile, sink.helperEvidence.choice, choice.TerminalMetadata{State: choice.TerminalComplete, Limit: sink.request.Runtime.Choice.Limit, Records: model.Records, SHA256: sha256.Sum256(sink.helperEvidence.choice)})
	if err != nil {
		return err
	}
	result.ChoiceTrace.Summary.PeakGoroutines = model.Peak
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
