# Temporal Lean model

Lean owns behavioral meaning in this directory. `umpire-gen-api` imports protobuf and gRPC
descriptor structure into `Temporal/Generated`; generated declarations do not assign product
semantics to fields or RPCs.

- `Types.lean` contains structural message, enum, map, oneof, presence, and recursion projections.
- `Catalog/*.lean` groups file, enum, message, and field descriptors by public, internal, CHASM,
  and external source.
- `GRPC/*.lean` provides typed method descriptions and complete service inventories. It does not
  contain an RPC client or server runtime.
- `schema.json` exposes the same inventory to tools, while `manifest.json` binds it to normalized
  descriptor inputs and generated-file digests.

Bytes and recursive links are deliberately bounded abstractions. Arbitrary custom protobuf option
values are covered by the descriptor digest but are not interpreted as product semantics; model
families must explicitly interpret the wire metadata they use.

Run `make umpire-gen-api` from the Temporal repository root after changing public, internal, or
CHASM APIs. Run `make umpire-check-api` to check generated drift and build the Lean model.
