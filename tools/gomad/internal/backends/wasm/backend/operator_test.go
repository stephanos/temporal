package backend

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"go.temporal.io/server/tools/gomad/internal/backends"
	"go.temporal.io/server/tools/gomad/internal/backends/target"
	simulation "go.temporal.io/server/tools/gomad/internal/backends/wasm/cluster"
	"go.temporal.io/server/tools/gomad/internal/backends/wasm/toolchain"
	"go.temporal.io/server/tools/gomad/internal/backends/wasm/wasi"
	"go.temporal.io/server/tools/gomad/internal/evidence/record"
	"go.temporal.io/server/tools/gomad/internal/model/choice"
	model "go.temporal.io/server/tools/gomad/internal/model/cluster"
)

func TestScenarioPreparationSealsInputsAndRejectsBeforeCompiler(t *testing.T) {
	opts := portableProviderOptions(wasi.CooperativeProfile)
	opts.Config.Clock.ReadStepNanos = 0
	started := filepath.Join(t.TempDir(), "started")
	protocol := filepath.Join(t.TempDir(), "protocol")
	captured := filepath.Join(t.TempDir(), "execute")
	helper := fmt.Appendf(nil, "#!/bin/sh\nIFS= read -r frame\nprintf '%%s\\n' \"$frame\" > %q\n: > %q\nif [ -f %q ] && [ ! -f %q ]; then\n: > %q\nprintf 'invalid-frame\\n'\nfi\nexec /bin/sleep 30\n", captured, started, protocol, protocol+"-observed", protocol+"-observed")
	opts.HelperPath = filepath.Join(t.TempDir(), "helper")
	if err := os.WriteFile(opts.HelperPath, helper, 0700); err != nil {
		t.Fatal(err)
	}
	opts.HelperSHA256 = string(record.HashBytes(helper))
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
	pr.InitialMemoryPages = 1
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
	if err := p.ValidatePrepared(target.Spec{Backend: Name, Kind: pr.Kind, Source: pr.Source, Scenario: normalized}, prepared, []string{"GOMAD_IO_PROFILE=wasm"}); err != nil {
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
	for _, cancelled := range []bool{false, true} {
		t.Run(fmt.Sprintf("host-stop-cancelled=%t", cancelled), func(t *testing.T) {
			if err := os.Remove(started); err != nil && !errors.Is(err, os.ErrNotExist) {
				t.Fatal(err)
			}
			ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
			defer cancel()
			timeout := 250 * time.Millisecond
			if cancelled {
				timeout = 5 * time.Second
				done := make(chan struct{})
				go func() {
					defer close(done)
					poll := time.NewTicker(time.Millisecond)
					defer poll.Stop()
					for {
						select {
						case <-poll.C:
							if _, err := os.Stat(started); err == nil {
								cancel()
								return
							}
						case <-ctx.Done():
							return
						}
					}
				}()
				defer func() { cancel(); <-done }()
			}
			result, err := p.Execute(ctx, backends.ExecutionRequest{Target: prepared, OutputLimit: 1024, ExecutionTimeout: timeout})
			if err != nil || !result.Captured || !result.GroupGone || result.Cancelled != cancelled || result.WatchdogTimeout == cancelled {
				t.Fatalf("host termination result=%+v err=%v", result, err)
			}
			for _, output := range []backends.Output{result.Stdout, result.Stderr} {
				if output.FullSHA256 != sha256.Sum256(nil) || output.RetainedSHA256 != sha256.Sum256(nil) || output.TotalBytes != 0 || output.RetainedBytes != 0 {
					t.Fatalf("host termination cannot publish empty stream: %+v", output)
				}
			}
		})
	}
	for _, tc := range []struct {
		name   string
		choice *backends.ChoiceCapability
		limit  uint64
	}{
		{name: "default whole-program choice bound", limit: choice.MaximumTraceBytes},
		{name: "explicit choice bound", choice: &backends.ChoiceCapability{Mode: choice.ModeRecord, Limit: 4096}, limit: 4096},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := os.Remove(captured); err != nil {
				t.Fatal(err)
			}
			result, err := p.Execute(t.Context(), backends.ExecutionRequest{Target: prepared, OutputLimit: 1024, ExecutionTimeout: 250 * time.Millisecond, Choice: tc.choice})
			if err != nil || !result.Captured || !result.GroupGone || !result.WatchdogTimeout {
				t.Fatalf("scripted helper did not capture budget: result=%+v err=%v", result, err)
			}
			data, err := os.ReadFile(captured)
			if err != nil {
				t.Fatal(err)
			}
			type boot struct {
				LocalModel struct{ Limit uint64 } `json:"local_model"`
			}
			var frame struct {
				Type             string
				Instances, Boots []boot
			}
			if err := json.Unmarshal(data, &frame); err != nil {
				t.Fatal(err)
			}
			if frame.Type != "cluster_execute" || len(frame.Instances)+len(frame.Boots) == 0 {
				t.Fatalf("missing execution boot configuration: %s", data)
			}
			for _, b := range append(frame.Instances, frame.Boots...) {
				if b.LocalModel.Limit != tc.limit {
					t.Fatalf("transmitted choice bound=%d, want %d", b.LocalModel.Limit, tc.limit)
				}
			}
		})
	}
	t.Run("protocol failure precedes cleanup watchdog", func(t *testing.T) {
		if err := os.WriteFile(protocol, nil, 0600); err != nil {
			t.Fatal(err)
		}
		result, err := p.Execute(t.Context(), backends.ExecutionRequest{Target: prepared, OutputLimit: 1024, ExecutionTimeout: 250 * time.Millisecond})
		var failure *backends.FailureError
		if !errors.As(err, &failure) || failure.Termination != backends.Infrastructure || !strings.Contains(failure.Message, "malformed JSON") || result.Cancelled || result.WatchdogTimeout {
			t.Fatalf("protocol failure replaced by cleanup: result=%+v err=%v", result, err)
		}
	})
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

func TestScenarioPreservesChoiceCapacityBeforeRetention(t *testing.T) {
	p, source := inspectionArtifactFixture(t)
	handle := model.NodeHandle{Node: "a", Incarnation: 1}
	s := simulation.OperatorScenario{Schema: simulation.OperatorSchema, Spec: model.Spec{Schema: model.SpecSchema, Backend: model.BackendWASM, Fidelity: model.FidelitySimulationModel, Limits: model.DefaultLimits(), Nodes: []model.NodeSpec{{ID: handle.Node, Boot: "main", Address: "10.0.0.1"}}}, Boots: []simulation.OperatorBoot{{ID: "main"}}, Actions: []simulation.OperatorAction{{Action: "start", Handle: handle}, {Action: "wait", Handle: handle}}}
	data, err := json.Marshal(s)
	if err != nil {
		t.Fatal(err)
	}
	prepared, err := p.Prepare(t.Context(), target.Spec{Backend: Name, Kind: target.KindGoRun, Source: ".", WorkingDir: source, PreparationRoot: t.TempDir(), BuildTags: []string{"gomad", "test_dep"}, Scenario: data})
	if err != nil {
		t.Fatal(err)
	}
	_, err = p.Execute(t.Context(), backends.ExecutionRequest{Target: prepared, OutputLimit: 1 << 20, ExecutionTimeout: time.Minute, Choice: &backends.ChoiceCapability{Mode: choice.ModeRecord, Limit: choice.MinimumTraceBytes}})
	var failure *backends.FailureError
	if !errors.As(err, &failure) || failure.Termination != backends.Capacity || !strings.Contains(failure.Message, "runtime-choice-bytes") {
		t.Fatalf("choice capacity replaced by retention failure: %v", err)
	}
}

func TestScenarioCannotRunAsStandaloneModule(t *testing.T) {
	p, err := New(portableProviderOptions(wasi.CooperativeProfile))
	if err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(provenance{Scenario: []byte(`{"schema":"gomad-wasm.scenario/v1"}`)})
	if err != nil {
		t.Fatal(err)
	}
	reference := record.BackendPayload{Schema: provenanceSchema, File: "backend/provenance.json", SHA256: record.HashBytes(data), Bytes: record.Uint64String(len(data))}
	prepared := target.Prepared{Backend: &record.BackendMetadata{Name: Name, ReplayMode: record.ReplayExact, Provenance: reference}, BackendPayloads: []target.BackendPayload{{Reference: reference, Data: data}}}
	_, err = p.Run(t.Context(), backends.Request{Target: prepared})
	if err == nil || !strings.Contains(err.Error(), "require Execute") {
		t.Fatalf("typed plan reached standalone execution admission: %v", err)
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
