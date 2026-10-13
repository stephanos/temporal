---
satisfies: [R2, R16]
---
# fn-141-shrink-the-ir-generator-one-description.2 Retire unused authoring spellings

## Description
Part D: recount the DSL's alternative spellings, retain every spelling used by the current mini/shared/kit/sample sources or a kept sugar's implementation, and retire unused author-facing alternatives with their lifter arms and obsolete fixtures. MILESTONES.md's pull-forward, mini-only and faster-tasks policies supersede the original paths and full-gate cadence. The owner authorizes the conductor to follow its recommendations for routine choices; report the count-based list to the conductor before removal, without a new user-approval gate.

**Size:** M
**Touches:** [model/framework/dsl/**, model/framework/realize/**, model/framework/export/**, model/framework/lint/**, model/framework/check/**, model/temporal/check/**, model/temporal/migrate/README.md, model/README.md, model/SEMANTICS.md]

### Approach
- Re-read the current framework and exporter and recount first. Inventory every multi-spelling construct, separating live mini/shared/kit/sample use, internal core/sugar dependencies, generic tests and exporter fixtures. Real-feature-only use does not constrain retirement and real-feature sources are not edited. Immutable historical sources and payloads remain untouched.
- Re-check the historical candidates individually: hand-bound Bindings, hand-written Typed* realization constructors, Instruction.readUntil versus scoped readUntil, receiver-less capabilities(limits)(...), chained overriding, and explicit Query.total(n). A class used as a kept sugar's return type or core representation is not automatically an unused author-facing spelling; retain the core mechanism and type safety.
- Reserve spellings explicitly needed by remaining approved source specs. Consult fn-141.3-.13 and fn-164/fn-165 and the current milestone before deciding a spelling is unused. A used core form or future requirement stays; record why. Do not preserve obsolete public syntax merely because an exporter fixture uses it.
- Report the full count-based removal/retention list to the conductor before deletion. The conductor's recommendation authority already comes from the user. Delete each retired author-facing spelling, its exporter recognition, obsolete positive/reject fixtures and obsolete guide text together. A surviving DSL test using retired surface syntax is converted to its canonical spelling without reducing its assertion coverage; do not substitute a renamed compatibility alias for retirement.
- Add meaningful compile-negative controls proving retired public spellings are unavailable, alongside positive controls for retained/canonical forms. Retain surviving fixture expected IR and refusal contracts; expected artifacts solely for explicitly retired fixtures may be removed with them. Keep changes minimal and split affected construct tests from Fixtures.test.scala beside their construct as required by the milestone.
- Write a concise migration recipe in model/temporal/migrate/README.md for the real-feature sweep. Record before/after source counts and the exact uses/removal decisions in the task receipt; no bespoke proof scripts, immutable captures or hash manifests.
- Baseline is fn-141.1's green .flow/tmp/checks/20261013-023527/record.md (432 tests). Iterate only affected suites, then run make model-check-changed BASE=<task-base> once at the end, including mini step-table equivalence. Strict model-mini generated-IR comparison, Go, regeneration, broad lint and independent review remain held. Existing generated artifacts are not edited.

## Acceptance
- R2: a use-count and dependency table covers every multi-spelling construct; removal/retention decisions follow the current mini/shared scope, retained core/sugar and upcoming approved requirements, under the owner's recommendation delegation.
- Retired public spellings no longer compile, their lifter arms and obsolete fixtures/docs are gone, and canonical spellings still compile. Internal representations required by kept sugar stay usable without preserving an obsolete author-facing alternative.
- Surviving fixture expected IR/refusal behavior and mini/shared semantics are preserved. The task's model-check-changed record passes with mini step-table equivalence. Generated/real-feature sources are unchanged.
- R16: before/after source counts, retired fixture inventory, migration recipe, focused checks and deferred gates are recorded in the done summary.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
