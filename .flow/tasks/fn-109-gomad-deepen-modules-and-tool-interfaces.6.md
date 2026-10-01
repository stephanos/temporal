---
satisfies: [R5]
---
# fn-109-gomad-deepen-modules-and-tool-interfaces.6 Move public executor injection behind private dependencies (fulfils fn-105.3 D3)

## Description
Stage 3, R5 (F5). Public `Executor` and `ReplayExecutor` take and return `runner/internal/execution` types that no consumer outside the Runner subtree can import. This task is the single implementation owner of fn-105.3 (D3, origin brief `.flow/tasks/fn-102-gomad-architecture-consolidate.4.md`): do the work here, then close fn-105.3 by reference to this task's evidence. Do not implement it twice.

**External ordering:** start only after the fn-108 R6/R7 tasks are done and verified: `fn-108-gomad-reduce-code-size-without-removing.5` (shared assessment, R6) and `.6` (retention and artifact-input composition, R7) in `tools/gomad3/runner`. Re-anchor the line references below against the post-fn-108 source first. flowctl cannot record a cross-spec task edge, so check `flowctl tasks --spec fn-108-gomad-reduce-code-size-without-removing` before `flowctl start`. Public Go changes overlapping fn-108 follow its completed extraction.

**Size:** M
**Files:** `tools/gomad3/runner/{runner.go,replay_operation.go,minimize_operation.go,campaign_shard_execution.go,resume.go,portable_plan.go}`, the runner tests that inject executors, an external-consumer compile fixture, `.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/go-interface-changes.md`.
**Touches:** [tools/gomad3/runner/**, tools/gomad3/qualification/**, tools/gomad3/cmd/gomad/**, tools/gomad3/internal/gomadtool/conformance/testdata/**, .flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/go-interface-changes.md]

### Approach
- Inventory first, written to `go-interface-changes.md` before any edit: every consumer of the exported names, the exact removed/changed declarations, the replacement construction and each caller migration. Current inventory to re-verify: `Executor` (`runner.go:118-120`) and fields `CampaignSpec.Executor` (`:172`), `ShardSpec.Executor` (`campaign_shard_execution.go:29`), `MinimizeSpec.Executor` (`minimize_operation.go:35`); `ReplayExecutor` (`replay_operation.go:28`) and `ReplaySpec.Executor` (`:41`). Production code outside `runner/` never sets them: `cmd/gomad/internal/cli` and `qualification/workload/workload.go:31-32,115-118` inject whole operations as functions. Test users: `runner_test.go`, `replay_operation_test.go`, `portable_plan_test.go`, `minimize_operation_test.go`, `inspect_test.go` (all package `runner`). `tools/gomad3sim` and `tools/gomad3integration` do not import Runner.
- Remove the two interfaces and four fields from the public surface. Thread a private dependencies value through internal entry points (the `toolchain.Build` / `buildWith` shape at `toolchain/build.go:63-82` is the house pattern). Same-package tests pass fakes through the private entry point. No package-level mutable hook.
- Keep the usable seams public: `Preparer` (`runner.go:114`), `ArtifactReplayer` (`:122`), `CampaignSpec.Preparer`/`Replayer` and their shard/minimize counterparts.
- Behaviour that keys on injection must survive: `Explore` rejects injection with a coordinator command (`runner.go:371-373`), resume requires a supervisor command without an executor (`:1202`), replay checks `replayProcessExecutor` (`replay_operation.go:229`), and `simulationCapabilityForJob(executor, job)` (`runner.go:1542`) depends on the executor kind.
- Do not replace the removed surface with descriptor, bootstrap or `execution.Spec` types in any exported signature.
- External-consumer fixture: a module outside the Runner subtree (a temp module with a `replace` to `tools/gomad3`, built by a test, or a fixture under conformance `testdata`) that constructs `CampaignSpec`, `ReplaySpec`, `MinimizeSpec`, a custom `Preparer` and an `ArtifactReplayer` and compiles.

### Investigation targets
**Required:**
- `tools/gomad3/runner/runner.go:112-177,298-315,362-377,1194-1210,1450-1560`
- `tools/gomad3/runner/replay_operation.go:28-125,225-235`
- `tools/gomad3/runner/minimize_operation.go:28-70,250-265,370-390`
- `tools/gomad3/runner/campaign_shard_execution.go:20-95,135-177`
- `.flow/tasks/fn-102-gomad-architecture-consolidate.4.md`
**Optional:**
- `tools/gomad3/toolchain/build.go:59-82` (private dependencies pattern)

### Quick commands
```bash
cd tools/gomad3
env -u GOMADSEED -u GOMAD3_CHILD_SEED GOWORK=off .toolchain/bin/go test -count=1 -tags test_dep ./runner/... ./qualification/... ./cmd/gomad/...
GOWORK=off go vet -tags test_dep ./runner/... ./qualification/... ./cmd/...
env -u GOMADSEED -u GOMAD3_CHILD_SEED GOWORK=off .toolchain/bin/go test -count=1 -tags test_dep . -run 'TestPackageArchitecture|TestPublicPackagesDoNotExportTypeAliases'
cd ../.. && flowctl show fn-105-gomad-follow-ups-deferred-scope.3
```

### Constraints
- No `git add`, commit, stash or worktree: the user owns commits. Record `"commits": []` in the `flowctl done` evidence and say so in the summary.
- No new third-party dependency. `tools/gomad3/go.mod` requires only `golang.org/x/mod`, so testify is unavailable inside `tools/gomad3`: follow the existing `t.Fatalf` style with whole-value comparisons there. In the root module (`tools/gomad3sim`, `tools/gomad3integration`) use `require` with `Equal`/`EqualValues`.
- Preserve existing comments with their owning code, CLI grammar/defaults, canonical bytes for fixed supplied identities, and error precedence/classification.
- This host is `darwin/arm64`. `linux/amd64` gates cannot run here: list them as incomplete in the done summary, never claim them.
- fn-105 D12/D14 replay-divergence dispositions stay unchanged. Attribute a failure to those owners with retained evidence instead of relaxing an expectation.
- Run tests with `-tags test_dep`. Baseline the Quick commands before editing so a pre-existing failure is not attributed to this task.
- Evidence and decision records go under `.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/`.
## Acceptance
- [ ] `go-interface-changes.md` lists every removed or changed exported declaration, its consumers, the replacement and the migration, written before the edits and updated to match the result.
- [ ] No exported Runner signature or field mentions a `runner/internal/execution` type; `Executor`/`ReplayExecutor` injection exists only as private dependencies.
- [ ] `Preparer` and `ArtifactReplayer` seams remain public and usable; the external-consumer fixture compiles from outside the Runner subtree.
- [ ] Fake-executor failure, cancellation and watchdog tests pass through the private entry points with unchanged semantics; no global mutable test hook exists.
- [ ] Default supervisor/coordinator behaviour and the injection-versus-isolation rejections are unchanged.
- [ ] fn-105.3 is closed by reference to this task (one owner); its summary cites this task's evidence and does not claim separate work.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
