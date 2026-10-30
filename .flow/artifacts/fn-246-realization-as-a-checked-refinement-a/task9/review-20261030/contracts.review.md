**Review scope.** Spec task fn-246.9 against diff 902d2d2..ed931f6. I read the full source hunks (runner admission/replay, protocol version bump, new prefix and admission tests, Rust latest-witness test, consumer helper rewrite, both READMEs) and cross-checked them against current `replay.go`, `session.go`, `runner.go`, `host.go` and the evidence logs under `task9/current-checkpoint`. No shell was available, so verification is by trace plus the committed logs, which show the three new/renamed Go tests and the new Rust test passing (generic-go-tests.log:331-464, runtime-tests.log:99).

## Contract checks (docs/tests vs code)

- **Admission before Driver effects.** `replay.go:15-31` checks hash, version 2, verdict, nonempty events, then the recorded Reset's `PROOF_CONTRACT_WHOLE_PREFIX_V1`. Replay never constructs a Driver; reduce goes through `replayRun` before `execute`; question admits before `newSession`. `TestCurrentRecordingAdmission` confirms the Driver factory count stays at one and the recording is not mutated. Matches README claims.
- **Refused Reset stays an exact recorded call.** `admit` skips the contract check only when event 0 is not a made Reset; a non-Reset event 0 is then caught by `check` ("a Run begins with its one Reset"). Consistent with the README sentence added at line 123.
- **No contract probe on replay.** Replay reproduces the recorded Reset via `invoke`; `host.go:384-389` refuses a non-whole-prefix contract, which then differs from the recording. Matches "it adds no contract probe".
- **Legacy refusal normalization removed.** `legacyRefusalVerdict` deleted; `TestRefusalReplayRequiresTheCurrentRecordedReason` now expects the "legacy" reason to be refused. Matches README/proto README.
- **Whole-prefix witnesses, immutable earlier reports.** `prefix_test.go` script (Current=20 after callback, repeated Current, then final) yields the expected cumulative witnesses at events 2/4/6/7/9/10, question prefixes 7/8 reproduce them, and all three mutants (latest-only, wrong earlier prefix, concatenated deltas) are refused on replay. Marshalled bytes compared before/after. Rust test checks repeated Current and callback/proposal ordering with byte-stable prior reports. Matches acceptance bullet 3 and README text.
- **Consumer migration.** `explainedSteps` now returns the last Explained witness; the removed dedup of repeated `Client::Cancel Rejected(NotFound)` is replaced by a test asserting repeated edges are kept. Red/green overlay evidence shows the old concatenation yields 6 edges vs 4. Remaining callers (`require.Empty` on refused runs, `[0]` index, `slices.Index`) are consistent with a complete final witness; the live slice passed 59 rows.

## Findings

- **Severity**: P3
- **Confidence**: 100
- **Classification**: introduced
- **File:Line**: `model/umpire/hosts/go/runner/modes_test.go:161`
- **R-IDs**: []
- **Problem**: The alteration uses `ProtocolVersion + 1` but the expected reason string is hard-coded `"protocol version 3"`. The two halves of the test disagree on where the version comes from; the next bump silently breaks this row for the wrong reason.
- **Suggestion**: Build the expected string with `fmt.Sprintf("protocol version %d", ProtocolVersion+1)`, or hard-code both sides.

FYI (not findings):
- `tests/testpilot_model_slice_umpire_test.go:162` indexes `Outcomes[i]` by the Steps index; an Explanation with fewer outcomes than steps panics the test rather than failing it. Same shape as the prior code; the Model guarantees equal lengths and replay refuses mismatched recordings before this helper sees them.
- `replayRun` instantiates the Host (`newSession`) before `admit`, while `question` admits first. No Driver effect either way, so the documented contract holds; only a wasted guest instantiation on an inadmissible record.
- `model/umpire/hosts/go/runner/worker_test.go:18` still builds a `ProtocolVersion: 1` fixture against the `run-wire.txt` vector named "legacy". That is a wire-encoding test unaffected by admission; untouched by this diff.

## Requirements coverage

| R-ID | Status | Evidence |
|------|--------|----------|
| R1 | partial | Current reports replay from Reset against all prior observations (`prefix_test.go`, `prefix.rs`); earlier reports immutable (byte comparisons); guest/Host/Run contract and exact replay agree; malformed/incompatible inputs refuse (`admission_test.go`, `TestReplayRejectsAlteredRuns`); current-contract mismatch controls pass. Frontier/Reach/changed-canonical-past portions are explicitly owned by .10 and .6 per the task and spec. |
| R5 | partial | This task supplies the current-format recording/presentation R5's reporting relies on; the Model/TLC/native-refinement proof is owned by .13/.15 and later tasks, not this change. |

Unaddressed R-IDs: []

Classification counts: 1 introduced, 0 pre_existing.

```json
{"classification_counts":{"introduced":1,"pre_existing":0},"unaddressed":[]}
```

<verdict>SHIP</verdict>