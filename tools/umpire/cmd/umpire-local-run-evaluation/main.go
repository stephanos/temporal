package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"

	"go.temporal.io/server/tools/umpire/artifact"
	"go.temporal.io/server/tools/umpire/runevaluation"
)

const (
	exitSatisfied    = 0
	exitToolingError = 1
	exitNotSatisfied = 2

	summaryFormat = "umpire-local-run-evaluation-summary/v2"
	errorFormat   = "umpire-local-run-evaluation-error/v2"
)

type commandDependencies struct {
	loadSet    func(string) (artifact.AdmittedSet, error)
	check      func(artifact.AdmittedSet) (artifact.AdmittedSet, error)
	publishSet func(string, artifact.AdmittedSet) (string, error)
}

type commandSummary struct {
	FormatVersion               string  `json:"formatVersion"`
	RunIdentity                 string  `json:"runIdentity"`
	OperationalStatus           string  `json:"operationalStatus"`
	ObservationEvaluationStatus string  `json:"observationEvaluationStatus"`
	SemanticStatus              string  `json:"semanticStatus"`
	EvidenceArtifactChecksum    string  `json:"evidenceArtifactChecksum"`
	ResultArtifactChecksum      string  `json:"resultArtifactChecksum"`
	EvaluationOutcomeChecksum   *string `json:"evaluationOutcomeChecksum"`
	ArtifactSetChecksum         string  `json:"artifactSetChecksum"`
	ManifestSHA256              string  `json:"manifestSha256"`
	Destination                 string  `json:"destination"`
}

type commandError struct {
	FormatVersion       string  `json:"formatVersion"`
	Kind                string  `json:"kind"`
	Phase               string  `json:"phase"`
	Subject             string  `json:"subject"`
	Code                string  `json:"code"`
	CheckingOccurred    bool    `json:"checkingOccurred"`
	PublicationOccurred bool    `json:"publicationOccurred"`
	RunIdentity         *string `json:"runIdentity"`
	ArtifactSetChecksum *string `json:"artifactSetChecksum"`
	ManifestSHA256      *string `json:"manifestSha256"`
	Destination         *string `json:"destination"`
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(arguments []string, stdout io.Writer, stderr io.Writer) int {
	return execute(context.Background(), arguments, stdout, stderr, commandDependencies{
		loadSet:    loadInputSet,
		check:      runevaluation.Check,
		publishSet: artifact.PublishSet,
	})
}

func execute(
	ctx context.Context,
	arguments []string,
	stdout io.Writer,
	stderr io.Writer,
	dependencies commandDependencies,
) int {
	setPath, outputRoot, err := parseArguments(arguments)
	if err != nil {
		return reportError(stderr, commandError{
			Kind: "arguments", Phase: "admission", Subject: "arguments",
			Code: "umpire.run-evaluation.arguments.invalid",
		})
	}
	physicalSet, err := physicalDirectory(setPath)
	if err != nil {
		return reportError(stderr, commandError{
			Kind: "input", Phase: "admission", Subject: "set",
			Code: "umpire.run-evaluation.input.unsafe-path",
		})
	}
	physicalOutputRoot, err := physicalDirectory(outputRoot)
	if err != nil || pathsOverlap(physicalSet, physicalOutputRoot) {
		return reportError(stderr, commandError{
			Kind: "publication", Phase: "publication", Subject: "output-root",
			Code: "umpire.run-evaluation.publication.unsafe-root",
		})
	}
	input, err := dependencies.loadSet(setPath)
	if err != nil {
		return reportError(stderr, commandError{
			Kind: "input", Phase: "admission", Subject: "set",
			Code: inputErrorCode(err),
		})
	}
	execution, ok := input.Execution()
	if !ok {
		return reportError(stderr, commandError{
			Kind: "input", Phase: "admission", Subject: "set",
			Code: "umpire.run-evaluation.input.exact-four-member-set",
		})
	}
	runIdentity := execution.ExperimentRun().RunIdentity
	if err := ctx.Err(); err != nil {
		return reportError(stderr, commandError{
			Kind: "checker", Phase: "Observation Evaluation", Subject: "checker",
			Code: "umpire.run-evaluation.checker.canceled", RunIdentity: &runIdentity,
		})
	}
	output, err := dependencies.check(input)
	if err != nil {
		kind, phase, code := evaluationError(err)
		return reportError(stderr, commandError{
			Kind: kind, Phase: phase, Subject: "set", Code: code,
			CheckingOccurred: kind != "input", RunIdentity: &runIdentity,
		})
	}
	destination, err := dependencies.publishSet(outputRoot, output)
	if err != nil {
		return reportError(stderr, commandError{
			Kind: "publication", Phase: "publication", Subject: "output-root",
			Code: "umpire.run-evaluation.publication.failed", CheckingOccurred: true,
			RunIdentity: &runIdentity,
		})
	}
	artifactSetChecksum := output.Checksum()
	manifestSHA256 := output.ManifestSHA256()
	summary, err := evaluationSummary(destination, output)
	if err != nil {
		return reportError(stderr, publishedReportingError(
			runIdentity, artifactSetChecksum, manifestSHA256, destination,
		))
	}
	encoded, err := json.Marshal(summary)
	if err != nil {
		return reportError(stderr, publishedReportingError(
			runIdentity, artifactSetChecksum, manifestSHA256, destination,
		))
	}
	encoded = append(encoded, '\n')
	if _, err := stdout.Write(encoded); err != nil {
		return reportError(stderr, publishedReportingError(
			runIdentity, artifactSetChecksum, manifestSHA256, destination,
		))
	}
	return summaryExitStatus(summary)
}

func summaryExitStatus(summary commandSummary) int {
	if summary.OperationalStatus == "succeeded" &&
		summary.ObservationEvaluationStatus == "accepted" &&
		summary.SemanticStatus == "satisfied" {
		return exitSatisfied
	}
	return exitNotSatisfied
}

func parseArguments(arguments []string) (setPath string, outputRoot string, err error) {
	if len(arguments) != 4 || arguments[0] != "--set" || arguments[2] != "--output-root" ||
		arguments[1] == "" || arguments[3] == "" || strings.HasPrefix(arguments[1], "--") ||
		strings.HasPrefix(arguments[3], "--") {
		return "", "", errors.New("expected --set <directory> --output-root <directory>")
	}
	return arguments[1], arguments[3], nil
}

func loadInputSet(root string) (artifact.AdmittedSet, error) {
	root, err := physicalDirectory(root)
	if err != nil {
		return artifact.AdmittedSet{}, err
	}
	rootEntries, err := os.ReadDir(root)
	if err != nil {
		return artifact.AdmittedSet{}, err
	}
	if len(rootEntries) != 2 || rootEntries[0].Name() != "artifacts" || !rootEntries[0].IsDir() ||
		rootEntries[1].Name() != "manifest.json" || !rootEntries[1].Type().IsRegular() {
		return artifact.AdmittedSet{}, errors.New("input set has unexpected root entries")
	}
	artifactRoot := filepath.Join(root, "artifacts")
	artifactEntries, err := os.ReadDir(artifactRoot)
	if err != nil {
		return artifact.AdmittedSet{}, err
	}
	want := []string{"experiment-run.json", "experiment.json", "raw-evidence.json", "runtime-configuration.json"}
	if len(artifactEntries) != len(want) {
		return artifact.AdmittedSet{}, errors.New("input set does not have exactly four members")
	}
	for index, entry := range artifactEntries {
		if entry.Name() != want[index] || !entry.Type().IsRegular() {
			return artifact.AdmittedSet{}, errors.New("input set contains an unexpected member")
		}
	}
	files := make(map[string][]byte, 5)
	for _, path := range []string{
		"manifest.json",
		"artifacts/experiment.json",
		"artifacts/runtime-configuration.json",
		"artifacts/experiment-run.json",
		"artifacts/raw-evidence.json",
	} {
		encoded, err := readRegularFile(filepath.Join(root, filepath.FromSlash(path)))
		if err != nil {
			return artifact.AdmittedSet{}, err
		}
		files[path] = encoded
	}
	return artifact.AdmitSetFiles(files)
}

func readRegularFile(path string) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Size() > artifact.MaximumDocumentBytes {
		return nil, errors.New("input member is not a bounded regular file")
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()
	opened, err := file.Stat()
	if err != nil {
		return nil, err
	}
	if !opened.Mode().IsRegular() || !os.SameFile(info, opened) {
		return nil, errors.New("input member changed while it was opened")
	}
	encoded, err := io.ReadAll(io.LimitReader(file, artifact.MaximumDocumentBytes+1))
	if err != nil {
		return nil, err
	}
	if int64(len(encoded)) > artifact.MaximumDocumentBytes {
		return nil, errors.New("input member exceeds the byte limit")
	}
	return encoded, nil
}

func physicalDirectory(path string) (string, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	for current := absolute; ; current = filepath.Dir(current) {
		info, err := os.Lstat(current)
		if err != nil {
			return "", err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return "", errors.New("path contains a symbolic link")
		}
		if current == absolute && !info.IsDir() {
			return "", errors.New("path is not a directory")
		}
		if parent := filepath.Dir(current); parent == current {
			break
		}
	}
	resolved, err := filepath.EvalSymlinks(absolute)
	if err != nil || filepath.Clean(resolved) != filepath.Clean(absolute) {
		return "", errors.New("path is not physical")
	}
	return resolved, nil
}

func pathsOverlap(left string, right string) bool {
	for _, pair := range [][2]string{{left, right}, {right, left}} {
		relative, err := filepath.Rel(pair[0], pair[1])
		if err == nil && relative != ".." &&
			!strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
			return true
		}
	}
	return false
}

func evaluationSummary(destination string, admitted artifact.AdmittedSet) (commandSummary, error) {
	loaded, err := artifact.LoadSet(destination)
	if err != nil {
		return commandSummary{}, err
	}
	if loaded.Checksum() != admitted.Checksum() || loaded.ManifestSHA256() != admitted.ManifestSHA256() {
		return commandSummary{}, errors.New("published set identity drifted")
	}
	evidenceBytes, err := readRegularFile(filepath.Join(destination, "artifacts", "evidence.json"))
	if err != nil {
		return commandSummary{}, err
	}
	resultBytes, err := readRegularFile(filepath.Join(destination, "artifacts", "result.json"))
	if err != nil {
		return commandSummary{}, err
	}
	evidence, err := artifact.DecodeEvidenceV2(evidenceBytes)
	if err != nil {
		return commandSummary{}, err
	}
	result, err := artifact.DecodeResultV2(resultBytes)
	if err != nil {
		return commandSummary{}, err
	}
	if result.Evidence.ArtifactChecksum != evidence.ArtifactChecksum {
		return commandSummary{}, errors.New("published Evidence binding drifted")
	}
	return commandSummary{
		FormatVersion: summaryFormat, RunIdentity: result.RunIdentity,
		OperationalStatus:           result.OperationalStatus,
		ObservationEvaluationStatus: result.ObservationEvaluationStatus,
		SemanticStatus:              result.SemanticStatus,
		EvidenceArtifactChecksum:    evidence.ArtifactChecksum,
		ResultArtifactChecksum:      result.ArtifactChecksum,
		EvaluationOutcomeChecksum:   result.EvaluationOutcomeChecksum,
		ArtifactSetChecksum:         admitted.Checksum(), ManifestSHA256: admitted.ManifestSHA256(),
		Destination: destination,
	}, nil
}

func inputErrorCode(err error) string {
	if code, ok := artifact.CodeOf(err); ok {
		return "umpire.run-evaluation.input." + string(code)
	}
	return "umpire.run-evaluation.input.invalid"
}

func evaluationError(err error) (kind string, phase string, code string) {
	type classified interface {
		Kind() string
		Phase() string
		Code() string
	}
	var failure classified
	if errors.As(err, &failure) {
		return failure.Kind(), failure.Phase(), failure.Code()
	}
	return "checker", "evaluation", "umpire.run-evaluation.checker.failed"
}

func publishedReportingError(
	runIdentity string,
	artifactSetChecksum string,
	manifestSHA256 string,
	destination string,
) commandError {
	return commandError{
		Kind: "reporting", Phase: "reporting", Subject: "stdout",
		Code: "umpire.run-evaluation.reporting.failed", CheckingOccurred: true,
		PublicationOccurred: true, RunIdentity: &runIdentity,
		ArtifactSetChecksum: &artifactSetChecksum, ManifestSHA256: &manifestSHA256,
		Destination: &destination,
	}
}

func reportError(stderr io.Writer, report commandError) int {
	report.FormatVersion = errorFormat
	encoded, err := json.Marshal(report)
	if err == nil {
		encoded = append(encoded, '\n')
		_, _ = stderr.Write(encoded)
	}
	return exitToolingError
}
