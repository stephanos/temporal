// Package measurement names the WASM backend's measurement producers and
// their inputs (fn-187.2). cmd/wasmmeasure turns the inputs into flags; the
// production executor runs the producer; retained artifacts record the
// wasmmeasure command that reproduces them.
package measurement

import (
	"path/filepath"
	"sort"
	"strings"
)

// Exit statuses of a producer run.
const (
	ExitFailed  = 1
	ExitRefused = 2
)

// Input is one producer input: the GOMAD_* variable the producer reads and
// whether wasmmeasure refuses to start without its flag.
type Input struct {
	Variable string
	Required bool
}

// Names lists the producers in name order.
func Names() []string {
	names := make([]string, 0, len(producerInputs))
	for name := range producerInputs {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// Inputs lists a producer's inputs, required ones first, or nil for an
// unknown producer. A producer's gate inputs (its former skip condition) and
// its helper and compiler identities are required; the producer itself still
// refuses any other missing receipt or admission, as it did under go test.
func Inputs(name string) []Input { return producerInputs[name] }

// flags overrides the derived flag name of an input.
var flags = map[string]string{"GOMAD_WASM_DIAGNOSTIC_HELPER_FEATURES": "features"}

// Flag is the wasmmeasure flag for an input variable: GOMAD_WASM_PROFILE_ROOT
// is --profile-root and GOMAD_STOCK_GO is --stock-go.
func Flag(variable string) string {
	if flag, ok := flags[variable]; ok {
		return flag
	}
	name := strings.TrimPrefix(strings.TrimPrefix(variable, "GOMAD_WASM_"), "GOMAD_")
	return strings.ReplaceAll(strings.ToLower(name), "_", "-")
}

// Command is the wasmmeasure invocation, run from tools/gomad, that reproduces
// a producer with the given inputs: each GOMAD_* input becomes its flag.
// goCommand is the go executable named in the command.
func Command(goCommand, repo, name string, inputs map[string]string) []string {
	command := []string{goCommand, "run", "./internal/backends/wasm/cmd/wasmmeasure", name, "--repo", repo}
	keys := make([]string, 0, len(inputs))
	for key := range inputs {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		if strings.HasPrefix(key, "GOMAD_") && inputs[key] != "" {
			command = append(command, "--"+Flag(key), inputs[key])
		}
	}
	return command
}

// Environment keeps the non-GOMAD entries of a rerun environment; the GOMAD_*
// inputs travel as flags.
func Environment(inputs map[string]string) map[string]string {
	environment := map[string]string{}
	for key, value := range inputs {
		if !strings.HasPrefix(key, "GOMAD_") {
			environment[key] = value
		}
	}
	return environment
}

// RerunRecord is a retained artifact's rerun record: the working directory,
// the toolchain environment and the wasmmeasure command (a shell string when
// the record always carried one, otherwise argv).
func RerunRecord(goCommand, repo, name string, inputs map[string]string, shell bool) map[string]any {
	environment := Environment(inputs)
	for key, value := range map[string]string{"GOENV": "off", "GOTOOLCHAIN": "local", "GOWORK": "off", "GOROOT": ""} {
		if _, set := environment[key]; !set {
			environment[key] = value
		}
	}
	command := Command(goCommand, repo, name, inputs)
	record := map[string]any{"working_directory": filepath.Join(repo, "tools", "gomad"), "environment": environment, "command": command}
	if shell {
		record["command"] = "env -u GOROOT GOENV=off GOTOOLCHAIN=local GOWORK=off " + Shell(command)
	}
	return record
}

// Shell renders a command for the string-valued rerun records.
func Shell(command []string) string {
	quoted := make([]string, len(command))
	for index, word := range command {
		if word != "" && strings.IndexFunc(word, func(r rune) bool {
			return r != '/' && r != '-' && r != '_' && r != '.' && r != '=' && r != ':' && r != ',' &&
				(r < '0' || r > '9') && (r < 'a' || r > 'z') && (r < 'A' || r > 'Z')
		}) < 0 {
			quoted[index] = word
		} else {
			quoted[index] = "'" + strings.ReplaceAll(word, "'", `'\''`) + "'"
		}
	}
	return strings.Join(quoted, " ")
}

var producerInputs = map[string][]Input{
	"stress": {
		{"GOMAD_WASM_STRESS_ROOT", true},
		{"GOMAD_WASM_TEST_HELPER", true},
		{"GOMAD_STOCK_GO", true},
		{"GOMAD_WASM_STRESS_HOST_DESCRIPTION", true},
		{"GOMAD_WASM_STRESS_REPETITIONS", false},
		{"GOMAD_WASM_STRESS_PARALLEL", false},
		{"GOMAD_WASM_STRESS_CPU_WORKERS", false},
		{"GOMAD_WASM_STRESS_ONLY", false},
		{"GOMAD_WASM_STRESS_SEED", false},
		{"GOMAD_WASM_STRESS_AXIS", false},
		{"GOMAD_WASM_TOOLCHAIN_CACHE", false},
	},
	"compiled-identity-owners": {
		{"GOMAD_WASM_DIAGNOSTIC_IDENTITY_ROOT", true},
		{"GOMAD_WASM_TOOLCHAIN_CACHE", false},
		{"GOMAD_WASM_PROTOTYPE_PARALLEL", false},
		{"GOMAD_WASM_PROTOTYPE_HEAVY", false},
	},
	"diagnostic-helper-build": {
		{"GOMAD_WASM_DIAGNOSTIC_HELPER_BUILD_ROOT", true},
		{"GOMAD_WASM_DIAGNOSTIC_HELPER_FEATURES", true},
	},
	"diagnostic-cluster-baseline": {
		{"GOMAD_STOCK_GO", true},
		{"GOMAD_WASM_TEMPORAL_RECOVERY_ROOT", true},
		{"GOMAD_WASM_TEST_HELPER", true},
		{"GOMAD_WASM_DIAGNOSTIC_HELPER_ADMISSION", false},
		{"GOMAD_WASM_TEMPORAL_RECOVERY_COMPILE_CACHE", false},
		{"GOMAD_WASM_TEMPORAL_RECOVERY_DIAGNOSTIC_BUDGET", false},
		{"GOMAD_WASM_TEMPORAL_RECOVERY_EXPECT_MODULE_SHA256", false},
		{"GOMAD_WASM_TOOLCHAIN_CACHE", false},
		{"GOMAD_WASM_PROTOTYPE_PARALLEL", false},
		{"GOMAD_WASM_PROTOTYPE_HEAVY", false},
	},
	"failing-suite-profile": {
		{"GOMAD_WASM_DIAGNOSTIC_PROFILE_ROOT", true},
		{"GOMAD_WASM_TEST_HELPER", true},
		{"GOMAD_WASM_MATCHED_CACHE", false},
		{"GOMAD_WASM_MATCHED_COLD_MEMORY_GUARD", false},
		{"GOMAD_WASM_MATCHED_MODULE", false},
		{"GOMAD_WASM_MATCHED_TOOLCHAIN_ID", false},
		{"GOMAD_WASM_PROFILE_REFERENCE_CACHE", false},
		{"GOMAD_WASM_PROFILE_REFERENCE_HELPER", false},
		{"GOMAD_WASM_TOOLCHAIN_CACHE", false},
		{"GOMAD_WASM_PROTOTYPE_PARALLEL", false},
		{"GOMAD_WASM_PROTOTYPE_HEAVY", false},
	},
	"guest-profile": {
		{"GOMAD_STOCK_GO", true},
		{"GOMAD_WASM_GUEST_PROFILE_ROOT", true},
		{"GOMAD_WASM_TEST_HELPER", true},
		{"GOMAD_WASM_GUEST_HELPER_ADMISSION", false},
		{"GOMAD_WASM_MATCHED_COLD_MEMORY_GUARD", false},
		{"GOMAD_WASM_TOOLCHAIN_CACHE", false},
		{"GOMAD_WASM_PROTOTYPE_PARALLEL", false},
		{"GOMAD_WASM_PROTOTYPE_HEAVY", false},
	},
	"guest-variants": {
		{"GOMAD_STOCK_GO", true},
		{"GOMAD_WASM_GUEST_PROFILE_ROOT", true},
		{"GOMAD_WASM_TEST_HELPER", true},
		{"GOMAD_WASM_DIAGNOSTIC_HELPER_ADMISSION", false},
		{"GOMAD_WASM_FUEL_OFF_HELPER", false},
		{"GOMAD_WASM_FUEL_OFF_HELPER_ADMISSION", false},
		{"GOMAD_WASM_MATCHED_COLD_MEMORY_GUARD", false},
		{"GOMAD_WASM_TOOLCHAIN_CACHE", false},
		{"GOMAD_WASM_PROTOTYPE_PARALLEL", false},
		{"GOMAD_WASM_PROTOTYPE_HEAVY", false},
	},
	"host-call-profile": {
		{"GOMAD_STOCK_GO", true},
		{"GOMAD_WASM_HOST_CALL_PROFILE_ROOT", true},
		{"GOMAD_WASM_TEST_HELPER", true},
		{"GOMAD_WASM_HOST_CALL_PROFILE_MODULE", false},
		{"GOMAD_WASM_HOST_CALL_PROFILE_REPETITIONS", false},
		{"GOMAD_WASM_MATCHED_COLD_MEMORY_GUARD", false},
		{"GOMAD_WASM_TOOLCHAIN_CACHE", false},
		{"GOMAD_WASM_PROTOTYPE_PARALLEL", false},
		{"GOMAD_WASM_PROTOTYPE_HEAVY", false},
	},
	"legacy-workflow-profile": {
		{"GOMAD_WASM_TEST_HELPER", true},
		{"GOMAD_WASM_WORKFLOW_PROFILE_ROOT", true},
		{"GOMAD_WASM_MATCHED_CACHE", false},
		{"GOMAD_WASM_MATCHED_MODULE", false},
		{"GOMAD_WASM_MATCHED_TOOLCHAIN_ID", false},
		{"GOMAD_WASM_TOOLCHAIN_CACHE", false},
		{"GOMAD_WASM_PROTOTYPE_PARALLEL", false},
		{"GOMAD_WASM_PROTOTYPE_HEAVY", false},
	},
	"matched-release-candidate": {
		{"GOMAD_STOCK_GO", true},
		{"GOMAD_WASM_CLUSTER_MEASUREMENT_ROOT", true},
		{"GOMAD_WASM_TEST_HELPER", true},
		{"GOMAD_WASM_CLUSTER_HELPER_ADMISSION", false},
		{"GOMAD_WASM_TEMPORAL_RECOVERY_DIAGNOSTIC_BUDGET", false},
		{"GOMAD_WASM_TEMPORAL_RECOVERY_EXPECT_MODULE_SHA256", false},
		{"GOMAD_WASM_TOOLCHAIN_CACHE", false},
		{"GOMAD_WASM_PROTOTYPE_PARALLEL", false},
		{"GOMAD_WASM_PROTOTYPE_HEAVY", false},
	},
	"matched-suite": {
		{"GOMAD_STOCK_GO", true},
		{"GOMAD_WASM_MATCHED_BASELINE_HELPER", true},
		{"GOMAD_WASM_MATCHED_MODULE", true},
		{"GOMAD_WASM_TEST_HELPER", true},
		{"GOMAD_WASM_MATCHED_BASELINE_CACHE", false},
		{"GOMAD_WASM_MATCHED_CACHE", false},
		{"GOMAD_WASM_MATCHED_COLD_MEMORY_GUARD", false},
		{"GOMAD_WASM_MATCHED_SUITE", false},
		{"GOMAD_WASM_MATCHED_TOOLCHAIN_ID", false},
		{"GOMAD_WASM_PROTOTYPE_MIRROR", false},
		{"GOMAD_WASM_PROTOTYPE_ROOT", false},
		{"GOMAD_WASM_TOOLCHAIN_CACHE", false},
		{"GOMAD_WASM_PROTOTYPE_PARALLEL", false},
		{"GOMAD_WASM_PROTOTYPE_HEAVY", false},
	},
	"migration": {
		{"GOMAD_STOCK_GO", true},
		{"GOMAD_WASM_MIGRATION_BASELINE", true},
		{"GOMAD_WASM_PROTOTYPE_ROOT", true},
		{"GOMAD_WASM_TEST_HELPER", true},
		{"GOMAD_WASM_MIGRATION_CACHE", false},
		{"GOMAD_WASM_PROTOTYPE_MIRROR", false},
		{"GOMAD_WASM_TOOLCHAIN_CACHE", false},
		{"GOMAD_WASM_PROTOTYPE_PARALLEL", false},
		{"GOMAD_WASM_PROTOTYPE_HEAVY", false},
	},
	"namespace-activity": {
		{"GOMAD_WASM_NAMESPACE_MEASUREMENT_ROOT", true},
		{"GOMAD_WASM_NAMESPACE_BASELINE_CACHE", false},
		{"GOMAD_WASM_NAMESPACE_BASELINE_HELPER_RECEIPT", false},
		{"GOMAD_WASM_NAMESPACE_BASELINE_SOURCE", false},
		{"GOMAD_WASM_NAMESPACE_CACHE", false},
		{"GOMAD_WASM_NAMESPACE_HELPER_RECEIPT", false},
		{"GOMAD_WASM_NAMESPACE_MODULE", false},
		{"GOMAD_WASM_NAMESPACE_MODULE_RECEIPT", false},
		{"GOMAD_WASM_NAMESPACE_STOCK_GO", false},
		{"GOMAD_WASM_TOOLCHAIN_CACHE", false},
		{"GOMAD_WASM_PROTOTYPE_PARALLEL", false},
		{"GOMAD_WASM_PROTOTYPE_HEAVY", false},
	},
	"namespace-sample": {
		{"GOMAD_WASM_NAMESPACE_SAMPLE_ROOT", true},
		{"GOMAD_WASM_NAMESPACE_CACHE", false},
		{"GOMAD_WASM_NAMESPACE_COMMON", false},
		{"GOMAD_WASM_NAMESPACE_HELPER_RECEIPT", false},
		{"GOMAD_WASM_NAMESPACE_REPETITION", false},
		{"GOMAD_WASM_NAMESPACE_REPLAY_TAPE", false},
		{"GOMAD_WASM_NAMESPACE_ROLE", false},
		{"GOMAD_WASM_NAMESPACE_SAMPLE_INDEX", false},
		{"GOMAD_WASM_NAMESPACE_SEED", false},
		{"GOMAD_WASM_TOOLCHAIN_CACHE", false},
		{"GOMAD_WASM_PROTOTYPE_PARALLEL", false},
		{"GOMAD_WASM_PROTOTYPE_HEAVY", false},
	},
	"operation-retention": {
		{"GOMAD_WASM_OPERATION_COUNT_ROOT", true},
		{"GOMAD_WASM_TEST_HELPER", true},
		{"GOMAD_WASM_MATCHED_CACHE", false},
		{"GOMAD_WASM_MATCHED_COLD_MEMORY_GUARD", false},
		{"GOMAD_WASM_MATCHED_MODULE", false},
		{"GOMAD_WASM_MATCHED_TOOLCHAIN_ID", false},
		{"GOMAD_WASM_OPERATION_CONTROL_ROOT", false},
		{"GOMAD_WASM_TOOLCHAIN_CACHE", false},
		{"GOMAD_WASM_PROTOTYPE_PARALLEL", false},
		{"GOMAD_WASM_PROTOTYPE_HEAVY", false},
	},
	"runtime-cost-profile": {
		{"GOMAD_STOCK_GO", true},
		{"GOMAD_WASM_PROFILE_ROOT", true},
		{"GOMAD_WASM_TEST_HELPER", true},
		{"GOMAD_WASM_TOOLCHAIN_CACHE", false},
		{"GOMAD_WASM_PROTOTYPE_PARALLEL", false},
		{"GOMAD_WASM_PROTOTYPE_HEAVY", false},
	},
	"scratch-smoke-profile": {
		{"GOMAD_STOCK_GO", true},
		{"GOMAD_WASM_PROTOTYPE_ROOT", true},
		{"GOMAD_SCRATCH_HELPERS", false},
		{"GOMAD_SCRATCH_ONLY", false},
		{"GOMAD_SCRATCH_SKIP_PREPARE", false},
		{"GOMAD_SCRATCH_TRANSCRIPT", false},
		{"GOMAD_WASM_MATCHED_COLD_MEMORY_GUARD", false},
		{"GOMAD_WASM_TOOLCHAIN_CACHE", false},
		{"GOMAD_WASM_PROTOTYPE_PARALLEL", false},
		{"GOMAD_WASM_PROTOTYPE_HEAVY", false},
	},
	"terminal-exit": {
		{"GOMAD_STOCK_GO", true},
		{"GOMAD_WASM_TERMINAL_EXIT_ROOT", true},
		{"GOMAD_WASM_TEST_HELPER", true},
		{"GOMAD_WASM_TERMINAL_EXIT_HELPER_RECEIPT", false},
		{"GOMAD_WASM_TERMINAL_EXIT_MODULE", false},
		{"GOMAD_WASM_TERMINAL_EXIT_MODULE_RECEIPT", false},
		{"GOMAD_WASM_TOOLCHAIN_CACHE", false},
		{"GOMAD_WASM_PROTOTYPE_PARALLEL", false},
		{"GOMAD_WASM_PROTOTYPE_HEAVY", false},
	},
	"terminal-exit-helper-build": {
		{"GOMAD_WASM_TERMINAL_EXIT_HELPER_BUILD_ROOT", true},
		{"GOMAD_WASM_TOOLCHAIN_CACHE", false},
		{"GOMAD_WASM_PROTOTYPE_PARALLEL", false},
		{"GOMAD_WASM_PROTOTYPE_HEAVY", false},
	},
	"terminal-exit-helper-stage": {
		{"GOMAD_WASM_TERMINAL_EXIT_HELPER_STAGE_ROOT", true},
		{"GOMAD_WASM_TERMINAL_EXIT_HELPER_BUILD_RECEIPT", false},
		{"GOMAD_WASM_TOOLCHAIN_CACHE", false},
		{"GOMAD_WASM_PROTOTYPE_PARALLEL", false},
		{"GOMAD_WASM_PROTOTYPE_HEAVY", false},
	},
	"warm-preparation": {
		{"GOMAD_STOCK_GO", true},
		{"GOMAD_WASM_TEST_HELPER", true},
		{"GOMAD_WASM_WARM_PREPARATION_ROOT", true},
		{"GOMAD_WASM_WARM_BASELINE_RECEIPT", false},
		{"GOMAD_WASM_WARM_CACHE", false},
		{"GOMAD_WASM_WARM_HELPER_RECEIPT", false},
		{"GOMAD_WASM_WARM_MODULE", false},
		{"GOMAD_WASM_WARM_MODULE_RECEIPT", false},
		{"GOMAD_WASM_TOOLCHAIN_CACHE", false},
		{"GOMAD_WASM_PROTOTYPE_PARALLEL", false},
		{"GOMAD_WASM_PROTOTYPE_HEAVY", false},
	},
	"warm-preparation-helper-build": {
		{"GOMAD_WASM_WARM_HELPER_BUILD_ROOT", true},
		{"GOMAD_WASM_TOOLCHAIN_CACHE", false},
		{"GOMAD_WASM_PROTOTYPE_PARALLEL", false},
		{"GOMAD_WASM_PROTOTYPE_HEAVY", false},
	},
	"warm-preparation-prime": {
		{"GOMAD_STOCK_GO", true},
		{"GOMAD_WASM_TEST_HELPER", true},
		{"GOMAD_WASM_WARM_PRIME_ROOT", true},
		{"GOMAD_WASM_WARM_CACHE", false},
		{"GOMAD_WASM_WARM_HELPER_RECEIPT", false},
		{"GOMAD_WASM_WARM_MODULE", false},
		{"GOMAD_WASM_WARM_MODULE_RECEIPT", false},
		{"GOMAD_WASM_TOOLCHAIN_CACHE", false},
		{"GOMAD_WASM_PROTOTYPE_PARALLEL", false},
		{"GOMAD_WASM_PROTOTYPE_HEAVY", false},
	},
}
