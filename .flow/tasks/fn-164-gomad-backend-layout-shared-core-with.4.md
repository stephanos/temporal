---
satisfies: [R5]
---
# fn-164-gomad-backend-layout-shared-core-with.4 Split target into shared types and native build preparation

Touches: tools/gomad3/target, tools/gomad3/backends/native/build

## Description
Leave `Spec`, `Prepared` and the capability types in `target`. Move toolchain-bound preparation, build adapters, live capability collection and `target/internal/capabilitypolicy` consumers into `backends/native/build`. `target` must import no `backends/*` package after this task. Pure file `target/capability_evaluation.go` either stays pure in `target` or moves whole to `backends/native/build`; do not split its body.

## Acceptance
- `go list -deps ./target` shows no `backends/`, `toolchain` or `upgrade` package.
- The Provider interface still compiles against `target.Spec` and `target.Prepared` from both backends.
- fn-164.1's allowlist has no `target` entry.
- Capability review and prepared-inspection tests pass from their new location.

## Done summary
target keeps shared Spec/Prepared/capability types and imports no backends package; native build preparation lives in backends/native/build.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 1c69376cbd, e0583f0aca
- Tests: make -C tools/gomad3 test-wasm (0); root architecture selection with GOMAD3_ARCH_ALL_PLATFORMS=1 (0); make -C tools/gomad3 validate (0); make lint-code-fast base e43c7a59c7 (0), evidence: .flow/artifacts/wasm-section-acceptance-2026-10-12.md
- PRs: