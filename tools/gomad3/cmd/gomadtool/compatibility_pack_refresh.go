package main

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"sort"
	"strings"

	"go.temporal.io/server/tools/gomad3/internal/compatibilitypack/authoring"
	capabilityanalysis "go.temporal.io/server/tools/gomad3/qualification/analysis"
	"go.temporal.io/server/tools/gomad3/upgrade/pinimpact"
)

const compatibilityPackRefreshUsage = "usage: gomadtool compatibility-pack refresh --root=DIR [--compatibility-root=DIR] [--baseline-ref=REV] [--go=GO] [--impact-report=FILE]"

// compatibilityPackReviewer reviews a request's target in its working
// directory with the checkout's patched toolchain; tests replace it.
var compatibilityPackReviewer = func(toolchainRoot string) authoring.Reviewer {
	return func(ctx context.Context, request authoring.Request, workingDirectory string) (authoring.CapabilityReview, error) {
		ctx, cancel := context.WithTimeout(ctx, compatibilityPackTimeout)
		defer cancel()
		return capabilityanalysis.ReviewCompatibilityTarget(ctx, request.ReviewSpec(workingDirectory, toolchainRoot))
	}
}

// runCompatibilityPackRefresh runs discover and review for every request the
// checked-out bump invalidates, each in its own working directory, and stops
// at approval. It exits 0 when every selected request is current, 1 when a
// request awaits approval, cannot be evaluated on this host, failed, or is
// no longer selected by its module, 2 for invalid input, and 3 for
// infrastructure failures.
func runCompatibilityPackRefresh(arguments []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("gomadtool compatibility-pack refresh", flag.ContinueOnError)
	flags.SetOutput(stderr)
	root := flags.String("root", "", "Gomad v3 module root")
	compatibilityRootOverride := flags.String("compatibility-root", "", "absolute pack authoring root owned by another module (default: internal/compatibilitypack)")
	baselineRef := flags.String("baseline-ref", "HEAD", "Git revision holding each working directory's go.mod and go.sum before the bump")
	impactPath := flags.String("impact-report", "", "existing pin-impact JSON report for the bumped checkout")
	goCommand := flags.String("go", os.Getenv("GOMAD3_BOOTSTRAP_GO"), "go command that resolves module graphs")
	if err := flags.Parse(arguments); err != nil || flags.NArg() != 0 || *root == "" || *baselineRef == "" {
		fmt.Fprintln(stderr, compatibilityPackRefreshUsage)
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
	directories, err := authoring.LoadWorkingDirectories(compatibilityRoot)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return compatibilityPackRefreshStatus(err)
	}
	if *goCommand == "" {
		*goCommand = "go"
	}
	resolvedGo, err := exec.LookPath(*goCommand)
	if err == nil {
		resolvedGo, err = filepath.Abs(resolvedGo)
	}
	if err != nil {
		fmt.Fprintf(stderr, "compatibility-pack refresh requires a go command; set GOMAD3_BOOTSTRAP_GO or pass --go: %v\n", err)
		return 3
	}
	ctx := context.Background()
	// Judge the packs of the root being refreshed, not the packs the build
	// would load, which come from GOMAD3_COMPATIBILITY_PACKS for an external root.
	impact, err := packPinImpact(ctx, resolvedRoot, resolvedGo, *baselineRef, directories, filepath.Join(compatibilityRoot, "packs"))
	if err != nil {
		fmt.Fprintln(stderr, err)
		if pinimpact.IsInputError(err) {
			return 2
		}
		return 3
	}
	if *impactPath != "" {
		saved, readErr := readPackImpactReport(*impactPath, directories)
		if readErr == nil {
			impact, readErr = mergeSavedPackImpact(impact, saved, directories)
		}
		if readErr != nil {
			fmt.Fprintln(stderr, readErr)
			if pinimpact.IsInputError(readErr) {
				return 2
			}
			return 3
		}
	}
	platform := runtime.GOOS + "/" + runtime.GOARCH
	spec := authoring.RefreshSpec{
		Root: compatibilityRoot, Platform: platform, Invalidated: impact.invalidated,
		Review: compatibilityPackReviewer(filepath.Join(resolvedRoot, ".toolchain")),
	}
	if name, implementation, ok := capabilityanalysis.HostDeterministicProfile(); ok {
		spec.Profile = &authoring.ProfileIdentity{Name: name, ImplementationSHA256: implementation}
	}
	results, err := authoring.Refresh(ctx, spec)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return compatibilityPackRefreshStatus(err)
	}
	base := ""
	rootFlag := "--root=" + resolvedRoot
	if *compatibilityRootOverride == "" {
		base = "internal/compatibilitypack/"
	} else {
		rootFlag += " --compatibility-root=" + compatibilityRoot
	}
	status := 0
	fmt.Fprintf(stdout, "gomad3 compatibility-pack refresh on %s: %d requests selected\n", platform, len(results))
	for _, result := range results {
		switch result.Status {
		case authoring.RefreshCurrent:
			fmt.Fprintf(stdout, "current %s\n", result.ID)
			continue
		case authoring.RefreshAwaitingApproval:
			fmt.Fprintf(stdout, "awaiting-approval %s %s (%s)\n", result.ID, result.ReviewSHA256, result.Reason)
			fmt.Fprintf(stdout, "  review %sreports/%s.md, then approve with:\n  gomadtool compatibility-pack generate %s --request=%srequests/%s.json --approve-review=%s\n", base, result.ID, rootFlag, base, result.ID, result.ReviewSHA256)
		default:
			fmt.Fprintf(stdout, "%s %s: %s\n", result.Status, result.ID, result.Reason)
		}
		status = 1
	}
	for _, id := range impact.unselected {
		if slices.ContainsFunc(results, func(result authoring.RefreshResult) bool { return result.ID == id }) {
			continue
		}
		fmt.Fprintf(stdout, "unselected %s: its working directory no longer requires the pack's modules; remove the pack, request, report, and working-directory entry once nothing selects it\n", id)
		status = 1
	}
	return status
}

func compatibilityPackRefreshStatus(err error) int {
	if authoring.IsInputError(err) {
		return 2
	}
	return 3
}

type packImpact struct {
	// invalidated maps each request whose pack the bump invalidates, or
	// leaves unknown, to the first reason.
	invalidated map[string]string
	// unselected requests' packs applied before the bump and their working
	// directory no longer requires their modules.
	unselected []string
	// identities bind a live evaluation to each mapped module. A saved
	// report may cover one of these modules, never replace the others.
	identities    map[string]packModuleIdentity
	savedIdentity packModuleIdentity
	reportedIDs   map[string]bool
}

type packModuleIdentity struct {
	candidateMod, candidateSum, baselineMod, baselineSum string
	candidateCombined, baselineCombined                  string
}

func moduleIdentity(candidate, baseline pinimpact.ModuleFiles) packModuleIdentity {
	digest := func(contents []byte) string { return fmt.Sprintf("sha256:%x", sha256.Sum256(contents)) }
	combined := func(files pinimpact.ModuleFiles) string {
		joined := append(append(append([]byte{}, files.GoMod...), 0), files.GoSum...)
		return digest(joined)
	}
	return packModuleIdentity{
		candidateMod: digest(candidate.GoMod), candidateSum: digest(candidate.GoSum),
		baselineMod: digest(baseline.GoMod), baselineSum: digest(baseline.GoSum),
		candidateCombined: combined(candidate), baselineCombined: combined(baseline),
	}
}

func mergeSavedPackImpact(live, saved packImpact, directories map[string]string) (packImpact, error) {
	matched := ""
	for directory, identity := range live.identities {
		if (saved.savedIdentity.candidateCombined != "" && saved.savedIdentity.candidateCombined == identity.candidateCombined && saved.savedIdentity.baselineCombined == identity.baselineCombined) ||
			(saved.savedIdentity.candidateMod != "" && saved.savedIdentity.candidateMod == identity.candidateMod && saved.savedIdentity.candidateSum == identity.candidateSum && saved.savedIdentity.baselineMod == identity.baselineMod && saved.savedIdentity.baselineSum == identity.baselineSum) {
			matchesPins := true
			for id := range saved.reportedIDs {
				if directories[id] != directory {
					matchesPins = false
					break
				}
			}
			if matchesPins {
				matched = directory
				break
			}
		}
	}
	if matched == "" {
		return packImpact{}, &pinimpact.InputError{Err: errors.New("pin-impact report does not match a mapped module's current candidate and requested baseline, including its pack mappings")}
	}
	for id, reason := range saved.invalidated {
		if live.invalidated[id] == "" {
			live.invalidated[id] = reason
		}
	}
	return live, nil
}

// readPackImpactReport accepts either pin-impact report representation. Both
// identify pack rules by exact request ID, and neither grants an approval.
func readPackImpactReport(path string, directories map[string]string) (packImpact, error) {
	contents, err := readCompatibilityPackFile(path, 16<<20)
	if err != nil {
		return packImpact{}, &pinimpact.InputError{Err: err}
	}
	var report struct {
		Schema          string                   `json:"schema"`
		CandidateSHA256 string                   `json:"candidate_sha256"`
		BaselineSHA256  string                   `json:"baseline_sha256"`
		Candidate       pinimpact.ModuleEvidence `json:"candidate"`
		Baseline        pinimpact.ModuleEvidence `json:"baseline"`
		Pins            []struct {
			Class      string `json:"class"`
			Status     string `json:"status"`
			ID         string `json:"id"`
			Pack       string `json:"pack"`
			ImportPath string `json:"import_path"`
			Reason     string `json:"reason"`
		} `json:"pins"`
	}
	if err := json.Unmarshal(contents, &report); err != nil || report.Schema != pinimpact.Schema {
		return packImpact{}, &pinimpact.InputError{Err: errors.New("invalid pin-impact report")}
	}
	identity := packModuleIdentity{}
	switch {
	case report.CandidateSHA256 != "" && report.BaselineSHA256 != "" && report.Candidate.GoModSHA256 == "" && report.Candidate.GoSumSHA256 == "" && report.Baseline.GoModSHA256 == "" && report.Baseline.GoSumSHA256 == "":
		identity.candidateCombined, identity.baselineCombined = report.CandidateSHA256, report.BaselineSHA256
	case report.CandidateSHA256 == "" && report.BaselineSHA256 == "" && report.Candidate.GoModSHA256 != "" && report.Candidate.GoSumSHA256 != "" && report.Baseline.GoModSHA256 != "" && report.Baseline.GoSumSHA256 != "":
		identity.candidateMod, identity.candidateSum = report.Candidate.GoModSHA256, report.Candidate.GoSumSHA256
		identity.baselineMod, identity.baselineSum = report.Baseline.GoModSHA256, report.Baseline.GoSumSHA256
	default:
		return packImpact{}, &pinimpact.InputError{Err: errors.New("pin-impact report lacks a complete candidate and baseline module identity")}
	}
	impact := packImpact{invalidated: map[string]string{}, savedIdentity: identity, reportedIDs: map[string]bool{}}
	stale := map[string]bool{}
	for _, pin := range report.Pins {
		if pin.Class != string(pinimpact.ClassPackRule) && pin.Class != "pack_rule" {
			continue
		}
		id := pin.Pack
		if id == "" {
			id, _, _ = strings.Cut(pin.ID, ":")
		}
		if directories[id] == "" {
			return packImpact{}, &pinimpact.InputError{Err: fmt.Errorf("pin-impact report names unmapped compatibility pack %q", id)}
		}
		impact.reportedIDs[id] = true
		switch pin.Status {
		case "invalidated", "unknown":
			if impact.invalidated[id] == "" {
				impact.invalidated[id] = fmt.Sprintf("%s %s: %s", pin.Status, pin.ID, pin.Reason)
			}
		case "stale":
			stale[id] = true
		}
	}
	for id := range stale {
		if impact.invalidated[id] == "" {
			impact.unselected = append(impact.unselected, id)
		}
	}
	sort.Strings(impact.unselected)
	return impact, nil
}

// packPinImpact runs the pin impact report once per working directory, with
// the working tree as the candidate and baselineRef as the baseline, and keeps
// the pack-rule pins of the requests mapped to that directory.
func packPinImpact(ctx context.Context, root, goCommand, baselineRef string, directories map[string]string, packsDirectory string) (packImpact, error) {
	byDirectory := map[string][]string{}
	for id, directory := range directories {
		byDirectory[directory] = append(byDirectory[directory], id)
	}
	ordered := make([]string, 0, len(byDirectory))
	for directory := range byDirectory {
		ordered = append(ordered, directory)
	}
	sort.Strings(ordered)
	resolver, err := pinimpact.NewGoResolver(goCommand, os.Environ())
	if err != nil {
		return packImpact{}, err
	}
	impact := packImpact{invalidated: map[string]string{}, identities: map[string]packModuleIdentity{}}
	stale := map[string]bool{}
	for _, directory := range ordered {
		candidate, err := readModuleFiles(directory)
		if err != nil {
			return packImpact{}, errors.Join(&pinimpact.InputError{Err: err}, resolver.Close())
		}
		baseline, err := gitModuleFiles(ctx, directory, baselineRef)
		if err != nil {
			return packImpact{}, errors.Join(&pinimpact.InputError{Err: err}, resolver.Close())
		}
		impact.identities[directory] = moduleIdentity(candidate, baseline)
		report, err := pinimpact.Evaluate(ctx, pinimpact.Spec{Root: root, Baseline: baseline, Candidate: candidate, Resolver: resolver, PacksDirectory: packsDirectory})
		if err != nil {
			return packImpact{}, errors.Join(fmt.Errorf("pin impact of %s: %w", directory, err), resolver.Close())
		}
		mapped := byDirectory[directory]
		for _, pin := range report.Pins {
			if pin.Class != pinimpact.ClassPackRule {
				continue
			}
			if pin.Pack == "" {
				return packImpact{}, errors.Join(fmt.Errorf("pin impact of %s cannot evaluate compatibility packs: %s", directory, pin.Reason), resolver.Close())
			}
			if !slices.Contains(mapped, pin.Pack) {
				continue
			}
			switch pin.Status {
			case pinimpact.StatusInvalidated, pinimpact.StatusUnknown:
				if impact.invalidated[pin.Pack] == "" {
					impact.invalidated[pin.Pack] = fmt.Sprintf("%s %s: %s", pin.Status, pin.ImportPath, pin.Reason)
				}
			case pinimpact.StatusStale:
				stale[pin.Pack] = true
			case pinimpact.StatusUnaffected, pinimpact.StatusNotSelected:
			}
		}
	}
	for id := range stale {
		if _, invalidated := impact.invalidated[id]; !invalidated {
			impact.unselected = append(impact.unselected, id)
		}
	}
	sort.Strings(impact.unselected)
	return impact, resolver.Close()
}

func requestIDs(directories map[string]string) []string {
	ids := make([]string, 0, len(directories))
	for id := range directories {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}
