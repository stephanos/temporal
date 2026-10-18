---
satisfies: [R1]
---
# fn-206-apalache-backend-in-scala.1 TLA+ emitter over the lifted tree with finite monitor automata

## Description
Implement `Tla.emit(lifted): Either[Unsupported, Module]` in framework/export/tla: types as TLA+ values, functions and expressions one to one, per machine the variables, Init, Next and end, monitors as finite automata, Properties as invariants or action properties, compositions as products with syncs conjoined; refusals for channels, holes, anonymous functions outside ends and unspellable names; `.tla` text and `.cfg`. Shared with fn-207: if fn-207 lands first, this task extends its emitter instead.

**Files:** new model/framework/export/tla/*.scala, tests over the six samples.

**Quick commands:** `mise exec -- scala-cli test --server=false model/framework/export/tla --require-tests`

## Acceptance
- Each sample emits or is unsupported with a positioned reason.
- No history variable in any module.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
