package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestRunAdapterRegenerateInputStatuses(t *testing.T) {
	root := t.TempDir()
	for _, test := range []struct {
		name      string
		arguments []string
		status    int
		stderr    string
	}{
		{name: "usage", arguments: []string{"--root", root}, status: 2, stderr: "regenerable adapters: "},
		{name: "unknown module", arguments: []string{"--root", root, "--module=example.com/none", "--version=v1.0.0"}, status: 2, stderr: "no adapter is pinned for example.com/none"},
		{name: "verify usage", arguments: []string{"--verify", "--module=google.golang.org/grpc"}, status: 2, stderr: "usage:"},
		{name: "recover nothing", arguments: []string{"--root", root, "--recover"}, status: 0},
		{name: "conflicting approval aliases", arguments: []string{"--approve=a", "--approve-review=b"}, status: 2, stderr: "must name the same digest"},
		{name: "stage requires approval", arguments: []string{"--stage-only"}, status: 2, stderr: "requires an approval digest"},
	} {
		t.Run(test.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			if status := run(append([]string{"adapter-regenerate"}, test.arguments...), &stdout, &stderr); status != test.status || !strings.Contains(stderr.String(), test.stderr) {
				t.Fatalf("status = %d, stderr = %q; want %d with %q", status, stderr.String(), test.status, test.stderr)
			}
		})
	}
}
