---
satisfies: [R3]
---
# fn-169-a-temporal-free-framework-the-lifter.2 Plug-in seam: layout conventions and registered API descriptors (after fn-174.2)

## Description
**Recut 2026-10-18 (one-language series):** `framework/export` is `framework/lift` (fn-201); proto registration moves to `model/framework/proto` (fn-201.3, fn-202.1).

Runs after the committed local closure of fn-174.2, the approved successor of superseded fn-141.8.
Realization types now own conversion beside their definitions; this task does not restore a
tree-side registered kit or duplicate that producer.

1. **Plug-in seam.** The generic `framework/lift` receives Temporal specifics only from `temporal/export`: layout conventions (where features, foundations, actors and capabilities live) as configuration, and the target API's descriptors. The Temporal kit supplies its constructed realization messages through the fn-174.2 conversion boundary, not source-tree recognition.
2. **Generic proto support (owner, 2026-10-12).** A system registers its API proto roots and descriptor sets once. The proto-name lint, gate proto stage and constructed realization conversion consume that registration. Temporal supplies its `temporal.api.*` roots/descriptors through its own modules. Re-anchor the current locations and configuration after .2 rather than treating historical hard-coded lists as the final interface.
3. Prove it: a fixture system with its own small proto package and no Temporal API builds, lints and exports through the framework alone; the mini lift is byte-identical relative to this seam's immediate predecessor (apart from explicitly permitted recorded source paths), and mini equivalence holds.

**Touches:** model/framework/lift/**, model/framework/check/**, model/framework/realize/**, model/temporal/export/**, model/temporal/check/**, model/temporal/realize/**, model/project.scala, Makefile, model/README.md, .plans/UMPIRE_MODULES.md
**Scope:** generic layout/API registration and its Temporal configuration, exercised by `activity_mini`, the six samples and a small non-Temporal synthetic proto fixture. No real-feature source/payload, IR namespace/schema, Go consumer or checked-in regeneration is selected. The framework remains compilable without Temporal API jars or `temporal.*` source. The constructor conversion is a prerequisite, not an invitation to recreate a lifter or change Model spellings.

### Re-anchor and verification

- Verify fn-174.2 is `done` with its closure committed on `umpire`; read its final handover and
  conversion/context API. This cross-spec task precondition is conductor-checked because Flow
  task dependency edges cannot cross specs. The current macro-host design blocker is not waived.
- Survey current residual system constants and analogous registrations before selecting the
  smallest seam. Inject configuration through existing module boundaries; no generic module
  imports a Temporal implementation, and no empty registration silently selects Temporal defaults.
- Use one registered API source for proto-name admission, descriptor resolution and build inputs.
  Preserve descriptor freshness, compiler-error rejection and correct warm-cache invalidation when
  a selected schema/registration changes. Missing descriptors and unknown registered names must
  fail explicitly rather than resolve through another system's classpath.
- Start with focused tests and a synthetic non-Temporal package, including wrong/missing
  registration, descriptor/schema edits and the enforced classpath boundary. Register new suites
  in the existing coverage registry. Prove the fixture compiles, lints, constructs and exports,
  not just that the configuration data class accepts its package string.
- At task end format changed Scala, freeze source and run one `make model-check-changed
  BASE=<committed predecessor>` with the mini comparison. Reuse the preceding task's record,
  explicitly recording any necessary Go memory budget. Broad/full Scala validation, independent
  review and held consumer/sweep work retain the owner's source-batch cadence.
## Acceptance
- [ ] R3: layout conventions and API descriptors reach the framework only through registered generic configuration; Temporal supplies its own configuration.
- [ ] R3: proto-name lint, proto build stage and constructed realization conversion use the same registered API roots/descriptors, including freshness and explicit missing/unknown-name errors.
- [ ] R2/R3: the non-Temporal fixture compiles, lints, constructs and exports without Temporal API or source; the boundary check detects a deliberate forbidden dependency.
- [ ] R4: immediate-predecessor mini IR/source-path comparison, mini equivalence and focused registered suites pass; no held real-feature/consumer claim is inferred.

## Done summary
Closed without implementation on 2026-10-24 at the owner's request. The Scala framework this task targets was frozen under `model/archived/` on 2026-10-18 and Umpire continues in Rust (`model/umpire`); the Scala-era backlog is retired rather than carried. Any intent here that still matters is re-filed against the Rust workspace when it is needed.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs: