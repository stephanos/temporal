package wasi

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	captured "go.temporal.io/server/tools/gomad/internal/model/capture"
	"go.temporal.io/server/tools/gomad/internal/model/choice"
)

func TestNamespaceImageLiteralAndWritableRootRefusal(t *testing.T) {
	c := testConfig()
	c.WorkingDirectory, c.Environment, c.WritableDirectories, c.Stdin = "/", []string{"PWD=/"}, nil, []byte("hi")
	c.Limits.Files, c.Limits.Descriptors, c.Limits.FilesystemBytes, c.Limits.OutputBytes, c.Clock.EpochNanos = 10, 5, 16, 32, 9
	e := newTestEnvironment(t, c)
	image, declaration, err := declareNamespaceImage(e)
	if err != nil {
		t.Fatal(err)
	}
	var wire bytes.Buffer
	if err := image.write(&wire); err != nil {
		t.Fatal(err)
	}
	want, err := hex.DecodeString("474d574e010000000a000000000000000500000000000000100000000000000020000000000000000900000000000000000000000000000000000000000000000000000000000000000000000000000001000000000000002f0200000000000000686900")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(wire.Bytes(), want) {
		t.Fatalf("wire = %x", wire.Bytes())
	}
	digest := sha256.Sum256(want)
	if *declaration != (namespaceDeclaration{1, 1, uint64(len(want)), hex.EncodeToString(digest[:])}) {
		t.Fatalf("declaration = %+v", declaration)
	}
	if !e.namespace.root.readonly {
		t.Fatal("Go implicit root must remain readonly")
	}
	if digest := namespaceConformanceDigest(e); digest != "30433144fd02a9eeca8a996fcd1ac435506af1fb83f63013e7315e2b12e722bb" {
		t.Fatalf("root state digest = %s", digest)
	}
	c.WritableDirectories = []string{"/"}
	if _, err := NewEnvironment(c); err == nil || err.Error() != "WASI invalid: writable directory" {
		t.Fatalf("writable-root preflight = %v", err)
	}
}

func TestNamespaceImageCanonicalCreationOrderAndFragmentation(t *testing.T) {
	c := testConfig()
	c.Stdin = bytes.Repeat([]byte{7}, 2*namespaceChunkBytes+3)
	e := newTestEnvironment(t, c)
	image, declaration, err := declareNamespaceImage(e)
	if err != nil {
		t.Fatal(err)
	}
	if len(image.entries) != 2 || image.entries[0].path != "/workspace" || image.entries[1].path != "/tmp" {
		t.Fatalf("inode order = %+v", image.entries)
	}
	var first, second bytes.Buffer
	if err := image.write(&first); err != nil {
		t.Fatal(err)
	}
	if err := image.write(&second); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first.Bytes(), second.Bytes()) {
		t.Fatal("noncanonical traversal")
	}
	var frames bytes.Buffer
	sink := &frameSink{input: &frames, namespaceImage: image, environment: e}
	e.namespaceOwned = true
	if err := sink.sendHelperNamespace(); err != nil {
		t.Fatal(err)
	}
	var joined []byte
	lines := bytes.Split(bytes.TrimSpace(frames.Bytes()), []byte{'\n'})
	if uint64(len(lines)) != declaration.Chunks+1 {
		t.Fatalf("frames = %d", len(lines))
	}
	for sequence, line := range lines[:len(lines)-1] {
		var frame struct {
			Type     string `json:"type"`
			Sequence uint64 `json:"sequence"`
			Data     []byte `json:"data_base64"`
		}
		if err := json.Unmarshal(line, &frame); err != nil {
			t.Fatal(err)
		}
		if frame.Type != "namespace_input" || frame.Sequence != uint64(sequence) || len(frame.Data) == 0 || len(frame.Data) > namespaceChunkBytes {
			t.Fatalf("frame = %+v", frame)
		}
		if sequence+1 < len(lines)-1 && len(frame.Data) != namespaceChunkBytes {
			t.Fatal("short nonfinal chunk")
		}
		joined = append(joined, frame.Data...)
	}
	if !bytes.Equal(joined, first.Bytes()) || string(lines[len(lines)-1]) != `{"type":"namespace_inputs_end"}` {
		t.Fatal("namespace chunk carriage changed")
	}
}

func TestNamespaceImagePreservesMetadataAndWriterRefusals(t *testing.T) {
	c := testConfig()
	c.WritableDirectories = append(c.WritableDirectories, "/"+strings.Repeat("x", 4097))
	e := newTestEnvironment(t, c)
	image, declaration, err := declareNamespaceImage(e)
	if err != nil || declaration.Bytes < 4097 {
		t.Fatalf("metadata declaration %+v: %v", declaration, err)
	}
	if err := image.write(shortNamespaceWriter{}); !errors.Is(err, io.ErrShortWrite) {
		t.Fatalf("short write = %v", err)
	}
	marker := errors.New("sink failed")
	if err := image.write(failedNamespaceWriter{marker}); !errors.Is(err, marker) {
		t.Fatalf("writer failure = %v", err)
	}
	e.helperOwned = true
	if !helperOwnsNamespace(e) {
		t.Fatal("pure local namespace not eligible")
	}
	// fn-176: storage runs keep the namespace and volume in the helper.
	e.config.Storage = &StorageModel{}
	if !helperOwnsNamespace(e) {
		t.Fatal("storage namespace not helper-owned")
	}
	e.config.Storage = nil
	e.helperOwned, e.splitOwned = false, true
	if !helperOwnsNamespace(e) {
		t.Fatal("split-owned namespace not helper-owned")
	}
	e.splitOwned = false
	if helperOwnsNamespace(e) {
		t.Fatal("remote model escaped Go ownership")
	}
}

func TestNamespaceImageCapturedOwnershipAndPreflightPriority(t *testing.T) {
	source := t.TempDir()
	if err := os.WriteFile(filepath.Join(source, "input"), []byte("frozen"), 0600); err != nil {
		t.Fatal(err)
	}
	inputs, err := captured.CaptureReadOnlyMountInputs([]captured.Mapping{{Source: source, Target: "/fixtures/input-root"}}, captured.DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	c := testConfig()
	c.CapturedInputs = inputs
	c.WritableDirectories = append(c.WritableDirectories, "/fixtures")
	e := newTestEnvironment(t, c)
	image, _, err := declareNamespaceImage(e)
	if err != nil {
		t.Fatal(err)
	}
	if len(image.captured) != 1 || image.captured[0] != "/fixtures/input-root" {
		t.Fatalf("capture ownership = %v", image.captured)
	}
	for _, payload := range inputs.Payloads {
		for index := range payload {
			payload[index] = 0
		}
	}
	var data []byte
	for _, entry := range image.entries {
		if entry.path == "/fixtures/input-root/input" {
			data = entry.node.data
			if !entry.node.readonly {
				t.Fatal("capture became mutable")
			}
		}
	}
	if string(data) != "frozen" {
		t.Fatalf("image reopened or aliased caller payload: %q", data)
	}
	for _, directory := range []string{"/fixtures/input-root", "/fixtures/input-root/child"} {
		c.WritableDirectories = []string{directory}
		// Restore the cloned admission input; the earlier caller mutation must
		// not affect the already constructed image.
		c.CapturedInputs, err = captured.CaptureReadOnlyMountInputs([]captured.Mapping{{Source: source, Target: "/fixtures/input-root"}}, captured.DefaultLimits())
		if err != nil {
			t.Fatal(err)
		}
		if _, err := NewEnvironment(c); err == nil || err.Error() != "WASI invalid: writable mount overlaps captured input" {
			t.Fatalf("capture overlap %s = %v", directory, err)
		}
	}
	c = testConfig()
	c.Limits.Files = 1
	if _, err := NewEnvironment(c); err == nil || err.Error() != "WASI capacity: initial-namespace" {
		t.Fatalf("file preflight = %v", err)
	}
	c = testConfig()
	c.Limits.FilesystemBytes = 1
	c.Stdin = []byte("too big")
	if _, err := NewEnvironment(c); err == nil || err.Error() != "WASI capacity: stdin-bytes" {
		t.Fatalf("stdin preflight = %v", err)
	}
}

func TestNamespaceImagePreInstantiationInvalidReceipt(t *testing.T) {
	e := newTestEnvironment(t, testConfig())
	e.helperOwned, e.namespaceOwned = true, true
	digest := sha256.Sum256(nil)
	e.runtime = &runtimeSession{control: RuntimeControl{Choice: &choice.Request{Mode: choice.ModeRecord}}}
	sink := &frameSink{environment: e, started: &Started{}, terminal: &resultFrame{Termination: "invalid", LocalModel: &helperModelResult{Now: e.now, TraceSHA256: hex.EncodeToString(digest[:]), TranscriptSHA256: hex.EncodeToString(digest[:])}}}
	result := Result{Termination: "invalid"}
	if err := sink.collectHelper(&result); err != nil {
		t.Fatalf("pre-instantiation invalid receipt = %v", err)
	}
	if result.Calls != 0 || len(result.Stdout) != 0 || len(result.Transcript) != 0 || result.TranscriptSHA256 != digest {
		t.Fatal("image refusal published modeled activity")
	}
}

func TestNamespaceImageCarriageFollowsAuthorizationAndReplay(t *testing.T) {
	e := newTestEnvironment(t, testConfig())
	e.helperOwned, e.namespaceOwned = true, true
	e.runtime = &runtimeSession{control: RuntimeControl{Choice: &choice.Request{Mode: choice.ModeReplay, Limit: 8192}}}
	image, _, err := declareNamespaceImage(e)
	if err != nil {
		t.Fatal(err)
	}
	request := Request{Engine: PinnedEngine(), ModuleSHA256: strings.Repeat("0", 64), InitialMemoryPages: 1, Imports: []Import{}}
	started := Started{Schema: "gomad.wasm-host/v1", Type: "started", Engine: request.Engine, ModuleSHA256: request.ModuleSHA256, InitialMemoryPages: 1, Imports: []Import{}, Exports: []Export{{"memory", "memory"}, {"_start", "func"}}}
	var wire bytes.Buffer
	sink := &frameSink{request: request, environment: e, namespaceImage: image, input: &wire}
	bad := started
	bad.ModuleSHA256 = strings.Repeat("1", 64)
	data, err := json.Marshal(bad)
	if err != nil {
		t.Fatal(err)
	}
	if err := sink.frame(data); err == nil || wire.Len() != 0 {
		t.Fatal("image published before authorization identity")
	}
	data, err = json.Marshal(started)
	if err != nil {
		t.Fatal(err)
	}
	if err := sink.frame(data); err != nil {
		t.Fatal(err)
	}
	lines := bytes.Split(bytes.TrimSpace(wire.Bytes()), []byte{'\n'})
	for index, want := range []string{"authorize", "replay_trace", "replay_trace_end", "namespace_input", "namespace_inputs_end"} {
		if index >= len(lines) {
			t.Fatal("truncated authorization carriage")
		}
		var frame struct {
			Type string `json:"type"`
		}
		if err := json.Unmarshal(lines[index], &frame); err != nil {
			t.Fatal(err)
		}
		if frame.Type != want {
			t.Fatalf("frame%d=%s want%s", index, frame.Type, want)
		}
	}
	if len(lines) != 5 {
		t.Fatalf("extra carriage frames=%d", len(lines))
	}
}

type shortNamespaceWriter struct{}

func (shortNamespaceWriter) Write(data []byte) (int, error) { return len(data) - 1, nil }

type failedNamespaceWriter struct{ err error }

func (w failedNamespaceWriter) Write([]byte) (int, error) { return 0, w.err }
