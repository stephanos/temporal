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
- [x] `Campaign.next` walks the Switch exploratory set's target order deterministically, one bounded Search per candidate, and classifies unreachable targets without a Run.
- [x] Every candidate's planned witness contains its selected target; a `satisfied` observation credits every row, result and class member on that path; `violated` marks them; a preparation rejection or a non-decisive observation marks them `attempted`; crossed and stale observations reject at the cursor; admission errors other than `notSelected` are `tooling-failure`.
- [x] A violated class-member target on the test-local classed machine is one counterexample naming class, target and candidate identity, with its `AdmittedQuery` retained; the Switch fixtures are unchanged.
- [x] The Space-based Core, Language, Engine, Candidate, Selection, Guided and Coverage modules and `beginSession` are gone with their tests; `Umpire.Variations` and its users are untouched; the cursor's exact-binding semantics are pinned on the new surface; the UMPIRE4 Exploration concept carries the GOV-02 draft.
- [x] A 10x synthetic Model explores within the same limits; identical inputs give byte-identical results.
## Done summary
Done 2026-09-22; plan reviewed through `flowctl claude plan-review` (SHIP); implementation reviewed through `flowctl claude impl-review` (opus at high): NEEDS_WORK with four findings, all applied in 8c0cf817e, then SHIP. Commits 241b78581, 8c0cf817e.

`Umpire.Exploration` is now a campaign over one exploratory set's coverage targets. `Target`
turns a target into the Query the campaign runs for it -- the row it names (`chooseRow`), the
shortest admissible path to that row's source (`pathTo` over `shortestPaths`), and the exact-trace
Scenario with `actionsExactly` beside `traceExactly` plus the outcome-clause Property on the final
step (`targetAuthors`) -- and `coveredTargets` reads what a planned witness reaches. `Ledger` is
the per-target status (`pending`, `planned`, `covered`, `unreachable`, `violated`, `attempted`),
the class ledger and the counterexamples, and `credit` is the one place an observation becomes a
status. `Campaign.check` admits the set against its machine's identity and the `Limits` value its
budget names; `Campaign.next` plans the first pending target through the new
`Umpire.Command.checkAdmitted`, marks a target no admissible path or `notSelected` reaches
unreachable and moves on, and ends the campaign as a tooling failure on any other admission
error, a candidate without a Plan, or a witness that does not contain its own target
(`admissionFailure` is the classification, pinned alone); `observe` credits; `summary` is the closed
record. `Session` keeps one candidate outstanding with exact-binding `observe`. The Space-based
`Core`, `Language`, `Engine`, `Candidate`, `Selection`, `Guided`, `Coverage` and their seven test
modules are gone; `Umpire.Variations` and its users are untouched; the module keeps its name.

### What the implementation found

A transition contract binds every occurrence of its action at search time (`Property.Evaluate`),
and the Producer lowers only action-triggered clauses, from the first occurrence with a window to
the end. So the prefix to a target's row may take the row's action earlier only with the row's
outcome, or the Query is unsatisfiable. `pathTo` searches under that admissibility, one bounded
search per candidate; a row with no such path is `unreachable` under this Query form, and the
counter Model's last self-loop (`row:s9-advance`: nine advances that move, then one that stays)
is the pinned example. The spec's Decision Context records it.

The set records its machine under the machine's own identity, not the Model's target, and class
members enumerate in the machine's action order, which is by member name.

### Tests

`Umpire.Exploration.Tests.Campaign` walks the Switch's new exploratory set (`switchExploration`,
one row and two results under `one`): the selection order, what each candidate's planned path
covers, credit under satisfied, violated, inconclusive and prepare-rejected observations, a covered
target staying covered, an unreachable target skipped without a Run, the four campaign rejections,
`admissionFailure`, byte-identical results for identical inputs, and the session's one-outstanding
and exact-binding rules. `Tests.Classed` is a test-local lamp Model with a classed action and
examples: two class-member targets, a violated member recorded as one counterexample with its
`AdmittedQuery` retained, a satisfied member as a class verdict, a non-decisive Run as neither.
`Tests.Scale` is a ten-slot counter, twenty rows: twenty-one targets covered in nineteen
candidates and the one self-loop honestly unreachable. Adding a set changes no machine identity, so
the Switch fixtures are unchanged.

### Docs and spec

`model/README.md`, `model/ARCHITECTURE.md` and `model/Umpire/ARCHITECTURE.md` describe the
set-based exploration; two docstrings that named the retired modules are reworded. The UMPIRE4
spec's Exploration concept and the Variations concept's "draws candidates from" sentence carry
amendments drafted by fn-33 awaiting GOV-02 approval, with the exact-prefix shortest witness named
as EXP-05's minimization.

### Review

Round one found four things, fixed in 8c0cf817e: a counterexample was only recorded when the class
had no verdict yet, so a class satisfied by a row candidate could never report a later violation
(now every violated Run crossing a class member is a counterexample, once per candidate, and a
violation supersedes an earlier satisfied verdict; pinned both ways in `Tests.Classed`); a row
target tried only its first result, so a row plannable under its second was called unreachable
(now each result in order, the named ones first; `Tests.Results` pins the walker whose far row
plans under `moved`); exhausted selection fuel read as an honest exhaustion (now a tooling
failure); a zero-step budget still planned one step (now `none`). Round two: SHIP, with two notes
for task .2's author: the class-member branch takes each row's first result only, failing honest
rather than wrong, and the class ledger keys on the class spelling alone, which merges two claims
that reuse a spelling across actions.

### Gates

`cd model && lake build`, `lake exe umpire-lint-tests`, `make umpire-check-goldens
umpire-check-regression-views umpire-check-inventory umpire-check-retired-vocabulary`, `go test
-count=1 -tags test_dep ./tools/umpire/...`, `LEAN_NUM_THREADS=1 make lint-model` at the fn-86
closeout baseline.
## Evidence
- Commits: 241b78581, 8c0cf817e
- Tests: cd model && lake build, cd model && lake exe umpire-lint-tests, make umpire-check-goldens umpire-check-regression-views umpire-check-inventory umpire-check-retired-vocabulary, go test -count=1 -tags test_dep ./tools/umpire/..., LEAN_NUM_THREADS=1 make lint-model
- PRs: