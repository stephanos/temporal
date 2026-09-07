# Extract the reusable Temporal Testpilot Driver

> HTML render lens: [.flow/artifacts/fn-72-extract-the-reusable-temporal-testpilot/spec.html](../artifacts/fn-72-extract-the-reusable-temporal-testpilot/spec.html) — regenerable, markdown is the record. <!-- flow-next:artifact-link -->

## Overview

Relocate the existing composite Temporal Testpilot Driver from the functional-test tree into the
shared `common/testing/testpilot/temporal` package. Preserve its complete API, lifecycle, delivery,
authority, and error contracts while keeping generated fixtures, cluster provisioning, and live
test policy in `tests/`.

## Quick commands

```bash
mise exec -- go test -count=1 -tags test_dep ./common/testing/testpilot/temporal/... ./tests/testcore/testpilot
mise exec -- go test -count=1 -tags test_dep ./tools/umpire/regression -run 'Test(TestpilotOwnsCaseProtocolAndRuntime|UmpireCIWorkflowRunsSeparatedUnitAndLiveProofs)'
make umpire-check-live-tests
```

## Goal & Context
<!-- scope: business -->

Make the existing Temporal Testpilot Driver available to functional tests and future environment runners without importing the functional-test harness. The current Driver already owns reusable controller transport, SDK execution, reservation delivery, cancellation, completion capabilities, and quarantine. Its placement under `tests/testcore/testpilot` forces prospective callers to depend on a test tree even though its implementation needs no cluster fixture.

This is architecture fn-72-extract-the-reusable-temporal-testpilot from the approved architecture review. It moves the existing implementation and preserves its contracts; it does not create another Driver. Functional tests remain the concrete consumer demonstrating the extraction. The future canary runner consumes this shared implementation under its own environment and lifecycle policy.

C is independently implementable in the first architecture wave. fn-73-explicit-environment-binding-for (environment binding) and fn-74-deepen-testpilot-worker-activation (activation state) depend on C's ownership boundary. Ongoing fn-68 lowering and live integration work may touch the same consumer or Driver files: coordinate those edits and preserve their latest behavior without making completion of all fn-68 work a prerequisite. The fn-70 canary work consumes C and B and does not own this extraction.

## Architecture & Data Models
<!-- scope: technical -->

The destination is `common/testing/testpilot/temporal`, with Go package name `temporal`. This location is an architectural decision: it is a reusable sibling of `common/testing/testpilot`, follows the repository's existing `common/testing` convention, and lies outside the generic runtime's Go `internal` import boundary. It does not introduce a Go module or third-party dependency.

The destination owns the composite Driver, WorkflowService descriptor Catalog helpers, and the existing `server`, `worker`, and `internal/delivery` children. Server and worker remain peers composed by the parent. The private delivery ledger, carrier codec, ownership records, and cancellation state remain private to this Driver tree. Server does not import worker, worker does not import server, and delivery does not import either adapter. The generic Testpilot facade remains independent of the Temporal Driver.

Production Driver code depends on the public Testpilot facade, its wire API, Temporal API and SDK packages, existing transport dependencies, and its own private packages. Its production dependency closure must not reach repository `tests/`, Umpire generators, or canary orchestration. Driver packages must not directly import the generic Testpilot runtime's private IR, execution, or verification packages; those remain implementation details reached only through the public facade. There is no embedding or runtime loading of retained Case fixtures in the shared Driver.

Move implementation-focused unit tests with their owners so they can still exercise private delivery and SDK lifecycle behavior. Retained Lean-generated functional fixtures remain in `tests/testcore/testpilot/testdata`, along with fixture admission/reuse tests that import the shared Driver's public Catalog helpers. Cluster startup, namespace and Nexus endpoint provisioning, SDK client creation, environment-specific configuration, assertions, and resource cleanup registration remain under `tests/`. Do not duplicate generated fixtures or move their generator ownership merely to accommodate the extraction.

Migration consists of relocating the implementation and its focused tests, changing functional consumers to the shared import, then aligning executable dependency checks, CI/package test selection, and current ownership documentation. Remove the former implementation rather than retaining a forwarding package or second copy. A test-only fixture package may remain at the old location.

## API Contracts
<!-- scope: technical -->

Preserve exported names, signatures, option fields, returned types, supported operations, and error behavior, modulo the new import paths and composite package name. This includes `New(Options) (*Driver, error)`, `Snapshot`, `Identity`, `Open`, `Close`, `NewWorkflowServiceCatalog`, `WorkflowServiceDescriptorSet`, the `Endpoint` and `RoleBinding` aliases, and the server/worker public surfaces used by composition. No new wrapper facade is required.

`Driver` continues to implement the public Testpilot `Profile` and `Driver` interfaces. The execution sequence remains `testpilot.Prepare(case, profile)` then `prepared.Run(ctx, driver)`. Preparation performs no target I/O, and the Driver neither constructs a Contract nor chooses a Monitor.

Preserve the existing configuration contract: callers supply the Profile, authorized server endpoints and credentials, trusted callback base URL and optional HTTP client, SDK client, namespace, worker role, task-queue and Nexus endpoint bindings, and worker stop timeout. Preserve current validation and snapshot behavior. The supplied SDK client remains caller-owned; extraction does not add client dialing or client closure to Driver ownership.

Preserve the current lifecycle: composite `Open` creates the server Session, opens the worker Session only for a prepared Program with worker entrypoints, and closes an already-open controller Session when worker setup fails. Session cleanup invokes both owned components and joins their errors. Transport-only use through the server Driver stays available; extraction does not relax the composite constructor's existing worker configuration requirements. `Driver.Close` retains its existing behavior rather than acquiring new session or SDK ownership semantics.

Only prepared, authorized unary requests reach server transport. Worker execution continues using SDK workflow, activity, and Nexus-handler APIs. Existing carrier injection is restricted to the reserved delivery protocol, preserves application request semantics, checks final request size, and pins the returned Temporal Run ID. Existing mismatch rejection is preserved; environment request rewriting is not added.

## Edge Cases & Constraints
<!-- scope: technical -->

Preserve comments with moved code. Update location-bearing prose only where relocation changes its meaning; unrelated comments and historical completed-work evidence remain untouched.

Construction failure must retain existing cleanup and joined-error behavior. Invalid configuration, nil inputs, closed sessions, unauthorized operations, duplicate identities, registration conflicts, capacity exhaustion, and context cancellation retain their present rejection behavior. Extraction must not convert transport or cancellation errors into successful outcomes.

The in-memory delivery ledger preserves exact Run/session/activation identity, replay admission, duplicate rejection, late delivery handling, cancellation retries, and quarantine accounting. Capacity is released only according to existing actual-completion rules. The move adds no persistence or recovery promise after process crash, no reconnect/retry policy, and no scheduler. At ten times offered load, existing finite capacities and rejection behavior remain the constraint; there is no new buffering or throughput target.

Preserve separation of Run disposition, cleanup status, and Verdict, including proved violation after cleanup failure, and immutable results after Run return. Profile, request, outcome, and capability ownership must not become weaker because code has become reusable.

Align the normative MOD-13 owner locations in the Umpire 4 specification to the new server and worker packages, preserving its authority split and existing stable rule ID. Align active package READMEs and executable documentation checks to the same relocation. This is a location correction authorized by the extraction, not a relaxation of MOD-12, MOD-14, EVD-17, or other behavior rules. The architecture review remains a historical account of the pre-extraction state.

## Acceptance Criteria
<!-- scope: both -->

- **R1:** The composite Driver, descriptor Catalog helpers, server, worker, and private delivery implementation exist only under `common/testing/testpilot/temporal`; former runtime implementation imports are removed from active consumers. Implementation-focused unit tests move with their owners; test fixtures and provisioning remain under `tests/`. Errors: build/import failures or a remaining production dependency on the former implementation fail acceptance; no new runtime error surface.
- **R2:** An external-package consumer compiles against the shared composite Driver and public Catalog helpers, and interface conformance to Testpilot `Profile` and `Driver` is checked. Existing exported contracts and configuration/client ownership remain unchanged except import paths and the parent package name. Errors: existing invalid configuration and nil/context rejection cases retain their categories and behavior; extraction introduces no string-parsing error API or new configuration requirement.
- **R3:** Executable dependency checks cover the new Driver tree and prevent its production dependency closure from reaching repository `tests/`, Umpire generators, or canary orchestration; direct Driver imports of generic Testpilot private packages are forbidden while ordinary transitive reach through the public facade remains valid. Checks also preserve server/worker/delivery authority direction and generic-runtime independence. Add negative test inputs proving forbidden edges are rejected. Errors: forbidden direct or transitive dependencies and unreadable/unparseable inspected source fail the gate, rather than silently omitting coverage.
- **R4:** Focused tests demonstrate preserved lifecycle and ownership through construction/open failure cleanup, shared versus conflicting worker registrations, controller/worker dispatch separation, cancellation before and during admission, cancellation retries, bounded Close, quarantine retention until actual completion, and cross-Run delivery isolation. Reuse the existing corresponding tests and add coverage only for a relocation-exposed gap. Errors: setup, close, cancellation, transport, and capacity failures retain their existing outcomes and do not leak successfully created resources or admit foreign work.
- **R5:** Existing carrier and completion-capability tests pass from the shared location, including request preservation, foreign physical binding rejection, header collisions and size limits, replay/late completion, system callback validation, and late diagnostics. Errors: malformed or crossed delivery, untrusted callback authority, rejected triggers, and completion after closure remain rejected or quarantined as currently specified, without mutating a closed Run or its Verdict.
- **R6:** Existing fixture admission and prepared-Case reuse/correlation tests remain under the test tree and consume the shared public Driver helpers. Canonical retained fixture bytes and their generator destination remain unchanged by C. Errors: static rejection still occurs before Driver I/O, correlation failures do not establish success, and missing or invalid fixtures fail tests; tests do not invoke Lean or rewrite fixtures.
- **R7:** The existing live `TestTestpilotAsyncNexusCase` consumes the shared Driver and passes with its existing completed Run, satisfied Verdict, and correlated history-only evidence assertions. Namespace/endpoint creation, SDK client ownership, and bounded cleanup remain caller-owned in the functional test. Preserve any fn-68 live successor present at implementation time using the same migration. Errors: resource provisioning, execution, evidence, or cleanup failure fails the live test; compilation or mocked execution alone does not satisfy this criterion.
- **R8:** Current documentation, MOD-13 owner locations, package test commands, CI selection, vocabulary scans, and architecture regression checks recognize the new ownership boundary, while still covering retained functional fixture tests. Errors: stale active paths that omit the shared package or contradict its ownership fail checks; no error surface beyond document/check alignment and existing tool failures.
- **R9:** Focused shared Driver, generic Testpilot, retained fixture, and affected architecture regression suites pass with `-tags test_dep`, followed by the existing Umpire Go gate with the new package included and `make lint-code`. Run the live functional check using the repository's established persistence configuration and test flags; report that evidence separately. Errors: a missing compiler, SDK dependency, persistence service, or resource limit is an explicit failed/blocked verification step, never a passing check or permission to weaken assertions.

## Boundaries
<!-- scope: business -->

- No activation-state redesign or generic interpreter API deepening; fn-74-deepen-testpilot-worker-activation owns those changes after C.
- No symbolic environment binding, request-literal migration, or environment portability claim; fn-73-explicit-environment-binding-for owns those changes after C.
- No standalone Lean Testpilot IR/codec extraction or Lean semantic refactoring.
- No model-to-Case lowering or expansion of fn-68's semantic scope.
- No canary scheduler, CLI, deployment, credentials acquisition, environment provisioning, or canary policy implementation.
- No public delivery library, additional Driver implementation, replacement resident service, new persistence, or retry semantics.
- No new Testpilot wire schema, Contract behavior, Monitor access, third-party dependencies, or fixture regeneration.

## Decision Context
<!-- scope: both -->

A sibling package under `common/testing` makes the reuse intent clear while preserving Go's structural prohibition on importing the generic runtime's private packages. Placing the Driver beneath the generic Testpilot package would make those imports technically possible; leaving it under `tests/` preserves the dependency problem. Exporting delivery independently would expose protocol-sensitive state without a second independent consumer.

A coordinated move preserves the already-tested deep delivery module and server/worker authority split. It incurs import and path-gate churn but adds no runtime layer, allocation, network call, concurrency, or authorization surface. Keeping fixtures and provisioning in the test tree avoids coupling the reusable Driver to Lean production and functional-cluster setup. Keeping current binding and activation behavior limits this change to ownership, making later B and D changes easier to review independently.

The contract is grounded in the approved Umpire architecture review's reusable-Driver finding and Umpire 4's MOD-08, MOD-12 through MOD-14, ART-10, EVD-14 through EVD-17, and QLF-05 rules. Existing SDK, transport, delivery, fixture, and live tests supply regression evidence; no broader canary-readiness or model-refinement claim follows from a successful extraction.

## Early proof point

Task fn-72-extract-the-reusable-temporal-testpilot.1 proves that the complete Driver tree can move
under `common/testing` without weakening Go `internal` visibility or changing its focused tests. If
that atomic relocation fails, re-evaluate the destination boundary before migrating consumers or
rewriting ownership gates.

## Requirement coverage

| Req | Description | Task(s) | Gap justification |
| --- | --- | --- | --- |
| R1 | Sole shared implementation ownership and retained test-only fixture tree | fn-72-extract-the-reusable-temporal-testpilot.1, fn-72-extract-the-reusable-temporal-testpilot.2 | — |
| R2 | Public API and interface compatibility from an external consumer | fn-72-extract-the-reusable-temporal-testpilot.2 | — |
| R3 | Direct and transitive dependency-direction enforcement | fn-72-extract-the-reusable-temporal-testpilot.3 | — |
| R4 | Composite lifecycle, cancellation, quarantine, and isolation preservation | fn-72-extract-the-reusable-temporal-testpilot.1 | — |
| R5 | Carrier and completion-capability behavior preservation | fn-72-extract-the-reusable-temporal-testpilot.1 | — |
| R6 | Retained fixture admission, reuse, and correlation | fn-72-extract-the-reusable-temporal-testpilot.2 | — |
| R7 | Live async Nexus consumer on the shared Driver | fn-72-extract-the-reusable-temporal-testpilot.2 | — |
| R8 | Normative ownership, active docs, and package-selection alignment | fn-72-extract-the-reusable-temporal-testpilot.3, fn-72-extract-the-reusable-temporal-testpilot.4 | — |
| R9 | Focused, aggregate, lint, and separately reported live verification | fn-72-extract-the-reusable-temporal-testpilot.4 | — |

## References

- `.plans/UMPIRE4_ORDER.md` — first architecture wave and downstream sequencing.
- `.plans/UMPIRE4_SPEC.md` — MOD-12 through MOD-14 ownership contracts.
- `.flow/memory/bug/integration/moved-conformance-tests-must-not-import-2026-09-06.md` — generic runtime dependency direction.
- `.flow/memory/bug/integration/full-integration-gates-must-select-the-2026-09-04.md` — complete package and live-test selection.
- `.flow/memory/bug/integration/behavior-neutral-refactors-must-not-2026-09-04.md` — relocation compatibility discipline.
