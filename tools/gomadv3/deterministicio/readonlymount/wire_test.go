package readonlymount

import (
	"bytes"
	"encoding/binary"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	iowire "go.temporal.io/server/tools/gomadv3/deterministicio/internal/wire"
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

func TestReadResponseRejectsInvalidProtocolFields(t *testing.T) {
	valid := func() []byte {
		var data bytes.Buffer
		if err := writeResponse(&data, Response{
			Ordinal: 4,
			Status:  StatusOK,
			Entry:   Entry{Kind: KindFile, Mode: 0o640, Data: []byte("contents")},
		}, DefaultLimits()); err != nil {
			t.Fatal(err)
		}
		return data.Bytes()
	}
	for name, mutate := range map[string]func([]byte){
		"reserved response byte": func(data []byte) { data[21] = 1 },
		"unknown status":         func(data []byte) { binary.BigEndian.PutUint16(data[10:12], 99) },
		"unknown entry kind":     func(data []byte) { data[20] = 99 },
		"non-OK response payload": func(data []byte) {
			binary.BigEndian.PutUint16(data[10:12], uint16(StatusNotExist))
		},
		"directory with contents": func(data []byte) { data[20] = byte(KindDirectory) },
	} {
		t.Run(name, func(t *testing.T) {
			data := valid()
			mutate(data)
			if _, err := ReadResponse(bytes.NewReader(data), DefaultLimits()); err == nil {
				t.Fatal("ReadResponse() accepted invalid protocol fields")
			}
		})
	}
}

func TestReadResponseRejectsInvalidChildFields(t *testing.T) {
	var encoded bytes.Buffer
	if err := writeResponse(&encoded, Response{
		Ordinal: 1,
		Status:  StatusOK,
		Entry: Entry{Kind: KindDirectory, Mode: 0o755, Children: []Child{
			{Name: "child", Kind: KindFile, Mode: 0o600},
		}},
	}, DefaultLimits()); err != nil {
		t.Fatal(err)
	}
	for name, offset := range map[string]int{
		"reserved child byte": iowire.MountResponseHeaderBytes + 3,
		"unknown child kind":  iowire.MountResponseHeaderBytes + 2,
		"file with children":  20,
	} {
		t.Run(name, func(t *testing.T) {
			data := append([]byte(nil), encoded.Bytes()...)
			if name == "file with children" {
				data[offset] = byte(KindFile)
			} else {
				data[offset] = 99
			}
			if _, err := ReadResponse(bytes.NewReader(data), DefaultLimits()); err == nil {
				t.Fatal("ReadResponse() accepted invalid child fields")
			}
		})
	}
}
