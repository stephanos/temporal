# GoMaD v2 Entrypoint Design

## Goal

Make the GoMaD v2 nested module own its executable entrypoint. Remove the
repository-level `cmd/tools/gomad` wrapper so the command can be built and
installed entirely from `tools/gomad2`.

## Package layout

Add `tools/gomad2/cmd/gomad/main.go` as `package main`. It will delegate to
`gomadmain.Main`, preserving `gomadmain` as the importable, independently tested
command implementation. The nested module root remains the public
`github.com/temporalio/gomad` library package; Go does not allow that directory
to contain both library and command packages.

Delete `cmd/tools/gomad`. No root-module package will import the nested command
implementation after the cutover.

## Build and install flow

Build and install instructions will target `./cmd/gomad` from the nested module.
Repository-root commands will use `go -C tools/gomad2`; `tools/gomad2/test.sh`
will build the command directly within its module. The executable name remains
`gomad`, and its flags, output directory, cache behavior, and exit behavior do
not change.

## Error handling

The new entrypoint has no additional error boundary: `gomadmain.Main` retains
the existing parsing, logging, and process-exit behavior. Documentation and
runtime diagnostics will point to the nested-module install command so a
missing executable produces actionable instructions.

## Verification

Focused verification will:

- build `./cmd/gomad` from `tools/gomad2` with `-tags test_dep`;
- run the existing `gomadmain` unit tests;
- run the focused Go 1.26 translator tests; and
- rebuild the translator before resuming the selected Temporal simulation test.

No broad `go test ./...` invocation is required for this package-layout change.

## Trade-offs and failure modes

Keeping `gomadmain` adds one small delegation layer but preserves its unit-test
boundary and avoids turning the public module root into a command. Removing the
root wrapper makes the nested module self-contained and prevents its command
from depending on the parent repository layout. Builds from the old path fail
immediately after cutover, so all tracked references must change in the same
commit.
