package romount

import (
	"bytes"
	"encoding/binary"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestServeReturnsCapturedFileAndUnmountedStatus(t *testing.T) {
	source := t.TempDir()
	if err := os.WriteFile(filepath.Join(source, "file"), []byte("contents"), 0o600); err != nil {
		t.Fatal(err)
	}
	broker, err := Prepare([]Mapping{{Source: source, Target: "/mounted"}}, DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = broker.Close() })
	requestRead, requestWrite := io.Pipe()
	responseRead, responseWrite := io.Pipe()
	served := make(chan error, 1)
	go func() { served <- broker.Serve(requestRead, responseWrite) }()

	for ordinal, test := range []struct {
		path   string
		status Status
		data   string
	}{{path: "/mounted/file", status: StatusOK, data: "contents"}, {path: "/other", status: StatusUnmounted}} {
		if err := WriteLookupRequest(requestWrite, uint64(ordinal), test.path); err != nil {
			t.Fatal(err)
		}
		response, err := ReadResponse(responseRead, DefaultLimits())
		if err != nil {
			t.Fatal(err)
		}
		if response.Ordinal != uint64(ordinal) || response.Status != test.status || string(response.Entry.Data) != test.data {
			t.Fatalf("response = %#v", response)
		}
	}
	if err := requestWrite.Close(); err != nil {
		t.Fatal(err)
	}
	if err := <-served; err != nil {
		t.Fatal(err)
	}
}

func TestServeRejectsOversizedAndOutOfOrderRequests(t *testing.T) {
	broker, err := Prepare(nil, DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = broker.Close() })
	for name, input := range map[string][]byte{
		"oversized": func() []byte {
			var data bytes.Buffer
			_ = WriteLookupRequest(&data, 0, "/path")
			binary.BigEndian.PutUint32(data.Bytes()[20:24], uint32(DefaultLimits().PathBytes+1))
			return data.Bytes()
		}(),
		"out of order": func() []byte {
			var data bytes.Buffer
			_ = WriteLookupRequest(&data, 1, "/path")
			return data.Bytes()
		}(),
	} {
		t.Run(name, func(t *testing.T) {
			var response bytes.Buffer
			if err := broker.Serve(bytes.NewReader(input), &response); err == nil || !strings.Contains(err.Error(), name) {
				t.Fatalf("Serve() error = %v", err)
			}
		})
	}
}
