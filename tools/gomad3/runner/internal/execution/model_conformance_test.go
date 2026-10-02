package execution_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"go.temporal.io/server/tools/gomad3/deterministicio"
	"go.temporal.io/server/tools/gomad3/runner/internal/execution"
	"go.temporal.io/server/tools/gomad3/target"
	gomadversion "go.temporal.io/server/tools/gomad3/toolchain/version"
)

type modelOperationResult struct {
	Operation string `json:"operation"`
	Result    string `json:"result"`
	Error     string `json:"error"`
}

type modelDifference struct {
	Name      string
	Reason    string
	Normalize func(modelOperationResult) (modelOperationResult, error)
}

// Directory allocation sizes belong to the host filesystem, not file contents.
// All other metadata, including regular-file sizes, remains comparable.
var modelDeclaredDifferences = map[string][]modelDifference{
	"darwin/arm64": {{"directory-size", "Native directories occupy filesystem allocation bytes; Gomad directories have no data bytes.", normalizeModelDirectorySize}},
	"linux/amd64":  {{"directory-size", "Native directories occupy filesystem allocation bytes; Gomad directories have no data bytes.", normalizeModelDirectorySize}},
}

func normalizeModelDirectorySize(row modelOperationResult) (modelOperationResult, error) {
	if row.Operation != "stat-dir workspace/dir" || row.Error != "ok" {
		return row, nil
	}
	var info struct {
		Name      *string `json:"name"`
		Directory *bool   `json:"directory"`
		Size      *int64  `json:"size"`
		Mode      *uint32 `json:"mode"`
	}
	decoder := json.NewDecoder(strings.NewReader(row.Result))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&info); err != nil {
		return row, err
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return row, fmt.Errorf("trailing directory metadata")
	}
	if info.Name == nil || info.Directory == nil || info.Size == nil || info.Mode == nil || !*info.Directory || *info.Name != "dir" || *info.Size < 0 {
		return row, fmt.Errorf("invalid directory metadata: %s", row.Result)
	}
	*info.Size = 0
	data, err := json.Marshal(info)
	row.Result = string(data)
	return row, err
}

func compareModelLogs(platform string, model, native []modelOperationResult) error {
	differences, ok := modelDeclaredDifferences[platform]
	if !ok {
		return fmt.Errorf("undeclared platform %s", platform)
	}
	if len(model) != len(native) {
		return fmt.Errorf("operation count: model=%d native=%d", len(model), len(native))
	}
	for index := range model {
		left, right := model[index], native[index]
		for _, difference := range differences {
			var err error
			left, err = difference.Normalize(left)
			if err != nil {
				return fmt.Errorf("model operation %d: %w", index+1, err)
			}
			right, err = difference.Normalize(right)
			if err != nil {
				return fmt.Errorf("native operation %d: %w", index+1, err)
			}
		}
		if left != right {
			return fmt.Errorf("operation %d: model=%+v native=%+v", index+1, model[index], native[index])
		}
	}
	return nil
}

// Prefix outcomes need not be monotonic: closing a handle can hide an earlier
// difference. Re-execute every prefix in order instead of binary searching.
func shortestModelDivergence(length int, compare func(int) error) (int, error) {
	for prefix := 1; prefix <= length; prefix++ {
		if err := compare(prefix); err != nil {
			return prefix, err
		}
	}
	return 0, nil
}

func TestModelConformanceFilesystem(t *testing.T) { runModelConformance(t, "model_fs") }
func TestModelConformanceTCP(t *testing.T)        { runModelConformance(t, "model_net") }

func runModelConformance(t *testing.T, fixture string) {
	t.Helper()
	platform := runtime.GOOS + "/" + runtime.GOARCH
	if _, ok := modelDeclaredDifferences[platform]; !ok {
		t.Skipf("model conformance unsupported on %s", platform)
	}
	toolchainRoot, err := filepath.Abs(filepath.Join("..", "..", "..", ".toolchain"))
	if err != nil {
		t.Fatal(err)
	}
	working, err := filepath.Abs(filepath.Join("..", "..", "..", "internal", "gomadtool", "conformance", "testdata"))
	if err != nil {
		t.Fatal(err)
	}
	prepared, err := target.Prepare(context.Background(), target.Spec{Kind: target.KindGoRun, Source: "./" + fixture, WorkingDir: working, PreparationRoot: t.TempDir(), ToolchainRoot: toolchainRoot})
	if err != nil {
		t.Fatal(err)
	}
	stock := modelStockGo(t, toolchainRoot)
	nativePath := filepath.Join(t.TempDir(), fixture)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	command := exec.CommandContext(ctx, stock, "build", "-tags=test_dep", "-o", nativePath, "./"+fixture)
	command.Dir = working
	command.Env = modelBuildEnvironment()
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("stock build: %v: %s", err, output)
	}
	profile := deterministicio.Default()
	const length = 64
	for _, seed := range []uint64{0, 1, 7, 42, 89} {
		t.Run(strconv.FormatUint(seed, 10), func(t *testing.T) {
			run := func(prefix int) ([]modelOperationResult, []modelOperationResult, error) {
				args := []string{strconv.FormatUint(seed, 10), strconv.Itoa(prefix)}
				frame, err := profile.BootstrapFrame(prepared, "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", seed)
				if err != nil {
					return nil, nil, err
				}
				result, err := execution.Run(context.Background(), execution.Spec{
					SupervisorCommand: []string{os.Args[0], "-test.run=TestEntropySupervisorHelper"}, BootstrapCommand: []string{os.Args[0], "-test.run=TestEntropyBootstrapHelper"},
					Command: prepared.Path, Args: args, Argv0: prepared.Argv[0], Dir: t.TempDir(), Env: []string{"GOMAD3_IO_PROFILE=" + profile.Name(), "GOMADSEED=" + args[0], "TZ=UTC"},
					ExecutionTimeout: 10 * time.Second, TerminateGrace: time.Second, OutputLimit: 1 << 20,
					World: execution.WorldCapability{RecordLimit: 1 << 20, TransitionLimit: 1 << 20, Seed: seed}, IO: &execution.IOCapability{Config: frame, Transcript: &execution.IOTranscriptCapability{Limit: 64 << 20}},
				})
				if err != nil || result.Termination != execution.TerminationExit || result.ExitCode != 0 {
					return nil, nil, fmt.Errorf("model run: %v termination=%s/%d stdout=%q stderr=%q", err, result.Termination, result.ExitCode, result.Stdout.Bytes, result.Stderr.Bytes)
				}
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()
				command := exec.CommandContext(ctx, nativePath, args...)
				command.Dir = t.TempDir()
				command.Env = []string{"TZ=UTC"}
				var stdout, stderr bytes.Buffer
				command.Stdout = &stdout
				command.Stderr = &stderr
				if err := command.Run(); err != nil {
					return nil, nil, fmt.Errorf("native run: %v stdout=%q stderr=%q", err, stdout.String(), stderr.String())
				}
				model, err := decodeModelLog(result.Stdout.Bytes, prefix)
				if err != nil {
					return nil, nil, err
				}
				native, err := decodeModelLog(stdout.Bytes(), prefix)
				if err != nil {
					return nil, nil, err
				}
				return model, native, nil
			}
			model, native, err := run(length)
			if err == nil {
				for name, rows := range map[string][]modelOperationResult{"model": model, "native": native} {
					data, err := json.Marshal(rows)
					if err != nil {
						t.Fatal(err)
					}
					t.Logf("fixture=%s seed=%d operations=%d %s log=%s", fixture, seed, length, name, data)
				}
			}
			if err == nil {
				err = compareModelLogs(platform, model, native)
			}
			if err != nil {
				original := err
				prefix, divergence := shortestModelDivergence(length, func(prefix int) error {
					model, native, err := run(prefix)
					if err != nil {
						return err
					}
					return compareModelLogs(platform, model, native)
				})
				if prefix == 0 {
					t.Fatalf("fixture=%s seed=%d length=%d non-reproducing divergence: %v", fixture, seed, length, original)
				}
				t.Fatalf("fixture=%s seed=%d shortest diverging prefix=%d; reproduce argv=%d %d: %v", fixture, seed, prefix, seed, prefix, divergence)
			}
			for _, prefix := range []int{1, 3, 4, 14, 15, 16, 30, 31, 51, 52, 54, 55, 57, 59, 60, 63} {
				prefixModel, prefixNative, err := run(prefix)
				if err != nil {
					t.Fatalf("fixture=%s seed=%d prefix=%d: %v", fixture, seed, prefix, err)
				}
				for _, pair := range [][2][]modelOperationResult{{model[:prefix], prefixModel}, {native[:prefix], prefixNative}} {
					if err := compareModelLogs(platform, pair[0], pair[1]); err != nil {
						t.Fatalf("fixture=%s seed=%d generation changed prefix=%d: %v", fixture, seed, prefix, err)
					}
				}
			}
		})
	}
}

func decodeModelLog(data []byte, length int) ([]modelOperationResult, error) {
	lines := bytes.Split(bytes.TrimSuffix(data, []byte("\n")), []byte("\n"))
	if len(lines) != length {
		return nil, fmt.Errorf("log has %d operations, want %d: %q", len(lines), length, data)
	}
	rows := make([]modelOperationResult, length)
	for index, line := range lines {
		decoder := json.NewDecoder(bytes.NewReader(line))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&rows[index]); err != nil {
			return nil, err
		}
		if err := decoder.Decode(new(any)); err != io.EOF {
			return nil, fmt.Errorf("trailing operation data at %d", index+1)
		}
		if rows[index].Operation == "" || rows[index].Error == "" {
			return nil, fmt.Errorf("incomplete operation %d", index+1)
		}
	}
	return rows, nil
}

func modelBuildEnvironment() []string {
	var env []string
	for _, entry := range os.Environ() {
		key, _, _ := strings.Cut(entry, "=")
		switch key {
		case "GOROOT", "GOMADSEED", "GOMAD3_CHILD_SEED", "GOFLAGS", "GOTOOLCHAIN", "GOWORK", "CGO_ENABLED", "GOENV":
			continue
		}
		env = append(env, entry)
	}
	return append(env, "GOFLAGS=", "GOTOOLCHAIN=local", "GOWORK=off", "CGO_ENABLED=0", "GOENV=off")
}

func modelStockGo(t *testing.T, toolchainRoot string) string {
	t.Helper()
	stock := os.Getenv("GOMAD3_STOCK_GO")
	if stock == "" {
		command := exec.Command("go", "env", "GOROOT")
		command.Env = modelBuildEnvironment()
		for index, entry := range command.Env {
			if strings.HasPrefix(entry, "GOTOOLCHAIN=") {
				command.Env[index] = "GOTOOLCHAIN=" + gomadversion.GoVersion
			}
		}
		output, err := command.Output()
		if err != nil {
			t.Fatalf("resolve stock Go (or set GOMAD3_STOCK_GO): %v", err)
		}
		stock = filepath.Join(strings.TrimSpace(string(output)), "bin", "go")
	}
	command := exec.Command(stock, "version")
	command.Env = modelBuildEnvironment()
	output, err := command.CombinedOutput()
	if err != nil || !strings.HasPrefix(string(output), "go version "+gomadversion.GoVersion+" ") {
		t.Fatalf("stock Go must be %s: %v %s", gomadversion.GoVersion, err, output)
	}
	command = exec.Command(stock, "env", "GOROOT")
	command.Env = modelBuildEnvironment()
	output, err = command.Output()
	if err != nil {
		t.Fatal(err)
	}
	stockRoot, err := filepath.EvalSymlinks(strings.TrimSpace(string(output)))
	if err != nil {
		t.Fatal(err)
	}
	customRoot, err := filepath.EvalSymlinks(toolchainRoot)
	if err != nil {
		t.Fatal(err)
	}
	if stockRoot == customRoot || strings.HasPrefix(stockRoot, customRoot+string(filepath.Separator)) {
		t.Fatal("stock Go resolves to Gomad toolchain")
	}
	return stock
}
