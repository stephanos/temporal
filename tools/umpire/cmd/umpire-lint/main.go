// Command umpire-lint lints IR files: it prints each file's findings, accepted or not, and its
// coverage summary, and fails on a finding no acceptance matches and on an acceptance that matches
// no finding. A count never fails it.
//
//	umpire-lint [--update] [--must-not-pinned] [--tables] [ir files...]
//
// With no file it lints every IR file of model/ir, run from the repository root as the model gate
// runs it. The accepted findings of `<file>.json` are in `<file>.lint.json` beside it, each with
// its reason; one beside no IR file accepts nothing and fails the run.
//
// The reasons of the laws a file's capability declarations waive are in its law sidecar,
// `<file>.laws.json`, and forwarded into its accepted findings: --update writes them there, as the
// model gate's update does, and a run without it fails on a file that does not carry them. A law's
// instantiating machines are counted across the law sidecars of every directory the run lints.
package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	// The realizations of model/ir read Temporal API messages by name; the workflow service links
	// every one of them, and lowering links the rest.
	_ "go.temporal.io/api/workflowservice/v1"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	"go.temporal.io/server/tools/umpire/internal/cli"
	"go.temporal.io/server/tools/umpire/lint"
	"go.temporal.io/server/tools/umpire/lower"
	umpiremodel "go.temporal.io/server/tools/umpire/model"
)

const irDirectory = "model/ir"

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// lowering is what lint reads of lowering: the manifest's `no-realization` standing, and the
// descriptors of the evidence a realization reads.
var lowering = lint.Lowering{
	Unrealized: func(q *umpirespb.Query, scenario *umpirespb.Scenario, realizations []*umpirespb.Realization) (bool, error) {
		standing, _, err := lower.Realizable(q, scenario, realizations)
		return standing == lower.NoRealization, err
	},
	Element: lower.EvidenceElement,
	Field:   lower.FieldAt,
}

// tally is what a run found across its files.
type tally struct {
	unaccepted, stale, errors int
}

func (t tally) failed() bool { return t.unaccepted+t.stale+t.errors > 0 }

// run lints as the command line says and answers the exit status: 0 when nothing failed, 1 when a
// file has an unaccepted finding, a stale acceptance or an error, and 2 on bad usage.
func run(arguments []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("umpire-lint", flag.ContinueOnError)
	flags.SetOutput(stderr)
	flags.Usage = func() {
		cli.WriteLine(stderr, "usage: umpire-lint [--update] [--must-not-pinned] [--tables] [ir files...]")
		flags.PrintDefaults()
	}
	mustNotPinned := flags.Bool("must-not-pinned", false, "also report each disabled pair of a system action no claim pins (H5)")
	tables := flags.Bool("tables", false, "also print each machine's per-operation modality table")
	update := flags.Bool("update", false, "forward each law sidecar's waivers into the accepted findings beside it")
	if err := flags.Parse(arguments); err != nil {
		return 2
	}
	var t tally
	paths, directories, err := irFiles(flags.Args())
	if err != nil {
		cli.WriteLine(stderr, "umpire-lint: %v", err)
		return 1
	}
	options := lint.Options{MustNotPinned: *mustNotPinned, Instances: instances(directories)}
	for _, path := range paths {
		if err := lintFile(path, options, *tables, *update, stdout, &t); err != nil {
			t.errors++
			report(stderr, path, err)
		}
	}
	for _, directory := range directories {
		orphans, err := orphans(directory)
		if err != nil {
			t.errors++
			cli.WriteLine(stderr, "umpire-lint: %v", err)
		}
		for _, orphan := range orphans {
			t.errors++
			cli.WriteLine(stderr, "%s accepts findings of %s, which does not exist: remove it, or rename it beside its IR file",
				orphan, ir(orphan))
		}
	}
	if !t.failed() {
		return 0
	}
	cli.WriteLine(stderr, "umpire-lint: %d unaccepted findings, %d stale acceptances and %d errors; fix the Model, "+
		"or accept a finding with a reason in <file>.lint.json beside its IR file, and remove each stale acceptance",
		t.unaccepted, t.stale, t.errors)
	return 1
}

// irFiles is the IR files a run lints, sorted, and the directories whose acceptance files it holds
// to an IR file: model/ir and every IR file of it, or the files named and their directories.
func irFiles(named []string) (paths, directories []string, err error) {
	if len(named) == 0 {
		paths, err := umpiremodel.IRPaths(irDirectory)
		if err != nil {
			return nil, nil, err
		}
		// A gate that lints nothing must not pass: model/ir is read from the repository root.
		if len(paths) == 0 {
			return nil, nil, fmt.Errorf("no IR file in %s; run umpire-lint from the repository root", irDirectory)
		}
		return paths, []string{irDirectory}, nil
	}
	paths = slices.Compact(slices.Sorted(slices.Values(named)))
	for _, path := range paths {
		directories = append(directories, filepath.Dir(path))
	}
	return paths, slices.Compact(slices.Sorted(slices.Values(directories))), nil
}

// lintFile writes the findings and coverage of one IR file, and its tables when asked. A file the
// reader refuses writes nothing: its error is the reader's. Its accepted findings carry the waivers
// of its law sidecar, written there by an update; a file that does not is an error of a check, and
// is judged as it is.
func lintFile(path string, options lint.Options, tables, update bool, stdout io.Writer, t *tally) error {
	m, err := lint.Read(path, lowering, options)
	if err != nil {
		return err
	}
	result, err := m.Lint()
	if err != nil {
		return err
	}
	accepted, err := lint.ReadAccepted(path)
	if err != nil {
		return err
	}
	unforwarded, err := forward(path, accepted, lint.Forward(accepted, m.Laws), update)
	if err != nil {
		return err
	}
	if !unforwarded {
		accepted = lint.Forward(accepted, m.Laws)
	}
	verdict := accepted.Judge(result.Findings())
	t.unaccepted += len(verdict.Unaccepted)
	t.stale += len(verdict.Stale)
	if err := lint.WriteFindings(stdout, path, verdict); err != nil {
		return err
	}
	if err := lint.WriteCoverage(stdout, result); err != nil {
		return err
	}
	if tables {
		if err := lint.WriteTables(stdout, result); err != nil {
			return err
		}
	}
	if unforwarded {
		return fmt.Errorf("%s does not carry the law waivers of %s: rerun make umpire-gen-model, whose update forwards them (umpire-lint --update)",
			lint.AcceptedPath(path), umpiremodel.LawSidecarPath(path))
	}
	return nil
}

// forward holds the accepted findings of the IR file at path to the forwarded ones: an update writes
// them where they differ, and reports nothing; a check reports whether they differ. A file that
// would accept nothing is not created.
func forward(path string, accepted, forwarded *lint.Accepted, update bool) (bool, error) {
	was, err := accepted.Encode()
	if err != nil {
		return false, err
	}
	now, err := forwarded.Encode()
	if err != nil {
		return false, err
	}
	switch {
	case bytes.Equal(was, now):
		return false, nil
	case !update:
		return true, nil
	default:
		return false, os.WriteFile(lint.AcceptedPath(path), now, 0o644)
	}
}

// instances counts each law's instantiating machines across the law sidecars of the directories a
// run lints. A sidecar the reader refuses counts nothing here: linting its IR file reports it.
func instances(directories []string) lint.Instances {
	var sidecars []*umpiremodel.LawSidecar
	for _, directory := range directories {
		paths, err := umpiremodel.IRPaths(directory)
		if err != nil {
			continue
		}
		for _, path := range paths {
			if s, err := umpiremodel.ReadLawSidecar(path); err == nil {
				sidecars = append(sidecars, s)
			}
		}
	}
	return lint.CountInstances(sidecars...)
}

// report writes an error of one file, naming the file once and indenting each further line of a
// joined error. The reader positions a malformed file's error at the file already.
func report(stderr io.Writer, path string, err error) {
	text := err.Error()
	if !strings.HasPrefix(text, path+":") {
		text = path + ": " + text
	}
	cli.WriteLine(stderr, "%s", strings.ReplaceAll(text, "\n", "\n  "))
}

// orphans is every acceptance file of a directory beside which there is no IR file.
func orphans(directory string) ([]string, error) {
	accepted, err := filepath.Glob(filepath.Join(directory, "*"+lint.AcceptedSuffix))
	if err != nil {
		return nil, err
	}
	var out []string
	for _, path := range accepted {
		if _, err := os.Stat(ir(path)); errors.Is(err, fs.ErrNotExist) {
			out = append(out, path)
		} else if err != nil {
			return out, err
		}
	}
	return out, nil
}

// ir is the IR file an acceptance file is beside, the inverse of lint.AcceptedPath.
func ir(accepted string) string { return strings.TrimSuffix(accepted, lint.AcceptedSuffix) + ".json" }
