# fn-188.5 implementation checkpoint

The Rust stress producer and linked jitter implementation are ready for conductor review. Task .5 remains in progress: no full workload matrix or qualification has run on this checkpoint. Root owns integration, review, the frozen matrix, task .6 findings and Flow completion.

## Implementation

The admitted `stress --spec FILE` command drives fresh copies of the actual Rust binary through existing preparation, standalone execution and scenario publication. The matrix preserves 32 cooperative runtime seeds, active/standby seeds 7/19/42, and unchanged full `./tests` workflow/activity selections at 0/1000003/2000006. CPU, concurrent cohorts, actual process layout and cluster handoff cells require complete K executions and authenticated equal identities. Missing repetitions are skipped; forcing and identity drift are rejected. Raw published streams, choice projections, actual PIDs, host facts, comparisons and executable rerun commands remain retained. Default release behavior stays unchanged.

One stress-compiled role supplies both unperturbed references and every axis. Jitter runtime controls stay outside recorded guest environment. Existing baton/grant perturbations now work inside the linked role; network operation replies also receive bounded host-random delay after their shared owner is released. The delay implementation and counters are feature-only. Qualification refuses every compiled nonordinary role through both CLI admission and the cells freeze seam.

The typed cluster seam is `stress::cluster::bind(&mut campaign::scenario::Bound)`. It updates actual per-boot environment, node-derived entropy and grouped crashes together with retained replay configuration. Original scenario capacities, topology, fault schedule, storage bounds, arguments, history/event oracle byte bounds and reopened history/identity assertions remain enforced. `scenario::publication` is crate-visible; standalone and cluster publication retain actual supplemental helper event order. These are the seams for the live corpus/mandatory producers. `history_oracle` is crate-visible for later shared-owner reconciliation.

The report uses the existing authenticated first-divergence schema and chooses the earliest helper event. Expected/actual values are encoded strings. An inspection stop appears only after the linked helper authenticates a common preceding prefix; its replay argv uses `--stop choice=N`. Missing diagnostics or authenticated prefixes produce undiagnosable findings. Every finding names task .6. See `tools/gomad/rust/STRESS.md` for the strict spec, bounds and runtime controls.

## Verification

Baseline default release library tests passed before edits: 66 tests, exit 0 (`.flow/tmp/fn188-baseline.log`). Root explicitly scoped this worker to focused Rust checks and deferred the shared full gates and heavy guest execution.

Initial checkpoint `79dbece295` checks used that source plus attribution's exact typed-role checkpoint `cac5b294eb` as a temporary overlay. The overlay was reversed before staging; this commit does not own `linked_conditions.rs`. Integrate that checkpoint with this implementation for the diagnostic refusal proof.

| Check | Result | Retained log under `.flow/tmp/` |
| --- | --- | --- |
| Default release library | 72 passed, exit 0 | `fn188-final-default-lib.log` |
| Stress-feature producer | 6 passed, exit 0 | `fn188-final-stress-lib.log` |
| Helper stress jitter | 3 passed, exit 0 | `fn188-final-helper-jitter.log` |
| Helper-only diagnostic qualification refusal | 1 passed, exit 0 | `fn188-final-diagnostic-role.log` |
| Default all-target Clippy, warnings denied | exit 0 | `fn188-final-clippy-clean.log` |
| Stress-feature all-target Clippy, warnings denied | exit 0 | `fn188-final-stress-clippy.log` |
| Ordinary/stress/helper-only-diagnostic release builds | each exit 0 | `fn188-final-build-{ordinary,stress,diagnostic}.log` |
| Actual CLI role/refusal and release exclusion | exit 0 | `fn188-final-role-check.json` |
| Python compile and Git whitespace check | each exit 0 | command evidence |

The CLI check verifies that the ordinary release excludes the jitter implementation, the stress binary includes it, and both stress and helper-only diagnostic binaries refuse qualification before opening an absent spec. Expected refusal subprocesses exit 2; the verification script exits 0.

Test-first observations retained the intended failures: missing producer API exit 101 (`fn188-red.log`), hidden forced-model comparison incorrectly passing exit 101 (`fn188-auth-red.log`), and the old linked jitter suppression yielding zero actual delays exit 101 (`fn188-jitter-red-detail.log`). Each corresponding focused check subsequently passed. Intermediate strict fixture and Clippy failures are retained too; final results above supersede them. No failed check was relabeled green.

## Pending acceptance and frozen invocation

Root must integrate the public install/preparation owner, typed helper role, attribution changes and live corpus before freezing the candidate. The old ordinary activity preparation cannot be relabeled: its helper/host identity, engine, diagnostics, digest mode, time and choice limits differ from the original stress role. The producer preserves provenance admission; no duplicate full `./tests` preparation or workload execution occurred here.

The reserved concrete invocation for the integrated owner is:

```sh
GOMAD_WASM_STRESS=1 /Users/stephan/Workspace/skunkworks/gomad/temporal/.flow/artifacts/wasm-determinism-stress/frozen/gomad-stress stress --spec /Users/stephan/Workspace/skunkworks/gomad/temporal/.flow/artifacts/wasm-determinism-stress/frozen/stress-spec.json
```

Those frozen inputs are pending, not fabricated artifacts. Root must populate the immutable final stress binary, actual same-role installations, complete source manifest and strict spec before the scheduled run; the spec's output root must be new. The full 41-by-4 matrix must execute with the selected K and cohort size. All applicable cells require complete unforced repetitions; standalone handoff remains explicitly inapplicable while actual process-layout cells still run full K. A divergence or undiagnosable result goes to .6. Task .6 gate wiring/retirement remains with fn-232 task .13. No full-pass claim or Flow completion is made here.

## Ownership and retained directories

Workspace `/home/agent/wt/fn188-rust-stress`, branch `fn188-rust-stress`, base `b17912883f2a6f8b248bd4689c84774d281f7e67`. The worktree stays registered for integration/review. All other registered worktrees belong to other active owners and were left untouched. `.flow/tmp/` retains the baseline, red/green and final logs, source overlay patch, focused binaries and evidence JSON for root review. Failed unit-fixture scratch roots were removed after log capture. The isolated target `/home/agent/.cache/codex-build/fn188-rust-stress` remains reusable while .5 is open (about 1.1 GiB); no shared cache was wiped. Last observed free disk was 26 GiB. No owned command remains running.

stage: impl-review - skipped(policy: conductor owns the gate)

## Review repair checkpoint

The canonical outcome gate now runs before every standalone or cluster custom oracle. Zero-exit PASS output cannot override `target_failure`, including soft assertions and inconclusive assertion assessment. The original runtime exit-17 canary remains narrowly admitted only with its exact witness, target-failure class and no retained assertion report.

Retained reproducer commands now invoke `stress --rerun REQUEST --root NEW_ROOT`. Typed supervision bounds preserve CPU workers, cohort size, repetition and member. The supervisor starts the original CPU load and launches the complete cohort through the existing barrier, retaining every result. Actual direct child argv remains in `executed_command`. No guest execution or qualification owner was added.

The regression uses the actual existing execution publisher to classify zero-exit/PASS plus soft-assertion or inconclusive output, then checks standalone and cluster stress admission. The unchanged gate reproduced all four bad accepts with exit 101 (`fn188-review-outcome-both-red.log`). The focused suite additionally checks the narrow runtime exception and launches a small Python rendezvous probe that fails if the full concurrent cohort is serialized or incomplete. CPU bounds and retained supervisor argv are checked without running a heavy workload. Root authorized this one bounded focused review-fix round; current-source focused suite passed 9 tests, exit 0, retained in `fn188-review-fixes-final-green.log`. Git whitespace checks and rustfmt also exited 0. No all-target or release-binary rebuild ran in this repair round. Earlier all-target/binary checks above describe the initial implementation checkpoint; integrated final gates remain with root.
