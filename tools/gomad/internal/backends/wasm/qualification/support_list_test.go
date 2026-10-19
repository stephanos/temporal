package qualification

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomad/internal/backends/wasm/backend"
	gomadqualification "go.temporal.io/server/tools/gomad/internal/evidence/qualification"
	"go.temporal.io/server/tools/gomad/internal/evidence/record"
	hostexec "go.temporal.io/server/tools/gomad/internal/host/exec"
)

type supportFixture struct {
	key, cell   string
	pid         uint64
	helper      string
	module      string
	run         string
	provisional bool
	manifest    func(*Manifest)
	extraArgs   []string
}

// supportTestCell synthesizes a Result that passes ValidateResult with costs
// and budget recomputed from its own raw samples, plus its receipt.
func supportTestCell(t *testing.T, fixture supportFixture) SupportCellInput {
	t.Helper()
	workload, id, _ := strings.Cut(fixture.key, "/")
	selection := supportCells[fixture.key]
	if fixture.run == "" {
		fixture.run = selection.Run
		if fixture.cell == "canary" {
			fixture.run = supportCanaryRun
		}
	}
	if selection.Package == "" {
		selection.Package = "./unknown"
	}
	digest := func(value string) record.SHA256 { return record.HashBytes([]byte(value)) }
	if fixture.helper == "" {
		fixture.helper = "helper"
	}
	if fixture.module == "" {
		fixture.module = "module-" + fixture.key
	}
	sourceManifest := []byte(strings.TrimPrefix(string(digest("file")), "sha256:") + "  /workspace/tools/gomad/go.mod\n")
	manifest := Manifest{Provisional: fixture.provisional, IntegratedSourceManifestPath: "/evidence/source.sha256", Schema: ManifestSchema, ID: id, Workload: workload, Profile: "cooperative", Cell: fixture.cell, IntegratedSourceSHA256: record.HashBytes(sourceManifest), CompilerSHA256: digest("compiler"), RuntimeSHA256: digest("runtime"), HelperSHA256: digest(fixture.helper), EngineSHA256: digest("engine"), ModelSHA256: digest("model"), HostConfigSHA256: digest("config-" + fixture.key), WorkloadSourceSHA256: digest("source-" + fixture.key), HostOS: "linux", HostArch: "arm64", Seed: 7, Repeat: 2, Fuel: 1 << 50, MemoryBytes: 4 << 30}
	if fixture.manifest != nil {
		fixture.manifest(&manifest)
		if manifest.IntegratedSourceManifestPath != "/evidence/source.sha256" {
			sourceManifest = []byte(strings.TrimPrefix(string(digest("other-file")), "sha256:") + "  /workspace/tools/gomad/go.mod\n")
			manifest.IntegratedSourceSHA256 = record.HashBytes(sourceManifest)
		}
	}
	domain := "success"
	if fixture.cell == "canary" {
		domain = "target_failure"
	}
	evidence := workloadEvidence(7)
	evidence.Target = record.Target{Kind: "go-test", Source: selection.Package, SHA256: digest(fixture.module), Size: 306428662, Argv: []string{"gomad-target", "-test.v", "-test.count=1", "-test.timeout=72h", "-test.run=" + fixture.run, "-persistenceType=sql", "-persistenceDriver=sqlite"}, BuildTags: backend.TemporalGuestBuildTags()}
	evidence.Target.Argv = append(evidence.Target.Argv, fixture.extraArgs...)
	evidence.Outcome.Domain = domain
	executions := []gomadqualification.QualificationExecution{}
	for iteration := range 2 {
		artifact := fmt.Sprintf("/evidence/%s/%s/run-%d", fixture.key, fixture.cell, iteration)
		executions = append(executions, gomadqualification.QualificationExecution{CampaignPath: artifact, ArtifactPath: artifact, WallElapsedNanos: 2_000_000_000, Evidence: evidence, Replay: &gomadqualification.QualificationReplay{ArtifactPath: artifact, Attempted: true, Match: true}})
	}
	report, err := gomadqualification.BuildQualificationReport(gomadqualification.QualificationInput{Command: []string{"wasm-qualify", workload, fixture.cell}, Executions: executions})
	if err != nil {
		t.Fatal(err)
	}
	reportPath, err := gomadqualification.WriteQualificationReport(t.TempDir(), report)
	if err != nil {
		t.Fatal(err)
	}
	reportBytes, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatal(err)
	}
	if report, err = gomadqualification.DecodeQualificationReport(bytes.TrimSuffix(reportBytes, []byte{'\n'})); err != nil {
		t.Fatal(err)
	}
	samples := []Sample{}
	for iteration := uint64(1); iteration <= 2; iteration++ {
		for _, phase := range []string{"execute", "replay"} {
			sample := Sample{Phase: phase, Iteration: iteration, Incarnation: "onebox", HelperWallNanos: 2_000_000_000, ExecutionWallNanos: 2_000_000_000, CPUNanos: 1_000_000_000, GuestPeakMemoryBytes: 1 << 30, HostPeakRSSBytes: 2 << 30, CompileCacheHit: true, VirtualTimeNanos: 10_000_000_000}
			if phase == "execute" {
				sample.OutcomeSHA256 = digest(domain)
				if fixture.cell == "canary" {
					sample.ReplayableBugSHA256 = digest("invalid-regexp")
				}
			}
			samples = append(samples, sample)
		}
	}
	result := Result{
		PreparationProvenanceSHA256: digest("provenance"), OperatorProcessID: fixture.pid,
		OperatorPreparationBefore: &hostexec.ResourceUsage{UserNanos: 1, SystemNanos: 1, PeakRSSBytes: 1 << 20}, OperatorPreparationAfter: &hostexec.ResourceUsage{UserNanos: 2, SystemNanos: 2, PeakRSSBytes: 64 << 20},
		OperatorExecutionBefore: &hostexec.ResourceUsage{UserNanos: 3, SystemNanos: 3, PeakRSSBytes: 64 << 20}, OperatorExecutionAfter: &hostexec.ResourceUsage{UserNanos: 9, SystemNanos: 9, PeakRSSBytes: 128 << 20},
		ResidentGuests: 1, Manifest: manifest, Qualification: report, QualificationPath: reportPath, QualificationBytes: reportBytes, Samples: samples,
		CompilerCommands:            []backend.CompilerCommandMetrics{{Arguments: []string{"test", "-c"}, WallElapsedNanos: 60_000_000_000, ResourceUsage: &hostexec.ResourceUsage{UserNanos: 50_000_000_000, SystemNanos: 5_000_000_000, PeakRSSBytes: 3 << 30}}},
		InitialPreparationWallNanos: 60_000_000_000, InitialPreparationCompiled: true, InitialCompilerCommandCount: 1, IntegratedSourceManifest: sourceManifest,
	}
	result.RunOrder = sampleOrder(result.Samples)
	budget, err := EvaluateBudget(result.Samples, result.CompilerCommands, result.InitialPreparationWallNanos, result.ResidentGuests, operatorPeak(result))
	if err != nil || !budget.WithinBudget {
		t.Fatalf("fixture budget: %+v %v", budget, err)
	}
	result.Budget = &budget
	cost, err := measuredCost(result)
	if err != nil {
		t.Fatal(err)
	}
	result.Cost = &cost
	if err := ValidateResult(result); err != nil {
		t.Fatal(err)
	}
	encoded, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	hash := record.HashBytes(append(encoded, '\n'))
	path := "/evidence/wasm-qualifications/v1/result-" + strings.TrimPrefix(string(hash), "sha256:") + ".json"
	receipt := PublicationReceipt{ResultSHA256: hash, OperatorProcessID: fixture.pid, OperatorAfter: &hostexec.ResourceUsage{UserNanos: 10, SystemNanos: 10, PeakRSSBytes: 128 << 20}}
	return SupportCellInput{ResultPath: path, Result: result, Receipt: receipt}
}

func supportTestGates(keys ...string) []SupportGate {
	gates := []SupportGate{}
	for _, key := range keys {
		for _, name := range supportGateOrder {
			gates = append(gates, SupportGate{Cell: key, Name: name, Command: "run-onebox.sh " + name, Exit: 0, Log: "/evidence/" + name + ".log"})
		}
	}
	return gates
}

const (
	smokeCell    = "onebox/unchanged-workflow-type-smoke"
	frontendCell = "onebox/frontend-system-info"
)

func supportTestCells(t *testing.T) []SupportCellInput {
	t.Helper()
	return []SupportCellInput{
		supportTestCell(t, supportFixture{key: smokeCell, cell: "success", pid: 11}),
		supportTestCell(t, supportFixture{key: smokeCell, cell: "canary", pid: 12}),
		supportTestCell(t, supportFixture{key: frontendCell, cell: "success", pid: 13}),
		supportTestCell(t, supportFixture{key: frontendCell, cell: "canary", pid: 14}),
	}
}

func TestSupportListRendersDeterministicPerTestRows(t *testing.T) {
	cells := supportTestCells(t)
	gates := supportTestGates(smokeCell, frontendCell)
	list, err := BuildSupportList(cells, gates)
	if err != nil {
		t.Fatal(err)
	}
	if len(list.Rows) != 2 || list.Rows[0].Package != "./tests" || list.Rows[0].Function != "TestWorkflowTypeEncodingSuite/TestPlainASCII/Succeeds" || list.Rows[1].Function != "TestFrontendSystemInfo" {
		t.Fatalf("rows = %+v", list.Rows)
	}
	if row := list.Rows[1]; row.Success.MatchedReplays != 2 || row.Canary.ReplayableBugs != 1 || len(row.Gates) != 4 || row.ModuleSHA256 != record.HashBytes([]byte("module-"+frontendCell)) {
		t.Fatalf("row evidence = %+v", row)
	}
	encoded, err := MarshalSupportList(list)
	if err != nil {
		t.Fatal(err)
	}
	rendered, err := RenderSupportListMarkdown(list)
	if err != nil {
		t.Fatal(err)
	}
	slices.Reverse(cells)
	slices.Reverse(gates)
	again, err := BuildSupportList(cells, gates)
	if err != nil {
		t.Fatal(err)
	}
	encodedAgain, _ := MarshalSupportList(again)
	renderedAgain, _ := RenderSupportListMarkdown(again)
	if !bytes.Equal(encoded, encodedAgain) || !bytes.Equal(rendered, renderedAgain) {
		t.Fatal("support list rendering depends on input order")
	}
	for _, want := range []string{SupportListStatement, "`./tests/gomadfunctional` | `TestFrontendSystemInfo`", "| storage-transport-inspection | 0 |", string(list.Candidate.HelperSHA256)} {
		if !strings.Contains(string(rendered), want) {
			t.Fatalf("markdown lacks %q:\n%s", want, rendered)
		}
	}
	base := filepath.Join(t.TempDir(), "fn222-support-list")
	jsonPath, mdPath, err := WriteSupportList(base, list)
	if err != nil {
		t.Fatal(err)
	}
	if written, _ := os.ReadFile(jsonPath); !bytes.Equal(written, encoded) {
		t.Fatal("written JSON differs from canonical bytes")
	}
	if written, _ := os.ReadFile(mdPath); !bytes.Equal(written, rendered) {
		t.Fatal("written Markdown differs from canonical rendering")
	}
	if _, _, err := WriteSupportList(base, list); err == nil {
		t.Fatal("historical support list replaced")
	}
	opened, err := OpenSupportList(jsonPath)
	if err != nil {
		t.Fatal(err)
	}
	if reopened, _ := MarshalSupportList(opened); !bytes.Equal(reopened, encoded) {
		t.Fatal("retained support list did not round-trip")
	}
}

func TestSupportListRefusesUnqualifiedEvidence(t *testing.T) {
	valid := supportTestCells(t)[:2]
	gates := supportTestGates(smokeCell)
	if _, err := BuildSupportList(valid, gates); err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name  string
		cells func() []SupportCellInput
		gates []SupportGate
		want  string
	}{
		{"empty", func() []SupportCellInput { return nil }, nil, "empty"},
		{"provisional", func() []SupportCellInput {
			return []SupportCellInput{supportTestCell(t, supportFixture{key: smokeCell, cell: "success", pid: 11, provisional: true}), valid[1]}
		}, gates, "provisional"},
		{"missing-canary", func() []SupportCellInput { return valid[:1] }, gates, "canary"},
		{"repeated-success", func() []SupportCellInput {
			return []SupportCellInput{valid[0], supportTestCell(t, supportFixture{key: smokeCell, cell: "success", pid: 15}), valid[1]}
		}, gates, "repeated"},
		{"mismatched-helper", func() []SupportCellInput {
			return []SupportCellInput{valid[0], supportTestCell(t, supportFixture{key: smokeCell, cell: "canary", pid: 12, helper: "other-helper"})}
		}, gates, "one candidate"},
		{"mismatched-module", func() []SupportCellInput {
			return []SupportCellInput{valid[0], supportTestCell(t, supportFixture{key: smokeCell, cell: "canary", pid: 12, module: "other-module"})}
		}, gates, "different modules"},
		{"unknown-workload", func() []SupportCellInput {
			return append([]SupportCellInput{supportTestCell(t, supportFixture{key: "onebox/sweep-pass", cell: "success", pid: 21})}, valid...)
		}, gates, "unknown support cell"},
		{"unknown-workload-family", func() []SupportCellInput {
			return append([]SupportCellInput{supportTestCell(t, supportFixture{key: "fixture/ready-select-timer", cell: "success", pid: 21})}, valid...)
		}, gates, "unknown support cell"},
		{"wrong-canary-selection", func() []SupportCellInput {
			return []SupportCellInput{valid[0], supportTestCell(t, supportFixture{key: smokeCell, cell: "canary", pid: 12, run: "^TestFrontendSystemInfo$"})}
		}, gates, "did not run"},
		{"repeated-selection", func() []SupportCellInput {
			return []SupportCellInput{supportTestCell(t, supportFixture{key: smokeCell, cell: "success", pid: 11, extraArgs: []string{"-test.run=^TestActivityTestSuite$"}}), valid[1]}
		}, gates, "did not run"},
		{"mismatched-host-config", func() []SupportCellInput {
			return []SupportCellInput{valid[0], supportTestCell(t, supportFixture{key: smokeCell, cell: "canary", pid: 12, manifest: func(m *Manifest) { m.HostConfigSHA256 = record.HashBytes([]byte("other-config")) }})}
		}, gates, "host config"},
		{"wrong-selection", func() []SupportCellInput {
			return []SupportCellInput{supportTestCell(t, supportFixture{key: smokeCell, cell: "success", pid: 11, run: "^TestActivityTestSuite$"}), valid[1]}
		}, gates, "did not run"},
		{"reused-process", func() []SupportCellInput {
			return []SupportCellInput{valid[0], supportTestCell(t, supportFixture{key: smokeCell, cell: "canary", pid: 11})}
		}, gates, "distinct"},
		{"tampered-result", func() []SupportCellInput {
			tampered := slices.Clone(valid)
			tampered[0].Result.QualificationPath = "/elsewhere/qualification.json"
			return tampered
		}, gates, "content binding"},
		{"missing-gate", func() []SupportCellInput { return valid }, gates[1:], "no recorded"},
		{"failed-gate", func() []SupportCellInput { return valid }, append([]SupportGate{{Cell: smokeCell, Name: SupportGatePreparation, Command: "prepare", Exit: 1, Log: "/evidence/prepare.log"}}, gates[1:]...), "passing"},
		{"unknown-gate", func() []SupportCellInput { return valid }, append(slices.Clone(gates), SupportGate{Cell: smokeCell, Name: "sweep-pass", Command: "sweep", Log: "/evidence/sweep.log"}), "unknown support gate"},
		{"gate-for-unlisted-test", func() []SupportCellInput { return valid }, append(slices.Clone(gates), supportTestGates(frontendCell)...), "no qualified support cell"},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := BuildSupportList(test.cells(), test.gates)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("unqualified evidence: error = %v, want %q", err, test.want)
			}
		})
	}
	for name, mutate := range map[string]func(*Manifest){
		"integrated-source": func(m *Manifest) {
			m.IntegratedSourceManifestPath = "/other/source.sha256"
		},
		"compiler":  func(m *Manifest) { m.CompilerSHA256 = record.HashBytes([]byte("other")) },
		"runtime":   func(m *Manifest) { m.RuntimeSHA256 = record.HashBytes([]byte("other")) },
		"engine":    func(m *Manifest) { m.EngineSHA256 = record.HashBytes([]byte("other")) },
		"model":     func(m *Manifest) { m.ModelSHA256 = record.HashBytes([]byte("other")) },
		"host-arch": func(m *Manifest) { m.HostArch = "amd64" },
	} {
		t.Run("mismatched-"+name, func(t *testing.T) {
			_, err := BuildSupportList([]SupportCellInput{valid[0], supportTestCell(t, supportFixture{key: smokeCell, cell: "canary", pid: 12, manifest: mutate})}, gates)
			if err == nil || !strings.Contains(err.Error(), "one candidate") {
				t.Fatalf("mismatched %s identity: error = %v", name, err)
			}
		})
	}
}

func TestSupportListShapeRejectsEditedClaims(t *testing.T) {
	list, err := BuildSupportList(supportTestCells(t), supportTestGates(smokeCell, frontendCell))
	if err != nil {
		t.Fatal(err)
	}
	for name, mutate := range map[string]func(*SupportList){
		"blanket-statement": func(l *SupportList) { l.Statement = "The functional suite is supported." },
		"unlisted-test":     func(l *SupportList) { l.Rows[0].Function = "TestActivityTestSuite" },
		"failed-gate":       func(l *SupportList) { l.Rows[0].Gates[2].Exit = 1 },
		"missing-gate":      func(l *SupportList) { l.Rows[0].Gates = l.Rows[0].Gates[:3] },
		"unsorted":          func(l *SupportList) { l.Rows[0], l.Rows[1] = l.Rows[1], l.Rows[0] },
		"no-canary-bug":     func(l *SupportList) { l.Rows[1].Canary.ReplayableBugs = 0 },
		"no-rows":           func(l *SupportList) { l.Rows = nil },
	} {
		edited := list
		edited.Rows = slices.Clone(list.Rows)
		for index := range edited.Rows {
			edited.Rows[index].Gates = slices.Clone(list.Rows[index].Gates)
		}
		mutate(&edited)
		if err := ValidateSupportList(edited); err == nil {
			t.Fatalf("%s: edited support list accepted", name)
		}
	}
}

// TestActualSupportList is the support list producer of the qualification
// flow. It reads retained Results from absolute qualification roots and the
// gate exits recorded by the driver, then publishes base.json and base.md.
func TestActualSupportList(t *testing.T) {
	output, roots, gatesPath := os.Getenv("GOMAD_WASM_SUPPORT_LIST_OUTPUT"), os.Getenv("GOMAD_WASM_SUPPORT_LIST_ROOTS"), os.Getenv("GOMAD_WASM_SUPPORT_LIST_GATES")
	if output == "" && roots == "" && gatesPath == "" {
		t.Skip("set GOMAD_WASM_SUPPORT_LIST_OUTPUT, GOMAD_WASM_SUPPORT_LIST_ROOTS and GOMAD_WASM_SUPPORT_LIST_GATES to produce a support list")
	}
	if !filepath.IsAbs(gatesPath) || roots == "" {
		t.Fatal("support list producer requires absolute gate record and qualification roots")
	}
	cells := []SupportCellInput{}
	for _, root := range filepath.SplitList(roots) {
		if !filepath.IsAbs(root) {
			t.Fatal("support list qualification roots must be absolute")
		}
		directory := filepath.Join(root, "wasm-qualifications", "v1")
		entries, err := os.ReadDir(directory)
		if err != nil {
			t.Fatal(err)
		}
		names := make([]string, len(entries))
		for index, entry := range entries {
			names[index] = entry.Name()
		}
		if err := validateCorpusEntryPairs(names); err != nil {
			t.Fatal(err)
		}
		for _, name := range names {
			path, isResult, err := corpusResultEntry(directory, name)
			if err != nil {
				t.Fatal(err)
			}
			if !isResult {
				continue
			}
			result, err := OpenResult(path)
			if err != nil {
				t.Fatal(err)
			}
			receipt, err := OpenPublicationReceipt(path)
			if err != nil {
				t.Fatal(err)
			}
			cells = append(cells, SupportCellInput{ResultPath: path, Result: result, Receipt: receipt})
		}
	}
	gates, err := readSupportGates(gatesPath)
	if err != nil {
		t.Fatal(err)
	}
	list, err := BuildSupportList(cells, gates)
	if err != nil {
		t.Fatal(err)
	}
	jsonPath, mdPath, err := WriteSupportList(output, list)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("support list rows=%d json=%s markdown=%s", len(list.Rows), jsonPath, mdPath)
}

func readSupportGates(path string) ([]SupportGate, error) {
	encoded, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if len(encoded) > 1<<20 {
		return nil, errors.New("support gate record exceeds 1MiB")
	}
	var gates []SupportGate
	if err := strictDecode(encoded, &gates); err != nil {
		return nil, err
	}
	return gates, nil
}
