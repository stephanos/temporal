---
satisfies: [R1, R4, R5, R8]
---
# fn-62-make-ordinary-temporal-model-authoring.3 Add explicit identity source Limit and transition primitives

## Description
Implement R1, R4, and R5 with narrow family-scoped identity/source helpers, named Query Limit construction, and common transition-result constructors. Preserve explicit stable values and checker authority while removing copy/paste and positional ambiguity.

**Size:** M
**Files:** `model/Temporal/Shared.lean`, `model/Umpire/Property/Language.lean`, `model/Umpire/Property/Tests/Validation.lean`, `model/Umpire/Query/Language.lean`, `model/Umpire/Query/Tests/Validation.lean`
**Touches:** [model/Temporal/Shared.lean, model/Umpire/Property/Language.lean, model/Umpire/Property/Tests/Validation.lean, model/Umpire/Query/Language.lean, model/Umpire/Query/Tests/Validation.lean]

### Approach
- Extend the thin `Temporal.Shared` layer at `model/Temporal/Shared.lean:7-21` only for identity/source helpers with multiple Temporal model consumers.
- Keep dot-separated ID suffixes and definition kinds author-visible; centralize family prefixes and source capture without deriving identity from declaration order or instances.
- Extend existing semantic patterns such as `PropertyPattern.exact` and `PropertyClause.transitionContract` in `model/Umpire/Property/Language.lean:43-106`; return existing clauses and keep Action/state/outcome/observation choices explicit.
- Add a named constructor over the existing typed fields in `model/Umpire/Query/Language.lean:30-86`; preserve `QueryLimits` representation and canonical serialization.
- Retain the raw `DefinitionId`, `SourceLocation`, Property-clause, and `QueryLimits` construction paths for advanced and invalid tests.

### Investigation targets
**Required** (read before coding):
- `model/Temporal/Shared.lean:7-21` — existing narrow Temporal helper layer.
- `model/Umpire/Core.lean:9-56` — Definition ID syntax and canonical identity.
- `model/Umpire/Property/Language.lean:43-106` — existing transition patterns and clauses.
- `model/Umpire/Query/Language.lean:30-86` — typed per-stage Limits and positional helper.
- `model/Umpire/Query/Tests/Validation.lean` — Limit and identity diagnostic patterns.

**Optional** (reference as needed):
- `model/Umpire/Query/Tests/Identity.lean` — stable query identity regressions.
- `.flow/specs/fn-58-partition-the-property-language.md:17-34` — frozen Property facade and ownership boundary.

### Acceptance
- [ ] Family prefixes, definition kinds, stable suffixes, transition patterns, and source locations remain explicit at helper call sites.
- [ ] Invalid syntax/kind/duplicates/crossed references and invalid transition clauses fail through owning checkers at the closest declaration location.
- [ ] Query authoring names each independent stage Limit and unit; zero/invalid values retain exact typed diagnostics.
- [ ] Helper-produced clauses and Limits retain exact canonical metadata and Behavior Fingerprints for equivalent values.
- [ ] No global registry, ambient instance, order-based identity, inferred outcome, or hidden default is added.

## Acceptance
- [ ] R1, R4, R5, and R8 are satisfied for reusable identity, source, transition, and Limit primitives.
- [ ] `cd model && mise exec -- lake build Umpire.CoreTests Umpire.Property.Tests Umpire.Query.Tests` passes.
- [ ] Equivalent helper-produced declarations retain exact canonical identity and no hidden compiler-trust path is introduced.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
