package backend

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"go.temporal.io/server/tools/gomad/internal/backends/target"
	"go.temporal.io/server/tools/gomad/internal/backends/wasm/wasi"
)

func TestPrepareTemporalTerminalExitModule(t *testing.T) {
	root := os.Getenv("GOMAD_WASM_TERMINAL_EXIT_PREPARE_ROOT")
	if root == "" {
		t.Skip("set GOMAD_WASM_TERMINAL_EXIT_PREPARE_ROOT to prepare actual cooperative ./tests")
	}
	if !filepath.IsAbs(root) || !filepath.IsAbs(os.Getenv("GOMAD_WASM_TEST_HELPER")) {
		t.Fatal("explicit absolute preparation root and candidate release helper are required")
	}
	if err := os.MkdirAll(root, 0700); err != nil {
		t.Fatal(err)
	}
	root, err := os.MkdirTemp(root, "temporal-module-")
	if err != nil {
		t.Fatal(err)
	}
	repo, err := filepath.Abs("../../../../../..")
	if err != nil {
		t.Fatal(err)
	}
	provider := integrationProvider(t)
	options := provider.options
	options.CacheRoot = filepath.Join(root, "build-cache")
	options.RuntimeRoot = filepath.Join(repo, "tools/gomad/internal/model/runtimesource")
	options.MemoryBytes, options.Fuel = 4<<30, 1<<50
	options.Config.Profile, options.Config.Clock.ReadStepNanos = wasi.CooperativeProfile, 0
	var commands []CompilerCommandMetrics
	options.ObserveCompilerCommand = func(command CompilerCommandMetrics) { commands = append(commands, command) }
	provider, err = New(options)
	if err != nil {
		t.Fatal(err)
	}
	spec := target.Spec{Backend: Name, Kind: target.KindGoTest, Source: "./tests", WorkingDir: repo, PreparationRoot: filepath.Join(root, "prepared"), BuildTags: []string{"disable_grpc_modules", "test_dep", "sqlite3_dotlk"}}
	prepared, err := provider.Prepare(t.Context(), spec)
	if err != nil {
		t.Fatal(err)
	}
	if err := provider.ValidatePrepared(spec, prepared, nil); err != nil {
		t.Fatal(err)
	}
	data, err := json.MarshalIndent(struct {
		Spec             target.Spec
		Options          Options
		Prepared         target.Prepared
		CompilerCommands []CompilerCommandMetrics
	}{spec, options, prepared, commands}, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	receipt := filepath.Join(root, "temporal-producer.json")
	if err := os.WriteFile(receipt, append(data, '\n'), 0600); err != nil {
		t.Fatal(err)
	}
	t.Logf("actual ./tests producer=%s module=%s sha256=%s", receipt, prepared.Path, prepared.SHA256)
}
