package backends

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// Tool is an external checker's command line. A backend's agreement is run only with its tool; with
// none there is nothing to report but that it was not run.
type Tool struct {
	Name    string
	Command string
}

// QuintTool is the Quint command line: the executable UMPIRE_QUINT names, or `quint` on the path.
func QuintTool() (Tool, bool) { return tool("quint", "UMPIRE_QUINT") }

// PTool is the P command line: the executable UMPIRE_P names, or `p` on the path. P runs on .NET,
// which it finds by DOTNET_ROOT or on the path.
func PTool() (Tool, bool) { return tool("p", "UMPIRE_P") }

func tool(name, variable string) (Tool, bool) {
	command := os.Getenv(variable)
	if command == "" {
		command = name
	}
	found, err := exec.LookPath(command)
	if err != nil {
		return Tool{Name: name}, false
	}
	return Tool{Name: name, Command: found}, true
}

func (t Tool) run(ctx context.Context, dir string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, t.Command, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return out, fmt.Errorf("%s %v: %w\n%s", t.Name, args, err, out)
	}
	return out, nil
}

// RunQuint writes an export into a directory and has Quint evaluate it: one run of one step, which
// takes the module's initializer and writes the dump as an ITF trace. Nothing is sampled: the module
// has one behavior, and its dump is computed by pure definitions.
func RunQuint(ctx context.Context, t Tool, x *QuintExport, dir string) ([]byte, error) {
	source := filepath.Join(dir, x.Module+".qnt")
	if err := os.WriteFile(source, []byte(x.Text), 0o644); err != nil {
		return nil, err
	}
	dump := filepath.Join(dir, x.Module+".itf.json")
	if _, err := t.run(ctx, dir, "run", source, "--main", x.Module, "--backend", "typescript", "--max-samples", "1", "--max-steps", "1",
		"--seed", "0x1", "--out-itf", dump, "--verbosity", "1"); err != nil {
		return nil, err
	}
	return os.ReadFile(dump)
}
