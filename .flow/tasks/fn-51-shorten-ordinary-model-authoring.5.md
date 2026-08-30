---
satisfies: [R4, R5, R6]
---
# fn-51-shorten-ordinary-model-authoring.5 Add and migrate forward mapping constructors

## Description
Add explicit source-to-destination constructors for value and semantic Link mappings and migrate ordinary uses (R4-R6). This task is independent of Core/Query/Space migration after the parent fn-43 surface lands.

**Size:** M
**Files:** `model/Umpire/ImplementationLink/Language.lean`, `model/Umpire/ImplementationLink/Tests/Fixtures.lean`, `model/Umpire/ImplementationLink/Tests/Application.lean`, `model/Umpire/ImplementationLink/Tests/Compilation.lean`, `model/Umpire/ImplementationLink/ImportTests.lean`, `model/Temporal/System/Nexus/ImplementationLink.lean`, `model/Temporal/System/Nexus/ImplementationLinkTests.lean`
**Touches:** [model/Umpire/ImplementationLink/Language.lean, model/Umpire/ImplementationLink/Tests/Fixtures.lean, model/Umpire/ImplementationLink/Tests/Application.lean, model/Umpire/ImplementationLink/Tests/Compilation.lean, model/Umpire/ImplementationLink/ImportTests.lean, model/Temporal/System/Nexus/ImplementationLink.lean, model/Temporal/System/Nexus/ImplementationLinkTests.lean]

### Approach
- Define `.forward` beside both existing two-field mapping records with no lookup/checking.
- Inventory mapping-pair literals across `model/Umpire` and `model/Temporal`; migrate setup/state/action/outcome/observation and semantic pairs in ordinary Nexus links and positive fixtures, including the Application identity-link fixture.
- Preserve mapping list order, known gaps, coverage, forward witness, fingerprints, and all negative raw mutations.

### Investigation targets
**Required** (read before coding):
- `model/Umpire/ImplementationLink/Language.lean:13-36` — value and semantic mapping records
- `model/Umpire/ImplementationLink/Tests/Fixtures.lean:164-180` — reusable mapping literals
- `model/Umpire/ImplementationLink/Tests/Application.lean:193-230` — positive identity-link semantic and value mappings
- `model/Temporal/System/Nexus/ImplementationLink.lean:138-175` — ordinary lifecycle mapping tables
- `model/Temporal/System/Nexus/ImplementationLink.lean:496-525` — CallerClosure semantic/value mappings
- `model/Umpire/ImplementationLink/Tests/Compilation.lean` — Link diagnostic matrix and deliberate mutations
- `model/Umpire/ImplementationLink/ImportTests.lean` — facade visibility
## Acceptance
- [ ] Both forward constructors are documented, explicit, and structurally equal to their record literals.
- [ ] A repository-wide inventory covers ordinary production and positive fixture mapping pairs, including the Application identity link; all use the constructors without changing list order.
- [ ] Deliberate mutation, diagnostic, and known-gap witness literals retain their raw shapes where that shape is under test.
- [ ] Link checking, known gaps, coverage, witnesses, diagnostics, fingerprints, and composed results are unchanged.
- [ ] Link import, compilation, Application, fixture, System, and composed tests pass.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
