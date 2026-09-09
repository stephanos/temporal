# Umpire 4 delivery order

Build on the completed Testpilot and model authoring cutovers. Flow owns task status;
this document records delivery order. Architecture and terminology live in the
[Umpire 4 specification](UMPIRE4_SPEC.md).

## Current work

### 1. Close the model-to-Case seam — fn-80

[fn-80 — Close the model-to-Case seam and harden](../.flow/specs/fn-80-close-the-model-to-case-seam-and-harden.md),
derived from the 2026-09-08 assessment against [UMPIRE4_VISION](UMPIRE4_VISION.md). The assessment
found the substrate rigorous and the product-facing seams thin: the shipped Nexus3 Producer
hand-writes its Program and monitor and rejects any Property not clause-for-clause equal to one
expected value, so a model edit yields a lowering error instead of a different Case. The
proof-carrying scoped lowering in `Umpire.Case.Scoped` is the only path where wire bytes provably
mean the checked model, and no shipped Case uses it.

Eight requirements: general checked lowering replacing the equality gate (R1), generalized command
syntax and the always-erroring `query ... all ...` form (R2, R8), an event-count liveness horizon
replacing the one the recorder derives from the test host's clock (R3), one realizable worker fault
where today only labeled intent exists (R4), Profile derivation and a runner helper for the 60-to-80
line cost of running a Case from Go (R5), the discarded recorder close error (R6), and gates plus
the axiom baseline (R7).

Nine tasks are defined with a SHIP plan review. Progress: .1 done (`rule_events` horizon and the
hardened retired-vocabulary gate); .4 blocked, see below; the rest in flight or ready.

**fn-77 is complete, so the ordering constraint on this spec is gone.** R1 and R5 had serialized
behind fn-77.10 to avoid a byte conflict on regenerated fixtures; that conflict no longer exists.

**Task .4, the early proof point, is blocked — and the finding matters beyond this spec.** R1 wants
the shipped async-nexus Case to carry `contract.scoped`, drop its hand-written correlated-history
monitor rule, and still be satisfied live. Those three cannot hold together today: `bindScoped`
refuses any Case whose `Contract.scoped` names an `evidence_observation_id` that is not a Program
Observation typed exactly `ScopedEvidence`, no version-one `Instruction` can produce that value
(`run.proto:125` states it is supplied only through the capability's declared typed Observation),
and a scoped clause receiving zero evidence answers SATISFIED **vacuously** — so forcing it through
would convert a real regression test into a hollow one.

The spec's stop-condition asked whether the scoped clause form can carry state, outcome, and fact
predicates. It can: the three success clauses are expressible as `bounded_response%` clauses sharing
an `awaitSuccess` trigger. The blocker is the evidence path, not the clause form, so R2 and R8 are
unaffected and stay startable. This is the same gap fn-77 closed out as
`bounded-completion-is-model-only`, now reached from the other side; it needs a
`ScopedEvidence`-emitting projection. R1's own product defect — the clause-for-clause equality gate
in `produceCompletionCase` that turns a model edit into a lowering error — is separable and worth
closing without the evidence path.

Boundaries worth carrying forward: no canary Profile or production authorization (fn-70, fn-29 own
those), no Nexus cancellation lowering (fn-79, deferred), no second fault kind, and no removal of
`elapsed_milliseconds`.

### 2. Delete the pre-Testpilot Go generations — fn-81

[fn-81 — Delete the pre-Testpilot Go generations](../.flow/specs/fn-81-delete-the-pre-testpilot-go-generations.md),
a mechanical deletion sweep from the same 2026-09-08 assessment. Nothing here changes modeled
behavior. The repository still carries every earlier generation as live, compiled, partly CI-wired
code that no current Umpire, Testpilot, or Temporal model imports — about 345,000 Go lines against
roughly 42,000 for live Umpire plus Testpilot:

| Tree | Go lines | Still wired into |
| ---- | -------- | ---------------- |
| gomad through gomad3integration | 209,000 | a workflow, Makefile targets, four nested go.mod files, a root go.mod `replace` |
| umpire1, umpire2, umpire3 | 92,700 | two workflows, ~70 Makefile targets, a second Lake project (677 MB on disk) |
| `common/testing/umpire` and the testcore monitor | 26,400 | history workflow cache instrumentation, observer comments in six history files, the functional harness monitor and gRPC interceptor |
| agentworkflow | 11,200 | Makefile, its own go.mod |
| legacy tests under `tests` | 6,200 | the live-test gate, which pins nine expected failures by name |

Two costs are paid today: the gomad3 workflow triggers on any change to go.mod or the Makefile, so
it runs on unrelated pull requests, and umpire3's Lake build alone holds 677 MB on a disk at 97
percent.

Five tasks are defined with a SHIP plan review, all ready. Task .2 is the declared early proof
point — the white-box seam comes out of the history service and the functional harness and the
retained live gate still passes; if a retained test turns out to depend on monitor facts,
re-evaluate the seam disposition before deleting any tree. That seam is the one part of this sweep
that is not confined to `tools`, and it produces no Verdict for any consumer, so R6 removes it and
restores the history workflow cache to upstream shape.

Boundaries: no Lean deletions — Nexus v1, Nexus2, Umpire Artifact, Space, and Exploration stay,
since live modules import them and fn-22, fn-33, fn-79, and fn-80 reserve them; they need a roadmap
decision rather than a sweep. `tools/fairsim`, `cmd/tools/fairsim`, and fn-66's `tools/planindex`
also stay. Task .5 owns the roadmap reconciliation under R7, so it will edit this document.

### 3. Unify the Umpire and Testpilot vocabulary — fn-82

[fn-82 — Unify the Umpire and Testpilot vocabulary](../.flow/specs/fn-82-unify-the-umpire-and-testpilot.md),
from the 2026-09-08 vocabulary investigation of `model/`, the Testpilot protocol, and the Go facade.
The model works; its vocabulary does not. A reader who moves from a Nexus3 model to the Umpire types
behind it to the Case that comes out meets the same idea under several names and the same name for
several ideas. Measured on the current tree:

| Word | Distinct meanings | Word | Distinct meanings |
| ---- | ----------------- | ---- | ----------------- |
| Projection | 9 | Target | 6 types |
| Evidence | 8 | Outcome | 5 |
| Observation | 6 | Capability | 5 |

The reverse is as common: the transition relation is a `TransitionKernel`, a `FiniteMachine`, a
`FiniteTable`, or a `ValidatedFiniteModel` depending on the file; something that must hold is a
Property, Clause, Obligation, Requirement, Claim, Law, or Rule; a budget is a Limit, Bound, Ceiling,
Horizon, or `bounds`. `Scoped` prefixes 24 proto names and 40 Lean files with nothing stating that it
means "tracked per operation, correlated by a key". Some vocabulary is simply dead — a 142-structure
duplicate Testpilot mirror inside `Temporal.API` with zero consumers, five `DefinitionKind`
constructors that exist only to be rejected, and alias families whose own files say "remove this
alias".

Ten tasks are defined with a SHIP plan review. Breaking changes are accepted throughout: nothing is
versioned, aliased, or deprecated, old names are retired, and a hardened retired-vocabulary gate
rejects them. The spec rewrites `UMPIRE4_SPEC.md` under GOV-02, and its task .10 owns the roadmap
reconciliation, so it will edit this document.

**This spec starts only after fn-77, fn-80, fn-81, and fn-67 close.** The dependency is byte
conflict, not semantics: fn-80 tasks .4 to .8 edit `Umpire/Target`, `Umpire/Property`,
`Umpire/Query`, `Umpire/Space`, and `Umpire/Case`, fn-81 deletes the legacy Go trees and their
Makefile blocks, and fn-67 has an open documentation task on the same Nexus3 files — so no area of
this spec can land while those are open. It deliberately does not touch fn-77's five in-flight terms
(typed operation, parameterized Action, field-level Property, occurrence, capture); the
`Umpire.Operation` and `Umpire.Value` renames wait for fn-77 .11 and are limited to `ValueShape` to
`Shape` plus moving `Parameterized.lean`.

Task .2, the first rename, is the declared early proof point: it renames `DefinitionKind.kernel` to
`machine` and `.observation` to `.fact`, adds the `umpire-goldens` writer, and regenerates every
fingerprint, golden, and Case fixture through the owning targets. If that loop cannot converge
without hand-editing a golden, stop and re-evaluate the "regenerate, never edit" rule before any
other rename starts.

Boundaries: no deletion of the offline `Umpire.Evidence` evaluator, `Umpire.Artifact.RunRecord`,
`Umpire.Variations`, `Umpire.Exploration`, or `Umpire.Promotion` — they are renamed and moved, and
fn-22, fn-33, fn-79, and fn-80 reserve the decision to retire them. No proto field-number changes, no
new command syntax beyond respelling what fn-80 R2 generalizes, and no Go changes outside
`common/testing/testpilot`, `tests/testcore/testpilot`, `tests` fixture helpers, and `tools/umpire`.

### Additional open specs

These remain open in Flow and are outside the first-canary critical path. fn-67 is not outside fn-82's path, however: fn-82 cannot start until it closes.

| Spec                                                                                              | Dependencies | Next action                                                                                                                                                                                                                  |
| ------------------------------------------------------------------------------------------------- | ------------ | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| [fn-46 — Lean module impact index](../.flow/specs/fn-46-export-lean-model-module-impact-index.md) | fn-45        | Refreshed three-task plan is SHIP against current model owners. Deliver the shared loader, pure dependency/facade/test index, and opt-in export/check commands. |
| [fn-67 — Nexus3 authoring draft](../.flow/specs/fn-67-refine-simple-nexus3-authoring-draft.md)    | None         | Reviewed follow-up plan is SHIP: one documentation task reconciles optional draft compatibility overrides with the success demonstration and clarifies generic versus cancellation support. Preserve the historical draft iteration; no cancellation implementation is required. **fn-82 blocks on this closing** — it edits the same Nexus3 files. |

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

- [fn-77](../.flow/specs/fn-77-typed-operations-parameterized-actions.md): delivered typed generated
  operation references, parameterized Actions over finite and runtime domains, and field-level
  Properties with keyed captures, portable capture evaluation, and checked Case coverage. Qualified
  on the real Driver by two examples — a generated unary operation and two Nexus SDK operations with
  captured field relations. All thirteen tasks and the completion review are SHIP; nine of nine
  R-IDs covered with none unaddressed. Two tasks were added mid-spec: `.12` bounded a
  parameterized-operation identity that encoded the whole generated `RpcSchema` (27.7M characters
  per payload key, making qualification infeasible) down to ~4.4K under a `parameterized-v2-`
  migration, and `.13` derived the unary example's Contract read path from its Property so the two
  examples share one R8 mechanism. Trust comparison against the task `.1` substrate shows no
  unexplained inventory loss and no axiom beyond `propext`/`Classical.choice`/`Quot.sound`; 204 of
  208 frozen fixtures are byte-identical. `go vet ./...` is clean at zero and Go lint added nothing
  to the inherited 1,284; the two `lint-model` findings this spec introduced were fixed, leaving 169
  in generated `Temporal/API` only. Known Gap `bounded-completion-is-model-only` remains: the scoped
  bounded-response clause is proven portable in Lean and Go but does not run on the real Driver,
  because no Program emits a `ScopedEvidence` Observation. Reviews from task `.9` onward ran on the
  same-family `claude` backend after both cross-family bridges ran out of budget.
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


**fn-70 — Scheduled canary proof of concept:** deferred by user decision; it was previously
queued after fn-78 as the second model consumer. Resume on an explicit user request. Nothing in
the delivery queue depends on it.

[Spec](../.flow/specs/fn-70-scheduled-canary-proof-of-concept-as-a.md).
Retained scope on resume: fn-78 first, with fn-68, fn-71, fn-72, and fn-73 as transitive
prerequisites.
Nine tasks cover all ten requirements, with a SHIP plan review. Implementation follows fn-77
to serialize shared Producer edits; fn-77 is not a semantic prerequisite.

Implement the second consumer under `tools/canary`: manual check selection, a fresh scheduled
Workflow each minute, Activity-owned Testpilot execution, bounded results, and isolated repeated
measurements. Consume the Driver and binding interfaces delivered above; do not repeat their
implementation work. Retain the cross-consumer proof; fn-73 already owns the live proof that one
Case byte sequence runs against two environment bindings.

This is a local/development prototype. It does not depend on fn-26 or fn-29 and does not authorize
production deployment or replace fn-29's separately scoped production-canary design.


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
