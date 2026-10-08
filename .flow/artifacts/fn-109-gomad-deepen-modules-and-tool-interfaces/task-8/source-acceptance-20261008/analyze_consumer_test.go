package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	capabilityanalysis "go.temporal.io/server/tools/gomad3/qualification/analysis"
	"go.temporal.io/server/tools/gomad3/target"
)

type sourceAnalyzeWriter struct {
	output    io.Writer
	attempted bytes.Buffer
	calls     int
	err       error
}

func (writer *sourceAnalyzeWriter) Write(data []byte) (int, error) {
	writer.calls++
	if _, err := writer.attempted.Write(data); err != nil {
		return 0, err
	}
	n, err := writer.output.Write(data)
	writer.err = err
	return n, err
}
func sourceAnalyzeOutput(t *testing.T, output io.Writer, closed bool) *sourceAnalyzeWriter {
	t.Helper()
	if closed {
		file, err := os.Open(os.DevNull)
		if err != nil {
			t.Fatal(err)
		}
		if err := file.Close(); err != nil {
			t.Fatal(err)
		}
		output = file
	}
	return &sourceAnalyzeWriter{output: output}
}

func TestAnalyzeSourceActualOwnerConsumer(t *testing.T) {
	for _, test := range []struct {
		name, format, body              string
		failedRemoval                   bool
		failedOutput, failedDiagnostics bool
		status                          int
	}{
		{name: "text", format: "text", body: "func main() {}", status: 0},
		{name: "json", format: "json", body: "func main() {}", status: 0},
		{name: "unsupported", format: "json", body: "import \"os/exec\"\nfunc main() { _ = exec.Command(\"SOURCE-never-run\") }", status: 1},
		{name: "unresolved", format: "json", body: "import _ \"example.com/SOURCE-unavailable\"\nfunc main() {}", status: 2},
		{name: "cleanup-supported", format: "json", body: "func main() {}", failedRemoval: true, status: -1},
		{name: "cleanup-unsupported", format: "json", body: "import \"os/exec\"\nfunc main() { _ = exec.Command(\"SOURCE-never-run\") }", failedRemoval: true, status: 1},
		{name: "cleanup-unresolved", format: "json", body: "import _ \"example.com/SOURCE-unavailable\"\nfunc main() {}", failedRemoval: true, status: 2},
		{name: "failed-output", format: "json", body: "func main() {}", failedOutput: true, status: 3},
		{name: "cleanup-unsupported-failed-diagnostic", format: "json", body: "import \"os/exec\"\nfunc main() { _ = exec.Command(\"SOURCE-never-run\") }", failedRemoval: true, failedDiagnostics: true, status: 3},
		{name: "cleanup-unresolved-failed-diagnostic", format: "json", body: "import _ \"example.com/SOURCE-unavailable\"\nfunc main() {}", failedRemoval: true, failedDiagnostics: true, status: 3},
		{name: "unresolved-failed-diagnostic", format: "json", body: "import _ \"example.com/SOURCE-unavailable\"\nfunc main() {}", failedDiagnostics: true, status: 3},
	} {
		t.Run(test.name, func(t *testing.T) {
			working := t.TempDir()
			for file, body := range map[string]string{"go.mod": "module example.com/sourceconsumer\n\ngo 1.27.1\n", "main.go": "package main\n" + test.body + "\n"} {
				if err := os.WriteFile(filepath.Join(working, file), []byte(body), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			root, err := filepath.Abs(filepath.Join("..", "..", "..", "..", ".toolchain"))
			if err != nil {
				t.Fatal(err)
			}
			calls := 0
			var owned string
			remove := func(path string) error {
				calls++
				owned = path
				if test.failedRemoval {
					t.Cleanup(func() {
						if err := os.RemoveAll(path); err != nil {
							t.Error(err)
						}
					})
					return errors.New("SOURCE-remove-failed")
				}
				return os.RemoveAll(path)
			}
			var stdout, stderr bytes.Buffer
			output, diagnostic := sourceAnalyzeOutput(t, &stdout, test.failedOutput), sourceAnalyzeOutput(t, &stderr, test.failedDiagnostics)
			dependencies := sourceConsumerDependencies(remove)
			dependencies.toolchain = func(explicit string) (string, error) {
				if explicit != root {
					t.Fatalf("resolved root=%q", explicit)
				}
				return root, nil
			}
			dependencies.identity = func(observed string) (target.ToolchainIdentity, error) {
				if observed != root {
					t.Fatalf("identity root=%q", observed)
				}
				return target.ToolchainIdentity{GoVersion: "go1.27.1", BuildKey: strings.Repeat("7", 64), TargetGOOS: "linux", TargetGOARCH: "amd64"}, nil
			}
			dependencies.workingDirectory = func() (string, error) { return working, nil }
			status := runAnalyzeWith([]string{"--format=" + test.format, "--working-dir=" + working, "--toolchain-root=" + root, "go-run", "."}, output, diagnostic, dependencies)
			if test.status >= 0 && status != test.status {
				t.Fatalf("status=%d want=%d stderr=%s", status, test.status, &stderr)
			}
			if calls != 1 || owned == "" {
				t.Fatalf("owner removal calls=%d path=%q", calls, owned)
			}
			_, statErr := os.Stat(owned)
			if test.failedRemoval {
				if statErr != nil {
					t.Fatalf("owner removed despite failure: %v", statErr)
				}
				if !strings.Contains(diagnostic.attempted.String(), "SOURCE-remove-failed") {
					t.Fatalf("cleanup error missing: %s", &stderr)
				}
				if err := os.RemoveAll(owned); err != nil {
					t.Fatal(err)
				}
			} else if !os.IsNotExist(statErr) {
				t.Fatalf("owned root remained: %v", statErr)
			}
			if test.failedOutput && !errors.Is(output.err, os.ErrClosed) || test.failedDiagnostics && !errors.Is(diagnostic.err, os.ErrClosed) {
				t.Fatalf("real write failure not observed: stdout=%v stderr=%v", output.err, diagnostic.err)
			}
			if strings.HasPrefix(test.name, "cleanup-unresolved") && diagnostic.calls != 2 {
				t.Fatalf("primary+cleanup diagnostic attempts=%d", diagnostic.calls)
			}
			if test.name == "json" {
				var report capabilityanalysis.Report
				if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
					t.Fatal(err)
				}
				if report.Schema != capabilityanalysis.AnalysisSchema || report.Classification != capabilityanalysis.ClassificationSupported || report.Toolchain.BuildKey != strings.Repeat("7", 64) {
					t.Fatalf("incomplete report: %#v", report)
				}
			}
			encoded, err := json.Marshal(map[string]any{"name": test.name, "status": status, "stdout": stdout.String(), "stderr": stderr.String(), "stdout_attempted": output.attempted.String(), "stderr_attempted": diagnostic.attempted.String(), "stdout_write_calls": output.calls, "stderr_write_calls": diagnostic.calls, "remove_calls": calls})
			if err != nil {
				t.Fatal(err)
			}
			t.Logf("SOURCE_OBSERVATION %s", encoded)
		})
	}
}
