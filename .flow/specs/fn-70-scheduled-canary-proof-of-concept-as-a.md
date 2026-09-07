# Scheduled canary proof of concept as a second model consumer

## Goal & Context
<!-- scope: business -->

Demonstrate that functional tests and a continuously scheduled canary can consume the same Lean-owned behavior through Testpilot without either consumer adding scenario semantics. An engineer manually selects a small set of supported checks; Temporal starts a new Workflow for each selected check once per minute. Begin with exactly the Nexus3 scheduled → started → succeeded Case produced by fn-68. This is a second consumer of the model and Testpilot, not a second interpreter or an independently implemented Temporal Driver.

The first demonstration targets a local/development Temporal environment with dedicated resources. Production deployment and qualification are outside this proof of concept. This spec is separate from fn-29, whose protected production-canary design excludes automatic scheduling. It does not amend fn-29 or require its receipt/release machinery.

## Architecture & Data Models
<!-- scope: technical -->

The dependency path is:

```text
Nexus3 checked Query / Property / selected witness
    -> fn-68 Producer -> versioned Case artifact
    -> Testpilot Prepare / PreparedCase.Run
    -> shared Temporal Driver
         ^                    ^
    functional tests     tools/canary Activity
                              ^
                         Check Workflow
                              ^
                   manually configured Schedule
```

- `Temporal.Feature.Nexus3.Testpilot` remains the owner of checked lowering, history correlation and Contract meaning. Complete fn-68 before relying on this Producer. Canary does not assemble an alternative success monitor.
- `api/testpilot/v1` and `common/testing/testpilot` retain their existing protocol and execution authority. Runtime execution never invokes Lean or imports Umpire Go generation tools.
- Consume the shared Temporal Driver delivered by `fn-72-extract-the-reusable-temporal-testpilot`, preserving its server/worker authority split and private delivery ledger. This spec owns canary integration only; Driver extraction and functional-test migration belong to that prerequisite. Do not introduce a second driver implementation or a dependency on `tests/`.
- `tools/canary` owns its manually selected check catalog, schedule reconciliation, orchestration Workflow, Activity, environment configuration, and bounded result reporting. It contains no Nexus assertions, request rewriting, private Testpilot imports or alternate evaluator.
- Build/package canonical Cases ahead of execution through their owning Producer. Pin each catalog entry to a Case digest and retain its producer provenance. Updating a selected artifact is an explicit configuration/update action, never a periodic regeneration or latest-version lookup.
- Keep process-owned SDK clients, driver registrations and immutable prepared Cases in the Activity worker. Workflow history carries serializable check/artifact identities and bounded outcome summaries, not clients, credentials, PreparedCase values, or complete Run payloads.

Consume the explicit environment-binding contract delivered by `fn-73-explicit-environment-binding-for`. Request construction and worker activation must use that same immutable binding mechanism, with isolated per-measurement resources and unchanged checked Query/Property and Contract meaning. Canary selects and supplies supported bindings; it does not introduce a competing binding representation, patch Case JSON, or rewrite prepared requests. Cross-environment integration remains an acceptance test here; shared binding implementation and its wire/Producer changes belong to the prerequisite.

## API Contracts
<!-- scope: technical -->

A check selection names a known catalog entry, pinned artifact digest and environment binding. Each entry supplies one bounded Case and uses the same generic execution path. Empty selection starts no schedules; unknown names, duplicate selections, digest mismatch, unsupported Cases and incompatible environment bindings reject before activating schedules or dispatching a check. No auto-discovery, random selection, exploration or arbitrary executable selection.

Provide the smallest operator interface that can list supported checks, apply an explicit selection, run the worker, pause/resume an owned schedule and inspect check status. Schedule reconciliation has stable IDs scoped to this canary installation and check identity. Reapplying the same selection does not create duplicates; a conflicting existing schedule is reported rather than silently taken over. List/status/worker startup do not create schedules. Removing a check from an explicitly applied selection pauses its owned schedule; it does not delete historical measurements or unrelated schedules.

Use one Temporal Schedule per selected check with a 60-second interval and overlap policy Skip. Each tick starts a fresh Check Workflow; do not implement cadence as sleep-after-completion in a permanent workflow. A slow check causes skipped ticks rather than queued overlapping executions. Use a bounded catch-up window so recovery does not replay a backlog of missed minutes. The exact supported SDK setting and minimum catch-up behavior must be verified against the pinned SDK during implementation and documented.

The Check Workflow invokes one bounded Activity that prepares/obtains the pinned Case and calls PreparedCase.Run through the shared Driver. It does not interpret the Program itself. Disable Workflow execution retries and set the check Activity's maximum attempts to one; normal Workflow Task replay remains supported. One scheduled tick identifies one measurement, not a retry-until-green loop. Do not claim exactly-once external effects merely from these retry settings.

A completed measurement retains check identity, orchestration Workflow/run identity, Case digest and producer provenance, Profile/catalog identity, Testpilot Run identity, separate execution disposition, Verdict and cleanup status, plus a reference to the retained Run. Return only a bounded summary in Workflow history and keep full Run data in a configured bounded local artifact sink for the prototype. Define finite byte/count retention and observable publication failures; use atomic publication for files and preserve the owning module's path-safety conventions. No credentials or opaque completion capabilities enter reporting. A reporting failure cannot trigger another execution.

Model violation and inconclusive results are measurements, not retryable Activity failures. Operational inability to obtain an authoritative result is recorded distinctly as uncertain/incomplete at the canary layer. Never synthesize a satisfied Verdict, successful cleanup or a closed Testpilot Run after process loss.

## Edge Cases & Constraints
<!-- scope: technical -->

- Replay: workflow replay must not invoke Testpilot or duplicate external work. Activity results, including violated/inconclusive Verdicts, are replayed as data.
- Process loss/timeouts: the Activity may have caused effects even if its response is lost. Mark the measurement uncertain, propagate orchestration failure and configure the schedule to pause on such failure/timeouts. No automatic redispatch or automatic resume. The operator inspects and reconciles exact owned resources before resuming. This prototype needs no general recovery service.
- Cancellation/cleanup: cancellation reaches Testpilot's bounded cleanup path when the worker is alive. Give the Activity enough time for execution plus cleanup and the Workflow enough time for the Activity terminal result. Target workflows/resources need finite server-enforced lifetimes as a backstop when the Activity worker disappears. A timeout is not evidence that all effects stopped; schedule Skip alone cannot establish resource isolation.
- Shared process: reuse immutable prepared Cases and driver registrations, but each measurement has distinct Run identity, Slots, captures, capabilities and resource ownership. A late result cannot mutate another measurement. Cap worker concurrency explicitly for the small configured selection; a 10x tick/load increase cannot create an unbounded dispatch queue or retained-output growth.
- Dependency failure: scheduling and target execution may use the same local cluster for the demonstration. A cluster outage can prevent measurement creation. Status inspection must expose last scheduled/started/completed observations and freshness separately from Verdict. Unreachable or stale status is never healthy. This is inspection by an external operator/client, not a claim that an unavailable cluster can report its own outage.
- Resource portability: reuse the same exact Case bytes across two distinct namespace/task-queue bindings. Binding changes alter the prepared/Driver binding identity, but not the source artifact, checked success requirement, Contract, provenance, or correlation interpretation. Unsupported bindings fail explicitly.
- Shutdown: stop new owned scheduling when requested and drain/close worker-owned resources within configured bounds. Do not clean unrelated workflows or queues. Preserve independent cleanup and Verdict outcomes.
- Schema evolution: consume the Testpilot protocol and fn-68 Producer as evolved by the binding prerequisite and its standalone Lean protocol dependency. This spec owns no protocol extraction, expression-language redesign or binding wire change.

## Acceptance Criteria
<!-- scope: both -->

- **R1:** `tools/canary` exposes only explicitly selected supported checks, initially Nexus3 success. Unknown/duplicate names, empty selection, stale or mismatched artifacts and unauthorized/incompatible bindings are covered by tests; invalid selections create no active schedule or target Run.
- **R2:** Canary and the existing functional test consume the fn-68 Producer's checked success behavior through the same Testpilot facade. Source IDs/fingerprints and exact Known Gaps remain bound to checked inputs. No duplicated monitor, Nexus assertion, scenario interpreter, or runtime Lean invocation exists in canary.
- **R3:** One shared Temporal Driver serves both consumers from outside `tests/`, preserving server/worker/delivery ownership. Dependency checks show canary and shared Driver production code import neither `tests/`, Umpire generator tools nor private Testpilot execution/verification packages. Canary integration preserves the prerequisite Driver guarantees; its extraction and migration tests remain owned by `fn-72-extract-the-reusable-temporal-testpilot`.
- **R4:** The same checked selection executes against two explicit namespace/task-queue configurations using one binding mechanism and isolated run-owned resources. Neither consumer rewrites prepared requests or independently hard-codes corresponding worker/request bindings; mismatch tests fail before check dispatch.
- **R5:** Applying an explicit selection creates one stable, 60-second, Skip-overlap Schedule per check, starting a fresh Workflow for each measurement. Reapply is idempotent, conflicts reject, removal pauses only owned schedules, and pause/resume work. A check exceeding the interval does not overlap or accumulate buffered starts; catch-up is bounded and tested.
- **R6:** Workflow orchestration contains only deterministic SDK operations and calls a bounded Activity for Testpilot execution. Workflow execution retries and check Activity retries are disabled. Replay tests show no duplicate external dispatch; a violated or inconclusive measurement is retained without an automatic rerun.
- **R7:** Repeated completed measurements retain distinct Run/resource identities and unchanged semantics, with separate disposition, Verdict, cleanup and bounded artifact references. Cross-run isolation, late completions, reporting failure and retention exhaustion have focused tests; none overwrites a prior measurement or causes redispatch.
- **R8:** Activity worker loss, cancellation and timeouts cannot fabricate Run closure or success. Uncertain orchestration failures pause the schedule until explicit resume; bounded cleanup and finite target resource lifetimes are demonstrated. Status inspection distinguishes missing/stale measurements and unavailable orchestration from actual violated/inconclusive Verdicts.
- **R9:** A live local demonstration runs the existing Nexus3 functional test and at least two successive scheduled canary measurements through the shared Driver. Successful canary Runs have completed disposition, satisfied Verdict, successful cleanup and the Producer-defined correlated history support. Assert event-driven results rather than fixed test sleeps; use a generous bounded deadline for the real minute cadence.
- **R10:** Documentation under `tools/canary` gives reproducible build/artifact provisioning, explicit selection, worker, schedule apply/pause/resume/status and cleanup commands; names required environment settings, finite limits, retry/overlap behavior, result storage and the same-cluster freshness limitation. It clearly identifies the non-production proof-of-concept scope and fn-68 dependency.

## Boundaries
<!-- scope: business -->

No production deployment, customer traffic, release qualification, protected GitHub workflow, general lease/fencing/recovery platform, dashboard, alerting service, remote result store, or fn-29 receipt machinery. No arbitrary scenario upload, discovery, randomized exploration, cancellation-model expansion, fault injection, new runtime opcode, alternate evaluator, or universal Property compiler. No unrelated Lean authoring redesign, blanket codec migration, broad generator changes, new third-party libraries, or CI expansion. Shared Driver extraction and environment-binding implementation are owned by `fn-72-extract-the-reusable-temporal-testpilot` and `fn-73-explicit-environment-binding-for`, respectively; do not duplicate that work here. Driver activation-interface deepening and Lean semantic refactors remain independent work. This spec authorizes design/implementation scope; actual infrastructure deployment is not performed by creating the spec.

## Decision Context
<!-- scope: both -->

- User requested `tools/canary` as a second Lean-model consumer, with a very small manually selected scenario set, repeated starts once per minute, and a Temporal Workflow for each check.
- Reuse fn-68's checked success lowering and live proof rather than create competing integration work. This spec depends on fn-68, `fn-72-extract-the-reusable-temporal-testpilot` and `fn-73-explicit-environment-binding-for`; re-anchor on their final interfaces, artifact identities and metadata policy before implementation.
- Temporal Schedules are preferred over a local ticker because scheduling is part of the requested demonstration and remains inspectable/durable. Fresh scheduled Workflows are preferred over an endless per-check workflow because each measurement has an independent orchestration record and fixed start cadence.
- Execute the Case within an Activity to keep runtime I/O out of Workflow replay. Keep the same Driver in tests and canary; consumer differences belong in orchestration, bindings and reporting.
- Prefer the smallest complete success slice. Add a second scenario only through a later request; a new generic framework is not the acceptance test.
- The prototype intentionally differs from fn-29's manual-only production controller. Neither spec supersedes the other; do not inherit production gates or claim production readiness from this demonstration.
- Model-to-Case continuity is owned by fn-68, shared Driver placement by `fn-72-extract-the-reusable-temporal-testpilot`, and environment binding by `fn-73-explicit-environment-binding-for`. The standalone Lean protocol is a transitive prerequisite through binding. Activation-interface deepening and Lean Target/inventory refactors do not block this proof unless an evidenced integration requirement changes that decision.

## Early proof point

Before adding schedules, prepare a Nexus3-produced Case through the non-test shared Driver and execute two isolated measurements using the canary Activity entry point. Compare with the functional consumer and demonstrate the alternate namespace/queue binding. If this requires canary-specific request interpretation or a copied Contract, report the unmet prerequisite acceptance criterion and resolve it in its owning spec before adding cadence.

## Verification

Task planning must choose focused commands after the shared package path is settled. All Go tests use `-tags test_dep`; only live tests add `integration`. Run the existing Testpilot facade/driver tests, scheduler reconciliation and Workflow replay tests, negative failure/retention tests, and the real local two-tick demonstration. Run Lean checks and owner-managed fixture generation/drift checks if Producer inputs/artifacts change. Finish implementation with project-required `make lint-code` and applicable model gates. Report environment failures separately; do not weaken assertions or silently skip the live acceptance proof.

## References

- `.plans/UMPIRE4_SPEC.md` — semantic authority, Testpilot facade and Driver ownership.
- `.plans/UMPIRE_ARCHITECTURE_REVIEW.md` — findings 1, 3, 4 and 5.
- `fn-68-minimal-nexus3-success-demonstration` — checked lowering and existing fixture/live integration.
- `fn-72-extract-the-reusable-temporal-testpilot` — reusable Temporal Driver extraction.
- `fn-73-explicit-environment-binding-for` — explicit shared environment binding.
- `fn-29-bounded-production-canary-execution-and` — separate production qualification scope.
- `common/testing/temporaltestpilot/README.md`, `tests/testcore/testpilot/README.md`, and `common/testing/testpilot/README.md` — existing runtime and fixture boundaries.
- `tests/testpilot_async_nexus_case_test.go` — existing live consumer to preserve.
- Temporal Go schedule documentation: https://github.com/temporalio/documentation/blob/main/docs/develop/go/workflows/schedules.mdx. Verify details against the repository's pinned SDK during implementation.
