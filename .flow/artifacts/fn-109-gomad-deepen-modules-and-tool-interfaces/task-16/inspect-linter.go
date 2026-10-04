//go:build ignore

package main

import (
	"crypto/sha256"
	"debug/macho"
	"encoding/json"
	"fmt"
	"os"
	"runtime"
)

func main() {
	if len(os.Args) != 2 {
		os.Exit(2)
	}
	file, err := macho.Open(os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	defer file.Close()
	data, err := os.ReadFile(os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	if err := json.NewEncoder(os.Stdout).Encode(map[string]any{
		"path": os.Args[1], "format": "Mach-O", "cpu": file.Cpu.String(), "cpu_value": uint32(file.Cpu),
		"magic": fmt.Sprintf("%08x", file.Magic), "sha256": fmt.Sprintf("%x", sha256.Sum256(data)),
		"host_os": runtime.GOOS, "host_arch": runtime.GOARCH, "go": runtime.Version(),
	}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
}
