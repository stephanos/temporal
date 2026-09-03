---
satisfies: [R5, R6]
---
# fn-51-shorten-ordinary-model-authoring.6 Document and verify ordinary authoring constructors

## Description
Update public convenience guidance and run complete identity/quality gates (R5, R6).

**Size:** S
**Files:** `model/Umpire/ARCHITECTURE.md`
**Touches:** [model/Umpire/ARCHITECTURE.md]

### Approach
- List only exported ordinary conveniences and emphasize their inert, additive relationship to raw records/checkers.
- Audit eligible migrated call sites and preserve every existing comment.
- Run focused suites, aggregate builds, exact regression, trust/import, and lint gates.

### Investigation targets
**Required** (read before coding):
- `model/Umpire/ARCHITECTURE.md:160-175,290-335` — public language/convenience inventory
- `model/Umpire/Core.lean:82-105` — Core Limit/Model Value terminology
- `model/Umpire/Query/Language.lean:24-39` — Query Limit terminology
- `model/Umpire/Space/Language.lean:7-70` — Space leaf terminology
- `model/Umpire/ImplementationLink/Language.lean:13-36` — mapping terminology

## Acceptance
- [ ] Public docs describe each exported constructor as inert shorthand over the existing record/checker.
- [ ] Eligible ordinary boilerplate is migrated or has an existing explicit custom/negative-test reason.
- [ ] All existing comments are preserved.
- [ ] Focused and aggregate builds, exact regression, import/trust checks, and `make lint-model` pass with no identity or byte drift; literal repository lint reproduces only its documented inherited baseline and its no-fix golangci phase scoped from the spec base reports zero findings.

## Done summary
Documented all ten ordinary authoring constructors as inert additive shorthand over existing records and checker authority. The repository-wide audit found no unexplained eligible raw call sites; retained literals are equivalence/representation witnesses, negative or mutation inputs, runtime/compiler/generated values, explicit non-default records, or non-target record types. Full focused, aggregate, regression, trust, and index gates pass; repository lint reproduces only its inherited baseline, with zero golangci findings scoped from the spec base and no task Go paths.
## Evidence
- Commits: f5a79965a2003d2a994cd4732b4648a65f2fdd74
- Tests: cd model && mise exec -- lake build Umpire.CoreTests Umpire.Query.Tests Umpire.Space.Tests.Compilation Umpire.Space.Tests.Determinism Umpire.Space.Tests.Intent Umpire.Space.Tests.Metadata Umpire.Space.Tests.Validation Umpire.ImplementationLink.Tests, cd model && mise exec -- lake build UmpireTests TemporalModelTests TemporalExperimentalTests, make umpire-check-regression, make lint-model, make lint-code (inherited 1,379 golangci findings plus tools/umpire/runtime/errors.go:60:9 errortype finding; no task Go paths), GOLANGCI_LINT_BASE_REV=7d777dbdc8930d36a453bd3ed515e7d0a0ede77d GOLANGCI_LINT_FIX=false make lint-code (golangci phase: 0 findings; wrapper stops only on the inherited errortype finding), constructor facade import/trust probe, make umpire-check-plan-index, git diff --check
- PRs:
