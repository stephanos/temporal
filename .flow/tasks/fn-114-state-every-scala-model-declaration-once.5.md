---
satisfies: [R1, R10, R11]
---
# fn-114-state-every-scala-model-declaration-once.5 Split the close-policy Model into the four-file layout

## Description
Move the converted close-policy declarations into fn-112's file names and remove the literals the conversion made avoidable.

**Size:** S
**Files:** `model/temporal/nexuscaller/closepolicy/{Model.scala, Claims.scala -> Properties.scala + Queries.scala}`, its task-1 IR-file declaration.
**Touches:** [model/temporal/nexuscaller/closepolicy/**, model/ir/nexus-close.json, tools/umpire/internal/golden/config.json]

### Approach
- Domains, actions, step functions, machines, compositions and monitors in `Model.scala`; Properties and progress claims in `Properties.scala`; Scenarios, Queries and Limits in `Queries.scala`. Omit a file with nothing of its kind. Keep the package so IDs stay exact. Moving functions out of `Claims.scala` renames their `Claims$package$` symbols (e.g. `awaitingOwner`, `nothingOwed`, `knows`); record each as a `function_name_substitutions` entry. If a move would change an IR name outside the allowed list, keep the declaration in its file and say so (R10 errors).
- Count this folder's literals with task 1's command and remove any outside fn-112 R18's three kinds that can go.

### Quick commands
```bash
make umpire-gen-model && git diff --stat model/ir
make umpire-check-model
```

### Execution constraints
- `nexus-close.json` changes only within the R1 allowed-difference list recorded in the spec (fn-120 inert choice names, file-move source paths/lines and `source` root strings, and moved-function symbols recorded as `function_name_substitutions` in `tools/umpire/internal/golden/config.json`, as fn-112's harness allows).
## Acceptance
- [ ] closepolicy has `Model.scala`, `Properties.scala`, `Queries.scala` and no `Claims.scala`.
- [ ] IR differs only by allowed file-move metadata; R1 goldens and the model gate pass.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
