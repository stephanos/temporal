---
satisfies: [R4, R5]
---
# fn-11-basic-nexus-umpire-dsl-showcases.3 Publish the basic Nexus learning path

## Description
Expose the new examples through the final Temporal build/test surfaces, reconcile all live architecture documentation with the layout dependency, and publish the progressive Umpire learning path (R4, R5). Position caller-closure as the advanced integration reference while keeping the inspector and canonical scenario surface unchanged.

**Size:** M
**Files:** `model/Temporal.lean`, `model/TemporalModelTests.lean`, `model/README.md`, `model/ARCHITECTURE.md`
**Touches:** [model/Temporal.lean, model/TemporalModelTests.lean, model/README.md, model/ARCHITECTURE.md]

### Approach
- Import the production example module from the final Temporal aggregate and its focused test module from the import-only Temporal model test root.
- Reconcile the full architecture document with `fn-10`'s final Feature/System/Tool layout, including the library table, focused imports, dependency map, Temporal semantic APIs, test root, inspector executable, and reference-example progression.
- Update the semantic-authoring overview to teach Switch first, then the basic Nexus target/walkthroughs, then caller-closure for advanced composition.
- Describe what Property, Behavior, and Query each contribute, and include focused build commands using the final target names from `fn-10`.
- State explicitly that model artifacts are pure and the examples do not execute Nexus operations.
- Leave the Tool inspector registry, regression manifest, fixtures, Lake targets, and Makefile unchanged; use stale-name checks plus the full regression command to verify those boundaries.

### Investigation targets
**Required** (read before coding):
- `model/README.md:67-103` — current authoring, regression, and inspection learning path
- `model/ARCHITECTURE.md:7-66` — current library table, focused imports, and dependency map with pre-cutover names
- `model/ARCHITECTURE.md:284-339` — current Temporal semantic APIs, inspector, and reference-example progression
- `.flow/specs/fn-10-temporal-semantic-model-layout-and.md` — binding final Feature/System/Tool architecture and public names
- `.flow/tasks/fn-10-temporal-semantic-model-layout-and.6.md` — final import-only test aggregate contract
- `.flow/tasks/fn-10-temporal-semantic-model-layout-and.7.md` — final aggregate, target, command, and README cutover

**Optional** (reference as needed):
- `model/Temporal.lean:1-4` — current aggregate before the `fn-10` replacement

### Key context
- `fn-10` does not schedule `model/ARCHITECTURE.md`; this task owns removing all pre-cutover names from that live document, not only adding the new example section.
- The inspector remains a production regression surface, not a requirement for every teaching module. Avoid adding fixture ceremony to examples whose contract is direct Lean compilation.
- Preserve every existing explanatory comment in touched Lean files; aggregate changes should remain import-only.

### Acceptance
- [ ] Final Temporal and TemporalModelTests aggregates compile the production examples and focused tests respectively.
- [ ] The complete architecture document matches the final Feature/System/Tool layout, imports, dependency graph, test root, and inspector executable.
- [ ] Live model documentation contains no `NexusAutoClose` standalone-library entry, `Temporal.Umpire.*`, `TemporalUmpireTests`, or `temporal-umpire-inspect` references after the dependency cutover.
- [ ] Model documentation gives an explicit Switch -> basic Nexus -> caller-closure learning progression and distinguishes each DSL's role.
- [ ] Documented commands use the post-`fn-10` target names and pass.
- [ ] Caller-closure remains documented as advanced and its inspector identity/output/diagnostics remain unchanged.
- [ ] `make umpire-check-regression` passes with no registry, fixture, manifest, Lake, Makefile, or reusable-Umpire changes.
## Acceptance
- [ ] R4's final-layout reconciliation, public/test imports, learning-path documentation, and commands are complete.
- [ ] Explicit stale-name checks cover all live model architecture and usage documentation.
- [ ] R5's unchanged advanced/reusable contracts are verified by the full regression.
- [ ] Existing comments in touched files are preserved.
## Done summary
Published the basic Nexus learning path through the final Temporal production/test aggregates and reconciled the live model documentation with the Feature/System/Tool layout. The guide now progresses from Switch through the shared Nexus lifecycle and two basic operations to caller-closure as the advanced integration reference.

baseline: green via handoff (verified at 27f5b3e60 by fn-11-basic-nexus-umpire-dsl-showcases.2; HEAD moved afterward only through that task's plan-sync receipt)
GATE_SKIPPED:unittest:green-receipt 5e9f94a2 - baseline reused from prior post-gate pass
stage: impl-review - ran (codex; SHIP at 2026-08-26T01:56:34Z)
## Evidence
- Commits: 18cf9ba2a3676e7c5a7d65db694be7383000a768
- Tests: baseline: green via handoff (verified at 27f5b3e60 by fn-11-basic-nexus-umpire-dsl-showcases.2; HEAD moved afterward only through that task's plan-sync receipt), GATE_SKIPPED:unittest:green-receipt 5e9f94a2 - baseline reused from prior post-gate pass, cd model && mise exec -- lake build Umpire.Examples.Switch Temporal.Feature.Nexus.Examples.BasicLifecycleTests Temporal.Feature.Nexus.Examples.BasicOperationsTests Temporal.Feature.Nexus.CallerClosureTests Temporal TemporalModelTests temporal-model-inspect, stale-name and four-file boundary checks, cd model && mise exec -- lake build TemporalModelTests, make umpire-check-regression
- PRs: