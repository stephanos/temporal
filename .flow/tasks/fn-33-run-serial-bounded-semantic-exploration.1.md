---
satisfies: [R1, R3, R6]
---
# fn-33-run-serial-bounded-semantic-exploration.1 Re-found Umpire.Exploration on the exploratory set

## Description
Replace the Space-based exploration core with a campaign over one exploratory set's coverage targets. Temporal-free; the caller Model is named only in tests through the Switch example's own exploratory set.

**Size:** L
**Files:** `model/Umpire/Exploration.lean`, `model/Umpire/Exploration/{Campaign,Target,Ledger,Session}.lean`, `model/Umpire/Exploration/Tests/**`, `model/Umpire/Examples/Switch.lean`
**Touches:** [model/Umpire/Exploration.lean, model/Umpire/Exploration/**, model/Umpire/Examples/Switch.lean, model/Umpire.lean, model/UmpireTests.lean, model/Umpire/ARCHITECTURE.md, model/README.md, model/lakefile.lean, .plans/UMPIRE4_SPEC.md]

### Approach
- `Campaign.check`: one `SetDeclaration` with `purpose: exploratory`, its `DeclaredModel`, its enumerated `targets`, and the `Limits` value its `budget:` names (the declaration holds only the name); rejects a non-exploratory set, a target whose Definition IDs the Model does not declare, and limits the Model's Search rejects.
- `Target.query`: the target Query for one `CoverageTarget`, formed as every command Query is: `Scenario.exactly` over a table-walked prefix from a start state to the target's source (the walk `coverageTargets` performs) plus the target's action, and a Property `.action A` with an `outcomeClause` naming the row's result or the result value, or the action alone for a class member; admitted and searched through `Umpire.Command.check` within the limits, keeping the `AdmittedQuery`. `unreachable` when no prefix reaches the source within `steps` or the outcome is not `.found _ .satisfyingWitness`.
- `Campaign.next`: the first target in enumeration order that is neither covered, unreachable nor violated; returns the candidate (`CheckedModel`, identity = its artifact checksum, the row, result and class-member targets on its planned witness path) or `exhausted`. No scoring, no seed.
- `Ledger`: per-target status (`pending`, `covered`, `unreachable`, `violated`), the class ledger (class → decisive verdict of its member target's candidate), counterexample = a class-member target whose candidate's Verdict is `violated`, and credit: a `satisfied` decisive observation credits every target on the candidate's planned path; `violated` marks them; anything else credits nothing.
- Keep `Session.next`/`observe` (one outstanding, exact binding) over the new candidate; retire `beginSession`, `Core`, `Language`, `Engine`, `Candidate`, `Selection`, `Guided`, `Coverage` over `VariationSpace` and their tests. `Umpire.Variations` stays. Update the importers: `Umpire.lean`, `UmpireTests.lean`, `Umpire/ARCHITECTURE.md`, `README.md`; keep the module name `Umpire.Exploration` (a facade root of fn-46's module index); amend the UMPIRE4 spec's Exploration concept to the set-based definition as a GOV-02 draft.
- Tests: the Switch example gains an exploratory set; pin its selection order, credit for a path that covers several targets, unreachable and violated classification, counterexample detection, exact-binding rejection of crossed and stale observations, byte-identical results for identical inputs, and a synthetic 10x Model (rows x10) explored within the same limits with one Search per candidate.

### Investigation targets
**Required** (read before coding):
- `model/Umpire/Command/Coverage.lean` and `Records.lean:150-200` — `CoverageTarget`, `coverageTargets`, `SetDeclaration`.
- `model/Umpire/Exploration/{Engine,Session}.lean` — what is retired and the cursor that stays.
- `model/Umpire/Search.lean`, `model/Umpire/Query.lean:23` — planning a `find` Query within `Limits`.
- `model/Umpire/Case/Producer.lean:334,963` — `Realization` and `produce`, which task .2 calls with the target Query's Plan.
- `model/Umpire/Examples/Switch.lean` — the command-authored example to extend with an exploratory set.

### Quick commands
`cd model && lake build Umpire UmpireTests && lake exe umpire-lint-tests && cd .. && make umpire-check-goldens umpire-check-regression-views umpire-check-inventory && LEAN_NUM_THREADS=1 make lint-model`

### Re-plan note (2026-09-21)
Re-planned on fn-85's exploratory set after fn-86 R6 deleted the variation Space this task was first written against; see the spec's **Re-plan on fn-85** section. Start only after the spec's fresh plan review.
## Acceptance
- [ ] `Campaign.next` walks the Switch exploratory set's target order deterministically, one bounded Search per candidate, and classifies unreachable targets without a Run.
- [ ] A `satisfied` observation credits every row, result and class member on the candidate's planned path; `violated` marks them; inconclusive, crossed and stale observations credit nothing and reject at the cursor.
- [ ] A violated class-member target is one counterexample naming class, target and candidate identity.
- [ ] The Space-based Core, Language, Engine, Candidate, Selection, Guided and Coverage modules and `beginSession` are gone with their tests; `Umpire.Variations` and its users are untouched; the cursor's exact-binding semantics are pinned on the new surface; the UMPIRE4 Exploration concept carries the GOV-02 draft.
- [ ] A 10x synthetic Model explores within the same limits; identical inputs give byte-identical results.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
