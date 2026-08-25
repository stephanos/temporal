# `genleanmodeldescriptors` Migration Design

## Goal

Move the Umpire-owned Go descriptor-set generator to `cmd/tools/genleanmodeldescriptors` with a
hard cutover. Preserve its flags, deterministic output, validation, atomic publication, and failure
behavior while removing Umpire-specific ownership from the command implementation.

## Architecture

`cmd/tools/genleanmodeldescriptors` is one `main` package. `main.go` passes the process context and
arguments to the package-local `Run` implementation and reports failures using the new command name.

The implementation moves from `tools/umpire/internal` into the command package and two focused
helper packages:

- `cmd/tools/genleanmodeldescriptors/run.go` owns argument parsing, package discovery, temporary
  helper generation, descriptor selection, deterministic serialization, and orchestration.
- `tools/common/artifactio` owns atomic artifact publication and removal. The Lean descriptor tool
  and the existing Umpire API generator both use it.
- `tools/common/protofile` owns safe protobuf path-prefix normalization. The Lean descriptor tool
  uses it without returning that generic validation to Umpire ownership.

No other Umpire-internal package moves. Descriptor test fixtures move beside the command under
`cmd/tools/genleanmodeldescriptors/testdata`.

## Data Flow

The command accepts repeatable `--package-pattern` and `--file-prefix` flags plus `--output`.
`Run` normalizes the prefixes, uses `go list` to find generated protobuf packages, and creates a
temporary Go helper that blank-imports the matching packages. The helper reads the
registered protobuf files, includes transitive imports, orders files deterministically, and emits a
serialized `FileDescriptorSet`. `artifactio.Publish` then atomically replaces the requested output.

The root Makefile invokes the new command to generate `proto/umpire-public.binpb`; no compatibility
wrapper remains at the old path.

## Errors and Failure Modes

Existing validation and diagnostics remain intact except that command and temporary-path labels use
`genleanmodeldescriptors`. Invalid flags, unsafe prefixes, empty selections, `go list` failures,
helper failures, and publication failures continue to return errors without publishing partial
output.

Temporary helper files are removed after success or failure. Atomic publication preserves the
existing crash behavior: the destination is either the previous complete artifact or the newly
generated complete artifact. Package discovery remains sequential and deterministic; a larger
package graph increases generation time without changing memory or concurrency semantics.

## Migration

- Add `cmd/tools/genleanmodeldescriptors` and remove the former Umpire command entrypoint.
- Move the exporter implementation and its fixtures into `cmd/tools/genleanmodeldescriptors`.
- Move `artifactio` and `protofile` to `tools/common`, updating the Umpire API generator imports.
- Rename the Makefile command variable and update its invocation.
- Replace old command/path references in current documentation and planning artifacts so repository
  searches no longer advertise the removed command.
- Preserve existing source comments while moving code.

## Testing

The existing exporter tests move first, producing a failing test build before the implementation is
moved. They continue to cover deterministic output, transitive imports, flag validation, empty
selections, package-list failures, and compatibility-copy filtering.

After implementation, run focused tests for `genleanmodeldescriptors`, `artifactio`, `protofile`,
and the affected Umpire generator packages with the `test_dep` build tag. Build or invoke the command
through its Makefile-shaped arguments, confirm the removed command name has no repository
references, and run `make lint-code` for repository standards verification.

## Trade-offs

Co-locating descriptor orchestration with its only consumer avoids a common package with no reuse.
The genuinely generic atomic-write and prefix-validation helpers remain separate, preventing their
duplication. The longer command name is intentionally specific to its Lean-model purpose and avoids
confusion with the existing `umpire-genmodels` command.
