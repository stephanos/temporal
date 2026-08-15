package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
	"strings"

	"go.temporal.io/server/tools/gomadv3/toolchain"
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
	gates := []toolchain.UpgradeGate{
		{Name: "manifest-validation", Command: []string{"make", "validate"}},
		{Name: "patch-and-compiler", Command: []string{"make", "patch-test", "intercept-test"}},
		{Name: "runner-world-and-probes", Command: []string{"make", "runner-test", "overlay-test", "world-test"}},
		{Name: "builder", Command: []string{"./test.sh", "test-builder"}},
		{Name: "runtime", Command: []string{"./test.sh", "test-runtime"}},
		{Name: "disabled-upstream", Command: []string{"./test.sh", "test-upstream"}},
		{Name: "host-clock-escape", Command: []string{"make", "clock-audit"}},
		{Name: "cached-toolchain-build", Command: []string{"make", "toolchain"}},
	}
	err = toolchain.Upgrade(context.Background(), toolchain.UpgradeSpec{
		Root: absoluteRoot, Output: outputPath, BaselineManifest: baseline,
		ApprovedBoundaryDiffSHA256: *approvedBoundaryDiff,
		CorpusReport:               *corpusReport, Gates: gates, Writer: stdout,
	})
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	fmt.Fprintf(stdout, "gomadv3 upgrade qualification dossier: %s\n", outputPath)
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
