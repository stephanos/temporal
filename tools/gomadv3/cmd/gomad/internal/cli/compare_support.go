package cli

import (
	"flag"
	"fmt"
	"io"

	"go.temporal.io/server/tools/gomadv3/evidence"
	"go.temporal.io/server/tools/gomadv3/qualification"
)

type compareSupportDependencies struct {
	open    func(string) (qualification.SuiteReport, error)
	compare func(qualification.ComparisonInput) (qualification.Comparison, error)
}

func runCompareSupport(arguments []string, stdout, stderr io.Writer) int {
	return runCompareSupportWith(arguments, stdout, stderr, compareSupportDependencies{
		open: qualification.OpenSuiteReport, compare: qualification.Compare,
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
	var approval evidence.SHA256
	if *approvedBoundary != "" {
		parsed, err := evidence.ParseSHA256(*approvedBoundary)
		if err != nil {
			return writeCommandError(stderr, 2, "invalid boundary approval: %v\n", err)
		}
		approval = parsed
	}
	baseline, err := dependencies.open(*baselinePath)
	if err != nil {
		if qualification.IsInvalidSuiteReport(err) {
			return writeCommandError(stderr, 2, "open baseline support report: %v\n", err)
		}
		return writeCommandError(stderr, 3, "open baseline support report: %v\n", err)
	}
	candidate, err := dependencies.open(*candidatePath)
	if err != nil {
		if qualification.IsInvalidSuiteReport(err) {
			return writeCommandError(stderr, 2, "open candidate support report: %v\n", err)
		}
		return writeCommandError(stderr, 3, "open candidate support report: %v\n", err)
	}
	result, err := dependencies.compare(qualification.ComparisonInput{Baseline: baseline, Candidate: candidate, ApprovedBoundaryDiff: approval})
	if err != nil {
		return writeCommandError(stderr, 2, "compare support reports: %v\n", err)
	}
	if *format == "json" {
		encoded, encodeErr := evidence.CanonicalJSON(result)
		if encodeErr != nil {
			return writeCommandError(stderr, 3, "encode support comparison: %v\n", encodeErr)
		}
		if _, err := fmt.Fprintf(stdout, "%s\n", encoded); err != nil {
			return 3
		}
	} else if _, err := fmt.Fprint(stdout, qualification.FormatComparisonText(result)); err != nil {
		return 3
	}
	if result.Classification == qualification.ComparisonIncomparable {
		return 2
	}
	if result.ReviewRequired {
		return 1
	}
	return 0
}
