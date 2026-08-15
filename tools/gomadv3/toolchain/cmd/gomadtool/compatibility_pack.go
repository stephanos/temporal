package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go.temporal.io/server/tools/gomadv3/qualification"
	"go.temporal.io/server/tools/gomadv3/target/packdev"
)

const compatibilityPackTimeout = 2 * time.Minute

func runCompatibilityPack(arguments []string, stdout, stderr io.Writer) int {
	if len(arguments) == 0 {
		fmt.Fprintln(stderr, "usage: gomadtool compatibility-pack discover|review|generate|check|qualify [flags]")
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
	default:
		fmt.Fprintln(stderr, "usage: gomadtool compatibility-pack discover|review|generate|check|qualify [flags]")
		return 2
	}
}

func runCompatibilityPackDiscover(arguments []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("gomadtool compatibility-pack discover", flag.ContinueOnError)
	flags.SetOutput(stderr)
	root := flags.String("root", "", "Gomad v3 module root")
	requestPath := flags.String("request", "", "compatibility-pack request path")
	workingDirectory := flags.String("working-dir", "", "target working directory")
	if err := flags.Parse(arguments); err != nil || flags.NArg() != 0 || *root == "" || *requestPath == "" || *workingDirectory == "" {
		return 2
	}
	resolvedRoot, compatibilityRoot, resolvedRequest, err := resolveCompatibilityPackPaths(*root, *requestPath)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	draftBytes, err := readCompatibilityPackFile(resolvedRequest, packdev.MaximumRequestBytes)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	draft, err := packdev.DecodeDraftRequest(draftBytes)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	ctx, cancel := context.WithTimeout(context.Background(), compatibilityPackTimeout)
	defer cancel()
	prepared, err := qualification.PrepareCapabilityReview(
		ctx,
		draft.ReviewSpec(*workingDirectory, filepath.Join(resolvedRoot, ".toolchain")),
	)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	discovered, digest, err := packdev.Discover(draft, prepared.Review)
	err = errors.Join(err, prepared.Close())
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if !pathWithin(compatibilityRoot, resolvedRequest) {
		fmt.Fprintln(stderr, "compatibility-pack request must be below target/internal/compatibility")
		return 2
	}
	if err := packdev.PublishRequest(resolvedRequest, discovered); err != nil {
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
	requestPath := flags.String("request", "", "compatibility-pack request path")
	outputPath := flags.String("output", "", "review report path")
	if err := flags.Parse(arguments); err != nil || flags.NArg() != 0 || *root == "" || *requestPath == "" || *outputPath == "" {
		return 2
	}
	_, compatibilityRoot, resolvedRequest, err := resolveCompatibilityPackPaths(*root, *requestPath)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	resolvedOutput, err := resolveBelow(*root, *outputPath)
	if err != nil || !pathWithin(compatibilityRoot, resolvedOutput) {
		fmt.Fprintln(stderr, "compatibility-pack review output must be below target/internal/compatibility")
		return 2
	}
	request, status := readReviewedCompatibilityPackRequest(resolvedRequest, stderr)
	if status != 0 {
		return status
	}
	digest, err := packdev.PublishReview(resolvedOutput, request)
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
		if err := packdev.Regenerate(filepath.Join(resolvedRoot, "target", "internal", "compatibility")); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		fmt.Fprintln(stdout, "generated compatibility packs")
		return 0
	}
	if *requestPath == "" || *approval == "" {
		return 2
	}
	_, compatibilityRoot, resolvedRequest, err := resolveCompatibilityPackPaths(*root, *requestPath)
	if err != nil || !pathWithin(compatibilityRoot, resolvedRequest) {
		fmt.Fprintln(stderr, "compatibility-pack request must be below target/internal/compatibility")
		return 2
	}
	request, status := readReviewedCompatibilityPackRequest(resolvedRequest, stderr)
	if status != 0 {
		return status
	}
	if err := packdev.Generate(compatibilityRoot, request, *approval); err != nil {
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
	if err := flags.Parse(arguments); err != nil || flags.NArg() != 0 || *root == "" {
		return 2
	}
	resolvedRoot, err := filepath.Abs(*root)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	if err := packdev.Check(filepath.Join(resolvedRoot, "target", "internal", "compatibility")); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	fmt.Fprintln(stdout, "compatibility packs are current")
	return 0
}

func runCompatibilityPackQualify(arguments []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("gomadtool compatibility-pack qualify", flag.ContinueOnError)
	flags.SetOutput(stderr)
	root := flags.String("root", "", "Gomad v3 module root")
	requestPath := flags.String("request", "", "compatibility-pack request path")
	workingDirectory := flags.String("working-dir", "", "target working directory")
	if err := flags.Parse(arguments); err != nil || flags.NArg() != 0 || *root == "" || *requestPath == "" || *workingDirectory == "" {
		return 2
	}
	resolvedRoot, compatibilityRoot, resolvedRequest, err := resolveCompatibilityPackPaths(*root, *requestPath)
	if err != nil || !pathWithin(compatibilityRoot, resolvedRequest) {
		fmt.Fprintln(stderr, "compatibility-pack request must be below target/internal/compatibility")
		return 2
	}
	request, status := readReviewedCompatibilityPackRequest(resolvedRequest, stderr)
	if status != 0 {
		return status
	}
	ctx, cancel := context.WithTimeout(context.Background(), compatibilityPackTimeout)
	defer cancel()
	prepared, err := qualification.PrepareCapabilityReview(
		ctx,
		request.ReviewSpec(*workingDirectory, filepath.Join(resolvedRoot, ".toolchain")),
	)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	err = packdev.Qualify(request, prepared.Review)
	err = errors.Join(err, prepared.Close())
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	fmt.Fprintf(stdout, "qualified compatibility-pack request %s\n", request.ID)
	return 0
}

func resolveCompatibilityPackPaths(root, request string) (string, string, string, error) {
	resolvedRoot, err := filepath.Abs(root)
	if err != nil {
		return "", "", "", fmt.Errorf("resolve Gomad v3 root: %w", err)
	}
	requestPath, err := resolveBelow(resolvedRoot, request)
	if err != nil {
		return "", "", "", err
	}
	return resolvedRoot, filepath.Join(resolvedRoot, "target", "internal", "compatibility"), requestPath, nil
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

func readReviewedCompatibilityPackRequest(path string, stderr io.Writer) (packdev.Request, int) {
	contents, err := readCompatibilityPackFile(path, packdev.MaximumRequestBytes)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return packdev.Request{}, 2
	}
	request, err := packdev.DecodeRequest(contents)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return packdev.Request{}, 2
	}
	return request, 0
}
