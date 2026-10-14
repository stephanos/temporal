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
TBD

## Evidence
- Commits:
- Tests:
- PRs:
