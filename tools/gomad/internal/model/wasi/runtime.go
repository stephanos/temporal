package wasi

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"slices"

	"go.temporal.io/server/tools/gomad/internal/model/choice"
)

const CooperativeProfile = "gomad.wasi-cooperative/v1"

type RuntimeControl struct {
	Seed        uint64
	Choice      *choice.Request
	Diagnostics bool
	Inspection  json.RawMessage `json:"-"`
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
	switch request.Mode {
	case choice.ModeRecord:
		if len(request.Trace) != 0 || request.TraceSHA256 != ([32]byte{}) || request.TapeSHA256 != ([32]byte{}) || request.Decisions != 0 || request.PrefixControl != nil {
			return nil, invalid("record mode has replay input")
		}
	case choice.ModeReplay, choice.ModePrefix:
		request.Trace = slices.Clone(request.Trace)
		if control := request.PrefixControl; control != nil {
			cloned := *control
			if override := control.Override; override != nil {
				copyOverride := *override
				cloned.Override = &copyOverride
			}
			request.PrefixControl = &cloned
		}
	default:
		return nil, invalid("cooperative choice mode")
	}
	if control.Diagnostics {
		if _, err := choice.DiagnosticLimit(request.Limit); err != nil {
			return nil, err
		}
	}
	cloned := *control
	cloned.Inspection = slices.Clone(control.Inspection)
	if len(cloned.Inspection) != 0 && request.Mode != choice.ModeReplay {
		return nil, invalid("inspection requires exact replay")
	}
	cloned.Choice = &request
	return &runtimeSession{control: cloned}, nil
}

func (session *runtimeSession) prepareOracleTape() error {
	request := session.control.Choice
	if session.tape != nil || request.Mode == choice.ModeRecord {
		return nil
	}
	count, err := choice.TraceRecordCount(request.Trace)
	if err != nil {
		return err
	}
	trace, err := choice.DecodeStoredTrace(choice.Profile, request.Trace, choice.TerminalMetadata{State: choice.TerminalComplete, Limit: request.Limit, Records: count, SHA256: request.TraceSHA256})
	if err != nil {
		return err
	}
	plan, err := choice.ProjectReplayPlan(trace, request.ExecutionIdentity)
	if err != nil {
		return err
	}
	if control := request.PrefixControl; control != nil {
		if request.Mode != choice.ModePrefix {
			return invalid("exact replay has prefix controls")
		}
		if override := control.Override; override != nil {
			if control.Cutoff == 0 || override.Ordinal != control.Cutoff-1 {
				return invalid("prefix override is not final")
			}
			if override.Reconstruct {
				plan, err = choice.BuildForcedRankPrefix(plan, override.Ordinal, override.Rank)
			} else {
				plan, err = choice.BuildRankPrefix(plan, override.Ordinal, override.Rank)
			}
		} else {
			plan, err = plan.Prefix(control.Cutoff)
		}
		if err != nil {
			return err
		}
	}
	if plan.SHA256 != request.TapeSHA256 || uint64(len(plan.Decisions)) != request.Decisions {
		return invalid("oracle replay plan differs from expectations")
	}
	session.tape = &plan
	return nil
}

func (session *runtimeSession) handle(op string, input json.RawMessage) (bytesOutput, error) {
	if err := session.prepareOracleTape(); err != nil {
		return bytesOutput{}, err
	}
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
		// Header v2 (choice trace v4) appends one 16-byte creation site per
		// runnable alternative, mirroring the helper's runtime decision import.
		if len(data) < 32 || binary.BigEndian.Uint32(data) != 1 && binary.BigEndian.Uint32(data) != 2 {
			return bytesOutput{}, invalid("runtime decision header")
		}
		rawKind := binary.BigEndian.Uint32(data[4:])
		kind, site := choice.Kind(rawKind), binary.BigEndian.Uint64(data[8:])
		flags, count := binary.BigEndian.Uint32(data[16:]), binary.BigEndian.Uint32(data[20:])
		selected, value := binary.BigEndian.Uint32(data[24:]), binary.BigEndian.Uint32(data[28:])
		creation := binary.BigEndian.Uint32(data) == 2 && kind == choice.KindRunnable
		diagnosticBytes := 0
		if session.control.Diagnostics {
			diagnosticBytes = 96
		}
		stride := 32
		if creation {
			stride = 48
		}
		if rawKind != uint32(kind) || (kind != choice.KindRunnable && kind != choice.KindSelectPoll) || flags&^uint32(choice.FlagSiteMissing) != 0 || count < 2 || count > 256 || selected >= count || len(data) != 32+int(count)*stride+diagnosticBytes {
			return bytesOutput{}, invalid("runtime decision alternatives or layout")
		}
		alternatives := make([][32]byte, count)
		for i := range alternatives {
			copy(alternatives[i][:], data[32+32*i:64+32*i])
		}
		diagnostic := data[32+int(count)*stride:]
		observed, err := choice.CanonicalDecision(uint64(len(session.records)), kind, site, flags != 0, alternatives, alternatives[selected], value)
		if err != nil {
			return bytesOutput{}, fmt.Errorf("%w: %w", invalid("runtime decision identities"), err)
		}
		var creations []choice.Record
		if creation {
			creations = make([]choice.Record, count)
			for i, identity := range alternatives {
				entry := data[32+32*int(count)+16*i : 32+32*int(count)+16*(i+1)]
				siteOffset, missing := binary.BigEndian.Uint64(entry), entry[8]
				if missing&^uint8(choice.FlagSiteMissing) != 0 || !allZero(entry[9:]) || missing != 0 && siteOffset != 0 {
					return bytesOutput{}, invalid("runtime creation site")
				}
				creations[i] = choice.Record{Kind: choice.KindGoroutineCreation, Flags: choice.Flags(missing), Alternatives: count, SiteOffset: siteOffset, SelectedIdentity: identity}
			}
			slices.SortFunc(creations, func(a, b choice.Record) int { return compareIdentity(a.SelectedIdentity, b.SelectedIdentity) })
			if 64+uint64(len(session.records)+int(count)+1)*96 > session.control.Choice.Limit {
				return bytesOutput{}, capacity("runtime-choice-bytes")
			}
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
		for i, record := range creations {
			record.Ordinal, record.Selected = uint64(len(session.records)), uint32(i)
			if err := session.append(record, session.diagnosticAt(diagnostic)); err != nil {
				return bytesOutput{}, err
			}
		}
		observed.Ordinal = uint64(len(session.records))
		if err := session.append(observed.Record(), session.diagnosticAt(diagnostic)); err != nil {
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
		// The public decoder validates the observation as a one-record trace with
		// its ordinal and origin cleared; both are checked here against the
		// preceding records, so validation stays linear in the trace length.
		ordinal, origin := binary.BigEndian.Uint64(data[:8]), binary.BigEndian.Uint64(data[32:40])
		if ordinal != uint64(len(session.records)) || origin > ordinal {
			return bytesOutput{}, invalid("runtime select observation ordinal")
		}
		payload := slices.Clone(data[:96])
		clear(payload[:8])
		clear(payload[32:40])
		trace, err := choice.DecodeStoredTrace(choice.Profile, payload, choice.TerminalMetadata{State: choice.TerminalComplete, Limit: session.control.Choice.Limit, Records: 1, SHA256: sha256.Sum256(payload)})
		if err != nil || trace.Records[0].Kind != choice.KindSelectResult {
			return bytesOutput{}, invalid("runtime select observation")
		}
		record := trace.Records[0]
		record.Ordinal, record.Origin = ordinal, origin
		if err := session.append(record, data[96:]); err != nil {
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

// diagnosticAt stamps the guest's diagnostic record with the ordinal of the
// trace record it is appended for; one guest decision can append several.
func (session *runtimeSession) diagnosticAt(diagnostic []byte) []byte {
	stamped := slices.Clone(diagnostic)
	if session.control.Diagnostics && len(stamped) == 96 {
		binary.BigEndian.PutUint64(stamped, uint64(len(session.records)))
	}
	return stamped
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

func (session *runtimeSession) interrupted() ([]byte, []byte, error) {
	if !session.configured || session.finished {
		return nil, nil, invalid("runtime interruption lifetime")
	}
	if session.tape != nil && session.cursor != uint64(len(session.tape.Decisions)) {
		expected := session.tape.Decisions[session.cursor]
		return nil, nil, session.diverge(choice.DivergenceTapeUnconsumed, &expected, nil)
	}
	validated, err := choice.BuildTrace(session.records, choice.TerminalComplete)
	if err != nil {
		return nil, nil, err
	}
	return validated.Bytes, slices.Clone(session.diagnostics), nil
}

func (e *Environment) runtimeIdleInput(input json.RawMessage) ([]byte, uint64, error) {
	if !e.runtime.configured || e.runtime.finished {
		return nil, 0, invalid("runtime idle outside active lifetime")
	}
	var request bytesOutput
	if err := decodeInput(input, &request, "data_base64"); err != nil {
		return nil, 0, err
	}
	data := slices.Clone(request.Data)
	if len(data) != 16 || binary.BigEndian.Uint32(data) != 1 || binary.BigEndian.Uint32(data[4:]) != 0 {
		return nil, 0, invalid("runtime idle layout")
	}
	deadline := binary.BigEndian.Uint64(data[8:])
	return data, deadline, nil
}

func (e *Environment) runtimeIdle(input json.RawMessage) (bytesOutput, error) {
	data, deadline, err := e.runtimeIdleInput(input)
	if err != nil {
		return bytesOutput{}, err
	}
	if e.brokerTime != nil {
		now := *e.brokerTime
		e.brokerTime = nil
		if now < e.now {
			return bytesOutput{}, invalid("broker clock moved backwards")
		}
		if now > e.now {
			e.advances++
		}
		e.now = now
		if err = e.networkV2Advance(now); err != nil {
			return bytesOutput{}, err
		}
		if e.networkV2 != nil {
			binary.BigEndian.PutUint32(data[4:], 1)
		}
		binary.BigEndian.PutUint64(data[8:], now)
		return bytesOutput{Data: data}, nil
	}
	if e.networkV2 != nil {
		return e.networkV2Idle(data, deadline)
	}
	if deadline == 0 || deadline > math.MaxInt64-e.config.Clock.EpochNanos {
		return bytesOutput{}, invalid("runtime idle deadline")
	}
	if deadline > e.now {
		e.advances++
	}
	e.now = max(e.now, deadline)
	binary.BigEndian.PutUint64(data[8:], e.now)
	return bytesOutput{Data: data}, nil
}
