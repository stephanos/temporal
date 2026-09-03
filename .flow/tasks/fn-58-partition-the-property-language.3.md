---
satisfies: [R1, R4]
---
# fn-58-partition-the-property-language.3 Lock the Property facade, documentation, and compatibility

## Description
Expand facade checks across Language, Check, Trace, and Evaluation; document the internal ownership while retaining public author guidance; audit moved comments and theorem trust; and run aggregate compatibility gates.

**Size:** S
**Files:** `model/Umpire/Property/ImportTests.lean`, `model/Umpire/ARCHITECTURE.md`, `model/ARCHITECTURE.md`, `Makefile`
**Touches:** [model/Umpire/Property/ImportTests.lean, model/Umpire/ARCHITECTURE.md, model/ARCHITECTURE.md]

### Approach
- Expand the facade regression to check representative Language, Check, Trace, and Evaluation declarations while retaining the Behavior and Query negative guards.
- Preserve the package gate and its exact direct Language import requirement.
- Document the four internal modules as implementation modules normally reached through `Umpire.Property`.
- Preserve the existing Property lifecycle, Limit, and raw-checker guidance; refresh documentation anchors after concurrent Observation documentation work.
- Audit moved comments, declaration names, warnings, imports, theorem statements, and axiom inventories.
- Run focused, aggregate, artifact regression, model lint, and repository lint gates.

### Investigation targets
**Required** (read before coding):
- `model/Umpire/Property.lean:1` — stable facade imports
- `model/Umpire/Property/ImportTests.lean:1-13` — facade visibility and negative guards
- `Makefile:1185-1203` — package facade and import enforcement
- `model/Umpire/ARCHITECTURE.md:34-42` — public versus internal imports
- `model/Umpire/ARCHITECTURE.md:163-192` — Property lifecycle and checking guidance
- `model/ARCHITECTURE.md:431-434` — facade implementation navigation

**Optional** (reference as needed):
- `model/UmpireTests.lean:1-24` — aggregate reusable test imports

### Key context
- Do not rewrite unchanged public semantics or edit the Makefile unless the exact existing gate cannot remain intact.
- This documentation task may follow Observation documentation edits without coupling the Property implementation to those specs.

## Acceptance
- [ ] R1 and R4 are satisfied by a stable `Umpire.Property` facade that exposes representative Language, Check, Trace, and Evaluation declarations with unchanged types.
- [ ] Behavior and Query authoring declarations remain absent from the narrow facade, and the package gate retains the exact direct Language import contract.
- [ ] Architecture docs describe internal ownership and continue directing ordinary authors to `Umpire.Property` without rewriting unchanged semantics.
- [ ] No consumer source change, generated output, artifact byte/checksum, warning, import-boundary, comment, theorem, or trust drift is introduced.
- [ ] `cd model && mise exec -- lake build Umpire.Property.ImportTests UmpireTests Temporal TemporalModelTests TemporalExperimentalTests` passes.
- [ ] `make umpire-build-model`, `make umpire-check-regression`, `make lint-model`, and `make lint-code` pass.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
