---
satisfies: [R2, R3, R4, R5]
---
# fn-60-deepen-authored-lean-canonical-json.4 Migrate Space language and compiler canonical construction

## Description
Replace Space language and compiler generic JSON mechanics with the shared typed seam while keeping Space semantic metadata, point compilation, and diagnostic policies separate.

**Size:** M
**Files:** `model/Umpire/Space/Language.lean`, `model/Umpire/Space/Compiler.lean`, `model/Umpire/Space/Tests/Validation.lean`, `model/Umpire/Space/Tests/Compilation.lean`, `model/Umpire/Space/Tests/Determinism.lean`
**Touches:** [model/Umpire/Space/Language.lean, model/Umpire/Space/Compiler.lean, model/Umpire/Space/Tests/Validation.lean, model/Umpire/Space/Tests/Compilation.lean, model/Umpire/Space/Tests/Determinism.lean]

### Approach
- Use typed boolean, natural, string, array, object, and optional/null construction for checked choices, axes, faults, goals, limits, Space metadata, and compiler errors.
- Keep canonical ID/assignment ordering, point-count logic, validation stages, source attribution, and domain-specific projection functions in Space.
- Establish exact success/error baselines before helper deletion, including baseline booleans, independent collections, crossed-type failures, and deterministic reordering.

### Investigation targets
**Required** (read before coding):
- `model/Umpire/Space/Language.lean:346-587` — Space helper family and semantic/metadata projections.
- `model/Umpire/Space/Language.lean:920-995` — final checked metadata and fingerprint construction.
- `model/Umpire/Space/Compiler.lean:62-105` — compiler-specific assignment/error JSON.
- `model/Umpire/Space/Tests/Validation.lean:107-114` — current exact validation JSON coverage.

**Optional** (reference as needed):
- `model/Umpire/Space/Tests/Determinism.lean` — ordering and stable-point identity tests.

### Key context
Space has several distinct semantic identities; do not collapse collections, scalar kinds, or cardinalities just because their JSON construction looks similar.

### Quick commands
```bash
(cd model && mise exec -- lake build Umpire.Space.Tests.Validation Umpire.Space.Tests.Compilation Umpire.Space.Tests.Determinism)
make umpire-check-regression
make lint-model
GOLANGCI_LINT_FIX=false make lint-code
```

## Acceptance
- [ ] Space language and compiler formatters use `CanonicalJson` for generic construction while retaining their separate domain projections, canonical ordering, point compilation, validation, and error ownership.
- [ ] Exact metadata, compiler-error, source, boolean, natural, optional/null, field/element-order, and Behavior Fingerprint outputs match the pre-task baseline.
- [ ] Tests preserve independent semantic collections and cardinalities and include reordered, duplicate, empty, crossed-type, and invalid fixtures without synthetic schema simplification.
- [ ] Public facades/types, existing comments, imports, trust, and asymptotic traversal/sorting behavior remain unchanged.
- [ ] The focused build, regression gate, model lint, and repository lint commands pass or report only a verified inherited baseline.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
