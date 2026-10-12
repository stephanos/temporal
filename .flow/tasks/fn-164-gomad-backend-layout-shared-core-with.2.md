---
satisfies: [R2]
---
# fn-164-gomad-backend-layout-shared-core-with.2 Create runtimesemantics and make both overlays include it verbatim

Touches: tools/gomad3/runtimesemantics, tools/gomad3/toolchain/runtime, tools/gomad_wasm/toolchain/overlay.go

## Description
Move the 27 functions and 2 types that `tools/gomad_wasm/toolchain/overlay.go` extracts by AST from the native `runtime/gomad.go`, plus `gomad_choicewire_generated.go`, into `tools/gomad3/runtimesemantics` as files that compile inside package `runtime`. The native overlay and the wasm overlay builder both copy these files in byte for byte; the wasm builder stops parsing the native source. Each overlay's identity hash covers the copied bytes. Add a native inventory test that fails when an exported function or type the wasm overlay needs is missing or renamed.

## Acceptance
- `overlay.go` contains no `go/parser` or `go/ast` use against `runtime/gomad.go`.
- Changing one byte in a `runtimesemantics` file changes both overlays' identity hashes, and a stale copy in either overlay fails that overlay's identity test naming the file.
- Renaming one of the 27 functions in `runtimesemantics` fails the native inventory test naming it.
- Existing wasm cooperative guest tests and native toolchain overlay tests pass unchanged in behavior.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
