---
satisfies: [R1, R5]
---
# fn-4-umpire-observation-and-semantic-verdicts.1 Define and compile the typed Observation DSL

## Description
Create the reusable Observation declaration and checked-plan boundary for R1/R5. Evidence processing and Temporal profiles remain in later tasks.

**Size:** M
**Files:** `model/Umpire/Observation.lean`, `model/Umpire/Observation/Language.lean`, `model/Umpire/Observation/Tests/Fixtures.lean`, `model/Umpire/Observation/Tests/Compilation.lean`, `model/Umpire.lean`
**Touches:** [model/Umpire/Observation.lean, model/Umpire/Observation/Language.lean, model/Umpire/Observation/Tests/Fixtures.lean, model/Umpire/Observation/Tests/Compilation.lean, model/Umpire.lean]

### Approach

- Define profile, evidence kind/field, mapping, binding, ordering, closure, disposition, and positive `evidence-records` bound declarations.
- Define the closed typed expression grammar: literals, field/binding references, fixed named/versioned total normalizers, presence/equality/Boolean predicates, contribution markers, and named/versioned digest tokens. Exclude callbacks, interpolation, recursion, and user code.
- Assign static information-flow labels and reject every clear-value path from redacted/hashed/rejected inputs; include canonical typed expressions and the evidence bound in checked-plan identity.
- Compile against checked target declarations and meanings into one canonical inert plan.
- Follow existing authored-to-checked `Except`, source diagnostic, canonical ordering, and semantic identity patterns.
- Keep the package Temporal-independent and callback-free.

### Investigation targets
**Required** (read before coding):
- `model/Umpire/Core.lean:8-105` — identities, kinds, values, and pure traces.
- `model/Umpire/Core.lean:430-464` — deterministic identity/kind validation.
- `model/Umpire/Property/Language.lean:192-270` — authored/checked/error boundary.
- `model/Umpire/Property/Language.lean:640-670` — canonical checked compilation.
- `model/Temporal/Feature/Nexus/Examples/BasicLifecycle.lean:211-245` — later profile vocabulary.

## Acceptance
- [ ] Equivalent reordered declarations compile to the same checked-plan identity.
- [ ] Every R1 structural conflict has an exact typed-error fixture.
- [ ] Every consumed field has exactly one checked disposition.
- [ ] Operator/version/type failures and every forbidden disposition-to-output flow have exact compile-error fixtures.
- [ ] Zero/wrong-unit evidence bounds fail, and reordered equivalent expressions plus the same bound preserve checked identity.
- [ ] The package has no Temporal import or evidence callback.
- [ ] `cd model && mise exec -- lake build Umpire.Observation.Tests.Compilation` passes.

## Done summary
Implemented the Temporal-independent typed Observation mapping DSL, canonical checked plans with resolved typed expressions and connector-reconciled meanings, positive evidence-record bounds, dispositions, and deterministic R1/R5 compile errors. The task and aggregate Lean targets plus the repository regression gate pass; the final Make gate used an isolated `GOCACHE` because the inherited global cache symlink was broken.

baseline: red (new and later-task Observation targets absent pre-edit); `make umpire-check-regression` green pre-edit

stage: impl-review - ran [2026-08-26T17:14:37Z..2026-08-26T17:24:10Z]

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 50a5f3c7183ea40495b34c2af4d4990731fbef5e, dbbcd249c8e86e5ac4b690429bf3d98c5a0a3bfd
- Tests: cd model && mise exec -- lake build Umpire.Observation.Tests.Compilation, cd model && mise exec -- lake build Umpire.Observation.Tests, cd model && mise exec -- lake build Umpire.Property.Tests.Validation, cd model && mise exec -- lake build Umpire, cd model && mise exec -- lake build UmpireTests, make umpire-check-regression, baseline: red (cd model && mise exec -- lake build Umpire.Observation.Tests.Compilation failed pre-edit: task target absent), baseline: red (cd model && mise exec -- lake build Umpire.Observation.Tests failed pre-edit: task aggregate absent), baseline: red (cd model && mise exec -- lake build Temporal.Feature.Nexus.ObservationTests failed pre-edit: later-task target absent), baseline: green (make umpire-check-regression)
- PRs:
