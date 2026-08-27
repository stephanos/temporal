---
satisfies: [R1, R2, R3, R4]
---
# fn-31-deepen-umpire-target-and-simplify.2 Deepen Target checking and approachable diagnostics

## Description
### Review reconciliation (normative)

This task owns only the Target-side declaration/checker facade and the elaboration-only diagnostic adapter. Preserve `composeTarget` as the pure low-level semantic checker, the existing `DeclarationErrorKind` set, and the current validation order: validate declaration identities and declaration duplicates; require the target declaration kind; reject duplicate provider and connector identities; validate providers, including declaration references, laws, and meanings; validate connectors, including declaration references, laws, and reconciliation membership; validate required capability references and provider coverage; reject provider conflicts and connector ambiguity; then validate kernel availability and kernel identity. Bounds, query-level finite completeness, and planner-kernel ordering remain in Query/Planning and are assigned to Task `.7`.

Keep stable serialized `SemanticSource` separate from an authored-occurrence table captured by a narrow compiler-facing wrapper. Each syntax occurrence receives a nonsemantic source-span/ordinal token, its declaration identity, and a closed occurrence role/path distinguishing at least declaration metadata, target declaration, provider definition/reference, connector definition/reference, capability requirement, law requirement/witness, meaning, reconciliation, and kernel occurrences. The single validation pass retains the role/path of its failure; the existing `composeTarget : Except DeclarationError CheckedTarget` erases only that diagnostic context for compatibility, while the ordinary adapter uses it to select the matching role before source sorting. Thus one identity reused across metadata, definitions, and references cannot point at the wrong span, and duplicate IDs still report an unambiguous original/offending pair independently of input-list order. The programmatic adapter returns `Except AuthoringDiagnostic CheckedTarget`; the compiler-facing wrapper emits the same diagnostic at the captured occurrence. Neither path may create a second semantic checker. `AuthoringDiagnostic` may expose file/line/column, but occurrence data must never enter checked values, semantic identities/digests, canonical metadata, or artifacts.

Introduce finite planning data additively rather than adding mandatory fields to the existing checked-kernel constructor. A Target without it remains a complete checked semantic value and represents planning availability explicitly; Task `.7` maps absence to Query's existing `missingFiniteCompleteness`. An opted-in Target carries the explicit action list, focused `actionSound`/`actionComplete` proofs, and stable role/action-domain compatibility tokens used by current canonical Query artifacts. The tokens are supplied once by semantic-model maintainers and copied verbatim downstream, not inferred or rebuilt. This transitional representation must allow Task `.2` to compile before Switch, BasicLifecycle, CallerClosure, Query/Planning fixtures, and other callers migrate in Tasks `.3`, `.4`, and `.7`.

Implement the cohesive checked authoring boundary and diagnostics for R1–R4 without changing target meaning.

**Size:** M
**Files:** `model/Umpire/Target.lean`, `model/Umpire/Target/**`, `model/Umpire/Core.lean`
**Touches:** [model/Umpire/Target.lean, model/Umpire/Target/**, model/Umpire/Core.lean]

### Approach
- Move target-owned checking/canonicalization behind the public facade.
- Refactor one detailed validation result beneath `composeTarget`; preserve `composeTarget` as the focused `Except DeclarationError` expert seam and use the same result for the ordinary adapter.
- Add only the minimal source-capture/elaboration surface required for precise Lean diagnostics; do not introduce a general feature grammar.
- Add an optional checked finite-planning capability with a source-compatible unavailable case; do not require unrelated callers to migrate atomically in this task.
- Reuse the deterministic error ordering in `model/Umpire/Target/Language.lean:222-389`.

### Investigation targets
**Required** (read before coding):
- `model/Umpire/Target.lean` — current facade
- `model/Umpire/Target/Language.lean:222-401` — validation, composition, and canonical projections
- `model/Umpire/Target/Tests/Validation.lean` — negative-case pattern
- `model/Umpire/Behavior/Language.lean` — adjacent deep checked-language pattern

### Acceptance
- [ ] Ordinary programmatic callers receive a checked target or one precise deterministic `AuthoringDiagnostic`; the compiler-facing wrapper emits that same typed failure at the captured span.
- [ ] Reusing one identity in declaration metadata, a provider/connector definition, and one or more references still selects the occurrence role associated with the actual validation stage before source sorting.
- [ ] Provider/connector selection and all semantic choices remain explicit.
- [ ] No partial or unchecked target enters downstream APIs.
- [ ] Reordered declarations and duplicate occurrences select the same canonical original/offending pair; moving an authored occurrence changes only diagnostic location.
- [ ] Existing target constructors compile through the explicit planning-unavailable case until migrated; opted-in finite-planning data carries action proofs and the exact existing stable domain tokens without changing Target validation errors.
## Acceptance
- [ ] R1–R4 public contracts and negative cases are covered through the facade.
- [ ] Existing lower-level semantics remain available only through the focused expert seam and are not reimplemented by the authoring adapter.
- [ ] Focused Target tests pass.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
