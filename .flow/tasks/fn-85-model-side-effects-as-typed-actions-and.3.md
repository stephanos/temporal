---
satisfies: [R3]
---
# fn-85-model-side-effects-as-typed-actions-and.3 Finite domains and the step-function enumerator: the prototype that decides the authoring form

## Description
Build the enumeration the `machine` command will use, and prove on it that a step function is the same Model as a row table (R3, the part the authoring decision rests on). A `Finite` class with instances for enum-like inductives, `Bool` and `Fin`, plus a deriving handler for a structure of finite fields, gives the derived domain; an enumerator evaluates a step function `State → Input → List (State × Outcome)` over that domain into the `BehaviorTransitionRow`s `model` produces today. The success Model rewritten as a step function must enumerate to a `BehaviorTable` and Behavior Fingerprint equal to the row form's. This is the stop condition the user's 2026-09-12 decision named: if the fingerprint differs or elaboration is an order of magnitude slower than the Race baselines, stop and report — the fallback is the row grammar, and `.14` and `.15` do not start.

Split out of the former single `.3` by plan review round 1 (finding F2): the enumerator with its stop condition is the decision, and it has to be actionable on its own.

**Size:** S
**Files:** `model/Umpire/Command/Finite.lean` (new: a `Finite` class with instances for enum-like inductives, `Bool` and `Fin`, and a deriving handler for structures of finite fields; a located rejection for any other field type), `model/Umpire/Command/Authoring.lean` (the enumerator alone: evaluate a step function through the `Meta.evalExpr` bridge `evalDiagnostic` already uses into `BehaviorTransitionRow`s; `count` fields as `Fin (bound+1)` saturating to `limitReached`), `model/Umpire/Command/Tests/Finite.lean` (new: the prototype `#guard`s and the fingerprint equality)
**Touches:** [model/Umpire/Command/Finite.lean, model/Umpire/Command/Authoring.lean, model/Umpire/Command/Tests/**]

### Approach
- Read `.plans/LEAN_GUIDELINES.md` first.
- `Finite` is a small local class, not `Fintype`: no Mathlib.
- Enumeration: domain = the product of the state structure's fields × the action's input class members (task .2's constructors with fields); evaluate the step for every pair; each successor becomes a transition row keyed by (state, action input, successor, outcome). The bound is the existing `transitionBound`, re-checked against the product state space and reported as a located error, never a truncation.
- Prototype: rewrite the success Model's two rows as a step function over `structure { phase : State }`, enumerate it, and `#guard` that the resulting `BehaviorTable` and fingerprint equal the row form's. Measure elaboration with `lake env lean` three times against the Race baselines (6 to 12 ms per check) and record all three numbers.
- Do not add the `machine` command here, and do not touch `model`: the prototype builds its step function as a plain definition and calls the enumerator directly, so the decision is measured on the enumeration and not on a grammar.
- A non-finite field is a located rejection at the field, not a class-resolution failure surfaced from deep inside elaboration; pin the message.

### Investigation targets
**Required:**
- `model/Umpire/Command/Syntax.lean:54-57,73-75,97,144-166,195-345,548-566` — the row grammar, `transitionBound`, `relationKey`, reachability, `elabModel`, the `evalExpr` bridge
- `model/Umpire/Command/Authoring.lean:138-168,215-323` — `DeclaredModel`, `step`, `declaredTable`, `declareModel`
- `model/Umpire/Model/Types.lean:52-76` — `BehaviorTransitionRow`, `BehaviorTable` (the canonical form the enumeration fills)
- `.plans/UMPIRE_CMP_FIZZBEE.md` section 4.1 — the chosen form and its rule reading (AUT-05, AUT-07a, AUT-09)

**Optional:**
- `model/Temporal/Feature/Nexus/Race/COVERAGE.md:28-30` — the elaboration baselines
- `model/Umpire/Core.lean:311,367-390` — vocabulary materialization and `Machine`

### Key context
- AUT-09 today admits enum-like inductives; task .13 drafts the amendment admitting a structure of finite fields and an enumerated step function. Until approved the enumerator is drafted under that rule the way AUT-07a's commands were.
- Memory: bound `count` fields and the product state space before enumerating; never enumerate eagerly beyond Limits.

## Acceptance
- [ ] `Finite` derives for a structure of finite fields and rejects any other field type in place, pinned by `#guard_msgs`
- [ ] the enumerator turns a step function into `BehaviorTransitionRow`s; a `count` field saturates to `limitReached`; exceeding `transitionBound` is a located error, never a truncation
- [ ] prototype: the success Model as a step function enumerates to a `BehaviorTable` and fingerprint equal to the row form's, pinned by `#guard`; three elaboration measurements recorded against the Race baselines
- [ ] `lake build UmpireTests` green; `make lint-model` green
- [ ] if either prototype pin fails, the task stops with the numbers reported and `.14`/`.15` are re-planned on the row grammar

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
