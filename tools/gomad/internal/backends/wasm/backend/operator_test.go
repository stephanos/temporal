package backend

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomad/internal/backends"
	"go.temporal.io/server/tools/gomad/internal/backends/target"
	simulation "go.temporal.io/server/tools/gomad/internal/backends/wasm/cluster"
	"go.temporal.io/server/tools/gomad/internal/backends/wasm/toolchain"
	"go.temporal.io/server/tools/gomad/internal/backends/wasm/wasi"
	"go.temporal.io/server/tools/gomad/internal/evidence/record"
	model "go.temporal.io/server/tools/gomad/internal/model/cluster"
)

func TestScenarioPreparationSealsInputsAndRejectsBeforeCompiler(t *testing.T) {
	opts := portableProviderOptions(wasi.CooperativeProfile)
	opts.Config.Clock.ReadStepNanos = 0
	p, err := New(opts)
	if err != nil {
		t.Fatal(err)
	}
	s := simulation.OperatorScenario{Schema: simulation.OperatorSchema, Spec: model.Spec{Schema: model.SpecSchema, Backend: model.BackendWASM, Fidelity: model.FidelitySimulationModel, Limits: model.DefaultLimits(), Nodes: []model.NodeSpec{{ID: "a", Boot: "main", Address: "10.0.0.1"}}}, Boots: []simulation.OperatorBoot{{ID: "main"}}, Actions: []simulation.OperatorAction{{Action: "start", Handle: model.NodeHandle{Node: "a", Incarnation: 1}}, {Action: "wait", Handle: model.NodeHandle{Node: "a", Incarnation: 1}}}}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	normalized, err := p.normalizeScenario(data)
	if err != nil {
		t.Fatal(err)
	}
	canonical, err := json.Marshal(s)
	if err != nil {
		t.Fatal(err)
	}
	if string(normalized) != string(canonical) {
		t.Fatal("scenario was not normalized")
	}
	module := []byte("module")
	pr := provenance{BuildCachePolicy: "private-build-key/v1", Schema: provenanceSchema, Profile: wasi.CooperativeProfile, ReplayMode: record.ReplayExact, CompilerSHA256: opts.CompilerSHA256, GoVersion: opts.GoVersion, HelperSHA256: opts.HelperSHA256, ModelSHA256: opts.ModelSHA256, Engine: opts.Engine, MemoryBytes: opts.MemoryBytes, Fuel: opts.Fuel, Config: opts.Config, ModuleSHA256: string(record.HashBytes(module)), ModuleBytes: uint64(len(module)), Kind: target.KindGoRun, Source: "fixture", RuntimeSHA256: string(record.HashBytes([]byte("runtime"))), RuntimeImplementationSHA256: toolchain.ImplementationSHA256(), Scenario: normalized, ScenarioImplementationSHA256: simulation.ImplementationSHA256()}
	pr.BuildKey, err = buildKey(pr)
	if err != nil {
		t.Fatal(err)
	}
	provenanceData, err := json.Marshal(pr)
	if err != nil {
		t.Fatal(err)
	}
	prepared := preparedFrom(pr, provenanceData)
	prepared.Path = filepath.Join(t.TempDir(), "module.wasm")
	if err := os.WriteFile(prepared.Path, module, 0500); err != nil {
		t.Fatal(err)
	}
	if err := p.ValidatePrepared(target.Spec{Backend: Name, Kind: pr.Kind, Source: pr.Source, Scenario: normalized}, prepared, []string{"GOMAD3_IO_PROFILE=wasm"}); err != nil {
		t.Fatalf("typed preparation incorrectly needs standalone choices: %v", err)
	}
	first, err := buildKey(provenance{Scenario: normalized})
	if err != nil {
		t.Fatal(err)
	}
	s.Boots[0].Args = []string{"different"}
	changed, _ := json.Marshal(s)
	second, err := buildKey(provenance{Scenario: changed})
	if err != nil || first == second {
		t.Fatal("scenario did not bind build identity", err)
	}
	_, err = p.Prepare(t.Context(), target.Spec{Backend: Name, Scenario: []byte(`{"schema":"unsupported"}`)})
	if err == nil || !strings.Contains(err.Error(), "scenario") {
		t.Fatalf("preflight reached missing compiler: %v", err)
	}
	stock, err := New(portableProviderOptions(wasi.StockProfile))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := stock.normalizeScenario(canonical); err == nil {
		t.Fatal("stock scenario admitted")
	}
}

func TestScenarioFailureClassification(t *testing.T) {
	for _, tc := range []struct {
		err  error
		want backends.Termination
	}{
		{&model.CapacityError{Resource: "actions", Required: 2, Maximum: 1}, backends.Capacity},
		{&wasi.BoundaryError{Kind: "capacity", Dimension: "calls"}, backends.Capacity},
		{&wasi.BoundaryError{Kind: "invalid", Dimension: "evidence"}, backends.Infrastructure},
		{&backends.FailureError{Termination: backends.Unsupported, Message: "unsupported"}, backends.Unsupported},
	} {
		failure := scenarioFailure(fmt.Errorf("scenario: %w", tc.err))
		var observed *backends.FailureError
		if !errors.As(failure, &observed) || observed.Termination != tc.want {
			t.Fatalf("%v => %v want %s", tc.err, failure, tc.want)
		}
	}
}

func TestScenarioReportOnlyBindsEveryIncarnation(t *testing.T) {
	s := simulation.OperatorScenario{Spec: model.Spec{Limits: model.DefaultLimits()}, Actions: []simulation.OperatorAction{{Action: "start", Handle: model.NodeHandle{Node: "a", Incarnation: 1}}, {Action: "crash", Handle: model.NodeHandle{Node: "a", Incarnation: 1}}, {Action: "restart", Handle: model.NodeHandle{Node: "a", Incarnation: 2}}}}
	plan := scenarioSoftAssertions(s, true)
	if plan == nil || plan.Schema != simulation.SoftAssertionObservationSchema || len(plan.SoftAssertions) != 2 || !plan.SoftAssertions[0].ReportOnly || !plan.SoftAssertions[1].ReportOnly || plan.SoftAssertions[1].Handle.Incarnation != 2 {
		t.Fatalf("report-only plan %+v", plan)
	}
	if scenarioSoftAssertions(s, false) != nil {
		t.Fatal("default policy needs no explicit plan")
	}
}
