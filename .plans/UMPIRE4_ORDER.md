# Umpire 4 delivery order

Build on the completed Testpilot and model authoring cutovers. Flow owns task status;
this document records delivery order. Architecture and terminology live in the
[Umpire 4 specification](UMPIRE4_SPEC.md).

## Current work

### 1. First architecture wave — fn-71 and fn-72

These specs have no hard prerequisites and can proceed together. Coordinate changes to Producer
imports, functional consumers, and path-based gates; neither spec owns Nexus3 lowering.

| Spec | Deliver |
| --- | --- |
| [fn-71 — Standalone Lean Testpilot protocol](../.flow/specs/fn-71-standalone-lean-testpilot-protocol.md) | Independent `Testpilot.*` types, context-safe expressions, one current codec, and producer-owned Umpire provenance. |
| [fn-72 — Shared Temporal Driver](../.flow/specs/fn-72-extract-the-reusable-temporal-testpilot.md) | Move the existing Driver to `common/testing/temporaltestpilot`, preserve server/worker/delivery ownership, and migrate functional consumers without changing behavior. |

### 2. fn-73 — Explicit environment binding

[Spec](../.flow/specs/fn-73-explicit-environment-binding-for.md).
Deliver after **fn-68, fn-71, and fn-72**.

Own the shared symbolic resource-binding contract, its protocol/preparation changes, and static
Driver validation. Prove that the same checked Nexus3 Case runs against two environment bindings
without request rewriting or changed Contract meaning. Preserve legacy literal-only Case behavior.

### 3. fn-70 — Scheduled canary proof of concept

[Spec](../.flow/specs/fn-70-scheduled-canary-proof-of-concept-as-a.md).
Deliver after **fn-68, fn-72, and fn-73**; fn-71 is a transitive prerequisite through fn-73.

Implement the second consumer under `tools/canary`: manual check selection, a fresh scheduled
Workflow each minute, Activity-owned Testpilot execution, bounded results, and isolated repeated
measurements. Consume the Driver and binding interfaces delivered above; do not repeat their
implementation work. Retain the cross-consumer and two-environment integration proof.

This is a local/development prototype. It does not depend on fn-26 or fn-29 and does not authorize
production deployment or replace fn-29's separately scoped production-canary design.

### Independent architecture tracks

These specs do not block the first canary. Coordinate overlapping imports and interface changes
without adding dependencies solely because files overlap.

| Spec | Dependencies | Deliver |
| --- | --- | --- |
| [fn-74 — Activation semantics and preparation diagnostics](../.flow/specs/fn-74-deepen-testpilot-worker-activation.md) | fn-72 | Private activation-state ownership and public preparation diagnostics; preserve SDK scheduling and the controller execution boundary. |
| [fn-75 — Lean Target semantic seam](../.flow/specs/fn-75-separate-lean-target-semantics-from.md) | None | Separate checked Target semantics from authoring/elaboration machinery while preserving checked construction, fingerprints, and proof trust. |
| [fn-76 — Semantic inventory dependency direction](../.flow/specs/fn-76-make-lean-semantic-inventory-consume.md) | None | Make inventory consume semantic outcome and Known Gap contracts; preserve generated inventory and semantic results. |

The new architecture and canary specs need task breakdown and plan review before implementation.
Their dependency order does not itself mark them ready in Flow.

### DSL semantics and authoring — standalone spec

[Typed temporal authoring and checked scoped monitoring](UMPIRE_DSL_EVOLUTION_SPEC.md) captures
the five recommended changes from the [DSL experiment](UMPIRE_DSL_EXPERIMENT.md). This track is
recorded outside Flow-Next and does not block fn-73 or the first fn-70 canary.

| Stage | Deliver | Order |
| --- | --- | --- |
| D1 — Query validity | Separate satisfiability, trigger coverage, answer, completeness, and endpoint policy. | Can start now. |
| D2 — Existing Behavior surface | Typed occurrence, ordering, adjacency, and bound forms; preserve exact regressions. | Alongside D1. |
| D3 — Event/evidence boundary | Distinguish command submission from confirmed events; bounded correlated projection with exact support. | Alongside D1/D2; coordinate the semantic seam with fn-75. |
| D4 — Shared scoped lowering | One obligation semantics, checked Property correspondence, exact Contract lowering, and required generic cancellation support. | After D3; production integration uses fn-71 and fn-72. |
| D5 — Typed temporal authoring | Readable bounded temporal notation, complete Nexus cancellation Case, and online/offline/live qualification. | After D1/D2/D4. |

Preserve the completed fn-68 success path during this work. Coordinate with fn-74/75/76 without
absorbing their interface refactors. Keep the finite checker; full Veil adoption, general LTL, and
general scenario algebra remain deferred. The experiment does not substitute for the production
lowering and evidence correspondence required by this spec.

### Typed operations and field-level Properties — fn-77

[fn-77 — Typed operations, parameterized Actions, and field-level Properties](../.flow/specs/fn-77-typed-operations-parameterized-actions.md)
adds direct generated API references, typed SDK-command/event declarations, modeled request/result
values, and cross-field/cross-occurrence requirements. Product assertions remain in checked models;
integration owns faithful execution, observation, and correlation.

| Stage | Deliver | Order |
| --- | --- | --- |
| O1 — Operation/value contracts | Typed references, exact supported field values, and explicit fidelity limits. | Alongside DSL D1/D2; coordinate fn-75. |
| O2 — Parameterized semantics | Action/outcome arguments, field expressions/captures, finite domains, and independent Properties. | After O1; supplies parameterized DSL authoring. |
| O3 — Checked concrete lowering | Typed request construction, result projections, clause coverage, and field-expression correspondence. | After O2; uses fn-71/fn-72 and coordinates with DSL D4. |
| O4 — Field-level qualification | Unary RPC and workflow-owned Nexus examples, field mutations, and real Driver evidence. | After O3; temporal Nexus qualification also uses DSL D3/D4. |

Task breakdown and plan review remain pending. Keep this off the fn-73/first-canary critical path.
DSL D1/D2 and label-only
semantics can proceed independently; parameterized D3/D4/D5 variants consume O1/O2. O3 owns field
lowering, D4 owns scoped temporal lowering, and same-step qualification does not wait for temporal
support. Share value/capture contracts across the tracks rather than creating two representations.

## Downstream delivery

Each spec needs a fresh Testpilot plan review before implementation. Prior reviews of the
retired execution architecture do not approve the rewritten plans. Completed dependencies do not
block replanning or execution.

| Spec | Dependencies | Next action |
| --- | --- | --- |
| [fn-33 — Bounded exploration](../.flow/specs/fn-33-run-serial-bounded-semantic-exploration.md) | fn-40, fn-64, and fn-69 | Review whole-Case candidates, serial coordination through Testpilot, lost iterations, semantic coverage, and bounded 10x behavior. |
| [fn-22 — Replay and reduction](../.flow/specs/fn-22-deterministic-replay-semantic.md) | fn-5, fn-64, and fn-69 | Resolve **MAJOR_RETHINK** before implementation: separate exact candidate identity from Contract-relative violation equivalence, prove the negative Case before reduction, and retain explicit offline semantic replay and checked promotion. |
| [fn-26 — Qualification receipts](../.flow/specs/fn-26-local-qualification-receipts-and-staged.md) | fn-48, fn-64, and fn-69 | Review offline Testpilot Case/Profile/Run/Verdict admission, receipt multiplicity, and idempotent publication. Assessment must never create or replay a Run. |
| [fn-29 — Production canary](../.flow/specs/fn-29-bounded-production-canary-execution-and.md) | **fn-26**; fn-48, fn-64, and fn-69 | After fn-26 ships, review external policy and credentials, serial Testpilot Runs, leases, lost Runs, reconciliation without redispatch, and publication. |

All runtime work uses `testpilot.Prepare(case, profile)` → `PreparedCase.Run(ctx, driver)` and the
server/worker authority split. New scenarios remain Case data; canary policy, credentials,
leases, recovery, and publication stay outside Testpilot and Umpire.

## Completed cutovers

- [fn-68](../.flow/specs/fn-68-minimal-nexus3-success-demonstration.md): proved the approachable
  five-block Nexus3 success model through checked lowering, deterministic fixture generation,
  offline evidence rejection, and the existing real Temporal Driver test.
- [fn-69](../.flow/specs/fn-69-extract-testpilot-from-umpire.md): moved the Case protocol and
  reusable runtime to `common/testing/testpilot`, moved the functional Driver to
  `tests/testcore/testpilot`, refined the Testpilot protobuf model, migrated all consumers, and
  removed the former Umpire protocol/runtime owners.
- [fn-66](../.flow/specs/fn-66-remove-unused-umpire-tooling-after.md): removed the unused public
  Artifact package and CLI plus orphaned internal codecs; retained the narrow Experiment reader,
  Testpilot, generators, regression gates, and downstream contracts.
- [fn-64](../.flow/specs/fn-64-umpire-case-runtime.md): Testpilot, Lean-produced Nexus proof,
  independent six-class facade corpus, and full regression gate.
- [fn-65](../.flow/specs/fn-65-design-and-prototype-approachable.md): separate Nexus2 authoring
  prototype. Its [requirement comparison](../model/Temporal/Feature/Nexus2/EVIDENCE.md) covered fn-62 R3.
- [fn-62](../.flow/specs/fn-62-make-ordinary-temporal-model-authoring.md): remaining established
  authoring requirements, including Observation construction and model-owned Known Gaps;
  [compatibility evidence](../model/Temporal/Feature/Nexus/EVIDENCE.md).

All six passed whole-spec completion review. Nexus2 remains a prototype with explicit adoption
boundaries.

## Deferred and superseded

These entries are outside the delivery queue and are not prerequisites for it.

| Deferred spec | Revisit when |
| --- | --- |
| [fn-60](../.flow/specs/fn-60-deepen-authored-lean-canonical-json.md) | Optional handwritten canonical JSON consolidation becomes worth prioritizing; it has no downstream dependency. |
| [fn-15](../.flow/specs/fn-15-standalone-api-and-config-input-catalogs.md) | Platform completeness is needed beyond the proven model family. |
| [fn-23](../.flow/specs/fn-23-veil-toolchain-compatibility-and.md) | Optional checker adoption becomes valuable. |
| [fn-24](../.flow/specs/fn-24-lean-native-verification-receipts-and.md) | A verification receipt/profile platform is justified. |
| [fn-25](../.flow/specs/fn-25-optional-callerclosure-veil-binding-and.md) | A second verification backend is justified; caller closure remains historical. |
| [fn-30](../.flow/specs/fn-30-release-evidence-graph-and-manual.md) | Real Claim Assessment evidence supports release governance. |

[fn-14](../.flow/specs/fn-14-milestone-a-pilot-baseline-and-lean.md) is historical;
[fn-61](../.flow/specs/fn-61-simplify-the-umpire-go-execution-surface.md) and
[fn-63](../.flow/specs/fn-63-consolidate-umpire-go-tests-into-golden.md) are superseded by fn-64.
Any remaining `todo` children do not reactivate them. Broader test consolidation needs a new
Testpilot proposal with an independent oracle.
