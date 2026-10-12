---
satisfies: [R8]
---
# fn-164-gomad-backend-layout-shared-core-with.8 Keep pre-move artifacts inspectable and make identity mismatch explicit

Touches: tools/gomad3/artifact, tools/gomad3/runner, tools/gomad3/backends

## Description
Collect one retained native artifact and one retained wasm artifact produced before the move (from the qualification corpus or a fresh pre-move build on the parent commit). Verify `inspect` output is unchanged. Where the move changes a model or runtime identity, replay must fail with a message naming the recorded identity and the current one, not a generic decode or mismatch error. Add a fixture for each backend.

## Acceptance
- `gomad inspect` on both pre-move artifacts produces output identical to the pre-move binary's output.
- Replay of a pre-move artifact whose identity changed exits non-zero with both identities in the message; replay of one whose identity did not change succeeds.
- The two fixtures are checked in and run in the ordinary test suite.

## Done summary
Checked-in pre-move native and WASM artifacts run in the ordinary suite; inspect output matches goldens produced by the pre-move binary and identity changes fail with recorded and current identities.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 34fefe68ad, 8991a402e1, 12b306c77a, e807723f35, 8e249316e3
- Tests: make -C tools/gomad3 test-wasm (0); root architecture selection with GOMAD3_ARCH_ALL_PLATFORMS=1 (0); make -C tools/gomad3 validate (0); make lint-code-fast base e43c7a59c7 (0), evidence: .flow/artifacts/wasm-section-acceptance-2026-10-12.md
- PRs: