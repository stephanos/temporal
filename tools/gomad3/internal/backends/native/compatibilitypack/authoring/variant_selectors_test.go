package authoring

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestSelectedLibcVariantsResolveTheirMappedModuleVersions(t *testing.T) {
	directories, err := LoadWorkingDirectories("..")
	if err != nil {
		t.Fatal(err)
	}
	for id, version := range map[string]string{"modernc-libc-xsys-v041": "v0.41.0", "modernc-libc-xsys-v047": "v0.47.0"} {
		t.Run(id, func(t *testing.T) {
			directory, err := filepath.Abs(directories[id])
			if err != nil {
				t.Fatal(err)
			}
			before := map[string][]byte{}
			for _, name := range []string{"go.mod", "go.sum"} {
				contents, err := os.ReadFile(filepath.Join(directory, name))
				if err != nil {
					t.Fatal(err)
				}
				before[name] = contents
			}
			requestBytes, err := os.ReadFile(filepath.Join("..", "requests", id+".json"))
			if err != nil {
				t.Fatal(err)
			}
			request, err := DecodeRequest(requestBytes)
			if err != nil {
				t.Fatal(err)
			}
			command := exec.CommandContext(t.Context(), "go", "list", "-mod=readonly", "-m", "-json", "golang.org/x/sys")
			command.Dir = directory
			command.Env = append(os.Environ(), "GOENV=off", "GOWORK=off", "GOTOOLCHAIN=local", "GOFLAGS=")
			output, err := command.CombinedOutput()
			if err != nil {
				t.Fatalf("resolve selected %s: %v\n%s", id, err, output)
			}
			var selected struct{ Path, Version, Sum string }
			if err := json.Unmarshal(output, &selected); err != nil {
				t.Fatal(err)
			}
			if selected.Path != "golang.org/x/sys" || selected.Version != version || selected.Sum == "" {
				t.Fatalf("selected %s module = %+v", id, selected)
			}
			matched := false
			for _, activation := range request.Activation {
				if activation.Path == selected.Path && activation.Evidence.Version == selected.Version && activation.Evidence.Sum == selected.Sum {
					matched = true
				}
			}
			if !matched {
				t.Fatalf("%s request does not bind its actually selected module %+v", id, selected)
			}
			for name, expected := range before {
				actual, err := os.ReadFile(filepath.Join(directory, name))
				if err != nil || !bytes.Equal(expected, actual) {
					t.Fatalf("read-only selector listing changed %s: %v", name, err)
				}
			}
			t.Logf("mapped %s in %s resolves %s@%s %s; variant remains selected, native closure qualification is separate", id, directory, selected.Path, selected.Version, selected.Sum)
		})
	}
}
