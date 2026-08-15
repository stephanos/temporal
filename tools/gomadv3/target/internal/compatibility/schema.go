package compatibility

import (
	"encoding/base64"
	"errors"
	"fmt"
	"path"
	"regexp"
	"slices"
	"strings"
	"time"

	"go.temporal.io/server/tools/gomadv3/evidence"
	"golang.org/x/mod/module"
)

const PackSchema = "gomadv3.compatibility-pack/v2"

const MaximumPackBytes = 16 << 20

const (
	maximumPackStringBytes = 4096
	maximumPackModules     = 256
	maximumPackRules       = 4096
	maximumPackSources     = 16384
	maximumPackFacts       = 16384
)

var (
	packIDPattern   = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,127}$`)
	platformPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]*/[a-z0-9][a-z0-9._-]*$`)
	workloadPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{0,255}$`)
)

type Pack struct {
	Schema        string         `json:"schema"`
	ID            string         `json:"id"`
	RequestSHA256 string         `json:"request_sha256"`
	Governance    PackGovernance `json:"governance"`
	Activation    []PackModule   `json:"activation"`
	Rules         []PackRule     `json:"rules"`
}

type PackGovernance struct {
	Owner          string   `json:"owner"`
	ReviewedAt     string   `json:"reviewed_at"`
	Justification  string   `json:"justification"`
	Workloads      []string `json:"workloads"`
	Platforms      []string `json:"platforms"`
	ApprovalSHA256 string   `json:"approval_sha256"`
}

type PackModule struct {
	Path        string          `json:"path"`
	Version     string          `json:"version"`
	Sum         string          `json:"sum"`
	Replacement PackReplacement `json:"replacement"`
}

type ReplacementKind string

const (
	ReplacementNone    ReplacementKind = "none"
	ReplacementAdapter ReplacementKind = "adapter"
)

type PackReplacement struct {
	Kind    ReplacementKind `json:"kind"`
	Adapter *PackAdapter    `json:"adapter,omitempty"`
}

type PackAdapter struct {
	ProfileName                      string `json:"profile_name"`
	ProfileImplementationSHA256      string `json:"profile_implementation_sha256"`
	Module                           string `json:"module"`
	Version                          string `json:"version"`
	Sum                              string `json:"sum"`
	OriginalSourceInventorySHA256    string `json:"original_source_inventory_sha256"`
	ReplacementSourceInventorySHA256 string `json:"replacement_source_inventory_sha256"`
	PreparedSourceSetSHA256          string `json:"prepared_source_set_sha256"`
}

type PackRule struct {
	ImportPath      string              `json:"import_path"`
	Module          PackModule          `json:"module"`
	SourceSetSHA256 string              `json:"source_set_sha256"`
	GoSources       []PackSource        `json:"go_sources"`
	ForeignSources  []PackForeignSource `json:"foreign_sources"`
	Capabilities    []string            `json:"capabilities"`
	Linknames       []PackLinkname      `json:"linknames"`
}

type PackSource struct {
	Name   string `json:"name"`
	SHA256 string `json:"sha256"`
}

type PackForeignSource struct {
	Kind   string `json:"kind"`
	Name   string `json:"name"`
	SHA256 string `json:"sha256"`
}

type PackLinkname struct {
	Source     string   `json:"source"`
	SHA256     string   `json:"sha256"`
	Directives []string `json:"directives"`
}

func DecodePack(data []byte) (Pack, error) {
	if len(data) == 0 || len(data) > MaximumPackBytes {
		return Pack{}, fmt.Errorf("compatibility pack must be between 1 and %d bytes", MaximumPackBytes)
	}
	var decoded Pack
	if err := evidence.StrictDecode(data, &decoded); err != nil {
		return Pack{}, fmt.Errorf("decode compatibility pack: %w", err)
	}
	if err := ValidatePack(decoded); err != nil {
		return Pack{}, err
	}
	return decoded, nil
}

func ValidatePack(candidate Pack) error {
	if candidate.Schema != PackSchema || !packIDPattern.MatchString(candidate.ID) {
		return errors.New("compatibility pack identity is invalid")
	}
	if err := validateDigest(candidate.RequestSHA256); err != nil {
		return fmt.Errorf("compatibility pack request identity is invalid: %w", err)
	}
	if err := validatePackGovernance(candidate.Governance); err != nil {
		return err
	}
	if len(candidate.Activation) == 0 || len(candidate.Activation) > maximumPackModules {
		return errors.New("compatibility pack activation count is invalid")
	}
	if len(candidate.Rules) == 0 || len(candidate.Rules) > maximumPackRules {
		return errors.New("compatibility pack rule count is invalid")
	}
	for index, activation := range candidate.Activation {
		if index > 0 && comparePackModule(candidate.Activation[index-1], activation) >= 0 {
			return errors.New("compatibility pack activations are not sorted and unique")
		}
		if err := validatePackModule(activation); err != nil {
			return fmt.Errorf("compatibility pack activation %d: %w", index, err)
		}
	}
	for index, rule := range candidate.Rules {
		if index > 0 && comparePackRule(candidate.Rules[index-1], rule) >= 0 {
			return errors.New("compatibility pack rules are not sorted and unique")
		}
		if err := validatePackRule(rule); err != nil {
			return fmt.Errorf("compatibility pack rule %d: %w", index, err)
		}
	}
	encoded, err := evidence.CanonicalJSON(candidate)
	if err != nil {
		return fmt.Errorf("encode compatibility pack: %w", err)
	}
	if len(encoded) > MaximumPackBytes {
		return errors.New("compatibility pack exceeds its size bound")
	}
	return nil
}

func validatePackGovernance(governance PackGovernance) error {
	if err := validateText("owner", governance.Owner, 1, 256); err != nil {
		return fmt.Errorf("compatibility pack governance: %w", err)
	}
	if err := validateText("justification", governance.Justification, 1, maximumPackStringBytes); err != nil {
		return fmt.Errorf("compatibility pack governance: %w", err)
	}
	reviewedAt, err := time.Parse(time.RFC3339, governance.ReviewedAt)
	if err != nil || !strings.HasSuffix(governance.ReviewedAt, "Z") || reviewedAt.Location() != time.UTC {
		return errors.New("compatibility pack governance review time is invalid")
	}
	if len(governance.Workloads) == 0 || len(governance.Workloads) > maximumPackFacts || !sortedUniqueBy(governance.Workloads, workloadPattern.MatchString) {
		return errors.New("compatibility pack governance workloads are not canonical")
	}
	if len(governance.Platforms) == 0 || len(governance.Platforms) > maximumPackModules || !sortedUniqueBy(governance.Platforms, platformPattern.MatchString) {
		return errors.New("compatibility pack governance platforms are not canonical")
	}
	if err := validateDigest(governance.ApprovalSHA256); err != nil {
		return fmt.Errorf("compatibility pack governance approval is invalid: %w", err)
	}
	return nil
}

func validatePackModule(candidate PackModule) error {
	if err := module.Check(candidate.Path, candidate.Version); err != nil {
		return fmt.Errorf("module identity is invalid: %w", err)
	}
	if !validModuleSum(candidate.Sum) {
		return errors.New("module sum is invalid")
	}
	switch candidate.Replacement.Kind {
	case ReplacementNone:
		if candidate.Replacement.Adapter != nil {
			return errors.New("unreplaced module has adapter evidence")
		}
	case ReplacementAdapter:
		if candidate.Replacement.Adapter == nil {
			return errors.New("adapter replacement evidence is missing")
		}
		if err := validatePackAdapter(*candidate.Replacement.Adapter); err != nil {
			return err
		}
	default:
		return fmt.Errorf("module replacement kind %q is invalid", candidate.Replacement.Kind)
	}
	return nil
}

func validatePackAdapter(adapter PackAdapter) error {
	if err := validateText("adapter profile name", adapter.ProfileName, 1, 256); err != nil {
		return err
	}
	if err := module.Check(adapter.Module, adapter.Version); err != nil || !validModuleSum(adapter.Sum) {
		return errors.New("adapter module identity is invalid")
	}
	for name, digest := range map[string]string{
		"adapter profile implementation":       adapter.ProfileImplementationSHA256,
		"adapter original source inventory":    adapter.OriginalSourceInventorySHA256,
		"adapter replacement source inventory": adapter.ReplacementSourceInventorySHA256,
		"adapter prepared source set":          adapter.PreparedSourceSetSHA256,
	} {
		if err := validateDigest(digest); err != nil {
			return fmt.Errorf("%s identity is invalid: %w", name, err)
		}
	}
	return nil
}

func validatePackRule(rule PackRule) error {
	if err := module.CheckImportPath(rule.ImportPath); err != nil {
		return fmt.Errorf("import path is invalid: %w", err)
	}
	if err := validatePackModule(rule.Module); err != nil {
		return err
	}
	if err := validateDigest(rule.SourceSetSHA256); err != nil {
		return fmt.Errorf("source-set identity is invalid: %w", err)
	}
	if len(rule.GoSources) == 0 || len(rule.GoSources)+len(rule.ForeignSources) > maximumPackSources {
		return errors.New("source inventory count is invalid")
	}
	if rule.ForeignSources == nil || rule.Capabilities == nil || rule.Linknames == nil {
		return errors.New("rule collections must not be null")
	}
	if !slices.IsSortedFunc(rule.GoSources, comparePackSource) || hasDuplicatePackSources(rule.GoSources) {
		return errors.New("Go source inventory is not sorted and unique")
	}
	for _, source := range rule.GoSources {
		if err := validatePackSource(source); err != nil {
			return err
		}
	}
	if !slices.IsSortedFunc(rule.ForeignSources, comparePackForeignSource) || hasDuplicatePackForeignSources(rule.ForeignSources) {
		return errors.New("foreign source inventory is not sorted and unique")
	}
	for _, source := range rule.ForeignSources {
		if err := validatePackForeignSource(source); err != nil {
			return err
		}
	}
	sources := make([]Source, 0, len(rule.GoSources)+len(rule.ForeignSources))
	for _, source := range rule.GoSources {
		sources = append(sources, Source{Name: source.Name, SHA256: source.SHA256})
	}
	for _, source := range rule.ForeignSources {
		sources = append(sources, Source{Name: source.Kind + ":" + source.Name, SHA256: source.SHA256})
	}
	if DigestSources(sources) != rule.SourceSetSHA256 {
		return errors.New("source-set identity does not match the source inventories")
	}
	if len(rule.Capabilities)+len(rule.Linknames) == 0 || len(rule.Capabilities)+len(rule.Linknames) > maximumPackFacts || !sortedUniqueBy(rule.Capabilities, validCapability) {
		return errors.New("capability inventory is not canonical")
	}
	if !slices.IsSortedFunc(rule.Linknames, comparePackLinkname) || hasDuplicatePackLinknames(rule.Linknames) {
		return errors.New("linkname inventory is not sorted and unique")
	}
	for _, linkname := range rule.Linknames {
		if err := validatePackLinkname(linkname); err != nil {
			return err
		}
	}
	return nil
}

func validatePackSource(source PackSource) error {
	if !validSourceName(source.Name) {
		return errors.New("Go source name is invalid")
	}
	if err := validateDigest(source.SHA256); err != nil {
		return fmt.Errorf("Go source identity is invalid: %w", err)
	}
	return nil
}

func validatePackForeignSource(source PackForeignSource) error {
	if !validSourceName(source.Name) || !workloadPattern.MatchString(source.Kind) {
		return errors.New("foreign source identity is invalid")
	}
	if err := validateDigest(source.SHA256); err != nil {
		return fmt.Errorf("foreign source identity is invalid: %w", err)
	}
	return nil
}

func validatePackLinkname(linkname PackLinkname) error {
	if !validSourceName(linkname.Source) {
		return errors.New("linkname source is invalid")
	}
	if err := validateDigest(linkname.SHA256); err != nil {
		return fmt.Errorf("linkname source identity is invalid: %w", err)
	}
	if len(linkname.Directives) == 0 || len(linkname.Directives) > maximumPackFacts {
		return errors.New("linkname directives are missing or oversized")
	}
	seen := make(map[string]struct{}, len(linkname.Directives))
	for _, directive := range linkname.Directives {
		if err := validateText("linkname directive", directive, 1, maximumPackStringBytes); err != nil {
			return err
		}
		if len(strings.Fields(directive)) != 2 {
			return errors.New("linkname directive is invalid")
		}
		if _, duplicate := seen[directive]; duplicate {
			return errors.New("linkname directives are not unique")
		}
		seen[directive] = struct{}{}
	}
	return nil
}

func validateDigest(value string) error {
	_, err := evidence.ParseSHA256(value)
	return err
}

func validateText(name, value string, minimum, maximum int) error {
	if len(value) < minimum || len(value) > maximum || strings.IndexByte(value, 0) >= 0 {
		return fmt.Errorf("%s is invalid", name)
	}
	for _, character := range value {
		if character < 0x20 && character != '\n' && character != '\t' {
			return fmt.Errorf("%s is invalid", name)
		}
	}
	return nil
}

func validModuleSum(sum string) bool {
	if !strings.HasPrefix(sum, "h1:") {
		return false
	}
	decoded, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(sum, "h1:"))
	return err == nil && len(decoded) == 32
}

func validSourceName(name string) bool {
	return name != "" && len(name) <= maximumPackStringBytes && path.Base(name) == name && name != "." && name != ".."
}

func validCapability(value string) bool {
	if len(value) == 0 || len(value) > maximumPackStringBytes {
		return false
	}
	prefix, detail, found := strings.Cut(value, ":")
	return found && detail != "" && (prefix == "import" || prefix == "foreign")
}

func sortedUniqueBy(values []string, valid func(string) bool) bool {
	for index, value := range values {
		if !valid(value) || index > 0 && values[index-1] >= value {
			return false
		}
	}
	return true
}

func comparePackModule(left, right PackModule) int {
	if comparison := strings.Compare(left.Path, right.Path); comparison != 0 {
		return comparison
	}
	return strings.Compare(left.Version, right.Version)
}

func comparePackRule(left, right PackRule) int {
	if comparison := strings.Compare(left.ImportPath, right.ImportPath); comparison != 0 {
		return comparison
	}
	return comparePackModule(left.Module, right.Module)
}

func comparePackSource(left, right PackSource) int {
	return strings.Compare(left.Name, right.Name)
}

func comparePackForeignSource(left, right PackForeignSource) int {
	if comparison := strings.Compare(left.Kind, right.Kind); comparison != 0 {
		return comparison
	}
	return strings.Compare(left.Name, right.Name)
}

func comparePackLinkname(left, right PackLinkname) int {
	return strings.Compare(left.Source, right.Source)
}

func hasDuplicatePackSources(values []PackSource) bool {
	for index := 1; index < len(values); index++ {
		if comparePackSource(values[index-1], values[index]) == 0 {
			return true
		}
	}
	return false
}

func hasDuplicatePackForeignSources(values []PackForeignSource) bool {
	for index := 1; index < len(values); index++ {
		if comparePackForeignSource(values[index-1], values[index]) == 0 {
			return true
		}
	}
	return false
}

func hasDuplicatePackLinknames(values []PackLinkname) bool {
	for index := 1; index < len(values); index++ {
		if comparePackLinkname(values[index-1], values[index]) == 0 {
			return true
		}
	}
	return false
}
