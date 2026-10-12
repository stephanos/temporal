package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"
)

type preparedOverlay struct {
	Manifest string            `json:"manifest"`
	Hashes   map[string]string `json:"sha256"`
}

func instrumentRefuse(original []byte) ([]byte, error) {
	anchor := []byte("\trecord(\"net.syscall.refused\", []byte(operation), nil, 0, resultClass(err), 0, 0)\n")
	call := []byte("\tgomadNativeObserveRefuse(operation)\n")
	if bytes.Count(original, anchor) != 1 || bytes.Contains(original, call) {
		return nil, errors.New("actual backend reporting seam differs or is already instrumented")
	}
	instrumented := bytes.Replace(original, anchor, append(append([]byte{}, call...), anchor...), 1)
	if !bytes.Equal(bytes.Replace(instrumented, call, nil, 1), original) {
		return nil, errors.New("backend reporting instrumentation is not reversible")
	}
	return instrumented, nil
}

func prepareOverlay(root, goroot, output string) (preparedOverlay, error) {
	var err error
	for _, path := range []*string{&root, &goroot, &output} {
		*path, err = filepath.Abs(*path)
		if err != nil {
			return preparedOverlay{}, err
		}
	}
	if err := os.MkdirAll(output, 0700); err != nil {
		return preparedOverlay{}, err
	}
	resolvedOutput, err := filepath.EvalSymlinks(output)
	if err != nil {
		return preparedOverlay{}, err
	}
	for _, sourceRoot := range []string{goroot, filepath.Join(root, "toolchain/runtime/overlay"), filepath.Join(root, "toolchain/runtime/testdata")} {
		resolved, err := filepath.EvalSymlinks(sourceRoot)
		if err != nil {
			return preparedOverlay{}, err
		}
		relative, err := filepath.Rel(resolved, resolvedOutput)
		if err != nil {
			return preparedOverlay{}, err
		}
		if relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
			return preparedOverlay{}, errors.New("private overlay must be outside source trees")
		}
	}
	prepared := preparedOverlay{Manifest: filepath.Join(output, "overlay.json"), Hashes: make(map[string]string)}
	var backend []byte
	for _, source := range []string{"runtime/gomad_vfd.go", "runtime/gomad.go", "syscall/gomad_vfd_unix.go", "syscall/gomad_vfd_linux_amd64.go", "syscall/gomad_vfd_darwin_arm64.go", "internal/gomadio/descriptor_backend.go", "internal/gomadio/network.go", "internal/gomadio/gomadio.go", "internal/gomadvfd/descriptor.go", "net/gomad.go"} {
		expected, err := readRegular(filepath.Join(root, "toolchain/runtime/overlay/src", source))
		if err != nil {
			return preparedOverlay{}, err
		}
		actual, err := readRegular(filepath.Join(goroot, "src", source))
		if err != nil {
			return preparedOverlay{}, err
		}
		if !bytes.Equal(actual, expected) {
			return preparedOverlay{}, fmt.Errorf("selected toolchain source differs: %s", source)
		}
		prepared.Hashes[source] = digest(actual)
		if source == "internal/gomadio/descriptor_backend.go" {
			backend = actual
		}
	}
	replacements := make(map[string]string)
	backendTarget := filepath.Join(goroot, "src/internal/gomadio/descriptor_backend.go")
	instrumented, err := instrumentRefuse(backend)
	if err != nil {
		return preparedOverlay{}, err
	}
	backendReplacement := filepath.Join(output, "descriptor_backend.go")
	if err := writePrivate(backendReplacement, instrumented); err != nil {
		return preparedOverlay{}, err
	}
	if err := writePrivate(filepath.Join(output, "descriptor_backend.original.go"), backend); err != nil {
		return preparedOverlay{}, err
	}
	replacements[backendTarget] = backendReplacement
	prepared.Hashes["instrumented_backend"] = digest(instrumented)
	for _, file := range []struct{ source, target string }{
		{"fixture_test.go.txt", "syscall/gomad_native_fixture_test.go"},
		{"runtime_probe.go.txt", "runtime/gomad_native_probe.go"},
		{"linux_probe.go.txt", "syscall/gomad_native_linux_amd64_test.go"},
		{"darwin_probe.go.txt", "syscall/gomad_native_darwin_arm64_test.go"},
		{"gomadio_probe.go.txt", "internal/gomadio/gomad_native_gomadio_probe.go"},
	} {
		contents, err := readRegular(filepath.Join(root, "toolchain/runtime/testdata/vfdnative", file.source))
		if err != nil {
			return preparedOverlay{}, err
		}
		target := filepath.Join(goroot, "src", file.target)
		if _, err := os.Lstat(target); !errors.Is(err, os.ErrNotExist) {
			return preparedOverlay{}, errors.Join(fmt.Errorf("synthetic source already exists: %s", target), err)
		}
		replacement := filepath.Join(output, filepath.Base(file.target))
		if err := writePrivate(replacement, contents); err != nil {
			return preparedOverlay{}, err
		}
		replacements[target] = replacement
		prepared.Hashes[file.source] = digest(contents)
	}
	manifest, err := json.MarshalIndent(struct{ Replace map[string]string }{replacements}, "", "  ")
	if err != nil {
		return preparedOverlay{}, err
	}
	manifest = append(manifest, '\n')
	if err := writePrivate(prepared.Manifest, manifest); err != nil {
		return preparedOverlay{}, err
	}
	prepared.Hashes["manifest"] = digest(manifest)
	return prepared, nil
}

func admitNative(goos, goarch, help string) error {
	if goos+"/"+goarch != "linux/amd64" && goos+"/"+goarch != "darwin/arm64" {
		return fmt.Errorf("native fixture requires linux/amd64 or darwin/arm64; got %s/%s", goos, goarch)
	}
	for _, line := range strings.Split(help, "\n") {
		if fields := strings.Fields(line); len(fields) == 1 && fields[0] == "-gomadguard" {
			return nil
		}
	}
	return errors.New("selected compiler lacks Gomad guard; a materialized source tree is not a rebuilt compiler")
}

func probeEnvironment(original []string) []string {
	env := make([]string, 0, len(original))
	for _, value := range original {
		key, _, _ := strings.Cut(value, "=")
		if strings.HasPrefix(key, "GOMAD") {
			continue
		}
		switch key {
		case "GOOS", "GOARCH", "GOEXPERIMENT", "GOENV", "GOFLAGS", "GOWORK", "GOTOOLCHAIN", "CGO_ENABLED", "GODEBUG", "GOROOT", "TZ":
			continue
		}
		env = append(env, value)
	}
	return append(env, "CGO_ENABLED=0", "GOEXPERIMENT=nogreenteagc", "GOENV=off", "GOFLAGS=", "GOWORK=off", "GOTOOLCHAIN=local", "TZ=UTC")
}

func checkChild(output []byte, name string, exitCode int) error {
	if exitCode != 0 {
		return fmt.Errorf("child exited %d", exitCode)
	}
	var runs, passes int
	for _, line := range strings.Split(string(output), "\n") {
		if line == "=== RUN   "+name {
			runs++
		}
		if strings.HasPrefix(line, "--- PASS: "+name+" (") {
			passes++
		}
		if strings.HasPrefix(line, "--- SKIP: "+name+" (") {
			return errors.New("selected test was skipped: inconclusive")
		}
	}
	if runs != 1 || passes != 1 {
		return fmt.Errorf("inconclusive test selection: runs=%d passes=%d", runs, passes)
	}
	return nil
}

func readRegular(path string) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("input is not a regular file: %s", path)
	}
	return os.ReadFile(path)
}

func digest(contents []byte) string {
	sum := sha256.Sum256(contents)
	return hex.EncodeToString(sum[:])
}

func writePrivate(path string, contents []byte) error {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	_, err = file.Write(contents)
	return errors.Join(err, file.Close())
}

type commandResult struct {
	Args     []string `json:"args"`
	Log      string   `json:"log"`
	ExitCode int      `json:"exit_code"`
	Elapsed  string   `json:"elapsed"`
	SHA256   string   `json:"output_sha256"`
}

func runCommand(ctx context.Context, root string, args, env []string, log string) (commandResult, []byte, error) {
	result := commandResult{Args: args, Log: log, ExitCode: -1}
	file, err := os.OpenFile(log, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return result, nil, err
	}
	started := time.Now()
	command := exec.CommandContext(ctx, args[0], args[1:]...)
	command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	command.Cancel = func() error {
		err := syscall.Kill(-command.Process.Pid, syscall.SIGKILL)
		if errors.Is(err, syscall.ESRCH) {
			return os.ErrProcessDone
		}
		return err
	}
	command.Dir, command.Env, command.Stdout, command.Stderr = root, env, file, file
	err = command.Run()
	result.Elapsed = time.Since(started).String()
	if command.ProcessState != nil {
		result.ExitCode = command.ProcessState.ExitCode()
	}
	err = errors.Join(err, ctx.Err(), file.Close())
	contents, readErr := os.ReadFile(log)
	if readErr == nil {
		result.SHA256 = digest(contents)
	}
	return result, contents, errors.Join(err, readErr)
}

var nativeCases = []string{"TestGomadNativeTCP", "TestGomadNativeReadyWake", "TestGomadNativeAcceptDeadline", "TestGomadNativeReadDeadline", "TestGomadNativeWriteDeadline", "TestGomadNativeSimultaneousDeadlines", "TestGomadNativeCloseParkedRead", "TestGomadNativePollDescReuse"}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() (returnedErr error) {
	rootFlag := flag.String("root", ".", "Gomad module root")
	goFlag := flag.String("go", "", "selected patched Go executable")
	outputFlag := flag.String("output", "", "private evidence directory")
	prepareFlag := flag.Bool("prepare-only", false, "prepare source overlay without compiling or executing")
	flag.Parse()
	if flag.NArg() != 0 || *goFlag == "" {
		return errors.New("vfdnative requires -go and accepts no positional arguments")
	}
	root, err := filepath.Abs(*rootFlag)
	if err != nil {
		return err
	}
	goPath, err := filepath.Abs(*goFlag)
	if err != nil {
		return err
	}
	goPath, err = filepath.EvalSymlinks(goPath)
	if err != nil {
		return err
	}
	if !*prepareFlag && runtime.GOOS+"/"+runtime.GOARCH != "linux/amd64" && runtime.GOOS+"/"+runtime.GOARCH != "darwin/arm64" {
		return admitNative(runtime.GOOS, runtime.GOARCH, "")
	}
	output := *outputFlag
	if output == "" {
		if *prepareFlag {
			return errors.New("prepare-only requires -output")
		}
		output, err = os.MkdirTemp(filepath.Join(root, ".toolchain"), "vfdnative-")
		if err != nil {
			return err
		}
	}
	output, err = filepath.Abs(output)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(output, 0700); err != nil {
		return err
	}
	var prepared preparedOverlay
	var commands []commandResult
	defer func() {
		report := struct {
			Mode     string          `json:"mode"`
			Boundary string          `json:"boundary"`
			Host     string          `json:"host"`
			Prepared preparedOverlay `json:"prepared"`
			Commands []commandResult `json:"commands"`
			Open     []string        `json:"open_requirements"`
			Error    string          `json:"error,omitempty"`
		}{Mode: "native", Boundary: "internal-switch/guard-off/direct-seed/no-profile", Host: runtime.GOOS + "/" + runtime.GOARCH, Prepared: prepared, Commands: commands, Open: []string{"whole-process actual OS zero-socket audit", "UDP transcript persistence (disabled direct-seed profile)", "unobserved native paths including reporting observer and platform qualification"}}
		if *prepareFlag {
			report.Mode = "source-only"
		}
		if returnedErr != nil {
			report.Error = returnedErr.Error()
		}
		contents, err := json.MarshalIndent(report, "", "  ")
		if err == nil {
			err = writePrivate(filepath.Join(output, "receipt.json"), append(contents, '\n'))
		}
		returnedErr = errors.Join(returnedErr, err)
	}()
	env := probeEnvironment(os.Environ())
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	invoke := func(args, childEnv []string, name string, timeout time.Duration) ([]byte, commandResult, error) {
		childContext, childCancel := context.WithTimeout(ctx, timeout)
		defer childCancel()
		result, contents, err := runCommand(childContext, root, args, childEnv, filepath.Join(output, name+".log"))
		commands = append(commands, result)
		if err != nil {
			return contents, result, fmt.Errorf("%s (exit %d, log %s): %w", name, result.ExitCode, result.Log, err)
		}
		return contents, result, nil
	}
	identity, _, err := invoke([]string{goPath, "env", "GOROOT", "GOVERSION", "GOOS", "GOARCH"}, env, "go-env", time.Minute)
	if err != nil {
		return err
	}
	fields := strings.Split(strings.TrimSpace(string(identity)), "\n")
	if len(fields) != 4 || fields[1] != "go1.27.1" {
		return fmt.Errorf("unexpected selected Go identity: %q", identity)
	}
	prepared, err = prepareOverlay(root, fields[0], output)
	if err != nil {
		return err
	}
	for name, path := range map[string]string{"go": goPath, "compiler": filepath.Join(fields[0], "pkg/tool", runtime.GOOS+"_"+runtime.GOARCH, "compile"), "netpoll": filepath.Join(fields[0], "src/runtime/netpoll.go")} {
		contents, err := readRegular(path)
		if err != nil {
			return err
		}
		prepared.Hashes[name] = digest(contents)
		if name == "netpoll" {
			for _, anchor := range []string{"gomadVirtualPollBlockCommit", "gomadVirtualPollOpen(pd)", "gomadVirtualPollClose(pd)", "pd.gomadVirtual == 0"} {
				if !bytes.Contains(contents, []byte(anchor)) {
					return fmt.Errorf("selected poll source lacks virtual branch: %s", anchor)
				}
			}
		}
	}
	if *prepareFlag {
		fmt.Printf("source-only overlay: %s\n", prepared.Manifest)
		return nil
	}
	if fields[2] != runtime.GOOS || fields[3] != runtime.GOARCH {
		return errors.New("selected compiler targets a different platform")
	}
	help, _, err := invoke([]string{goPath, "tool", "compile", "-help"}, env, "compiler-help", time.Minute)
	if err != nil {
		return err
	}
	if err := admitNative(fields[2], fields[3], string(help)); err != nil {
		return err
	}
	binary := filepath.Join(output, "syscall.test")
	if _, _, err := invoke([]string{goPath, "test", "-tags", "test_dep", "-c", "-overlay=" + prepared.Manifest, "-o", binary, "syscall"}, env, "compile", 8*time.Minute); err != nil {
		return err
	}
	binaryBytes, err := readRegular(binary)
	if err != nil {
		return err
	}
	prepared.Hashes["binary"] = digest(binaryBytes)
	for _, name := range nativeCases {
		contents, result, err := invoke([]string{binary, "-test.run=^" + name + "$", "-test.timeout=0", "-test.v"}, append(env, "GOMADSEED=17"), name, 30*time.Second)
		if err != nil {
			return err
		}
		if err := checkChild(contents, name, result.ExitCode); err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
	}
	fmt.Printf("native internal-switch fixtures passed; OS socket audit remains open; evidence: %s\n", output)
	return nil
}
