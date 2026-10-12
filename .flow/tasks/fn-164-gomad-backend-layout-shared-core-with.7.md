---
satisfies: [R7]
---
# fn-164-gomad-backend-layout-shared-core-with.7 One cmd/gomad launcher driven by Provider capabilities

Touches: tools/gomad3/cmd/gomad, tools/gomad3/cmd/gomad/api

## Description
Collapse `cmd/gomad`, `cmd/gomad/api` and the `gomad-wasm` launcher into one `cmd/gomad` that selects the backend from the installation manifest and registers commands and flags from `Capabilities()`. Remove every per-backend flag reject list (for example the wasm semantic-coverage and network/storage rejections) in favor of flags not being registered. Unknown flag exits 2 with the standard usage message; a manifest naming an unregistered backend exits 2 naming it.

## Acceptance
- `gomad --help` under a wasm installation shows no native-only flags, and passing one exits 2 with the usage message.
- A manifest with backend `nope` exits 2 with a message containing `nope`.
- No `reject` list keyed by backend name remains under `cmd/`.
- Existing CLI golden tests for both backends pass with only path and help-text updates.

## Done summary
One cmd/gomad launcher registers commands and flags from Provider capabilities; native installation precedence and top-level grammar are preserved; choice-only coverage values come from Runner.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: e6bead51b3, 8d584361a7, 956331b83a
- Tests: make -C tools/gomad3 test-wasm (0); root architecture selection with GOMAD3_ARCH_ALL_PLATFORMS=1 (0); make -C tools/gomad3 validate (0); make lint-code-fast base e43c7a59c7 (0), ./cmd/gomad/internal/cli launcher tests (native-host failures only), evidence: .flow/artifacts/wasm-section-acceptance-2026-10-12.md
- PRs: