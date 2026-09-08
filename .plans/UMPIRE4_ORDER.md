# Umpire 4 delivery order

Build on the completed Testpilot and model authoring cutovers. Flow owns task status;
this document records delivery order. Architecture and terminology live in the
[Umpire 4 specification](UMPIRE4_SPEC.md).

## Current work

### 1. fn-78 — Typed temporal authoring and checked scoped monitoring

[Spec](../.flow/specs/fn-78-typed-temporal-authoring-and-checked.md).
Deliver after **fn-73** and before **fn-70**.

Own shared scoped-obligation semantics, explicit command/event evidence projection, independently
reported Query validity, typed bounded temporal notation, and the readable surface for existing
Behavior constraints. Preserve the completed fn-68 success path and coordinate with fn-74/75/76
and fn-77 without absorbing their separate interface and typed-operation work.

### 2. Independent architecture tracks

These specs do not block the first canary. Coordinate overlapping imports and interface changes
without adding dependencies solely because files overlap.

| Spec                                                                                                                   | Dependencies | Deliver                                                                                                                                                                     |
| ---------------------------------------------------------------------------------------------------------------------- | ------------ | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| [fn-74 — Activation semantics and preparation diagnostics](../.flow/specs/fn-74-deepen-testpilot-worker-activation.md) | fn-72        | Private activation-state ownership and public preparation diagnostics, including later error classification; preserve SDK scheduling and the controller execution boundary. |
| [fn-75 — Lean Target semantic seam](../.flow/specs/fn-75-separate-lean-target-semantics-from.md)                       | None         | Separate checked Target semantics from authoring/elaboration machinery while preserving checked construction, fingerprints, and proof trust.                                |
| [fn-76 — Semantic inventory dependency direction](../.flow/specs/fn-76-make-lean-semantic-inventory-consume.md)        | None         | Make inventory consume semantic outcome and Known Gap contracts; preserve generated inventory and semantic results.                                                         |

The open architecture tracks and fn-70 need task breakdown and plan review before implementation.
Their dependency order does not itself mark them ready in Flow.

### 3. Typed operations and field-level Properties — fn-77

[fn-77 — Typed operations, parameterized Actions, and field-level Properties](../.flow/specs/fn-77-typed-operations-parameterized-actions.md)
adds direct generated API references, typed SDK-command/event declarations, modeled request/result
values, and cross-field/cross-occurrence requirements. Product assertions remain in checked models;
integration owns faithful execution, observation, and correlation.

| Stage                          | Deliver                                                                                               | Order                                                       |
| ------------------------------ | ----------------------------------------------------------------------------------------------------- | ----------------------------------------------------------- |
| O1 — Operation/value contracts | Typed references, exact supported field values, and explicit fidelity limits.                         | Alongside DSL D1/D2; coordinate fn-75.                      |
| O2 — Parameterized semantics   | Action/outcome arguments, field expressions/captures, finite domains, and independent Properties.     | After O1; supplies parameterized DSL authoring.             |
| O3 — Checked concrete lowering | Typed request construction, result projections, clause coverage, and field-expression correspondence. | After O2; uses fn-71/fn-72 and coordinates with DSL D4.     |
| O4 — Field-level qualification | Unary RPC and workflow-owned Nexus examples, field mutations, and real Driver evidence.               | After O3; temporal Nexus qualification also uses DSL D3/D4. |

Task breakdown and plan review remain pending. Keep this off the fn-73/first-canary critical path.
DSL D1/D2 and label-only
semantics can proceed independently; parameterized D3/D4/D5 variants consume O1/O2. O3 owns field
lowering, D4 owns scoped temporal lowering, and same-step qualification does not wait for temporal
support. Share value/capture contracts across the tracks rather than creating two representations.

### 4. fn-70 — Scheduled canary proof of concept

[Spec](../.flow/specs/fn-70-scheduled-canary-proof-of-concept-as-a.md).
Deliver after **fn-78**; fn-68, fn-71, fn-72, and fn-73 are transitive prerequisites.

Implement the second consumer under `tools/canary`: manual check selection, a fresh scheduled
Workflow each minute, Activity-owned Testpilot execution, bounded results, and isolated repeated
measurements. Consume the Driver and binding interfaces delivered above; do not repeat their
implementation work. Retain the cross-consumer proof; fn-73 already owns the live proof that one
Case byte sequence runs against two environment bindings.

This is a local/development prototype. It does not depend on fn-26 or fn-29 and does not authorize
production deployment or replace fn-29's separately scoped production-canary design.

### Additional open specs

These remain open in Flow and are outside the first-canary critical path.

| Spec                                                                                              | Dependencies | Next action                                                                                                                                                                                                                  |
| ------------------------------------------------------------------------------------------------- | ------------ | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| [fn-46 — Lean module impact index](../.flow/specs/fn-46-export-lean-model-module-impact-index.md) | fn-45        | Refresh the previously approved plan against current module owners, then deliver the on-demand dependency, facade, and focused-test index. Task breakdown already exists.                                                    |
| [fn-67 — Nexus3 authoring draft](../.flow/specs/fn-67-refine-simple-nexus3-authoring-draft.md)    | None         | Reconcile its design-only requirements with the delivered fn-68 model and ongoing fn-78 authoring work; verify completion or retain uncovered draft requirements. This spec does not implement syntax or a runtime compiler. |

## Downstream delivery

Each spec needs a fresh Testpilot plan review before implementation. Prior reviews of the
retired execution architecture do not approve the rewritten plans. Completed dependencies do not
block replanning or execution.

| Spec                                                                                              | Dependencies                       | Next action                                                                                                                                                                                                                                   |
| ------------------------------------------------------------------------------------------------- | ---------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| [fn-33 — Bounded exploration](../.flow/specs/fn-33-run-serial-bounded-semantic-exploration.md)    | fn-40, fn-64, and fn-69            | Review whole-Case candidates, serial coordination through Testpilot, lost iterations, semantic coverage, and bounded 10x behavior.                                                                                                            |
| [fn-22 — Replay and reduction](../.flow/specs/fn-22-deterministic-replay-semantic.md)             | fn-5, fn-64, and fn-69             | Resolve **MAJOR_RETHINK** before implementation: separate exact candidate identity from Contract-relative violation equivalence, prove the negative Case before reduction, and retain explicit offline semantic replay and checked promotion. |
| [fn-26 — Qualification receipts](../.flow/specs/fn-26-local-qualification-receipts-and-staged.md) | fn-48, fn-64, and fn-69            | Review offline Testpilot Case/Profile/Run/Verdict admission, receipt multiplicity, and idempotent publication. Assessment must never create or replay a Run.                                                                                  |
| [fn-29 — Production canary](../.flow/specs/fn-29-bounded-production-canary-execution-and.md)      | **fn-26**; fn-48, fn-64, and fn-69 | After fn-26 ships, review external policy and credentials, serial Testpilot Runs, leases, lost Runs, reconciliation without redispatch, and publication.                                                                                      |

All runtime work uses `testpilot.Prepare(case, profile)` → `PreparedCase.Run(ctx, driver)` and the
server/worker authority split. New scenarios remain Case data; canary policy, credentials,
leases, recovery, and publication stay outside Testpilot and Umpire.

## Completed cutovers

- [fn-73](../.flow/specs/fn-73-explicit-environment-binding-for.md): established exact Case 1.0
  symbolic resource bindings, immutable Profile-owned snapshots, static Driver validation, and the
  two-environment Nexus3 proof without request rewriting or Case-byte drift.
- [fn-71](../.flow/specs/fn-71-standalone-lean-testpilot-protocol.md): established independent
  `Testpilot.*` types, context-safe expressions, one current codec, and producer-owned Umpire
  provenance.
- [fn-72](../.flow/specs/fn-72-extract-the-reusable-temporal-testpilot.md): moved the reusable
  Temporal Driver to `common/testing/testpilot/temporal`, preserved server/worker/delivery ownership,
  and retained functional fixtures and provisioning under `tests/`.
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

Whole-spec completion-review status is tracked in Flow and is not implied by placement in this
list. Nexus2 remains a prototype with explicit adoption boundaries.

## Deferred and superseded

These entries are outside the delivery queue and are not prerequisites for it.

| Deferred spec                                                             | Revisit when                                                                                                   |
| ------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------- |
| [fn-60](../.flow/specs/fn-60-deepen-authored-lean-canonical-json.md)      | Optional handwritten canonical JSON consolidation becomes worth prioritizing; it has no downstream dependency. |
| [fn-15](../.flow/specs/fn-15-standalone-api-and-config-input-catalogs.md) | Platform completeness is needed beyond the proven model family.                                                |
| [fn-23](../.flow/specs/fn-23-veil-toolchain-compatibility-and.md)         | Optional checker adoption becomes valuable.                                                                    |
| [fn-24](../.flow/specs/fn-24-lean-native-verification-receipts-and.md)    | A verification receipt/profile platform is justified.                                                          |
| [fn-25](../.flow/specs/fn-25-optional-callerclosure-veil-binding-and.md)  | A second verification backend is justified; caller closure remains historical.                                 |
| [fn-30](../.flow/specs/fn-30-release-evidence-graph-and-manual.md)        | Real Claim Assessment evidence supports release governance.                                                    |

[fn-14](../.flow/specs/fn-14-milestone-a-pilot-baseline-and-lean.md) is historical;
[fn-61](../.flow/specs/fn-61-simplify-the-umpire-go-execution-surface.md) and
[fn-63](../.flow/specs/fn-63-consolidate-umpire-go-tests-into-golden.md) are superseded by fn-64.
Any remaining `todo` children do not reactivate them. Broader test consolidation needs a new
Testpilot proposal with an independent oracle.
