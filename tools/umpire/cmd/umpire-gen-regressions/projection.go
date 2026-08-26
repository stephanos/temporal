package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
)

const supportedExperimentFormat = "umpire-experiment/v1"

var canonicalProjectedJSONKeys = map[string]string{
	"formatversion":           "formatVersion",
	"plan":                    "plan",
	"queryidentity":           "queryIdentity",
	"properties":              "properties",
	"identity":                "identity",
	"observationrequirements": "observationRequirements",
	"semanticidentity":        "semanticIdentity",
	"provenance":              "provenance",
	"sources":                 "sources",
	"path":                    "path",
}

type sourceProjection struct {
	CanonicalPath  string
	RepositoryPath string
}

type projectionRecord struct {
	Identity                string
	Format                  string
	FixturePath             string
	GoOutputPath            string
	MarkdownOutputPath      string
	TestName                string
	Sources                 []sourceProjection
	Properties              []string
	ObservationRequirements []string
	SemanticFingerprint     string
}

type experimentEnvelope struct {
	FormatVersion           string               `json:"formatVersion"`
	Plan                    experimentPlan       `json:"plan"`
	Properties              []experimentProperty `json:"properties"`
	ObservationRequirements []string             `json:"observationRequirements"`
	SemanticIdentity        string               `json:"semanticIdentity"`
	Provenance              experimentProvenance `json:"provenance"`
}

type experimentPlan struct {
	QueryIdentity string `json:"queryIdentity"`
}

type experimentProperty struct {
	Identity string `json:"identity"`
}

type experimentProvenance struct {
	Sources []experimentSource `json:"sources"`
}

type experimentSource struct {
	Path string `json:"path"`
}

func extractProjection(entry manifestEntry, encoded []byte, modelRoot string) (projectionRecord, error) {
	if err := validateManifest([]manifestEntry{entry}); err != nil {
		return projectionRecord{}, err
	}
	document, err := decodeExperiment(encoded)
	if err != nil {
		return projectionRecord{}, fmt.Errorf("extract projection %q: %w", entry.Identity, err)
	}
	if document.FormatVersion != supportedExperimentFormat {
		return projectionRecord{}, fmt.Errorf(
			"extract projection %q: unsupported format %q",
			entry.Identity,
			document.FormatVersion,
		)
	}
	if document.Plan.QueryIdentity != entry.Identity {
		return projectionRecord{}, fmt.Errorf(
			"extract projection %q: query identity mismatch: got %q",
			entry.Identity,
			document.Plan.QueryIdentity,
		)
	}
	if strings.TrimSpace(document.SemanticIdentity) == "" {
		return projectionRecord{}, fmt.Errorf("extract projection %q: semantic identity is empty", entry.Identity)
	}

	sources, err := projectSources(modelRoot, document.Provenance.Sources)
	if err != nil {
		return projectionRecord{}, fmt.Errorf("extract projection %q: %w", entry.Identity, err)
	}
	properties, err := canonicalIdentities("property", propertyIdentities(document.Properties))
	if err != nil {
		return projectionRecord{}, fmt.Errorf("extract projection %q: %w", entry.Identity, err)
	}
	requirements, err := canonicalIdentities("observation requirement", document.ObservationRequirements)
	if err != nil {
		return projectionRecord{}, fmt.Errorf("extract projection %q: %w", entry.Identity, err)
	}
	testName, err := deriveTestName(entry.Identity)
	if err != nil {
		return projectionRecord{}, fmt.Errorf("extract projection %q: %w", entry.Identity, err)
	}
	digest := sha256.Sum256([]byte(document.SemanticIdentity))
	return projectionRecord{
		Identity:                entry.Identity,
		Format:                  document.FormatVersion,
		FixturePath:             entry.FixturePath,
		GoOutputPath:            entry.GoOutputPath,
		MarkdownOutputPath:      entry.MarkdownOutputPath,
		TestName:                testName,
		Sources:                 sources,
		Properties:              properties,
		ObservationRequirements: requirements,
		SemanticFingerprint:     "sha256:" + hex.EncodeToString(digest[:]),
	}, nil
}

func decodeExperiment(encoded []byte) (experimentEnvelope, error) {
	if len(bytes.TrimSpace(encoded)) == 0 {
		return experimentEnvelope{}, errors.New("canonical ExperimentSpec JSON is empty")
	}
	if err := validateCanonicalJSON(encoded); err != nil {
		return experimentEnvelope{}, fmt.Errorf("decode canonical ExperimentSpec JSON: %w", err)
	}
	var document experimentEnvelope
	if err := json.Unmarshal(encoded, &document); err != nil {
		return experimentEnvelope{}, fmt.Errorf("decode canonical ExperimentSpec JSON: %w", err)
	}
	return document, nil
}

func validateCanonicalJSON(encoded []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.UseNumber()
	first, err := decoder.Token()
	if err != nil {
		return err
	}
	if err := validateJSONValue(decoder, first); err != nil {
		return err
	}
	if _, err := decoder.Token(); err == nil {
		return errors.New("trailing JSON value")
	} else if !errors.Is(err, io.EOF) {
		return err
	}
	return nil
}

func validateJSONValue(decoder *json.Decoder, token json.Token) error {
	delimiter, structured := token.(json.Delim)
	if !structured {
		return nil
	}
	switch delimiter {
	case '{':
		seen := make(map[string]struct{})
		for decoder.More() {
			keyToken, err := decoder.Token()
			if err != nil {
				return err
			}
			key, ok := keyToken.(string)
			if !ok {
				return fmt.Errorf("JSON object key has type %T", keyToken)
			}
			if _, duplicate := seen[key]; duplicate {
				return fmt.Errorf("duplicate JSON object key %q", key)
			}
			seen[key] = struct{}{}
			if canonical, projected := canonicalProjectedJSONKeys[strings.ToLower(key)]; projected && key != canonical {
				return fmt.Errorf("JSON object key %q must be spelled %q", key, canonical)
			}
			value, err := decoder.Token()
			if err != nil {
				return err
			}
			if err := validateJSONValue(decoder, value); err != nil {
				return err
			}
		}
		closing, err := decoder.Token()
		if err != nil {
			return err
		}
		if closing != json.Delim('}') {
			return fmt.Errorf("unexpected JSON object delimiter %q", closing)
		}
	case '[':
		for decoder.More() {
			value, err := decoder.Token()
			if err != nil {
				return err
			}
			if err := validateJSONValue(decoder, value); err != nil {
				return err
			}
		}
		closing, err := decoder.Token()
		if err != nil {
			return err
		}
		if closing != json.Delim(']') {
			return fmt.Errorf("unexpected JSON array delimiter %q", closing)
		}
	default:
		return fmt.Errorf("unexpected JSON delimiter %q", delimiter)
	}
	return nil
}

func propertyIdentities(properties []experimentProperty) []string {
	result := make([]string, 0, len(properties))
	for _, property := range properties {
		result = append(result, property.Identity)
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

func projectSources(modelRoot string, sources []experimentSource) ([]sourceProjection, error) {
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
	result := make([]sourceProjection, 0, len(canonical))
	for index, source := range canonical {
		if index > 0 && source == canonical[index-1] {
			return nil, fmt.Errorf("duplicate provenance source %q", source)
		}
		result = append(result, sourceProjection{
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

func validateTestNames(records []projectionRecord) error {
	owners := make(map[string]string, len(records))
	for _, record := range records {
		derived, err := deriveTestName(record.Identity)
		if err != nil {
			return err
		}
		if record.TestName != derived {
			return fmt.Errorf("projection %q has invalid Go test name %q", record.Identity, record.TestName)
		}
		if previous, exists := owners[record.TestName]; exists {
			return fmt.Errorf(
				"projection identities %q and %q collide as Go test name %q",
				previous,
				record.Identity,
				record.TestName,
			)
		}
		owners[record.TestName] = record.Identity
	}
	return nil
}
