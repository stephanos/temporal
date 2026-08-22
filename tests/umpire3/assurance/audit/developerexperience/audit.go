package developerexperience

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"time"

	"go.temporal.io/server/tests/umpire3/regression"
	"go.temporal.io/server/tests/umpire3/scenario"
	"go.temporal.io/server/tests/umpire3/scenario/nexus"
	"go.temporal.io/server/tests/umpire3/scenario/workflow"
)

const FormatVersion = "umpire3/developer-ux-audit/v1"

type Capability string

const (
	CapabilityFirstRegression Capability = "first-regression"
	CapabilityPartialOrder    Capability = "partial-order"
	CapabilityRuntimeIdentity Capability = "runtime-identity"
	CapabilityTypedFault      Capability = "typed-fault"
)

type Case struct {
	Capability      Capability `json:"capability"`
	ScenarioDigest  string     `json:"scenarioDigest"`
	ExplainDigest   string     `json:"explainDigest"`
	PathCount       int        `json:"pathCount"`
	IdentityCount   int        `json:"identityCount"`
	FaultCount      int        `json:"faultCount"`
	ModelReplay     string     `json:"modelReplay"`
	ExperimentCount int        `json:"experimentCount"`
}

type Promotion struct {
	SourceDigest      string   `json:"sourceDigest"`
	Imports           []string `json:"imports"`
	RequireRegression bool     `json:"requireRegression"`
}

type Report struct {
	FormatVersion  string    `json:"formatVersion"`
	EntryPoint     string    `json:"entryPoint"`
	Cases          []Case    `json:"cases"`
	Promotion      Promotion `json:"promotion"`
	ArtifactDigest string    `json:"artifactDigest"`
}

func Run(promotionSource string) (Report, error) {
	authored := authoringCases()
	report := Report{
		FormatVersion: FormatVersion,
		EntryPoint:    "regression.RequireRegression",
		Cases:         make([]Case, 0, len(authored)),
	}
	for _, item := range authored {
		compiled, err := compileCase(item.capability, item.scenario)
		if err != nil {
			return Report{}, err
		}
		report.Cases = append(report.Cases, compiled)
	}
	promotion, err := inspectPromotion(promotionSource)
	if err != nil {
		return Report{}, err
	}
	report.Promotion = promotion
	report.ArtifactDigest, err = report.computedDigest()
	if err != nil {
		return Report{}, err
	}
	if err := report.Validate(); err != nil {
		return Report{}, err
	}
	return report, nil
}

func (r Report) Validate() error {
	if r.FormatVersion != FormatVersion || r.EntryPoint != "regression.RequireRegression" {
		return errors.New("developer UX audit requires the public regression entry point")
	}
	required := []Capability{
		CapabilityFirstRegression,
		CapabilityPartialOrder,
		CapabilityRuntimeIdentity,
		CapabilityTypedFault,
	}
	if len(r.Cases) != len(required) {
		return errors.New("developer UX audit requires every representative authoring case")
	}
	for index, item := range r.Cases {
		if item.Capability != required[index] || !validDigest(item.ScenarioDigest) ||
			!validDigest(item.ExplainDigest) || item.PathCount <= 0 || item.ExperimentCount != item.PathCount ||
			item.ModelReplay != "checked" {
			return fmt.Errorf("developer UX case %q is incomplete", item.Capability)
		}
		switch item.Capability {
		case CapabilityFirstRegression:
			if item.PathCount != 1 {
				return errors.New("first regression must compile to one deterministic path")
			}
		case CapabilityPartialOrder:
			if item.PathCount < 2 {
				return errors.New("partial-order authoring must enumerate multiple paths")
			}
		case CapabilityRuntimeIdentity:
			if item.IdentityCount == 0 {
				return errors.New("runtime-identity authoring must ground a projected identity")
			}
		case CapabilityTypedFault:
			if item.FaultCount == 0 {
				return errors.New("typed-fault authoring must compile a fault")
			}
		default:
			return fmt.Errorf("unknown developer UX capability %q", item.Capability)
		}
	}
	if !r.Promotion.RequireRegression || !validDigest(r.Promotion.SourceDigest) ||
		len(r.Promotion.Imports) == 0 || !slices.IsSorted(r.Promotion.Imports) ||
		len(slices.Compact(append([]string(nil), r.Promotion.Imports...))) != len(r.Promotion.Imports) {
		return errors.New("developer UX audit requires a public campaign promotion")
	}
	for _, imported := range r.Promotion.Imports {
		if isArtifactPlumbing(imported) {
			return fmt.Errorf("campaign promotion imports artifact plumbing %q", imported)
		}
	}
	expectedDigest, err := r.computedDigest()
	if err != nil {
		return err
	}
	if r.ArtifactDigest != expectedDigest {
		return errors.New("developer UX audit digest does not match its contents")
	}
	return nil
}

type authoredCase struct {
	capability Capability
	scenario   scenario.Scenario
}

func authoringCases() []authoredCase {
	first := scenario.ProtocolAtomicScenario("ux-first-regression",
		[]scenario.Resource{scenario.Callback("callback")},
		scenario.OnePath(
			scenario.RecordCallbackResponse("respond"),
			scenario.RequireCallbackResponseConsistency(),
		))
	partial := scenario.ProtocolAtomicScenario("ux-partial-order",
		[]scenario.Resource{scenario.Callback("callback")},
		scenario.AllPaths(
			scenario.AnyOrder(
				scenario.RecordCallbackResponse("left"),
				scenario.RecordCallbackResponse("right"),
			),
			scenario.RequireCallbackResponseConsistency(),
		))
	update := workflow.Update("update")
	identity := workflow.Scenario("ux-runtime-identity", update,
		scenario.OnePath(update.Lifecycle(), update.CompletionThroughHistory()))
	operation := nexus.Operation("operation")
	fault := nexus.Scenario("ux-typed-fault", operation, scenario.OnePath(
		operation.Schedule(),
		operation.Dispatch(),
		operation.RequestCancellation(),
		operation.CommitCancellation(),
		operation.AcquireOwnership(),
		scenario.During(
			scenario.Drop("drop-retry",
				scenario.OnServices("nexus"),
				scenario.OnRoutes("/service/operation"),
				scenario.AtOccurrence(1, 1),
			),
			operation.Retry(),
		),
		operation.WorkerReturnsSuccess(),
		operation.PersistSuccess(),
		operation.CancellationSafety(),
	))
	return []authoredCase{
		{capability: CapabilityFirstRegression, scenario: first},
		{capability: CapabilityPartialOrder, scenario: partial},
		{capability: CapabilityRuntimeIdentity, scenario: identity},
		{capability: CapabilityTypedFault, scenario: fault},
	}
}

func compileCase(capability Capability, authored scenario.Scenario) (Case, error) {
	limits := scenario.Limits{
		MaxPaths: 8, MaxActions: 32, MaxStates: 256,
		MaxMemoryBytes: 4 << 20, MaxTime: 2 * time.Second,
	}
	first, err := scenario.Compile(context.Background(), authored, limits)
	if err != nil {
		return Case{}, fmt.Errorf("compile developer UX case %q: %w", capability, err)
	}
	second, err := scenario.Compile(context.Background(), authored, limits)
	if err != nil {
		return Case{}, fmt.Errorf("repeat developer UX case %q: %w", capability, err)
	}
	firstJSON, err := first.CanonicalJSON()
	if err != nil {
		return Case{}, err
	}
	secondJSON, err := second.CanonicalJSON()
	if err != nil {
		return Case{}, err
	}
	if !bytes.Equal(firstJSON, secondJSON) {
		return Case{}, fmt.Errorf("developer UX case %q is not deterministic", capability)
	}
	explained, err := regression.Explain(authored, limits)
	if err != nil {
		return Case{}, fmt.Errorf("explain developer UX case %q: %w", capability, err)
	}
	if !reflect.DeepEqual(first.Explain, explained) {
		return Case{}, fmt.Errorf("developer UX case %q differs through the public explain entry point", capability)
	}
	explainJSON, err := json.Marshal(explained)
	if err != nil {
		return Case{}, err
	}
	faultCount := 0
	for _, experiment := range first.Experiments {
		faultCount += len(experiment.Faults)
	}
	return Case{
		Capability: capability, ScenarioDigest: first.ScenarioDigest,
		ExplainDigest: digest(explainJSON), PathCount: len(first.Explain.Paths),
		IdentityCount: len(first.Explain.Identities), FaultCount: faultCount,
		ModelReplay: string(first.Explain.ModelReplay.Status), ExperimentCount: len(first.Experiments),
	}, nil
}

func inspectPromotion(source string) (Promotion, error) {
	parsed, err := parser.ParseFile(token.NewFileSet(), "promotion.go", source, 0)
	if err != nil {
		return Promotion{}, fmt.Errorf("parse campaign promotion: %w", err)
	}
	promotion := Promotion{SourceDigest: digest([]byte(source))}
	for _, imported := range parsed.Imports {
		path, unquoteErr := strconv.Unquote(imported.Path.Value)
		if unquoteErr != nil {
			return Promotion{}, fmt.Errorf("decode campaign promotion import: %w", unquoteErr)
		}
		if isArtifactPlumbing(path) {
			return Promotion{}, fmt.Errorf("campaign promotion imports artifact plumbing %q", path)
		}
		promotion.Imports = append(promotion.Imports, path)
	}
	slices.Sort(promotion.Imports)
	promotion.Imports = slices.Compact(promotion.Imports)
	ast.Inspect(parsed, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		selector, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || selector.Sel.Name != "RequireRegression" {
			return true
		}
		packageName, ok := selector.X.(*ast.Ident)
		promotion.RequireRegression = ok && packageName.Name == "regression"
		return true
	})
	if !promotion.RequireRegression {
		return Promotion{}, errors.New("campaign promotion does not call regression.RequireRegression")
	}
	return promotion, nil
}

func isArtifactPlumbing(path string) bool {
	for _, forbidden := range []string{
		"/mutation", "/cmd/", "/internal/", "/model", "/checker", "/protocol", "/replay",
	} {
		if strings.Contains(path, forbidden) {
			return true
		}
	}
	return false
}

func (r Report) computedDigest() (string, error) {
	canonical := r
	canonical.ArtifactDigest = ""
	encoded, err := json.Marshal(canonical)
	if err != nil {
		return "", fmt.Errorf("encode developer UX audit: %w", err)
	}
	return digest(encoded), nil
}

func digest(value []byte) string {
	encoded := sha256.Sum256(value)
	return "sha256:" + hex.EncodeToString(encoded[:])
}

func validDigest(value string) bool {
	if !strings.HasPrefix(value, "sha256:") || len(value) != len("sha256:")+sha256.Size*2 {
		return false
	}
	_, err := hex.DecodeString(strings.TrimPrefix(value, "sha256:"))
	return err == nil
}
