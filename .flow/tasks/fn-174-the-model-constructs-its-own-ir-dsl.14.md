---
satisfies: [R3, R4, R6, R9]
---
# fn-174-the-model-constructs-its-own-ir-dsl.14 Construct exact Actor names with compiled old/native equivalence

## Description
Construct exact Actor names for isolated native signature messages (R3/R4/R6/R9), after .13's committed source closure. This is a separate source checkpoint. It neither activates signature export nor completes full .3.

**Size:** M
**Files:** Actor definition in Action.scala, Signature.scala, existing plugin naming seams only if required, focused native/compiled Actor controls, fixture sources and README.
**Touches:** [model/framework/dsl/Action.scala, model/framework/dsl/Signature.scala, model/framework/dsl/Actor.test.scala, model/framework/reify/plugin/ReifyPlugin.scala, model/framework/reify/plugin/SchemaPhase.scala, model/framework/export/ConstructedActors.test.scala, model/framework/export/testdata/constructedActors/**, model/README.md]

### Approach
- Before production changes, compile an independent framework-only old-reader Actor fixture and retain its exact source, raw output, command and task-base provenance. Reuse the existing compiled signature/residual suite packaging and current-reader APIs. Prove actual old admission/refusal rather than converting source-read hypotheses into acceptance claims. Do not generate the old oracle from native conversion or rewrite it after a fix.
- Compare complete position-stripped Actions, not just actor strings. Cover default, explicitly empty and explicitly named Actor objects through direct references and this; Actor.system; nonempty and empty Actor factories; nested and inherited objects; source-authored backticked identifiers. Assert exact names, canonical Action IDs and unchanged non-actor fields, including modifier copies.
- Keep public Actor.name, toString, constructor arguments, native references and ordinary evaluation unchanged. Prefer one private effective IR-name seam on Actor consumed by Signature.action. Fix confirmed empty-factory and named-object precedence defects there. Pin getter overrides and their side effects separately; the old module naming branch ignores those getters, so IR naming must not acquire new getter evaluation. No general declaration naming registry or Actor identity table is needed.
- Use the smallest established compiler naming seam only where the native representation cannot prove exact source spelling. NamesPhase.transformTemplate already registers compiler metadata before object-body declarations; prefer post-pickler runtime attachment where feasible to preserve saved trees. Retain the compiler's module spelling for Actor Module classes only, including inherited/nested/encoded spellings, without guessing from a JVM class name or overriding a concrete object's name with its base class. Preserve saved TASTy consumed by the current reader. Object-body registration follows superclass/trait initialization. Independently classify an inherited action(this), deferred conversion and conversion attempted during inherited initialization; prove safe resolution or contextual refusal before registration, never a base-class-name fallback.
- Separately pin actual old admission for imported renames, val aliases to in-place actors, module aliases, helper calls, direct new/named/default constructor forms and helper jars. Newly supported old-refused forms get qualified native controls, not historical parity. Preserve aliases' Actor reference, evaluate helper/constructor arguments once, and do not evaluate an Actor getter to discover declaration ownership.
- Missing/conflicting retained module naming evidence and unsupported identity/name shapes fail with owning Action context. Do not invent a broad nonempty-actor rule; exact empty strings are established conversion values. Keep class/method-owned Action declaration admission with the existing identity checks, not a new Actor naming policy.
- Run deterministic projections in two JVMs and replay the current reader against plugin-produced saved trees. Iterate only touched suites; at the final source boundary run affected DSL/realization and compiled Actor, Signature, ResidualSignature, References, Names, FiniteSchema, Boundary, Samples and ModuleFixtures checks serially with no source edits underneath them. Join every original handle. Check configured formatting on explicit changed Scala paths and diff integrity. No new proof script or hash-manifest framework.
- Update the staged README naming section and the affected authoring explanation with precise native versus IR semantics, supported forms and qualifications. Preserve .13's old oracle and source receipt, the nested Some(None) qualification, unresolved Inbox/consumer ownership and the active-reader/full integration boundaries.

### Investigation targets
**Required:**
- model/framework/dsl/Action.scala:9 and Signature.scala:85. Current Actor constructor/getter and isolated IR consumer.
- model/framework/export/Declarations.scala:136 and model/framework/reify/body/Constants.scala:17,48,72. Old module versus constant naming, alias resolution and constructor/helper boundaries.
- model/framework/reify/plugin/ReifyPlugin.scala:306 and SchemaPhase.scala:104,397. Existing template metadata and phase ordering; Actor currently has no declaration/provenance attachment.
- model/framework/dsl/Machine.scala:52 and model/framework/reify/body/BodyContext.scala:318. Runtime name helper versus compiler identity/admission.
- model/framework/export/ConstructedResidualSignature.test.scala:93,142,164. Independent whole-message oracle, two-JVM projection, current-reader replay and qualified helper-jars controls.
**Optional:** model/framework/dsl/Signature.test.scala:20; model/README.md:837,891,1191,1512 and model/SEMANTICS.md:162 for current authoring/qualification text.

### Verification and boundaries
Six unchanged framework samples are read-only regression inputs. No Temporal, tools/umpire, Go/integration, root Model/Make, schemas/protos/generated/IR regeneration, registry/module-map or migration changes/runs. Independent review and broad validation remain deferred to source-batch end. Root owns Flow/MILESTONES/completion and independent committed-source verification. Source worker returns with .14 in_progress, a source commit and terminal handoff; all original handles must be joined. Preserve owner design dirt and use exact scoped staging. Normal git commit is prohibited following the earlier unexplained empty-tree publication; verify a literal nonempty tree and parent, commit-tree, inspect exact paths and CAS update-ref.

### Planning receipt
SHORT delta repo/docs-gap, spec/dependency and memory research with a flow-gap decision pass. No new library/API needs external research. Historical source-shape/default-empty memory provides constraints, not current Actor defect authority. Full .3 retains full .2/.10/.11/.12/.13 and adds this source prerequisite. Full .4-.8, fn-141 ledger obligations, fn-169.2 and R5 remain unchanged. Inbox ownership is a later independent source task. Rejected a general naming registry as unnecessary. Generated API drift verification and new CI remain declined and out of scope.
stage: plan-review - skipped(policy: owner defers independent review to source-batch end)
stage: html-render - skipped(policy: owner focus remains framework source and bounded tracking)
memory: bm25 (jev-unavailable(no_key)); targeted all-track memory scout completed.

## Acceptance
- Independent pre-production old-reader evidence proves admitted Actor forms and exact complete Action messages. Native conversion matches those messages after removing only positions, including empty in-place names and object-name precedence; two separate JVM projections and current-reader saved-tree replay pass without regenerating old evidence.
- Default/empty/named objects, direct this/references, system, empty/nonempty factories, nested/inherited/backticked objects and Action modifiers retain exact names and non-actor fields. Native public getters/overrides, toString, constructor behavior, reference identity and evaluation counts remain unchanged. IR naming adds no authored getter evaluation. Object-body and superclass/trait initialization controls prove safe deferred naming or contextual refusal before metadata is available, never a base-class-name fallback. Source module spelling is not guessed from a JVM-encoded class name.
- Alias/import/helper/jar/new/default/named-constructor forms are classified by actual old-reader admission. Newly supported old-refused forms are qualified separately. Missing/conflicting metadata and unsupported forms fail with Action context; no new blanket nonempty-name requirement, catalog enumeration, transition execution or general Actor identity registry is added.
- Final affected suites, unchanged sample inputs, configured scoped formatting and diff checks pass on unchanged source. README documents precise IR/native semantics and support limits. Full .2/.3/R5, Inbox/type ownership, consumer fairness, activation, complete machine/composition conversion and reader retirement remain open; held surfaces stay untouched.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
