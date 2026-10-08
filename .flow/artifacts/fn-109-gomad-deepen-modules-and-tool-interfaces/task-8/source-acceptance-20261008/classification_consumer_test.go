package cli

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomad3/deterministicio"
	"go.temporal.io/server/tools/gomad3/target"
)

func TestAnalyzeSourceActualOwnerClassification(t *testing.T) {
	for _, name := range []string{"missing-sum", "invalid-sum", "malformed-linked"} {
		t.Run(name, func(t *testing.T) {
			working, root := t.TempDir(), t.TempDir()
			if name == "missing-sum" {
				working = filepath.Join(sourceClassificationFixture, name)
				if err := os.Mkdir(working, 0o700); err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() {
					if err := os.RemoveAll(working); err != nil {
						t.Error(err)
					}
				})
			}
			module := []byte("module example.com/sourceclassification\n\ngo 1.27.1\n")
			if name != "malformed-linked" {
				var err error
				module, err = os.ReadFile(filepath.Join("..", "..", "..", "..", "deterministicio", "testdata", "sprig", "go.mod"))
				if err != nil {
					t.Fatal(err)
				}
			}
			for file, body := range map[string][]byte{"go.mod": module, "main.go": []byte("package main\nfunc main() { panic(\"SOURCE launched target\") }\n")} {
				if err := os.WriteFile(filepath.Join(working, file), body, 0o600); err != nil {
					t.Fatal(err)
				}
			}
			if name == "invalid-sum" {
				if err := os.WriteFile(filepath.Join(working, "go.sum"), []byte("github.com/Masterminds/sprig/v3 v3.3.0 h1:SOURCE-deliberately-invalid\n"), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			driver, err := os.ReadFile(sourceClassificationDriver)
			if err != nil {
				t.Fatal(err)
			}
			for _, directory := range []string{"bin", "builds/" + strings.Repeat("7", 64) + "/bin"} {
				if err := os.MkdirAll(filepath.Join(root, directory), 0o700); err != nil {
					t.Fatal(err)
				}
			}
			for file, body := range map[string][]byte{"bin/go": driver, "builds/" + strings.Repeat("7", 64) + "/bin/go": driver, "build-key": []byte(strings.Repeat("7", 64) + "\n"), "allowed-working": []byte(working)} {
				if err := os.WriteFile(filepath.Join(root, file), body, 0o700); err != nil {
					t.Fatal(err)
				}
			}
			calls := 0
			var observed error
			dependencies := sourceClassificationDependencies(func(path string) error { calls++; return os.RemoveAll(path) }, func(err error) { observed = err })
			dependencies.toolchain = func(string) (string, error) { return root, nil }
			dependencies.identity = func(string) (target.ToolchainIdentity, error) {
				return target.ToolchainIdentity{GoVersion: "go1.27.1", BuildKey: strings.Repeat("7", 64), TargetGOOS: "linux", TargetGOARCH: "amd64"}, nil
			}
			dependencies.workingDirectory = func() (string, error) { return working, nil }
			var stdout, stderr bytes.Buffer
			output, diagnostic := sourceAnalyzeOutput(t, &stdout, false), sourceAnalyzeOutput(t, &stderr, false)
			args := []string{"--format=json", "--toolchain-root=" + root, "--working-dir=" + working}
			if name == "malformed-linked" {
				args = append(args, "--capability-mode=linked")
			}
			status := runAnalyzeWith(append(args, "go-run", "."), output, diagnostic, dependencies)
			want := 2
			if name == "malformed-linked" {
				want = 3
			}
			if status != want || calls != 1 || observed == nil || stdout.Len() != 0 || diagnostic.calls != 1 || deterministicio.IsInvalidBuildAdapterConfiguration(observed) != (name != "malformed-linked") || target.IsInvalidCapabilityReview(observed) || target.IsUnsupportedCapability(observed) {
				t.Fatalf("status=%d removals=%d diagnostic writes=%d error=%v", status, calls, diagnostic.calls, observed)
			}
			var command struct {
				Commands []struct {
					Command []string `json:"command"`
					Output  string   `json:"output"`
					Bytes   string   `json:"output_bytes"`
					SHA256  string   `json:"output_sha256"`
					Length  int      `json:"output_length"`
				} `json:"commands"`
			}
			if name == "malformed-linked" {
				if !strings.Contains(stderr.String(), "extract linked target capability manifest") {
					t.Fatalf("not actual extraction: %s", &stderr)
				}
				data, err := os.ReadFile(filepath.Join(root, "build-command.json"))
				if err != nil {
					t.Fatal(err)
				}
				if err := json.Unmarshal(data, &command); err != nil {
					t.Fatal(err)
				}
				if len(command.Commands) != 1 {
					t.Fatalf("actual interceptions=%d", len(command.Commands))
				}
				entry := command.Commands[0]
				hash := sha256.Sum256([]byte(entry.Bytes))
				if entry.Bytes != "SOURCE-malformed-linked-object\n" || entry.SHA256 != hex.EncodeToString(hash[:]) || entry.Length != len(entry.Bytes) {
					t.Fatalf("actual output ledger=%#v", entry)
				}
				if _, err := os.Stat(entry.Output); !os.IsNotExist(err) {
					t.Fatalf("linked output cleanup=%v", err)
				}
			} else if _, err := os.Stat(filepath.Join(root, "build-command.json")); !os.IsNotExist(err) {
				t.Fatalf("invalid sum reached build: %v", err)
			}
			driverHash, moduleHash := sha256.Sum256(driver), sha256.Sum256(module)
			row := map[string]any{"name": name, "status": status, "stdout": stdout.String(), "stderr": stderr.String(), "attempted": diagnostic.attempted.String(), "write_calls": diagnostic.calls, "remove_calls": calls, "invalid_adapter": deterministicio.IsInvalidBuildAdapterConfiguration(observed), "invalid_review": target.IsInvalidCapabilityReview(observed), "unsupported": target.IsUnsupportedCapability(observed), "driver_sha256": hex.EncodeToString(driverHash[:]), "fixture_mod_sha256": hex.EncodeToString(moduleHash[:])}
			if name == "invalid-sum" {
				bytes, err := os.ReadFile(filepath.Join(working, "go.sum"))
				if err != nil {
					t.Fatal(err)
				}
				hash := sha256.Sum256(bytes)
				row["fixture_sum_sha256"] = hex.EncodeToString(hash[:])
			}
			encoded, err := json.Marshal(row)
			if err != nil {
				t.Fatal(err)
			}
			t.Logf("SOURCE_CLASSIFICATION %s", encoded)
			encoded, err = json.Marshal(map[string]any{"toolchain_root": root, "build_key": strings.Repeat("7", 64), "command": command, "interceptions": len(command.Commands)})
			if err != nil {
				t.Fatal(err)
			}
			t.Logf("SOURCE_COMMAND_BOUNDARY %s", encoded)
		})
	}
}
