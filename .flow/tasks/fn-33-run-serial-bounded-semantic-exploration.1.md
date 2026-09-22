---
satisfies: [R1, R3, R6]
---
# fn-33-run-serial-bounded-semantic-exploration.1 Re-found Umpire.Exploration on the exploratory set

## Description
Replace the Space-based exploration core with a campaign over one exploratory set's coverage targets. Temporal-free; the caller Model is named only in tests through the Switch example's own exploratory set.

**Size:** L
**Files:** `model/Umpire/Exploration.lean`, `model/Umpire/Exploration/{Campaign,Target,Ledger,Session}.lean`, `model/Umpire/Exploration/Tests/**`, `model/Umpire/Examples/Switch.lean`
**Touches:** [model/Umpire/Exploration.lean, model/Umpire/Exploration/**, model/Umpire/Command/Authoring.lean, model/Umpire/Examples/Switch.lean, model/Umpire.lean, model/UmpireTests.lean, model/Umpire/ARCHITECTURE.md, model/README.md, model/lakefile.lean, .plans/UMPIRE4_SPEC.md]

### Approach
- `Campaign.check`: one `SetDeclaration` with `purpose: exploratory`, its `DeclaredModel`, its enumerated `targets`, and the `Limits` value its `budget:` names (the declaration holds only the name); rejects a non-exploratory set, a target whose Definition IDs the Model does not declare, and limits the Model's Search rejects.
- `Target.row`: the row a target names: itself for a `row` target; for a `result` target the first row in table order among the reachable rows whose results contain the outcome; for a `classMember` target the first reachable row whose action is the member.
- `Target.prefix`: a shortest path over the table from a start state to that row's source within the limits' steps, as an `AuthoredExactTrace` (each step's action, outcome and resulting state), new code beside `coverageTargets`, whose `within` closure keeps no paths.
- `Target.query`: a Scenario with `traceExactly` = the exact trace plus the final step (the row's action and outcome) and `actionsExactly` = that trace's action list in the same order (the Producer reads `actionsExactly`; the checker rejects the two disagreeing), and the Property `.action A` with the `outcomeClause` naming the row's outcome for every target kind (a class member's row is the one `Target.row` chose, so its Property carries that row's outcome and never zero clauses). Admitted and searched with the authoring `Umpire.Command.check` uses, through a new `Umpire.Command.checkAdmitted` that returns the `AdmittedQuery` beside the `CheckedModel` (`check` keeps its signature and calls it). `unreachable` when no prefix reaches the row within `steps` or admission reports `notSelected`; `invalidTarget`, `invalidVocabulary`, `admission` and `instances` are the campaign's own defect and end it as `tooling-failure`. A witness that does not contain the selected target is the same invariant failure.
- `Campaign.next`: the first target in enumeration order whose status is `pending`; returns the candidate (`CheckedModel`, its `AdmittedQuery`, identity = the artifact checksum, the row, result and class-member targets on its planned witness path) or `exhausted`. No scoring, no seed; a target is planned at most once.
- `Ledger`: per-target status (`pending`, `covered`, `unreachable`, `violated`, `attempted`), the class ledger (class → decisive verdict of its member target's candidate), counterexample = a class-member target whose candidate's Verdict is `violated`, and credit: a `satisfied` decisive observation credits every target on the candidate's planned path; `violated` marks them; a preparation rejection or any non-decisive observation marks them `attempted`.
- Keep `Session.next`/`observe` (one outstanding, exact binding) over the new candidate; retire `beginSession`, `Core`, `Language`, `Engine`, `Candidate`, `Selection`, `Guided`, `Coverage` over `VariationSpace` and their tests. `Umpire.Variations` stays. Update the importers: `Umpire.lean`, `UmpireTests.lean`, `Umpire/ARCHITECTURE.md`, `README.md`; keep the module name `Umpire.Exploration` (a facade root of fn-46's module index); amend the UMPIRE4 spec's Exploration concept and the Variations concept's "Exploration draws candidates from" sentence to the set-based definition as one GOV-02 draft, and state there that the exact-prefix shortest witness is EXP-05's minimization.
- Tests: the Switch example gains an exploratory set over `rows` and `results` (adding a set changes no machine identity, so its pinned fixtures stay); class-member and counterexample material comes from a test-local classed machine under `Exploration/Tests/**`, never from `twoState`. Pin selection order, credit for a path that covers several targets, unreachable, violated and attempted classification, admission errors as `tooling-failure`, counterexample detection, exact-binding rejection of crossed and stale observations, byte-identical results for identical inputs, and a synthetic 10x Model (rows x10) explored within the same limits with one Search per candidate.

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
- [ ] Every candidate's planned witness contains its selected target; a `satisfied` observation credits every row, result and class member on that path; `violated` marks them; a preparation rejection or a non-decisive observation marks them `attempted`; crossed and stale observations reject at the cursor; admission errors other than `notSelected` are `tooling-failure`.
- [ ] A violated class-member target on the test-local classed machine is one counterexample naming class, target and candidate identity, with its `AdmittedQuery` retained; the Switch fixtures are unchanged.
- [ ] The Space-based Core, Language, Engine, Candidate, Selection, Guided and Coverage modules and `beginSession` are gone with their tests; `Umpire.Variations` and its users are untouched; the cursor's exact-binding semantics are pinned on the new surface; the UMPIRE4 Exploration concept carries the GOV-02 draft.
- [ ] A 10x synthetic Model explores within the same limits; identical inputs give byte-identical results.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
