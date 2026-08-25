---
status: done
---

# Plan: Make Lean API generation descriptor-driven

## Context

`umpire-gen-api` already builds one global declaration plan and generates a useful Lean structural
description of Temporal's protobuf messages, enums, files, services, and typed gRPC methods. Its
input discovery, source registry, Lean support namespace, artifact paths, and manifest still encode
Temporal-specific policy, while its primary integration fixture is a large programmatic Go value.

Implement the approved design in
`docs/superpowers/specs/2026-08-24-descriptor-driven-lean-generator-design.md`: make the existing
command consume only named descriptor sets and repeated source flags, extract registered Go
descriptor acquisition into a generic companion command, derive a self-contained generated tree
from a configurable Lean root, replace the synthetic integration fixture with readable `.proto`
inputs and complete goldens, and keep Temporal-specific composition in the root `Makefile`.

## Pattern Survey

### Analogous Features

- `tools/umpire/internal/generate/api/main.go:30` — The existing command already exposes `generate`, `check`, and `inspect` operations through one orchestration boundary.
- `tools/umpire/internal/generate/api/main.go:100` — Descriptor inputs are merged before one projection, Lean plan, and complete in-memory artifact set are built.
- `tools/umpire/internal/generate/api/main.go:140` — Generated-tree checking compares every expected artifact and uses the previous manifest to detect unexpected stale files.
- `tools/umpire/internal/generate/api/main.go:171` — Publication sorts artifacts, validates managed paths, publishes the manifest last, and removes only manifest-owned stale files.
- `tools/umpire/internal/generate/api/descriptors.go:94` — The current public-API acquisition path already implements the temporary Go helper strategy: discover packages, blank-import them, inspect registered descriptors, and serialize a descriptor set.
- `cmd/tools/getproto/main.go:73` — The repository has another registered-Go-descriptor exporter that maps protobuf imports to Go packages and materializes a `FileDescriptorSet`.
- `tools/umpire3/internal/generate/api/main.go:38` — Another Umpire generator follows the same load → project → render → sorted generate/check lifecycle with atomic artifact publication.
- `tools/gomad2/internal/prettylog/prettylog_test.go:38` — Existing golden tests enumerate readable `testdata` inputs, compare checked-in output byte-for-byte, and offer an explicit rewrite mode.
- `tools/gomad2/internal/translate/translate_test.go:83` — Multi-file source/output fixtures are kept together in `testdata`, reconstructed in a temporary work directory, deterministically sorted, and compared as a complete output set.

### Reusable Utilities

- `tools/common/artifactio/artifact.go:10` — `Publish` — atomically writes an artifact through a protected temporary file, syncs it, and renames it into place.
- `tools/common/artifactio/artifact.go:42` — `Remove` — idempotently removes a generated artifact and syncs its containing directory.
- `tools/umpire/internal/generate/api/descriptors.go:28` — `descriptorFileInput` — reads a serialized descriptor set and attaches input identity and digest metadata.
- `tools/umpire/internal/generate/api/descriptors.go:39` — `newDescriptorInput` — computes a file-order-independent digest through deterministic protobuf serialization.
- `tools/umpire/internal/generate/api/descriptors.go:56` — `mergeDescriptorInputs` — deduplicates identical protobuf files, rejects conflicting definitions at the same path, and returns a path-sorted descriptor set.
- `tools/umpire/internal/generate/api/model.go:134` — `indexDescriptors` — resolves the complete descriptor graph with `protodesc.NewFiles`, indexes messages, enums, and services, and sorts files deterministically.
- `tools/umpire/internal/generate/api/lean_plan.go:190` — `buildLeanPlan` — centralizes global naming, dependency ordering, source ownership, method typing, and validation before rendering.
- `tools/umpire/internal/generate/api/message_graph.go:17` — `buildMessageGraph` — computes stable dependency ordering and strongly connected components for recursive messages.
- `tools/umpire/internal/generate/api/schema.go:55` — `buildSchemaProjection` — derives the JSON representation from the same resolved Lean plan and verifies that every projected declaration has a planned counterpart.
- `tools/umpire/internal/generate/api/render.go:243` — `canonicalIndentedJSON` — provides the generator’s canonical newline-terminated, indented JSON encoding.
- `tools/agentworkflow/cmd/agentworkflow/main.go:527` — `stringList` — established `flag.Value` implementation for validated repeatable command-line flags.
- `tools/umpire3/internal/command/command.go:393` — `repeatedPaths` — analogous repeatable-path flag with empty-value validation and a required-at-least-one check at its consumer.
- `tools/umpire/internal/generate/api/lean_plan.go:287` — `buildLeanPackageNames` — converts dotted protobuf packages into hierarchical Lean namespaces with deterministic collision allocation.
- `tools/umpire/internal/generate/api/lean_plan.go:897` — `validateLeanPlan` — verifies declaration counts, names, dependency order, namespace ownership, services, and source-module completeness.

### Convention Anchors

- Command boundary: `tools/umpire/cmd/umpire-gen-api/main.go:11` keeps the executable thin and delegates behavior to an internal package accepting context, arguments, and output.
- Whole-model data flow: `tools/umpire/internal/generate/api/main.go:100`, `tools/umpire/internal/generate/api/lean_plan.go:190`, and `tools/umpire/internal/generate/api/render.go:37` establish merge → projection → one declaration plan → complete artifact map as the generator’s structural pipeline.
- Deterministic collections: descriptor paths, names, dependency edges, component members, modules, and artifact paths are explicitly sorted throughout `descriptors.go:82`, `model.go:160`, `message_graph.go:58`, and `render.go:76`.
- Contextual validation: generator errors are wrapped at subsystem boundaries, while plan validation reports the offending protobuf identity, as shown in `descriptors.go:62`, `model.go:143`, and `lean_plan.go:950`.
- Managed generated trees: `tools/umpire/internal/generate/api/main.go:155` treats the manifest as the ownership ledger, and `main.go:223` constrains every removable path to the generated tree.
- Self-contained structural Lean support: `model/Temporal/Proto/Core.lean:1` defines the runtime-independent `Bytes`, `MessageRef`, field/message/enum/file descriptors, typed `Method`, and service descriptors consumed by generated modules.
- Protobuf-derived Lean namespaces: `tools/umpire/internal/generate/api/lean_plan.go:287` maps package segments hierarchically, while generated support imports are separately anchored by module specifications at `lean_plan.go:155`.
- Paired generation checks: root targets conventionally come in `gen`/`check` pairs; the current API pair is at `Makefile:962`, and Umpire3 repeats that shape at `Makefile:952`.
- Root Make orchestration: descriptor artifacts and their dependencies are declared centrally at `Makefile:197` and `Makefile:475`; Umpire commands are defined centrally at `Makefile:118` and invoked by root recipes.
- Mise-pinned tool execution: protobuf generation is overridden to `mise exec -- protoc` at `Makefile:962`, Lean execution is defined through `mise exec -- lake` at `Makefile:79`, and tool versions are pinned in `mise.toml:1`.
- Fixture placement: Umpire3 keeps declarative generator input beneath the modeled subsystem’s `testdata/fixtures` tree (`tools/umpire3/model/Temporal/API/testdata/fixtures/selection.json:1`), while generated retained artifacts use `testdata/generated`.
- Focused invariant tests: `tools/umpire/internal/generate/api/lean_plan_test.go:10` keeps collision, ordering, recursive-component, unknown-reference, and module-completeness behavior as direct table-like Go assertions rather than broad output snapshots.

### Proposed Alignment

Blend the existing generator’s deep whole-model pipeline, deterministic planning, manifest ownership, and atomic artifact lifecycle with the repository’s repeatable-flag and source/golden-fixture conventions. Descriptor acquisition aligns with the already-proven registered-Go-descriptor helper pattern, while Temporal-specific composition aligns with the root Makefile’s existing descriptor prerequisites and paired generation/check targets.

## Implementation Steps

1. **Introduce a validated, generic generation configuration**
   - Add `tools/umpire/internal/generate/api/config.go` with `generationConfig`, `descriptorSpec`,
     `sourceRule`, `sourceGroup`, and `outputLayout` types so parsed policy is explicit and passed
     through the pipeline instead of read from package globals.
   - Implement repeatable `flag.Value` types for `--descriptor NAME=PATH` and
     `--source GROUP=PREFIX`, using `strings.Cut` so paths remain opaque after the first `=`.
   - Add `parseGenerationConfig(arguments)` and keep `Run` in
     `tools/umpire/internal/generate/api/main.go` responsible only for operation dispatch,
     orchestration, and output. Remove `RepositoryRoot`, `PublicModule`, `PublicDescriptor`,
     `APIDependencyDescriptor`, `InternalDescriptor`, and `CHASMDescriptor` from `options`.
   - Validate required descriptors, unique input names, source-prefix conflicts, longest-prefix
     classification, the required default source, nonempty output requirements for `generate` and
     `check`, and the absence of positional arguments before loading descriptor bytes.
   - Parse `--lean-root` into a dotted module path and validate each segment against the existing
     Lean identifier/reserved-word rules without rewriting the user-supplied segment. Derive all
     filesystem paths and module names once in `newOutputLayout`.
   - Add `tools/umpire/internal/generate/api/config_test.go` with table-driven coverage for malformed
     repeated values, duplicate descriptor names, invalid roots/groups, repeated prefixes for one
     group, conflicting identical prefixes, nested-prefix precedence, default classification,
     flag-order independence, and operation-specific output requirements.

2. **Make descriptor loading and protobuf projection repository-independent**
   - Narrow `tools/umpire/internal/generate/api/descriptors.go` to descriptor-file input,
     normalization, digesting, and merging. Remove `exportPublicDescriptors`,
     `packageHasTemporalProto`, `commandOutput`, and `publicDescriptorHelper` from the generator
     package.
   - Change `descriptorFileInput` to retain the slash-normalized locator supplied on the CLI while
     reading its absolute or working-directory-relative path. Sort input metadata independently of
     flag order before manifest construction.
   - Enhance `mergeDescriptorInputs` to retain the owning input name for each protobuf path,
     deduplicate `FileDescriptorProto` values using protobuf semantic equality, and name both
     inputs in conflict errors. Keep deterministic protobuf encoding for input and merged-set
     digests.
   - Replace fixed `sourceKind` constants and `classifySource` in
     `tools/umpire/internal/generate/api/model.go` with the validated dynamic `sourceGroup` and a
     classifier passed to `buildProjection`. Classify each file once, then propagate that ownership
     to its messages, enums, and services.
   - Move descriptor merge/digest/conflict cases out of the large `main_test.go` into
     `tools/umpire/internal/generate/api/descriptors_test.go`; add assertions for equal duplicates,
     conflicting owners, unreadable/malformed sets, empty file paths, and stable merged ordering.

3. **Parameterize the global Lean declaration plan and output modules**
   - Change `buildLeanPlan` in `tools/umpire/internal/generate/api/lean_plan.go` to accept the
     validated group registry and `outputLayout`, while preserving one global symbol table,
     message graph, SCC ordering, collision allocator, and reference-resolution pass.
   - Replace `typesModuleSpec`, `sourceModuleSpecs`, and `newSourceModuleSpec` globals with module
     plans derived from `--lean-root` and the sorted configured source groups. Emit Catalog and
     GRPC plans even for empty groups.
   - Construct support references such as `<LeanRoot>.Proto.Bytes`,
     `<LeanRoot>.Proto.MessageRef`, and `<LeanRoot>.Proto.Method` from the layout rather than string
     literals. Keep protobuf-generated type namespaces derived only from protobuf package names.
   - Reserve every generated support and inventory declaration name during plan validation. Reject
     a protobuf declaration whose direct package-derived name would collide with Core structures or
     configured Catalog/GRPC inventory definitions, with both identities in the diagnostic.
   - Update `buildLeanSourcePlans` and `validateLeanSourceModules` to verify exact, single ownership
     across a dynamic group set and to compare against the configuration-derived module registry.
   - Extend `tools/umpire/internal/generate/api/lean_plan_test.go` with non-Temporal Lean roots,
     reordered source flags, empty configured groups, longest-prefix ownership, and support-type
     qualification, including support/inventory collisions, while retaining all collision,
     nesting, cross-package, SCC, and malformed-plan invariants.

4. **Render a self-contained generic artifact tree and manifest**
   - Add `renderCore` in `tools/umpire/internal/generate/api/render.go` to generate the current
     runtime-independent descriptor structures under `<LeanRoot>.Proto`; include its path in the
     artifact map and managed manifest.
   - Parameterize `renderUmbrella`, `renderCatalog`, `renderGRPC`, `writeModuleHeader`, and all
     inventory namespaces with `outputLayout`. Continue direct `strings.Builder` rendering; do not
     introduce templates or reconstruct names outside the validated plan.
   - Replace hardcoded `manifestPath` and `schemaPath` constants with layout-derived paths. Change
     `generationManifest` to format `umpire/protobuf-lean/v1` and record the Lean root, sorted
     inputs, sorted source groups/rules, default source, merged descriptor digest, counts, and
     generated-file digests; keep the manifest's own digest omitted.
   - Update `buildSchemaProjection` in `tools/umpire/internal/generate/api/schema.go` to expose the
     configured source-group labels while continuing to join every Lean name and method type from
     the same declaration plan used by Lean rendering.
   - Make `checkArtifacts`, `publishArtifacts`, `loadPreviousManifest`, and
     `validateManagedPath` in `main.go` accept the layout/manifest path. Allow only the exact Core
     module, umbrella module, and current root's `Generated` descendants; preserve per-file atomic
     publication, previous-manifest stale removal, and manifest-last publication.
   - Replace Temporal-specific drift text with a generic diagnostic that includes the invoked
     operation. Add lifecycle tests for the Core artifact, dynamic roots, stale dynamic-group
     modules, unsafe previous-manifest paths, interrupted/old-manifest detection, and `inspect`
     leaving the output tree untouched.

5. **Extract registered Go descriptor acquisition into a companion command**
   - Add a thin entry point at `cmd/tools/genleanmodeldescriptors/main.go` and a common
     implementation at `tools/common/godescriptors/run.go`, following the existing
     context/arguments/output error boundary used by `umpire-gen-api`.
   - Parse repeatable `--package-pattern` and `--file-prefix` flags plus required `--output`; reject
     empty lists, unexpected positional arguments, and invalid output paths before invoking Go.
   - Adapt the removed temporary-helper logic to run `go list` for all patterns, deterministically
     deduplicate importable generated-protobuf packages, blank-import them in one temporary helper,
     select registered root descriptors by any normalized protobuf prefix, recursively include
     their import closure, sort files by protobuf path, and marshal deterministically.
   - Keep helper-source construction in one small `helperSource` function and format it with the Go
     formatter before execution. Do not reuse the Temporal-specific import mapping or generated
     `files.go` lifecycle in `cmd/tools/getproto`; that command continues to serve `API_BINPB`.
   - Publish `--output` with `artifactio.Publish` and return contextual package-pattern, `go list`,
     helper-build, empty-selection, and write errors.
   - Add `run_test.go` plus a small importable registered-descriptor package beneath
     `tools/common/godescriptors/testdata/godescriptors/` to cover repeated pattern/prefix parsing,
     package deduplication, prefix selection, transitive imports, no matches, deterministic file
     order/bytes, and command failure diagnostics without network access.

6. **Replace programmatic integration descriptors with readable proto and golden fixtures**
   - Create `tools/umpire/internal/generate/api/testdata/basic/input/` with the approved
     `public/v1/model.proto`, `internal/v1/service.proto`, `shared/v1/types.proto`, and
     `legacy/v1/options.proto` fixture set. Keep imports relative to this input root so protobuf
     descriptor paths also exercise Public/Internal prefixes and External default classification.
   - Cover nested messages/enums, same- and cross-package references, maps, real oneofs, proto3
     optional fields, mutual recursion, unary/client/server/bidirectional streaming, deprecation,
     and proto2 required/default/packed metadata.
   - Check in deterministic `testdata/basic/input.pb`, compiled with the Mise-pinned `protoc`,
     `--include_imports`, and all fixture proto files in sorted order.
   - Add `tools/umpire/internal/generate/api/fixture_test.go` with `TestBasicFixture`. Run the normal
     orchestration using `--lean-root Fixture`, generate into a temporary output root, walk the
     entire artifact tree in sorted order, and compare it byte-for-byte with
     `testdata/basic/expected/`. Support the repository's explicit `-rewrite` golden-test pattern.
   - Populate complete expected Core, umbrella, Types, Catalog, GRPC, schema, and manifest artifacts
     beneath `testdata/basic/expected/`; assert that no unlisted expected or actual files exist.
   - Split remaining focused projection and lifecycle tests into purpose-named test files, retain
     direct plan invariant tests, and remove `testDescriptorSet`, `metadataDescriptorSet`, and their
     duplicated builders from `main_test.go` once the fixture pins the same behavior.

7. **Wire generic descriptor acquisition and generation through the root Makefile**
   - Add `GEN_LEAN_MODEL_DESCRIPTORS_COMMAND` and `UMPIRE_PUBLIC_BINPB` variables alongside
     `UMPIRE_GEN_API_COMMAND` and the existing protobuf descriptor variables in `Makefile`.
   - Add a file target for `proto/umpire-public.binpb`, dependent on `go.mod` and `go.sum`, that runs
     the exporter with `go.temporal.io/api/...` and `temporal/api/`. Keep `API_BINPB`,
     `INTERNAL_BINPB`, and `CHASM_BINPB` generation unchanged.
   - Make `umpire-gen-api` and `umpire-check-api` depend on all four descriptor sets and invoke the
     generator exactly once with four `--descriptor` flags, the Public/Internal/CHASM prefixes,
     External default source, `Temporal` Lean root, and `model` output root.
   - Add `umpire-gen-api-fixture` to rebuild `input.pb` and rewrite goldens, and
     `umpire-check-api-fixture` to compile the fixture descriptor into a temporary path, compare it
     with the checked-in binary, and run `TestBasicFixture` without rewriting.
   - Change the root `umpire-check-api` recipe to run `cd model && $(LEAN_LAKE) build`, delete
     `model/Makefile`, and register all new Make shortcuts in the root `.PHONY` declaration.

8. **Regenerate the Temporal model and align its documentation**
   - Run the new exporter and single generator invocation to replace the checked-in public input and
     all managed files under `model/Temporal`. The newly generated
     `model/Temporal/Proto/Core.lean` becomes manifest-owned while direct protobuf-derived type
     namespaces remain unchanged.
   - Update `model/README.md` to describe descriptor-only inputs, configurable source groups,
     self-contained generated support, the `umpire/protobuf-lean/v1` manifest, typed structural
     gRPC descriptions without transport behavior, and the root-only generation/check workflow.
   - Remove obsolete v3 public-module/version manifest fields and verify counts and source ownership
     against the merged four-descriptor input rather than treating generated byte changes as
     compatibility failures.
   - Remove dead Temporal acquisition helpers, fixed-source constants, hardcoded output paths, and
     programmatic fixture code only after focused and golden coverage exercises their replacements.

## Verification

Run all commands from the repository root after implementation:

- Format and statically validate the Go tools:

  ```sh
  go fmt ./tools/umpire/...
  go vet -tags test_dep ./tools/umpire/...
  ```

- Run focused and race-enabled tests:

  ```sh
  go test -count=1 -tags test_dep ./tools/umpire/...
  go test -race -count=1 -tags test_dep ./tools/umpire/...
  ```

- Verify the human-readable fixture, checked descriptor set, complete golden tree, and repeated-run
  determinism:

  ```sh
  make umpire-check-api-fixture
  ```

- Exercise the CLI independently of Temporal discovery and confirm `inspect` is deterministic:

  ```sh
  mise exec -- go run -tags test_dep ./tools/umpire/cmd/umpire-gen-api inspect \
    --descriptor fixture=tools/umpire/internal/generate/api/testdata/basic/input.pb \
    --source Public=public/ \
    --source Internal=internal/ \
    --default-source External \
    --lean-root Fixture
  ```

- Regenerate and check the complete Temporal model, including the pinned Lean build:

  ```sh
  make umpire-gen-api
  make umpire-check-api
  ```

- Confirm the generator implementation has no Temporal acquisition or fixed source paths, and that
  the model-local Make interface is gone:

  ```sh
  ! rg -n 'go\.temporal\.io/api|temporal/api/|temporal/server/api/|chasm/lib/' \
    tools/umpire/internal/generate/api -g '!testdata/**'
  test ! -f model/Makefile
  git diff --check
  ```

Expected results: all Go checks pass; fixture compilation matches `input.pb`; every golden and
managed Temporal artifact is current; repeated generation is byte-identical; `inspect` reports
`umpire/protobuf-lean/v1` with dynamic groups and no Go-module metadata; Lean 4.33.1 builds the
generated model; and no stale or unmanaged file is removed.

## Context Files

- `docs/superpowers/specs/2026-08-24-descriptor-driven-lean-generator-design.md` — Approved behavior,
  boundaries, outputs, and acceptance criteria.
- `tools/umpire/internal/generate/api/main.go` — Existing CLI orchestration and generated-tree
  lifecycle boundary.
- `tools/umpire/internal/generate/api/descriptors.go` — Descriptor normalization/merge logic and the
  Temporal acquisition code to extract.
- `tools/umpire/internal/generate/api/model.go` — Protobuf projection and fixed source
  classification to parameterize.
- `tools/umpire/internal/generate/api/lean_plan.go` — Global names, SCC-informed types, module plans,
  and validation that must remain authoritative.
- `tools/umpire/internal/generate/api/render.go` — Artifact composition, hardcoded paths/namespaces,
  and manifest schema to generalize.
- `tools/umpire/internal/generate/api/main_test.go` — Programmatic descriptor fixture and lifecycle
  coverage to migrate without losing invariants.
- `tools/common/artifactio/artifact.go` — Atomic publish/remove primitives shared by the
  generator and exporter.
- `cmd/tools/getproto/main.go` — Existing dependency-descriptor lifecycle that the new public
  exporter must coexist with rather than replace.
- `Makefile` — Temporal descriptor prerequisites, paired Umpire generation/check recipes, Lake
  wrapper, and fixture shortcuts.
- `model/Temporal/Proto/Core.lean` — Structural support declarations that become generated.
- `model/README.md` — User-facing model contract and root workflow documentation.
- `mise.toml` — Pinned Go, protoc, and Lean versions used by fixture and end-to-end verification.
