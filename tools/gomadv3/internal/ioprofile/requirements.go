package ioprofile

import (
	"sort"
	"strings"

	"go.temporal.io/server/tools/gomadv3/internal/record"
	"go.temporal.io/server/tools/gomadv3/internal/target"
)

type Requirement struct {
	Feature  string                              `json:"feature"`
	Modeled  bool                                `json:"modeled"`
	Packages []target.CapabilityPackageReference `json:"packages"`
}

func (profile ProfileSpec) Requirements(closure target.CapabilityClosure, adapters []record.TargetAdapter) ([]Requirement, error) {
	if _, err := profile.validated(); err != nil {
		return nil, err
	}
	if err := profile.VerifyAdapters(adapters); err != nil {
		return nil, err
	}
	evidence := packageRequirementEvidence(closure)
	if err := addAdapterRequirementEvidence(evidence, closure, adapters); err != nil {
		return nil, err
	}
	return projectRequirements(evidence), nil
}

func packageRequirementEvidence(closure target.CapabilityClosure) map[string]map[target.CapabilityPackageReference]struct{} {
	evidence := make(map[string]map[target.CapabilityPackageReference]struct{})
	for _, pkg := range closure.Packages {
		packageReference := target.CapabilityPackageReference{ImportPath: pkg.ImportPath, ForTest: pkg.ForTest, Name: pkg.Name}
		for _, imported := range pkg.Imports {
			for _, feature := range requirementFeatures(imported) {
				if evidence[feature] == nil {
					evidence[feature] = make(map[target.CapabilityPackageReference]struct{})
				}
				evidence[feature][packageReference] = struct{}{}
			}
		}
	}
	return evidence
}

func addAdapterRequirementEvidence(evidence map[string]map[target.CapabilityPackageReference]struct{}, closure target.CapabilityClosure, adapters []record.TargetAdapter) error {
	for _, adapter := range adapters {
		feature := "adapter:" + adapter.Module
		packages := make(map[target.CapabilityPackageReference]struct{})
		for _, pkg := range closure.Packages {
			if pkg.Module != nil && pkg.Module.Path == adapter.Module {
				packages[target.CapabilityPackageReference{ImportPath: pkg.ImportPath, ForTest: pkg.ForTest, Name: pkg.Name}] = struct{}{}
			}
		}
		if len(packages) != 0 {
			evidence[feature] = packages
		}
	}
	return nil
}

func projectRequirements(evidence map[string]map[target.CapabilityPackageReference]struct{}) []Requirement {
	features := make([]string, 0, len(evidence))
	for feature := range evidence {
		features = append(features, feature)
	}
	sort.Strings(features)
	result := make([]Requirement, len(features))
	for index, feature := range features {
		packages := make([]target.CapabilityPackageReference, 0, len(evidence[feature]))
		for pkg := range evidence[feature] {
			packages = append(packages, pkg)
		}
		sort.Slice(packages, func(i, j int) bool {
			if packages[i].ImportPath != packages[j].ImportPath {
				return packages[i].ImportPath < packages[j].ImportPath
			}
			if packages[i].ForTest != packages[j].ForTest {
				return packages[i].ForTest < packages[j].ForTest
			}
			return packages[i].Name < packages[j].Name
		})
		result[index] = Requirement{Feature: feature, Modeled: true, Packages: packages}
	}
	return result
}

func requirementFeatures(importPath string) []string {
	switch {
	case importPath == "crypto/rand":
		return []string{"entropy"}
	case importPath == "net" || strings.HasPrefix(importPath, "net/"):
		return []string{"loopback_tcp"}
	case importPath == "os" || importPath == "path/filepath" || importPath == "io/fs":
		return []string{"filesystem"}
	case importPath == "time":
		return []string{"time"}
	default:
		return nil
	}
}
