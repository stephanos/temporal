---
satisfies: [R1, R3]
---
# fn-114-state-every-scala-model-declaration-once.7 Retire the string-named declaration forms from the DSL and the lifter

## Description
Once no Model, fixture or test uses them, remove the string-named overloads of every R2 declaration kind from `model/umpire` and their cases from the lifter, keeping only fn-112's one explicit-name form. Touch the standalone activity Model only if it still uses a retired form (spec Boundaries).

**Size:** M
**Files:** `model/umpire/{Action,Assume,Monitor,Channel,Machine,Compose,Claims}.scala`; `model/lifter/{Declarations,Claims,Compositions,Constants}.scala`; refusal fixtures under `model/lifter/testdata/`; `model/README.md`, `model/SEMANTICS.md` author-surface sections.
**Touches:** [model/umpire/**, model/lifter/**, model/temporal/standaloneactivity/**, model/README.md, model/SEMANTICS.md]

### Approach
- Inventory callers per overload with grep before deleting (line refs as of 2026-10-03, re-locate after fn-112: `action`/`timer`/`internal` Action.scala:107-113, `assume`/`hole`/`leadsTo` Assume.scala:15-56, `monitor` Monitor.scala:37, `channel` Channel.scala:88, `machine`/`restrict` Machine.scala:59/:140, `compose`/`sync` Compose.scala:23/:47, `property`/`scenario`/`Limits`/`query` Claims.scala:81-159).
- Delete each overload and its lifter branch (Declarations.scala:35-429, Claims.scala:82-252, Compositions.scala:46) in the same change; for each, add a fixture proving the old form no longer compiles or is refused at its line.
- Rewrite any test still using a form in the same task (R3 errors clause). Update docs to show one way to name a declaration.

### Investigation targets
**Required:**
- `model/umpire/Claims.scala`, `model/umpire/Machine.scala`
- `model/lifter/Declarations.scala`, `model/lifter/Claims.scala`
- `model/lifter/test/Fixtures.test.scala:200-250` - refusal assertion format `File.scala:line:col`

### Quick commands
```bash
grep -rnE '(machine|property|scenario|query|action|timer|monitor|channel|assume|hole)\("' model/temporal model/lifter/testdata
scala-cli test model/lifter
make umpire-check-model
```

### Execution constraints
- No IR or Case byte changes; refusal fixtures are new files only.
## Acceptance
- [ ] No string-named overload of an R2 kind remains in `model/umpire` or the lifter, apart from fn-112's one explicit-name form.
- [ ] Each retired form has a fixture proving it no longer compiles or is refused at its line.
- [ ] README/SEMANTICS show one naming form; lifter tests, model gate and R1 goldens pass.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
