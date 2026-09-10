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

Originally nine tasks with a SHIP plan review; re-planned to fifteen after four escalations.
**10 of 15 done.**

**R2 and R8 are fully covered.** Task .5 escalated SCOPE_EXCEEDED — R2 looked like deleting four
identifier whitelists, but `Authoring.successModel` was arity-fixed at three states, two actions, two
outcomes, two facts and two transitions, and that shape was load-bearing across ~30 reads in
`Tests.lean`, several inside `native_decide` theorems. It was split into four steps that each land
green, and all four shipped: **.10** turned the 14 + 22 named fields into parallel ordered lists with
positional accessors and made the canonical-table law range over any number of rows, with zero syntax
change so conformance proved byte-identity by itself; **.11** replaced the whitelists with a
`Lean.Elab.Command` elaborator reading each inductive's constructors, generalizing the grammar to N
initial/terminal states, N transitions, N facts per row and N occurrences; **.12** added five located
diagnostics at the author's own coordinates, each pinned by `#guard_msgs`; **.13** added a genuinely
second lifecycle (`RaceSyntaxTests.lean` — role `handler`, five states, four Actions, two terminals, a
losing row with no Fact) which forced generalizing `property` to N `require` clauses and surfaced two
more diagnostics the planner had previously rejected late. **.6** then landed the `query ... all ...`
verify form through `QueryForm.verify`, making `CheckedModel.witness` optional and splitting
`AdmissionError.noWitness` into `notSelected (outcome)` so an unsatisfiable Behavior and a violating
counterexample are distinguishable.

**R1's user-visible half is delivered.** Task **.15** deleted the clause-for-clause equality gate in
the Nexus3 Producer: the correlated-history monitor is now derived from the Facts the selected witness
records, through declared per-Fact history evidence projections. A model edit now produces different
Case bytes instead of a lowering error, which was the Goal section's first defect. Output is
byte-identical, so no fixtures moved and live behaviour is unchanged by construction.

Also done earlier: **.1** the `rule_events` Contract horizon, **.2** the `InjectFault` wire plus `Run`
returning the recorder close error, **.3** worker stop/resume in the Driver, **.7** `DeriveProfile`
with the hand-written fixture Profiles as oracles.

**Five tasks remain, and every one is blocked on the same thing.** The completion review reads
NEEDS_WORK, but it flagged no untracked work — each gap maps to an already-blocked task: R1's scoped
route (.14 then .4), R4's fault Case and R3's Lean `rule_events` Case (.8), and R7's spec rules and
documentation (.9, behind .8). Task .5 stays blocked as the superseded record of why R2 was split.

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
`bounded-completion-is-model-only`, now reached from the other side.

Both halves of R1 are now tasks. **.14** adds the `ScopedEvidence`-emitting projection both specs
need — a Program-declared source lifting recorded history into `ScopedEvidence` with `identity`,
`operation`, `kind` and `fields` — and must also close the vacuous-satisfaction hole, since a clause
receiving no evidence answering SATISFIED is the defect that makes the naive fix dangerous. Task .4
now depends on it. **.14 has since landed and closed both halves**: the lift is a
`ScopedEvidenceProjection` sink on `ProjectionTarget`, the typed-nexus Case runs its bounded-response
clause live through it, and a capability that admitted no evidence now answers unresolved in both
the Lean portable interpreter and the Go runtime. The paragraphs above describe the wall as it stood
before that. **.15** carries R1's separable product defect, the clause-for-clause equality gate
in `produceCompletionCase` that turns a model edit into a lowering error; it needs no evidence path
and does not wait on .14.

Boundaries worth carrying forward: no canary Profile or production authorization (fn-70, fn-29 own
those), no Nexus cancellation lowering (fn-79, deferred), no second fault kind, and no removal of
`elapsed_milliseconds`.

### 2. Unify the Umpire and Testpilot vocabulary — fn-82

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

**Only fn-80 still blocks this spec; fn-77, fn-81, and fn-67 have all closed.** The dependency is
byte conflict, not semantics: fn-80 tasks .4 to .8 edit `Umpire/Target`, `Umpire/Property`,
`Umpire/Query`, `Umpire/Space`, and `Umpire/Case`, so no area of this spec can land while those are
open. fn-81's deletion of the legacy Go trees and their Makefile blocks shrinks this spec's Go
surface considerably. One seam fn-67 left deliberately for this spec to sweep: it added
"Semantic fingerprints — the Behavior Fingerprint named above is the same value —" to reconcile a
pre-existing loose term against the UMPIRE4_SPEC one; collapse that to a single term and delete the
parenthetical. It deliberately does not touch fn-77's five in-flight terms
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

These remain open in Flow and are outside the first-canary critical path.

| Spec                                                                                              | Dependencies | Next action                                                                                                                                                                                                                  |
| ------------------------------------------------------------------------------------------------- | ------------ | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| [fn-46 — Lean module impact index](../.flow/specs/fn-46-export-lean-model-module-impact-index.md) | fn-45        | Refreshed three-task plan is SHIP against current model owners. Deliver the shared loader, pure dependency/facade/test index, and opt-in export/check commands. |

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

- [fn-67](../.flow/specs/fn-67-refine-simple-nexus3-authoring-draft.md): reconciled the broader
  Nexus3 authoring draft with the delivered success slice. The optional per-declaration compatibility
  ID is now documented beside the success slice's derivation-only identity — declaration-local,
  identity-only, never freezing the Behavior Fingerprint — and the "no overrides" claims are scoped
  to the delivered syntax. `cancellationResolves` now states that generic operation-scoped counting
  is delivered and qualified, with only its cancellation-specific use rejecting at Case production,
  replacing a stale "proposed extension" line. The historical draft iteration is preserved intact and
  fn-79 was not resumed. Documentation only: `Nexus.lean` changed by module docstring alone, with
  every executable declaration byte-identical. Three impl-review rounds and the completion review are
  SHIP with zero gaps. Review round 1 caught a false claim that the historical `Cancellation.lean`
  Target has no evidence adapter — `System/Nexus/ImplementationLink.lean:431-530` holds an offline
  Lean evidence projection; the boundary is that nothing lowers it into a Testpilot evidence path,
  operation capability, or Case.
- [fn-81](../.flow/specs/fn-81-delete-the-pre-testpilot-go-generations.md): deleted the
  pre-Testpilot Go generations — **1,525 files, 386,985 deletions**, tracked files 7,785 -> 6,345,
  packages 546 -> 422, and twelve third-party modules out of `go.mod` with zero hits in the retained
  closure. No Lean was deleted. All five tasks and the completion review are SHIP. Go lint fell
  **1,284 -> 128** (forbidigo 209->0, revive 732->106, errcheck 220->1); 1,156 findings had lived in
  the deleted trees and none of the remaining 128 is in a file this spec touched. `lint-model` stayed
  byte-identical at 169 and `go vet` at its 15 inherited diagnostics, both correctly unmoved.
  Gomad v3 is retained per [GOMAD_MILESTONES](GOMAD_MILESTONES.md) F0 — it is the only path to
  running an unchanged Temporal functional test under a deterministic runtime — and its one edit was
  retiring the SIM-0 parity manifest, whose every source path pointed into the deleted `tools/gomad2`.
  Task .1's evidence ledger before any deletion is what caught two consumers the plan had not
  predicted: three upstream test bodies in `tests/nexus_workflow_test.go` that this branch had
  replaced with umpire2 delegations (restored from `origin/main`), and `tools/common/formal`, whose
  only test invocation sat inside the deleted `gomad-prototype` block and now has its own target.
  The live-test gate was redesigned rather than repinned: it selects on the `^TestTestpilot` prefix so
  new tests join without enumeration, and adds a passing-identity floor, because an empty baseline
  alone cannot distinguish "everything passed" from "the selector matched nothing". R5 is partial for
  a documented reason — `make lint-code` and `planindex` were red before any fn-81 edit and the
  spec's own Boundaries forbid the branch-wide fix. Ten `docs/superpowers/specs` records could not
  receive their historical banners: a global gitignore excludes them from the repo, so R7 cannot
  reach them.
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
