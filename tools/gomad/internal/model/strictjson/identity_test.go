package strictjson

import (
	"bytes"
	"os"
	"strings"
	"testing"
	"testing/fstest"
)

func TestImplementationIdentityBindsSourceAndDependencyPin(t *testing.T) {
	baseline := ImplementationSHA256()
	if baseline == "" {
		t.Fatal("strict JSON implementation identity is empty")
	}
	sources := fstest.MapFS{}
	for _, path := range implementationPaths {
		content, err := implementationSources.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		sources[path] = &fstest.MapFile{Data: content}
	}
	sources["decode.go"].Data = append(bytes.Clone(sources["decode.go"].Data), '\n')
	changed, err := implementationSHA256(sources, dependencyVersion, dependencySum)
	if err != nil || changed == baseline {
		t.Fatalf("source drift identity = %q, %v", changed, err)
	}
	changed, err = implementationSHA256(implementationSources, dependencyVersion+"-changed", dependencySum)
	if err != nil || changed == baseline {
		t.Fatalf("dependency drift identity = %q, %v", changed, err)
	}
	if _, err := implementationSHA256(fstest.MapFS{}, dependencyVersion, dependencySum); err == nil {
		t.Fatal("missing strict JSON implementation source produced an identity")
	}
}

func TestDependencyPinMatchesModuleFiles(t *testing.T) {
	module, err := os.ReadFile("../../../go.mod")
	if err != nil {
		t.Fatal(err)
	}
	foundModule := false
	for _, line := range strings.Split(string(module), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 && fields[0] == dependencyPath && fields[1] == dependencyVersion {
			foundModule = true
			break
		}
	}
	if !foundModule {
		t.Fatalf("%s %s is not pinned in go.mod", dependencyPath, dependencyVersion)
	}
	sums, err := os.ReadFile("../../../go.sum")
	if err != nil {
		t.Fatal(err)
	}
	want := dependencyPath + " " + dependencyVersion + " " + dependencySum
	if !containsLine(sums, want) {
		t.Fatalf("dependency content sum is not pinned in go.sum: %s", want)
	}
}

func containsLine(data []byte, want string) bool {
	for _, line := range bytes.Split(data, []byte{'\n'}) {
		if string(line) == want {
			return true
		}
	}
	return false
}
