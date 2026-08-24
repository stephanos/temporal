---
status: draft
---

# Plan: Resolve Lean declarations before rendering

## Context

`umpire-gen-api` currently mixes protobuf projection, Lean name allocation, recursion analysis, source grouping, type construction, and text rendering. That makes correctness depend on several loosely coordinated passes: declaration names are stored in the neutral projection, field and oneof collisions are resolved in different places, recursive references are discovered with repeated graph walks, and renderers reconstruct both types and source partitions.

Introduce one unexported Lean declaration-plan seam inside `tools/umpire/internal/generate/api`. It will consume a Lean-neutral protobuf projection, allocate every Lean name and type, analyze the message graph once, assign declarations to logical namespaces and existing physical modules, and validate the complete result. Renderers will receive only resolved declarations and mechanically serialize them. `Run` remains the package's only exported entry point.

The generated API adopts direct protobuf-derived Lean namespaces and does not preserve flattened generated identifiers byte for byte. For example:

- `temporal.server.api.adminservice.v1.DescribeMutableStateRequest` becomes `Temporal.Server.Api.Adminservice.V1.DescribeMutableStateRequest`.
- `temporal.api.common.v1.Link.WorkflowEvent.EventReference` preserves its nested declaration hierarchy as `Temporal.Api.Common.V1.Link.WorkflowEvent.EventReference`.
- Same-package references use the shortest unambiguous name; cross-package references remain fully qualified.

Physical output remains coarse-grained: one type file, four catalog files, four gRPC files, the umbrella module, schema, and manifest. Catalog and gRPC inventory declarations remain under `Temporal.Proto.Generated.Catalog.*` and `Temporal.Proto.Generated.GRPC.*`; generated wire types and typed RPC method declarations use the direct protobuf namespaces. The manifest format advances to `umpire/temporal-api/v3`, and no aliases for the old flattened identifiers are generated.

The plan deliberately retains protobuf semantics already handled by the generator: proto2 default lexical values, presence and required/repeated cardinality, maps and synthetic map entries, synthetic optional versus real oneof behavior, recursive `MessageRef` placement, source classification including CHASM spelling, public dynamic descriptor helpers, manifest self-digest omission, managed paths, and atomic publication. Descriptor acquisition/normalization and generated-tree lifecycle extraction are separate follow-ups.

No new third-party dependency is needed. Continue to use `protoreflect`/`protodesc` for descriptor semantics and `strings.Builder` for Lean output. The available Lean protobuf/gRPC libraries do not improve this description-only generator enough to justify coupling the model to runtime serialization or clients.

## Pattern Survey

### Analogous Features

- `tools/umpire2/internal/protocol/protocol.go:13` — Separates a neutral `Declaration` authoring form from an immutable compiled `Protocol`, keeping the compiled representation unexported behind a small interface.
- `tools/umpire2/internal/protocol/compile.go:13` — `Compile` constructs indexes and validates duplicate identities, factories, cross-references, lifecycles, actions, and gaps before returning a usable compiled value; failures carry declaration context.
- `docs/plans/2026-08-09-canonical-go-ir-v2-design.md:90` — Records the established architectural rationale for one deep module whose compiler validates relationships once and hides source-registry complexity from consumers.
- `common/testing/umpire/verify/toolchain/internal/tla/generate.go:14` — Backend generation validates the neutral model, canonicalizes it, creates an unexported target generator, and only then renders artifacts.
- `common/testing/umpire/verify/toolchain/internal/tla/generate.go:175` — Target-specific identifier validation detects when distinct source declarations normalize to the same backend identifier before rendering.
- `tools/umpire3/internal/generate/api/projection.go:78` — The sibling protobuf generator resolves descriptor closure, collects names globally, derives a type-name table, and orders messages before its renderer consumes them.
- `tools/umpire3/internal/generate/api/projection.go:133` — Type names are assigned from a complete message-and-enum set rather than independently at each reference site.
- `tools/umpire3/internal/generate/api/projection.go:393` — Its dependency ordering uses a deterministic depth-first traversal over sorted dependencies, analogous to the current Umpire generator’s ordering logic.
- `tools/umpire/internal/generate/api/main_test.go:15` — Synthetic descriptor tests exercise messages, oneofs, maps, recursion, services, and streaming RPCs through projection and artifact generation together.
- `tools/umpire/internal/generate/api/main_test.go:43` — Determinism is asserted by changing descriptor-file traversal order and comparing complete projections.
- `tools/umpire/internal/generate/api/main_test.go:146` — Adversarial identifier tests pin collision behavior, including already-emitted suffixes and negative discriminators.
- `tools/umpire2/internal/protocol/compile_test.go:164` — Compiler validation is conventionally tested through table-driven mutations of one valid declaration, with each malformed relationship asserting a contextual failure.
- `tools/umpire3/model/Temporal/API/Workflow.lean:4` — Authored Lean already uses dotted logical namespaces and short local references under an imported namespace.
- `tools/umpire3/model/Umpire3/NativeCertificateRunner.lean:10` — Multi-segment dotted namespaces are routinely used for deeper Lean ownership and qualification.

### Reusable Utilities

- `tools/umpire/internal/generate/api/model.go:134` — `indexDescriptors` — Uses `protodesc.NewFiles` and `protoreflect.FullName` keys to resolve and index the complete descriptor graph with canonical protobuf identities.
- `tools/umpire/internal/generate/api/model.go:282` — `collectMessages` / `collectEnums` — Recursively collect nested protobuf declarations while preserving descriptor parentage.
- `tools/umpire/internal/generate/api/model.go:415` — `sortedNames` — Provides deterministic ordering for maps keyed by `protoreflect.FullName`.
- `tools/umpire/internal/generate/api/model.go:424` — `classifySource` — Centralizes the Public/Internal/CHASM/External classification already used by files, messages, enums, services, catalogs, and gRPC output.
- `tools/umpire/internal/generate/api/model.go:445` — `upperIdentifier`, `lowerIdentifier`, and `identifierParts` — Hold the current Lean identifier normalization and reserved-word policy.
- `tools/umpire/internal/generate/api/render.go:316` — `canonicalIndentedJSON` — Produces the canonical indented JSON representation used by schema, manifest, and inspect output.
- `tools/umpire/internal/generate/api/main.go:234` — `sortedArtifactPaths` — Canonicalizes artifact traversal independently of Go map iteration.
- `tools/umpire/internal/artifactio/artifact.go:10` — `artifactio.Publish` / `Remove` — Provide atomic publication and managed stale-file removal without coupling those concerns to projection or Lean rendering.
- `google.golang.org/protobuf/reflect/protoreflect` and `protodesc` — Already supply descriptor identity, parent/package relationships, kinds, cardinality, map-entry semantics, oneof semantics, and resolved type references used throughout `model.go`.
- No reusable SCC implementation was found in the repository. Existing graph handling is task-specific DFS, such as `tools/umpire/internal/generate/api/model.go:350`, `tools/umpire3/internal/generate/api/projection.go:299`, and cycle rejection in `tools/umpire3/protocol/catalog/catalog.go:403`; Gonum appears only transitively and has no repository usage for this purpose.

### Convention Anchors

- Unexported generator internals: `tools/umpire/internal/generate/api/main.go:30` exposes only `Run`; projection, rendering, artifact management, and helpers remain package-private. The command wrapper at `tools/umpire/cmd/umpire-gen-api/main.go:11` contains no generator logic.
- Staged data flow: `tools/umpire/internal/generate/api/main.go:100` follows descriptor merge → neutral projection → artifact generation → inspect/check/publish, with filesystem effects occurring only after in-memory artifacts exist.
- Validate-before-render: formal backends validate and canonicalize their model before emitting text (`common/testing/umpire/verify/toolchain/internal/tla/generate.go:14`), while compiled-domain modules refuse to return partial usable state (`tools/umpire2/internal/protocol/compile.go:13`).
- Explicit determinism: descriptor files, packages, dependencies, names, drift reports, and artifact paths are explicitly sorted with `slices` rather than relying on map or registry iteration (`tools/umpire/internal/generate/api/descriptors.go:43`, `model.go:160`, `main.go:165`, `render.go:77`).
- Direct typed rendering: both API generators use `strings.Builder`/`fmt.Fprintf` rather than `text/template` (`tools/umpire/internal/generate/api/render.go:108`, `tools/umpire3/internal/generate/api/generate.go:56`); no template abstraction exists for Lean generation.
- Current Lean decisions leak across seams: `LeanName` fields live in the protobuf projection (`tools/umpire/internal/generate/api/model.go:32`), field and oneof collisions are partly resolved during projection (`model.go:226`) and partly during rendering (`render.go:115`, `render.go:125`, `render.go:134`), and type construction remains renderer-side (`render.go:236`).
- Current recursion/order analysis is duplicated: each field performs a fresh reachability search (`tools/umpire/internal/generate/api/model.go:321`), followed by a separate dependency traversal over projected messages (`model.go:383`).
- Current source grouping is renderer-side: the artifact set is declared explicitly for four sources (`tools/umpire/internal/generate/api/render.go:43`), then catalogs and gRPC renderers each rescan the complete projection and filter by source (`render.go:157`, `render.go:203`).
- Current oneof representation duplicates canonical fields: `oneofProjection.Fields` contains full `fieldProjection` values (`tools/umpire/internal/generate/api/model.go:63`), populated by copying each projected field (`model.go:240`).
- Artifact grouping is a stable contract: one umbrella, one type file, four catalog files, four gRPC files, schema, and manifest are declared together in `tools/umpire/internal/generate/api/render.go:43`; the umbrella imports all generated Lean modules at `render.go:94`.
- Schema versioning is colocated with artifact composition: the manifest format is selected where projection-derived artifacts are assembled (`tools/umpire/internal/generate/api/render.go:61`), and schema shape is defined by JSON tags on projection types (`model.go:26`).
- Generated-tree tests cover both content and lifecycle: `tools/umpire/internal/generate/api/main_test.go:176` verifies publication, drift detection, and managed-path safety; `main_test.go:195` verifies detection of artifacts removed from the generator.
- Lean module organization uses dotted namespaces while physical files remain coarse-grained: authored files such as `tools/umpire3/model/Temporal/API/Workflow.lean:4` and `tools/umpire3/model/Umpire3/NativeCertificateRunner.lean:10` establish logical nesting without requiring one file per namespace.

### Proposed Alignment

Blend the existing compiled-declaration pattern from `tools/umpire2/internal/protocol` with the formal toolchain’s validate-and-canonicalize-before-render convention, while retaining the current generator’s single exported `Run`, deterministic artifact pipeline, coarse Lean file grouping, and direct builder rendering. No existing feature maps protobuf packages directly to Lean namespaces or provides SCC analysis, so those semantics remain specific responsibilities of the unexported declaration-plan module rather than reusable repository utilities.

## Implementation Steps

1. Characterize the v3 contract with plan-level and artifact-level tests in `tools/umpire/internal/generate/api/main_test.go` before changing production code.
   - Extend the synthetic descriptor fixture with nested messages/enums, same-package and cross-package references, mutually recursive and acyclic messages, real and synthetic oneofs, maps, enum values, services, streaming methods, and deliberately colliding protobuf identifiers.
   - Assert direct package namespaces, preserved nested declaration ownership, short same-package type references, fully qualified cross-package references, and the absence of old `Temporal_*` flattened declarations.
   - Assert deterministic name allocation when descriptor/file traversal order changes. Cover package, nested declaration, field, oneof constructor, enum value, service, and method collisions, including names that already resemble generated suffixes.
   - Pin stable suffix policy: start from protobuf identity, prefer the wire number only when it uniquely distinguishes members in that scope, and use a short deterministic digest fallback when normalized names still collide. Never use traversal order.
   - Assert that recursive edges use `Temporal.Proto.MessageRef`, nonrecursive dependencies are declared first, and multi-node strongly connected components remain stable under input permutation.
   - Update schema assertions for dotted `leanName` values and oneof field references by protobuf full name, and pin `umpire/temporal-api/v3` in the manifest.
   - Preserve the existing descriptor metadata, deterministic merge/digest, external service, array-shape, publication, drift, managed-path, and removed-artifact coverage.

2. Make `projection` in `tools/umpire/internal/generate/api/model.go` strictly Lean-neutral.
   - Remove `LeanName` from `enumProjection`, `fieldProjection`, `oneofProjection`, `messageProjection`, `methodProjection`, and `serviceProjection`; remove `InputLeanType` and `OutputLeanType` from methods.
   - Preserve canonical protobuf full names, descriptor names, source paths/classification, field numbers/kinds/cardinality/presence/defaults, map key/value facts, oneof membership, streaming flags, input/output protobuf types, and other descriptor metadata required by schema and planning.
   - Replace copied `oneofProjection.Fields []fieldProjection` values with canonical `FieldNames []string` references containing stable protobuf field full names. Resolve a oneof's fields through the message's single canonical field collection.
   - Keep `indexDescriptors`, `collectMessages`, `collectEnums`, `sortedNames`, `classifySource`, `descriptorTypeName`, and descriptor metadata extraction as the protobuf boundary.
   - Remove `leanTypeName`, `uniqueName`, `reaches`, and `dependencyOrder` from the neutral projection path after equivalent responsibilities are covered by the declaration planner.

3. Add an unexported declaration compiler in `tools/umpire/internal/generate/api/lean_plan.go`.
   - Define a closed, typed plan representation: `leanPlan`, `leanModulePlan`, `leanNamespacePlan`, typed enum/message/oneof/service/method declarations, qualified Lean identifiers, and a `leanType` sum representation for named types, `Option`, `List`, map products, scalar types, and `Temporal.Proto.MessageRef`.
   - Implement `buildLeanPlan(projection) (leanPlan, error)` as the sole owner of Lean-specific decisions. Build a global symbol table keyed by canonical protobuf full name before resolving any references.
   - Map protobuf package segments and nested declaration ancestors independently through the existing Lean identifier rules, preserving hierarchy. Allocate names per namespace/scope so unrelated packages do not lengthen one another's declarations.
   - Resolve all valid normalized-name collisions deterministically from protobuf identity. Use a unique field/enum wire number when it is readable and sufficient; otherwise append a short digest derived from the full protobuf identity. Detect a digest collision and lengthen or otherwise deterministically disambiguate it rather than depending on insertion order.
   - Allocate field, oneof constructor, structure oneof-slot, enum value, service, and method names through the same scoped allocator instead of renderer-specific special cases. Reserve Lean keywords and generated constructors such as `notSet` through allocator inputs.
   - Resolve named type references only after the symbol table is complete. Render same-package references relative to their package namespace when unambiguous and cross-package references as fully qualified identifiers; do not emit `open` directives or compatibility aliases.
   - Return contextual errors containing the protobuf declaration/field/method identity whenever a name, reference, type, module, import, or ordering invariant cannot be resolved.

4. Analyze message dependencies once as part of declaration-plan construction.
   - Add a small package-private graph implementation in `tools/umpire/internal/generate/api/message_graph.go`; do not introduce Gonum or another dependency.
   - Build a deterministic adjacency list from canonical message-valued fields, including map value messages, and traverse nodes/edges in sorted protobuf full-name order.
   - Compute strongly connected components once. Mark an edge recursive when its endpoints share a cyclic component, including self-loops, and wrap only those references in `Temporal.Proto.MessageRef`.
   - Collapse the graph to an SCC DAG and produce a deterministic topological order for nonrecursive declaration dependencies. Group the resulting declaration order by protobuf package namespace inside `Types.lean` without violating dependency order.
   - Validate that every message reference is represented in the graph and that the final order respects all edges between distinct components.

5. Construct logical namespaces and physical module partitions once in the plan.
   - Define one ordered source/module registry for Public, Internal, CHASM, and External. Use it to construct both catalog and gRPC module plans, replacing eight hardcoded artifact entries and repeated full-projection filtering.
   - Assign enums, messages, nested oneofs, services, and typed methods to direct protobuf-derived namespaces. Keep generated catalog and gRPC inventory declarations in `Temporal.Proto.Generated.Catalog.<Source>` and `Temporal.Proto.Generated.GRPC.<Source>`.
   - Store required imports and logical namespace ownership on each `leanModulePlan`; validate that every declaration belongs to exactly one physical module and every referenced declaration is available through the module's imports.
   - Retain the existing artifact paths and umbrella imports so module granularity does not expand to one file per protobuf package.

6. Make `tools/umpire/internal/generate/api/render.go` a mechanical serializer of the validated plan.
   - Change `generateArtifacts` to build/receive the `leanPlan` and return planner validation errors before producing any artifact bytes.
   - Update `renderTypes`, `renderCatalog`, and `renderGRPC` to iterate preordered module declarations and emit namespace blocks from structured qualified names.
   - Give `leanType` one recursive printer responsible only for syntax and parenthesization. Remove renderer-side `leanFieldType`, `leanFieldBaseType`, `leanScalarType`, `leanTypeNameFromString`, collision tracking, source filtering, and type/reference reconstruction.
   - Keep direct `strings.Builder`/`fmt.Fprintf` output and the separate type, catalog, and gRPC render shapes; do not add templates or a universal rendering abstraction.
   - Retain generated headers, canonical JSON encoding, manifest hashing/self-digest omission, and current artifact path ordering.

7. Build the v3 schema from the neutral projection plus the resolved declaration plan.
   - Add a package-private schema view (for example `schemaProjection`) that joins protobuf facts with plan-owned Lean names at serialization time rather than putting Lean data back into `projection`.
   - Include resolved dotted Lean names for declarations and members, and input/output Lean types for methods, from the same plan objects used by the Lean renderer.
   - Represent oneof membership with stable protobuf full-name references instead of duplicated field objects or array indices, and validate that every reference resolves within its containing message.
   - Set the manifest format to `umpire/temporal-api/v3`; retain the descriptor input digest and deterministic artifact digests.

8. Regenerate and document the model contract.
   - Regenerate all managed artifacts under `model/Temporal/Generated/`, plus `model/schema.json` and `model/manifest.json`, in one run so no v2/flattened output remains.
   - Update `model/README.md` examples and naming documentation to show direct package namespaces, nested declarations, same-package shorthand, fully qualified cross-package references, unchanged physical file grouping, and the v3 schema/manifest contract.
   - Keep `model/Temporal/Proto/Core.lean` focused on shared model primitives such as `MessageRef` and method metadata; add no client transport or protobuf runtime dependency.

9. Replace implementation-shaped tests with seam tests and retain end-to-end lifecycle coverage.
   - Remove tests that directly pin obsolete helpers such as `uniqueName` or hand-build projections containing Lean fields.
   - Add focused table-driven tests for `buildLeanPlan`: symbol collisions, nested ownership, short versus qualified references, scalar/container type construction, recursive SCCs, deterministic order, module placement/imports, oneof field resolution, and contextual validation errors.
   - Keep artifact tests for generated Lean syntax/content, schema joins, manifest v3, determinism, publish/check drift, stale managed-file cleanup, and output removed from the generator.
   - Ensure malformed plan fixtures cannot reach a renderer; render tests may use only validated plan fixtures or the public generation pipeline.

## Verification

Run all checks from the repository root after implementation:

1. Format and statically check the generator:

   ```bash
   gofmt -w tools/umpire/internal/generate/api/*.go
   go vet -tags test_dep ./tools/umpire/...
   ```

2. Run the focused generator suite, including the race detector:

   ```bash
   go test -count=1 -tags test_dep ./tools/umpire/...
   go test -race -count=1 -tags test_dep ./tools/umpire/...
   ```

3. Generate twice into separate temporary directories and compare them recursively to prove byte-level determinism for identical descriptor input. Repeat with a test descriptor set whose file order is permuted.

4. Regenerate the checked-in model and verify it is current:

   ```bash
   make umpire-gen-api
   make umpire-check-api
   ```

5. Inspect generated contracts:
   - Confirm `Types.lean` contains direct namespace blocks such as `Temporal.Server.Api.Adminservice.V1` and preserves nested declarations.
   - Confirm same-package references are short, cross-package references are fully qualified, and recursive references alone use `Temporal.Proto.MessageRef`.
   - Confirm no old flattened `Temporal_*` generated declarations or compatibility aliases remain.
   - Confirm the four catalog and four gRPC artifacts still exist and their inventories match the schema source classifications.
   - Confirm every schema oneof field reference resolves to exactly one canonical field and every schema Lean name/type equals the corresponding declaration-plan value.
   - Confirm the manifest reports `umpire/temporal-api/v3`, omits its own digest, and records correct hashes for every other managed artifact.

6. Build the Lean model using the repository-pinned Lean 4.33.1 toolchain, ensuring all generated module imports and qualified references resolve.

7. Finish with `git diff --check` and review the generated diff for unexpected descriptor-count or source-classification changes. The descriptor digest and protobuf declaration/service/method counts should remain unchanged unless the Temporal descriptor inputs changed independently.

## Context Files

- `tools/umpire/internal/generate/api/model.go` — Current protobuf projection, Lean-name leakage, repeated recursion analysis, and deterministic descriptor traversal.
- `tools/umpire/internal/generate/api/render.go` — Current artifact composition, renderer-side naming/type decisions, source rescans, schema serialization, and manifest version.
- `tools/umpire/internal/generate/api/main.go` — `Run` boundary and descriptor → projection → artifact → inspect/check/publish pipeline that must remain the only exported interface.
- `tools/umpire/internal/generate/api/main_test.go` — Existing synthetic descriptor, determinism, schema, collision, and generated-tree lifecycle coverage to evolve around the declaration-plan seam.
- `tools/umpire/internal/generate/api/descriptors.go` — Descriptor-set merge and normalization boundary that remains outside this refactor.
- `tools/umpire/internal/artifactio/artifact.go` — Existing atomic publish and managed-file removal behavior that remains outside the plan.
- `tools/umpire2/internal/protocol/protocol.go` — Repository pattern for hiding a compiled, immutable representation behind a narrow interface.
- `tools/umpire2/internal/protocol/compile.go` — Repository pattern for resolving and validating relationships before consumers can use compiled data.
- `common/testing/umpire/verify/toolchain/internal/tla/generate.go` — Repository pattern for target-specific validation/canonicalization before direct rendering.
- `tools/umpire3/internal/generate/api/projection.go` — Sibling generator pattern for global symbol collection and deterministic dependency ordering.
- `model/Temporal/Proto/Core.lean` — Shared Lean primitives whose `MessageRef` and method metadata shape the planned types.
- `model/README.md` — User-facing generated-model contract and examples that must describe the new namespaces and v3 format.
