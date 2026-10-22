package captured

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"reflect"
	"strings"
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

func TestDecodeCapturedInputsAcceptsAuthenticatedOrdinaryWhitespace(t *testing.T) {
	limits := DefaultLimits()
	limits.Requests = 17
	mappings := []Mapping{{Target: "/schema"}, {Target: "/vendor"}}
	snapshot := Snapshot{
		Requests: 9, TotalBytes: 8, NotExist: []string{"/schema/missing", "/vendor/absent"},
		Entries: []Entry{
			{Path: "/schema", Mode: 0o755, Kind: KindDirectory, Children: []Child{{Name: "file", Mode: 0o640, Kind: KindFile}, {Name: "subdir", Mode: 0o750, Kind: KindDirectory}}},
			{Path: "/schema/file", Mode: 0o640, Kind: KindFile, Data: []byte("data"), Children: []Child{}},
			{Path: "/vendor/file", Mode: 0o600, Kind: KindFile, Data: []byte("more"), Children: []Child{}},
		},
	}
	encoded, err := EncodeCapturedInputs(mappings, limits, snapshot)
	if err != nil {
		t.Fatal(err)
	}
	var indented bytes.Buffer
	if err := json.Indent(&indented, encoded.Descriptor, "", "  "); err != nil {
		t.Fatal(err)
	}
	descriptor := indented.Bytes()
	manifest := encoded.Manifest
	manifest.Bytes = uint64(len(descriptor))
	manifest.SHA256 = HashBytes(descriptor)
	decodedMappings, decodedLimits, decoded, err := DecodeCapturedInputs(manifest, descriptor, func(name string, maximum uint64) ([]byte, error) {
		return encoded.Payloads[name], nil
	})
	if err != nil {
		t.Fatalf("DecodeCapturedInputs(indented descriptor): %v", err)
	}
	for index := range mappings {
		mappings[index].Source = ""
	}
	if !reflect.DeepEqual(decodedMappings, mappings) || decodedLimits != limits || !reflect.DeepEqual(decoded, snapshot) {
		t.Fatalf("decoded mount state = %#v, %#v, %#v; want %#v, %#v, %#v", decodedMappings, decodedLimits, decoded, mappings, limits, snapshot)
	}
}

func TestEncodeCapturedInputsRejectsInvalidOriginalString(t *testing.T) {
	invalid := "/schema/" + string([]byte{0xff})
	_, err := EncodeCapturedInputs([]Mapping{{Target: "/schema"}}, DefaultLimits(), Snapshot{Entries: []Entry{{Path: invalid, Mode: 0o755, Kind: KindDirectory}}})
	if err == nil || !strings.Contains(err.Error(), "valid UTF-8") {
		t.Fatalf("EncodeCapturedInputs(invalid path) error = %v", err)
	}
}

func TestDecodeCapturedInputsRejectsStrictSyntaxSchemaAndDecimalErrors(t *testing.T) {
	encoded, err := EncodeCapturedInputs([]Mapping{{Target: "/schema"}}, DefaultLimits(), Snapshot{})
	if err != nil {
		t.Fatal(err)
	}
	invalidUTF8 := append(append([]byte(nil), encoded.Descriptor...), 0xff)
	for _, test := range []struct {
		name       string
		descriptor []byte
	}{
		{name: "unknown member", descriptor: append(append([]byte(nil), encoded.Descriptor[:len(encoded.Descriptor)-1]...), []byte(`,"unknown":true}`)...)},
		{name: "duplicate member", descriptor: append([]byte(`{"schema":"gomad.io-read-only-mounts/v1",`), encoded.Descriptor[1:]...)},
		{name: "trailing value", descriptor: append(append([]byte(nil), encoded.Descriptor...), []byte(` {}`)...)},
		{name: "invalid UTF-8", descriptor: invalidUTF8},
		{name: "integer token", descriptor: bytes.Replace(encoded.Descriptor, []byte(`"requests":"0"`), []byte(`"requests":0`), 1)},
		{name: "decimal overflow", descriptor: bytes.Replace(encoded.Descriptor, []byte(`"requests":"0"`), []byte(`"requests":"18446744073709551616"`), 1)},
	} {
		t.Run(test.name, func(t *testing.T) {
			manifest := encoded.Manifest
			manifest.Bytes = uint64(len(test.descriptor))
			manifest.SHA256 = HashBytes(test.descriptor)
			if _, _, _, err := DecodeCapturedInputs(manifest, test.descriptor, nil); err == nil {
				t.Fatalf("DecodeCapturedInputs(%q) succeeded", test.descriptor)
			}
		})
	}
}
