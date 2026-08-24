package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
	"strings"

	"go.temporal.io/server/tools/gomad3/upgrade"
)

func runUpgradeDossier(arguments []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("gomadtool upgrade-dossier", flag.ContinueOnError)
	flags.SetOutput(stderr)
	root := flags.String("root", ".", "Gomad v3 module root")
	output := flags.String("output", ".toolchain/upgrade-dossier.json", "qualification dossier output")
	baselineRef := flags.String("baseline-ref", "", "Git revision containing the baseline boundary manifest")
	approvedBoundaryDiff := flags.String("approve-boundary-diff", "", "approve the exact canonical boundary diff SHA-256")
	corpusReport := flags.String("corpus-report", "", "retained-corpus qualification report")
	if err := flags.Parse(arguments); err != nil {
		return 2
	}

	absoluteRoot, err := filepath.Abs(*root)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	outputPath := *output
	if !filepath.IsAbs(outputPath) {
		outputPath = filepath.Join(absoluteRoot, filepath.FromSlash(outputPath))
	}
	var baseline []byte
	if *baselineRef != "" {
		var found bool
		baseline, found, err = gitFile(context.Background(), absoluteRoot, *baselineRef, "deterministicio/boundary/manifest.json")
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		if !found {
			baseline = []byte(`{"manifest_version":"absent","intercepts":[]}`)
		}
	}
	gates := []upgrade.Gate{
		{Name: "manifest-validation", Command: []string{"make", "validate"}},
		{Name: "toolchain-and-compiler", Command: []string{"make", "test-toolchain", "intercept-test"}},
		{Name: "host-world-and-probes", Command: []string{"make", "test-host", "overlay-test", "world-test"}},
		{Name: "builder", Command: []string{"make", "test-builder"}},
		{Name: "runtime", Command: []string{"make", "test-runtime"}},
		{Name: "disabled-upstream", Command: []string{"make", "test-upstream"}},
		{Name: "host-clock-escape", Command: []string{"make", "clock-audit"}},
		{Name: "cached-toolchain-build", Command: []string{"make", "toolchain"}},
	}
	err = upgrade.Run(context.Background(), upgrade.Spec{
		Root: absoluteRoot, Output: outputPath, BaselineManifest: baseline,
		ApprovedBoundaryDiffSHA256: *approvedBoundaryDiff,
		CorpusReport:               *corpusReport, Gates: gates, Writer: stdout,
	})
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	fmt.Fprintf(stdout, "gomad3 upgrade qualification dossier: %s\n", outputPath)
	return 0
}

func gitFile(ctx context.Context, root, revision, relative string) ([]byte, bool, error) {
	prefixCommand := exec.CommandContext(ctx, "git", "-C", root, "rev-parse", "--show-prefix")
	prefix, err := prefixCommand.Output()
	if err != nil {
		return nil, false, fmt.Errorf("locate Gomad v3 Git prefix: %w", err)
	}
	path := filepath.ToSlash(filepath.Join(strings.TrimSpace(string(prefix)), filepath.FromSlash(relative)))
	revisionCommand := exec.CommandContext(ctx, "git", "-C", root, "cat-file", "-e", revision+"^{commit}")
	if err := revisionCommand.Run(); err != nil {
		return nil, false, fmt.Errorf("resolve baseline %s: %w", revision, err)
	}
	command := exec.CommandContext(ctx, "git", "-C", root, "show", revision+":"+path)
	contents, err := command.Output()
	if err != nil {
		pathCommand := exec.CommandContext(ctx, "git", "-C", root, "cat-file", "-e", revision+":"+path)
		if pathCommand.Run() != nil {
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("read %s from baseline %s: %w", relative, revision, err)
	}
	return contents, true, nil
}
