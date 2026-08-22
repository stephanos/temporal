package readonlymount

import (
	"errors"
	"os"
	"testing"
)

func TestArtifactRoundTripBuildsHostIndependentReplay(t *testing.T) {
	limits := DefaultLimits()
	mappings := []Mapping{{Source: "/host/schema", Target: "/schema"}}
	snapshot := Snapshot{
		Requests: 3, TotalBytes: 4,
		NotExist: []string{"/schema/missing"},
		Entries: []Entry{
			{Path: "/schema", Mode: 0o755, Kind: KindDirectory, Children: []Child{{Name: "file", Mode: 0o640, Kind: KindFile}}},
			{Path: "/schema/file", Mode: 0o640, Kind: KindFile, Data: []byte("data")},
		},
	}
	encoded, err := EncodeCapturedInputs(mappings, limits, snapshot)
	if err != nil {
		t.Fatal(err)
	}
	decodedMappings, decodedLimits, decoded, err := DecodeCapturedInputs(encoded.Manifest, encoded.Descriptor, func(name string, maximum uint64) ([]byte, error) {
		data, found := encoded.Payloads[name]
		if !found || uint64(len(data)) > maximum {
			return nil, os.ErrNotExist
		}
		return append([]byte(nil), data...), nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(decodedMappings) != 1 || decodedMappings[0].Source != "" || decodedMappings[0].Target != "/schema" || decodedLimits != limits || len(decoded.Entries) != 2 || string(decoded.Entries[1].Data) != "data" {
		t.Fatalf("decoded artifact = %#v, %#v, %#v", decodedMappings, decodedLimits, decoded)
	}
	broker, err := PrepareReplay(decodedMappings, decodedLimits, decoded)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = broker.Close() })
	entry, err := broker.Lookup("/schema/file")
	if err != nil || string(entry.Data) != "data" {
		t.Fatalf("Lookup() = %#v, %v", entry, err)
	}
	if _, err := broker.Lookup("/schema/uncaptured"); !errors.Is(err, ErrReplayDivergence) {
		t.Fatalf("uncaptured Lookup() error = %v", err)
	}
	if _, err := broker.Lookup("/schema/missing"); !errors.Is(err, os.ErrNotExist) || errors.Is(err, ErrReplayDivergence) {
		t.Fatalf("captured missing Lookup() error = %v", err)
	}
}

func TestDecodeCapturedInputsRejectsCorruptPayload(t *testing.T) {
	limits := DefaultLimits()
	encoded, err := EncodeCapturedInputs([]Mapping{{Target: "/schema"}}, limits, Snapshot{
		TotalBytes: 4, Entries: []Entry{{Path: "/schema/file", Mode: 0o600, Kind: KindFile, Data: []byte("data")}},
	})
	if err != nil {
		t.Fatal(err)
	}
	_, _, _, err = DecodeCapturedInputs(encoded.Manifest, encoded.Descriptor, func(string, uint64) ([]byte, error) {
		return []byte("evil"), nil
	})
	if err == nil {
		t.Fatal("DecodeCapturedInputs() accepted corrupt payload")
	}
}
