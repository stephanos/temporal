---
satisfies: [R3, R4, R6, R9]
---
# fn-174-the-model-constructs-its-own-ir-dsl.17 Retain typed Machine header provenance and construct isolated header/type contributions

## Description
After the committed .16 closure, retain Machine declaration identity and exact S/O/F witness selections and construct a private, explicitly partial header/type contribution (R3/R4/R6/R9). This is source preparation for full .4, with no production producer switch.

**Size:** M
**Files:** `model/framework/dsl/Machine.scala`, an adjacent private header metadata/contribution file, `model/framework/reify/plugin/SchemaPhase.scala`, focused DSL/export tests and fixtures, `model/README.md`. Change shared schema/name helpers only where required by this contribution.
**Touches:** [model/framework/dsl/**, model/framework/reify/plugin/SchemaPhase.scala, model/framework/export/ConstructedHeaders.test.scala, model/framework/export/testdata/constructedHeaders/**, model/README.md]

### Approach

- Before production edits, retain an independent complete old-reader fixture output and run disposable compiler-tree/native timing controls using the current source. Cover direct and Derived objects, empty bindings, concrete generic bases, opaque aliases sharing one witness and helper jars. Record actual old admissions/refusals; helper/class/local/anonymous forms without old admission remain qualified native controls.
- Follow post-pickler template attachment at `SchemaPhase.scala:35` for private compiler metadata. Read the declaring module's concrete Machine base type for canonical S/O/F selection keys and its package/module spelling. Retain each selected structural schema on the actual corresponding witness when needed, following existing typed-consumer attachment, rather than assuming a forwarded Given already installed every view. Preserve saved TASTy, native getter behavior and the exact `fs`/`fo`/`ff` references at `Machine.scala:481`. Do not perform another implicit search, use JVM naming or choose an arbitrary retained schema.
- Keep declaration identity separate from inherited type selections. A concrete Derived object owns its family/name; native Built intermediates forward selections without inventing declaration identities. Metadata attachment is idempotent for identical evidence and refuses conflicting evidence. Early conversion before registration refuses contextually.
- Construct a dedicated partial contribution with only family, name, stateType, outcomeType, factType and complete selected type messages. Match `Declarations.scala:326` exactly, including `.getNamed` scalar behavior and omitted Nothing facts. Do not return a partially filled `ir.Machine` as a complete message or sample a runtime state to manufacture init/end IR.
- Reuse exact `schemaFor` selection in `Domain.scala:23` and `Signature.stateSchema/stateTypes` at `Signature.scala:34`. Preserve ordinary nested records and supported direct Inbox ownership. Nested Inbox closure requires exact privately supplied witness/key evidence; unresolved ownership refuses with Machine/type/field context. Do not add an author API, global registry, catalog inference or automatic summoning. Automatic retention of all nested witnesses remains an explicit full .4 obligation.
- Conversion adds no init/end/rules/table/phase/refinement getter reads, callbacks or catalog enumeration. Separate ordinary construction evaluation from conversion's incremental evaluation in counters, including existing Built init/refinement behavior.
- Extend established unit tests and the compiled old/native oracle pattern from `ConstructedBindings.test.scala`. Preserve whole raw old output and explicitly project the tested fields plus all selected type messages. Normalize only Position; no name/type/body normalization. Run two native JVMs and old-reader replay on plugin-compiled trees.
- Update the README with the exact partial capability, nested ownership boundary and remaining full .4 work. Keep all existing source-checkpoint qualifications and unchanged six-sample regressions.

### Investigation targets

**Required** (read before coding):
- `model/framework/dsl/Machine.scala:481-710` for native witnesses and Derived/Built forwarding.
- `model/framework/reify/plugin/SchemaPhase.scala:35-110` for post-pickler registration and typed consumers.
- `model/framework/dsl/Domain.scala:19-65` and `model/framework/dsl/Signature.scala:34-150` for keyed schemas and exact Inbox closure.
- `model/framework/export/Declarations.scala:310-355` and `:1393` for historical header projection and derived ownership.
- `model/framework/export/ConstructedBindings.test.scala` for independent old output, native JVMs and saved-tree replay.

**Optional**:
- `model/framework/export/FiniteSchema.test.scala:370` and `testdata/constructedInboxes/NativeBaseline.scala:21` for shared opaque witnesses and explicit nested ownership.

### Boundaries and dependencies

The task follows .16 only. Add it as a prerequisite of full .4 while retaining full .1, full .3 and .16 there. Preserve every other edge, including fn-169.2's full-.2 prerequisite and conductor-owned cross-spec checks.

Hold `model/temporal`, `tools/umpire`, Go/integration, root Model/Make builds, schema/proto/generated outputs, registry/module-map/migration edits and regeneration. Six samples are read-only regression inputs. No new library, production activation, reader retirement or duplicate producer. Preserve owner design edits and every retained old source/raw/jar proof. Independent review and broad gates run at source-batch end; do not claim SHIP.

Full .4 retains complete Machine headers/init/end/evidence, ordinary Rules capture/lowering, monitors, refinement, complete derivations, automatic supported nested-witness propagation, G01 delivery/loss checks and consumer integration. Full .1/.2/.3/.4, R1/R5, mini/Temporal/system/Go equivalence and all held acceptance remain open.

### Resolved planning proof

Disposable pre-change sources and logs are under `/tmp/fn174-header-tree.Sl98d8`. Generic.scala compiled into a separate jar, then Headers.scala/Probe.scala compiled and ran on Scala 3.9.0/JVM 27 against the committed .16 framework/plugin jars. Both original handles joined exit 0 (34413 and 68388). `client.log` contains the post-pickler `reifySignature` trees. Direct alias, jar-generic base, Derived and equal-bound opaque Machine triples retain concrete types and the original constructor witness expressions. Native observations were direct/inherited witness equality false, direct/Derived true, opaque state/shared true and opaque outcome/shared true. The opaque witness's unselected schema exposed only Left, despite the saved Right Given attachment. This requires explicit typed-consumer retention/selection and rules out `.schema` inference. Witness reads did not call the opaque Machine's throwing init/end. These discovery controls do not prove new metadata attachment, historical reader admission or full header conversion; the worker's independent old fixture and red/green controls remain required.

### Quick commands

Run compilers serially. If host scratch is insufficient, use a disposable source-identical temporary snapshot and record that qualification; preserve existing proof classpaths.

```bash
mise exec -- scala-cli test model/framework/dsl model/framework/realize --server=false
mise exec -- scala-cli test model/framework/export --server=false --test-only framework.export.ConstructedHeadersSuite
mise exec -- scala-cli test model/framework/export --server=false --test-only framework.export.FiniteSchemaSuite
mise exec -- scala-cli test model/framework/export --server=false --test-only framework.export.ConstructedInboxesSuite
mise exec -- scala-cli test model/framework/export --server=false --test-only framework.export.ConstructedBindingsSuite
```

The new header suite is created by this task. Baseline the other affected suites before edits and run configured formatting on changed Scala files. Scope fixture coverage rather than treating existing tests as proof of complete Machine construction.

## Acceptance
- [ ] Independent pre-change complete old output is retained. Exact five-field header projection and complete selected type messages agree with native contributions after removing Position only; two native JVMs agree and plugin-compiled saved trees replay unchanged through the old reader. Named/scalar/Boolean/Nothing/opaque/alias and derived cases retain their actual admission qualifications.
- [ ] Compiler-owned declaration identity and exact S/O/F schema keys use the original witnesses, including shared equal-bound opaque witnesses and empty-binding Machines. Concrete Derived objects own their names while preserving source selections; bare Built/class/local/helper forms have explicit provenance/admission classifications. Native names, getters, references, initialization and witness identity remain unchanged.
- [ ] Missing/premature/conflicting identity or selection, wrong/absent schema keys and incomplete/conflicting type closure refuse with Machine/type context and no published partial result. Direct Inbox ownership and nested ordinary records resolve; nested Inbox closure resolves only with exact selected nested evidence or refuses with field context. No guessing, catalogs, registry or new author spelling is used.
- [ ] Conversion and metadata retention add no init/end/rules/table/phase/refinement, guard/effect or catalog evaluation. Throwing/empty catalogs, throwing getter defs, repeat conversion and helper-jar counters distinguish existing native construction behavior from incremental conversion behavior. All owned command handles are joined and affected focused suites plus configured formatting pass.
- [ ] README describes the isolated contribution and preserved full Machine/nested ownership obligations. Only authorized framework sources/tests/README change; samples, owner dirt and retained proofs remain unchanged. Full .4 and original integration dependencies, activation/retirement/R5 and deferred batch-end checks remain open.


## Done summary
Closes .17 isolated Machine header/type source preparation at `cb871e440656572adfd86e9a14958b3f9679bda5`, parent `7a769a651e29566b33584eb36536dc56117a2976`, tree `22bd5d445573cb496c605422b490a3b55da62dc8`. Private five-field contributions retain compiler-owned family/name and exact S/O/F selections on original witnesses. Scalar blanks and omitted Nothing facts preserve historical projection. Derived declarations own names; Built values forward selections without declaration identity.

The worker's frozen source checks passed 232 cases (210 DSL/realize, four headers, eight finite schema, five Inbox, five bindings), with configured formatting on 12 Scala files. Root independently reran nine MachineHeaders cases, four compiled header cases and three Actor cases; original handles 59805, 79917 and 18376 all joined exit 0. Root compiled artifacts are `.flow/tmp/constructed-headers.9378358983584697604` and `.flow/tmp/constructed-actors.7994341751838866078`. The compiled checks preserve whole old output, position-only comparison, two-JVM equality, saved-tree replay, helper jars, native timing and actual refusal qualifications.

Root verified all 15 committed blobs and live files against the frozen manifest `081f4253d60e47af2ec79e0e8f4570f5a3d870cbb5d5a8e2dbf6040f8cb4c204`, exact commit parent/tree/scope, empty index, both owner dirty hashes and unchanged independent old source/framework jar/original jar/raw hashes. The successful old oracle remains `.flow/tmp/constructed-headers.11348373142731438684`. Earlier refused discovery captures and parser/import/safe-init setup failures remain retained and separately classified. Compiled missing-registration, malformed-type and wrong-scalar-key controls have genuine semantic reds; selected-witness discoveries were green and supply no missing-key red claim.

Missing, premature or conflicting provenance, wrong selections, missing roots/field references/shapes and incomplete/conflicting closure refuse contextually. Conversion reads no public names, init/end, rules/table, phase/refinement, guards/effects or catalogs. Ordinary Built construction retains its existing init/refinement reads. Nested Inbox ownership requires exact private nested witness/key evidence; automatic supported nested propagation remains full .4 work. UpTo/opaque states, shared opaque outcome and counter-augmented helper forms retain actual old-refusal qualifications.

Production Scala delta is +197/-11, net 186 across three files, excluding tests/fixtures/README. The collector emits no complete Machine and activates no producer. Full .1/.2/.3/.4, init/end/evidence, ordinary Rules, monitors/refinement, complete derivations, G01, consumer fairness/integration, production activation, reader retirement, R5/system/Go equivalence and independent source-batch review remain open. Full .4 retains full .1, full .3, .16 and .17; fn-169.2 still requires full .2. The six samples remain unchanged regression inputs. The read-only gate classification returned FULL/exit 1 for README; the held Make/integration gate was not run or counted as passed. Review is deferred by the owner source-batch policy, with explicit review=none for this source unit; the configured backend was not treated as none and no SHIP verdict was issued.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: cb871e440656572adfd86e9a14958b3f9679bda5
- Tests: mise exec -- scala-cli test model/framework/dsl model/framework/realize --server=false, mise exec -- scala-cli test model/framework/export --server=false --test-only framework.export.ConstructedHeadersSuite, mise exec -- scala-cli test model/framework/export --server=false --test-only framework.export.FiniteSchemaSuite, mise exec -- scala-cli test model/framework/export --server=false --test-only framework.export.ConstructedInboxesSuite, mise exec -- scala-cli test model/framework/export --server=false --test-only framework.export.ConstructedBindingsSuite, mise exec -- scala-cli fmt --scalafmt-conf model/.scalafmt.conf --check model/framework/dsl/Machine.scala model/framework/dsl/MachineHeaders.scala model/framework/dsl/MachineHeaders.test.scala model/framework/reify/plugin/SchemaPhase.scala model/framework/export/ConstructedHeaders.test.scala model/framework/export/testdata/constructedHeaders/Headers.scala model/framework/export/testdata/constructedHeaders/Native.scala model/framework/export/testdata/constructedHeaders/Selection.scala model/framework/export/testdata/constructedHeaders/Timing.scala model/framework/export/testdata/constructedHeaders/Helpers.scala model/framework/export/testdata/constructedHeaders/Client.scala model/framework/export/testdata/constructedHeaders/HelperNative.scala, mise exec -- scala-cli test model/framework/dsl model/framework/realize --server=false --test-only framework.dsl.MachineHeadersSuite, mise exec -- scala-cli test model/framework/export --server=false --test-only framework.export.ConstructedActorsSuite, sha256sum -c .flow/tmp/handoffs/fn174-machine-headers-source.sha256, git diff --check
- PRs: