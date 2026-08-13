package compatibility

import (
	"bytes"
	"crypto/sha256"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"slices"
	"sort"

	"go.temporal.io/server/tools/gomadv3/internal/record"
)

const packSchema = "gomadv3.compatibility-pack/v1"

//go:embed packs/*.json
var packFiles embed.FS

type Identity struct {
	ID     string `json:"id"`
	SHA256 string `json:"sha256"`
}

type Module struct {
	Path             string
	Version          string
	Sum              string
	Replaced         bool
	LocalReplacement bool
}

type Package struct {
	ImportPath      string
	Module          Module
	SourceSetSHA256 string
}

type Source struct {
	Name   string
	SHA256 string
}

type Selection struct {
	packs []loadedPack
}

type pack struct {
	Schema     string          `json:"schema"`
	ID         string          `json:"id"`
	Activation []modulePattern `json:"activation"`
	Rules      []rule          `json:"rules"`
}

type loadedPack struct {
	pack
	digest string
}

type modulePattern struct {
	Path        string `json:"path"`
	Version     string `json:"version"`
	Sum         string `json:"sum"`
	Replacement string `json:"replacement"`
}

type rule struct {
	ImportPath      string         `json:"import_path"`
	Module          modulePattern  `json:"module"`
	SourceSetSHA256 string         `json:"source_set_sha256,omitempty"`
	Capabilities    []string       `json:"capabilities"`
	Linknames       []linknameRule `json:"linknames"`
}

type linknameRule struct {
	Source     string   `json:"source"`
	SHA256     string   `json:"sha256"`
	Directives []string `json:"directives"`
}

func Select(packages []Package) (Selection, error) {
	packs, err := loadPacks()
	if err != nil {
		return Selection{}, err
	}
	selected := make([]loadedPack, 0, len(packs))
	for _, candidate := range packs {
		if matchesActivation(candidate.Activation, packages) {
			selected = append(selected, candidate)
		}
	}
	return Selection{packs: selected}, nil
}

func (selection Selection) Identities() []Identity {
	identities := make([]Identity, len(selection.packs))
	for index, selected := range selection.packs {
		identities[index] = Identity{ID: selected.ID, SHA256: selected.digest}
	}
	return identities
}

func VerifyIdentities(identities []Identity) error {
	packs, err := loadPacks()
	if err != nil {
		return err
	}
	available := make(map[string]string, len(packs))
	for _, candidate := range packs {
		available[candidate.ID] = candidate.digest
	}
	for index, identity := range identities {
		if index > 0 && identities[index-1].ID >= identity.ID {
			return errors.New("compatibility pack identities are not sorted and unique")
		}
		if digest, found := available[identity.ID]; !found || digest != identity.SHA256 {
			return fmt.Errorf("compatibility pack %s is unavailable or modified", identity.ID)
		}
	}
	return nil
}

func (selection Selection) AllowsCapability(pkg Package, capability string) bool {
	for _, selected := range selection.packs {
		for _, candidate := range selected.Rules {
			if matchesRule(candidate, pkg) && slices.Contains(candidate.Capabilities, capability) {
				return true
			}
		}
	}
	return false
}

func (selection Selection) HasPackage(pkg Package) bool {
	for _, selected := range selection.packs {
		for _, candidate := range selected.Rules {
			if matchesRule(candidate, pkg) {
				return true
			}
		}
	}
	return false
}

func (selection Selection) AllowsLinkname(pkg Package, source, digest string, directives []string) bool {
	for _, selected := range selection.packs {
		for _, candidate := range selected.Rules {
			if !matchesRule(candidate, pkg) {
				continue
			}
			for _, allowed := range candidate.Linknames {
				if allowed.Source == source && allowed.SHA256 == digest && slices.Equal(allowed.Directives, directives) {
					return true
				}
			}
		}
	}
	return false
}

func DigestSources(sources []Source) string {
	hash := sha256.New()
	_, _ = hash.Write([]byte("gomadv3.compatibility-source-set/v1\x00"))
	for _, source := range sources {
		_, _ = hash.Write([]byte(source.Name))
		_, _ = hash.Write([]byte{0})
		_, _ = hash.Write([]byte(source.SHA256))
		_, _ = hash.Write([]byte{0})
	}
	return fmt.Sprintf("sha256:%x", hash.Sum(nil))
}

func loadPacks() ([]loadedPack, error) {
	entries, err := packFiles.ReadDir("packs")
	if err != nil {
		return nil, fmt.Errorf("read compatibility packs: %w", err)
	}
	result := make([]loadedPack, 0, len(entries))
	seen := make(map[string]struct{}, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			return nil, fmt.Errorf("compatibility pack entry %s is a directory", entry.Name())
		}
		contents, err := packFiles.ReadFile("packs/" + entry.Name())
		if err != nil {
			return nil, err
		}
		decoder := json.NewDecoder(bytes.NewReader(contents))
		decoder.DisallowUnknownFields()
		var decoded pack
		if err := decoder.Decode(&decoded); err != nil {
			return nil, fmt.Errorf("decode compatibility pack %s: %w", entry.Name(), err)
		}
		if err := decoder.Decode(&struct{}{}); err != io.EOF {
			return nil, fmt.Errorf("compatibility pack %s has trailing data", entry.Name())
		}
		if err := validatePack(decoded); err != nil {
			return nil, fmt.Errorf("compatibility pack %s: %w", entry.Name(), err)
		}
		if _, duplicate := seen[decoded.ID]; duplicate {
			return nil, fmt.Errorf("compatibility pack ID is duplicated: %s", decoded.ID)
		}
		seen[decoded.ID] = struct{}{}
		hash := sha256.Sum256(contents)
		result = append(result, loadedPack{pack: decoded, digest: fmt.Sprintf("sha256:%x", hash)})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result, nil
}

func validatePack(candidate pack) error {
	if candidate.Schema != packSchema || candidate.ID == "" || len(candidate.Activation) == 0 || len(candidate.Rules) == 0 {
		return errors.New("identity is invalid")
	}
	for _, activation := range candidate.Activation {
		if err := validateModulePattern(activation); err != nil {
			return err
		}
	}
	for index, candidateRule := range candidate.Rules {
		if candidateRule.ImportPath == "" {
			return fmt.Errorf("rule %d has no import path", index)
		}
		if err := validateModulePattern(candidateRule.Module); err != nil {
			return err
		}
		if candidateRule.SourceSetSHA256 != "" {
			if _, err := record.ParseSHA256(candidateRule.SourceSetSHA256); err != nil {
				return fmt.Errorf("rule %s has an invalid source-set identity", candidateRule.ImportPath)
			}
		}
		if candidateRule.Module.Replacement == "local" && len(candidateRule.Capabilities) != 0 && candidateRule.SourceSetSHA256 == "" {
			return fmt.Errorf("rule %s has no source-set identity for a local replacement", candidateRule.ImportPath)
		}
		if !sortedUnique(candidateRule.Capabilities) {
			return fmt.Errorf("rule %s capabilities are not sorted and unique", candidateRule.ImportPath)
		}
		for _, linkname := range candidateRule.Linknames {
			if _, err := record.ParseSHA256(linkname.SHA256); linkname.Source == "" || err != nil || len(linkname.Directives) == 0 {
				return fmt.Errorf("rule %s has an invalid linkname source", candidateRule.ImportPath)
			}
		}
	}
	return nil
}

func validateModulePattern(pattern modulePattern) error {
	if pattern.Path == "" || pattern.Version == "" || pattern.Replacement != "none" && pattern.Replacement != "local" || pattern.Replacement == "none" && pattern.Sum == "" {
		return errors.New("module identity is invalid")
	}
	return nil
}

func matchesActivation(patterns []modulePattern, packages []Package) bool {
	for _, pattern := range patterns {
		found := false
		for _, pkg := range packages {
			if matchesModule(pattern, pkg.Module) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func matchesModule(pattern modulePattern, module Module) bool {
	if module.Path != pattern.Path || module.Version != pattern.Version || module.Sum != pattern.Sum {
		return false
	}
	if pattern.Replacement == "local" {
		return module.LocalReplacement
	}
	return !module.Replaced && !module.LocalReplacement
}

func matchesRule(candidate rule, pkg Package) bool {
	return candidate.ImportPath == pkg.ImportPath && matchesModule(candidate.Module, pkg.Module) && (candidate.SourceSetSHA256 == "" || candidate.SourceSetSHA256 == pkg.SourceSetSHA256)
}

func sortedUnique(values []string) bool {
	for index, value := range values {
		if value == "" || index > 0 && values[index-1] >= value {
			return false
		}
	}
	return true
}
