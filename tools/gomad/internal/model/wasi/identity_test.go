package wasi

import (
	"io/fs"
	"testing"
	"testing/fstest"
)

func TestImplementationIdentityBindsStorageEvidenceSources(t *testing.T) {
	for _, name := range []string{"debug_output.go", "measurement.go", "transport.go", "storage_evidence.go", "storage_image.go", "storage_selection.go"} {
		t.Run(name, func(t *testing.T) {
			sources := fstest.MapFS{}
			entries, err := fs.ReadDir(implementationSources, ".")
			if err != nil {
				t.Fatal(err)
			}
			for _, entry := range entries {
				data, err := implementationSources.ReadFile(entry.Name())
				if err != nil {
					t.Fatal(err)
				}
				sources[entry.Name()] = &fstest.MapFile{Data: data}
			}
			baseline := implementationSHA256(sources)
			sources[name] = &fstest.MapFile{Data: []byte("changed storage implementation")}
			if changed := implementationSHA256(sources); baseline == "" || changed == "" || changed == baseline {
				t.Fatalf("%s did not rotate model identity: %q / %q", name, baseline, changed)
			}
			delete(sources, name)
			if got := implementationSHA256(sources); got != "" {
				t.Fatalf("missing %s returned valid model identity %q", name, got)
			}
		})
	}
}

func TestImplementationIdentityBindsCompiledModelDigest(t *testing.T) {
	const expected = "sha256:bd45d4831ca0f417b6411a29b70cf1070a779e5f04591a679c180d925b586d15"
	if got := ImplementationSHA256(); got != expected {
		t.Fatalf("compiled model identity = %q, want %q", got, expected)
	}
}

func TestImplementationIdentityFailsClosedOnMissingSource(t *testing.T) {
	if got := implementationSHA256(fstest.MapFS{}); got != "" {
		t.Fatalf("missing implementation source returned valid identity %q", got)
	}
}

func TestImplementationIdentityBindsStrictJSONOwner(t *testing.T) {
	baseline := implementationSHA256WithStrictJSON(implementationSources, "sha256:strict-json")
	changed := implementationSHA256WithStrictJSON(implementationSources, "sha256:changed")
	if baseline == "" || changed == baseline {
		t.Fatalf("strict JSON owner did not rotate model identity: %q / %q", baseline, changed)
	}
	if got := implementationSHA256WithStrictJSON(implementationSources, ""); got != "" {
		t.Fatalf("missing strict JSON identity returned valid model identity %q", got)
	}
}
