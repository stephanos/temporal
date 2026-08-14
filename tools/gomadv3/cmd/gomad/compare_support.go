package main

import (
	"flag"
	"fmt"
	"io"

	"go.temporal.io/server/tools/gomadv3/internal/qualificationset"
	"go.temporal.io/server/tools/gomadv3/internal/record"
	"go.temporal.io/server/tools/gomadv3/internal/supportcompare"
)

type compareSupportDependencies struct {
	open    func(string) (qualificationset.SetReport, error)
	compare func(supportcompare.Input) (supportcompare.Result, error)
}

func runCompareSupport(arguments []string, stdout, stderr io.Writer) int {
	return runCompareSupportWith(arguments, stdout, stderr, compareSupportDependencies{
		open: qualificationset.OpenReport, compare: supportcompare.Compare,
	})
}

func runCompareSupportWith(arguments []string, stdout, stderr io.Writer, dependencies compareSupportDependencies) int {
	flags := flag.NewFlagSet("gomad compare-support", flag.ContinueOnError)
	flags.SetOutput(stderr)
	baselinePath := flags.String("baseline", "", "baseline qualification-set report")
	candidatePath := flags.String("candidate", "", "candidate qualification-set report")
	approvedBoundary := flags.String("approve-boundary-diff", "", "exact approved boundary diff SHA-256")
	format := flags.String("format", "text", "text or json")
	if err := flags.Parse(arguments); err != nil || flags.NArg() != 0 {
		return 2
	}
	if *baselinePath == "" || *candidatePath == "" || (*format != "text" && *format != "json") {
		return writeCommandError(stderr, 2, "compare-support requires --baseline, --candidate, and --format=text|json\n")
	}
	var approval record.SHA256
	if *approvedBoundary != "" {
		parsed, err := record.ParseSHA256(*approvedBoundary)
		if err != nil {
			return writeCommandError(stderr, 2, "invalid boundary approval: %v\n", err)
		}
		approval = parsed
	}
	baseline, err := dependencies.open(*baselinePath)
	if err != nil {
		if qualificationset.IsInvalidReport(err) {
			return writeCommandError(stderr, 2, "open baseline support report: %v\n", err)
		}
		return writeCommandError(stderr, 3, "open baseline support report: %v\n", err)
	}
	candidate, err := dependencies.open(*candidatePath)
	if err != nil {
		if qualificationset.IsInvalidReport(err) {
			return writeCommandError(stderr, 2, "open candidate support report: %v\n", err)
		}
		return writeCommandError(stderr, 3, "open candidate support report: %v\n", err)
	}
	result, err := dependencies.compare(supportcompare.Input{Baseline: baseline, Candidate: candidate, ApprovedBoundaryDiff: approval})
	if err != nil {
		return writeCommandError(stderr, 2, "compare support reports: %v\n", err)
	}
	if *format == "json" {
		encoded, encodeErr := record.CanonicalJSON(result)
		if encodeErr != nil {
			return writeCommandError(stderr, 3, "encode support comparison: %v\n", encodeErr)
		}
		if _, err := fmt.Fprintf(stdout, "%s\n", encoded); err != nil {
			return 3
		}
	} else if _, err := fmt.Fprint(stdout, supportcompare.FormatText(result)); err != nil {
		return 3
	}
	if result.Classification == supportcompare.ClassificationIncomparable {
		return 2
	}
	if result.ReviewRequired {
		return 1
	}
	return 0
}
