// Package regression verifies checked-in generated views of Lean-owned Umpire regressions.
// It validates generated view metadata only; it does not execute Temporal or interpret evidence.
package regression

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tools/umpire/internal/artifactv2"
)

const supportedFormatVersion = artifactv2.ExperimentFormat

// Reference is the complete metadata carried by a generated Go view.
// Source paths are canonical and relative to the repository's model directory;
// FixturePath is relative to the repository root.
type Reference struct {
	FormatVersion           string
	Identity                string
	FixturePath             string
	Sources                 []string
	Properties              []string
	ObservationRequirements []string
	ArtifactChecksum        string
}

// RequireGeneratedView verifies that reference still describes its checked-in
// canonical fixture. It deliberately performs no runtime execution or semantic
// interpretation.
func RequireGeneratedView(t testing.TB, reference Reference) {
	t.Helper()

	repositoryRoot, err := sourceRepositoryRoot()
	require.NoError(t, err, "resolve repository root for generated view %q", reference.Identity)
	actual, err := loadGeneratedView(repositoryRoot, reference)
	require.NoError(t, err, "verify generated view %q", reference.Identity)
	require.Equal(t, reference, actual, "generated view %q differs from its canonical fixture", reference.Identity)
}

type fixtureEnvelope = artifactv2.Experiment
type fixturePlan = artifactv2.DrivePlan
type fixtureProperty = artifactv2.Property
type fixtureProvenance = artifactv2.Provenance
type fixtureSource = artifactv2.SourceLocation

func sourceRepositoryRoot() (string, error) {
	_, sourceFile, _, ok := runtime.Caller(0)
	if !ok || sourceFile == "" {
		return "", errors.New("locate generated view verifier source")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(sourceFile), "..", "..", "..")), nil
}

func loadGeneratedView(repositoryRoot string, reference Reference) (Reference, error) {
	resolvedRepositoryRoot, err := resolveDirectory(repositoryRoot, "repository root")
	if err != nil {
		return Reference{}, err
	}
	if err := validateReference(resolvedRepositoryRoot, reference); err != nil {
		return Reference{}, err
	}

	fixturePath, err := resolveRegularFile(
		resolvedRepositoryRoot,
		reference.FixturePath,
		"fixture",
	)
	if err != nil {
		return Reference{}, err
	}
	encoded, err := os.ReadFile(fixturePath)
	if err != nil {
		return Reference{}, fmt.Errorf("read fixture %q: %w", reference.FixturePath, err)
	}
	document, err := decodeFixture(encoded)
	if err != nil {
		return Reference{}, fmt.Errorf("decode fixture %q: %w", reference.FixturePath, err)
	}
	if document.FormatVersion != supportedFormatVersion {
		return Reference{}, fmt.Errorf(
			"fixture %q has unsupported format %q",
			reference.FixturePath,
			document.FormatVersion,
		)
	}
	if strings.TrimSpace(document.Plan.QueryDefinitionID) == "" {
		return Reference{}, fmt.Errorf("fixture %q has empty query identity", reference.FixturePath)
	}
	if strings.TrimSpace(document.ArtifactChecksum) == "" {
		return Reference{}, fmt.Errorf("fixture %q has empty artifact checksum", reference.FixturePath)
	}

	sources := make([]string, 0, len(document.Provenance.SourceLocations))
	for _, source := range document.Provenance.SourceLocations {
		sources = append(sources, source.Path)
	}
	modelRoot, err := resolveDirectory(filepath.Join(resolvedRepositoryRoot, "model"), "model root")
	if err != nil {
		return Reference{}, err
	}
	if err := validateSources(modelRoot, sources); err != nil {
		return Reference{}, fmt.Errorf("fixture %q provenance: %w", reference.FixturePath, err)
	}

	properties := make([]string, 0, len(document.Properties))
	for _, property := range document.Properties {
		properties = append(properties, property.DefinitionID)
	}
	if err := validateIdentities("property", properties); err != nil {
		return Reference{}, fmt.Errorf("fixture %q: %w", reference.FixturePath, err)
	}
	if err := validateIdentities("observation requirement", document.ObservationRequirementDefinitionIDs); err != nil {
		return Reference{}, fmt.Errorf("fixture %q: %w", reference.FixturePath, err)
	}

	return Reference{
		FormatVersion:           document.FormatVersion,
		Identity:                document.Plan.QueryDefinitionID,
		FixturePath:             reference.FixturePath,
		Sources:                 sources,
		Properties:              properties,
		ObservationRequirements: document.ObservationRequirementDefinitionIDs,
		ArtifactChecksum:        document.ArtifactChecksum,
	}, nil
}

func validateReference(repositoryRoot string, reference Reference) error {
	if reference.FormatVersion != supportedFormatVersion {
		return fmt.Errorf("reference has unsupported format %q", reference.FormatVersion)
	}
	if strings.TrimSpace(reference.Identity) == "" {
		return errors.New("reference identity is empty")
	}
	if err := validateRelativePath(reference.FixturePath); err != nil {
		return fmt.Errorf("fixture path %q is unsafe: %w", reference.FixturePath, err)
	}
	modelRoot, err := resolveDirectory(filepath.Join(repositoryRoot, "model"), "model root")
	if err != nil {
		return err
	}
	if err := validateSources(modelRoot, reference.Sources); err != nil {
		return fmt.Errorf("reference provenance: %w", err)
	}
	if err := validateIdentities("property", reference.Properties); err != nil {
		return fmt.Errorf("reference: %w", err)
	}
	if err := validateIdentities("observation requirement", reference.ObservationRequirements); err != nil {
		return fmt.Errorf("reference: %w", err)
	}
	if !artifactv2.ValidDigest(reference.ArtifactChecksum) {
		return fmt.Errorf("reference artifact checksum %q is invalid", reference.ArtifactChecksum)
	}
	return nil
}

func decodeFixture(encoded []byte) (fixtureEnvelope, error) {
	return artifactv2.DecodeExperiment(encoded)
}

func validateSources(modelRoot string, sources []string) error {
	if len(sources) == 0 {
		return errors.New("at least one Lean source is required")
	}
	if !slices.IsSorted(sources) {
		return errors.New("Lean sources are not in canonical order")
	}
	for index, source := range sources {
		if index > 0 && source == sources[index-1] {
			return fmt.Errorf("duplicate Lean source %q", source)
		}
		if !strings.HasSuffix(source, ".lean") {
			return fmt.Errorf("source %q is not a Lean file", source)
		}
		if _, err := resolveRegularFile(modelRoot, source, "Lean source"); err != nil {
			return err
		}
	}
	return nil
}

func validateIdentities(kind string, identities []string) error {
	if len(identities) == 0 {
		return fmt.Errorf("at least one %s identity is required", kind)
	}
	if !slices.IsSorted(identities) {
		return fmt.Errorf("%s identities are not in canonical order", kind)
	}
	for index, identity := range identities {
		if strings.TrimSpace(identity) == "" {
			return fmt.Errorf("%s identity is empty", kind)
		}
		if index > 0 && identity == identities[index-1] {
			return fmt.Errorf("duplicate %s identity %q", kind, identity)
		}
	}
	return nil
}

func resolveDirectory(value, label string) (string, error) {
	absolute, err := filepath.Abs(value)
	if err != nil {
		return "", fmt.Errorf("resolve %s %q: %w", label, value, err)
	}
	resolved, err := filepath.EvalSymlinks(absolute)
	if err != nil {
		return "", fmt.Errorf("resolve %s %q: %w", label, value, err)
	}
	info, err := os.Stat(resolved)
	if err != nil {
		return "", fmt.Errorf("read %s %q: %w", label, value, err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("%s %q is not a directory", label, value)
	}
	return resolved, nil
}

func resolveRegularFile(root, relative, label string) (string, error) {
	if err := validateRelativePath(relative); err != nil {
		return "", fmt.Errorf("%s path %q is unsafe: %w", label, relative, err)
	}
	target := filepath.Join(root, filepath.FromSlash(relative))
	resolved, err := filepath.EvalSymlinks(target)
	if err != nil {
		return "", fmt.Errorf("resolve %s %q: %w", label, relative, err)
	}
	if !pathWithin(root, resolved) {
		return "", fmt.Errorf("%s %q resolves outside its root", label, relative)
	}
	info, err := os.Stat(resolved)
	if err != nil {
		return "", fmt.Errorf("read %s %q: %w", label, relative, err)
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("%s %q is not a regular file", label, relative)
	}
	return resolved, nil
}

func validateRelativePath(value string) error {
	if value == "" {
		return errors.New("path is empty")
	}
	if strings.Contains(value, "\\") {
		return errors.New("path must use forward slashes")
	}
	if path.IsAbs(value) || filepath.IsAbs(value) || filepath.VolumeName(value) != "" {
		return errors.New("path is absolute")
	}
	cleaned := path.Clean(value)
	if cleaned == "." || cleaned != value || cleaned == ".." || strings.HasPrefix(cleaned, "../") {
		return errors.New("path is not a clean contained relative path")
	}
	return nil
}

func pathWithin(root, target string) bool {
	relative, err := filepath.Rel(root, target)
	return err == nil && relative != ".." && !filepath.IsAbs(relative) &&
		!strings.HasPrefix(relative, ".."+string(filepath.Separator))
}
