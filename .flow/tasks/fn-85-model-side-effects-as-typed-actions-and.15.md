---
satisfies: [R3]
---
# fn-85-model-side-effects-as-typed-actions-and.15 Property bodies are predicates over a machine's steps

## Description
A `property` names a machine and a Lean predicate rather than a keyed `require:` block (R3, the second half of the 2026-09-12 authoring decision): `Step → Bool` for a same-step claim, or `Step → Step → Bool` over the step before and the step after for a transition claim. The command enumerates the predicate over the machine's finite table into the existing `PropertyClause` records, so Search, fingerprints and Contract lowering never see a function. The keyed `require: state:/outcome:/fact:` form of fn-83 is retired with `model`. Reachability stays the `find` Query rather than a second `exists` word (SEM-19), and a guard needs no keyword — it is an `if` in the predicate.

Split out of the former single `.3` by plan review round 1 (findings F2 and F3): it is a separate authoring decision from the `machine` command, and it had no requirement row until R3's text was amended to name it.

**Size:** S
**Files:** `model/Umpire/Command/Syntax.lean` (`property` takes `machine:` and `holds:`; the keyed form removed), `model/Umpire/Command/Authoring.lean` (enumerate the predicate over the machine's table into `PropertyClause` records through `.3`'s enumerator), `model/Umpire/Command/Registry.lean`, `model/Temporal/Feature/Nexus/Success/Model.lean` and `Success/Tests.lean` (the Properties rewritten as predicates; `#guard_msgs` for the rejections), `tools/umpire/internal/retiredvocabulary/check.go` (the keyed clause spelling retired)
**Touches:** [model/Umpire/Command/**, model/Temporal/Feature/Nexus/Success/**, tools/umpire/internal/retiredvocabulary/check.go]

### Approach
- Read `.plans/LEAN_GUIDELINES.md` first. Depends on `.14`, whose `machine` command owns the table the predicate is enumerated over.
- The enumeration is `.3`'s, applied to `Step` rather than to a transition: evaluate the predicate at every step (or every adjacent pair) of the machine's finite table and emit the same `PropertyClause` records the keyed form emits, so the Behavior Fingerprint does not move.
- Pin the equality: `successfulResult` written as a predicate has the fingerprint its keyed form had. That is the proof that this is a spelling change and not a semantic one.
- A predicate whose `Step` type belongs to another machine, and a predicate that is not decidable, reject in place.

### Investigation targets
**Required:**
- `model/Umpire/Command/Syntax.lean` — the `property` command and its keyed clause grammar
- `model/Umpire/Property.lean` and `model/Umpire/Property/Evaluate.lean` — `PropertyClause` and its evaluation
- `model/Temporal/Feature/Nexus/Success/Model.lean` — `successfulResult` in its keyed form
- `.plans/UMPIRE_CMP_FIZZBEE.md` sections 4.1 and 4.8 — the decision and its rule reading

### Key context
- Task `.10` writes the Nexus Properties in this form, so this task lands before it.
- Contract lowering reads `PropertyClause`; nothing downstream may learn that a predicate existed.

### Moved here from task .4, 2026-09-19

A Property clause over a machine's state field, and the two acceptance items that rest on it.

`.4` put a machine's state fields on the wire and taught both evaluators to read them: a STATE
condition reads the state and every field the machine keeps, so the Contract compares `attempts` as
a number and `phase` as an enum. The Contract path already admits a Property that names a field --
`Umpire.Case.Correlated.pattern` lowers a `.resultingState` reference with whatever definition it
carries. What does not admit it is the *checked* side: `ModelTrace` carries a state as one value, so
a Property naming a field is true on the portable side and false on the checked one, and the
correspondence certificate is what fails.

So this task's foundation is the Model trace carrying a machine's fields. Two items come with it:

- `PropertyTraceField.state` per field, with `naturalAtMost` reading a `count` field.
- The end-to-end two-instance conformance scenario `.4` names, whose whole point is a rule that
  reads a field. `.4` proved instead that the two evaluators compute the field rule identically,
  by paired focused tests from one worked transition; what is still owed is one Case replayed
  through the Go evaluator against a field-reading rule.

`Umpire.Case.CorrelatedProofs.trace` is where the fields are dropped today, and it says so.

## Acceptance
- [ ] `property` takes a `Step → Bool` or `Step → Step → Bool` predicate and enumerates it into `PropertyClause` records; the keyed `require:` form no longer elaborates
- [ ] `successfulResult` as a predicate has the fingerprint of its keyed form, pinned by `#guard`
- [ ] a predicate over another machine's `Step` type and a non-decidable predicate reject in place, pinned by `#guard_msgs`
- [ ] the success Model regenerates the async-Nexus fixture byte-identical through `case`; `lake build TemporalModelTests UmpireTests` green; `make lint-model` green

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
