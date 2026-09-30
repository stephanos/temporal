---
satisfies: [R4]
---
# fn-102-gomad-architecture-consolidate.4 Move inaccessible public executor injection behind private dependencies

## Description
R4. Executor and ReplayExecutor expose runner/internal/execution.Spec and Result, which external callers cannot import. Inventory all repository callers before editing, including sibling modules. Remove only unusable executor interfaces/fields from public request types; keep usable Preparer and ArtifactReplayer contracts. Thread private dependencies through internal operation entry points, including resume/shard/minimize delegation, with production defaults in exported entry points. Migrate fake executors atomically. Add an external-consumer compilation test using a separate package outside the runner subtree. Explicitly document source compatibility change in task 6; do not export execution internals or add a forwarding public interface. Preserve existing comments. No new dependencies. Run focused commands from tools/gomad3 with GOWORK=off and -tags test_dep; use the patched toolchain for runtime-consumer tests.

**Size:** M

**Touches:** [tools/gomad3/runner/**, tools/gomad3/qualification/**, tools/gomad3/cmd/gomad/**] — WIDER: migrate discovered in-module callers; expand declared scope before editing any newly discovered sibling consumer.

**Files:** `tools/gomad3/runner/{runner.go,replay_operation.go,resume.go,campaign_shard_execution.go,minimize_operation.go}`; private dependency wiring; corresponding runner/replay/minimize/resume/plan tests.

### Quick commands

`go test -tags test_dep ./runner/... ./qualification/... ./cmd/gomad/...`
`go test -tags test_dep ./...` in each sibling module importing Runner, if the consumer inventory finds one.

## Acceptance
- [ ] R4 removes inaccessible public interfaces/fields across all five use cases.
- [ ] Repository consumer inventory and migration are complete; usable existing public seams remain.
- [ ] External intended use compiles; fake-executor failure/cancellation tests pass via private dependencies.
- [ ] Default supervisor/coordinator behavior is unchanged; no mutable package-global injection.

## Done summary
NOT IMPLEMENTED. Moved to fn-105-gomad-follow-ups-deferred-scope.3 (D3) on 2026-09-29 as a scope cut; the task text above remains the implementation brief.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs: