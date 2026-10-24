---
satisfies: [R9]
---
# fn-195-modelsuite-a-models-test-file-holds-its.6 Standardize primary machine declarations on class Model

## Description
**Owner requests (2026-10-18):** "let's file a task for standardize primary machine declarations on class Model" and "we should also remove support for object Model entirely for consitency".

After the six samples have ModelSuite tests, make every primary native sample machine a `class Model`, including parameterless models. Each suite constructs a fresh instance in a stable val holder, or through the ModelSuite instance-binding form settled by the prototype. Bank, Voting and TwoPhaseCommit already use classes; migrate Kettle, ReliableBroadcast and TwoLayeredCache and reconcile all six test/realization clients with the same convention.

**Size:** L

**Touches:** active native sources and tests under `model/framework/dsl/**`, `model/framework/checker/**`, `model/framework/lint/**`, `model/framework/check/**`, `model/framework/realize/**`, `model/framework/samples/**`; `model/design/runs-in-tests.md` (primary-model construction examples only). Held export/IR/Temporal snapshots remain excluded.

Preserve state, starts, transitions, claims, domains, expected verdicts, counterexamples and realization ownership. Prove two instances of a formerly singleton model have independent state and exact instance-bound claims, and that their suites do not share mutable state. Keep stable instance naming and valid Derived/Composition source bindings. Update affected source and separate-library fixtures without weakening their checks.

Remove singleton primary-machine declaration support from native admission, naming and execution paths, rather than retaining an alternate object form behind a sample convention. Refuse `object Model extends Machine[...]` with a located diagnostic; renaming that singleton must not bypass the rule. Replace old positive object-form compatibility controls with negative authoring tests and migrate all active native Machine fixtures/clients to class instances while preserving their test purposes. No test-only compatibility escape hatch may keep the retired author form reachable.

Keep objects for namespaces, constants, actors and other shared declarations. This request concerns primary Machine declarations; do not blanket-convert unrelated object-based DSL forms or silently remove Derived/Composition functionality. fn-195.5 follows this task and documents the class-only primary declaration form in README, SEMANTICS and the user-facing reference without Scala-file references.

No held Temporal source, IR/export/lifter, Go, generated data or regeneration work. Historical uncompiled snapshots do not imply live support and need not be rewritten to make a grep clean. Future Temporal model migrations follow the class-only primary recipe when their holds are released. Keep this task todo until the preceding ModelSuite migration is complete; preserve Phase3 obligations explicitly if a held consumer depends on the retired form.

**Verification:** current full native and source-lint gates, fresh declaration/core/plugin compilation, and six-sample realization/public-question source and separate-library clients. Existing mathematical answers and test purposes remain unchanged.
## Acceptance
- [ ] All six primary native sample declarations use `class Model`, including parameterless models.
- [ ] Each ModelSuite binds a fresh stable-named instance; two-instance isolation and foreign-instance claim/root refusal have regression coverage.
- [ ] Sample behavior, intended verdicts, negative-control paths and realization bindings are unchanged; native and separate-library gates pass.
- [ ] Singleton primary Machine declarations are refused with a located diagnostic, regardless of object name; native admission/naming/execution have no compatibility escape hatch for the retired form.
- [ ] All active native object-Machine fixtures and clients are migrated with their original test purposes; source and separate-library negative controls reject the retired declaration.
- [ ] Objects remain available for namespaces, constants, actors and shared declarations; Derived/Composition behavior and held Temporal/IR bodies are not removed by this task.
- [ ] Construction examples are consistent, and the final documentation task explicitly consumes this convention.
## Done summary
Closed without implementation on 2026-10-24 at the owner's request. The Scala framework this task targets was frozen under `model/archived/` on 2026-10-18 and Umpire continues in Rust (`model/umpire`); the Scala-era backlog is retired rather than carried. Any intent here that still matters is re-filed against the Rust workspace when it is needed.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs: