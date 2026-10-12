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
Import-direction checker enforces shared/backends/cmd edges with per-edge allowlist entries, stale and missing-entry detection and the WASM module-edge rule; fixtures cover forbidden, deleted, stale and WASM-outside cases. Post-fold allowlists for the retired standalone WASM module were removed in fn-165.6.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: d42190ccb4, ed29f1cdea, 2bb9a8b3e4, 016ec47b74, d36476f98a
- Tests: make -C tools/gomad3 test-wasm (0); root architecture selection with GOMAD3_ARCH_ALL_PLATFORMS=1 (0); make -C tools/gomad3 validate (0); make lint-code-fast base e43c7a59c7 (0), evidence: .flow/artifacts/wasm-section-acceptance-2026-10-12.md
- PRs: