// Command protocolgen writes or checks gomad's shared generated wire codecs.
//
// Usage: go run ./internal/model/cmd/protocolgen [-check] [-root DIR]
//
// Exit status is 0 on success, 1 when -check finds stale outputs (one
// "stale: <path>" line each), and 2 on a usage, schema or template error.
package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"go.temporal.io/server/tools/gomad/internal/model/generation/protocol"
)

const modulePath = "go.temporal.io/server/tools/gomad"

func main() {
	os.Exit(run(os.Args[1:], os.Stderr))
}

func run(arguments []string, stderr io.Writer) int {
	flags := flag.NewFlagSet("protocolgen", flag.ContinueOnError)
	flags.SetOutput(stderr)
	check := flags.Bool("check", false, "report stale outputs without writing")
	root := flags.String("root", "", "gomad module root (default: nearest enclosing gomad module)")
	if err := flags.Parse(arguments); err != nil {
		return 2
	}
	if flags.NArg() != 0 {
		_, _ = fmt.Fprintf(stderr, "protocolgen: unexpected arguments %q\n", flags.Args())
		return 2
	}
	directory := *root
	if directory == "" {
		working, err := os.Getwd()
		if err != nil {
			_, _ = fmt.Fprintln(stderr, "protocolgen:", err)
			return 2
		}
		directory, err = moduleRoot(working)
		if err != nil {
			_, _ = fmt.Fprintln(stderr, "protocolgen:", err)
			return 2
		}
	}
	stale, err := protocol.Generate(directory, directory, protocol.SharedTargets(), protocol.SharedHeaders, *check)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, "protocolgen:", err)
		return 2
	}
	for _, path := range stale {
		_, _ = fmt.Fprintf(stderr, "stale: %s\n", path)
	}
	if len(stale) != 0 {
		return 1
	}
	return 0
}

// moduleRoot walks up from directory to the go.mod that declares the gomad
// module, skipping nested modules such as the native backend.
func moduleRoot(directory string) (string, error) {
	for {
		contents, err := os.ReadFile(filepath.Join(directory, "go.mod"))
		if err == nil && bytes.HasPrefix(contents, []byte("module "+modulePath+"\n")) {
			return directory, nil
		}
		parent := filepath.Dir(directory)
		if parent == directory {
			return "", errors.New("gomad module root not found")
		}
		directory = parent
	}
}
