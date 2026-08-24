---
satisfies: [R1, R2, R3, R4, R5, R6, R7]
---
# fn-1-lean-regression-dsl-and-nexus.2 Bind Nexus caller-closure pilot and inspector

## Description
Bind the compiler contract from task `.1` to the existing standalone Nexus caller-closure semantics, then add the single stdout-only inspection path required by R1-R7.

**Size:** M
**Files:** `model/Temporal/Experiment/NexusCallerClosure.lean`, `model/Temporal/Experiment/Inspect.lean`, `model/Temporal/ExperimentTests.lean`, `model/Temporal.lean`, `model/lakefile.toml`
**Touches:** [model/Temporal/Experiment/NexusCallerClosure.lean, model/Temporal/Experiment/Inspect.lean, model/Temporal/ExperimentTests.lean, model/Temporal.lean, model/lakefile.toml]

### Approach
- Define one current-model target and regression around the existing caller-closure clash configuration without modifying or reconstructing its semantics.
- Represent the pending user cancellation in the resolved setup, request caller force-close as intent, and project that mapped attempt through upgrade resolution; bind the non-empty expectation collection to the existing honored-delivery and cancellation-uniqueness properties as two distinct identities.
- Define a pure inspector runner that accepts a pilot registry and arguments and returns exit status, stdout, and stderr values. Keep the production registry closed to the single pilot, with a thin Lean executable that writes the runner result.
- Test CLI-level success and unknown-pilot behavior. Inject malformed registry entries only into runner tests to exercise incompatible-target and compiler-failure paths, asserting non-zero status, empty stdout, and exactly one structured stderr diagnostic.
- Extend the test umbrella with exact pilot output, repeated-output determinism, changed consumed semantic-contract identity, and action-attempt/outcome separation.
- Import the authored public entrypoint so established model builds include the pilot and its proofs.

### Investigation targets
**Required** (read before coding):
- `model/NexusAutoClose.lean:467-584` — existing initiator, policy, configuration, delivery, and auto-close semantics.
- `model/NexusAutoClose.lean:739-755` — caller-closure clash witness and reachability.
- `model/NexusAutoClose.lean:817-845` — checked honored-delivery property for upgrade.
- `model/NexusAutoClose.lean:870-932` — checked cancellation uniqueness property.
- `model/lakefile.toml:2` — current model build inclusion.

**Optional** (reference as needed):
- `model/Temporal.lean:1` — current authored entrypoint to extend.

### Key context
- Import and consume the standalone `NexusAutoClose` declarations directly; do not duplicate their state machine or infer semantics from generated descriptors.
- Identity is scoped to the canonical target slice consumed by this regression. Tests change a projected outcome or property observation contract, not unrelated metadata.
- Test injection belongs below the closed production registry and does not create a production mutation or configuration surface.
- Inspection is stdout-only. Do not add a Go wrapper, persistent output path, or artifact publisher.
- Do not inspect, cite, search, import, copy, adapt, or depend on any Umpire3 source or artifact.
- Preserve all existing comments in touched files.

### Task-scoped verification
- The baseline and completion command for this task is `make -C model check`, plus the focused Lean inspector executable/tests added here.
- `make umpire-check-regression` is a final spec Quick command whose top-level target is created by task `.3`; its absence is expected until that task and must not block `.2`.

## Acceptance
- [ ] The checked regression selects the caller-closure clash setup, force-close attempt, upgrade projection, finite declaration bounds, and both existing honored-delivery and cancellation-uniqueness expectations.
- [ ] The action attempt and setup-dependent projected outcome are separate artifact fields; a mapped attempt that is impossible for its setup produces `impossibleAction` instead of an `ExperimentSpec`.
- [ ] The inspector emits exactly one complete canonical `ExperimentSpec` JSON document on success without a Temporal environment or file write.
- [ ] CLI tests cover success and unknown pilot identity; pure runner tests cover incompatible target identity and compile failure. Every failure asserts non-zero status, empty stdout, and one structured stderr diagnostic.
- [ ] Repeated inspection is byte-identical; changing a consumed projected outcome or property observation contract produces distinguishable model identity/output.
- [ ] The exact pilot artifact contains both named property expectations and their resolved observation contracts.
- [ ] Established current-model builds elaborate the DSL, compiler, pilot binding, inspector, and tests.
- [ ] New pilot/inspector/test/build files contain no Umpire3 import, reference, copied contract, or dependency.

## Done summary
Bound the checked Nexus caller-closure clash and upgrade semantics to the regression compiler, including distinct force-close intent, projected outcome, and proof-backed honored-delivery and cancellation-uniqueness expectations. Added the closed stdout-only Lean inspector and focused deterministic success/failure fixtures; both task gates and the focused executable checks passed.

stage: impl-review - ran [2026-08-24T16:35:36Z..2026-08-24T16:39:13Z]

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 530fecb062b3b6aa1ecc35783aa44e374cbfe1c3
- Tests: make -C model check, make umpire-check-api, cd model && mise exec -- lake build ExperimentTests temporal-experiment-inspect, model/.lake/build/bin/temporal-experiment-inspect nexus-caller-closure-upgrade (success, canonical JSON, repeated-byte identity assertions), model/.lake/build/bin/temporal-experiment-inspect missing-pilot (non-zero, empty stdout, one structured stderr diagnostic assertions)
- PRs:
