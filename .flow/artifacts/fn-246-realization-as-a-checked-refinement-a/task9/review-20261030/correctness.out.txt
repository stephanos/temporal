**Scope read:** task .9 spec, the nine source hunks in the diff (artifact logs skipped except summary and the consumer red log), current `replay.go`, `runner.go`, `session.go`, `publish.go` Load, all `explainedSteps` callers in `tests/`, and the other Run consumers (canary, reconcile test).

## Findings

**P3 — Confidence 100 — introduced — `model/umpire/hosts/go/runner/modes_test.go:161`** — R-IDs: []
Problem: the alteration uses `ProtocolVersion + 1` but the expected reason string is hardcoded `"protocol version 3"`. Correct today; the next format bump silently breaks this row rather than tracking the constant.
Suggestion: build the expected string with `fmt.Sprintf("protocol version %d", ProtocolVersion+1)`.

**P3 — Confidence 75 — introduced — `model/umpire/hosts/go/runner/prefix_test.go:65-92`** — R-IDs: [R1]
Problem: the three mutants (latest-only, wrong-earlier-prefix, concatenated deltas) all fail replay because any edit to a recorded response breaks `proto.Equal(reproduced, recorded)`. They would fail identically against a guest that emitted latest-only witnesses, so they prove replay exactness, not the whole-prefix semantics. The semantic claim rests on the positive `expected` map at lines 19-28, the question-mode checks at prefixes 7/8, and the Rust test. That coverage is sufficient for the acceptance bullet; this is a note on what the mutants actually discriminate.
Suggestion: none required. If you want the mutants to discriminate semantics, assert the refusal event index equals the mutated event (10 or 6) so a wrong-event rejection cannot mask a passing replay.

## Pre-existing issues (not blocking this verdict)

- [P3, confidence 75, introduced=false] `tests/testpilot_model_slice_umpire_test.go:161` — `explanation.GetOutcomes()[i]` panics on a recording with fewer outcomes than steps. The rewritten helper retains the base's unchecked index; the guest always emits equal lengths (`prefix_test.go:27` asserts it), so only a hand-forged Run triggers it.
- [P3, confidence 75, introduced=false] `tests/testpilot_model_slice_umpire_test.go:464` — `explainedSteps(run)[0]` panics if the final witness is empty. Untouched line; the live row passed in the checkpoint evidence.

## Verified correct

- **Admission before Driver effects:** `admit` runs in `replayRun` (after Host instantiation, before any Model call), in `question` before `newSession`, and `reduce` goes through `replayRun` before `execute`. The admission test confirms Driver factory call count stays at the one explore call across all three modes, and the recording bytes are unchanged.
- **Contract gate:** `reset.GetResponse().GetProofContract()` on a nil Response yields UNSPECIFIED, so "missing reset response" is rejected by the same branch. A refused Reset at event 0 skips the contract check by design and is reproduced exactly by `replay`, matching the README statement.
- **Guest mismatch without hash mismatch:** a guest with another contract and the same hash is impossible; a different guest fails the hash first, and if it somehow passed, event 0's reproduced Reset response differs, refusing before any subsequent call.
- **Legacy normalization removal:** no other caller of `legacyRefusalVerdict`; the renamed test now requires every non-current reason to refuse. `worker_test.go:18` still builds a version-1 literal but only for wire-tag vectors, never replayed. No committed `.run` fixtures exist to go stale.
- **Consumer migration:** `explainedSteps` now returns the last complete witness. The in-file unit test covers the repeated-unchanged-observation case (the fn-256.1 failure shape, red log confirms the old concatenation fails 4-vs-6), an earlier partial witness, genuinely different final witnesses, and actual repeated edges. No other consumer in `tools/canary`, `common/testing/testpilot`, or the runner's non-test code concatenates steps across judgments. The `CancelAfterCompletion` and `RetryThenComplete` subtests use ordered slicing within one witness, which is valid under whole-prefix.
- **Immutability:** `prefix_test.go` marshals the Run before replay, publication, load, question and mutant replays and compares bytes afterward; the Rust test re-encodes every prior report after each later judgment.

## Requirements coverage

| R-ID | Status | Evidence |
|------|--------|----------|
| R1 | partial | Current-contract/hash admission, exact replay from Reset, immutable prior reports, Current-after-nonempty-history and callback ordering covered by `prefix_test.go`, `admission_test.go`, `prefix.rs`. Changed-canonical-past/frontier controls explicitly owned by .10 and .6 per the task; not claimed here. |
| R5 | deferred | Nothing in this diff touches the full protocol Model, TLC or native refinement; the task's own acceptance list does not cover R5 and the spec routes it through .13/.15. |

Unaddressed R-IDs: []

Suppressed findings: 0
Classification counts: 2 introduced, 2 pre_existing.

```json
{"classification_counts":{"introduced":2,"pre_existing":2},"unaddressed":[]}
```

<verdict>SHIP</verdict>