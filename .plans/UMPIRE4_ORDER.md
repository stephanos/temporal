# Umpire 4 delivery order

Build on the completed Testpilot and model authoring cutovers. Flow owns task status;
this document records delivery order. Architecture and terminology live in the
[Umpire 4 specification](UMPIRE4_SPEC.md).

## Current work

### 1. Typed operations and field-level Properties — fn-77

[fn-77 — Typed operations, parameterized Actions, and field-level Properties](../.flow/specs/fn-77-typed-operations-parameterized-actions.md)
adds direct generated API references, typed SDK-command/event declarations, modeled request/result
values, and cross-field/cross-occurrence requirements. Product assertions remain in checked models;
integration owns faithful execution, observation, and correlation.

Eleven tasks are defined with a SHIP plan review. They consume the delivered semantic and scoped
monitoring interfaces; Nexus operation cancellation remains deferred to fn-79.

| Stage                          | Deliver                                                                                               | Order                                                       |
| ------------------------------ | ----------------------------------------------------------------------------------------------------- | ----------------------------------------------------------- |
| O1 — Operation/value contracts | Typed references, exact supported field values, and explicit fidelity limits.                         | Uses delivered fn-75 semantics.                            |
| O2 — Parameterized semantics   | Action/outcome arguments, field expressions/captures, finite domains, and independent Properties.     | After O1; supplies parameterized DSL authoring.             |
| O3 — Checked concrete lowering | Typed request construction, result projections, clause coverage, and field-expression correspondence. | After O2; uses delivered fn-71/fn-72/fn-78 interfaces.     |
| O4 — Field-level qualification | Unary RPC and workflow-owned Nexus examples, field mutations, and real Driver evidence.               | After O3; temporal Nexus qualification also uses DSL D3/D4. |

Keep this off the first-canary critical path. Field lowering composes with fn-78's delivered scoped
temporal lowering. Share value/capture contracts across these owners rather than creating two
representations. Tasks with overlapping source files run serially without artificial dependencies.

### 2. fn-70 — Scheduled canary proof of concept

[Spec](../.flow/specs/fn-70-scheduled-canary-proof-of-concept-as-a.md).
Deliver after **fn-78**; fn-68, fn-71, fn-72, and fn-73 are transitive prerequisites.
Nine tasks cover all ten requirements, with a SHIP plan review. Implementation follows fn-77
to serialize shared Producer edits; fn-77 is not a semantic prerequisite.

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
| [fn-46 — Lean module impact index](../.flow/specs/fn-46-export-lean-model-module-impact-index.md) | fn-45        | Refreshed three-task plan is SHIP against current model owners. Deliver the shared loader, pure dependency/facade/test index, and opt-in export/check commands. |
| [fn-67 — Nexus3 authoring draft](../.flow/specs/fn-67-refine-simple-nexus3-authoring-draft.md)    | None         | Reviewed follow-up plan is SHIP: one documentation task reconciles optional draft compatibility overrides with the success demonstration and clarifies generic versus cancellation support. Preserve the historical draft iteration; no cancellation implementation is required. |

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

- [fn-76](../.flow/specs/fn-76-make-lean-semantic-inventory-consume.md): moved shared outcome and
  Known Gap contracts to semantic owners and enforced inventory dependency direction. Preserved
  generated inventory, canonical fixtures, and proof trust; all tasks and completion review are SHIP.
  Model build/lint pass; exact inherited Go lint and live-test failures remain recorded in Flow.
- [fn-75](../.flow/specs/fn-75-separate-lean-target-semantics-from.md): separated checked Target
  semantics, pure projection, and authoring; enforced the transitive semantic import boundary.
  Preserved canonical fixtures and proof trust; all three tasks and completion review are SHIP.
- [fn-74](../.flow/specs/fn-74-deepen-testpilot-worker-activation.md): established private activation
  state and work accounting for both SDK interpreters, plus public preparation diagnostics. Preserved
  SDK scheduling, delivery authority, replay, and cleanup behavior; completion review is SHIP.
- [fn-78](../.flow/specs/fn-78-typed-temporal-authoring-and-checked.md): delivered shared scoped
  obligations, command/event evidence projection, independent Query validity, typed bounded temporal
  notation, and readable existing Behavior constraints while preserving Nexus success. Completion
  review is SHIP; Nexus operation cancellation remains deferred to fn-79.

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

**fn-79 — Nexus operation cancellation:** [spec](../.flow/specs/fn-79-deferred-nexus-operation-cancellation.md).
Includes former fn-78.5/.8/.9 cancellation scope and fn-77’s cancellation qualification. Resume only
on an explicit user request; autonomous delivery approval does not override this deferral. Generic
fn-78 syntax/monitoring/qualification and fn-70 remain deliverable without it. Existing shutdown
and bounded cleanup cancellation behavior stays in scope.


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
