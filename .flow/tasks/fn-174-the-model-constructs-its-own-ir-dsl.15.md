---
satisfies: [R3, R4, R6, R9]
---
# fn-174-the-model-constructs-its-own-ir-dsl.15 Retain exact Inbox state-field ownership and construct native state types

## Description
Prepare the dormant native state-type contribution for R3/R4/R6/R9 after the committed Actor checkpoint. Retain exact Inbox field ownership on existing state Finite witnesses and resolve structural placeholders to constructed Channel IDs. Full .3 integration and .4 machine validation remain their original owners.

**Size:** M
**Files:** model/framework/dsl/Domain.scala, model/framework/dsl/FiniteSchema.scala, model/framework/dsl/Signature.scala, model/framework/reify/plugin/SchemaPhase.scala, focused DSL tests, one compiled export suite and its fixture directory, model/README.md.
**Touches:** [model/framework/dsl/Domain.scala, model/framework/dsl/FiniteSchema.scala, model/framework/dsl/Signature.scala, model/framework/dsl/InboxOwnership.scala, model/framework/dsl/InboxOwnership.test.scala, model/framework/dsl/FiniteSchema.test.scala, model/framework/reify/plugin/SchemaPhase.scala, model/framework/export/ConstructedInboxes.test.scala, model/framework/export/testdata/constructedInboxes/**, model/README.md]

### Approach
- Before production edits, compile an independent framework-only fixture with the active old reader. Retain complete raw type/channel output, original source, commands, base and hashes. Establish actual admission for record/enum fields, separate states sharing a message type, aliases, bounded/opaque messages and nested record ownership. Compile negatives separately. Preserve every old oracle through final checks.
- Keep FiniteSchema's existing structural schema and unresolved message identities immutable. Add private witness-owned ownership evidence indexed by the exact state schema key and typed declaring-type/case/field coordinates. Reuse existing idempotent schema retention and Channel source identity; avoid a global registry or Machine exporter.
- Capture the existing lexical `given Finite[Inbox[M]] = channel.contents` receiver after pickling. Preserve the original witness expression and its catalog behavior. A deferred, memoized Channel supplier may be necessary. Never force the local Inbox Given, read values or contents, sample init, or infer ownership by message type, equal bounds, catalogs or native object equality.
- Add an isolated Signature state-schema/type contribution selecting an exact Finite witness and schema key. Rebind field placeholders using retained evidence and validate typed message identity, complete closure and declaration identity before returning any output. Ordinary input/channel schema conversion must continue refusing unresolved Inbox references.
- Preserve the old admission boundary. Separate state types can hold distinct channels of one message type. Duplicate storage of a channel in root record or enum-case fields refuses. Same-state distinct same-message channels require an independently supported authored spelling; do not introduce one here. Nested record evidence belongs to the nested declaring type and does not become recursive root-held storage.
- Preserve field/case order, bounds, opaque nominal identities, public witness identity, catalogs and authored evaluation. Qualify old-refused helper/alias forms separately from historical parity. Use compiler/source evidence rather than JVM names.
- Update README with the exact dormant capability, its refusal/evaluation boundaries and retained integration qualifications. Preserve the .11 nested Some(None) qualification and all prior frozen fixtures and six sample sources.

### Investigation targets
**Required:**
- model/framework/dsl/Domain.scala:19-104 and FiniteSchema.scala:6-38. Existing witness retention and catalog behavior.
- model/framework/reify/plugin/SchemaPhase.scala:160-295 and :531-590. Pre-pickler structural retention and post-pickler signature attachment.
- model/framework/dsl/Signature.scala:13-69 and :171-239. Schema selection, closure validation and Channel conversion.
- model/framework/reify/body/Types.scala:74-86, :110-143 and :205-224. Historical lexical ownership and nested declaring-type scope.
- model/framework/export/Declarations.scala:1410-1440 and reify/body/BodyEngine.scala:113-132. Existing duplicate storage and receiver lookup boundaries.
**Optional:**
- model/framework/export/FiniteSchema.test.scala:280-340 and ConstructedActors.test.scala. Compiled fixture, independent oracle, helper jar and replay patterns.
- .flow/tmp/handoffs/fn174-inbox-root-research.md and /tmp/fn174-inbox-tree.Pds3Jx/Probe.scala. Disposable probe receipt and evaluation control.

### Key context
Root baseline session 69793 passed all eight FiniteSchemaSuite cases at Actor closure 0be852b8856cfe4e038e642492f68f7d4ed5a5c4. Compiler probe original3548 exited 0; root independently replayed its main. Lazy custom state witness construction and schema reads made zero message catalog calls, values made one, eager Finite.derived construction made the second. The post-pickler tree retains the local lazy Given and original channel receiver beneath retainSchema wrappers. Generated ownership insertion still needs red/green proof.

Equal-range opaque message types retain distinct canonical keys even when their TypeRefs are equal. Match the channel's retained typed schema key, not structural range equality. The old duplicate-storage check flattens enum-case fields as well as record fields. Full held delivery/loss checks are .4 work.

Planning used parallel read-only repo/spec/memory/gap scouts. Memory retrieval was bm25 with jev-unavailable(no_key). Historical portable schema/source, channel catalog and empty-extension lessons apply as constraints, not evidence of a present defect. Declined broad generated-API drift verification stays excluded.

### Boundaries
Only framework source and model/README.md may change. No author annotations or new author API, production signature switch, reader retirement, complete Machine messages, delivery/loss binding execution, consumer fairness or R5 claim. Hold model/temporal, tools/umpire, Go/integration, root Model/Make edits/runs, schema/proto/generated IR, registry/module-map/migration and regeneration. Six samples are unchanged read-only regressions. Independent review and broad gates remain deferred to source-batch end.

### Quick commands
```bash
mise exec -- scala-cli test model/framework/dsl model/framework/realize --server=false
mise exec -- scala-cli test model/framework/export --server=false --test-only framework.export.ConstructedInboxesSuite
mise exec -- scala-cli test model/framework/export --server=false --test-only framework.export.FiniteSchemaSuite
```
Final unchanged-source verification also runs ConstructedActorsSuite, ConstructedSignatureSuite, ConstructedResidualSignatureSuite, ConstructedReferencesSuite, ConstructedNamesSuite, ReifyBoundarySuite, ReifySamplesSuite and framework.lint.ReifyModuleFixtures serially, plus configured formatting on every changed Scala file. Samples prove only their existing 167 old-host checks unless stronger evidence is explicitly gathered.

## Acceptance
- [ ] Independent old output/source/command/base are frozen before production edits. Complete position-stripped type/channel messages match native contributions in two JVMs for every old-admitted positive. New or old-refused controls are explicitly qualified; current-reader replay of plugin-compiled saved trees remains compatible.
- [ ] Ownership selects the exact witness/schema and declaring-type/case/field coordinate. Separate states sharing a message type, bounded and equal-range distinct opaque domains, record and admitted enum/nested scopes, aliases and helper-jar forwarding retain exact channel IDs, ordered fields/cases and closure. Repeated calls are deterministic and idempotent.
- [ ] Missing schema/key/field evidence, conflicting ownership, wrong case/field, extra evidence, message-key mismatch, missing channel identity and incomplete closure refuse with state/schema/field context before any result escapes. Duplicate root channel storage across record or enum-case fields remains refused. Unsupported old lexical forms never silently fall back to catalogs or message-only guesses.
- [ ] Schema attachment and conversion never read state values, channel contents or init. Throwing/empty witnesses succeed where metadata suffices. Local lazy Given timing, actual Channel supplier evaluation/memoization, witness identity and ordinary catalog values remain unchanged; premature/uninitialized references refuse contextually.
- [ ] All Quick and final scoped suites plus configured changed-file formatting pass on unchanged final source. README and evidence state held integration/review/R5 obligations; prior source/oracle hashes and six sample inputs are unchanged. Record exact production-only Scala line delta and all original process handles joined. No held-path edits/runs or producer activation.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
