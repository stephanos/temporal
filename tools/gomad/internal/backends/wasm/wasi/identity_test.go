package wasi

import (
	"io/fs"
	"testing"
	"testing/fstest"
)

func TestImplementationIdentityBindsStorageEvidenceSources(t *testing.T) {
	for _, name := range []string{"storage_evidence.go", "storage_image.go", "storage_selection.go"} {
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
		})
	}
}

func TestImplementationIdentityBindsCompiledModelDigest(t *testing.T) {
	const expected = "sha256:2b93e302289dbc978c8322c4abdcc037963266c3080be12735923f0187e88bcb"
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
