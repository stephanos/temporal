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
	Seed          uint64       `json:"seed"`
	Mode          choice.Mode  `json:"mode"`
	Limit         uint64       `json:"limit"`
	Diagnostics   bool         `json:"diagnostics"`
	TapeChunks    uint64       `json:"tape_chunks"`
	EpochNanos    uint64       `json:"epoch_nanos"`
	ReadStepNanos uint64       `json:"read_step_nanos"`
	Now           uint64       `json:"now"`
	Limits        helperLimits `json:"limits"`
	DigestOnly    bool         `json:"digest_only"`
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
type helperEvidence struct{ choice, diagnostic, transcript []byte }

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
	return &helperModelConfig{Seed: r.control.Seed, Mode: r.control.Choice.Mode, Limit: r.control.Choice.Limit, Diagnostics: r.control.Diagnostics, TapeChunks: chunks, EpochNanos: e.config.Clock.EpochNanos, ReadStepNanos: e.config.Clock.ReadStepNanos, Now: e.now, Limits: helperLimits{e.config.Limits.Calls, e.config.Limits.TranscriptBytes, e.config.Limits.PendingEvents}, DigestOnly: digestOnly}
}

func (e *Environment) helperCallbackReply(owned validatedCall, state helperLocalState) (helperReply, error) {
	call := owned.call
	if call.ID < e.nextCall || call.ID > e.config.Limits.Calls || state.Now < e.now || state.Now > math.MaxInt64-e.config.Clock.EpochNanos || state.Advances < e.advances || state.TranscriptBytes < e.transcriptLength {
		return helperReply{}, invalid("helper callback state")
	}
	e.nextCall, e.now, e.advances, e.transcriptLength = call.ID, state.Now, state.Advances, state.TranscriptBytes
	e.transcript = nil
	reply, err := e.handleValidated(owned)
	if err != nil {
		return helperReply{}, err
	}
	callBytes, err := json.Marshal(call)
	if err != nil {
		return helperReply{}, err
	}
	replyBytes, err := json.Marshal(reply)
	if err != nil {
		return helperReply{}, err
	}
	canonical := append(append(append(callBytes, '\n'), replyBytes...), '\n')
	if strings.HasPrefix(call.Op, "runtime_") {
		canonical = []byte{}
	}
	e.transcriptLength = state.TranscriptBytes + uint64(len(canonical))
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
	default:
		return invalid("helper evidence stream")
	}
	if uint64(len(*target))+uint64(len(frame.Data)) > limit {
		return capacity("helper evidence bytes")
	}
	*target = append(*target, frame.Data...)
	return nil
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
