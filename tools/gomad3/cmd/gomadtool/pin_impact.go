package main

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"go.temporal.io/server/tools/gomad3/internal/hostfs"
	"go.temporal.io/server/tools/gomad3/upgrade"
	"go.temporal.io/server/tools/gomad3/upgrade/pinimpact"
)

const pinImpactUsage = "usage: gomadtool pin-impact [--root=DIR] [--module=DIR] [--baseline-module=DIR | --baseline-ref=REV] [--go=GO] [--json] [--output=FILE]\n" +
	"       gomadtool pin-impact [--root=DIR] --baseline=GO.MOD --candidate=GO.MOD [--format=human|json]"

// runPinImpact exits 0 when the candidate invalidates no pin, 1 when it
// invalidates or leaves unknown at least one, 2 for invalid input, and 3 for
// infrastructure failures.
func runPinImpact(arguments []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("gomadtool pin-impact", flag.ContinueOnError)
	flags.SetOutput(stderr)
	root := flags.String("root", ".", "Gomad v3 module root")
	moduleDirectory := flags.String("module", "", "candidate module directory (default: the repository root module)")
	baselineModule := flags.String("baseline-module", "", "baseline module directory")
	baselineRef := flags.String("baseline-ref", "", "Git revision holding the baseline go.mod and go.sum (default: HEAD)")
	baselinePath := flags.String("baseline", "", "baseline go.mod path, with adjacent go.sum")
	candidatePath := flags.String("candidate", "", "candidate go.mod path, with adjacent go.sum")
	format := flags.String("format", "human", "human or json output")
	goCommand := flags.String("go", os.Getenv("GOMAD3_BOOTSTRAP_GO"), "go command that resolves module graphs")
	jsonOutput := flags.Bool("json", false, "write the canonical JSON report to stdout")
	output := flags.String("output", "", "also write the canonical JSON report to this file")
	if err := flags.Parse(arguments); err != nil || flags.NArg() != 0 || *baselineModule != "" && *baselineRef != "" || *format != "human" && *format != "json" {
		if _, writeErr := fmt.Fprintln(stderr, pinImpactUsage); writeErr != nil {
			return 2
		}
		return 2
	}
	if *baselinePath != "" || *candidatePath != "" {
		if *baselineModule != "" || *baselineRef != "" || *moduleDirectory != "" || *baselinePath == "" || *candidatePath == "" {
			if _, writeErr := fmt.Fprintln(stderr, pinImpactUsage); writeErr != nil {
				return 2
			}
			return 2
		}
		return runFilePinImpact(*root, *candidatePath, *baselinePath, *format == "json" || *jsonOutput, *output, stdout, stderr)
	}
	if *format == "json" {
		*jsonOutput = true
	}
	ctx := context.Background()
	absoluteRoot, err := filepath.Abs(*root)
	if err != nil {
		if _, writeErr := fmt.Fprintln(stderr, err); writeErr != nil {
			return 2
		}
		return 2
	}
	candidateDirectory := *moduleDirectory
	if candidateDirectory == "" {
		candidateDirectory, err = gitOutput(ctx, absoluteRoot, "rev-parse", "--show-toplevel")
		if err != nil {
			if _, writeErr := fmt.Fprintf(stderr, "locate the repository root module: %v; pass --module\n", err); writeErr != nil {
				return 2
			}
			return 2
		}
	}
	candidateDirectory, err = filepath.Abs(candidateDirectory)
	if err != nil {
		if _, writeErr := fmt.Fprintln(stderr, err); writeErr != nil {
			return 2
		}
		return 2
	}
	candidate, err := readModuleFiles(candidateDirectory)
	if err != nil {
		if _, writeErr := fmt.Fprintln(stderr, err); writeErr != nil {
			return 2
		}
		return 2
	}
	var baseline pinimpact.ModuleFiles
	if *baselineModule != "" {
		directory, absErr := filepath.Abs(*baselineModule)
		if absErr == nil {
			baseline, err = readModuleFiles(directory)
		}
		err = errors.Join(absErr, err)
	} else {
		revision := *baselineRef
		if revision == "" {
			revision = "HEAD"
		}
		baseline, err = gitModuleFiles(ctx, candidateDirectory, revision)
	}
	if err != nil {
		if _, writeErr := fmt.Fprintln(stderr, err); writeErr != nil {
			return 2
		}
		return 2
	}
	if *goCommand == "" {
		*goCommand = "go"
	}
	// A bare name such as "go" is searched on PATH rather than taken
	// relative to the working directory.
	*goCommand, err = exec.LookPath(*goCommand)
	if err != nil {
		if _, writeErr := fmt.Fprintf(stderr, "gomad3 pin impact requires a go command; set GOMAD3_BOOTSTRAP_GO or pass --go: %v\n", err); writeErr != nil {
			return 3
		}
		return 3
	}
	absoluteGo, err := filepath.Abs(*goCommand)
	if err != nil {
		if _, writeErr := fmt.Fprintln(stderr, err); writeErr != nil {
			return 2
		}
		return 2
	}
	resolver, err := pinimpact.NewGoResolver(absoluteGo, os.Environ())
	if err != nil {
		if _, writeErr := fmt.Fprintln(stderr, err); writeErr != nil {
			return 3
		}
		return 3
	}
	report, err := pinimpact.Evaluate(ctx, pinimpact.Spec{Root: absoluteRoot, Baseline: baseline, Candidate: candidate, Resolver: resolver})
	err = errors.Join(err, resolver.Close())
	if err == nil {
		err = requireUnchangedModule(candidateDirectory, candidate)
	}
	if err != nil {
		if _, writeErr := fmt.Fprintln(stderr, err); writeErr != nil {
			if pinimpact.IsInputError(err) {
				return 2
			}
			return 3
		}
		if pinimpact.IsInputError(err) {
			return 2
		}
		return 3
	}
	encoded, err := pinimpact.Encode(report)
	if err != nil {
		if _, writeErr := fmt.Fprintln(stderr, err); writeErr != nil {
			return 3
		}
		return 3
	}
	if *output != "" {
		if err := hostfs.Replace(*output, encoded, 0o644); err != nil {
			if _, writeErr := fmt.Fprintln(stderr, err); writeErr != nil {
				return 3
			}
			return 3
		}
	}
	if *jsonOutput {
		_, err = stdout.Write(encoded)
	} else {
		err = pinimpact.Render(stdout, report)
	}
	if err != nil {
		if _, writeErr := fmt.Fprintln(stderr, err); writeErr != nil {
			return 3
		}
		return 3
	}
	if report.Invalidated {
		return 1
	}
	return 0
}

func runFilePinImpact(root, candidatePath, baselinePath string, jsonOutput bool, output string, stdout, stderr io.Writer) int {
	report, err := upgrade.ReadPinImpact(root, candidatePath, baselinePath)
	if err != nil {
		if _, writeErr := fmt.Fprintln(stderr, err); writeErr != nil {
			var input *upgrade.InvalidPinImpactInput
			if errors.As(err, &input) {
				return 2
			}
			return 3
		}
		var input *upgrade.InvalidPinImpactInput
		if errors.As(err, &input) {
			return 2
		}
		return 3
	}
	encoded, err := report.CanonicalJSON()
	if err != nil {
		if _, writeErr := fmt.Fprintln(stderr, err); writeErr != nil {
			return 3
		}
		return 3
	}
	encoded = append(encoded, '\n')
	if output != "" {
		if err := hostfs.Replace(output, encoded, 0o644); err != nil {
			if _, writeErr := fmt.Fprintln(stderr, err); writeErr != nil {
				return 3
			}
			return 3
		}
	}
	if jsonOutput {
		_, err = stdout.Write(encoded)
	} else {
		for _, pin := range report.Pins {
			_, err = fmt.Fprintf(stdout, "%s %s: %s (%s)\n", pin.Class, pin.ID, pin.Status, pin.Reason)
			if err != nil {
				break
			}
		}
	}
	if err != nil {
		if _, writeErr := fmt.Fprintln(stderr, err); writeErr != nil {
			return 3
		}
		return 3
	}
	if report.Invalidated() {
		return 1
	}
	return 0
}

func readModuleFiles(directory string) (pinimpact.ModuleFiles, error) {
	goMod, err := os.ReadFile(filepath.Join(directory, "go.mod"))
	if err != nil {
		return pinimpact.ModuleFiles{}, fmt.Errorf("read module file: %w", err)
	}
	goSum, err := os.ReadFile(filepath.Join(directory, "go.sum"))
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return pinimpact.ModuleFiles{}, fmt.Errorf("read module sums: %w", err)
	}
	return pinimpact.ModuleFiles{GoMod: goMod, GoSum: goSum, Directory: directory}, nil
}

// gitModuleFiles reads directory's go.mod and go.sum as of revision. Relative
// local replacements resolve against the checked-out directory.
func gitModuleFiles(ctx context.Context, directory, revision string) (pinimpact.ModuleFiles, error) {
	prefix, err := gitOutput(ctx, directory, "rev-parse", "--show-prefix")
	if err != nil {
		return pinimpact.ModuleFiles{}, fmt.Errorf("locate baseline module in Git: %w; pass --baseline-module", err)
	}
	if _, err := gitOutput(ctx, directory, "rev-parse", "--verify", "--quiet", revision+"^{commit}"); err != nil {
		return pinimpact.ModuleFiles{}, fmt.Errorf("resolve baseline revision %s: %w", revision, err)
	}
	files := pinimpact.ModuleFiles{Directory: directory}
	for _, name := range []string{"go.mod", "go.sum"} {
		object := revision + ":" + prefix + name
		if _, err := gitOutput(ctx, directory, "cat-file", "-e", object); err != nil {
			if name == "go.sum" {
				continue
			}
			return pinimpact.ModuleFiles{}, fmt.Errorf("baseline revision %s has no %s%s; pass --baseline-module", revision, prefix, name)
		}
		command := exec.CommandContext(ctx, "git", "-C", directory, "show", object)
		contents, err := command.Output()
		if err != nil {
			return pinimpact.ModuleFiles{}, fmt.Errorf("read baseline %s: %w", object, err)
		}
		if name == "go.mod" {
			files.GoMod = contents
		} else {
			files.GoSum = contents
		}
	}
	return files, nil
}

func gitOutput(ctx context.Context, directory string, arguments ...string) (string, error) {
	command := exec.CommandContext(ctx, "git", append([]string{"-C", directory}, arguments...)...)
	output, err := command.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSuffix(string(output), "\n"), nil
}

// requireUnchangedModule fails when the run changed the candidate's files,
// which would mean resolution escaped its scratch copy.
func requireUnchangedModule(directory string, before pinimpact.ModuleFiles) error {
	after, err := readModuleFiles(directory)
	if err != nil {
		return err
	}
	if !bytes.Equal(before.GoMod, after.GoMod) || !bytes.Equal(before.GoSum, after.GoSum) {
		return errors.New("gomad3 pin impact changed the candidate module's go.mod or go.sum")
	}
	return nil
}
