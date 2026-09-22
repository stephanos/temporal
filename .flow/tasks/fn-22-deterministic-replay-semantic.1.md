---
satisfies: [R1, R2, R9]
---
# fn-22-deterministic-replay-semantic.1 Admit the replay subject, derive the Contract-relative violation key, and export offline semantic replay

## Description
Define `Subject` admission in `tools/umpire/replay`: one canonical Case (decoded and re-packed byte-identical), its `testpilot.DriverIdentity`, and one closed Run with its Verdict in the admissible violated form (`STOPPED_BY_MONITOR` or `COMPLETED`, cleanup `SUCCEEDED`, Verdict `VIOLATED`); crossed ids (Case, Program, Run), an `INCOMPLETE` or unclosed Run, a non-violated Verdict, a supporting sequence naming no event and a duplicate sequence reject before any target effect. Add `binding.Campaign.Prepare(ctx, identity, source)` (catalog, `DeriveProfile`, `testpilot.Prepare`; no SDK client, no Driver), build `Bind` on it and expose the prepared Case on `Bound`; admission prepares the subject under the Profile the deployment flags derive and decides `stale` by comparing `DriverIdentity`, before any Driver. Export offline semantic replay on the facade, `PreparedCase.Evaluate(ctx, run) (*Verdict, error)` over the Case Runtime's own evaluator, covered by the protocol and facade tests, and have `Subject.Replay` compare the recorded Verdict with the re-evaluated one. Derive the `ViolationKey` in Definition IDs through `provenance.local_names`: the violated rules, each with its terminal state and the evidence kinds of the event at the shortest Run prefix whose offline Verdict names the rule violated; pin that per-Run transport values, instruction ids, the Verdict's accumulated support and the Case identity never enter it.

### Approach
- Admission is a pure function of the three inputs plus the prepared Case; `Open` creates its gRPC client lazily, so `Prepare` touches no deployment.
- The key's terminal step is found by evaluating prefixes of the Run's events offline, shortest first, through the facade; the evidence kinds come from that event's observation ids resolved to Definition IDs.
- A functional caller fixture and its recorded live Run (checked in as test data from the live suite) pin the key; renaming the fixture's local names by hand in a test copy must not move it.

### Quick commands
`go test -count=1 -tags test_dep ./tools/umpire/replay/ ./tools/umpire/binding/ ./common/testing/testpilot/`

**Size:** M
**Files:** `tools/umpire/replay/subject.go`, `tools/umpire/replay/subject_test.go`, `tools/umpire/replay/key.go`, `tools/umpire/replay/key_test.go`, `tools/umpire/binding/binding.go`, `tools/umpire/binding/binding_test.go`, `common/testing/testpilot/prepared_case.go`, `common/testing/testpilot/protocol_test.go`, `common/testing/testpilot/facade_external_test.go`
**Touches:** `tools/umpire/replay/**`, `tools/umpire/binding/**`, `common/testing/testpilot/prepared_case.go`, `common/testing/testpilot/*_test.go`

### Re-plan note (2026-09-22)
Rewritten on fn-85, fn-86, fn-87 and fn-33 after the first plan's MAJOR_RETHINK; revised after plan review round one; see the spec's **Re-plan** and **Plan review** sections. Start only after the spec's fresh plan review.
## Acceptance
- [ ] Crossed, stale, noncanonical, incomplete, unsupported, non-violated and duplicate inputs fail before target effects, each with its own reason, and `stale` is decided by the prepared Case's identity before any Driver opens.
- [ ] Per-Run identities, sequences, times, instruction ids and the Verdict's accumulated support do not change the key; a different violated rule set, terminal state or terminal-step evidence does; a Case-local renaming does not; the Case identity is reported beside the key and is not in it.
- [ ] `PreparedCase.Evaluate` reproduces a recorded Run's Verdict offline and is the only semantic-replay path; `binding.Campaign.Prepare` dials nothing; no bundle, digest, trust store or reader is introduced.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
