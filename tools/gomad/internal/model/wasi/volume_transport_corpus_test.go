package wasi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestVolumeTransportComparisonGuards(t *testing.T) {
	want := volumeTransportRead(t, "capacity-pending-operations.json")
	initial, observations := volumeTransportRun(t, namespaceConformanceHelper(t), want)
	if err := volumeTransportCompare(want, initial, observations); err != nil {
		t.Fatal(err)
	}
	for _, mutation := range []string{"initial-reply", "reply-byte", "observation-count"} {
		t.Run(mutation, func(t *testing.T) {
			changed := want
			changed.Observations = append([]volumeConformanceObservation(nil), want.Observations...)
			switch mutation {
			case "initial-reply":
				changed.InitialReply = append(append([]byte(nil), want.InitialReply...), '\n')
			case "reply-byte":
				changed.Observations[0].Reply = append(append([]byte(nil), want.Observations[0].Reply...), '\n')
			case "observation-count":
				changed.Observations = nil
			}
			if volumeTransportCompare(changed, initial, observations) == nil {
				t.Fatalf("changed %s accepted", mutation)
			}
		})
	}
	for name, input := range map[string]string{"unknown-observation-field": `{"unexpected":1}`, "trailing-observation": `{} {}`} {
		t.Run(name, func(t *testing.T) {
			var observation volumeConformanceObservation
			if volumeTransportDecode([]byte(input), &observation) == nil {
				t.Fatal("invalid observation accepted")
			}
		})
	}
}

func TestVolumeTransportCorpus(t *testing.T) {
	helper := namespaceConformanceHelper(t)
	entries, err := os.ReadDir(filepath.Join("testdata", "volume-conformance"))
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		count++
		want := volumeTransportRead(t, entry.Name())
		t.Run(want.Name, func(t *testing.T) {
			initial, observations := volumeTransportRun(t, helper, want)
			if err := volumeTransportCompare(want, initial, observations); err != nil {
				t.Fatal(err)
			}
		})
	}
	if count != 115 {
		t.Fatalf("volume corpus roster: got %d want 115", count)
	}
}

func volumeTransportDecode(data []byte, value any) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(value); err != nil {
		return err
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return fmt.Errorf("trailing volume JSON: %v", err)
	}
	return nil
}

func volumeTransportRead(t *testing.T, name string) volumeConformanceCorpus {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", "volume-conformance", name))
	if err != nil {
		t.Fatal(err)
	}
	var fixture volumeConformanceCorpus
	if err := volumeTransportDecode(data, &fixture); err != nil {
		t.Fatal(err)
	}
	if fixture.Schema != "gomad.volume-conformance/v1" || fixture.Name != strings.TrimSuffix(name, ".json") || fixture.Origin == "" || fixture.Session.Model != volumeConformanceModel || !json.Valid(fixture.InitialReply) || len(fixture.Operations) != len(fixture.Observations) {
		t.Fatal("invalid volume transport schema/name/model/count")
	}
	for step, operation := range fixture.Operations {
		if operation.Op == "" || !json.Valid(operation.Input) {
			t.Fatalf("invalid volume operation at step %d", step)
		}
	}
	return fixture
}

func volumeTransportRun(t *testing.T, helper string, fixture volumeConformanceCorpus) ([]byte, []volumeConformanceObservation) {
	t.Helper()
	exchange, closeDriver := namespaceConformanceTransport(t, helper, fixture.Name)
	defer closeDriver()
	initial := exchange(fixture.Session)
	var observations []volumeConformanceObservation
	for step, operation := range fixture.Operations {
		var observation volumeConformanceObservation
		if err := volumeTransportDecode(exchange(operation), &observation); err != nil {
			t.Fatalf("step=%d invalid volume reply: %v", step, err)
		}
		observations = append(observations, observation)
	}
	return initial, observations
}

func volumeTransportCompare(want volumeConformanceCorpus, initial []byte, observations []volumeConformanceObservation) error {
	if !bytes.Equal(want.InitialReply, initial) {
		return fmt.Errorf("volume initial reply differs")
	}
	if len(want.Observations) != len(want.Operations) || len(observations) != len(want.Observations) {
		return fmt.Errorf("volume observation count differs")
	}
	for step, got := range observations {
		if !reflect.DeepEqual(want.Observations[step], got) {
			return fmt.Errorf("volume complete observation differs at step %d", step)
		}
	}
	return nil
}
