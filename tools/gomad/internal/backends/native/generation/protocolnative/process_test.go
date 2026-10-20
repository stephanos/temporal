package protocolnative

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestGeneratedProcessTransportConsumer(t *testing.T) {
	root := filepath.Join("..", "..", "..", "..", "..")
	schema, err := os.ReadFile(filepath.Join(root, "internal", "model", "schema", "processwire.json"))
	if err != nil {
		t.Fatal(err)
	}
	var definition struct {
		Profile    string `json:"profile"`
		GoldenWait string `json:"golden_wait"`
	}
	if err := json.Unmarshal(schema, &definition); err != nil {
		t.Fatal(err)
	}
	codec, err := os.ReadFile(filepath.Join(root, "internal/backends/native/toolchain/runtime/overlay/src/internal/gomadsim/process_wire_generated.go"))
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
	goCommand := os.Getenv("GOMAD_STOCK_GO")
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
