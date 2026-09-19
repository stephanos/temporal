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

## Acceptance
- [x] `property` takes a `Step → Bool` or `Step → Step → Bool` predicate and enumerates it into `PropertyClause` records; the keyed `require:` form no longer elaborates
- [x] `successfulResult` as a predicate has the fingerprint of its keyed form, pinned by `#guard`
- [x] a predicate over another machine's `Step` type and a non-decidable predicate reject in place, pinned by `#guard_msgs`
- [x] the success Model regenerates the async-Nexus fixture byte-identical through `case`; `lake build TemporalModelTests UmpireTests` green; `make lint-model` green

## Done summary

A `property` names a machine and an ordinary Lean predicate, and the command enumerates it over the
machine's own table into the clause records a Property has always carried -- so Search, the
Behavior Fingerprint and Contract lowering never see a function, and `successfulResult`,
`cancellationSettles`, `renamedModelResult`, `probeStart` and `attemptEnds` carry the fingerprints
their keyed `require:` blocks had, each pinned by `#guard`.

### The reading

The keyed block spelled out what the predicate now says. Over the steps the trigger admits -- the
steps one Action produces under `when:`, or the steps out of one prior state for a transition claim
-- the predicate **fixes** a value when every step it accepts carries it, the domain has another
value to change it to, and the predicate rejects every accepted step with the value changed: the
state, the outcome, and each fact whose removal it rejects. The fixed values are the clauses, in the
order state, outcome, facts, which is the order the keyed blocks were written in. They must carry
the predicate exactly over the table, both ways; a predicate they cannot carry -- a disjunction
across fields -- is refused with the step the clauses cannot tell apart, rather than approximated.

Read off the predicate, not off the table's coincidences: `Umpire.Command.Predicate` probes the
predicate with altered steps, so a predicate that only names a fact gains no state clause because
every accepted step happened to share one. The probe is what makes the fingerprint pin hold on
every migrated Property and what stops a one-member outcome domain from being "fixed" by every
predicate.

A transition claim, `Step → Step → Bool` with no `when:`, enumerates into one group per prior state
it constrains, triggered by `priorState` and labelled `from-<prior>-...`; a prior state at which it
accepts every step and fixes nothing contributes nothing. The step before is every step the table
arrives at that state by, or, for a state nothing arrives at, that state under each outcome with no
facts. `startedThenSucceeds` in the success tests is the pin.

### Decisions

- **`when:` decides the shape.** A same-step claim is `Step → Bool` under `when:`; a transition claim
  has no `when:`. The other pairing is rejected naming the shape it wanted. `when:` takes a bare
  action or a classed one with its inputs applied (`handlerReply (handlerError true)`), keyed the way
  the machine keys its members, so a Property over a classed action can be written.
- **A predicate that holds on no admitted step is refused at the `property`**, not found
  unsatisfiable by a later Query: `unreachableResult` moved from a Query diagnostic to a `property`
  one, and the Query diagnostic "no trace the Scenario admits satisfies the Property" is pinned on
  `forkedLifecycle`, whose `awaitStart` records `succeeded` only from `started`.
- **A predicate that holds on every admitted step and fixes nothing claims nothing**, and says so.
- **A misspelled member is Lean's own error** at the spelling, since the predicate is Lean over the
  machine's domains; `misspelledState` and `misspelledOutcome` pin that.
- **The keyed form is rejected at its key** (`require:`, and `model:` on `property`), naming
  `holds:` and `machine:`, rather than gated: `require` and `model` are bare words, which SEM-20 keeps
  out of the vocabulary gate, and the gate's existing `require +[label]: resultingState` rule stays for
  the older Nexus spelling. `scenario` and `query` keep `model:`, which this task does not touch.
- **The `machine` command emits `outcomeKeyFor` and `factKeyFor`** beside `stateKeyFor`, so a
  Property names what a step produced in the spelling its clauses carry.
- **What Lean logs while trying a shape is collected, not reported**: the attempt at the claim's own
  shape reports exactly what it logged, and the diagnosis attempts leave nothing behind.

### Gates

`lake build` green (613 jobs); `make umpire-check-testpilot-authoring` exit 0 with the async-Nexus
fixture byte-identical; `make umpire-check-goldens` exit 0; `LEAN_NUM_THREADS=1 make lint-model` at
the 163 baseline. `DESIGN.md` section 3 carries a dated amendment showing `terminalIsFinal` as a
transition predicate.

Self-review: no second backend is installed in this cloud session, so this owes a cross-model
re-review before the completion review, as `.1`, `.2`, `.3`, `.14` and `.16` do.

## Evidence
- Commits: fcbc068
- Tests: cd model && lake build; make umpire-check-testpilot-authoring; make umpire-check-goldens; LEAN_NUM_THREADS=1 make lint-model
- PRs:
