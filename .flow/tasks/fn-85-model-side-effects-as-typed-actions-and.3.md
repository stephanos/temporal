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
The prototype holds, and the row-grammar fallback does not fire.

**The claim.** The user's 2026-09-12 decision — a machine's logic is an ordinary Lean step function,
enumerated at elaboration into the finite table the `model` command has always produced, rather than a
row grammar — rests on the enumeration producing the same table, and therefore the same Behavior
Fingerprint, as the rows an author would have written. The Nexus success slice's two rows, written as
a step function over a state structure, enumerate to exactly those rows: same rows, same order, same
keys. The two `FiniteTable`s built from them are equal, and the fingerprint is a pure function of the
table, so it is equal too.

**`Finite`** is the domain: an ordered, complete member list, with instances for `Bool` and
`Fin (n + 1)` and a deriving handler for enum-like inductives and for structures whose fields are all
finite — which is what a machine's per-instance state is. Both refusals are located and named rather
than surfacing as an instance-search failure from inside the enumeration: a `String` field reports the
field, a constructor taking an argument reports the constructor. Both are pinned with `#guard_msgs`.

**`enumerate`** walks the declared domains, evaluating the step function once per (state, action)
pair. An empty result contributes no row, because the finite table already calls an absent pair
disabled — so a step that rejects an action needs no separate encoding. `enumerateBounded` refuses an
oversized domain with both factors and the bound, never a truncated table.

**A `count` field saturates rather than wraps.** The last member *is* "at the limit", which is what a
Property reads off it; wrapping would send a count that overran back to zero, reading as a Model that
never counted.

**The measurement, honestly.** The task asked for elaboration time recorded against the Race baselines
(6 to 12 ms per *check*). Those come from a different harness; what is measurable here is whole-file
elaboration, which startup and imports dominate — 1104, 1313 and 1543 ms for the prototype against
1607, 1131 and 1061 ms for a module that only imports `Finite`, so the prototype is inside the noise
of its own import. The numbers are recorded as a reference point rather than a comparison, and the
module says so. They do answer the question the measurement exists for: enumeration is not measurably
slower than written rows, let alone an order of magnitude. A per-check comparison at the Race scale
needs the `machine` command, which is `.14`.

**Self-review found two things**, both fixed in 618c0eff3. The enumeration had introduced its own
`256` beside `Syntax.lean`'s private `transitionBound` — one decision in two places, which is what
fn-84 spent five tasks removing; there is now one `elaborationBound` and `Syntax.lean` reads it. And
the count helpers sat in `Umpire.Command.Fin`, which reads as core `Fin` at a glance; they are plain
`saturatingSucc` and `limitReached`.

**Left to .14 and .15:** the `machine` command with its witness diagnostics, the Limits accounting,
the migration of the 18 `model` declarations and their 33 specimens, and predicate Properties. This
task is the enumerator and the decision it carries, which is why it was split out of the former
single `.3`.

**Review:** self-review, SHIP, recorded through `flowctl`. Implementer and reviewer are the same
session, so a session with a second backend should re-review before the spec's completion review.

**Gates.** `lake build UmpireTests TemporalModelTests` green (399 jobs); `make lint-model` at its
baseline — import-graph linting passed, Batteries clean for `Shared` and for `Umpire.Lint`, and the
163 diagnostics are all in generated `Temporal/API/Proto.lean`, so this task's declarations added
none.
## Evidence
- Commits: 69350b703, 618c0eff3
- Tests: lake build UmpireTests TemporalModelTests, lake build Umpire.Command.Tests.Finite Umpire.Command.Syntax, make lint-model
- PRs: