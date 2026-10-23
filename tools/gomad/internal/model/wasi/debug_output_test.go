package wasi

import (
	"bytes"
	"encoding/binary"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func debugEnvelope(op uint32, payload []byte) []byte {
	b := binary.LittleEndian.AppendUint32(binary.LittleEndian.AppendUint32(nil, op), uint32(len(payload)))
	for _, value := range []uint64{7, 9, 11} {
		b = binary.LittleEndian.AppendUint64(b, value)
	}
	return append(b, payload...)
}

func debugFrameFixture(id uint64, index uint32, function, file string) []byte {
	b := binary.LittleEndian.AppendUint64(nil, id)
	for _, value := range []uint32{0, index} {
		b = binary.LittleEndian.AppendUint32(b, value)
	}
	b = binary.LittleEndian.AppendUint64(b, 0x1234)
	for _, value := range []uint32{42, uint32(len(function)), uint32(len(file))} {
		b = binary.LittleEndian.AppendUint32(b, value)
	}
	return append(append(b, function...), file...)
}

func TestDebugHeapProfileOpensInPprof(t *testing.T) {
	sample := []byte{}
	for _, value := range []uint64{1, 4, 1, 4096, 1024} {
		sample = binary.LittleEndian.AppendUint64(sample, value)
	}
	sample = binary.LittleEndian.AppendUint32(sample, 0)
	sample = binary.LittleEndian.AppendUint32(sample, 1)
	sample = binary.LittleEndian.AppendUint64(sample, 0x1235)
	metadata := make([]byte, 48)
	binary.LittleEndian.PutUint64(metadata, 1)
	binary.LittleEndian.PutUint32(metadata[40:], ^uint32(0))
	raw := debugEnvelope(9, metadata)
	raw = append(raw, debugEnvelope(9, sample)...)
	raw = append(raw, debugEnvelope(11, debugFrameFixture(0, 0, "main.workload", "fixture.go"))...)
	raw = append(raw, debugEnvelope(8, debugFrameFixture(17, 0, "main.locked", "fixture.go"))...)
	output, err := DecodeDebugOutput(raw)
	if err != nil {
		t.Fatal(err)
	}
	if output.HeapSampleRate != 1 {
		t.Fatalf("sampling policy lost: %d", output.HeapSampleRate)
	}
	if len(output.Stacks) != 1 || output.Stacks[0].Goroutine != 17 || output.Stacks[0].Frames[0].Function != "main.locked" {
		t.Fatalf("stack lost: %+v", output.Stacks)
	}
	path := filepath.Join(t.TempDir(), "heap.pprof")
	if err := os.WriteFile(path, output.HeapProfile, 0600); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("go", "tool", "pprof", "-top", "-unit=bytes", "-sample_index=inuse_space", path)
	command.Env = append(os.Environ(), "GOTOOLCHAIN=local", "GOROOT=")
	data, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("pprof: %v\n%s", err, data)
	}
	if !strings.Contains(string(data), "main.workload") || !strings.Contains(string(data), "3072") {
		t.Fatalf("profile samples/symbols lost:\n%s", data)
	}
}

func TestDebugLibraryRecordsPreserveFailuresAndRawStrings(t *testing.T) {
	record := func(kind, id, status uint32, data []byte) []byte {
		b := []byte{}
		for _, value := range []uint32{kind, id, status, uint32(len(data))} {
			b = binary.LittleEndian.AppendUint32(b, value)
		}
		return debugEnvelope(4, append(b, data...))
	}
	raw := record(2, 3, 1, []byte("panic"))
	raw = append(raw, record(2, 4, 2, []byte("partial"))...)
	raw = append(raw, record(2, 5, 0, []byte("next dump"))...)
	raw = append(raw, record(3, 6, 2, nil)...)
	raw = append(raw, record(1, 7, 1, []byte{5, 2, 0, 0, 0, 0xff, 0})...)
	output, err := DecodeDebugOutput(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(output.Dumps) != 3 || output.Dumps[0].Status != "panic" || output.Dumps[1].Status != "truncated" || string(output.Dumps[2].Data) != "next dump" {
		t.Fatalf("dump isolation lost: %+v", output.Dumps)
	}
	if len(output.Checks) != 1 || output.Checks[0].Status != "panic" {
		t.Fatalf("panic became predicate truth: %+v", output.Checks)
	}
	if len(output.Probes) != 1 || !bytes.Equal(output.Probes[0].Fields[0].Bytes, []byte{0xff, 0}) {
		t.Fatalf("raw string changed: %+v", output.Probes)
	}
	if output.Probes[0].Decision != 7 || output.Probes[0].Trace != 9 || output.Probes[0].VirtualTimeNanos != 11 {
		t.Fatalf("host coordinates lost: %+v", output.Probes[0])
	}
	for _, raw := range [][]byte{{4, 0, 0}, debugEnvelope(99, nil), debugEnvelope(4, []byte{1}), record(1, 0, 1, []byte{3, 1, 0, 0, 0, 2}), record(3, 0, 0, []byte{1})} {
		if _, err := DecodeDebugOutput(raw); err == nil {
			t.Fatalf("malformed debug record admitted: %x", raw)
		}
	}
}
