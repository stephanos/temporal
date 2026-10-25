---
satisfies: [R3, R7]
---
# fn-251-one-probe-engine-and-one-catalog-build.3 Generated request_findings replaces Temporal's Wire enum

## Description
Implement **Generated findings (B2)**: generate `request_findings(method, bytes)` per service with supplemental-catalog hooks; delete `Wire` in `temporal/realization/src/probes.rs:311-551`. Answer whether `mutate`/`mutate_start` can use the generated `mutate_workflow_request` and do it if so.

## Acceptance
- [ ] No hand-written `Wire` enum; Temporal probe findings retain their semantic coverage and accuracy under the new contract, with affected output goldens regenerated.
- [ ] The `mutate` question is answered in the done summary.
- [ ] Re-accept goldens after reviewing their diffs; `make model-check` passes.


## Done summary
Temporal and Kettle now use generated per-service request findings dispatch, with typed decoding and recursive Inspect running in the consumer crate. Temporal's handwritten Wire enum and recursive findings walker are removed; contextual witness preparation, supplemental intents, recursive failures and retry normalization pass the focused semantic controls.

The generated `<prefix>_request_findings!` hook and `dispatch_<prefix>_request!` callback expand where catalog Inspect implementations live, preserving the API-to-realization dependency direction. Start/Cancel request-ID supplemental field intents already flow through Inspect; the Temporal hook adds WorkerFailure presence only. Sorting each violation's fields retains the existing prediction identity contract.

Mutation reuse decision: yes, partially. `mutate` now uses `mutate_workflow_request` for actual required root-field omissions, including Start's activity_type/task_queue omissions, and namespace size witnesses. Contextual `mutate_start` construction remains for any_of group omissions, exact encoded Payloads size, normalized retry coefficients, duration shape and catalog-selected negatives; generic Absent/Empty/Negative operations alone cannot express those same witnesses. The strict every-inventory-case check caught and prevented applying a single Absent operation to an any_of group.

Baseline: green. The pre-edit focused catalog-build/API/Temporal/Kettle command exited 0; no parent Quick-command section was defined. The conductor explicitly owns the full `make model-check` gate at join.

Verification: the final combined focused suite passed 51 tests with 0 failures and 0 ignored. The broader catalog-build/API/Kettle tests also passed; their live-snapshot import helper remains explicitly ignored without supplied live data. Strict scoped clippy across all four packages and all targets, workspace fmt check and diff check exited 0. The new generated-symbol test was observed red before implementation; the direct intent/recursive/retry control was observed red on duplicate UUID claims before redundant hooks were removed. Native Kettle coverage golden assertions passed unchanged; no snapshot was reaccepted or changed.

Parent obligations: integrate and independently review this commit, regenerate the committed Temporal guest `model/temporal/hosts/go/umpire_temporal_guest.wasm` with its current identity/freshness metadata, run the full model gate and Go host tests, review any resulting golden/asset diffs, and call flowctl done only after the conductor's review and gates pass. This worker made no lifecycle mutation and leaves the task in_progress. No manifests, lockfiles, TLA/Apalache replay files or guest assets changed. No mapped user route changed.

Logs are preserved beside this summary as fn2513-baseline.log, fn2513-final.log, fn2513-build-api-kettle.log, fn2513-clippy.log, fn2513-red.log and fn2513-intents-red.log. Build environment used the assigned fn2516 cache, CARGO_PROFILE_DEV_DEBUG=0, CARGO_PROFILE_TEST_DEBUG=0, CARGO_INCREMENTAL=0 and CARGO_BUILD_JOBS=2; all Cargo commands ran through mise from model/.

Tier: session (jev-unavailable(no_key)).
stage: impl-review - skipped(policy: parallel-wave - conductor owns the gate)

Conductor acceptance: integrated in umpire at code checkpoint d30d679fc1; task implementation review reached actual SHIP (including supplemental fixes). The joined UMPIRE_APALACHE=require make model-validate exited 0 on the unchanged model tree, including required TLC/Apalache controls, strict workspace clippy/docs/tests, embedded guest freshness, all standard wasm/native controls and all three Go hosts. Gate 1308.170s exceeds the 60s soft target; edit/test 8.882s meets 10s. Logs: consolidation-retry-model-validate.log and .exit. Final boundary/snapshot checks and reviewed golden/guest updates are integrated.
The Kettle Go parity control required one reviewed commutative any_of operand-order golden update (c928d8ff0a); coverage was unchanged and actual Go parity passed after regeneration.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: bfde956d50ab27e1fa23be7a327cd4c8c96d8322, d30d679fc1
- Tests: mise exec -- cargo test -p umpire-catalog-build -p umpire-temporal-api -p umpire-temporal -p umpire-reference-kettle --lib --test neutral --test wire_generated --test services --test wire_schema --test probes --test annotations, mise exec -- cargo test -p umpire-catalog-build -p umpire-temporal-api -p umpire-reference-kettle --tests, mise exec -- cargo clippy -p umpire-catalog-build -p umpire-temporal-api -p umpire-temporal -p umpire-reference-kettle --all-targets -- -D warnings, mise exec -- cargo fmt --all --check, git diff --check, UMPIRE_APALACHE=require make model-validate (exit 0; TLC and Apalache required; 1308.170s full gate, 8.882s inner loop; unchanged model tree), Actual Codex gpt-6.1-sol:high SHIP task implementation review and supplemental re-review; same-family CLI reviewer
- PRs: