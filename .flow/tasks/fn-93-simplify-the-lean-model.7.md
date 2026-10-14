---
satisfies: [R5]
---
# fn-93-simplify-the-lean-model.7 Typed keywords in the command elaborators (A3)

## Description
Lane A3, part one. One helper resolves a keyword ident to a constructor of a given inductive and builds the "one of …" diagnostic from the constructor list in constructor order. It replaces the string matches for set purpose, `driven`/`observed`, coverage goal and Known Gap kind (`gapKindTerm`), and the authoring-role matches in the parts of `Property/Scenario/Query Elab` that B4 keeps. `Registry.SetEntry.purpose` becomes the enum.

**Size:** M
**Files:** `model/Umpire/Command/Syntax.lean` (set elaborator ~2670-2780, `gapKindTerm` ~653), `model/Umpire/Command/Registry.lean` (`SetEntry.purpose` ~160-169), `model/Temporal/Case/Syntax.lean` (purpose string compares ~208-212), `model/Umpire/{Property,Scenario,Query}/Elab.lean` (role matches outside the syntax tails), a new helper in `model/Umpire/Command/` (e.g. `Keyword.lean`)
**Touches:** [model/Umpire/Command/**, model/Temporal/Case/Syntax.lean, model/Umpire/Property/Elab.lean, model/Umpire/Scenario/Elab.lean, model/Umpire/Query/Elab.lean]
**Depends on other specs:** fn-92.2 and fn-92.4 edit `Command/Syntax.lean` and `Registry.lean`; re-read line positions at start.

### Approach
- Helper derives spellings from the WireName `all`/`name` of task 4-6 (e.g. `SetPurpose`, `CoverageGoal` in `Command/Records.lean`), so the diagnostic lists constructors in declaration order.
- Before switching each site, capture its current unknown-keyword `#guard_msgs` text; the new helper must reproduce it byte for byte. If a text would change, stop and list it (spec §Edge Cases: pinned diagnostics).
- `Registry.SetEntry.purpose : SetPurpose`; update the readers in `Temporal/Case/Syntax.lean`.

### Investigation targets
**Required:**
- `model/Umpire/Command/Syntax.lean:640-700` (`gapKindTerm`), `:2670-2780` (set elaborator)
- `model/Umpire/Command/Registry.lean:155-175`
- `model/Temporal/Case/Syntax.lean:200-215`
- `model/Umpire/Command/Tests/**` — unknown-keyword pins

### Quick commands
```sh
cd model && lake build Umpire.Command.Syntax UmpireTests TemporalModelTests
make umpire-check-goldens
```

## Acceptance
- [ ] No elaborator in the listed files matches a keyword string against an existing enum
- [ ] `SetEntry.purpose` carries the enum; every unknown-keyword diagnostic byte-identical (existing `#guard_msgs` unchanged)
- [ ] Goldens, Case fixtures, Definition IDs byte-identical


## Done summary
Blocked:
Won't do (2026-10-01): the Lean model is retired in favour of the Scala front end (model/scalav2), and the Lean toolchain is removed. Spec closed as won't-do by the owner.
## Evidence
- Commits:
- Tests:
- PRs:
