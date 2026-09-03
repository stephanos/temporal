---
satisfies: [R1, R2, R4]
---
# fn-58-partition-the-property-language.2 Extract typed Property checking and narrow Language

## Description
Move typed errors, capability and check contexts, resolved and checked contracts, canonical rendering, fingerprint construction, and the full checker into Check. Leave Language with author-facing data and inert helpers, then complete the downward import chain.

**Size:** M
**Files:** `model/Umpire/Property/Language.lean`, `model/Umpire/Property/Check.lean`, `model/Umpire/Property/Trace.lean`, `model/Umpire/Property.lean`, `model/Umpire/Property/Tests/Validation.lean`, `model/Umpire/Property/Tests/Canonicalization.lean`, `model/Umpire/Examples/SwitchTests.lean`
**Touches:** [model/Umpire/Property/Language.lean, model/Umpire/Property/Check.lean, model/Umpire/Property/Trace.lean, model/Umpire/Property.lean, model/Umpire/Property/Tests/Validation.lean, model/Umpire/Property/Tests/Canonicalization.lean, model/Umpire/Examples/SwitchTests.lean]

### Approach
- Move the Property error vocabulary, capability/check context, resolved and checked types, canonical rendering, Behavior Fingerprint construction, `checkProperty`, and `checkedProperty` into Check without reordering validation.
- Leave Language as author-facing data and inert helpers, including a structurally identical `PropertyPattern.exact`.
- Make Trace import Check and retain the final Language-to-Check-to-Trace-to-Evaluation direction.
- Keep private canonical ordering and validation helpers single-owned; do not export them or introduce another plan/helper module.
- Complete typed characterization for every existing Property error kind, especially wrong-kind, unknown-reference/profile, invalid-clause, unknown/missing-capability, and duplicate-profile cases.
- Preserve exact diagnostic fields and rendering, source fallback, Limit resolution, canonical identity, and explicit-proof checking.

### Investigation targets
**Required** (read before coding):
- `model/Umpire/Property/Language.lean:174-338` — declaration, context, error, and checked types
- `model/Umpire/Property/Language.lean:340-701` — canonicalization and checker pipeline
- `model/Umpire/Property/Tests/Validation.lean:9-79` — typed error and precedence coverage
- `model/Umpire/Property/Tests/Canonicalization.lean:9-77` — identity and fingerprint coverage
- `model/Umpire/Examples/SwitchTests.lean:70-160` — proof-taking authoring consumer
- `model/Umpire/Property/Tests/Fixtures.lean:66-145` — reusable declarations and checked fixtures

**Optional** (reference as needed):
- `model/Temporal/Tool/Inspect.lean:35-44` — checked Property inspection consumer

### Key context
- Raw checking remains the diagnostic authority; `checkedProperty` still requires proof of that same success.
- Diagnostic and canonicalization order are observable contracts. Preserve all existing comments.

## Acceptance
- [ ] R2 is satisfied by the same `checkProperty` and `checkedProperty` interfaces with no partial checked property, bypass, hidden evaluation, or new trust dependency.
- [ ] Every existing error kind has typed characterization coverage and retains exact kind, owner, source path, offending value, related identities, rendering, and deterministic precedence.
- [ ] Opaque or malformed declarations, duplicates, capability/reference/profile failures, named-Limit errors, invalid fields/units, clauses, and missing logical time remain fail-closed.
- [ ] Equivalent definition/provider/meaning/clause reordering preserves identity; documentation/source changes remain fingerprint-neutral; semantic clause/reference/Limit changes still alter it.
- [ ] `PropertyPattern.exact` remains structurally identical, and canonical ordering/fingerprint machinery remains single-owned and private.
- [ ] The final internal import graph is acyclic and downward-only with no Temporal dependency or public consumer migration.
- [ ] `cd model && mise exec -- lake build Umpire.Property.Language Umpire.Property.Check Umpire.Property.Tests Umpire.Examples.SwitchTests` passes.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
