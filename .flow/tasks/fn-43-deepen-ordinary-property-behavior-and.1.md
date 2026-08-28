---
satisfies: [R2, R3]
---
# fn-43-deepen-ordinary-property-behavior-and.1 Add Core semantic and Definition identity primitives

## Description
Build the Core-owned primitives for R2 and R3 before language modules consume them. Keep each API in the namespace of its principal type and expose semantic equations instead of representation details.

**Size:** M
**Files:** `model/Umpire/Core.lean`, `model/Umpire/CoreTests.lean`, `model/UmpireTests.lean`
**Touches:** [model/Umpire/Core.lean, model/Umpire/CoreTests.lean, model/UmpireTests.lean]

### Approach
- Add documented `DefinitionId` canonical ordering/set, deterministic duplicate discovery/validation primitives, and `SourceLocation.displayPath` at the shared ownership seam established by fn-38.
- Add `ModelTraceStep.result` and `TransitionResult.map` as narrow constructors/combinators over the existing structures; provide semantic equations callers can use without unfolding.
- Keep validation results structural: language-specific error construction remains outside Core.
- Add a focused Core test module and import it through the existing `UmpireTests` aggregate.
- Preserve the existing `ModelTrace` no-runtime-data comment and all unaffected comments.

### Investigation targets
**Required** (read before coding):
- `model/Umpire/Core.lean:9-27` — Definition ID ownership and naming.
- `model/Umpire/Core.lean:66-80` — Source location representation and current fallback inputs.
- `model/Umpire/Core.lean:102-124` — trace step, trace, and transition result structures to deepen.
- `model/Umpire/Property/Language.lean:309-380` — representative local ID/source/duplicate behavior to preserve.
- `model/Umpire/Behavior/Language.lean:192-301` — second independent implementation and deterministic error behavior.

**Optional** (reference as needed):
- `model/Umpire/Target/Language.lean:348-558` — established canonical ID and source-path semantics.

### Key context
- fn-38 owns the constructor/fixture consolidation seam; extend its final API rather than creating a competing helper namespace.
- Public reusable declarations must not acquire a new `native_decide` or other compiler-trust dependency.

### Quick commands
```bash
cd model && mise exec -- lake build UmpireTests
```

## Acceptance
- [ ] Shared Definition ID/source-path primitives have public docstrings, deterministic equations, and focused coverage for sorted/deduplicated output, duplicate witness choice, malformed/blank IDs, ties, and missing source paths.
- [ ] `ModelTraceStep.result` and `TransitionResult.map` return the existing types and preserve outcome, resulting-state, and observation meaning under focused equational tests.
- [ ] Core exposes no Property-, Behavior-, Query-, or Observation-specific error type and introduces no parallel model representation.
- [ ] Existing comments are preserved and `cd model && mise exec -- lake build UmpireTests` passes.
- [ ] Axiom/trust inspection shows no new unapproved dependency in changed public declarations.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
