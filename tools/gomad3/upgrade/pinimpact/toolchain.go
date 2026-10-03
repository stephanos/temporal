package pinimpact

import (
	"encoding/json"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strconv"

	gomadversion "go.temporal.io/server/tools/gomad3/toolchain/version"
)

const (
	boundaryManifestPath = "deterministicio/boundary/manifest.json"
	clockInventoryPath   = "toolchain/clock_inventory_test.go"
	clockInventoryName   = "reviewedHostClockReferences"
)

// Interception fingerprints and host-clock references pin standard-library
// source of the pinned Go release. A module bump cannot change them; a
// candidate that needs a newer Go than the pinned toolchain leaves them
// unknown, because only a Go upgrade can tell whether they still hold.
func toolchainPin(class Class, id, reason string) Pin {
	if reason == "" {
		return Pin{Class: class, Status: StatusUnaffected, ID: id}
	}
	return Pin{Class: class, Status: StatusUnknown, ID: id, Reason: reason}
}

func (evaluation *evaluation) evaluateInterceptions(root, toolchainReason string) {
	identities, err := interceptionFingerprints(root)
	if err != nil {
		evaluation.record(Pin{Class: ClassInterception, Status: StatusUnknown, ID: boundaryManifestPath, Reason: err.Error()})
		return
	}
	for _, identity := range identities {
		pin := toolchainPin(ClassInterception, identity.ID, toolchainReason)
		pin.SourceSetSHA256, pin.Platforms, pin.PinnedVersion = identity.SourceSetSHA256, identity.Platforms, gomadversion.GoVersion
		evaluation.record(pin)
	}
}

func (evaluation *evaluation) evaluateClockReferences(root, toolchainReason string) {
	identities, err := clockReferences(root)
	if err != nil {
		evaluation.record(Pin{Class: ClassClockReference, Status: StatusUnknown, ID: clockInventoryPath, Reason: err.Error()})
		return
	}
	for _, identity := range identities {
		pin := toolchainPin(ClassClockReference, identity.ID, toolchainReason)
		pin.Platforms, pin.PinnedVersion = identity.Platforms, gomadversion.GoVersion
		evaluation.record(pin)
	}
}

// interceptionFingerprints names every declaration fingerprint in the
// boundary manifest, including platform overrides, after checking that the manifest belongs to the
// pinned Go release.
func interceptionFingerprints(root string) ([]Pin, error) {
	contents, err := readRootFile(root, boundaryManifestPath)
	if err != nil {
		return nil, err
	}
	var manifest struct {
		GoVersion       string `json:"go_version"`
		ManifestVersion string `json:"manifest_version"`
		Intercepts      []struct {
			Package  string `json:"package"`
			Receiver *struct {
				Name    string `json:"name"`
				Pointer bool   `json:"pointer"`
			} `json:"receiver"`
			Symbol            string `json:"symbol"`
			DeclarationSHA256 string `json:"declaration_sha256"`
			PlatformOverrides map[string]struct {
				DeclarationSHA256 string `json:"declaration_sha256"`
			} `json:"platform_overrides"`
		} `json:"intercepts"`
	}
	if err := json.Unmarshal(contents, &manifest); err != nil {
		return nil, fmt.Errorf("decode boundary manifest: %w", err)
	}
	if manifest.GoVersion != gomadversion.GoVersion || manifest.ManifestVersion != gomadversion.BoundaryManifestVersion {
		return nil, fmt.Errorf("boundary manifest %s for %s does not match the pinned %s for %s",
			manifest.ManifestVersion, manifest.GoVersion, gomadversion.BoundaryManifestVersion, gomadversion.GoVersion)
	}
	identities := make([]Pin, 0, len(manifest.Intercepts))
	for _, intercept := range manifest.Intercepts {
		if intercept.DeclarationSHA256 == "" {
			return nil, fmt.Errorf("boundary intercept %s.%s has no declaration fingerprint", intercept.Package, intercept.Symbol)
		}
		name := intercept.Package + "." + intercept.Symbol
		if intercept.Receiver != nil {
			receiver := intercept.Receiver.Name
			if intercept.Receiver.Pointer {
				receiver = "*" + receiver
			}
			name = intercept.Package + ".(" + receiver + ")." + intercept.Symbol
		}
		identities = append(identities, Pin{ID: name, SourceSetSHA256: intercept.DeclarationSHA256, Platforms: gomadversion.SupportedPlatforms[:]})
		for _, platform := range slices.Sorted(maps.Keys(intercept.PlatformOverrides)) {
			if intercept.PlatformOverrides[platform].DeclarationSHA256 != "" {
				identities = append(identities, Pin{ID: name + " " + platform, SourceSetSHA256: intercept.PlatformOverrides[platform].DeclarationSHA256, Platforms: []string{platform}})
			}
		}
	}
	if len(identities) == 0 {
		return nil, errors.New("boundary manifest has no fingerprinted entries")
	}
	return identities, nil
}

// readRootFile reads a checked-in file below root. Its error names the file by
// its root-relative path, so the report stays free of host paths.
func readRootFile(root, relative string) ([]byte, error) {
	contents, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(relative)))
	if err != nil {
		var pathErr *fs.PathError
		if errors.As(err, &pathErr) {
			err = pathErr.Err
		}
		return nil, fmt.Errorf("read %s: %w", relative, err)
	}
	return contents, nil
}

// clockReferences names every reviewed host-clock reference. The inventory is
// the literal the toolchain tier checks the built GOROOT against, so it is
// read from that test source rather than duplicated.
func clockReferences(root string) ([]Pin, error) {
	contents, err := readRootFile(root, clockInventoryPath)
	if err != nil {
		return nil, err
	}
	files := token.NewFileSet()
	parsed, err := parser.ParseFile(files, clockInventoryPath, contents, parser.SkipObjectResolution)
	if err != nil {
		return nil, fmt.Errorf("parse host-clock inventory: %w", err)
	}
	for _, declaration := range parsed.Decls {
		general, ok := declaration.(*ast.GenDecl)
		if !ok || general.Tok != token.VAR {
			continue
		}
		for _, specification := range general.Specs {
			value, ok := specification.(*ast.ValueSpec)
			if !ok || len(value.Names) != 1 || value.Names[0].Name != clockInventoryName || len(value.Values) != 1 {
				continue
			}
			return clockReferenceIdentities(value.Values[0])
		}
	}
	return nil, fmt.Errorf("host-clock inventory %s is missing", clockInventoryName)
}

func clockReferenceIdentities(expression ast.Expr) ([]Pin, error) {
	literal, ok := expression.(*ast.CompositeLit)
	if !ok || len(literal.Elts) == 0 {
		return nil, fmt.Errorf("host-clock inventory %s is not a non-empty literal", clockInventoryName)
	}
	identities := make([]Pin, 0, len(literal.Elts))
	for index, element := range literal.Elts {
		reference, ok := element.(*ast.CompositeLit)
		if !ok || len(reference.Elts) < 3 {
			return nil, fmt.Errorf("host-clock reference %d is not a positional literal", index)
		}
		fields := make([]string, 3)
		for field := range fields {
			basic, ok := reference.Elts[field].(*ast.BasicLit)
			if !ok || basic.Kind != token.STRING {
				return nil, fmt.Errorf("host-clock reference %d field %d is not a string literal", index, field)
			}
			unquoted, err := strconv.Unquote(basic.Value)
			if err != nil {
				return nil, fmt.Errorf("host-clock reference %d field %d: %w", index, field, err)
			}
			fields[field] = unquoted
		}
		identities = append(identities, Pin{ID: fields[0] + " " + fields[1] + " " + fields[2], Platforms: []string{fields[0]}})
	}
	return identities, nil
}
