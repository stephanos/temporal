package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"

	"go.temporal.io/server/tools/common/artifactio"
	"go.temporal.io/server/tools/umpire/caseartifact"
)

const (
	fixtureRoot        = "tools/umpire/testdata/case-runtime-conformance"
	rendererExecutable = "temporal-case-runtime"
)

type generationConfig struct {
	RepositoryRoot string
	OutputRoot     string
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

type rendererOutput struct {
	Stdout []byte
	Stderr []byte
}

type generationDependencies struct {
	Render  func(modelRoot, argument string) (rendererOutput, error)
	Publish func(artifactio.Set, string, map[string][]byte, func(string) error) error
}

func Run(arguments []string) error {
	configuration, err := parseGenerationConfig(arguments)
	if err != nil {
		return err
	}
	return runGeneration(configuration, productionManifest(), defaultGenerationDependencies())
}

func parseGenerationConfig(arguments []string) (generationConfig, error) {
	configuration := generationConfig{RepositoryRoot: ".", OutputRoot: "."}
	flags := flag.NewFlagSet("umpire-gen-case-runtime-conformance", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	flags.StringVar(&configuration.RepositoryRoot, "repository-root", configuration.RepositoryRoot, "repository root containing the built Case renderer")
	flags.StringVar(&configuration.OutputRoot, "output-root", configuration.OutputRoot, "repository-shaped root receiving the complete fixture tree")
	if err := flags.Parse(arguments); err != nil {
		return generationConfig{}, fmt.Errorf("parse Case Runtime conformance generation arguments: %w", err)
	}
	if flags.NArg() != 0 {
		return generationConfig{}, errors.New("unexpected positional arguments for Case Runtime conformance generation")
	}
	if strings.TrimSpace(configuration.RepositoryRoot) == "" || strings.TrimSpace(configuration.OutputRoot) == "" {
		return generationConfig{}, errors.New("repository root and output root are required")
	}
	return configuration, nil
}

func defaultGenerationDependencies() generationDependencies {
	return generationDependencies{
		Render: renderLeanCase,
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
	artifacts := make(map[string][]byte, len(entries)*2)
	for _, entry := range entries {
		output, renderErr := dependencies.Render(modelRoot, entry.RendererArg)
		encoded, err := requireRendererArtifact(entry.Class, output, renderErr)
		if err != nil {
			return err
		}
		decoded, err := caseartifact.DecodeProtoJSON(encoded)
		if err != nil {
			return fmt.Errorf("decode %q Case fixture: %w", entry.Class, err)
		}
		if decoded.GetCaseId() != entry.CaseID {
			return fmt.Errorf("decode %q Case fixture: got Case ID %q, want %q", entry.Class, decoded.GetCaseId(), entry.CaseID)
		}
		if _, err := caseartifact.Pack(encoded); err != nil {
			return fmt.Errorf("pack %q Case fixture: %w", entry.Class, err)
		}
		expected, err := marshalExpected(entry.Expected)
		if err != nil {
			return fmt.Errorf("encode %q expected result: %w", entry.Class, err)
		}
		artifacts[casePath(entry.Class)] = slices.Clone(encoded)
		artifacts[expectedPath(entry.Class)] = expected
	}
	if err := validateArtifacts(entries, artifacts); err != nil {
		return err
	}
	outputRoot, err := filepath.Abs(configuration.OutputRoot)
	if err != nil {
		return fmt.Errorf("resolve fixture output root: %w", err)
	}
	paths := managedPaths(entries)
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
		return validateArtifacts(entries, candidate)
	}
	if err := dependencies.Publish(set, outputRoot, artifacts, validate); err != nil {
		return fmt.Errorf("publish Case Runtime conformance fixtures: %w", err)
	}
	return nil
}

func renderLeanCase(modelRoot, argument string) (rendererOutput, error) {
	command := exec.Command(filepath.Join(modelRoot, ".lake", "build", "bin", rendererExecutable), argument)
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
			return fmt.Errorf("duplicate Case Runtime conformance class %q", entry.Class)
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
		decoded, err := caseartifact.DecodeProtoJSON(encoded)
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
