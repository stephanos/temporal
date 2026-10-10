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

type insertion struct{ anchor, call string }

var insertions = []insertion{
	{"func gomadGenericWritev(fd int, p *Iovec, count uintptr) (uintptr, error) {\n", "\tgomadPointerGrowWritev(p, 64)\n"},
	{"func gomadVirtualAccept(fd int, raw *RawSockaddrAny, length *_Socklen, flags int) (int, error) {\n", "\tgomadPointerGrowPair(raw, length, 64)\n"},
	{"func gomadVirtualName(fd int, raw *RawSockaddrAny, length *_Socklen, peer bool) error {\n", "\tgomadPointerGrowPair(raw, length, 64)\n"},
	{"func gomadVirtualGetsockopt(fd, level, name int, p unsafe.Pointer, length *_Socklen) error {\n", "\tgomadPointerGrowOption(p, length, 64)\n"},
}

type preparedOverlay struct {
	Manifest string            `json:"manifest"`
	Hashes   map[string]string `json:"sha256"`
}

type commandResult struct {
	Args     []string `json:"args"`
	Log      string   `json:"log"`
	ExitCode int      `json:"exit_code"`
	Elapsed  string   `json:"elapsed"`
}

func instrument(original []byte) ([]byte, error) {
	instrumented := bytes.Clone(original)
	for _, insertion := range insertions {
		if bytes.Count(original, []byte(insertion.anchor)) != 1 || bytes.Contains(original, []byte(insertion.call)) {
			return nil, fmt.Errorf("unmatched or instrumented helper signature: %s", strings.TrimSpace(insertion.anchor))
		}
		instrumented = bytes.Replace(instrumented, []byte(insertion.anchor), []byte(insertion.anchor+insertion.call), 1)
	}
	reversed := bytes.Clone(instrumented)
	for _, insertion := range insertions {
		reversed = bytes.Replace(reversed, []byte(insertion.anchor+insertion.call), []byte(insertion.anchor), 1)
	}
	if !bytes.Equal(original, reversed) {
		return nil, errors.New("growth insertions did not reverse to the frozen helper")
	}
	return instrumented, nil
}

func prepareOverlay(root, goroot, output string) (preparedOverlay, error) {
	for _, path := range []*string{&root, &goroot, &output} {
		absolute, err := filepath.Abs(*path)
		if err != nil {
			return preparedOverlay{}, err
		}
		*path = absolute
	}
	if err := os.MkdirAll(output, 0o700); err != nil {
		return preparedOverlay{}, err
	}
	resolvedOutput, err := filepath.EvalSymlinks(output)
	if err != nil {
		return preparedOverlay{}, err
	}
	for _, sourceRoot := range []string{goroot, filepath.Join(root, "toolchain", "runtime", "overlay")} {
		resolvedRoot, err := filepath.EvalSymlinks(sourceRoot)
		if err != nil {
			return preparedOverlay{}, err
		}
		relative, err := filepath.Rel(resolvedRoot, resolvedOutput)
		if err != nil {
			return preparedOverlay{}, err
		}
		if relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
			return preparedOverlay{}, errors.New("private overlay output must be outside toolchain sources")
		}
	}
	repository := filepath.Join(root, "toolchain", "runtime", "overlay", "src", "syscall", "gomad_vfd_unix.go")
	selected := filepath.Join(goroot, "src", "syscall", "gomad_vfd_unix.go")
	original, err := readRegular(repository)
	if err != nil {
		return preparedOverlay{}, err
	}
	actual, err := readRegular(selected)
	if err != nil {
		return preparedOverlay{}, err
	}
	if !bytes.Equal(original, actual) {
		return preparedOverlay{}, errors.New("selected toolchain helper differs from repository source")
	}
	instrumented, err := instrument(original)
	if err != nil {
		return preparedOverlay{}, err
	}
	prepared := preparedOverlay{Manifest: filepath.Join(output, "overlay.json"), Hashes: make(map[string]string)}
	replacements := make(map[string]string)
	inputs := []struct{ source, target string }{
		{"growth.go.txt", "gomad_pointer_growth.go"},
		{"fixture_test.go.txt", "gomad_pointer_fixture_test.go"},
		{"linux_export_test.go.txt", "gomad_pointer_linux_amd64_test.go"},
		{"darwin_export_test.go.txt", "gomad_pointer_darwin_arm64_test.go"},
	}
	for _, input := range inputs {
		contents, err := readRegular(filepath.Join(root, "toolchain", "runtime", "testdata", "vfdpointer", input.source))
		if err != nil {
			return preparedOverlay{}, err
		}
		target := filepath.Join(goroot, "src", "syscall", input.target)
		if _, err := os.Lstat(target); !errors.Is(err, os.ErrNotExist) {
			return preparedOverlay{}, errors.Join(fmt.Errorf("synthetic overlay path already exists: %s", target), err)
		}
		replacement := filepath.Join(output, input.target)
		if err := writePrivate(replacement, contents); err != nil {
			return preparedOverlay{}, err
		}
		replacements[target] = replacement
		prepared.Hashes[input.source] = digest(contents)
	}
	replacement := filepath.Join(output, "gomad_vfd_unix.go")
	for _, file := range []struct {
		path string
		data []byte
	}{{filepath.Join(output, "original.go.txt"), original}, {replacement, instrumented}} {
		if err := writePrivate(file.path, file.data); err != nil {
			return preparedOverlay{}, err
		}
	}
	replacements[selected] = replacement
	manifest, err := json.MarshalIndent(struct{ Replace map[string]string }{replacements}, "", "  ")
	if err != nil {
		return preparedOverlay{}, err
	}
	if err := writePrivate(prepared.Manifest, append(manifest, '\n')); err != nil {
		return preparedOverlay{}, err
	}
	prepared.Hashes["original"] = digest(original)
	prepared.Hashes["instrumented"] = digest(instrumented)
	prepared.Hashes["manifest"] = digest(append(manifest, '\n'))
	return prepared, nil
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
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	_, err = file.Write(contents)
	return errors.Join(err, file.Close())
}

func admitNative(goos, goarch, compilerHelp string) error {
	if goos+"/"+goarch != "linux/amd64" && goos+"/"+goarch != "darwin/arm64" {
		return fmt.Errorf("native pointer fixture requires linux/amd64 or darwin/arm64; got %s/%s", goos, goarch)
	}
	for _, line := range strings.Split(compilerHelp, "\n") {
		if fields := strings.Fields(line); len(fields) == 1 && fields[0] == "-gomadguard" {
			return nil
		}
	}
	return errors.New("selected compiler lacks Gomad guard; source materialization is not a rebuilt toolchain")
}

func runCommand(ctx context.Context, args, env []string, log string) (commandResult, error) {
	result := commandResult{Args: args, Log: log, ExitCode: -1}
	file, err := os.OpenFile(log, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return result, err
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
	command.Env, command.Stdout, command.Stderr = env, file, file
	err = command.Run()
	result.Elapsed = time.Since(started).String()
	if command.ProcessState != nil {
		result.ExitCode = command.ProcessState.ExitCode()
	}
	return result, errors.Join(err, ctx.Err(), file.Close())
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() (returnedErr error) {
	rootFlag := flag.String("root", ".", "Gomad module root")
	goFlag := flag.String("go", "", "selected patched Go executable")
	outputFlag := flag.String("output", "", "private retained evidence directory")
	prepareFlag := flag.Bool("prepare-only", false, "prepare source overlay without compiling or executing")
	flag.Parse()
	if flag.NArg() != 0 || *goFlag == "" {
		return errors.New("vfdpointer requires -go and accepts no positional arguments")
	}
	root, err := filepath.Abs(*rootFlag)
	if err != nil {
		return err
	}
	goPath, err := filepath.Abs(*goFlag)
	if err != nil {
		return err
	}
	if !*prepareFlag && runtime.GOOS+"/"+runtime.GOARCH != "linux/amd64" && runtime.GOOS+"/"+runtime.GOARCH != "darwin/arm64" {
		return admitNative(runtime.GOOS, runtime.GOARCH, "")
	}
	output := *outputFlag
	if output == "" {
		if *prepareFlag {
			return errors.New("prepare-only requires an explicit -output directory")
		}
		output, err = os.MkdirTemp(filepath.Join(root, ".toolchain"), "vfdpointer-")
		if err != nil {
			return err
		}
	}
	output, err = filepath.Abs(output)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(output, 0o700); err != nil {
		return err
	}
	env := probeEnvironment(os.Environ())
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	var commands []commandResult
	var prepared preparedOverlay
	defer func() {
		receipt := struct {
			Mode     string          `json:"mode"`
			Prepared preparedOverlay `json:"prepared"`
			Commands []commandResult `json:"commands"`
			Error    string          `json:"error,omitempty"`
		}{Mode: "native", Prepared: prepared, Commands: commands}
		if *prepareFlag {
			receipt.Mode = "source-only"
		}
		if returnedErr != nil {
			receipt.Error = returnedErr.Error()
		}
		contents, err := json.MarshalIndent(receipt, "", "  ")
		if err == nil {
			err = writePrivate(filepath.Join(output, "receipt.json"), append(contents, '\n'))
		}
		returnedErr = errors.Join(returnedErr, err)
	}()
	invoke := func(args []string, name string) (string, error) {
		result, err := runCommand(ctx, args, env, filepath.Join(output, name+".log"))
		commands = append(commands, result)
		if err != nil {
			return "", fmt.Errorf("%s (exit %d, log %s): %w", name, result.ExitCode, result.Log, err)
		}
		contents, err := os.ReadFile(result.Log)
		return string(contents), err
	}
	goenv, err := invoke([]string{goPath, "env", "GOROOT", "GOVERSION", "GOOS", "GOARCH"}, "go-env")
	if err != nil {
		return err
	}
	fields := strings.Fields(goenv)
	if len(fields) != 4 || fields[1] != "go1.27.1" {
		return fmt.Errorf("unexpected selected Go identity: %q", goenv)
	}
	prepared, err = prepareOverlay(root, fields[0], output)
	if err != nil {
		return err
	}
	compiler := filepath.Join(fields[0], "pkg", "tool", runtime.GOOS+"_"+runtime.GOARCH, "compile")
	compilerBytes, err := readRegular(compiler)
	if err != nil {
		return err
	}
	prepared.Hashes["compiler"] = digest(compilerBytes)
	goBytes, err := readRegular(goPath)
	if err != nil {
		return err
	}
	prepared.Hashes["go"] = digest(goBytes)
	if *prepareFlag {
		fmt.Printf("source-only overlay: %s\n", prepared.Manifest)
		return nil
	}
	help, err := invoke([]string{goPath, "tool", "compile", "-help"}, "compiler-help")
	if err != nil {
		return err
	}
	if fields[2] != runtime.GOOS || fields[3] != runtime.GOARCH {
		return errors.New("selected Go targets a different platform; refusing native execution")
	}
	if err := admitNative(fields[2], fields[3], help); err != nil {
		return err
	}
	binary := filepath.Join(output, "syscall.test")
	if _, err := invoke([]string{goPath, "test", "-tags", "test_dep", "-c", "-overlay=" + prepared.Manifest, "-o", binary, "syscall"}, "compile"); err != nil {
		return err
	}
	cases := []string{"no-growth"}
	variants := 4
	if runtime.GOOS == "darwin" {
		variants = 6
	}
	for _, operation := range []string{"writev-partial", "writev-full", "accept", "accept-null", "accept-mismatch", "sockname", "sockname-short", "peername", "peername-short", "option-type", "option-error", "option-short"} {
		for variant := range variants {
			if strings.HasPrefix(operation, "option-") && variant%2 == 0 {
				continue
			}
			cases = append(cases, fmt.Sprintf("%s/%d", operation, variant))
		}
	}
	for _, selection := range cases {
		childContext, childCancel := context.WithTimeout(ctx, 30*time.Second)
		result, err := runCommand(childContext, []string{binary, "-test.run=^TestGomadVFDPointerFixture$", "-test.timeout=20s", "-test.v"}, append(env, "GOMAD_VFD_POINTER_CASE="+selection), filepath.Join(output, "case-"+strings.ReplaceAll(selection, "/", "-")+".log"))
		childCancel()
		commands = append(commands, result)
		if err != nil {
			return fmt.Errorf("pointer case %s (exit %d, log %s): %w", selection, result.ExitCode, result.Log, err)
		}
		contents, err := os.ReadFile(result.Log)
		if err != nil {
			return err
		}
		if err := verifyFixture(result.ExitCode, contents); err != nil {
			return fmt.Errorf("pointer case %s: %w", selection, err)
		}
	}
	fmt.Printf("native instrumented pointer fixtures passed; evidence: %s\n", output)
	return nil
}

func probeEnvironment(original []string) []string {
	env := make([]string, 0, len(original))
	for _, value := range original {
		key, _, _ := strings.Cut(value, "=")
		switch key {
		case "GOMADSEED", "GOMAD3_CHILD_SEED", "GOMAD_VFD_POINTER_CASE", "GOEXPERIMENT", "GOENV", "GOFLAGS", "GOWORK", "GOTOOLCHAIN":
			continue
		}
		env = append(env, value)
	}
	return append(env, "GOEXPERIMENT=nogreenteagc", "GOENV=off", "GOFLAGS=", "GOWORK=off", "GOTOOLCHAIN=local")
}

func verifyFixture(status int, output []byte) error {
	if status != 0 || bytes.Count(output, []byte("=== RUN   TestGomadVFDPointerFixture\n")) != 1 || bytes.Count(output, []byte("--- PASS: TestGomadVFDPointerFixture (")) != 1 || bytes.Contains(output, []byte("--- SKIP:")) {
		return errors.New("pointer child did not complete exactly one selected fixture")
	}
	return nil
}
