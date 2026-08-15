package packdev

import (
	"errors"
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	"go.temporal.io/server/tools/gomadv3/evidence"
	"go.temporal.io/server/tools/gomadv3/target"
	"go.temporal.io/server/tools/gomadv3/target/internal/compatibility"
	"golang.org/x/mod/module"
)

const RequestSchema = "gomadv3.compatibility-pack-request/v1"

const MaximumRequestBytes = 16 << 20

const (
	maximumRequestStringBytes = 4096
	maximumRequestArguments   = 256
	maximumRequestBuildTags   = 64
	maximumRequestActivations = 256
	maximumRequestPackages    = 4096
	maximumRequestFacts       = 16384
)

const emptySHA256 = "sha256:0000000000000000000000000000000000000000000000000000000000000000"

type Request struct {
	Schema         string       `json:"schema"`
	ID             string       `json:"id"`
	Target         Target       `json:"target"`
	Activation     []Activation `json:"activation"`
	Packages       []Package    `json:"packages"`
	Owner          string       `json:"owner"`
	ReviewedAt     string       `json:"reviewed_at"`
	Justification  string       `json:"justification"`
	Workloads      []string     `json:"workloads"`
	Platforms      []string     `json:"platforms"`
	ApprovalSHA256 string       `json:"approval_sha256"`
}

type Target struct {
	Kind           target.Kind `json:"kind"`
	Package        string      `json:"package"`
	TestArguments  []string    `json:"test_arguments"`
	BuildTags      []string    `json:"build_tags"`
	ExpectedModule string      `json:"expected_module"`
}

type Activation struct {
	Path     string                   `json:"path"`
	Evidence compatibility.PackModule `json:"evidence"`
}

type Package struct {
	ImportPath string                 `json:"import_path"`
	Facts      []Fact                 `json:"facts"`
	Evidence   compatibility.PackRule `json:"evidence"`
}

type FactKind string

const (
	FactCapability FactKind = "capability"
	FactLinkname   FactKind = "linkname"
)

type Disposition string

const (
	DispositionAllow Disposition = "allow"
	DispositionDeny  Disposition = "deny"
)

type Fact struct {
	Kind        FactKind    `json:"kind"`
	Capability  string      `json:"capability,omitempty"`
	Source      string      `json:"source,omitempty"`
	SHA256      string      `json:"sha256,omitempty"`
	Directives  []string    `json:"directives"`
	Disposition Disposition `json:"disposition"`
}

func (request Request) ReviewSpec(workingDirectory, toolchainRoot string) target.Spec {
	return target.Spec{
		Kind:          request.Target.Kind,
		Source:        request.Target.Package,
		Args:          append([]string{}, request.Target.TestArguments...),
		BuildTags:     append([]string{}, request.Target.BuildTags...),
		WorkingDir:    workingDirectory,
		ToolchainRoot: toolchainRoot,
	}
}

func DecodeRequest(data []byte) (Request, error) {
	if len(data) == 0 || len(data) > MaximumRequestBytes {
		return Request{}, fmt.Errorf("compatibility-pack request must be between 1 and %d bytes", MaximumRequestBytes)
	}
	var request Request
	if err := evidence.StrictDecode(data, &request); err != nil {
		return Request{}, fmt.Errorf("decode compatibility-pack request: %w", err)
	}
	if err := ValidateRequest(request); err != nil {
		return Request{}, err
	}
	return request, nil
}

func DecodeDraftRequest(data []byte) (Request, error) {
	if len(data) == 0 || len(data) > MaximumRequestBytes {
		return Request{}, fmt.Errorf("compatibility-pack draft request must be between 1 and %d bytes", MaximumRequestBytes)
	}
	var request Request
	if err := evidence.StrictDecode(data, &request); err != nil {
		return Request{}, fmt.Errorf("decode compatibility-pack draft request: %w", err)
	}
	if err := validateDraft(request); err != nil {
		return Request{}, err
	}
	return request, nil
}

func ValidateRequest(request Request) error {
	if request.Schema != RequestSchema {
		return errors.New("compatibility-pack request schema is unsupported")
	}
	if request.Target.Kind != target.KindGoRun && request.Target.Kind != target.KindGoTest {
		return errors.New("compatibility-pack request target kind is invalid")
	}
	if request.Target.Package == "" || len(request.Target.Package) > maximumRequestStringBytes || filepath.IsAbs(request.Target.Package) || strings.IndexByte(request.Target.Package, 0) >= 0 {
		return errors.New("compatibility-pack request target package is invalid")
	}
	if request.Target.TestArguments == nil || len(request.Target.TestArguments) > maximumRequestArguments ||
		request.Target.BuildTags == nil || len(request.Target.BuildTags) > maximumRequestBuildTags || !sortedUnique(request.Target.BuildTags) {
		return errors.New("compatibility-pack request target arguments or tags are not canonical")
	}
	for _, argument := range request.Target.TestArguments {
		if !validPortableArgument(argument) {
			return errors.New("compatibility-pack request target argument is invalid")
		}
	}
	if err := module.CheckPath(request.Target.ExpectedModule); err != nil {
		return fmt.Errorf("compatibility-pack request expected module is invalid: %w", err)
	}
	if len(request.Activation) == 0 || len(request.Activation) > maximumRequestActivations || len(request.Packages) == 0 || len(request.Packages) > maximumRequestPackages {
		return errors.New("compatibility-pack request has no activation or package selections")
	}
	factCount := 0
	for index, activation := range request.Activation {
		if activation.Path != activation.Evidence.Path || index > 0 && request.Activation[index-1].Path >= activation.Path {
			return errors.New("compatibility-pack request activations are not canonical")
		}
	}
	for index, pkg := range request.Packages {
		factCount += len(pkg.Facts)
		if factCount > maximumRequestFacts {
			return errors.New("compatibility-pack request fact count exceeds its bound")
		}
		if pkg.ImportPath != pkg.Evidence.ImportPath || index > 0 && request.Packages[index-1].ImportPath >= pkg.ImportPath || len(pkg.Facts) == 0 {
			return errors.New("compatibility-pack request packages are not canonical")
		}
		if len(pkg.Evidence.Capabilities) != 0 || len(pkg.Evidence.Linknames) != 0 {
			return errors.New("compatibility-pack request evidence contains policy decisions")
		}
		for factIndex, fact := range pkg.Facts {
			if factIndex > 0 && compareFact(pkg.Facts[factIndex-1], fact) >= 0 {
				return errors.New("compatibility-pack request facts are not sorted and unique")
			}
			if err := validateFact(fact); err != nil {
				return err
			}
		}
	}
	if request.ApprovalSHA256 != "" {
		if _, err := evidence.ParseSHA256(request.ApprovalSHA256); err != nil {
			return errors.New("compatibility-pack request approval is invalid")
		}
	}
	if _, err := projectPack(request, emptySHA256, emptySHA256, true); err != nil {
		return err
	}
	encoded, err := evidence.CanonicalJSON(request)
	if err != nil {
		return fmt.Errorf("encode compatibility-pack request: %w", err)
	}
	if len(encoded) > MaximumRequestBytes {
		return errors.New("compatibility-pack request exceeds its size bound")
	}
	return nil
}

func ApprovalSHA256(request Request) (string, error) {
	if err := ValidateRequest(request); err != nil {
		return "", err
	}
	projection := struct {
		Schema        string       `json:"schema"`
		ID            string       `json:"id"`
		Target        Target       `json:"target"`
		Activation    []Activation `json:"activation"`
		Packages      []Package    `json:"packages"`
		Owner         string       `json:"owner"`
		ReviewedAt    string       `json:"reviewed_at"`
		Justification string       `json:"justification"`
		Workloads     []string     `json:"workloads"`
		Platforms     []string     `json:"platforms"`
	}{
		Schema: request.Schema, ID: request.ID, Target: request.Target,
		Activation: request.Activation, Packages: request.Packages,
		Owner: request.Owner, ReviewedAt: request.ReviewedAt, Justification: request.Justification,
		Workloads: request.Workloads, Platforms: request.Platforms,
	}
	encoded, err := evidence.CanonicalJSON(projection)
	if err != nil {
		return "", fmt.Errorf("encode compatibility-pack approval projection: %w", err)
	}
	return string(evidence.DomainHash("gomadv3.compatibility-pack-review/v1", encoded)), nil
}

func projectPack(request Request, requestSHA256, approvalSHA256 string, includeDenied bool) (compatibility.Pack, error) {
	activation := make([]compatibility.PackModule, len(request.Activation))
	for index, selected := range request.Activation {
		activation[index] = selected.Evidence
	}
	rules := make([]compatibility.PackRule, len(request.Packages))
	for index, selected := range request.Packages {
		rule := selected.Evidence
		rule.Capabilities = []string{}
		rule.Linknames = []compatibility.PackLinkname{}
		for _, fact := range selected.Facts {
			if fact.Disposition != DispositionAllow && !includeDenied {
				continue
			}
			switch fact.Kind {
			case FactCapability:
				rule.Capabilities = append(rule.Capabilities, fact.Capability)
			case FactLinkname:
				rule.Linknames = append(rule.Linknames, compatibility.PackLinkname{
					Source: fact.Source, SHA256: fact.SHA256, Directives: append([]string{}, fact.Directives...),
				})
			}
		}
		rules[index] = rule
	}
	pack := compatibility.Pack{
		Schema: compatibility.PackSchema, ID: request.ID, RequestSHA256: requestSHA256,
		Governance: compatibility.PackGovernance{
			Owner: request.Owner, ReviewedAt: request.ReviewedAt, Justification: request.Justification,
			Workloads: append([]string{}, request.Workloads...), Platforms: append([]string{}, request.Platforms...), ApprovalSHA256: approvalSHA256,
		},
		Activation: activation, Rules: rules,
	}
	if err := compatibility.ValidatePack(pack); err != nil {
		return compatibility.Pack{}, fmt.Errorf("validate compatibility-pack request policy: %w", err)
	}
	return pack, nil
}

func validateFact(fact Fact) error {
	if fact.Disposition != DispositionAllow && fact.Disposition != DispositionDeny {
		return errors.New("compatibility-pack request fact disposition is invalid")
	}
	switch fact.Kind {
	case FactCapability:
		if fact.Capability == "" || fact.Source != "" || fact.SHA256 != "" || len(fact.Directives) != 0 {
			return errors.New("compatibility-pack capability fact is invalid")
		}
	case FactLinkname:
		if fact.Capability != "" || fact.Source == "" || len(fact.Directives) == 0 {
			return errors.New("compatibility-pack linkname fact is invalid")
		}
		if _, err := evidence.ParseSHA256(fact.SHA256); err != nil {
			return errors.New("compatibility-pack linkname fact source identity is invalid")
		}
	default:
		return errors.New("compatibility-pack request fact kind is invalid")
	}
	return nil
}

func compareFact(left, right Fact) int {
	if comparison := strings.Compare(string(left.Kind), string(right.Kind)); comparison != 0 {
		return comparison
	}
	if comparison := strings.Compare(left.Capability, right.Capability); comparison != 0 {
		return comparison
	}
	return strings.Compare(left.Source, right.Source)
}

func sortedUnique(values []string) bool {
	return slices.IsSorted(values) && !hasDuplicates(values)
}

func hasDuplicates(values []string) bool {
	for index, value := range values {
		if value == "" || index > 0 && values[index-1] == value {
			return true
		}
	}
	return false
}

func validPortableArgument(argument string) bool {
	if len(argument) > maximumRequestStringBytes || strings.IndexByte(argument, 0) >= 0 || filepath.IsAbs(argument) {
		return false
	}
	if _, value, found := strings.Cut(argument, "="); found && filepath.IsAbs(value) {
		return false
	}
	return true
}
