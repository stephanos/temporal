---
satisfies: [R1, R2, R5, R6, R12]
---
# fn-114-state-every-scala-model-declaration-once.4 Convert the Nexus close-policy declarations to the final DSL in place

## Description
The close-policy Model is the largest remaining one (`Model.scala` 702 lines, `Claims.scala` 465 lines, 147 literals, 19 roots in `nexus-close.json`), so it is split across two tasks: this one converts declarations without moving files; task 5 splits the files. Depends on task 2 because it imports the caller's renamed declarations. It also defines the one construct this spec adds beyond the IR-file declaration, the monitor pattern `sticky` (spec API Contracts, R12), because both of its uses are this Model's `retainedOutcome` and `ownerAcknowledgment`.

**Size:** M
**Files:** `model/temporal/nexuscaller/closepolicy/Model.scala`, `Claims.scala`; `model/umpire/Syntax.scala` (`sticky`, `stickyAcross`, documented with the `monitor` form they stand for; `Monitor.scala` is not edited and imports nothing from it); `model/lifter/Syntax.scala` (their matching, lowering to the tree `monitorOf` at `model/lifter/Declarations.scala:365-405` builds for the `monitor[...](name, initial)(next)(violated)` form; `Declarations.scala` gains at most one hook); `model/lifter/testdata/lifts/` (a lifting fixture declaring both forms on a fixture machine beside their `monitor` spellings, expected JSON proving the two trees equal) and `Rejects.scala` (a predicate of the wrong state type, recorded as the compiler refusal it is); `model/lifter/test/Fixtures.test.scala` roots.
**Touches:** [model/temporal/nexuscaller/closepolicy/**, model/umpire/Syntax.scala, model/lifter/Syntax.scala, model/lifter/Declarations.scala, model/lifter/testdata/**, model/lifter/test/Fixtures.test.scala, model/ir/nexus-close.json]

### Approach
- Same surface as task 2: captured names (rename the `val` or the one explicit form for differing names), DefinitionScope pins per former owner, family `given`, machine derivation in place of copied machines, typed compositions and syncs, step helpers, captured action inputs, named choices, evidence exceptions only, refinement reads instead of `Reads.through`.
- `sticky(predicate)` and `stickyAcross(predicate)` (`.plans/TEMPORAL_PATTERNS.md` section 2.4): plain `def`s in `model/umpire/Syntax.scala` the lifter matches by name in `model/lifter/Syntax.scala`, the monitor's name taken from the `val` (R2). Lower to the existing `ir.Monitor` with Boolean state, `initial = false`, `next = or(m, not(call(p, after)))` (or `call(p, before, after)` for the two-state form), `violated` the identity, read after every step; this is the tree the two lambdas lift to today, so the `nexus-close.json` monitor entries and the monitor-agreement exports stay byte-identical. Rewrite `retainedOutcome` (`Model.scala:436`) and `ownerAcknowledgment` (:445) with it; `singleOutcome` and `cancelPrincipal` need history and stay hand-written.
- Core and sugar (fn-112's rule, this spec's API Contracts): the conversion uses only sugar forms fn-112 placed in `Syntax.scala` files and adds none outside them; `sticky` is the one sugar form this spec adds.
- Keep files where they are so diffs stay reviewable; record each construct that does not fit with its reason (R6).

### Investigation targets
**Required:**
- `model/temporal/nexuscaller/closepolicy/Model.scala`, `Claims.scala`
- `model/temporal/nexuscaller/` after task 2 - the pattern
- `model/lifter/Declarations.scala:365-405` - `monitorOf`
- `model/umpire/Syntax.scala` and `model/lifter/Syntax.scala` (post fn-112) - where sugar and its matching live
**Optional:**
- `model/ir/nexus-close.json` - names and IDs to keep
- `model/lifter/testdata/lifts/CloseReset.scala` - fixture style

### Quick commands
```bash
make umpire-gen-model && git diff --stat model/ir
make umpire-check-model
scala-cli test model/lifter
```

### Execution constraints
- `nexus-close.json` changes only within the R1 allowed-difference list recorded in the spec (fn-120 inert choice names, file-move source paths/lines and `source` root strings, moved-function symbols recorded as `function_name_substitutions` in `tools/umpire/internal/golden/config.json`, and fn-112 R1's function projection); tables, IDs, fingerprints and answers exact; the two `sticky` monitors are tree-identical to today's.
- fn-122 (capabilities) may be running alongside: it adds `model/umpire/Capabilities.scala` and `model/lifter/Capabilities.scala` and does not edit `Monitor.scala`, `monitorOf` or the `Syntax.scala` files; whichever lands second rebases.
## Acceptance
- [ ] Every R2 declaration in closepolicy takes its name from its `val` or the one explicit form; evidence lists only exceptions; no `Reads.through` remains.
- [ ] Machines are derived rather than copied, compositions use typed selectors, branches use named choices, inputs use captured tokens; kept old forms are listed with reasons.
- [ ] `sticky` and `stickyAcross` exist in `model/umpire/Syntax.scala` as plain `def`s documented with their `monitor` form, matched in `model/lifter/Syntax.scala`; `retainedOutcome` and `ownerAcknowledgment` use them; the monitor entries of `nexus-close.json` are byte-identical; one lifting fixture proves both arities IR-equal to their `monitor` spellings and one refusal (wrong state type) is recorded at the compiler layer; `Monitor.scala` imports nothing from `Syntax.scala`.
- [ ] R1 goldens and fn-112.1 equivalence pass; model gate, lifter tests and lint-model pass.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
