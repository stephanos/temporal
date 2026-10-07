// Package pinimpact reports which exact-version pins a candidate go.mod
// invalidates. It reads adapter identities from the deterministic I/O adapter
// registry, pack rules from the compatibility-pack loader, and the
// toolchain-bound pins from the checked-in descriptors, so it judges each pin
// by the same identity the build's fail-closed checks compare.
package pinimpact

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	goversion "go/version"
	"os"
	"slices"
	"strings"

	"go.temporal.io/server/tools/gomad3/deterministicio"
	compatibility "go.temporal.io/server/tools/gomad3/internal/compatibilitypack"
	gomadversion "go.temporal.io/server/tools/gomad3/toolchain/version"
)

const Schema = "gomad3.pin-impact/v1"

type Class string

const (
	ClassAdapter        Class = "adapter"
	ClassPackRule       Class = "pack-rule"
	ClassInterception   Class = "interception-fingerprint"
	ClassClockReference Class = "clock-reference"
)

var classes = []Class{ClassAdapter, ClassPackRule, ClassInterception, ClassClockReference}

type Status string

const (
	// StatusUnaffected pins match the candidate exactly.
	StatusUnaffected Status = "unaffected"
	// StatusNotSelected pins do not apply to the baseline module, such as a
	// pack variant for another module version or an adapter for a module the
	// target does not require.
	StatusNotSelected Status = "not-selected"
	// StatusStale pins applied to the baseline, but the candidate no longer
	// requires their module, so the build stops using them without rejecting.
	StatusStale Status = "stale"
	// StatusInvalidated pins are rejected by the build for the candidate.
	StatusInvalidated Status = "invalidated"
	// StatusUnknown pins could not be evaluated; they count as invalidated.
	StatusUnknown Status = "unknown"
)

// ModuleFiles is one module's go.mod and go.sum. Directory resolves relative
// local replacements and may be empty when there are none.
type ModuleFiles struct {
	GoMod     []byte
	GoSum     []byte
	Directory string
}

// Resolver returns the version the module graph selects for every module other
// than the main module. It must not modify files.Directory.
type Resolver interface {
	Resolve(ctx context.Context, files ModuleFiles) (map[string]string, error)
}

type Spec struct {
	// Root is the Gomad v3 module root holding the boundary manifest and the
	// reviewed host-clock inventory.
	Root      string
	Baseline  ModuleFiles
	Candidate ModuleFiles
	Resolver  Resolver
	// PacksDirectory contains explicitly authored pack files. Empty uses the
	// packs selected by the build, including its normal environment configuration.
	PacksDirectory string
	// IncludeAll retains unaffected and unselected entries for callers that
	// need the complete pin inventory rather than the actionable subset.
	IncludeAll bool
}

type Report struct {
	Schema          string         `json:"schema"`
	Invalidated     bool           `json:"invalidated"`
	PinnedGoVersion string         `json:"pinned_go_version"`
	Baseline        ModuleEvidence `json:"baseline"`
	Candidate       ModuleEvidence `json:"candidate"`
	Platforms       []string       `json:"platforms"`
	Classes         []ClassSummary `json:"classes"`
	Pins            []Pin          `json:"pins"`
}

type ModuleEvidence struct {
	GoModSHA256 string `json:"go_mod_sha256"`
	GoSumSHA256 string `json:"go_sum_sha256"`
	GoDirective string `json:"go_directive"`
}

type ClassSummary struct {
	Class       Class `json:"class"`
	Total       int   `json:"total"`
	Unaffected  int   `json:"unaffected"`
	NotSelected int   `json:"not_selected"`
	Stale       int   `json:"stale"`
	Invalidated int   `json:"invalidated"`
	Unknown     int   `json:"unknown"`
}

// Pin is one pin the candidate invalidates, leaves unknown, or makes stale.
type Pin struct {
	Class            Class    `json:"class"`
	Status           Status   `json:"status"`
	ID               string   `json:"id"`
	Module           string   `json:"module,omitempty"`
	PinnedVersion    string   `json:"pinned_version,omitempty"`
	PinnedSum        string   `json:"pinned_sum,omitempty"`
	CandidateVersion string   `json:"candidate_version,omitempty"`
	CandidateSum     string   `json:"candidate_sum,omitempty"`
	Pack             string   `json:"pack,omitempty"`
	PackSHA256       string   `json:"pack_sha256,omitempty"`
	ImportPath       string   `json:"import_path,omitempty"`
	SourceSetSHA256  string   `json:"source_set_sha256,omitempty"`
	Platforms        []string `json:"platforms,omitempty"`
	Reason           string   `json:"reason"`
}

// InputError marks a candidate or baseline that cannot be read as a module.
type InputError struct {
	Err error
}

func (err *InputError) Error() string {
	return err.Err.Error()
}

func (err *InputError) Unwrap() error {
	return err.Err
}

func IsInputError(err error) bool {
	var input *InputError
	return errors.As(err, &input)
}

// Evaluate reports every pin the candidate invalidates relative to the
// baseline. Errors are invalid input (IsInputError) or infrastructure
// failures; a pin that cannot be evaluated is reported unknown instead.
func Evaluate(ctx context.Context, spec Spec) (Report, error) {
	if spec.Resolver == nil {
		return Report{}, errors.New("pin impact report requires a module resolver")
	}
	baseline, err := loadModule(ctx, "baseline", spec.Baseline, spec.Resolver)
	if err != nil {
		return Report{}, err
	}
	candidate, err := loadModule(ctx, "candidate", spec.Candidate, spec.Resolver)
	if err != nil {
		return Report{}, err
	}
	evaluation := evaluation{baseline: baseline, candidate: candidate, adapters: deterministicio.Default().Adapters(), includeAll: spec.IncludeAll}
	evaluation.evaluateAdapters()
	var packs []compatibility.ValidatedPack
	var packErr error
	if spec.PacksDirectory == "" {
		packs, packErr = compatibility.LoadPacks()
	} else {
		packs, packErr = compatibility.LoadPackDirectory(spec.PacksDirectory)
	}
	evaluation.evaluatePacks(packs, packErr, spec.PacksDirectory)
	toolchainReason := candidate.toolchainReason()
	evaluation.evaluateInterceptions(spec.Root, toolchainReason)
	evaluation.evaluateClockReferences(spec.Root, toolchainReason)
	return evaluation.report(spec), nil
}

type evaluation struct {
	baseline   moduleState
	candidate  moduleState
	adapters   []gomadversion.AdapterIdentity
	platforms  []string
	summaries  map[Class]*ClassSummary
	pins       []Pin
	includeAll bool
}

func (evaluation *evaluation) record(pin Pin) {
	if evaluation.summaries == nil {
		evaluation.summaries = make(map[Class]*ClassSummary, len(classes))
	}
	summary := evaluation.summaries[pin.Class]
	if summary == nil {
		summary = &ClassSummary{Class: pin.Class}
		evaluation.summaries[pin.Class] = summary
	}
	summary.Total++
	switch pin.Status {
	case StatusUnaffected:
		summary.Unaffected++
		if !evaluation.includeAll {
			return
		}
	case StatusNotSelected:
		summary.NotSelected++
		if !evaluation.includeAll {
			return
		}
	case StatusStale:
		summary.Stale++
	case StatusInvalidated:
		summary.Invalidated++
	default:
		pin.Status = StatusUnknown
		summary.Unknown++
	}
	evaluation.pins = append(evaluation.pins, pin)
}

// evaluateAdapters mirrors the adapter registry's build check: a target that
// requires or replaces an adapted module must require exactly the pinned
// version and record exactly the pinned sum.
func (evaluation *evaluation) evaluateAdapters() {
	for _, identity := range evaluation.adapters {
		pin := Pin{
			Class: ClassAdapter, ID: identity.Module + "@" + identity.Version, Module: identity.Module,
			PinnedVersion: identity.Version, PinnedSum: identity.Sum,
		}
		pin.CandidateVersion, pin.CandidateSum = evaluation.candidate.observed(identity.Module)
		match := evaluation.candidate.matchAdapter(identity.Module, identity.Version, identity.Sum)
		switch {
		case match.ok:
			pin.Status = StatusUnaffected
		case match.absent && evaluation.baseline.requires(identity.Module):
			pin.Status, pin.Reason = StatusStale, "candidate no longer requires "+identity.Module
		case match.absent:
			pin.Status = StatusNotSelected
		case match.unknown:
			pin.Status, pin.Reason = StatusUnknown, match.reason
		default:
			pin.Status, pin.Reason = StatusInvalidated, match.reason
		}
		evaluation.record(pin)
	}
}

// evaluatePacks judges each rule of every pack the baseline selects. A pack is
// selected when all its activation modules match exactly, as the build's pack
// selection requires.
func (evaluation *evaluation) evaluatePacks(packs []compatibility.ValidatedPack, loadErr error, directory string) {
	if loadErr != nil {
		reason := loadErr.Error()
		label := "$PacksDirectory"
		if directory == "" {
			directory = os.Getenv(compatibility.ExternalPacksEnvironment)
			label = "$" + compatibility.ExternalPacksEnvironment
		}
		if directory != "" {
			reason = strings.ReplaceAll(reason, directory, label)
		}
		evaluation.record(Pin{Class: ClassPackRule, Status: StatusUnknown, ID: "compatibility packs", Reason: "load compatibility packs: " + reason})
		return
	}
	loaded := make([]compatibility.Pack, len(packs))
	for index, validated := range packs {
		loaded[index] = validated.Pack()
	}
	order := make([]int, len(packs))
	for index := range order {
		order[index] = index
	}
	slices.SortFunc(order, func(left, right int) int { return strings.Compare(loaded[left].ID, loaded[right].ID) })
	for _, index := range order {
		validated, pack := packs[index], loaded[index]
		evaluation.platforms = append(evaluation.platforms, pack.Governance.Platforms...)
		baselineSelected := evaluation.activates(evaluation.baseline, pack.Activation).ok
		candidateActivation := evaluation.activates(evaluation.candidate, pack.Activation)
		for _, rule := range pack.Rules {
			pin := Pin{
				Class: ClassPackRule, ID: pack.ID + " " + rule.ImportPath, Module: rule.Module.Path,
				PinnedVersion: rule.Module.Version, PinnedSum: rule.Module.Sum,
				Pack: pack.ID, PackSHA256: validated.SHA256(), ImportPath: rule.ImportPath,
				SourceSetSHA256: rule.SourceSetSHA256, Platforms: slices.Clone(pack.Governance.Platforms),
			}
			pin.CandidateVersion, pin.CandidateSum = evaluation.candidate.observed(rule.Module.Path)
			match := evaluation.packModule(evaluation.candidate, rule.Module)
			if candidateActivation.unknown && (match.ok || match.unknown) {
				pin.Status, pin.Reason = StatusUnknown, evaluation.candidate.unresolved
				if pin.Reason == "" {
					pin.Reason = candidateActivation.reason
				}
				evaluation.record(pin)
				continue
			}
			if !baselineSelected || !evaluation.packModule(evaluation.baseline, rule.Module).ok {
				pin.Status = StatusNotSelected
				// A pack whose activation modules the candidate still requires,
				// at other versions, was stranded by a bump; once the bump is
				// the baseline it is never unaffected, so it stays invalidated.
				if !baselineSelected && strandedActivation(evaluation.candidate, pack.Activation, candidateActivation) {
					pin.Status, pin.Reason = StatusInvalidated, "pack activation "+candidateActivation.reason+"; the candidate still requires every activation module"
				}
				evaluation.record(pin)
				continue
			}
			switch {
			case match.ok && candidateActivation.ok:
				pin.Status = StatusUnaffected
			case match.absent:
				pin.Status, pin.Reason = StatusStale, "candidate no longer requires "+rule.Module.Path
			case !match.ok && !match.unknown:
				pin.Status, pin.Reason = StatusInvalidated, match.reason
			case !candidateActivation.ok:
				pin.Status, pin.Reason = StatusInvalidated, "pack activation "+candidateActivation.reason
			default:
				pin.Status, pin.Reason = StatusUnknown, match.reason
			}
			evaluation.record(pin)
		}
	}
}

// strandedActivation reports a pack the candidate does not select although it
// requires every one of the pack's activation modules.
func strandedActivation(target moduleState, activation []compatibility.PackModule, match moduleMatch) bool {
	if match.ok || match.unknown || len(activation) == 0 {
		return false
	}
	for _, required := range activation {
		if len(target.required[required.Path]) == 0 {
			return false
		}
	}
	return true
}

func (evaluation *evaluation) activates(target moduleState, activation []compatibility.PackModule) moduleMatch {
	result := moduleMatch{ok: true}
	for _, required := range activation {
		if match := evaluation.packModule(target, required); !match.ok {
			match.reason = required.Path + ": " + match.reason
			if !match.unknown {
				return match
			}
			if result.ok {
				result = match
			}
		}
	}
	return result
}

// packModule matches one pack module the way pack selection matches a built
// package's module: the exact identity, and an adapter replacement exactly
// when the registry adapts that identity.
func (evaluation *evaluation) packModule(target moduleState, required compatibility.PackModule) moduleMatch {
	match := target.match(required.Path, required.Version, required.Sum)
	if !match.ok {
		return match
	}
	adapterIndex := slices.IndexFunc(evaluation.adapters, func(identity gomadversion.AdapterIdentity) bool {
		return identity.Module == required.Path
	})
	if adapterIndex >= 0 {
		if match := target.matchAdapter(required.Path, required.Version, required.Sum); !match.ok {
			return match
		}
	}
	switch required.Replacement.Kind {
	case compatibility.ReplacementNone:
		if adapterIndex >= 0 {
			return moduleMatch{reason: "the build replaces " + required.Path + " with its deterministic I/O adapter"}
		}
	case compatibility.ReplacementAdapter:
		evidence := required.Replacement.Adapter
		if adapterIndex < 0 || evidence == nil {
			return moduleMatch{reason: "no deterministic I/O adapter replaces " + required.Path}
		}
		adapter := evaluation.adapters[adapterIndex]
		if adapter.Version != required.Version || adapter.Sum != required.Sum ||
			evidence.Module != adapter.Module || evidence.Version != adapter.Version || evidence.Sum != adapter.Sum {
			return moduleMatch{reason: fmt.Sprintf("pack expects the %s@%s adapter; the registry pins %s", required.Path, required.Version, adapter.Version)}
		}
	default:
		return moduleMatch{reason: fmt.Sprintf("pack replacement kind %q is unknown", required.Replacement.Kind)}
	}
	return match
}

func (evaluation *evaluation) report(spec Spec) Report {
	platforms := append(slices.Clone(evaluation.platforms), gomadversion.SupportedPlatforms[:]...)
	slices.Sort(platforms)
	report := Report{
		Schema: Schema, PinnedGoVersion: gomadversion.GoVersion,
		Baseline:  moduleEvidence(spec.Baseline, evaluation.baseline),
		Candidate: moduleEvidence(spec.Candidate, evaluation.candidate),
		Platforms: slices.Compact(platforms),
		Classes:   make([]ClassSummary, 0, len(classes)),
		Pins:      evaluation.pins,
	}
	for _, class := range classes {
		summary := ClassSummary{Class: class}
		if recorded := evaluation.summaries[class]; recorded != nil {
			summary = *recorded
		}
		report.Classes = append(report.Classes, summary)
		report.Invalidated = report.Invalidated || summary.Invalidated+summary.Unknown > 0
	}
	if report.Pins == nil {
		report.Pins = []Pin{}
	}
	return report
}

func moduleEvidence(files ModuleFiles, parsed moduleState) ModuleEvidence {
	return ModuleEvidence{GoModSHA256: digest(files.GoMod), GoSumSHA256: digest(files.GoSum), GoDirective: parsed.goDirective}
}

func digest(contents []byte) string {
	return fmt.Sprintf("sha256:%x", sha256.Sum256(contents))
}

// toolchainReason explains why the pinned toolchain cannot build the
// candidate, or returns "" when it can. Builds run with GOTOOLCHAIN=local, so
// only the go directive matters.
func (target moduleState) toolchainReason() string {
	if target.goDirective != "" && goversion.Compare("go"+target.goDirective, gomadversion.GoVersion) > 0 {
		return fmt.Sprintf("candidate requires go %s, newer than the pinned %s; a Go upgrade re-derives this pin through upgrade-dossier", target.goDirective, gomadversion.GoVersion)
	}
	if target.graphGo != "" && goversion.Compare("go"+target.graphGo, gomadversion.GoVersion) > 0 {
		return fmt.Sprintf("candidate module graph requires go %s, newer than the pinned %s; a Go upgrade re-derives this pin through upgrade-dossier", target.graphGo, gomadversion.GoVersion)
	}
	return ""
}
