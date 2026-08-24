package migration

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	protocolcatalog "go.temporal.io/server/tools/umpire3/protocol/catalog"
	"go.temporal.io/server/tools/umpire3/scenario"
)

const FormatVersion = "umpire3/migration-ledger/v3"

//go:embed testdata/generated/ledger.json
var defaultLedgerJSON []byte

type ExecutedContract struct {
	Variant           string           `json:"variant,omitempty"`
	ScenarioDigest    string           `json:"scenarioDigest"`
	ExperimentDigests []string         `json:"experimentDigests"`
	Explain           scenario.Explain `json:"explain"`
}

type Entry struct {
	Behavior             string                        `json:"behavior"`
	Umpire2              string                        `json:"umpire2"`
	Umpire3              string                        `json:"umpire3"`
	ModelTarget          string                        `json:"modelTarget"`
	Properties           []string                      `json:"properties"`
	Entities             []string                      `json:"entities"`
	Actions              []string                      `json:"actions"`
	Faults               []string                      `json:"faults,omitempty"`
	Relations            []string                      `json:"relations"`
	Variants             []string                      `json:"variants,omitempty"`
	Evidence             []string                      `json:"evidence"`
	Fidelity             protocolcatalog.Fidelity      `json:"fidelity"`
	EvidenceLevel        protocolcatalog.EvidenceLevel `json:"evidenceLevel"`
	Scenario             string                        `json:"scenario"`
	Profile              string                        `json:"profile"`
	RequiredCapabilities []string                      `json:"requiredCapabilities"`
	ExpectedVerdict      string                        `json:"expectedVerdict"`
	NegativeControl      string                        `json:"negativeControl"`
	ArtifactReplay       bool                          `json:"artifactReplay"`
	ExecutedContracts    []ExecutedContract            `json:"executedContracts"`
}

type Ledger struct {
	FormatVersion string  `json:"formatVersion"`
	Entries       []Entry `json:"entries"`
}

func DecodeLedger(encoded []byte) (Ledger, error) {
	var ledger Ledger
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&ledger); err != nil {
		return Ledger{}, fmt.Errorf("decode migration ledger: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return Ledger{}, errors.New("decode migration ledger: trailing JSON data")
	}
	if err := ledger.Validate(); err != nil {
		return Ledger{}, err
	}
	return ledger, nil
}

func DefaultLedger() (Ledger, []byte, error) {
	ledger, err := DecodeLedger(defaultLedgerJSON)
	if err != nil {
		return Ledger{}, nil, err
	}
	return ledger, append([]byte(nil), defaultLedgerJSON...), nil
}

func Build(testsRoot string) (Ledger, error) {
	umpire2, err := inventory(filepath.Join(testsRoot, "umpire2_test.go"),
		filepath.Join(testsRoot, "umpire2_probe_test.go"), filepath.Join(testsRoot, "umpire2_regress_test.go"))
	if err != nil {
		return Ledger{}, err
	}
	umpire3, err := inventory(filepath.Join(testsRoot, "umpire3_test.go"),
		filepath.Join(testsRoot, "umpire3_probe_test.go"), filepath.Join(testsRoot, "umpire3_regress_test.go"))
	if err != nil {
		return Ledger{}, err
	}
	byBehavior := make(map[string]location, len(umpire3))
	for _, item := range umpire3 {
		if _, duplicate := byBehavior[item.behavior]; duplicate {
			return Ledger{}, fmt.Errorf("duplicate Umpire3 migration behavior %q", item.behavior)
		}
		byBehavior[item.behavior] = item
	}
	contracts := behaviorContracts()
	entries := make([]Entry, 0, len(umpire2))
	for _, previous := range umpire2 {
		current, exists := byBehavior[previous.behavior]
		if !exists {
			return Ledger{}, fmt.Errorf("Umpire2 behavior %q has no Umpire3 test", previous.behavior)
		}
		if !slices.Contains(current.contractReferences, previous.behavior) {
			return Ledger{}, fmt.Errorf("Umpire3 behavior %q does not execute its behavior contract", previous.behavior)
		}
		contract, exists := contracts[previous.behavior]
		if !exists {
			return Ledger{}, fmt.Errorf("Umpire2 behavior %q has no executable behavior contract", previous.behavior)
		}
		executedContracts, err := compileContract(contract)
		if err != nil {
			return Ledger{}, err
		}
		entries = append(entries, Entry{
			Behavior: previous.behavior, Umpire2: previous.source, Umpire3: current.source,
			ModelTarget: string(contract.ModelTarget), Properties: []string{string(contract.Property)},
			Entities: stringsOf(contract.Entities), Actions: stringsOf(contract.Actions),
			Faults: stringsOf(contract.Faults), Relations: stringsOf(contract.Relations),
			Variants: contract.Variants, Evidence: contract.Evidence,
			Fidelity: contract.Fidelity, EvidenceLevel: contract.EvidenceLevel,
			Scenario: "behavior-contract/" + previous.behavior, Profile: "local-in-process",
			RequiredCapabilities: stringsOf(contract.RequiredCapabilities),
			ExpectedVerdict:      string(contract.ExpectedVerdict),
			NegativeControl:      contract.NegativeControl, ArtifactReplay: true,
			ExecutedContracts: executedContracts,
		})
		delete(byBehavior, previous.behavior)
		delete(contracts, previous.behavior)
	}
	if len(byBehavior) != 0 {
		var extra []string
		for behavior := range byBehavior {
			extra = append(extra, behavior)
		}
		slices.Sort(extra)
		return Ledger{}, fmt.Errorf("Umpire3 ledger has unmatched behaviors %v", extra)
	}
	if len(contracts) != 0 {
		var missing []string
		for behavior := range contracts {
			missing = append(missing, behavior)
		}
		slices.Sort(missing)
		return Ledger{}, fmt.Errorf("behavior contracts have no root test pair %v", missing)
	}
	slices.SortFunc(entries, func(left, right Entry) int { return strings.Compare(left.Behavior, right.Behavior) })
	return Ledger{FormatVersion: FormatVersion, Entries: entries}, nil
}

func compileContract(contract BehaviorContract) ([]ExecutedContract, error) {
	variants := append([]string(nil), contract.Variants...)
	if len(variants) == 0 {
		variants = []string{""}
	}
	result := make([]ExecutedContract, len(variants))
	for index, variant := range variants {
		authored, err := Scenario(contract.Behavior, variant)
		if err != nil {
			return nil, err
		}
		suite, err := scenario.Compile(context.Background(), authored, scenario.Limits{
			MaxPaths: 64, MaxActions: 128, MaxStates: 10000,
			MaxMemoryBytes: 32 << 20, MaxTime: 10 * time.Second,
		})
		if err != nil {
			return nil, fmt.Errorf("compile behavior contract %q variant %q: %w", contract.Behavior, variant, err)
		}
		result[index] = ExecutedContract{
			Variant: variant, ScenarioDigest: suite.ScenarioDigest,
			ExperimentDigests: append([]string(nil), suite.Digests...), Explain: suite.Explain,
		}
	}
	return result, nil
}

func (l Ledger) CanonicalJSON() ([]byte, error) {
	if err := l.Validate(); err != nil {
		return nil, err
	}
	encoded, err := json.Marshal(l)
	if err != nil {
		return nil, fmt.Errorf("encode migration ledger: %w", err)
	}
	return append(encoded, '\n'), nil
}

func (l Ledger) Validate() error {
	if l.FormatVersion != FormatVersion || len(l.Entries) == 0 {
		return errors.New("complete migration ledger is required")
	}
	for _, entry := range l.Entries {
		if entry.Behavior == "" || entry.Umpire2 == "" || entry.Umpire3 == "" ||
			entry.ModelTarget == "" || len(entry.ExecutedContracts) == 0 {
			return fmt.Errorf("migration entry %q is incomplete", entry.Behavior)
		}
		switch entry.Fidelity {
		case protocolcatalog.FidelityExact, protocolcatalog.FidelitySemanticEquivalent, protocolcatalog.FidelityPartial:
		case protocolcatalog.FidelityInventoryOnly:
			return fmt.Errorf("migration entry %q has inventory-only fidelity despite paired execution", entry.Behavior)
		default:
			return fmt.Errorf("migration entry %q has unknown fidelity %q", entry.Behavior, entry.Fidelity)
		}
		switch entry.EvidenceLevel {
		case protocolcatalog.EvidenceLocalIntegration, protocolcatalog.EvidenceProfileQualified:
		case protocolcatalog.EvidenceInventory, protocolcatalog.EvidenceModelProof:
			if entry.Fidelity == protocolcatalog.FidelityExact || entry.Fidelity == protocolcatalog.FidelitySemanticEquivalent {
				return fmt.Errorf("migration entry %q claims equivalence but requires live integration evidence", entry.Behavior)
			}
		default:
			return fmt.Errorf("migration entry %q has unknown evidence level %q", entry.Behavior, entry.EvidenceLevel)
		}
	}
	return nil
}

type location struct {
	behavior           string
	source             string
	contractReferences []string
}

func inventory(paths ...string) ([]location, error) {
	var result []location
	for _, path := range paths {
		fileSet := token.NewFileSet()
		file, err := parser.ParseFile(fileSet, path, nil, 0)
		if err != nil {
			return nil, fmt.Errorf("parse %s: %w", path, err)
		}
		for _, declaration := range file.Decls {
			function, ok := declaration.(*ast.FuncDecl)
			if !ok || !migrationFunction(function) {
				continue
			}
			position := fileSet.Position(function.Pos())
			result = append(result, location{
				behavior: normalizeBehavior(function.Name.Name), source: filepath.Base(path) + fmt.Sprintf(":%d", position.Line),
				contractReferences: stringLiterals(function),
			})
		}
	}
	slices.SortFunc(result, func(left, right location) int { return strings.Compare(left.behavior, right.behavior) })
	return result, nil
}

func stringLiterals(function *ast.FuncDecl) []string {
	var result []string
	ast.Inspect(function.Body, func(node ast.Node) bool {
		literal, ok := node.(*ast.BasicLit)
		if !ok || literal.Kind != token.STRING {
			return true
		}
		value, err := strconv.Unquote(literal.Value)
		if err == nil {
			result = append(result, value)
		}
		return true
	})
	slices.Sort(result)
	return slices.Compact(result)
}

func migrationFunction(function *ast.FuncDecl) bool {
	name := function.Name.Name
	if name == "TestUmpire2TestSuite" || name == "TestUmpire3TestSuite" {
		return false
	}
	if strings.HasPrefix(name, "Test") {
		return function.Recv != nil || strings.Contains(name, "SparseRegression")
	}
	return name == "runUmpire2SparseRegressionStartToCloseTimeout" ||
		name == "runUmpire2SparseRegressionCallbackAfterCallerCompletion" ||
		name == "runUmpire2SparseRegressionBidirectionalNexusActivityLinks"
}

func normalizeBehavior(name string) string {
	name = strings.TrimPrefix(name, "run")
	name = strings.TrimPrefix(name, "Test")
	name = strings.ReplaceAll(name, "Umpire2", "")
	name = strings.ReplaceAll(name, "Umpire3", "")
	return name
}

func Write(path string, ledger Ledger) error {
	encoded, err := ledger.CanonicalJSON()
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, encoded, 0o600); err != nil {
		return fmt.Errorf("write migration ledger: %w", err)
	}
	return nil
}
