---
satisfies: [R1, R2, R3, R4, R6, R9]
---
# fn-174-the-model-constructs-its-own-ir-dsl.19 Capture Machine endpoint expressions and construct isolated contributions

## Description
After committed .18, capture authored Machine starts and explicit/default ends and construct complete isolated endpoint contributions (bounded R1/R2/R3/R4/R6/R9). This supplies the remaining expression surface before ordinary Rules lowering. Production Machine assembly stays in full .4.

**Size:** M
**Files:** `model/framework/dsl/Machine.scala`, a private adjacent `MachineEndpoints.scala`, `dsl/CoreBindings.scala`, `reify/plugin/ReifyPlugin.scala`, `reify/body/BodyEngine.scala`, `export/Declarations.scala`, focused DSL/export tests and fixtures, `model/README.md`. These existing seams form one expression pipeline; the adjacent collector keeps its private interface separate without splitting default end or closure validation into an incomplete metadata-only unit.
**Touches:** [model/framework/dsl/Machine.scala, model/framework/dsl/MachineEndpoints.scala, model/framework/dsl/MachineEndpoints.test.scala, model/framework/dsl/CoreBindings.scala, model/framework/dsl/CoreBindings.test.scala, model/framework/reify/plugin/ReifyPlugin.scala, model/framework/reify/plugin/PluginContext.scala, model/framework/reify/body/BodyEngine.scala, model/framework/export/Declarations.scala, model/framework/export/ConstructedEndpoints.test.scala, model/framework/export/testdata/constructedEndpoints/**, model/framework/export/ConstructedHeaders.test.scala, model/framework/export/testdata/constructedHeaders/ProspectiveHeaders.scala, model/framework/export/testdata/constructedHeaders/Timing.scala, model/framework/export/testdata/constructedHeaders/Selection.scala, model/framework/export/testdata/constructedHeaders/Helpers.scala, model/framework/export/ConstructedBindings.test.scala, model/framework/export/testdata/constructedBindings/Arities.scala, model/framework/export/FiniteSchema.test.scala, model/framework/export/testdata/finiteSchema/Native.scala, model/framework/export/testdata/finiteSchema/HelperNative.scala, model/framework/export/testdata/finiteSchema/Prospective*.scala, model/framework/export/ReifyBoundary.test.scala, model/README.md]

### Approach

- Before production edits, freeze complete independent plugin-free old output for the selected positive fixtures, including starts/end, Functions, Types and external references. Establish actual refusals for direct contracts, unsupported bodies, missing Closed, field-bearing Closed sets and unmarked foreign helpers. Use a genuine compiled semantic red, such as the existing plugin accepting a direct unsupported endpoint that the new strict capture must reject; missing APIs, parser/import/safe-init failures are setup observations, not that proof.
- Move the explicit/default end expression builders from `Declarations.scala:277-297` into the shared `BodyEngine.scala` and delegate the existing reader to them. The shared default builder accepts the already-selected local projection, state type and declaration context. Keep existing production reader arms and historical admission. Initial values remain direct Exprs from authored val or parameterless def RHS; explicit ends remain Lambda Exprs with authored parameter names. Do not create a Function and discard its wrapper.
- Follow `ReifyPlugin.scala:217` for pre-pickler calculation and `:308` for post-pickler private payload retention. Capture source-owned declaration-module members strictly, preserving the established class/local/anonymous ownership boundary. Select a direct Phased parent projection from local typed trees; explicit end wins. Reuse or narrowly generalize `PluginContext.scala:142`'s parent-argument recognition if needed, without selecting an unrelated writer lambda. Preserve native bodies and saved author trees. Do not copy the exporter's foreign derivation-tree traversal into the plugin, read foreign helper trees, introduce an author opt-out, or silently swallow capture failures. Default end uses canonical `s` and ordered Closed case literals through the shared role rules.
- Retain compiler-owned endpoint provenance independently from public native getters. Add a dedicated private contribution for ordered starts, end, reachable Functions/Types and exact channel/hole reference identities. It must not masquerade as a complete `ir.Machine`. Missing, premature, stale or conflicting payload/provenance and incomplete expression/parameter/type/function closure refuse with endpoint/declaration context before any contribution escapes.
- Extract the expression-rooted portion of `CoreBindings.scala:169-217` so both collectors share dependency traversal and conflict checks. Include called helpers, their requires/body and named type closure, parameters, field/construct/literal references and external channel/hole IDs. Remove unreachable messages deterministically; preserve original names and Function positions. Validate reference completeness, while leaving external declaration ownership and G01 to full .4. Preserve all existing binding behavior and raw duplicate qualifications.
- Use `PluginContext.scala:95-119`'s existing stored-IR import boundary for foreign `@lifted` helpers. Add same-unit, cross-file and helper-jar tests for this new entry point, including unmarked helper refusal. Preserve measured contracts. Direct require/ensuring in init/end refuse; helper requires stays on the called Function and helper ensuring follows the old function builder's stripping behavior.
- Forward payloads through the actual native Derived/Built source relationships at `Machine.scala:667-715`. Derived declarations retain their identity while source endpoint/helper names remain unchanged. Collection and attachment add zero init/end, public-name, rules/table, phase/refinement, callback or catalog reads. Account separately for the existing Built construction reads at `Machine.scala:571-579`. Early superclass collection refuses pending registration; inherited native-only support endpoints remain uncaptured and collection refuses missing metadata.
- Preserve `.18`'s original `finiteSchema/Roles.scala`, `original.json`, `helper-old.json` and all retained old sources/jars byte-for-byte. That source mixes admitted declarations with intentional endpoint negatives. Keep it for plugin-free old-oracle/refusal reads; add separately named prospective plugin fixtures containing the same admitted declaration spellings and ordinary support-base native controls. Compile direct OpenDefault/FieldDefault as separate semantic negatives. Keep FieldConsumer's old unindexed-sugar stage qualified; its native behavior supplies no historical field-bearing endpoint parity.
- Adapt only the old-refused native counter/throwing controls in `constructedHeaders/{Timing,Selection,Helpers}.scala` and `constructedBindings/Arities.scala` through ordinary support bases. Preserve their counter/reference assertions, ordinary getter/val initialization and helper-jar behavior. Preserve frozen originals and old-admitted sources. No package/test-name skip, cast, manual Finite replacement or new annotation may excuse an unsupported Model body. The six actual samples remain unchanged read-only regressions.
- Preserve original `constructedHeaders/Headers.scala` and its frozen JSON/jars unchanged. Its mixed file also contains Scalar/Opaque/OpaqueSeparate, whose complete endpoint export already refuses historically. Use a separately named `ProspectiveHeaders.scala` for plugin/native header checks, retaining all old-admitted declaration bodies verbatim and moving only those three native-only endpoint bodies into ordinary support bases. Keep historical reads/refusals on the original, assert absent endpoint metadata for prospective native-only controls, and preserve header schema/reference/timing assertions. The final header failure at explicit end parameters (lines 25/42/53) establishes this fixture partitioning need; it authorizes no change to production type/capture admission.
- Extend the established full-old/projection/two-JVM/saved-tree harness in `ConstructedHeaders.test.scala:83-147` and `ConstructedBindings.test.scala:174-243`. Compare starts/end plus complete reachable dependencies with Position-only removal, and check Function positions separately. Update README to state the partial capability, schema-versus-endpoint distinction and all remaining full .4 obligations. Run focused checks and formatting on frozen final source; join every owned process handle.

### Investigation targets

**Required** (read before coding):
- `model/framework/export/Declarations.scala:232-363` and `model/framework/reify/body/BodyEngine.scala` for old expression lowering and the shared-engine seam.
- `model/framework/reify/plugin/ReifyPlugin.scala:201-338` and `model/framework/reify/plugin/PluginContext.scala:95-159` for capture/store/runtime phases and helper imports.
- `model/framework/dsl/CoreBindings.scala:169-217` for shared complete expression closure.
- `model/framework/dsl/Machine.scala:489-715` for metadata, construction and source forwarding.
- `model/framework/reify/body/Expressions.scala:412-433` and `Roles.scala:118-159` for contracts and default-role admission.
- `model/framework/export/FiniteSchema.test.scala:143-251` and `testdata/finiteSchema/Roles.scala` for the mixed historical fixture boundary.

**Optional**:
- `model/framework/export/ConstructedHeaders.test.scala` and `ConstructedBindings.test.scala` for retained old/native controls and fixture-building patterns.

### Resolved planning proof

Artifacts are under `/tmp/fn174-endpoints-tree.F9Mmgz`, based on committed .18 closure `5cc765b8f798bc9dbfa52cd952c84a1c42093a8f`. Handle 57123 compiled direct val/def starts, explicit/default Phased ends, Derived and nested modules against the committed .18 plugin with strict settings and printed pre-pickler reifyStore/post-pickler reifySignature trees. The native probe exited 0, preserving inherited helper-val init=1/end=1, zero incremental header reads, default Closed behavior and explicit override. These controls establish surviving local trees and native timing, not new endpoint retention.

Handle 99917 joined exit 0 after strict plugin-free contract fixture compilation. Old InitRequire, InitEnsuring, EndRequire and EndEnsuring each refused outside the liftable subset at their own bodies. HelperRequire and HelperEnsuring admitted. The guarded Function is `fixture.endpointcontracts.Helpers$.guarded` with requires Var(s) and body Var(s); assured is `Helpers$.assured`, requires absent and body Var(s). Whole raw Models and exact diagnostics remain retained.

Handle 83330 joined exit 0 after strict plugin-free endpoint fixture compilation. The old reader admitted six Machines, named default, directDef, directVal, explicit, forwarded and nested, plus Phase/Snapshot Types. Default end has canonical s and ordered done/failed Closed literals; Forwarded retains the source expressions. The inherited support module refused because it declares no init, while its native helper-jar timing probe passes. The independent whole old output SHA256 is `c56e93a404c96a5d5cfd8bdc197fceceb4910bae27e11f323313904fb484091c`; source is `1cd5ed33d56aec1ce1ff32bb4dd9b63b08b8ab408c35e5791950695748cd58ab` and original jar `5c8d206eec5d30a97a898967fa6c96ff9055687f0b1649a27360c852b71e7557`. No production source was changed during these probes and all handles joined.

### Boundaries and dependencies

Follow committed .18 only. Add .19 to full .4 while preserving full .1, full .3, .16, .17 and .18. Full .3 keeps full .2, .10-.15 and .18; fn-169.2 still waits for full .2. Preserve every conductor-owned cross-spec prerequisite.

Hold `model/temporal`, `tools/umpire`, Go/integration, root Model/Make builds, schema/proto/generated outputs, registry/module-map/migration edits and regeneration. No new library, author API, sample edits, production producer switch, duplicate producer or reader retirement. Existing independent artifacts and owner design edits remain unchanged. Review and broad gates run at source-batch end; explicit review=none for this unit yields no SHIP claim.

Full .4 retains complete Machine assembly, ordinary Rules, evidence, monitors/refinement, complete derivation and channel validation, automatic supported nested-witness propagation, G01, consumers, sole production ownership, activation/retirement, full R1/R3/R4/R5 and every original prerequisite. Isolated endpoint metadata and closure do not discharge those obligations.

### Quick commands

```bash
mise exec -- scala-cli test model/framework/dsl model/framework/realize --server=false
mise exec -- scala-cli test model/framework/export --server=false --test-only framework.export.ConstructedEndpointsSuite
mise exec -- scala-cli test model/framework/export --server=false --test-only framework.export.ConstructedHeadersSuite
mise exec -- scala-cli test model/framework/export --server=false --test-only framework.export.ConstructedBindingsSuite
mise exec -- scala-cli test model/framework/export --server=false --test-only framework.export.FiniteSchemaSuite
mise exec -- scala-cli test model/framework/export --server=false --test-only framework.export.ReifyBoundarySuite
```

The endpoint suite is created by this task. Baseline affected existing suites before edits and use direct framework/IR-classpath compilation for default-end controls; the existing DefaultEndsSuite's Model-jar build route remains held. Run compilers serially and configured formatting on changed Scala files. A source-identical disposable snapshot is permitted only for diagnosed disk setup failure, with the qualification recorded and protected old classpaths retained.

## Acceptance
- [ ] Fresh complete pre-change old output is retained. Ordered starts, exact explicit/default end Lambda expressions and complete reachable Function/Type/external-reference dependencies match isolated native contributions after removing Position only. Explicit parameter names, default s, helper/type identities, case order and bodies receive no normalization. Two native JVMs and current-reader saved-tree replay agree; Function positions remain present and are checked independently.
- [ ] Own-module val/parameterless-def starts and explicit/default ends capture strictly through the shared body implementation before pickling, with runtime retention afterward. Direct contracts and unsupported bodies retain measured located refusals; helper contracts remain on Functions under old semantics. Local, cross-file and jar helper positives use stored foreign IR, unmarked helpers refuse, and direct missing-Closed/field-bearing default ends keep actual diagnostics. Explicit end takes precedence. No foreign helper-tree reads, error swallowing, skiplist or author opt-out appears.
- [ ] A dedicated partial contribution returns starts/end and complete reachable closure using the shared binding walker. Missing or malformed roots, expression/parameter/type data, absent callees, stale/conflicting provenance/definitions, incomplete shapes and empty external IDs refuse contextually before return. Unreachable dependencies are removed deterministically. External declaration ownership and G01 remain unclaimed. Derived/Built payloads follow their actual source relationships without helper renaming or expression recomputation.
- [ ] Retention and repeat collection add zero native getter, callback, rules/table, phase/refinement, public-name or catalog reads. Early collection refuses pending registration, and inherited native-only controls refuse absent endpoint metadata. Counters distinguish ordinary Built reads from collection. Custom empty/throwing catalogs, initialized helper vals, witness/callable/source identity, direct/default/Derived forms and chained Built routes preserve native behavior.
- [ ] Original .18 Roles.scala/raw JSON/jars and earlier frozen proofs remain unchanged. Prospective plugin fixtures retain old-admitted declaration spellings; dedicated direct endpoint negatives and inherited native controls retain honest stage qualifications, including FieldConsumer's old sugar refusal. Scoped final suites, unchanged six-sample regressions and configured formatting pass with all handles joined. README records the partial capability and remaining full .4/R5/batch-review obligations; only authorized framework/tests/README changes occur, with every original dependency and held path preserved.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
