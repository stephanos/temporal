---
satisfies: [R2, R3, R4, R5]
---
# fn-60-deepen-authored-lean-canonical-json.2 Migrate Target and Behavior canonical construction

## Description
Replace the duplicated generic JSON mechanics in Target and Behavior with the Task 1 interface. Keep both domains' semantic projection, canonical ordering, checks, and public formatters in their current owners.

**Size:** M
**Files:** `model/Umpire/Target/Language.lean`, `model/Umpire/Target/Tests/Canonicalization.lean`, `model/Umpire/Target/Tests/Compatibility/CanonicalMetadata.lean`, `model/Umpire/Behavior/Language.lean`, `model/Umpire/Behavior/Tests/Canonicalization.lean`
**Touches:** [model/Umpire/Target/Language.lean, model/Umpire/Target/Tests/Canonicalization.lean, model/Umpire/Target/Tests/Compatibility/CanonicalMetadata.lean, model/Umpire/Behavior/Language.lean, model/Umpire/Behavior/Tests/Canonicalization.lean]

### Approach
- Replace private quoting, array joining, source encoding, and object punctuation with typed `CanonicalJson` construction.
- Retain Target and Behavior field selection/order, canonical-set and canonical-list logic, semantic-vs-metadata distinctions, checker order, and existing string-returning public declarations.
- Pin current successful and typed-error bytes before relocation, then keep Behavior Fingerprints unchanged for canonical and reordered inputs.
- Move existing comments with their declarations; improve private names only where the old helper disappears or projection ownership becomes clearer.

### Investigation targets
**Required** (read before coding):
- `model/Umpire/Target/Language.lean:242-490` — duplicated helpers and Target semantic/metadata projections.
- `model/Umpire/Behavior/Language.lean:658-901` — Behavior projections, public canonical accessors, and fingerprint construction.
- `model/Umpire/Target/Tests/Canonicalization.lean:17-205` — exact Target ordering and canonicalization tests.
- `model/Umpire/Behavior/Tests/Canonicalization.lean:19-76` — reordered Behavior compatibility pattern.

**Optional** (reference as needed):
- `model/Umpire/Tests/MigrationCompatibility.lean:147-190` — cross-layout Target/Query byte and fingerprint compatibility.

### Key context
Any stricter validation or new traversal is semantic work outside this task. A canonical-string difference is a compatibility failure even when the parsed JSON values compare equal.

### Quick commands
```bash
(cd model && mise exec -- lake build Umpire.TargetTests Umpire.Behavior.Tests Umpire.Tests.MigrationCompatibility)
make umpire-check-regression
make lint-model
GOLANGCI_LINT_FIX=false make lint-code
```

## Acceptance
- [ ] Target and Behavior no longer own duplicate generic quoting, array-joining, optional/null, source-object, or object-punctuation helpers covered by `CanonicalJson`.
- [ ] Public Target/Behavior declarations, facade imports, checked record shapes, validation stages, error precedence, and comments remain unchanged.
- [ ] Exact canonical metadata, error JSON, source coordinates, field/element order, and Behavior Fingerprints match the pre-task baseline for normal, reordered, empty, nested, duplicate, and invalid fixtures.
- [ ] No `Umpire.Property` or generated source is modified and no new import/trust dependency is introduced.
- [ ] The focused build, regression gate, model lint, and repository lint commands pass or report only a verified inherited baseline.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
