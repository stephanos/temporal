---
satisfies: [R2, R3, R4, R5]
---
# fn-60-deepen-authored-lean-canonical-json.5 Migrate Observation compiler canonical construction

## Description
Move Observation compilation metadata, expression identity, plan, and diagnostic JSON construction to the shared typed seam. Keep checking, disposition, information-flow, and identity policy within Observation.

**Size:** M
**Files:** `model/Umpire/Observation/Compiler.lean`, `model/Umpire/Observation/Tests/Compilation.lean`, `model/Umpire/Observation/Tests/Mutations.lean`, `model/Umpire/Observation/Tests/Check.lean`
**Touches:** [model/Umpire/Observation/Compiler.lean, model/Umpire/Observation/Tests/Compilation.lean, model/Umpire/Observation/Tests/Mutations.lean, model/Umpire/Observation/Tests/Check.lean]

### Approach
- Baseline exact observation-plan, expression-identity, literal, source, disposition, and error bytes before replacing private generic builders.
- Use `CanonicalJson` only for value construction/rendering; retain canonical field-reference analysis, operator/profile/policy selection, validation sequence, and typed error ownership.
- Exercise the real checked compilation path and mutation fixtures rather than constructing synthetic checked records that bypass admission.

### Investigation targets
**Required** (read before coding):
- `model/Umpire/Observation/Compiler.lean:273-523` — duplicated generic builders, expression identity, plan metadata, and errors.
- `model/Umpire/Observation/Tests/Compilation.lean:1-120` — checked compiler path and fixture pattern.
- `model/Umpire/Observation/Tests/Mutations.lean` — established diagnostic and mutation coverage.
- `model/Umpire/Observation/Tests/Check.lean` — downstream checked-plan integration surface.

**Optional** (reference as needed):
- `.flow/memory/bug/integration/behavior-neutral-refactors-must-not-2026-09-04.md` — no-hardening compatibility lesson.

### Key context
Do not add accepted-envelope consistency checks, validation guards, or another nested traversal. Any such hardening is separate semantic work.

### Quick commands
```bash
(cd model && mise exec -- lake build Umpire.Observation.Tests Umpire.Observation.ImportTests Umpire.ImplementationLink.Tests)
make umpire-check-regression
make lint-model
GOLANGCI_LINT_FIX=false make lint-code
```

## Acceptance
- [ ] Observation compiler generic JSON construction is centralized without moving field analysis, validation, disposition, information-flow, or diagnostic precedence into `Umpire.Json`.
- [ ] Exact expression identities, plan metadata, source coordinates, literals including booleans/naturals/null, diagnostic JSON, and identity predicates match the pre-task baseline.
- [ ] Tests use normal checked compilation/admission paths and cover mutation/error precedence without adding validation or accepting/rejecting a new input.
- [ ] Public facades/types, existing comments, import boundaries, trust, and traversal behavior remain unchanged.
- [ ] The focused build, regression gate, model lint, and repository lint commands pass or report only a verified inherited baseline.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
