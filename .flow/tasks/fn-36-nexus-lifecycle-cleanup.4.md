---
satisfies: [R4, R5]
---
# fn-36-nexus-lifecycle-cleanup.4 Publish and enforce the core versus experimental Nexus layout

## Description
Update live architecture/learning documentation and exact import-policy references, then run the complete verification matrix (R4, R5). Do not edit concurrent fn-31/fn-34 Flow planning artifacts.

**Size:** M
**Files:** `model/README.md`, `model/ARCHITECTURE.md`, `.plans/UMPIRE4_SPEC.md`, `.plans/UMPIRE4_SPEC_MODEL_ARCH.md`, `.plans/UMPIRE4_SPEC_COMPS.md`, `model/ModelLint/ImportGraph.lean`, `model/ModelLint/ImportGraphTests.lean`
**Touches:** [model/README.md, model/ARCHITECTURE.md, .plans/UMPIRE4_SPEC.md, .plans/UMPIRE4_SPEC_MODEL_ARCH.md, .plans/UMPIRE4_SPEC_COMPS.md, model/ModelLint/ImportGraph.lean, model/ModelLint/ImportGraphTests.lean]

### Approach
- Rewrite the live learning path and dependency map to lead with Lifecycle/Operations start-cancel-complete and label AutoClose/CallerClosure as inspectable experimental material.
- Update exact CallerClosure Veil verification-consumer namespace references in normative architecture docs and import-graph policy/tests.
- Scan live nonhistorical source/build/docs for deleted Examples/root experimental paths, excluding completed Flow history and the design record that explains the migration.
- Run focused Lean targets, Temporal/experimental aggregates, deterministic projection regression, and model lint.

### Investigation targets
**Required** (read before coding):
- `model/README.md:77-109` — current learning path/build commands.
- `model/ARCHITECTURE.md:40-93` — imports and dependency diagram.
- `model/ARCHITECTURE.md:141-225` — model ownership and references.
- `.plans/UMPIRE4_SPEC.md:180-188` — exact verification consumer rule.
- `model/ModelLint/ImportGraph.lean:100-115` — executable exact consumer policy.

### Key context
- Do not add a new general Experimental import-lint subsystem; update the exact existing names and keep ordinary facade isolation directly verified.
- Historical completed Flow specs/tasks may retain old paths; active dirty fn-31/fn-34 Flow state belongs to the user.

### Acceptance
- [ ] Live docs lead with root Lifecycle/Operations and accurately describe Experimental opt-in and build targets.
- [ ] Normative/executable exact verification-consumer references use the experimental namespace.
- [ ] Live obsolete-path scan is clean within the defined nonhistorical scope.
- [ ] All focused, aggregate, regression, generation, and lint gates pass.

## Acceptance
- [ ] R4 facade/build/import separation is documented and mechanically verified.
- [ ] R5 live references and docs contain no obsolete paths.
- [ ] Full verification matrix passes.
- [ ] Concurrent dirty Flow planning artifacts remain untouched.

## Done summary
Published the core-versus-experimental Nexus layout in README, architecture, and normative model documents; classified the separate experimental test aggregate; updated the exact CallerClosure Veil consumer namespace; and verified ordinary facade isolation and a clean live obsolete-path scan. Preserved the concurrent import-linter refactor while applying narrow policy changes. stage: plan-sync - skipped(config: planSync.enabled != true). A concurrent external wip commit captured the shared-worktree changes; this agent did not create it.
## Evidence
- Commits: 7f407525e
- Tests: cd model && mise exec -- lake build Temporal.Feature.Nexus.LifecycleTests Temporal.Feature.Nexus.OperationsTests Temporal.Feature.Nexus.Experimental.CallerClosureTests TemporalModelTests TemporalExperimentalTests, GOCACHE=<task-cache> make umpire-gen-regression-projections, GOCACHE=<task-cache> make umpire-check-regression, make lint-model
- PRs: