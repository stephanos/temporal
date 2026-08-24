package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"go.temporal.io/server/tools/gomad3/internal/canonicaljson"
	qualificationset "go.temporal.io/server/tools/gomad3/qualification/set"
)

type qualifySetDependencies struct {
	executable func() (string, error)
	load       func(string) (qualificationset.Manifest, error)
	run        func(context.Context, qualificationset.Spec) (qualificationset.Report, error)
}

func runQualifySet(arguments []string, stdout, stderr io.Writer) int {
	return runQualifySetWith(arguments, stdout, stderr, qualifySetDependencies{
		executable: os.Executable, load: qualificationset.LoadManifest, run: qualificationset.Run,
	})
}

func runQualifySetWith(arguments []string, stdout, stderr io.Writer, dependencies qualifySetDependencies) int {
	flags := flag.NewFlagSet("gomad qualify-set", flag.ContinueOnError)
	flags.SetOutput(stderr)
	manifestPath := flags.String("manifest", "", "qualification manifest")
	workingDirectory := flags.String("working-dir", "", "target module directory")
	artifacts := flags.String("artifacts", ".gomad/qualification", "qualification artifact root")
	output := flags.String("output", ".gomad/qualification-set.json", "qualification set report")
	format := flags.String("format", "text", "text or json")
	check := flags.Bool("check", false, "validate the manifest without executing targets")
	if err := flags.Parse(arguments); err != nil || flags.NArg() != 0 {
		return 2
	}
	if *manifestPath == "" || *workingDirectory == "" || (*format != "text" && *format != "json") {
		return writeCommandError(stderr, 2, "qualify-set requires --manifest, --working-dir, and --format=text|json\n")
	}
	manifest, err := dependencies.load(*manifestPath)
	if err != nil {
		return writeCommandError(stderr, 2, "load qualification manifest: %v\n", err)
	}
	if *check {
		if *format == "json" {
			encoded, encodeErr := canonicaljson.CanonicalJSON(struct {
				Schema    string `json:"schema"`
				Name      string `json:"name"`
				Workloads uint64 `json:"workloads"`
			}{"gomad3.qualification-set-check/v1", manifest.Name, uint64(len(manifest.Workloads))})
			if encodeErr != nil {
				return writeCommandError(stderr, 3, "encode qualification manifest result: %v\n", encodeErr)
			}
			if _, err := fmt.Fprintf(stdout, "%s\n", encoded); err != nil {
				return 3
			}
		} else if _, err := fmt.Fprintf(stdout, "qualification manifest: name=%s workloads=%d\n", manifest.Name, len(manifest.Workloads)); err != nil {
			return 3
		}
		return 0
	}
	executable, err := dependencies.executable()
	if err != nil {
		return writeCommandError(stderr, 3, "resolve gomad executable: %v\n", err)
	}
	report, runErr := dependencies.run(context.Background(), qualificationset.Spec{
		ManifestPath: *manifestPath, GomadPath: executable, WorkingDir: *workingDirectory,
		ArtifactRoot: *artifacts, OutputPath: *output,
	})
	if err := writeQualificationSetResult(stdout, *format, report); err != nil {
		return writeCommandError(stderr, 3, "write qualification set result: %v\n", err)
	}
	if runErr == nil {
		return 0
	}
	status, message := classifyQualificationSetError(report, runErr)
	if message == "" {
		return status
	}
	return writeCommandError(stderr, status, "%s: %v\n", message, runErr)
}

func classifyQualificationSetError(report qualificationset.Report, runErr error) (int, string) {
	for _, workload := range report.Workloads {
		if workload.Classification == "invalid_input" || workload.AnalysisError == "invalid_input" {
			return 2, "invalid qualification set workload"
		}
	}
	if errors.Is(runErr, context.Canceled) || errors.Is(runErr, context.DeadlineExceeded) || report.InfrastructureErrors != 0 {
		return 3, "qualification set infrastructure failure"
	}
	var mismatch *qualificationset.ExpectationError
	if errors.As(runErr, &mismatch) {
		return 1, ""
	}
	if report.Schema == "" {
		return 2, "invalid qualification set input"
	}
	return 3, "qualification set failure"
}

func writeQualificationSetResult(output io.Writer, format string, report qualificationset.Report) error {
	if format == "json" {
		encoded, err := canonicaljson.CanonicalJSON(report)
		if err != nil {
			return err
		}
		_, err = fmt.Fprintf(output, "%s\n", encoded)
		return err
	}
	_, err := fmt.Fprintf(output, "qualification set: name=%s expectations-met=%t supported=%d unsupported=%d failed=%d infrastructure-errors=%d completed=%d/%d\n", report.Name, report.ExpectationsMet, report.Supported, report.Unsupported, report.Failed, report.InfrastructureErrors, report.Completed, report.Selected)
	return err
}
