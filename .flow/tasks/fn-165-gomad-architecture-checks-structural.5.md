---
satisfies: [R5, R9]
---
# fn-165-gomad-architecture-checks-structural.5 Mark the gomad3 pure set and switch the module tests to the new checkers

Touches: tools/gomad3/world, tools/gomad3/record, tools/gomad3/architecture_test.go, tools/gomad3/architecture_purity_test.go, tools/gomad3/choice, tools/gomad3/internal/compatibilitypack, tools/gomad3/backends/native/execution

## Description
Run `puritycheck` over the gomad3 pure set (`world`, `record`, `world/mailbox`, `runner/internal/exploration`, `target/internal/capabilitypolicy`, `internal/canonicaljson` while it exists, and the three single pure files). Add a dated review mark with a reason to every dynamic call it reports; do not change pure-set membership and do not restructure code to avoid a mark. Rewire `TestPureModulesHaveNoHostEffects` and `TestPublicPackagesDoNotExportTypeAliases` to `puritycheck`, running the host platform always and the other qualified platform when `GOMAD3_ARCH_ALL_PLATFORMS=1`. Record the mark count per package in the task notes.

Applied clarification: preserve the original pure roots and explicitly check the reviewed pure dependency files identified by the first real bounded pass, as recorded in the spec's Decision Context. Split pure declarations from mixed host-loader files only with unchanged bodies/APIs. Such static host-effect findings cannot be bypassed with dynamic marks. Paths follow fn-164's moves; exact membership/dependency inventory must fail closed on missing files.

## Acceptance
- `go test -run 'TestPureModulesHaveNoHostEffects|TestPublicPackagesDoNotExportTypeAliases' ./` passes on the host platform with zero `host-effect` and zero unmarked `dynamic-call` findings, and the log lists every mark.
- The same run with `GOMAD3_ARCH_ALL_PLATFORMS=1` passes for darwin/arm64 and linux/amd64.
- Both tests complete in under 60 seconds combined on the host platform.
- No file was removed from the pure set and no function body was changed other than adding mark comments.

## Done summary
The gomad3 pure set passes puritycheck with zero host-effect and zero unmarked dynamic-call findings on host and both qualified platforms (about 21s combined); review marks are logged per package; pure-set membership and function bodies are unchanged apart from marks.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: c6afac5097, cae07fe76f, 016ec47b74
- Tests: root architecture/purity selection with GOMAD3_ARCH_ALL_PLATFORMS=1 plus ./internal/archcheck ./internal/puritycheck (0), evidence: .flow/artifacts/wasm-section-acceptance-2026-10-12.md
- PRs: