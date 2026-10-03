package main

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"go.temporal.io/server/tools/gomad3/choice"
)

func main() {
	if err := measure(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func measure() error {
	root, err := filepath.Abs(os.Args[1])
	if err != nil {
		return err
	}
	output, err := filepath.Abs(os.Args[2])
	if err != nil {
		return err
	}
	env := []string{}
	for _, entry := range os.Environ() {
		name, _, _ := strings.Cut(entry, "=")
		if name == "GOROOT" || name == "GOBIN" || name == "GOMADSEED" || strings.HasPrefix(name, "GOMAD3_") || name == "GOTOOLCHAIN" || name == "CGO_ENABLED" {
			continue
		}
		env = append(env, entry)
	}
	env = append(env, "GOTOOLCHAIN=local", "CGO_ENABLED=0")
	var measurements []map[string]any
	source := filepath.Join(root, "internal/gomadtool/conformance/testdata/runtime_owned/main.go")
	sourceBytes, err := os.ReadFile(source)
	if err != nil {
		return err
	}
	for _, version := range []struct{ name, key string }{
		{"before", "2008ea81459afbd1ee019beb4a231968c1af8a24f4b46d4a8426b3188b1a9b87"},
		{"after", "4a6e5b695ea538f0a56eb70874ff945693223b53d0fcee56cc555f89e1a9ac0e"},
	} {
		directory := filepath.Join(output, "same-source-"+version.name)
		if err := os.MkdirAll(directory, 0o700); err != nil {
			return err
		}
		probe := filepath.Join(directory, "probe")
		build := exec.Command(filepath.Join(root, ".toolchain/builds", version.key, "bin/go"), "build", "-o", probe, "./runtime_owned")
		build.Dir = filepath.Join(root, "internal/gomadtool/conformance/testdata")
		build.Env = env
		if text, err := build.CombinedOutput(); err != nil {
			return fmt.Errorf("build %s: %w: %s", version.name, err, text)
		}
		for seed := 1; seed <= 32; seed++ {
			name := filepath.Join(directory, strconv.Itoa(seed))
			traceFile, err := os.Create(name + ".backing")
			if err != nil {
				return err
			}
			header := make([]byte, 64)
			copy(header, "GOMADCH\x03")
			binaryEncoding := binary.BigEndian
			binaryEncoding.PutUint32(header[8:12], 3)
			binaryEncoding.PutUint64(header[16:24], 1<<20)
			binaryEncoding.PutUint64(header[24:32], 64)
			if _, err := traceFile.Write(header); err != nil {
				return err
			}
			if err := traceFile.Truncate(1 << 20); err != nil {
				return err
			}
			terminalFile, err := os.Create(name + ".terminal")
			if err != nil {
				return err
			}
			command := exec.Command(probe, "two")
			command.Env = append(append([]string{}, env...), "GOMADSEED="+strconv.Itoa(seed), "GOMAD3_CHOICE_MODE=1", "GOMAD3_CHOICE_TRACE_FD=3", "GOMAD3_CHOICE_TERMINAL_FD=4", "GOMAD3_CHOICE_TRACE_BYTES=1048576")
			command.ExtraFiles = []*os.File{traceFile, terminalFile}
			text, err := command.CombinedOutput()
			traceErr, terminalErr := traceFile.Close(), terminalFile.Close()
			if err != nil {
				return fmt.Errorf("run %s/%d: %w: %s", version.name, seed, err, text)
			}
			if traceErr != nil {
				return traceErr
			}
			if terminalErr != nil {
				return terminalErr
			}
			if string(text) != "[64 64]\n" {
				return fmt.Errorf("unexpected user output: %q", text)
			}
			backing, err := os.ReadFile(name + ".backing")
			if err != nil {
				return err
			}
			terminal, err := os.ReadFile(name + ".terminal")
			if err != nil {
				return err
			}
			next := binaryEncoding.Uint64(backing[24:32])
			if next < 64 || next > uint64(len(backing)) {
				return fmt.Errorf("invalid next offset %d", next)
			}
			trace, err := choice.DecodeTrace(backing[64:next], terminal, 1<<20)
			if err != nil {
				return err
			}
			if err := os.WriteFile(name+".trace", trace.Bytes, 0o600); err != nil {
				return err
			}
			if err := os.Remove(name + ".backing"); err != nil {
				return err
			}
			decisions, runtimeSelected := 0, 0
			for _, record := range trace.Records {
				if record.Kind != choice.KindRunnable {
					continue
				}
				decisions++
				for ordinal := uint64(1); ordinal <= 8; ordinal++ {
					var encoded [8]byte
					binaryEncoding.PutUint64(encoded[:], ordinal)
					identity := sha256.Sum256(append([]byte("gomad3-choice-goroutine-runtime/v1"), encoded[:]...))
					if identity == record.SelectedIdentity {
						runtimeSelected++
					}
				}
			}
			measurements = append(measurements, map[string]any{"version": version.name, "build_key": version.key, "source_sha256": fmt.Sprintf("%x", sha256.Sum256(sourceBytes)), "seed": seed, "decisions": decisions, "runtime_selected": runtimeSelected, "trace_sha256": fmt.Sprintf("%x", trace.SHA256)})
		}
	}
	data, err := json.MarshalIndent(measurements, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(output, "same-source-counts.json"), append(data, '\n'), 0o600)
}
