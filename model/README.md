# Temporal Lean model

Lean owns behavioral meaning in this directory. `umpire-gen-api` imports protobuf and gRPC
descriptor structure into `Temporal/Generated`; generated declarations do not assign product
semantics to fields or RPCs. The complete generated catalog is an inventory: each model family must
still select and explicitly interpret its bounded wire surface.

Run `make umpire-gen-api` from the Temporal repository root after changing public, internal, or
CHASM APIs. Run `make umpire-check-api` to check generated drift and build the Lean model.
