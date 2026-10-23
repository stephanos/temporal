package runner

import (
	"fmt"
	"io"
	"strings"
)

// progress writes one line per proposal and per finished Run. It prints only the mode, counters,
// seeds and verdict names: never payloads, reasons or paths, which may carry evidence or
// credentials.
type progress struct {
	out       io.Writer
	mode      Mode
	iteration int
	seed      uint64
}

func (p progress) proposal(event int, request uint64) {
	p.line(fmt.Sprintf("event=%d request=%d", event, request))
}

func (p progress) finished(run *Run) {
	verdict := run.GetVerdict()
	var outcome string
	switch {
	case verdict.GetUnexplained() != nil:
		outcome = fmt.Sprintf("verdict=unexplained at=%d", verdict.GetUnexplained().GetEvent())
	case verdict.GetRefused() != nil:
		outcome = fmt.Sprintf("verdict=refused at=%d", verdict.GetRefused().GetEvent())
	default:
		termination := strings.TrimPrefix(verdict.GetExplained().GetTermination().String(), "TERMINATION_")
		outcome = "verdict=explained termination=" + strings.ToLower(termination)
	}
	p.line(fmt.Sprintf("events=%d %s", len(run.GetEvents()), outcome))
}

func (p progress) line(detail string) {
	if p.out == nil {
		return
	}
	// Progress is best effort; a failed write must not change a Run.
	_, _ = fmt.Fprintf(p.out, "umpire: mode=%s iteration=%d seed=%d %s\n", p.mode, p.iteration, p.seed, detail)
}
