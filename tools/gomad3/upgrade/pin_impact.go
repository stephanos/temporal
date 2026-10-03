package upgrade

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"go.temporal.io/server/tools/gomad3/deterministicio"
	"go.temporal.io/server/tools/gomad3/internal/canonicaljson"
	compatibility "go.temporal.io/server/tools/gomad3/internal/compatibilitypack"
	gomadversion "go.temporal.io/server/tools/gomad3/toolchain/version"
	"golang.org/x/mod/modfile"
)

type PinImpact struct {
	Schema          string      `json:"schema"`
	CandidateSHA256 string      `json:"candidate_sha256"`
	BaselineSHA256  string      `json:"baseline_sha256"`
	Pins            []PinResult `json:"pins"`
}

type PinResult struct {
	Class           string   `json:"class"`
	ID              string   `json:"id"`
	Module          string   `json:"module,omitempty"`
	Version         string   `json:"version,omitempty"`
	Sum             string   `json:"sum,omitempty"`
	Platforms       []string `json:"platforms,omitempty"`
	SourceSetSHA256 string   `json:"source_set_sha256,omitempty"`
	Status          string   `json:"status"`
	Reason          string   `json:"reason"`
}

type InvalidPinImpactInput struct{ Err error }

func (err *InvalidPinImpactInput) Error() string { return err.Err.Error() }
func (err *InvalidPinImpactInput) Unwrap() error { return err.Err }
func invalidPinInput(err error) error            { return &InvalidPinImpactInput{Err: err} }

func (report PinImpact) Invalidated() bool {
	for _, pin := range report.Pins {
		if pin.Status == "invalidated" || pin.Status == "unknown" {
			return true
		}
	}
	return false
}

func (report PinImpact) CanonicalJSON() ([]byte, error) { return canonicaljson.CanonicalJSON(report) }

type pinModule struct {
	version, sum string
	replaced     bool
}
type pinModules struct {
	file    *modfile.File
	modules map[string]pinModule
	digest  string
}

func readPinModules(path string) (pinModules, error) {
	contents, err := os.ReadFile(path)
	if err != nil {
		return pinModules{}, invalidPinInput(errors.New("cannot read input go.mod"))
	}
	parsed, err := modfile.Parse("go.mod", contents, nil)
	if err != nil {
		return pinModules{}, invalidPinInput(errors.New("invalid input go.mod"))
	}
	if parsed.Module == nil || parsed.Module.Mod.Path == "" {
		return pinModules{}, invalidPinInput(errors.New("input go.mod has no module directive"))
	}
	sums, err := os.ReadFile(strings.TrimSuffix(path, filepath.Ext(path)) + ".sum")
	if err != nil {
		return pinModules{}, invalidPinInput(errors.New("cannot read input go.sum"))
	}
	knownSums := make(map[string]string)
	for _, line := range strings.Split(string(sums), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		if len(fields) != 3 {
			return pinModules{}, invalidPinInput(errors.New("invalid input go.sum"))
		}
		key := fields[0] + "@" + fields[1]
		if _, exists := knownSums[key]; exists {
			return pinModules{}, invalidPinInput(errors.New("duplicate input go.sum identity"))
		}
		knownSums[key] = fields[2]
	}
	result := pinModules{file: parsed, modules: make(map[string]pinModule), digest: fmt.Sprintf("sha256:%x", sha256.Sum256(append(append(append([]byte{}, contents...), 0), sums...)))}
	for _, req := range parsed.Require {
		if _, found := result.modules[req.Mod.Path]; found {
			return pinModules{}, invalidPinInput(errors.New("duplicate module requirement"))
		}
		module := pinModule{version: req.Mod.Version, sum: knownSums[req.Mod.Path+"@"+req.Mod.Version]}
		for _, replacement := range parsed.Replace {
			if replacement.Old.Path == req.Mod.Path {
				module.replaced = true
			}
		}
		result.modules[req.Mod.Path] = module
	}
	return result, nil
}

func modulePinStatus(module, version, sum string, candidate, baseline pinModules) (string, string) {
	actual, present := candidate.modules[module]
	if !present {
		if _, existed := baseline.modules[module]; existed {
			return "invalidated", "module_removed"
		}
		return "not_selected", "module_absent"
	}
	if actual.replaced {
		return "invalidated", "module_replaced"
	}
	if actual.version != version {
		return "invalidated", "version_changed"
	}
	if actual.sum == "" {
		return "unknown", "module_sum_missing"
	}
	if actual.sum != sum {
		return "invalidated", "sum_changed"
	}
	return "unaffected", "exact_module_identity"
}

func ReadPinImpact(root, candidatePath, baselinePath string) (PinImpact, error) {
	candidate, err := readPinModules(candidatePath)
	if err != nil {
		return PinImpact{}, err
	}
	baseline, err := readPinModules(baselinePath)
	if err != nil {
		return PinImpact{}, err
	}
	if candidate.file.Go == nil || baseline.file.Go == nil || candidate.file.Go.Version != baseline.file.Go.Version {
		return PinImpact{}, invalidPinInput(errors.New("Go directive changes require the upgrade dossier"))
	}
	candidateToolchain, baselineToolchain := "", ""
	if candidate.file.Toolchain != nil {
		candidateToolchain = candidate.file.Toolchain.Name
	}
	if baseline.file.Toolchain != nil {
		baselineToolchain = baseline.file.Toolchain.Name
	}
	if candidateToolchain != baselineToolchain {
		return PinImpact{}, invalidPinInput(errors.New("toolchain directive changes require the upgrade dossier"))
	}
	report := PinImpact{Schema: "gomad3.pin-impact/v1", CandidateSHA256: candidate.digest, BaselineSHA256: baseline.digest, Pins: []PinResult{}}
	for _, identity := range deterministicio.Default().Adapters() {
		status, reason := modulePinStatus(identity.Module, identity.Version, identity.Sum, candidate, baseline)
		report.Pins = append(report.Pins, PinResult{Class: "adapter", ID: identity.Module, Module: identity.Module, Version: identity.Version, Sum: identity.Sum, Platforms: gomadversion.SupportedPlatforms[:], Status: status, Reason: reason})
	}
	packs, err := compatibility.PinEvidence()
	if err != nil {
		return PinImpact{}, errors.New("cannot load compatibility pack pins")
	}
	for _, pack := range packs {
		relevant := activationVersionsMatch(pack.Activation, candidate) || activationVersionsMatch(pack.Activation, baseline)
		activationStatus, activationReason := "unaffected", "exact_module_identity"
		if relevant {
			for _, module := range pack.Activation {
				status, reason := modulePinStatus(module.Path, module.Version, module.Sum, candidate, baseline)
				if status == "invalidated" || status == "unknown" {
					activationStatus, activationReason = status, reason
					break
				}
			}
		}
		for _, rule := range pack.Rules {
			status, reason := "not_selected", "pack_variant_unselected"
			if relevant && (moduleVersionMatches(rule.Module, candidate) || moduleVersionMatches(rule.Module, baseline)) {
				status, reason = modulePinStatus(rule.Module.Path, rule.Module.Version, rule.Module.Sum, candidate, baseline)
				if activationStatus != "unaffected" {
					status, reason = activationStatus, "activation_"+activationReason
				}
			}
			report.Pins = append(report.Pins, PinResult{Class: "pack_rule", ID: pack.ID + ":" + rule.ImportPath, Module: rule.Module.Path, Version: rule.Module.Version, Sum: rule.Module.Sum, Platforms: pack.Governance.Platforms, SourceSetSHA256: rule.SourceSetSHA256, Status: status, Reason: reason})
		}
	}
	runtimePins, err := readRuntimePins(root)
	if err != nil {
		return PinImpact{}, err
	}
	report.Pins = append(report.Pins, runtimePins...)
	sort.Slice(report.Pins, func(i, j int) bool {
		if report.Pins[i].Class != report.Pins[j].Class {
			return report.Pins[i].Class < report.Pins[j].Class
		}
		return report.Pins[i].ID < report.Pins[j].ID
	})
	return report, nil
}

func activationVersionsMatch(activation []compatibility.ModuleEvidence, modules pinModules) bool {
	for _, module := range activation {
		if !moduleVersionMatches(module, modules) {
			return false
		}
	}
	return true
}

func moduleVersionMatches(module compatibility.ModuleEvidence, modules pinModules) bool {
	actual, found := modules.modules[module.Path]
	return found && actual.version == module.Version
}

func readRuntimePins(root string) ([]PinResult, error) {
	contents, err := os.ReadFile(filepath.Join(root, "deterministicio/boundary/manifest.json"))
	if err != nil {
		return nil, errors.New("cannot read interception manifest")
	}
	var manifest struct {
		GoVersion  string `json:"go_version"`
		Intercepts []struct {
			Package           string            `json:"package"`
			Symbol            string            `json:"symbol"`
			Receiver          *boundaryReceiver `json:"receiver"`
			Declaration       string            `json:"declaration_sha256"`
			PlatformOverrides map[string]struct {
				Declaration string `json:"declaration_sha256"`
			} `json:"platform_overrides"`
		} `json:"intercepts"`
	}
	if err := json.Unmarshal(contents, &manifest); err != nil || manifest.GoVersion != gomadversion.GoVersion || len(manifest.Intercepts) == 0 {
		return nil, errors.New("invalid interception manifest")
	}
	pins := make([]PinResult, 0, len(manifest.Intercepts))
	for _, intercept := range manifest.Intercepts {
		symbol := intercept.Symbol
		if intercept.Receiver != nil {
			symbol = intercept.Receiver.Name + "." + symbol
		}
		pins = append(pins, PinResult{Class: "interception_fingerprint", ID: intercept.Package + "." + symbol, Version: manifest.GoVersion, SourceSetSHA256: intercept.Declaration, Platforms: gomadversion.SupportedPlatforms[:], Status: "unaffected", Reason: "runtime_pin_dependency_bump_only"})
		for platform, override := range intercept.PlatformOverrides {
			pins = append(pins, PinResult{Class: "interception_fingerprint", ID: intercept.Package + "." + symbol + "@" + platform, Version: manifest.GoVersion, SourceSetSHA256: override.Declaration, Platforms: []string{platform}, Status: "unaffected", Reason: "runtime_pin_dependency_bump_only"})
		}
	}
	file, err := parser.ParseFile(token.NewFileSet(), filepath.Join(root, "toolchain/clock_inventory_test.go"), nil, 0)
	if err != nil {
		return nil, errors.New("cannot read clock inventory")
	}
	found := false
	for _, decl := range file.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}
		for _, spec := range gen.Specs {
			value, ok := spec.(*ast.ValueSpec)
			if !ok || len(value.Names) != 1 || value.Names[0].Name != "reviewedHostClockReferences" || len(value.Values) != 1 {
				continue
			}
			inventory, ok := value.Values[0].(*ast.CompositeLit)
			if !ok {
				return nil, errors.New("unknown clock inventory shape")
			}
			found = true
			for _, entry := range inventory.Elts {
				literal, ok := entry.(*ast.CompositeLit)
				if !ok || len(literal.Elts) != 6 {
					return nil, errors.New("unknown clock reference shape")
				}
				fields := make([]string, 3)
				for index := 0; index < 3; index++ {
					token, ok := literal.Elts[index].(*ast.BasicLit)
					if !ok {
						return nil, errors.New("unknown clock identity")
					}
					fields[index], err = strconv.Unquote(token.Value)
					if err != nil {
						return nil, errors.New("invalid clock identity")
					}
				}
				pins = append(pins, PinResult{Class: "clock_inventory_reference", ID: strings.Join(fields, " "), Version: gomadversion.GoVersion, Platforms: []string{fields[0]}, Status: "unaffected", Reason: "runtime_pin_dependency_bump_only"})
			}
		}
	}
	if !found {
		return nil, errors.New("clock inventory missing")
	}
	return pins, nil
}
