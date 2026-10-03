---
satisfies: [R1, R2, R5, R6]
---
# fn-114-state-every-scala-model-declaration-once.4 Convert the Nexus close-policy declarations to the final DSL in place

## Description
The close-policy Model is the largest remaining one (`Model.scala` 702 lines, `Claims.scala` 465 lines, 147 literals, 19 roots in `nexus-close.json`), so it is split across two tasks: this one converts declarations without moving files; task 5 splits the files. Depends on task 2 because it imports the caller's renamed declarations.

**Size:** M
**Files:** `model/temporal/nexuscaller/closepolicy/Model.scala`, `Claims.scala`.
**Touches:** [model/temporal/nexuscaller/closepolicy/**, model/ir/nexus-close.json]

### Approach
- Same surface as task 2: captured names (rename the `val` or the one explicit form for differing names), DefinitionScope pins per former owner, family `given`, machine derivation in place of copied machines, typed compositions and syncs, step helpers, captured action inputs, named choices, evidence exceptions only, refinement reads instead of `Reads.through`.
- Keep files where they are so diffs stay reviewable; record each construct that does not fit with its reason (R6).

### Investigation targets
**Required:**
- `model/temporal/nexuscaller/closepolicy/Model.scala`, `Claims.scala`
- `model/temporal/nexuscaller/` after task 2 - the pattern
**Optional:**
- `model/ir/nexus-close.json` - names and IDs to keep

### Quick commands
```bash
make umpire-gen-model && git diff --stat model/ir
make umpire-check-model
```

### Execution constraints
- `nexus-close.json` changes only within the R1 allowed-difference list recorded in the spec (fn-120 inert choice names, file-move source paths/lines and `source` root strings, and moved-function symbols recorded as `function_name_substitutions` in `tools/umpire/internal/golden/config.json`, as fn-112's harness allows); tables, IDs, fingerprints and answers exact.
## Acceptance
- [ ] Every R2 declaration in closepolicy takes its name from its `val` or the one explicit form; evidence lists only exceptions; no `Reads.through` remains.
- [ ] Machines are derived rather than copied, compositions use typed selectors, branches use named choices, inputs use captured tokens; kept old forms are listed with reasons.
- [ ] R1 goldens and fn-112.1 equivalence pass; model gate and lint-model pass.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
