package choice

import (
	"encoding/binary"
	"reflect"
	"testing"

	"go.temporal.io/server/tools/gomad3/choice/internal/wire"
)

func diagnosticBytes(records ...wire.DiagnosticRecord) []byte {
	header := wire.EncodeDiagnosticHeader(4096)
	header[12] = byte(wire.DiagnosticComplete)
	binary.BigEndian.PutUint64(header[24:32], uint64(64+96*len(records)))
	binary.BigEndian.PutUint64(header[32:40], uint64(len(records)))
	contents := append([]byte(nil), header[:]...)
	for _, record := range records {
		encoded := wire.EncodeDiagnosticRecord(record)
		contents = append(contents, encoded[:]...)
	}
	return contents
}

func TestDiagnosticDiffFindsFirstOrdinalAndFields(t *testing.T) {
	plain := []wire.DiagnosticRecord{{Ordinal: 0, Allocations: 2}, {Ordinal: 1, Allocations: 3, RuntimeCheapRandDraws: 2}}
	changed := append([]wire.DiagnosticRecord(nil), plain...)
	changed[1].RuntimeCheapRandDraws++
	difference, err := DiffDiagnostics(diagnosticBytes(plain...), diagnosticBytes(changed...))
	if err != nil {
		t.Fatal(err)
	}
	if difference == nil || difference.Ordinal != 1 || !reflect.DeepEqual(difference.Fields, []string{"runtime_cheap_rand_draws"}) {
		t.Fatalf("difference = %+v", difference)
	}
	changed[0].Allocations++
	difference, err = DiffDiagnostics(diagnosticBytes(plain...), diagnosticBytes(changed...))
	if err != nil || difference == nil || difference.Ordinal != 0 {
		t.Fatalf("ordinal zero difference = %+v, %v", difference, err)
	}
	difference, err = DiffDiagnostics(diagnosticBytes(plain...), diagnosticBytes(plain[:1]...))
	if err != nil || difference == nil || difference.Ordinal != 1 || !reflect.DeepEqual(difference.Fields, []string{"record_presence"}) {
		t.Fatalf("length difference = %+v, %v", difference, err)
	}
	difference, err = DiffDiagnostics(diagnosticBytes(plain...), diagnosticBytes(plain...))
	if err != nil || difference != nil {
		t.Fatalf("equal traces = %+v, %v", difference, err)
	}
}

func TestDiagnosticDiffValidatesEntireInputsBeforeReporting(t *testing.T) {
	plain := diagnosticBytes(wire.DiagnosticRecord{Ordinal: 0}, wire.DiagnosticRecord{Ordinal: 1})
	for name, mutate := range map[string]func([]byte) []byte{
		"truncated header":   func(data []byte) []byte { return data[:63] },
		"truncated record":   func(data []byte) []byte { return data[:len(data)-1] },
		"extra bytes":        func(data []byte) []byte { return append(data, 0) },
		"recording":          func(data []byte) []byte { data[12] = 0; return data },
		"overflow":           func(data []byte) []byte { data[12] = 2; return data },
		"ordinal":            func(data []byte) []byte { data[64+96+7] = 3; return data },
		"reserved tail":      func(data []byte) []byte { data[64+96+29] = 1; return data },
		"counter regression": func(data []byte) []byte { data[64+47] = 2; return data },
	} {
		t.Run(name, func(t *testing.T) {
			broken := mutate(append([]byte(nil), plain...))
			different := diagnosticBytes(wire.DiagnosticRecord{Ordinal: 0, Allocations: 1}, wire.DiagnosticRecord{Ordinal: 1, Allocations: 1})
			if difference, err := DiffDiagnostics(different, broken); err == nil || difference != nil {
				t.Fatalf("invalid input returned %+v, %v", difference, err)
			}
		})
	}
}

func TestDiagnosticSessionCollectsOnlyCompleteBoundedTrace(t *testing.T) {
	session, err := NewDiagnosticSession(4096)
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	if _, err := session.Collect(); err == nil {
		t.Fatal("recording trace accepted")
	}
	data := diagnosticBytes(wire.DiagnosticRecord{Ordinal: 0, RuntimeRandDraws: 1})
	if _, err := session.File().WriteAt(data, 0); err != nil {
		t.Fatal(err)
	}
	trace, err := session.Collect()
	if err != nil {
		t.Fatal(err)
	}
	if len(trace.Bytes) != 160 || len(trace.Records) != 1 || trace.Records[0].RuntimeRandDraws != 1 {
		t.Fatalf("trace %+v", trace)
	}
	if limit, err := DiagnosticLimit(MaximumTraceBytes); err != nil || limit > MaximumDiagnosticBytes || limit < 160 {
		t.Fatalf("maximum bound %d: %v", limit, err)
	}
	if _, err := NewDiagnosticSession(MaximumDiagnosticBytes + 1); err == nil {
		t.Fatal("oversized diagnostic capacity accepted")
	}
}
