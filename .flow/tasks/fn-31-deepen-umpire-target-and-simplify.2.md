---
satisfies: [R1, R2, R3, R4]
---
# fn-31-deepen-umpire-target-and-simplify.2 Deepen Target checking and approachable diagnostics

## Description
### Review reconciliation (normative)

This task owns only the Target-side declaration/checker facade and the elaboration-only diagnostic adapter. Preserve `composeTarget` as the pure low-level semantic checker, the existing `DeclarationErrorKind` set, and the current validation order: validate declaration identities and declaration duplicates; require the target declaration kind; reject duplicate provider and connector identities; validate providers, including declaration references, laws, and meanings; validate connectors, including declaration references, laws, and reconciliation membership; validate required capability references and provider coverage; reject provider conflicts and connector ambiguity; then validate kernel availability and kernel identity. Bounds, query-level finite completeness, and planner-kernel ordering remain in Query/Planning and are assigned to Task `.7`.

Keep stable serialized `SemanticSource` separate from an authored-occurrence table captured by a narrow compiler-facing wrapper. Each syntax occurrence receives a nonsemantic source-span/ordinal token, its declaration identity, and a closed occurrence role/path distinguishing at least declaration metadata, target declaration, provider definition/reference, connector definition/reference, capability requirement, law requirement/witness, meaning, reconciliation, and kernel occurrences. The single validation pass retains the role/path of its failure; the existing `composeTarget : Except DeclarationError CheckedTarget` erases only that diagnostic context for compatibility, while the ordinary adapter uses it to select the matching role before source sorting. Thus one identity reused across metadata, definitions, and references cannot point at the wrong span, and duplicate IDs still report an unambiguous original/offending pair independently of input-list order. The programmatic adapter returns `Except AuthoringDiagnostic CheckedTarget`; the compiler-facing wrapper emits the same diagnostic at the captured occurrence. Neither path may create a second semantic checker. `AuthoringDiagnostic` may expose file/line/column, but occurrence data must never enter checked values, semantic identities/digests, canonical metadata, or artifacts.

Introduce finite planning data additively rather than adding mandatory fields to the existing checked-kernel constructor. A Target without it remains a complete checked semantic value and represents planning availability explicitly; Task `.7` maps absence to Query's existing `missingFiniteCompleteness`. An opted-in Target carries the explicit action list, focused `actionSound`/`actionComplete` proofs, and stable role/action-domain compatibility tokens used by current canonical Query artifacts. The tokens are supplied once by semantic-model maintainers and copied verbatim downstream, not inferred or rebuilt. This transitional representation must allow Task `.2` to compile before Switch, Nexus Lifecycle, Experimental CallerClosure, Query/Planning fixtures, and other callers migrate in Tasks `.3`, `.4`, and `.7`.

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
Implemented a single Target-owned detailed checker beneath the compatibility-preserving `composeTarget` seam, precise authored-occurrence diagnostics shared by programmatic and compiler-facing adapters, and an additive finite-planning capability whose proofs remain indexed by the validated kernel relation. Existing Target semantics and error order remain unchanged; focused and full verification are green, and Codex review reached SHIP with no surviving findings.

The task declared only `model/Umpire/Target.lean`, `model/Umpire/Target/**`, and `model/Umpire/Core.lean`. The proof-indexed optional field additionally required minimal transitional `planning := .unavailable` compatibility at the four exhaustive live checked-kernel re-ascriptions in `model/Umpire/Query/Tests/Identity.lean`, `model/Umpire/Examples/Switch.lean`, `model/Temporal/Feature/Nexus/Lifecycle.lean`, and `model/Temporal/Feature/Nexus/Experimental/CallerClosure.lean`; these preserve existing comments and semantics and perform none of the `.3`, `.4`, or `.7` migrations. Green gate receipts were not warrantable because the unrelated user-owned `.plans/UMPIRE4_ORDER.md` edit remained intentionally dirty and untouched.

baseline: green
stage: impl-review - ran [2026-08-27T03:21:12Z..2026-08-27T03:40:09Z]
stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 94b24e6670640ca59f40a68b6c78c001d3be0fe5, b0fa604b7d89a12c920781682f042a00cb0c6022, 8a41f5a16587402a79ac45fc04d5bec94baa58ea, c59c6965b86296a6d583282a1140d047cfbe9b1e, 9d9b594bd770859345ae195da7addd6fc3b3392f, 0b03084e310bff479fbb99787891404a6ba581f9, 42fbca68ab200aa2960685037918be26b79e95e8, 23d7b1e639df50fcbffd5ca8f78dbb1de4823df2, 411dd55f7365009227078a6f10b3b58ed30856a3, 0fe40b2d1cf25ef0b4c5f5dd176b50c7f9705e62
- Tests: cd model && mise exec -- lake build Umpire.TargetTests Umpire.Query.Tests.Identity Umpire.Examples.SwitchTests Temporal.Feature.Nexus.LifecycleTests Temporal.Feature.Nexus.Experimental.CallerClosureTests, cd model && mise exec -- lake build Umpire.TargetTests Umpire.Query.Tests Umpire.Planning.Tests, cd model && mise exec -- lake build Temporal.Feature.Nexus.LifecycleTests Temporal.Feature.Nexus.OperationsTests Temporal.Feature.Nexus.Experimental.CallerClosureTests, cd model && mise exec -- lake build UmpireTests TemporalModelTests, make umpire-check-regression, make lint-model
- PRs:
