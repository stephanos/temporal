package wasi

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNamespaceTranscriptConformanceCorpus(t *testing.T) {
	type corpus struct {
		Name       string `json:"name"`
		Transcript []byte `json:"transcript_base64"`
		Valid      bool   `json:"valid"`
		Message    string `json:"message"`
	}
	for _, operation := range []struct{ name, input, message string }{
		{"fd_fdstat_set_flags", `{"fd":3,"flags":2}`, "descriptor flags outside volatile profile"},
		{"path_open", `{"fd":3,"dirflags":0,"path_base64":"eA==","oflags":0,"rights_base":2,"rights_inheriting":0,"fdflags":2}`, "open flags outside volatile profile"},
		{"path_readlink", `{"fd":3,"path_base64":"eA==","length":0}`, "operation outside stock WASI profile: path_readlink"},
		{"fd_fdstat_set_flags_supported", `{"fd":3,"flags":1}`, ""},
		{"path_open_supported", `{"fd":3,"dirflags":0,"path_base64":"eA==","oflags":1,"rights_base":2,"rights_inheriting":0,"fdflags":0}`, ""},
	} {
		environment := newTestEnvironment(t, testConfig())
		op := strings.TrimSuffix(operation.name, "_supported")
		call(t, environment, op, operation.input)
		for _, valid := range []bool{true, false} {
			if !valid && operation.message == "" {
				continue
			}
			transcript := bytes.Clone(environment.Transcript())
			name := operation.name + "-valid-unsupported-message"
			if operation.message == "" {
				name = op + "-valid-supported-record"
			}
			if !valid {
				transcript[len(transcript)-1] = 0xff
				name = operation.name + "-invalid-utf8-unsupported-message"
			}
			test := corpus{name, transcript, valid, operation.message}
			t.Run(name, func(t *testing.T) {
				data, err := json.MarshalIndent(test, "", "  ")
				if err != nil {
					t.Fatal(err)
				}
				data = append(data, '\n')
				path := filepath.Join("testdata", "namespace-transcript-conformance", test.Name+".json")
				if *updateNamespaceConformance {
					if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
						t.Fatal(err)
					}
					if err := os.WriteFile(path, data, 0644); err != nil {
						t.Fatal(err)
					}
				}
				want, err := os.ReadFile(path)
				if err != nil {
					t.Fatal("namespace transcript corpus missing; regenerate with -update-conformance: ", err)
				}
				if !bytes.Equal(data, want) {
					t.Fatalf("namespace transcript drift in %s", path)
				}
				var fixture corpus
				decoder := json.NewDecoder(bytes.NewReader(want))
				decoder.DisallowUnknownFields()
				if err := decoder.Decode(&fixture); err != nil {
					t.Fatal(err)
				}
				records, err := DecodeTranscript(fixture.Transcript)
				if !fixture.Valid {
					if err == nil {
						t.Fatal("accepted malformed UTF8 unsupported message")
					}
					return
				}
				if err != nil {
					t.Fatal(err)
				}
				if len(records) != 1 || records[0].Message != fixture.Message {
					t.Fatalf("decoded unsupported message=%+v", records)
				}
			})
		}
	}
	namespaceTranscriptReservationConformanceCorpus(t)
	namespaceTranscriptPublicConformanceCorpus(t)
}
