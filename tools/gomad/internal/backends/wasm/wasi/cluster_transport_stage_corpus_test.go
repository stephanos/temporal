package wasi

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

const clusterTransportStageDirectory = "testdata/cluster-transport-stage-conformance"
const clusterTransportStageSchema = "gomad.go-cluster-transport-stage-corpus/v1"

type clusterTransportStageCase struct {
	Schema                 string                      `json:"schema"`
	Name                   string                      `json:"name"`
	GoImplementationSHA256 string                      `json:"go_implementation_sha256"`
	Target                 clusterTransportCodecTarget `json:"target"`
	IncomingFrame          string                      `json:"incoming_frame"`
	Expected               clusterTransportCodecError  `json:"expected"`
}

func clusterTransportStageObservation(value clusterTransportStageCase) (clusterTransportCodecError, error) {
	err := clusterTransportCodecFrame(value.Target, value.IncomingFrame)
	var boundary *BoundaryError
	if !errors.As(err, &boundary) {
		return clusterTransportCodecError{}, fmt.Errorf("expected typed boundary, got %v", err)
	}
	return clusterTransportCodecError{Kind: boundary.Kind, Dimension: boundary.Dimension, Message: err.Error()}, nil
}

func clusterTransportStageLiterals(value clusterTransportStageCase) error {
	pin, err := clusterTransportCodecImplementation()
	if err != nil {
		return err
	}
	if value.Schema != clusterTransportStageSchema || value.GoImplementationSHA256 != pin || value.Target != (clusterTransportCodecTarget{Node: "a", Incarnation: 2}) {
		return fmt.Errorf("stage schema/implementation/target differs")
	}
	var frame, kind, dimension string
	switch value.Name {
	case "current-tag-reaches-payload":
		frame, kind, dimension = "{\"instance\":\"a\",\"incarnation\":2,\"type\":\"reply\"}\n", "unsupported", "unknown cluster frame reply"
	case "malformed-json-before-attribution":
		frame, kind, dimension = "{\n", "invalid", "malformed JSON"
	default:
		return fmt.Errorf("unknown stage case %q", value.Name)
	}
	if value.IncomingFrame != frame || value.Expected != (clusterTransportCodecError{Kind: kind, Dimension: dimension, Message: "WASI " + kind + ": " + dimension}) {
		return fmt.Errorf("stage input/exact refusal differs")
	}
	return nil
}

func newClusterTransportStageCases() ([]clusterTransportStageCase, error) {
	pin, err := clusterTransportCodecImplementation()
	if err != nil {
		return nil, err
	}
	values := []clusterTransportStageCase{
		{Schema: clusterTransportStageSchema, Name: "current-tag-reaches-payload", GoImplementationSHA256: pin, Target: clusterTransportCodecTarget{Node: "a", Incarnation: 2}, IncomingFrame: "{\"instance\":\"a\",\"incarnation\":2,\"type\":\"reply\"}\n"},
		{Schema: clusterTransportStageSchema, Name: "malformed-json-before-attribution", GoImplementationSHA256: pin, Target: clusterTransportCodecTarget{Node: "a", Incarnation: 2}, IncomingFrame: "{\n"},
	}
	for i := range values {
		values[i].Expected, err = clusterTransportStageObservation(values[i])
		if err != nil {
			return nil, err
		}
		if err := clusterTransportStageLiterals(values[i]); err != nil {
			return nil, err
		}
	}
	return values, nil
}

func TestClusterTransportStageCorpusIsCurrent(t *testing.T) {
	values, err := newClusterTransportStageCases()
	if err != nil {
		t.Fatal(err)
	}
	for _, value := range values {
		data, err := json.MarshalIndent(value, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		data = append(data, '\n')
		directory := clusterTransportStageDirectory
		if os.Getenv("GOMAD_CLUSTER_TRANSPORT_STAGE_REGENERATE") == "1" {
			if external := os.Getenv("GOMAD_CLUSTER_TRANSPORT_STAGE_OUTPUT_DIR"); external != "" {
				directory = external
			}
			if err := os.MkdirAll(directory, 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(directory, value.Name+".json"), data, 0o644); err != nil {
				t.Fatal(err)
			}
			continue
		}
		actual, err := os.ReadFile(filepath.Join(directory, value.Name+".json"))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(actual, data) {
			t.Fatalf("%s differs; regenerate and review", value.Name)
		}
	}
}

func TestClusterTransportStageCorpus(t *testing.T) {
	names, err := filepath.Glob(filepath.Join(clusterTransportStageDirectory, "*.json"))
	if err != nil {
		t.Fatal(err)
	}
	want := []string{filepath.Join(clusterTransportStageDirectory, "current-tag-reaches-payload.json"), filepath.Join(clusterTransportStageDirectory, "malformed-json-before-attribution.json")}
	if !reflect.DeepEqual(names, want) {
		t.Fatalf("stage corpus exact two-case roster differs: %v", names)
	}
	for _, name := range names {
		data, err := os.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		decoder := json.NewDecoder(bytes.NewReader(data))
		decoder.DisallowUnknownFields()
		var value clusterTransportStageCase
		if err := decoder.Decode(&value); err != nil {
			t.Fatal(err)
		}
		var trailing any
		if err := decoder.Decode(&trailing); err != io.EOF {
			t.Fatalf("stage trailing input: %v", err)
		}
		if filepath.Base(name) != value.Name+".json" {
			t.Fatal("stage filename/name mismatch")
		}
		if err := clusterTransportStageLiterals(value); err != nil {
			t.Fatal(err)
		}
		actual, err := clusterTransportStageObservation(value)
		if err != nil {
			t.Fatal(err)
		}
		if actual != value.Expected {
			t.Fatalf("%s actual stage differs: got%+v want%+v", value.Name, actual, value.Expected)
		}
	}
}

func TestClusterTransportStageLiteralGuards(t *testing.T) {
	values, err := newClusterTransportStageCases()
	if err != nil {
		t.Fatal(err)
	}
	for _, value := range values {
		if err := clusterTransportStageLiterals(value); err != nil {
			t.Fatal(err)
		}
		for _, mutation := range []struct {
			name string
			edit func(*clusterTransportStageCase)
		}{
			{"wrong-stage", func(c *clusterTransportStageCase) {
				c.Expected = clusterTransportCodecError{Kind: "invalid", Dimension: "unattributable cluster frame", Message: "WASI invalid: unattributable cluster frame"}
			}},
			{"wrong-target", func(c *clusterTransportStageCase) { c.Target.Node = "b" }},
			{"stale-input", func(c *clusterTransportStageCase) { c.IncomingFrame = "{\"instance\":\"a\",\"type\":\"reply\"}\n" }},
		} {
			t.Run(value.Name+"/"+mutation.name, func(t *testing.T) {
				changed := value
				mutation.edit(&changed)
				if clusterTransportStageLiterals(changed) == nil {
					t.Fatal("stage mutation accepted")
				}
			})
		}
	}
}
