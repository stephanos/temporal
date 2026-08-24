# Temporal Lean model

Lean owns behavioral meaning in this directory. `umpire-gen-api` imports protobuf and gRPC
descriptor structure into `Temporal/Generated`; generated declarations do not assign product
semantics to fields or RPCs.

- `Types.lean` contains structural message, enum, map, oneof, presence, and recursion projections in
  namespaces derived from protobuf packages. For example,
  `temporal.server.api.adminservice.v1.DescribeMutableStateRequest` is
  `Temporal.Server.Api.Adminservice.V1.DescribeMutableStateRequest`; nested protobuf declarations
  retain the same nesting in Lean.
- `Catalog/*.lean` groups file, enum, message, and field descriptors by public, internal, CHASM,
  and external source.
- `GRPC/*.lean` provides typed method descriptions in each service's protobuf-derived namespace and
  complete service inventories. Same-package message references are short, while cross-package
  references are fully qualified. These files contain no RPC client or server runtime.
- `schema.json` exposes the same resolved names and inventory to tools. Every `leanName`, including
  fields, enum values, and methods, is fully qualified; oneofs reference their canonical fields by
  protobuf full name rather than copying field records. `manifest.json` uses the
  `umpire/temporal-api/v3` format and binds the model to normalized descriptor inputs and
  generated-file digests.

Bytes and recursive links are deliberately bounded abstractions. Arbitrary custom protobuf option
values are covered by the descriptor digest but are not interpreted as product semantics; model
families must explicitly interpret the wire metadata they use.

Run `make umpire-gen-api` from the Temporal repository root after changing public, internal, or
CHASM APIs. Run `make umpire-check-api` to check generated drift and build the Lean model.

## Bounded regression experiment

The Lean-first declaration for the bounded Nexus caller-closure pilot is
`Temporal/Experiment/NexusCallerClosure.lean`. It names the regression's resources, requested
action attempts, ordering, expected properties, and finite declaration bounds. The selected
`ModelTarget` in the same module resolves the setup and projects each requested action through the
Lean-owned model semantics; requesting an action does not itself establish a successful semantic
transition.

From the Temporal repository root, build the full model or run the focused regression check:

```sh
make -C model check
make -C model check-regression
```

The focused check builds the pure compiler, pilot, compile-time positive and negative fixtures, and
inspector. It also checks that repeated inspection is byte-for-byte deterministic and that a
rejected pilot emits one structured diagnostic with no JSON on standard output. Neither command
requires a running Temporal server.

Inspect the checked pilot directly with:

```sh
cd model
mise exec -- lake exe temporal-experiment-inspect nexus-caller-closure-upgrade
```

On success the inspector writes one canonical JSON `ExperimentSpec` to standard output. Its stable
contract contains the format version, regression and target identities, derived model identity,
declared resources and resolved setup, requested action attempts and projected model outcomes,
ordering, expected properties and their observation contracts, declaration bounds, omissions, and
provenance. The compiler and inspector do not write an artifact file or contact a runtime.

Generated API declarations remain structural inputs only. Behavioral meaning, including whether a
requested attempt is applicable and which transition outcome it produces, remains owned by the
authored Lean model.
