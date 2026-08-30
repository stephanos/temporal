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
- [ ] Focused and aggregate builds, exact regression, import/trust checks, `make lint-model`, and `make lint-code` pass with no identity or byte drift.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
