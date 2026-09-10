# Scheduled canary proof of concept as a second model consumer

Status: task planning against delivered dependencies. Nexus operation cancellation is deferred to fn-79. This canary consumes the existing success Case and fn-78’s generic monitoring delivery; it does not wait for a cancellation adapter, capability, or demonstration. Activity/Workflow cancellation and bounded cleanup below remain operational requirements.

> HTML render lens: `.flow/artifacts/fn-70-scheduled-canary-proof-of-concept-as-a/spec.html` — open locally; regenerable, markdown is the record. <!-- flow-next:artifact-link -->

## Goal & Context
<!-- scope: business -->

Demonstrate that functional tests and a continuously scheduled canary can consume the same Lean-owned behavior through Testpilot without either consumer adding scenario semantics. An engineer manually selects a small set of supported checks; Temporal starts a new Workflow for each selected check once per minute. Begin with exactly the Nexus success scheduled → started → succeeded Case produced by fn-68. This is a second consumer of the model and Testpilot, not a second interpreter or an independently implemented Temporal Driver.

The first demonstration targets a local/development Temporal environment with dedicated resources. Production deployment and qualification are outside this proof of concept. This spec is separate from fn-29, whose protected production-canary design excludes automatic scheduling. It does not amend fn-29 or require its receipt/release machinery.

## Architecture & Data Models
<!-- scope: technical -->

The dependency path is:

```text
Nexus success checked Query / Property / selected witness
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

- `Temporal.Feature.Nexus.Success.Producer` remains the owner of checked lowering, history correlation and Contract meaning. The delivered fn-68 Producer is the compatibility baseline. Canary does not assemble an alternative success monitor.
- `api/testpilot/v1` and `common/testing/testpilot` retain their existing protocol and execution authority. Runtime execution never invokes Lean or imports Umpire Go generation tools.
- Consume the shared Temporal Driver delivered by `fn-72-extract-the-reusable-temporal-testpilot`, preserving its server/worker authority split and private delivery ledger. This spec owns canary integration only; Driver extraction and functional-test migration belong to that prerequisite. Do not introduce a second driver implementation or a dependency on `tests/`.
- `tools/canary` owns its manually selected check catalog, schedule reconciliation, orchestration Workflow, Activity, environment configuration, and bounded result reporting. It contains no Nexus assertions, request rewriting, private Testpilot imports or alternate evaluator.
- Build/package canonical Cases ahead of execution through their owning Producer. Pin each catalog entry to a Case digest and retain its producer provenance. Updating a selected artifact is an explicit configuration/update action, never a periodic regeneration or latest-version lookup.
- Keep process-owned SDK clients, driver registrations and immutable prepared Cases in the Activity worker. Workflow history carries serializable check/artifact identities and bounded outcome summaries, not clients, credentials, PreparedCase values, or complete Run payloads.

Consume the explicit environment-binding contract delivered by `fn-73-explicit-environment-binding-for`. Request construction and worker activation must use that same immutable binding mechanism, with isolated per-measurement resources and unchanged checked Query/Property and Contract meaning. Canary selects and supplies supported bindings; it does not introduce a competing binding representation, patch Case JSON, or rewrite prepared requests. Cross-environment integration remains an acceptance test here; shared binding implementation and its wire/Producer changes belong to the prerequisite.

Consume the checked Query, Property, obligation, and evidence-projection contracts delivered by `fn-78-typed-temporal-authoring-and-checked`. Canary schedules and reports the resulting Case; it does not define another temporal language, evaluator, or Nexus-specific assertion path.

fn-68, fn-72, fn-73, and fn-78 have delivered these contracts. fn-78's administrative open status
does not indicate missing generic semantics. Execution follows fn-77 in the delivery order and
re-anchors its final Producer/Case interfaces before edits; this source coordination is not a new
semantic dependency on parameterized field Properties.

### Narrow prerequisite amendments

Inspection identified three concrete gaps in the integration required below. This spec includes
their smallest fixes in the existing owners, followed by canary integration:

- Add `PreparedCase.ValidateDriver(ctx, driver) error` to the public Testpilot facade. Extract and
  reuse the existing preflight identity/binding and `Driver.Validate` checks without creating a
  Monitor, Session, Run, or effect. `Run` repeats current preflight validation; an earlier successful
  check grants no lasting execution authority. Canary validates every selected prepared Case and
  Driver before scheduling, without duplicating private execution rules.
- Ensure the checked Nexus success Producer supplies a finite target `workflow_execution_timeout` of
  60 seconds. Current request construction omits a server lifetime. Make this amendment in the
  Producer, regenerate and explicitly repin both consumers, and qualify lowering/provenance/trust.
  Preserve the success Property and correlated Contract; canary never patches requests. If fn-77
  already delivers the required backstop, reuse and qualify it instead of adding another timeout.
- Make the CHASM scheduler honor its existing namespace-scoped
  `scheduler.tweakables.CanceledTerminatedCountAsFailures` setting when applying `PauseOnFailure`.
  Its pause predicate currently ignores that field. Preserve the default `false` behavior and
  all other statuses. The dedicated local canary namespace requires CHASM scheduling with this
  field explicitly `true`; live tests prove cancellation and termination pause future ticks.

These amendments do not reopen unrelated completed work, add an operation-cancellation capability,
or introduce another scheduler/recovery service. They belong to their existing facade, Producer,
and scheduler owners, with focused tests before integration.

## API Contracts
<!-- scope: technical -->

A check selection names a known catalog entry, pinned artifact digest and environment binding. Each entry supplies one bounded Case and uses the same generic execution path. Empty selection starts no schedules; unknown names, duplicate selections, digest mismatch, unsupported Cases and incompatible environment bindings reject before activating schedules or dispatching a check. No auto-discovery, random selection, exploration or arbitrary executable selection.

Provide the smallest operator interface that can list supported checks, apply an explicit selection, run the worker, pause/resume an owned schedule and inspect check status. Schedule reconciliation has stable IDs scoped to this canary installation and check identity. Reapplying the same selection does not create duplicates; a conflicting existing schedule is reported rather than silently taken over. List/status/worker startup do not create schedules. Removing a check from an explicitly applied selection pauses its owned schedule; it does not delete historical measurements or unrelated schedules.

Use one Temporal Schedule per selected check with a 60-second interval and overlap policy Skip. Each tick starts a fresh Check Workflow; do not implement cadence as sleep-after-completion in a permanent workflow. A slow check causes skipped ticks rather than queued overlapping executions. Use a bounded catch-up window so recovery does not replay a backlog of missed minutes. The exact supported SDK setting and minimum catch-up behavior must be verified against the pinned SDK during implementation and documented.

Set catch-up explicitly to 10 seconds and verify the effective Schedule policy with Describe.
The pinned SDK comments and server defaults differ; no default is assumed. Public API documentation
also specifies a 10-second minimum and a one-year default, so zero is unsuitable for this prototype.
See [SchedulePolicies](https://pkg.go.dev/go.temporal.io/api/schedule/v1#SchedulePolicies).

The Check Workflow invokes one bounded Activity that prepares/obtains the pinned Case and calls PreparedCase.Run through the shared Driver. It does not interpret the Program itself. Disable Workflow execution retries and set the check Activity's maximum attempts to one; normal Workflow Task replay remains supported. One scheduled tick identifies one measurement, not a retry-until-green loop. Do not claim exactly-once external effects merely from these retry settings.

A completed measurement retains check identity, orchestration Workflow/run identity, Case digest and producer provenance, Profile/catalog identity, Testpilot Run identity, separate execution disposition, Verdict and cleanup status, plus a reference to the retained Run. Return only a bounded summary in Workflow history and keep full Run data in a configured bounded local artifact sink for the prototype. Define finite byte/count retention and observable publication failures; use atomic publication for files and preserve the owning module's path-safety conventions. No credentials or opaque completion capabilities enter reporting. A reporting failure cannot trigger another execution.

Model violation and inconclusive results are measurements, not retryable Activity failures. Operational inability to obtain an authoritative result is recorded distinctly as uncertain/incomplete at the canary layer. Never synthesize a satisfied Verdict, successful cleanup or a closed Testpilot Run after process loss.

### Selection, ownership, and reporting

The supported catalog initially contains exactly `nexus-success`. Operator configuration supplies
an installation identity, pinned artifact digest, explicit environment bindings, and caller-owned
connection/provisioning policy. A production catalog may define its own public ProfileSpec policy;
it cannot import the functional test fixture helper or infer new product assertions from a Case.

The command surface is `list`, `apply`, `worker`, `pause`, `resume`, and `status`. `apply` receives
the complete explicit selection; empty selection pauses previously owned checks. Validate every
selection and inspect every existing schedule conflict before mutation. Stage new schedules paused
and enable only after the whole selection has validated and staging succeeds. Reapply preserves
operator/failure pauses; only explicit `resume` can clear them. Revalidate ownership and expected
configuration inside updates. Network failure can leave partial state: report exact acknowledged
mutations and uncertainty, never claim reconciliation is a cross-schedule transaction.

Schedule ownership combines stable installation/check IDs with a versioned configuration identity.
Artifact or binding changes require explicit update input. Neither apply nor resume triggers an
immediate run or backfill. Removing a selection preserves retained measurements and unrelated
schedules. Pause scheduling before operator cancellation or shutdown and report an unacknowledged
pause as uncertain. Server pause-on-failure remains necessary when no worker is available to act.

Use the existing immutable artifact publication owner under `tools/common/artifactio`. Retain the
actual Run and measurement metadata together under an immutable reference. Default limits are
16 KiB for Workflow summaries, 8 MiB per measurement, 128 retained measurements, and 256 MiB total.
Bound decoding and encoding; reserve count/byte capacity before dispatch and reject exhaustion
without executing again or overwriting prior measurements. Coordinate capacity reservations across
concurrent writers to the same owned sink, or reject a second sink owner explicitly; a Go mutex
alone does not protect separate processes. No new distributed lease service is needed.

Reserve a durable local measurement identity before invoking Run. Duplicate or unresolved claimed
identities cannot dispatch again; a retained identical result can be read without re-execution.
Crash leftovers remain accounted for until explicit reconciliation. Reclaiming storage does not
authorize another attempt, and these local rules do not promise exactly-once external effects.

Publication failure is an operational measurement/reporting failure and causes failed orchestration
to pause scheduling. Preserve any authoritative Run already produced and distinguish publication
ambiguity from absence. A late durable publication cannot change a timed-out Workflow's outcome,
overwrite another measurement, or cause redispatch. Full Run data, credentials, and opaque effect
capabilities never enter Workflow history; producer provenance there is bounded and digest-bound.

### Time and resource budgets

The current Case allows 30 seconds of execution and three distinct 5-second termination, cleanup,
and Close phases. Budget 45 seconds before reporting, not merely execution plus one cleanup phase.
Default Activity StartToClose is 60 seconds, ScheduleToClose 75 seconds, and Check Workflow execution
and run timeouts 90 seconds. Publication has a 10-second bound. Heartbeat every 5 seconds with a
20-second HeartbeatTimeout; wait for Activity cancellation so live-worker cleanup can finish.
Recompute the budget from fn-77's delivered Case and reject incompatible larger limits rather than
silently tightening execution semantics. Target server lifetime is independently bounded as above.

Use worker concurrency one for the initial one-check catalog, bounded pollers, and no local dispatch
queue. Drain within 60 seconds and close worker-owned clients/Driver resources with an explicit
final bound. Status reports actual scheduled/started/completed timestamps and treats a result older
than three minutes as stale; a fresh violated result remains violated. A stopped worker, unreachable
cluster, missing result, stale result, and failed publication remain distinct from Verdict.

The local deployment uses dedicated orchestration resources and explicitly configured CHASM
creation, 100% creation rollout, routing, required sentinel setup, and the cancellation/termination
pause setting. Use the existing server configuration and test-cluster mechanisms; do not change
global defaults, migrate unrelated schedules, or deploy infrastructure as part of writing this plan.
Verify canceled and terminated runs automatically pause and allow no subsequent tick before explicit
resume. A manual status warning alone does not satisfy R8.

## Edge Cases & Constraints
<!-- scope: technical -->

- Replay: workflow replay must not invoke Testpilot or duplicate external work. Activity results, including violated/inconclusive Verdicts, are replayed as data.
- Process loss/timeouts: the Activity may have caused effects even if its response is lost. Mark the measurement uncertain, propagate orchestration failure and configure the schedule to pause on such failure/timeouts. No automatic redispatch or automatic resume. The operator inspects and reconciles exact owned resources before resuming. This prototype needs no general recovery service.
- Cancellation/cleanup: cancellation reaches Testpilot's bounded cleanup path when the worker is alive. Give the Activity enough time for execution plus cleanup and the Workflow enough time for the Activity terminal result. Target workflows/resources need finite server-enforced lifetimes as a backstop when the Activity worker disappears. A timeout is not evidence that all effects stopped; schedule Skip alone cannot establish resource isolation.
- Shared process: reuse immutable prepared Cases and driver registrations, but each measurement has distinct Run identity, Slots, captures, capabilities and resource ownership. A late result cannot mutate another measurement. Cap worker concurrency explicitly for the small configured selection; a 10x tick/load increase cannot create an unbounded dispatch queue or retained-output growth.
- Dependency failure: scheduling and target execution may use the same local cluster for the demonstration. A cluster outage can prevent measurement creation. Status inspection must expose last scheduled/started/completed observations and freshness separately from Verdict. Unreachable or stale status is never healthy. This is inspection by an external operator/client, not a claim that an unavailable cluster can report its own outage.
- Resource portability: reuse the same exact Case bytes across two distinct namespace/task-queue bindings. Binding changes alter the prepared/Driver binding identity, but not the source artifact, checked success requirement, Contract, provenance, or correlation interpretation. Unsupported bindings fail explicitly.
- Shutdown: stop new owned scheduling when requested and drain/close worker-owned resources within configured bounds. Do not clean unrelated workflows or queues. Preserve independent cleanup and Verdict outcomes.
- Schema evolution: consume the Testpilot protocol and fn-68 Producer as evolved by the binding and typed temporal authoring prerequisites. This spec owns no protocol extraction, expression-language redesign, temporal semantics, evidence projection, or binding wire change.

## Acceptance Criteria
<!-- scope: both -->

- **R1:** `tools/canary` exposes only explicitly selected supported checks, initially Nexus success. Unknown/duplicate names, empty selection, stale or mismatched artifacts and unauthorized/incompatible bindings are covered by tests; invalid selections create no active schedule or target Run.
- **R2:** Canary and the existing functional test consume the fn-68 Producer's checked success behavior through the same Testpilot facade. Source IDs/fingerprints and exact Known Gaps remain bound to checked inputs. No duplicated monitor, Nexus assertion, scenario interpreter, or runtime Lean invocation exists in canary.
- **R3:** One shared Temporal Driver serves both consumers from outside `tests/`, preserving server/worker/delivery ownership. Dependency checks show canary and shared Driver production code import neither `tests/`, Umpire generator tools nor private Testpilot execution/verification packages. Canary integration preserves the prerequisite Driver guarantees; its extraction and migration tests remain owned by `fn-72-extract-the-reusable-temporal-testpilot`.
- **R4:** The same checked selection executes against two explicit namespace/task-queue configurations using one binding mechanism and isolated run-owned resources. Neither consumer rewrites prepared requests or independently hard-codes corresponding worker/request bindings; mismatch tests fail before check dispatch.
- **R5:** Applying an explicit selection creates one stable, 60-second, Skip-overlap Schedule per check, starting a fresh Workflow for each measurement. Reapply is idempotent, conflicts reject, removal pauses only owned schedules, and pause/resume work. A check exceeding the interval does not overlap or accumulate buffered starts; catch-up is bounded and tested.
- **R6:** Workflow orchestration contains only deterministic SDK operations and calls a bounded Activity for Testpilot execution. Workflow execution retries and check Activity retries are disabled. Replay tests show no duplicate external dispatch; a violated or inconclusive measurement is retained without an automatic rerun.
- **R7:** Repeated completed measurements retain distinct Run/resource identities and unchanged semantics, with separate disposition, Verdict, cleanup and bounded artifact references. Cross-run isolation, late completions, reporting failure and retention exhaustion have focused tests; none overwrites a prior measurement or causes redispatch.
- **R8:** Activity worker loss, cancellation and timeouts cannot fabricate Run closure or success. Uncertain orchestration failures pause the schedule until explicit resume; bounded cleanup and finite target resource lifetimes are demonstrated. Status inspection distinguishes missing/stale measurements and unavailable orchestration from actual violated/inconclusive Verdicts.
- **R9:** A live local demonstration runs the existing Nexus success functional test and at least two successive scheduled canary measurements through the shared Driver. Successful canary Runs have completed disposition, satisfied Verdict, successful cleanup and the Producer-defined correlated history support. Assert event-driven results rather than fixed test sleeps; use a generous bounded deadline for the real minute cadence.
- **R10:** Documentation under `tools/canary` gives reproducible build/artifact provisioning, explicit selection, worker, schedule apply/pause/resume/status and cleanup commands; names required environment settings, finite limits, retry/overlap behavior, result storage and the same-cluster freshness limitation. It clearly identifies the non-production proof-of-concept scope and fn-68 dependency.

## Boundaries
<!-- scope: business -->

No production deployment, customer traffic, release qualification, protected GitHub workflow, general lease/fencing/recovery platform, dashboard, alerting service, remote result store, or fn-29 receipt machinery. No arbitrary scenario upload, discovery, randomized exploration, cancellation-model expansion, fault injection, new runtime opcode, alternate evaluator, or universal Property compiler. No unrelated Lean authoring redesign, blanket codec migration, broad generator changes, new third-party libraries, or CI expansion. Shared Driver extraction and environment-binding implementation are owned by `fn-72-extract-the-reusable-temporal-testpilot` and `fn-73-explicit-environment-binding-for`, respectively; do not duplicate that work here. Driver activation-interface deepening and Lean semantic refactors remain independent work. This spec authorizes design/implementation scope; actual infrastructure deployment is not performed by creating the spec.

The three prerequisite amendments above are included solely to satisfy existing R1/R4/R8. No
other scheduler policy, public runtime interface, Producer scenario, or server configuration change
is authorized by this plan. General artifact administration and production recovery remain outside
scope; exhaustion is an explicit pause requiring operator storage management.

## Decision Context
<!-- scope: both -->

- User requested `tools/canary` as a second Lean-model consumer, with a very small manually selected scenario set, repeated starts once per minute, and a Temporal Workflow for each check.
- Reuse fn-68's checked success lowering and live proof rather than create competing integration work. This spec depends on fn-68, `fn-72-extract-the-reusable-temporal-testpilot`, `fn-73-explicit-environment-binding-for`, and `fn-78-typed-temporal-authoring-and-checked`; re-anchor on their final interfaces, artifact identities, semantic contracts, and metadata policy before implementation.
- Temporal Schedules are preferred over a local ticker because scheduling is part of the requested demonstration and remains inspectable/durable. Fresh scheduled Workflows are preferred over an endless per-check workflow because each measurement has an independent orchestration record and fixed start cadence.
- Execute the Case within an Activity to keep runtime I/O out of Workflow replay. Keep the same Driver in tests and canary; consumer differences belong in orchestration, bindings and reporting.
- Prefer the smallest complete success slice. Add a second scenario only through a later request; a new generic framework is not the acceptance test.
- The prototype intentionally differs from fn-29's manual-only production controller. Neither spec supersedes the other; do not inherit production gates or claim production readiness from this demonstration.
- Model-to-Case continuity is owned by fn-68, shared Driver placement by `fn-72-extract-the-reusable-temporal-testpilot`, environment binding by `fn-73-explicit-environment-binding-for`, and scoped temporal semantics and evidence projection by `fn-78-typed-temporal-authoring-and-checked`. The standalone Lean protocol is a transitive prerequisite through binding. Activation-interface deepening and Lean Target/inventory refactors do not block this proof unless an evidenced integration requirement changes that decision.

Honoring the existing CHASM pause setting is preferable to adding a worker-side recovery service:
the server can observe terminal cancellation or termination even when the worker cannot run code.
Operator-only warnings were rejected because they permit another tick after an uncertain run.
The default remains unchanged; this prototype explicitly enables and qualifies the setting in its
dedicated local namespace. Broad generated-API drift/CI work remains excluded under
[the existing decision](../memory/declined/generated-api-drift-verification.md); focused Producer
fixture and compatibility checks remain required.

### Delivery units

| Task | Deliverable | Dependencies |
| --- | --- | --- |
| 1 | Public Driver validation without execution and preflight reuse | — |
| 2 | Producer-owned finite target lifetime and compatibility qualification | — |
| 3 | Existing CHASM cancellation/termination pause setting honored and qualified | — |
| 4 | Pinned catalog/Profile and whole-selection admission | 1, 2 |
| 5 | Immutable bounded measurement storage and capacity admission | 4 |
| 6 | Activity/shared Driver composition and early real two-Run/two-binding proof | 4, 5 |
| 7 | Deterministic Check Workflow, replay, and uncertain-outcome handling | 6 |
| 8 | Owned Schedule reconciliation and operator commands/status/shutdown | 3, 7 |
| 9 | Real minute-cadence, cross-consumer/failure qualification, and operator runbook | 8 |

These are cohesive implementation units; each includes its own negative tests. Source overlap and
shared test/build resources may require serial execution despite independent dependency edges.
Before cadence is added, the Activity entry point must execute two isolated measurements and the
alternate binding through the shared Driver. Final qualification runs the existing functional
consumer and at least two successive real 60-second schedule ticks with named run/pass/no-skip
receipts; synthetic or unmatched test selections cannot stand in for that evidence.

## Early proof point

Before adding schedules, prepare a Nexus success-produced Case through the non-test shared Driver and execute two isolated measurements using the canary Activity entry point. Compare with the functional consumer and demonstrate the alternate namespace/queue binding. If this requires canary-specific request interpretation or a copied Contract, report the unmet prerequisite acceptance criterion and resolve it in its owning spec before adding cadence.

## Verification

The shared package path is established at `common/testing/testpilot/temporal`. All Go tests use `-tags test_dep`; only live tests add `integration`. Run the existing Testpilot facade/driver tests, scheduler reconciliation and Workflow replay tests, negative failure/retention tests, and the real local two-tick demonstration. Run Lean checks and owner-managed fixture generation/drift checks if Producer inputs/artifacts change. Finish implementation with project-required `make lint-code` and applicable model gates. Report environment failures separately; do not weaken assertions or silently skip the live acceptance proof.

## References

- `.plans/UMPIRE4_SPEC.md` — semantic authority, Testpilot facade and Driver ownership.
- `.plans/UMPIRE_ARCHITECTURE_REVIEW.md` — findings 1, 3, 4 and 5.
- `fn-68-minimal-nexus3-success-demonstration` — checked lowering and existing fixture/live integration.
- `fn-72-extract-the-reusable-temporal-testpilot` — reusable Temporal Driver extraction.
- `fn-73-explicit-environment-binding-for` — explicit shared environment binding.
- `fn-78-typed-temporal-authoring-and-checked` — typed temporal semantics, lowering, and evidence projection.
- `fn-29-bounded-production-canary-execution-and` — separate production qualification scope.
- `common/testing/testpilot/temporal/README.md`, `tests/testcore/testpilot/README.md`, and `common/testing/testpilot/README.md` — existing runtime and fixture boundaries.
- `tests/testpilot_async_nexus_case_test.go` — existing live consumer to preserve.
- Temporal Go schedule documentation: https://github.com/temporalio/documentation/blob/main/docs/develop/go/workflows/schedules.mdx. Verify details against the repository's pinned SDK during implementation.

## Requirement coverage

| Requirement | Tasks |
| --- | --- |
| R1 | fn-70-scheduled-canary-proof-of-concept-as-a.1, fn-70-scheduled-canary-proof-of-concept-as-a.4, fn-70-scheduled-canary-proof-of-concept-as-a.8 |
| R2 | fn-70-scheduled-canary-proof-of-concept-as-a.2, fn-70-scheduled-canary-proof-of-concept-as-a.4, fn-70-scheduled-canary-proof-of-concept-as-a.6, fn-70-scheduled-canary-proof-of-concept-as-a.9 |
| R3 | fn-70-scheduled-canary-proof-of-concept-as-a.1, fn-70-scheduled-canary-proof-of-concept-as-a.4, fn-70-scheduled-canary-proof-of-concept-as-a.6, fn-70-scheduled-canary-proof-of-concept-as-a.9 |
| R4 | fn-70-scheduled-canary-proof-of-concept-as-a.1, fn-70-scheduled-canary-proof-of-concept-as-a.2, fn-70-scheduled-canary-proof-of-concept-as-a.4, fn-70-scheduled-canary-proof-of-concept-as-a.6, fn-70-scheduled-canary-proof-of-concept-as-a.9 |
| R5 | fn-70-scheduled-canary-proof-of-concept-as-a.3, fn-70-scheduled-canary-proof-of-concept-as-a.8, fn-70-scheduled-canary-proof-of-concept-as-a.9 |
| R6 | fn-70-scheduled-canary-proof-of-concept-as-a.7, fn-70-scheduled-canary-proof-of-concept-as-a.8, fn-70-scheduled-canary-proof-of-concept-as-a.9 |
| R7 | fn-70-scheduled-canary-proof-of-concept-as-a.5, fn-70-scheduled-canary-proof-of-concept-as-a.6, fn-70-scheduled-canary-proof-of-concept-as-a.7, fn-70-scheduled-canary-proof-of-concept-as-a.9 |
| R8 | fn-70-scheduled-canary-proof-of-concept-as-a.2, fn-70-scheduled-canary-proof-of-concept-as-a.3, fn-70-scheduled-canary-proof-of-concept-as-a.5, fn-70-scheduled-canary-proof-of-concept-as-a.6, fn-70-scheduled-canary-proof-of-concept-as-a.7, fn-70-scheduled-canary-proof-of-concept-as-a.8, fn-70-scheduled-canary-proof-of-concept-as-a.9 |
| R9 | fn-70-scheduled-canary-proof-of-concept-as-a.9 |
| R10 | fn-70-scheduled-canary-proof-of-concept-as-a.9 |
