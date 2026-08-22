package release

import (
	"encoding/json"
	"errors"
	"fmt"
	"slices"
)

const ReleaseAssuranceFormatVersion = "umpire3/release-assurance/v1"

type ReleaseEvidenceNode struct {
	Identifier  string      `json:"identifier"`
	ResultClass ResultClass `json:"resultClass"`
	TrustBadge  TrustBadge  `json:"trustBadge"`
	Digest      string      `json:"digest"`
}

type ReleaseEvidenceGoal struct {
	Identifier string   `json:"identifier"`
	Requires   []string `json:"requires"`
	Omissions  []string `json:"omissions"`
}

type ReleaseAssurance struct {
	FormatVersion string                `json:"formatVersion"`
	Digest        string                `json:"digest"`
	Nodes         []ReleaseEvidenceNode `json:"nodes"`
	Goals         []ReleaseEvidenceGoal `json:"goals"`
}

var requiredVisionGoals = []string{
	"clock-skew-safety",
	"coverage-guided-fuzzing",
	"deterministic-plans",
	"developer-authoring",
	"first-class-faults",
	"guided-exploration",
	"known-regression-verification",
	"non-linear-identity",
	"portable-profiles",
	"programmable-participants",
	"single-semantic-model",
	"unknown-bug-exploration",
	"white-box-black-box",
}

var requiredReleaseProfiles = []string{
	"ci-test-cluster",
	"grpc-only-black-box",
	"local-in-process",
	"production-canary",
	"remote-deployment",
}

func (a ReleaseAssurance) Validate() error {
	if a.FormatVersion != ReleaseAssuranceFormatVersion || !validHash(a.Digest) ||
		len(a.Nodes) == 0 || len(a.Goals) == 0 {
		return errors.New("complete release assurance identity, nodes, and goals are required")
	}
	if !slices.IsSortedFunc(a.Nodes, func(left, right ReleaseEvidenceNode) int {
		return stringCompare(left.Identifier, right.Identifier)
	}) {
		return errors.New("release assurance nodes must be sorted")
	}
	nodes := make(map[string]struct{}, len(a.Nodes))
	for _, node := range a.Nodes {
		if node.Identifier == "" || !node.ResultClass.Valid() || !node.TrustBadge.Valid() ||
			!validHash(node.Digest) {
			return errors.New("release assurance nodes require typed result, trust, and digest evidence")
		}
		if _, duplicate := nodes[node.Identifier]; duplicate {
			return fmt.Errorf("release assurance contains duplicate node %q", node.Identifier)
		}
		nodes[node.Identifier] = struct{}{}
	}
	if !slices.IsSortedFunc(a.Goals, func(left, right ReleaseEvidenceGoal) int {
		return stringCompare(left.Identifier, right.Identifier)
	}) {
		return errors.New("release assurance goals must be sorted")
	}
	seenGoals := make(map[string]struct{}, len(a.Goals))
	referencedNodes := make(map[string]struct{}, len(a.Nodes))
	for _, goal := range a.Goals {
		if !slices.Contains(requiredVisionGoals, goal.Identifier) {
			return fmt.Errorf("release assurance contains unknown vision goal %q", goal.Identifier)
		}
		if _, duplicate := seenGoals[goal.Identifier]; duplicate {
			return fmt.Errorf("release assurance contains duplicate vision goal %q", goal.Identifier)
		}
		seenGoals[goal.Identifier] = struct{}{}
		if len(goal.Requires) == 0 {
			return fmt.Errorf("release vision goal %q has no evidence requirements", goal.Identifier)
		}
		if err := validateSortedUniqueStrings(goal.Requires, "release assurance requirements"); err != nil {
			return fmt.Errorf("release vision goal %q: %w", goal.Identifier, err)
		}
		if err := validateSortedUniqueStrings(goal.Omissions, "release assurance omissions"); err != nil {
			return fmt.Errorf("release vision goal %q: %w", goal.Identifier, err)
		}
		for _, requirement := range goal.Requires {
			if _, exists := nodes[requirement]; !exists {
				return fmt.Errorf("release vision goal %q requires missing node %q", goal.Identifier, requirement)
			}
			referencedNodes[requirement] = struct{}{}
		}
	}
	if len(seenGoals) != len(requiredVisionGoals) {
		return errors.New("release assurance does not disposition every Umpire vision goal")
	}
	if len(referencedNodes) != len(nodes) {
		return errors.New("release assurance contains an unreferenced evidence node")
	}
	expectedDigest, err := a.computedDigest()
	if err != nil {
		return err
	}
	if a.Digest != expectedDigest {
		return errors.New("release assurance digest does not match its graph")
	}
	return nil
}

func (a ReleaseAssurance) Complete() bool {
	if err := a.Validate(); err != nil {
		return false
	}
	for _, goal := range a.Goals {
		if len(goal.Omissions) != 0 {
			return false
		}
	}
	return true
}

func SealReleaseAssurance(assurance ReleaseAssurance) (ReleaseAssurance, error) {
	assurance.FormatVersion = ReleaseAssuranceFormatVersion
	slices.SortFunc(assurance.Nodes, func(left, right ReleaseEvidenceNode) int {
		return stringCompare(left.Identifier, right.Identifier)
	})
	slices.SortFunc(assurance.Goals, func(left, right ReleaseEvidenceGoal) int {
		return stringCompare(left.Identifier, right.Identifier)
	})
	for index := range assurance.Goals {
		slices.Sort(assurance.Goals[index].Requires)
		slices.Sort(assurance.Goals[index].Omissions)
	}
	digest, err := assurance.computedDigest()
	if err != nil {
		return ReleaseAssurance{}, err
	}
	assurance.Digest = digest
	if err := assurance.Validate(); err != nil {
		return ReleaseAssurance{}, err
	}
	return assurance, nil
}

func (a ReleaseAssurance) computedDigest() (string, error) {
	payload := struct {
		FormatVersion string                `json:"formatVersion"`
		Nodes         []ReleaseEvidenceNode `json:"nodes"`
		Goals         []ReleaseEvidenceGoal `json:"goals"`
	}{
		FormatVersion: a.FormatVersion,
		Nodes:         a.Nodes,
		Goals:         a.Goals,
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("encode release assurance digest payload: %w", err)
	}
	return digestBytes(encoded), nil
}

func validateSortedUniqueStrings(values []string, name string) error {
	if !slices.IsSorted(values) || len(slices.Compact(append([]string(nil), values...))) != len(values) {
		return fmt.Errorf("%s must be sorted and unique", name)
	}
	for _, value := range values {
		if value == "" {
			return fmt.Errorf("%s contain an empty value", name)
		}
	}
	return nil
}
