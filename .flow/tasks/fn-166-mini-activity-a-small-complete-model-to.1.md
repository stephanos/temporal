---
satisfies: [R1, R6]
---
# fn-166-mini-activity-a-small-complete-model-to.1 Build switch: MODEL_FEATURES narrows every Scala step to named feature folders

## Description
Add the build switch that narrows every Scala step to named feature folders without moving the parked Models (R1, R6).

**Size:** S
**Files:** `Makefile` (`MODEL_SOURCES`, the jar, gate, lint and test targets that use it), `model/check/Gate.scala` if the gate lists sources itself, `model/temporal/IrFiles.test.scala` and `MachineTablePins.test.scala` (read the compiled set, not the disk)
**Touches:** [Makefile, model/check/*.scala, model/temporal/*.test.scala]

### Approach
- `MODEL_FEATURES ?= activity nexus`; `MODEL_SOURCES` becomes project.scala, framework, foundations, actors, capabilities and `model/temporal/features/$(f)` per feature. Every target that lists Model sources uses the variable; the gate's own source list follows it (check how `Gate.scala` enumerates sources for the lift and pass the folders through or read the same variable).
- `IrFile.discover("temporal")` already finds only compiled classes; confirm the gate lifts exactly the discovered files and that `MachineTablePins`/`IrFilesTest` compare against the compiled set, so the narrowed switch does not fail on parked features' files on disk.
- Prove with `MODEL_FEATURES=nexus`: nexus alone compiles, lifts and passes the Scala suites, and `git status` shows no change under `features/activity`.

## Acceptance
- [ ] `MODEL_FEATURES` narrows compile, jar, gate, lint and the Scala test suites; the default is every feature and no existing command changes meaning.
- [ ] With one feature selected, the others are neither compiled nor lifted and are untouched on disk.
- [ ] `IrFilesTest` and `MachineTablePins` pass under both the default and a narrowed switch.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
