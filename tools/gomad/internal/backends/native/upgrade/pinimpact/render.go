package pinimpact

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"unicode/utf8"
)

// Encode returns the report's deterministic JSON with a trailing newline. The
// report holds no host paths, so equal inputs encode to equal bytes.
func Encode(report Report) ([]byte, error) {
	if err := validateReportStrings(report); err != nil {
		return nil, fmt.Errorf("encode pin impact report: %w", err)
	}
	encoded, err := json.Marshal(report)
	if err != nil {
		return nil, fmt.Errorf("encode pin impact report: %w", err)
	}
	return append(encoded, '\n'), nil
}

func validateReportStrings(report Report) error {
	values := []string{
		report.Schema, report.PinnedGoVersion,
		report.Baseline.GoModSHA256, report.Baseline.GoSumSHA256, report.Baseline.GoDirective,
		report.Candidate.GoModSHA256, report.Candidate.GoSumSHA256, report.Candidate.GoDirective,
	}
	values = append(values, report.Platforms...)
	for _, summary := range report.Classes {
		values = append(values, string(summary.Class))
	}
	for _, pin := range report.Pins {
		values = append(values,
			string(pin.Class), string(pin.Status), pin.ID, pin.Module, pin.PinnedVersion, pin.PinnedSum,
			pin.CandidateVersion, pin.CandidateSum, pin.Pack, pin.PackSHA256, pin.ImportPath,
			pin.SourceSetSHA256, pin.Reason,
		)
		values = append(values, pin.Platforms...)
	}
	for _, value := range values {
		if !utf8.ValidString(value) {
			return errors.New("JSON string is not valid UTF-8")
		}
	}
	return nil
}

// Render writes the report for a person: one line per class, then one line
// per invalidated, unknown, or stale pin.
func Render(writer io.Writer, report Report) error {
	invalidated, unknown, stale := 0, 0, 0
	for _, summary := range report.Classes {
		invalidated += summary.Invalidated
		unknown += summary.Unknown
		stale += summary.Stale
	}
	var builder strings.Builder
	fmt.Fprintf(&builder, "gomad pin impact (pinned %s; platforms %s): %d invalidated, %d unknown, %d stale\n",
		report.PinnedGoVersion, strings.Join(report.Platforms, ", "), invalidated, unknown, stale)
	for _, summary := range report.Classes {
		fmt.Fprintf(&builder, "  %-25s %4d pins: %d invalidated, %d unknown, %d stale, %d unaffected, %d not selected\n",
			summary.Class, summary.Total, summary.Invalidated, summary.Unknown, summary.Stale, summary.Unaffected, summary.NotSelected)
	}
	for _, pin := range report.Pins {
		fmt.Fprintf(&builder, "%s %s %s", pin.Status, pin.Class, pin.ID)
		if pin.Module != "" && pin.Class == ClassPackRule {
			fmt.Fprintf(&builder, " (%s@%s)", pin.Module, pin.PinnedVersion)
		}
		if pin.Reason != "" {
			fmt.Fprintf(&builder, ": %s", pin.Reason)
		}
		builder.WriteByte('\n')
	}
	_, err := io.WriteString(writer, builder.String())
	return err
}
