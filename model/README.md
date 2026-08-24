# Temporal Lean model

Lean owns behavioral meaning in this directory. `umpire-gen-api` consumes serialized protobuf
descriptor sets and projects their protobuf and gRPC structure behind the stable `Temporal.API`
module boundary. Generated declarations do not assign product semantics to fields or RPCs.

The generator exclusively owns `Temporal/API.lean` and the complete `Temporal/API/` directory:

- `API/Proto.lean` contains the runtime-independent `Bytes`, `MessageRef`, and typed `Method`
  support structures.
- `API/Types.lean` contains structural message, enum, map, oneof, presence, and recursion
  projections. Namespaces continue to derive from protobuf packages; for example,
  `temporal.server.api.adminservice.v1.DescribeMutableStateRequest` becomes
  `Temporal.Server.Api.Adminservice.V1.DescribeMutableStateRequest`.
- `API.lean` imports both child modules and declares every typed RPC in its protobuf-derived
  service namespace. Same-package message references are short and cross-package references are
  qualified. These declarations do not provide an RPC client or server runtime.

Bytes and recursive links remain deliberately bounded abstractions. The generator does not
interpret arbitrary protobuf options as product semantics; authored model families explicitly
interpret the structural metadata they use.

The repository root `Makefile` is the only Make interface for this model. After changing public,
internal, or CHASM APIs, regenerate the descriptor-backed modules and verify them locally:

```sh
make umpire-gen-api
go test -count=1 -tags test_dep ./tools/umpire/internal/generate/api
cd model && mise exec -- lake build
```

Generation is deterministic and silent on success. Each run validates all three artifacts and
their output paths before mutation, then replaces the owned API outputs while preserving adjacent
authored modules.

## Bounded regression experiment

The Lean-first declaration for the bounded Nexus caller-closure pilot is
`Temporal/Experiment/NexusCallerClosure.lean`. It names the regression's resources, requested
action attempts, ordering, expected properties, and finite declaration bounds. The selected
`ModelTarget` in the same module resolves the setup and projects each requested action through the
Lean-owned model semantics; requesting an action does not itself establish a successful semantic
transition.

From the Temporal repository root, run the focused regression check:

```sh
make umpire-check-regression
```

The focused check builds the pure compiler, pilot, compile-time positive and negative fixtures,
and inspector. It also checks that repeated inspection is byte-for-byte deterministic and that a
rejected pilot emits one structured diagnostic with no JSON on standard output. It does not
require a running Temporal server.

Inspect the checked pilot directly with:

```sh
cd model
mise exec -- lake exe temporal-experiment-inspect nexus-caller-closure-upgrade
```

On success the inspector writes one canonical JSON `ExperimentSpec` to standard output. Its
stable contract contains the format version, regression and target identities, derived model
identity, declared resources and resolved setup, requested action attempts and projected model
outcomes, ordering, expected properties and their observation contracts, declaration bounds,
omissions, and provenance. The compiler and inspector do not write an artifact file or contact a
runtime.

Generated API declarations remain structural inputs only. Behavioral meaning, including whether
a requested attempt is applicable and which transition outcome it produces, remains owned by the
authored Lean model.
