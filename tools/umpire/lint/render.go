package lint

import (
	"fmt"
	"io"
	"text/tabwriter"
)

// WriteFindings writes one line per finding of a verdict: its owner, kind, message and position, an
// accepted one with its reason, then each stale acceptance.
func WriteFindings(w io.Writer, file string, v Verdict) error {
	if _, err := fmt.Fprintf(w, "lint %s\n", file); err != nil {
		return err
	}
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	for _, f := range v.Unaccepted {
		if _, err := fmt.Fprintf(tw, "  %s\t%s\t%s\t%s\n", f.Owner, f.Kind, f.Message, f.Position); err != nil {
			return err
		}
	}
	for _, j := range v.Accepted {
		if _, err := fmt.Fprintf(tw, "  %s\t%s\t%s\t%s\taccepted: %s\n", j.Owner, j.Kind, j.Message, j.Position, j.Because); err != nil {
			return err
		}
	}
	if err := tw.Flush(); err != nil {
		return err
	}
	for _, s := range v.Stale {
		if _, err := fmt.Fprintf(w, "  stale acceptance: %s\n", s); err != nil {
			return err
		}
	}
	return nil
}
