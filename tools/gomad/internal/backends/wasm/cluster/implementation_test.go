package simulation

import (
	"io/fs"
	"testing"
	"testing/fstest"
)

func TestImplementationIdentityBindsInspectionOwners(t *testing.T) {
	for _, name := range []string{"inspection_control.go", "inspection_network.go", "replay_inspection.go"} {
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
			before := implementationSHA256(sources)
			sources[name] = &fstest.MapFile{Data: []byte("changed inspection implementation")}
			if after := implementationSHA256(sources); before == "" || after == "" || before == after {
				t.Fatalf("%s did not rotate implementation identity: %q / %q", name, before, after)
			}
			delete(sources, name)
			if implementationSHA256(sources) != "" {
				t.Fatalf("missing %s did not refuse implementation identity", name)
			}
		})
	}
}
