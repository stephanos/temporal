---
satisfies: [R1]
---
# fn-164-gomad-backend-layout-shared-core-with.1 Enforce the shared-versus-backend import direction on the current tree

Touches: tools/gomad3/architecture_test.go, tools/gomad3/internal/archcheck

## Description
Configure the structural checker from fn-165 (or, if fn-165.1 has not landed, a `go list` import test in `tools/gomad3/architecture_test.go`) with the fn-164 dependency direction: shared packages never import `backends/*`, backends never import each other, `cmd` may import both. Because nothing has moved yet, express today's backend-bound packages by name (`toolchain`, `upgrade`, `deterministicio` root, `runner/internal/execution`, `target` build and adapters, conformance and generation tooling) and keep an allowlist of the shared packages that still import them, one entry per edge with the task that removes it. Add the allowlist of gomad3 packages the wasm module may import, taken from its current import set.

## Acceptance
- The test passes on the current tree with every existing violation in the allowlist and no wildcard entries.
- A new import from `world` into `toolchain` fails the test naming both packages.
- Deleting an allowlist entry whose edge still exists fails the test naming the entry; an entry whose edge is gone fails the test as stale.
- The wasm module importing a gomad3 package outside its allowlist fails the test.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
