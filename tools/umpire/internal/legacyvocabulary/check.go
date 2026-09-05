package legacyvocabulary

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// Violation identifies one retired public Umpire token in the active repository surface.
type Violation struct {
	Path  string
	Line  int
	Token string
}

type tokenRule struct {
	name    string
	pattern *regexp.Regexp
}

var downstreamSpecs = []string{
	"fn-5-umpire-discovery-promotion-and-artifact",
	"fn-18-versioned-umpire-artifact-boundary",
	"fn-19-bounded-local-temporal-execution-and",
	"fn-20-local-execution-semantic-conformance",
	"fn-21-nexus-duplicate-observation-control",
	"fn-22-deterministic-replay-semantic",
	"fn-24-lean-native-verification-receipts-and",
	"fn-25-optional-callerclosure-veil-binding-and",
	"fn-26-local-qualification-receipts-and-staged",
	"fn-27-hermetic-ci-execution-and-qualification",
	"fn-28-authorized-remote-staging-black-box",
	"fn-29-bounded-production-canary-execution-and",
	"fn-30-release-evidence-graph-and-manual",
	"fn-32-add-umpire-refinement-and-the-first",
	"fn-33-run-resumable-semantic-exploration",
}

var retiredRules = buildRetiredRules()

// Check scans only live Umpire source, current Generated Views, active Umpire4
// documentation, and the open downstream Flow closure.
func Check(repositoryRoot string) ([]Violation, error) {
	paths, err := scopedPaths(repositoryRoot)
	if err != nil {
		return nil, err
	}

	var violations []Violation
	for _, relativePath := range paths {
		// The registry must spell the tokens it rejects. Keep that unavoidable
		// implementation file out of its own input rather than allowlisting each
		// token and accidentally weakening checks in the rest of the package.
		if relativePath == "tools/umpire/internal/legacyvocabulary/check.go" {
			continue
		}
		content, err := os.ReadFile(filepath.Join(repositoryRoot, filepath.FromSlash(relativePath)))
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", relativePath, err)
		}
		for lineIndex, line := range strings.Split(string(content), "\n") {
			for _, rule := range retiredRules {
				if rule.pattern.MatchString(line) && !allowedNegativeFixture(relativePath, rule.name) {
					violations = append(violations, Violation{
						Path:  relativePath,
						Line:  lineIndex + 1,
						Token: rule.name,
					})
				}
			}
		}
	}

	sort.Slice(violations, func(i, j int) bool {
		if violations[i].Path != violations[j].Path {
			return violations[i].Path < violations[j].Path
		}
		if violations[i].Line != violations[j].Line {
			return violations[i].Line < violations[j].Line
		}
		return violations[i].Token < violations[j].Token
	})
	return violations, nil
}

func scopedPaths(repositoryRoot string) ([]string, error) {
	seen := make(map[string]struct{})
	addFile := func(relativePath string) error {
		path := filepath.Join(repositoryRoot, filepath.FromSlash(relativePath))
		info, err := os.Lstat(path)
		if os.IsNotExist(err) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("stat %s: %w", relativePath, err)
		}
		if info.Mode().IsRegular() {
			seen[filepath.ToSlash(relativePath)] = struct{}{}
		}
		return nil
	}
	addTree := func(relativeRoot string, extensions map[string]bool) error {
		root := filepath.Join(repositoryRoot, filepath.FromSlash(relativeRoot))
		if _, err := os.Lstat(root); os.IsNotExist(err) {
			return nil
		} else if err != nil {
			return fmt.Errorf("stat %s: %w", relativeRoot, err)
		}
		return filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.Type()&os.ModeSymlink != 0 {
				if entry.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
			if entry.IsDir() || !extensions[filepath.Ext(path)] {
				return nil
			}
			relativePath, err := filepath.Rel(repositoryRoot, path)
			if err != nil {
				return err
			}
			seen[filepath.ToSlash(relativePath)] = struct{}{}
			return nil
		})
	}

	for _, root := range []string{"model/Umpire", "model/Temporal"} {
		if err := addTree(root, map[string]bool{".lean": true, ".md": true, ".json": true}); err != nil {
			return nil, err
		}
	}
	if err := addTree("tools/umpire", map[string]bool{".go": true, ".md": true, ".json": true}); err != nil {
		return nil, err
	}
	for _, path := range []string{
		"model/Umpire.lean",
		"model/UmpireTests.lean",
		"model/Temporal.lean",
		"model/TemporalModelTests.lean",
		"model/TemporalExperimentalTests.lean",
		"model/README.md",
		"model/ARCHITECTURE.md",
		"model/Umpire/ARCHITECTURE.md",
	} {
		if err := addFile(path); err != nil {
			return nil, err
		}
	}

	planMatches, err := filepath.Glob(filepath.Join(repositoryRoot, ".plans", "UMPIRE4_*.md"))
	if err != nil {
		return nil, fmt.Errorf("find active Umpire4 plans: %w", err)
	}
	for _, path := range planMatches {
		relativePath, err := filepath.Rel(repositoryRoot, path)
		if err != nil {
			return nil, err
		}
		seen[filepath.ToSlash(relativePath)] = struct{}{}
	}

	for _, specID := range downstreamSpecs {
		if err := addOpenFlowRecord(repositoryRoot, ".flow/specs", specID, "open", seen); err != nil {
			return nil, err
		}
		if err := addOpenTasks(repositoryRoot, specID, seen); err != nil {
			return nil, err
		}
	}

	paths := make([]string, 0, len(seen))
	for path := range seen {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	return paths, nil
}

func addOpenFlowRecord(repositoryRoot, relativeDirectory, id, wantedStatus string, seen map[string]struct{}) error {
	relativeJSON := filepath.ToSlash(filepath.Join(relativeDirectory, id+".json"))
	path := filepath.Join(repositoryRoot, filepath.FromSlash(relativeJSON))
	content, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read %s: %w", relativeJSON, err)
	}
	var metadata struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal(content, &metadata); err != nil {
		return fmt.Errorf("decode %s: %w", relativeJSON, err)
	}
	if metadata.Status != wantedStatus {
		return nil
	}
	seen[relativeJSON] = struct{}{}
	relativeMarkdown := strings.TrimSuffix(relativeJSON, ".json") + ".md"
	if info, err := os.Lstat(filepath.Join(repositoryRoot, filepath.FromSlash(relativeMarkdown))); err == nil && info.Mode().IsRegular() {
		seen[relativeMarkdown] = struct{}{}
	} else if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("stat %s: %w", relativeMarkdown, err)
	}
	return nil
}

func addOpenTasks(repositoryRoot, specID string, seen map[string]struct{}) error {
	pattern := filepath.Join(repositoryRoot, ".flow", "tasks", specID+".*.json")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return fmt.Errorf("find tasks for %s: %w", specID, err)
	}
	for _, path := range matches {
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		var metadata struct {
			Status string `json:"status"`
		}
		if err := json.Unmarshal(content, &metadata); err != nil {
			return fmt.Errorf("decode %s: %w", path, err)
		}
		if metadata.Status == "done" {
			continue
		}
		relativeJSON, err := filepath.Rel(repositoryRoot, path)
		if err != nil {
			return err
		}
		relativeJSON = filepath.ToSlash(relativeJSON)
		seen[relativeJSON] = struct{}{}
		relativeMarkdown := strings.TrimSuffix(relativeJSON, ".json") + ".md"
		if info, err := os.Lstat(filepath.Join(repositoryRoot, filepath.FromSlash(relativeMarkdown))); err == nil && info.Mode().IsRegular() {
			seen[relativeMarkdown] = struct{}{}
		} else if err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("stat %s: %w", relativeMarkdown, err)
		}
	}
	return nil
}

func buildRetiredRules() []tokenRule {
	exactTokens := []string{
		"Declaration" + "Id",
		"Declaration" + "Kind",
		"Declaration" + "Metadata",
		"Declaration" + "Error",
		"Semantic" + "Source",
		"Semantic" + "Value",
		"Semantic" + "TraceStep",
		"Semantic" + "Trace",
		"Semantic" + "Coordinate",
		"Semantic" + "Derivation",
		"semantic" + "DigestOf",
		"semantic" + "Digest",
		"semantic" + "Identity",
		"Bound" + "Unit",
		"Typed" + "Bound",
		"Property" + "Bound",
		"Query" + "Bounds",
		"Qualification" + "Status",
		"Qualification" + "FailureKind",
		"Qualification" + "Diagnostic",
		"Qualification" + "Result",
		"Qualification" + "Receipt",
		"Qualification" + "Profile",
		"Qualified" + "Trace",
		"qualify" + "Evidence",
		"evaluate" + "QualifiedProperty",
		"validate" + "QualifiedTrace",
		"Catalog" + "ProjectionBinding",
		"Require" + "Projection",
		"Projection" + "Record",
		"Projection" + "Manifest",
		"Checked" + "Refinement",
		"Refinement" + "Declaration",
		"Refinement" + "Result",
		"Refinement" + "Error",
		"check" + "Refinement",
		"Conformance" + "Result",
		"evaluate" + "Conformance",
		"check" + "Conformance",
		"semantic" + "Conformance",
		"umpire-drive-plan/" + "v1",
		"umpire-experiment/" + "v1",
		"umpire-gen-regression-" + "projections",
		"umpire-check-regression-" + "projections",
		"Umpire.Observation." + "Qualification",
		"Umpire." + "Refinement",
		"Temporal.System.Nexus." + "Refinement",
	}

	rules := make([]tokenRule, 0, len(exactTokens)+5)
	for _, token := range exactTokens {
		variants := []string{regexp.QuoteMeta(token)}
		if token[0] >= 'A' && token[0] <= 'Z' && !strings.ContainsAny(token, "./") {
			lowerCamel := strings.ToLower(token[:1]) + token[1:]
			variants = append(variants, regexp.QuoteMeta(lowerCamel))
		}
		rules = append(rules, tokenRule{
			name:    token,
			pattern: regexp.MustCompile(`(^|[^A-Za-z0-9_])(?:` + strings.Join(variants, "|") + `)(?:V[0-9]+)?([^A-Za-z0-9_]|$)`),
		})
	}
	for _, token := range []string{"bounds", "omissions", "qualification", "qualified"} {
		rules = append(rules, tokenRule{
			name:    `"` + token + `"`,
			pattern: regexp.MustCompile(`"` + token + `"`),
		})
	}
	rules = append(rules, tokenRule{
		name:    ".qualified",
		pattern: regexp.MustCompile(`[.]qualified([^A-Za-z0-9_]|$)`),
	})
	return rules
}

func allowedNegativeFixture(relativePath, token string) bool {
	allowed := map[string]map[string]bool{
		"model/Umpire/Case/ProtoJSON.lean": {
			`"bounds"`: true,
		},
		"tools/umpire/cmd/umpire-gen-regression-views/render_test.go": {
			"umpire-experiment/" + "v1": true,
		},
		"tools/umpire/internal/artifactv2/artifact_test.go": {
			"umpire-experiment/" + "v1": true,
			"semantic" + "Identity":     true,
		},
		"tools/umpire/internal/ir/catalog.go": {
			"." + "qualified": true,
		},
		"tools/umpire/temporal/testdata/async-nexus-case.json": {
			`"bounds"`: true,
		},
		"tools/umpire/temporal/testdata/get-system-info-case.json": {
			`"bounds"`: true,
		},
	}
	for _, class := range []string{
		"cleanup-failure-after-proved-violation",
		"cross-run-isolation",
		"inconclusive",
		"satisfied",
		"static-preparation-rejection",
		"violated",
	} {
		allowed[filepath.ToSlash(filepath.Join(
			"tools/umpire/testdata/case-runtime-conformance", class, "case.json",
		))] = map[string]bool{`"bounds"`: true}
	}
	return allowed[relativePath][token]
}
