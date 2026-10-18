---
satisfies: [R3, R5, R7, R8]
---
# fn-65-design-and-prototype-approachable.17 Prototype the source-aware checked Property frontend

## Description
Implements R3, R5, R7, R8; use the parent spec and Nexus2 DESIGN.md for the approved semantics and prototype exceptions.

**Size:** M
**Files:** `model/Umpire/Property/Authoring.lean`, `model/Umpire/Property/ImportTests.lean`, `model/Temporal/Feature/Nexus2/AuthoringTests.lean`, `model/Temporal/Feature/Nexus2/README.md`
**Touches:** [model/Umpire/Property/Authoring.lean, model/Umpire/Property/ImportTests.lean, model/Temporal/Feature/Nexus2/AuthoringTests.lean, model/Temporal/Feature/Nexus2/README.md]

### Approach
Compare focused property% forms in ordinary def declarations over the same checked constructors; semantic tests and conflicts are already established dependencies. Reuse source occurrence capture and diagnostic elaboration (Target/Language.lean:924,952) while producing kernel-checked checker-success evidence. Keep syntax outside FiniteMachine; use the narrow recorded AUT-07 prototype exception.

### Investigation targets
**Required** (read before coding):
- `model/Umpire/Target/Language.lean` — source capture, elaboration and native default
- `model/Umpire/Property/Check.lean` — typed failures/checked extraction
- `model/Umpire/Behavior/Language.lean` — checker result
- `model/Umpire/Query/Language.lean` — Query checker/limits
- `model/Temporal/Feature/Nexus2/DESIGN.md` — comparison decisions and evaluation tasks
- `.plans/lean/LEAN_GUIDELINES.md` — trust/editor/checked example requirements

### Quick commands
```bash
(cd model && mise exec -- lake build Temporal.Feature.Nexus2.Tests Umpire.Property.Tests Umpire.Behavior.Tests Umpire.Query.Tests)
make lint-model
make lint-code GOLANGCI_LINT_FIX=false
```

Baseline only existing roots before creation; after implementation include the new roots named below. Run focused commands during iteration, and the parent final gates at prototype completion. Use the Makefile LEAN_LAKE platform wrapper if direct Lake invocation cannot find the macOS SDK. Preserve comments and existing unrelated changes. No commits unless the user requests them.

Post-edit focused gate (mandatory after creating the root): `(cd model && mise exec -- lake build Temporal.Feature.Nexus2.AuthoringTests)`.

Scope this slice to property% and its Target/source capture reuse. Behavior and Query syntax stay constructor-only until the next task; do not add their frontend files here. Required negative semantics/coverage/conflicts are already dependencies.

Export reusable APIs through their existing owning facades and add the corresponding focused import checks when the public surface changes; keep new generic modules in the named owner. New tests must be imported into the named gate root immediately.

## Acceptance
- [ ] Compile and admit property% specimens for baseline/race, same-step cases and guarded bounded responses over the same constructors and fixed IDs; compare checked meaning/fingerprints to constructor fixtures. Syntax supplies no separate evaluator or implicit precedence.
- [ ] Source-diagnostic compilation tests freeze typed error kind, related IDs and exact parent/case/exception/clause spans for malformed/duplicate IDs, missing/wrong-kind references, missing capabilities, unsupported operators/syntax, future/result guards, empty groups and invalid/missing units. No partial checked constant escapes.
- [ ] Synthesize kernel-checked checker-success evidence using ordinary reduction or reusable proved reflection, never the existing native checkedTarget default. Audit exported checked Property/Target dependency inventories and a failing admission/proof path. If impractical, report measured costs and preserve successful-branch constructors rather than silently change trust.
- [ ] Measure Property constructor/frontend compilation and admission for baseline/10x declarations and record available completion/hover/navigation/error-recovery observations, with tool/environment and explicitly unmeasured items. No human usability or final grammar approval claim.
- [ ] Add AuthoringTests as a runnable root immediately and keep a single default example plus comparison specimens. Behavior/Query frontend and cross-language final selection belong to the next task.

## Done summary
Bookkeeping 2026-10-18: parent spec already closed

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs: