---
satisfies: [R3]
---
# fn-164-gomad-backend-layout-shared-core-with.3 Move toolchain, upgrade, native transcript and conformance tooling under backends/native

Touches: tools/gomad3/backends/native, tools/gomad3/toolchain, tools/gomad3/upgrade, tools/gomad3/deterministicio

## Description
Move `toolchain` to `backends/native/toolchain`, `upgrade` to `backends/native/upgrade`, the `deterministicio` root package to `backends/native/transcript`, and `internal/gomadtool/{conformance,generation}` under `backends/native`. `deterministicio/volume` becomes `volume` and `deterministicio/readonlymount` becomes `captured` (or keep their names if the rename is contentious; record the choice). Update every importer with `gofmt -r` or `gopls rename` and remove the corresponding allowlist entries from fn-164.1. Keep the Makefile `.toolchain` build path and the `GOMAD3_STOCK_GO` conventions working.

## Acceptance
- No package outside `backends/native` and `cmd` imports any of the moved packages; fn-164.1's allowlist has no entry for them.
- `make -C tools/gomad3 build` and the native toolchain tests pass; wasm tests pass against the renamed `volume` and read-only mount packages.
- `git log --follow` on three moved files shows history preserved (moves, not copies).

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
