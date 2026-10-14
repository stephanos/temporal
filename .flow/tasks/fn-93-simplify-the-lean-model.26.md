---
satisfies: [R6]
---
# fn-93-simplify-the-lean-model.26 Machine command duplicates and the shared admission setup (A4)

## Description
Lane A4, part three. Remove the duplicates inside `machine` (`*KeyFor`, `keyArray`/`keyList`, the `ends:`/`starts:` field lookup, the `ClassValue.head` match), make `find:`/`verify:` share one rule, and have `Command/Instances.lean`'s admission setup call `Authoring.checkAdmitted`'s.

**Size:** M
**Files:** `model/Umpire/Command/Syntax.lean` (machine elaborator), `model/Umpire/Command/Instances.lean`, `model/Umpire/Command/Authoring.lean`
**Touches:** [model/Umpire/Command/Syntax.lean, model/Umpire/Command/Instances.lean, model/Umpire/Command/Authoring.lean]
**Depends on other specs:** fn-92.2 (`compose`) and fn-92.4 (`restrict:`/`extend:`) extend the machine path.

### Approach
- Pin the machine command's current diagnostics (`#guard_msgs` in `Command/Tests/**` and Nexus `Tests/Machines`) and machine outputs (Definition IDs of every machine in the Models) before refactoring.
- `find:`/`verify:`: one rule parameterized by the verb; keep each verb's text.

### Investigation targets
**Required:**
- `model/Umpire/Command/Syntax.lean` machine elaborator (~1690-2420)
- `model/Umpire/Command/Instances.lean` admission setup
- `model/Umpire/Command/Authoring.lean` `checkAdmitted`

### Quick commands
```sh
cd model && lake build
make umpire-check-goldens umpire-check-regression
```

## Acceptance
- [ ] Listed duplicates gone; `find:`/`verify:` share one rule; Instances reuses `checkAdmitted`'s setup
- [ ] Machine diagnostics and every Definition ID byte-identical


## Done summary
Blocked:
Won't do (2026-10-01): the Lean model is retired in favour of the Scala front end (model/scalav2), and the Lean toolchain is removed. Spec closed as won't-do by the owner.
## Evidence
- Commits:
- Tests:
- PRs:
