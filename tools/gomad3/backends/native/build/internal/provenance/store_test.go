package provenance

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStoreRoundTripsOrdinaryPrivateDocument(t *testing.T) {
	path := filepath.Join(t.TempDir(), "provenance.json")
	written, err := Store(path, map[string]any{"schema": "test/v1", "value": "<ordinary>"})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(written), `{"schema":"test/v1","value":"\u003cordinary\u003e"}`; got != want {
		t.Fatalf("Store() = %q, want ordinary stdlib JSON %q", got, want)
	}
	var decoded struct {
		Schema string `json:"schema"`
		Value  string `json:"value"`
	}
	read, err := Load(path, 1024, &decoded)
	if err != nil {
		t.Fatal(err)
	}
	if string(read) != string(written) || decoded.Schema != "test/v1" || decoded.Value != "<ordinary>" {
		t.Fatalf("Load() = %q, %#v", read, decoded)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("mode = %o", info.Mode().Perm())
	}
}

func TestLoadAcceptsOrdinarySpellingAndRejectsAmbiguousInput(t *testing.T) {
	type document struct {
		Schema string `json:"schema"`
		Value  string `json:"value"`
	}
	invalidUTF8 := append([]byte(`{"schema":"test/v1","value":"`), 0xff)
	invalidUTF8 = append(invalidUTF8, []byte(`"}`)...)
	for _, test := range []struct {
		name    string
		input   []byte
		want    document
		wantErr bool
	}{
		{name: "ordinary spelling", input: []byte("{\n  \"VALUE\": \"\\u003cvalue\\u003e\", \"schema\": \"test/v1\"\n}"), want: document{Schema: "test/v1", Value: "<value>"}},
		{name: "duplicate", input: []byte(`{"schema":"first","schema":"second"}`), wantErr: true},
		{name: "unknown", input: []byte(`{"schema":"test/v1","unknown":true}`), wantErr: true},
		{name: "trailing", input: []byte(`{"schema":"test/v1"} {}`), wantErr: true},
		{name: "invalid UTF-8", input: invalidUTF8, wantErr: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "provenance.json")
			if err := os.WriteFile(path, test.input, 0o600); err != nil {
				t.Fatal(err)
			}
			var got document
			_, err := Load(path, 1024, &got)
			if test.wantErr {
				if err == nil {
					t.Fatalf("Load(%q) succeeded: %#v", test.input, got)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got != test.want {
				t.Fatalf("Load(%q) = %#v, want %#v", test.input, got, test.want)
			}
		})
	}
}
