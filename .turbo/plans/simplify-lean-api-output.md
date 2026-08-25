---
status: draft
---

# Plan: Simplify Lean API Generator Output

## Context

`umpire-gen-api` currently spreads one protobuf-to-Lean projection across
`Temporal/Generated.lean`, a `Temporal/Generated/` tree, and `Temporal/Proto/Core.lean`. It also
emits Catalog inventories, a machine-readable schema, a persisted generation manifest,
source-group-specific GRPC modules, and descriptor structures that have no current consumer. The
generator consequently carries source classification, artifact ledgers, inspection reporting, and
drift-checking behavior that are not required to build the checked-in Temporal Lean model.

Replace that surface with a generation-only command and one semantic module boundary. For a Lean
root such as `Temporal`, the generator owns exactly:

```text
Temporal/API.lean
Temporal/API/Proto.lean
Temporal/API/Types.lean
```

`API.lean` imports both child modules and contains all typed RPC declarations. `Proto.lean`
contains only `Bytes`, `MessageRef`, and typed `Method`. `Types.lean` retains protobuf-derived
namespaces such as `Temporal.Api.Common.V1` and `Google.Protobuf`. The generated declarations remain
structural inputs only; behavioral meaning stays in authored Lean modules.

This is a clean internal break for the checked-in Temporal Lean model. There are no compatibility
shims for the old imports or command flags, and no external-consumer compatibility promise. The
same three-file contract applies to any valid `--lean-root`, not only `Temporal`.

## Interview Decisions

- Optimize first for maintainer clarity, then for a stable `Temporal.API` import surface and lower
  generator maintenance. Success requires both the exact three-file output and removal of the
  obsolete concepts that produced the old tree.
- Treat `Temporal.API` as the supported consumer entry point. The child modules remain ordinary
  Lean modules but are generated implementation details.
- Preserve protobuf-derived declaration names and types, including the current typed RPC metadata:
  request type, response type, full protobuf name, streaming flags, and deprecation flag. Textual
  layout, imports, support qualification, and formatting may change.
- Give the generator exclusive ownership of `<root>/API.lean` and the complete `<root>/API/`
  directory. Authored siblings elsewhere beneath the Lean root remain untouched.
- Render and validate all three artifacts before mutation, then reset the owned output and publish
  `Proto.lean`, `Types.lean`, and `API.lean` in dependency order. An interrupted publication may
  leave an incomplete disposable tree; rerunning generation repairs it.
- Make generation the command's only operation. Remove `generate`, `check`, and `inspect`
  subcommands, use repeatable `--descriptor PATH`, require `--lean-root` and `--output-root`, emit
  nothing on success, and remove the unused context and stdout parameters from the internal entry
  point.
- Remove Catalog, schema, manifest, source grouping, service inventories, artifact digests,
  descriptor digests, inspection reports, and every dormant hook maintained only for those
  features.
- Remove `make umpire-check-api` and `make umpire-check-api-fixture`. Retain generation and golden
  fixture coverage, but add no drift-verification or GitHub Actions work in this change.
- Revise the existing generator and model design documents in place so the repository has one
  current description of the contract.

## Pattern Survey

### Analogous Features

- `tools/umpire3/internal/generate/api/main.go:47` renders a fixed in-memory artifact map before
  publishing it, matching the required render-before-mutate boundary.
- `tools/umpire/internal/generate/api/fixture_test.go:29` compares the complete generated fixture
  tree and rewrites it from a pure artifact map, which remains the readable contract for the new
  three-file output.
- `tools/umpire3/model/Temporal.lean:7` already uses `Temporal.API.*` as the semantic boundary for
  Temporal API modules.
- `tools/umpire3/model/Temporal/Observation.lean:1` demonstrates that a Lean umbrella module can
  coexist with a same-stem directory and import its child modules.

### Reusable Utilities

- `tools/common/artifactio/artifact.go:10` — `Publish` atomically replaces one artifact
  through a same-directory temporary file and preserves the existing generated-file mode.
- `tools/common/artifactio/artifact.go:42` — `Remove` idempotently removes the standalone
  generated `API.lean` artifact and syncs its parent directory.
- `tools/umpire/internal/generate/api/main.go:178` — `sortedArtifactPaths` can remain useful for
  validation and tests even though publication uses explicit dependency order.
- `tools/umpire/internal/generate/api/lean_plan.go:735` — `planService` already resolves stable Lean
  method names and request/response types independently of source-group rendering.
- `tools/umpire/internal/generate/api/render.go:283` — the method-declaration portion of
  `renderGRPC` can move into the `API.lean` renderer; the inventory portion is deleted.
- `tools/umpire/internal/generate/api/render.go:320` — `writeGeneratedHeader` and
  `writeModuleHeader` centralize the existing generated provenance and structural-projection
  comments, which must be preserved.
- `tools/umpire/internal/generate/api/lean_plan.go:1074` — `renderLeanType` continues to render the
  resolved types shared by fields and RPC methods.
- `tools/umpire/internal/generate/api/model.go:348` — `sortedNames` preserves deterministic
  descriptor declaration order.
- `tools/umpire/internal/generate/api/fixture_test.go:43` — `readTree` inventories the entire golden
  directory for exact comparison.

### Convention Anchors

- Descriptor loading, merging, projection, Lean planning, and complete rendering already occur
  before publication in `tools/umpire/internal/generate/api/main.go:26`; keep that deep boundary.
- Protobuf reflection remains separated into projection (`model.go`), Lean naming and type planning
  (`lean_plan.go`), and textual rendering (`render.go`).
- Protobuf type namespaces are independent of Lean module paths. Moving a declaration to
  `Temporal/API/Types.lean` does not change a name such as `Temporal.Api.Common.V1.Payload`.
- The current support renderer combines the live `Bytes`, `MessageRef`, and `Method` declarations
  with inventory-only descriptor records. The new `Proto.lean` keeps the former and deletes the
  latter.
- Focused Go tests cover configuration, projection, planning, rendering, and publication behavior;
  the source-backed fixture covers the exact integrated output.
- The root Makefile owns descriptor acquisition and the production generator invocation. The
  generator remains generic and unaware of Temporal package policy.

### Proposed Alignment

Retain the projection-plan-render pipeline, deterministic in-memory artifacts, generated comments,
protobuf-derived namespaces, typed method values, exact-tree golden fixture, and generic Lean-root
configuration. Replace source-partitioned rendering and manifest reconciliation with one deep
publication operation that owns and resets exactly the API umbrella and API directory.

## Implementation Steps

1. **Collapse the command configuration to generation inputs and one output layout**
   - Change `api.Run` and the command entry point to accept only `[]string` and return an error.
     Remove the unused `context.Context`, stdout writer, operation dispatch, and successful output.
   - Parse flags directly, without a positional operation. Replace `--descriptor NAME=PATH` with a
     repeatable `--descriptor PATH`; keep descriptor path normalization and deterministic sorting.
   - Remove `--source`, `--default-source`, `sourceGroup`, `sourceRule`, `sourceValues`, `Classify`,
     `Sources`, `Groups`, and `DefaultSource`. Always require `--lean-root` and `--output-root`.
   - Replace `outputLayout` with the root module plus paths for exactly `<root>/API.lean`,
     `<root>/API/Proto.lean`, `<root>/API/Types.lean`, and the owned `<root>/API` directory.
   - Update `config_test.go`, `test_config_test.go`, and command-facing tests to pin the path-only
     descriptor syntax, generation-only errors, dotted roots such as `Acme.Model`, and the three
     derived paths.

2. **Remove source, digest, and output-only data from descriptor projection**
   - Reduce `descriptorInput` to the normalized locator and encoded descriptor set. Remove its
     caller-assigned name, SHA-256 digest, JSON tags, and digest-only deterministic re-encoding.
   - Keep semantic deduplication when descriptor sets overlap. Diagnose conflicting protobuf files
     with the two supplied descriptor paths rather than source names.
   - Change `buildProjection` to accept only the merged descriptor set. Remove classifier input,
     ownership maps, every `Source` field, `DescriptorDigest`, and file metadata used only by
     Catalog, schema, manifest, or inspect output.
   - Retain message, enum, field, oneof, service, and method facts consumed by Lean planning and
     rendering, including recursion, presence/cardinality, field and enum numbers, streaming, and
     deprecation.
   - Preserve deterministic descriptor indexing and declaration ordering through
     `indexDescriptors`, `sortedNames`, and sorted merged protobuf file paths.
   - Refocus `descriptors_test.go` and projection assertions on decoding, semantic deduplication,
     conflict diagnostics, deterministic order, and the structural facts still rendered to Lean.

3. **Make the Lean plan describe three modules and one ordered API surface**
   - Remove `leanSourcePlan`, `sourceModuleSpec`, `buildLeanSourcePlans`, source-module cloning and
     equality helpers, source-partition validation, and inventory collision reservations.
   - Define module plans for `<root>.API.Proto`, `<root>.API.Types`, and `<root>.API`. Make Types
     import Proto; make API import both Proto and Types explicitly.
   - Change the support namespace used by bytes, recursive message references, and typed methods
     from `<root>.Proto` to `<root>.API.Proto`.
   - Retain `planService` and expose services in deterministic projection order so the API renderer
     receives a complete ordered declaration surface rather than recovering order from a map.
   - Reserve only the live support declarations `Bytes`, `MessageRef`, and `Method`. Preserve
     existing method-name collision handling and same-package versus qualified request/response
     type rendering.
   - Update `lean_plan_test.go` for the new module paths, support namespace, ordered services,
     collision behavior, recursion, and short versus qualified method types.

4. **Render exactly `Proto.lean`, `Types.lean`, and `API.lean`**
   - Make `generateArtifacts` return only the validated three-entry artifact map; remove inputs,
     manifest/report values, counts, digests, and JSON encoding from its interface.
   - Replace `renderCore` with `renderProto`. Preserve the existing generated comments and emit only
     `Bytes`, `MessageRef`, and `Method` in `<root>.API.Proto`; keep every current field and deriving
     clause on those three structures.
   - Keep `renderTypes` structurally unchanged apart from importing `<root>.API.Proto` and using its
     `Bytes` and `MessageRef` names. Continue reopening protobuf-derived namespaces in global
     dependency order.
   - Replace `renderUmbrella` with the real API renderer. Import `<root>.API.Proto` and
     `<root>.API.Types`, then render every planned service namespace and typed method declaration
     using the method portion of the current `renderGRPC` implementation.
   - Delete `renderCatalog`, grouped GRPC modules, service inventories, `schema.go`, manifest types,
     artifact digests, inspection reports, and their JSON artifacts. Do not retain extension hooks
     for these removed outputs.

5. **Replace operation dispatch and manifest reconciliation with exclusive publication**
   - Keep one orchestration sequence: load descriptor inputs, merge them, build the neutral
     projection, build the Lean plan, render and validate all artifacts, resolve the output root,
     then publish.
   - Extract a small, independently tested publication boundary that accepts the output root,
     layout, and complete artifact map. Validate that the map contains exactly the three managed
     paths and that every resolved target remains inside the configured output root before any
     mutation.
   - Remove the previous `<root>/API.lean` and complete `<root>/API/` entry regardless of whether
     stale contents are regular files, directories, or symlinks. Do not touch authored siblings or
     the legacy layout as an ongoing generator responsibility.
   - Publish `<root>/API/Proto.lean`, then `<root>/API/Types.lean`, then `<root>/API.lean` through
     `artifactio.Publish`. Surface the exact failing path on removal or publication error; a rerun
     repairs any incomplete disposable tree.
   - Delete `writeInspect`, `checkArtifacts`, `loadPreviousManifest`, manifest-based stale cleanup,
     and operation-specific diagnostics. Replace their tests with full reset, unexpected-entry
     removal, sibling preservation, exact-map/path validation, pre-mutation failure, dependency
     publication order, idempotence, and deterministic output coverage.

6. **Migrate fixtures, repository wiring, generated output, and documentation**
   - Change `UMPIRE_GEN_API_ARGS` in the root `Makefile` to pass the four descriptor paths directly,
     followed by `--lean-root Temporal` and `--output-root model`. Invoke `umpire-gen-api` without a
     `generate` operation word.
   - Remove `umpire-check-api` and `umpire-check-api-fixture` recipes and phony declarations. Retain
     `umpire-gen-api` and `umpire-gen-api-fixture`; the latter continues to rebuild `input.pb` and
     rewrite the golden output through the fixture test.
   - Update `fixture_test.go` arguments and regenerate
     `tools/umpire/internal/generate/api/testdata/basic/expected/` as exactly
     `Fixture/API.lean`, `Fixture/API/Proto.lean`, and `Fixture/API/Types.lean`.
   - Regenerate production output as `model/Temporal/API.lean`,
     `model/Temporal/API/Proto.lean`, and `model/Temporal/API/Types.lean`. Delete
     `model/Temporal/Generated.lean`, `model/Temporal/Generated/`, and
     `model/Temporal/Proto/Core.lean` as a one-time repository migration; remove the empty legacy
     directory where applicable.
   - Change `model/Temporal.lean` to import `Temporal.API`. Preserve all existing authored model
     behavior and existing code comments outside the generated replacement.
   - Revise `model/README.md`,
     `docs/superpowers/specs/2026-08-24-descriptor-driven-lean-generator-design.md`, and the
     superseded check-command claims in
     `docs/superpowers/specs/2026-08-23-root-owned-umpire-model-build-design.md`. Document the
     generation-only CLI, three-file contract, exclusive reset, inline RPC declarations,
     namespace/module distinction, direct local verification commands, and removed outputs.
   - Do not add or modify GitHub Actions workflows and do not introduce another drift-verification
     wrapper in this change.

## Acceptance Criteria

- The command accepts repeatable `--descriptor PATH`, `--lean-root`, and `--output-root` flags with
  no operation word, source classification, inspection mode, check mode, or successful stdout.
  Malformed or missing descriptors, duplicate locators, conflicting protobuf definitions, invalid
  Lean roots, and unsafe output targets fail with specific diagnostics before mutation.
- A successful run produces exactly `<root>/API.lean`, `<root>/API/Proto.lean`, and
  `<root>/API/Types.lean` within its owned output. Repeated unchanged runs produce byte-identical
  files and remove every unexpected entry within the owned API tree while preserving adjacent
  authored modules.
- `Proto.lean` declares only `Bytes`, `MessageRef`, and typed `Method` beneath
  `<root>.API.Proto`, preserving their existing fields and deriving clauses. No descriptor records,
  service inventories, Catalog modules, schema, manifest, report, digest, or source-group concept
  remains in the generator contract.
- `Types.lean` preserves protobuf-derived Lean declaration names and types, including recursion,
  maps, oneofs, presence, enum values, and same-package versus qualified references, with support
  references moved to `<root>.API.Proto`.
- `API.lean` imports Proto and Types and declares every unary or streaming RPC in its existing
  protobuf-derived service namespace. Each method preserves its request and response types, full
  protobuf name, streaming flags, deprecation flag, and deterministic collision-resolved Lean name.
- The source-backed fixture and focused Go tests cover the exact three-file artifact set,
  deterministic planning/rendering, descriptor conflicts, full output reset, sibling preservation,
  and validation-before-mutation. No fixture check target, production check target, inspection
  report, or CI drift job is added.
- The checked-in Temporal model imports `Temporal.API`, builds successfully after regeneration, and
  contains no live reference to `Temporal.Generated`, `Temporal.Proto.Core`, generated Catalog or
  GRPC directories, `schema.json`, `manifest.json`, `--source`, `--default-source`, generator
  `check`, or generator `inspect`.

## Boundaries

- The generator remains a general descriptor-set-to-Lean tool; Temporal-specific descriptor
  acquisition stays in the root Makefile.
- Lean protobuf serialization, wire codecs, reflection compatible with a protobuf runtime, and a
  gRPC client or server remain out of scope.
- The refactor does not change authored Lean behavioral semantics or infer product meaning from
  protobuf descriptors.
- Compatibility shims, configurable output layouts, source partitions, future Catalog/schema
  hooks, check/inspect operations, persisted generation state, drift verification, and GitHub
  Actions work are explicitly out of scope.
- The generator owns only the new API umbrella and API directory. Removing the old Generated and
  Proto/Core tree is a one-time repository migration, not a continuing cleanup contract.

## Verification

- Run `gofmt -w tools/umpire/internal/generate/api/*.go tools/umpire/cmd/umpire-gen-api/*.go` and
  `go vet -tags test_dep ./tools/umpire/...`; expect clean formatting and static analysis after the
  obsolete configuration, projection, schema, manifest, inspect, and check code is deleted.
- Run `go test -count=1 -tags test_dep ./tools/umpire/internal/generate/api`; expect focused and
  golden tests to prove deterministic descriptor merging, planning, three-artifact rendering,
  inline unary/streaming RPCs, full replacement, sibling preservation, and pre-mutation validation.
- Run `make umpire-gen-api-fixture` followed by the focused Go test; expect the Mise-pinned `protoc`
  descriptor and exact three-file golden tree to agree.
- Run `make umpire-gen-api`; expect generation to produce only `Temporal/API.lean`,
  `Temporal/API/Proto.lean`, and `Temporal/API/Types.lean` and to remove stale entries in that owned
  tree.
- Run `cd model && mise exec -- lake build`; expect the complete authored and generated Lean model
  to compile through the new `Temporal.API` import boundary.
- Run `find model/Temporal -maxdepth 4 -type f | sort` and
  `rg 'Temporal\.Generated|Temporal\.Proto\.Core|Generated/Catalog|Generated/GRPC|Generated/(schema|manifest)\.json|--source|--default-source' model tools/umpire Makefile docs/superpowers`;
  expect no live code, fixture, command, or current documentation reference to the removed
  contract.
- Run `make umpire-check-regression`; expect the bounded regression compiler and inspector to retain
  their existing deterministic behavior against the renamed generated API boundary.

## Context Files

- `tools/umpire/cmd/umpire-gen-api/main.go` — Adapts process arguments and stderr handling to the
  internal generation entry point.
- `tools/umpire/internal/generate/api/config.go` — Owns the CLI contract and derives every generated
  module path.
- `tools/umpire/internal/generate/api/descriptors.go` — Loads, validates, normalizes, and merges
  descriptor-set inputs; its names and digests become unnecessary.
- `tools/umpire/internal/generate/api/model.go` — Builds the neutral descriptor projection and
  currently attaches source ownership, descriptor digest, and output-only file metadata.
- `tools/umpire/internal/generate/api/lean_plan.go` — Owns protobuf-to-Lean names, types, dependency
  order, services, module specifications, and plan validation.
- `tools/umpire/internal/generate/api/render.go` — Currently renders support, types, Catalog, grouped
  GRPC, schema, manifest, and inspection data; it becomes the three-module renderer.
- `tools/umpire/internal/generate/api/schema.go` — Contains the schema-only projection and is
  deleted.
- `tools/umpire/internal/generate/api/main.go` — Currently dispatches inspect/check/generate and
  reconciles output through the manifest; it becomes generation-only orchestration and exclusive
  publication.
- `tools/umpire/internal/generate/api/main_test.go` — Pins rendering, schema/manifest, lifecycle,
  check, and inspect behavior that must be replaced with the smaller generation contract.
- `tools/umpire/internal/generate/api/fixture_test.go` — Implements exact-tree golden generation and
  comparison.
- `Makefile` — Acquires descriptors, invokes production generation, and currently exposes the two
  check targets being removed.
- `model/README.md` — Defines the generated structural API boundary consumed by authored Lean.
- `docs/superpowers/specs/2026-08-23-root-owned-umpire-model-build-design.md` — Contains the previous
  root-owned check-target contract that this plan supersedes.
- `docs/superpowers/specs/2026-08-24-descriptor-driven-lean-generator-design.md` — Describes the
  current generated tree and lifecycle contract that this simplification replaces.

## Resolved via Codebase

- The current CLI operation, source-group configuration, and old layout are centralized in
  `tools/umpire/internal/generate/api/config.go:29`, `config.go:39`, `config.go:112`, and
  `config.go:206`; they can be removed without changing descriptor acquisition.
- The current runtime branches into inspect, check, and manifest-backed publication in
  `tools/umpire/internal/generate/api/main.go:26`, `main.go:60`, `main.go:71`, and `main.go:102`.
- Rendering already separates support, types, umbrella, and GRPC method declarations in
  `tools/umpire/internal/generate/api/render.go:34`, `render.go:104`, `render.go:192`,
  `render.go:205`, and `render.go:283`, so the method renderer can move without redesigning type
  planning.
- Stable typed service planning and type rendering already live independently of source output in
  `tools/umpire/internal/generate/api/lean_plan.go:735` and `lean_plan.go:1074`.
- The Makefile owns all four descriptor inputs and currently invokes separate generator check and
  fixture-check modes at `Makefile:124`, `Makefile:988`, `Makefile:993`, `Makefile:1006`, and
  `Makefile:1009`.
- No existing GitHub Actions workflow invokes this generator or its Lean model; adding CI is not
  required to unwind an existing integration.

## Resolved via Project Docs

- `model/README.md:3` and `model/README.md:68` establish that generated declarations are structural
  inputs while authored Lean owns behavioral meaning; the refactor must preserve that authority
  boundary.
- `docs/superpowers/specs/2026-08-24-descriptor-driven-lean-generator-design.md:121` documents the
  current Generated/Catalog/GRPC/schema/manifest contract and is the authoritative design document
  to revise in place.
- `docs/superpowers/specs/2026-08-23-root-owned-umpire-model-build-design.md:5` makes
  `make umpire-check-api` the prior root verification entry point, so removing that target requires
  revising this document as well as the model guide.
- `.flow/specs/fn-1-lean-regression-dsl-and-nexus.md:23` confirms that the downstream regression DSL
  depends on mechanical protobuf structure without assigning it Temporal product behavior.
