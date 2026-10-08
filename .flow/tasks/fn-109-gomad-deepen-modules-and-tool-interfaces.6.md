---
satisfies: [R5]
---
# fn-109-gomad-deepen-modules-and-tool-interfaces.6 Move public executor injection behind private dependencies (fulfils fn-105.3 D3)

## Description


Owner amendment (2026-10-07). This task's remaining native darwin/arm64 execution, native reports/packs/replay, native qualification measurements, soak and platform-specific qualification guidance transfer to [fn-149-gomad-deferred-darwin-qualification](../specs/fn-149-gomad-deferred-darwin-qualification.md), with exact owners in the [native transfer manifest](../artifacts/native-scope-transfer-2026-10-07.md). Native linux/amd64 qualification and Linux CI work remain deferred under fn-128. Missing transferred native evidence cannot block this task or its source admission. This supersedes older native-first, missing-Darwin and no-renewed-deferral clauses only for transferred obligations. Implementation, ordinary host-source coverage, lint, both-source-set static checks, generated-output validation, byte equivalence, fixed-identity/matched-first-baseline preservation, non-native measurements, source review, docs consistency, actual checkout prerequisites and predecessor source integration/review/retained acceptance remain required. Full native test-host execution belongs to the native owner; partial portable runs cannot stand in for it or excuse portable failures. All other criteria and historical evidence below retain their original meaning. No task completion, native pass, PR, push or CI action follows from this transfer.

Owner amendment (2026-10-04): this task transfers every remaining native Linux execution, Linux pack/report/replay and Linux-specific qualification-documentation requirement to [fn-128.1](../tasks/fn-128-gomad-deferred-linux-qualification-and.1.md), [fn-128.4](../tasks/fn-128-gomad-deferred-linux-qualification-and.4.md), [fn-128.7](../tasks/fn-128-gomad-deferred-linux-qualification-and.7.md). Native execution/full/affected gates still owned here apply to Darwin. Missing transferred Linux proof cannot block this task. Static coverage of both supported source sets, shared implementation, preservation, review and other non-Linux requirements remain unchanged. Retained scope: Implementation, both-source-set static coverage, R18 preservation, admission dependencies, lint, formal review and Darwin/full/affected gates. See the [transfer manifest](../artifacts/linux-scope-transfer-2026-10-04.md). Historical progress below retains its original meaning and is not current-candidate proof.

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


Owner amendment (2026-10-07). This task's remaining native darwin/arm64 execution, native reports/packs/replay, native qualification measurements, soak and platform-specific qualification guidance transfer to [fn-149-gomad-deferred-darwin-qualification](../specs/fn-149-gomad-deferred-darwin-qualification.md), with exact owners in the [native transfer manifest](../artifacts/native-scope-transfer-2026-10-07.md). Native linux/amd64 qualification and Linux CI work remain deferred under fn-128. Missing transferred native evidence cannot block this task or its source admission. This supersedes older native-first, missing-Darwin and no-renewed-deferral clauses only for transferred obligations. Implementation, ordinary host-source coverage, lint, both-source-set static checks, generated-output validation, byte equivalence, fixed-identity/matched-first-baseline preservation, non-native measurements, source review, docs consistency, actual checkout prerequisites and predecessor source integration/review/retained acceptance remain required. Full native test-host execution belongs to the native owner; partial portable runs cannot stand in for it or excuse portable failures. All other criteria and historical evidence below retain their original meaning. No task completion, native pass, PR, push or CI action follows from this transfer.

Current native-execution acceptance is Darwin-only here. The corresponding Linux clauses and any older missing-Linux completion rule are transferred to [fn-128.1](../tasks/fn-128-gomad-deferred-linux-qualification-and.1.md), [fn-128.4](../tasks/fn-128-gomad-deferred-linux-qualification-and.4.md), [fn-128.7](../tasks/fn-128-gomad-deferred-linux-qualification-and.7.md). All other acceptance below remains in force.

- [x] `go-interface-changes.md` lists every removed or changed exported declaration, its consumers, the replacement and the migration, written before the edits and updated to match the result.
- [x] No exported Runner signature or field mentions a `runner/internal/execution` type; `Executor`/`ReplayExecutor` injection exists only as private dependencies.
- [x] `Preparer` and `ArtifactReplayer` seams remain public and usable; the external-consumer fixture compiles from outside the Runner subtree.
- [x] Fake-executor failure, cancellation and watchdog tests pass through the private entry points with unchanged semantics; no global mutable test hook exists.
- [x] Default supervisor/coordinator behaviour and the injection-versus-isolation rejections are unchanged.
- [x] fn-105.3 is closed by reference to this task (one owner); its summary cites this task's evidence and does not claim separate work.
## Done summary
Removed the inaccessible public Executor/ReplayExecutor interfaces and all five public executor fields. Explore, portable planning, shards, resume, replay and minimization use private executionDependencies through six private entrypoints. Public Preparer and ArtifactReplayer remain usable; the external-consumer fixture compiles from a separate module. Resume keeps fake dependencies, default process checks and coordinator injection rejection retain behavior, and minimizer candidates/default replay share the same fake.

The pre-edit inventory is updated at `.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/go-interface-changes.md`. Actual original byte snapshots, final-source.json (31 files), task-only.patch, and parent-source-verification.json bind the migration. Original comments and test assertions are preserved; canonical CampaignOptions characterization passes.

Darwin evidence: pre-edit broad Quick pass; API negative-control assertion red; private failure/cancellation/watchdog/replay/minimize/plan tests pass; root architecture/public API/external compile and scoped vet pass. Final task-only lint has zero issues; unfiltered lint retains 627 historical findings. Root make lint-code-fast remains the previously recorded nested-module loading exit2.

The frozen full host gate actually exited 2 after 179 seconds: 44 package results passed and Runner failed only TestCoordinatorTransportCoversEveryCampaignSpecField because its local-only map still named the removed Executor. The only post-gate source delta is coordinator_transport_test.go: the stale entry was removed and private injection rejection is checked. The exact failed selector and coordinator family pass; independent parent canonical and architecture/external checks pass. The original failed gate, immutable source manifest and log are retained. No repeated broad gate or original exit0 is claimed.

Formal read-only review returned SHIP at 2026-10-03T13:23:49.288652Z, with zero introduced/preexisting findings and no unaddressed requirements. Writer and reviewer both used gpt-6-sol high in fresh contexts (same-family limitation). Post-review source verification has no mismatches. Fn-105-gomad-follow-ups-deferred-scope.3 is closed and verified done by reference to this implementation; d3-reference-summary.md/evidence retain that single ownership.

Native linux/amd64 and full-spec R18/R19 qualification remain incomplete under task21. No staging, commit, stash, push or worktree was used; commits=[] by user instruction.

stage: impl-review - ran (receipt dated 2026-10-03T13:23:49.288652Z; model: gpt-6-sol at high)
stage: plan-sync - skipped(config: planSync.enabled != true)
Tracker sync: n/a (bridge inactive).

Evidence: handover.json, final-completion-evidence.json, parent-final-checks.json, parent-source-verification.json, post-review-verification.json, working-tree-review.json, full-host-start/result/source.json and retained logs in this directory.
## Evidence
- Commits:
- Tests: tools/gomad3: .toolchain/bin/go test -count=1 -tags test_dep ./runner/... ./qualification/... ./cmd/gomad/... => exit 0; baseline-quick.log, tools/gomad3: .toolchain/bin/go test -count=1 -tags test_dep . -run 'TestRunnerExecutionInjectionIsPrivate' => exit 1; red-public-interface.log; expected red: two exported interfaces and five public executor fields, tools/gomad3: .toolchain/bin/go test -count=1 -tags test_dep . -run 'TestRunnerExecutionInjectionIsPrivate|TestRunnerRequestsCompileInExternalModule|TestPackageArchitecture|TestPublicPackagesDoNotExportTypeAliases' => exit 0; external-architecture.log, tools/gomad3: .toolchain/bin/go test -count=1 -tags test_dep ./runner -run 'TestRunPreparesOnce|TestRunFirstFailure|TestRunCancellation|TestRunChoiceExplorationResume|TestRunSimulationExplorationResume|TestRunCampaignShard|TestMinimize|TestReplay|TestWatchdog|TestGuidedResume|TestRunEarlyCompletion|TestDiagnosticsPlanShard|TestCreateCampaignPlan' => exit 0; focused-runner.log, tools/gomad3: /Users/stephan/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.darwin-arm64/bin/go vet -tags test_dep . ./runner/... ./qualification/... ./cmd/... => exit 0; scoped-vet.log, tools/gomad3: .toolchain/bin/go test -count=1 -tags test_dep ./runner -run 'TestCoordinatorTransportCoversEveryCampaignSpecField|TestExploreRejectsPrivateExecutionForIsolatedCampaign|TestCoordinatorTransportRoundTripsEveryTransportedField' => exit 0; coordinator-delta.log, tools/gomad3: .bin/golangci-lint-v2.13.0 run --config ../../.github/.golangci.yml --build-tags test_dep --timeout 10m --new-from-patch ../../.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-6/task-only.patch . ./runner/... ./qualification/... ./cmd/... => exit 0; task-only-lint-post-delta.log; 0 task-only issues; unfiltered lint still has inherited/retained findings, tools/gomad3: .toolchain/bin/go test -count=1 -tags test_dep ./runner -run TestCampaignOptionsCharacterization|TestCoordinatorTransport|TestExploreRejectsPrivateExecutionForIsolatedCampaign => exit 0; parent-final-check-1.log, tools/gomad3: .toolchain/bin/go test -count=1 -tags test_dep . -run TestPackageArchitecture|TestPublicPackagesDoNotExportTypeAliases|TestRunnerExecutionInjectionIsPrivate|TestRunnerRequestsCompileInExternalModule => exit 0; parent-final-check-2.log, Frozen Darwin test-host => actual exit 2, one stale coordinator test field map; full-host.log/full-host-result.json. Only test file changed afterward; exact failed selector and parent canonical/architecture/external checks pass. Full host not rerun., Parent verified 31 final-source hashes, actual byte preimages, original comments, task-only reverse apply, and git diff --check; no source changes after SHIP. post-review-verification.json., Native linux/amd64 and full-spec R18/R19 qualification remain incomplete under task21., Root make lint-code-fast known unchanged nested-module-loading exit2 retained; unfiltered scoped lint627 historical findings, task-only filtered lint0.
- PRs:
