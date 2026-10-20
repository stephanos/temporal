package wasi

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

const clusterTransportCodecDirectory = "../wasmhost/tests/testdata/cluster_transport_codec"
const clusterTransportCodecName = "restarted-a-incarnation-two-rejects-old-prefix"
const clusterTransportCodecSchema = "gomad.cluster-transport-codec-corpus/v1"

type clusterTransportCodecTarget struct {
	Node        string `json:"node"`
	Incarnation uint64 `json:"incarnation"`
}
type clusterTransportCodecError struct {
	Kind      string `json:"kind"`
	Dimension string `json:"dimension"`
	Message   string `json:"message"`
}
type clusterTransportCodecExpected struct {
	TaggedResult       string                     `json:"tagged_result"`
	GoAttributionError clusterTransportCodecError `json:"go_attribution_error"`
}
type clusterTransportCodecCase struct {
	Schema                 string                        `json:"schema"`
	Name                   string                        `json:"name"`
	GoImplementationSHA256 string                        `json:"go_implementation_sha256"`
	Target                 clusterTransportCodecTarget   `json:"target"`
	ResultPayload          string                        `json:"result_payload"`
	IncomingFrame          string                        `json:"incoming_frame"`
	Expected               clusterTransportCodecExpected `json:"expected"`
}

func clusterTransportCodecImplementation() (string, error) {
	source, err := os.ReadFile("cluster.go")
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(source)
	return hex.EncodeToString(digest[:]), nil
}
func clusterTransportCodecFrame(target clusterTransportCodecTarget, frame string) error {
	session := &ClusterSession{states: map[BrokerNode]*clusterState{
		{ID: target.Node, Incarnation: target.Incarnation}: {done: make(chan struct{})},
	}}
	return (&clusterStream{session: session}).frame([]byte(frame))
}
func clusterTransportCodecIsUnattributable(err error) bool {
	var boundary *BoundaryError
	return errors.As(err, &boundary) && boundary.Kind == "invalid" && boundary.Dimension == "unattributable cluster frame"
}
func clusterTransportCodecObservation(value clusterTransportCodecCase) (clusterTransportCodecExpected, error) {
	tagged := appendClusterTag(nil, value.Target.Node, value.Target.Incarnation, []byte(value.ResultPayload))
	err := clusterTransportCodecFrame(value.Target, value.IncomingFrame)
	if !clusterTransportCodecIsUnattributable(err) {
		return clusterTransportCodecExpected{}, fmt.Errorf("not an attribution-stage refusal: %v", err)
	}
	var boundary *BoundaryError
	if !errors.As(err, &boundary) {
		return clusterTransportCodecExpected{}, fmt.Errorf("attribution error has no boundary: %v", err)
	}
	return clusterTransportCodecExpected{TaggedResult: string(tagged), GoAttributionError: clusterTransportCodecError{Kind: boundary.Kind, Dimension: boundary.Dimension, Message: err.Error()}}, nil
}
func clusterTransportCodecLiterals(value clusterTransportCodecCase) error {
	pin, err := clusterTransportCodecImplementation()
	if err != nil {
		return err
	}
	if value.Schema != clusterTransportCodecSchema || value.Name != clusterTransportCodecName || value.GoImplementationSHA256 != pin {
		return fmt.Errorf("codec schema/name/Go implementation differs")
	}
	if value.Target != (clusterTransportCodecTarget{Node: "a", Incarnation: 2}) {
		return fmt.Errorf("codec target must be original a/incarnation2")
	}
	if value.ResultPayload != "{\"type\":\"result\"}\n" {
		return fmt.Errorf("codec result payload differs from original minimal result")
	}
	if value.IncomingFrame != "{\"instance\":\"a\",\"type\":\"reply\"}\n" {
		return fmt.Errorf("codec stale input differs from original old tagged reply")
	}
	if value.Expected.TaggedResult != "{\"instance\":\"a\",\"incarnation\":2,\"type\":\"result\"}\n" {
		return fmt.Errorf("codec raw tagged result differs")
	}
	want := clusterTransportCodecError{Kind: "invalid", Dimension: "unattributable cluster frame", Message: "WASI invalid: unattributable cluster frame"}
	if value.Expected.GoAttributionError != want {
		return fmt.Errorf("codec Go refusal must be exact attribution error")
	}
	return nil
}
func newClusterTransportCodecCase() (clusterTransportCodecCase, error) {
	pin, err := clusterTransportCodecImplementation()
	if err != nil {
		return clusterTransportCodecCase{}, err
	}
	value := clusterTransportCodecCase{
		Schema: clusterTransportCodecSchema, Name: clusterTransportCodecName, GoImplementationSHA256: pin,
		Target:        clusterTransportCodecTarget{Node: "a", Incarnation: 2},
		ResultPayload: "{\"type\":\"result\"}\n", IncomingFrame: "{\"instance\":\"a\",\"type\":\"reply\"}\n",
	}
	value.Expected, err = clusterTransportCodecObservation(value)
	if err != nil {
		return value, err
	}
	return value, clusterTransportCodecLiterals(value)
}
func readClusterTransportCodecCase(t *testing.T) clusterTransportCodecCase {
	t.Helper()
	path := filepath.Join(clusterTransportCodecDirectory, clusterTransportCodecName+".json")
	files, err := filepath.Glob(filepath.Join(clusterTransportCodecDirectory, "*.json"))
	if err != nil || !reflect.DeepEqual(files, []string{path}) {
		t.Fatalf("codec corpus must contain exactly its one named case: %v %v", files, err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var value clusterTransportCodecCase
	if err := decoder.Decode(&value); err != nil {
		t.Fatal(err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		t.Fatalf("codec trailing input: %v", err)
	}
	if err := clusterTransportCodecLiterals(value); err != nil {
		t.Fatal(err)
	}
	return value
}
func TestClusterTransportCodecCorpusIsCurrent(t *testing.T) {
	value, err := newClusterTransportCodecCase()
	if err != nil {
		t.Fatal(err)
	}
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	data = append(data, '\n')
	path := filepath.Join(clusterTransportCodecDirectory, clusterTransportCodecName+".json")
	if os.Getenv("GOMAD_CLUSTER_TRANSPORT_CODEC_REGENERATE") == "1" {
		directory := clusterTransportCodecDirectory
		if external := os.Getenv("GOMAD_CLUSTER_TRANSPORT_CODEC_OUTPUT_DIR"); external != "" {
			directory = external
		}
		path = filepath.Join(directory, clusterTransportCodecName+".json")
		if err := os.MkdirAll(directory, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, data, 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	actual, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(actual, data) {
		t.Fatal("shared codec fixture differs; regenerate and review")
	}
}
func TestClusterTransportCodecCorpus(t *testing.T) {
	value := readClusterTransportCodecCase(t)
	actual, err := clusterTransportCodecObservation(value)
	if err != nil {
		t.Fatal(err)
	}
	if actual != value.Expected {
		t.Fatalf("actual Go tag/attribution observation differs: got%+v want%+v", actual, value.Expected)
	}
}
func TestClusterTransportCodecLiteralStageGuards(t *testing.T) {
	value, err := newClusterTransportCodecCase()
	if err != nil {
		t.Fatal(err)
	}
	current := string(appendClusterTag(nil, "a", 2, []byte("{\"type\":\"reply\"}\n")))
	currentErr := clusterTransportCodecFrame(value.Target, current)
	var currentBoundary *BoundaryError
	if !errors.As(currentErr, &currentBoundary) || currentBoundary.Kind != "unsupported" || currentBoundary.Dimension != "unknown cluster frame reply" {
		t.Fatalf("current tag did not pass attribution before payload refusal: %v", currentErr)
	}
	if clusterTransportCodecIsUnattributable(currentErr) {
		t.Fatal("current-tag payload refusal counted as stale attribution")
	}
	malformedErr := clusterTransportCodecFrame(value.Target, "{\n")
	if malformedErr == nil || clusterTransportCodecIsUnattributable(malformedErr) {
		t.Fatalf("malformed JSON counted as stale attribution: %v", malformedErr)
	}
	mutations := []struct {
		name string
		edit func(*clusterTransportCodecCase)
	}{
		{"raw-result-kind", func(c *clusterTransportCodecCase) {
			c.Expected.TaggedResult = "{\"instance\":\"a\",\"incarnation\":2,\"type\":\"call\"}\n"
		}},
		{"target", func(c *clusterTransportCodecCase) { c.Target.Node = "b" }},
		{"incarnation", func(c *clusterTransportCodecCase) { c.Target.Incarnation = 1 }},
		{"result-payload", func(c *clusterTransportCodecCase) { c.ResultPayload = "{\"type\":\"call\"}\n" }},
		{"current-tag-not-stale", func(c *clusterTransportCodecCase) { c.IncomingFrame = current }},
		{"payload-stage-error", func(c *clusterTransportCodecCase) {
			c.Expected.GoAttributionError = clusterTransportCodecError{Kind: "unsupported", Dimension: "unknown cluster frame reply", Message: "WASI unsupported: unknown cluster frame reply"}
		}},
	}
	for _, mutation := range mutations {
		t.Run(mutation.name, func(t *testing.T) {
			changed := value
			mutation.edit(&changed)
			if err := clusterTransportCodecLiterals(changed); err == nil {
				t.Fatal("literal/stage mutation accepted")
			}
		})
	}
}
