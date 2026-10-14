---
satisfies: [R1, R3, R4, R6, R9]
---
# fn-174-the-model-constructs-its-own-ir-dsl.16 Retain captured core bindings and construct isolated binding contributions

## Description
Prepare dormant core binding contributions for R1/R3/R4/R6/R9 after the committed .15 source closure. Preserve captured StepBinding metadata through the native table and core derivations, then collect ordered binding messages and their reachable function/type closure. Complete Machine messages remain .4 work.

**Size:** M
**Files:** model/framework/dsl/Machine.scala, model/framework/dsl/Reify.scala, an adjacent private binding collector and DSL tests, model/framework/reify/plugin/ReifyPlugin.scala, narrowly required provenance hooks in model/framework/reify/body/BodyEngine.scala or SchemaPhase.scala, one compiled export suite and fixtures, model/README.md.
**Touches:** [model/framework/dsl/Machine.scala, model/framework/dsl/Reify.scala, model/framework/dsl/CoreBindings.scala, model/framework/dsl/CoreBindings.test.scala, model/framework/reify/plugin/ReifyPlugin.scala, model/framework/reify/plugin/SchemaPhase.scala, model/framework/reify/body/BodyEngine.scala, model/framework/export/ConstructedBindings.test.scala, model/framework/export/testdata/constructedBindings/**, model/README.md]

### Approach
- Start from the conductor's disposable old-reader discovery and independently retain a focused pre-change complete raw Model, original source, exact command and base before production edits. Reuse existing compiled fixture and current-reader replay patterns. Project the selected machines' ordered bindings and reachable function/type closure without renaming or alpha normalization. Qualify every old-refused or native-only form separately.
- Preserve private capture payloads at StepBinding, Bound.Function, Bindings.table, Machine.bindings and core derivation seams. Keep the identical callable, two-field public StepBinding constructor/product/equality/extractor and native invocation behavior. Retain pass-through source bindings through restrict, unmonitored and Derived.table.
- Retain compiler-authoritative named-versus-anonymous provenance using the shared body engine's forwarded-def interpretation. A forwarding lambda can name a def. No inference from root string suffixes, JVM names or native function identity. Rebase only anonymous roots to the historical binding owner/action name before merging payloads. Untouched derived entries retain their source roots; replacements and additions retain the actual derived declaration's origin. Do not mutate a source capture shared by later derivations.
- Retain ordered core operation operands and source admission context before rebind/extend reduce them. The isolated contribution checks actual old-refused duplicate replacements, unknown targets and already-bound extensions, including invalid intermediate operations hidden by later restriction. Initial Bindings preserves duplicate rows. For initial anonymous duplicates with the exact same historical owner/action root, preserve the observed ordered last-definition function payload, then project its reachable closure. Do not add indexes or rename those roots to avoid this historical collision. Distinguish that documented source case from unrelated or stale conflicting closures. Do not re-run contextual binding functions or change ordinary native derivation behavior.
- Build one private collector for captured Bound.Function rows. Resolve Action IDs and effective names through Signature.action. Traverse complete reachable Function body/requires/call and type dependencies, preserving intrinsic type references and exact external signature references. Validate roots, callees, type closure and conflicting definitions before returning any result. Sort function/type catalogs deterministically while preserving binding, parameter, field, case and expression order. Complete channel/hole consumer validation remains integration work.
- RuleGroup, Bound.Ruled and Bound.Disabled remain contextual unsupported contribution forms. Rebinding a Ruled entry captures an effect only and cannot pretend to hold a complete lowered step function. Complete rule lowering remains .4 work.
- Treat copied bindings as admission controls, without adding generic copy capture or an author API. Unchanged declaration/callable copies may retain compatible metadata; a changed callable must clear or refuse stale capture. A changed declaration requires authoritative binding identity. Preserve ordinary copy/equality behavior and qualify old-refused spellings.
- Conversion adds no callback, guard, state/message catalog, init or refinement getter evaluation. Record existing native derivation construction separately since built already reads init/refinement and Rules overlap already runs catalogs/guards. Keep the six sample sources and all previous source/oracle inputs unchanged.
- Update README with the exact dormant capability, naming/closure/refusal/evaluation boundaries and remaining Machine work. Preserve all prior qualified differences and held integration obligations.

### Investigation targets
**Required:**
- model/framework/dsl/Machine.scala:90-140, :310-345 and :517-682. Capture storage, native tables and core derivations.
- model/framework/dsl/Reify.scala:14-44 and model/framework/reify/plugin/ReifyPlugin.scala:225-302. Capture payload and runtime attachment.
- model/framework/reify/body/BodyEngine.scala:79-91 and Expressions.scala:116-170. Shared named/forwarded/anonymous classification.
- model/framework/export/Declarations.scala:1161-1176 and :1215-1340. Exact historical binding naming and derived admission.
- model/framework/dsl/Signature.scala:180-215 and model/framework/reify/plugin/SchemaPhase.scala. Existing Action identity/name and compiler-owned declaration provenance.
**Optional:**
- model/framework/export/ReifyBoundary.test.scala:525-700 and ConstructedInboxes.test.scala. Callable/product, arity, old/native projection, helper-jar and replay patterns.

### Key context
Bindings.table currently discards StepBinding.captured, Bound.Function stores only its callable, Machine.bindings recreates unlinked bindings, and core rebind/extend destructure away capture. Named defs already keep their full name; anonymous plugin captures use an API-level root instead of the old machine/action root. Rebase before deduplication because distinct lambda captures can share that API-level name.

Planning uses parallel read-only repo/spec/memory scouts and a gap check. Memory retrieval is bm25 with jev-unavailable(no_key). Historical catalog/outcome, refinement-read, empty-extension and source-kind lessons constrain preservation; they do not establish current defects. Broad generated API drift verification and new CI remain excluded.

The disposable discovery lives at /tmp/fn174-bindings-discovery.OWQICb. Original sessions 27201 and 75480 both joined exit 0. Independent original and plugin-replay complete raw Models match byte-for-byte for all 13 admitted roots. Base/Named/Eta/Forwarding/Explicit/ActionAlias/Replaced/Added/Restricted/Unmonitored/Chained/DuplicateInitial accept; function-val alias, all three copy expressions, duplicate rebind/extend, unknown rebind, already-bound extend and unknown-then-restrict refuse. Chained final bindings use chained.named and chained.second; its whole old Model also retains discarded/source functions, so an isolated reachable projection does not claim whole Model equivalence. DuplicateAnonymous preserves two identical action/root rows with the second lambda body as the single function definition. Its raw SHA256 is 1a4821fa3a83e5231b13b67bf9b6db56712dafcd0ef1c730daf0111d3137c9ba. Full header/rule/monitor/refinement/through coverage was not probed. Do not generalize these discovery controls into full .4 acceptance.

Named/eta/forwarding controls retain fixture.bindingdiscovery.Bindings$package$.namedStep. Explicit-name and Action-alias anonymous roots are explicit.authoredName and actionAlias.authoredName, with canonical action ID fixture.bindingdiscovery.explicit. Historical anonymous roots use the machine's IR name without a family prefix; do not concatenate full source IDs blindly. Probe fixture compilation omitted Werror because trailing underscore eta syntax warns. Final task fixtures use the existing strict compiler flags and retain the same admitted semantics. Closure validation must recognize the actual intrinsic construction spelling framework.dsl.Step (present in the raw body without a declared Type), as well as the shared lifter's other intrinsic references; do not guess a different Step namespace.

### Boundaries
Only framework source and model/README.md may change. No complete Machine header/init/end/evidence/monitor/refinement messages, ordinary Rules capture/lowering, complete C06/G01 validation, producer activation, reader retirement or R5 claim. Full .4 retains .1/.3 and adds this source prerequisite; full .3 remains unchanged. Hold Temporal, tools/umpire, Go/integration, root Model/Make edits/runs, schema/proto/generated IR, registry/module-map/migration and regeneration. Six samples are unchanged read-only regressions. Independent review and broad gates remain deferred to source-batch end.

### Quick commands
```bash
mise exec -- scala-cli test model/framework/dsl model/framework/realize --server=false
mise exec -- scala-cli test model/framework/export --server=false --test-only framework.export.ConstructedBindingsSuite
mise exec -- scala-cli test model/framework/export --server=false --test-only framework.export.ReifyBoundarySuite
```
Final frozen verification also runs ConstructedInboxesSuite, FiniteSchemaSuite, ConstructedActorsSuite, ConstructedSignatureSuite, ConstructedResidualSignatureSuite, ConstructedReferencesSuite, ConstructedNamesSuite, ReifySamplesSuite and framework.lint.ReifyModuleFixtures serially, plus configured formatting on all changed Scala files. Samples still prove only their existing 167 old-host checks. Record actual command exits and join every original handle.

## Acceptance
- [ ] Independent complete old output/source/command/base precedes production edits. Every old-admitted focused core shape matches complete ordered binding/function/type contributions with only Position removed, including named, anonymous, forwarding, explicit/alias Action names and supported derivation chains. Two native JVM runs and current-reader saved-tree replay agree. Old-refused/native-only controls remain explicitly qualified.
- [ ] Capture and exact compiler-owned binding origin survive Bindings/Bound.Function/Machine.bindings and core restrict/unmonitored/rebind/extend/Derived forwarding. Anonymous roots rebase before merging; named and untouched source roots remain exact. Public StepBinding product/copy/extractor/equality and identical callable behavior are preserved; changed copies never inherit stale capture.
- [ ] Missing metadata/root/callee/type closure, unrelated or stale conflicting definitions and incomplete origins refuse contextually before output. Ordered operation evidence preserves actual old core derivation refusals even when later operations overwrite them. Initial duplicate Bindings rows remain admitted, including the independently observed duplicate-anonymous ordered last-definition payload and unchanged native behavior. Ruled/effect-only, RuleGroup and Disabled contributions never masquerade complete lowered functions.
- [ ] Conversion is deterministic and invokes no callbacks, guards, values/contents, init or refinement getters. Functions retain positions and complete reachable dependency messages. Assertions separate pre-existing native construction evaluation from conversion evaluation and exercise all current binding arities 0,1,2,3,4,6 plus helper jars and repeated conversion.
- [ ] All Quick/final scoped suites and configured changed-file formatting pass on unchanged final source. README and evidence preserve held .1/.2/.3/.4/R5/review obligations, prior captures and six sample inputs. Record exact production-only Scala delta, qualified failures and all original handles joined. No held-path edits/runs or producer activation.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
