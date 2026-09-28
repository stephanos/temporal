package set

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"go.temporal.io/server/tools/gomad3/qualification"
	"go.temporal.io/server/tools/gomad3/record"
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
