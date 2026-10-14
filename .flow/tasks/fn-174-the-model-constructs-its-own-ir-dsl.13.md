---
satisfies: [R3, R4, R6, R9]
---
# fn-174-the-model-constructs-its-own-ir-dsl.13 Construct residual signature declarations with exact name provenance

## Description
Construct complete dormant Hole, Limits and Assumption contributions and exact declaration/reference name provenance (R3/R4/R6/R9), consuming committed .12. This is one cohesive source checkpoint: these declarations share omitted-versus-explicit naming, and implicit sync names consume the Action effective-name rule. No production producer is switched.

**Size:** M
**Files:** Existing DSL declaration/conversion and plugin naming seams, focused native/compiled tests, residual-signature fixtures and README. Most edits adapt established retention; no new subsystem.
**Touches:** [model/framework/dsl/Assume.scala, model/framework/dsl/Claims.scala, model/framework/dsl/Action.scala, model/framework/dsl/Signature.scala, model/framework/dsl/ComposedReference.scala, model/framework/dsl/DeclarationName.scala, model/framework/dsl/ResidualSignature.test.scala, model/framework/dsl/Action.test.scala, model/framework/dsl/Observation.test.scala, model/framework/reify/plugin/ReifyPlugin.scala, model/framework/reify/plugin/SchemaPhase.scala, model/framework/export/ConstructedResidualSignature.test.scala, model/framework/export/testdata/constructedResidualSignature/**, model/README.md]

### Approach
- First pin independent old-reader evidence for all complete Assumption messages in unchanged Declarations/ComposedProgress, the Hole and embedded Query Limits in Declarations. Read frozen old outputs; never regenerate them. Before production edits, compile a small framework-only old-reader fixture that demonstrates omitted, nonempty and explicitly empty Action/Entity/Observation/Assumption/Limits names, including the implicit sync derived from an explicitly empty Action. Retain its source, original raw output, exact command, base and hashes for native comparison. Setup failures are not semantic red evidence. Assert exact IDs and names, not just flattened fairness references.
- Retain explicit-versus-omitted name provenance at the factory/constructor boundary. Reuse NamesPhase canonical owner/ID retention and SchemaPhase post-pickler declaration-name attachment. Do not infer omission from name.nonEmpty. Preserve raw public native values, ordinary evaluation, helper forwarding, aliases and supported copies; retain constructor, equality, product/copy/unapply shape (Observation two fields, Entity three, Limits four). A minimal scoped DeclarationName carrier is allowed only if it removes duplication; no general naming-framework redesign or global registry.
- Add native conversions on the owning DSL declarations, using the existing Signature collection boundary. Hole builds canonical ID/name without changing reached behavior. Limits builds exact name/bounds; validate each nonnegative bound, accepting zero. Extend isolated collection with deterministic ID/name conflict checks: distinct Assumption/Hole IDs cannot share a name; repeated identical definitions deduplicate; equal Limits names with different bounds refuse. Conversion/collection failure returns no partial Model. Do not add a publication command or production exporter.
- Complete Assumption messages from .12's ordered ordinary/composed fairness contributions. Reuse exact canonical Action IDs and Composed.toIR, preserving duplicates/order. Empty fair() segments alone do not imply mixed fairness. Reject nonempty ordinary plus composed fairness and composed references spanning multiple owners, with declaration context. Do not rebuild sync/member validation, enumerate catalogs for bare actions, or execute guards/effects/starts/ends/exploration.
- Retain Entity/Observation effective reference names and their on/key/refer relationships; they are not new standalone IR declaration tables. Correct Action effective names across input/results/example/on/creates copies via existing boundTo retention. Signature.action and implicit sync naming must use the same effective-name rule. Explicit empty names are valid exact conversion values unless a separately established collision rule applies.
- Plain aliases preserve originating declaration identity. Omitted copy name and explicit copy(name = "") are distinct. Test helper jars and copy/evaluation behavior, qualifying forms not accepted by the old reader as newly supported controls rather than historical parity. Missing/conflicting identity/name metadata refuses contextually; anonymous inferred names cannot become silent empty/default declarations. Preserve current-reader TASTy and .12's early validation/late runtime retarget seam.
- Keep progress-owner and machine-held composed fairness validation with their original consumers. Keep state-field Inbox ownership, machine schemas/rules, complete Composition messages, production activation and reader retirement separate. Do not add a .3 dependency on full .5 or change the original integration ladder.

### Investigation targets
**Required:**
- model/framework/dsl/Assume.scala:6 and Claims.scala:86 — retained ordered fairness, Hole and Limits shapes.
- model/framework/dsl/Action.scala:23,38,46,86,323 and Signature.scala:72,159 — reference names, Action copy metadata and local collection reuse.
- model/framework/reify/plugin/ReifyPlugin.scala:342 and SchemaPhase.scala:94 — existing canonical declaration/spelling seams, preserving saved trees.
- model/framework/export/Declarations.scala:88,1563,1770,1820; Claims.scala:196 — original exact names, intrinsic fairness and collision/bound validation.
- model/framework/reify/body/Constants.scala:63 and BodyContext.scala:301,318 — omitted-default syntax, alias resolution and canonical-name refusal policy.
- model/framework/export/ConstructedSignature.test.scala and ConstructedReferences.test.scala:115 — isolated library/plugin/helper packaging and two-JVM oracle patterns; unchanged lifts/Declarations.scala + expected/declarations.json and lifts/ComposedProgress.scala + expected/composedProgress.json.
**Optional:** model/framework/dsl/Observation.test.scala:6 for native product/raw-value constraints; model/framework/export/testdata/lifts/Captured.scala for inferred names. The observations fixture that imports Temporal is held and must not run.

### Verification and boundaries
Run the new native/compiled residual signature suites; affected DSL/realization, ConstructedSignature, ConstructedReferences, ConstructedNames, FiniteSchema and all three reify suites. Format configured explicit changed paths and freeze source before final serial checks; join every original handle. All six samples remain unchanged read-only regression inputs. Do not run root Model/Make, Temporal, tools/umpire or Go, or modify schemas/generated outputs, registry/module-map/migration files. Root owns Flow/MILESTONES/closure. Independent review and broad validation remain deferred to source-batch end. README must state dormant messages and precise naming/copy support, preserving .11's nested-default qualification and unresolved Inbox/held integration boundaries; remove the explicit-empty Action gap only after its oracle and implicit-sync control pass.

### Planning receipt
SHORT delta research: repo/docs-gap, spec/dependency and active-memory scouts; flow-gap analyst reconciled intrinsic versus consumer checks. No new library/API needs external research. Historical memory constrains identity, default-empty semantics and diagnostic coordinates, not current defect claims. Declaration product shape and explicit-name provenance are tested independently. Full .3 retains full .2/.10/.11/.12 and adds this source prerequisite; R5, activation, reader retirement, .4->.5/.6->.7/.8 and fn-169.2 remain unchanged. Rejected general naming registry/producer as unnecessary; generated API drift verification remains declined and out of scope.
stage: plan-review - skipped(policy: owner defers independent review to source-batch end)
stage: html-render - skipped(policy: owner focus remains framework source and bounded tracking)
memory: bm25 (jev-unavailable(no_key)); targeted memory scout completed.

## Acceptance
- Complete native Assumption/Hole messages and Limits contributions match the independent unchanged old messages after removing only positions; IDs, exact names, bounds and every ordered fairness contribution are asserted. A pre-production old-reader fixture separately pins omitted/nonempty/explicit-empty names, including implicit sync naming from an empty Action; native parity and two-JVM deterministic projections pass without regenerating old evidence.
- Exact name provenance survives supported aliases, helper jars and copies, distinguishing omitted copy names from explicit empty copy names. Action modifiers preserve it; Action IR and implicit sync use one effective name rule. Entity/Observation references preserve native raw values, relationships, case-class constructor/product/equality/copy/unapply shapes and callable evaluation. Newly supported old-refused forms are explicitly qualified rather than claimed as historical parity.
- Intrinsic assumptions preserve duplicates, order and empty segments; mixed nonempty ordinary/composed selections and multiple composed owners refuse contextually. Existing composed-reference ownership/class admission/refusals still pass. No converter executes guards/effects/starts/ends, explores states or enumerates catalogs to discover references.
- Each negative Limits bound refuses, zero bounds pass, equal repeated definitions deduplicate, differing same-name Limits refuse; distinct Assumption/Hole IDs sharing names refuse. Missing/conflicting declaration identity/name provenance and anonymous inferred names refuse contextually. Local collection returns no partial Model on failure and has deterministic ordering, with no global registry.
- Frozen affected suites, six unchanged sample regressions and configured scoped formatting pass. README documents dormant complete contributions and precise new support while preserving remaining default/Inbox qualifications. Production reader remains active; no held surfaces change/run. Full .2/.3/R5, consumer-owner checks, Inbox ownership, production activation and retirement remain open.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
