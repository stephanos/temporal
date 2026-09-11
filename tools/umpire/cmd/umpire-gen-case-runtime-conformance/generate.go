package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"

	"go.temporal.io/server/common/testing/testpilot"
	"go.temporal.io/server/tools/common/artifactio"
)

const (
	fixtureRoot           = "common/testing/testpilot/testdata/case-runtime-conformance"
	functionalFixtureRoot = "tests/testcore/testpilot/testdata"
	rendererExecutable    = "umpire-case"
)

type generationMode string

const (
	generationModeConformance generationMode = "conformance"
	generationModeFunctional  generationMode = "functional"
)

type generationConfig struct {
	RepositoryRoot string
	OutputRoot     string
	Mode           generationMode
}

type stableRuleProjection struct {
	RuleID                   string  `json:"ruleId"`
	Kind                     string  `json:"kind"`
	TerminalStateID          string  `json:"terminalStateId"`
	SupportingEventSequences []int64 `json:"supportingEventSequences"`
}

type stableEventProjection struct {
	Kind                string `json:"kind"`
	EntrypointID        string `json:"entrypointId,omitempty"`
	InstructionID       string `json:"instructionId,omitempty"`
	Attempt             int64  `json:"attempt,omitempty"`
	OutcomeStatus       string `json:"outcomeStatus,omitempty"`
	ExecutionIncomplete bool   `json:"executionIncomplete"`
}

type stableDiagnosticProjection struct {
	Kind string `json:"kind"`
	Code string `json:"code"`
}

type stableRunProjection struct {
	CaseID                   string                       `json:"caseId"`
	ProgramID                string                       `json:"programId"`
	Disposition              string                       `json:"disposition"`
	CleanupStatus            string                       `json:"cleanupStatus"`
	CleanupDiagnostics       []stableDiagnosticProjection `json:"cleanupDiagnostics"`
	Events                   []stableEventProjection      `json:"events"`
	Diagnostics              []stableDiagnosticProjection `json:"diagnostics"`
	VerdictKind              string                       `json:"verdictKind"`
	Rules                    []stableRuleProjection       `json:"rules"`
	SupportingEventSequences []int64                      `json:"supportingEventSequences"`
}

type expectedResult struct {
	Class       string               `json:"class"`
	Preparation string               `json:"preparation"`
	RunCount    int                  `json:"runCount"`
	Projection  *stableRunProjection `json:"projection,omitempty"`
}

type manifestEntry struct {
	Class       string
	RendererArg string
	CaseID      string
	Expected    expectedResult
}

// functionalEntry is one checked-in functional fixture: how to render it, what Case ID it must
// carry, and where it is stored. Every entry but the synthetic one comes from the renderer's own
// registry, so adding a Case never edits this file.
type functionalEntry struct {
	RendererArgs []string
	CaseID       string
	Filename     string
}

type rendererOutput struct {
	Stdout []byte
	Stderr []byte
}

type generationDependencies struct {
	RenderCorrelated func(modelRoot string) (rendererOutput, error)
	Render           func(modelRoot string, arguments ...string) (rendererOutput, error)
	Publish          func(artifactio.Set, string, map[string][]byte, func(string) error) error
}

func Run(arguments []string) error {
	configuration, err := parseGenerationConfig(arguments)
	if err != nil {
		return err
	}
	switch configuration.Mode {
	case generationModeConformance:
		return runGeneration(configuration, productionManifest(), defaultGenerationDependencies())
	case generationModeFunctional:
		return runFunctionalGeneration(configuration, defaultGenerationDependencies())
	default:
		return fmt.Errorf("unknown generation mode %q", configuration.Mode)
	}
}

func parseGenerationConfig(arguments []string) (generationConfig, error) {
	configuration := generationConfig{RepositoryRoot: ".", OutputRoot: ".", Mode: generationModeConformance}
	flags := flag.NewFlagSet("umpire-gen-case-runtime-conformance", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	flags.StringVar(&configuration.RepositoryRoot, "repository-root", configuration.RepositoryRoot, "repository root containing the built Case renderer")
	flags.StringVar(&configuration.OutputRoot, "output-root", configuration.OutputRoot, "repository-shaped root receiving the complete fixture tree")
	flags.Var((*generationModeValue)(&configuration.Mode), "mode", "generation mode: conformance or functional")
	if err := flags.Parse(arguments); err != nil {
		return generationConfig{}, fmt.Errorf("parse Testpilot conformance generation arguments: %w", err)
	}
	if flags.NArg() != 0 {
		return generationConfig{}, errors.New("unexpected positional arguments for Testpilot conformance generation")
	}
	if strings.TrimSpace(configuration.RepositoryRoot) == "" || strings.TrimSpace(configuration.OutputRoot) == "" {
		return generationConfig{}, errors.New("repository root and output root are required")
	}
	if configuration.Mode != generationModeConformance && configuration.Mode != generationModeFunctional {
		return generationConfig{}, fmt.Errorf("unknown generation mode %q", configuration.Mode)
	}
	return configuration, nil
}

type generationModeValue generationMode

func (value *generationModeValue) String() string {
	return string(*value)
}

func (value *generationModeValue) Set(encoded string) error {
	*value = generationModeValue(encoded)
	return nil
}

func defaultGenerationDependencies() generationDependencies {
	return generationDependencies{
		Render: renderLeanCase,
		RenderCorrelated: func(modelRoot string) (rendererOutput, error) {
			return renderExecutable(modelRoot, "umpire-correlated-fixtures")
		},
		Publish: func(set artifactio.Set, root string, artifacts map[string][]byte, validate func(string) error) error {
			return set.Publish(root, artifacts, validate)
		},
	}
}

func runGeneration(configuration generationConfig, entries []manifestEntry, dependencies generationDependencies) error {
	if dependencies.Render == nil || dependencies.Publish == nil {
		return errors.New("missing Case renderer or fixture publisher")
	}
	if err := validateManifest(entries); err != nil {
		return err
	}
	repositoryRoot, err := filepath.Abs(configuration.RepositoryRoot)
	if err != nil {
		return fmt.Errorf("resolve repository root: %w", err)
	}
	modelRoot := filepath.Join(repositoryRoot, "model")
	artifacts, err := renderConformanceArtifacts(entries, modelRoot, dependencies)
	if err != nil {
		return err
	}
	if err := validateArtifacts(entries, artifacts); err != nil {
		return err
	}
	if err := renderCorrelatedArtifacts(dependencies, modelRoot, artifacts); err != nil {
		return err
	}
	outputRoot, err := filepath.Abs(configuration.OutputRoot)
	if err != nil {
		return fmt.Errorf("resolve fixture output root: %w", err)
	}
	paths := make([]string, 0, len(artifacts))
	for path := range artifacts {
		paths = append(paths, path)
	}
	slices.Sort(paths)
	set := artifactio.Set{Roots: []string{fixtureRoot}, Paths: slices.Clone(paths)}
	validate := func(candidateRoot string) error {
		candidate := make(map[string][]byte, len(paths))
		for _, relative := range paths {
			encoded, err := os.ReadFile(filepath.Join(candidateRoot, filepath.FromSlash(relative)))
			if err != nil {
				return fmt.Errorf("read staged fixture %q: %w", relative, err)
			}
			candidate[relative] = encoded
		}
		return validateGeneratedArtifacts(entries, artifacts, candidate)
	}
	if err := dependencies.Publish(set, outputRoot, artifacts, validate); err != nil {
		return fmt.Errorf("publish Testpilot conformance fixtures: %w", err)
	}
	return nil
}

// renderConformanceArtifacts renders each conformance class twice, checks what it decoded to, and
// stores it in the persisted form.
func renderConformanceArtifacts(entries []manifestEntry, modelRoot string, dependencies generationDependencies) (map[string][]byte, error) {
	artifacts := make(map[string][]byte, len(entries)*2)
	for _, entry := range entries {
		encoded, err := renderStable(entry.Class, func() (rendererOutput, error) {
			return dependencies.Render(modelRoot, entry.RendererArg)
		})
		if err != nil {
			return nil, err
		}
		if err := requireRenderedCase(entry.Class, entry.CaseID, encoded); err != nil {
			return nil, err
		}
		expected, err := marshalExpected(entry.Expected)
		if err != nil {
			return nil, fmt.Errorf("encode %q expected result: %w", entry.Class, err)
		}
		stored, err := persistedForm(encoded)
		if err != nil {
			return nil, fmt.Errorf("store %q Case fixture: %w", entry.Class, err)
		}
		artifacts[casePath(entry.Class)] = stored
		artifacts[expectedPath(entry.Class)] = expected
	}
	return artifacts, nil
}

// requireRenderedCase confirms the renderer produced the Case the manifest names, and that it packs.
func requireRenderedCase(class, caseID string, encoded []byte) error {
	decoded, err := testpilot.DecodeCaseProtoJSON(encoded)
	if err != nil {
		return fmt.Errorf("decode %q Case fixture: %w", class, err)
	}
	if decoded.GetCaseId() != caseID {
		return fmt.Errorf("decode %q Case fixture: got Case ID %q, want %q", class, decoded.GetCaseId(), caseID)
	}
	if _, err := testpilot.PackCaseProtoJSON(encoded); err != nil {
		return fmt.Errorf("pack %q Case fixture: %w", class, err)
	}
	return nil
}

func renderStable(class string, render func() (rendererOutput, error)) ([]byte, error) {
	output, renderErr := render()
	encoded, err := requireRendererArtifact(class, output, renderErr)
	if err != nil {
		return nil, err
	}
	output, renderErr = render()
	repeated, err := requireRendererArtifact(class, output, renderErr)
	if err != nil {
		return nil, err
	}
	if !bytes.Equal(encoded, repeated) {
		return nil, fmt.Errorf("render %q Testpilot Case fixture: non-deterministic bytes", class)
	}
	return encoded, nil
}

func renderCorrelatedArtifacts(dependencies generationDependencies, modelRoot string, artifacts map[string][]byte) error {
	if dependencies.RenderCorrelated == nil {
		return nil
	}
	encoded, err := renderStable("correlated", func() (rendererOutput, error) { return dependencies.RenderCorrelated(modelRoot) })
	if err != nil {
		return err
	}
	if !json.Valid(encoded) {
		return errors.New("invalid correlated fixtures")
	}
	stored, err := persistedForm(encoded)
	if err != nil {
		return fmt.Errorf("store correlated fixtures: %w", err)
	}
	artifacts[fixtureRoot+"/correlated.json"] = stored
	return nil
}

func validateGeneratedArtifacts(entries []manifestEntry, artifacts, candidate map[string][]byte) error {
	if expected, ok := artifacts[fixtureRoot+"/correlated.json"]; ok {
		encoded := candidate[fixtureRoot+"/correlated.json"]
		if err := requirePersistedForm(fixtureRoot+"/correlated.json", encoded); err != nil {
			return err
		}
		if !bytes.Equal(encoded, expected) {
			return errors.New("invalid staged correlated fixtures")
		}
		delete(candidate, fixtureRoot+"/correlated.json")
	}
	return validateArtifacts(entries, candidate)
}

func runFunctionalGeneration(configuration generationConfig, dependencies generationDependencies) error {
	if dependencies.Render == nil || dependencies.Publish == nil {
		return errors.New("missing Case renderer or fixture publisher")
	}
	repositoryRoot, err := filepath.Abs(configuration.RepositoryRoot)
	if err != nil {
		return fmt.Errorf("resolve repository root: %w", err)
	}
	modelRoot := filepath.Join(repositoryRoot, "model")
	entries, err := functionalEntries(modelRoot, dependencies)
	if err != nil {
		return err
	}
	artifacts, err := renderFunctionalArtifacts(entries, modelRoot, dependencies)
	if err != nil {
		return err
	}
	paths := make([]string, 0, len(entries))
	for _, entry := range entries {
		paths = append(paths, functionalCasePath(entry))
	}
	slices.Sort(paths)
	if err := validateFunctionalArtifacts(entries, artifacts); err != nil {
		return err
	}
	outputRoot, err := filepath.Abs(configuration.OutputRoot)
	if err != nil {
		return fmt.Errorf("resolve fixture output root: %w", err)
	}
	set := artifactio.Set{Roots: []string{functionalFixtureRoot}, Paths: slices.Clone(paths)}
	validate := func(candidateRoot string) error {
		candidate := make(map[string][]byte, len(paths))
		for _, relative := range paths {
			encoded, err := os.ReadFile(filepath.Join(candidateRoot, filepath.FromSlash(relative)))
			if err != nil {
				return fmt.Errorf("read staged fixture %q: %w", relative, err)
			}
			candidate[relative] = encoded
		}
		return validateFunctionalArtifacts(entries, candidate)
	}
	if err := dependencies.Publish(set, outputRoot, artifacts, validate); err != nil {
		return fmt.Errorf("publish functional Testpilot Case fixtures: %w", err)
	}
	return nil
}

// renderFunctionalArtifacts renders each functional Case twice, checks what it decoded to, and
// stores it in the persisted form.
func renderFunctionalArtifacts(entries []functionalEntry, modelRoot string, dependencies generationDependencies) (map[string][]byte, error) {
	artifacts := make(map[string][]byte, len(entries))
	for _, entry := range entries {
		encoded, err := renderStable(entry.Filename, func() (rendererOutput, error) {
			return dependencies.Render(modelRoot, entry.RendererArgs...)
		})
		if err != nil {
			return nil, err
		}
		if err := requireRenderedCase(entry.Filename, entry.CaseID, encoded); err != nil {
			return nil, err
		}
		stored, err := persistedForm(encoded)
		if err != nil {
			return nil, fmt.Errorf("store %q Testpilot Case fixture: %w", entry.Filename, err)
		}
		artifacts[functionalCasePath(entry)] = stored
	}
	return artifacts, nil
}

func validateFunctionalArtifacts(entries []functionalEntry, artifacts map[string][]byte) error {
	if len(artifacts) != len(entries) {
		return fmt.Errorf("functional fixture set has %d files, want %d", len(artifacts), len(entries))
	}
	for _, entry := range entries {
		encoded, ok := artifacts[functionalCasePath(entry)]
		if !ok {
			return fmt.Errorf("missing functional Case fixture %q", entry.Filename)
		}
		if err := requirePersistedForm(functionalCasePath(entry), encoded); err != nil {
			return err
		}
		decoded, err := testpilot.DecodeCaseProtoJSON(encoded)
		if err != nil || decoded.GetCaseId() != entry.CaseID {
			return fmt.Errorf("invalid functional Case fixture %q", entry.Filename)
		}
	}
	return nil
}

func functionalCasePath(entry functionalEntry) string {
	return filepath.ToSlash(filepath.Join(functionalFixtureRoot, entry.Filename))
}

// syntheticEntry is the one functional fixture the registry does not model: it carries no Case
// value the renderer registers, so it keeps being named by its own renderer argument.
func syntheticEntry() functionalEntry {
	return functionalEntry{
		RendererArgs: []string{"synthetic"},
		CaseID:       "testpilot.synthetic.case",
		Filename:     "synthetic-case.json",
	}
}

// functionalEntries asks the renderer what Cases exist rather than being told. `--list` is read
// twice and compared for the same reason every fixture is rendered twice: an unstable enumeration
// would silently reorder or drop a checked-in file.
func functionalEntries(modelRoot string, dependencies generationDependencies) ([]functionalEntry, error) {
	listed, err := renderStable("--list", func() (rendererOutput, error) {
		return dependencies.Render(modelRoot, "--list")
	})
	if err != nil {
		return nil, err
	}
	entries := []functionalEntry{}
	for _, line := range strings.Split(strings.TrimSpace(string(listed)), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) != 2 {
			return nil, fmt.Errorf("list registered Cases: %q is not \"<case-id> <fixture-name>\"", line)
		}
		entries = append(entries, functionalEntry{
			RendererArgs: []string{"--render", fields[0]},
			CaseID:       fields[0],
			Filename:     fields[1] + "-case.json",
		})
	}
	if len(entries) == 0 {
		return nil, errors.New("list registered Cases: the renderer registered none")
	}
	entries = append(entries, syntheticEntry())
	return entries, nil
}

func renderLeanCase(modelRoot string, arguments ...string) (rendererOutput, error) {
	return renderExecutable(modelRoot, rendererExecutable, arguments...)
}

func renderExecutable(modelRoot, executable string, arguments ...string) (rendererOutput, error) {
	command := exec.Command(filepath.Join(modelRoot, ".lake", "build", "bin", executable), arguments...)
	command.Dir = modelRoot
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	err := command.Run()
	return rendererOutput{Stdout: stdout.Bytes(), Stderr: stderr.Bytes()}, err
}

func requireRendererArtifact(class string, output rendererOutput, renderErr error) ([]byte, error) {
	stdout := bytes.TrimSpace(output.Stdout)
	stderr := bytes.TrimSpace(output.Stderr)
	if renderErr != nil {
		if len(stdout) != 0 {
			return nil, fmt.Errorf("render %q Case: renderer failed while also producing stdout: %w", class, renderErr)
		}
		return nil, fmt.Errorf("render %q Case: %s: %w", class, stderr, renderErr)
	}
	if len(stdout) == 0 {
		return nil, fmt.Errorf("render %q Case: renderer produced an empty artifact", class)
	}
	if len(stderr) != 0 {
		return nil, fmt.Errorf("render %q Case: renderer produced contradictory stderr: %s", class, stderr)
	}
	return output.Stdout, nil
}

func validateManifest(entries []manifestEntry) error {
	if len(entries) != 6 {
		return fmt.Errorf("conformance manifest has %d classes, want exactly 6", len(entries))
	}
	classes := make(map[string]struct{}, len(entries))
	for _, entry := range entries {
		if strings.TrimSpace(entry.Class) == "" || strings.TrimSpace(entry.RendererArg) == "" || strings.TrimSpace(entry.CaseID) == "" {
			return errors.New("conformance manifest entries require class, renderer argument and Case ID")
		}
		if _, duplicate := classes[entry.Class]; duplicate {
			return fmt.Errorf("duplicate Testpilot conformance class %q", entry.Class)
		}
		classes[entry.Class] = struct{}{}
		if entry.Expected.Class != entry.Class {
			return fmt.Errorf("expected result class %q does not match manifest class %q", entry.Expected.Class, entry.Class)
		}
	}
	return nil
}

func validateArtifacts(entries []manifestEntry, artifacts map[string][]byte) error {
	paths := managedPaths(entries)
	if len(artifacts) != len(paths) {
		return fmt.Errorf("conformance fixture set has %d files, want %d", len(artifacts), len(paths))
	}
	for _, entry := range entries {
		encoded, ok := artifacts[casePath(entry.Class)]
		if !ok {
			return fmt.Errorf("missing Case fixture for %q", entry.Class)
		}
		if err := requirePersistedForm(casePath(entry.Class), encoded); err != nil {
			return err
		}
		decoded, err := testpilot.DecodeCaseProtoJSON(encoded)
		if err != nil || decoded.GetCaseId() != entry.CaseID {
			return fmt.Errorf("invalid Case fixture for %q", entry.Class)
		}
		expected, ok := artifacts[expectedPath(entry.Class)]
		if !ok {
			return fmt.Errorf("missing expected result for %q", entry.Class)
		}
		canonical, err := marshalExpected(entry.Expected)
		if err != nil || !bytes.Equal(expected, canonical) {
			return fmt.Errorf("non-canonical expected result for %q", entry.Class)
		}
	}
	return nil
}

func managedPaths(entries []manifestEntry) []string {
	paths := make([]string, 0, len(entries)*2)
	for _, entry := range entries {
		paths = append(paths, casePath(entry.Class), expectedPath(entry.Class))
	}
	slices.Sort(paths)
	return paths
}

func casePath(class string) string {
	return filepath.ToSlash(filepath.Join(fixtureRoot, class, "case.json"))
}

func expectedPath(class string) string {
	return filepath.ToSlash(filepath.Join(fixtureRoot, class, "expected.json"))
}

func productionManifest() []manifestEntry {
	return []manifestEntry{
		acceptedEntry("satisfied", "temporal.case.conformance.satisfied", "SATISFIED", "SATISFIED", "COMPLETED", "SUCCEEDED", 1),
		acceptedEntry("violated", "temporal.case.conformance.violated", "VIOLATED", "VIOLATED", "STOPPED_BY_MONITOR", "SUCCEEDED", 1),
		acceptedEntry("inconclusive", "temporal.case.conformance.inconclusive", "INCONCLUSIVE", "INCONCLUSIVE", "COMPLETED", "SUCCEEDED", 1),
		{
			Class: "static-preparation-rejection", RendererArg: "conformance-static-preparation-rejection",
			CaseID:   "temporal.case.conformance.static-rejection",
			Expected: expectedResult{Class: "static-preparation-rejection", Preparation: "rejected"},
		},
		acceptedEntry("cleanup-failure-after-proved-violation", "temporal.case.conformance.cleanup-failure", "VIOLATED", "VIOLATED", "STOPPED_BY_MONITOR", "FAILED", 1),
		acceptedEntry("cross-run-isolation", "temporal.case.conformance.cross-run-isolation", "SATISFIED", "SATISFIED", "COMPLETED", "SUCCEEDED", 2),
	}
}

func acceptedEntry(class, caseID, verdict, rule, disposition, cleanup string, runCount int) manifestEntry {
	support := []int64{4}
	terminal := "terminal"
	if verdict == "INCONCLUSIVE" {
		support = []int64{}
		terminal = ""
	}
	events := completedEvents()
	if verdict == "VIOLATED" {
		events = stoppedEvents()
	}
	if cleanup == "FAILED" {
		events = cleanupFailureEvents()
	}
	diagnostics := []stableDiagnosticProjection{}
	cleanupDiagnostics := []stableDiagnosticProjection{}
	if cleanup == "FAILED" {
		diagnostic := stableDiagnosticProjection{Kind: "EXECUTION", Code: "cleanup_failed"}
		diagnostics = append(diagnostics, diagnostic)
		cleanupDiagnostics = append(cleanupDiagnostics, diagnostic)
	}
	return manifestEntry{
		Class: class, RendererArg: "conformance-" + class, CaseID: caseID,
		Expected: expectedResult{
			Class: class, Preparation: "accepted", RunCount: runCount,
			Projection: &stableRunProjection{
				CaseID: caseID, ProgramID: caseID + ".program", Disposition: disposition,
				CleanupStatus: cleanup, CleanupDiagnostics: cleanupDiagnostics, Events: events,
				Diagnostics: diagnostics, VerdictKind: verdict,
				Rules:                    []stableRuleProjection{{RuleID: "result", Kind: rule, TerminalStateID: terminal, SupportingEventSequences: slices.Clone(support)}},
				SupportingEventSequences: slices.Clone(support),
			},
		},
	}
}

func completedEvents() []stableEventProjection {
	return []stableEventProjection{
		{Kind: "RUN_OPENED"},
		{Kind: "ACTIVATION_OPENED", EntrypointID: "controller"},
		{Kind: "INSTRUCTION_STARTED", EntrypointID: "controller", InstructionID: "execute", Attempt: 1},
		{Kind: "INSTRUCTION_COMPLETED", EntrypointID: "controller", InstructionID: "execute", Attempt: 1, OutcomeStatus: "SUCCEEDED"},
		{Kind: "ACTIVATION_CLOSED", EntrypointID: "controller"},
		{Kind: "CLEANUP_STARTED", EntrypointID: "cleanup"},
		{Kind: "CLEANUP_COMPLETED", EntrypointID: "cleanup"},
		{Kind: "RUN_CLOSED"},
	}
}

func stoppedEvents() []stableEventProjection {
	return []stableEventProjection{
		{Kind: "RUN_OPENED"},
		{Kind: "ACTIVATION_OPENED", EntrypointID: "controller"},
		{Kind: "INSTRUCTION_STARTED", EntrypointID: "controller", InstructionID: "execute", Attempt: 1},
		{Kind: "INSTRUCTION_COMPLETED", EntrypointID: "controller", InstructionID: "execute", Attempt: 1, OutcomeStatus: "SUCCEEDED"},
		{Kind: "CLEANUP_STARTED", EntrypointID: "cleanup"},
		{Kind: "CLEANUP_COMPLETED", EntrypointID: "cleanup"},
		{Kind: "RUN_CLOSED"},
	}
}

func cleanupFailureEvents() []stableEventProjection {
	events := stoppedEvents()
	return slices.Insert(events, len(events)-2,
		stableEventProjection{Kind: "INSTRUCTION_STARTED", EntrypointID: "cleanup", InstructionID: "fail-cleanup", Attempt: 1},
	)
}
