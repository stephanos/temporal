---
satisfies: [R1, R2, R9]
---
# fn-22-deterministic-replay-semantic.1 Admit the replay subject, derive the Contract-relative violation key, and export offline semantic replay

## Description
Define `Subject` admission in `tools/umpire/replay`: one canonical Case (decoded and re-packed byte-identical), its `testpilot.DriverIdentity`, and one closed Run with its Verdict; crossed ids (Case, Program, Run), an incomplete or unclosed Run, a non-violated Verdict, a supporting sequence naming no event and a duplicate sequence reject before any target effect. Derive the `ViolationKey` (Contract id, violated rules with terminal states, supporting event roles as a sorted set) and pin that per-Run transport values and the Case identity never enter it. Export offline semantic replay on the facade: `PreparedCase.Evaluate(ctx, run) (*Verdict, error)` over the Case Runtime's own evaluator, covered by the protocol tests, and have `Subject.Replay` compare the recorded Verdict with the re-evaluated one.

### Approach
- Admission is a pure function of the three inputs; the Profile identity is compared at rerun time (task .3), where stale is decided.
- The key's supporting roles come from the Run's events by sequence: kind, `entrypoint_id`, `instruction_id`, observation ids; nothing else.
- The facade export is the evaluator `verification.PreparedContract.Evaluate` already prepared per Case; no new package.

### Quick commands
`go test -count=1 -tags test_dep ./tools/umpire/replay/ ./common/testing/testpilot/`

**Size:** M
**Files:** `tools/umpire/replay/subject.go`, `tools/umpire/replay/subject_test.go`, `tools/umpire/replay/key.go`, `tools/umpire/replay/key_test.go`, `common/testing/testpilot/prepared_case.go`, `common/testing/testpilot/protocol_test.go`

### Re-plan note (2026-09-22)
Rewritten on fn-85, fn-86, fn-87 and fn-33 after the first plan's MAJOR_RETHINK; see the spec's **Re-plan** section. Start only after the spec's fresh plan review.
## Acceptance
- [ ] Crossed, stale, noncanonical, incomplete, unsupported, non-violated and duplicate inputs fail before target effects, each with its own reason.
- [ ] Per-Run identities, sequences and times do not change the key; a different Contract, rule set, terminal state or supporting-role set does; the Case identity is reported beside the key and is not in it.
- [ ] `PreparedCase.Evaluate` reproduces a recorded Run's Verdict offline and is the only semantic-replay path; no bundle, digest, trust store or reader is introduced.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
