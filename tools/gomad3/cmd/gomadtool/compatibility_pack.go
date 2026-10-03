package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"time"

	"go.temporal.io/server/tools/gomad3/internal/compatibilitypack/authoring"
	capabilityanalysis "go.temporal.io/server/tools/gomad3/qualification/analysis"
)

const compatibilityPackTimeout = 2 * time.Minute

func runCompatibilityPack(arguments []string, stdout, stderr io.Writer) int {
	if len(arguments) == 0 {
		fmt.Fprintln(stderr, "usage: gomadtool compatibility-pack discover|review|generate|check|qualify|refresh [flags]")
		return 2
	}
	switch arguments[0] {
	case "discover":
		return runCompatibilityPackDiscover(arguments[1:], stdout, stderr)
	case "review":
		return runCompatibilityPackReview(arguments[1:], stdout, stderr)
	case "generate":
		return runCompatibilityPackGenerate(arguments[1:], stdout, stderr)
	case "check":
		return runCompatibilityPackCheck(arguments[1:], stdout, stderr)
	case "qualify":
		return runCompatibilityPackQualify(arguments[1:], stdout, stderr)
	case "refresh":
		return runCompatibilityPackRefresh(arguments[1:], stdout, stderr)
	default:
		fmt.Fprintln(stderr, "usage: gomadtool compatibility-pack discover|review|generate|check|qualify|refresh [flags]")
		return 2
	}
}

func runCompatibilityPackDiscover(arguments []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("gomadtool compatibility-pack discover", flag.ContinueOnError)
	flags.SetOutput(stderr)
	root := flags.String("root", "", "Gomad v3 module root")
	compatibilityRootOverride := flags.String("compatibility-root", "", "absolute pack authoring root owned by another module (default: internal/compatibilitypack)")
	requestPath := flags.String("request", "", "compatibility-pack request path")
	workingDirectory := flags.String("working-dir", "", "target working directory")
	if err := flags.Parse(arguments); err != nil || flags.NArg() != 0 || *root == "" || *requestPath == "" || *workingDirectory == "" {
		return 2
	}
	resolvedRoot, compatibilityRoot, resolvedRequest, err := resolveCompatibilityPackPaths(*root, *compatibilityRootOverride, *requestPath)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	draftBytes, err := readCompatibilityPackFile(resolvedRequest, authoring.MaximumRequestBytes)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	draft, err := authoring.DecodeDraftRequest(draftBytes)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	ctx, cancel := context.WithTimeout(context.Background(), compatibilityPackTimeout)
	defer cancel()
	prepared, err := capabilityanalysis.PrepareCapabilityReview(
		ctx,
		draft.ReviewSpec(*workingDirectory, filepath.Join(resolvedRoot, ".toolchain")),
	)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	discovered, digest, err := authoring.Discover(draft, prepared.Review)
	err = errors.Join(err, prepared.Close())
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if !pathWithin(compatibilityRoot, resolvedRequest) {
		fmt.Fprintln(stderr, "compatibility-pack request must be below internal/compatibilitypack")
		return 2
	}
	if err := authoring.PublishRequest(resolvedRequest, discovered); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	fmt.Fprintln(stdout, digest)
	return 0
}

func runCompatibilityPackReview(arguments []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("gomadtool compatibility-pack review", flag.ContinueOnError)
	flags.SetOutput(stderr)
	root := flags.String("root", "", "Gomad v3 module root")
	compatibilityRootOverride := flags.String("compatibility-root", "", "absolute pack authoring root owned by another module (default: internal/compatibilitypack)")
	requestPath := flags.String("request", "", "compatibility-pack request path")
	outputPath := flags.String("output", "", "review report path")
	if err := flags.Parse(arguments); err != nil || flags.NArg() != 0 || *root == "" || *requestPath == "" || *outputPath == "" {
		return 2
	}
	_, compatibilityRoot, resolvedRequest, err := resolveCompatibilityPackPaths(*root, *compatibilityRootOverride, *requestPath)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	resolvedOutput, err := resolveBelow(compatibilityPathBase(*root, compatibilityRoot, *compatibilityRootOverride), *outputPath)
	if err != nil || !pathWithin(compatibilityRoot, resolvedOutput) {
		fmt.Fprintln(stderr, "compatibility-pack review output must be below internal/compatibilitypack")
		return 2
	}
	request, status := readReviewedCompatibilityPackRequest(resolvedRequest, stderr)
	if status != 0 {
		return status
	}
	digest, err := authoring.PublishReview(resolvedOutput, request)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	fmt.Fprintln(stdout, digest)
	return 0
}

func runCompatibilityPackGenerate(arguments []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("gomadtool compatibility-pack generate", flag.ContinueOnError)
	flags.SetOutput(stderr)
	root := flags.String("root", "", "Gomad v3 module root")
	compatibilityRootOverride := flags.String("compatibility-root", "", "absolute pack authoring root owned by another module (default: internal/compatibilitypack)")
	requestPath := flags.String("request", "", "compatibility-pack request path")
	approval := flags.String("approve-review", "", "exact canonical review SHA-256")
	if err := flags.Parse(arguments); err != nil || flags.NArg() != 0 || *root == "" {
		return 2
	}
	if *requestPath == "" && *approval == "" {
		resolvedRoot, err := filepath.Abs(*root)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 2
		}
		compatibilityRoot, err := compatibilityRootFor(resolvedRoot, *compatibilityRootOverride)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 2
		}
		if err := authoring.Regenerate(compatibilityRoot); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		fmt.Fprintln(stdout, "generated compatibility packs")
		return 0
	}
	if *requestPath == "" || *approval == "" {
		return 2
	}
	_, compatibilityRoot, resolvedRequest, err := resolveCompatibilityPackPaths(*root, *compatibilityRootOverride, *requestPath)
	if err != nil || !pathWithin(compatibilityRoot, resolvedRequest) {
		fmt.Fprintln(stderr, "compatibility-pack request must be below internal/compatibilitypack")
		return 2
	}
	request, status := readReviewedCompatibilityPackRequest(resolvedRequest, stderr)
	if status != 0 {
		return status
	}
	if err := authoring.Generate(compatibilityRoot, request, *approval); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	fmt.Fprintf(stdout, "generated compatibility pack %s\n", request.ID)
	return 0
}

func runCompatibilityPackCheck(arguments []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("gomadtool compatibility-pack check", flag.ContinueOnError)
	flags.SetOutput(stderr)
	root := flags.String("root", "", "Gomad v3 module root")
	compatibilityRootOverride := flags.String("compatibility-root", "", "absolute pack authoring root owned by another module (default: internal/compatibilitypack)")
	stagedCopy := flags.Bool("staged-copy", false, "root is a staged copy of the module without its repository: require the working-directory table but not a go.mod in the directories it maps outside the copy")
	if err := flags.Parse(arguments); err != nil || flags.NArg() != 0 || *root == "" {
		return 2
	}
	resolvedRoot, err := filepath.Abs(*root)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	compatibilityRoot, err := compatibilityRootFor(resolvedRoot, *compatibilityRootOverride)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	check := authoring.Check
	if *stagedCopy {
		check = authoring.CheckStagedCopy
	}
	if err := check(compatibilityRoot); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	// The repository's own packs are refreshed and qualified through the
	// table, so it must exist; deleting it must not pass validation.
	if *compatibilityRootOverride == "" {
		if err := authoring.CheckWorkingDirectories(compatibilityRoot, !*stagedCopy); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
	}
	fmt.Fprintln(stdout, "compatibility packs are current")
	return 0
}

func runCompatibilityPackQualify(arguments []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("gomadtool compatibility-pack qualify", flag.ContinueOnError)
	flags.SetOutput(stderr)
	root := flags.String("root", "", "Gomad v3 module root")
	compatibilityRootOverride := flags.String("compatibility-root", "", "absolute pack authoring root owned by another module (default: internal/compatibilitypack)")
	requestPath := flags.String("request", "", "compatibility-pack request path")
	workingDirectory := flags.String("working-dir", "", "target working directory")
	all := flags.Bool("all", false, "qualify every request that names the host platform in the working directory its table entry names")
	if err := flags.Parse(arguments); err != nil || flags.NArg() != 0 || *root == "" {
		return 2
	}
	if *all {
		if *requestPath != "" || *workingDirectory != "" {
			return 2
		}
		return qualifyAllCompatibilityPacks(*root, *compatibilityRootOverride, stdout, stderr)
	}
	if *requestPath == "" || *workingDirectory == "" {
		return 2
	}
	resolvedRoot, compatibilityRoot, resolvedRequest, err := resolveCompatibilityPackPaths(*root, *compatibilityRootOverride, *requestPath)
	if err != nil || !pathWithin(compatibilityRoot, resolvedRequest) {
		fmt.Fprintln(stderr, "compatibility-pack request must be below internal/compatibilitypack")
		return 2
	}
	return qualifyCompatibilityPackRequest(resolvedRoot, resolvedRequest, *workingDirectory, stdout, stderr)
}

// qualifyAllCompatibilityPacks qualifies, in table order, every request that
// names the host platform. Packs are scoped to one platform, so each host
// qualifies only its own; a host no request names fails.
func qualifyAllCompatibilityPacks(root, override string, stdout, stderr io.Writer) int {
	resolvedRoot, err := filepath.Abs(root)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	compatibilityRoot, err := compatibilityRootFor(resolvedRoot, override)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	directories, err := authoring.LoadWorkingDirectories(compatibilityRoot)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return compatibilityPackRefreshStatus(err)
	}
	platform := runtime.GOOS + "/" + runtime.GOARCH
	qualified := 0
	for _, id := range requestIDs(directories) {
		path := filepath.Join(compatibilityRoot, "requests", id+".json")
		request, status := readReviewedCompatibilityPackRequest(path, stderr)
		if status != 0 {
			return status
		}
		if !slices.Contains(request.Platforms, platform) {
			continue
		}
		if status := qualifyCompatibilityPackRequest(resolvedRoot, path, directories[id], stdout, stderr); status != 0 {
			return status
		}
		qualified++
	}
	if qualified == 0 {
		fmt.Fprintf(stderr, "no compatibility-pack request names %s\n", platform)
		return 1
	}
	fmt.Fprintf(stdout, "qualified %d compatibility-pack requests for %s\n", qualified, platform)
	return 0
}

func qualifyCompatibilityPackRequest(resolvedRoot, resolvedRequest, workingDirectory string, stdout, stderr io.Writer) int {
	request, status := readReviewedCompatibilityPackRequest(resolvedRequest, stderr)
	if status != 0 {
		return status
	}
	ctx, cancel := context.WithTimeout(context.Background(), compatibilityPackTimeout)
	defer cancel()
	prepared, err := capabilityanalysis.PrepareCapabilityReview(
		ctx,
		request.ReviewSpec(workingDirectory, filepath.Join(resolvedRoot, ".toolchain")),
	)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	err = authoring.Qualify(request, prepared.Review)
	err = errors.Join(err, prepared.Close())
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	fmt.Fprintf(stdout, "qualified compatibility-pack request %s\n", request.ID)
	return 0
}

func resolveCompatibilityPackPaths(root, override, request string) (string, string, string, error) {
	resolvedRoot, err := filepath.Abs(root)
	if err != nil {
		return "", "", "", fmt.Errorf("resolve Gomad v3 root: %w", err)
	}
	compatibilityRoot, err := compatibilityRootFor(resolvedRoot, override)
	if err != nil {
		return "", "", "", err
	}
	requestPath, err := resolveBelow(compatibilityPathBase(resolvedRoot, compatibilityRoot, override), request)
	if err != nil {
		return "", "", "", err
	}
	return resolvedRoot, compatibilityRoot, requestPath, nil
}

// compatibilityPathBase is the directory relative request and report paths are
// resolved from: the Gomad v3 root by default, where they are spelled
// internal/compatibilitypack/..., or the external authoring root itself.
func compatibilityPathBase(root, compatibilityRoot, override string) string {
	if override != "" {
		return compatibilityRoot
	}
	return root
}

// compatibilityRootFor returns the pack authoring root: this module's
// internal/compatibilitypack, or an absolute directory another module owns so
// that packs naming its dependencies never enter this repository. Packs
// generated there are loaded through GOMAD3_COMPATIBILITY_PACKS=ROOT/packs.
func compatibilityRootFor(resolvedRoot, override string) (string, error) {
	if override == "" {
		return filepath.Join(resolvedRoot, "internal", "compatibilitypack"), nil
	}
	if !filepath.IsAbs(override) || filepath.Clean(override) != override {
		return "", fmt.Errorf("--compatibility-root %q must be an absolute, clean path", override)
	}
	return override, nil
}

func resolveBelow(root, path string) (string, error) {
	if !filepath.IsAbs(path) {
		path = filepath.Join(root, path)
	}
	resolved, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	if !pathWithin(root, resolved) {
		return "", errors.New("compatibility-pack path is outside the Gomad v3 root")
	}
	return resolved, nil
}

func pathWithin(root, path string) bool {
	relative, err := filepath.Rel(root, path)
	return err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) && !filepath.IsAbs(relative)
}

func readCompatibilityPackFile(path string, maximum int) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Size() < 0 || info.Size() > int64(maximum) {
		return nil, fmt.Errorf("compatibility-pack input is not a bounded regular file: %s", path)
	}
	contents, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read compatibility-pack input: %w", err)
	}
	return contents, nil
}

func readReviewedCompatibilityPackRequest(path string, stderr io.Writer) (authoring.Request, int) {
	contents, err := readCompatibilityPackFile(path, authoring.MaximumRequestBytes)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return authoring.Request{}, 2
	}
	request, err := authoring.DecodeRequest(contents)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return authoring.Request{}, 2
	}
	return request, 0
}
