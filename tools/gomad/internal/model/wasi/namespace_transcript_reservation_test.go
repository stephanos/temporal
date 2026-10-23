package wasi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

type namespaceTranscriptReservation struct {
	Op               string          `json:"op"`
	Input            json.RawMessage `json:"input"`
	Limit            uint64          `json:"limit"`
	NormalTranscript []byte          `json:"normal_transcript_base64"`
	ShortRefusal     string          `json:"short_refusal"`
}

func checkNamespaceTranscriptReservation(reservation namespaceTranscriptReservation, transcript []byte) error {
	if reservation.Limit < 2 {
		return fmt.Errorf("invalid exact reservation limit %d", reservation.Limit)
	}
	run := func(limit uint64) (*Environment, error) {
		config := testConfig()
		config.Limits.TranscriptBytes = limit
		environment, err := NewEnvironment(config)
		if err != nil {
			return nil, err
		}
		_, err = environment.Handle(Call{Type: "call", ID: 1, Op: reservation.Op, Input: reservation.Input})
		return environment, err
	}
	normal, err := run(testConfig().Limits.TranscriptBytes)
	if err != nil {
		return err
	}
	if !bytes.Equal(normal.Transcript(), reservation.NormalTranscript) {
		return fmt.Errorf("default-limit transcript differs from committed bytes")
	}
	exact, err := run(reservation.Limit)
	if err != nil {
		return err
	}
	if !bytes.Equal(exact.Transcript(), transcript) {
		return fmt.Errorf("exact-limit transcript differs from committed bytes")
	}
	if !bytes.Equal(exact.Transcript(), normal.Transcript()) {
		return fmt.Errorf("exact-limit transcript differs from default-limit transcript")
	}
	_, err = run(reservation.Limit - 1)
	if err == nil {
		return fmt.Errorf("one-byte-short reservation was accepted")
	}
	boundary, ok := err.(*BoundaryError)
	if !ok || boundary.Kind+":"+boundary.Dimension != reservation.ShortRefusal {
		return fmt.Errorf("one-byte-short refusal=%v, expected %s", err, reservation.ShortRefusal)
	}
	return nil
}

func TestNamespaceTranscriptReservationRejectsChangedNormalBytes(t *testing.T) {
	environment := newTestEnvironment(t, testConfig())
	input := json.RawMessage(`{"fd":3,"flags":2}`)
	call(t, environment, "fd_fdstat_set_flags", string(input))
	transcript := environment.Transcript()
	reservation := namespaceTranscriptReservation{Op: "fd_fdstat_set_flags", Input: input, Limit: 75, NormalTranscript: bytes.Clone(transcript), ShortRefusal: "capacity:transcript-bytes"}
	if err := checkNamespaceTranscriptReservation(reservation, transcript); err != nil {
		t.Fatal(err)
	}
	reservation.NormalTranscript[0] ^= 1
	if err := checkNamespaceTranscriptReservation(reservation, transcript); err == nil {
		t.Fatal("changed committed default-limit bytes were accepted")
	}
}

func namespaceTranscriptReservationConformanceCorpus(t *testing.T) {
	t.Helper()
	type corpus struct {
		Name        string                         `json:"name"`
		Reservation namespaceTranscriptReservation `json:"reservation"`
		Transcript  []byte                         `json:"transcript_base64"`
	}
	for _, test := range []struct {
		name, op, input string
		limit           uint64
	}{
		{"fd_fdstat_set_flags-unsupported", "fd_fdstat_set_flags", `{"fd":3,"flags":2}`, 75},
		{"fd_fdstat_set_flags-supported", "fd_fdstat_set_flags", `{"fd":3,"flags":1}`, 75},
		{"path_open-unsupported", "path_open", `{"fd":3,"dirflags":0,"path_base64":"eA==","oflags":0,"rights_base":2,"rights_inheriting":0,"fdflags":2}`, 96},
		{"path_open-supported", "path_open", `{"fd":3,"dirflags":0,"path_base64":"eA==","oflags":1,"rights_base":2,"rights_inheriting":0,"fdflags":0}`, 96},
		{"path_readlink-unsupported", "path_readlink", `{"fd":3,"path_base64":"eA==","length":0}`, 92},
	} {
		t.Run(test.name+"-exact-reservation", func(t *testing.T) {
			normal := newTestEnvironment(t, testConfig())
			call(t, normal, test.op, test.input)
			config := testConfig()
			config.Limits.TranscriptBytes = test.limit
			exact := newTestEnvironment(t, config)
			call(t, exact, test.op, test.input)
			fixture := corpus{Name: test.name + "-exact-reservation", Reservation: namespaceTranscriptReservation{Op: test.op, Input: json.RawMessage(test.input), Limit: test.limit, NormalTranscript: normal.Transcript(), ShortRefusal: "capacity:transcript-bytes"}, Transcript: exact.Transcript()}
			data, err := json.MarshalIndent(fixture, "", "  ")
			if err != nil {
				t.Fatal(err)
			}
			data = append(data, '\n')
			path := filepath.Join("testdata", "namespace-transcript-conformance", fixture.Name+".json")
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
				t.Fatal("namespace reservation corpus missing; regenerate with -update-conformance: ", err)
			}
			if !bytes.Equal(data, want) {
				t.Fatalf("namespace reservation corpus drift in %s", path)
			}
			decoder := json.NewDecoder(bytes.NewReader(want))
			decoder.DisallowUnknownFields()
			if err := decoder.Decode(&fixture); err != nil {
				t.Fatal(err)
			}
			if err := checkNamespaceTranscriptReservation(fixture.Reservation, fixture.Transcript); err != nil {
				t.Fatal(err)
			}
		})
	}
}
