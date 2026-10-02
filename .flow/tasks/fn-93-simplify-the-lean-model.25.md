---
satisfies: [R6]
---
# fn-93-simplify-the-lean-model.25 Shared command helpers: seen keys, name resolution, undeclared names, evalDecl, ToExpr (A4)

## Description
Lane A4, part two, all in the command elaborators. One seen-keys loop per keyed command replaces the 28 `duplicateKeyMessage` checks (defined ~`Syntax.lean:1177`); one `resolveRegistered` helper replaces the 10 `realizeGlobalConstNoOverloadWithInfo` try/catch copies (~237, 689, 727, 1385, 1901, 1965, 2013, 2691, 2701, 2840) and `Temporal/Case/Syntax.lean:204`; one undeclared-name message function with a hint replaces the 8 (~187, 703, 858, 1183, 1186, 1678, 1682, 1780); one `evalDecl` replaces the `unsafe`/`implemented_by` pairs (~342, 350, 815, 830, 834, 2836, `Temporal/Case/Syntax.lean:194`); derived `ToExpr` replaces the hand-quoting functions.

**Size:** M
**Files:** `model/Umpire/Command/Syntax.lean`, `model/Temporal/Case/Syntax.lean`, a helper module (e.g. `model/Umpire/Command/Support.lean`)
**Touches:** [model/Umpire/Command/Syntax.lean, model/Umpire/Command/Support.lean, model/Temporal/Case/Syntax.lean]
**Depends on other specs:** fn-92.2/.4 edit `Syntax.lean`; line numbers are planning-time.

### Approach
- Capture every duplicate-key, unresolvable-name and undeclared-name `#guard_msgs` before refactoring; texts stay byte-identical.
- `ToExpr`: core deriving (needs `Lean` import; nested/mutual types become `partial`; no instance for `Json`/`HashMap`/`Float`). Keep a hand quote where deriving cannot serve and list it.
- `collectAxioms` reports nothing about `implemented_by` bodies; one `evalDecl` does not widen any axiom inventory, confirm with E2's checker.

### Investigation targets
**Required:**
- `model/Umpire/Command/Syntax.lean:170-200,330-360,1170-1200,2830-2848`
- `model/Temporal/Case/Syntax.lean:190-210`

### Quick commands
```sh
cd model && lake build Umpire.Command.Syntax UmpireTests TemporalModelTests
make umpire-check-goldens
```

## Acceptance
- [ ] One seen-keys check, one resolution helper, one undeclared-name function, one `evalDecl`; hand quoting replaced where deriving serves
- [ ] Every command `#guard_msgs` unchanged; goldens byte-identical


## Done summary
Blocked:
Won't do (2026-10-01): the Lean model is retired in favour of the Scala front end (model/scalav2), and the Lean toolchain is removed. Spec closed as won't-do by the owner.
## Evidence
- Commits:
- Tests:
- PRs:
