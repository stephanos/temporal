package wasi

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestDeterminismRefuseExecuteRequest(t *testing.T) {
	t.Parallel()
	for _, field := range []string{`"future_field":true`, `"type":"future_execute"`} {
		t.Run(field, func(t *testing.T) {
			t.Parallel()
			request := `{"schema":"gomad.wasm-host/v1","module_path":"/unused","module_sha256":"` + strings.Repeat("0", 64) + `","memory_bytes":65536,"fuel":1,"output_bytes":1,`
			if !strings.HasPrefix(field, `"type"`) {
				request += `"type":"execute",`
			}
			command := exec.CommandContext(t.Context(), helperPath(t))
			command.Stdin = strings.NewReader(request + field + "}\n")
			output, err := command.Output()
			if err != nil {
				t.Fatal(err)
			}
			var terminal resultFrame
			if err := json.Unmarshal(output, &terminal); err != nil {
				t.Fatal(err)
			}
			if terminal.Type != "result" || terminal.Termination != "unsupported" || !strings.Contains(terminal.Message, "future") {
				t.Fatalf("execute refusal did not precede started: %s", output)
			}
		})
	}
}

func TestDeterminismRefuseUnknownHelperFrames(t *testing.T) {
	t.Parallel()
	for _, split := range []bool{false, true} {
		for _, data := range []string{
			`{"type":"future_reply"}`,
			`{"type":"future_z","future_a":true}`,
			`{"type":"call","id":1,"op":"fd_write","input":{"fd":1,"data_base64":"eA=="},"future":true}`,
			`{"type":"call","id":1,"op":"fd_write","input":{"fd":1,"data_base64":"eA==","future":true}}`,
		} {
			t.Run(data+map[bool]string{false: "/remote", true: "/split"}[split], func(t *testing.T) {
				t.Parallel()
				environment := newTestEnvironment(t, testConfig())
				environment.splitOwned = split
				environment.namespaceOwned = split
				if split && strings.Contains(data, `"type":"call"`) {
					data = strings.TrimSuffix(data, "}") + `,"local_state":{"now":1,"advances":0}}`
				}
				sink := frameSink{environment: environment, started: &Started{}, input: &bytes.Buffer{}}
				err := sink.frame([]byte(data))
				var boundary *BoundaryError
				if !errors.As(err, &boundary) || boundary.Kind != "unsupported" || !strings.Contains(boundary.Dimension, "future") {
					t.Fatalf("unknown helper frame not classified unsupported: %v", err)
				}
				if len(environment.stdout) != 0 || len(environment.Transcript()) != 0 || sink.terminal != nil {
					t.Fatal("refused callback published output, transcript, or a valid terminal")
				}
			})
		}
	}
}

func TestDeterminismRefuseUnknownFieldOrder(t *testing.T) {
	t.Parallel()
	for range 100 {
		err := refuseUnknownFields(map[string]json.RawMessage{"z": []byte("0"), "a": []byte("0")}, nil)
		if err == nil || err.Error() != `WASI unsupported: unknown field "a"` {
			t.Fatalf("unknown fields changed refusal with traversal order: %v", err)
		}
	}
	if err := refuseUnknownFields(map[string]json.RawMessage{"": []byte("0")}, nil); err == nil || err.Error() != `WASI unsupported: unknown field ""` {
		t.Fatalf("empty unknown field accepted: %v", err)
	}
}

func TestDeterminismRefuseUnknownHelperTerminal(t *testing.T) {
	t.Parallel()
	request := terminalExitSyntheticRequest(t)
	if err := os.WriteFile(request.HelperPath, []byte("#!/bin/sh\nread request\nprintf '%s\\n' '{\"type\":\"future_reply\"}'\n"), 0700); err != nil {
		t.Fatal(err)
	}
	request.HelperSHA256 = fileHash(t, request.HelperPath)
	result, err := Run(t.Context(), request)
	if err != nil {
		t.Fatal(err)
	}
	if result.Termination != "unsupported" || !strings.Contains(result.Message, "future_reply") || result.Started != nil || !result.Reaped {
		t.Fatalf("unknown reply not classified as terminal refusal: %+v", result)
	}
}

func TestDeterminismRefuseOffPinStarted(t *testing.T) {
	t.Parallel()
	engine := PinnedEngine()
	engine.Configuration += "-future-setting"
	started := Started{Schema: "gomad.wasm-host/v1", Type: "started", Engine: engine, Imports: []Import{}, Exports: []Export{}}
	data, err := json.Marshal(started)
	if err != nil {
		t.Fatal(err)
	}
	var sent bytes.Buffer
	sink := frameSink{request: Request{Engine: engine}, input: &sent}
	err = sink.frame(data)
	var boundary *BoundaryError
	if !errors.As(err, &boundary) || boundary.Kind != "unsupported" || !strings.Contains(boundary.Dimension, engine.Configuration) {
		t.Fatalf("off-pin setting not refused: %v", err)
	}
	if sent.Len() != 0 || sink.started != nil {
		t.Fatal("off-pin helper authorized")
	}
	if err := validateClusterStarted(started, Request{Engine: engine}); !errors.As(err, &boundary) || boundary.Kind != "unsupported" {
		t.Fatalf("off-pin cluster helper not refused: %v", err)
	}
}

func TestDeterminismRefuseNestedHelperFields(t *testing.T) {
	t.Parallel()
	for _, data := range []string{
		`{"type":"call","id":1,"op":"fd_write","input":{"fd":1,"data_base64":"eA=="},"local_state":{"now":1,"advances":0,"future":true}}`,
		`{"type":"started","schema":"gomad.wasm-host/v1","engine":{"name":"wasmtime","future":true},"module_sha256":"","imports":[],"exports":[],"initial_memory_pages":1}`,
	} {
		t.Run(data, func(t *testing.T) {
			t.Parallel()
			environment := newTestEnvironment(t, testConfig())
			environment.splitOwned = true
			sink := frameSink{environment: environment}
			if strings.Contains(data, `"type":"call"`) {
				sink.started = &Started{}
			}
			err := sink.frame([]byte(data))
			var boundary *BoundaryError
			if !errors.As(err, &boundary) || boundary.Kind != "unsupported" || !strings.Contains(boundary.Dimension, "future") {
				t.Fatalf("unknown nested field not refused: %v", err)
			}
		})
	}
}
