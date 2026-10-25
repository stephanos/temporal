---
satisfies: [R1, R2]
---
# fn-242-gomad-shrink-the-qualification.53 Process receipt field review: keep only fields that catch a named failure

## Description
Review the process custody receipt, field by field, against the obligation inventory. Keep a field only if it catches a named failure.

Filed by Claude on 2026-10-25 at the owner's direction; the fn-242 conductor schedules it. The aim is for the Rust cycle verb to carry a lean receipt rather than the Python one field for field.

The port of `launch.py` (`gomad/src/cycle/process/mod.rs` + `sys.rs`, ≈1.9k production lines and ≈1.5k test lines) records the following per command (`Registration`, `Completed`, `Resources`):
- five process IDs: pid, parent_pid, stage_pid, supervisor_pid, supervisor_parent_pid
- three monotonic timestamps: started, reaped, cleaned
- start_ns/end_ns, plus the clock name
- the environment four ways: environment, declared, effective, execution, plus `environment_mode`
- temporary paths
- resource usage, including `ru_maxrss_unit`, `cpu_conversion`, `rss_conversion`, `wait4_calls` and `reaps`
- cleanup state

Some fields catch real failures, and `.flow/artifacts/fn242-obligation-inventory.json` has the checks:
- `environment_mode == 'exact'` together with `effective_environment` catches a command that inherits an ambient or unreviewed environment (identity producers, the SQLite oracle, the GNU probe).
- `parent_pid`/`stage_pid` lineage catches a "foreign or incomplete installed CLI command".
- start/reap within the recipe deadline catches lifecycle drift.

Others look as though they only re-prove one another or record bookkeeping: a second copy of the environment, unit and conversion strings, the `wait4` call count, the supervisor's parent pid.

Do:
1. For every receipt field, record in the R1 inventory either the failure it catches (with the existing catching check or mutation) or a deletion reason.
2. Remove the fields that catch nothing, along with their readers and tests, in the Rust verb.
3. Where two fields prove the same thing, keep one.
The result is the lean result type that the fn-240 shared process crate exposes.

This work stays inside R1/R2: every removal has an inventory entry, and no kept check loses its catching proof.

## Acceptance
- Every `Registration`/`Completed`/`Resources` field has an R1 inventory row naming either its caught failure (with its check or mutation) or its deletion reason.
- Removed fields are gone from the Rust verb, its readers and its tests; the before/after field count and code-line count (`gomad-source-size/v1`) are recorded.
- Mutation controls for the kept fields still fail as before (environment inheritance, foreign command lineage, deadline lifecycle).


## Done summary
Blocked:
# fn-242 parked by owner

The owner requested: "can we move past fn-242 for now and leave that be?"

Stop fn-242 source changes, preparation cohorts, callbacks, proof reconciliation, reviews and task closure. Resume only on explicit owner direction. Preserve the worktrees, commits, admissions and evidence; parking supplies no completion credit.

The last frozen source is `7e713d3d1a4adee10e85917a72c102a72e470dc1`. Its prepare-only run completed all 15 admissions, then refused configuration because setup supplied empty `live_corpus.families`. There is no genuine configuration or executed B10 callback proof from this run. Preserve `/home/agent/wt/evidence/fn242-b11-workload-reference-oracle-producers/genuine-prepare-7e713-run.json` and `genuine-prepare-7e713-root` unchanged.

The process-field source checkpoints `01902d7970` and `46e77aa9a7` and B10 landing `9469156b9d` remain unaccepted work. Existing passing checks retain only their original scope.

The overall MILESTONES goal continues through independent fn-253 work: inner loop, nextest, standalone entrypoints, gate visibility and evidence-store work that does not resume the parked cycle integration. fn-253.2/.3 still wait for cutover; downstream dependency gates are not waived. No full qualification attempt has been used: 0 of 2.
## Evidence
- Commits:
- Tests:
- PRs:
