package main

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const maximumBytes = 32 << 20

//go:embed request.json
var requestBytes []byte

//go:embed response.json
var responseBytes []byte

func main() {
	executable, err := os.Executable()
	if err != nil {
		os.Exit(91)
	}
	directory := filepath.Dir(executable)
	if err := os.WriteFile(filepath.Join(directory, "child.pid"),
		[]byte(fmt.Sprint(os.Getpid())), 0o600); err != nil {
		os.Exit(92)
	}
	mode := filepath.Base(directory)
	input, err := io.ReadAll(io.LimitReader(os.Stdin, maximumBytes+1))
	if err != nil || !bytes.Equal(input, requestBytes) {
		os.Exit(93)
	}

	switch {
	case strings.HasPrefix(mode, "valid"):
		_, _ = os.Stdout.Write(responseBytes)
	case mode == "wrong-handshake":
		_, _ = os.Stdout.Write(bytes.Replace(responseBytes,
			[]byte("temporal.nexus.caller-closure.run-evaluation"),
			[]byte("temporal.checker.substituted"), 1))
	case mode == "wrong-version":
		_, _ = os.Stdout.Write(bytes.Replace(responseBytes,
			[]byte("\"checkerVersion\": 2"), []byte("\"checkerVersion\": 3"), 1))
	case mode == "stale-response":
		_, _ = os.Stdout.Write(bytes.Replace(responseBytes,
			[]byte("sha256:3333333333333333333333333333333333333333333333333333333333333333"),
			[]byte("sha256:9999999999999999999999999999999999999999999999999999999999999999"), 1))
	case mode == "malformed":
		_, _ = io.WriteString(os.Stdout, "{\n  \"formatVersion\":\n")
	case mode == "noncanonical":
		var compact bytes.Buffer
		_ = json.Compact(&compact, responseBytes)
		_ = compact.WriteByte('\n')
		_, _ = os.Stdout.Write(compact.Bytes())
	case mode == "trailing":
		_, _ = os.Stdout.Write(responseBytes)
		_, _ = io.WriteString(os.Stdout, " ")
	case mode == "nonzero":
		os.Exit(7)
	case mode == "stderr":
		_, _ = io.WriteString(os.Stderr, "fixture stderr must be sanitized")
		time.Sleep(10 * time.Second)
	case mode == "oversized":
		chunk := make([]byte, 32<<10)
		remaining := maximumBytes + 1
		for remaining > 0 {
			written, writeErr := os.Stdout.Write(chunk[:min(remaining, len(chunk))])
			if writeErr != nil {
				return
			}
			remaining -= written
		}
	case mode == "sleep":
		time.Sleep(10 * time.Second)
	default:
		os.Exit(94)
	}
}
