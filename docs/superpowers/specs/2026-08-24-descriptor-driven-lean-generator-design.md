# Descriptor-driven protobuf-to-Lean generation

## Status

Implemented on 2026-08-24.

## Context

`umpire-gen-api` is a general descriptor-set-to-Lean generator. Repository-specific descriptor
acquisition remains in the root `Makefile`; the generator owns only structural protobuf and RPC
projection. Authored Lean modules retain behavioral meaning.

Maintainer clarity is the primary design constraint. One run has one operation, one semantic
module boundary, and one three-file output contract. The previous source partitions, inventories,
machine-readable schema, persisted generation state, artifact digests, and reporting modes have no
consumer and are not part of the current design.

## Command-line contract

A Temporal invocation has this shape:

```sh
umpire-gen-api \
  --descriptor proto/umpire-public.binpb \
  --descriptor proto/api.binpb \
  --descriptor proto/image.bin \
  --descriptor proto/chasm.bin \
  --lean-root Temporal \
  --output-root model
```

`--descriptor PATH` is required and repeatable. Locators are slash-normalized, sorted, and must be
unique. `--lean-root MODULE` and `--output-root PATH` are also required. There is no positional
operation and successful generation writes nothing to standard output.

Relative descriptor paths are resolved from the working directory. Multiple descriptor sets are
merged by protobuf file path. Semantically equal files are deduplicated; conflicts identify the
protobuf file and both supplied descriptor paths. The merged file set is sorted before descriptor
indexing, and declaration names are resolved deterministically.

## Architecture

A run has six stages behind one orchestration boundary:

1. Parse and validate all command arguments.
2. Load and decode every descriptor set.
3. Merge files into one deterministic descriptor graph.
4. Build a neutral projection and one global Lean declaration plan.
5. Render and validate the complete three-artifact map in memory.
6. Validate filesystem targets, reset the owned output, and publish in dependency order.

Protobuf reflection remains separated into projection, Lean naming and type planning, and textual
rendering. The declaration plan resolves package namespaces, nested declarations, name collisions,
message dependency order, recursive components, fields, oneofs, and typed service methods before
any renderer runs.

The generator does not infer product behavior, implement wire serialization, or provide an RPC
transport.

## Generated module boundary

For `--lean-root Temporal --output-root model`, the exact generated contract is:

```text
model/Temporal/API.lean
model/Temporal/API/Proto.lean
model/Temporal/API/Types.lean
```

`Temporal.API` is the supported consumer entry point. The child modules are ordinary Lean modules
but are generated implementation details.

`Proto.lean` contains only `Bytes`, `MessageRef`, and `Method Request Response`. Their existing
fields and deriving clauses remain intact. `Types.lean` imports the support module and contains all
protobuf-derived types. `API.lean` imports both child modules and contains every typed RPC
declaration.

Lean module paths and protobuf-derived namespaces are independent. Moving a type into
`Temporal/API/Types.lean` does not nest its declaration below `Temporal.API`; for example,
`temporal.api.common.v1.Payload` remains `Temporal.Api.Common.V1.Payload`.

Each method declaration retains its request type, response type, full protobuf name, streaming
flags, deprecation flag, and deterministic collision-resolved Lean name. Same-package references
remain short while cross-package references are qualified.

## Exclusive publication

The generator exclusively owns `<root>/API.lean` and the complete `<root>/API/` directory. It does
not manage authored siblings beneath the Lean root.

All artifacts and paths are validated before mutation. Publication rejects maps that are not the
exact three-file contract, lexically escaping targets, non-directory ancestors, and ancestor
symlinks that resolve outside the configured output root. The owned umbrella and directory are
then removed regardless of whether stale entries are files, directories, or symlinks.

Files publish atomically one at a time in dependency order:

1. `Proto.lean`
2. `Types.lean`
3. `API.lean`

An interruption can leave an incomplete disposable API tree. Rerunning generation reconstructs
the entire owned output. Repeated unchanged runs are byte-identical and remove unexpected entries
from the owned directory.

## Temporal repository integration

The root `Makefile` acquires the public, dependency, internal, and CHASM descriptor sets and passes
their paths directly to one generator invocation. The generator remains unaware of Temporal
package policy.

The local verification sequence is:

```sh
make umpire-gen-api-fixture
go test -count=1 -tags test_dep ./tools/umpire/internal/generate/api
make umpire-gen-api
cd model && mise exec -- lake build
make umpire-check-regression
```

The fixture target rebuilds the source-backed descriptor set and rewrites the exact three-file
golden tree. No separate drift wrapper or workflow integration is part of this design.

## Test strategy

Focused Go tests cover path-only flag parsing, duplicate locators, malformed inputs, semantic
descriptor deduplication, conflict diagnostics, declaration ordering, recursion, collision
handling, support qualification, same-package and qualified method types, exact rendering, output
containment, pre-mutation validation, full reset, sibling preservation, dependency publication
order, idempotence, and deterministic output.

The source-backed fixture covers nested messages and enums, maps, oneofs, presence, proto2 field
metadata, mutual recursion, unary methods, streaming methods, and the exact generated tree. The
checked-in Temporal output and full Lake build provide the end-to-end integration boundary.

## Error handling

Configuration, descriptor, projection, planning, rendering, artifact-map, and filesystem-target
errors occur before publication begins. Diagnostics identify the offending descriptor locator,
protobuf file, Lean root, generated relative path, or resolved filesystem target.

Removal and publication errors identify the exact generated relative path that failed. Because the
owned tree is disposable, a later successful generation repairs an interrupted publication.

## Boundaries

- Descriptor acquisition and Temporal package selection remain root-Makefile concerns.
- Generated declarations remain structural inputs only.
- Protobuf serialization, wire codecs, runtime reflection, and RPC clients or servers are outside
  scope.
- Compatibility aliases, configurable output layouts, partitions, inventories, JSON projections,
  persisted state, reporting operations, drift wrappers, and workflow changes are outside scope.
