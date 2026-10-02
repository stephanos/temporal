---
satisfies: [R15]
---
# fn-93-simplify-the-lean-model.40 Delete Nexus DESIGN.md and fold COMPATIBILITY.md (G1, G5)

## Description
Lanes G1 and G5. DG1 (recommended: delete) — delete `model/Temporal/Feature/Nexus/DESIGN.md` (887), or move it verbatim to `.plans/` with a historical banner if DG1 is declined. Rewrite the 21 Lean comments citing it and `AUTHORING.md:9,45` (the `:45` line is a quoted region of `Caller/Model.lean:10`; change both together) and `Caller/COVERAGE.md:102` to stand alone. Fold `Umpire/Property/COMPATIBILITY.md` (28) into the Property module docstring. Share the Nexus action docstrings that four redeclarations copy.

**Size:** M
**Files:** `model/Temporal/Feature/Nexus/DESIGN.md`, Lean comments in `Temporal/Feature/Nexus/Tests/Commands.lean` (5), `Umpire/Command/Syntax.lean` (6), `Temporal/Feature/Nexus/Tests/Machines.lean` (3), `Umpire/Command/Authoring.lean` (2), `Umpire/Command/{Finite,Refinement}.lean`, `Temporal/Case/Catalog.lean`, `Temporal/Feature/Nexus/Caller/Model.lean:10`, `Temporal/Feature/Nexus/Success/Tests.lean:294`; `model/AUTHORING.md`; `model/Temporal/Feature/Nexus/Caller/COVERAGE.md`; `model/Umpire/Property/COMPATIBILITY.md`; `model/Umpire/Property.lean` docstring; `.plans/index.json` (`allowedMissingLinks` for `UMPIRE_CMP_FIZZBEE.md` → DESIGN.md and `UMPIRE_DSL_EXPERIMENT.md` → COMPATIBILITY.md, with reasons)
**Touches:** [model/Temporal/Feature/Nexus/**, model/Umpire/Command/**, model/Temporal/Case/Catalog.lean, model/AUTHORING.md, model/Umpire/Property/COMPATIBILITY.md, model/Umpire/Property.lean, .plans/index.json]

### Approach
- Record DG1 in the Done summary.
- Comment rewrites keep only the why; no section numbers of the removed doc.
- `.plans/index.json`: edit only the two documents' `allowedMissingLinks` arrays (shape at `:18-30`); run `go run ./tools/planindex`.

### Quick commands
```sh
cd model && lake build
go test ./tools/umpire/authoring/...
make umpire-check-plan-index
```

## Acceptance
- [ ] DG1 recorded; DESIGN.md out of `model/`; no Lean comment or doc cites it
- [ ] COMPATIBILITY.md folded into the Property docstring; Nexus action docstrings shared
- [ ] Drift test and plan-index check green


## Done summary
Blocked:
Won't do (2026-10-01): the Lean model is retired in favour of the Scala front end (model/scalav2), and the Lean toolchain is removed. Spec closed as won't-do by the owner.
## Evidence
- Commits:
- Tests:
- PRs:
