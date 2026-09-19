---
satisfies: [R4, R6]
---
# fn-31-deepen-umpire-target-and-simplify.5 Enforce Target architecture and authoring documentation

## Description
Close R4 and R6 with facade/import/mutation coverage and synchronized authoring guidance.

### Review reconciliation (normative)

Target mutations in this task are exactly the existing `DeclarationErrorKind` cases: identity syntax/duplicates, unknown/wrong kind, missing/unexpected/mismatched law, missing/conflicting provider, ambiguous connector, and `KernelAvailability.incomplete`. Query bound/unit, role/action finite-completeness, and `targetKernelMismatch` mutations belong exclusively to Tasks `.7` and `.6`; they must not be moved into Target.

**Size:** M
**Files:** `model/Umpire.lean`, `model/UmpireTests.lean`, `model/TemporalModelTests.lean`, `model/ModelLint/ImportGraph.lean`, `model/ModelLint/ImportGraphTests.lean`, `.plans/UMPIRE4_SPEC.md`, `.plans/UMPIRE4_DSL.md`, `.plans/UMPIRE4_SPEC_MODEL_ARCH.md`, `model/README.md`, `model/Umpire/ARCHITECTURE.md`, `model/ARCHITECTURE.md`
**Touches:** [model/Umpire.lean, model/UmpireTests.lean, model/TemporalModelTests.lean, model/ModelLint/ImportGraph.lean, model/ModelLint/ImportGraphTests.lean, .plans/UMPIRE4_SPEC.md, .plans/UMPIRE4_DSL.md, .plans/UMPIRE4_SPEC_MODEL_ARCH.md, model/README.md, model/Umpire/ARCHITECTURE.md, model/ARCHITECTURE.md]

### Approach
- Test public facades and forbidden import directions.
- Extend fn-34's import-graph policy/tests with a Target-specific transitive rule: every `Umpire.Target.*` module, including `Umpire.Target.Tests.*`, must remain unable to reach Query, Planning, Artifact, Temporal, runtime, or verification modules. Cross-layer compatibility tests live under downstream `Umpire.Examples.*` or `Umpire.Tests.*` namespaces instead. Add controlled positive and negative import-graph fixtures; do not add another import scanner or a test exemption.
- Add independent Target mutations for provider, connector, law, identity/kind, and `KernelAvailability.incomplete` errors.
- Reconcile AUT-07 and the DSL core decision so Property/Behavior/Query are the only scenario/question languages while checked Target is their semantic-model substrate.
- Document the implemented typed facade, ordinary versus maintainer responsibilities, Query/Planning ownership, and Switch-to-Nexus learning path after the API is final; do not invent macro syntax or update Umpire3 docs.

### Investigation targets
**Required** (read before coding):
- `model/Umpire.lean` — public aggregate
- `model/UmpireTests.lean` — aggregate test boundary
- `model/TemporalModelTests.lean` — ordinary Temporal test aggregate
- `model/ModelLint/ImportGraph.lean` and `model/ModelLint/ImportGraphTests.lean` — fn-34 enforcement substrate and controlled fixtures
- `.plans/UMPIRE4_SPEC.md:211-228` and `.plans/UMPIRE4_DSL.md:68-82` — normative authoring-path wording to reconcile
- `.plans/UMPIRE4_SPEC_MODEL_ARCH.md:114-197` — deep-module and author-role contract to preserve
- `model/Umpire/ARCHITECTURE.md:37-46` — current checked lifecycle

### Acceptance
- [ ] Import and domain-purity checks enforce the architecture, including Target test modules and transitive rejection of Query/Planning/Artifact/Temporal/runtime/verification reachability from `Umpire.Target.*` with positive and negative lint fixtures and no compatibility-test exemption.
- [ ] The exact Target mutation set fails at the Target boundary with source-located diagnostics; Query-owned mutations remain in Tasks `.7`/`.6`.
- [ ] Normative docs distinguish Target substrate from the Property/Behavior/Query scenario languages while retaining one semantic authority and no compatibility facade.
- [ ] Model docs teach the compiled typed facade, explicit semantic choices, maintainer/query-author roles, and Target-to-Query-to-Planning flow without ordinary raw provider/connector, completeness, finite-order, or planner-kernel construction.
## Acceptance
- [ ] R4 diagnostic mutations and R6 isolation checks pass.
- [ ] Aggregate Umpire/Temporal builds and regression gate pass.
- [ ] Documentation reflects the implemented interface without duplicating long-form architecture, promising a general macro DSL, or changing independent Umpire3 documentation.

## Done summary
Extended the existing import-graph checker with transitive Target isolation for production and test modules, and added source-located mutation coverage for all 12 Target-owned declaration failures. Public facade checks and synchronized architecture guidance now put the compiled AuthoredTarget → CheckedTarget → Query → Planning path and Switch-to-Nexus learning order front and center.

The pre-existing broad `.plans/UMPIRE4_SPEC.md` rewrite was preserved: only the AUT-07 substrate sentence was isolated into the task commit, and the unrelated `.flow/memory/declined/generated-api-drift-verification.md` edit remains untouched. Green gate receipts were not warrantable while those unrelated working-tree changes remained dirty.

baseline: red (`cd model && mise exec -- lake build Umpire.TargetTests Umpire.Query.Tests Umpire.Planning.Tests` hit the inherited transient Lake `.ilean` ENOENT in Query Forms/Validation; its exact retry passed, and the other four Quick commands passed pre-edit)
stage: impl-review - ran [2026-08-27T05:01:07Z..2026-08-27T05:05:13Z]
stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 8d014366201b7231a0497c358230a3702f6315cc
- Tests: cd model && mise exec -- lake build Umpire.TargetTests Umpire.Query.Tests Umpire.Planning.Tests, cd model && mise exec -- lake build Temporal.Feature.Nexus.LifecycleTests Temporal.Feature.Nexus.OperationsTests Temporal.Feature.Nexus.Experimental.CallerClosureTests, cd model && mise exec -- lake build UmpireTests TemporalModelTests, make umpire-check-regression, make lint-model
- PRs:
