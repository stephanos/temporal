package qualification

import (
	"encoding/json"
	"errors"
	"fmt"

	"go.temporal.io/server/tools/gomadv3/deterministicio"
	"go.temporal.io/server/tools/gomadv3/evidence"
	"go.temporal.io/server/tools/gomadv3/target"
)

const previousAnalysisSchema = "gomadv3.capability-analysis/v1"

type previousSetReport struct {
	Schema               string                   `json:"schema"`
	Name                 string                   `json:"name"`
	Description          string                   `json:"description"`
	ManifestSHA256       evidence.SHA256          `json:"manifest_sha256"`
	Seeds                []evidence.Uint64String  `json:"seeds"`
	Module               ModuleIdentity           `json:"module"`
	Platform             PlatformIdentity         `json:"platform"`
	Toolchain            AnalysisToolchain        `json:"toolchain"`
	IOProfile            deterministicio.Contract `json:"io_profile"`
	Dimensions           EvidenceDimensions       `json:"dimensions"`
	ExpectationsMet      bool                     `json:"expectations_met"`
	Selected             uint64                   `json:"selected"`
	AnalysisCompleted    uint64                   `json:"analysis_completed"`
	Completed            uint64                   `json:"completed"`
	Supported            uint64                   `json:"supported"`
	Unsupported          uint64                   `json:"unsupported"`
	Failed               uint64                   `json:"failed"`
	InfrastructureErrors uint64                   `json:"infrastructure_errors"`
	Replayed             uint64                   `json:"replayed"`
	ReplayDiverged       uint64                   `json:"replay_diverged"`
	Cancelled            uint64                   `json:"cancelled"`
	TimedOut             uint64                   `json:"timed_out"`
	ElapsedNanos         evidence.Uint64String    `json:"elapsed_nanos"`
	ArtifactBytes        evidence.Uint64String    `json:"artifact_bytes"`
	TraceBytes           evidence.Uint64String    `json:"trace_bytes"`
	Suites               []previousWorkloadReport `json:"workloads"`
}

type previousWorkloadReport struct {
	ID             string              `json:"id"`
	Name           string              `json:"name"`
	Tier           uint64              `json:"tier"`
	Invariant      string              `json:"invariant"`
	Expected       WorkloadExpectation `json:"expected"`
	ExpectationMet bool                `json:"expectation_met"`
	Classification string              `json:"classification"`
	Analysis       json.RawMessage     `json:"analysis,omitempty"`
	AnalysisError  string              `json:"analysis_error,omitempty"`
	Seeds          []SeedReport        `json:"seeds"`
	Blockers       []AnalysisBlocker   `json:"blockers"`
	Choice         ChoiceCoverage      `json:"choice"`
}

type previousAnalysisReport AnalysisReport

func decodePreviousSetReport(data []byte, schema string) (SuiteReport, error) {
	var previous previousSetReport
	if err := evidence.DecodeCanonicalJSON(data, &previous); err != nil {
		return SuiteReport{}, fmt.Errorf("decode previous qualification set report: %w", err)
	}
	if previous.Schema != schema || schema != PreviousSuiteReportSchema && schema != PreLinkedSuiteReportSchema && schema != PreChoiceSuiteReportSchema {
		return SuiteReport{}, errors.New("previous qualification set report schema is invalid")
	}
	report := SuiteReport{
		Schema: SuiteReportSchema, Name: previous.Name, Description: previous.Description,
		ManifestSHA256: previous.ManifestSHA256, Seeds: previous.Seeds, Module: previous.Module,
		Platform: previous.Platform, Toolchain: previous.Toolchain, IOProfile: previous.IOProfile,
		Dimensions: previous.Dimensions, ExpectationsMet: previous.ExpectationsMet,
		Selected: previous.Selected, AnalysisCompleted: previous.AnalysisCompleted, Completed: previous.Completed,
		Supported: previous.Supported, Unsupported: previous.Unsupported, Failed: previous.Failed,
		InfrastructureErrors: previous.InfrastructureErrors, Replayed: previous.Replayed,
		ReplayDiverged: previous.ReplayDiverged, Cancelled: previous.Cancelled, TimedOut: previous.TimedOut,
		ElapsedNanos: previous.ElapsedNanos, ArtifactBytes: previous.ArtifactBytes, TraceBytes: previous.TraceBytes,
		Suites: make([]WorkloadReport, len(previous.Suites)),
	}
	for index, workload := range previous.Suites {
		var analysis *AnalysisReport
		if previous.Dimensions.Analysis {
			if len(workload.Analysis) == 0 && workload.AnalysisError == "" || len(workload.Analysis) != 0 && workload.AnalysisError != "" {
				return SuiteReport{}, fmt.Errorf("previous qualification set workload %s analysis state is invalid", workload.ID)
			}
			if len(workload.Analysis) != 0 {
				decoded, err := decodePreviousAnalysis(workload.Analysis, schema)
				if err != nil {
					return SuiteReport{}, fmt.Errorf("previous qualification set workload %s analysis is invalid: %w", workload.ID, err)
				}
				analysis = decoded
			}
		}
		if schema == PreChoiceSuiteReportSchema && (workload.Choice.ExactReplayAvailable || hasExactChoiceReplay(workload.Seeds)) {
			return SuiteReport{}, fmt.Errorf("previous qualification set workload %s has unsupported exact choice evidence", workload.ID)
		}
		report.Suites[index] = WorkloadReport{
			ID: workload.ID, Name: workload.Name, Tier: workload.Tier, Invariant: workload.Invariant,
			Expected: workload.Expected, ExpectationMet: workload.ExpectationMet, Classification: workload.Classification,
			Analysis: analysis, AnalysisError: workload.AnalysisError, Seeds: workload.Seeds, Blockers: workload.Blockers, Choice: workload.Choice,
			CapabilityMode: target.CapabilityModeClosure,
		}
	}
	if err := validateSetReport(report); err != nil {
		return SuiteReport{}, err
	}
	report.Dimensions.Analysis = false
	for index := range report.Suites {
		report.Suites[index].Analysis = nil
		report.Suites[index].AnalysisError = "dimension_unavailable"
	}
	return report, nil
}

func decodePreviousAnalysis(data []byte, setSchema string) (*AnalysisReport, error) {
	if setSchema == PreviousSuiteReportSchema {
		var header struct {
			Schema string `json:"schema"`
		}
		if err := json.Unmarshal(data, &header); err != nil {
			return nil, err
		}
		if header.Schema != PriorAnalysisSchema {
			return nil, fmt.Errorf("unsupported previous capability analysis schema %q", header.Schema)
		}
		report, err := DecodeAnalysisReport(data)
		if err != nil {
			return nil, err
		}
		return &report, nil
	}
	var previous previousAnalysisReport
	if err := evidence.DecodeCanonicalJSON(data, &previous); err != nil {
		return nil, err
	}
	if previous.Schema != previousAnalysisSchema {
		return nil, fmt.Errorf("unsupported previous capability analysis schema %q", previous.Schema)
	}
	current := AnalysisReport(previous)
	current.Schema = AnalysisSchema
	current.Target.CapabilityMode = target.CapabilityModeClosure
	current.EliminatedBlockers = []AnalysisBlocker{}
	if err := validateAnalysisReport(current); err != nil {
		return nil, err
	}
	return &current, nil
}

func hasExactChoiceReplay(seeds []SeedReport) bool {
	for _, seed := range seeds {
		if seed.ChoiceReplayExact || seed.Choice.ExactReplayAvailable {
			return true
		}
	}
	return false
}
