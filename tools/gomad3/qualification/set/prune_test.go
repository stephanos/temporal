package set

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomad3/artifact"
	"go.temporal.io/server/tools/gomad3/deterministicio"
	"go.temporal.io/server/tools/gomad3/qualification"
	capabilityanalysis "go.temporal.io/server/tools/gomad3/qualification/analysis"
	"go.temporal.io/server/tools/gomad3/record"
	"go.temporal.io/server/tools/gomad3/runner"
)

func TestPruneQualifiedCampaignsRemovesOnlyReplayedCampaigns(t *testing.T) {
	root := t.TempDir()
	first := retainedCampaign(t, root, "campaign-first")
	second := retainedCampaign(t, root, "campaign-second")
	unrelated := retainedCampaign(t, root, "campaign-unrelated")
	reportPath := filepath.Join(root, "qualifications", "v1", "report.json")
	writeFile(t, reportPath)

	if err := pruneQualifiedCampaigns(root, qualifiedReport(first, second)); err != nil {
		t.Fatal(err)
	}
	for _, removed := range []string{first, second} {
		if _, err := os.Lstat(removed); !os.IsNotExist(err) {
			t.Fatalf("campaign %s survived pruning: %v", removed, err)
		}
	}
	for _, kept := range []string{unrelated, reportPath} {
		if _, err := os.Lstat(kept); err != nil {
			t.Fatalf("pruning removed %s: %v", kept, err)
		}
	}
}

func TestPruneQualifiedCampaignsRemovesASharedTargetOnlyWithItsLastArtifact(t *testing.T) {
	root := t.TempDir()
	pool := artifact.TargetPool(root)
	shared := filepath.Join(pool, "sha256-"+strings.Repeat("a", 64))
	abandoned := filepath.Join(pool, "sha256-"+strings.Repeat("b", 64))
	for _, entry := range []string{shared, abandoned} {
		writeFile(t, entry)
	}
	first := retainedCampaign(t, root, "campaign-first")
	second := retainedCampaign(t, root, "campaign-second")
	for _, campaign := range []string{first, second} {
		if err := os.Link(shared, filepath.Join(campaign, "successes", "sha256-x", "target")); err != nil {
			t.Fatal(err)
		}
	}

	if err := pruneQualifiedCampaigns(root, qualifiedReport(first)); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(shared); err != nil {
		t.Fatalf("pruning removed a target a retained Campaign shares: %v", err)
	}
	if _, err := os.Lstat(abandoned); !os.IsNotExist(err) {
		t.Fatalf("a target no artifact shares survived pruning: %v", err)
	}
	if err := pruneQualifiedCampaigns(root, qualifiedReport(second)); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(shared); !os.IsNotExist(err) {
		t.Fatalf("the target of the last pruned Campaign survived: %v", err)
	}
}

func TestPruneQualifiedCampaignsRefusesUnverifiedOrEscapingEvidence(t *testing.T) {
	outside := t.TempDir()
	outsideFile := filepath.Join(outside, "keep")
	writeFile(t, outsideFile)
	for name, mutate := range map[string]func(root string, report *qualification.QualificationReport){
		"not qualified": func(_ string, report *qualification.QualificationReport) { report.Qualified = false },
		"replay not attempted": func(_ string, report *qualification.QualificationReport) {
			report.Executions[0].Replay = nil
		},
		"replay diverged": func(_ string, report *qualification.QualificationReport) {
			report.Executions[0].Replay.Match = false
		},
		"campaign outside root": func(_ string, report *qualification.QualificationReport) {
			report.Executions[0].CampaignPath = outside
		},
		"campaign is the root layout": func(root string, report *qualification.QualificationReport) {
			report.Executions[0].CampaignPath = filepath.Join(root, "v1")
		},
		"artifact outside campaign": func(root string, report *qualification.QualificationReport) {
			report.Executions[0].ArtifactPath = filepath.Join(root, "v1", "campaign-other", "successes", "sha256-x")
		},
		"campaign is a symlink": func(root string, report *qualification.QualificationReport) {
			link := filepath.Join(root, "v1", "campaign-link")
			if err := os.Symlink(outside, link); err != nil {
				t.Fatal(err)
			}
			report.Executions[0].CampaignPath = link
			report.Executions[0].ArtifactPath = filepath.Join(link, "successes", "sha256-x")
		},
	} {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			campaign := retainedCampaign(t, root, "campaign-kept")
			report := qualifiedReport(campaign)
			mutate(root, &report)
			if err := pruneQualifiedCampaigns(root, report); err == nil {
				t.Fatal("pruneQualifiedCampaigns() accepted the evidence")
			}
			for _, kept := range []string{campaign, outsideFile} {
				if _, err := os.Lstat(kept); err != nil {
					t.Fatalf("refused pruning removed %s: %v", kept, err)
				}
			}
		})
	}
}

func TestRunRecordsPruningAndKeepsUnqualifiedSeedArtifacts(t *testing.T) {
	root := t.TempDir()
	artifacts := filepath.Join(root, "artifacts")
	output := filepath.Join(root, "set-report.json")
	report, err := Run(context.Background(), Spec{
		ManifestPath: writeManifestWithSeeds(t, root, []uint64{7}, "nondeterministic"),
		GomadPath:    filepath.Join(root, "gomad"), WorkingDir: root,
		ArtifactRoot: artifacts, OutputPath: output, PruneQualifiedArtifacts: true,
		Execute: failureExecutor(t, "nondeterministic"),
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if !report.QualifiedArtifactsPruned || report.Workloads[0].Seeds[0].ArtifactsPruned {
		t.Fatalf("report = %#v", report)
	}
	reports, err := os.ReadDir(filepath.Join(artifacts, "qualifications", "v1"))
	if err != nil || len(reports) != 1 {
		t.Fatalf("qualification reports = %v, %v", reports, err)
	}
	opened, err := OpenReport(output)
	if err != nil {
		t.Fatal(err)
	}
	if !opened.QualifiedArtifactsPruned {
		t.Fatalf("published report does not record pruning: %#v", opened)
	}
}

func TestRunKeepsCampaignsOfQualifiedSeedsWithoutReplay(t *testing.T) {
	root := t.TempDir()
	manifestPath := writeManifestWithSeeds(t, root, []uint64{7}, "qualified")
	contents, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	var manifest map[string]any
	if err := json.Unmarshal(contents, &manifest); err != nil {
		t.Fatal(err)
	}
	suite := manifest["suites"].([]any)[0].(map[string]any)
	suite["replay_successes"], suite["choice_bytes"], suite["success_artifact_limit"], suite["success_bytes_limit"] = false, 0, 0, 0
	if contents, err = json.Marshal(manifest); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(manifestPath, contents, 0o600); err != nil {
		t.Fatal(err)
	}
	artifacts := filepath.Join(root, "artifacts")
	report, err := Run(context.Background(), Spec{
		ManifestPath: manifestPath, GomadPath: filepath.Join(root, "gomad"), WorkingDir: root,
		ArtifactRoot: artifacts, OutputPath: filepath.Join(root, "set-report.json"), PruneQualifiedArtifacts: true,
		Execute: qualifiedWithoutReplayExecutor(t),
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if seed := report.Workloads[0].Seeds[0]; seed.Classification != "qualified" || seed.ArtifactsPruned {
		t.Fatalf("seed evidence = %#v", seed)
	}
	campaigns, err := os.ReadDir(filepath.Join(artifacts, "v1"))
	if err != nil || len(campaigns) != 2 {
		t.Fatalf("retained campaigns = %v, %v", campaigns, err)
	}
}

func TestRecordSeedCheckpointsPruningBeforeDeletingCampaigns(t *testing.T) {
	root := t.TempDir()
	deleted := retainedCampaign(t, root, "campaign-a")
	// A non-directory Campaign fails deletion after campaign-a is gone.
	failing := filepath.Join(root, "v1", "campaign-b")
	writeFile(t, failing)
	output := filepath.Join(root, "set-report.json")
	report := Report{Workloads: []WorkloadReport{{ID: "fixture-case", Seeds: []SeedReport{}}}}
	seed := SeedReport{Seed: 7, Classification: "qualified", Replayed: true, ReplayMatch: true, ArtifactsPruned: true, Choice: emptyChoiceCoverage()}

	err := recordSeed(context.Background(), Spec{ArtifactRoot: root, OutputPath: output}, &report, 0, seed, qualifiedReport(deleted, failing))
	if err == nil || !strings.Contains(err.Error(), "fixture-case seed 7") {
		t.Fatalf("recordSeed() error = %v", err)
	}
	if _, err := os.Lstat(deleted); !os.IsNotExist(err) {
		t.Fatalf("campaign-a survived: %v", err)
	}
	contents, err := os.ReadFile(output + ".partial")
	if err != nil {
		t.Fatal(err)
	}
	var persisted checkpoint
	if err := json.Unmarshal(bytes.TrimSuffix(contents, []byte{'\n'}), &persisted); err != nil {
		t.Fatal(err)
	}
	if seeds := persisted.Report.Workloads[0].Seeds; len(seeds) != 1 || !seeds[0].ArtifactsPruned {
		t.Fatalf("checkpoint seeds = %#v", seeds)
	}
}

func qualifiedWithoutReplayExecutor(t *testing.T) ExecuteFunc {
	t.Helper()
	analysis := encodedAnalysisResult(t, capabilityanalysis.ClassificationSupported)
	return func(_ context.Context, command Command) CommandResult {
		if command.Args[0] == "analyze" {
			return analysis
		}
		evidence := runner.ExecutionEvidence{
			Schema: runner.ExecutionEvidenceSchema, Seed: 7, RunnerBuild: "sha256:runner",
			Toolchain:   record.Toolchain{GoVersion: "go1.26.4", BuildKey: strings.Repeat("a", 64), TargetGOOS: "darwin", TargetGOARCH: "arm64"},
			Target:      record.Target{Kind: "go-test", Source: "./pkg", SHA256: "sha256:target", Size: 12, Argv: []string{"gomad3-target"}, BuildTags: []string{}, Adapters: []record.TargetAdapter{}, Compatibility: []record.CompatibilityPack{}},
			IOProfile:   deterministicio.Default().Identity(),
			Environment: []record.Environment{{Name: "GOMADSEED", Value: "7"}},
			Outcome:     runner.OutcomeEvidence{Domain: "success", Reason: "success", Termination: "exit"}, GroupGone: true,
			Stdout: record.Stream{FullSHA256: "sha256:stdout"}, Stderr: record.Stream{FullSHA256: "sha256:stderr"},
			IOTranscriptSHA256: "sha256:transcript", IOTranscriptRecords: 1, IOTranscriptComplete: true,
			SemanticCoverage: deterministicio.SemanticCoverage{Schema: deterministicio.SemanticCoverageSchema, Digest: "sha256:coverage", Probes: []string{"stdlib.os.openfile"}},
		}
		executions := []qualification.QualificationExecution{}
		for _, name := range []string{"campaign-1", "campaign-2"} {
			campaign := filepath.Join(command.ArtifactRoot, "v1", name)
			writeFile(t, filepath.Join(campaign, "campaign.json"))
			executions = append(executions, qualification.QualificationExecution{CampaignPath: campaign, Evidence: evidence})
		}
		report, err := qualification.BuildQualificationReport(qualification.QualificationInput{Command: append([]string{"gomad"}, command.Args...), Executions: executions})
		if err != nil {
			t.Fatal(err)
		}
		path, err := qualification.WriteQualificationReport(command.ArtifactRoot, report)
		if err != nil {
			t.Fatal(err)
		}
		var event bytes.Buffer
		if err := qualification.WriteResultEvent(&event, report, path); err != nil {
			t.Fatal(err)
		}
		return CommandResult{ExitCode: 0, Stdout: event.Bytes()}
	}
}

func TestValidateSetReportRejectsPruningWithoutReplayedQualification(t *testing.T) {
	valid := publishedPrunedReport(t)
	if err := validateSetReport(valid); err != nil {
		t.Fatalf("validateSetReport() rejected a pruned qualified seed: %v", err)
	}
	for name, mutate := range map[string]func(*Report){
		"mode not recorded": func(report *Report) { report.QualifiedArtifactsPruned = false },
		"not replayed":      func(report *Report) { report.Workloads[0].Seeds[0].Replayed = false },
		"replay diverged":   func(report *Report) { report.Workloads[0].Seeds[0].ReplayMatch = false },
	} {
		t.Run(name, func(t *testing.T) {
			report := publishedPrunedReport(t)
			mutate(&report)
			finalizeSetReportCounters(&report)
			if err := validateSetReport(report); err == nil {
				t.Fatal("validateSetReport() accepted pruning without replayed qualification")
			}
		})
	}
	unqualified := publishedPrunedReport(t)
	unqualified.Workloads[0].Classification = "nondeterministic"
	unqualified.Workloads[0].Seeds[0].Classification = "nondeterministic"
	finalizeSetReportCounters(&unqualified)
	if err := validateSetReport(unqualified); err == nil {
		t.Fatal("validateSetReport() accepted pruning of a nondeterministic seed")
	}
}

// publishedPrunedReport returns a valid one-seed report whose qualified seed
// was replayed exactly and pruned.
func publishedPrunedReport(t *testing.T) Report {
	t.Helper()
	root := t.TempDir()
	report, err := Run(context.Background(), Spec{
		ManifestPath: writeManifestWithSeeds(t, root, []uint64{7}, "nondeterministic"),
		GomadPath:    filepath.Join(root, "gomad"), WorkingDir: root,
		ArtifactRoot: filepath.Join(root, "artifacts"), OutputPath: filepath.Join(root, "set-report.json"),
		PruneQualifiedArtifacts: true, Execute: failureExecutor(t, "nondeterministic"),
	})
	if err != nil {
		t.Fatal(err)
	}
	seed := &report.Workloads[0].Seeds[0]
	seed.Classification, seed.Replayed, seed.ReplayMatch, seed.ArtifactsPruned = "qualified", true, true, true
	report.Workloads[0].Classification = "qualified"
	report.Workloads[0].Expected = WorkloadExpectation{Classification: "qualified"}
	finalizeSetReportCounters(&report)
	return report
}

func retainedCampaign(t *testing.T, root, name string) string {
	t.Helper()
	campaign := filepath.Join(root, "v1", name)
	writeFile(t, filepath.Join(campaign, "successes", "sha256-x", "manifest.json"))
	return campaign
}

func qualifiedReport(campaigns ...string) qualification.QualificationReport {
	report := qualification.QualificationReport{Qualified: true, Seed: 7, EvidenceDigest: record.HashBytes([]byte("evidence"))}
	for _, campaign := range campaigns {
		artifactPath := filepath.Join(campaign, "successes", "sha256-x")
		report.Executions = append(report.Executions, qualification.QualificationExecutionReport{
			CampaignPath: campaign, ArtifactPath: artifactPath,
			Replay: &qualification.QualificationReplay{ArtifactPath: artifactPath, Attempted: true, Match: true, ChoiceReplayStatus: qualification.ChoiceReplayExact},
		})
	}
	return report
}

func writeFile(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("evidence\n"), 0o600); err != nil {
		t.Fatal(err)
	}
}
