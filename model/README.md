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
