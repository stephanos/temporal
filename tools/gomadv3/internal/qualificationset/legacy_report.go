package qualificationset

import (
	"encoding/json"
	"errors"
	"fmt"

	"go.temporal.io/server/tools/gomadv3/internal/capabilityanalysis"
	"go.temporal.io/server/tools/gomadv3/internal/qualify"
	"go.temporal.io/server/tools/gomadv3/internal/record"
)

type legacySetReport struct {
	Schema               string              `json:"schema"`
	Name                 string              `json:"name"`
	ExpectationsMet      bool                `json:"expectations_met"`
	Selected             uint64              `json:"selected"`
	Completed            uint64              `json:"completed"`
	Supported            uint64              `json:"supported"`
	Unsupported          uint64              `json:"unsupported"`
	Failed               uint64              `json:"failed"`
	InfrastructureErrors uint64              `json:"infrastructure_errors"`
	ManifestSHA256       record.SHA256       `json:"manifest_sha256"`
	Manifest             legacyManifest      `json:"manifest"`
	Suites               []legacySuiteReport `json:"suites"`
}

type legacySuiteReport struct {
	Name           string          `json:"name"`
	Expected       Expectation     `json:"expected"`
	ExpectationMet bool            `json:"expectation_met"`
	Classification string          `json:"classification,omitempty"`
	ExitCode       int             `json:"exit_code"`
	Command        []string        `json:"command"`
	StdoutSHA256   record.SHA256   `json:"stdout_sha256,omitempty"`
	StderrSHA256   record.SHA256   `json:"stderr_sha256,omitempty"`
	ReportPath     string          `json:"report_path,omitempty"`
	ReportSHA256   record.SHA256   `json:"report_sha256,omitempty"`
	Report         json.RawMessage `json:"report,omitempty"`
	Error          string          `json:"error,omitempty"`
}

func decodeLegacySetReport(data []byte) (SetReport, error) {
	var legacy legacySetReport
	if err := record.DecodeCanonicalJSON(data, &legacy); err != nil {
		return SetReport{}, fmt.Errorf("decode legacy qualification set report: %w", err)
	}
	if err := validateLegacySetReport(legacy); err != nil {
		return SetReport{}, err
	}
	report := SetReport{
		Schema: ReportSchema, Name: legacy.Name, ManifestSHA256: legacy.ManifestSHA256,
		Dimensions: EvidenceDimensions{}, ExpectationsMet: legacy.ExpectationsMet,
		Selected: legacy.Selected, Completed: legacy.Completed, Supported: legacy.Supported,
		Unsupported: legacy.Unsupported, Failed: legacy.Failed, InfrastructureErrors: legacy.InfrastructureErrors,
		Suites: make([]SuiteReport, len(legacy.Suites)),
	}
	for index, suite := range legacy.Suites {
		normalized := SuiteReport{
			ID: suite.Name, Name: suite.Name, Tier: 1, Expected: suite.Expected,
			ExpectationMet: suite.ExpectationMet, Classification: suite.Classification,
			AnalysisError: "dimension_unavailable", Seeds: []SeedReport{},
			Blockers: []capabilityanalysis.Blocker{}, Choice: emptyChoiceCoverage(),
		}
		if suite.ReportPath != "" {
			qualification, err := qualify.Decode(suite.Report)
			if err != nil {
				return SetReport{}, fmt.Errorf("decode legacy workload %s qualification: %w", suite.Name, err)
			}
			seed := SeedReport{
				Seed: qualification.Seed, Classification: qualify.Classify(qualification), EvidenceSHA256: qualification.EvidenceDigest,
				ReplayMatch: true, Choice: emptyChoiceCoverage(),
			}
			for _, run := range qualification.Runs {
				if run.Replay != nil {
					seed.Replayed = true
					if !run.Replay.Match {
						seed.ReplayMatch = false
						seed.ReplayDivergence = run.Replay.Divergence
					}
				}
			}
			if !seed.Replayed {
				seed.ReplayMatch = false
			}
			normalized.Seeds = append(normalized.Seeds, seed)
		}
		report.Suites[index] = normalized
	}
	return report, nil
}

func validateLegacySetReport(report legacySetReport) error {
	if report.Schema != LegacyReportSchema || report.Name == "" || report.Selected != uint64(len(report.Manifest.Suites)) || report.Completed > report.Selected || len(report.Suites) > int(report.Selected) {
		return errors.New("legacy qualification set report identity or counts are invalid")
	}
	manifest := normalizeLegacyManifest(report.Manifest)
	if err := validateManifest(manifest); err != nil {
		return err
	}
	manifestBytes, err := record.CanonicalJSON(report.Manifest)
	if err != nil || record.HashBytes(manifestBytes) != report.ManifestSHA256 {
		return errors.Join(errors.New("legacy qualification set manifest digest is invalid"), err)
	}
	counts, err := summarizeLegacySuites(report)
	if err != nil {
		return err
	}
	if counts.completed != report.Completed || counts.supported != report.Supported || counts.unsupported != report.Unsupported || counts.failed != report.Failed || counts.infrastructure != report.InfrastructureErrors || report.ExpectationsMet != counts.expectationsMet {
		return errors.New("legacy qualification set result is inconsistent")
	}
	return nil
}

type legacyReportCounts struct {
	completed       uint64
	supported       uint64
	unsupported     uint64
	failed          uint64
	infrastructure  uint64
	expectationsMet bool
}

func summarizeLegacySuites(report legacySetReport) (legacyReportCounts, error) {
	counts := legacyReportCounts{expectationsMet: len(report.Suites) == int(report.Selected)}
	for index, suite := range report.Suites {
		manifestSuite := report.Manifest.Suites[index]
		if suite.Name != manifestSuite.Name || suite.Expected != manifestSuite.Expectation || len(suite.Command) == 0 || suite.Command[0] == "" {
			return legacyReportCounts{}, fmt.Errorf("legacy qualification set workload %d identity is invalid", index)
		}
		if suite.ReportPath != "" {
			counts.completed++
			qualification, err := qualify.Decode(suite.Report)
			if err != nil || suite.Classification != qualify.Classify(qualification) {
				return legacyReportCounts{}, errors.Join(fmt.Errorf("legacy qualification set workload %s evidence is invalid", suite.Name), err)
			}
			if record.HashBytes(append(append([]byte(nil), suite.Report...), '\n')) != suite.ReportSHA256 {
				return legacyReportCounts{}, fmt.Errorf("legacy qualification set workload %s report digest is invalid", suite.Name)
			}
			switch classificationBucket(suite.Classification) {
			case qualificationSupported:
				counts.supported++
			case qualificationUnsupported:
				counts.unsupported++
			case qualificationFailed:
				counts.failed++
			default:
				counts.infrastructure++
			}
		}
		counts.expectationsMet = counts.expectationsMet && suite.ExpectationMet
	}
	counts.infrastructure += report.Selected - counts.completed
	counts.expectationsMet = counts.expectationsMet && counts.completed == report.Selected
	return counts, nil
}
