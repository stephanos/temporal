package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"go.temporal.io/server/tools/gomadv3/internal/buildkey"
	"go.temporal.io/server/tools/gomadv3/internal/commandrun"
	"go.temporal.io/server/tools/gomadv3/internal/patchset"
	"go.temporal.io/server/tools/gomadv3/internal/scriptpolicy"
	"go.temporal.io/server/tools/gomadv3/internal/testdriver"
	"go.temporal.io/server/tools/gomadv3/internal/testtier"
	"go.temporal.io/server/tools/gomadv3/internal/toolchainbuild"
)

const usage = "usage: hosttool build-key|checked-run|patch-materialize|patch-regenerate|patch-validate|script-validate|test|test-mode|toolchain-build [flags]"

const canonicalBuildPath = "/usr/bin:/bin:/usr/sbin:/sbin:/usr/xpg4/bin:/opt/freeware/bin:/usr/local/bin:/opt/homebrew/bin:/opt/local/bin"

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(arguments []string, stdout, stderr io.Writer) int {
	if len(arguments) == 0 {
		fmt.Fprintln(stderr, usage)
		return 2
	}
	switch arguments[0] {
	case "build-key":
		return runBuildKey(arguments[1:], stdout, stderr)
	case "checked-run":
		return runChecked(arguments[1:], stdout, stderr)
	case "patch-materialize":
		return runPatchMaterialize(arguments[1:], stdout, stderr)
	case "patch-regenerate":
		return runPatchRegenerate(arguments[1:], stdout, stderr)
	case "patch-validate":
		return runPatchValidate(arguments[1:], stdout, stderr)
	case "script-validate":
		return runScriptValidate(arguments[1:], stdout, stderr)
	case "test-mode":
		return runTestMode(arguments[1:], stdout, stderr)
	case "test":
		return runTest(arguments[1:], stdout, stderr)
	case "toolchain-build":
		return runToolchainBuild(arguments[1:], stdout, stderr)
	default:
		fmt.Fprintln(stderr, usage)
		return 2
	}
}

func runChecked(arguments []string, stdout, stderr io.Writer) int {
	if len(arguments) < 6 || arguments[4] != "--" {
		fmt.Fprintln(stderr, "usage: hosttool checked-run <seconds> <expected-status> <label> <result-dir> -- <command> [args...]")
		return 125
	}
	seconds, secondsErr := strconv.ParseUint(arguments[0], 10, 31)
	expected, expectedErr := strconv.Atoi(arguments[1])
	if secondsErr != nil || seconds == 0 || expectedErr != nil || expected < 0 || expected > 255 || arguments[2] == "" {
		fmt.Fprintln(stderr, "gomadv3 checked runner requires a positive timeout and numeric expected status")
		return 125
	}
	resultRoot, err := filepath.Abs(arguments[3])
	if err != nil || resultRoot == string(filepath.Separator) {
		fmt.Fprintf(stderr, "gomadv3 checked runner result directory is invalid: %v\n", err)
		return 125
	}
	if err := os.MkdirAll(resultRoot, 0o700); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	workingDirectory, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	executed, runErr := commandrun.Run(context.Background(), commandrun.Request{
		Command: append([]string(nil), arguments[5:]...), Dir: workingDirectory, Env: os.Environ(),
		Timeout: time.Duration(seconds) * time.Second, TerminateGrace: 100 * time.Millisecond, OutputLimit: 1 << 20,
	})
	actual := executed.ExitCode
	if executed.WatchdogTimeout {
		actual = 124
	} else if executed.Termination == commandrun.TerminationSignal {
		actual = 128 + executed.SignalNumber
	}
	if runErr != nil {
		fmt.Fprintln(stderr, runErr)
		return 1
	}
	files := map[string][]byte{
		"stdout": executed.Stdout.RawBytes, "stderr": executed.Stderr.RawBytes,
		"status":           []byte(strconv.Itoa(actual) + "\n"),
		"timed-out":        []byte(strconv.Itoa(boolInt(executed.WatchdogTimeout)) + "\n"),
		"output-truncated": []byte(strconv.Itoa(boolInt(executed.Stdout.Truncated || executed.Stderr.Truncated)) + "\n"),
	}
	for name, contents := range files {
		if err := os.WriteFile(filepath.Join(resultRoot, name), contents, 0o600); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
	}
	if actual == expected {
		if expected == 124 && !executed.WatchdogTimeout {
			fmt.Fprintf(stderr, "gomadv3 process failed: %s: status 124 was not a timeout\n", arguments[2])
			return 1
		}
		return 0
	}
	fmt.Fprintf(stderr, "gomadv3 process failed: %s: status %d, want %d\n", arguments[2], actual, expected)
	if len(executed.Stdout.RawBytes) != 0 {
		fmt.Fprintf(stderr, "--- stdout ---\n%s", executed.Stdout.RawBytes)
	}
	if len(executed.Stderr.RawBytes) != 0 {
		fmt.Fprintf(stderr, "--- stderr ---\n%s", executed.Stderr.RawBytes)
	}
	return 1
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func patchFlags(name string, arguments []string, stderr io.Writer) (patchset.Config, bool) {
	flags := flag.NewFlagSet("hosttool "+name, flag.ContinueOnError)
	flags.SetOutput(stderr)
	var config patchset.Config
	flags.StringVar(&config.Root, "root", "", "Gomad v3 module root")
	flags.StringVar(&config.Patch, "patch", "", "versioned patch override")
	flags.StringVar(&config.Overlay, "overlay", "", "overlay root override")
	if name == "patch-materialize" {
		flags.StringVar(&config.SourceRoot, "source-root", "", "Go source root")
	}
	if err := flags.Parse(arguments); err != nil || flags.NArg() != 0 || config.Root == "" || name == "patch-materialize" && config.SourceRoot == "" {
		return patchset.Config{}, false
	}
	return config, true
}

func runPatchValidate(arguments []string, stdout, stderr io.Writer) int {
	config, ok := patchFlags("patch-validate", arguments, stderr)
	if !ok {
		return 2
	}
	if err := patchset.Validate(config); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	fmt.Fprintln(stdout, "gomadv3 patch and overlay inputs are valid")
	return 0
}

func runScriptValidate(arguments []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("hosttool script-validate", flag.ContinueOnError)
	flags.SetOutput(stderr)
	root := flags.String("root", "", "Gomad v3 module root")
	if err := flags.Parse(arguments); err != nil || flags.NArg() != 0 || *root == "" {
		return 2
	}
	if err := scriptpolicy.Validate(*root); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	fmt.Fprintln(stdout, "gomadv3 script ownership is valid")
	return 0
}

func runPatchMaterialize(arguments []string, stdout, stderr io.Writer) int {
	config, ok := patchFlags("patch-materialize", arguments, stderr)
	if !ok {
		return 2
	}
	if err := patchset.Materialize(context.Background(), config); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	fmt.Fprintln(stdout, "gomadv3 patch materialized")
	return 0
}

func runPatchRegenerate(arguments []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("hosttool patch-regenerate", flag.ContinueOnError)
	flags.SetOutput(stderr)
	var config patchset.Config
	var bootstrapGo string
	flags.StringVar(&config.Root, "root", "", "Gomad v3 module root")
	flags.StringVar(&config.CandidateRoot, "candidate-root", "", "modified Go source root")
	flags.StringVar(&config.Archive, "archive", "", "verified Go source archive override")
	flags.StringVar(&config.Output, "output", "", "regenerated patch output")
	flags.StringVar(&config.Gofmt, "gofmt", "", "bootstrap gofmt executable")
	flags.StringVar(&config.ToolchainRoot, "toolchain-root", "", "Gomad v3 toolchain state root")
	flags.StringVar(&bootstrapGo, "bootstrap-go", "", "bootstrap Go executable")
	if err := flags.Parse(arguments); err != nil || flags.NArg() != 0 || config.Root == "" || config.CandidateRoot == "" {
		return 2
	}
	if config.Gofmt == "" {
		var err error
		config.Gofmt, err = bootstrapGofmt(bootstrapGo)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
	}
	if err := patchset.Regenerate(context.Background(), config); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	fmt.Fprintln(stdout, "gomadv3 patch regenerated")
	return 0
}

func bootstrapGofmt(command string) (string, error) {
	if command == "" {
		command = os.Getenv("GOMADV3_BOOTSTRAP_GO")
	}
	if command == "" {
		var err error
		command, err = exec.LookPath("go")
		if err != nil {
			return "", errors.New("gomadv3 requires an installed bootstrap Go; set GOMADV3_BOOTSTRAP_GO")
		}
	}
	workingDirectory, err := os.Getwd()
	if err != nil {
		return "", err
	}
	result, err := commandrun.Run(context.Background(), commandrun.Request{
		Command: []string{command, "env", "GOROOT"}, Dir: workingDirectory, Env: withoutEnvironment(os.Environ(), "GOMADSEED"),
		Timeout: 30 * time.Second, TerminateGrace: 100 * time.Millisecond, OutputLimit: 64 << 10,
	})
	if err != nil {
		return "", fmt.Errorf("inspect bootstrap Go root: %w", err)
	}
	if result.Termination != commandrun.TerminationExit || result.ExitCode != 0 || result.WatchdogTimeout {
		return "", errors.New("inspect bootstrap Go root: command failed")
	}
	gofmt := filepath.Join(strings.TrimSpace(string(result.Stdout.RawBytes)), "bin", "gofmt")
	info, err := os.Stat(gofmt)
	if err != nil || !info.Mode().IsRegular() || info.Mode()&0o111 == 0 {
		return "", errors.Join(fmt.Errorf("gomadv3 bootstrap gofmt is missing: %s", gofmt), err)
	}
	return gofmt, nil
}

func withoutEnvironment(environment []string, names ...string) []string {
	filtered := make([]string, 0, len(environment))
	for _, item := range environment {
		keep := true
		for _, name := range names {
			if strings.HasPrefix(item, name+"=") {
				keep = false
				break
			}
		}
		if keep {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func runBuildKey(arguments []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("hosttool build-key", flag.ContinueOnError)
	flags.SetOutput(stderr)
	var input buildkey.Input
	var patch, overlay string
	flags.StringVar(&input.GoVersion, "go-version", "", "Go release")
	flags.StringVar(&input.ArchiveSHA256, "archive-sha256", "", "source archive digest")
	flags.StringVar(&patch, "patch", "", "patch path")
	flags.StringVar(&overlay, "overlay", "", "overlay root")
	flags.StringVar(&input.HostOS, "host-os", "", "host operating system")
	flags.StringVar(&input.HostArch, "host-arch", "", "host architecture")
	flags.StringVar(&input.BootstrapVersion, "bootstrap-version", "", "bootstrap Go version")
	flags.StringVar(&input.RecipeVersion, "recipe-version", "", "build recipe version")
	flags.StringVar(&input.BuildPath, "build-path", "", "sterile build PATH")
	flags.StringVar(&input.BashPath, "bash-path", "", "build bash path")
	flags.StringVar(&input.BashVersion, "bash-version", "", "build bash version")
	if err := flags.Parse(arguments); err != nil || flags.NArg() != 0 || patch == "" || overlay == "" {
		return 2
	}
	var err error
	input.PatchSHA256, err = buildkey.FileDigest(patch)
	if err != nil {
		fmt.Fprintf(stderr, "hash patch: %v\n", err)
		return 1
	}
	input.OverlaySHA256, err = buildkey.TreeDigest(overlay)
	if err != nil {
		fmt.Fprintf(stderr, "hash overlay: %v\n", err)
		return 1
	}
	key, err := buildkey.Compute(input)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	fmt.Fprintln(stdout, key)
	return 0
}

func runTestMode(arguments []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("hosttool test-mode", flag.ContinueOnError)
	flags.SetOutput(stderr)
	modeName := flags.String("mode", "", "test mode")
	output := flags.String("output", "", "tiers or success")
	if err := flags.Parse(arguments); err != nil || flags.NArg() != 0 {
		return 2
	}
	mode, err := testtier.Resolve(*modeName)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	switch *output {
	case "tiers":
		for _, tier := range mode.Tiers {
			if _, err := fmt.Fprintln(stdout, tier); err != nil {
				fmt.Fprintln(stderr, err)
				return 1
			}
		}
	case "success":
		if _, err := fmt.Fprintln(stdout, mode.Success); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
	default:
		fmt.Fprintln(stderr, "test-mode output must be tiers or success")
		return 2
	}
	return 0
}

func runTest(arguments []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("hosttool test", flag.ContinueOnError)
	flags.SetOutput(stderr)
	var config testdriver.Config
	var patch, overlay string
	flags.StringVar(&config.Root, "root", "", "Gomad v3 module root")
	flags.StringVar(&config.Mode, "mode", "", "test mode")
	flags.StringVar(&config.Go, "go", "", "Gomad toolchain Go executable")
	flags.StringVar(&config.Compiler, "compiler", "", "Gomad compiler-test executable")
	flags.StringVar(&patch, "patch", os.Getenv("GOMADV3_PATCH_FILE"), "versioned patch override")
	flags.StringVar(&overlay, "overlay", os.Getenv("GOMADV3_OVERLAY_DIR"), "overlay root override")
	if err := flags.Parse(arguments); err != nil || flags.NArg() != 0 || config.Root == "" || config.Mode == "" || config.Go == "" || config.Mode == "test-interception" && config.Compiler == "" {
		return 2
	}
	if err := patchset.Validate(patchset.Config{Root: config.Root, Patch: patch, Overlay: overlay}); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	report, err := testdriver.Run(context.Background(), config)
	if err != nil {
		fmt.Fprintln(stderr, err)
		for _, result := range report.Cases {
			if !result.Passed {
				if len(result.Stdout) != 0 {
					fmt.Fprintf(stderr, "--- stdout ---\n%s", result.Stdout)
				}
				if len(result.Stderr) != 0 {
					fmt.Fprintf(stderr, "--- stderr ---\n%s", result.Stderr)
				}
			}
		}
		return 1
	}
	mode, err := testtier.Resolve(config.Mode)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	fmt.Fprintln(stdout, mode.Success)
	return 0
}

func runToolchainBuild(arguments []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("hosttool toolchain-build", flag.ContinueOnError)
	flags.SetOutput(stderr)
	var config toolchainbuild.Config
	flags.StringVar(&config.Root, "root", "", "Gomad v3 module root")
	flags.StringVar(&config.ToolchainRoot, "toolchain-root", os.Getenv("GOMADV3_TOOLCHAIN_DIR"), "toolchain state root")
	flags.StringVar(&config.Patch, "patch", os.Getenv("GOMADV3_PATCH_FILE"), "versioned patch override")
	flags.StringVar(&config.Overlay, "overlay", os.Getenv("GOMADV3_OVERLAY_DIR"), "overlay root override")
	flags.StringVar(&config.BootstrapGo, "bootstrap-go", os.Getenv("GOMADV3_BOOTSTRAP_GO"), "bootstrap Go executable")
	flags.StringVar(&config.BuildBash, "build-bash", os.Getenv("BASH"), "Bash executable for make.bash")
	if err := flags.Parse(arguments); err != nil || flags.NArg() != 0 || config.Root == "" {
		return 2
	}
	var err error
	if config.BootstrapGo == "" {
		config.BootstrapGo, err = exec.LookPath("go")
		if err != nil {
			fmt.Fprintln(stderr, "gomadv3 requires an installed bootstrap Go; set GOMADV3_BOOTSTRAP_GO")
			return 1
		}
	}
	if config.BuildBash == "" {
		config.BuildBash, err = exec.LookPath("bash")
		if err != nil {
			fmt.Fprintln(stderr, "gomadv3 requires Bash to run upstream make.bash")
			return 1
		}
	}
	config.BuildBashVersion = os.Getenv("BASH_VERSION")
	config.BuildPath = canonicalBuildPath
	config.Testing = os.Getenv("GOMADV3_TESTING") == "1"
	config.FailurePhase = os.Getenv("GOMADV3_TEST_FAIL_PHASE")
	result, err := toolchainbuild.Build(context.Background(), config)
	if err != nil {
		fmt.Fprintln(stderr, err)
		var injected *toolchainbuild.InjectedFailure
		if errors.As(err, &injected) {
			return 86
		}
		return 1
	}
	if result.Waited {
		fmt.Fprintf(stdout, "waiting for gomadv3 build key %s\n", result.BuildKey)
	}
	fmt.Fprintf(stdout, "gomadv3 toolchain is ready (%s/%s, key %s)\n", result.HostOS, result.HostArch, result.BuildKey)
	return 0
}
