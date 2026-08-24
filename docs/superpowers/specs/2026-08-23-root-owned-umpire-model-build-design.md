# Root-owned Umpire model build

## Status

Superseded in place by the generation-only three-module API contract.

## Goal

Keep the repository root `Makefile` as the sole Make interface for generating the Temporal Lean
API model. Keep compilation as a direct Lake command so generation and verification remain
separate, explicit operations.

## Design

The root recipe acquires all required descriptor sets and invokes `umpire-gen-api` once. The
generator validates and replaces its exact API module boundary. The `model/` directory remains a
normal Lake project whose `lean-toolchain`, `lakefile.toml`, and Mise pin define compilation.

## Verification

Run these commands from the repository root:

```sh
make umpire-gen-api
go test -count=1 -tags test_dep ./tools/umpire/internal/generate/api
cd model && mise exec -- lake build
```

No model-local Makefile, generated-artifact comparison subcommand, or additional drift wrapper is
part of the current contract.
