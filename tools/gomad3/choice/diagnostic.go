package choice

import (
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"

	"go.temporal.io/server/tools/gomad3/choice/internal/wire"
)

const DiagnosticProfile = wire.DiagnosticProfile
const DiagnosticProfileEnvironment = "GOMAD3_DIAGNOSTIC_PROFILE"
const MaximumDiagnosticBytes = wire.DiagnosticMaximumBytes

var ErrDiagnosticIncomplete = errors.New("diagnostic trace incomplete")

type DiagnosticRecord wire.DiagnosticRecord

type DiagnosticTrace struct {
	Bytes    []byte
	SHA256   [sha256.Size]byte
	Records  []DiagnosticRecord
	Capacity uint64
}

type DiagnosticDivergence struct {
	Ordinal  uint64            `json:"ordinal"`
	Fields   []string          `json:"fields"`
	Expected *DiagnosticRecord `json:"expected,omitempty"`
	Actual   *DiagnosticRecord `json:"actual,omitempty"`
}

type DiagnosticSession struct {
	trace *os.File
	limit uint64
}

func BuildDiagnosticTrace(records []DiagnosticRecord, capacity uint64) (DiagnosticTrace, error) {
	header := wire.EncodeDiagnosticHeader(capacity)
	if _, err := wire.DecodeDiagnosticHeader(header[:]); err != nil {
		return DiagnosticTrace{}, err
	}
	if uint64(len(records)) > (capacity-wire.DiagnosticHeaderBytes)/wire.DiagnosticRecordBytes {
		return DiagnosticTrace{}, errors.New("diagnostic trace exceeds its capacity")
	}
	header[12] = byte(wire.DiagnosticComplete)
	binary.BigEndian.PutUint64(header[24:32], wire.DiagnosticHeaderBytes+uint64(len(records))*wire.DiagnosticRecordBytes)
	binary.BigEndian.PutUint64(header[32:40], uint64(len(records)))
	data := make([]byte, wire.DiagnosticHeaderBytes, wire.DiagnosticHeaderBytes+len(records)*wire.DiagnosticRecordBytes)
	copy(data, header[:])
	for _, record := range records {
		encoded := wire.EncodeDiagnosticRecord(wire.DiagnosticRecord(record))
		data = append(data, encoded[:]...)
	}
	return DecodeDiagnosticTrace(data)
}

func DiagnosticLimit(choiceLimit uint64) (uint64, error) {
	if err := ValidateTraceLimit(choiceLimit); err != nil {
		return 0, err
	}
	limit := uint64(wire.DiagnosticHeaderBytes) + (choiceLimit-traceHeaderBytes)/traceRecordBytes*wire.DiagnosticRecordBytes
	if limit > MaximumDiagnosticBytes {
		return 0, errors.New("diagnostic trace capacity exceeds 64 MiB")
	}
	return limit, nil
}

func NewDiagnosticSession(limit uint64) (_ *DiagnosticSession, retErr error) {
	header := wire.EncodeDiagnosticHeader(limit)
	if _, err := wire.DecodeDiagnosticHeader(header[:]); err != nil {
		return nil, err
	}
	trace, err := os.CreateTemp("", "gomad3-diagnostic-trace-")
	if err != nil {
		return nil, fmt.Errorf("create diagnostic backing: %w", err)
	}
	defer func() {
		if retErr != nil {
			retErr = errors.Join(retErr, trace.Close())
		}
	}()
	if err := os.Remove(trace.Name()); err != nil {
		return nil, fmt.Errorf("unlink diagnostic backing: %w", err)
	}
	if err := trace.Truncate(int64(limit)); err != nil {
		return nil, fmt.Errorf("size diagnostic backing: %w", err)
	}
	if _, err := trace.WriteAt(header[:], 0); err != nil {
		return nil, fmt.Errorf("initialize diagnostic backing: %w", err)
	}
	return &DiagnosticSession{trace: trace, limit: limit}, nil
}

func (session *DiagnosticSession) File() *os.File { return session.trace }

func (session *DiagnosticSession) Close() error {
	if session == nil {
		return nil
	}
	return closeSessionFile(&session.trace)
}

func (session *DiagnosticSession) Collect() (DiagnosticTrace, error) {
	headerBytes := make([]byte, wire.DiagnosticHeaderBytes)
	if _, err := session.trace.ReadAt(headerBytes, 0); err != nil {
		return DiagnosticTrace{}, fmt.Errorf("read diagnostic header: %w", err)
	}
	header, err := wire.DecodeDiagnosticHeader(headerBytes)
	if err != nil {
		return DiagnosticTrace{}, err
	}
	if header.Capacity != session.limit {
		return DiagnosticTrace{}, errors.New("diagnostic capacity changed")
	}
	if header.State != wire.DiagnosticComplete {
		return DiagnosticTrace{}, ErrDiagnosticIncomplete
	}
	data := make([]byte, header.NextOffset)
	if _, err := session.trace.ReadAt(data, 0); err != nil {
		return DiagnosticTrace{}, fmt.Errorf("read diagnostic trace: %w", err)
	}
	return DecodeDiagnosticTrace(data)
}

func ReadDiagnosticTrace(path string) (DiagnosticTrace, error) {
	file, err := os.Open(path)
	if err != nil {
		return DiagnosticTrace{}, err
	}
	data, readErr := io.ReadAll(io.LimitReader(file, MaximumDiagnosticBytes+1))
	if err := errors.Join(readErr, file.Close()); err != nil {
		return DiagnosticTrace{}, err
	}
	return DecodeDiagnosticTrace(data)
}

func DecodeDiagnosticTrace(data []byte) (DiagnosticTrace, error) {
	if len(data) < wire.DiagnosticHeaderBytes || len(data) > MaximumDiagnosticBytes {
		return DiagnosticTrace{}, errors.New("diagnostic trace size invalid")
	}
	header, err := wire.DecodeDiagnosticHeader(data[:wire.DiagnosticHeaderBytes])
	if err != nil {
		return DiagnosticTrace{}, err
	}
	if header.State != wire.DiagnosticComplete {
		return DiagnosticTrace{}, ErrDiagnosticIncomplete
	}
	if uint64(len(data)) != header.NextOffset {
		return DiagnosticTrace{}, errors.New("diagnostic trace length does not match header")
	}
	records := make([]DiagnosticRecord, header.RecordCount)
	for index := range records {
		offset := wire.DiagnosticHeaderBytes + index*wire.DiagnosticRecordBytes
		decoded, err := wire.DecodeDiagnosticRecord(data[offset : offset+wire.DiagnosticRecordBytes])
		if err != nil {
			return DiagnosticTrace{}, fmt.Errorf("diagnostic record %d: %w", index, err)
		}
		record := DiagnosticRecord(decoded)
		if record.Ordinal != uint64(index) {
			return DiagnosticTrace{}, fmt.Errorf("diagnostic record %d ordinal invalid", index)
		}
		if index > 0 && diagnosticRegressed(records[index-1], record) {
			return DiagnosticTrace{}, fmt.Errorf("diagnostic record %d counters moved backwards", index)
		}
		records[index] = record
	}
	return DiagnosticTrace{Bytes: append([]byte(nil), data...), SHA256: sha256.Sum256(data), Records: records, Capacity: header.Capacity}, nil
}

func diagnosticRegressed(previous, current DiagnosticRecord) bool {
	return current.VirtualTime < previous.VirtualTime || current.Allocations < previous.Allocations || current.GCCycle < previous.GCCycle ||
		current.RunqDraws < previous.RunqDraws || current.SchedulerDraws < previous.SchedulerDraws || current.SelectDraws < previous.SelectDraws ||
		current.RuntimeRandDraws < previous.RuntimeRandDraws || current.RuntimeCheapRandDraws < previous.RuntimeCheapRandDraws || current.TimerDraws < previous.TimerDraws || current.ClockTickDraws < previous.ClockTickDraws
}

func DiffDiagnostics(expectedBytes, actualBytes []byte) (*DiagnosticDivergence, error) {
	expected, err := DecodeDiagnosticTrace(expectedBytes)
	if err != nil {
		return nil, fmt.Errorf("expected diagnostic trace: %w", err)
	}
	actual, err := DecodeDiagnosticTrace(actualBytes)
	if err != nil {
		return nil, fmt.Errorf("actual diagnostic trace: %w", err)
	}
	for index := 0; index < max(len(expected.Records), len(actual.Records)); index++ {
		difference := &DiagnosticDivergence{Ordinal: uint64(index)}
		if index < len(expected.Records) {
			value := expected.Records[index]
			difference.Expected = &value
		}
		if index < len(actual.Records) {
			value := actual.Records[index]
			difference.Actual = &value
		}
		if difference.Expected == nil || difference.Actual == nil {
			difference.Fields = []string{"record_presence"}
			return difference, nil
		}
		left, right := *difference.Expected, *difference.Actual
		for _, field := range []struct {
			name  string
			equal bool
		}{
			{"virtual_time", left.VirtualTime == right.VirtualTime}, {"allocations", left.Allocations == right.Allocations},
			{"gc_cycle", left.GCCycle == right.GCCycle}, {"gc_phase", left.GCPhase == right.GCPhase}, {"run_queue_length", left.RunQueueLength == right.RunQueueLength},
			{"runq_draws", left.RunqDraws == right.RunqDraws}, {"scheduler_draws", left.SchedulerDraws == right.SchedulerDraws}, {"select_draws", left.SelectDraws == right.SelectDraws},
			{"runtime_rand_draws", left.RuntimeRandDraws == right.RuntimeRandDraws}, {"runtime_cheap_rand_draws", left.RuntimeCheapRandDraws == right.RuntimeCheapRandDraws},
			{"timer_draws", left.TimerDraws == right.TimerDraws}, {"clock_tick_draws", left.ClockTickDraws == right.ClockTickDraws},
		} {
			if !field.equal {
				difference.Fields = append(difference.Fields, field.name)
			}
		}
		if len(difference.Fields) != 0 {
			return difference, nil
		}
	}
	return nil, nil
}
