package wasi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

const namespacePublicCorpusSchema = "gomad.namespace-public-corpus/v1"

func namespacePublicFixture(t *testing.T, path string, generate func() any, fixture any) {
	t.Helper()
	if *updateNamespaceConformance {
		data, err := json.MarshalIndent(generate(), "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, append(data, '\n'), 0644); err != nil {
			t.Fatal(err)
		}
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal("public namespace corpus missing; regenerate with -update-conformance: ", err)
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(fixture); err != nil {
		t.Fatal(err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		t.Fatal("public namespace corpus trailing JSON: ", err)
	}
}

type namespaceAdapterStep struct {
	Error  string `json:"error"`
	Mirror []byte `json:"mirror_base64"`
	Stdout []byte `json:"stdout_base64"`
	Stderr []byte `json:"stderr_base64"`
}

type namespaceAdapterResult struct {
	Steps              []namespaceAdapterStep `json:"steps"`
	PartialTranscript  []byte                 `json:"partial_transcript_base64"`
	PartialChoice      []byte                 `json:"partial_choice_base64"`
	PartialDiagnostic  []byte                 `json:"partial_diagnostic_base64"`
	Calls              uint64                 `json:"calls"`
	RuntimeInterrupted bool                   `json:"runtime_interrupted"`
	Termination        string                 `json:"termination"`
	Served             map[string]uint64      `json:"filesystem_served"`
}

type namespaceAdapterCorpus struct {
	Schema      string                          `json:"schema"`
	Name        string                          `json:"name"`
	Kind        string                          `json:"kind"`
	OutputLimit uint64                          `json:"output_limit"`
	Frames      []json.RawMessage               `json:"frames"`
	Transcript  []byte                          `json:"transcript_base64"`
	Choice      []byte                          `json:"choice_base64"`
	Diagnostic  []byte                          `json:"diagnostic_base64"`
	Operations  []namespaceConformanceOperation `json:"operations"`
	Expected    namespaceAdapterResult          `json:"expected"`
}

func runNamespaceAdapterCorpus(fixture namespaceAdapterCorpus) (namespaceAdapterResult, error) {
	var actual namespaceAdapterResult
	if fixture.Schema != namespacePublicCorpusSchema || fixture.Name == "" {
		return actual, fmt.Errorf("invalid namespace adapter corpus schema/name")
	}
	switch fixture.Kind {
	case "output":
		if fixture.OutputLimit == 0 || len(fixture.Frames) == 0 || len(fixture.Operations) != 0 || fixture.Transcript != nil || fixture.Choice != nil || fixture.Diagnostic != nil {
			return actual, fmt.Errorf("invalid output corpus fields")
		}
		environment, err := NewEnvironment(testConfig())
		if err != nil {
			return actual, err
		}
		environment.helperOwned, environment.namespaceOwned = true, true
		var mirror bytes.Buffer
		environment.mirror = &mirror
		sink := frameSink{environment: environment, started: &Started{}, request: Request{Config: testConfig()}}
		sink.request.Config.Limits.OutputBytes = fixture.OutputLimit
		for _, raw := range fixture.Frames {
			frame, err := newValidatedFrame(raw)
			if err != nil {
				return actual, err
			}
			step := namespaceAdapterStep{}
			if err := sink.receiveHelperEvidence(frame); err != nil {
				step.Error = err.Error()
			}
			step.Mirror = bytes.Clone(mirror.Bytes())
			step.Stdout, step.Stderr = sink.guestOutput()
			actual.Steps = append(actual.Steps, step)
		}
	case "partial":
		if fixture.OutputLimit != 0 || len(fixture.Frames) != 0 || len(fixture.Operations) != 0 || fixture.Transcript == nil || fixture.Choice == nil || fixture.Diagnostic == nil {
			return actual, fmt.Errorf("invalid partial corpus fields")
		}
		sink := frameSink{helperEvidence: helperEvidence{transcript: fixture.Transcript, choice: fixture.Choice, diagnostic: fixture.Diagnostic}}
		result := Result{Termination: "infrastructure"}
		sink.retainPartialHelperEvidence(&result)
		actual.PartialTranscript, actual.PartialChoice, actual.PartialDiagnostic = result.PartialHelperTranscript, result.PartialHelperChoice, result.PartialHelperDiagnostic
		actual.Calls, actual.RuntimeInterrupted, actual.Termination = result.Calls, result.RuntimeInterrupted, result.Termination
	case "served":
		if fixture.OutputLimit != 0 || len(fixture.Frames) != 0 || len(fixture.Operations) == 0 || fixture.Transcript != nil || fixture.Choice != nil || fixture.Diagnostic != nil {
			return actual, fmt.Errorf("invalid served corpus fields")
		}
		environment, err := NewEnvironment(testConfig())
		if err != nil {
			return actual, err
		}
		for i, operation := range fixture.Operations {
			if operation.RepeatData != 0 {
				return actual, fmt.Errorf("unexpected served corpus repeat_data")
			}
			if _, err := environment.Handle(Call{Type: "call", ID: uint64(i + 1), Op: operation.Op, Input: operation.Input}); err != nil {
				return actual, err
			}
		}
		actual.Served = environment.filesystemServed()
	default:
		return actual, fmt.Errorf("invalid namespace adapter corpus kind")
	}
	return actual, nil
}

func checkNamespaceAdapterCorpus(fixture namespaceAdapterCorpus) error {
	actual, err := runNamespaceAdapterCorpus(fixture)
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(actual, fixture.Expected) {
		return fmt.Errorf("namespace adapter projection differs from committed expected bytes/counters")
	}
	return nil
}

func namespaceAdapterInputs() []namespaceAdapterCorpus {
	op := func(name, input string) namespaceConformanceOperation {
		return namespaceConformanceOperation{Op: name, Input: json.RawMessage(input)}
	}
	return []namespaceAdapterCorpus{
		{Schema: namespacePublicCorpusSchema, Name: "helper-output-ab-x-aggregate3", Kind: "output", OutputLimit: 3, Frames: []json.RawMessage{
			json.RawMessage(`{"type":"evidence","stream":"stdout","data_base64":"YWI="}`),
			json.RawMessage(`{"type":"evidence","stream":"stderr","data_base64":"eA=="}`),
			json.RawMessage(`{"type":"evidence","stream":"stderr","data_base64":"eQ=="}`),
		}},
		{Schema: namespacePublicCorpusSchema, Name: "partial-abnormal-prefix-publication", Kind: "partial", Transcript: []byte("GMWT\x01\x00\x00\x00\x08"), Choice: []byte("partial choice"), Diagnostic: []byte("partial diagnostic")},
		{Schema: namespacePublicCorpusSchema, Name: "filesystem-served-errors-versus-clock-poll", Kind: "served", Operations: []namespaceConformanceOperation{
			op("fd_read", `{"fd":99,"length":1}`),
			op("path_create_directory", `{"fd":99,"path_base64":"eA=="}`),
			op("poll_oneoff", `{"subscriptions":[{"userdata":1,"type":1,"fd":99}]}`),
			op("clock_time_get", `{"clock_id":1,"precision":0}`),
			op("poll_oneoff", `{"subscriptions":[{"userdata":2,"type":0,"clock_id":1,"timeout":1,"precision":0,"flags":0}]}`),
		}},
	}
}

func TestNamespaceHostAdapterConformanceCorpus(t *testing.T) {
	for _, input := range namespaceAdapterInputs() {
		t.Run(input.Name, func(t *testing.T) {
			var fixture namespaceAdapterCorpus
			path := filepath.Join("testdata", "namespace-host-adapter-conformance", input.Name+".json")
			namespacePublicFixture(t, path, func() any {
				actual, err := runNamespaceAdapterCorpus(input)
				if err != nil {
					t.Fatal(err)
				}
				input.Expected = actual
				return input
			}, &fixture)
			if fixture.Name != input.Name {
				t.Fatal("namespace adapter case name mismatch")
			}
			if err := checkNamespaceAdapterCorpus(fixture); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestNamespacePublicCorpusRejectsChangedExpectedBytes(t *testing.T) {
	for _, input := range namespaceAdapterInputs()[:2] {
		t.Run(input.Name, func(t *testing.T) {
			actual, err := runNamespaceAdapterCorpus(input)
			if err != nil {
				t.Fatal(err)
			}
			input.Expected = actual
			if err := checkNamespaceAdapterCorpus(input); err != nil {
				t.Fatal(err)
			}
			if input.Kind == "output" {
				input.Expected.Steps[0].Mirror[0] ^= 1
			} else {
				input.Expected.PartialChoice[0] ^= 1
			}
			if err := checkNamespaceAdapterCorpus(input); err == nil {
				t.Fatal("changed committed intermediate mirror/partial bytes were accepted")
			}
		})
	}
}
