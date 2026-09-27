---
satisfies: [R5, R6]
---
# fn-88-veil-concrete-checker-as-the-umpire.4 Property monitor lowering v1 with evaluator agreement

## Description
Lower `CheckedPropertyClause` to bounded three-valued monitors for the seven version-one clause kinds, reading each monitor's state and its terminal and partial verdicts off the evaluator, and prove or exhaustively test that each monitor's answer equals `clauseEndpointAnswer`. Everything else returns typed `Unsupported`. Adopt mode only.

**Size:** M
**Files:** `model/Umpire/Search/Product/Monitor.lean` (new), `model/Umpire/Search/Product/MonitorProofs.lean` (new), `model/Umpire/Search/Tests/Monitor.lean` (new), `model/Umpire/Search/Tests.lean` (register), `model/Umpire/Search/Product.lean` (plug the monitor family in)
**Touches:** [model/Umpire/Search/Product/Monitor.lean, model/Umpire/Search/Product/MonitorProofs.lean, model/Umpire/Search/Tests/Monitor.lean, model/Umpire/Search/Tests.lean, model/Umpire/Search/Product.lean]

### Approach
- Monitors answer `PropertyEndpointAnswer` (`model/Umpire/Property/Evaluate.lean:2175`) for closed endings (`final` or `terminal`) and for `partial`. Shapes from the evaluator: `stateInvariant` seen bit + failed bit, `unresolved` while nothing matched (`:984-988`); `identityRelation` existential over the trace, monotone seen bit, `unresolved` under partial until it fires (`:1033-1036`); `transitionContract` and `inputOutput` one-step implications with a seen bit for coverage; `ordered` seen-before bit per pair, `unresolved` under partial until decided; `eventuallyWithin` and `neverWithin` countdown bounded by the clause Limit, `unresolved` before the deadline (`:2248-2268`). `LimitUnit.actions` counts as steps (`:2203`); `logicalTime` is `Unsupported`.
- Each monitor reports whether its trigger fired, feeding the product's fired-clause bitset.
- `branches`, `guardedEventuallyWithin`, `guardedNeverWithin`, correlated clauses, and `logicalTime` Limits return `Unsupported` naming the kind, in the typed shape `Compiler.Error` uses (`model/Umpire/Case/Compiler.lean:47-54`).
- Agreement: prove against `clauseEndpointAnswer` and `evaluatePropertyClause_agrees` (`:1699`) for the one-step kinds; for `ordered` and the countdown kinds either prove or run the exhaustive differential over every checked-in model's traces within Limits under both endings; follow `endpoint_agrees` (`model/Umpire/Case/CorrelatedProofs.lean:166`).
- Expose a per-kind `TrustBasis` (`kernel` | `testing`) the receipt reads.
- Register `Tests/Monitor.lean` in `model/Umpire/Search/Tests.lean`.

### Investigation targets
**Required:**
- `model/Umpire/Property/Check.lean:157` — clause kinds
- `model/Umpire/Property/Evaluate.lean:984-988, 1033-1036, 1699, 2175-2268, 2283-2337` — per-clause semantics, endpoint answers, agreement theorems
- `model/Umpire/Search.lean:948-1000` — `observeCandidate`, how endpoint verdicts and triggers are computed today
- `model/Umpire/Case/CorrelatedProofs.lean:166` — proof shape to mirror

**Optional:**
- `model/Shared/CorrelatedObligation.lean:198-254` — an existing monitor's `consume`/`answer` shape (correlated clauses stay unsupported here)

### Key context
- The Case Producer's lowering (`model/Umpire/Case/Producer.lean:453`) is trace-bound and targets runtime Observations; do not modify or call it.
- Axiom inventories: `propext`, `Quot.sound`, `Classical.choice` only, pinned with `#guard_msgs in #print axioms`.

## Acceptance
- [ ] Monitors exist for the seven v1 clause kinds with the stated state, three-valued answers under both endings, and bounds; all other kinds and `logicalTime` return typed `Unsupported`
- [ ] Per kind, a theorem or an exhaustive differential test against `clauseEndpointAnswer`, and a `TrustBasis` recorded
- [ ] Axiom pins pass; a deliberately wrong monitor for one kind is caught by the differential (then removed)
- [ ] Monitor family plugged into `Umpire.Search.Product`; new test module registered; `make lint-model` passes
- [ ] Defer mode: closed as not applicable citing the R1 receipt identity, nothing added

## Done summary
Added `Umpire.Search.Product.Monitor`: it lowers the seven version-one clause kinds to bounded three-valued monitors. Each monitor's state and its closed and partial answers are read off the evaluator. The kinds are `stateInvariant`, `transitionContract`, `identityRelation`, `inputOutput`, `ordered`, `eventuallyWithin` and `neverWithin`; the countdown kinds use `steps` or `actions` Limits. The monitors see the same view the evaluator does: capability-admitted values, state fields admitted with their state, and prior-state occurrences one position back when counting steps. Everything else returns a typed `MonitorUnsupported` naming the Property, the clause and its kind: `branches`, the guarded kinds, correlated clauses, and `logicalTime` (or search/plans) Limits. `QueryMonitors` combines the clause answers per Property in the reference order and exposes fired-clause bits. `MonitorFamily.ofQuery` and `MonitoredProduct.build`/`answers` plug them into `Umpire.Search.Product`.

R6 evidence is testing for every kind (`MonitorKind.trustBasis`). The evaluator's clause semantics are private to `Evaluate.lean`/`Check.lean`, which are outside this task's Touches, so a kernel proof could not unfold them. `Umpire/Search/Tests/Monitor.lean` (registered in `Tests.lean`) compares answers under both endings, and the fired bits, with `evaluatePropertyEndpoint`:
- over 682 synthetic traces (every trace of up to four steps, from two initial states), for a 124-Property table covering every kind, field, unit and bounds 0 to 2;
- over every trace within and past the Limits of the Switch, Search-fixture and parameterized Models, with clause tables generated from their values;
- through `MonitoredProduct` over the Switch Queries' own Property.

Answer counts per kind are pinned so the comparison is not vacuous. A deliberately wrong `eventuallyWithin` monitor was caught by four guards and then reverted. `MonitorProofs.lean` proves that closed answers are never `unresolved`, that budgets stay within the clause Limit at a root and across every step, and that untriggered clauses never fire. Axioms are pinned (propext, Quot.sound, Classical.choice only).

Deviations: the step helpers are public under `Monitor.` so the proofs module can unfold them. `Product.lean` disables `linter.extra.dupNamespace`: fn-88.3's `Umpire.Search.Product.Product` tripped it under `make lint-model`, and renaming it would touch `Tests/Product.lean`, which is outside this task's Touches.

Follow-ups (reviewer P3s, not blocking):
- Apply `monitorsAgree`/`modelAgrees` to the Temporal feature Models (Nexus Success and others) alongside fn-88.9's Caller and Pair pins.
- Share one Property-order helper between `observeCandidate`, `QueryMonitors.lower` and the test oracle.
- Rename `Product.Product` so the linter suppression can go.
- `HANDWRITTEN_INVENTORY.md`'s Switch importer count gains `Tests/Monitor.lean`.

`make lint-model` was inconclusive: it failed only on .olean files missing because of concurrent builds and on another session's in-progress `Umpire/Case/Tests/FieldLowering.lean`. The same lint (`lake --wfail lint --builtin-only`) passes on this task's modules.

Defer mode: not applicable (R22 adopt).

stage: impl-review - ran [2026-09-27..2026-09-27] (claude backend; SHIP, re-reviewed SHIP after the lint fix; range 4dfc90dedf..HEAD because other sessions' commits landed after the pre-edit base 928f7deb34)
## Evidence
- Commits: 964b57e54d2a533603354e98ca426f9ce4fd4865, a94bfa142c92275cf74603853522472e941dbc41
- Tests: mise exec -- lake build Umpire.Search Umpire.Search.Product Umpire.Search.Tests Umpire.Search.VisibilityTests (green), make umpire-check-goldens (green), mise exec -- lake --wfail lint --builtin-only --lint-only=.all,.extra,-.missingDocs Umpire.Search.Product.Monitor Umpire.Search.Product.MonitorProofs Umpire.Search.Product Umpire.Search.Tests.Monitor (green), LEAN_NUM_THREADS=1 make lint-model: INCONCLUSIVE - import-graph and Batteries lint passed and every module of this task built warning-free under --wfail; the run failed only on missing .olean files clobbered by concurrent builds and on Umpire/Case/Tests/FieldLowering.lean from another session's in-progress edits, baseline: green (focused lake build of the Search targets, pre-edit)
- PRs: