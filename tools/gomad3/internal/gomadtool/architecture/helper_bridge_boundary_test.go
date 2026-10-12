package architecture

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWASIHelperModelCalculationRemainsPureChecked(t *testing.T) {
	for _, hostile := range []bool{false, true} {
		t.Run(map[bool]string{false: "pure-calculation", true: "hostile-calculation"}[hostile], func(t *testing.T) {
			root := t.TempDir()
			calculation := "e.now++;return e.now"
			imports := ""
			if hostile {
				imports = `import "time"`
				calculation = "_=time.Now();e.now++;return e.now"
			}
			for name, source := range map[string]string{
				"go.mod":                             "module go.temporal.io/server/tools/gomad3\n\ngo 1.27.1\n",
				"backends/wasm/wasi/helper_model.go": "package wasi\n" + imports + "\ntype Environment struct{now uint64};func(e *Environment)helperCallbackReply()uint64{" + calculation + "}",
				"backends/wasm/wasi/process.go":      "package wasi;import \"time\";func helperTransport(){_=time.Now()}",
			} {
				path := filepath.Join(root, name)
				if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, []byte(source), 0600); err != nil {
					t.Fatal(err)
				}
			}
			program, err := Load(root, "go", Platform{OS: "linux", Arch: "amd64"})
			if err != nil {
				t.Fatal(err)
			}
			findings := program.Effects()
			modelHost := false
			for _, finding := range findings {
				if strings.Contains(filepath.ToSlash(finding.Path), "/backends/wasm/wasi/process.go:") {
					t.Fatalf("existing transport root exemption lost: %v", finding)
				}
				if finding.Category == "host-effect" && strings.Contains(filepath.ToSlash(finding.Path), "/backends/wasm/wasi/helper_model.go:") && strings.Contains(finding.Detail, "Environment.helperCallbackReply") {
					modelHost = true
				}
			}
			if modelHost != hostile || !hostile && len(findings) != 0 {
				t.Fatalf("helper model host mutation escaped (hostile=%v): %v", hostile, findings)
			}
		})
	}
}
