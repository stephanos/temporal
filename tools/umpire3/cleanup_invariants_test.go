package umpire3_test

import (
	"bytes"
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tools/umpire3/checker/finite"
	protocolcatalog "go.temporal.io/server/tools/umpire3/protocol/catalog"
	protocolchecker "go.temporal.io/server/tools/umpire3/protocol/checker"
	protocolexperiment "go.temporal.io/server/tools/umpire3/protocol/experiment"
)

func TestCheckedNexusTargetUsesOneCanonicalFamily(t *testing.T) {
	catalog, err := protocolcatalog.DefaultCatalog()
	require.NoError(t, err)
	view, found, err := finite.DefaultFirstOrderView(protocolcatalog.TargetIDNexusCancellation, "sound")
	require.NoError(t, err)
	require.True(t, found)

	target, found := catalogTarget(catalog, protocolcatalog.TargetIDNexusCancellation)
	require.True(t, found)
	canonicalSystem := strings.TrimSuffix(strings.TrimPrefix(view.CanonicalModel, "Umpire3."), ".behavior")
	require.Contains(t, target.Modules, canonicalSystem)
	family := strings.TrimPrefix(canonicalSystem, "Temporal.System.")
	require.Contains(t, target.Modules, "Temporal.Feature."+family)
	require.Contains(t, target.Modules, "Temporal.Refinement."+family)

	experimentBytes, err := os.ReadFile("testdata/generated/nexus-cancellation.json")
	require.NoError(t, err)
	experiment, err := protocolexperiment.DecodeExperiment(bytes.NewReader(experimentBytes), protocolexperiment.DefaultDecodeLimit)
	require.NoError(t, err)
	for _, module := range target.Modules {
		require.Contains(t, experiment.Model.Modules, module)
	}

	manifests, err := protocolchecker.DefaultProofManifests()
	require.NoError(t, err)
	manifest, found := proofManifestByIdentifier(manifests, experiment.Provenance.ProofManifest)
	require.True(t, found)
	require.Equal(t, experiment.Model.SemanticHash, manifest.SemanticHash)

	coverage, err := protocolchecker.DefaultCheckerCoverage()
	require.NoError(t, err)
	for _, entry := range coverage.Entries {
		if entry.Target == protocolcatalog.TargetIDNexusCancellation && entry.Status == protocolchecker.CheckerCoverageChecked {
			require.Equal(t, view.SemanticHash, entry.SemanticHash)
		}
	}
}

func TestSystemModelsDoNotImportFeatureSemantics(t *testing.T) {
	var violations []string
	systems := 0
	err := filepath.WalkDir("model/Temporal/Families", func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() || filepath.Ext(path) != ".lean" {
			return err
		}
		contents, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		source := string(contents)
		if !strings.Contains(source, "namespace Umpire3.Temporal.System.") {
			return nil
		}
		systems++
		for _, forbidden := range []string{
			"import Temporal.Families.",
			"import Temporal.Feature.",
			"import Temporal.Product.",
			"Umpire3.Temporal.Feature.",
			"Umpire3.Temporal.Product.",
			"productRun",
		} {
			if strings.HasPrefix(forbidden, "import Temporal.Families.") {
				for _, line := range strings.Split(source, "\n") {
					if strings.HasPrefix(line, forbidden) && strings.Contains(line, "Feature") {
						violations = append(violations, path+": "+line)
					}
				}
				continue
			}
			if strings.Contains(source, forbidden) {
				violations = append(violations, path+": "+forbidden)
			}
		}
		return nil
	})
	require.NoError(t, err)
	require.Positive(t, systems)
	require.Empty(t, violations)
}

func TestLeanModelHasNoHistoricalNamespaces(t *testing.T) {
	for _, path := range []string{"model/Temporal/Product", "model/Temporal/System/MigratedFamilies.lean",
		"model/Temporal/Refinement/MigratedFamilies.lean"} {
		_, err := os.Stat(path)
		require.ErrorIs(t, err, os.ErrNotExist)
	}
	var violations []string
	err := filepath.WalkDir("model", func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() || filepath.Ext(path) != ".lean" {
			return err
		}
		contents, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if strings.Contains(string(contents), "MigratedFamilies") ||
			strings.Contains(string(contents), "Temporal.Product") {
			violations = append(violations, path)
		}
		return nil
	})
	require.NoError(t, err)
	require.Empty(t, violations)
}

func TestProtocolExportsTransportBehaviorOnly(t *testing.T) {
	for _, name := range []string{"catalog", "checker", "experiment", "monitor", "release"} {
		info, err := os.Stat(filepath.Join("protocol", name))
		require.NoError(t, err)
		require.True(t, info.IsDir())
	}
	forbiddenPrefixes := []string{"Bind", "Evaluate", "Execute", "Promote", "Qualify", "Replay", "Run", "Sign"}
	var violations []string
	err := filepath.WalkDir("protocol", func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() || filepath.Ext(path) != ".go" || strings.HasSuffix(path, "_test.go") {
			return err
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil {
			return err
		}
		for _, declaration := range file.Decls {
			function, ok := declaration.(*ast.FuncDecl)
			if !ok || !function.Name.IsExported() {
				continue
			}
			if function.Name.Name == "SigningPayload" {
				continue
			}
			for _, prefix := range forbiddenPrefixes {
				if strings.HasPrefix(function.Name.Name, prefix) {
					violations = append(violations, filepath.Base(path)+":"+function.Name.Name)
				}
			}
		}
		return nil
	})
	require.NoError(t, err)
	require.Empty(t, violations)
}

func TestBuildOnlyCommandsUseOneDeveloperEntryPoint(t *testing.T) {
	entries, err := os.ReadDir("cmd")
	require.NoError(t, err)
	allowed := []string{
		"umpire3",
		"umpire3-canary",
		"umpire3-canary-worker",
		"umpire3-dev",
		"umpire3-native",
		"umpire3-participant",
		"umpire3-veil",
	}
	for _, entry := range entries {
		if entry.IsDir() {
			require.Contains(t, allowed, entry.Name())
		}
	}
}

func TestCommandsAreThinMainAdapters(t *testing.T) {
	err := filepath.WalkDir("cmd", func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() || entry.Name() != "main.go" {
			return err
		}
		contents, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		require.LessOrEqual(t, len(strings.Split(string(contents), "\n")), 40, path)
		file, err := parser.ParseFile(token.NewFileSet(), path, contents, 0)
		if err != nil {
			return err
		}
		functions := 0
		for _, declaration := range file.Decls {
			if function, ok := declaration.(*ast.FuncDecl); ok {
				functions++
				require.Equal(t, "main", function.Name.Name, path)
			}
		}
		require.Equal(t, 1, functions, path)
		return nil
	})
	require.NoError(t, err)
}

func TestRemovedCompatibilityCommandsStayRemoved(t *testing.T) {
	for _, path := range []string{"cmd/umpire3-run", "cmd/umpire3-qualify"} {
		_, err := os.Stat(path)
		require.ErrorIs(t, err, os.ErrNotExist)
	}
	support, err := os.ReadFile("docs/support.md")
	require.NoError(t, err)
	require.Contains(t, string(support), "removed on 2026-08-21")
}

func TestTrackedJSONHasDeclaredArtifactOwnership(t *testing.T) {
	encoded, err := os.ReadFile("artifact-manifest.json")
	require.NoError(t, err)
	var manifest struct {
		FormatVersion string `json:"formatVersion"`
		Artifacts     []struct {
			Path            string `json:"path"`
			Class           string `json:"class"`
			Owner           string `json:"owner"`
			SourceCommand   string `json:"sourceCommand"`
			FormatVersion   string `json:"formatVersion"`
			RetentionReason string `json:"retentionReason"`
		} `json:"artifacts"`
	}
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.DisallowUnknownFields()
	require.NoError(t, decoder.Decode(&manifest))
	require.Equal(t, "umpire3/artifact-manifest/v1", manifest.FormatVersion)

	owned := make(map[string]struct{}, len(manifest.Artifacts))
	for _, artifact := range manifest.Artifacts {
		require.NotEmpty(t, artifact.Path)
		require.Contains(t, []string{"generated", "retained", "fixture"}, artifact.Class)
		require.NotEmpty(t, artifact.Owner)
		require.NotEmpty(t, artifact.SourceCommand)
		require.NotEmpty(t, artifact.FormatVersion)
		require.NotEmpty(t, artifact.RetentionReason)
		_, duplicate := owned[artifact.Path]
		require.False(t, duplicate, artifact.Path)
		owned[artifact.Path] = struct{}{}
		require.FileExists(t, artifact.Path)
		command := exec.Command("git", "check-ignore", "--quiet", "--", artifact.Path)
		err := command.Run()
		var exitErr *exec.ExitError
		require.ErrorAs(t, err, &exitErr, artifact.Path)
		require.Equal(t, 1, exitErr.ExitCode(), artifact.Path)
	}
	tracked := trackedJSONFiles(t)
	require.Equal(t, tracked, owned)
}

func TestCleanRemovesOnlyResolvedUmpire3Caches(t *testing.T) {
	script, err := os.ReadFile("clean.sh")
	require.NoError(t, err)
	root := t.TempDir()
	umpireRoot := filepath.Join(root, "tools", "umpire3")
	require.NoError(t, os.MkdirAll(umpireRoot, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(umpireRoot, "clean.sh"), script, 0o700))

	caches := []string{
		filepath.Join(umpireRoot, "model", ".lake"),
	}
	for _, cache := range caches {
		require.NoError(t, os.MkdirAll(cache, 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(cache, "sentinel"), []byte("cache"), 0o600))
	}
	preserved := filepath.Join(umpireRoot, "model", "keep")
	require.NoError(t, os.MkdirAll(preserved, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(preserved, "sentinel"), []byte("source"), 0o600))

	command := exec.Command("sh", filepath.Join(umpireRoot, "clean.sh"))
	output, err := command.CombinedOutput()
	require.NoError(t, err, string(output))
	for _, cache := range caches {
		_, err := os.Stat(cache)
		require.ErrorIs(t, err, os.ErrNotExist)
	}
	require.FileExists(t, filepath.Join(preserved, "sentinel"))
}

func proofManifestByIdentifier(manifests []protocolchecker.ProofManifest, identifier string) (protocolchecker.ProofManifest, bool) {
	for _, manifest := range manifests {
		if manifest.Identifier == identifier {
			return manifest, true
		}
	}
	return protocolchecker.ProofManifest{}, false
}

func catalogTarget(catalog protocolcatalog.Catalog, identifier protocolcatalog.TargetID) (protocolcatalog.TargetDeclaration, bool) {
	for _, target := range catalog.Targets {
		if target.Identifier == string(identifier) {
			return target, true
		}
	}
	return protocolcatalog.TargetDeclaration{}, false
}

func trackedJSONFiles(t *testing.T) map[string]struct{} {
	t.Helper()
	result := make(map[string]struct{})
	err := filepath.WalkDir(".", func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if entry.Name() == ".lake" {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) == ".json" && filepath.ToSlash(path) != "artifact-manifest.json" {
			result[strings.TrimPrefix(filepath.ToSlash(path), "./")] = struct{}{}
		}
		return nil
	})
	require.NoError(t, err)
	return result
}
