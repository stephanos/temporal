package main

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"go.temporal.io/server/tools/umpire/internal/artifactv2"
)

const supportedExperimentFormat = artifactv2.ExperimentFormat

type sourceView struct {
	CanonicalPath  string
	RepositoryPath string
}

type generatedViewRecord struct {
	Identity                string
	Format                  string
	FixturePath             string
	GoOutputPath            string
	MarkdownOutputPath      string
	TestName                string
	Sources                 []sourceView
	Properties              []string
	ObservationRequirements []string
	ArtifactChecksum        string
}

type experimentEnvelope = artifactv2.Experiment
type experimentPlan = artifactv2.DrivePlan
type experimentProperty = artifactv2.Property
type experimentProvenance = artifactv2.Provenance
type experimentSource = artifactv2.SourceLocation

func extractGeneratedView(entry manifestEntry, encoded []byte, modelRoot string) (generatedViewRecord, error) {
	if err := validateManifest([]manifestEntry{entry}); err != nil {
		return generatedViewRecord{}, err
	}
	document, err := decodeExperiment(encoded)
	if err != nil {
		return generatedViewRecord{}, fmt.Errorf("extract generated view %q: %w", entry.Identity, err)
	}
	if document.FormatVersion != supportedExperimentFormat {
		return generatedViewRecord{}, fmt.Errorf(
			"extract generated view %q: unsupported format %q",
			entry.Identity,
			document.FormatVersion,
		)
	}
	if document.Plan.QueryDefinitionID != entry.Identity {
		return generatedViewRecord{}, fmt.Errorf(
			"extract generated view %q: query definition ID mismatch: got %q",
			entry.Identity,
			document.Plan.QueryDefinitionID,
		)
	}
	if strings.TrimSpace(document.ArtifactChecksum) == "" {
		return generatedViewRecord{}, fmt.Errorf("extract generated view %q: artifact checksum is empty", entry.Identity)
	}

	sources, err := projectSources(modelRoot, document.Provenance.SourceLocations)
	if err != nil {
		return generatedViewRecord{}, fmt.Errorf("extract generated view %q: %w", entry.Identity, err)
	}
	properties, err := canonicalIdentities("property", propertyIdentities(document.Properties))
	if err != nil {
		return generatedViewRecord{}, fmt.Errorf("extract generated view %q: %w", entry.Identity, err)
	}
	requirements, err := canonicalIdentities("observation requirement", document.ObservationRequirementDefinitionIDs)
	if err != nil {
		return generatedViewRecord{}, fmt.Errorf("extract generated view %q: %w", entry.Identity, err)
	}
	testName, err := deriveTestName(entry.Identity)
	if err != nil {
		return generatedViewRecord{}, fmt.Errorf("extract generated view %q: %w", entry.Identity, err)
	}
	return generatedViewRecord{
		Identity:                entry.Identity,
		Format:                  document.FormatVersion,
		FixturePath:             entry.FixturePath,
		GoOutputPath:            entry.GoOutputPath,
		MarkdownOutputPath:      entry.MarkdownOutputPath,
		TestName:                testName,
		Sources:                 sources,
		Properties:              properties,
		ObservationRequirements: requirements,
		ArtifactChecksum:        document.ArtifactChecksum,
	}, nil
}

func decodeExperiment(encoded []byte) (experimentEnvelope, error) {
	document, err := artifactv2.DecodeExperiment(encoded)
	if err != nil {
		return experimentEnvelope{}, fmt.Errorf("decode canonical ExperimentSpec JSON: %w", err)
	}
	return document, nil
}

func propertyIdentities(properties []experimentProperty) []string {
	result := make([]string, 0, len(properties))
	for _, property := range properties {
		result = append(result, property.DefinitionID)
	}
	return result
}

func canonicalIdentities(kind string, identities []string) ([]string, error) {
	if len(identities) == 0 {
		return nil, fmt.Errorf("at least one %s identity is required", kind)
	}
	result := append([]string(nil), identities...)
	slices.Sort(result)
	for index, identity := range result {
		if strings.TrimSpace(identity) == "" {
			return nil, fmt.Errorf("%s identity is empty", kind)
		}
		if index > 0 && identity == result[index-1] {
			return nil, fmt.Errorf("duplicate %s identity %q", kind, identity)
		}
	}
	return result, nil
}

func projectSources(modelRoot string, sources []experimentSource) ([]sourceView, error) {
	if len(sources) == 0 {
		return nil, errors.New("at least one provenance source is required")
	}
	absoluteRoot, resolvedRoot, err := resolveModelRoot(modelRoot)
	if err != nil {
		return nil, err
	}
	canonical := make([]string, 0, len(sources))
	for _, source := range sources {
		if err := validateModelPath(source.Path); err != nil {
			return nil, fmt.Errorf("provenance source: %w", err)
		}
		if err := validateLeanSource(absoluteRoot, resolvedRoot, source.Path); err != nil {
			return nil, err
		}
		canonical = append(canonical, source.Path)
	}
	slices.Sort(canonical)
	result := make([]sourceView, 0, len(canonical))
	for index, source := range canonical {
		if index > 0 && source == canonical[index-1] {
			return nil, fmt.Errorf("duplicate provenance source %q", source)
		}
		result = append(result, sourceView{
			CanonicalPath:  source,
			RepositoryPath: path.Join("model", source),
		})
	}
	return result, nil
}

func resolveModelRoot(modelRoot string) (string, string, error) {
	if modelRoot == "" {
		return "", "", errors.New("model root is required")
	}
	absoluteRoot, err := filepath.Abs(modelRoot)
	if err != nil {
		return "", "", fmt.Errorf("resolve model root %q: %w", modelRoot, err)
	}
	info, err := os.Stat(absoluteRoot)
	if err != nil {
		return "", "", fmt.Errorf("read model root %q: %w", modelRoot, err)
	}
	if !info.IsDir() {
		return "", "", fmt.Errorf("model root %q is not a directory", modelRoot)
	}
	resolvedRoot, err := filepath.EvalSymlinks(absoluteRoot)
	if err != nil {
		return "", "", fmt.Errorf("resolve model root %q: %w", modelRoot, err)
	}
	return absoluteRoot, resolvedRoot, nil
}

func validateModelPath(value string) error {
	if err := validateRepositoryPath(value); err != nil {
		return fmt.Errorf("model-root-relative path %q is unsafe", value)
	}
	if !strings.HasSuffix(value, ".lean") {
		return fmt.Errorf("model-root-relative path %q is not a Lean source", value)
	}
	return nil
}

func validateLeanSource(absoluteRoot, resolvedRoot, source string) error {
	target := filepath.Join(absoluteRoot, filepath.FromSlash(source))
	resolvedTarget, err := filepath.EvalSymlinks(target)
	if err != nil {
		return fmt.Errorf("resolve provenance source %q: %w", source, err)
	}
	if !pathWithin(resolvedRoot, resolvedTarget) {
		return fmt.Errorf("provenance source %q resolves outside model root", source)
	}
	info, err := os.Stat(resolvedTarget)
	if err != nil {
		return fmt.Errorf("read provenance source %q: %w", source, err)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("provenance source %q is not a regular file", source)
	}
	return nil
}

func pathWithin(root, target string) bool {
	relative, err := filepath.Rel(root, target)
	return err == nil && relative != ".." && !filepath.IsAbs(relative) &&
		!strings.HasPrefix(relative, ".."+string(filepath.Separator))
}

func deriveTestName(identity string) (string, error) {
	var name strings.Builder
	name.WriteString("Test")
	capitalize := true
	for _, character := range identity {
		switch {
		case character >= 'a' && character <= 'z':
			if capitalize {
				character -= 'a' - 'A'
			}
			name.WriteRune(character)
			capitalize = false
		case character >= 'A' && character <= 'Z', character >= '0' && character <= '9':
			name.WriteRune(character)
			capitalize = false
		default:
			capitalize = true
		}
	}
	if name.Len() == len("Test") {
		return "", fmt.Errorf("identity %q does not produce a valid Go test name", identity)
	}
	return name.String(), nil
}

func validateGeneratedViewTestNames(records []generatedViewRecord) error {
	owners := make(map[string]string, len(records))
	for _, record := range records {
		derived, err := deriveTestName(record.Identity)
		if err != nil {
			return err
		}
		if record.TestName != derived {
			return fmt.Errorf("generated view %q has invalid Go test name %q", record.Identity, record.TestName)
		}
		if previous, exists := owners[record.TestName]; exists {
			return fmt.Errorf(
				"generated view identities %q and %q collide as Go test name %q",
				previous,
				record.Identity,
				record.TestName,
			)
		}
		owners[record.TestName] = record.Identity
	}
	return nil
}
