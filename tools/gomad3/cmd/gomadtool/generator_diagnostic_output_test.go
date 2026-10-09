package main

import (
	"bytes"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"syscall"
	"testing"
)

type generatorDiagnosticOutput struct {
	writer   io.Writer
	attempts int
	contents []byte
}

func (output *generatorDiagnosticOutput) Write(contents []byte) (int, error) {
	output.attempts++
	output.contents = append(output.contents, contents...)
	return output.writer.Write(contents)
}

func TestRunGeneratorDiagnosticOutputPreservesStatus(t *testing.T) {
	for _, test := range []struct {
		name, input, contents, output, diagnostic string
		arguments                                 []string
		status                                    int
	}{
		{
			name: "missing manifest flags", output: "qualification/manifest.json",
			arguments: []string{"qualification-manifest-generate"}, status: 2,
			diagnostic: "qualification-manifest-generate requires --spec and --output\n",
		},
		{
			name: "manifest failure", input: "qualification/spec.json", contents: "{}\n", output: "qualification/manifest.json",
			arguments: []string{"qualification-manifest-generate", "--spec=qualification/spec.json"}, status: 1,
			diagnostic: "qualification manifest spec schema must be gomad3.qualification-set-generator/v1\n",
		},
		{
			name: "protocol failure", input: "deterministicio/schema/iowire.json", contents: "{} {}\n", output: "deterministicio/internal/wire/wire_generated.go",
			arguments: []string{"protocol-generate"}, status: 1,
			diagnostic: "I/O wire schema has trailing data\n",
		},
		{
			name: "version failure", input: "toolchain/version/version.json", contents: "{}\n", output: "version_generated.mk",
			arguments: []string{"version-generate"}, status: 1,
			diagnostic: "version descriptor schema version 0 is unsupported\n",
		},
		{
			name: "boundary failure", output: "compiler-overlay/overlay.json",
			arguments: []string{"boundary-generate"}, status: 1,
			diagnostic: "-goroot is required with -compiler-test-overlay\n",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			for _, existing := range []bool{false, true} {
				name := "absent output"
				if existing {
					name = "existing output"
				}
				t.Run(name, func(t *testing.T) {
					root := t.TempDir()
					maintainerWrite(t, root, "sentinels/keep", "unchanged publication sentinel\n")
					if test.input != "" {
						maintainerWrite(t, root, test.input, test.contents)
					}
					outputPath := filepath.Join(root, filepath.FromSlash(test.output))
					if existing {
						maintainerWrite(t, root, test.output, "existing generated output sentinel\n")
					} else if _, err := os.Lstat(outputPath); !errors.Is(err, os.ErrNotExist) {
						t.Fatalf("absent output stat = %v", err)
					}
					arguments := append(append([]string(nil), test.arguments...), "--root="+root)
					switch test.name {
					case "missing manifest flags", "manifest failure":
						arguments = append(arguments, "--output="+test.output)
					case "boundary failure":
						arguments = append(arguments, "--compiler-test-overlay="+outputPath)
					}
					for _, failed := range []bool{false, true} {
						name := "healthy stderr"
						if failed {
							name = "EBADF stderr"
						}
						t.Run(name, func(t *testing.T) {
							before := generatorDiagnosticSnapshot(t, root)
							var stdout, stderr bytes.Buffer
							output := &generatorDiagnosticOutput{writer: &stdout}
							diagnostics := &generatorDiagnosticOutput{writer: &stderr}
							var failedOutput *refreshOutput
							if failed {
								failedOutput = newRefreshOutput(t, 0)
								diagnostics.writer = failedOutput
							}
							status := run(arguments, output, diagnostics)
							if status != test.status || diagnostics.attempts != 1 || string(diagnostics.contents) != test.diagnostic {
								t.Fatalf("status=%d attempts=%d diagnostic=%q, want %d, 1, %q", status, diagnostics.attempts, diagnostics.contents, test.status, test.diagnostic)
							}
							if output.attempts != 0 || len(output.contents) != 0 || stdout.Len() != 0 {
								t.Fatalf("stdout attempts=%d attempted=%q written=%q", output.attempts, output.contents, stdout.String())
							}
							if failed {
								if !errors.Is(failedOutput.err, syscall.EBADF) || failedOutput.Len() != 0 || stderr.Len() != 0 {
									t.Fatalf("failed diagnostic error=%v written=%q healthy=%q", failedOutput.err, failedOutput.String(), stderr.String())
								}
								t.Logf("stderr Write returned %T: %v; errors.Is(EBADF)=true; attempted=%q", failedOutput.err, failedOutput.err, diagnostics.contents)
							} else if stderr.String() != test.diagnostic {
								t.Fatalf("written diagnostic=%q, want %q", stderr.String(), test.diagnostic)
							}
							if after := generatorDiagnosticSnapshot(t, root); !reflect.DeepEqual(before, after) {
								t.Fatalf("publication changed: before=%#v after=%#v", before, after)
							}
							if !existing {
								if _, err := os.Lstat(outputPath); !errors.Is(err, os.ErrNotExist) {
									t.Fatalf("absent output changed: stat=%v", err)
								}
							}
						})
					}
				})
			}
		})
	}
}

type generatorDiagnosticEntry struct {
	mode     fs.FileMode
	contents string
	children []string
}

func generatorDiagnosticSnapshot(t *testing.T, root string) map[string]generatorDiagnosticEntry {
	t.Helper()
	snapshot := map[string]generatorDiagnosticEntry{}
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, visitErr error) error {
		if visitErr != nil {
			return visitErr
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		value := generatorDiagnosticEntry{mode: info.Mode()}
		if entry.IsDir() {
			children, err := os.ReadDir(path)
			if err != nil {
				return err
			}
			for _, child := range children {
				value.children = append(value.children, child.Name())
			}
		} else {
			contents, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			value.contents = string(contents)
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		snapshot[filepath.ToSlash(relative)] = value
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return snapshot
}
