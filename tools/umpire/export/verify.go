package export

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"go.temporal.io/server/tools/umpire/check"
	"go.temporal.io/server/tools/umpire/interp"
)

// QuintVerdict is what `quint verify` found of one invariant: no violation in any run within its
// bound, or a counterexample as an ITF trace.
type QuintVerdict struct {
	Violated bool
	Trace    []byte
}

// verifierPort is the port the Apalache server of these checks listens on. Quint starts a server
// where none answers and leaves it running; one on a port of its own can be stopped without touching
// a server someone else started.
const verifierPort = "38822"

// StopVerifier stops the Apalache server the checks left running, if any.
func StopVerifier() {
	// pkill exits with an error where no process matches, which is the usual case.
	_ = exec.Command("pkill", "-f", "apalache.jar server --port="+verifierPort).Run()
}

// VerifyTool is the Quint command line where its Apalache-backed `verify` can run: Apalache is a JVM
// program, so it needs a working `java` on the path.
func VerifyTool() (Tool, bool) {
	t, ok := QuintTool()
	if !ok {
		return t, false
	}
	t.Name = "quint verify"
	return t, exec.Command("java", "-version").Run() == nil
}

// RunQuintVerify has Apalache check one monitor's invariant of a check module on all runs of at most
// the check's steps: bounded model checking, which finds a violation within the bound if there is one
// and says nothing of longer runs.
func RunQuintVerify(ctx context.Context, t Tool, c *QuintCheck, monitor int, dir string) (QuintVerdict, error) {
	source := filepath.Join(dir, c.Module+".qnt")
	if err := os.WriteFile(source, []byte(c.Text), 0o644); err != nil {
		return QuintVerdict{}, err
	}
	trace := filepath.Join(dir, fmt.Sprintf("%s.inv_m%d.itf.json", c.Module, monitor))
	out, err := t.run(ctx, dir, "verify", source, "--main", c.Module, "--invariant", fmt.Sprintf("inv_m%d", monitor),
		"--max-steps", strconv.Itoa(c.Steps), "--out-itf", trace, "--verbosity", "1", "--server-endpoint", "localhost:"+verifierPort)
	if err == nil {
		// A run that exits cleanly found no violation; one that says so in other words is not trusted.
		if !strings.Contains(string(out), "No violation found") {
			return QuintVerdict{}, fmt.Errorf("quint verify exited cleanly without saying it found no violation:\n%s", out)
		}
		return QuintVerdict{}, nil
	}
	counterexample, readErr := os.ReadFile(trace)
	if readErr != nil || !strings.Contains(string(out), "violation") {
		return QuintVerdict{}, err
	}
	return QuintVerdict{Violated: true, Trace: counterexample}, nil
}

// NotVerified is the receipt of a check the model checker could not be run on, with the tool's own
// reason: nothing was checked, and it is no agreement.
func (c *QuintCheck) NotVerified(monitor int, cause error) Receipt {
	reason := cause.Error()
	if i := strings.LastIndex(reason, "error: "); i >= 0 {
		reason = strings.TrimSpace(reason[i:])
	}
	return Receipt{Backend: quintBackend, Model: c.from.from.Name, Claim: CheckerCoverage, Subject: c.Machine + "." + c.Monitors[monitor], Kind: NotRun,
		Explanation: "`quint verify` did not check the module: " + reason}
}

// witness reads a check module's counterexample back as a trace of the machine: the first state's
// machine state, and the steps its last state says were taken.
func (c *QuintCheck) witness(table *interp.Table, itf []byte) (*check.Trace, error) {
	var trace struct {
		States []map[string]any `json:"states"`
	}
	if err := json.Unmarshal(itf, &trace); err != nil {
		return nil, fmt.Errorf("the counterexample is no ITF trace: %w", err)
	}
	if len(trace.States) == 0 {
		return nil, errors.New("the counterexample holds no state")
	}
	r := c.from.reader(c.index)
	start, err := r.state(trace.States[0]["st"])
	if err != nil {
		return nil, fmt.Errorf("the counterexample's first state: %w", err)
	}
	out := &check.Trace{Initial: table.StateAtom(start)}
	taken, err := list(trace.States[len(trace.States)-1]["hist"])
	if err != nil {
		return nil, fmt.Errorf("the counterexample's steps: %w", err)
	}
	for _, raw := range taken {
		entry, err := record(raw, "a step of the counterexample")
		if err != nil {
			return nil, err
		}
		class, err := r.class(entry["cls"])
		if err != nil {
			return nil, err
		}
		res, err := r.result(entry["step"])
		if err != nil {
			return nil, err
		}
		step := check.TraceStep{Action: table.ActionAtom(class), Outcome: table.OutcomeAtom(res.Outcome), State: table.StateAtom(res.State)}
		for _, f := range res.Facts {
			step.Facts = append(step.Facts, table.FactAtom(f))
		}
		out.Steps = append(out.Steps, step)
	}
	return out, nil
}

// QuintVerified compares what Apalache found of one monitor of a check module with Go's product of
// the machine and its monitors, and replays Apalache's counterexample through Go: through a fresh
// interpretation's table and monitor, and through the reader's checker over the path's classes. A
// counterexample that does not replay is an error, whatever Go's own verdict.
func (s *Slice) QuintVerified(c *QuintCheck, monitor int, v QuintVerdict) Receipt {
	name := c.Monitors[monitor]
	r := Receipt{Backend: quintBackend, Model: s.Name, Claim: CheckerCoverage, Subject: c.Machine + "." + name, Kind: Agreed}
	mm := s.machines[c.Machine]
	p, err := s.product(mm)
	if err != nil {
		r.Kind, r.Explanation = Disagreed, err.Error()
		return r
	}
	r.ProductStates = len(p.States)
	at, violated := p.violatedAt[name]
	bound := fmt.Sprintf("Apalache checked every run of %s of at most %d steps, which reach every one of the %d states of Go's product", c.Machine, c.Steps, r.ProductStates)
	if !v.Violated {
		r.Explanation = bound + fmt.Sprintf(", and found no violation of %s; Go finds none either", name)
		if violated {
			r.Kind = Disagreed
			r.Explanation = bound + fmt.Sprintf(", and found no violation of %s", name)
			r.Differences = []string{fmt.Sprintf("Go finds %s violated after %d steps", name, at)}
		}
		return r
	}
	rejected := func(err error) Receipt {
		r.Kind = WitnessRejected
		r.Explanation = fmt.Sprintf("Apalache's counterexample of %s did not replay through Go: %v", name, err)
		return r
	}
	trace, err := c.witness(mm.Table, v.Trace)
	if err != nil {
		return rejected(err)
	}
	r.Violated, r.Witnesses = []string{name}, []Witness{{Monitor: name, Trace: trace}}
	if err := s.Replay(c.Machine, name, trace); err != nil {
		return rejected(err)
	}
	confirmable := []Receipt{{Claim: MonitorAgreement, Subject: c.Machine, Kind: Agreed, ProductStates: r.ProductStates, Violated: r.Violated, Witnesses: r.Witnesses}}
	if err := s.confirm(confirmable, false); err != nil {
		return rejected(err)
	}
	if confirmable[0].Kind != Agreed {
		r.Kind, r.Explanation, r.Differences = confirmable[0].Kind, confirmable[0].Explanation, confirmable[0].Differences
		return r
	}
	r.Explanation = fmt.Sprintf("Apalache found a run of %d steps that violates %s, which replays through Go's table and monitor and through goir's checker; Go's shortest is %d steps",
		len(trace.Steps), name, at)
	if !violated {
		r.Kind, r.Differences = Disagreed, []string{fmt.Sprintf("Go finds %s violated on no path", name)}
	}
	return r
}
