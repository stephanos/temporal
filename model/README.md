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

## Semantic authoring and planning

Model scenarios use three concise forms. A `Property` describes portable meaning over a
capability-limited semantic trace. A `Behavior` constrains setup, controllable actions, ordering,
and exactness while leaving outcomes to the target model. A `Query` combines checked properties
and behavior with explicit bounds and a deterministic planning policy.

`Temporal/Experiment/NexusCallerClosure.lean` applies that flow to Workflow–Nexus caller closure.
`Temporal/Experiment/SwitchScenario.lean` uses the same public interfaces for an independent
two-state switch, including exploratory, exact-action, and exact-trace authoring. The resulting
`DrivePlan` and `ExperimentSpec` values are pure model artifacts: they describe selected requests,
model-owned outcomes, and semantic observations, but do not claim that runtime execution occurred.

From the Temporal repository root, run the focused regression check:

```sh
make umpire-check-regression
```

The focused check builds the semantic languages, both scenarios, aggregate positive and negative
fixtures, and inspector. It checks that repeated inspection is byte-for-byte deterministic and
that a rejected scenario emits one structured diagnostic with no artifact JSON on standard
output. It does not require or contact a running Temporal server.

Inspect either checked scenario directly with:

```sh
cd model
mise exec -- lake exe temporal-experiment-inspect workflow-nexus.query.exact-action-caller-closure
mise exec -- lake exe temporal-experiment-inspect switch.query.exact-action
```

On success the inspector writes one canonical JSON `ExperimentSpec` to standard output. The
compiler and inspector do not write an artifact file, start a live server, execute a workflow, or
collect evidence. Runtime driving, evidence qualification, and promotion are separate work.

Generated API declarations remain structural inputs only. Behavioral meaning, including whether
a selected action is applicable and which transition outcomes are possible, remains owned by the
authored Lean model.
