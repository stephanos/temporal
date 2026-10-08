# Activity coverage

> HTML render lens: `.flow/artifacts/fn-129-activity-coverage/spec.html` (local only; open from the checkout). Regenerable; markdown is the record. <!-- flow-next:artifact-link -->

## Goal & Context

Extend standalone activity coverage with heartbeat, by-ID answers, reset and useful bounded exploration. The owner approved R1-R5 on 2026-10-05 from the activity comparison recommendations P2-5, P2-6, P2-7 and P3-13. This fresh plan follows integrated fn-138.3 source and its separately sealed original/adopted comparison. The active activity batch retains one shared closure boundary.

The current form uses separate client RPC actions and token worker answers with `Failure.fatal` and `Failure.retryable`. Its independent delivery count starts at zero; public Describe counts scheduling. Retry and Deadline capabilities are integrated. Realizations include the ordinary activity, timeout retry and admission races. New coverage must preserve those distinctions and the original expectations.

## Acceptance Criteria

- **R1:** Heartbeat adds `worker.heartbeat` while the attempt is held, an armed `deadline.heartbeat` that retries under the existing policy, and a typed raw details Observation where Describe can read it. SDK invocation and server receipt have separate evidence. Errors and boundaries include unheld/closed requests, unarmed timeout, missing receipt/details, wrong timeout type, crossed operation or attempt, unsupported worker sequences and exhausted policy; none establishes success.
- **R2:** A service actor supplies completion, failure and cancellation by activity ID through their typed unary APIs, with independently authored phase and rejection rows. Scheduled/paused force completion differs from answers requiring a held attempt. Errors include absent/terminal execution, missing held attempt, cancellation without a request, crossed namespace/activity/run, malformed request and missing visibility. A scheduled no-poll witness must have no SDK worker entrypoint.
- **R3:** Reset carries `keepPaused`, records a deferred reset while an attempt is held and applies it on eligible settlement. Properties cover keep-paused and Cancel > Reset > Pause. Completion can win; schedule-to-close stays terminal. Actual SDK attempt rewind, delivery ordinal and independent Model count remain distinct. Errors include terminal/absent execution, conflicting pending controls, a distinct repeated reset, unsupported identity/publication order and invalid numbering declarations; rejected requests leave state unchanged.
- **R4:** Activity find Queries gain bounded `.explore` only where it adds witnesses missed by pinned Scenarios. Each examined Query records added witnesses, search work and candidate lowering standings; Queries with no added evidence remain pinned. Errors include invalid variation domains, lost final action, unrealisable candidates and reached bounds; they supply no coverage credit.
- **R5:** Every new behavior is realized, every R1-R4 requirement maps to at least one live Query/Case, and the final done summary lists new Cases. One shared generation, full gates, independent implementation and completion reviews, generated-Case live invocation and offline replay serve fn-128.6/fn-129.5. Errors include unsupported/missing Cases, mismatched identities, unexpected assessment/status/reason, failed cleanup, resource limit or incomplete evidence; source-only completion cannot close the spec.

## Early proof point

Task fn-129-activity-coverage.1 proves that one admitted heartbeat prefix and pending disposition drain one reservation without an SDK answer RPC, then execute the second delivery under its own truthful identity. If the existing typed Program/Profile/VM seams cannot support this bounded protocol, return the concrete missing contract to the conductor before fn-129.2+; do not introduce a generic worker framework.

## Execution and evidence boundary

The existing five task IDs, priorities and `.1 -> .2 -> .3 -> .4 -> .5` dependencies remain fixed. Tasks .1-.4 produce original source commits, focused Model/native bridge regressions, seeded negative proofs and scratch current-source lift/lower results. Necessary changed internal-proto native mirrors may be generated for source compilation. Production Model IR, Cases and managed fixtures remain unchanged until the shared boundary.

After .4 integrates, fn-128.7 and fn-128.8 perform their focused corrections in parallel with disjoint ownership, re-anchoring the final source and exact Profile shape. The conductor joins both before the conjunctive fn-128.6/fn-129.5 boundary. Preserve fn-138's sealed comparison separately from intentional fn-129 coverage and later fatal-evidence changes. A cross-spec source gate supplements Flow metadata; waiting for prerequisite activity specs to close would deadlock this batch.

Task .2 requires completion, failure and cancellation live Cases under R5. Its held paths add the smallest explicit typed external-control settlement basis and bounded pending-publication handoff; a valid external-control basis admits no selected timer, while missing/unknown basis rejects. R1's heartbeat timeout basis still requires its positive timer and selected occurrence. Task .3 consumes the proven publication seam, retaining a timer-settled deferred-reset witness. The shared conductor requires separate fn-128/fn-129 completion reviews and separate fn-138 completion review/closure after its independent R3 seal plus final shared gates.

The final suite compares authored disposition, cleanup, Contract, conformance, Property and every reason by equality, live and replayed. Retain the existing retry, retryAfterTimeout and retryExhaustion Property-only `explanationsDisagree` expectations exactly. Fatal Property SATISFIED, satisfied Contracts, conformant retry paths and bounded pauseResume execution/replay remain required. Only the specifically named ShutdownWorker race permits an additional inconclusive result.

## Decision Context

- Reuse typed unary RPC authoring and transport for by-ID/reset. Heartbeat, held external settlement and reset need only their bounded typed native bridge extensions. Rejected generic activity interpreter, reset epoch and driver framework as outside the requested coverage.
- SDK `RecordHeartbeat` returns void and discards transport errors. Its local outcome establishes invocation; bounded Describe evidence establishes receipt and HEARTBEAT timeout. Raw Payloads observation supplies no correlated payload-equality claim.
- An empty activity script still registers an SDK worker. The scheduled no-poll Case uses the existing controller-only realization with no workers.
- The inherited retryable-failure-under-cancellation disagreement remains recorded in fn-138. It is known source context, not new live evidence. The independent Model must never be fitted to Go/server behavior; an actual conformance disagreement requires human judgment under AGENTS.
- Preserve `nonRetryableFails`, its ordinary no-reset Scenario/pinned FIND and fatal SATISFIED expectation exactly. Separate independent reset-settlement Properties/witnesses cover reset; scoped universal capability overrides cover complete reset/no-reset transitions without vacuity or shared companion changes. The ordinary FIND never claims reset coverage.
- Broad generated API drift verification and new CI coverage remain declined in `.flow/memory/declined/generated-api-drift-verification.md`. Required focused schema, fixture and identity checks remain in scope.

## Boundaries

- Standalone activity only. Workflow-scheduled activity fn-119, deferred fn-125/fn-130 and captured fn-149/fn-150 remain outside this batch.
- Upstream Go Model and harness remain comparison inputs. No upstream edits, new dependency, generic recorder/control subsystem, default Profile widening or canary-policy change.
- Preserve downstream order fn-142 -> fn-143 -> fn-140 -> fn-123 -> fn-145 -> fn-146 -> fn-147 -> fn-148 -> fn-141 last. This plan refresh starts no tasks, marks none done, closes no spec and changes no MILESTONES entry.

## Quick commands

```bash
python3 /home/agent/.codex/scripts/flowctl.py validate --spec fn-129 --coverage --json
go test -tags test_dep -p 2 -timeout 30m ./common/testing/testpilot/temporal/worker -run 'Activity|Transport'
```

Task Quick commands are scoped source checks. The final shared boundary owns production generation, complete suites and the single live invocation.

## Requirement coverage

| Req | Description | Task(s) | Gap justification |
| --- | --- | --- | --- |
| R1 | Heartbeat adds `worker.heartbeat` while the attempt is held, an armed `deadline.heartbeat` that retries under the existing policy, and a typed raw details Observation where Describe can read it. SDK invocation and server receipt have separate evidence. Errors and boundaries include unheld/closed requests, unarmed timeout, missing receipt/details, wrong timeout type, crossed operation or attempt, unsupported worker sequences and exhausted policy; none establishes success. | fn-129-activity-coverage.1 | — |
| R2 | A service actor supplies completion, failure and cancellation by activity ID through their typed unary APIs, with independently authored phase and rejection rows. Scheduled/paused force completion differs from answers requiring a held attempt. Errors include absent/terminal execution, missing held attempt, cancellation without a request, crossed namespace/activity/run, malformed request and missing visibility. A scheduled no-poll witness must have no SDK worker entrypoint. | fn-129-activity-coverage.2 | — |
| R3 | Reset carries `keepPaused`, records a deferred reset while an attempt is held and applies it on eligible settlement. Properties cover keep-paused and Cancel > Reset > Pause. Completion can win; schedule-to-close stays terminal. Actual SDK attempt rewind, delivery ordinal and independent Model count remain distinct. Errors include terminal/absent execution, conflicting pending controls, a distinct repeated reset, unsupported identity/publication order and invalid numbering declarations; rejected requests leave state unchanged. | fn-129-activity-coverage.3 | — |
| R4 | Activity find Queries gain bounded `.explore` only where it adds witnesses missed by pinned Scenarios. Each examined Query records added witnesses, search work and candidate lowering standings; Queries with no added evidence remain pinned. Errors include invalid variation domains, lost final action, unrealisable candidates and reached bounds; they supply no coverage credit. | fn-129-activity-coverage.4 | — |
| R5 | Every new behavior is realized, every R1-R4 requirement maps to at least one live Query/Case, and the final done summary lists new Cases. One shared generation, full gates, independent implementation and completion reviews, generated-Case live invocation and offline replay serve fn-128.6/fn-129.5. Errors include unsupported/missing Cases, mismatched identities, unexpected assessment/status/reason, failed cleanup, resource limit or incomplete evidence; source-only completion cannot close the spec. | fn-129-activity-coverage.1, fn-129-activity-coverage.2, fn-129-activity-coverage.3, fn-129-activity-coverage.5 | — |
