package wasi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomad3/internal/backends/target"
	"go.temporal.io/server/tools/gomad3/internal/backends/wasm/toolchain"
)

type terminalExitHelperBuild struct {
	Crate, Helper, HelperSHA256 string
	RustcPath                   string
	Inputs                      map[string]string
	Command, Environment        []string
	Features                    []string
	RustcVersion                string
	CargoVersion                string
}

func TestTerminalExitCompilerEnvironmentControls(t *testing.T) {
	for _, test := range []struct {
		name        string
		environment []string
		want        bool
	}{
		{"pinned", []string{"RUSTC=/pinned/rustc", "RUSTC_WRAPPER=", "RUSTC_WORKSPACE_WRAPPER="}, true},
		{"missing-compiler", []string{"RUSTC_WRAPPER=", "RUSTC_WORKSPACE_WRAPPER="}, false},
		{"wrong-compiler", []string{"RUSTC=/other/rustc", "RUSTC_WRAPPER=", "RUSTC_WORKSPACE_WRAPPER="}, false},
		{"conflicting-compiler", []string{"RUSTC=/pinned/rustc", "RUSTC=/other/rustc", "RUSTC_WRAPPER=", "RUSTC_WORKSPACE_WRAPPER="}, false},
		{"missing-wrapper-override", []string{"RUSTC=/pinned/rustc", "RUSTC_WORKSPACE_WRAPPER="}, false},
		{"compiler-wrapper", []string{"RUSTC=/pinned/rustc", "RUSTC_WRAPPER=/wrapper", "RUSTC_WORKSPACE_WRAPPER="}, false},
		{"workspace-wrapper", []string{"RUSTC=/pinned/rustc", "RUSTC_WRAPPER=", "RUSTC_WORKSPACE_WRAPPER=/wrapper"}, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			build := terminalExitHelperBuild{RustcPath: "/pinned/rustc", Environment: test.environment, Inputs: map[string]string{"/pinned/rustc": strings.Repeat("a", 64)}}
			if got := terminalExitCompilerEnvironment(build); got != test.want {
				t.Fatalf("compiler environment admitted=%v, want %v", got, test.want)
			}
		})
	}
}

func terminalExitCompilerEnvironment(build terminalExitHelperBuild) bool {
	if !filepath.IsAbs(build.RustcPath) || len(build.Inputs[build.RustcPath]) != 64 {
		return false
	}
	for _, setting := range []struct{ name, value string }{{"RUSTC", build.RustcPath}, {"RUSTC_WRAPPER", ""}, {"RUSTC_WORKSPACE_WRAPPER", ""}} {
		count := 0
		for _, entry := range build.Environment {
			name, value, _ := strings.Cut(entry, "=")
			if name == setting.name {
				if value != setting.value {
					return false
				}
				count++
			}
		}
		if count != 1 {
			return false
		}
	}
	return true
}

func TestBuildTerminalExitReleaseHelper(t *testing.T) {
	root := os.Getenv("GOMAD_WASM_TERMINAL_EXIT_HELPER_BUILD_ROOT")
	if root == "" {
		t.Skip("set GOMAD_WASM_TERMINAL_EXIT_HELPER_BUILD_ROOT to build and retain the isolated release helper")
	}
	if !filepath.IsAbs(root) {
		t.Fatal("helper build evidence root must be absolute")
	}
	evidence, err := diagnosticRunRoot(root, "release-helper-")
	if err != nil {
		t.Fatal(err)
	}
	crate, err := filepath.Abs("../wasmhost")
	if err != nil {
		t.Fatal(err)
	}
	inputs := terminalExitRustInputs(t, crate)
	engine, err := os.ReadFile(filepath.Join(crate, "src/engine.rs"))
	if err != nil || bytes.Contains(engine, []byte("std::mem::forget")) || !bytes.Contains(engine, []byte("native_unwind_info(false)")) {
		t.Fatal("release helper must use the no-unwind source candidate without the leak", err)
	}
	var environment []string
	for _, entry := range os.Environ() {
		name, _, _ := strings.Cut(entry, "=")
		if slices.Contains([]string{"PATH", "HOME", "CARGO_HOME", "RUSTUP_HOME", "TMPDIR"}, name) {
			environment = append(environment, entry)
		}
	}
	environment = append(environment, "CARGO_TARGET_DIR="+filepath.Join(evidence, "target"))
	rustcPath := strings.TrimSpace(string(temporalCommand(t, evidence, "rustc-path", crate, environment, []string{"rustup", "which", "--toolchain", "1.94.1", "rustc"}).Stdout.RawBytes))
	cargoPath := strings.TrimSpace(string(temporalCommand(t, evidence, "cargo-path", crate, environment, []string{"rustup", "which", "--toolchain", "1.94.1", "cargo"}).Stdout.RawBytes))
	inputs[rustcPath], inputs[cargoPath] = fileHash(t, rustcPath), fileHash(t, cargoPath)
	rustcVersion := temporalCommand(t, evidence, "rustc-version", crate, environment, []string{rustcPath, "-vV"})
	cargoVersion := temporalCommand(t, evidence, "cargo-version", crate, environment, []string{cargoPath, "--version"})
	if !strings.HasPrefix(string(rustcVersion.Stdout.RawBytes), "rustc 1.94.1 ") || !strings.HasPrefix(string(cargoVersion.Stdout.RawBytes), "cargo 1.94.1 ") {
		t.Fatal("release helper requires pinned Rust/Cargo 1.94.1")
	}
	environment = append(environment, "RUSTC="+rustcPath, "RUSTC_WRAPPER=", "RUSTC_WORKSPACE_WRAPPER=")
	command := []string{cargoPath, "build", "--manifest-path", filepath.Join(crate, "Cargo.toml"), "--release", "--locked", "--no-default-features", "--message-format=json"}
	build := temporalCommand(t, evidence, "cargo-build", crate, environment, command)
	helper, err := terminalExitCargoHelper(build.Stdout.RawBytes)
	if err != nil {
		t.Fatal(err)
	}
	if helper != filepath.Join(evidence, "target/release/gomad3-wasmhost") {
		t.Fatal("Cargo did not produce the fresh isolated release executable")
	}
	current := terminalExitRustInputs(t, crate)
	current[rustcPath], current[cargoPath] = fileHash(t, rustcPath), fileHash(t, cargoPath)
	if !maps.Equal(inputs, current) {
		t.Fatal("Rust semantic inputs changed during build")
	}
	for _, name := range []string{"cargo-build.stdout", "cargo-build.json", "rustc-version.stdout", "cargo-version.stdout"} {
		path := filepath.Join(evidence, name)
		inputs[path] = fileHash(t, path)
	}
	receipt := filepath.Join(evidence, "helper-build.json")
	temporalJSON(t, receipt, terminalExitHelperBuild{Crate: crate, Helper: helper, HelperSHA256: fileHash(t, helper), RustcPath: rustcPath, Inputs: inputs, Command: command, Environment: environment, Features: []string{}, RustcVersion: string(rustcVersion.Stdout.RawBytes), CargoVersion: string(cargoVersion.Stdout.RawBytes)})
	t.Logf("release helper=%s receipt=%s sha256=%s", helper, receipt, fileHash(t, helper))
}

func terminalExitCargoHelper(data []byte) (string, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	helper := ""
	finished := false
	for {
		var frame struct {
			Reason     string
			Executable string
			Features   *[]string
			Success    bool
			Target     struct{ Name string }
			Profile    struct {
				OptLevel        string `json:"opt_level"`
				DebugAssertions bool   `json:"debug_assertions"`
			}
		}
		if err := decoder.Decode(&frame); err == io.EOF {
			break
		} else if err != nil {
			return "", err
		}
		if frame.Reason == "compiler-artifact" && frame.Target.Name == "gomad3-wasmhost" && frame.Executable != "" {
			if helper != "" || frame.Features == nil || len(*frame.Features) != 0 || frame.Profile.OptLevel != "3" || frame.Profile.DebugAssertions {
				return "", fmt.Errorf("duplicate or diagnostic Cargo helper artifact")
			}
			helper = frame.Executable
		}
		if frame.Reason == "build-finished" {
			finished = frame.Success
		}
	}
	if helper == "" || !finished {
		return "", fmt.Errorf("successful release helper artifact absent from Cargo output")
	}
	return helper, nil
}

func terminalExitRustInputs(t *testing.T, crate string) map[string]string {
	t.Helper()
	inputs := map[string]string{}
	if err := filepath.WalkDir(filepath.Join(crate, "src"), func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() {
			inputs[path] = fileHash(t, path)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"Cargo.toml", "Cargo.lock", "rust-toolchain.toml", "../Makefile"} {
		path := filepath.Clean(filepath.Join(crate, name))
		inputs[path] = fileHash(t, path)
	}
	buildScript := filepath.Join(crate, "build.rs")
	if _, err := os.Stat(buildScript); err == nil {
		inputs[buildScript] = fileHash(t, buildScript)
	} else if !os.IsNotExist(err) {
		t.Fatal(err)
	}
	cargoHome := os.Getenv("CARGO_HOME")
	if cargoHome == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			t.Fatal(err)
		}
		cargoHome = filepath.Join(home, ".cargo")
	}
	for _, name := range []string{"config", "config.toml"} {
		path := filepath.Join(cargoHome, name)
		if _, err := os.Stat(path); err == nil {
			inputs[path] = fileHash(t, path)
		} else if !os.IsNotExist(err) {
			t.Fatal(err)
		}
	}
	for directory := crate; ; directory = filepath.Dir(directory) {
		for _, name := range []string{".cargo/config", ".cargo/config.toml"} {
			path := filepath.Join(directory, name)
			if _, err := os.Stat(path); err == nil {
				inputs[path] = fileHash(t, path)
			} else if !os.IsNotExist(err) {
				t.Fatal(err)
			}
		}
		if filepath.Dir(directory) == directory {
			break
		}
	}
	return inputs
}

func TestTerminalExitCargoArtifactControls(t *testing.T) {
	for _, test := range []struct {
		name, output string
		want         bool
	}{
		{"release", `{"reason":"compiler-artifact","target":{"name":"gomad3-wasmhost"},"profile":{"opt_level":"3","debug_assertions":false},"executable":"/release/helper","features":[]} {"reason":"build-finished","success":true}`, true},
		{"debug-build", `{"reason":"compiler-artifact","target":{"name":"gomad3-wasmhost"},"profile":{"opt_level":"0","debug_assertions":true},"executable":"/release/helper","features":[]} {"reason":"build-finished","success":true}`, false},
		{"diagnostic-feature", `{"reason":"compiler-artifact","target":{"name":"gomad3-wasmhost"},"profile":{"opt_level":"3","debug_assertions":false},"executable":"/release/helper","features":["diagnostic-profile"]} {"reason":"build-finished","success":true}`, false},
		{"missing-features", `{"reason":"compiler-artifact","target":{"name":"gomad3-wasmhost"},"profile":{"opt_level":"3","debug_assertions":false},"executable":"/release/helper"} {"reason":"build-finished","success":true}`, false},
		{"failed-build", `{"reason":"compiler-artifact","target":{"name":"gomad3-wasmhost"},"profile":{"opt_level":"3","debug_assertions":false},"executable":"/release/helper","features":[]} {"reason":"build-finished","success":false}`, false},
		{"missing-finish", `{"reason":"compiler-artifact","target":{"name":"gomad3-wasmhost"},"profile":{"opt_level":"3","debug_assertions":false},"executable":"/release/helper","features":[]}`, false},
		{"missing-helper", `{"reason":"build-finished","success":true}`, false},
		{"malformed", `{`, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			helper, err := terminalExitCargoHelper([]byte(test.output))
			if (err == nil) != test.want || test.want && helper != "/release/helper" {
				t.Fatalf("Cargo artifact = %q, %v", helper, err)
			}
		})
	}
}

func terminalExitHelperInputs(t *testing.T, receipt string, request Request, crate string) map[string]string {
	t.Helper()
	data, err := os.ReadFile(receipt)
	if err != nil {
		t.Fatal(err)
	}
	var build terminalExitHelperBuild
	if err := json.Unmarshal(data, &build); err != nil {
		t.Fatal(err)
	}
	if build.Crate != crate || build.Helper != request.HelperPath || build.HelperSHA256 != request.HelperSHA256 || build.Features == nil || len(build.Features) != 0 || !slices.Contains(build.Command, "--release") || !slices.Contains(build.Command, "--locked") || !slices.Contains(build.Command, "--no-default-features") || !strings.HasPrefix(build.RustcVersion, "rustc 1.94.1 ") || !strings.HasPrefix(build.CargoVersion, "cargo 1.94.1 ") {
		t.Fatal("helper receipt does not bind the pinned release build and measured executable")
	}
	if !terminalExitCompilerEnvironment(build) {
		t.Fatal("Cargo must use the retained pinned rustc executable with compiler wrappers disabled")
	}
	current := terminalExitRustInputs(t, crate)
	for path := range build.Inputs {
		if _, found := current[path]; !found {
			current[path] = fileHash(t, path)
		}
	}
	if !maps.Equal(current, build.Inputs) {
		t.Fatal("helper Rust semantic sources, Cargo inputs or toolchain changed after build")
	}
	cargoOutput, err := os.ReadFile(filepath.Join(filepath.Dir(receipt), "cargo-build.stdout"))
	if err != nil {
		t.Fatal(err)
	}
	if helper, err := terminalExitCargoHelper(cargoOutput); err != nil || helper != request.HelperPath {
		t.Fatal("Cargo artifact does not bind the measured release helper", err)
	}
	inputs := maps.Clone(build.Inputs)
	inputs[receipt], inputs[request.HelperPath] = fileHash(t, receipt), request.HelperSHA256
	temporalVerifyInputs(t, inputs)
	return inputs
}

func terminalExitModuleInputs(t *testing.T, receipt string, request Request) map[string]string {
	t.Helper()
	data, err := os.ReadFile(receipt)
	if err != nil {
		t.Fatal(err)
	}
	var producer struct {
		Spec     target.Spec
		Prepared target.Prepared
		Options  struct{ CompilerPath string }
	}
	if err := json.Unmarshal(data, &producer); err != nil {
		t.Fatal(err)
	}
	prepared := producer.Prepared
	if producer.Spec.Source != "./tests" || prepared.Source != "./tests" || prepared.Kind != target.KindGoTest || prepared.Path != request.ModulePath || prepared.SHA256 != "sha256:"+request.ModuleSHA256 || !slices.Equal(prepared.BuildTags, []string{"disable_grpc_modules", "sqlite3_dotlk", "test_dep"}) {
		t.Fatal("producer must bind actual ./tests, unchanged tags, and the measured module")
	}
	if err := prepared.ValidateBackendPayloads(); err != nil {
		t.Fatal(err)
	}
	var provenance struct {
		Schema, Profile, Source, Kind, ModuleSHA256, GoVersion, RuntimeSHA256, RuntimeImplementationSHA256, ModelSHA256, CompilerSHA256 string
		Sources                                                                                                                         []struct{ Path, SHA256 string }
	}
	if len(prepared.BackendPayloads) != 1 {
		t.Fatal("module producer has no unique retained WASM provenance")
	}
	if err := json.Unmarshal(prepared.BackendPayloads[0].Data, &provenance); err != nil {
		t.Fatal(err)
	}
	if provenance.Schema != "gomad-wasm.provenance/v1" || provenance.Profile != CooperativeProfile || provenance.Source != "./tests" || provenance.Kind != "go-test" || provenance.ModuleSHA256 != prepared.SHA256 || provenance.GoVersion != "go1.27.1" || provenance.RuntimeSHA256 != "sha256:"+request.Runtime.Choice.ExecutionIdentity.ToolchainBuildKey || provenance.RuntimeImplementationSHA256 != toolchain.ImplementationSHA256() || provenance.ModelSHA256 != ImplementationSHA256() || provenance.CompilerSHA256 != "sha256:"+fileHash(t, producer.Options.CompilerPath) || len(provenance.Sources) == 0 {
		t.Fatal("module provenance is stale or lacks current compiler/runtime/model/source bindings")
	}
	inputs := map[string]string{receipt: fileHash(t, receipt), request.ModulePath: request.ModuleSHA256}
	for _, source := range provenance.Sources {
		inputs[source.Path] = strings.TrimPrefix(source.SHA256, "sha256:")
	}
	temporalVerifyInputs(t, inputs)
	return inputs
}

func mustTerminalExitRead(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}
