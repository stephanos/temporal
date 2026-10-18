package build

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"slices"
	"strings"
	"time"
	"unicode"

	targetbuild "go.temporal.io/server/tools/gomad/internal/backends/native/build/internal/build"
	"go.temporal.io/server/tools/gomad/internal/backends/native/build/internal/capabilitypolicy"
	"go.temporal.io/server/tools/gomad/internal/backends/native/build/internal/gocommand"
	compatibility "go.temporal.io/server/tools/gomad/internal/backends/native/compatibilitypack"
	gomadversion "go.temporal.io/server/tools/gomad/internal/backends/native/toolchain/version"
	"go.temporal.io/server/tools/gomad/internal/backends/target"
)

// ReviewSourceCapabilities inventories a declared platform without building or
// executing a target. It supplies no execution admission or qualification.
func ReviewSourceCapabilities(ctx context.Context, spec target.Spec, goCommand, goos, goarch string) (target.CapabilityReview, error) {
	if !slices.Contains(gomadversion.SupportedPlatforms[:], goos+"/"+goarch) {
		return target.CapabilityReview{}, invalidCapabilityReview(fmt.Errorf("unsupported source review platform %s/%s", goos, goarch))
	}
	if (spec.Kind != target.KindGoRun && spec.Kind != target.KindGoTest) || (spec.CapabilityMode != "" && spec.CapabilityMode != target.CapabilityModeClosure) {
		return target.CapabilityReview{}, invalidCapabilityReview(errors.New("source review requires a closure go-run or go-test target"))
	}
	if spec.Source == "" || spec.WorkingDir == "" || strings.HasPrefix(spec.Source, "-") || strings.Contains(spec.Source, "...") || strings.IndexFunc(spec.Source, unicode.IsSpace) >= 0 || strings.IndexByte(spec.Source, 0) >= 0 {
		return target.CapabilityReview{}, invalidCapabilityReview(errors.New("source review requires one package and a working directory"))
	}
	environment := append(targetbuild.Environment(), "GOOS="+goos, "GOARCH="+goarch)
	runner := gocommand.Default()
	if _, err := SourceReviewGoRoot(ctx, goCommand); err != nil {
		return target.CapabilityReview{}, err
	}
	tags, err := targetbuild.NormalizeTags(spec.BuildTags)
	if err != nil {
		return target.CapabilityReview{}, invalidCapabilityReview(err)
	}
	buildContext, err := targetbuild.Resolve(spec.WorkingDir, spec.Source, tags)
	if err != nil {
		return target.CapabilityReview{}, invalidCapabilityReview(err)
	}
	packages, overlay, err := collectCapabilityListingWithEnvironment(ctx, goCommand, spec, buildContext.Tags, buildContext.Directory, buildContext.Package, environment, runner)
	if err != nil {
		return target.CapabilityReview{}, err
	}
	collected, err := collectCapabilityPackages(packages, overlay, spec.AdapterReplacements)
	if err != nil {
		return target.CapabilityReview{}, err
	}
	packs, err := compatibility.LoadPacks()
	if err != nil {
		return target.CapabilityReview{}, err
	}
	return evaluateCollectedCapabilities(capabilitypolicy.Policy{Packs: packs, Platform: goos + "/" + goarch}, collected, buildContext.Tags)
}

// SourceReviewGoRoot obtains the pinned compiler's own root so executable
// aliases outside its installation do not change source resolution.
func SourceReviewGoRoot(ctx context.Context, goCommand string) (string, error) {
	result, err := gocommand.Default().StructuredCommand(ctx, gocommand.Request{
		Command: []string{goCommand, "env", "GOVERSION", "GOROOT"},
		Env:     targetbuild.Environment(), Timeout: 30 * time.Second, OutputLimit: maximumGoEnvironmentBytes,
	})
	if err == nil {
		err = result.OutputError()
	}
	if err != nil {
		return "", fmt.Errorf("query source review Go identity: %w", err)
	}
	fields := strings.Split(strings.TrimSuffix(string(result.Stdout), "\n"), "\n")
	if len(fields) != 2 || fields[0] != gomadversion.GoVersion || !filepath.IsAbs(fields[1]) || strings.IndexByte(fields[1], 0) >= 0 {
		return "", fmt.Errorf("source review requires %s and an absolute compiler GOROOT, got %q", gomadversion.GoVersion, result.Stdout)
	}
	return fields[1], nil
}
