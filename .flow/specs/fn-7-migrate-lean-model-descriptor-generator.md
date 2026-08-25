# Migrate Lean model descriptor generator

> HTML render lens (local): `.flow/artifacts/fn-7-migrate-lean-model-descriptor-generator/spec.html` — regenerable, markdown is the record. <!-- flow-next:artifact-link -->

## Overview

Move the descriptor-set acquisition command into Temporal's standard generator-tool surface under
the approved `genleanmodeldescriptors` name. Extract its reusable implementation and the two helper
packages shared with Lean API generation into deep, independently tested common-tool modules.

## Goal & Context
<!-- scope: business -->

Developers generating the checked-in Lean model should encounter one consistently named `gen*`
tool rather than an Umpire-owned exporter. The migration is a hard cutover: repository generation
uses the new command, generic functionality has generic ownership, and no obsolete wrapper or
documentation remains.

## Architecture & Data Models
<!-- scope: technical -->

The command remains a thin process adapter over a common descriptor-generation module. That module
continues to discover registered Go protobuf packages, generate and execute a temporary helper,
collect matching descriptors and transitive imports, serialize deterministically, and publish the
result atomically.

Atomic artifact publication and protobuf-prefix normalization become separate common-tool modules.
The Lean descriptor command and the Umpire Lean API generator share those modules without
duplicating their validation or filesystem behavior.

## API Contracts
<!-- scope: technical -->

- The command is named `genleanmodeldescriptors` and retains repeatable `--package-pattern` and
  `--file-prefix` flags plus required `--output`.
- The descriptor module retains `Run(context.Context, []string) error`.
- Atomic publication retains same-directory temporary files, durable sync, `0700` directory
  creation, `0600` temporary files, idempotent removal, and the existing wrapped errors.
- Prefix normalization retains whitespace and separator normalization, trailing-slash semantics,
  and rejection of empty, absolute, current-directory, and parent-traversing prefixes.
- Diagnostics remain unchanged except for the renamed executable, flag set, and generic temporary
  path labels.

## Edge Cases & Constraints
<!-- scope: technical -->

Repeated package patterns and prefixes remain deterministic and deduplicated. Compatibility-copy
packages that do not contain the requested protobuf source prefix remain excluded. Empty
selections, invalid flags, unsafe paths, package-list failures, helper failures, cancellation, and
publication failures do not report a partial artifact as successful. Temporary-helper cleanup and
artifact publication retain their existing wrapped-error behavior.

Concurrent writers retain the existing last-writer-wins behavior. The checked-in descriptor artifact
is compared with freshly generated output but is not rewritten when byte-identical. Existing source
comments are preserved during every move.

## Quick commands

```bash
go test -count=1 -tags test_dep ./tools/common/artifactio ./tools/common/protofile ./tools/common/godescriptors ./tools/umpire/internal/generate/api ./cmd/tools/genleanmodeldescriptors
go build -tags test_dep ./cmd/tools/genleanmodeldescriptors
make lint-code
```

## Acceptance Criteria
<!-- scope: both -->

- **R1:** Repository generation invokes `genleanmodeldescriptors` through the standard command-tool
  location, with the existing flags and descriptor artifact unchanged, and the former entrypoint is
  removed without a compatibility wrapper. Errors: missing or invalid flags, unexpected positional
  arguments, and direct invocation failures remain non-zero and use the new command prefix.
- **R2:** Descriptor generation, atomic artifact I/O, and protobuf-prefix normalization live in
  focused common-tool modules, and the Lean API generator consumes the shared artifact and prefix
  modules. Errors: old duplicate packages are absent; invalid artifact paths and unsafe prefixes
  retain their existing rejection behavior.
- **R3:** Generated descriptor sets remain byte-deterministic, include matching files and transitive
  imports, and publish atomically with existing permissions, cleanup, and error semantics. Errors:
  empty selections, package-list failures, helper failures, cancellation, and controlled publication
  failures leave an existing destination intact; lower-level close, sync, rename, and cleanup errors
  retain the existing propagation and joining code rather than being redesigned during the move.
- **R4:** Existing source comments are preserved and live code, build integration, current design
  documentation, and the prior implementation plan contain no obsolete command name or path.
  Migration records may name removed paths as historical inputs. Errors: no runtime error surface
  beyond R1-R3; a repository search scoped outside migration records must identify no stale live
  reference.

## Boundaries
<!-- scope: business -->

- Do not rename or rewrite the checked-in public descriptor artifact when its content is unchanged.
- Do not change descriptor-selection flags, output format, generated Lean semantics, or API-generator
  behavior.
- Do not add compatibility wrappers, concurrency locking, signal-handling redesign, drift checks,
  CI workflows, or new Make verification targets.
- Do not consolidate unrelated Umpire3 artifact helpers.

## Decision Context
<!-- scope: both — conditionally substructured -->

Common modules preserve the existing deep `Run` boundary and avoid copying security-relevant prefix
validation or durable artifact publication into a shallow command package. `genleanmodeldescriptors`
is intentionally longer than `genmodeldescriptors` to distinguish this protobuf input stage from the
existing model generators. The migration waits for the active Lean API output simplification because
both changes touch its imports, root generation wiring, and current design documentation. A wrapper
was rejected because the requested hard cutover has only one in-repository caller. New drift or CI
verification was rejected as previously declined scope.

## Early proof point

Task fn-7-migrate-lean-model-descriptor-generator.1 proves that the shared artifact and prefix
modules preserve their contracts while the Lean API generator consumes them. If it fails, reconsider
the common-module boundary before migrating the descriptor command in task 2.

## References

- Approved `genleanmodeldescriptors` migration design (2026-08-24)
- `fn-6-plan-simplify-lean-api-generator-output` for the prerequisite Lean API simplification
- Declined generated-API drift-verification decision for the explicit CI boundary

## Requirement coverage

| Req | Description | Task(s) | Gap justification |
|-----|-------------|---------|-------------------|
| R1 | Hard-cutover command and build integration | fn-7-migrate-lean-model-descriptor-generator.2 | — |
| R2 | Common deep modules and shared consumers | fn-7-migrate-lean-model-descriptor-generator.1, fn-7-migrate-lean-model-descriptor-generator.2 | — |
| R3 | Deterministic and failure-safe behavior parity | fn-7-migrate-lean-model-descriptor-generator.1, fn-7-migrate-lean-model-descriptor-generator.2 | — |
| R4 | Comment preservation and stale-reference cleanup | fn-7-migrate-lean-model-descriptor-generator.2 | — |
