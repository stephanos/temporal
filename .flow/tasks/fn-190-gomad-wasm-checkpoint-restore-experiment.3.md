---
satisfies: [R2, R3]
---
# fn-190-gomad-wasm-checkpoint-restore-experiment.3 checkpoint-probe verb: restore, recorded and fresh suffix, forced-prefix baseline

## Description
gomad checkpoint-probe --image DIR --mode recorded|fresh [--seed N] [--repeat K] [--baseline] ARTIFACT, compiled only with the determinism-fixture feature set. Instantiate a fresh store from the prepared module, overwrite memory/globals/tables from the image, restore each crate from its snapshot, compare restored commitment with the captured one, re-enter. Fresh mode: an experimental runtime hook resets only the scheduling-choice PRNG streams to the continuation seed at the validated cut, applied identically on the restored path and the forced-prefix baseline, same instrumented module. Report restore and continuation wall time and the first divergent field and ordinal. Depends on .2.

## Acceptance
Fixture: recorded suffix equals fresh replay at all three boundaries; fresh suffix equals forced-prefix baseline for at least eight seeds per boundary; memory that cannot be sized from the image is an infrastructure outcome; verb absent from release builds.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
