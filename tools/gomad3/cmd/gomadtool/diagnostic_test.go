package main

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func toolDiagnosticBytes(draw byte) []byte {
	data := make([]byte, 160)
	copy(data, []byte{'G', 'O', 'M', 'A', 'D', 'D', 'G', 1})
	binary.BigEndian.PutUint32(data[8:12], 1)
	data[12] = 1
	binary.BigEndian.PutUint64(data[16:24], 160)
	binary.BigEndian.PutUint64(data[24:32], 160)
	binary.BigEndian.PutUint64(data[32:40], 1)
	data[143] = draw
	return data
}

func TestDiagnosticDiffStatusesAndNoPartialInvalidResult(t *testing.T) {
	directory := t.TempDir()
	left := filepath.Join(directory, "left.bin")
	right := filepath.Join(directory, "right.bin")
	if err := os.WriteFile(left, toolDiagnosticBytes(0), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name      string
		actual    []byte
		arguments []string
		status    int
		want      string
	}{
		{"equal", toolDiagnosticBytes(0), nil, 0, "diagnostic traces match"},
		{"different", toolDiagnosticBytes(1), nil, 1, "first-divergent-ordinal=0 fields=runtime_cheap_rand_draws"},
		{"json", toolDiagnosticBytes(1), []string{"--json"}, 1, `"ordinal":0`},
		{"truncated", toolDiagnosticBytes(1)[:159], nil, 2, ""},
		{"recording", func() []byte { data := toolDiagnosticBytes(1); data[12] = 0; return data }(), nil, 2, ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			if err := os.WriteFile(right, test.actual, 0o600); err != nil {
				t.Fatal(err)
			}
			var stdout, stderr bytes.Buffer
			status := run(append([]string{"diagnostic-diff"}, append(test.arguments, left, right)...), &stdout, &stderr)
			if status != test.status || !strings.Contains(stdout.String(), test.want) || status == 2 && stdout.Len() != 0 {
				t.Fatalf("status %d stdout %s stderr %s", status, stdout.String(), stderr.String())
			}
		})
	}
	if err := os.WriteFile(right, toolDiagnosticBytes(1), 0o600); err != nil {
		t.Fatal(err)
	}
	if status := run([]string{"diagnostic-diff", left, right}, diagnosticBrokenWriter{}, io.Discard); status != 3 {
		t.Fatalf("write failure status %d", status)
	}
	if status := run([]string{"diagnostic-diff", left}, io.Discard, io.Discard); status != 2 {
		t.Fatalf("argument status %d", status)
	}
}

type diagnosticBrokenWriter struct{}

func (diagnosticBrokenWriter) Write([]byte) (int, error) { return 0, errors.New("closed output") }
