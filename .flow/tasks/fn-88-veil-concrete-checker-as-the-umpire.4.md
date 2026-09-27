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
TBD

## Evidence
- Commits:
- Tests:
- PRs:
