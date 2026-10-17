---
satisfies: [R18]
---
# fn-190-models-are-plain-scala-state-as-vars.13 Validate every native effect result and lint inferred multiline definitions

## Description
Owner follow-up 2026-10-17. Add a compiler check for every method in native Machine, Facet and native Derived effects sections, including private, parameterized and unused methods: the inferred or declared result must be exactly Unit. Add separate author lint forbidding explicit return annotations and requiring multiline bodies, allowing typed parameters. Keep unrelated helpers, objects merely named effects and legacy RecordMachine compatibility fixtures unrestricted. Apply inferred multiline style to all current sample effects. Use existing compiler and typed lint seams with positioned errors; focused positive/refusal fixtures prove both checks, including unused defs, aliases, Nothing, typed params, facets and derivations. No schema, Go, Temporal or broader exporter release.

## Acceptance
Every authored native effect def is checked even unused; inferred Unit compiles, non-Unit and Nothing fail at the definition. Style lint rejects explicit return annotations and same-line bodies; multiline inferred bodies with typed params pass. Native sample effects follow the rules and all relevant checks pass.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
