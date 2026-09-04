---
satisfies: [R2, R3, R4, R5]
---
# fn-60-deepen-authored-lean-canonical-json.3 Migrate Query and Exploration canonical diagnostics

## Description
Move Query's two local JSON helper families and Exploration's diagnostic formatter onto the shared typed construction seam. Preserve the Query/Exploration facades and their domain-owned canonicalization.

**Size:** M
**Files:** `model/Umpire/Query/Language.lean`, `model/Umpire/Query/Tests/Identity.lean`, `model/Umpire/Query/Tests/Validation.lean`, `model/Umpire/Exploration/Language.lean`, `model/Umpire/Exploration/Tests/Validation.lean`
**Touches:** [model/Umpire/Query/Language.lean, model/Umpire/Query/Tests/Identity.lean, model/Umpire/Query/Tests/Validation.lean, model/Umpire/Exploration/Language.lean, model/Umpire/Exploration/Tests/Validation.lean]

### Approach
- Consolidate finite-domain and checked-query generic JSON construction through `CanonicalJson` without merging their domain projection responsibilities.
- Retain Query role/action canonicalization, exact optional completeness representation, limits/policy semantics, fingerprint inputs, and error precedence.
- Render Exploration errors through the same typed seam while keeping request validation and canonical ID ordering local.
- Add exact-byte fixtures for both successful Query identity and invalid Query/Exploration diagnostics before deleting duplicate private helpers.

### Investigation targets
**Required** (read before coding):
- `model/Umpire/Query/Language.lean:171-205` — finite-domain helper family and fingerprint inputs.
- `model/Umpire/Query/Language.lean:323-578` — checked Query helpers, metadata, errors, and public accessors.
- `model/Umpire/Query/Tests/Identity.lean:1-80` — current Query identity coverage.
- `model/Umpire/Exploration/Language.lean:25-60` — Exploration diagnostic helper and public formatter.

**Optional** (reference as needed):
- `model/Umpire/Tests/MigrationCompatibility.lean:184-190` — stable Query bytes across model layouts.

### Key context
Preserve array duplicates/order unless existing domain code canonicalizes them first. Optional values continue to render exactly as today, including literal `null`.

### Quick commands
```bash
(cd model && mise exec -- lake build Umpire.Query.Tests Umpire.Exploration.Tests.Validation Umpire.Tests.MigrationCompatibility)
make umpire-check-regression
make lint-model
GOLANGCI_LINT_FIX=false make lint-code
```

## Acceptance
- [ ] Query has one shared generic construction path for both finite-domain and checked-query JSON, and Exploration uses it for diagnostic JSON, without moving domain sorting/validation into `Umpire.Json`.
- [ ] Public declarations, facade imports, checked values, limits/policy/completeness semantics, validation order, and existing comments remain unchanged.
- [ ] Exact success, identity, optional/null, escaping, ordering, duplicate, and typed-error bytes and Behavior Fingerprints match the pre-task baseline.
- [ ] No `Umpire.Property`, generated source, parsing/re-rendering, new sort/traversal, or new import/trust dependency is introduced.
- [ ] The focused build, regression gate, model lint, and repository lint commands pass or report only a verified inherited baseline.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
