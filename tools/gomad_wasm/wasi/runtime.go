package wasi

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"slices"

	"go.temporal.io/server/tools/gomad3/choice"
	runnerbackend "go.temporal.io/server/tools/gomad3/runner/backend"
)

const CooperativeProfile = "gomad3.wasi-cooperative/v1"

type RuntimeControl struct {
	Seed        uint64
	Choice      *runnerbackend.ChoiceRequest
	Diagnostics bool
}

type RuntimeReplayError struct{ Divergence choice.Divergence }

func (e *RuntimeReplayError) Error() string {
	return fmt.Sprintf("WASM choice %d: %s", e.Divergence.Ordinal, choice.DivergenceReasonName(e.Divergence.Reason))
}

type runtimeSession struct {
	control              RuntimeControl
	tape                 *choice.ReplayPlan
	cursor               uint64
	records              []choice.Record
	diagnostics          []byte
	configured, finished bool
	peak                 uint32
	elapsed              uint64
	terminal             [16]byte
}

func newRuntimeSession(control *RuntimeControl) (*runtimeSession, error) {
	if control == nil || control.Choice == nil {
		return nil, invalid("cooperative choice control required")
	}
	request := *control.Choice
	if err := choice.ValidateExecutionIdentity(request.ExecutionIdentity); err != nil {
		return nil, err
	}
	if err := choice.ValidateTraceLimit(request.Limit); err != nil {
		return nil, err
	}
	var tape *choice.ReplayPlan
	switch request.Mode {
	case choice.ModeRecord:
		if request.Tape != nil {
			return nil, invalid("record mode has a tape")
		}
	case choice.ModeReplay, choice.ModePrefix:
		if request.Tape == nil {
			return nil, invalid("forced mode has no tape")
		}
		validate := choice.ValidateReplayPlan
		if request.Mode == choice.ModePrefix {
			validate = choice.ValidatePrefixReplayPlan
		}
		validated, err := validate(*request.Tape, request.ExecutionIdentity)
		if err != nil {
			return nil, err
		}
		if !slices.Equal(validated.Decisions, request.Tape.Decisions) || !slices.Equal(validated.Readiness, request.Tape.Readiness) {
			return nil, invalid("tape projection differs from canonical bytes")
		}
		tape = &validated
	default:
		return nil, invalid("cooperative choice mode")
	}
	if control.Diagnostics {
		if _, err := choice.DiagnosticLimit(request.Limit); err != nil {
			return nil, err
		}
	}
	cloned := *control
	cloned.Choice = &request
	return &runtimeSession{control: cloned, tape: tape}, nil
}

func (session *runtimeSession) handle(op string, input json.RawMessage) (bytesOutput, error) {
	var request bytesOutput
	if err := decodeInput(input, &request, "data_base64"); err != nil {
		return bytesOutput{}, err
	}
	data := slices.Clone(request.Data)
	if op == "runtime_config" {
		if session.configured || len(data) != 16 || !allZero(data) {
			return bytesOutput{}, invalid("runtime bootstrap")
		}
		binary.BigEndian.PutUint32(data, 1)
		if session.control.Diagnostics {
			binary.BigEndian.PutUint32(data[4:], 1)
		}
		binary.BigEndian.PutUint64(data[8:], session.control.Seed)
		session.configured = true
		return bytesOutput{Data: data}, nil
	}
	if !session.configured || session.finished {
		return bytesOutput{}, invalid("runtime operation outside active lifetime")
	}
	switch op {
	case "runtime_decision":
		if len(data) < 32 || binary.BigEndian.Uint32(data) != 1 {
			return bytesOutput{}, invalid("runtime decision header")
		}
		rawKind := binary.BigEndian.Uint32(data[4:])
		kind, site := choice.Kind(rawKind), binary.BigEndian.Uint64(data[8:])
		flags, count := binary.BigEndian.Uint32(data[16:]), binary.BigEndian.Uint32(data[20:])
		selected, value := binary.BigEndian.Uint32(data[24:]), binary.BigEndian.Uint32(data[28:])
		diagnosticBytes := 0
		if session.control.Diagnostics {
			diagnosticBytes = 96
		}
		if rawKind != uint32(kind) || (kind != choice.KindRunnable && kind != choice.KindSelectPoll) || flags&^uint32(choice.FlagSiteMissing) != 0 || count < 2 || count > 256 || selected >= count || len(data) != 32+int(count)*32+diagnosticBytes {
			return bytesOutput{}, invalid("runtime decision alternatives or layout")
		}
		alternatives := make([][32]byte, count)
		for i := range alternatives {
			copy(alternatives[i][:], data[32+32*i:64+32*i])
		}
		observed, err := choice.CanonicalDecision(uint64(len(session.records)), kind, site, flags != 0, alternatives, alternatives[selected], value)
		if err != nil {
			return bytesOutput{}, fmt.Errorf("%w: %w", invalid("runtime decision identities"), err)
		}
		forced := session.control.Choice.Mode == choice.ModeReplay || session.control.Choice.Mode == choice.ModePrefix && session.cursor < uint64(len(session.tape.Decisions))
		if forced {
			if session.cursor >= uint64(len(session.tape.Decisions)) {
				return bytesOutput{}, session.diverge(choice.DivergenceTapeExhausted, nil, &observed)
			}
			expected := session.tape.Decisions[session.cursor]
			for _, check := range []struct {
				failed bool
				reason choice.DivergenceReason
			}{{expected.Kind != observed.Kind, choice.DivergenceKind}, {expected.SiteOffset != observed.SiteOffset || expected.SiteMissing != observed.SiteMissing, choice.DivergenceSite}, {expected.Alternatives != observed.Alternatives, choice.DivergenceAlternatives}, {expected.AlternativeSetDigest != observed.AlternativeSetDigest, choice.DivergenceAlternativeSet}} {
				if check.failed {
					return bytesOutput{}, session.diverge(check.reason, &expected, &observed)
				}
			}
			ordered := slices.Clone(alternatives)
			slices.SortFunc(ordered, compareIdentity)
			if expected.Selected >= uint32(len(ordered)) || !expected.RankOverride && ordered[expected.Selected] != expected.SelectedIdentity {
				return bytesOutput{}, session.diverge(choice.DivergenceSelected, &expected, &observed)
			}
			selectedIdentity := ordered[expected.Selected]
			selected = uint32(slices.Index(alternatives, selectedIdentity))
			observed.Selected, observed.SelectedIdentity = expected.Selected, selectedIdentity
			session.cursor++
		}
		if err := session.append(observed.Record(), data[32+32*int(count):]); err != nil {
			return bytesOutput{}, err
		}
		binary.BigEndian.PutUint32(data[24:], selected)
	case "runtime_observation":
		length := 96
		if session.control.Diagnostics {
			length += 96
		}
		if len(data) != length {
			return bytesOutput{}, invalid("runtime observation layout")
		}
		payload := data[:96]
		// The public decoder validates observations and their origin against the preceding records.
		existing, err := choice.BuildTrace(session.records, choice.TerminalComplete)
		if err != nil {
			return bytesOutput{}, err
		}
		combined := append(slices.Clone(existing.Bytes), payload...)
		trace, err := choice.DecodeStoredTrace(choice.Profile, combined, choice.TerminalMetadata{State: choice.TerminalComplete, Limit: session.control.Choice.Limit, Records: uint64(len(session.records) + 1), SHA256: sha256.Sum256(combined)})
		if err != nil || trace.Records[len(trace.Records)-1].Kind != choice.KindSelectResult {
			return bytesOutput{}, invalid("runtime select observation")
		}
		if err := session.append(trace.Records[len(trace.Records)-1], data[96:]); err != nil {
			return bytesOutput{}, err
		}
	case "runtime_finish":
		if len(data) != 16 || binary.BigEndian.Uint32(data) != 1 {
			return bytesOutput{}, invalid("runtime finish layout")
		}
		if session.tape != nil && session.cursor != uint64(len(session.tape.Decisions)) {
			expected := session.tape.Decisions[session.cursor]
			return bytesOutput{}, session.diverge(choice.DivergenceTapeUnconsumed, &expected, nil)
		}
		session.peak = binary.BigEndian.Uint32(data[4:])
		session.elapsed = binary.BigEndian.Uint64(data[8:])
		copy(session.terminal[:], data)
		session.finished = true
	default:
		return bytesOutput{}, invalid("runtime operation")
	}
	return bytesOutput{Data: data}, nil
}

func compareIdentity(a, b [32]byte) int {
	for i := range a {
		if a[i] < b[i] {
			return -1
		}
		if a[i] > b[i] {
			return 1
		}
	}
	return 0
}

func (session *runtimeSession) diverge(reason choice.DivergenceReason, expected, observed *choice.Decision) error {
	return &RuntimeReplayError{Divergence: choice.Divergence{Ordinal: session.cursor, Reason: reason, Expected: expected, Observed: observed, TapeRecords: uint64(len(session.tape.Decisions))}}
}

func (session *runtimeSession) append(record choice.Record, diagnostic []byte) error {
	if 64+uint64(len(session.records)+1)*96 > session.control.Choice.Limit {
		return capacity("runtime-choice-bytes")
	}
	if session.control.Diagnostics && len(diagnostic) != 96 || !session.control.Diagnostics && len(diagnostic) != 0 {
		return invalid("runtime diagnostic layout")
	}
	session.records = append(session.records, record)
	session.diagnostics = append(session.diagnostics, diagnostic...)
	return nil
}

func (session *runtimeSession) collect() (choice.Trace, choice.DiagnosticTrace, error) {
	if !session.finished {
		return choice.Trace{}, choice.DiagnosticTrace{}, invalid("runtime trace unfinished")
	}
	trace, err := choice.BuildTrace(session.records, choice.TerminalComplete)
	if err != nil {
		return choice.Trace{}, choice.DiagnosticTrace{}, err
	}
	trace.Summary.PeakGoroutines = session.peak
	var diagnostic choice.DiagnosticTrace
	if session.control.Diagnostics {
		limit, err := choice.DiagnosticLimit(session.control.Choice.Limit)
		if err != nil {
			return choice.Trace{}, choice.DiagnosticTrace{}, err
		}
		records := make([]choice.DiagnosticRecord, len(session.records))
		for i := range records {
			records[i].Ordinal = uint64(i)
		}
		diagnostic, err = choice.BuildDiagnosticTrace(records, limit)
		if err != nil {
			return choice.Trace{}, choice.DiagnosticTrace{}, err
		}
		copy(diagnostic.Bytes[len(diagnostic.Bytes)-len(session.diagnostics):], session.diagnostics)
		diagnostic, err = choice.DecodeDiagnosticTrace(diagnostic.Bytes)
		if err != nil {
			return choice.Trace{}, choice.DiagnosticTrace{}, err
		}
	}
	return trace, diagnostic, nil
}

func allZero(data []byte) bool {
	for _, b := range data {
		if b != 0 {
			return false
		}
	}
	return true
}

func (e *Environment) runtimeIdle(input json.RawMessage) (bytesOutput, error) {
	if !e.runtime.configured || e.runtime.finished {
		return bytesOutput{}, invalid("runtime idle outside active lifetime")
	}
	var request bytesOutput
	if err := decodeInput(input, &request, "data_base64"); err != nil {
		return bytesOutput{}, err
	}
	data := slices.Clone(request.Data)
	if len(data) != 16 || binary.BigEndian.Uint32(data) != 1 || binary.BigEndian.Uint32(data[4:]) != 0 {
		return bytesOutput{}, invalid("runtime idle layout")
	}
	deadline := binary.BigEndian.Uint64(data[8:])
	if deadline == 0 || deadline > math.MaxInt64-e.config.Clock.EpochNanos {
		return bytesOutput{}, invalid("runtime idle deadline")
	}
	e.now = max(e.now, deadline)
	binary.BigEndian.PutUint64(data[8:], e.now)
	return bytesOutput{Data: data}, nil
}
