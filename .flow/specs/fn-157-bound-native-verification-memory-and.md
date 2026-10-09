# Bound native verification memory and scratch storage

## Goal and disposition

Restore the complete native verification gates without reducing their Models, states, actions, Queries, receipt populations, strict assertions or package concurrency. The conductor defers this resource work on 2026-10-09 under the owner's MILESTONES.md rule for gates stuck for roughly an hour. This spec is not ready for execution and has no implementation tasks yet. The fn-151 subject split can proceed through independent complete equivalence and review while these named gates retain their actual RED results.

The deferral changes fn-151 R4's gate acceptance. It does not certify a passing model gate, passing full Go suite, concurrent memory fit, or execution of tests interrupted by a kill. Canary publication/controller failures require a separate disposition. Strict completion, fatal-failure and pause/resume conformance remain Batch 5 obligations. Incremental Quint ITF JSON generation and consumption remain fn-154 obligations.

## Observed failures

Evidence is under `.worktrees/fn-151-split-standalone-activity-into-smaller/.flow/tmp/fn151/task3/` at source HEAD `31ac0e5e88e67a45c23a0623454ca302d5b3d18b` plus the task's pinned regenerated artifacts and exact authorized consumer corrections. Preserve the receipts, logs, input hashes and kernel records before worktree cleanup. No allocation owner or leak is proven by these RSS observations.

| Command | Actual result | Retained evidence |
| --- | --- | --- |
| `mise exec -- go test -json -tags test_dep -p 2 -timeout 30m ./tools/umpire/... ./common/testing/testpilot/... ./tools/canary/...` | Exit 1 after 1939.213 seconds. 42 packages passed, three had no tests and eight failed. Kernel-confirmed kills affected conformance, check, export, lint and lower. Two separate consumer mismatches received exact corrections and passing affected tests; Canary failures remain separate. | `canonical-go.{json,log}`, `classification/canonical-final.{json,md}`, four `canonical-*-kernel-oom-utc.log` files |
| `mise exec -- make umpire-check-model MODEL_GATE_ARGS=--skip-go-checks` | Exit 2 after 587.630 seconds. Case generator OOM and distinct Scala/lift scratch-storage exhaustion. | `model-check-no-update.{json,log}`, `model-check-no-update-kernel-oom-utc.log` |
| `mise exec -- make umpire-check-cases` | Exit 2 after 76.828 seconds. Kernel killed the generator with anon RSS 8,581,584 KiB. Runtime TMPDIR retained the original overlay; only GOTMPDIR used a fresh host directory. | `case-check.{json,log}`, `case-check-kernel-oom-utc.log` |
| `mise exec -- make umpire-check-fixtures` | Exit 2 after 63.319 seconds. Kernel killed the generator with anon RSS 8,548,272 KiB under the same temporary-directory arrangement. No fixture-content difference is established. | `fixture-check.{json,log}`, `fixture-check-kernel-oom-utc.log` |
| `mise exec -- go test -json -count=1 -tags test_dep -p 2 -timeout 30m ./tools/umpire/conformance -run '^TestEvidenceOfTheRunsRecordIsReadOnlyFromAnEventItsSourceTakes$/^a_guard_that_cannot_be_evaluated_on_the_event$'` | Exit 1 after 39.446 seconds. Exact parent and child started, then kernel killed conformance with anon RSS 8,446,896 KiB at 23:02:45 UTC. Guard/assertion result remains unobserved. | `completion-guard-negative.{json,log}`, `completion-guard-negative-kernel-oom-utc.log` |

The five canonical victims were active in `TestTheRetryContractRefusesItsEvidenceOutOfOrder`, `TestActivityChecksClean`, `TestQuintDumpIndexPreservesTheCompleteAgreement/record_with_ordered_named_alternatives`, `TestUnmodeledAPIValueCountsNoZeroValue` and `TestEveryQueryOfTheActivityModelLowersOrNamesItsLimit`. Conformance, check, lint and lower use native paths without Quint calls. The export victim's native-versus-JSON allocation phase remains unknown. Standalone generator kills show that package concurrency alone cannot explain every failure.

## Investigation and repair boundaries

Measure phase-labelled retained heap and transient allocation before choosing a repair. Separate IR admission, Realizer construction, complete interpreter tables, Check, Producer setup, Query lowering, assessment preparation and dump serialization/admission. Inspect the seven-Model loop in `tools/umpire/lower/generated.go`, the simultaneous Realizer/check views in lint and lowering, and independent Producer lifetimes in the deterministic Case test. Preserve independent readers rather than deleting an oracle to save memory.

Measure scratch-directory growth and file/inode use separately. Identify task-owned allocation and cleanup lifetimes, and document the runner's storage requirement. Broad cache deletion, shared-daemon termination and unowned temporary-directory cleanup are outside this spec. Preserve the ordinary no-update gate's build/lift/Case arrangement until a measured cause and independently checked repair justify a change.

Any sharing or compact representation needs an explicit ownership and immutability contract, caller-mutation isolation, invalidation rules and a pre-repair oracle that does not call the optimized helper. Prefer a private lifetime or representation correction supported by the measurements. Do not implement a general cache or gate framework without separate approval.

## Requirements for restoring the deferred gates

- **R1:** Pin source, all seven IR Models, artifacts, toolchains and resource settings; reproduce and measure the named memory/storage failure before repair. Report process RSS, aggregate concurrent pressure, heap observations and kernel/cgroup evidence with their different scopes.
- **R2:** Preserve complete ordered machine tables, evidence, starts/ends, assumptions, monitors, refinements, Product/System projections, exact declaration ownership and every Query's bounds, totals, expectations and answer. Keep all original 154 Activity Query occurrences and all added focused owners. Compare complete generated Programs, Contracts, Cases, manifest standings, deterministic identities, receipts, errors and replay witnesses against independent oracles.
- **R3:** Retain all owner-qualified inventory negatives, completion guard-error mutation, out-of-order evidence refusals, fresh-instance/repeated-run/caller-mutation isolation checks and strict Batch 5 assertions. Restore actual execution for tests the recorded kills prevented. Killed or unselected tests earn no pass credit.
- **R4:** Observe natural terminal results for the unchanged canonical Go command with `test_dep`, `-p 2`, `-timeout 30m` and JSON output under the shared heavy-run lock. Preserve full fixtures and assertion populations. Reduced Models, selected-only checks, `-p 1`, forced GC or runtime memory knobs cannot establish restored fit. Report remaining Quint and semantic debt separately.
- **R5:** Run normal no-update `make umpire-check-cases`, `make umpire-check-fixtures` and `make umpire-check-model MODEL_GATE_ARGS=--skip-go-checks` with complete managed-tree comparisons, unchanged bytes, documented scratch capacity and separately covering canonical Go evidence. Passing isolated equivalence or update-mode generation cannot replace these gates.

Revisit after this spec is planned and scheduled with measured phase ownership and a runner budget, or when a provisioned runner can execute the exact commands without resource failure. Activation requires a new baseline pinned to the then-current Models. Preserve fn-154's JSON protocol and exhaustive agreement obligations independently; native improvement alone cannot close it.
