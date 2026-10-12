package main

import (
	"bytes"
	"errors"
	"io"
	"os"
	"testing"
)

type compatibilitySourceWriter struct {
	output    io.Writer
	attempted bytes.Buffer
	calls     int
	err       error
}

func (writer *compatibilitySourceWriter) Write(data []byte) (int, error) {
	writer.calls++
	if _, err := writer.attempted.Write(data); err != nil {
		return 0, err
	}
	n, err := writer.output.Write(data)
	writer.err = err
	return n, err
}

func TestCompatibilityPackSourceUsagePreservesBytesAndWriteFailure(t *testing.T) {
	const usage = "usage: gomadtool compatibility-pack discover|review|refresh|generate|check|qualify [flags]\n"
	for _, test := range []struct {
		name      string
		arguments []string
		closed    bool
		status    int
	}{
		{name: "no command", status: 2},
		{name: "unknown command", arguments: []string{"source-unknown"}, status: 2},
		{name: "no command closed writer", closed: true, status: 1},
		{name: "unknown command closed writer", arguments: []string{"source-unknown"}, closed: true, status: 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			writer := compatibilitySourceWriter{output: &stderr}
			if test.closed {
				file, err := os.CreateTemp(t.TempDir(), "closed-output")
				if err != nil {
					t.Fatal(err)
				}
				if err := file.Close(); err != nil {
					t.Fatal(err)
				}
				writer.output = file
			}
			status := runCompatibilityPack(test.arguments, &stdout, &writer)
			if status != test.status || writer.calls != 1 || writer.attempted.String() != usage || stdout.Len() != 0 {
				t.Errorf("status=%d want=%d, calls=%d, attempted=%q, stdout=%q", status, test.status, writer.calls, writer.attempted.String(), stdout.String())
			}
			if test.closed {
				if !errors.Is(writer.err, os.ErrClosed) || stderr.Len() != 0 {
					t.Fatalf("closed writer error=%v, diagnostics=%q", writer.err, stderr.String())
				}
			} else if writer.err != nil || stderr.String() != usage {
				t.Fatalf("ordinary writer error=%v, diagnostics=%q", writer.err, stderr.String())
			}
		})
	}
}
