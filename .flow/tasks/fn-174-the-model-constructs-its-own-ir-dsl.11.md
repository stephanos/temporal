---
satisfies: [R3, R4, R6, R9]
---
# fn-174-the-model-constructs-its-own-ir-dsl.11 Construct Action/Input/Channel signature messages in isolation

## Description
Construct Action/Input/Channel signature contributions beside their native DSL types, consuming .10's retained schemas (R3/R4/R9). Exercise these conversions through isolated framework fixtures. Full .3 retains production collection, activation, signature-reader retirement, remaining signature kinds and R5.

**Size:** M
**Files:** model/framework/dsl/Action.scala; model/framework/dsl/Channel.scala; model/framework/reify/plugin/SchemaPhase.scala; model/framework/reify/plugin/ReifyPlugin.scala; focused native and compiled-client signature tests; model/README.md. A small co-located signature/value helper or narrow realization encoder extraction is allowed where existing code can be reused.
**Touches:** [model/framework/dsl/Action.scala, model/framework/dsl/Channel.scala, model/framework/dsl/InputSource.scala, model/framework/dsl/Signature.scala, model/framework/dsl/Signature.test.scala, model/framework/realize/Conversion.scala, model/framework/reify/plugin/SchemaPhase.scala, model/framework/reify/plugin/ReifyPlugin.scala, model/framework/export/ConstructedSignature.test.scala, model/framework/export/testdata/constructedSignature/**, model/README.md]

### Approach
- Re-anchor .10's committed schema API before designing consumers. Prove one compiled action/channel fixture first, including two equal-bound opaque types sharing a Finite witness and a plain Int action input. A generic converter cannot select an ambiguous witness's schema.
- Retain compiler-authored schema identity at concrete Input, Action and Channel boundaries before generic erasure. Reuse SchemaEngine.identity and schemaFor(identity). Preserve typed selection through ordinary helpers, helper jars, aliases and every action copy, including results, both input overloads, on and creates. Do not infer identities from runtime classes, values, equal bounds or map order.
- Keep contextual type policy. Old-accepted plain Int action inputs emit TypeRef.Int; plain Int channel messages refuse. A Finite.upTo catalog does not turn a declared Int input into UpTo[N]. Retain this policy explicitly without introducing a second type interpreter.
- Extend existing declaration naming hooks to retain the spelling used by the old exporter. Preserve explicit names, canonical sourceIdentity, sourceBinding, Entity.sourceName and Channel.sourceChannel ownership. Channel's raw blank-name Delivery/Loss strings are insufficient. Derive implicit action IDs and names from the retained channel declaration.
- Convert ordered Params and action fields, channel policies/messages and separately requested deliver/lose actions. Retain default/token metadata for later consumers because defaults are not fields in ir.Action. Preserve duplicate-token refusal, required versus defaulted inputs, false/zero, nested Option and structured defaults, examples and results. Reuse schema-directed value encoding in realize/Conversion.scala rather than adding a second encoder; preserve existing catalog admission checks when encoding supplied values.
- Collect complete ordered type contributions using retained closure, with identical-definition deduplication and contextual refusal of conflicts or missing metadata. Signature shape collection must not enumerate Finite.values or Channel.contents. Explicit unresolved Inbox references remain unresolved and cannot be emitted as channel Definition IDs; machine state-field ownership belongs to integration.
- Freeze old-accepted source fixtures and independent pre-edit old outputs before production changes. Reuse existing bindings/channels/default cases where possible. Compare selected native actions, channels and their referenced types after stripping only positions. Pin defaults/token selection separately. Newly supported helper forms get separate native tests and no historical-parity claim.
- Keep the production signature producer unchanged. Do not activate native converters in Lift, delete reader arms, register a parallel producer, or add a new declaration registry. Update README with the exact dormant consumer boundary.

### Investigation targets
**Required:**
- model/framework/dsl/Action.scala:43 - ActionDecl, Input, copies and native token/default semantics.
- model/framework/dsl/Channel.scala:28 - retained channel ownership and synthetic actions.
- model/framework/reify/plugin/SchemaPhase.scala:91 - typed schema selection and canonical identity.
- model/framework/reify/plugin/ReifyPlugin.scala:310 - current declaration naming hooks.
- model/framework/export/Declarations.scala:81 - action/default and channel message contracts; channelOf at 1624.
- model/framework/realize/Conversion.scala:251 - schema-directed input/value encoding.
- model/framework/export/ConstructedNames.test.scala:744 - independent bindings oracle and compiled-client harness.

### Verification and boundaries
Include zero-input actions, named/token order, shared opaque witnesses in both attachment orders, aliases and bound copies, custom/restricted/empty domains, helper jars, missing/conflicting metadata, explicit/default names, timers/internal actions, entity references, examples, scalar Int action admission and Int channel refusal. Cover FIFO/unordered, reliable/lossy, duplicates, UpTo/opaque messages and two channels sharing one message type with distinct IDs. Prove shape conversion with a witness whose values throw, without using that witness for value/default admission. Compare two separate-JVM native projections for determinism.

Run new native and compiled-client suites plus affected FiniteSchema, ConstructedNames, input/default, realization-conversion and bounded-integer checks. Freeze source before final affected checks and scoped formatting. Broad gates and independent review remain at the owner-authorized source-batch boundary. No model/temporal, tools/umpire, Go, root Model/Make builds, schema/generated outputs, module-map/migration edits or sample changes. Six samples remain read-only regressions. Root retains Flow and MILESTONES.

### Planning receipt
SHORT delta research used the current schema API, local signature contract, dependency graph and active memory. Action and Channel share synthetic-action conversion and remain one M task. Original full integration dependencies remain intact. Historical memory paths are lessons, not current-defect evidence. No new library, registry, CI or generated-output scope is added.

stage: plan-review - skipped(policy: owner defers independent review to source-batch end)
stage: html-render - skipped(policy: owner focus remains framework code and bounded task tracking)
memory: bm25 (jev-unavailable(no_key)); targeted memory scout completed.
## Acceptance
- Native Action/Input/Channel conversions retain exact declaration IDs/names, ordered Params, actor/flags/results/entity bindings, channel policies and requested deliver/lose messages. Referenced type closure matches the independently frozen old-accepted fixture projection after stripping only positions.
- Exact typed schema selection survives generic/helper-jar forwarding, aliases and every action copy, including two nominal opaque types sharing one witness. Plain Int action inputs retain old ir.Int semantics; plain Int channels refuse. Missing/conflicting metadata refuses with declaration/input context.
- Required/defaulted tokens, structured and nested Option defaults, examples and input order preserve native behavior and separately pinned metadata. Schema collection stays independent of empty/restricted catalogs and does not enumerate values or channel contents; supplied-value admission retains existing checks.
- Focused fixtures cover the listed identity, ownership, copy, policy, error and default boundaries and separate-JVM deterministic output. No unresolved Inbox identity is emitted as a channel Definition ID.
- Production signature export keeps its existing producer. Reader retirement, remaining signature kinds, state-field Inbox ownership, full .3/R5 and original dependency obligations remain open. Affected frozen checks and configured scoped formatting pass; README states the dormant converter boundary.


## Done summary
Completed the isolated framework Action/Input/Channel signature source checkpoint at 26c7d076fa37b88b563b28ce918c6c42e6ec2acf. Exact typed keys, canonical and binding identities, helper jars, aliases, generic prefixes, channel forwarding, ordered messages, defaults/token metadata and contextual refusals are covered.

Frozen affected checks: 227 cases passed; configured formatting passed on ten Scala paths. Root independently ran ConstructedSignatureSuite and joined original handle 88705 with exit 0 and four passing cases. The independently retained old actions/channels and referenced type closure match after stripping only positions. Native Some(None) encoding deliberately uses the declared Option type rather than the old reader's scala.None; full old-default equality is not claimed. The old temporary framework jar is now absent; unchanged raw output, original client and fixture source retain the pre-edit provenance.

An unexpected empty-tree commit 1d6ca0e8d4 was repaired additively at 26c7d076fa without worktree writes or history rewriting. Root verified the repaired tree differs from base 139c7e185f only in the 13 approved paths (+1527/-39); production Scala grows by 395 lines (+434/-39). The incident's cause remains unestablished. No attributable command remains live.

Production export keeps its existing signature reader. Activation, remaining signature kinds, state-field Inbox ownership, full .2/.3 dependencies, complete reader retirement, full R5, held integration/Go checks and batch-end review remain open. No held source or sample was edited.

Detailed worker summary and evidence: .flow/tmp/handoffs/fn174-signatures-framework-summary.md and .flow/tmp/handoffs/fn174-signatures-framework-evidence.json.

stage: impl-review - skipped(policy: owner defers independent review to source-batch end)
stage: plan-sync - skipped(config: planSync.enabled=false)
## Evidence
- Commits: 1d6ca0e8d4bc014f3b0c94ea9e79dffb81e3bc02, 26c7d076fa37b88b563b28ce918c6c42e6ec2acf
- Tests: mise exec -- scala-cli test model/framework/export --server=false --test-only framework.export.ConstructedSignatureSuite, mise exec -- scala-cli test model/framework/dsl model/framework/realize --server=false --test-only 'framework.dsl.*', mise exec -- scala-cli test model/framework/export --server=false --test-only framework.export.FiniteSchemaSuite, mise exec -- scala-cli test model/framework/export --server=false --test-only 'framework.*.Reify*', mise exec -- scala-cli test model/framework/export --server=false --test-only framework.export.ConstructedNamesSuite, mise exec -- scala-cli test model/framework/export --server=false --test-only framework.export.InputDefaultsTest, mise exec -- scala-cli test model/framework/export --server=false --test-only framework.export.BoundedIntegersTest, mise exec -- scala-cli test model/framework/dsl model/framework/realize --server=false --test-only 'framework.realize.*', mise exec -- scala-cli fmt model/framework/dsl/Action.scala model/framework/dsl/Channel.scala model/framework/dsl/Signature.scala model/framework/dsl/Signature.test.scala model/framework/realize/Conversion.scala model/framework/reify/plugin/SchemaPhase.scala model/framework/reify/plugin/ReifyPlugin.scala model/framework/export/ConstructedSignature.test.scala model/framework/export/testdata/constructedSignature/Shapes.scala model/framework/export/testdata/constructedSignature/Native.scala --scalafmt-conf model/.scalafmt.conf --check, git diff --check, git diff --cached --check
- PRs: