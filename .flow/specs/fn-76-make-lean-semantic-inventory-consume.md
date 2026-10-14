# Make Lean semantic inventory consume semantic contracts

> Local HTML render lens: `.flow/artifacts/fn-76-make-lean-semantic-inventory-consume/spec.html` — open locally; regenerable, markdown is the record. <!-- flow-next:artifact-link -->

## Goal & Context
<!-- scope: business -->

The semantic inventory already documents owner-defined outcomes; it does not override their behavior. Its dependency direction nevertheless places documentation vocabulary beneath execution and evaluation: `Umpire.Planning.Engine`, `Umpire.Observation.Evaluation.Types`, `Umpire.Observation.Verdict`, `Umpire.ImplementationLink.Application`, and `Umpire.Artifact.Runtime` import `Umpire.SemanticInventory.Types`. The Result Artifact also obtains its Known Gap carry contract through these dependencies.

Make semantic owners publish their own contracts through small semantic dependencies, with the inventory consuming those declarations. Preserve the generated inventory and all existing semantic and assurance behavior. This is an independent architecture track: it requires neither fn-75-separate-lean-target-semantics-from's Target split nor the Nexus3 model-to-Case lowering work. It implements the inventory portion of architecture review finding 6 and preserves SEM-01, EVD-05, and ART-07.

## Architecture & Data Models
<!-- scope: technical -->

Use the existing owner-specific status types and classifier declarations. Do not introduce a shared status enum or move status evaluation into the inventory.

The ownership split is:

| Contract | Owner after the change |
| --- | --- |
| `OutcomeConstructorDescriptor`, `OutcomeConstructorClassifier`, `ofValue`, and `OutcomeConstructorClassifiers` operations/propositions | One minimal neutral `Umpire.OutcomeClassification` module |
| `ProjectionSentinelDescriptor` | The same neutral module, because Implementation Link publishes an owner-defined projection sentinel without depending on an inventory catalog |
| Concrete `constructorClassifiers` and `constructorClassifiers_exactlyOne` declarations | Their existing Planning, Artifact runtime, Observation, Implementation Link, and Verdict owners |
| `ImplementationLinkStatus.notEvaluatedProjectionSentinel` | Its existing Implementation Link owner |
| `KnownGapCarryMapping` and its stable `name` rendering | The existing `Umpire.KnownGap` semantic owner |
| `EvidenceGap.knownGapAdmissionMapping` and `ResultArtifact.knownGapCarryMapping` | Their existing Observation and Result Artifact owners |
| `OutcomeFamilyDescriptor`, Known Gap catalog descriptors, lineage/scope/source-shape vocabulary, and catalog uniqueness | Semantic inventory |
| Assembly, validation, sorting, and Markdown rendering | Existing Temporal semantic inventory tool |

Keep qualified declaration names where possible: moving a declaration between modules does not require renaming its `Umpire` namespace. `Umpire.SemanticInventory.Types` imports the narrow semantic contracts it needs; it may continue to expose those names to documentation consumers through ordinary imports, but semantic owners must never import that compatibility surface.

The neutral classification module contains only the existing reusable classifier/projection vocabulary and supporting list operations and propositions. It imports the smallest established foundation needed to compile and has no dependency on SemanticInventory, concrete stage owners, Temporal, or tool aggregation. This is a small extraction of already-shared contracts, not a generalized documentation framework. Keep unrelated Core declarations and Target interfaces untouched.

The resulting dependency direction is semantic owner → neutral classification/Known Gap contracts, and inventory/tool → semantic owner contracts. Update every affected production owner in the transitive graph, including Artifact runtime and Implementation Link, rather than removing only the two imports named in the review.

Use the current model import-graph checker and its tests to enforce this direction. Production Umpire modules outside the SemanticInventory namespace must not directly or transitively reach `Umpire.SemanticInventory` or its descendants. Use the checker's existing distinction between production and test consumers so dedicated inventory tests, including the Planning Known Gap catalog tests, remain valid. Inventory consumers may import semantic owners; test fixtures must not become production intermediaries. Remove inventory aggregation from the ordinary Umpire umbrella if it would violate this direction, and migrate inventory consumers to an explicit inventory import; do not exempt the umbrella merely to preserve a transitive dependency. Preserve deterministic qualified-path diagnostics and existing graph reconciliation.

## API Contracts
<!-- scope: technical -->

Preserve the existing type parameters, fields, constructors, operation signatures, and proposition meanings of relocated declarations. In particular, `OutcomeConstructorClassifier Outcome` retains its descriptor and `Outcome → Bool` matcher; `ExactlyOne` continues to quantify over every value, including arbitrary payloads, rather than an enumerated sample.

`KnownGapCarryMapping` remains closed to `exact` and `observationAdmission`. Its renderings remain exactly:

- `kind -> kind; code -> code; subject -> subject; detail -> detail`
- `code -> code; subject.toList -> relatedDefinitionIds; kind -> absent; detail -> absent`

The Observation projection remains intentionally lossy: no new preservation of `kind` or `detail`, no inferred subjects, and no new acceptance semantics. Exact Result Artifact carry retains all four Known Gap fields. The catalog observes these owner declarations; it must not maintain an independent mapping specification.

Keep the inventory's ten owner-defined outcome families, their constructor order and descriptions, and its projection-only `not-evaluated` distinction unchanged. Keep generated Markdown byte-for-byte stable, including catalog IDs, owner labels, row ordering, mappings, escaping, and descriptions. No artifact schema, JSON, command-line, Definition ID, Behavior Fingerprint, or checksum contract changes are authorized.

## Edge Cases & Constraints
<!-- scope: technical -->

Payload-bearing Planning outcomes such as `found` and `invalid` must remain classified for every payload; finite fixture examples do not replace the existing exhaustive theorem. Projection sentinels must not become outcome constructors. Stage statuses remain independent, and all current success, unknown, conflict, unsupported, invalid, limit-reached, unsatisfiable, and complete-search interpretations remain unchanged.

Retain current inventory validation of malformed or duplicate descriptors and Known Gap catalog rows, together with fail-before-render behavior. Preserve Known Gap validation and rejection semantics for invalid identifiers, duplicate entries, conflicting detail, and noncanonical order. No classifier or metadata change may affect model admission, evidence handling, planning, or canonical serialization.

Preserve existing comments alongside moved declarations and update only documentation whose ownership description changes. Respect the pinned Lean toolchain and existing libraries. Preserve proof statements, constructor privacy, checked admission boundaries, and the approved axiom baseline; no `sorry`, new custom axioms, or newly introduced compiler-trust proof path.

This extraction adds no runtime state, persistence, concurrency, network access, or input limits. Crash recovery and behavior under increased execution load are unchanged. Compilation dependency size can improve, but a new benchmark or build-time target is unnecessary.

## Acceptance Criteria
<!-- scope: both -->

- **R1:** Shared classifier/projection contracts are owned by one minimal neutral semantic module, Known Gap carry contracts are owned by `Umpire.KnownGap`, and concrete classifiers/mappings remain with their semantic owners. Inventory-only catalogs remain inventory-owned. Errors: no new runtime error surface; Lean compilation rejects missing dependencies or incompatible signatures.
- **R2:** The complete first-party import graph rejects every direct or transitive production Umpire dependency on SemanticInventory outside that namespace, including paths through Artifact runtime, Result Artifact, Implementation Link, a facade, or a helper. The neutral classification module stays independent of concrete owners and inventory. Existing inventory/test consumers remain allowed. Errors: focused graph regressions demonstrate deterministic rejection of both direct and indirect forbidden paths, successful reverse-direction consumption, and no blanket exception for production facades.
- **R3:** Existing qualified classifier APIs, constructor order, descriptions, exhaustive `ExactlyOne` proofs, and projection sentinel distinction are preserved for all ten outcome families. Errors: duplicate names, unmatched or multiply matched outcomes, and accidental sentinel membership remain detectable; arbitrary Planning payloads remain covered by the owner theorem.
- **R4:** Exact Result Artifact carry and lossy Observation admission retain their current contracts and generated field-mapping strings. Existing Known Gap catalog validation and production/test scope behavior remain unchanged. Errors: current malformed identifiers, duplicates, conflicting detail, invalid catalog sources, and noncanonical data still reject through their existing owners.
- **R5:** The semantic inventory generator produces byte-identical output for unchanged checked declarations, and its existing renderer, CLI, and Make publication/verification regressions pass. Existing checks for invalid inventory input and failed generation remain effective. Errors: validation failures emit no successful inventory; a stale checked document remains a check failure rather than being silently refreshed.
- **R6:** Planning, Observation, Implementation Link, Result Artifact, and fingerprint/serialization regressions pass with unchanged semantic results, IDs, fingerprints, checksums, and diagnostic classifications. Existing checked-constructor restrictions remain intact. Errors: invalid, unsupported, incomplete, or ambiguous inputs retain their existing failure outcomes; there is no new error surface from declaration relocation.
- **R7:** The affected Lean declarations and import tests compile under the pinned workspace configuration; the normal model build, `make lint-model`, applicable inventory/regression generation checks, and required `make lint-code` are run. Changed trust-bearing declarations have an explicit before/after transitive axiom comparison with no added assumptions. Errors: placeholders, strengthened assumptions, new unapproved native/compiler trust, failed gates, or unverified baseline failures prevent a completion claim unless handled under the project's explicit waiver policy.

## Boundaries
<!-- scope: business -->

- No Target semantic/elaboration/codec split; fn-75-separate-lean-target-semantics-from owns that work and is not a prerequisite.
- No Nexus3 model-to-Case lowering, new semantic frontend, finite-mapping proof library, or new model behavior.
- No standalone Testpilot protocol, environment binding, reusable Go driver, or activation-state refactor; fn-71-standalone-lean-testpilot-protocol, fn-73-explicit-environment-binding-for, fn-72-extract-the-reusable-temporal-testpilot, and fn-74-deepen-testpilot-worker-activation own those independent tracks.
- No new generalized documentation framework, registry, reflection mechanism, status enum, or renderer/publisher replacement.
- No artifact format changes, fingerprint migration, manually edited generated inventory, optional checker changes, or expansion of proof trust.

## Decision Context
<!-- scope: both -->

Extract the already-shared classifier vocabulary once because its multiple semantic owners need the same small interface. Moving it into Planning or Observation would make unrelated owners depend on one another. Putting all inventory types in Core would preserve the inversion under a broader name. Leave catalog-only structures behind and move Known Gap carry alongside Known Gap itself, where its semantic meaning already belongs.

The projection sentinel descriptor must participate in the extraction: Implementation Link currently publishes it, and Observation consumes Implementation Link transitively. Leaving that type in SemanticInventory would defeat the required boundary even after changing the two review-highlighted imports.

The existing import graph infrastructure already computes transitive reachability and deterministic paths, so extend its policy and tests rather than add a text-search gate or another import scanner. Test-only catalog consumers are legitimate and remain explicitly distinguished from production code.

fn-76-make-lean-semantic-inventory-consume can ship independently of all other architecture tracks. fn-68, fn-74, fn-75, and fn-78 are delivered compatibility baselines. Preserve fn-75's semantic import policy and complete external-metadata traversal without adding a dependency solely for overlapping files. Keep this change observationally inert: dependency ownership improves while semantic outputs and documentation stay stable. Nexus operation cancellation remains deferred to fn-79.

## Delivery and verification

Three cohesive tasks deliver this extraction: relocate existing contracts and migrate consumers; enforce the dependency direction; then qualify compatibility, trust, and documentation. The latter tasks consume the preceding changes and run sequentially in the shared checkout.

Before the first source edit, capture the checked inventory bytes, applicable canonical fixtures, and full transitive axiom inventories of relocated declarations, generated auxiliaries, all ten owner ExactlyOne proofs, carry mappings/renderings, and sentinel declarations. Retain complete multiline raw output and an explicit qualified-name mapping; truncated JSON or a post-edit baseline cannot establish unchanged trust. Final qualification compares against those original artifacts.

The neutral module must compile from the smallest existing list/Boolean foundation, without Core or any concrete semantic stage becoming an indirect dependency. Inventory compatibility imports remain available to explicit catalog consumers. Production facade, helper, external-wrapper, and test-fixture paths get no inventory exemption; dedicated test consumers remain allowed under the existing classification rules.

Run renderer tests and both inventory IO test executables, not only their builds. The CLI regression intentionally changes a source mtime; keep verification serial and complete it before the final build/import checks. Check generated inventory without publishing or changing expected Markdown. Preserve existing failure-before-output and atomic publication tests.

Final gates are the normal model build, complete model lint (including builtin lint), applicable inventory/regression checks, and non-fixing Go lint. The current inherited Go-lint baseline is 1,284 raw / 825 distinct path-and-message diagnostics; compare exact sets and report unreached phases. A killed process or missing result is not a passing gate. Preserve disk headroom and serial Lean execution, and recheck the same process handle before deciding it ended.

## Requirement coverage

| Requirement | Tasks |
| --- | --- |
| R1 | 1, 3 |
| R2 | 1, 2 |
| R3 | 1, 3 |
| R4 | 1, 3 |
| R5 | 3 |
| R6 | 1, 3 |
| R7 | 1, 2, 3 |

