---
satisfies: [R1, R2, R3, R4, R6, R9]
---
# fn-174-the-model-constructs-its-own-ir-dsl.23 Capture Machine evidence and construct complete isolated contributions

## Description
Construct complete isolated Machine evidence contributions after the committed rule-aware derivation checkpoint. Retain explicit evidence at its definition through the compiler plugin, complete omitted fieldless cases through shared IR-only lowering, and forward original evidence through native Derived/Built values. The result contains the evidence root, reached Functions/Types and external references. It is an independently usable component of full .4, whose production assembly and original acceptance remain open.

**Size:** M. This adapts endpoint capture and dependency closure for one complete behavioral contribution. Monitors, refinement/assuming and model-wide Machine assembly have separate ownership/state seams and remain subsequent source work.
**Files:** `model/framework/dsl/Machine.scala`, a private evidence contribution beside `MachineEndpoints.scala`, a shared evidence lowerer beside `RuleLowering.scala`, `model/framework/reify/plugin/ReifyPlugin.scala`, `model/framework/export/Declarations.scala`, focused DSL tests, `model/framework/export/ConstructedEvidence.test.scala`, its new standalone fixture directory, and `model/README.md`. Shared body/closure files may change only when the exact evidence contract requires it.
**Touches:** [model/framework/dsl/**, model/framework/reify/**, model/framework/export/Declarations.scala, model/framework/export/ConstructedEvidence.test.scala, model/framework/export/testdata/constructedEvidence/**, model/README.md]

### Approach

1. Freeze independent plugin-free old output and actual refusal behavior for fresh standalone evidence fixtures before changing production. Reuse the complete old-Model/saved-tree replay pattern in `ConstructedDerivations.test.scala`; retain exact source, commands and qualification of old-refused spellings. Establish a genuine semantic RED against the unchanged prior implementation. Existing original fixtures, samples, baselines and the .22 correction ledger stay immutable.
2. Extract the existing default evidence and direct-match completion algorithm at `export/Declarations.scala:1186-1288` into one IR-only helper used by both reader and constructed paths. Preserve the reader's actual conditional admission, ordering and positions. No second expression translator, new public spelling, generic validation framework or new dependency.
3. Follow `MachineEndpoints.scala:15,62` and `ReifyPlugin.scala:1437,1459,1571` for compiler-owned capture and exact live source association. Reuse the established body engine (`reify/body/Expressions.scala:103,416`) and reached closure checks (`CoreBindings.scala:211`). Read selected Fact schema metadata using existing header/Finite seams without constructing unrelated state headers or resolving nested Inbox suppliers.
4. Forward source evidence through the existing Machine Derived/Built boundaries (`Machine.scala:883,935`). Keep the declaring root/function/position when inherited, and compare repeated source/derived/source and A/B/A collections. New evidence collection must not acquire native evidence getters, catalogs, endpoints, RuleBooks, refinement modules or execute any callback. Preserve ordinary native construction behavior separately.
5. Add focused malformed-metadata controls and a standalone `ConstructedEvidenceSuite`. Document the usable private capability and still-open monitor/refinement/complete-assembly obligations in README after the existing derivations subsection. Do not edit the six samples, old test cases or original oracle bytes.

### Investigation targets

**Required before production changes:**
- `model/framework/export/Declarations.scala:342` and `:1186` - actual authored/default evidence selection, coverage, ordering and positions; include the final Nothing/default branch around `:516`.
- `model/framework/dsl/MachineEndpoints.scala:15` and `model/framework/dsl/MachineHeaders.scala:63` - live source seals, immutable contributions and selected Fact schema.
- `model/framework/reify/plugin/ReifyPlugin.scala:1437` - endpoint capture/retention, inherited owner handling and located refusal patterns.
- `model/framework/dsl/Machine.scala:883` - Derived/Built forwarding and native initialization timing.
- `model/framework/export/ConstructedDerivations.test.scala` and `model/framework/export/testdata/constructedDerivations/primitive-state-corrections.json` - standalone complete-old authority, semantic/position comparison, saved-tree replay and narrowly fixed primitive correction scope.
- `model/framework/export/testdata/lifts/Declarations.scala:29`, `Captured.scala:65`, `Derived.scala:92` and `Rejects.scala:314` - independent existing ordinary Function evidence, defaults, inherited evidence and refusal spellings; these are read-only authorities.

### Key context

- Authored evidence is not restricted to `PartialFunction[Fact, String]`. The reader admits ordinary `Fact => String` evidence and named helpers. Preserve exact root and parameter names and reachable helper/Type/Channel/Hole closure.
- Coverage/default completion runs only for the reader's recognized single-parameter direct Match on that parameter. Other explicit bodies pass through unchanged. Do not impose new enum, coverage or totality checks on those bodies.
- Omitted evidence uses fieldless Fact cases in enum declaration order. Field-bearing cases require the historically admitted explicit coverage; guarded cases remain partial. Preserve wildcard, bind, alternatives, constructor patterns and authored overlap order. Defaults insert after lines for the same case and before the first line for a later case when possible.
- Nothing suppresses omitted/default evidence. Freeze the reader's actual behavior for explicitly authored Nothing evidence instead of assuming every Nothing Machine has no evidence function.
- Explicit root positions come from the expression; inserted defaults use the evidence declaration; omitted defaults use the Machine declaration. Strip Position alone for semantic comparison, then check positions independently. Preserve measured old SourceFile qualifications rather than erasing positions globally.
- Keep the .21 contextual shared-helper combined-model qualification and .22 exact 79-slot primitive correction ledger unchanged. Isolated evidence equality cannot prove model-wide helper reconciliation, native-vs-IR execution equivalence, full .4 or R5.
- Prior memory on transition-table reads, paired-level validation, evidence payloads and authority reconciliation supplies verification principles, not new Machine API requirements.

### Measured source-position boundary

The same immutable preproduction helper/client/framework jars and classpath admit two complete old Models when only the helper/framework source-mapping order changes. Both lifts exit0. One full JSON is byte-identical to `original-jar-External.json`; the other is byte-identical to `original-External.json`. Position-only removal makes their semantic JSON identical. `inspect/Index.scala` overwrites source prefixes by raw SourceFile path in consumer order. A declaring compiler run cannot predict that future source-index order.

The preserved proof is `.flow/tmp/fn174-evidence-position-boundary.md`, with exact commands, full Models and unchanged input hashes under `.flow/tmp/fn174-evidence-build.H48VyK/position-probe2/`. Probe1's omitted source mapping refuses the root and does not prove two admitted contexts. Variant3-6 failures remain diagnostic evidence; none is a complete green suite.

Retain lossless definition-owned positions in canonical `MachineEvidence.collect(machine)` and attach immutable compiler-owned `SourceOrigin(canonicalFile, rawSourceFile)` provenance. Add a private IR-only `MachineEvidence.project(contribution, SourceIndex)` seam. SourceIndex carries ordered raw-source resolutions, with last-writer order matching the existing Index. Projection changes Position fields only. An available same-file resolution keeps exact coordinates; an explicitly unavailable source clears Position. Missing/stale/conflicting origin or an available different-owner resolution refuses before publishing a projected result. Projection performs no filesystem access, native callbacks or global registration. No source filename, run shape or fixture variant is inferred.

The harness derives contexts from the actual existing Inspector Index over exact artifact/input orders and resolves source availability before supplying immutable data. It must never derive projection maps from old expected JSON. Keep all81 frozen old Models, refusal ledger, seven original author/driver files and Native probe unchanged. Add a separate typed context probe that projects the same captured contribution under both independently pinned admitted orders in two JVMs, while separately verifying canonical definition positions. Main/unit/jar/edit comparisons retain strict semantic and position equality after invoking the production projection seam under their actual source-index contexts. Assert missing/stale/conflicting contexts refuse, valid unused provenance remains bounded, repeated projection is immutable and native acquisition counters stay unchanged. Full model assembly, global source-index integration, available conflicting-owner reconciliation and production activation remain in full .4. Existing .21 contextual qualification remains unchanged.

### Quick commands

Run serially because compiler outputs are shared. Use the already documented task-private bwrap /tmp and cache overlays if the host remains read-only; preserve all lower caches and artifacts. No dependency, pinned Scala or test selection changes to work around environment failures.

```bash
mise exec -- scala-cli test model/framework/dsl model/framework/realize --server=false
mise exec -- scala-cli test model/framework/export --server=false --test-only framework.export.ConstructedEvidenceSuite
mise exec -- scala-cli test model/framework/export --server=false --test-only framework.export.ConstructedHeadersSuite
mise exec -- scala-cli test model/framework/export --server=false --test-only framework.export.ConstructedEndpointsSuite
mise exec -- scala-cli test model/framework/export --server=false --test-only framework.export.ConstructedDerivationsSuite
mise exec -- scala-cli test model/framework/export --server=false --test-only framework.export.ReifyBoundarySuite
mise exec -- scala-cli test model/framework/export --server=false --test-only framework.export.ReifySamplesSuite
```

Run `ConstructedBindingsSuite`, `ConstructedRulesSuite`, `ConstructedFacetsSuite` and `FiniteSchemaSuite` when shared plugin/body/closure paths change. Run configured scalafmt on the exact owned non-frozen Scala paths. Do not run `Fixtures`, `Observations`, `ModelBuild`, `FixtureBuild`, root Model/Make/Go/integration gates or claim their credit. The owner holds them; record the deferral explicitly. No review verdict is earned here; independent review remains at source-batch end.

### Boundaries

Framework source, new focused fixtures/tests and README only. Root owns task state, MILESTONES and closure. No tools/umpire, model/temporal, Go, schema/proto/generated output, registry/module-map, migration/regeneration, SEMANTICS or design-document changes. Do not activate a second producer, switch signatures or output, retire the reader/public Bindings, bypass full .1/.3 dependencies or claim complete Machine/R1/R3/R4/R5 acceptance. Remaining passive monitor declarations/attachments, refinement/assuming/ordered operations, nested/channel/ownership validation, contextual-helper/source-index reconciliation, complete graph assembly and full integration stay open in .4.
## Acceptance
- [ ] Construct the complete isolated evidence root plus all reached Functions, Types and external references for default, authored direct-match and ordinary non-match Function evidence, using one shared body/default lowerer. Preserve exact names, parameter spelling, types, contracts, expression shape, declaration-order default insertion and authored overlap/guard semantics. Freeze explicit Nothing behavior independently; omitted Nothing evidence stays omitted. Cover fieldless/parameterized facts, mixed coverage, wildcard/bind/alternatives/guards, non-match pass-through and historically refused step-producing evidence dependencies without adding new author restrictions.
- [ ] Retain exact compiler-owned evidence provenance and selected live Fact schema through direct declarations, inherited/Derived/Built values and representative restrict/unmonitored/mixed .22 histories. Reject missing/stale/copied/conflicting/malformed or incorrectly typed metadata, root/closure/name/type/call/lexical violations and wrong live declaring sources before publishing any contribution. Include valid multi-root reuse and conflicting controls. Do not broaden completion/totality validation beyond the reader's exact recognized Match form.
- [ ] Conversion and repeated A/B/A and source/derived/source collection are immutable and add zero evidence getter/callback/catalog/endpoint/rule-table/refinement reads or evaluations after ordinary native construction. Assert separate acquisition/conversion counters. Same-unit, allowed cross-file stored helpers and precompiled helper jars retain definition-owned bodies and full dependency closure; opaque or unmarked foreign helper bodies retain located refusals. Warm body-only helper edit/restore changes the dependent evidence or refusal.
- [ ] A new standalone ConstructedEvidenceSuite compares against independently frozen complete plugin-free old Models and replays their saved tree authority. Establish a genuine semantic RED before production changes. Compare semantic messages with Position alone removed, verify explicit/default/inserted positions separately, and require two native JVM runs with deterministic output. Keep original test/source/oracle/sample bytes, .21 combined-helper qualifications and .22 primitive correction/native authority unchanged; record actual old-refused controls without substituting prospective acceptance for parity.
- [ ] All affected Quick suites and exact-path configured formatting pass on the final source; README records the staged evidence capability and still-open complete assembly/monitor/refinement/model-wide ownership obligations. Preserve every original full .1/.2/.3/.4 acceptance and dependency. Record held Model/Make/Temporal/Go/R5 gates and source-batch review as deferred, never passed. Hand back source/compiler/index ownership with every launched handle joined and exact verification commands/results.
- [ ] Preserve lossless canonical definition positions and compiler-owned raw SourceFile origins. A real private IR-only projection consumes explicit immutable ordered source-index context derived from actual artifact/input mappings, never expected output. Strictly reproduce both independently admitted same-artifact position results and main/unit/jar/edited results through that production seam, while independently checking canonical positions and retaining all frozen Models, refusal/author/Native bytes. Test contextual refusal for missing/stale/conflicting origins or available different owners, deterministic two-JVM output, projection/source/Built/A-B-A immutability and zero extra native reads. Clear only explicitly unavailable positions; no blanket erasure, filesystem access, global registry, filename/run-shape heuristics or .21 qualification changes. Full .4 still owns complete-model source-index integration and activation.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
