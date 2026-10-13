package protocol

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestGeneratedProcessTransportConsumer(t *testing.T) {
	root := filepath.Join("..", "..", "..", "..")
	definition, err := readProcessSchema(filepath.Join(root, "simulation", "schema", "processwire.json"))
	if err != nil {
		t.Fatal(err)
	}
	codec, err := os.ReadFile(filepath.Join(root, "backends/native/toolchain/runtime/overlay/src/internal/gomadsim/process_wire_generated.go"))
	if err != nil {
		t.Fatal(err)
	}
	directory := t.TempDir()
	testSource := fmt.Sprintf(`package gomadsim
import "testing"
func TestHostWaitFrame(t *testing.T) {
 if processRequestBlocksSimulationTime([]byte(%q)) {
  t.Fatal("host wait frame blocks simulation time")
 }
 for _, request := range []string{
  %q,
  %q,
  "",
 } {
  if !processRequestBlocksSimulationTime([]byte(request)) {
   t.Fatalf("non-wait request %%q did not block simulation time", request)
  }
 }
}
`, definition.GoldenWait,
		`{"profile":"`+definition.Profile+`","kind":"start","request":7}`,
		`{"kind":"wait","profile":"`+definition.Profile+`","request":7}`)
	for path, contents := range map[string][]byte{
		"go.mod":  []byte("module processcodec\ngo 1.27.1\n"),
		"wire.go": codec, "wire_test.go": []byte(testSource),
	} {
		if err := os.WriteFile(filepath.Join(directory, path), contents, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	goCommand := os.Getenv("GOMAD3_STOCK_GO")
	if goCommand == "" {
		goCommand = "go"
	}
	command := exec.Command(goCommand, "test", "-tags", "test_dep", "-count=1", ".")
	command.Dir = directory
	command.Env = append(os.Environ(), "GOWORK=off")
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("generated process transport: %v\n%s", err, output)
	}
}

func TestReadProcessSchemaRejectsInvalidDefinition(t *testing.T) {
	contents, err := os.ReadFile(filepath.Join("..", "..", "..", "..", "simulation", "schema", "processwire.json"))
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct{ name, contents string }{
		{"trailing", string(contents) + "{}"},
		{"unknown", strings.Replace(string(contents), `"version": 3`, `"version": 3, "unknown": true`, 1)},
		{"unsupported", strings.Replace(string(contents), `"version": 3`, `"version": 2`, 1)},
		{"prefix-order", strings.Replace(string(contents), `"name": "Profile"`, `"name": "Kind"`, 1)},
	} {
		t.Run(test.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "schema.json")
			if err := os.WriteFile(path, []byte(test.contents), 0o600); err != nil {
				t.Fatal(err)
			}
			if _, err := readProcessSchema(path); err == nil {
				t.Fatal("invalid process definition was accepted")
			}
		})
	}
}
