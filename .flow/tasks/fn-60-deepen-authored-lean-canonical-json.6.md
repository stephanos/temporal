---
satisfies: [R2, R3, R4, R5]
---
# fn-60-deepen-authored-lean-canonical-json.6 Migrate Implementation Link canonical construction

## Description
Move Implementation Link declaration, error, application-diagnostic, and observed-trace JSON construction to the shared typed seam while preserving the checked forward-correspondence interface.

**Size:** M
**Files:** `model/Umpire/ImplementationLink/Language.lean`, `model/Umpire/ImplementationLink/Application.lean`, `model/Umpire/ImplementationLink/Tests/Compilation.lean`, `model/Umpire/ImplementationLink/Tests/Application.lean`, `model/Umpire/ImplementationLink/ImportTests.lean`
**Touches:** [model/Umpire/ImplementationLink/Language.lean, model/Umpire/ImplementationLink/Application.lean, model/Umpire/ImplementationLink/Tests/Compilation.lean, model/Umpire/ImplementationLink/Tests/Application.lean, model/Umpire/ImplementationLink/ImportTests.lean]

### Approach
- Baseline the early implementation-law and capability fingerprint preimages, declaration/error fingerprints, diagnostic identities, optional counts, limits, coordinates, and observed-translation bytes before replacing direct or private quote/optional/object builders.
- Keep mapping canonicalization, Known Gaps, source/destination semantic ownership, validation, and application decisions inside Implementation Link.
- Use existing compilation/application fixtures and import tests to prove no Feature/System or public-facade change.
- Finish with a residual source scan across both production files so no scoped direct `Lean.Json.compress` or equivalent generic builder remains silently outside the shared seam.

### Investigation targets
**Required** (read before coding):
- `model/Umpire/ImplementationLink/Language.lean:102-131` — early semantic and capability fingerprint preimages using direct string JSON.
- `model/Umpire/ImplementationLink/Language.lean:715-904` — canonical declaration projection and private builders.
- `model/Umpire/ImplementationLink/Language.lean:1183-1195` — public error and identity surface.
- `model/Umpire/ImplementationLink/Application.lean:151-342` — diagnostic and observed-translation identity construction.
- `model/Umpire/ImplementationLink/Tests/Compilation.lean` — checked declaration compatibility tests.
- `model/Umpire/ImplementationLink/Tests/Application.lean` — application diagnostic/translation tests.

**Optional** (reference as needed):
- `model/Umpire/ImplementationLink/ImportTests.lean` — stable import visibility checks.

### Key context
The single allowed production Feature/System connection remains outside this reusable module. Exact model/compiler bindings, checksums, Known Gaps, and source/destination identities cannot be weakened or crossed.

### Quick commands
```bash
(cd model && mise exec -- lake build Umpire.ImplementationLink.Tests Umpire.Observation.Tests.Check UmpireTests TemporalModelTests)
make umpire-check-regression
make lint-model
GOLANGCI_LINT_FIX=false make lint-code
```

## Acceptance
- [ ] Implementation Link language/application formatters use `CanonicalJson` for generic construction while retaining mapping, Known Gap, validation, and application semantics in their existing owners.
- [ ] Exact implementation-law/capability fingerprint preimages, declaration/error fingerprints, source/destination references, optional/null counts, limits, coordinates, diagnostics, observed translations, and identity predicates match the pre-task baseline.
- [ ] A residual source scan covers both production files and finds no scoped direct string escaping or generic JSON helper family left outside `CanonicalJson`.
- [ ] Import tests prove the public facade and Feature/System isolation remain unchanged; exact model/compiler bindings, checksums, and trust assumptions are preserved.
- [ ] Existing comments remain intact and no parse/re-render, new validation, sort, traversal, cache, or generated/protocol change is introduced.
- [ ] The focused build, regression gate, model lint, and repository lint commands pass or report only a verified inherited baseline.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
