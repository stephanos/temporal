package retiredvocabulary

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
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
	"fn-28-portable-evaluation-contract-and",
	"fn-29-bounded-production-canary-execution-and",
	"fn-30-release-evidence-graph-and-manual",
	"fn-32-add-umpire-refinement-and-the-first",
	"fn-33-run-serial-bounded-semantic-exploration",
	"fn-46-export-lean-model-module-impact-index",
	"fn-70-scheduled-canary-proof-of-concept-as-a",
	"fn-74-deepen-testpilot-worker-activation",
	"fn-78-typed-temporal-authoring-and-checked",
	"fn-79-deferred-nexus-operation-cancellation",
}

// requiredFiles names the individual files the scan must cover. Each one is a
// facade or document a rename sweep is likely to move, so a missing entry is a
// silent scan hole rather than an absence to tolerate.
var requiredFiles = []string{
	"model/Umpire.lean",
	"model/UmpireTests.lean",
	"model/Temporal.lean",
	"model/TemporalModelTests.lean",
	"model/Shared.lean",
	"model/Testpilot.lean",
	"model/README.md",
	"model/AUTHORING.md",
	"model/ARCHITECTURE.md",
	"model/Umpire/ARCHITECTURE.md",
}

type scanRoot struct {
	path       string
	extensions map[string]bool
}

// scanRoots names the trees the scan walks. A root that has moved is a hole in
// the scan, so the walk fails closed on a missing one just as requiredFiles does.
var scanRoots = []scanRoot{
	{path: "model/Umpire", extensions: modelExtensions},
	{path: "model/Temporal", extensions: modelExtensions},
	{path: "model/Testpilot", extensions: modelExtensions},
	{path: "model/Shared", extensions: modelExtensions},
	{path: "tools/umpire", extensions: facadeExtensions},
	{path: "common/testing/testpilot", extensions: facadeExtensions},
	{path: "common/testing/testpilot/temporal", extensions: facadeExtensions},
	{path: "tests/testcore/testpilot", extensions: facadeExtensions},
	{path: "api/testpilot", extensions: facadeExtensions},
	{path: "proto/internal/temporal/server/api/testpilot", extensions: facadeExtensions},
}

var (
	modelExtensions  = map[string]bool{".lean": true, ".md": true, ".json": true}
	facadeExtensions = map[string]bool{".go": true, ".md": true, ".json": true, ".proto": true}
)

// DownstreamSpecs lists the Flow specs whose open records the scan covers.
func DownstreamSpecs() []string { return slices.Clone(downstreamSpecs) }

// RequiredFiles lists the individual files the scan refuses to run without.
func RequiredFiles() []string { return slices.Clone(requiredFiles) }

// ScanRoots lists the trees the scan refuses to run without.
func ScanRoots() []string {
	roots := make([]string, 0, len(scanRoots))
	for _, root := range scanRoots {
		roots = append(roots, root.path)
	}
	return roots
}

var retiredRules, retiredRulesError = buildRetiredRules()

// Check scans only live Umpire source, current Generated Views, active Umpire4
// documentation, and the open downstream Flow closure.
func Check(repositoryRoot string) ([]Violation, error) {
	if retiredRulesError != nil {
		return nil, retiredRulesError
	}

	paths, err := scopedPaths(repositoryRoot)
	if err != nil {
		return nil, err
	}

	var violations []Violation
	for _, relativePath := range paths {
		// The registry must spell the tokens it rejects. Keep that unavoidable
		// implementation file out of its own input rather than allowlisting each
		// token and accidentally weakening checks in the rest of the package.
		if relativePath == "tools/umpire/internal/retiredvocabulary/check.go" {
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

	slices.SortFunc(violations, func(left, right Violation) int {
		if comparison := strings.Compare(left.Path, right.Path); comparison != 0 {
			return comparison
		}
		if left.Line != right.Line {
			return left.Line - right.Line
		}
		return strings.Compare(left.Token, right.Token)
	})
	return violations, nil
}

func scopedPaths(repositoryRoot string) ([]string, error) {
	seen := make(map[string]struct{})

	for _, root := range scanRoots {
		if err := addTree(repositoryRoot, root.path, root.extensions, seen); err != nil {
			return nil, err
		}
	}
	for _, path := range requiredFiles {
		if err := addFile(repositoryRoot, path, seen); err != nil {
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
	slices.Sort(paths)
	return paths, nil
}

// missingScannedPath is the one message every fail-closed branch reports, so the
// test contract for a scan hole is spelled once.
func missingScannedPath(relativePath string) error {
	return fmt.Errorf("scanned path %s does not exist", relativePath)
}

func addFile(repositoryRoot, relativePath string, seen map[string]struct{}) error {
	path := filepath.Join(repositoryRoot, filepath.FromSlash(relativePath))
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return missingScannedPath(relativePath)
	}
	if err != nil {
		return fmt.Errorf("stat %s: %w", relativePath, err)
	}
	if info.Mode().IsRegular() {
		seen[filepath.ToSlash(relativePath)] = struct{}{}
	}
	return nil
}

func addTree(repositoryRoot, relativeRoot string, extensions map[string]bool, seen map[string]struct{}) error {
	root := filepath.Join(repositoryRoot, filepath.FromSlash(relativeRoot))
	if _, err := os.Lstat(root); os.IsNotExist(err) {
		return missingScannedPath(relativeRoot)
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

func addOpenFlowRecord(repositoryRoot, relativeDirectory, id, wantedStatus string, seen map[string]struct{}) error {
	relativeJSON := filepath.ToSlash(filepath.Join(relativeDirectory, id+".json"))
	path := filepath.Join(repositoryRoot, filepath.FromSlash(relativeJSON))
	content, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return missingScannedPath(relativeJSON)
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

// bareWord matches a token that is one ordinary word: a single letter followed
// only by lowercase letters. Retiring such a token would ban ordinary English,
// because every rule also matches its lowerCamel variant.
var bareWord = regexp.MustCompile(`^[A-Za-z][a-z]*$`)

func validateRetiredToken(token string) error {
	if token == "" {
		return errors.New("retired token must not be empty")
	}
	if bareWord.MatchString(token) {
		return fmt.Errorf("retired token %q is a bare word; retire a compound identifier, module path, macro name, or snake_case keyword instead", token)
	}
	return nil
}

func buildRetiredRules() ([]tokenRule, error) {
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
		"Transition" + "Kernel",
		"Transition" + "Result",
		"Kernel" + "Metadata",
		"Kernel" + "Availability",
		"CASE_DEFINITION_KIND_" + "KERNEL",
		"CASE_DEFINITION_KIND_" + "OBSERVATION",
		"CASE_DEFINITION_KIND_" + "BEHAVIOR",
		"CASE_DEFINITION_KIND_" + "EXPERIMENT_SPACE",
		"CASE_DEFINITION_KIND_" + "VARIATION_AXIS",
		"CASE_DEFINITION_KIND_" + "COVERAGE_GOAL",
		"CASE_DEFINITION_KIND_" + "CHOICE",
		"CASE_DEFINITION_KIND_" + "FAULT",
		"Checked" + "Target",
		"Authored" + "Target",
		"Query" + "Target",
		"Target" + "Declaration",
		"Target" + "Definition",
		"Target" + "Composition",
		"Target" + "Projection",
		"Target" + "BehaviorDomain",
		"Target" + "BehaviorDescription",
		"Target" + "BehaviorClosure",
		"FiniteTarget" + "Definition",
		"FiniteTarget" + "AdmissionError",
		"ValidatedFinite" + "Table",
		"ValidatedFinite" + "Model",
		"Authoring" + "Occurrence",
		"Authoring" + "Diagnostic",
		"Capability" + "Contract",
		"Capability" + "Provider",
		"Capability" + "Connector",
		"Law" + "Definition",
		"Law" + "Witness",
		"Meaning" + "Provision",
		"canonical" + "Behavior",
		"compose" + "Target",
		"check" + "Target",
		"elaborate" + "Target",
		"Umpire." + "Target",
		"Umpire." + "TargetTests",
		"Shared." + "Transition",
		"Shared." + "TraceReplay",
		"Behavior" + "Declaration",
		"Checked" + "Behavior",
		"Behavior" + "Spec",
		"ExactSequence" + "Spec",
		"Property" + "Declaration",
		"Property" + "Spec",
		"Property" + "Authoring",
		"Property" + "CaseGroup",
		"Property" + "Exception",
		"same" + "StepCases",
		"quiescent" + "Within",
		"model" + "Outcome",
		"behavior" + "%",
		"bounded_" + "response%",
		"Umpire." + "Behavior",
		"Umpire." + "Planning",
		"Umpire." + "ExecutionHandoff",
		"Planner" + "Run",
		"Planner" + "Instrumentation",
		"IncrementalPlanner" + "Kernel",
		"FinitePlanner" + "AdmissionError",
		"Execution" + "Handoff",
		"Experiment" + "Spec",
		"Drive" + "Plan",
		"Artifact" + "Intent",
		"ArtifactFault" + "Intent",
		"analyze" + "Cases",
		"Umpire.Property." + "Language",
		"Umpire.Property." + "Authoring",
		"Umpire.Property." + "Trace",
		"Umpire.Property." + "Evaluation",
		"Umpire.Property." + "Fields",
		"Property" + "Case",
		"Property" + "LimitProfile",
		"Property" + "ScopedClock",
		"Behavior" + "Trace",
		"Behavior" + "CheckContext",
		"Named" + "Occurrence",
		"Occurrence" + "Bound",
		"Occurrence" + "Order",
		"Resource" + "Role",
		"Sequence" + "Occurrence",
		"check" + "Behavior",
		"checked" + "Behavior",
		"check" + "Property",
		"Resolved" + "PropertyClause",
		"Resolved" + "PropertyCase",
		"Resolved" + "PropertyCaseGroup",
		"Resolved" + "PropertyException",
		"Resolved" + "PropertySameStepClause",
		"Resolved" + "PropertyScopedClause",
		"Resolved" + "GuardedTemporalClause",
		"guarded" + "QuiescentWithin",
		"Query" + "Declaration",
		"Query" + "Spec",
		"QueryAuthoring" + "Input",
		"QueryLimit" + "Spec",
		"Query" + "Quantifier",
		"Query" + "Claim",
		"Query" + "ExercisePolicy",
		"BehaviorPhase" + "Limits",
		"TieBreak" + "Policy",
		"CheckedQuery" + "Target",
		"CheckedQuery" + "Model",
		"check" + "Query",
		"Umpire.Query." + "Language",
		"Umpire.Query." + "Authoring",
		"find-" + "witness",
		"find-" + "counterexample",
		"select-" + "behavior",
		"semantic-" + "transitions",
		"selected-" + "actions",
		"observation-" + "positions",
		"candidate-" + "evaluations",
		"experiment-" + "specs",
		"Umpire." + "Observation",
		"ObservationMapping" + "Declaration",
		"ObservationMapping" + "Spec",
		"CheckedObservation" + "Plan",
		"ObservationCheck" + "Context",
		"check" + "Observation",
		"Observation" + "Configuration",
		"SemanticVerdict" + "Status",
		"SemanticVerdict" + "Diagnostic",
		"SemanticVerdict" + "FailureKind",
		"StrictQuery" + "Summary",
		"StrictQuery" + "Status",
		"Evidence" + "Link",
		"Evidence" + "Bundle",
		"ImplementationLink" + "KnownGap",
		"Forward" + "Simulation",
		"Kernel" + "Morphism",
		"ProjectionSentinel" + "Descriptor",
		"Umpire.Artifact." + "Runtime",
		"Umpire." + "Space",
		"Experiment" + "Space",
		"LoweredSpace" + "Point",
		"SpaceMetadata" + "Projection",
		"Candidate" + "Universe",
		"Exploration" + "Session",
		"PinnedExperiment" + "Spec",
		"Exploration" + "Omission",
		"Umpire." + "SemanticInventory",
		"SEMANTIC_" + "INVENTORY",
		"temporal-model-semantic-" + "inventory",
		"Scoped" + "Contract",
		"Scoped" + "Clause",
		"Scoped" + "Evidence",
		"Scoped" + "Endpoint",
		"Scoped" + "Value",
		"Scoped" + "Transition",
		"Scoped" + "Predicate",
		"Scoped" + "Correlation",
		"Scoped" + "Limits",
		"Scoped" + "Identity",
		"Scoped" + "Binding",
		"Scoped" + "Clock",
		"Scoped" + "Comparison",
		"Scoped" + "Operand",
		"Scoped" + "ProjectionRule",
		"Scoped" + "FieldPolicy",
		"Scoped" + "FieldDisposition",
		"Scoped" + "CaptureDeclaration",
		"Scoped" + "CaptureRef",
		"Scoped" + "EvidenceBinding",
		"Scoped" + "EvidenceRule",
		"Scoped" + "EvidenceProjection",
		"Scoped" + "EvidenceField",
		"Scoped" + "EvidenceMeaning",
		"Scoped" + "CorrelationGroup",
		"Scoped" + "ComparisonOperator",
		"Scoped" + "PredicateField",
		"runtime" + "Prefix",
		"deliberately" + "Closed",
		"ContractHorizon" + "Definition",
		"await_" + "outcome",
		"resulting_" + "state",
		"Testpilot." + "Scoped",
		"Shared.Scoped" + "Projection",
		"Shared.Scoped" + "Obligation",
		"Umpire.Case." + "Scoped",
		"Umpire.Case." + "ProtoJSON",
		"Umpire.Property." + "Scoped",
		"PropertyScoped" + "Clause",
		"PropertyScoped" + "Endpoint",
		"Lowering" + "Error",
		"Case" + "Metadata",
		"CaseDefinition" + "Kind",
		"CaseDefinition" + "Binding",
		"CaseKnown" + "Gap",
		"Slot" + "Bridge",
		"ProfileSpec." + "Capabilities",
		"terminal" + "Disposition",
		"Trigger" + "Disposition",
		"Entrypoint" + "Context",
		"umpire-scoped-" + "fixtures",
		"Authoring." + "Monitor",
		"Umpire." + "Refinement",
		"Temporal.System.Nexus." + "Refinement",
		"temporal-nexus" + "2",
		"temporal-nexus" + "3",
		"Temporal.ImplementationLink" + "Tests",
		"temporal-model-" + "inspect",
		"umpire-list-" + "nexus",
		"umpire-explain-" + "nexus",
		"selected_" + "actions",
		"candidate_" + "evaluations",
		// The compound rule requires a non-identifier boundary on both sides, so a longer
		// name built on a held one is not held by it.
		"ExperimentSpace" + "Declaration",
		"TargetBehaviorDomain" + "Availability",
		"experiment" + "Specs",
		// The retired Limit units, in the lowerCamel spelling their wire keys used.
		"candidate" + "Evaluations",
		"selected" + "Actions",
		"semantic" + "Transitions",
		"observation" + "Positions",
		// The Go instruction kind, now `Opcode`. `CapabilityBridge` and `CapabilityEffect`
		// are the live Driver seam and keep their names; the identifier boundary separates
		// them from the bare type.
		"testpilot." + "Capability",
		"model" + "Lint",
		"model" + "LintTests",
		// The whole-Program templates, the fixture-named `case` form and the success slice's
		// set, retired by fn-85 .11: a Case is produced from a set through a realization value.
		"Temporal.Case." + "Template",
		"Case.Template." + "NexusOperation",
		"Case.Template." + "Workflow",
		"Case/" + "Template",
		"Case.Tests." + "ProofPoint",
		"Case/Tests/" + "ProofPoint",
		"Case.Tests." + "Template",
		"Case/Tests/" + "Template",
		"case" + "Template",
		"nexusSuccess" + "Set",
		"Hook" + "Placement",
		"fault" + "RuleId",
		"testpilot" + "ProtoJSONFixture",
		// The observed read path is one use of the Case coordinate walker, derived by
		// `Umpire.Case.Projection.lower`.
		"Umpire.Case." + "Observed",
		// The per-caller admission-error arm and the Switch example's hand-built search view,
		// both replaced by `Umpire.Search.admit` and its `AdmissionDiagnostic`.
		"invalid" + "Planner",
		"Switch." + "incrementalKernel",
		// fn-87 protocol names the glossary renamed. `clauseId` is not held: Umpire's Property
		// clauses keep that identifier, and the protocol JSON key is pinned by the migration
		// equivalence test and the regenerated fixtures instead.
		"Run" + "Status",
		"clause" + "_id",
		"GetClause" + "Id",
		"Correlated" + "Value",
		"ContractRule" + "Definition",
		"ContractState" + "Definition",
		"ContractTransition" + "Definition",
		"ContractCapture" + "Definition",
		// `InvokeRPC` is not held: hand-written Go keeps the initialism for the Opcode and the Driver
		// method (staticcheck ST1003), so only the Lean spelling of the constructor is retired.
		"Response" + "Projections",
		"Response" + "Projection",
		"Projection" + "Target",
		"Projection" + "Kind",
		"OpaqueCapability" + "Type",
		"capability" + "_slot_id",
		"Capability" + "SlotId",
		// The Opcode facade on the generic package, now `InstructionOpcode`; hand-written Go no
		// longer calls an Opcode a capability.
		"Instruction" + "Capability",
		"invoke" + "RPC",
		"Role" + "Definition",
		"Slot" + "Definition",
		"Observation" + "Definition",
		"Entrypoint" + "Definition",
		"Cleanup" + "Definition",
		"Instruction" + "Definition",
		"Instruction" + "Ref",
		// fn-87 folds the Program and Contract expression vocabularies into one Expression whose
		// references are one Reference; the Lean namespaces and every per-context operator retire.
		"Program" + "Expression",
		"Contract" + "Expression",
		"Program" + "Expr",
		"Contract" + "Expr",
		"Equals" + "Expression",
		"Program" + "PathExpression",
		"Contract" + "PathExpression",
		"Program" + "PresentExpression",
		"Contract" + "PresentExpression",
		"Program" + "EqualsExpression",
		"Contract" + "EqualsExpression",
		"Program" + "CompareExpression",
		"Contract" + "CompareExpression",
		"Program" + "NotExpression",
		"Contract" + "NotExpression",
		"Program" + "AllExpression",
		"Contract" + "AllExpression",
		"Program" + "AnyExpression",
		"Contract" + "AnyExpression",
		"Slot" + "Ref",
		"Run" + "Ref",
		"Observation" + "Ref",
		"Capture" + "Ref",
		"Environment" + "Ref",
		"InstructionOutcome" + "Ref",
		"RunEventField" + "Ref",
		"CorrelatedCapture" + "Ref",
		// fn-87 folds the correlated condition vocabulary into the same Expression, and an evidence-lift
		// guard's string equality into an EQUAL comparison.
		"Correlated" + "Predicate",
		"Correlated" + "PredicateField",
		"Correlated" + "Comparison",
		"Correlated" + "ComparisonOperator",
		"Correlated" + "Operand",
		"Correlated" + "Correlation",
		"Correlated" + "CorrelationGroup",
		"guard" + "_equals_text",
		"Guard" + "EqualsText",
		// fn-87 gives a deadline its bound oneof, types a capture as a SingularType, and folds the three
		// binding shapes into one named value per side.
		"Contract" + "Deadline",
		"Contract" + "CaptureType",
		"Correlated" + "Binding",
		"CorrelatedEvidence" + "Field",
		"CorrelatedEvidence" + "Binding",
		// fn-87 removes the natural Value arm, which the unsigned integer arm already spelled.
		"Natural" + "Value",
		"natural" + "_value",
		// fn-87 replaces an instruction's dependency list with after, written only where it is not the
		// previous instruction. The bare field word stays ordinary prose; the generated accessor retires.
		"Get" + "Dependencies",
		// fn-87 derives a Program's environment bindings and each instruction's activation reservations
		// and outcome fields at preparation, so the declarations that wrote them retire.
		"Environment" + "Definition",
		"ActivationReservation" + "Definition",
		"Activation" + "Reservations",
		"activation" + "_reservations",
		"InstructionOutcome" + "Definition",
		"OutcomeField" + "Definition",
		// fn-87 replaces the opaque provenance bytes with typed rows the runtime still never reads.
		"Producer" + "Data",
		"Get" + "ProducerData",
		"producer" + "_data",
		// fn-86 .3 removes the superseded Testpilot instruction shapes: the untyped Nexus start, the
		// untyped completion whose result is an Expression, the untyped handler response and the enum
		// that named its kind. `RespondNexus` and `NexusResponseKind` are held bare: the identifier
		// boundary on both sides keeps the WorkflowService method `RespondNexusTaskCompleted` and the
		// typed `NexusOperationCompletion` live. The other two names are HistoryService methods the
		// generated `Temporal.API` spells in both cases, so only their protocol-qualified Go
		// spellings -- the oneof arm type and its accessor -- are held.
		"Respond" + "Nexus",
		"NexusResponse" + "Kind",
		"Instruction_StartNexus" + "Operation",
		"Instruction_CompleteNexus" + "Operation",
		"GetStartNexus" + "Operation",
		"GetCompleteNexus" + "Operation",
		// fn-87 spells a field path as a string in its grammar, so the structured path segment and its
		// selector messages retire; FieldPath stays the concept's name.
		"FieldPath" + "Segment",
		"Repeated" + "Wildcard",
		"MapKey" + "Selector",
		"Presence" + "Selector",
		"Oneof" + "Selector",
	}

	rules := make([]tokenRule, 0, len(exactTokens)+5)
	for _, token := range exactTokens {
		if err := validateRetiredToken(token); err != nil {
			return nil, err
		}
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
	// The retired protocol enum values only ever occur as whole SCREAMING_SNAKE constants, where the
	// compound rules' identifier boundary is an underscore on both sides and can never match. One
	// prefix rule covers the whole family, so a stale fixture carrying SCOPED_ENDPOINT_RUNTIME_PREFIX
	// is rejected.
	rules = append(rules, tokenRule{
		name:    "SCOPED_*",
		pattern: regexp.MustCompile(`(^|[^A-Za-z0-9_])SCOPED_[A-Z0-9_]+`),
	})
	// The fn-87 enum values, held the same way: the renamed enum's whole value family, and the two
	// values renamed alone, whether spelled as the whole constant or as its suffix.
	rules = append(rules,
		tokenRule{name: "RUN_STATUS_*", pattern: regexp.MustCompile(`(^|[^A-Za-z0-9_])RUN_STATUS_[A-Z0-9_]+`)},
		tokenRule{name: "CONTRACT_STATE_STATUS_NONTERMINAL", pattern: regexp.MustCompile(`(^|[^A-Za-z0-9_])CONTRACT_STATE_STATUS_NONTERMINAL([^A-Za-z0-9_]|$)`)},
		tokenRule{name: "PROTOCOL_NON_SUCCESS", pattern: regexp.MustCompile(`(^|[^A-Za-z0-9_])[A-Z0-9_]*PROTOCOL_NON_SUCCESS([^A-Za-z0-9_]|$)`)},
		tokenRule{name: "PROJECTION_KIND_*", pattern: regexp.MustCompile(`(^|[^A-Za-z0-9_])PROJECTION_KIND_[A-Z0-9_]+`)},
		tokenRule{name: "CORRELATED_PREDICATE_FIELD_*", pattern: regexp.MustCompile(`(^|[^A-Za-z0-9_])CORRELATED_PREDICATE_FIELD_[A-Z0-9_]+`)},
		tokenRule{name: "CORRELATED_COMPARISON_OPERATOR_*", pattern: regexp.MustCompile(`(^|[^A-Za-z0-9_])CORRELATED_COMPARISON_OPERATOR_[A-Z0-9_]+`)},
		// Fault data moved into the Run Event payload, read through a path rather than a coordinate.
		tokenRule{name: "RUN_EVENT_FIELD_FAULT_*", pattern: regexp.MustCompile(`(^|[^A-Za-z0-9_])RUN_EVENT_FIELD_FAULT_[A-Z0-9_]+`)},
		// The natural kind folded into UINT64, and the wire entrypoint kind became the Go runtime's own
		// classification, whose constants spell no prefix.
		tokenRule{name: "SCALAR_KIND_NATURAL", pattern: regexp.MustCompile(`(^|[^A-Za-z0-9_])SCALAR_KIND_NATURAL([^A-Za-z0-9_]|$)`)},
		tokenRule{name: "ENTRYPOINT_KIND_*", pattern: regexp.MustCompile(`(^|[^A-Za-z0-9_])ENTRYPOINT_KIND_[A-Z0-9_]+`)},
		// The provenance kinds the opaque payload spelled, now DEFINITION_KIND_ and KNOWN_GAP_KIND_ values.
		tokenRule{name: "CASE_DEFINITION_KIND_*", pattern: regexp.MustCompile(`(^|[^A-Za-z0-9_])CASE_DEFINITION_KIND_[A-Z0-9_]+`)},
		tokenRule{name: "CASE_KNOWN_GAP_KIND_*", pattern: regexp.MustCompile(`(^|[^A-Za-z0-9_])CASE_KNOWN_GAP_KIND_[A-Z0-9_]+`)},
	)
	// The generation-numbered module and identity roots. A leading hyphen is excluded because the
	// only occurrences in that shape are immutable Flow spec slugs, which name closed records rather
	// than anything in the tree; the kebab identity roots the model actually emitted are held by the
	// `temporal-nexus2` and `temporal-nexus3` rules above.
	for _, generation := range []string{"2", "3"} {
		rules = append(rules, tokenRule{
			name:    "Nexus" + generation,
			pattern: regexp.MustCompile(`(^|[^A-Za-z0-9_-])[Nn]exus` + generation + `([^A-Za-z0-9_]|$)`),
		})
	}
	// The retired `model` command, which fn-85 replaced with `machine` over step functions. The word
	// itself stays live everywhere -- `model/` is the tree, `model:` is a key of `property` and
	// `scenario`, and `DeclaredModel` is what both of them resolve to -- so the identifier-boundary
	// shape the rules above use would reject the whole surface. What is retired is the declaration,
	// and a declaration is a column-0 keyword followed by the author's name and nothing else, which
	// is a shape no prose line and no key takes.
	rules = append(rules, tokenRule{
		name:    "model <name>",
		pattern: regexp.MustCompile(`^model [A-Za-z][A-Za-z0-9_']*$`),
	})
	// The Lake executable this spec renamed to `umpire-case`. The Driver's reservation carriers
	// spell three unrelated wire constants that begin with the retired name, and Flow spec slugs end
	// with it, so a non-hyphen boundary on both sides holds the executable name alone.
	rules = append(rules, tokenRule{
		name:    "temporal-testpilot",
		pattern: regexp.MustCompile(`(^|[^A-Za-z0-9_-])temporal-testpilot([^A-Za-z0-9_-]|$)`),
	})
	// `resultingState` stays live as an `Umpire.Property` trace and clause field constructor, so
	// only the retired Nexus `require` spelling can be held. The keyword always follows the clause
	// label and its colon, which the field constructor never does.
	rules = append(rules, tokenRule{
		name:    "require <label>: resultingState",
		pattern: regexp.MustCompile(`require +[A-Za-z0-9_]+ *: *resultingState`),
	})
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
	return rules, nil
}

func allowedNegativeFixture(relativePath, token string) bool {
	allowed := map[string]map[string]bool{
		"tools/umpire/cmd/umpire-gen-regression-views/render_test.go": {
			"umpire-experiment/" + "v1": true,
		},
		"tools/umpire/internal/artifactv2/artifact_test.go": {
			"umpire-experiment/" + "v1": true,
			"semantic" + "Identity":     true,
		},
		"common/testing/testpilot/internal/ir/catalog.go": {
			"." + "qualified": true,
		},
		"tests/testcore/testpilot/testdata/get-system-info-case.json": {
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
			"common/testing/testpilot/testdata/case-runtime-conformance", class, "case.json",
		))] = map[string]bool{`"bounds"`: true}
	}
	return allowed[relativePath][token]
}
