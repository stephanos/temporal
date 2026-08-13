package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"go.temporal.io/server/tools/gomadv3/internal/upgrade"
)

func main() {
	root := flag.String("root", ".", "Gomad v3 module root")
	output := flag.String("output", ".toolchain/upgrade-dossier.json", "qualification dossier output")
	baselineRef := flag.String("baseline-ref", "", "Git revision containing the baseline boundary manifest")
	approvedBoundaryDiff := flag.String("approve-boundary-diff", "", "approve the exact canonical boundary diff SHA-256")
	corpusReport := flag.String("corpus-report", "", "retained-corpus qualification report")
	flag.Parse()

	absoluteRoot, err := filepath.Abs(*root)
	if err != nil {
		fail(err)
	}
	outputPath := *output
	if !filepath.IsAbs(outputPath) {
		outputPath = filepath.Join(absoluteRoot, filepath.FromSlash(outputPath))
	}
	var baseline []byte
	if *baselineRef != "" {
		var found bool
		baseline, found, err = gitFile(context.Background(), absoluteRoot, *baselineRef, "boundary/manifest.json")
		if err != nil {
			fail(err)
		}
		if !found {
			baseline = []byte(`{"manifest_version":"absent","intercepts":[]}`)
		}
	}
	gates := []upgrade.Gate{
		{Name: "manifest-validation", Command: []string{"make", "validate"}},
		{Name: "patch-and-compiler", Command: []string{"make", "patch-test", "intercept-test"}},
		{Name: "runner-world-and-probes", Command: []string{"make", "runner-test", "overlay-test", "world-test"}},
		{Name: "builder", Command: []string{"./test.sh", "test-builder"}},
		{Name: "runtime", Command: []string{"./test.sh", "test-runtime"}},
		{Name: "disabled-upstream", Command: []string{"./test.sh", "test-upstream"}},
		{Name: "host-clock-escape", Command: []string{"make", "clock-audit"}},
		{Name: "cached-toolchain-build", Command: []string{"make", "toolchain"}},
	}
	err = upgrade.Run(context.Background(), upgrade.Options{
		Root: absoluteRoot, Output: outputPath, BaselineManifest: baseline,
		ApprovedBoundaryDiffSHA256: *approvedBoundaryDiff,
		CorpusReport:               *corpusReport, Gates: gates, Writer: os.Stdout,
	})
	if err != nil {
		fail(err)
	}
	fmt.Printf("gomadv3 upgrade qualification dossier: %s\n", outputPath)
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

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
