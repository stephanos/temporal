---
satisfies: [R1, R3, R6]
---
# fn-33-run-serial-bounded-semantic-exploration.1 Re-found Umpire.Exploration on the exploratory set

## Description
Replace the Space-based exploration core with a campaign over one exploratory set's coverage targets. Temporal-free; the caller Model is named only in tests through the Switch example's own exploratory set.

**Size:** L
**Files:** `model/Umpire/Exploration.lean`, `model/Umpire/Exploration/{Campaign,Target,Ledger,Session}.lean`, `model/Umpire/Exploration/Tests/**`, `model/Umpire/Examples/Switch.lean`
**Touches:** [model/Umpire/Exploration.lean, model/Umpire/Exploration/**, model/Umpire/Examples/Switch.lean, model/UmpireTests.lean, model/lakefile.lean]

### Approach
- `Campaign.check`: one `SetDeclaration` with `purpose: exploratory`, its `DeclaredModel`, its enumerated `targets` and its `budget` (`Limits`); rejects a non-exploratory set, a target whose Definition IDs the Model does not declare, and a budget the Model's Search rejects.
- `Target.query`: the target Query for one `CoverageTarget`: a `find` goal that takes the row (source state, action, one of its results), reaches the result value, or performs the class member's action, planned by `Umpire.Search` within the budget. `unreachable` when Search reports no path or `limitReached`.
- `Campaign.next`: the first target in enumeration order that is neither covered, unreachable nor missed; returns the candidate (`Plan`, identity `ArtifactChecksum`, the targets its selected trace reaches) or `exhausted`. No scoring, no seed.
- `Ledger`: per-target status (`pending`, `covered`, `unreachable`, `missed`), the class-member ledger (member → decisive verdict), counterexample detection (two members of one class, different verdicts), and credit from a decisive reading: the rows, results and members a Run's Model Trace reaches, matched by Definition ID.
- Keep `Session`'s one-outstanding cursor and exact-binding `observe`; retire `Engine`, `Candidate`, `Selection`, `Guided`, `Coverage`, `Language`, `Core` over `VariationSpace` and their tests, and `Umpire.Variations` consumers that only they had. `UmpireTests` and the compatibility family pins move to the new surface.
- Tests: the Switch example gains an exploratory set; pin selection order, credit for a trace that reaches several targets, unreachable and missed classification, counterexample detection, exact-binding rejection of crossed and stale observations, byte-identical results under reordered targets, and a synthetic 10x Model (rows x10) explored within the same budget with one Search per candidate.

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
- [ ] `Campaign.next` walks `nexusCallerExploration`'s target order deterministically, one bounded Search per candidate, and classifies unreachable targets without a Run.
- [ ] A decisive reading credits every row, result and class member its trace reaches; inconclusive, crossed and stale observations credit nothing and reject at the cursor.
- [ ] Two members of one class with different decisive verdicts are one counterexample naming class, members and verdicts.
- [ ] The Space-based Engine, Candidate, Selection, Guided and Coverage modules are gone with their tests; the cursor's exact-binding semantics are pinned on the new surface.
- [ ] A 10x synthetic Model explores within the same budget; reordered targets and Model declarations give byte-identical results.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
