---
satisfies: [R1, R2, R9]
---
# fn-22-deterministic-replay-semantic.1 Admit the replay subject, derive the Contract-relative violation key, and export offline semantic replay

## Description
Define the *recorded Run* file shape in `tools/umpire/replay` (the closed Run with its Verdict and the `DriverIdentity` it was prepared under) and have `umpire-run --record <path>` write it beside its report. Define `Subject` admission: one canonical Case (decoded and re-packed byte-identical) and one recorded Run in the admissible violated form (`STOPPED_BY_MONITOR` or `COMPLETED`, cleanup `SUCCEEDED`, Verdict `VIOLATED`); crossed ids (Case, Program, Run), an `INCOMPLETE` or unclosed Run, a non-violated Verdict, a supporting sequence naming no event and a duplicate sequence reject before any target effect. Add the free `binding.Prepare(deployment, handlerQueue, identity, source)` (its own method catalog, `DeriveProfile`, `testpilot.Prepare`; no connection, no provisioning, no Driver), build `Campaign.Bind` on it and expose the prepared Case on `Bound`; admission prepares the subject under the Profile the deployment flags derive and decides `stale` against the recorded `DriverIdentity`, all before `binding.Open`. Export offline semantic replay on the facade: `PreparedCase.Evaluate(ctx, run) (*Verdict, *Evaluation, error)` over the Case Runtime's own evaluator, where `Evaluation` names, per violated rule, the sequence of the event whose evidence resolved the obligation and that evidence (a monitor rule's observation ids from its transition trace; a correlated rule's released `CorrelatedEvidence.Kind`), covered by the protocol and facade tests; `Subject.Replay` compares the recorded Verdict with the re-evaluated one. Derive the `ViolationKey` in Definition IDs (Case-local names resolved through `provenance.local_names` where a row exists, taken as-is otherwise; a correlated rule's terminal state is the runtime constant): the violated rules, each with its terminal state and its violating evidence; pin that per-Run transport values, instruction ids, the Verdict's accumulated support and the Case identity never enter it.

### Approach
- Admission is a pure function of the two inputs plus the prepared Case; `binding.Prepare` is what `Bind` calls first, and a test pins that it opens nothing.
- The key is pinned offline on the synthetic Case (`Testpilot.Examples.Synthetic`, `testdata/synthetic-case.json`) driven by a scripted facade Driver, as `conformance_test.go` and `facade_external_test.go` drive Cases, into a violated Run; a correlated Case's key is pinned on the control's recorded live Run in task .3.
- The evaluator's `transitionTrace` already records monitor-rule transitions; the correlated monitor learns to keep, per obligation, the evidence whose release resolved it.

### Quick commands
`go test -count=1 -tags test_dep ./tools/umpire/replay/ ./tools/umpire/binding/ ./tools/umpire/cmd/umpire-run/ ./common/testing/testpilot/...`

**Size:** L
**Files:** `tools/umpire/replay/subject.go`, `tools/umpire/replay/subject_test.go`, `tools/umpire/replay/key.go`, `tools/umpire/replay/key_test.go`, `tools/umpire/replay/recorded.go`, `tools/umpire/binding/binding.go`, `tools/umpire/binding/binding_test.go`, `tools/umpire/cmd/umpire-run/run.go`, `tools/umpire/cmd/umpire-run/run_test.go`, `common/testing/testpilot/prepared_case.go`, `common/testing/testpilot/internal/verification/evaluator.go`, `common/testing/testpilot/internal/verification/correlated.go`, `common/testing/testpilot/protocol_test.go`, `common/testing/testpilot/facade_external_test.go`
**Touches:** `tools/umpire/replay/**`, `tools/umpire/binding/**`, `tools/umpire/cmd/umpire-run/**`, `common/testing/testpilot/prepared_case.go`, `common/testing/testpilot/internal/verification/**`, `common/testing/testpilot/*_test.go`

### Re-plan note (2026-09-22)
Rewritten on fn-85, fn-86, fn-87 and fn-33 after the first plan's MAJOR_RETHINK; revised after plan review rounds one and two; see the spec's **Re-plan** and **Plan review** sections. Start only after the spec's fresh plan review.
## Acceptance
- [ ] Crossed, stale, noncanonical, incomplete, unsupported, non-violated and duplicate inputs fail before `binding.Open`, each with its own reason; `stale` compares the recorded `DriverIdentity` with the prepared Case's, and `binding.Prepare` opens no connection and provisions nothing.
- [ ] Per-Run identities, sequences, times, instruction ids and the Verdict's accumulated support do not change the key; a different violated rule set, terminal state or violating evidence does; a Case-local renaming does not; the Case identity is reported beside the key and is not in it.
- [ ] `PreparedCase.Evaluate` reproduces a recorded Run's Verdict offline and names each violated rule's violating evidence; it is the only semantic-replay path; `umpire-run --record` writes the recorded Run; no bundle, digest, trust store or reader is introduced.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
