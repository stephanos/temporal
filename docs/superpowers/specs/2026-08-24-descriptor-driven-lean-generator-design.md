# Descriptor-driven protobuf-to-Lean generation

## Status

Design direction approved on 2026-08-24; written specification pending review.

## Context

`umpire-gen-api` currently produces a useful Lean description of Temporal's public, internal,
CHASM, and dependency protobuf APIs. The projection itself is broadly useful, but descriptor
acquisition, source classification, output namespaces, and manifest metadata contain
Temporal-specific policy. The integration tests also construct a large descriptor set in Go,
which makes the intended input and generated output difficult to review.

The generator should instead be a general protobuf-descriptor-to-Lean tool. Temporal will remain
one caller, with its descriptor inputs and source boundaries declared in the root `Makefile`.

## Goals

- Generate a well-formed Lean structural model from one or more protobuf descriptor sets.
- Describe protobuf messages, enums, fields, files, services, and gRPC method signatures without
  implementing protobuf encoding or a gRPC transport.
- Accept all repository-specific policy through repeatable command-line flags.
- Resolve one global Lean declaration plan before rendering so that names, recursive types, and
  cross-source references are consistent.
- Emit a self-contained generated tree, including the small Lean support module used by generated
  declarations.
- Make the generator understandable through small `.proto` fixtures and complete golden output.
- Preserve deterministic generation, drift checking, and managed stale-file removal.

## Non-goals

- Lean protobuf serialization, reflection compatible with a protobuf runtime, or generated wire
  codecs.
- A Lean gRPC client or server. The generated gRPC modules only describe services and typed method
  signatures.
- A Temporal-specific wrapper command or generator configuration file.
- Byte-for-byte compatibility with the current generated model.
- Independent source-group invocations that later attempt to combine their output.

## Chosen architecture

The existing `umpire-gen-api` command will become descriptor-set driven. It will retain the
`generate`, `check`, and `inspect` operations, but remove its knowledge of Go modules, Temporal
package paths, and the fixed Public/Internal/CHASM/External source set.

A run has five stages behind a single orchestration boundary:

1. Parse and validate every flag before reading or writing generated artifacts.
2. Load all named descriptor sets and merge them by protobuf file path.
3. Build one global symbol table, message dependency graph, strongly connected component graph,
   and Lean declaration plan.
4. Classify files into configured source groups, then render all Lean and JSON artifacts in memory.
5. Inspect, compare, or publish the complete artifact set according to the selected operation.

The declaration plan is the central abstraction. Rendering receives resolved Lean names and type
references rather than independently rediscovering them. This keeps collision handling,
recursive-type handling, catalogs, gRPC declarations, and JSON schema in agreement.

## Command-line interface

A Temporal invocation will have this shape:

```sh
umpire-gen-api generate \
  --descriptor public=proto/umpire-public.binpb \
  --descriptor dependencies=proto/api.binpb \
  --descriptor internal=proto/image.bin \
  --descriptor chasm=proto/chasm.bin \
  --source Public=temporal/api/ \
  --source Internal=temporal/server/api/ \
  --source CHASM=chasm/lib/ \
  --default-source External \
  --lean-root Temporal \
  --output-root model
```

The interface is deliberately flag-based:

- `--descriptor NAME=PATH` is required and repeatable. Names must be non-empty and unique. Paths
  may be absolute or relative to the current working directory; the manifest records the supplied,
  slash-normalized locator rather than a machine-specific absolute path.
- `--source GROUP=PREFIX` is repeatable. A group may have more than one prefix. Prefixes are
  normalized to protobuf's slash-separated file-path form.
- `--default-source GROUP` is required and classifies every file not matched by a prefix.
- `--lean-root MODULE` is required and accepts a dotted Lean module path such as `Temporal` or
  `Acme.Model`.
- `--output-root PATH` is required for `generate` and `check`; it is the filesystem directory under
  which the Lean root is created.

The current `--repository-root`, `--public-module`, `--public-descriptor`, `--api-dependencies`,
`--internal-descriptor`, and `--chasm-descriptor` flags will be removed. The working directory is
the only base for relative paths.

`inspect` performs the same parsing, merge, classification, and declaration planning as
generation, then writes the canonical manifest-shaped inventory to standard output without
touching the output tree. It does not require `--output-root`.

## Validation and source classification

Configuration is validated as a whole so failures are reported before generation begins.

- Descriptor input names are unique.
- Every descriptor path exists and decodes as a `FileDescriptorSet`.
- The Lean root is a non-empty dotted sequence of valid Lean module segments.
- Source group labels must be valid, distinct Lean module segments and are used verbatim in module
  and declaration names.
- The default source is included in the complete group set even when it has no explicit prefix.
- An identical prefix cannot be assigned to different groups. Nested prefixes are allowed; the
  longest matching prefix wins.
- Every protobuf file belongs to exactly one group because unmatched files use the default source.

When multiple descriptor sets contain the same protobuf file path, descriptors equal according to
protobuf semantics are deduplicated. Different descriptors at the same path are a conflict and
identify both named inputs in the error. The merged file set is sorted by path before analysis.

Groups are sorted by canonical group name, inputs by name and locator, protobuf declarations by
their resolved plan order, and JSON keys by the existing canonical JSON rules. Flag order cannot
change generated bytes.

## Generated tree

For `--lean-root Temporal --output-root model`, the generator owns these artifacts:

| Artifact | Purpose |
| --- | --- |
| `model/Temporal/Proto/Core.lean` | Generated, runtime-independent descriptor structures such as `Bytes`, `MessageRef`, and `Method` |
| `model/Temporal/Generated/Types.lean` | Lean structures and enums projected from the merged descriptor graph |
| `model/Temporal/Generated/Catalog/<Group>.lean` | Per-source inventories of files, messages, enums, and services |
| `model/Temporal/Generated/GRPC/<Group>.lean` | Per-source typed method declarations and service descriptions |
| `model/Temporal/Generated.lean` | Umbrella module importing every generated Lean module |
| `model/Temporal/Generated/schema.json` | Machine-readable projection of the merged protobuf model |
| `model/Temporal/Generated/manifest.json` | Inputs, configuration, counts, digests, and managed generated files |

Every configured group gets Catalog and GRPC modules, including empty groups, so the flags define
a stable output contract.

“Self-contained” applies to the generated tree: `Core.lean` is generated alongside its consumers,
and generation does not depend on a pre-existing authored Lean protobuf or gRPC library. Individual
modules use ordinary Lean imports within that tree.

The Lean root owns support and inventory namespaces, such as `Temporal.Proto` and
`Temporal.Proto.Generated.Catalog`. Protobuf type declarations continue to derive their concise
namespaces directly from protobuf packages. For example,
`temporal.api.common.v1.Payload` becomes `Temporal.Api.Common.V1.Payload`, not a type nested below
`Temporal.Proto.Generated`. This preserves direct, readable references while allowing the same
generator to use an unrelated root such as `Acme.Model`.

The generated Core module is intentionally small and structural. Its typed `Method Request
Response` declaration is useful for describing gRPC endpoints, but it does not introduce a gRPC
runtime dependency.

## Manifest and lifecycle

The manifest format becomes generic, with the identifier `umpire/protobuf-lean/v1`. It records:

- the format version and configured Lean root;
- each input's name, supplied locator, and SHA-256 digest;
- source groups, normalized prefix rules, and default source;
- a digest of the canonical merged descriptor set;
- file, message, enum, service, and method counts;
- every generated file and its digest, except that the manifest records its own path without a
  recursive self-digest.

Temporal Go module names and versions are removed from the manifest.

`generate` renders and validates the complete artifact map before filesystem mutation. Files are
published atomically one at a time, stale files named by the previous manifest are removed only
after managed-path validation, and the new manifest is published last. A write failure can leave
already published files with the old manifest, which makes the interrupted run detectable by
`check`.

Managed-path validation is derived from the configured Lean root. It permits only the generated
umbrella module, generated support module, and descendants of that root's `Generated` directory;
it never removes files elsewhere in the output root. Changing `--lean-root` establishes a distinct
managed tree and does not implicitly delete the previous root.

`check` compares every expected artifact, reports missing or stale files, and reports files present
in the previous manifest that are no longer expected. Its generic error points to the command that
was run; Temporal's Make target may add the repository-specific `make umpire-gen-api` guidance.

## Generic Go descriptor exporter

The production `go.temporal.io/api` module contains generated Go descriptors but does not ship all
of its source `.proto` files. Descriptor acquisition therefore remains necessary, but it does not
belong inside the Lean generator.

A separate `umpire-export-go-descriptors` command will own the existing temporary-helper strategy.
It requires at least one repeatable Go package pattern, at least one repeatable protobuf file
prefix, and an output path, for example:

```sh
umpire-export-go-descriptors \
  --package-pattern go.temporal.io/api/... \
  --file-prefix temporal/api/ \
  --output proto/umpire-public.binpb
```

The exporter uses `go list` to resolve packages, generates a temporary helper that blank-imports
them, reads their registered `protoreflect.FileDescriptor` values, selects files by prefix, and
writes a deterministic `FileDescriptorSet`. Its command name, flags, diagnostics, and internal
code contain no Temporal policy. The exporter reports an error when no packages or no matching
descriptors are found.

This separation makes `umpire-gen-api` entirely descriptor-driven. It does not remove
`go.temporal.io/api` from Temporal's root module, where the server itself still depends on it.

## Temporal Make integration

The root `Makefile` remains the only Make interface for the model.

- A public-descriptor target invokes `umpire-export-go-descriptors` to produce
  `proto/umpire-public.binpb`.
- Existing targets continue to produce `proto/api.binpb`, `proto/image.bin`, and
  `proto/chasm.bin`.
- `make umpire-gen-api` invokes `umpire-gen-api generate` once with all four descriptors and the
  Temporal source rules shown above.
- `make umpire-check-api` invokes the corresponding `check` command and then runs the model's Lean
  build through Mise from the root recipe.
- No `model/Makefile` is introduced or required.

The generator is invoked once because all inputs participate in one type universe. Three or four
independent runs would each rediscover overlapping dependency closures, risk duplicate Lean
declarations, resolve cross-source names and recursive components without global knowledge, and
compete over stale-file manifests. Source groups are output partitions of one model, not isolated
models.

## Fixture-based tests

The integration fixture will move from a large Go-built `descriptorpb.FileDescriptorSet` to a
small, reviewable source-and-golden layout:

```text
tools/umpire/internal/generate/api/testdata/basic/
  input/
    public/v1/model.proto
    internal/v1/service.proto
    shared/v1/types.proto
    legacy/v1/options.proto
  input.pb
  expected/
    Fixture/Proto/Core.lean
    Fixture/Generated.lean
    Fixture/Generated/Types.lean
    Fixture/Generated/Catalog/External.lean
    Fixture/Generated/Catalog/Internal.lean
    Fixture/Generated/Catalog/Public.lean
    Fixture/Generated/GRPC/External.lean
    Fixture/Generated/GRPC/Internal.lean
    Fixture/Generated/GRPC/Public.lean
    Fixture/Generated/schema.json
    Fixture/Generated/manifest.json
```

The proto sources cover nested messages and enums, same-package and cross-package references,
maps, oneofs, proto3 optional fields, mutual recursion, unary and streaming service methods, source
prefixes and default classification, and the relevant proto2 field metadata.

`input.pb` is checked in so ordinary `go test` does not require `protoc`. Tests load it, run the
same orchestration used by the CLI, and compare the complete artifact set byte-for-byte with
`expected/`. A root `umpire-gen-api-fixture` target uses the Mise-pinned `protoc` with
`--include_imports` to refresh `input.pb` and the goldens. `umpire-check-api-fixture` rebuilds them
in a temporary directory and fails on any diff.

Focused Go tests remain for behavior that is clearer as a table than as a golden:

- repeated-flag parsing and configuration validation;
- descriptor merge deduplication and conflict diagnostics;
- longest-prefix source classification and default classification;
- Lean identifier collisions and configurable Lean roots;
- declaration ordering and strongly connected components;
- malformed or unresolved descriptor references;
- `generate`, `check`, manifest, and stale-file lifecycle behavior;
- Go descriptor exporter package discovery, prefix selection, empty selection, and deterministic
  output.

The current programmatic integration fixture builders are removed once the source fixture covers
their behavior. Full Temporal generation, drift checking, and `lake build` remain the end-to-end
integration verification.

## Error handling

Errors identify the operation and offending value. In particular, the command reports malformed
`NAME=PATH` or `GROUP=PREFIX` values, duplicate input names, unreadable descriptor files, conflicting
protobuf paths, invalid Lean roots, missing default sources, invalid or duplicate group names,
invalid prefix assignments, unknown type references, declaration-plan failures, unsafe managed
paths, and output drift.

All configuration, descriptor, projection, and rendering errors occur before publication begins.
External command failures in the Go descriptor exporter include the failed package pattern and the
captured `go list` or helper diagnostic.

## Alternatives rejected

### Generic core plus a Temporal wrapper command

A second Temporal CLI would mostly translate fixed Temporal defaults into the generic CLI while
duplicating command parsing and lifecycle decisions. The root Make recipe is already the right
adapter for repository-specific paths and source groups.

### A `protoc` plugin

A plugin is conventional for per-file code generation, but this model needs the complete descriptor
universe before it can resolve global Lean names, recursive components, source catalogs, a combined
schema, and one manifest. A descriptor-set command expresses that boundary directly and keeps
`check` mode straightforward.

### One generator invocation per source group

Separate invocations make each output look simpler locally but move the difficult work into import
coordination and aggregation. Shared dependencies would need a new ownership protocol, cross-group
type references would require stable contracts between runs, recursive components could cross run
boundaries, and multiple manifests could not safely manage one output tree. A single invocation
with dynamic source partitions is smaller as a complete system.

## Acceptance criteria

- `umpire-gen-api` has no Temporal package-path, Go-module-discovery, or fixed-source behavior.
- Its complete configuration is expressible with repeatable flags and descriptor-set files.
- One declaration plan drives every generated representation.
- A generated tree builds without a hand-authored protobuf or gRPC support module.
- Small `.proto` inputs and complete expected outputs define the generator's readable contract.
- Temporal's root Make targets generate and check the full public, internal, CHASM, and dependency
  model with one generator invocation and no model-local Makefile.
- Repeated generation is byte-deterministic, `check` detects all managed drift, and stale removal is
  confined to manifest-owned paths.
