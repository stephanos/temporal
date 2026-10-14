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
TBD

## Evidence
- Commits:
- Tests:
- PRs:
