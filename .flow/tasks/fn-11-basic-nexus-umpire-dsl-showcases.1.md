---
satisfies: [R1, R5]
---
# fn-11-basic-nexus-umpire-dsl-showcases.1 Build the basic Nexus lifecycle teaching target

## Description
Create the shared Temporal Feature adapter that lets the two walkthroughs use one small, authoritative Nexus lifecycle target (R1, R5). Keep target composition and bounded-planner scaffolding behind a narrow interface so later example code reads as Property, Behavior, and Query intent.

**Size:** M
**Files:** `model/Temporal/Feature/Nexus/Examples/BasicLifecycle.lean`, `model/Temporal/Feature/Nexus/Examples/BasicLifecycleTests.lean`
**Touches:** [model/Temporal/Feature/Nexus/Examples/BasicLifecycle.lean, model/Temporal/Feature/Nexus/Examples/BasicLifecycleTests.lean]

### Approach
- Target the final Feature namespace produced by `fn-10`; reuse its authoritative Nexus lifecycle transition function rather than copying state semantics.
- Adapt only scheduled, started, and succeeded teaching states plus start and succeed actions.
- Mirror the single-capability/single-provider target shape from the Switch example; do not add a connector, ownership relation, alternate provider, or domain-neutral Umpire declaration.
- Encapsulate resolved setups, transition results, finite-completeness evidence, deterministic bounds/policy, and the incremental planner kernel needed by downstream walkthroughs.
- Keep checked composition failures visible through existing typed results; avoid unsafe or silent fallback values.

### Investigation targets
**Required** (read before coding):
- `model/NexusAutoClose.lean:200-312` — current authoritative states, events, and partial lifecycle transition; `fn-10.3` relocates this module before this task runs
- `model/Umpire/Examples/Switch.lean:63-247` — canonical single-provider transition target and composition pattern
- `model/Umpire/Examples/Switch.lean:405-586` — finite completeness and incremental planner-kernel pattern
- `.flow/tasks/fn-10-temporal-semantic-model-layout-and.3.md` — final lifecycle ownership and namespace contract

**Optional** (reference as needed):
- `model/Temporal/Umpire/NexusCallerClosure.lean:410-714` — advanced shape whose extra capabilities and queries must not leak into this target

### Key context
- Nexus has start and cancel protocol verbs, while successful completion is handler-reported lifecycle progress; name the examples so they do not imply `succeed` is a caller command.
- The final source path and namespace come from `fn-10`; do not add transitional imports or compatibility aliases.

### Acceptance
- [ ] One Temporal-owned target composes with exactly one capability/provider path.
- [ ] Scheduled + start produces started, and started + succeed produces succeeded, with authoritative outcomes/observations owned by the target.
- [ ] At least one unsupported state/action pair produces no transition.
- [ ] Finite completeness and the incremental planner kernel cover exactly the exposed teaching surface and are deterministic.
- [ ] Focused Lean checks compile without changing reusable Umpire or caller-closure behavior.

## Acceptance
- [ ] R1's shared target, valid transitions, and unsupported-transition behavior are covered.
- [ ] Existing typed composition failures remain observable.
- [ ] No reusable Umpire or advanced scenario contract changes.
- [ ] Existing comments in touched files are preserved.

## Done summary
Added the Temporal-owned basic Nexus lifecycle target, deriving its exposed transitions from `AutoClose.step` and encapsulating checked composition, finite completeness, deterministic bounds/policy, and the incremental planner kernel. Added focused coverage for the two teaching transitions, unsupported pairs, exact target cardinality, deterministic enumeration, and typed missing/conflicting-provider failures.

stage: impl-review - ran (codex; SHIP at 2026-08-26T01:30:19Z)
## Evidence
- Commits: 4306c21056921b86ec2ed63b0be4fbd122997b25, 3bf844a6f5a6852e315bb106c44bd0abf999ec8e
- Tests: baseline: green (cd model && mise exec -- lake build TemporalModelTests), baseline: green (make umpire-check-regression), cd model && mise exec -- lake build Temporal.Feature.Nexus.Examples.BasicLifecycle, cd model && mise exec -- lake build Temporal.Feature.Nexus.Examples.BasicLifecycleTests, cd model && mise exec -- lake build TemporalModelTests, make umpire-check-regression
- PRs: