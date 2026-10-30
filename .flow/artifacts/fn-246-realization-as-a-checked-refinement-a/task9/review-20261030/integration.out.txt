Review complete. I read the task spec and the epic acceptance criteria, the full source hunks of the diff, the current state of `replay.go`, `session.go`, `runner.go`, the three new or changed test files, the testpilot consumer, and every other reader of `ProtocolVersion`, `runner.Load` and the explanation steps across `tools/`, `common/testing/testpilot/` and `tests/`.

## Summary

The change is a removal plus tests: the legacy refusal-reason normalization is gone, the recording format is bumped to version 2, and admission now requires the recorded Reset to carry the whole-prefix contract. The one consumer that concatenated witnesses as deltas now reads the latest complete witness. Integration with unchanged code holds up:

- **Admission before effects.** `admit` runs before `s.replay` in replay, before `newSession` in question, and inside the pre-flight `replayRun` in reduce, so no path reaches `newDriver` or `execute` on a rejected record. The live Reset contract check in `host.go:384` is unchanged and is what `reproduce` hits when replay re-invokes the recorded Reset. No probe is added.
- **Refused Reset records.** When event 0 is a Refusal of Reset, `admit` skips the contract check and replay reproduces the refusal exactly, which matches the README's stated contract.
- **Other producers of Runs.** The canary's scripted Run uses the `runner.ProtocolVersion` constant and is never replayed. The worker wire-vector test hard-codes version 1 but only checks encoding. The testpilot canary and reconcile tests replay live-produced Runs. No committed Run fixture outside historical `.flow/artifacts` carries version 1.
- **Consumer migration.** Every use of `explainedSteps` in `tests/` still holds under the latest-witness semantics, including the `require.Empty` scope-control checks and the index at `tests/testpilot_model_slice_umpire_test.go:464`. Only the deleted `judgmentPath` had callers, and none remain.

## Findings

- **Severity**: P3
- **Confidence**: 100
- **Classification**: introduced
- **File:Line**: `model/umpire/protocol/runtime/tests/main/prefix.rs:174`
- **R-IDs**: [R1]
- **Problem**: The "without changing prior reports" loop re-encodes owned clones stored in `reports` and compares them to their own earlier bytes. Nothing in the runtime holds a reference to those clones, so the assertion cannot fail and the immutability claim in the test name is unverified on the Rust side. The Go test in `prefix_test.go` does cover it meaningfully by marshaling the recorded Run before and after replay, question and publication.
- **Suggestion**: Drop the loop, or make it observe something the runtime can change, for example re-judging an earlier observation after the later ones and asserting its witness still equals the retained report.

- **Severity**: P3
- **Confidence**: 75
- **Classification**: introduced
- **File:Line**: `model/umpire/hosts/go/runner/admission_test.go:141`
- **R-IDs**: []
- **Problem**: The reason assertion is selected by substring-matching the subtest name. Adding a case named, say, "contract-free seed" would silently pick the wrong expectation. This is test-only and cannot produce a wrong product result.
- **Suggestion**: Add a `wantReason string` field to the table and assert it when non-empty.

## Pre-existing issues (not blocking this verdict)

- [P3, confidence 75, introduced=false] `tests/testpilot_model_slice_umpire_test.go:161` — `explanation.GetOutcomes()[i]` panics instead of failing the assertion when a witness has fewer outcomes than steps. The base code had the same unguarded index; the rewrite preserved it.

## FYI

- `TestUmpireWholePrefixJudgmentPath` needs no cluster but lives in a `test_dep && integration` file, so it only runs under the integration tag. Fine if that is the intended gate.
- The task frontmatter lists `satisfies: [R1, R5]`, but nothing in this diff touches R5's full protocol Model or TLC agreement. That looks like a tagging overclaim rather than a defect.

## Requirements coverage

| R-ID | Status | Evidence |
|------|--------|----------|
| R1 | partial | Whole-prefix replay from Reset, immutable earlier reports, current-contract and hash admission, malformed and incompatible refusal, publication/retry and callback ordering are covered by `prefix_test.go`, `admission_test.go`, `prefix.rs` and the migrated live consumer. Frontier, changed-canonical-past and fabricated-state controls are explicitly owned by .10 and the integrated .6 gate per the task file. |
| R5 | deferred | Owned by tasks .13/.15 and the TLC gate. No file in this change relates to it. |

Unaddressed R-IDs: []

Suppressed findings: 0
Classification counts: 2 introduced, 1 pre_existing.

```json
{"classification_counts":{"introduced":2,"pre_existing":1},"unaddressed":[]}
```

<verdict>SHIP</verdict>