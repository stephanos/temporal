---
satisfies: [R2]
---
# fn-156-wasm-gomad-productionization.2 Bounded exploration, resume and non-progress outcomes

## Description
Route choice exploration through existing frontier owners with forced-prefix identities, breadth-first round order, stopping predicates and immutable round transactions. Integrate retain-success, recovery/resume and compatible minimization. Report CPU-loop non-progress and capacity outcomes honestly.

Implementation may develop against the existing cooperative API while `.8` and `.1` complete, following the repository's batch-first strategy. Final admission/completion remains dependent on their verified gates. The shared Runner currently rejects non-seed backend strategies; narrowly extend that boundary and its backend resume/minimizer seams while preserving native defaults, canonical records and actual frontier/minimizer ownership.

**Touches:** `tools/gomad_wasm/backend/**`, `tools/gomad3/runner/**`

Quick commands: focused backend exploration/recovery/resume/non-progress integration tests using fresh guests, shared Runner regression tests, affected lint, `make -C tools/gomad3 validate` for changed protocol/generation inputs. Final evidence must compare canonical rounds and prove interrupted execution recovery without loss/duplication. A fuel or wall watchdog is a bounded non-progress/infrastructure observation, never a replayable application failure.
## Acceptance
Repeated campaigns have matching canonical rounds; interrupted campaigns resume without lost or duplicated executions; a CPU loop yields a bounded non-progress outcome.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
