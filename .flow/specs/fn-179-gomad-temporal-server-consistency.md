# Temporal server consistency oracles under Gomad

## Native execution acceptance amendment (2026-10-14)

Apply the [owner's native backend hibernation decision](../../MILESTONES.md#native-backend-hibernation) to this spec and its task acceptance. It explicitly transfers new as well as inherited native execution requirements to fn-128/fn-149, superseding earlier statements that the new gates remain source-completion blockers. Retained source and actual WASM requirements still need current-candidate evidence; native execution remains deferred and unverified. Historical selectors, semantics and receipts below remain the deferred ledger.

## Goal & Context

Gomad exploration only finds what something checks. Temporal already states many server invariants as soft assertions (`softassert.That`/`Fail`), which log an error tagged `FailedAssertion` with the prefix `failed assertion:`. Functional suites built on `testcore` fail the test when one fires. That check is missing in several places:
- suites with their own logger;
- assertions logged after the test that caused them has finished, from background goroutines on the shared cluster;
- workloads that do not use `testcore`, such as the multi-instance simulations.

Gomad also only sees a generic test failure, without the assertion that caused it.

The goal is to make every soft-assertion failure in a Gomad-run Temporal workload fail the run with its own classified failure, so exploration seeds that drive the server into an inconsistent state are found, deduplicated and replayable. Later oracles (post-run execution validators, rebuild-and-compare; mutable-state checksums at 100% were dropped on 2026-10-13) belong to this spec when they are planned.

## Architecture & Data Models

- A soft-assertion oracle observes each guest's captured log output (stdout/stderr, already part of the replayed evidence) for the `failed assertion:` marker, independently of the test logger.
- It plugs into the existing oracle plan for multi-instance simulations and into single-process WASM runs, so both report it the same way.
- A failure carries a stable signature: the assertion's static message (soft assertions use static messages by contract) and, for clusters, the instance. Dynamic tags stay in the evidence, not the signature, so the same bug found under different seeds deduplicates.


## Edge Cases & Constraints

- Tests that deliberately trigger soft assertions (those using `WithDisableTestloggerFailure`) must be able to opt out per run, explicitly and recorded in the run's identity, never silently.
- Assertions logged after the guest test reports success still fail the run.
- Detection must not change the guest's behaviour: it reads evidence only, so replay stays exact.
- Log formats may change; detection keys on the shared tag or prefix that `softassert` owns, not on incidental formatting.

## Acceptance Criteria

- **R1:** A Gomad run of a Temporal workload in which any soft assertion fires ends as a failure classified as a soft-assertion failure, even when the guest test passed, the suite uses its own logger, or the assertion fired after the test finished. Errors: unreadable or truncated log evidence → the run is reported as inconclusive for this oracle, never as a pass.
- **R2:** The failure signature is the assertion's static message (plus the instance for clusters), so the same assertion under different seeds deduplicates, and the failure artifact replays to the same signature. No error surface beyond R1.
- **R3:** A run can opt out of the oracle only through an explicit, recorded option, and an opted-out run reports the assertions it saw without failing. Errors: an opt-out on a workload that does not support it → invalid configuration before execution.
- ~~**R4:**~~ ~~Gomad-run Temporal workloads generate and verify mutable-state checksums on every write and load. A checksum mismatch fails the run as a classified checksum failure, with the same signature, deduplication and replay guarantees as R2. Errors: a workload whose dynamic config cannot be overridden → refused before execution; a mismatch on the documented known-gap list → reported, not failed.~~ Dropped (owner decision 2026-10-13).

## Boundaries

- Post-run execution validators and rebuild-and-compare are later tasks in this spec, not part of R1–R3.
- No changes to how `softassert` itself logs, and no new assertions in the server.

## Decision Context

Owner decision (2026-10-13): Temporal's own consistency checks are the cheapest strong oracle for Gomad exploration. The soft-assertion oracle comes first because the assertions already exist and functional suites already treat them as failures; the oracle closes the gaps and gives Gomad a typed failure signature.
