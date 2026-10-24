# fn-232.13 blockers and advice for the implementing worker (2026-10-24, conductor review)

Scope: what stands between the current `gomad` head and one green 96-stage cycle, ranked by
what blocks first. Facts come from cycle 8 (`fb016c995f`, 23 PASS / 2 FAIL / 71 unstarted),
the .29 diagnosis, the .30 task text, branch `fn232-inspection-poll-batch` (head `a2ea7bb1f6`,
one uncommitted test) and the gate logs. Nothing here waives a bound, assertion or stage.

## 1. fn-232.30 (a): retained-incarnation replay refuses valid payloads (blocks `temporal-inspection`)

**Symptom.** Detached replay of the 16 MiB diagnostic artifact exits 2 before execution with
`cluster payload exceeds caller bound or has invalid identity`. The choice payload is valid
(13,935,648 bytes); its `descriptor.limit` is the authenticated aggregate evidence allowance,
1,210,056,704 bytes. The reader passes a fixed `1 << 30` as `maximum`, and `Payload::bind`
refuses because `self.limit > maximum`.

**Where.** `tools/gomad/rust/gomad/src/replay/cluster.rs`:
- line 321, runtime forcing: `payload.read(opened, 1 << 30)` on the primary incarnation's
  `choice` descriptor;
- line 578, `authenticate_retained_incarnation`: `descriptor.read(opened, 1 << 30)` over every
  retained stream;
- the inspection authenticator (`authenticate_inspection`, line 726) reaches the same reader.
Lines 483, 589, 590 and 599 already do it right: they pass `item.limits.evidence_bytes`.

**Fix.** Replace the two fixed bounds with the owning incarnation's `limits.evidence_bytes`.
At line 321 the owner is `retained_primary` (look up its `Incarnation` by handle in
`inputs.incarnations`, the same way line 483 has `item`). Add one small helper so there is
exactly one place that answers "what is this incarnation's evidence bound", then grep
`replay/cluster.rs` for `1 << 30` until the only remaining literal is the 512 MiB module bound
at line 451. Do not touch `Payload::bind`: the `self.limit > maximum` check is the refusal the
`evidence_bytes + 1` test must keep hitting.

**Tests.** The worker's RED test `retained_incarnation_uses_authenticated_evidence_bound`
(retained_tests.rs:498) fails exactly where expected. The uncommitted
`retained_inspection_uses_authenticated_evidence_bound` covers the inspection path. Both use a
tiny fixture that only declares the 1154 MiB bound, so no GiB allocation; keep it that way.
GREEN must show all three readers accept `limit == evidence_bytes` and refuse
`limit == evidence_bytes + 1`.

## 2. fn-232.30 (b): onebox inspection canary budget (same stage)

**Decision already taken.** The owner-authorized measured budget is 8 MiB for this one canary
(measured 4,718,416 broker transition bytes on the full 3-minute workload; idles alone are
1,060,466, so 1 MiB can never fit). Commit `a2ea7bb1f6` already moves
`onebox-inspection-check.py`, `onebox-inspection-test.py` and the
`qualify/support/inspection/recording.rs` argv mirror from `1048576` to `8388608`.

**Check before running.** Grep for every remaining `--world-transition-limit` and `1048576` that
names the onebox inspection stage; `final_retained.py:352` and `core_lifecycle.py:252` are
different workloads and stay at 1 MiB. `onebox-inspection-check.py:88`
(`incarnation['model']['limit'] == 1 << 30`) is the choice-trace bound, not this budget;
leave it. QUALIFICATION.md's `onebox` row must state 8 MiB and cite the measurement so the
reader and the document agree.

**Stop iterating on polling.** The op14 readiness probe and positive-hint reuse
(`c6c109d089`, `024e407b09`, `0bc28fae03`, `22ae55f5ab`) are landed on the branch and keep
their 1 MiB linked RED/GREEN regression. Two full attempts at 1 MiB still refused; the budget
decision supersedes further polling work for .30. A further polling reduction is a separate
performance item for the quiet-host window, not a .30 deliverable.

**Run order.** Before any full cycle, run `testdata/parity/onebox-inspection-check.py`
standalone on the merged tree with the 8 MiB argv: explore must exit 1 with the one retained
target failure, detached replay must print the exact `reproduced=true ... choice-replay=exact`
line (this is what (a) unblocks), then stopped and reopened inspection. The stage took 699 s
in cycle 8 and ran last; proving it alone costs 12 minutes, proving it inside a cycle costs an
hour.

## 3. Reference-helper corpus red on head (merge-gate risk)

The fn-188.6 handover reported `helper_corpus` red on `gomad` head; the 2026-10-23 gate log
shows `cluster_broker_runs_recorded_go_orders_with_central_faults` FAILED under
`test-rust-engine`. Cycle 8 passed `rust-conformance-unit` and `rust-default-test`, so it may
already be fixed, but nothing recorded it fixed. Before merging .30: run
`make -C tools/gomad test-rust-engine` on the merged tree. If red, bisect between `2fb6e7ceb4`
and head (candidate: `429cd93008`, which re-gated cluster test helpers by profile) and add a
regression. Never regenerate corpus goldens to get green.

## 4. Diagnostic-test stage

Cycle 8's `rust-diagnostic-test` failure (16 fixtures selecting ordinary conditions instead of
their compiled role) is repaired by .26/.28 and merged. Confirm on the merged tree with the full
five-profile matrix the task names, not on the branch alone; this stage runs late in the cycle
(start +2772 s) and a repeat failure costs the whole hour.

## 5. Host admission: disk

`/` is at 90% (20 GB free) and the host volume at 93% (72 GB free). Every stage runs
`host-admission` with `minimum_free_bytes` 12 GiB; cycle 8 was down to 17.7 GB at the
inspection stage. A mid-cycle admission FAIL would look like a product failure. Before launch,
remove stale cargo targets under `.build-cache/` from closed repair cycles (keep only the
target the controller binds, plus sealed ones) and the superseded worktrees listed in the
cleanup policy. Record what was removed.

## 6. Wall time is not a blocker, but plan for it

The driver records an overrun as `deferred-perf` and never exits 3 (driver.py:113), in line
with the owner's performance deferral. It still matters for planning: cycle 8 spent 2,986 s of
driver time on 23 stages because every build stage runs with `CARGO_BUILD_JOBS=1`,
`GOFLAGS=-p=1` and, for some, its own `--target-dir` (`rust-lint` 570 s, `wasmmeasure-build`
545 s, `rust-default-test` 506 s, `helper-build` 472 s, `compile-only` 432 s). Expect the full
96 stages to take several hours; do not start other heavy jobs on the host during the run, and
do not try to fix the scheduling in .30. Sharing one release target across the build stages and
giving cargo the scheduler's core allocation is fn-242's cycle-verb work (or the quiet-host
window), where it can be done once with a plan golden.

## 7. Small hygiene items

- `/home/agent/wt/WORKER_RULES.md` still says Rust 1.94.1; the workspace pin and every cycle use
  1.99.1. Fix the rules file so a fresh worker does not build with the wrong toolchain.
- Keep committing per fix on the branch with the task ID in the message (as `a2ea7bb1f6` does);
  the conductor merges and owns Flow.
- Evidence under `/home/agent/wt/evidence/` is host-local. Each cycle's one-paragraph summary
  with command exits and identities belongs in `.flow/artifacts/`, as the previous cycles did.

## Order of work

1. Fix (1); GREEN both bound tests; `cargo test -p gomad replay::cluster`.
2. Verify (2)'s remaining mirrors and QUALIFICATION.md; run the standalone onebox check.
3. Merge-gate on the merged tree, including `test-rust-engine` (3) and the diagnostic matrix (4).
4. Clean disk (5), then launch the cycle with the current controller and leave the host alone (6).

## Addendum (same day): what the retained attempts have never exercised

Across the eight retained cycle attempts, 73 of the 96 stages have never produced a PASS receipt:
every `runtime-N` shard, all ten `workload-*` cells, `workload-corpus`, the four `sqlite-selection-*`
stages and `sqlite-catalogue`, `ordinary`, `artifact`, `operator`, `storage`, `network`, `clusters`,
the `terminal-*` and `busy-*` stages and `retained-*`. The cycles so far have only ever reached the
build, lint and unit-test half. Treat the first run that gets past `temporal-inspection` as a
discovery run: expect new failures in the second half, run failed stages standalone, and fix forward.

Two risks in that half are now filed as tasks:

- **fn-232.31** (predicted blocker; .13 depends on it): the sqlite and multi-instance cells have never
  run under the per-owner World transition budget that Rust enforces since `93be185443`. Onebox needed
  4.7 MiB for one cluster over three virtual minutes; the replication scenario runs two clusters for
  up to fifteen. Measure first, as .30 did, then set the budget with headroom. Do this before the cycle
  reaches `workload-multi-instance-*`, because that stage runs late and a failure there costs the hour.
- **fn-232.32** (same class as the .30 refusal, lower priority): readers still pass literal bounds
  (16 MiB versus 64 MiB for the same `cluster-replay.json`, ad hoc stream bounds in the workload and
  retry readers). Unify them with the authenticated limits so nothing qualifies on one path and is
  refused on another.

Checked and not a problem: phase attribution is enabled on the real `qualify` path
(`cli/qualify.rs`) and `/proc/self/clear_refs` is writable in this sandbox, so the 2 GiB operator
ceiling is measured against operator phases, not against the 2.75 GiB whole-process peak the onebox
run reaches. The runtime-N reader's 64 MiB bound matches what `runtime_qualification.rs` records.
