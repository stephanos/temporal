---
satisfies: [R6, R7, R9]
---
# fn-119-show-one-go-sdk-workflow-driven-end-to.6 Write the walkthrough, the one-command entry point and the fn-119 done summary

## Description
The newcomer-facing page and the single command, verified by a fresh reader, plus closing gates and the findings summary.

**Size:** M
**Files:** `model/examples/activityworkflow/WALKTHROUGH.md` (Model lines -> Umpire IR excerpt -> lowered Case -> Run workflow history -> Verdict, plus the faulty variant's counterexample and violated Verdict, each with the command that produces it); a Makefile target such as `umpire-example` (one command); `model/README.md` start-here link (intro after :16 and the "Where things are" table at :286); `.flow/tmp/fn119-6/**`.
**Touches:** [model/examples/**, Makefile, model/README.md, tests/testpilot_generated_test.go, tests/testcore/testpilot/**, .flow/tmp/fn119-6/**]

### Approach
- Add a generic per-Case log line to `TestTestpilotGeneratedCases` (Case file, Verdict status, assessment), naming no example, once per Case rather than per binding/round/Nexus switch; the Makefile target narrows the run to the example's Cases with a generic selector (e.g. by IR file name), not a Go constant.
- The command lifts, lowers and runs the example's Cases against the in-process cluster and prints per Query the Case, the Verdict and the assessment. History shown in the page comes from a real Run (`UMPIRE_EXPLORATION_DIR` + `tools/umpire/explore/trace.go:35` can render it).
- State the honest scope: the workflow is the Driver's interpreter; testing a user-written workflow function is out of scope.
- R6 check: give a fresh subagent only the page and the commands; it must reproduce the Run and say what was authored vs generated. Fix the page for whatever it cannot do; record the transcript path.
- Done summary: the command's wall-clock time, lines of Scala authored, lines of Go added for Driver primitives (tasks 1-3), and every R9 finding from `.flow/tmp/fn119-3/findings.md` with what was done.
- Run model gate, lint-model, Umpire/Testpilot Go tests, lint-code-fast and live tests once.

### Investigation targets
**Required:**
- `model/README.md:1-30,160-290`
- `tools/umpire/explore/trace.go`
**Optional:**
- `Makefile:495-545` - existing umpire-replay/live targets

### Quick commands
```bash
make umpire-example
make umpire-check-model && make lint-code-fast && make umpire-check-live-tests
```
## Acceptance
- [ ] The walkthrough follows one Query from Scala to Verdict, names each producing command, shows the faulty counterexample and violated Verdict, and is linked from model/README.md.
- [ ] A fresh subagent reproduces the Run from the page alone and states what was authored vs generated; gaps were fixed in the page.
- [ ] One documented command runs the example; done summary gives its wall time, Scala lines authored, Go lines added and every R9 finding with its disposition.
- [ ] Full gates and live tests pass.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
