---
satisfies: [R3, R4, R9]
---
# fn-174-the-model-constructs-its-own-ir-dsl.10 Retain native finite type schemas without activating signature export

Touches: [model/framework/dsl/Domain.scala, model/framework/dsl/FiniteSchema.scala, model/framework/dsl/FiniteSchema.test.scala, model/framework/reify/**, model/framework/export/FiniteSchema.test.scala, model/framework/export/testdata/**, model/README.md]

## Description
Establish native type-schema metadata for the signature migration. Finite currently stores only values; retain nominal IDs, ordered structure, references and bounds independently of catalogs. This is a metadata-only source foundation. Production signature emission and tree-reader retirement remain with .3.

**Size:** M
**Files:** model/framework/dsl/Domain.scala; a co-located schema carrier and tests if needed; model/framework/reify/plugin/ReifyPlugin.scala; a focused export schema comparison fixture; model/README.md.

### Approach
- Reuse reify/body/Types.scala's schema/reference semantics and BodyContext.scala's canonical naming. Keep one semantic definition; avoid a second type interpreter or runtime name guesses.
- First prove the attachment seam on a disposable fixture, especially ReliableBroadcast's custom new Finite[Broadcast] whose sole method delegates to Finite.derived.values. Adding an abstract method or attaching metadata only to the delegated instance breaks this existing spelling.
- Preserve Finite.of/derived/custom givens, restricted/reordered/empty catalogs, Option, Nothing, ordered enum/record cases and fields, opaque ranges and UpTo bounds. Do not infer structure from the first/reachable value or enumerate catalogs to discover schemas.
- Carry ownership-dependent Inbox/channel references explicitly, including distinct channels sharing a message type. Final state-field ownership resolution belongs to signature integration. Schema collection must not enumerate channel contents.
- Preserve public typing, native catalogs/callables and existing Model source spellings. Use compiler-authored typed metadata where structural derivation cannot preserve nominal identity or a custom witness; keep typed metadata available across helper jars without foreign author-tree reads.
- Test duplicate/conflicting nominal identities and recursive/cyclic schema collection with deterministic, located refusals. Preserve existing admission limits; count work before any newly introduced enumeration.
- Keep the production signature path unchanged. No fallback or merge between native and tree producers; fixture comparisons remain isolated.

### Investigation targets
**Required:**
- model/framework/dsl/Domain.scala:19 - Finite, Members and derivation.
- model/framework/reify/body/Types.scala:40 - existing type-reference and schema semantics.
- model/framework/reify/body/BodyContext.scala:433 - canonical type names and collisions.
- model/framework/reify/plugin/ReifyPlugin.scala:310 - compiler-owned declaration metadata.
- model/framework/dsl/Channel.scala:35 - messages, contents and ownership.
- model/framework/samples/ReliableBroadcast.scala:44 - custom Finite spelling (locate actual sample path before use).
- model/framework/export/BoundedIntegers.test.scala:73 - ranges, options and Inbox oracle cases.

### Verification and boundaries
Use co-located native tests and a focused compiled-client schema suite. Freeze old-accepted original fixture sources and compare exact schema messages to independently retained old-reader outputs after stripping only positions. Existing ReifySamples fixtures provide old type-message baselines; they do not prove plugin body parity. Cover empty branches, Nothing, custom catalogs, nested Option/records, opaque bounds, same-spelling nominal collisions and multiple channel owners. Qualify newly supported forms separately from historical old-reader parity. Re-run only affected suites plus formatting on frozen final source; broad/full validation and independent review remain at source-batch end. No model/temporal, tools/umpire, Go, root Model/Make builds, schema/generated outputs, registry duplication, module-map/migration edits, or sample edits. Six samples are read-only regression inputs. Conductor retains Flow and MILESTONES.

## Acceptance
- Native witnesses retain complete type IDs, ordered schemas and referenced schema closure independently of finite values, including empty/custom catalogs and opaque bounds.
- Existing Finite author spellings and public typing/callable/catalog behavior are unchanged; unchanged custom-witness and helper-jar fixtures compile and run without new annotations or schema arguments.
- Focused native and compiled-client cases cover the listed structural, ownership, collision, cycle and bound cases. Independent old-accepted schema comparisons match after position stripping, with intentional differences explicitly reviewed.
- Production signature emission remains on its existing path; no signature reader arm is retired, no second producer is activated, and full .3/R5 acceptance remains open.
- Affected frozen checks and scoped formatting pass; README records the metadata-only boundary, supported cases, exclusions and remaining integration work.


## Done summary
# Native finite-schema framework checkpoint

Task: fn-174-the-model-constructs-its-own-ir-dsl.10 — in_progress; conductor owns review and closure.
Range: 7754e2480b64d9ba9aed087d118027ecc0379ac0..183f41444a3c1f96c0c3fa3e4a3eccff746d6dc8
Commit: 183f41444a3c1f96c0c3fa3e4a3eccff746d6dc8

Native Finite witnesses now retain canonical schema metadata without evaluating their catalogs. The plugin reuses the shared type builder, preserves the exact values-only custom witness spelling and native object identity, and leaves the current signature exporter authoritative.

### Implemented boundary

- One witness-local carrier stores identity, typed root, ordered type closure and typed unresolved Inbox message references. Concrete schema reads select the statically typed view; generic or uninstrumented ambiguous reads fail closed with None. Shared opaque witnesses keep the same native pointer and separate nominal views. Equal reattachments are idempotent; conflicting same-identity schemas refuse without overwriting.
- The existing Types engine supplies enum/record order, Option identities, Nothing, Boolean, UpTo and opaque bounds. Private typed range annotations preserve helper-jar opaque metadata. Fresh contexts isolate unrelated witness traversal. Pending Inbox references distinguish bounded arguments and equal-bound nominal opaque domains without parsing IDs or enumerating channel contents.
- Schema retention runs after Store/Capture/Names and before API hashing/pickling. Existing carrier chunking transports closures larger than the JVM string limit. Typed schema selection also handles named arguments.
- The current signature declaration reader peels only the exact compiler-authored Finite.retainSchema symbol. Existing opaque and Inbox declaration forms remain readable from plugin-compiled TASTy. No signature producer was activated and no reader arm retired.
- Closure validation refuses Option and pending-Inbox cycles and canonical name collisions at the requesting source. Existing plain Int/list and invalid-bound refusals remain intact.
- README documents the metadata-only boundary and remaining integration work. Brainstorming/code-style/TDD skills kept the approved seam narrow; tests-first evidence preceded production fixes.

### Frozen verification

All original command handles were joined; success is the captured suite exit code, not a scraped green line. Total: 213 focused cases.

| Gate | Cases | Handle | Exit | Log |
| --- | ---: | ---: | ---: | --- |
| Native DSL | 117 | 47981 | 0 | /tmp/fn174-schema-final-dsl.log |
| Native realization | 40 | 31225 | 0 | /tmp/fn174-schema-final-realize.log |
| Finite schema | 8 | 97071 | 0 | /tmp/fn174-schema-frozen2-schema.log |
| Reification | 23 | 29711 | 0 | /tmp/fn174-schema-frozen2-reify.log |
| Constructed names | 17 | 30895 | 0 | /tmp/fn174-schema-final-names.log |
| Export bounded integers | 8 | 95809 | 0 | /tmp/fn174-schema-final-bounded.log |

Native gates cover the final unchanged carrier API; later production edits were confined to SchemaPhase cycle traversal. The final configured format check covers all ten affected Scala files and exits 0 (/tmp/fn174-schema-frozen2-format-check-config.log). git diff --check and staged diff checks exit 0. A mistaken default formatter invocation was restored using model/.scalafmt.conf; the final check uses --scalafmt-conf explicitly. Fixture/oracle hashes remain unchanged.

The schema suite exercises exact custom/throwing witnesses, restricted/reordered/empty catalogs, unrelated witness order, Nothing, nested generic Option records, helper jars, shared opaque views including named arguments, scalar/opaque Inbox references, large closures, cycles and refusals. It compiles all six unchanged sample sources WITH the plugin and reads ReliableBroadcast's original custom schema. Its final test also invokes the current signature reader on plugin-compiled opaque and Inbox declarations.

Reification includes module-boundary isolation and warm helper invalidation. Its six sample checks preserve all 167 old-host functions and type messages; this is NOT full plugin-167 parity. Constructed names includes warm family invalidation and existing refusal/sensitivity checks.

### Independent old oracle

Shapes.scala was compiled WITHOUT the plugin against the retained pre-production framework jar, then exported by the independent retained old exporter. The golden contains all nine type messages with only positions removed; the native test sorts top-level messages by name but keeps field/case order and all remaining messages exact. The conductor independently confirmed the full projected diff exits 0.

- Original source: model/framework/export/testdata/finiteSchemas/Shapes.scala; SHA256 237a7307be112bd26c896c9fd081ba97fe6c7bd18fa0fcc3e72e2b4bc06fdf63.
- Pre-production framework: .flow/tmp/finite-schema.16443570503894243763/framework.jar; SHA256 bdae488d22161b7a6ce1e175096df4c1fdf14aa7f63f5e9ae81a0293ffe9d48f.
- Retained old exporter: model/build/sample-check/lifter/lift-samples.jar; SHA256 6565737655135ac497a1eafe2e2b58644113356eb60bd3fc11f343fad83f915b.
- Old client: /tmp/fn174-schema-original2.jar; SHA256 8c845d3f47f35d12d4b6b7ee77be4e6842168491d8ce38794b18afa8f5f28e14. Build handle82068 exit0; /tmp/fn174-schema-original2-build.log.
- Old raw output: /tmp/fn174-schema-original2.json; SHA256 49702a2dcf4aebf99022653527bd8e5b9742ff71da4228487034e4386274f770. Export handle82655 exit0; /tmp/fn174-schema-original2-export.log.
- Golden: model/framework/export/testdata/finiteSchemas/original-types.json; SHA256 ac41edbbdfbb3c2e19e0238bffa2063b5cc5565cabe0f8c9d783ba20a58d2fb3.

The old-accepted fixture includes optional opaque Retries and separate Option[UpTo[1]]/Option[UpTo[3]]. Newly supported generic helpers, custom empty catalogs, schema-only Inbox references and witness sharing have separate native support proofs, not invented old-parity expectations.

### Red-to-green and qualifications

Focused pre-edit realization baseline: handle48277 exit0, 40 cases, /tmp/fn174-schema-baseline.log; no formal baseline handoff.
Initial schema red: handle48153, suite exit1, /tmp/fn174-schema-red.log — missing Finite.schema before production edits. That run retained the independent old framework above.

Behavioral reds included recursive empty witnesses (4802), missing identity (38345), classfile string overflow (40078), opaque-alias range (5945), nested generic A (33222), sample body capture regression (78678), pickled old-reader compatibility (73123), missing typed Inbox-reference payload (14492), and pending-Inbox recursion (9902). Evidence JSON retains their logs and qualifications.

The sample regression was NOT inherited: old plugin/framework control37806 passed, while new78678 failed; the early schema phase hypothesis was refuted and the phase was moved after capture/naming. The old-reader red demonstrated both missing opaque range and missing Inbox channel before exact-symbol peeling repaired it. The final pending-Inbox red accepted the recursive source (compile0 vs expected1); the single pending-edge DFS fix made the frozen suite pass.

Scratch seam evidence proved attachment without evaluating values, not schema parity. An initial closure fixture expected the wrong pre-existing opaque Option name; only that expectation was corrected to the existing raw bounded$.Retries spelling. Missing old artifact, unused import, and an invalid strict-equality assertion were setup/test issues, not behavioral feature reds. Superseded pre-final runs are not claimed as final gates.

### Deferred work and ownership

Explicit channel-owner resolution and signature conversion remain with .3; complete DSL capture points, full plugin-167 parity, held Temporal/model/Go/R5 checks and reader retirement remain in their assigned tasks. No held paths, sample sources, build roots, registries, migrations, schemas or generated IR were edited. Broad make/model-check-changed and root build gates were deferred by the conductor's scope override, not recorded as green or fabricated receipt skips.

Only owner dirt remains: MILESTONES.md, model/design/scala-macros-export.md, and untracked model/design/compiler-plugin-prototype.md. No Flow lifecycle/tracking edits were made. Live handles: none. Own compiler/daemon processes: none observed after all joins. No delegates, bridge, push, rebase, amend or history rewrite.

stage: impl-review - skipped(config: REVIEW_MODE=none; conductor owns deferred source-batch review)
Tier: session (jev-unavailable(no_key)); projectimplementerroute honored.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 183f41444a3c1f96c0c3fa3e4a3eccff746d6dc8
- Tests: timeout 600 mise exec -- scala-cli test model/framework/dsl model/framework/realize --server=false --test-only 'framework.dsl.*', timeout 600 mise exec -- scala-cli test model/framework/dsl model/framework/realize --server=false --test-only 'framework.realize.*', timeout 600 mise exec -- scala-cli test model/framework/export --server=false --test-only framework.export.FiniteSchemaSuite, timeout 600 mise exec -- scala-cli test model/framework/export --server=false --test-only 'framework.*.Reify*', timeout 600 mise exec -- scala-cli test model/framework/export --server=false --test-only framework.export.ConstructedNamesSuite, timeout 600 mise exec -- scala-cli test model/framework/export --server=false --test-only framework.export.BoundedIntegersTest, timeout 600 mise exec -- scala-cli fmt --check --scalafmt-conf model/.scalafmt.conf model/framework/dsl/Domain.scala model/framework/dsl/FiniteSchema.scala model/framework/dsl/FiniteSchema.test.scala model/framework/reify/body/BodyContext.scala model/framework/reify/body/Types.scala model/framework/reify/plugin/PluginContext.scala model/framework/reify/plugin/ReifyPlugin.scala model/framework/reify/plugin/SchemaPhase.scala model/framework/export/FiniteSchema.test.scala model/framework/export/testdata/finiteSchemas/Shapes.scala, git diff --check, git diff --cached --check
- PRs: