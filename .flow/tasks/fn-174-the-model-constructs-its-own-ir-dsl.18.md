---
satisfies: [R3, R4, R9]
---
# fn-174-the-model-constructs-its-own-ir-dsl.18 Preserve complete Finite schemas for role-bearing enum cases

## Description
Repair schema attachment for compiler-generated Finite witnesses of role-bearing enum singleton cases (R3/R4/R9). The current plugin rejects ordinary derived Phase catalogs before init/end capture. This source prerequisite restores the existing author spelling and keeps endpoint construction with subsequent Machine work.

**Size:** M
**Files:** `model/framework/reify/plugin/SchemaPhase.scala`, `model/framework/export/FiniteSchema.test.scala`, focused fixture files if needed, `model/README.md`.
**Touches:** [model/framework/reify/plugin/SchemaPhase.scala, model/framework/export/FiniteSchema.test.scala, model/framework/export/testdata/finiteSchema/**, model/README.md]

### Approach

- Start with the retained diagnostic sources under `/tmp/fn174-endpoint-tree.5fxHNt` and `.flow/tmp/handoffs/fn174-endpoint-planning-discovery.md`. Add compiled semantic regression cases before changing production. Preserve failure logs and verify each negative control failed for its asserted semantic reason, separately from setup failures.
- Follow `SchemaPhase.scala:323` and `:505`. Recognize actual enum-case singleton witness identity before widening/dealiasing loses the enum-case symbol on a role-bearing intersection. Preserve the existing exclusion of internal singleton pieces of automatic enum derivation. Keep attachment for the complete parent enum and enclosing records, selected Machine witnesses, custom values-only witnesses and helper-jar views.
- Reuse schema construction and role validation in `SchemaEngine` at `SchemaPhase.scala:654` and `reify/body/Types.scala:131`. Change the classification seam, not the author Model. Do not skip Phased/Machine forms, packages, tests, all intersections or errors generally; do not synthesize a schema for an unnamed intersection or swallow a lift failure. Unsupported non-enum intersections remain contextual refusals.
- Extend `FiniteSchemaSuite`'s strict compiled old/native and helper-jar controls. Before production edits retain complete old-reader output for an old-admitted role-bearing enum, containing record and Machine header. Exercise plain enum cases and singleton cases with Waiting, Closed/Succeeded/Failed or inherited/custom roles, aliases, nested state records, direct default Phased end, explicit override and concrete Derived declarations. Compare complete parent enum/record messages and the five-field Machine header/type projection with independent old-reader output; only Position may be stripped. Keep direct role-invalid cases' located old/native refusal behavior. An enum with no Closed case remains a valid schema. Field-bearing role cases with finite fields remain valid enum schemas, while the set-of-role-values consumer retains its refusal. Missing Closed and field-bearing default-end behavior must be classified at their actual current capture boundary; this task adds no endpoint capture or runtime conversion.
- Retain native catalog order, role case identity, original witness references, custom/empty/throwing catalog behavior and ordinary lazy initialization. Two JVM runs and old-reader replay of plugin-saved trees check structure and determinism; they do not establish complete Machine construction. Include same-unit and jar compilation boundaries so the fix is not dependent on inspecting foreign source trees.
- Run affected direct Scala suites and configured formatting serially on frozen source. Update README with the enum singleton repair and unchanged endpoint/integration boundary. Keep six samples as unchanged read-only regression inputs and preserve all earlier captured source/raw/jar artifacts.

### Investigation targets

**Required** (read before coding):
- `model/framework/reify/plugin/SchemaPhase.scala:210-345` and `:505-535` for general Finite attachment and concrete selection.
- `model/framework/reify/plugin/SchemaPhase.scala:654-725` and `model/framework/reify/body/Types.scala:15-145` for widening, schema identity and complete enum declaration.
- `model/framework/dsl/Domain.scala:90-126` for automatic Mirror-based enum case witnesses.
- `model/framework/reify/body/Roles.scala` for role conflict/field-bearing refusal rules.
- `model/framework/export/FiniteSchema.test.scala` for strict standalone compilation, old/native type comparison, helper jars and native timing.

**Optional**:
- `model/framework/export/ConstructedHeaders.test.scala` and `testdata/defaultEnds/Ends.scala` for unchanged Machine header projection and direct Phased/Derived spellings.

### Resolved planning evidence

At committed .17 closure `d63be555ec21b87cf1b463d15aa239e916bc2354`, exact full native probe compilation passed without the plugin (handle 99959). Plugin compilation of both default and explicit phased Machines failed with `<none> is neither an enum nor a case class` (12865, 76973). Renaming State to Snapshot did not affect the failure. Minimal RoleFinite.scala, containing only role-bearing enum cases and `summon[Finite[Phase]]`, failed identically (4880); PlainFinite.scala with plain cases passed the same plugin and strict compiler settings (60636). The pre-pickler tree contains synthetic Finite witnesses of singleton `Phase & Closed` types. The NoSymbol predicate explanation is a source-supported inference until implementation instruments or otherwise demonstrates it.

The separately compiled support base and inherited initialized val/getter client passed safe-init and runtime assertions (29415, 1772). Native construction read init once and header conversion added zero reads. These controls establish a future native-only timing-fixture boundary, not endpoint metadata or old admission. Setup failures 76734 and 38847 remain separately classified. All original handles joined; no command remains live.

### Boundaries and dependencies

Follow committed .17. Add this prerequisite to full .3 and full .4, preserving full .3's full-.2/.10-.15 dependencies and full .4's full-.1/full-.3/.16/.17 dependencies. All other edges remain, including fn-169.2's full-.2 prerequisite.

Hold `model/temporal`, `tools/umpire`, Go/integration, root Model/Make builds, schema/proto/generated outputs, coverage registry/module map/migration edits and regeneration. No new library or author API, no sample edits, production activation, reader retirement or duplicate producer. Owner design edits and all retained proof inputs remain unchanged. Independent review and broad gates run at source-batch end; explicit review=none for this unit earns no SHIP verdict.

Complete Machine init/end/evidence, ordinary Rules lowering, monitors/refinement, automatic nested witness propagation, complete derived/channel validation, consumer integration, sole production construction and R5 remain with the original integration tasks. This repair releases only further source preparation.

### Quick commands

```bash
mise exec -- scala-cli test model/framework/export --server=false --test-only framework.export.FiniteSchemaSuite
mise exec -- scala-cli test model/framework/export --server=false --test-only framework.export.ConstructedHeadersSuite
mise exec -- scala-cli test model/framework/export --server=false --test-only framework.export.ConstructedActorsSuite
mise exec -- scala-cli test model/framework/export --server=false --test-only framework.export.ConstructedSignatureSuite
mise exec -- scala-cli test model/framework/export --server=false --test-only framework.export.ConstructedInboxesSuite
```

Baseline the existing focused suites before edits and run configured formatting on changed Scala files. Do not run root Make or held DefaultEnds fixture builds; compiled standalone controls must use the existing framework/IR classpath, not a rebuilt Model jar.

## Acceptance
- [ ] Strict compiled regression proves the role-bearing enum failure before the production repair, then ordinary derived enum/record and direct default/explicit Phased/Derived author spellings compile with the plugin. Internal singleton case witnesses use actual enum-case identity; no new author syntax, blanket intersection exemption or swallowed failure is introduced.
- [ ] Complete retained parent enum/record schemas match independently obtained old type messages after Position-only removal. Case/field/name ordering and role identities remain exact. Two native JVMs, helper-jar boundaries and current-reader replay of plugin-saved trees agree; complete Machine and endpoint construction remain unclaimed.
- [ ] Invalid role combinations, unsupported non-enum/intersection schemas and prior scalar/opaque/recursive/Inbox/name-conflict boundaries retain located contextual refusals at their actual capture stages. Empty/throwing catalogs, native case and witness references and lazy evaluation remain unchanged. Enums without Closed and finite field-bearing role cases retain complete schemas; missing Closed and field-bearing role-set refusals remain with their existing consumers, with explicit default-end stage qualifications.
- [ ] Affected focused suites and configured formatting pass on frozen source; all owned process handles join. README records the repair and held endpoint/integration obligations. Only authorized framework/tests/README change; six samples, owner design edits and earlier immutable proof inputs remain unchanged. Production activation, reader retirement, full .1/.2/.3/.4/R5 and source-batch review stay open.


## Done summary
Closes .18 role-bearing enum Finite schema source repair at `385ebc0ab89c013102d533e254b1a93fa8b9bb6a`, tree `865556885e837a4f685b12f8d667f56baed3f53e`, sole parent `8582698d860670c8eb112d2c1d5048730fde8fc5`. The two added production lines exclude actual enum-case singleton witnesses by their term symbol while preserving all prior concrete-type exclusions. Parent enum/record schemas, role validation and original author spellings remain unchanged.

The worker passed 28 scoped cases. Final FiniteSchemaSuite passed 12 cases after isolating five native negative controls from Machine header attachment. The other four suites passed 4 Headers, 3 Actors, 4 Signature and 5 Inbox cases before that final test-only isolation; production, fixtures and README were byte-identical, while their test-source hash was `105791f16dba821b8e817a1b3aa74ae482ed7cbc93904e8b94afd25faca861dd`. Final test hash is `e444f78df026e9ee386a4ba452b3812409c438cf4504db972c3f82b7bfe6f98e`. This source qualification is preserved rather than claiming those four suites ran on the final test bytes.

Root independently reran all 12 final FiniteSchema cases on the committed live checkout. Original handle 83889 joined exit 0; log `/tmp/fn174-role-root-finite.log`, retained artifacts `.flow/tmp/finite-schema.14565763538415071319`. Complete fixed old Types and five-field header projections, same-unit/helper-jar native runs in two JVMs, saved-tree replay, standalone role/domain refusals, native catalogs/timing, typed opaque selections, Inbox closure and unchanged six-sample compilation pass. Root formatting on seven Scala files passed with handle 84206 joined exit 0, and whitespace checks passed.

Root verified all ten committed blobs and live files against manifest `5d9e977182d9d25d93ab5c30ca4bf0ecd887f4314992c7b2f30babaafc6d1940`, exact parent/tree/scope, all 9,320 parent paths preserved, empty index and both unchanged owner hashes. Fixed whole original/helper JSON hashes remain `11b5805acfa5312e25c5f0f40b97c2b81ac7f1e2db6ece05129efcfc01a69a7f` and `f6c3f5fd133662b7d68ab7099773a95a10662a913daef700734c7e21e5e09345`, identical to the retained pre-production capture `.flow/tmp/finite-schema.17182105448641695588`. Root also verified original source/jar/framework jar and retained .17 plugin hashes. All worker handles are joined and its writer/compiler lane is terminal.

The minimal RoleFinite source measured a genuine pre-repair error and passes unchanged after repair; the plain enum control passes. Diagnostic instrumentation measured absent role-singleton typeSymbol and an actual Enum/Case termSymbol, then was removed. The original safe-init, discarded-value and ambiguous-import setup failures remain separately classified in the worker handoff. No setup failure supplies defect proof.

Enums without Closed and finite field-bearing role cases retain valid complete schemas. Missing Closed remains the old default-end and first native default-end evaluation refusal. FieldDefault retains the old field-bearing role-set refusal while native role sets enumerate finite values. Original FieldConsumer `.in[Closed]` old reading stops earlier on unindexed jar sugar; its native behavior is tested separately, with no role-set parity claim. This task captures no endpoints and constructs no complete Machine.

Full .3 retains full .2, .10-.15 and .18. Full .4 retains full .1, full .3, .16, .17 and .18. Fn-169.2 still requires full .2. Complete signature/Machine production, init/end/evidence, ordinary Rules, monitors/refinement, nested-witness propagation, complete derived/channel validation, G01, consumers, activation, reader retirement, full R1/R3/R4/R5 and held system/Go equivalence remain open. The six samples and owner edits remain unchanged. Read-only gate classification was FULL/exit 1; the held root Model/Make, Temporal/tools/Go/integration/schema/generated/registry/module-map/migration/regeneration gates were not run or counted as passed. Independent review and broad checks remain at source-batch end.

stage: impl-review - skipped(policy: owner source-batch cadence; explicit review=none, configured codex backend unchanged; no SHIP verdict)

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 385ebc0ab89c013102d533e254b1a93fa8b9bb6a
- Tests: mise exec -- scala-cli test model/framework/export --server=false --test-only framework.export.FiniteSchemaSuite, mise exec -- scala-cli test model/framework/export --server=false --test-only framework.export.ConstructedHeadersSuite, mise exec -- scala-cli test model/framework/export --server=false --test-only framework.export.ConstructedActorsSuite, mise exec -- scala-cli test model/framework/export --server=false --test-only framework.export.ConstructedSignatureSuite, mise exec -- scala-cli test model/framework/export --server=false --test-only framework.export.ConstructedInboxesSuite, mise exec -- scala-cli fmt --scalafmt-conf model/.scalafmt.conf --check model/framework/reify/plugin/SchemaPhase.scala model/framework/export/FiniteSchema.test.scala model/framework/export/testdata/finiteSchema/Roles.scala model/framework/export/testdata/finiteSchema/Native.scala model/framework/export/testdata/finiteSchema/Helpers.scala model/framework/export/testdata/finiteSchema/Client.scala model/framework/export/testdata/finiteSchema/HelperNative.scala, sha256sum -c .flow/tmp/handoffs/fn174-role-schema-source-manifest.sha256, git diff --check
- PRs: