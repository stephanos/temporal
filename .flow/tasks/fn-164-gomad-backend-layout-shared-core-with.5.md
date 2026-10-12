---
satisfies: [R6]
---
# fn-164-gomad-backend-layout-shared-core-with.5 Make native a Provider behind backends/backend.go

Touches: tools/gomad3/backends/backend.go, tools/gomad3/backends/native, tools/gomad3/runner

## Description
Create `backends/backend.go` with `Provider` (today's four methods plus `Capabilities() Capabilities` as specified) and a registry. Move `runner/internal/execution`, `runner/coordinator_process_unix.go` and the native branches of `runner.go`, `backend.go` and `replay_operation.go` into `backends/native`, which implements `Provider`. `runner` takes a `Provider` and keeps no `config.Backend == nil` default path. If a native path needs a Provider method beyond those, add it and record the reason in fn-164's Decision Context.

## Acceptance
- `go list -deps ./runner` contains no `backends/` package.
- `TestRunnerExecutionInjectionIsPrivate` and the external Runner consumer compile test still pass.
- Native campaign, replay and minimize integration tests pass through the Provider path with artifacts identical in content to a pre-change run on the same seed.
- Any Provider method beyond `Capabilities` plus three is listed in the spec's Decision Context with its reason.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
