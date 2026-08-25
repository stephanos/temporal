---
satisfies: [R3, R4, R6]
---
# fn-9-umpire-reusable-dsl-package-split.5 Promote the switch as a generic Umpire example

## Description
Move the synthetic switch scenario and its reusable assertions under Umpire ownership (R3, R4, R6). This proves a non-Temporal domain can consume the complete public stack.

**Size:** M
**Files:** `model/Umpire/Examples/Switch.lean`, `model/Umpire/Examples/SwitchTests.lean`, `model/UmpireTests.lean`
**Touches:** [model/Umpire/Examples/Switch.lean, model/Umpire/Examples/SwitchTests.lean, model/UmpireTests.lean]

### Approach
- Move the two-state switch target, properties, behaviors, queries, planner kernel, and compiled artifact into `Umpire.Examples.Switch`.
- Replace the generic example's old facade import with the narrow Umpire modules it consumes.
- Update its source provenance to the truthful new location while preserving all semantic identities and digests.
- Move switch-only and generic cross-DSL assertions out of the mixed aggregate test root into Umpire tests.

### Investigation targets
**Required** (read before coding):
- `model/Temporal/Experiment/SwitchScenario.lean:1-140` — generic imports, source, vocabulary, and kernel
- `model/Temporal/Experiment/SwitchScenario.lean:299-460` — checked property/behavior/query assembly
- `model/Temporal/Experiment/SwitchScenario.lean:598-611` — planner runs and compiled artifact
- `model/Temporal/ExperimentTests.lean:194-242` — switch and cross-scenario canonical assertions

**Optional** (reference as needed):
- `model/Temporal/Experiment/Inspect.lean:44-85` — current scenario registry contract retained by the Temporal inspector

### Key context
The switch must not import Temporal or Nexus. The Temporal inspector may import the generic example in the allowed direction during the next task.
## Acceptance
- [ ] The switch builds and plans using only public Umpire modules.
- [ ] Its identity, digests, selected trace, and artifact fields remain stable; only its source path changes.
- [ ] Generic switch/cross-DSL tests are owned by `UmpireTests` and retain deterministic assertions.
- [ ] A source scan finds no Temporal or Nexus import under the generic example.
- [ ] Existing regression output remains green before inspector cutover.

## Done summary
Promoted the two-state switch into `Umpire.Examples.Switch` using only the public Umpire planning stack, preserving its semantics and all existing comments while changing only truthful source provenance. Added focused Umpire-owned cross-DSL tests for identities, digests, deterministic traces, planner outcomes, and portable artifact fields without changing the pre-cutover Temporal regression path.

baseline: green via receipt
GATE_SKIPPED:smoke:green-receipt 5f574723 - baseline reused from prior post-gate pass
stage: impl-review - ran (SHIP)
## Evidence
- Commits: dd7313030891c80140087251c0333dc4469e3a77
- Tests: GATE_SKIPPED:smoke:green-receipt 5f574723 - baseline reused from prior post-gate pass, mise exec -- lake build UmpireTests, source scan: no Temporal or Nexus imports under model/Umpire/Examples, Lean canonical old/new artifact equivalence proof with only source-path substitution, make umpire-check-regression
- PRs: