package checker

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"slices"

	"go.temporal.io/server/tests/umpire3/protocol/internal/generated"
)

const ProofManifestFormatVersion = "umpire3/proof-manifest/v3"

var defaultNexusProofManifestJSON = generated.Read(generated.NexusProofManifest)
var defaultNexusMutationRejectionProofManifestJSON = generated.Read(generated.NexusMutationProofManifest)
var defaultNexusExactMutationProofManifestJSON = generated.Read(generated.NexusExactMutationProofManifest)
var defaultUpdateProofManifestJSON = generated.Read(generated.UpdateProofManifest)

type ProofDependency struct {
	Identifier    string `json:"identifier"`
	StatementHash string `json:"statementHash"`
}

type SourceDependency struct {
	Path    string   `json:"path"`
	Digest  string   `json:"digest"`
	Imports []string `json:"imports"`
}

type ProofManifest struct {
	FormatVersion      string             `json:"formatVersion"`
	Identifier         string             `json:"identifier"`
	Theorem            string             `json:"theorem"`
	Statement          string             `json:"statement"`
	StatementHash      string             `json:"statementHash"`
	ResultClass        ResultClass        `json:"resultClass"`
	TrustBadge         TrustBadge         `json:"trustBadge"`
	Axioms             []string           `json:"axioms"`
	SemanticHash       string             `json:"semanticHash"`
	SourceDigest       string             `json:"sourceDigest"`
	DependencyDigest   string             `json:"dependencyDigest"`
	SourceDependencies []SourceDependency `json:"sourceDependencies"`
	LeanVersion        string             `json:"leanVersion"`
	Assumptions        []ProofDependency  `json:"assumptions"`
}

func NewProofManifest(
	identifier string,
	theorem string,
	statement string,
	resultClass ResultClass,
	axioms []string,
	leanVersion string,
	assumptions []ProofDependency,
	dependencies []SourceDependency,
) (ProofManifest, error) {
	dependencies = append([]SourceDependency(nil), dependencies...)
	for index := range dependencies {
		dependencies[index].Imports = append([]string(nil), dependencies[index].Imports...)
		slices.Sort(dependencies[index].Imports)
		dependencies[index].Imports = slices.Compact(dependencies[index].Imports)
	}
	slices.SortFunc(dependencies, func(left, right SourceDependency) int {
		return stringCompare(left.Path, right.Path)
	})
	axioms = append([]string(nil), axioms...)
	slices.Sort(axioms)
	axioms = slices.Compact(axioms)
	trustBadge := TrustBadgeKernel
	if len(axioms) != 0 {
		trustBadge = TrustBadgeKernelWithDeclaredAxioms
	}
	sourceHash, dependencyHash, err := DigestSourceDependencies(dependencies)
	if err != nil {
		return ProofManifest{}, err
	}
	manifest := ProofManifest{
		FormatVersion: ProofManifestFormatVersion, Identifier: identifier, Theorem: theorem,
		Statement: statement, StatementHash: digestBytes([]byte(statement)), ResultClass: resultClass,
		TrustBadge: trustBadge, Axioms: axioms, SemanticHash: sourceHash, SourceDigest: sourceHash,
		DependencyDigest: dependencyHash, SourceDependencies: dependencies,
		LeanVersion: leanVersion, Assumptions: append([]ProofDependency(nil), assumptions...),
	}
	if err := manifest.Validate(); err != nil {
		return ProofManifest{}, err
	}
	return manifest, nil
}

func DecodeProofManifest(reader io.Reader, limit int64) (ProofManifest, error) {
	var manifest ProofManifest
	if err := decodeStrictJSON(reader, limit, "proof manifest", &manifest); err != nil {
		return ProofManifest{}, err
	}
	if err := manifest.Validate(); err != nil {
		return ProofManifest{}, err
	}
	return manifest, nil
}

func DefaultProofManifests() ([]ProofManifest, error) {
	encodedManifests := [][]byte{
		defaultNexusProofManifestJSON,
		defaultNexusMutationRejectionProofManifestJSON,
		defaultNexusExactMutationProofManifestJSON,
		defaultUpdateProofManifestJSON,
	}
	manifests := make([]ProofManifest, 0, len(encodedManifests))
	identifiers := make(map[string]struct{}, len(encodedManifests))
	for _, encoded := range encodedManifests {
		manifest, err := DecodeProofManifest(bytes.NewReader(encoded), DefaultDecodeLimit)
		if err != nil {
			return nil, err
		}
		if _, duplicate := identifiers[manifest.Identifier]; duplicate {
			return nil, fmt.Errorf("duplicate default proof manifest %q", manifest.Identifier)
		}
		identifiers[manifest.Identifier] = struct{}{}
		manifests = append(manifests, manifest)
	}
	return manifests, nil
}

func (m ProofManifest) Validate() error {
	if m.FormatVersion != ProofManifestFormatVersion || m.Identifier == "" || m.Theorem == "" ||
		m.Statement == "" || m.LeanVersion == "" {
		return errors.New("complete proof manifest identity is required")
	}
	if m.ResultClass != ResultClassTraceWitness &&
		m.ResultClass != ResultClassInvariantProved && m.ResultClass != ResultClassTemporalProved &&
		m.ResultClass != ResultClassRefinementProved {
		return fmt.Errorf("proof manifest has non-proof result class %q", m.ResultClass)
	}
	if !m.TrustBadge.Valid() {
		return fmt.Errorf("proof manifest has unknown trust badge %q", m.TrustBadge)
	}
	expectedTrust := TrustBadgeKernel
	if len(m.Axioms) != 0 {
		expectedTrust = TrustBadgeKernelWithDeclaredAxioms
	}
	if m.TrustBadge != expectedTrust {
		return fmt.Errorf("proof trust badge %q does not match axiom inventory", m.TrustBadge)
	}
	if !slices.IsSorted(m.Axioms) || len(slices.Compact(append([]string(nil), m.Axioms...))) != len(m.Axioms) {
		return errors.New("proof axiom inventory must be sorted and unique")
	}
	for _, axiom := range m.Axioms {
		if axiom == "" {
			return errors.New("proof axiom identity is required")
		}
		if axiom == "sorryAx" || axiom == "Lean.ofReduceBool" {
			return fmt.Errorf("proof manifest depends on forbidden axiom %q", axiom)
		}
	}
	if m.StatementHash != digestBytes([]byte(m.Statement)) {
		return errors.New("proof statement hash does not match the resolved theorem statement")
	}
	if !validHash(m.SemanticHash) || !validHash(m.SourceDigest) || !validHash(m.DependencyDigest) {
		return errors.New("proof semantic, source, and dependency hashes must be sha256 digests")
	}
	if len(m.SourceDependencies) == 0 {
		return errors.New("proof transitive source dependencies are required")
	}
	if err := validateSourceDependencies(m.SourceDependencies); err != nil {
		return err
	}
	sourceHash, dependencyHash, err := DigestSourceDependencies(m.SourceDependencies)
	if err != nil {
		return err
	}
	if m.SourceDigest != sourceHash || m.SemanticHash != m.SourceDigest {
		return errors.New("proof source digest does not match transitive source dependencies")
	}
	if m.DependencyDigest != dependencyHash {
		return errors.New("proof dependency digest does not match transitive import graph")
	}
	for _, assumption := range m.Assumptions {
		if assumption.Identifier == "" || !validHash(assumption.StatementHash) {
			return errors.New("complete proof assumption is required")
		}
	}
	return nil
}

func validateSourceDependencies(dependencies []SourceDependency) error {
	if !slices.IsSortedFunc(dependencies, func(left, right SourceDependency) int {
		return stringCompare(left.Path, right.Path)
	}) {
		return errors.New("proof source dependencies must be sorted")
	}
	paths := make(map[string]struct{}, len(dependencies))
	for _, dependency := range dependencies {
		if dependency.Path == "" || !validHash(dependency.Digest) {
			return errors.New("complete proof source dependency is required")
		}
		if _, duplicate := paths[dependency.Path]; duplicate {
			return fmt.Errorf("duplicate proof source dependency %q", dependency.Path)
		}
		paths[dependency.Path] = struct{}{}
		if !slices.IsSorted(dependency.Imports) ||
			len(slices.Compact(append([]string(nil), dependency.Imports...))) != len(dependency.Imports) {
			return fmt.Errorf("proof source dependency %q imports must be sorted and unique", dependency.Path)
		}
	}
	for _, dependency := range dependencies {
		for _, imported := range dependency.Imports {
			if _, exists := paths[imported]; !exists {
				return fmt.Errorf("proof source dependency %q imports missing source %q", dependency.Path, imported)
			}
		}
	}
	return nil
}

func DigestSourceDependencies(
	dependencies []SourceDependency,
) (sourceDigest string, importDigest string, err error) {
	if err := validateSourceDependencies(dependencies); err != nil {
		return "", "", err
	}
	hash := sha256.New()
	for _, dependency := range dependencies {
		_, _ = fmt.Fprintf(hash, "%d:%s:%s", len(dependency.Path), dependency.Path, dependency.Digest)
	}
	sourceHash := "sha256:" + hex.EncodeToString(hash.Sum(nil))
	hash = sha256.New()
	for _, dependency := range dependencies {
		_, _ = fmt.Fprintf(hash, "%d:%s:%d", len(dependency.Path), dependency.Path, len(dependency.Imports))
		for _, imported := range dependency.Imports {
			_, _ = fmt.Fprintf(hash, ":%d:%s", len(imported), imported)
		}
	}
	return sourceHash, "sha256:" + hex.EncodeToString(hash.Sum(nil)), nil
}

func digestBytes(value []byte) string {
	digest := sha256.Sum256(value)
	return "sha256:" + hex.EncodeToString(digest[:])
}

func (m ProofManifest) Digest() (string, error) {
	if err := m.Validate(); err != nil {
		return "", err
	}
	encoded, err := json.Marshal(m)
	if err != nil {
		return "", fmt.Errorf("encode proof manifest: %w", err)
	}
	digest := sha256.Sum256(encoded)
	return "sha256:" + hex.EncodeToString(digest[:]), nil
}

func (m ProofManifest) CanonicalJSON() ([]byte, error) {
	if err := m.Validate(); err != nil {
		return nil, err
	}
	encoded, err := json.Marshal(m)
	if err != nil {
		return nil, fmt.Errorf("encode proof manifest: %w", err)
	}
	return encoded, nil
}
