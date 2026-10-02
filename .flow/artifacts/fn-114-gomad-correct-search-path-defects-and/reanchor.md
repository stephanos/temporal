# Search-path findings re-anchor

Re-anchored on 2026-10-02 at HEAD `1d7272e654f268f9a45f3fe965918fe2522827c6`
on darwin/arm64. The working tree already contains completed, uncommitted fn-112
changes. [Source bindings](task-1/source-bindings-before.json) record the SHA-256
of each production file read, the retained references, and the protected retention
test; HEAD alone does not identify this tree. This task adds a characterization
test and evidence, with no production correction or toolchain rebuild.

The existing patched toolchain key is
`6b775117cc6b13d04c2d00926818e102edb540f8c74ece2794b5e6f3cd2c19ee`.
Historical reports below have their own build keys and are not measurements of
this toolchain. Linux execution remains unverified.

| ID | Current source and line | Verdict | Evidence read and remaining limit |
| --- | --- | --- | --- |
| C1 | `tools/gomad3/runner/internal/corpus/model.go:37`, `:98`, `:121`; `tools/gomad3/runner/guidance.go:26`; `tools/gomad3/runner/runner.go:1328` | confirmed | `Identity` and `targetProjection` omit environment and tick policy. `openGuidance` builds identity without `baseEnvironment`, although it saves that environment for later runs. Runner already accepts explicit environment entries and inserts the forward-tick control into them. |
| C2 | `tools/gomad3/toolchain/runtime/overlay/src/runtime/gomad.go:881`, `:886`, `:902`; `tools/gomad3/toolchain/runtime/go1.27.1.patch:338` | confirmed | Identified parents bind a child ordinal and site; other creations hash the process-wide `gomadRuntimeGoroutineOrdinal.Add(1)`. The existing built Go source has `time.AfterFunc` using `goFunc`, which starts the callback with a `go` statement (`src/time/sleep.go:177-182`); timer dispatch calls that function (`src/runtime/time.go:1218`). Schedule-sensitive callback identity still needs task 2's runtime reproduction. |
| C3 | `tools/gomad3/runner/internal/execution/process_unix.go:635`; `tools/gomad3/runner/runner.go:1462`, `:1714`; `tools/gomad3/runner/choice_exploration_campaign.go:250`, `:279`, `:324`; `tools/gomad3/runner/runner_test.go:1099` | changed | The loss symptom reproduces. A typed executor divergence reaches the completion-error branch and becomes `HostError` with reason `target_supervision`, before `processExplorationCompletion` and its runner-domain branch. The entire prefix round remains uncommitted, completed sibling results do not advance the campaign, and no candidate execution records or typed divergence evidence are published. Raw partial captures remain. |
| C4 | `tools/gomad3/target/target.go:828` | confirmed | `validateDeterministicBuildInfo` rejects cgo, race, non-executable build modes, shared linking, and external/plugin linking, but has no coverage-instrumentation check. |
| E1 | `tools/gomad3/runner/seeds.go:27`; `tools/gomad3/runner/internal/corpus/model.go:76`, `:98` | confirmed | Selection reserves `ceil(count/4)` requested seeds and fills up to the rest with prioritized corpus seeds. Entries retain seed and replay status; there is no answered-execution exclusion in selection. Repeating retained executions is a source inference, subject to C1's environment omission, rather than a new campaign measurement. |
| E2 | `tools/gomad3/artifact/publication.go:44`, `:52`; `tools/gomad3/runner/internal/corpus/model.go:33` | confirmed | Publication always adds the full prepared target as the `target` source payload; corpus limits remain 1,024 entries and 1 GiB. The 155–179 MB binary range and roughly six-case estimate are historical assessment figures, while the README reports about 11 GiB for the representative set. No current disk-size measurement was performed. |
| E3 | `tools/gomad3/toolchain/runtime/go1.27.1.patch:689`; `tools/gomad3/toolchain/runtime/overlay/src/runtime/gomad.go:940`; `tools/gomad3/runner/internal/exploration/choice/engine.go:349` | confirmed | The poll-order hook runs before readiness is checked, and expansion treats every branching decision as eligible within bounds. The retained D14 report re-derives 26,865 select-poll of 57,801 decisions. The claim that fewer than two ready cases permits sound reduction remains an inference for tasks 2 and 12 to prove. |
| E4 | `tools/gomad3/toolchain/runtime/overlay/src/runtime/gomad.go:921`; `tools/gomad3/toolchain/runtime/go1.27.1.patch:583`; `.flow/artifacts/fn-105-gomad-follow-ups-deferred-scope/fn105-d21-probe-control.go.txt:17` | changed | Every local run-queue entry becomes an alternative, without a system-goroutine filter. The historical seed-11 control report has 26 branching Runnable decisions and peak goroutines 2. Its source starts no goroutine explicitly, so the peak does not establish the planned premise of two deliberately started user goroutines, or attribute each extra decision to runtime-owned alternatives. Task 13 keeps the explicit two-user fixture and cause check. |
| E5 | `tools/gomad3/runner/internal/exploration/choice/engine.go:349`, `:357` | confirmed | Expansion counts `ordinal+1` from decision zero and omits deeper choices. There is no exploration start ordinal. The 4,562-record boot-only probe remains a historical assessment reference (`.plans/GOMAD_CMP.md:279`), not a newly reproduced measurement. |
| E6 | `tools/gomad3/runner/minimize_operation.go:73`, `:97`, `:179`, `:189`, `:487` | confirmed | The session creates new minimizer state, commits attempts only into its local state, uses a temporary directory, and removes it on close. Publication happens only after the loop; interrupted execution cannot restore that in-memory progress. |

## Evidence-reference corrections

1. E2's binary-size range belongs to [GOMAD_CMP.md](../../../.plans/GOMAD_CMP.md)
   lines 85–86. [README](../../../tools/gomad3/README.md) line 334 supports the
   representative set's approximate total only. The corpus-size estimate is
   approximate: traces and other artifact payloads also consume its cap.
2. C2 has no separate goroutine-identity scheme version constant. Its domains are
   `gomad3-choice-goroutine-root/v1`, `gomad3-choice-goroutine-child/v1`, and
   `gomad3-choice-goroutine-runtime/v1` in the overlay at lines 883, 891, and 903.
   `gomadChoiceWireVersion = 2` belongs to the generated choice-wire contract
   (`gomad_choicewire_generated.go:6`), not a separate identity scheme version.
3. D14 and D21 reports are retained under
   `.flow/artifacts/fn-105-gomad-follow-ups-deferred-scope/`. Their exact paths,
   SHA-256 bindings, build keys, and extracted counters are in
   [retained-counts.json](task-1/retained-counts.json).

| Retained report | Re-derived counters | Historical toolchain |
| --- | --- | --- |
| [D14 seed 11](../fn-105-gomad-follow-ups-deferred-scope/fn105-d14-qualification-seed11.json) | `evidence.choices.runnable` 30,936 + `select_poll` 26,865 = `decisions` 57,801. With `select_result` 28,442, `records` = 86,243. `branching_records` = 57,801. | `8d28bd4486f0b6300e8d25efd4caf8cb6ccbf000e96dbd26b1d8f53bf5f251bc` |
| [D21 control seed 11](../fn-105-gomad-follow-ups-deferred-scope/fn105-d21-qualification-control-choices-11.json) | `runnable` 26 + `select_poll` 0 = `decisions` 26 = `branching_records` 26; `peak_goroutines` = 2. | `c0661e38b4e001c8d86912d4ce9e265eb0f33e5f3298015da27a4c97ac4019b9` |
| [D21 control seed 17](../fn-105-gomad-follow-ups-deferred-scope/fn105-d21-qualification-control-choices-17.json) | `runnable` 29 + `select_poll` 0 = `decisions` 29 = `branching_records` 29; `peak_goroutines` = 2. The count 26 is seed-specific. | `c0661e38b4e001c8d86912d4ce9e265eb0f33e5f3298015da27a4c97ac4019b9` |

These counts are re-derived from retained report counters, not decoded again from
the original trace payloads. The D21 [control source](../fn-105-gomad-follow-ups-deferred-scope/fn105-d21-probe-control.go.txt)
performs collections and reads reporting state without a `go` statement. The
runtime code confirms that queue identities are unfiltered; the reports alone
cannot identify which goroutines formed their alternative sets.

## C3 characterization

`TestRunChoiceExplorationDivergingPrefixDiscardsCompletedRound` runs the real
Explorer with an injected execution seam. Its successful root emits one four-way
decision. The next round has three forced-prefix candidates: one executor call
returns before the divergence, one returns a typed alternative-set divergence,
and one waits for that divergence before returning with an uncancelled context.
Both siblings reach `ExecutionCaptured`; the diverging candidate reaches
`ExecutionExited`. Thus collection finishes before classifying the error and
does not cancel its siblings on divergence.

The campaign then returns `HostError{Reason: "target_supervision"}`, retaining
the typed error in its error chain. Only the root is counted and committed:
attempted 1, succeeded 1, committed rounds 1. The prefix round has neither
`segment.json` nor candidate `executions` records. The campaign is resumable and
unpublished, and returns no candidate artifacts. The staged `round.json` still
identifies the candidates and prefixes, and each raw candidate partial contains
`partial.json`, `stdout.head`, `stderr.head`, and `work`. These are recovery
scaffolding, not committed outcomes or typed divergence evidence.

Task 4 must preserve a forced-prefix candidate's typed executor error through
collection and commit the divergence together with its sibling outcomes. Other
executor errors and runner-domain fallback outcomes keep their host-error
handling. The supplied task-4 annotation corrects this mechanism; task-13's
annotation corrects the historical control premise without changing its intended
two-user fixture or acceptance criteria. No finding is refuted, and no owner
task closes on this evidence alone.

## Verification scope

Host tests use `tools/gomad3/.toolchain/bin/go` with `GOMADSEED`,
`GOMAD3_CHILD_SEED`, and inherited `GOROOT` unset, always with `-tags test_dep`.
The pre-edit choice-exploration baseline passed. The C3 test passed ten
repetitions and focused Runner vet passed. The first full Runner run hit the
existing protected 100-job retention case's 10-second overall deadline at 97
executions; the same case passed in isolation in 5.27 seconds. Its SHA-256 is
unchanged. The original failure is retained in [runner-final.log](task-1/runner-final.log).
The full `go -C tools/gomad3 test -tags test_dep ./runner/...` rerun passed with
exit 0, with the Runner package taking 94.549 seconds and the internal packages
reusing passing cache entries. [Final Runner log](task-1/runner-final-recheck.log)
retains the successful result. Formatting and the task source diff check passed.

Root `make lint-code-fast` failed because its default `main` base is unavailable.
With the existing HEAD used explicitly (`GOLANGCI_LINT_BASE_REV=HEAD`), it failed
at root-module discovery of nested Gomad packages. The linter reports zero
issues but exits with a typechecking/discovery error; this is not a clean lint
result. [Root lint log](task-1/root-lint-head.log) and
[focused vet log](task-1/focused-vet.log) retain those results.

The task-only patch and final file hashes are separate from prior fn-112 changes.
Full host/runtime gates were already run by fn-112 task 4; this test-only task
does not rebuild or requalify the runtime. It makes no Linux claim.
