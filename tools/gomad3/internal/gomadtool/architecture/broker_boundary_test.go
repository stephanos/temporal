package architecture

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWASIBrokerTransportExceptionIsExact(t *testing.T) {
	root := t.TempDir()
	for name, source := range map[string]string{
		"go.mod":                                        "module go.temporal.io/server/tools/gomad3\n\ngo 1.27.1\n",
		"backends/wasm/wasi/broker_process.go":          "package wasi; import \"time\"; func HostWait(){ _=time.Now() }",
		"backends/wasm/wasi/broker.go":                  "package wasi; import \"time\"; func ModelClock(){ _=time.Now() }",
		"backends/wasm/wasi/broker_process_adjacent.go": "package wasi; import \"time\"; func Adjacent(){ _=time.Now() }",
		"backends/wasm/wasi/renamed_process.go":         "package wasi; import \"time\"; func Renamed(){ _=time.Now() }",
		"backends/wasm/wasi/nested/broker_process.go":   "package nested; import \"time\"; func Nested(){ _=time.Now() }",
	} {
		path := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(source), 0600); err != nil {
			t.Fatal(err)
		}
	}
	p, err := Load(root, "go", Platform{OS: "linux", Arch: "amd64"})
	if err != nil {
		t.Fatal(err)
	}
	findings := p.Effects()
	for _, name := range []string{"backends/wasm/wasi/broker.go", "backends/wasm/wasi/broker_process_adjacent.go", "backends/wasm/wasi/renamed_process.go", "backends/wasm/wasi/nested/broker_process.go"} {
		found := false
		for _, finding := range findings {
			found = found || strings.Contains(filepath.ToSlash(finding.Path), name+":") && finding.Category == "host-effect"
		}
		if !found {
			t.Errorf("model/adjacent/renamed/nested file escaped checking: %s, findings=%v", name, findings)
		}
	}
	for _, finding := range findings {
		if strings.Contains(filepath.ToSlash(finding.Path), "/backends/wasm/wasi/broker_process.go:") {
			t.Errorf("exact host boundary was checked as pure: %v", finding)
		}
	}
}
