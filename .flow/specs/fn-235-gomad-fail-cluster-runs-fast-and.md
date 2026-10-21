# Gomad: fail cluster runs fast and classified when a limit is exhausted

## Conversation Evidence

> user: "should we crash hard when we go aove the limit so it's not silent?"

Found in fn-224.3 (diagnosis: `/home/agent/wt/evidence/fn224/scenario/DIAGNOSIS.md`). A three-node ringpop scenario exhausted the scenario's `observation_bytes` World budget at 17.4 s virtual; the run then hung with zero CPU until the wall-clock execution watchdog (3-15 min) and was reported as `watchdog_observation` with empty streams and no limit named.

## Goal & Context

Any resource limit exhausted inside a cluster (multi-instance) run must end the whole run promptly, classified as capacity and naming the exhausted dimension, with the evidence needed to see what happened. A silent hang until the wall watchdog violates the fail-closed constraint and makes a configuration problem look like a scheduler defect.

Two defects in the Rust helper's manual cluster path combine:

1. **Misclassification.** Broker World errors are mapped to `Failure::Infrastructure` (`wasmhost/src/cluster/handoff/broker.rs:1024`, also `:581`, `:1048`, `:1052`) even when the underlying error is `world::Error::Capacity`. Instance-level World limits already classify as capacity.
2. **Lost terminal result.** When instance threads end in `Err` and call `coordinator.abort` (`wasmhost/src/cluster/execution/manual.rs:654`), the helper main loop stays blocked in `protocol::read_frame` on the Go control pipe (`manual.rs:422`); instance errors surface only at thread join after that loop ends. Go waits in `ClusterSession.Observe` (`wasi/cluster.go:644`, via `driveCluster` in `cluster/provider.go:918`) for an incarnation result that never arrives.

Per the Rust host decision (2026-10-20) the fix lands in the helper; the Go side only consumes the frame it already understands, or the smallest change to observe an abort frame, with its R-ID noted for fn-232's Rust Runner.

## Architecture & Data Models

- Map `world::Error::Capacity { dimension, .. }` from broker World calls to `Failure::Exhausted` with dimension, limit, used and delta preserved.
- On instance failure or broker abort in manual mode, the helper emits that incarnation's terminal result (or one cluster-abort frame) to the controller immediately, wakes the control loop, stops the remaining instances with the existing terminate path, and keeps every instance's partial stdout/stderr and the abort virtual time in the retained evidence.
- No new limit, no limit value change, no new scenario schema field.

## API Contracts

- A cluster run whose World budget is exhausted ends with outcome class `capacity`, dimension named (for example `payload-bytes`, limit and used values), within a bounded wall time independent of `--execution-timeout`.
- The retained failure artifact contains each instance's partial streams; `gomad inspect` shows them and the limit; `gomad replay` reproduces the same capacity outcome.
- The same promptness holds for any other instance or broker failure in manual cluster mode (guest trap, helper infrastructure error): no wait for the wall watchdog.

## Edge Cases & Constraints

- Two instances failing concurrently: the first terminal result in deterministic (virtual-time, node ID, incarnation) order wins; the other is recorded, not dropped.
- An abort while a declared crash/stop boundary is pending keeps existing crash/stop semantics for instances that already reached it.
- Fail-closed stays: no fallback, no limit widening, no retry.

## Acceptance Criteria

- **R1:** Broker World capacity exhaustion classifies as `capacity` naming the dimension, never `infrastructure` and never `watchdog_observation`. Errors: any other class is a finding.
- **R2:** Manual cluster runs end promptly (golden fixture wall time ≤ 10 s on a loaded host, with `--execution-timeout` ≥ 3m) after any instance or broker failure; all instances stop through the existing terminate path. Errors: a run that waits for the wall watchdog is a finding.
- **R3:** The retained artifact keeps each instance's partial stdout/stderr and the abort virtual time; inspect shows them and the limit; replay reproduces the outcome. Errors: empty streams on a capacity abort are a finding.
- **R4:** Golden regression first (testing preference R14): a two-node ping/echo scenario under `command/testdata/scenario/` with `observation_bytes` small enough to exhaust mid-run (about 4096 B) asserts R1-R3 end to end through the CLI; it fails (hangs to watchdog) before the fix. Existing scenario corpora stay green.

- **R5:** Task .2 measures the long-running three-node ringpop scenario before changing World inspection commitment computation. Retain same-host/same-window profiles and wall time per virtual second at 10, 60 and 120 seconds, repeated enough to report spread, with exact source/helper/module/configuration identities and commands. Attribute the growing cost to a named site before changing it; after the change, per-interval cost stays within measured spread rather than growing with accumulated history. Preserve commitment bytes and replay behavior, or explicitly rotate affected identities and qualify the changed contract. Existing golden scenarios and fresh retained replay pass. Errors: a profile from a different workload, an unmeasured attribution, changed commitment bytes without identity treatment, or a timing-only result without replay leaves R5 open.

## Boundaries

- No change to limit values, the 64 B/operation World charge, or observation-budget semantics (see open question).
- No new scenario or config fields. No CI run, PR or push.

## Decision Context

Task .1 is completed; its broker/manual/fixture surfaces overlap .2 and are not a parallel implementation wave. Any follow-up to .1 must reconcile those shared writes before .2 integration. Capacity mapping uses the existing accounting-preserving Exhausted representation, not a dimension-only error.

User decision 2026-10-21: exceeding a limit must not be silent. Chosen: immediate classified termination with retained evidence, rather than a helper panic (which would lose evidence and look like a helper bug). Lands in the Rust helper; coordinate the merge with fn-232's helper-as-library batch because it rotates helper identity.

Open question for the owner (not in scope): the broker World charges 64 payload bytes per brokered operation against `observation_bytes` for the whole run and never releases them, capping any brokered scenario at about 1M visible operations regardless of duration and coupling an observation budget to a transcript budget. A separate `world_payload_bytes` limit or releasing delivered bytes would decouple them; both change the model contract.

## Requirement coverage

| Req | Description | Task(s) | Gap justification |
| --- | --- | --- | --- |
| R1 | Capacity classification of broker World exhaustion | .1 | — |
| R2 | Prompt termination of manual cluster runs on failure | .1 | — |
| R3 | Partial per-instance evidence retained, inspect/replay | .1 | — |
| R4 | Golden regression scenario first | .1 | — |
| R5 | Measure and remove growing World inspection commitment cost | .2 | — |
