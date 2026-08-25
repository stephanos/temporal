---
satisfies: [R3, R4, R5, R6]
---
# fn-9-umpire-reusable-dsl-package-split.6 Migrate Temporal Nexus adapters and inspector

## Description
Move the Nexus caller-closure scenario, integration assertions, and inspector under Temporal Umpire ownership (R3-R6). Add the final target names while retaining the old targets until the cutover task.

**Size:** M
**Files:** `model/Temporal/Umpire/NexusCallerClosure.lean`, `model/Temporal/Umpire/NexusCallerClosureTests.lean`, `model/Temporal/Umpire/Inspect.lean`, `model/Temporal/Umpire/testdata/nexus-caller-closure-experiment-spec.json`, `model/Umpire/Examples/testdata/switch-experiment-spec.json`, `model/TemporalUmpireTests.lean`, `model/Temporal.lean`, `model/lakefile.toml`
**Touches:** [model/Temporal/Umpire/**, model/Umpire/Examples/testdata/switch-experiment-spec.json, model/TemporalUmpireTests.lean, model/Temporal.lean, model/lakefile.toml]

### Approach
- Move the Nexus model/proofs and update only its truthful source provenance and namespace/imports.
- Split Nexus and cross-scenario integration assertions into `TemporalUmpireTests`.
- Move the inspector to `Temporal.Umpire.Inspect`; continue registering both Nexus and `Umpire.Examples.Switch`.
- Add the `TemporalUmpireTests` and `temporal-umpire-inspect` targets alongside old targets so both build surfaces remain available until final cutover.
- Before cutover, capture canonical outputs from the old inspector for both scenarios, apply exactly the two approved source-path substitutions, persist those target-state JSON fixtures, and compare the new inspector byte-for-byte to them.
- Update the Temporal aggregate to import the new adapter surface without re-exporting an old compatibility namespace.

### Investigation targets
**Required** (read before coding):
- `model/Temporal/Experiment/NexusCallerClosure.lean:1-110` — domain import, source provenance, laws, and proofs
- `model/Temporal/Experiment/NexusCallerClosure.lean:373-633` — target/property/behavior/query assembly
- `model/Temporal/Experiment/NexusCallerClosure.lean:698-713` — planner runs and artifact
- `model/Temporal/Experiment/Inspect.lean:44-85` — registry, dispatch, and error/output contract
- `model/Temporal/ExperimentTests.lean:1-242` — mixed aggregate tests to partition
- `model/Temporal.lean:1-6` — current public aggregate imports

**Optional** (reference as needed):
- `model/lakefile.toml:1-16` — current target naming/root declarations

### Key context
All scenario/query identifiers and inspector JSON semantics remain stable. The only Nexus artifact delta is the truthful source path. Stay within the task-listed investigation targets and touched model/build files; do not broaden implementation searches beyond the plan's positive allowlist.
## Acceptance
- [ ] Nexus declarations/proofs compile under `Temporal.Umpire.*` and retain identities, digests, laws, and query outcomes.
- [ ] `TemporalUmpireTests` owns Nexus and cross-scenario integration assertions.
- [ ] `temporal-umpire-inspect` registers both scenario ids and preserves deterministic JSON plus unknown-scenario failure behavior.
- [ ] Target-state fixtures are derived from the old inspector by exactly the switch and Nexus source-path substitutions, and the new inspector matches them byte-for-byte.
- [ ] Literal suite assertions retain scenario identities, digests, format versions, planner outcomes, and portable artifact fields after old targets disappear.
- [ ] `Umpire` remains free of Temporal/Nexus imports while Temporal imports Umpire in one direction.
- [ ] New and old targets coexist and build until the final cutover.
- [ ] `make umpire-check-regression` remains green.
## Done summary
Moved the Nexus caller-closure scenario, inspector, and integration assertions into Temporal.Umpire ownership while retaining the old targets for cutover. Added target-state fixtures derived by exactly the two approved source-path substitutions; the new inspector matches both byte-for-byte and preserves unknown-scenario behavior.

baseline: green via receipt
GATE_SKIPPED:smoke:green-receipt dd731303 - baseline reused from prior post-gate pass
stage: impl-review - ran (SHIP)
## Evidence
- Commits: fb591d408089d1ed75152d67ad8c3fceab8417c4
- Tests: GATE_SKIPPED:smoke:green-receipt dd731303 - baseline reused from prior post-gate pass, cd model && mise exec -- lake build TemporalUmpireTests temporal-umpire-inspect, new temporal-umpire-inspect Nexus/switch outputs cmp byte-for-byte with target-state fixtures; unknown scenario exits 1 with canonical diagnostic, make umpire-check-regression
- PRs: